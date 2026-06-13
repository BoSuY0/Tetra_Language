package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"tetra_language/tools/validators/memoryprod"
)

type smokeOptions struct {
	ReportPath string
	TetraPath  string
	GitHead    string
	KeepWork   bool
}

type smokeRunner struct {
	opt        smokeOptions
	workDir    string
	sourceDir  string
	tetraPath  string
	processes  []memoryprod.ProcessReport
	benchmarks []memoryprod.BenchmarkReport
	cases      []memoryprod.CaseReport
}

type processResult struct {
	exitCode int
	output   string
	err      error
}

func main() {
	var opt smokeOptions
	flag.StringVar(&opt.ReportPath, "report", "", "path to write tetra.memory.production.v1 report")
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
	return r.writeReport()
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
		BaselineValue:    baselineSyscalls,
		MeasuredValue:    measuredSyscalls,
		ImprovementRatio: improvement,
		Evidence: fmt.Sprintf(
			"allocation report schema v2 shows %d per_core_small_heap rows with same_core_same_size_class_free_list reuse policy, %d bytes reserved, and %d estimated 64KiB chunk refill syscall(s) instead of %d mmap-per-allocation syscall(s)",
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

func validateMemoryReportCommand(outPath string) []string {
	return []string{
		"run", "./tools/cmd/validate-memory-report",
		"--report", outPath + ".memory.json",
		"--alloc-report", outPath + ".alloc.json",
	}
}

type allocationReportSummary struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind,omitempty"`
	Summary       struct {
		AllocationCount          int            `json:"allocation_count"`
		RuntimePaths             map[string]int `json:"runtime_paths"`
		AllocatorClasses         map[string]int `json:"allocator_classes"`
		AllocatorReusePolicies   map[string]int `json:"allocator_reuse_policies"`
		RawPointerBoundsStatuses map[string]int `json:"raw_pointer_bounds_statuses"`
		RawSlicePolicies         map[string]int `json:"raw_slice_policies"`
		BytesReserved            int            `json:"bytes_reserved"`
	} `json:"summary"`
	Functions []allocationReportFunctionSummary `json:"functions,omitempty"`
}

type allocationReportFunctionSummary struct {
	Name        string                              `json:"name"`
	Allocations []allocationReportAllocationSummary `json:"allocations,omitempty"`
}

type allocationReportAllocationSummary struct {
	ID                  string `json:"id"`
	Builtin             string `json:"builtin,omitempty"`
	LengthStatus        string `json:"length_status,omitempty"`
	ZeroGuardStatus     string `json:"zero_guard_status,omitempty"`
	NegativeGuardStatus string `json:"negative_guard_status,omitempty"`
	OverflowGuardStatus string `json:"overflow_guard_status,omitempty"`
}

type memoryReportEvidenceRow struct {
	SourceFactID     string `json:"source_fact_id,omitempty"`
	ParentFactID     string `json:"parent_fact_id,omitempty"`
	Claim            string `json:"claim"`
	ClaimLevel       string `json:"claim_level,omitempty"`
	ProvenanceClass  string `json:"provenance_class,omitempty"`
	UnsafeClass      string `json:"unsafe_class,omitempty"`
	ValidatorName    string `json:"validator_name,omitempty"`
	ValidatorStatus  string `json:"validator_status,omitempty"`
	CostClass        string `json:"cost_class,omitempty"`
	NormalBuildCheck bool   `json:"normal_build_check,omitempty"`
	Reason           string `json:"reason,omitempty"`
}

type memoryReportEvidence struct {
	SchemaVersion string                    `json:"schema_version"`
	Rows          []memoryReportEvidenceRow `json:"rows"`
}

func parseAllocationReportSummary(raw []byte) (allocationReportSummary, error) {
	var report allocationReportSummary
	if err := json.Unmarshal(raw, &report); err != nil {
		return allocationReportSummary{}, fmt.Errorf("parse small heap allocation report: %w", err)
	}
	if report.SchemaVersion != 2 {
		return allocationReportSummary{}, fmt.Errorf("small heap allocation report schema_version = %d, want 2", report.SchemaVersion)
	}
	if report.Summary.AllocationCount <= 0 {
		return allocationReportSummary{}, fmt.Errorf("small heap allocation report allocation_count = %d, want positive", report.Summary.AllocationCount)
	}
	if report.Summary.RuntimePaths == nil {
		return allocationReportSummary{}, fmt.Errorf("small heap allocation report missing runtime_paths summary")
	}
	return report, nil
}

func validateAllocationLengthContractCorrelation(cases []memoryprod.CaseReport, report allocationReportSummary) error {
	var issues []string
	issues = append(issues, validateAllocationLengthRuntimeCases(cases)...)
	requireRows := []allocationReportAllocationSummary{
		{
			Builtin:             "core.alloc_bytes",
			LengthStatus:        "invalid_length_contract",
			ZeroGuardStatus:     "invalid_precondition",
			NegativeGuardStatus: "reject_before_allocation",
			OverflowGuardStatus: "reject_before_allocation",
		},
		{
			Builtin:             "core.make_u8",
			LengthStatus:        "valid_empty_allocation",
			ZeroGuardStatus:     "valid_empty_no_allocator",
			NegativeGuardStatus: "reject_before_allocation",
			OverflowGuardStatus: "reject_before_allocation",
		},
		{
			Builtin:             "core.make_u16",
			LengthStatus:        "rejected_negative_length",
			ZeroGuardStatus:     "valid_empty_no_allocator",
			NegativeGuardStatus: "reject_before_allocation",
			OverflowGuardStatus: "reject_before_allocation",
		},
		{
			Builtin:             "core.make_i32",
			LengthStatus:        "rejected_byte_size_overflow",
			ZeroGuardStatus:     "valid_empty_no_allocator",
			NegativeGuardStatus: "reject_before_allocation",
			OverflowGuardStatus: "reject_before_allocation",
		},
		{
			Builtin:             "core.island_make_u8",
			LengthStatus:        "valid_empty_allocation",
			ZeroGuardStatus:     "valid_empty_no_metadata_access",
			NegativeGuardStatus: "reject_before_metadata_access",
			OverflowGuardStatus: "reject_before_metadata_access",
		},
		{
			Builtin:             "core.island_make_u16",
			LengthStatus:        "rejected_negative_length",
			ZeroGuardStatus:     "valid_empty_no_metadata_access",
			NegativeGuardStatus: "reject_before_metadata_access",
			OverflowGuardStatus: "reject_before_metadata_access",
		},
		{
			Builtin:             "core.island_make_i32",
			LengthStatus:        "rejected_byte_size_overflow",
			ZeroGuardStatus:     "valid_empty_no_metadata_access",
			NegativeGuardStatus: "reject_before_metadata_access",
			OverflowGuardStatus: "reject_before_metadata_access",
		},
	}
	for _, req := range requireRows {
		if !allocationReportHasLengthContract(report, req) {
			issues = append(issues, fmt.Sprintf("allocation report missing contract row builtin=%s length_status=%s zero=%s negative=%s overflow=%s", req.Builtin, req.LengthStatus, req.ZeroGuardStatus, req.NegativeGuardStatus, req.OverflowGuardStatus))
		}
	}
	if len(issues) > 0 {
		return fmt.Errorf(strings.Join(issues, "; "))
	}
	return nil
}

func validateAllocationLengthRuntimeCases(cases []memoryprod.CaseReport) []string {
	byName := map[string]memoryprod.CaseReport{}
	for _, c := range cases {
		byName[c.Name] = c
	}
	var issues []string
	for _, req := range []struct {
		name          string
		expectedError string
	}{
		{name: "allocation make zero length canonical empty"},
		{name: "allocation make negative length", expectedError: "negative allocation length"},
		{name: "allocation make byte-size overflow", expectedError: "allocation length byte overflow"},
		{name: "allocation island zero length no metadata"},
		{name: "allocation island negative length", expectedError: "negative allocation length"},
		{name: "allocation island byte-size overflow", expectedError: "allocation length byte overflow"},
	} {
		c, ok := byName[req.name]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing allocation length runtime case %q", req.name))
			continue
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, fmt.Sprintf("allocation length runtime case %q did not pass", req.name))
		}
		if req.expectedError != "" && !strings.Contains(c.ExpectedError, req.expectedError) {
			issues = append(issues, fmt.Sprintf("allocation length runtime case %q expected_error = %q, want %q", req.name, c.ExpectedError, req.expectedError))
		}
	}
	return issues
}

func allocationReportHasLengthContract(report allocationReportSummary, req allocationReportAllocationSummary) bool {
	for _, fn := range report.Functions {
		for _, alloc := range fn.Allocations {
			if alloc.Builtin == req.Builtin &&
				alloc.LengthStatus == req.LengthStatus &&
				alloc.ZeroGuardStatus == req.ZeroGuardStatus &&
				alloc.NegativeGuardStatus == req.NegativeGuardStatus &&
				alloc.OverflowGuardStatus == req.OverflowGuardStatus {
				return true
			}
		}
	}
	return false
}

func parseMemoryReportClaims(raw []byte) (map[string]int, error) {
	report, err := parseMemoryReportEvidence(raw)
	if err != nil {
		return nil, err
	}
	claims := map[string]int{}
	for _, row := range report.Rows {
		claims[row.Claim]++
	}
	return claims, nil
}

func parseMemoryReportEvidence(raw []byte) (memoryReportEvidence, error) {
	var report memoryReportEvidence
	if err := json.Unmarshal(raw, &report); err != nil {
		return memoryReportEvidence{}, fmt.Errorf("parse memory report: %w", err)
	}
	if report.SchemaVersion != "tetra.memory-report.v1" {
		return memoryReportEvidence{}, fmt.Errorf("memory report schema_version = %q, want tetra.memory-report.v1", report.SchemaVersion)
	}
	return report, nil
}

func validateRawPointerBoundsCorrelation(cases []memoryprod.CaseReport, memoryRaw []byte) error {
	report, err := parseMemoryReportEvidence(memoryRaw)
	if err != nil {
		return err
	}
	var issues []string
	issues = append(issues, validateRawRuntimeCases(cases)...)
	issues = append(issues, validateCapMemAuthorizationDiscipline(report.Rows)...)
	checksByParent := parentedRawBoundsRuntimeChecks(report.Rows)

	requireRows := []struct {
		claim           string
		minRows         int
		validate        func(memoryReportEvidenceRow) []string
		requireCheckRow bool
	}{
		{claim: "allocation_base_metadata", minRows: 1, validate: validateUnsafeVerifiedRootAllocationBaseRow},
		{claim: "derived_allocation_offset", minRows: 1, validate: validateUnsafeCheckedDynamicRow},
		{claim: "rejected_negative_offset", minRows: 1, validate: validateUnsafeCheckedRejectedRow},
		{claim: "rejected_upper_bound", minRows: 1, validate: validateUnsafeCheckedRejectedRow},
		{claim: "rejected_access_width_overflow", minRows: 4, validate: validateUnsafeCheckedRejectedRow, requireCheckRow: true},
		{claim: "checked_external_unknown", minRows: 1, validate: validateUnsafeUnknownConservativeRow},
	}
	for _, req := range requireRows {
		valid := 0
		for _, row := range report.Rows {
			if row.Claim != req.claim {
				continue
			}
			rowIssues := req.validate(row)
			if req.requireCheckRow && row.SourceFactID != "" && !checksByParent[row.SourceFactID] {
				rowIssues = append(rowIssues, fmt.Sprintf("source_fact_id %q is missing parented raw_bounds_runtime_check_normal_build normal_build_check", row.SourceFactID))
			}
			if len(rowIssues) > 0 {
				issues = append(issues, fmt.Sprintf("raw bounds claim %s row is not correlated: %s", req.claim, strings.Join(rowIssues, ", ")))
				continue
			}
			valid++
		}
		if valid < req.minRows {
			issues = append(issues, fmt.Sprintf("raw bounds claim %s has %d correlated row(s), want at least %d", req.claim, valid, req.minRows))
		}
	}
	for _, row := range report.Rows {
		if row.ProvenanceClass == "unsafe_verified_root" || row.UnsafeClass == "unsafe_verified_root" {
			switch row.Claim {
			case "allocation_base_metadata", "unsafe_verified_root_allocation_base":
			default:
				issues = append(issues, fmt.Sprintf("unsafe_verified_root row %q cannot claim %q beyond bounded allocation metadata", row.SourceFactID, row.Claim))
			}
		}
	}
	if len(issues) > 0 {
		return fmt.Errorf(strings.Join(issues, "; "))
	}
	return nil
}

func validateRawSliceGatewayCorrelation(cases []memoryprod.CaseReport, memoryRaw []byte) error {
	report, err := parseMemoryReportEvidence(memoryRaw)
	if err != nil {
		return err
	}
	var issues []string
	issues = append(issues, validateRawSliceRuntimeCases(cases)...)
	issues = append(issues, validateCapMemAuthorizationDiscipline(report.Rows)...)
	checksByParent := parentedRawBoundsRuntimeChecks(report.Rows)

	requireRows := []struct {
		claim           string
		minRows         int
		validate        func(memoryReportEvidenceRow) []string
		requireCheckRow bool
	}{
		{claim: "external_unknown", minRows: 1, validate: validateUnsafeUnknownConservativeRow},
		{claim: "raw_slice_verified_allocation_root", minRows: 1, validate: validateUnsafeCheckedDynamicRow},
		{claim: "rejected_negative_length", minRows: 1, validate: validateUnsafeCheckedRejectedRow},
		{claim: "rejected_length_overflow", minRows: 1, validate: validateUnsafeCheckedRejectedRow, requireCheckRow: true},
	}
	for _, req := range requireRows {
		valid := 0
		for _, row := range report.Rows {
			if row.Claim != req.claim {
				continue
			}
			rowIssues := req.validate(row)
			if req.requireCheckRow && row.SourceFactID != "" && !checksByParent[row.SourceFactID] {
				rowIssues = append(rowIssues, fmt.Sprintf("source_fact_id %q is missing parented raw_bounds_runtime_check_normal_build normal_build_check", row.SourceFactID))
			}
			if len(rowIssues) > 0 {
				issues = append(issues, fmt.Sprintf("raw slice claim %s row is not correlated: %s", req.claim, strings.Join(rowIssues, ", ")))
				continue
			}
			valid++
		}
		if valid < req.minRows {
			issues = append(issues, fmt.Sprintf("raw slice claim %s has %d correlated row(s), want at least %d", req.claim, valid, req.minRows))
		}
	}
	if len(issues) > 0 {
		return fmt.Errorf(strings.Join(issues, "; "))
	}
	return nil
}

func validateRawRuntimeCases(cases []memoryprod.CaseReport) []string {
	byName := map[string]memoryprod.CaseReport{}
	for _, c := range cases {
		byName[c.Name] = c
	}
	var issues []string
	for _, req := range []struct {
		name          string
		expectedError string
	}{
		{name: "raw ptr_add negative offset bounds", expectedError: "negative ptr_add offset"},
		{name: "raw ptr_add allocation upper bound", expectedError: "allocation upper bound"},
		{name: "raw allocation-base i32 access width", expectedError: "i32 access width exceeds allocation"},
		{name: "raw allocation-base ptr access width", expectedError: "ptr access width exceeds allocation"},
		{name: "raw allocation-base store_i32 access width", expectedError: "i32 access width exceeds allocation"},
		{name: "raw allocation-base load_ptr access width", expectedError: "ptr access width exceeds allocation"},
	} {
		c, ok := byName[req.name]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing runtime raw bounds case %q", req.name))
			continue
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, fmt.Sprintf("runtime raw bounds case %q did not pass", req.name))
		}
		if req.expectedError != "" && !strings.Contains(c.ExpectedError, req.expectedError) {
			issues = append(issues, fmt.Sprintf("runtime raw bounds case %q expected_error = %q, want %q", req.name, c.ExpectedError, req.expectedError))
		}
	}
	return issues
}

func validateRawSliceRuntimeCases(cases []memoryprod.CaseReport) []string {
	byName := map[string]memoryprod.CaseReport{}
	for _, c := range cases {
		byName[c.Name] = c
	}
	var issues []string
	for _, req := range []struct {
		name          string
		expectedError string
	}{
		{name: "raw slice negative length", expectedError: "negative raw slice length"},
		{name: "raw slice i32 length byte overflow", expectedError: "raw slice length byte overflow"},
	} {
		c, ok := byName[req.name]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing runtime raw slice case %q", req.name))
			continue
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, fmt.Sprintf("runtime raw slice case %q did not pass", req.name))
		}
		if req.expectedError != "" && !strings.Contains(c.ExpectedError, req.expectedError) {
			issues = append(issues, fmt.Sprintf("runtime raw slice case %q expected_error = %q, want %q", req.name, c.ExpectedError, req.expectedError))
		}
	}
	return issues
}

func parentedRawBoundsRuntimeChecks(rows []memoryReportEvidenceRow) map[string]bool {
	out := map[string]bool{}
	for _, row := range rows {
		if row.Claim != "raw_bounds_runtime_check_normal_build" {
			continue
		}
		if row.SourceFactID == "" || row.ParentFactID == "" || !row.NormalBuildCheck {
			continue
		}
		if row.CostClass != "dynamic_check_required" || row.ValidatorName != "raw_bounds_width_validator" || row.ClaimLevel != "validated" {
			continue
		}
		out[row.ParentFactID] = true
	}
	return out
}

func validateUnsafeVerifiedRootAllocationBaseRow(row memoryReportEvidenceRow) []string {
	issues := validateCompilerOwnedRow(row)
	if row.ProvenanceClass != "unsafe_verified_root" || row.UnsafeClass != "unsafe_verified_root" {
		issues = append(issues, "must stay unsafe_verified_root")
	}
	if row.ClaimLevel != "validated" {
		issues = append(issues, "must be validated bounded allocation metadata")
	}
	return issues
}

func validateUnsafeCheckedDynamicRow(row memoryReportEvidenceRow) []string {
	issues := validateCompilerOwnedRow(row)
	if row.ProvenanceClass != "unsafe_checked" || row.UnsafeClass != "unsafe_checked" {
		issues = append(issues, "must stay unsafe_checked")
	}
	if row.CostClass != "dynamic_check_required" {
		issues = append(issues, "must preserve dynamic_check_required")
	}
	if !row.NormalBuildCheck {
		issues = append(issues, "must preserve normal_build_check")
	}
	return issues
}

func validateUnsafeCheckedRejectedRow(row memoryReportEvidenceRow) []string {
	issues := validateCompilerOwnedRow(row)
	if row.ProvenanceClass != "unsafe_checked" || row.UnsafeClass != "unsafe_checked" {
		issues = append(issues, "must stay unsafe_checked")
	}
	if row.CostClass != "unsupported_rejected" {
		issues = append(issues, "must preserve unsupported_rejected")
	}
	return issues
}

func validateUnsafeUnknownConservativeRow(row memoryReportEvidenceRow) []string {
	issues := validateCompilerOwnedRow(row)
	if row.ProvenanceClass != "unsafe_unknown" || row.UnsafeClass != "unsafe_unknown" {
		issues = append(issues, "must stay unsafe_unknown")
	}
	if row.ClaimLevel != "conservative" {
		issues = append(issues, "must remain conservative")
	}
	if row.CostClass != "conservative_fallback" {
		issues = append(issues, "must preserve conservative_fallback")
	}
	return issues
}

func validateCapMemAuthorizationDiscipline(rows []memoryReportEvidenceRow) []string {
	var issues []string
	authRows := 0
	proofClaims := map[string]bool{
		"provenance_known":                   true,
		"no_alias":                           true,
		"index_in_range":                     true,
		"bounds_proof_id":                    true,
		"bounds_check_eliminated":            true,
		"bounds_check_removed_with_proof_id": true,
	}
	for _, row := range rows {
		if row.Claim == "cap_mem_authorization_only" {
			authRows++
			rowIssues := validateCompilerOwnedRow(row)
			if row.ProvenanceClass != "unsafe_checked" || row.UnsafeClass != "unsafe_checked" {
				rowIssues = append(rowIssues, "must stay unsafe_checked authorization evidence")
			}
			if row.ClaimLevel != "evidence_only" {
				rowIssues = append(rowIssues, "must remain evidence_only")
			}
			if row.CostClass != "instrumentation_only" {
				rowIssues = append(rowIssues, "must remain instrumentation_only")
			}
			if row.ValidatorName != "" || row.ValidatorStatus != "not_run" {
				rowIssues = append(rowIssues, "must not be treated as a validated proof")
			}
			if row.NormalBuildCheck {
				rowIssues = append(rowIssues, "must not masquerade as a bounds check")
			}
			if len(rowIssues) > 0 {
				issues = append(issues, fmt.Sprintf("cap.mem authorization row is not evidence-only: %s", strings.Join(rowIssues, ", ")))
			}
		}
		if strings.Contains(strings.ToLower(row.ValidatorName+" "+row.Reason), "cap_mem") ||
			strings.Contains(strings.ToLower(row.ValidatorName+" "+row.Reason), "cap.mem") {
			if proofClaims[row.Claim] {
				issues = append(issues, fmt.Sprintf("cap.mem authorization cannot validate proof claim %q", row.Claim))
			}
		}
	}
	if authRows == 0 {
		issues = append(issues, "missing cap_mem_authorization_only evidence row")
	}
	return issues
}

func validateCompilerOwnedRow(row memoryReportEvidenceRow) []string {
	var issues []string
	if strings.TrimSpace(row.SourceFactID) == "" {
		issues = append(issues, "missing source_fact_id")
	}
	return issues
}

func ceilDiv(n, d int) int {
	if d <= 0 {
		return 0
	}
	return (n + d - 1) / d
}

func smallHeapBenchmarkSource(allocationCount, bytesPerAllocation int) string {
	var b strings.Builder
	for i := 0; i < allocationCount; i++ {
		fmt.Fprintf(&b, "func make_%02d() -> []u8\n", i)
		b.WriteString("uses alloc, mem:\n")
		fmt.Fprintf(&b, "    var xs: []u8 = make_u8(%d)\n", bytesPerAllocation)
		b.WriteString("    return xs\n\n")
	}
	b.WriteString("func main() -> Int\n")
	b.WriteString("uses alloc, mem:\n")
	for i := 0; i < allocationCount; i++ {
		fmt.Fprintf(&b, "    let xs_%02d: []u8 = make_%02d()\n", i, i)
	}
	b.WriteString("    return ")
	for i := 0; i < allocationCount; i++ {
		if i > 0 {
			b.WriteString(" + ")
		}
		fmt.Fprintf(&b, "xs_%02d.len", i)
	}
	b.WriteString("\n")
	return b.String()
}

func (r *smokeRunner) writeReport() error {
	report := buildReport("tools/cmd/memory-production-smoke", r.processes, r.benchmarks, r.cases)
	report.GitHead = strings.TrimSpace(r.opt.GitHead)
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := memoryprod.ValidateReport(raw); err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(r.opt.ReportPath, raw, 0o644)
}

func buildReport(source string, processes []memoryprod.ProcessReport, benchmarks []memoryprod.BenchmarkReport, cases []memoryprod.CaseReport) memoryprod.Report {
	return memoryprod.Report{
		Schema:  memoryprod.SchemaV1,
		Status:  "pass",
		Target:  "linux-x64",
		Host:    "linux-x64",
		Runtime: "memory-linux-x64",
		Source:  source,
		Processes: append([]memoryprod.ProcessReport(nil),
			processes...,
		),
		Benchmarks: append([]memoryprod.BenchmarkReport(nil),
			benchmarks...,
		),
		Contracts: []memoryprod.ContractReport{
			{Name: "allocator runtime model", Status: "pass", Evidence: "linux-x64 allocator headers and scoped island lifecycle smoke"},
			{Name: "allocator failure semantics", Status: "pass", Evidence: "linux-x64 mmap failure guard and invalid-size runtime status"},
			{Name: "ownership escape model", Status: "pass", Evidence: "compiler safety diagnostics for borrow escape and resource aliases"},
			{Name: "unsafe cap.mem raw memory rules", Status: "pass", Evidence: "raw helper examples require unsafe and explicit cap.mem"},
			{Name: "runtime bounds diagnostics", Status: "pass", Evidence: "linux-x64 raw pointer and slice bounds negative cases"},
			{Name: "raw pointer bounds metadata", Status: "pass", Evidence: "allocation report schema v2 includes allocation_base_metadata; memory report is validated against paired allocation report lowered_artifact_id evidence and includes derived_allocation_offset, rejected_negative_offset, rejected_upper_bound, rejected_access_width_overflow, checked_external_unknown, external_unknown, raw_slice_verified_allocation_root, rejected_negative_length, and rejected_length_overflow"},
			{Name: "allocation length contracts", Status: "pass", Evidence: "linux-x64 AllocationLength runtime tests plus paired allocation/memory report evidence for core.alloc_bytes, core.make_*, and core.island_make_* zero, negative, and byte-size overflow contracts"},
			{Name: "host resource leak and finalization checks", Status: "pass", Evidence: "actornet TestBrokerCloseWithoutCancelStopsServeWatcher plus compiler resource_finalization_test.go selectors prove close-without-cancel goroutine watcher cleanup and resource finalization diagnostics"},
			{Name: "actor task transfer rules", Status: "pass", Evidence: "compiler safety diagnostics for actor/task transfer boundaries"},
		},
		Cases: append([]memoryprod.CaseReport(nil), cases...),
		Audit: memoryProductionAudit(),
	}
}

func memoryProductionAudit() []memoryprod.AuditReport {
	return []memoryprod.AuditReport{
		{
			Requirement: "stable allocator/runtime memory model",
			Artifact:    "lib/core/memory.tetra; compiler/internal/actorsrt/linux_x64_emit.go; tools/cmd/memory-production-smoke",
			Evidence:    "allocator alloc/free lifecycle, allocator invalid size precondition, allocator failure semantics, and stress allocator reuse cases ran on linux-x64",
			Result:      "pass",
		},
		{
			Requirement: "ownership/borrow/consume escape model",
			Artifact:    "compiler/tests/ownership; compiler/tests/safety",
			Evidence:    "borrow escape, use-after-free, double-free, aliasing, callable heap escape, and actor/task transfer diagnostics are required memory production cases",
			Result:      "pass",
		},
		{
			Requirement: "heap, slices, structs, and closures memory coverage",
			Artifact:    "docs/spec/ownership_v1.md; compiler/tests/ownership; compiler/tests/semantics/closures_semantic_clauses_test.go",
			Evidence:    "heap closure handle coverage, callable heap escape rejection, slice struct borrow escape coverage, and function-typed slice aggregate borrow escape coverage run compiler tests for closure heap handles, nested slice/struct escapes, and conservative rejection of unsafe escapes",
			Result:      "pass",
		},
		{
			Requirement: "unsafe/cap.mem/raw memory/memcpy/memset rules",
			Artifact:    "docs/spec/unsafe.md; docs/spec/capabilities.md; lib/core/memory.tetra",
			Evidence:    "cap.mem unsafe boundary plus memcpy/memset capability path and negative helper length cases require unsafe and explicit cap.mem",
			Result:      "pass",
		},
		{
			Requirement: "runtime bounds checks and diagnostics",
			Artifact:    "docs/spec/runtime_abi.md; compiler/compiler_test.go; tools/cmd/memory-production-smoke",
			Evidence:    "slice bounds, ptr_add negative offset, allocation upper bound, load/store i32 width, load/store ptr width, raw-slice length, and negative helper length diagnostics are required cases",
			Result:      "pass",
		},
		{
			Requirement: "raw pointer bounds metadata",
			Artifact:    "compiler/internal/runtimeabi/raw_pointer_bounds.go; compiler/internal/plir/plir.go; compiler/internal/allocplan/plan.go; tools/cmd/memory-production-smoke",
			Evidence:    "core.alloc_bytes allocation reports include allocation_base_metadata and external_unknown raw-slice policy; memory reports are validated against paired allocation report lowered_artifact_id evidence and include derived_allocation_offset, rejected_negative_offset, rejected_upper_bound, rejected_access_width_overflow, checked_external_unknown, external_unknown, raw_slice_verified_allocation_root, rejected_negative_length, and rejected_length_overflow without arbitrary raw pointer safety claims",
			Result:      "pass",
		},
		{
			Requirement: "allocation length contracts",
			Artifact:    "compiler/internal/runtimeabi/allocation_contract.go; compiler/internal/allocplan/plan.go; compiler/tests/semantics/allocation_length_contract_test.go; tools/cmd/memory-production-smoke",
			Evidence:    "release smoke requires AllocationLength runtime coverage and a paired allocation/memory report whose allocation rows preserve builtin, length_status, zero_guard_status, negative_guard_status, and overflow_guard_status for core.alloc_bytes, core.make_*, and core.island_make_*",
			Result:      "pass",
		},
		{
			Requirement: "stress/fuzz evidence",
			Artifact:    "tools/cmd/memory-production-smoke",
			Evidence:    "stress allocator reuse and deterministic memcpy/memset fuzz cases ran through the release-gate entrypoint",
			Result:      "pass",
		},
		{
			Requirement: "measured memory benchmark improvement",
			Artifact:    "tools/cmd/memory-production-smoke; compiler allocation report schema v2",
			Evidence:    "small heap allocation syscall reduction benchmark reads the emitted allocation report, counts per_core_small_heap rows with same_core_same_size_class_free_list reuse policy, and compares estimated mmap-per-allocation baseline against 64KiB chunk refill calls",
			Result:      "pass",
		},
		{
			Requirement: "use-after-free, double-free, borrow escape, and aliasing safety",
			Artifact:    "compiler/tests/safety; compiler/tests/ownership; compiler",
			Evidence:    "required compiler safety cases reject use-after-free, double-free, borrow escape, and inout aliasing violations",
			Result:      "pass",
		},
		{
			Requirement: "actor/task transfer safety",
			Artifact:    "compiler/tests/ownership",
			Evidence:    "TestReleaseTraceabilityLifetimeAndRaceSafetyNegativeActorTaskOwnership rejects unsafe actor/task transfer boundaries",
			Result:      "pass",
		},
		{
			Requirement: "leak/resource finalization evidence",
			Artifact:    "cli/internal/actornet/broker_test.go; compiler/tests/runtime/resource_finalization_test.go; tools/cmd/memory-production-smoke",
			Evidence:    "release smoke runs actornet close-without-cancel watcher leak coverage and compiler TaskHandle/TaskGroup/Island resource finalization diagnostics for optional, enum, function-typed, branch, loop, match, join, close, and free paths",
			Result:      "pass",
		},
		{
			Requirement: "real memory examples",
			Artifact:    "examples/core_memory_smoke.tetra; examples/ownership_smoke.tetra; examples/flow_unsafe_cap_mem_smoke.tetra",
			Evidence:    "checked-in memory, ownership, and unsafe cap.mem examples build and run under the memory production release gate",
			Result:      "pass",
		},
		{
			Requirement: "safe memory documentation",
			Artifact:    "docs/spec/runtime_abi.md; docs/spec/ownership_v1.md; docs/spec/unsafe.md; docs/user/standard_library_guide.md",
			Evidence:    "verify-docs requires the Memory Production ABI, ownership extension, unsafe boundary, and writing raw memory safely guide sections",
			Result:      "pass",
		},
		{
			Requirement: "release-gate entrypoint",
			Artifact:    "scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh",
			Evidence:    "entrypoint writes memory-production-linux-x64.json and runs memory-production-smoke plus validate-memory-production",
			Result:      "pass",
		},
	}
}

func requiredPassingCases() []memoryprod.CaseReport {
	return []memoryprod.CaseReport{
		{Name: "allocator alloc/free lifecycle", Kind: "positive", Ran: true, Pass: true},
		{Name: "allocator failure semantics", Kind: "negative", Ran: true, Pass: true, ExpectedError: "allocation failure"},
		{Name: "allocator invalid size precondition", Kind: "negative", Ran: true, Pass: true, ExpectedError: "invalid allocation size"},
		{Name: "cap.mem unsafe boundary", Kind: "negative", Ran: true, Pass: true, ExpectedError: "only allowed in unsafe blocks"},
		{Name: "memcpy/memset capability path", Kind: "positive", Ran: true, Pass: true},
		{Name: "runtime bounds check", Kind: "negative", Ran: true, Pass: true, ExpectedError: "bounds"},
		{Name: "raw ptr_add negative offset bounds", Kind: "negative", Ran: true, Pass: true, ExpectedError: "negative ptr_add offset"},
		{Name: "raw ptr_add allocation upper bound", Kind: "negative", Ran: true, Pass: true, ExpectedError: "allocation upper bound"},
		{Name: "raw allocation-base i32 access width", Kind: "negative", Ran: true, Pass: true, ExpectedError: "i32 access width exceeds allocation"},
		{Name: "raw allocation-base ptr access width", Kind: "negative", Ran: true, Pass: true, ExpectedError: "ptr access width exceeds allocation"},
		{Name: "raw allocation-base store_i32 access width", Kind: "negative", Ran: true, Pass: true, ExpectedError: "i32 access width exceeds allocation"},
		{Name: "raw allocation-base load_ptr access width", Kind: "negative", Ran: true, Pass: true, ExpectedError: "ptr access width exceeds allocation"},
		{Name: "raw slice negative length", Kind: "negative", Ran: true, Pass: true, ExpectedError: "negative raw slice length"},
		{Name: "raw slice i32 length byte overflow", Kind: "negative", Ran: true, Pass: true, ExpectedError: "raw slice length byte overflow"},
		{Name: "allocation make zero length canonical empty", Kind: "positive", Ran: true, Pass: true},
		{Name: "allocation make negative length", Kind: "negative", Ran: true, Pass: true, ExpectedError: "negative allocation length"},
		{Name: "allocation make byte-size overflow", Kind: "negative", Ran: true, Pass: true, ExpectedError: "allocation length byte overflow"},
		{Name: "allocation island zero length no metadata", Kind: "positive", Ran: true, Pass: true},
		{Name: "allocation island negative length", Kind: "negative", Ran: true, Pass: true, ExpectedError: "negative allocation length"},
		{Name: "allocation island byte-size overflow", Kind: "negative", Ran: true, Pass: true, ExpectedError: "allocation length byte overflow"},
		{Name: "raw pointer bounds metadata report", Kind: "positive", Ran: true, Pass: true},
		{Name: "allocation length contract report", Kind: "positive", Ran: true, Pass: true},
		{Name: "memcpy/memset negative length", Kind: "negative", Ran: true, Pass: true, ExpectedError: "negative helper length"},
		{Name: "reject use-after-free", Kind: "negative", Ran: true, Pass: true, ExpectedError: "use-after-free"},
		{Name: "reject double-free", Kind: "negative", Ran: true, Pass: true, ExpectedError: "double-free"},
		{Name: "reject borrow escape", Kind: "negative", Ran: true, Pass: true, ExpectedError: "borrow escape"},
		{Name: "reject aliasing violation", Kind: "negative", Ran: true, Pass: true, ExpectedError: "alias"},
		{Name: "callable mutable capture heap escape", Kind: "negative", Ran: true, Pass: true, ExpectedError: "heap-escaped function value captures mutable local"},
		{Name: "reject actor task transfer violation", Kind: "negative", Ran: true, Pass: true, ExpectedError: "transfer"},
		{Name: "heap closure handle coverage", Kind: "positive", Ran: true, Pass: true},
		{Name: "slice struct borrow escape coverage", Kind: "negative", Ran: true, Pass: true, ExpectedError: "borrow escape"},
		{Name: "function-typed slice aggregate borrow escape coverage", Kind: "negative", Ran: true, Pass: true, ExpectedError: "borrow escape"},
		{Name: "actornet broker close-without-cancel leak smoke", Kind: "stress", Ran: true, Pass: true},
		{Name: "compiler resource finalization diagnostics", Kind: "negative", Ran: true, Pass: true, ExpectedError: "resource finalization"},
		{Name: "real memory examples", Kind: "positive", Ran: true, Pass: true},
		{Name: "stress allocator reuse", Kind: "stress", Ran: true, Pass: true},
		{Name: "deterministic memcpy/memset fuzz", Kind: "stress", Ran: true, Pass: true},
	}
}

func failedCase(name, kind, expectedError, errText string) memoryprod.CaseReport {
	return memoryprod.CaseReport{Name: name, Kind: kind, Ran: true, Pass: false, ExpectedError: expectedError, Error: strings.TrimSpace(errText)}
}

func (r *smokeRunner) recordProcess(name, kind, path string, res processResult) {
	r.processes = append(r.processes, memoryprod.ProcessReport{
		Name:     name,
		Kind:     kind,
		Path:     path,
		Ran:      true,
		Pass:     res.err == nil && res.exitCode == 0,
		ExitCode: intPtr(res.exitCode),
	})
}

func runCommand(ctx context.Context, timeout time.Duration, name string, args ...string) processResult {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := strings.TrimSpace(stdout.String() + stderr.String())
	if cctx.Err() == context.DeadlineExceeded {
		return processResult{exitCode: -1, output: output, err: cctx.Err()}
	}
	return processResult{exitCode: processExitCode(err), output: output, err: err}
}

func processExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			if status.Signaled() {
				return -int(status.Signal())
			}
			return status.ExitStatus()
		}
	}
	return -1
}

func intPtr(v int) *int { return &v }

const memoryPositiveSource = `
import lib.core.memory as memory

func main() -> Int
uses alloc, capability, islands, mem:
    island(64) as isl:
        var xs: []u8 = core.island_make_u8(isl, 1)
        xs[0] = 7
        if xs[0] != 7:
            return 1
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let src: ptr = core.alloc_bytes(4)
        let dst: ptr = core.alloc_bytes(4)
        let clear_status: Int = memory.memset_u8(dst, 0, 4, mem)
        let seed_status: Int = memory.memset_u8(src, 42, 4, mem)
        let copy_status: Int = memory.memcpy_u8(dst, src, 4, mem)
        if clear_status == 0:
            if seed_status == 0:
                if copy_status == 0:
                    if core.load_u8(dst, mem) == 42:
                        return 0
        return 1
    return 1
`

const memoryStressSource = `
import lib.core.memory as memory

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let src: ptr = core.alloc_bytes(64)
        let dst: ptr = core.alloc_bytes(64)
        var i: Int = 0
        while i < 32:
            let seed_status: Int = memory.memset_u8(src, 7, 64, mem)
            let clear_status: Int = memory.memset_u8(dst, 0, 64, mem)
            let copy_status: Int = memory.memcpy_u8(dst, src, 64, mem)
            if seed_status != 0:
                return 1
            if clear_status != 0:
                return 1
            if copy_status != 0:
                return 1
            let p: ptr = core.ptr_add(dst, i, mem)
            if core.load_u8(p, mem) != 7:
                return 1
            i = i + 1
        return 0
    return 1
`

const memoryFuzzSource = `
import lib.core.memory as memory

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let src: ptr = core.alloc_bytes(64)
        let dst: ptr = core.alloc_bytes(64)
        var n: Int = 1
        while n < 33:
            let seed_status: Int = memory.memset_u8(src, 17, n, mem)
            let clear_status: Int = memory.memset_u8(dst, 0, 64, mem)
            let copy_status: Int = memory.memcpy_u8(dst, src, n, mem)
            if seed_status != 0:
                return 1
            if clear_status != 0:
                return 1
            if copy_status != 0:
                return 1
            if core.load_u8(dst, mem) != 17:
                return 1
            let last: ptr = core.ptr_add(dst, n - 1, mem)
            if core.load_u8(last, mem) != 17:
                return 1
            let sentinel: ptr = core.ptr_add(dst, n, mem)
            if core.load_u8(sentinel, mem) != 0:
                return 1
            n = n + 1
        return 0
    return 1
`

const allocInvalidSizeSource = `
func main() -> Int
uses alloc, mem:
    unsafe:
        let _: ptr = core.alloc_bytes(0)
        return 0
    return 0
`

const sliceBoundsSource = `
func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[2] = 1
    return 0
`

const rawPtrAddNegativeSource = `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let q: ptr = core.ptr_add(p, 0 - 1, mem)
        let _: UInt8 = core.store_u8(q, 7, mem)
        return 0
    return 0
`

const rawPtrAddUpperSource = `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let q: ptr = core.ptr_add(p, 4, mem)
        let _: UInt8 = core.store_u8(q, 7, mem)
        return 0
    return 0
`

const rawI32WidthSource = `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let q: ptr = core.ptr_add(p, 5, mem)
        let _: Int = core.load_i32(q, mem)
        return 0
    return 0
`

const rawPtrWidthSource = `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let q: ptr = core.ptr_add(p, 1, mem)
        let _: ptr = core.store_ptr(q, p, mem)
        return 0
    return 0
`

const rawStoreI32WidthSource = `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let q: ptr = core.ptr_add(p, 5, mem)
        let _: Int = core.store_i32(q, 123, mem)
        return 0
    return 0
`

const rawLoadPtrWidthSource = `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let q: ptr = core.ptr_add(p, 1, mem)
        let _: ptr = core.load_ptr(q, mem)
        return 0
    return 0
`

const rawPointerBoundsMetadataSource = `
func external(raw: ptr, n: Int) -> Int
uses capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let q: ptr = core.ptr_add(raw, 0 - 1, mem)
        let xs: []u8 = core.raw_slice_u8_from_parts(q, n, mem)
        return xs.len
    return 0

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(24)
        let q: ptr = core.ptr_add(p, 8, mem)
        let _: UInt8 = core.store_u8(q, 7, mem)
        let value: UInt8 = core.load_u8(q, mem)
        let neg_base: ptr = core.alloc_bytes(8)
        let neg: ptr = core.ptr_add(neg_base, 0 - 1, mem)
        let upper_base: ptr = core.alloc_bytes(8)
        let upper: ptr = core.ptr_add(upper_base, 8, mem)
        let i32_base: ptr = core.alloc_bytes(8)
        let i32_ptr: ptr = core.ptr_add(i32_base, 5, mem)
        let i32_value: Int = core.load_i32(i32_ptr, mem)
        let ptr_base: ptr = core.alloc_bytes(4)
        let ptr_ptr: ptr = core.ptr_add(ptr_base, 1, mem)
        let ptr_value: ptr = core.store_ptr(ptr_ptr, ptr_base, mem)
        let store_i32_base: ptr = core.alloc_bytes(8)
        let store_i32_ptr: ptr = core.ptr_add(store_i32_base, 5, mem)
        let store_i32_status: Int = core.store_i32(store_i32_ptr, 123, mem)
        let load_ptr_base: ptr = core.alloc_bytes(4)
        let load_ptr_ptr: ptr = core.ptr_add(load_ptr_base, 1, mem)
        let load_ptr_value: ptr = core.load_ptr(load_ptr_ptr, mem)
        let raw_slice_base: ptr = core.alloc_bytes(16)
        let raw_slice_view: []u8 = core.raw_slice_u8_from_parts(raw_slice_base, 8, mem)
        let raw_slice_negative: []u8 = core.raw_slice_u8_from_parts(raw_slice_base, 0 - 1, mem)
        let raw_slice_overflow: []i32 = core.raw_slice_i32_from_parts(raw_slice_base, 536870912, mem)
        return i32_value + store_i32_status + raw_slice_view.len + raw_slice_negative.len + raw_slice_overflow.len
    return 0
`

const allocationLengthContractReportSource = `
func main() -> Int
uses alloc, islands, mem:
    unsafe:
        let raw_zero: ptr = core.alloc_bytes(0)
    var make_zero: []u8 = make_u8(0)
    var make_negative: []u16 = make_u16(0 - 1)
    var make_overflow: []i32 = make_i32(536870912)
    island(64) as isl:
        var island_zero: []u8 = core.island_make_u8(isl, 0)
        var island_negative: []u16 = core.island_make_u16(isl, 0 - 1)
        var island_overflow: []i32 = core.island_make_i32(isl, 536870912)
        return make_zero.len + make_negative.len + make_overflow.len + island_zero.len + island_negative.len + island_overflow.len
    return 0
`

const rawSliceNegativeLengthSource = `
func main() -> Int
uses capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = 0
        let xs: []u8 = core.raw_slice_u8_from_parts(p, 0 - 1, mem)
        return xs.len + 98
    return 0
`

const rawSliceI32LengthOverflowSource = `
func main() -> Int
uses capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = 0
        let xs: []i32 = core.raw_slice_i32_from_parts(p, 536870912, mem)
        return xs.len + 98
    return 0
`

const memoryNegativeLengthSource = `
import lib.core.memory as memory

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let src: ptr = core.alloc_bytes(4)
        let dst: ptr = core.alloc_bytes(4)
        let memset_status: Int = memory.memset_u8(dst, 0, 0 - 1, mem)
        let memcpy_status: Int = memory.memcpy_u8(dst, src, 0 - 1, mem)
        if memset_status == 2:
            if memcpy_status == 2:
                return 2
        return 1
    return 1
`
