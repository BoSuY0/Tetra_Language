package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
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

func TestValidateMemoryProductionReleaseManifestAcceptsFreshProvenance(t *testing.T) {
	reportDir, reportPath, manifestPath := writeMemoryProductionReleaseFixture(t, nil)
	if err := validateMemoryProductionReleaseManifest(reportPath, manifestPath, reportDir); err != nil {
		t.Fatalf("validateMemoryProductionReleaseManifest failed: %v", err)
	}
}

func TestValidateMemoryProductionReleaseManifestRejectsHashMismatchedArtifact(t *testing.T) {
	reportDir, reportPath, manifestPath := writeMemoryProductionReleaseFixture(t, nil)
	summaryPath := filepath.Join(reportDir, "memory-fuzz-tier1", "summary.json")
	if err := os.WriteFile(summaryPath, []byte(`{"schema_version":"tetra.memory-fuzz-short.summary.v1","tier":1,"status":"tampered"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryProductionReleaseManifest(reportPath, manifestPath, reportDir)
	if err == nil {
		t.Fatalf("expected hash-mismatched release artifact to fail")
	}
	if !strings.Contains(err.Error(), "sha256 mismatch") || !strings.Contains(err.Error(), "memory-fuzz-tier1/summary.json") {
		t.Fatalf("error = %v, want summary hash mismatch", err)
	}
}

func TestValidateMemoryProductionReleaseManifestRejectsMissingGeneratorCommand(t *testing.T) {
	reportDir, reportPath, manifestPath := writeMemoryProductionReleaseFixture(t, func(manifest *memoryReleaseTestManifest) {
		for i := range manifest.Artifacts {
			if manifest.Artifacts[i].Kind == "memory_production_report" {
				manifest.Artifacts[i].Command = ""
			}
		}
	})
	err := validateMemoryProductionReleaseManifest(reportPath, manifestPath, reportDir)
	if err == nil {
		t.Fatalf("expected missing generator command to fail")
	}
	if !strings.Contains(err.Error(), "memory_production_report command is required") {
		t.Fatalf("error = %v, want generator command rejection", err)
	}
}

func TestValidateMemoryProductionReleaseManifestRejectsMissingHashEntry(t *testing.T) {
	reportDir, reportPath, manifestPath := writeMemoryProductionReleaseFixture(t, nil)
	writeMemoryReleaseTestHashManifest(t, reportDir, []string{
		"memory-production-linux-x64.json",
		"memory-fuzz-tier1/memory-fuzz-oracle.json",
		"memory-fuzz-tier1/summary.json",
		"memory-release-manifest.json",
	})
	err := validateMemoryProductionReleaseManifest(reportPath, manifestPath, reportDir)
	if err == nil {
		t.Fatalf("expected missing hash manifest entry to fail")
	}
	if !strings.Contains(err.Error(), "missing hash manifest entry for targets.json") {
		t.Fatalf("error = %v, want missing targets hash entry", err)
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
}`
}

type memoryReleaseTestManifest struct {
	Schema       string                      `json:"schema"`
	Target       string                      `json:"target"`
	GitHead      string                      `json:"git_head"`
	GeneratedAt  string                      `json:"generated_at"`
	ReportDir    string                      `json:"report_dir"`
	HashManifest string                      `json:"hash_manifest"`
	Commands     []memoryReleaseTestCommand  `json:"commands"`
	Artifacts    []memoryReleaseTestArtifact `json:"artifacts"`
}

type memoryReleaseTestCommand struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

type memoryReleaseTestArtifact struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Schema  string `json:"schema,omitempty"`
	Target  string `json:"target"`
	Command string `json:"command"`
}

type memoryReleaseTestHashManifest struct {
	Schema    string                          `json:"schema"`
	Root      string                          `json:"root"`
	Artifacts []memoryReleaseTestHashArtifact `json:"artifacts"`
}

type memoryReleaseTestHashArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Schema string `json:"schema,omitempty"`
}

func writeMemoryProductionReleaseFixture(t *testing.T, mutate func(*memoryReleaseTestManifest)) (string, string, string) {
	t.Helper()
	reportDir := t.TempDir()
	fuzzDir := filepath.Join(reportDir, "memory-fuzz-tier1")
	if err := os.MkdirAll(fuzzDir, 0o755); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(reportDir, "memory-production-linux-x64.json")
	if err := os.WriteFile(reportPath, []byte(validMemoryProductionReport()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "targets.json"), []byte(`[
  {"triple":"linux-x64","status":"supported","memory_claim_level":"production/host_runtime"}
]`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fuzzDir, "memory-fuzz-oracle.json"), []byte(`{"schema_version":"tetra.memory-fuzz.oracle.v1","tier":1,"target":"linux-x64"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fuzzDir, "summary.json"), []byte(`{"schema_version":"tetra.memory-fuzz-short.summary.v1","tier":1,"status":"pass"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := memoryReleaseTestManifest{
		Schema:       "tetra.memory.release-manifest.v1",
		Target:       "linux-x64",
		GitHead:      "0123456789abcdef0123456789abcdef01234567",
		GeneratedAt:  "2026-06-07T20:15:00Z",
		ReportDir:    ".",
		HashManifest: "artifact-hashes.json",
		Commands: []memoryReleaseTestCommand{
			{Name: "memory-production-smoke", Command: "go run ./tools/cmd/memory-production-smoke --report $report_path"},
			{Name: "target-report", Command: "go run ./cli/cmd/tetra targets --format=json > $targets_path"},
			{Name: "validate-targets", Command: "go run ./tools/cmd/validate-targets --report $targets_path"},
			{Name: "memory-fuzz-short", Command: "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir $memory_fuzz_dir"},
			{Name: "validate-memory-fuzz-oracle", Command: "go run ./tools/cmd/validate-memory-fuzz-oracle --report $memory_fuzz_dir/memory-fuzz-oracle.json --artifact-dir $memory_fuzz_dir"},
			{Name: "artifact-hashes-write", Command: "go run ./tools/cmd/validate-artifact-hashes --write --root $report_dir --out $report_dir/artifact-hashes.json"},
			{Name: "artifact-hashes-validate", Command: "go run ./tools/cmd/validate-artifact-hashes --manifest $report_dir/artifact-hashes.json"},
		},
		Artifacts: []memoryReleaseTestArtifact{
			{Path: "memory-production-linux-x64.json", Kind: "memory_production_report", Schema: "tetra.memory.production.v1", Target: "linux-x64", Command: "go run ./tools/cmd/memory-production-smoke --report $report_path"},
			{Path: "targets.json", Kind: "target_report", Target: "linux-x64", Command: "go run ./cli/cmd/tetra targets --format=json > $targets_path"},
			{Path: "memory-fuzz-tier1/memory-fuzz-oracle.json", Kind: "memory_fuzz_oracle_report", Schema: "tetra.memory-fuzz.oracle.v1", Target: "linux-x64", Command: "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir $memory_fuzz_dir"},
			{Path: "memory-fuzz-tier1/summary.json", Kind: "memory_fuzz_summary", Schema: "tetra.memory-fuzz-short.summary.v1", Target: "linux-x64", Command: "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir $memory_fuzz_dir"},
			{Path: "artifact-hashes.json", Kind: "artifact_hash_manifest", Schema: "tetra.release-artifact-hashes.v1alpha1", Target: "linux-x64", Command: "go run ./tools/cmd/validate-artifact-hashes --write --root $report_dir --out $report_dir/artifact-hashes.json"},
		},
	}
	if mutate != nil {
		mutate(&manifest)
	}
	manifestPath := filepath.Join(reportDir, "memory-release-manifest.json")
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	writeMemoryReleaseTestHashManifest(t, reportDir, []string{
		"memory-production-linux-x64.json",
		"targets.json",
		"memory-fuzz-tier1/memory-fuzz-oracle.json",
		"memory-fuzz-tier1/summary.json",
		"memory-release-manifest.json",
	})
	return reportDir, reportPath, manifestPath
}

func writeMemoryReleaseTestHashManifest(t *testing.T, root string, paths []string) {
	t.Helper()
	sort.Strings(paths)
	manifest := memoryReleaseTestHashManifest{
		Schema: "tetra.release-artifact-hashes.v1alpha1",
		Root:   ".",
	}
	for _, rel := range paths {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		manifest.Artifacts = append(manifest.Artifacts, memoryReleaseTestHashArtifact{
			Path:   rel,
			SHA256: "sha256:" + hex.EncodeToString(sum[:]),
			Size:   int64(len(raw)),
			Schema: memoryReleaseTestJSONSchema(raw),
		})
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(root, "artifact-hashes.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func memoryReleaseTestJSONSchema(raw []byte) string {
	var envelope struct {
		Schema        string `json:"schema"`
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ""
	}
	if envelope.Schema != "" {
		return envelope.Schema
	}
	return envelope.SchemaVersion
}
