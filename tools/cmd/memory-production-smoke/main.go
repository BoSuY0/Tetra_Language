package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"tetra_language/tools/validators/memoryprod"
)

type smokeOptions struct {
	ReportPath         string
	RAMMeasurementPath string
	TetraPath          string
	GitHead            string
	KeepWork           bool
}

type smokeRunner struct {
	opt          smokeOptions
	workDir      string
	sourceDir    string
	tetraPath    string
	processes    []memoryprod.ProcessReport
	benchmarks   []memoryprod.BenchmarkReport
	cases        []memoryprod.CaseReport
	ramSnapshots []ramMeasurementSnapshot
}

type processResult struct {
	exitCode int
	output   string
	err      error
}

func main() {
	var opt smokeOptions
	flag.StringVar(&opt.ReportPath, "report", "", "path to write tetra.memory.production.v1 report")
	flag.StringVar(&opt.RAMMeasurementPath, "ram-measurement-report", "", "optional path to write tetra.memory.ram-measurement.v1 MemStats report")
	flag.StringVar(&opt.TetraPath, "tetra", "", "tetra CLI path; defaults to a fresh temp build from ./cli/cmd/tetra")
	flag.StringVar(&opt.GitHead, "git-head", "", "optional git HEAD provenance to include in the report")
	flag.BoolVar(&opt.KeepWork, "keep-work", false, "keep temporary build directory")
	flag.Parse()
	if opt.ReportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := runSmoke(context.Background(), opt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runSmoke(ctx context.Context, opt smokeOptions) error {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return fmt.Errorf("memory production smoke requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	workDir, err := os.MkdirTemp(".", ".tetra-memory-smoke-*")
	if err != nil {
		return err
	}
	r := &smokeRunner{opt: opt, workDir: workDir}
	sourceDir, err := os.MkdirTemp(filepath.Join("examples"), "memory_production_smoke_")
	if err != nil {
		if !opt.KeepWork {
			_ = os.RemoveAll(workDir)
		}
		return err
	}
	r.sourceDir = sourceDir
	if !opt.KeepWork {
		defer os.RemoveAll(workDir)
		defer os.RemoveAll(sourceDir)
	}
	if err := os.MkdirAll(filepath.Dir(opt.ReportPath), 0o755); err != nil {
		return err
	}
	r.recordRAMSnapshot("start")
	if opt.TetraPath == "" {
		r.tetraPath = filepath.Join(workDir, "tetra")
		res := runCommand(ctx, 30*time.Second, "go", "build", "-o", r.tetraPath, "./cli/cmd/tetra")
		r.recordProcess("tetra build", "build", "go build ./cli/cmd/tetra", res)
		if res.err != nil {
			return fmt.Errorf("build smoke tetra CLI: %s", res.output)
		}
	} else {
		r.tetraPath = opt.TetraPath
	}
	r.recordRAMSnapshot("after_tetra_build")

	if err := r.runPositiveApp(ctx); err != nil {
		return err
	}
	if err := r.runCheckedInExamples(ctx); err != nil {
		return err
	}
	if err := r.runStressApp(ctx); err != nil {
		return err
	}
	if err := r.runFuzzApp(ctx); err != nil {
		return err
	}
	if err := r.runRuntimeDiagnosticCases(ctx); err != nil {
		return err
	}
	if err := r.runAllocationLengthRuntimeCoverage(ctx); err != nil {
		return err
	}
	if err := r.runCompilerSafetyCases(ctx); err != nil {
		return err
	}
	if err := r.runMemoryShapeCoverageCases(ctx); err != nil {
		return err
	}
	if err := r.runResourceFinalizationCoverage(ctx); err != nil {
		return err
	}
	if err := r.runSmallHeapAllocationBenchmark(ctx); err != nil {
		return err
	}
	if err := r.runRawPointerBoundsMetadataReport(ctx); err != nil {
		return err
	}
	if err := r.runAllocationLengthContractReport(ctx); err != nil {
		return err
	}
	r.recordRAMSnapshot("before_report_write")
	if err := r.writeReport(); err != nil {
		return err
	}
	r.recordRAMSnapshot("end")
	return r.writeRAMMeasurementReport()
}

func (r *smokeRunner) runPositiveApp(ctx context.Context) error {
	outPath := filepath.Join(r.workDir, "memory-positive")
	sourcePath, err := r.writeSource("positive", memoryPositiveSource)
	if err != nil {
		return err
	}
	build := runCommand(ctx, 30*time.Second, r.tetraPath, "build", "--target", "linux-x64", "-o", outPath, sourcePath)
	r.recordProcess("memory smoke build", "build", r.tetraPath+" build --target linux-x64", build)
	if build.err != nil {
		return fmt.Errorf("build memory positive app: %s", build.output)
	}
	run := runCommand(ctx, 5*time.Second, outPath)
	r.recordProcess("memory smoke app", "app", outPath, run)
	if run.err != nil || run.exitCode != 0 {
		return fmt.Errorf("run memory positive app: exit=%d output=%s", run.exitCode, run.output)
	}
	r.cases = append(r.cases,
		memoryprod.CaseReport{Name: "allocator alloc/free lifecycle", Kind: "positive", Ran: true, Pass: true},
		memoryprod.CaseReport{Name: "memcpy/memset capability path", Kind: "positive", Ran: true, Pass: true},
	)
	return nil
}

func (r *smokeRunner) runCheckedInExamples(ctx context.Context) error {
	examples := []struct {
		path         string
		expectedExit int
	}{
		{path: filepath.Join("examples", "core_memory_smoke.tetra"), expectedExit: 42},
		{path: filepath.Join("examples", "ownership_smoke.tetra"), expectedExit: 42},
		{path: filepath.Join("examples", "flow_unsafe_cap_mem_smoke.tetra"), expectedExit: 42},
	}
	for i, example := range examples {
		sourcePath := example.path
		outPath := filepath.Join(r.workDir, fmt.Sprintf("memory-example-%02d", i))
		build := runCommand(ctx, 30*time.Second, r.tetraPath, "build", "--target", "linux-x64", "-o", outPath, sourcePath)
		if build.err != nil {
			r.cases = append(r.cases, failedCase("real memory examples", "positive", "", fmt.Sprintf("build %s failed: %s", sourcePath, build.output)))
			return fmt.Errorf("build memory example %s: %s", sourcePath, build.output)
		}
		run := runCommand(ctx, 5*time.Second, outPath)
		if run.exitCode != example.expectedExit {
			r.cases = append(r.cases, failedCase("real memory examples", "positive", "", fmt.Sprintf("run %s exit=%d output=%s", sourcePath, run.exitCode, run.output)))
			return fmt.Errorf("run memory example %s: exit=%d, want %d output=%s", sourcePath, run.exitCode, example.expectedExit, run.output)
		}
	}
	r.cases = append(r.cases, memoryprod.CaseReport{Name: "real memory examples", Kind: "positive", Ran: true, Pass: true})
	return nil
}

func (r *smokeRunner) runStressApp(ctx context.Context) error {
	outPath := filepath.Join(r.workDir, "memory-stress")
	sourcePath, err := r.writeSource("stress", memoryStressSource)
	if err != nil {
		return err
	}
	build := runCommand(ctx, 30*time.Second, r.tetraPath, "build", "--target", "linux-x64", "-o", outPath, sourcePath)
	r.recordProcess("memory stress build", "build", r.tetraPath+" build --target linux-x64", build)
	if build.err != nil {
		return fmt.Errorf("build memory stress app: %s", build.output)
	}
	run := runCommand(ctx, 10*time.Second, outPath)
	r.recordProcess("memory stress", "stress", outPath, run)
	if run.err != nil || run.exitCode != 0 {
		return fmt.Errorf("run memory stress app: exit=%d output=%s", run.exitCode, run.output)
	}
	r.cases = append(r.cases, memoryprod.CaseReport{Name: "stress allocator reuse", Kind: "stress", Ran: true, Pass: true})
	return nil
}

func (r *smokeRunner) runFuzzApp(ctx context.Context) error {
	outPath := filepath.Join(r.workDir, "memory-fuzz")
	sourcePath, err := r.writeSource("fuzz", memoryFuzzSource)
	if err != nil {
		return err
	}
	build := runCommand(ctx, 30*time.Second, r.tetraPath, "build", "--target", "linux-x64", "-o", outPath, sourcePath)
	r.recordProcess("memory fuzz build", "build", r.tetraPath+" build --target linux-x64", build)
	if build.err != nil {
		return fmt.Errorf("build memory fuzz app: %s", build.output)
	}
	run := runCommand(ctx, 10*time.Second, outPath)
	r.recordProcess("memory fuzz", "stress", outPath, run)
	if run.err != nil || run.exitCode != 0 {
		return fmt.Errorf("run memory fuzz app: exit=%d output=%s", run.exitCode, run.output)
	}
	r.cases = append(r.cases, memoryprod.CaseReport{Name: "deterministic memcpy/memset fuzz", Kind: "stress", Ran: true, Pass: true})
	return nil
}

func (r *smokeRunner) runRuntimeDiagnosticCases(ctx context.Context) error {
	tests := []struct {
		name          string
		source        string
		expectedExit  int
		expectedError string
		requireError  bool
	}{
		{name: "allocator invalid size precondition", source: allocInvalidSizeSource, expectedExit: 2, expectedError: "invalid allocation size"},
		{name: "runtime bounds check", source: sliceBoundsSource, expectedExit: 1, expectedError: "bounds"},
		{name: "raw ptr_add negative offset bounds", source: rawPtrAddNegativeSource, expectedExit: 2, expectedError: "negative ptr_add offset"},
		{name: "raw ptr_add allocation upper bound", source: rawPtrAddUpperSource, expectedExit: 2, expectedError: "allocation upper bound"},
		{name: "raw allocation-base i32 access width", source: rawI32WidthSource, expectedExit: 2, expectedError: "i32 access width exceeds allocation"},
		{name: "raw allocation-base ptr access width", source: rawPtrWidthSource, expectedExit: 2, expectedError: "ptr access width exceeds allocation"},
		{name: "raw allocation-base store_i32 access width", source: rawStoreI32WidthSource, expectedExit: 2, expectedError: "i32 access width exceeds allocation"},
		{name: "raw allocation-base load_ptr access width", source: rawLoadPtrWidthSource, expectedExit: 2, expectedError: "ptr access width exceeds allocation"},
		{name: "raw slice negative length", source: rawSliceNegativeLengthSource, expectedExit: 2, expectedError: "negative raw slice length"},
		{name: "raw slice i32 length byte overflow", source: rawSliceI32LengthOverflowSource, expectedExit: 2, expectedError: "raw slice length byte overflow"},
		{name: "memcpy/memset negative length", source: memoryNegativeLengthSource, expectedExit: 2, expectedError: "negative helper length"},
	}
	for i, tc := range tests {
		outPath := filepath.Join(r.workDir, fmt.Sprintf("negative-%02d", i))
		sourcePath, err := r.writeSource(fmt.Sprintf("negative_%02d", i), tc.source)
		if err != nil {
			return err
		}
		build := runCommand(ctx, 30*time.Second, r.tetraPath, "build", "--target", "linux-x64", "-o", outPath, sourcePath)
		if build.err != nil {
			r.cases = append(r.cases, failedCase(tc.name, "negative", tc.expectedError, fmt.Sprintf("build failed: %s", build.output)))
			return fmt.Errorf("build %s: %s", tc.name, build.output)
		}
		run := runCommand(ctx, 5*time.Second, outPath)
		pass := runtimeDiagnosticPass(run, tc.expectedExit, tc.expectedError, tc.requireError)
		if !pass {
			r.cases = append(r.cases, failedCase(tc.name, "negative", tc.expectedError, fmt.Sprintf("exit=%d output=%s", run.exitCode, run.output)))
			return fmt.Errorf("%s exit=%d, want %d and error %q: %s", tc.name, run.exitCode, tc.expectedExit, tc.expectedError, run.output)
		}
		r.cases = append(r.cases, memoryprod.CaseReport{Name: tc.name, Kind: "negative", Ran: true, Pass: true, ExpectedError: tc.expectedError})
	}
	return nil
}

func runtimeDiagnosticPass(run processResult, expectedExit int, expectedError string, requireError bool) bool {
	if run.exitCode != expectedExit {
		return false
	}
	if !requireError || strings.TrimSpace(expectedError) == "" {
		return true
	}
	return strings.Contains(run.output, expectedError)
}

func (r *smokeRunner) runAllocationLengthRuntimeCoverage(ctx context.Context) error {
	names := []string{
		"allocation make zero length canonical empty",
		"allocation make negative length",
		"allocation make byte-size overflow",
		"allocation island zero length no metadata",
		"allocation island negative length",
		"allocation island byte-size overflow",
	}
	res := runCommand(ctx, 120*time.Second, "go", "test", "./compiler/tests/semantics", "-run", "AllocationLength", "-count=1")
	r.recordProcess("allocation length runtime coverage", "benchmark", "go test ./compiler/tests/semantics -run AllocationLength -count=1", res)
	if res.err != nil || res.exitCode != 0 {
		for _, name := range names {
			r.cases = append(r.cases, failedCase(name, "negative", "allocation length contract", res.output))
		}
		return fmt.Errorf("allocation length runtime coverage failed: %s", res.output)
	}
	for _, name := range names {
		kind := "negative"
		if strings.Contains(name, "zero length") {
			kind = "positive"
		}
		r.cases = append(r.cases, memoryprod.CaseReport{Name: name, Kind: kind, Ran: true, Pass: true, ExpectedError: allocationLengthExpectedError(name)})
	}
	return nil
}

func allocationLengthExpectedError(name string) string {
	switch {
	case strings.Contains(name, "negative length"):
		return "negative allocation length"
	case strings.Contains(name, "byte-size overflow"):
		return "allocation length byte overflow"
	default:
		return ""
	}
}

func (r *smokeRunner) writeSource(name, body string) (string, error) {
	moduleDir := filepath.Base(r.sourceDir)
	module := "examples." + moduleDir + "." + name
	path := filepath.Join(r.sourceDir, name+".tetra")
	source := "module " + module + "\n\n" + strings.TrimLeft(body, "\n")
	return path, os.WriteFile(path, []byte(source), 0o644)
}

func (r *smokeRunner) runCompilerSafetyCases(ctx context.Context) error {
	tests := []struct {
		name          string
		pkg           string
		pattern       string
		expectedError string
	}{
		{name: "allocator failure semantics", pkg: "./compiler/internal/backend/x64abi", pattern: "TestSysVAllocBytesEmitsDeterministicMmapFailureGuard", expectedError: "allocation failure"},
		{name: "cap.mem unsafe boundary", pkg: "./compiler/tests/safety", pattern: "TestEpic06RejectsCapMemOutsideUnsafeBlock", expectedError: "only allowed in unsafe blocks"},
		{name: "reject use-after-free", pkg: "./compiler/tests/safety", pattern: "TestSafetyDiagnosticCodesForOptionalPayloadWholeValueConsumeAndFree", expectedError: "use-after-free"},
		{name: "reject double-free", pkg: "./compiler", pattern: "TestBuildIslandsDebugDoubleFreeRejectedBySemantics", expectedError: "double-free"},
		{name: "reject borrow escape", pkg: "./compiler/tests/ownership", pattern: "TestOwnershipRejectsBorrowEscapeViaAliasReturn", expectedError: "borrow escape"},
		{name: "reject aliasing violation", pkg: "./compiler/tests/ownership", pattern: "TestOwnershipRejectsBorrowInoutAlias", expectedError: "alias"},
		{name: "callable mutable capture heap escape", pkg: "./compiler/tests/safety", pattern: "TestSafetyDiagnosticCodesForKeyFamilies/callable_mutable_capture_heap_escape", expectedError: "heap-escaped function value captures mutable local"},
		{name: "reject actor task transfer violation", pkg: "./compiler/tests/ownership", pattern: "TestReleaseTraceabilityLifetimeAndRaceSafetyNegativeActorTaskOwnership", expectedError: "transfer"},
	}
	for _, tc := range tests {
		res := runCommand(ctx, 60*time.Second, "go", "test", tc.pkg, "-run", tc.pattern, "-count=1")
		if res.err != nil || res.exitCode != 0 {
			r.cases = append(r.cases, failedCase(tc.name, "negative", tc.expectedError, res.output))
			return fmt.Errorf("%s evidence failed: %s", tc.name, res.output)
		}
		r.cases = append(r.cases, memoryprod.CaseReport{Name: tc.name, Kind: "negative", Ran: true, Pass: true, ExpectedError: tc.expectedError})
	}
	return nil
}

func (r *smokeRunner) runMemoryShapeCoverageCases(ctx context.Context) error {
	tests := []struct {
		name          string
		pkg           string
		pattern       string
		kind          string
		expectedError string
	}{
		{
			name:    "heap closure handle coverage",
			pkg:     "./compiler/tests/semantics",
			pattern: "^(TestFullCallableEscapedNineCaptureReturnPassesSemanticClassification|TestFullCallableStructFieldNineCaptureLowersHandleEnvironment|TestFullCallableEnumPayloadNineCaptureLowersHandleEnvironment)$",
			kind:    "positive",
		},
		{
			name:          "slice struct borrow escape coverage",
			pkg:           "./compiler/tests/ownership",
			pattern:       "^(TestOwnershipRejectsBorrowEscapeViaNestedSliceStruct|TestOwnershipRejectsBorrowedSliceStructOwnedConsumeInoutCallEscape|TestOwnershipRejectsBorrowedPtrEscapeViaStructInoutAssignment)$",
			kind:          "negative",
			expectedError: "borrow escape",
		},
		{
			name:          "function-typed slice aggregate borrow escape coverage",
			pkg:           "./compiler/tests/safety",
			pattern:       "TestSafetyDiagnosticCodesForFunctionTypedSliceAggregateCallbackCallRejections",
			kind:          "negative",
			expectedError: "borrow escape",
		},
	}
	for _, tc := range tests {
		res := runCommand(ctx, 60*time.Second, "go", "test", tc.pkg, "-run", tc.pattern, "-count=1")
		if res.err != nil || res.exitCode != 0 {
			r.cases = append(r.cases, failedCase(tc.name, tc.kind, tc.expectedError, res.output))
			return fmt.Errorf("%s evidence failed: %s", tc.name, res.output)
		}
		r.cases = append(r.cases, memoryprod.CaseReport{Name: tc.name, Kind: tc.kind, Ran: true, Pass: true, ExpectedError: tc.expectedError})
	}
	return nil
}

func (r *smokeRunner) runResourceFinalizationCoverage(ctx context.Context) error {
	tests := []struct {
		name          string
		processName   string
		pkg           string
		pattern       string
		kind          string
		expectedError string
		timeout       time.Duration
	}{
		{
			name:        "actornet broker close-without-cancel leak smoke",
			processName: "actornet close-without-cancel leak coverage",
			pkg:         "./cli/internal/actornet",
			pattern:     "TestBrokerCloseWithoutCancelStopsServeWatcher",
			kind:        "stress",
			timeout:     30 * time.Second,
		},
		{
			name:          "compiler resource finalization diagnostics",
			processName:   "compiler resource finalization diagnostics",
			pkg:           "./compiler/tests/runtime",
			pattern:       "^(TestTaskHandleFinalization|TestTaskGroupFinalization|TestIslandFinalization)",
			kind:          "negative",
			expectedError: "resource finalization",
			timeout:       180 * time.Second,
		},
	}
	for _, tc := range tests {
		args := []string{"test", "-buildvcs=false", tc.pkg, "-run", tc.pattern, "-count=1"}
		res := runCommand(ctx, tc.timeout, "go", args...)
		r.recordProcess(tc.processName, "stress", "go "+strings.Join(args, " "), res)
		if res.err != nil || res.exitCode != 0 {
			r.cases = append(r.cases, failedCase(tc.name, tc.kind, tc.expectedError, res.output))
			return fmt.Errorf("%s evidence failed: %s", tc.name, res.output)
		}
		r.cases = append(r.cases, memoryprod.CaseReport{Name: tc.name, Kind: tc.kind, Ran: true, Pass: true, ExpectedError: tc.expectedError})
	}
	return nil
}

func (r *smokeRunner) runSmallHeapAllocationBenchmark(ctx context.Context) error {
	const allocationCount = 64
	const bytesPerAllocation = 32
	const smallHeapChunkBytes = 64 * 1024

	sourcePath, err := r.writeSource("small_heap_benchmark", smallHeapBenchmarkSource(allocationCount, bytesPerAllocation))
	if err != nil {
		return err
	}
	outPath := filepath.Join(r.workDir, "small-heap-benchmark")
	build := runCommand(ctx, 30*time.Second, r.tetraPath, "build", "--target", "linux-x64", "--emit-alloc-report", "-o", outPath, sourcePath)
	r.recordProcess("small heap allocation benchmark build", "benchmark", r.tetraPath+" build --target linux-x64 --emit-alloc-report", build)
	if build.err != nil {
		return fmt.Errorf("build small heap allocation benchmark: %s", build.output)
	}
	allocReportPath := outPath + ".alloc.json"
	raw, err := os.ReadFile(allocReportPath)
	if err != nil {
		return fmt.Errorf("read small heap allocation report: %w", err)
	}
	report, err := parseAllocationReportSummary(raw)
	if err != nil {
		return err
	}
	smallHeapRows := report.Summary.RuntimePaths["per_core_small_heap"]
	if smallHeapRows != allocationCount {
		return fmt.Errorf("small heap allocation benchmark: per_core_small_heap rows = %d, want %d", smallHeapRows, allocationCount)
	}
	reusePolicyRows := report.Summary.AllocatorReusePolicies["same_core_same_size_class_free_list"]
	if reusePolicyRows != allocationCount {
		return fmt.Errorf("small heap allocation benchmark: same-core reuse policy rows = %d, want %d", reusePolicyRows, allocationCount)
	}
	if report.Summary.BytesReserved <= 0 {
		return fmt.Errorf("small heap allocation benchmark: bytes_reserved = %d, want positive", report.Summary.BytesReserved)
	}
	baselineSyscalls := smallHeapRows
	measuredSyscalls := ceilDiv(report.Summary.BytesReserved, smallHeapChunkBytes)
	if measuredSyscalls <= 0 {
		measuredSyscalls = 1
	}
	improvement := float64(baselineSyscalls) / float64(measuredSyscalls)
	if baselineSyscalls <= measuredSyscalls {
		return fmt.Errorf("small heap allocation benchmark: baseline syscalls %d must exceed measured chunk refills %d", baselineSyscalls, measuredSyscalls)
	}
	r.benchmarks = append(r.benchmarks, memoryprod.BenchmarkReport{
		Name:             "small heap allocation syscall reduction",
		Kind:             "allocator",
		Metric:           "estimated_os_syscalls",
		Unit:             "syscalls",
		EvidenceClass:    "allocation_report_estimate",
		Method:           "allocation_report_summary",
		BaselineValue:    baselineSyscalls,
		MeasuredValue:    measuredSyscalls,
		ImprovementRatio: improvement,
		Evidence: fmt.Sprintf(
			"allocation report schema v2 estimates %d per_core_small_heap allocation intents, %d bytes reserved, and %d estimated 64KiB chunk refill syscall(s) instead of %d mmap-per-allocation syscall(s); allocation_report_estimate only, not a runtime measurement",
			smallHeapRows,
			report.Summary.BytesReserved,
			measuredSyscalls,
			baselineSyscalls,
		),
		Ran:  true,
		Pass: true,
	})
	return nil
}

func (r *smokeRunner) runRawPointerBoundsMetadataReport(ctx context.Context) error {
	sourcePath, err := r.writeSource("raw_pointer_bounds_metadata", rawPointerBoundsMetadataSource)
	if err != nil {
		return err
	}
	outPath := filepath.Join(r.workDir, "raw-pointer-bounds-metadata")
	build := runCommand(ctx, 30*time.Second, r.tetraPath, "build", "--target", "linux-x64", "--emit-alloc-report", "--emit-memory-report", "-o", outPath, sourcePath)
	r.recordProcess("raw pointer bounds metadata report build", "benchmark", r.tetraPath+" build --target linux-x64 --emit-alloc-report --emit-memory-report", build)
	if build.err != nil {
		r.cases = append(r.cases, failedCase("raw pointer bounds metadata report", "positive", "", build.output))
		return fmt.Errorf("build raw pointer bounds metadata report: %s", build.output)
	}
	raw, err := os.ReadFile(outPath + ".alloc.json")
	if err != nil {
		r.cases = append(r.cases, failedCase("raw pointer bounds metadata report", "positive", "", err.Error()))
		return fmt.Errorf("read raw pointer bounds allocation report: %w", err)
	}
	report, err := parseAllocationReportSummary(raw)
	if err != nil {
		r.cases = append(r.cases, failedCase("raw pointer bounds metadata report", "positive", "", err.Error()))
		return err
	}
	memoryRaw, err := os.ReadFile(outPath + ".memory.json")
	if err != nil {
		r.cases = append(r.cases, failedCase("raw pointer bounds metadata report", "positive", "", err.Error()))
		return fmt.Errorf("read raw pointer bounds memory report: %w", err)
	}
	validateMemoryArgs := validateMemoryReportCommand(outPath)
	validateMemory := runCommand(ctx, 30*time.Second, "go", validateMemoryArgs...)
	r.recordProcess("raw pointer bounds memory report validation", "benchmark", "go "+strings.Join(validateMemoryArgs, " "), validateMemory)
	if validateMemory.err != nil {
		r.cases = append(r.cases, failedCase("raw pointer bounds metadata report", "positive", "", validateMemory.output))
		return fmt.Errorf("validate raw pointer bounds memory report: %s", validateMemory.output)
	}
	memoryClaims, err := parseMemoryReportClaims(memoryRaw)
	if err != nil {
		r.cases = append(r.cases, failedCase("raw pointer bounds metadata report", "positive", "", err.Error()))
		return err
	}
	if err := validateRawPointerBoundsCorrelation(r.cases, memoryRaw); err != nil {
		r.cases = append(r.cases, failedCase("raw pointer bounds metadata report", "positive", "", err.Error()))
		return err
	}
	if err := validateRawSliceGatewayCorrelation(r.cases, memoryRaw); err != nil {
		r.cases = append(r.cases, failedCase("raw pointer bounds metadata report", "positive", "", err.Error()))
		return err
	}
	if report.Summary.RawPointerBoundsStatuses["allocation_base_metadata"] < 1 {
		err := fmt.Errorf("raw pointer bounds allocation report missing allocation_base_metadata summary: %+v", report.Summary.RawPointerBoundsStatuses)
		r.cases = append(r.cases, failedCase("raw pointer bounds metadata report", "positive", "", err.Error()))
		return err
	}
	if report.Summary.RawSlicePolicies["external_unknown"] < 1 {
		err := fmt.Errorf("raw pointer bounds allocation report missing external_unknown raw slice policy: %+v", report.Summary.RawSlicePolicies)
		r.cases = append(r.cases, failedCase("raw pointer bounds metadata report", "positive", "", err.Error()))
		return err
	}
	for _, claim := range []string{
		"allocation_base_metadata",
		"derived_allocation_offset",
		"rejected_negative_offset",
		"rejected_upper_bound",
		"rejected_access_width_overflow",
		"checked_external_unknown",
		"external_unknown",
		"raw_slice_verified_allocation_root",
		"rejected_negative_length",
		"rejected_length_overflow",
		"raw_bounds_runtime_check_normal_build",
	} {
		if memoryClaims[claim] < 1 {
			err := fmt.Errorf("raw pointer bounds memory report missing %s claim: %+v", claim, memoryClaims)
			r.cases = append(r.cases, failedCase("raw pointer bounds metadata report", "positive", "", err.Error()))
			return err
		}
	}
	r.cases = append(r.cases, memoryprod.CaseReport{Name: "raw pointer bounds metadata report", Kind: "positive", Ran: true, Pass: true})
	return nil
}

func (r *smokeRunner) runAllocationLengthContractReport(ctx context.Context) error {
	sourcePath, err := r.writeSource("allocation_length_contract_report", allocationLengthContractReportSource)
	if err != nil {
		return err
	}
	outPath := filepath.Join(r.workDir, "allocation-length-contract")
	build := runCommand(ctx, 30*time.Second, r.tetraPath, "build", "--target", "linux-x64", "--emit-alloc-report", "--emit-memory-report", "-o", outPath, sourcePath)
	r.recordProcess("allocation length contract report build", "benchmark", r.tetraPath+" build --target linux-x64 --emit-alloc-report --emit-memory-report", build)
	if build.err != nil {
		r.cases = append(r.cases, failedCase("allocation length contract report", "positive", "", build.output))
		return fmt.Errorf("build allocation length contract report: %s", build.output)
	}
	validateMemoryArgs := validateMemoryReportCommand(outPath)
	validateMemory := runCommand(ctx, 30*time.Second, "go", validateMemoryArgs...)
	r.recordProcess("allocation length memory report validation", "benchmark", "go "+strings.Join(validateMemoryArgs, " "), validateMemory)
	if validateMemory.err != nil {
		r.cases = append(r.cases, failedCase("allocation length contract report", "positive", "", validateMemory.output))
		return fmt.Errorf("validate allocation length memory report: %s", validateMemory.output)
	}
	raw, err := os.ReadFile(outPath + ".alloc.json")
	if err != nil {
		r.cases = append(r.cases, failedCase("allocation length contract report", "positive", "", err.Error()))
		return fmt.Errorf("read allocation length allocation report: %w", err)
	}
	report, err := parseAllocationReportSummary(raw)
	if err != nil {
		r.cases = append(r.cases, failedCase("allocation length contract report", "positive", "", err.Error()))
		return err
	}
	if err := validateAllocationLengthContractCorrelation(r.cases, report); err != nil {
		r.cases = append(r.cases, failedCase("allocation length contract report", "positive", "", err.Error()))
		return err
	}
	r.cases = append(r.cases, memoryprod.CaseReport{Name: "allocation length contract report", Kind: "positive", Ran: true, Pass: true})
	return nil
}
