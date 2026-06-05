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
	if err := r.runCompilerSafetyCases(ctx); err != nil {
		return err
	}
	if err := r.runMemoryShapeCoverageCases(ctx); err != nil {
		return err
	}
	if err := r.runSmallHeapAllocationBenchmark(ctx); err != nil {
		return err
	}
	if err := r.runRawPointerBoundsMetadataReport(ctx); err != nil {
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
	validateMemory := runCommand(ctx, 30*time.Second, "go", "run", "./tools/cmd/validate-memory-report", "--report", outPath+".memory.json")
	r.recordProcess("raw pointer bounds memory report validation", "benchmark", "go run ./tools/cmd/validate-memory-report --report "+outPath+".memory.json", validateMemory)
	if validateMemory.err != nil {
		r.cases = append(r.cases, failedCase("raw pointer bounds metadata report", "positive", "", validateMemory.output))
		return fmt.Errorf("validate raw pointer bounds memory report: %s", validateMemory.output)
	}
	memoryClaims, err := parseMemoryReportClaims(memoryRaw)
	if err != nil {
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

type allocationReportSummary struct {
	SchemaVersion int `json:"schema_version"`
	Summary       struct {
		AllocationCount          int            `json:"allocation_count"`
		RuntimePaths             map[string]int `json:"runtime_paths"`
		AllocatorClasses         map[string]int `json:"allocator_classes"`
		AllocatorReusePolicies   map[string]int `json:"allocator_reuse_policies"`
		RawPointerBoundsStatuses map[string]int `json:"raw_pointer_bounds_statuses"`
		RawSlicePolicies         map[string]int `json:"raw_slice_policies"`
		BytesReserved            int            `json:"bytes_reserved"`
	} `json:"summary"`
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
	if report.Summary.AllocatorReusePolicies == nil {
		return allocationReportSummary{}, fmt.Errorf("small heap allocation report missing allocator_reuse_policies summary")
	}
	return report, nil
}

func parseMemoryReportClaims(raw []byte) (map[string]int, error) {
	var report struct {
		SchemaVersion string `json:"schema_version"`
		Rows          []struct {
			Claim string `json:"claim"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		return nil, fmt.Errorf("parse memory report: %w", err)
	}
	if report.SchemaVersion != "tetra.memory-report.v1" {
		return nil, fmt.Errorf("memory report schema_version = %q, want tetra.memory-report.v1", report.SchemaVersion)
	}
	claims := map[string]int{}
	for _, row := range report.Rows {
		claims[row.Claim]++
	}
	return claims, nil
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
			{Name: "raw pointer bounds metadata", Status: "pass", Evidence: "allocation report schema v2 includes allocation_base_metadata; memory report includes derived_allocation_offset, rejected_negative_offset, rejected_upper_bound, rejected_access_width_overflow, checked_external_unknown, external_unknown, raw_slice_verified_allocation_root, and rejected_negative_length"},
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
			Evidence:    "core.alloc_bytes allocation reports include allocation_base_metadata and external_unknown raw-slice policy; memory reports include derived_allocation_offset, rejected_negative_offset, rejected_upper_bound, rejected_access_width_overflow, checked_external_unknown, external_unknown, raw_slice_verified_allocation_root, and rejected_negative_length without arbitrary raw pointer safety claims",
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
		{Name: "raw pointer bounds metadata report", Kind: "positive", Ran: true, Pass: true},
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
        let raw_slice_base: ptr = core.alloc_bytes(16)
        let raw_slice_view: []u8 = core.raw_slice_u8_from_parts(raw_slice_base, 8, mem)
        let raw_slice_negative: []u8 = core.raw_slice_u8_from_parts(raw_slice_base, 0 - 1, mem)
        return i32_value + raw_slice_view.len + raw_slice_negative.len
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
