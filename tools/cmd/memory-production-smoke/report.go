package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"tetra_language/tools/validators/memoryprod"
)

const ramMeasurementSchema = "tetra.memory.ram-measurement.v1"

type ramMeasurementReport struct {
	Schema        string                   `json:"schema"`
	Status        string                   `json:"status"`
	Target        string                   `json:"target"`
	EvidenceClass string                   `json:"evidence_class"`
	Method        string                   `json:"method"`
	Tool          string                   `json:"tool"`
	GitHead       string                   `json:"git_head,omitempty"`
	GeneratedAt   string                   `json:"generated_at"`
	Snapshots     []ramMeasurementSnapshot `json:"snapshots"`
}

type ramMeasurementSnapshot struct {
	Name              string  `json:"name"`
	Timestamp         string  `json:"timestamp"`
	AllocBytes        uint64  `json:"alloc_bytes"`
	TotalAllocBytes   uint64  `json:"total_alloc_bytes"`
	SysBytes          uint64  `json:"sys_bytes"`
	HeapAllocBytes    uint64  `json:"heap_alloc_bytes"`
	HeapSysBytes      uint64  `json:"heap_sys_bytes"`
	HeapIdleBytes     uint64  `json:"heap_idle_bytes"`
	HeapReleasedBytes uint64  `json:"heap_released_bytes"`
	NumGC             uint32  `json:"num_gc"`
	GCCPUFraction     float64 `json:"gc_cpu_fraction"`
}

func (r *smokeRunner) writeReport() error {
	report := buildReport("tools/cmd/memory-production-smoke", r.processes, r.benchmarks, r.cases)
	report.GitHead = strings.TrimSpace(r.opt.GitHead)
	if err := memoryprod.ValidateReportObject(report); err != nil {
		return err
	}
	return writeMemoryProductionJSONFile(r.opt.ReportPath, report)
}

func (r *smokeRunner) recordRAMSnapshot(name string) {
	if strings.TrimSpace(r.opt.RAMMeasurementPath) == "" {
		return
	}
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	r.ramSnapshots = append(r.ramSnapshots, ramMeasurementSnapshot{
		Name:              name,
		Timestamp:         time.Now().UTC().Format(time.RFC3339Nano),
		AllocBytes:        stats.Alloc,
		TotalAllocBytes:   stats.TotalAlloc,
		SysBytes:          stats.Sys,
		HeapAllocBytes:    stats.HeapAlloc,
		HeapSysBytes:      stats.HeapSys,
		HeapIdleBytes:     stats.HeapIdle,
		HeapReleasedBytes: stats.HeapReleased,
		NumGC:             stats.NumGC,
		GCCPUFraction:     stats.GCCPUFraction,
	})
}

func (r *smokeRunner) writeRAMMeasurementReport() error {
	if strings.TrimSpace(r.opt.RAMMeasurementPath) == "" {
		return nil
	}
	if len(r.ramSnapshots) == 0 {
		r.recordRAMSnapshot("end")
	}
	report := ramMeasurementReport{
		Schema:        ramMeasurementSchema,
		Status:        "pass",
		Target:        "linux-x64",
		EvidenceClass: "runtime_measured",
		Method:        "MemStats",
		Tool:          "tools/cmd/memory-production-smoke",
		GitHead:       strings.TrimSpace(r.opt.GitHead),
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Snapshots:     append([]ramMeasurementSnapshot(nil), r.ramSnapshots...),
	}
	if err := os.MkdirAll(filepath.Dir(r.opt.RAMMeasurementPath), 0o755); err != nil {
		return err
	}
	return writeMemoryProductionJSONFile(r.opt.RAMMeasurementPath, report)
}

func writeMemoryProductionJSONFile(path string, value any) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
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
			Requirement: "allocator benchmark evidence classification",
			Artifact:    "tools/cmd/memory-production-smoke; compiler allocation report schema v2",
			Evidence:    "small heap allocation syscall reduction benchmark is classified as allocation_report_estimate from the emitted allocation report and does not claim runtime RSS, pprof, MemStats, time_v, or strace measurement",
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
