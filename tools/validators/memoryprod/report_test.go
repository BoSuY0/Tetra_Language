package memoryprod

import (
	"strings"
	"testing"
)

func TestValidateReportAcceptsLinuxX64MemoryProductionEvidence(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.memory.production.v1",
  "status": "pass",
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "memory-linux-x64",
  "source": "examples/core_memory_smoke.tetra",
  "processes": [
    {"name":"tetra build","kind":"build","path":"/tmp/tetra","ran":true,"pass":true,"exit_code":0},
    {"name":"memory smoke app","kind":"app","path":"/tmp/memory-smoke","ran":true,"pass":true,"exit_code":0},
    {"name":"memory stress","kind":"stress","path":"tools/cmd/memory-production-smoke","ran":true,"pass":true,"exit_code":0}
  ],
  "benchmarks": [
    {"name":"small heap allocation syscall reduction","kind":"allocator","metric":"estimated_os_syscalls","unit":"syscalls","baseline_value":64,"measured_value":1,"improvement_ratio":64.0,"evidence":"allocation report schema v2 shows 64 per_core_small_heap rows with same_core_same_size_class_free_list reuse policy inside one 64KiB chunk refill","ran":true,"pass":true}
  ],
  "contracts": [
    {"name":"allocator runtime model","status":"pass","evidence":"allocator lifecycle returns deterministic handles and failure status"},
    {"name":"allocator failure semantics","status":"pass","evidence":"linux-x64 mmap failure exits deterministically before returning an invalid pointer"},
    {"name":"ownership escape model","status":"pass","evidence":"heap, slices, structs, and closures preserve borrow/consume diagnostics"},
    {"name":"unsafe cap.mem raw memory rules","status":"pass","evidence":"raw memory helpers require unsafe and explicit cap.mem"},
    {"name":"runtime bounds diagnostics","status":"pass","evidence":"out-of-bounds memory access reports deterministic runtime diagnostic"},
    {"name":"raw pointer bounds metadata","status":"pass","evidence":"allocation_base_metadata, derived_allocation_offset, checked_external_unknown, and external_unknown raw-slice policy"},
    {"name":"actor task transfer rules","status":"pass","evidence":"memory-bearing values cannot cross actor/task boundaries without checked transfer"}
  ],
  "cases": [
    {"name":"allocator alloc/free lifecycle","kind":"positive","ran":true,"pass":true},
    {"name":"allocator failure semantics","kind":"negative","ran":true,"pass":true,"expected_error":"allocation failure"},
    {"name":"allocator invalid size precondition","kind":"negative","ran":true,"pass":true,"expected_error":"invalid allocation size"},
    {"name":"cap.mem unsafe boundary","kind":"negative","ran":true,"pass":true,"expected_error":"only allowed in unsafe blocks"},
    {"name":"memcpy/memset capability path","kind":"positive","ran":true,"pass":true},
    {"name":"runtime bounds check","kind":"negative","ran":true,"pass":true,"expected_error":"bounds"},
    {"name":"raw ptr_add negative offset bounds","kind":"negative","ran":true,"pass":true,"expected_error":"negative ptr_add offset"},
    {"name":"raw ptr_add allocation upper bound","kind":"negative","ran":true,"pass":true,"expected_error":"allocation upper bound"},
    {"name":"raw allocation-base i32 access width","kind":"negative","ran":true,"pass":true,"expected_error":"i32 access width exceeds allocation"},
    {"name":"raw allocation-base ptr access width","kind":"negative","ran":true,"pass":true,"expected_error":"ptr access width exceeds allocation"},
    {"name":"raw slice negative length","kind":"negative","ran":true,"pass":true,"expected_error":"negative raw slice length"},
    {"name":"raw slice i32 length byte overflow","kind":"negative","ran":true,"pass":true,"expected_error":"raw slice length byte overflow"},
    {"name":"raw pointer bounds metadata report","kind":"positive","ran":true,"pass":true},
    {"name":"memcpy/memset negative length","kind":"negative","ran":true,"pass":true,"expected_error":"negative helper length"},
    {"name":"reject use-after-free","kind":"negative","ran":true,"pass":true,"expected_error":"use-after-free"},
    {"name":"reject double-free","kind":"negative","ran":true,"pass":true,"expected_error":"double-free"},
    {"name":"reject borrow escape","kind":"negative","ran":true,"pass":true,"expected_error":"borrow escape"},
    {"name":"reject aliasing violation","kind":"negative","ran":true,"pass":true,"expected_error":"alias"},
    {"name":"callable mutable capture heap escape","kind":"negative","ran":true,"pass":true,"expected_error":"heap-escaped function value captures mutable local"},
    {"name":"reject actor task transfer violation","kind":"negative","ran":true,"pass":true,"expected_error":"transfer"},
    {"name":"heap closure handle coverage","kind":"positive","ran":true,"pass":true},
    {"name":"slice struct borrow escape coverage","kind":"negative","ran":true,"pass":true,"expected_error":"borrow escape"},
    {"name":"function-typed slice aggregate borrow escape coverage","kind":"negative","ran":true,"pass":true,"expected_error":"borrow escape"},
    {"name":"real memory examples","kind":"positive","ran":true,"pass":true},
    {"name":"stress allocator reuse","kind":"stress","ran":true,"pass":true},
    {"name":"deterministic memcpy/memset fuzz","kind":"stress","ran":true,"pass":true}
  ],
  "audit": [
    {"requirement":"stable allocator/runtime memory model","artifact":"lib/core/memory.tetra; compiler/internal/actorsrt/linux_x64_emit.go; tools/cmd/memory-production-smoke","evidence":"allocator alloc/free lifecycle, allocator invalid size precondition, allocator failure semantics, and stress allocator reuse cases ran on linux-x64","result":"pass"},
    {"requirement":"ownership/borrow/consume escape model","artifact":"compiler/tests/ownership; compiler/tests/safety","evidence":"borrow escape, use-after-free, double-free, aliasing, callable heap escape, and actor/task transfer diagnostics are required memory production cases","result":"pass"},
    {"requirement":"heap, slices, structs, and closures memory coverage","artifact":"docs/spec/ownership_v1.md; compiler/tests/ownership; compiler/tests/semantics/closures_semantic_clauses_test.go","evidence":"heap closure handle coverage, callable heap escape rejection, slice struct borrow escape coverage, and function-typed slice aggregate borrow escape coverage run compiler tests for closure heap handles, nested slice/struct escapes, and conservative rejection of unsafe escapes","result":"pass"},
    {"requirement":"unsafe/cap.mem/raw memory/memcpy/memset rules","artifact":"docs/spec/unsafe.md; docs/spec/capabilities.md; lib/core/memory.tetra","evidence":"cap.mem unsafe boundary plus memcpy/memset capability path and negative helper length cases require unsafe and explicit cap.mem","result":"pass"},
    {"requirement":"runtime bounds checks and diagnostics","artifact":"docs/spec/runtime_abi.md; compiler/compiler_test.go; tools/cmd/memory-production-smoke","evidence":"slice bounds, ptr_add negative offset, allocation upper bound, i32 width, ptr width, and negative helper length diagnostics are required cases","result":"pass"},
    {"requirement":"raw pointer bounds metadata","artifact":"compiler/internal/runtimeabi/raw_pointer_bounds.go; compiler/internal/plir/plir.go; compiler/internal/allocplan/plan.go; tools/cmd/memory-production-smoke","evidence":"core.alloc_bytes allocation reports include allocation_base_metadata and external_unknown raw-slice policy; PLIR records derived_allocation_offset and checked_external_unknown raw pointer paths","result":"pass"},
    {"requirement":"stress/fuzz evidence","artifact":"tools/cmd/memory-production-smoke","evidence":"stress allocator reuse and deterministic memcpy/memset fuzz cases ran through the release-gate entrypoint","result":"pass"},
    {"requirement":"measured memory benchmark improvement","artifact":"tools/cmd/memory-production-smoke; compiler allocation report schema v2","evidence":"small heap allocation syscall reduction benchmark reads the emitted allocation report, counts per_core_small_heap rows with same_core_same_size_class_free_list reuse policy, and compares estimated mmap-per-allocation baseline against 64KiB chunk refill calls","result":"pass"},
    {"requirement":"use-after-free, double-free, borrow escape, and aliasing safety","artifact":"compiler/tests/safety; compiler/tests/ownership; compiler","evidence":"required compiler safety cases reject use-after-free, double-free, borrow escape, and inout aliasing violations","result":"pass"},
    {"requirement":"actor/task transfer safety","artifact":"compiler/tests/ownership","evidence":"TestReleaseTraceabilityLifetimeAndRaceSafetyNegativeActorTaskOwnership rejects unsafe actor/task transfer boundaries","result":"pass"},
    {"requirement":"real memory examples","artifact":"examples/core_memory_smoke.tetra; examples/ownership_smoke.tetra; examples/flow_unsafe_cap_mem_smoke.tetra","evidence":"checked-in memory, ownership, and unsafe cap.mem examples build and run under the memory production release gate","result":"pass"},
    {"requirement":"safe memory documentation","artifact":"docs/spec/runtime_abi.md; docs/spec/ownership_v1.md; docs/spec/unsafe.md; docs/user/standard_library_guide.md","evidence":"verify-docs requires the Memory Production ABI, ownership extension, unsafe boundary, and writing raw memory safely guide sections","result":"pass"},
    {"requirement":"release-gate entrypoint","artifact":"scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh","evidence":"entrypoint writes memory-production-linux-x64.json and runs memory-production-smoke plus validate-memory-production","result":"pass"}
  ]
}`)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsPaperOnlyMemoryEvidence(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.memory.production.v1",
  "status": "pass",
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "memory-linux-x64",
  "source": "docs-only-placeholder.md",
  "processes": [],
  "contracts": [],
  "cases": []
}`)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected paper-only memory evidence to fail")
	}
	for _, want := range []string{"placeholder", "process", "case"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateBenchmarksRejectsMissingSmallHeapEvidence(t *testing.T) {
	issues := validateBenchmarks([]BenchmarkReport{{
		Name:             "unrelated memory benchmark",
		Kind:             "allocator",
		Metric:           "estimated_os_syscalls",
		Unit:             "syscalls",
		BaselineValue:    2,
		MeasuredValue:    1,
		ImprovementRatio: 2,
		Evidence:         "small_heap_bump comparison evidence",
		Ran:              true,
		Pass:             true,
	}})
	joined := strings.ToLower(strings.Join(issues, "; "))
	if !strings.Contains(joined, "small heap allocation syscall reduction") {
		t.Fatalf("validateBenchmarks issues = %v, want missing small heap benchmark", issues)
	}
}

func TestValidateBenchmarksRejectsLegacySmallHeapEvidenceWithoutPerCoreReuse(t *testing.T) {
	issues := validateBenchmarks([]BenchmarkReport{{
		Name:             "small heap allocation syscall reduction",
		Kind:             "allocator",
		Metric:           "estimated_os_syscalls",
		Unit:             "syscalls",
		BaselineValue:    64,
		MeasuredValue:    1,
		ImprovementRatio: 64,
		Evidence:         "allocation report schema v2 shows 64 small_heap_bump rows reserved inside one 64KiB chunk refill",
		Ran:              true,
		Pass:             true,
	}})
	joined := strings.ToLower(strings.Join(issues, "; "))
	for _, want := range []string{"per_core_small_heap", "same_core_same_size_class_free_list"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("validateBenchmarks issues = %v, want missing %q", issues, want)
		}
	}
}

func TestValidateReportRejectsMissingRawPointerBoundsMetadataEvidence(t *testing.T) {
	issues := validateContracts([]ContractReport{
		{Name: "allocator runtime model", Status: "pass", Evidence: "allocator lifecycle"},
		{Name: "allocator failure semantics", Status: "pass", Evidence: "failure"},
		{Name: "ownership escape model", Status: "pass", Evidence: "ownership"},
		{Name: "unsafe cap.mem raw memory rules", Status: "pass", Evidence: "unsafe cap.mem"},
		{Name: "runtime bounds diagnostics", Status: "pass", Evidence: "bounds diagnostics"},
		{Name: "actor task transfer rules", Status: "pass", Evidence: "actor task"},
	})
	joined := strings.ToLower(strings.Join(issues, "; "))
	if !strings.Contains(joined, "raw pointer bounds metadata") {
		t.Fatalf("validateContracts issues = %v, want missing raw pointer bounds metadata", issues)
	}

	issues = validateCases([]CaseReport{
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
	})
	joined = strings.ToLower(strings.Join(issues, "; "))
	if !strings.Contains(joined, "raw pointer bounds metadata report") {
		t.Fatalf("validateCases issues = %v, want missing raw pointer bounds metadata report", issues)
	}
}

func TestValidateReportRejectsMissingCompletionAudit(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.memory.production.v1",
  "status": "pass",
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "memory-linux-x64",
  "source": "examples/core_memory_smoke.tetra",
  "processes": [
    {"name":"tetra build","kind":"build","path":"/tmp/tetra","ran":true,"pass":true,"exit_code":0},
    {"name":"memory smoke app","kind":"app","path":"/tmp/memory-smoke","ran":true,"pass":true,"exit_code":0},
    {"name":"memory stress","kind":"stress","path":"tools/cmd/memory-production-smoke","ran":true,"pass":true,"exit_code":0}
  ],
  "contracts": [
    {"name":"allocator runtime model","status":"pass","evidence":"allocator lifecycle returns deterministic handles and failure status"},
    {"name":"allocator failure semantics","status":"pass","evidence":"linux-x64 mmap failure exits deterministically before returning an invalid pointer"},
    {"name":"ownership escape model","status":"pass","evidence":"heap, slices, structs, and closures preserve borrow/consume diagnostics"},
    {"name":"unsafe cap.mem raw memory rules","status":"pass","evidence":"raw memory helpers require unsafe and explicit cap.mem"},
    {"name":"runtime bounds diagnostics","status":"pass","evidence":"out-of-bounds memory access reports deterministic runtime diagnostic"},
    {"name":"actor task transfer rules","status":"pass","evidence":"memory-bearing values cannot cross actor/task boundaries without checked transfer"}
  ],
  "cases": [
    {"name":"allocator alloc/free lifecycle","kind":"positive","ran":true,"pass":true},
    {"name":"allocator failure semantics","kind":"negative","ran":true,"pass":true,"expected_error":"allocation failure"},
    {"name":"allocator invalid size precondition","kind":"negative","ran":true,"pass":true,"expected_error":"invalid allocation size"},
    {"name":"memcpy/memset capability path","kind":"positive","ran":true,"pass":true},
    {"name":"runtime bounds check","kind":"negative","ran":true,"pass":true,"expected_error":"bounds"},
    {"name":"raw ptr_add negative offset bounds","kind":"negative","ran":true,"pass":true,"expected_error":"negative ptr_add offset"},
    {"name":"raw ptr_add allocation upper bound","kind":"negative","ran":true,"pass":true,"expected_error":"allocation upper bound"},
    {"name":"raw allocation-base i32 access width","kind":"negative","ran":true,"pass":true,"expected_error":"i32 access width exceeds allocation"},
    {"name":"raw allocation-base ptr access width","kind":"negative","ran":true,"pass":true,"expected_error":"ptr access width exceeds allocation"},
    {"name":"memcpy/memset negative length","kind":"negative","ran":true,"pass":true,"expected_error":"negative helper length"},
    {"name":"reject use-after-free","kind":"negative","ran":true,"pass":true,"expected_error":"use-after-free"},
    {"name":"reject double-free","kind":"negative","ran":true,"pass":true,"expected_error":"double-free"},
    {"name":"reject borrow escape","kind":"negative","ran":true,"pass":true,"expected_error":"borrow escape"},
    {"name":"reject aliasing violation","kind":"negative","ran":true,"pass":true,"expected_error":"alias"},
    {"name":"reject actor task transfer violation","kind":"negative","ran":true,"pass":true,"expected_error":"transfer"},
    {"name":"stress allocator reuse","kind":"stress","ran":true,"pass":true},
    {"name":"deterministic memcpy/memset fuzz","kind":"stress","ran":true,"pass":true}
  ]
}`)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing completion audit to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "completion audit") {
		t.Fatalf("error missing completion audit:\n%v", err)
	}
}

func TestValidateReportRejectsMissingRequiredSafetyCases(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.memory.production.v1",
  "status": "pass",
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "memory-linux-x64",
  "source": "examples/core_memory_smoke.tetra",
  "processes": [
    {"name":"tetra build","kind":"build","path":"/tmp/tetra","ran":true,"pass":true,"exit_code":0},
    {"name":"memory smoke app","kind":"app","path":"/tmp/memory-smoke","ran":true,"pass":true,"exit_code":0}
  ],
  "contracts": [
    {"name":"allocator runtime model","status":"pass","evidence":"allocator lifecycle"},
    {"name":"ownership escape model","status":"pass","evidence":"ownership diagnostics"},
    {"name":"unsafe cap.mem raw memory rules","status":"pass","evidence":"cap.mem checks"},
    {"name":"runtime bounds diagnostics","status":"pass","evidence":"bounds diagnostics"},
    {"name":"actor task transfer rules","status":"pass","evidence":"transfer diagnostics"}
  ],
  "cases": [
    {"name":"allocator alloc/free lifecycle","kind":"positive","ran":true,"pass":true},
    {"name":"memcpy/memset capability path","kind":"positive","ran":true,"pass":true},
    {"name":"runtime bounds check","kind":"negative","ran":true,"pass":true,"expected_error":"bounds"},
    {"name":"stress allocator reuse","kind":"stress","ran":true,"pass":true}
  ]
}`)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing required memory safety cases to fail")
	}
	for _, want := range []string{"use-after-free", "double-free", "borrow escape", "aliasing", "actor task transfer", "cap.mem unsafe boundary", "callable mutable capture heap escape", "function-typed slice aggregate borrow escape", "deterministic memcpy/memset fuzz"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsMissingAllocatorFailureSemantics(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.memory.production.v1",
  "status": "pass",
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "memory-linux-x64",
  "source": "examples/core_memory_smoke.tetra",
  "processes": [
    {"name":"tetra build","kind":"build","path":"/tmp/tetra","ran":true,"pass":true,"exit_code":0},
    {"name":"memory smoke app","kind":"app","path":"/tmp/memory-smoke","ran":true,"pass":true,"exit_code":0},
    {"name":"memory stress","kind":"stress","path":"tools/cmd/memory-production-smoke","ran":true,"pass":true,"exit_code":0}
  ],
  "contracts": [
    {"name":"allocator runtime model","status":"pass","evidence":"allocator lifecycle returns deterministic handles"},
    {"name":"ownership escape model","status":"pass","evidence":"heap, slices, structs, and closures preserve borrow/consume diagnostics"},
    {"name":"unsafe cap.mem raw memory rules","status":"pass","evidence":"raw memory helpers require unsafe and explicit cap.mem"},
    {"name":"runtime bounds diagnostics","status":"pass","evidence":"out-of-bounds memory access reports deterministic runtime diagnostic"},
    {"name":"actor task transfer rules","status":"pass","evidence":"memory-bearing values cannot cross actor/task boundaries without checked transfer"}
  ],
  "cases": [
    {"name":"allocator alloc/free lifecycle","kind":"positive","ran":true,"pass":true},
    {"name":"memcpy/memset capability path","kind":"positive","ran":true,"pass":true},
    {"name":"runtime bounds check","kind":"negative","ran":true,"pass":true,"expected_error":"bounds"},
    {"name":"reject use-after-free","kind":"negative","ran":true,"pass":true,"expected_error":"use-after-free"},
    {"name":"reject double-free","kind":"negative","ran":true,"pass":true,"expected_error":"double-free"},
    {"name":"reject borrow escape","kind":"negative","ran":true,"pass":true,"expected_error":"borrow escape"},
    {"name":"reject aliasing violation","kind":"negative","ran":true,"pass":true,"expected_error":"alias"},
    {"name":"reject actor task transfer violation","kind":"negative","ran":true,"pass":true,"expected_error":"transfer"},
    {"name":"stress allocator reuse","kind":"stress","ran":true,"pass":true}
  ]
}`)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing allocator failure semantics to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "allocator failure semantics") {
		t.Fatalf("error missing allocator failure semantics:\n%v", err)
	}
}

func TestValidateReportRejectsMissingRawPtrAddNegativeOffsetBounds(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.memory.production.v1",
  "status": "pass",
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "memory-linux-x64",
  "source": "examples/core_memory_smoke.tetra",
  "processes": [
    {"name":"tetra build","kind":"build","path":"/tmp/tetra","ran":true,"pass":true,"exit_code":0},
    {"name":"memory smoke app","kind":"app","path":"/tmp/memory-smoke","ran":true,"pass":true,"exit_code":0},
    {"name":"memory stress","kind":"stress","path":"tools/cmd/memory-production-smoke","ran":true,"pass":true,"exit_code":0}
  ],
  "contracts": [
    {"name":"allocator runtime model","status":"pass","evidence":"allocator lifecycle returns deterministic handles and failure status"},
    {"name":"allocator failure semantics","status":"pass","evidence":"linux-x64 mmap failure exits deterministically before returning an invalid pointer"},
    {"name":"ownership escape model","status":"pass","evidence":"heap, slices, structs, and closures preserve borrow/consume diagnostics"},
    {"name":"unsafe cap.mem raw memory rules","status":"pass","evidence":"raw memory helpers require unsafe and explicit cap.mem"},
    {"name":"runtime bounds diagnostics","status":"pass","evidence":"out-of-bounds memory access reports deterministic runtime diagnostic"},
    {"name":"actor task transfer rules","status":"pass","evidence":"memory-bearing values cannot cross actor/task boundaries without checked transfer"}
  ],
  "cases": [
    {"name":"allocator alloc/free lifecycle","kind":"positive","ran":true,"pass":true},
    {"name":"allocator failure semantics","kind":"negative","ran":true,"pass":true,"expected_error":"allocation failure"},
    {"name":"memcpy/memset capability path","kind":"positive","ran":true,"pass":true},
    {"name":"runtime bounds check","kind":"negative","ran":true,"pass":true,"expected_error":"bounds"},
    {"name":"reject use-after-free","kind":"negative","ran":true,"pass":true,"expected_error":"use-after-free"},
    {"name":"reject double-free","kind":"negative","ran":true,"pass":true,"expected_error":"double-free"},
    {"name":"reject borrow escape","kind":"negative","ran":true,"pass":true,"expected_error":"borrow escape"},
    {"name":"reject aliasing violation","kind":"negative","ran":true,"pass":true,"expected_error":"alias"},
    {"name":"reject actor task transfer violation","kind":"negative","ran":true,"pass":true,"expected_error":"transfer"},
    {"name":"stress allocator reuse","kind":"stress","ran":true,"pass":true}
  ]
}`)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing raw ptr_add negative offset bounds case to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "raw ptr_add negative offset bounds") {
		t.Fatalf("error missing raw ptr_add negative offset bounds:\n%v", err)
	}
}

func TestValidateReportRejectsMissingRawPtrAddAllocationUpperBound(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.memory.production.v1",
  "status": "pass",
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "memory-linux-x64",
  "source": "examples/core_memory_smoke.tetra",
  "processes": [
    {"name":"tetra build","kind":"build","path":"/tmp/tetra","ran":true,"pass":true,"exit_code":0},
    {"name":"memory smoke app","kind":"app","path":"/tmp/memory-smoke","ran":true,"pass":true,"exit_code":0},
    {"name":"memory stress","kind":"stress","path":"tools/cmd/memory-production-smoke","ran":true,"pass":true,"exit_code":0}
  ],
  "contracts": [
    {"name":"allocator runtime model","status":"pass","evidence":"allocator lifecycle returns deterministic handles and failure status"},
    {"name":"allocator failure semantics","status":"pass","evidence":"linux-x64 mmap failure exits deterministically before returning an invalid pointer"},
    {"name":"ownership escape model","status":"pass","evidence":"heap, slices, structs, and closures preserve borrow/consume diagnostics"},
    {"name":"unsafe cap.mem raw memory rules","status":"pass","evidence":"raw memory helpers require unsafe and explicit cap.mem"},
    {"name":"runtime bounds diagnostics","status":"pass","evidence":"out-of-bounds memory access reports deterministic runtime diagnostic"},
    {"name":"actor task transfer rules","status":"pass","evidence":"memory-bearing values cannot cross actor/task boundaries without checked transfer"}
  ],
  "cases": [
    {"name":"allocator alloc/free lifecycle","kind":"positive","ran":true,"pass":true},
    {"name":"allocator failure semantics","kind":"negative","ran":true,"pass":true,"expected_error":"allocation failure"},
    {"name":"memcpy/memset capability path","kind":"positive","ran":true,"pass":true},
    {"name":"runtime bounds check","kind":"negative","ran":true,"pass":true,"expected_error":"bounds"},
    {"name":"raw ptr_add negative offset bounds","kind":"negative","ran":true,"pass":true,"expected_error":"negative ptr_add offset"},
    {"name":"reject use-after-free","kind":"negative","ran":true,"pass":true,"expected_error":"use-after-free"},
    {"name":"reject double-free","kind":"negative","ran":true,"pass":true,"expected_error":"double-free"},
    {"name":"reject borrow escape","kind":"negative","ran":true,"pass":true,"expected_error":"borrow escape"},
    {"name":"reject aliasing violation","kind":"negative","ran":true,"pass":true,"expected_error":"alias"},
    {"name":"reject actor task transfer violation","kind":"negative","ran":true,"pass":true,"expected_error":"transfer"},
    {"name":"stress allocator reuse","kind":"stress","ran":true,"pass":true}
  ]
}`)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing raw ptr_add allocation upper bound case to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "raw ptr_add allocation upper bound") {
		t.Fatalf("error missing raw ptr_add allocation upper bound:\n%v", err)
	}
}

func TestValidateReportRejectsMissingAllocatorInvalidSizePrecondition(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.memory.production.v1",
  "status": "pass",
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "memory-linux-x64",
  "source": "examples/core_memory_smoke.tetra",
  "processes": [
    {"name":"tetra build","kind":"build","path":"/tmp/tetra","ran":true,"pass":true,"exit_code":0},
    {"name":"memory smoke app","kind":"app","path":"/tmp/memory-smoke","ran":true,"pass":true,"exit_code":0},
    {"name":"memory stress","kind":"stress","path":"tools/cmd/memory-production-smoke","ran":true,"pass":true,"exit_code":0}
  ],
  "contracts": [
    {"name":"allocator runtime model","status":"pass","evidence":"allocator lifecycle returns deterministic handles and failure status"},
    {"name":"allocator failure semantics","status":"pass","evidence":"linux-x64 mmap failure exits deterministically before returning an invalid pointer"},
    {"name":"ownership escape model","status":"pass","evidence":"heap, slices, structs, and closures preserve borrow/consume diagnostics"},
    {"name":"unsafe cap.mem raw memory rules","status":"pass","evidence":"raw memory helpers require unsafe and explicit cap.mem"},
    {"name":"runtime bounds diagnostics","status":"pass","evidence":"out-of-bounds memory access reports deterministic runtime diagnostic"},
    {"name":"actor task transfer rules","status":"pass","evidence":"memory-bearing values cannot cross actor/task boundaries without checked transfer"}
  ],
  "cases": [
    {"name":"allocator alloc/free lifecycle","kind":"positive","ran":true,"pass":true},
    {"name":"allocator failure semantics","kind":"negative","ran":true,"pass":true,"expected_error":"allocation failure"},
    {"name":"memcpy/memset capability path","kind":"positive","ran":true,"pass":true},
    {"name":"runtime bounds check","kind":"negative","ran":true,"pass":true,"expected_error":"bounds"},
    {"name":"raw ptr_add negative offset bounds","kind":"negative","ran":true,"pass":true,"expected_error":"negative ptr_add offset"},
    {"name":"raw ptr_add allocation upper bound","kind":"negative","ran":true,"pass":true,"expected_error":"allocation upper bound"},
    {"name":"reject use-after-free","kind":"negative","ran":true,"pass":true,"expected_error":"use-after-free"},
    {"name":"reject double-free","kind":"negative","ran":true,"pass":true,"expected_error":"double-free"},
    {"name":"reject borrow escape","kind":"negative","ran":true,"pass":true,"expected_error":"borrow escape"},
    {"name":"reject aliasing violation","kind":"negative","ran":true,"pass":true,"expected_error":"alias"},
    {"name":"reject actor task transfer violation","kind":"negative","ran":true,"pass":true,"expected_error":"transfer"},
    {"name":"stress allocator reuse","kind":"stress","ran":true,"pass":true}
  ]
}`)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing allocator invalid size precondition case to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "allocator invalid size precondition") {
		t.Fatalf("error missing allocator invalid size precondition:\n%v", err)
	}
}
