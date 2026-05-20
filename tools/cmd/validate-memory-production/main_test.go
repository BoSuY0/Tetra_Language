package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateMemoryProductionReportAcceptsValidReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	if err := os.WriteFile(path, []byte(validMemoryProductionReport()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateMemoryProductionReport(path); err != nil {
		t.Fatalf("validateMemoryProductionReport failed: %v", err)
	}
}

func TestValidateMemoryProductionReportRejectsInvalidReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	raw := strings.Replace(validMemoryProductionReport(), `"schema": "tetra.memory.production.v1"`, `"schema": "tetra.memory.fake.v1"`, 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryProductionReport(path)
	if err == nil {
		t.Fatalf("expected invalid memory production report to fail")
	}
	if !strings.Contains(err.Error(), "tetra.memory.production.v1") {
		t.Fatalf("error = %v, want schema rejection", err)
	}
}

func TestValidateMemoryProductionReportRejectsMissingRealMemoryExamplesCase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	raw := strings.Replace(validMemoryProductionReport(), `    {"name":"real memory examples","kind":"positive","ran":true,"pass":true},
`, "", 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryProductionReport(path)
	if err == nil {
		t.Fatalf("expected missing real memory examples case to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "real memory examples") {
		t.Fatalf("error = %v, want real memory examples rejection", err)
	}
}

func TestValidateMemoryProductionReportRejectsMissingRealMemoryExamplesAudit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	raw := strings.Replace(validMemoryProductionReport(), `    {"requirement":"real memory examples","artifact":"examples/core_memory_smoke.tetra; examples/ownership_smoke.tetra; examples/flow_unsafe_cap_mem_smoke.tetra","evidence":"checked-in memory, ownership, and unsafe cap.mem examples build and run under the memory production release gate","result":"pass"},
`, "", 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryProductionReport(path)
	if err == nil {
		t.Fatalf("expected missing real memory examples audit to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "real memory examples") {
		t.Fatalf("error = %v, want real memory examples rejection", err)
	}
}

func TestValidateMemoryProductionReportRejectsMissingHeapClosureHandleCoverageCase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	raw := strings.Replace(validMemoryProductionReport(), `    {"name":"heap closure handle coverage","kind":"positive","ran":true,"pass":true},
`, "", 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryProductionReport(path)
	if err == nil {
		t.Fatalf("expected missing heap closure handle coverage case to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "heap closure handle coverage") {
		t.Fatalf("error = %v, want heap closure handle coverage rejection", err)
	}
}

func TestValidateMemoryProductionReportRejectsMissingSliceStructBorrowEscapeCoverageCase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	raw := strings.Replace(validMemoryProductionReport(), `    {"name":"slice struct borrow escape coverage","kind":"negative","ran":true,"pass":true,"expected_error":"borrow escape"},
`, "", 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryProductionReport(path)
	if err == nil {
		t.Fatalf("expected missing slice struct borrow escape coverage case to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "slice struct borrow escape coverage") {
		t.Fatalf("error = %v, want slice struct borrow escape coverage rejection", err)
	}
}

func TestValidateMemoryProductionReportRejectsMissingCapMemUnsafeBoundaryCase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	raw := strings.Replace(validMemoryProductionReport(), `    {"name":"cap.mem unsafe boundary","kind":"negative","ran":true,"pass":true,"expected_error":"only allowed in unsafe blocks"},
`, "", 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryProductionReport(path)
	if err == nil {
		t.Fatalf("expected missing cap.mem unsafe boundary case to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "cap.mem unsafe boundary") {
		t.Fatalf("error = %v, want cap.mem unsafe boundary rejection", err)
	}
}

func TestValidateMemoryProductionReportRejectsMissingCallableMutableCaptureHeapEscapeCase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	raw := strings.Replace(validMemoryProductionReport(), `    {"name":"callable mutable capture heap escape","kind":"negative","ran":true,"pass":true,"expected_error":"heap-escaped function value captures mutable local"},
`, "", 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryProductionReport(path)
	if err == nil {
		t.Fatalf("expected missing callable mutable capture heap escape case to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "callable mutable capture heap escape") {
		t.Fatalf("error = %v, want callable mutable capture heap escape rejection", err)
	}
}

func TestValidateMemoryProductionReportRejectsMissingFunctionTypedSliceAggregateBorrowEscapeCoverageCase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	raw := strings.Replace(validMemoryProductionReport(), `    {"name":"function-typed slice aggregate borrow escape coverage","kind":"negative","ran":true,"pass":true,"expected_error":"borrow escape"},
`, "", 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryProductionReport(path)
	if err == nil {
		t.Fatalf("expected missing function-typed slice aggregate borrow escape coverage case to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "function-typed slice aggregate borrow escape coverage") {
		t.Fatalf("error = %v, want function-typed slice aggregate borrow escape coverage rejection", err)
	}
}

func validMemoryProductionReport() string {
	return `{
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
    {"name":"cap.mem unsafe boundary","kind":"negative","ran":true,"pass":true,"expected_error":"only allowed in unsafe blocks"},
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
    {"requirement":"stress/fuzz evidence","artifact":"tools/cmd/memory-production-smoke","evidence":"stress allocator reuse and deterministic memcpy/memset fuzz cases ran through the release-gate entrypoint","result":"pass"},
    {"requirement":"use-after-free, double-free, borrow escape, and aliasing safety","artifact":"compiler/tests/safety; compiler/tests/ownership; compiler","evidence":"required compiler safety cases reject use-after-free, double-free, borrow escape, and inout aliasing violations","result":"pass"},
    {"requirement":"actor/task transfer safety","artifact":"compiler/tests/ownership","evidence":"TestReleaseTraceabilityLifetimeAndRaceSafetyNegativeActorTaskOwnership rejects unsafe actor/task transfer boundaries","result":"pass"},
    {"requirement":"real memory examples","artifact":"examples/core_memory_smoke.tetra; examples/ownership_smoke.tetra; examples/flow_unsafe_cap_mem_smoke.tetra","evidence":"checked-in memory, ownership, and unsafe cap.mem examples build and run under the memory production release gate","result":"pass"},
    {"requirement":"safe memory documentation","artifact":"docs/spec/runtime_abi.md; docs/spec/ownership_v1.md; docs/spec/unsafe.md; docs/user/standard_library_guide.md","evidence":"verify-docs requires the Memory Production ABI, ownership extension, unsafe boundary, and writing raw memory safely guide sections","result":"pass"},
    {"requirement":"release-gate entrypoint","artifact":"scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh","evidence":"entrypoint writes memory-production-linux-x64.json and runs memory-production-smoke plus validate-memory-production","result":"pass"}
  ]
}`
}
