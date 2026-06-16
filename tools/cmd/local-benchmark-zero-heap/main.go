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
	"sort"
	"strings"
	"tetra_language/tools/internal/heaptelemetry"
	"tetra_language/tools/internal/zeroheapbench"
	"time"
)

const schemaBenchmarkMemoryV1 = "tetra.local_benchmark.memory_evidence.v1"

type options struct {
	OutDir     string
	Iterations int
	Timeout    time.Duration
}

type report struct {
	Schema      string   `json:"schema"`
	Scope       string   `json:"scope"`
	GeneratedAt string   `json:"generated_at"`
	Policy      policy   `json:"policy"`
	NonClaims   []string `json:"non_claims"`
	Results     []row    `json:"results"`
}

type policy struct {
	Suite      string `json:"suite"`
	Iterations int    `json:"iterations"`
}

type row struct {
	Name               string        `json:"name"`
	Category           string        `json:"category"`
	AlgorithmID        string        `json:"algorithm_id"`
	InputDescription   string        `json:"input_description"`
	Language           string        `json:"language"`
	Status             string        `json:"status"`
	CompilerVersion    string        `json:"compiler_version"`
	BuildCommand       []string      `json:"build_command"`
	RunCommand         []string      `json:"run_command"`
	SourcePath         string        `json:"source_path"`
	BinaryPath         string        `json:"binary_path"`
	BinarySizeBytes    int64         `json:"binary_size_bytes"`
	CompileTimeMS      float64       `json:"compile_time_ms"`
	RunMeasurementsMS  []float64     `json:"run_measurements_ms"`
	MedianRuntimeMS    float64       `json:"median_runtime_ms"`
	RawOutputArtifacts []string      `json:"raw_output_artifacts"`
	TetraMetadata      tetraMetadata `json:"tetra_metadata"`
	Error              string        `json:"error,omitempty"`
}

type tetraMetadata struct {
	ProofReport       string         `json:"proof_report"`
	BoundsReport      string         `json:"bounds_report"`
	AllocationReport  string         `json:"allocation_report"`
	PerfBlockerReport string         `json:"perf_blocker_report"`
	BackendReport     string         `json:"backend_report"`
	BackendPath       string         `json:"backend_path"`
	BoundsLeft        int            `json:"bounds_left"`
	HeapAllocations   int            `json:"heap_allocations"`
	PerfBlockers      []string       `json:"perf_blockers"`
	MemoryEvidence    memoryEvidence `json:"memory_evidence"`
}

type memoryEvidence struct {
	Schema              string             `json:"schema"`
	HeapAllocBytes      memoryMetric       `json:"heap_alloc_bytes"`
	BytesRequested      memoryMetric       `json:"bytes_requested"`
	BytesReserved       memoryMetric       `json:"bytes_reserved"`
	BytesCommitted      memoryMetric       `json:"bytes_committed"`
	BytesCopied         memoryMetric       `json:"bytes_copied"`
	RSSCurrent          memoryMetric       `json:"rss_current"`
	RSSPeak             memoryMetric       `json:"rss_peak"`
	DomainBytesEvidence memoryMetric       `json:"domain_bytes_evidence"`
	DomainBytes         []memoryDomainByte `json:"domain_bytes"`
}

type memoryMetric struct {
	Bytes             uint64 `json:"bytes"`
	CurrentBytes      uint64 `json:"current_bytes"`
	PeakBytes         uint64 `json:"peak_bytes"`
	TotalAllocBytes   uint64 `json:"total_alloc_bytes"`
	AllocationCount   uint64 `json:"allocation_count"`
	EvidenceClass     string `json:"evidence_class"`
	Method            string `json:"method"`
	SourceArtifact    string `json:"source_artifact,omitempty"`
	UnsupportedReason string `json:"unsupported_reason,omitempty"`
	BlockedReason     string `json:"blocked_reason,omitempty"`
}

type memoryDomainByte struct {
	DomainID       string `json:"domain_id"`
	Kind           string `json:"kind"`
	RequestedBytes uint64 `json:"requested_bytes,omitempty"`
	ReservedBytes  uint64 `json:"reserved_bytes,omitempty"`
	CommittedBytes uint64 `json:"committed_bytes,omitempty"`
	CurrentBytes   uint64 `json:"current_bytes,omitempty"`
	PeakBytes      uint64 `json:"peak_bytes,omitempty"`
	BytesCopied    uint64 `json:"bytes_copied,omitempty"`
	EvidenceClass  string `json:"evidence_class"`
	Method         string `json:"method"`
	SourceArtifact string `json:"source_artifact,omitempty"`
}

type runtimeHeapEvidence struct {
	SourceArtifact string
	Sample         heaptelemetry.Sample
}

func main() {
	outDir := flag.String("out-dir", "reports/local-benchmark-zero-heap-v1", "output artifact directory")
	iterations := flag.Int("iterations", 3, "run iterations per zero-heap microbenchmark")
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
	if err := prepareOutDir(opt.OutDir); err != nil {
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

	report := report{
		Schema:      zeroheapbench.Schema,
		Scope:       zeroheapbench.Scope,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Policy:      policy{Suite: "zero_heap_microbenchmarks", Iterations: opt.Iterations},
		NonClaims: []string{
			"no official benchmark claim",
			"no cross-language performance claim",
			"no zero RSS claim",
			"no universal zero heap claim",
		},
	}
	version := commandOutput(opt.Timeout, []string{tetraTool, "version"}, env)
	for _, spec := range zeroheapbench.BuildSpecs(opt.OutDir) {
		report.Results = append(report.Results, executeSpec(spec, opt, env, tetraTool, version))
	}
	if err := writeJSON(filepath.Join(opt.OutDir, "report.json"), report); err != nil {
		return err
	}
	if err := writeSummary(filepath.Join(opt.OutDir, "summary.md"), report); err != nil {
		return err
	}
	return nil
}

func prepareOutDir(outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	for _, rel := range []string{"artifacts", "report.json", "summary.md"} {
		if err := os.RemoveAll(filepath.Join(outDir, rel)); err != nil {
			return err
		}
	}
	for _, rel := range []string{
		filepath.Join("artifacts", "src", "zero-heap"),
		filepath.Join("artifacts", "bin"),
		filepath.Join("artifacts", "raw"),
	} {
		if err := os.MkdirAll(filepath.Join(outDir, rel), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func executeSpec(spec zeroheapbench.Spec, opt options, env []string, tetraTool string, version string) row {
	buildStdout := filepath.Join(opt.OutDir, "artifacts", "raw", spec.Name+".build.stdout.txt")
	buildStderr := filepath.Join(opt.OutDir, "artifacts", "raw", spec.Name+".build.stderr.txt")
	runStdout := filepath.Join(opt.OutDir, "artifacts", "raw", spec.Name+".run.stdout.txt")
	runStderr := filepath.Join(opt.OutDir, "artifacts", "raw", spec.Name+".run.stderr.txt")
	heapTelemetryDir := filepath.Join(opt.OutDir, "artifacts", "heap-telemetry", spec.Name, "runtime")
	_ = os.MkdirAll(filepath.Dir(spec.SourceRelPath), 0o755)
	_ = os.WriteFile(spec.SourceRelPath, []byte(spec.Source), 0o644)

	buildCommand := []string{
		tetraTool, "build", "--target", "linux-x64", "--explain",
		"--emit-runtime-heap-telemetry", "--runtime-heap-telemetry-dir", heapTelemetryDir,
		"-o", spec.BinaryRelPath, spec.SourceRelPath,
	}
	runCommand := []string{spec.BinaryRelPath}
	result := row{
		Name:             spec.Name,
		Category:         spec.Category,
		AlgorithmID:      spec.AlgorithmID,
		InputDescription: spec.InputDescription,
		Language:         "tetra",
		Status:           "measured",
		CompilerVersion:  version,
		BuildCommand:     buildCommand,
		RunCommand:       runCommand,
		SourcePath:       spec.SourceRelPath,
		BinaryPath:       spec.BinaryRelPath,
		RawOutputArtifacts: []string{
			buildStdout,
			buildStderr,
			runStdout,
			runStderr,
		},
	}
	_, buildDuration, err := runCaptured(opt.Timeout, buildCommand, env, buildStdout, buildStderr)
	result.CompileTimeMS = millis(buildDuration)
	if err != nil {
		result.Status = "build_failed"
		result.Error = err.Error()
		ensureRawRunArtifacts(runStdout, runStderr, "not run because build failed\n")
		result.TetraMetadata = missingTetraMetadata(spec.BinaryRelPath, "Tetra build failed before memory artifacts were produced")
		return result
	}
	if info, err := os.Stat(spec.BinaryRelPath); err == nil {
		result.BinarySizeBytes = info.Size()
	}
	measurements, heapEvidence, heapArtifacts, runErr := runIterationsWithHeapTelemetry(opt.Timeout, runCommand, env, opt.Iterations, runStdout, runStderr, heapTelemetryDir, spec.Name, opt.OutDir)
	result.RawOutputArtifacts = append(result.RawOutputArtifacts, heapArtifacts...)
	result.RunMeasurementsMS = measurements
	result.MedianRuntimeMS = median(measurements)
	if runErr != nil {
		result.Status = "run_failed"
		result.Error = runErr.Error()
	}
	result.TetraMetadata = collectTetraMetadata(spec.Name, spec.BinaryRelPath, heapEvidence)
	if result.Status != "measured" {
		result.TetraMetadata.MemoryEvidence = *blockedMemoryEvidence("Tetra zero-heap benchmark run failed before runtime heap telemetry could be trusted: " + result.Error)
	}
	return result
}

func runIterationsWithHeapTelemetry(timeout time.Duration, argv []string, env []string, iterations int, stdoutPath string, stderrPath string, telemetryDir string, benchmarkName string, outDir string) ([]float64, *runtimeHeapEvidence, []string, error) {
	var stdoutAll bytes.Buffer
	var stderrAll bytes.Buffer
	var measurements []float64
	var artifacts []string
	var selected *runtimeHeapEvidence
	var firstErr error
	if err := os.MkdirAll(telemetryDir, 0o755); err != nil {
		return nil, nil, nil, err
	}
	sourceSidecar := filepath.Join(telemetryDir, filepath.Base(argv[0])+".heap.json")
	for i := 0; i < iterations; i++ {
		_ = os.Remove(sourceSidecar)
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
		heapArtifact := filepath.Join(outDir, "artifacts", "heap-telemetry", benchmarkName, fmt.Sprintf("iteration-%02d.heap.json", i+1))
		if err := copyFile(sourceSidecar, heapArtifact); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("runtime heap telemetry sidecar for %s iteration %d: %w", benchmarkName, i+1, err)
			}
			continue
		}
		artifacts = append(artifacts, heapArtifact)
		sample, err := heaptelemetry.ReadFile(heapArtifact, outDir)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("runtime heap telemetry sidecar for %s iteration %d: %w", benchmarkName, i+1, err)
			}
			continue
		}
		candidate := &runtimeHeapEvidence{SourceArtifact: heapArtifact, Sample: sample}
		if selected == nil || runtimeHeapSampleBetter(candidate.Sample, selected.Sample) {
			selected = candidate
		}
	}
	_ = os.WriteFile(stdoutPath, stdoutAll.Bytes(), 0o644)
	_ = os.WriteFile(stderrPath, stderrAll.Bytes(), 0o644)
	return measurements, selected, artifacts, firstErr
}

func runtimeHeapSampleBetter(candidate heaptelemetry.Sample, current heaptelemetry.Sample) bool {
	if candidate.HeapPeakBytes != current.HeapPeakBytes {
		return candidate.HeapPeakBytes > current.HeapPeakBytes
	}
	if candidate.HeapTotalAllocBytes != current.HeapTotalAllocBytes {
		return candidate.HeapTotalAllocBytes > current.HeapTotalAllocBytes
	}
	return candidate.HeapAllocationCount > current.HeapAllocationCount
}

func collectTetraMetadata(name string, binaryPath string, runtimeHeap *runtimeHeapEvidence) tetraMetadata {
	proof := binaryPath + ".proof.json"
	bounds := binaryPath + ".bounds.json"
	alloc := binaryPath + ".alloc.json"
	perf := binaryPath + ".perf.json"
	backend := binaryPath + ".backend.json"
	heap := readHeapAllocations(alloc)
	return tetraMetadata{
		ProofReport:       proof,
		BoundsReport:      bounds,
		AllocationReport:  alloc,
		PerfBlockerReport: perf,
		BackendReport:     backend,
		BackendPath:       readBackendPath(backend),
		BoundsLeft:        readBoundsLeft(bounds),
		HeapAllocations:   heap,
		PerfBlockers:      readPerfBlockers(perf, name, heap),
		MemoryEvidence:    *collectMemoryEvidence(alloc, runtimeHeap),
	}
}

func missingTetraMetadata(binaryPath string, reason string) tetraMetadata {
	return tetraMetadata{
		ProofReport:       binaryPath + ".proof.json",
		BoundsReport:      binaryPath + ".bounds.json",
		AllocationReport:  binaryPath + ".alloc.json",
		PerfBlockerReport: binaryPath + ".perf.json",
		BackendReport:     binaryPath + ".backend.json",
		BackendPath:       "fallback",
		MemoryEvidence:    *blockedMemoryEvidence(reason),
	}
}

func collectMemoryEvidence(allocationReport string, runtimeHeap *runtimeHeapEvidence) *memoryEvidence {
	var parsed struct {
		Summary struct {
			BytesRequested uint64 `json:"bytes_requested"`
			BytesReserved  uint64 `json:"bytes_reserved"`
			Domains        []struct {
				DomainID       string `json:"domain_id"`
				Kind           string `json:"kind"`
				RequestedBytes uint64 `json:"requested_bytes"`
				ReservedBytes  uint64 `json:"reserved_bytes"`
				BytesCopied    uint64 `json:"bytes_copied"`
			} `json:"domains"`
		} `json:"summary"`
	}
	if err := readJSON(allocationReport, &parsed); err != nil {
		return blockedMemoryEvidence("allocation report was unavailable while collecting zero-heap memory evidence")
	}
	heapMetric := blockedMemoryMetric("runtime heap telemetry was not collected")
	if runtimeHeap != nil {
		heapMetric = runtimeHeapMetric(runtimeHeap)
	}
	domains := make([]memoryDomainByte, 0, len(parsed.Summary.Domains))
	var copied uint64
	for _, domain := range parsed.Summary.Domains {
		copied += domain.BytesCopied
		domains = append(domains, memoryDomainByte{
			DomainID:       domain.DomainID,
			Kind:           domain.Kind,
			RequestedBytes: domain.RequestedBytes,
			ReservedBytes:  domain.ReservedBytes,
			BytesCopied:    domain.BytesCopied,
			EvidenceClass:  "allocation_report_estimate",
			Method:         "allocation_report_summary",
			SourceArtifact: allocationReport,
		})
	}
	domainEvidence := memoryMetric{
		EvidenceClass:  "allocation_report_estimate",
		Method:         "allocation_report_summary",
		SourceArtifact: allocationReport,
	}
	if len(domains) == 0 {
		domainEvidence = unsupportedMemoryMetric("allocation report summary does not include memory domains")
	}
	return &memoryEvidence{
		Schema:              schemaBenchmarkMemoryV1,
		HeapAllocBytes:      heapMetric,
		BytesRequested:      allocationReportMetric(parsed.Summary.BytesRequested, allocationReport),
		BytesReserved:       allocationReportMetric(parsed.Summary.BytesReserved, allocationReport),
		BytesCommitted:      unsupportedMemoryMetric("allocation report does not expose committed bytes"),
		BytesCopied:         allocationReportMetric(copied, allocationReport),
		RSSCurrent:          unsupportedMemoryMetric("zero-heap suite does not measure process RSS"),
		RSSPeak:             unsupportedMemoryMetric("zero-heap suite does not measure process RSS"),
		DomainBytesEvidence: domainEvidence,
		DomainBytes:         domains,
	}
}

func runtimeHeapMetric(evidence *runtimeHeapEvidence) memoryMetric {
	sample := evidence.Sample
	return memoryMetric{
		Bytes:           sample.HeapPeakBytes,
		CurrentBytes:    sample.HeapCurrentBytes,
		PeakBytes:       sample.HeapPeakBytes,
		TotalAllocBytes: sample.HeapTotalAllocBytes,
		AllocationCount: sample.HeapAllocationCount,
		EvidenceClass:   "runtime_measured",
		Method:          heaptelemetry.MethodLinuxX64HeapTelemetryV1,
		SourceArtifact:  evidence.SourceArtifact,
	}
}

func allocationReportMetric(bytes uint64, sourceArtifact string) memoryMetric {
	return memoryMetric{Bytes: bytes, EvidenceClass: "allocation_report_estimate", Method: "allocation_report_summary", SourceArtifact: sourceArtifact}
}

func unsupportedMemoryMetric(reason string) memoryMetric {
	return memoryMetric{EvidenceClass: "unsupported", Method: "not_collected", UnsupportedReason: reason}
}

func blockedMemoryEvidence(reason string) *memoryEvidence {
	metric := blockedMemoryMetric(reason)
	return &memoryEvidence{
		Schema:              schemaBenchmarkMemoryV1,
		HeapAllocBytes:      metric,
		BytesRequested:      metric,
		BytesReserved:       metric,
		BytesCommitted:      metric,
		BytesCopied:         metric,
		RSSCurrent:          metric,
		RSSPeak:             metric,
		DomainBytesEvidence: metric,
		DomainBytes:         []memoryDomainByte{},
	}
}

func blockedMemoryMetric(reason string) memoryMetric {
	return memoryMetric{EvidenceClass: "blocked", Method: "missing_artifacts", BlockedReason: reason}
}

func readBoundsLeft(path string) int {
	var parsed struct {
		Totals struct {
			Left int `json:"left"`
		} `json:"totals"`
	}
	if readJSON(path, &parsed) != nil {
		return 0
	}
	return parsed.Totals.Left
}

func readHeapAllocations(path string) int {
	var parsed struct {
		Totals struct {
			Heap int `json:"heap"`
		} `json:"totals"`
	}
	if readJSON(path, &parsed) != nil {
		return 0
	}
	return parsed.Totals.Heap
}

func readBackendPath(path string) string {
	var parsed struct {
		Summary struct {
			RegisterPath  int `json:"register_path"`
			StackFallback int `json:"stack_fallback"`
		} `json:"summary"`
	}
	if readJSON(path, &parsed) != nil {
		return "fallback"
	}
	if parsed.Summary.StackFallback > 0 {
		return "fallback"
	}
	if parsed.Summary.RegisterPath > 0 {
		return "register"
	}
	return "stack"
}

func readPerfBlockers(path string, benchmark string, heapAllocations int) []string {
	var parsed struct {
		Benchmarks []struct {
			Benchmark   string   `json:"benchmark"`
			ReasonCode  string   `json:"reason_code"`
			ReasonCodes []string `json:"reason_codes"`
		} `json:"benchmarks"`
	}
	if readJSON(path, &parsed) != nil {
		return nil
	}
	seen := map[string]bool{}
	var blockers []string
	for _, item := range parsed.Benchmarks {
		if item.Benchmark != "" && item.Benchmark != benchmark {
			continue
		}
		codes := append([]string{}, item.ReasonCodes...)
		if item.ReasonCode != "" {
			codes = append(codes, item.ReasonCode)
		}
		for _, code := range codes {
			if heapAllocations == 0 && code == "allocation.local_call_heap_fallback" {
				continue
			}
			if code != "" && !seen[code] {
				seen[code] = true
				blockers = append(blockers, code)
			}
		}
	}
	sort.Strings(blockers)
	return blockers
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

func commandEnv(root string) []string {
	env := os.Environ()
	env = append(env, "GOCACHE="+filepath.Join(root, ".cache", "go-build-zero-heap-suite-runner"))
	return env
}

func copyFile(src string, dst string) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, raw, 0o644)
}

func ensureRawRunArtifacts(stdoutPath string, stderrPath string, message string) {
	_ = os.WriteFile(stdoutPath, []byte(message), 0o644)
	_ = os.WriteFile(stderrPath, nil, 0o644)
}

func writeSummary(path string, parsed report) error {
	var b strings.Builder
	b.WriteString("# Local Zero-Heap Benchmark V1\n\n")
	b.WriteString("Status: local Tetra-only zero-heap guardrail evidence. No official, cross-language performance, zero RSS, or universal zero-heap claim is made.\n\n")
	b.WriteString("| Category | Status | Heap total alloc bytes | Heap allocation count |\n")
	b.WriteString("| --- | --- | ---: | ---: |\n")
	for _, row := range parsed.Results {
		heap := row.TetraMetadata.MemoryEvidence.HeapAllocBytes
		fmt.Fprintf(&b, "| %s | %s | %d | %d |\n", row.Category, row.Status, heap.TotalAllocBytes, heap.AllocationCount)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
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

func readJSON(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
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
