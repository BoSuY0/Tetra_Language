package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"tetra_language/compiler"
	"tetra_language/tools/internal/heaptelemetry"
	"tetra_language/tools/internal/rsstelemetry"
	"tetra_language/tools/internal/zeroheapbench"
	"time"
)

func TestBuildSpecsCoversP20MatrixAndRequiredCompilers(t *testing.T) {
	specs := buildBenchmarkSpecs("reports/local-benchmark-tier1-v1")
	wantRows := len(requiredP20Categories) * len(requiredLanguages)
	if len(specs) != wantRows {
		t.Fatalf("specs = %d, want %d", len(specs), wantRows)
	}
	seen := map[string]bool{}
	for _, spec := range specs {
		key := spec.Category + "\x00" + spec.Language
		if seen[key] {
			t.Fatalf("duplicate spec for %s/%s", spec.Category, spec.Language)
		}
		seen[key] = true
		if spec.AlgorithmID == "" || spec.InputDescription == "" || spec.Source == "" {
			t.Fatalf("spec %s missing equivalence/source metadata: %#v", spec.Name, spec)
		}
		switch spec.Language {
		case "tetra":
			if spec.BuildCommandKind != "tetra" {
				t.Fatalf("tetra spec %s build kind = %q", spec.Name, spec.BuildCommandKind)
			}
		case "c":
			if !containsSequence(spec.BuildArgs, "clang", "-O3") {
				t.Fatalf("c spec %s build args = %#v, want clang -O3", spec.Name, spec.BuildArgs)
			}
		case "cpp":
			if !containsSequence(spec.BuildArgs, "clang++", "-O3") {
				t.Fatalf("cpp spec %s build args = %#v, want clang++ -O3", spec.Name, spec.BuildArgs)
			}
		case "rust":
			if !containsSequence(spec.BuildArgs, "rustc", "-C", "opt-level=3") {
				t.Fatalf("rust spec %s build args = %#v, want rustc -C opt-level=3", spec.Name, spec.BuildArgs)
			}
		default:
			t.Fatalf("unexpected language %q", spec.Language)
		}
	}
	for _, category := range requiredP20Categories {
		for _, language := range requiredLanguages {
			if !seen[category+"\x00"+language] {
				t.Fatalf("missing spec for %s/%s", category, language)
			}
		}
	}
}

func TestZeroHeapMicrobenchSpecsStayOutsideTier1Matrix(t *testing.T) {
	specs := zeroheapbench.BuildSpecs("reports/local-zero-heap-benchmark-v1")
	if len(specs) == 0 {
		t.Fatalf("zero-heap microbenchmark specs are empty")
	}
	if len(specs) != len(zeroheapbench.Categories) {
		t.Fatalf("zero-heap specs = %d, want one Tetra spec per category %d", len(specs), len(zeroheapbench.Categories))
	}

	p20 := map[string]bool{}
	for _, category := range requiredP20Categories {
		p20[category] = true
	}
	seen := map[string]bool{}
	for _, spec := range specs {
		if spec.Language != "tetra" {
			t.Fatalf("zero-heap spec %s language = %q, want tetra-only", spec.Name, spec.Language)
		}
		if p20[spec.Category] {
			t.Fatalf("zero-heap spec %q must stay outside Tier 1 P20 matrix", spec.Category)
		}
		if seen[spec.Category] {
			t.Fatalf("duplicate zero-heap category %q", spec.Category)
		}
		seen[spec.Category] = true
		if spec.AlgorithmID == "" || spec.InputDescription == "" || spec.Source == "" {
			t.Fatalf("zero-heap spec %s missing metadata/source: %#v", spec.Name, spec)
		}
		if !containsSequence(spec.BuildArgs, "tetra", "build", "--target", "linux-x64", "--explain") {
			t.Fatalf("zero-heap spec %s build metadata = %#v, want tetra explain build", spec.Name, spec.BuildArgs)
		}
		if !strings.Contains(spec.SourceRelPath, "zero_heap") {
			t.Fatalf("zero-heap spec %s source path = %q, want zero_heap module artifact path", spec.Name, spec.SourceRelPath)
		}
	}
	for _, category := range zeroheapbench.Categories {
		if !seen[category] {
			t.Fatalf("missing zero-heap microbenchmark category %q", category)
		}
	}
}

func TestBuildCommandEnablesTetraRuntimeHeapTelemetry(t *testing.T) {
	spec := benchmarkSpec{
		Language:      "tetra",
		SourceRelPath: "src/main.tetra",
		BinaryRelPath: "bin/main",
	}
	cmd := buildCommand(spec, "tetra", "artifacts/heap-telemetry/main/runtime")
	if !containsSequence(cmd, "--emit-runtime-heap-telemetry", "--runtime-heap-telemetry-dir", "artifacts/heap-telemetry/main/runtime") {
		t.Fatalf("tetra build command = %#v, want runtime heap telemetry flags", cmd)
	}
}

func TestIntegerLoopsTetraSourceBuildsWithRegisterModuloLoopBackend(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "p25", "integer_loops.tetra")
	outPath := filepath.Join(dir, "integer_loops")
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatalf("create integer_loops module dir: %v", err)
	}
	if err := os.WriteFile(srcPath, []byte(tetraSource("integer loops")), 0o644); err != nil {
		t.Fatalf("write integer_loops source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:    1,
		Explain: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt(integer loops): %v", err)
	}

	raw, err := os.ReadFile(outPath + ".backend.json")
	if err != nil {
		t.Fatalf("read backend report: %v", err)
	}
	var report struct {
		Functions []struct {
			Function    string `json:"function"`
			BackendPath string `json:"backend_path"`
			Category    string `json:"category"`
			Detail      string `json:"detail"`
			Reason      string `json:"reason"`
		} `json:"functions"`
		MachineFunctions []struct {
			Function             string   `json:"function"`
			Path                 string   `json:"path"`
			SSAPath              string   `json:"ssa_path"`
			SSAVerified          bool     `json:"ssa_verified"`
			InstructionSelection []string `json:"instruction_selection"`
			Validation           struct {
				MachineVerifier    string `json:"machine_verifier"`
				AllocationVerifier string `json:"allocation_verifier"`
				StackChurnOps      int    `json:"stack_churn_ops"`
			} `json:"validation"`
		} `json:"machine_functions"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("parse backend report: %v\n%s", err, raw)
	}
	var foundRow bool
	for _, row := range report.Functions {
		if row.Function != "p25.integer_loops.main" {
			continue
		}
		foundRow = true
		if row.BackendPath != "register" || row.Category != "register_path" ||
			row.Detail != "machine-ir-const-modulo-loop" || row.Reason != "eligible_machine_ir_subset" {
			t.Fatalf("integer_loops backend row = %+v, want register const-modulo machine path", row)
		}
	}
	if !foundRow {
		t.Fatalf("integer_loops backend row missing: %+v", report.Functions)
	}
	for _, row := range report.MachineFunctions {
		if row.Function != "p25.integer_loops.main" {
			continue
		}
		if row.Path != "machine-ir-const-modulo-loop" || !row.SSAVerified || row.SSAPath != "value-ssa-v1" {
			t.Fatalf("integer_loops machine row = %+v, want verified const-modulo path", row)
		}
		if !containsString(row.InstructionSelection, "mod") {
			t.Fatalf("integer_loops instruction selection = %+v, want mod evidence", row.InstructionSelection)
		}
		if row.Validation.MachineVerifier != "pass" || row.Validation.AllocationVerifier != "pass" || row.Validation.StackChurnOps != 0 {
			t.Fatalf("integer_loops validation = %+v, want verifier pass and zero stack churn", row.Validation)
		}
		return
	}
	t.Fatalf("integer_loops machine report missing: %+v", report.MachineFunctions)
}

func TestRecursionTetraSourceBuildsWithRegisterRecursionBackend(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "p25", "recursion.tetra")
	outPath := filepath.Join(dir, "recursion")
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatalf("create recursion module dir: %v", err)
	}
	if err := os.WriteFile(srcPath, []byte(tetraSource("recursion")), 0o644); err != nil {
		t.Fatalf("write recursion source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:    1,
		Explain: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt(recursion): %v", err)
	}

	raw, err := os.ReadFile(outPath + ".backend.json")
	if err != nil {
		t.Fatalf("read backend report: %v", err)
	}
	var report struct {
		Functions []struct {
			Function    string `json:"function"`
			BackendPath string `json:"backend_path"`
			Category    string `json:"category"`
			Detail      string `json:"detail"`
			Reason      string `json:"reason"`
		} `json:"functions"`
		MachineFunctions []struct {
			Function             string   `json:"function"`
			Path                 string   `json:"path"`
			SSAPath              string   `json:"ssa_path"`
			SSAVerified          bool     `json:"ssa_verified"`
			InstructionSelection []string `json:"instruction_selection"`
			Validation           struct {
				MachineVerifier    string `json:"machine_verifier"`
				AllocationVerifier string `json:"allocation_verifier"`
				CallClobbers       string `json:"call_clobbers"`
				StackChurnOps      int    `json:"stack_churn_ops"`
			} `json:"validation"`
		} `json:"machine_functions"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("parse backend report: %v\n%s", err, raw)
	}
	wantFunctions := map[string]string{
		"p25.recursion.fib":  "machine-ir-recursive-fib",
		"p25.recursion.main": "machine-ir-recursion-main-loop",
	}
	seenRows := map[string]bool{}
	for _, row := range report.Functions {
		wantDetail, ok := wantFunctions[row.Function]
		if !ok {
			continue
		}
		seenRows[row.Function] = true
		if row.BackendPath != "register" || row.Category != "register_path" ||
			row.Detail != wantDetail || row.Reason != "eligible_machine_ir_subset" {
			t.Fatalf("recursion backend row = %+v, want register %s machine path", row, wantDetail)
		}
	}
	for name := range wantFunctions {
		if !seenRows[name] {
			t.Fatalf("recursion backend row missing for %s: %+v", name, report.Functions)
		}
	}
	seenMachine := map[string]bool{}
	for _, row := range report.MachineFunctions {
		wantPath, ok := wantFunctions[row.Function]
		if !ok {
			continue
		}
		seenMachine[row.Function] = true
		if row.Path != wantPath || !row.SSAVerified || row.SSAPath != "value-ssa-v1" {
			t.Fatalf("recursion machine row = %+v, want verified %s", row, wantPath)
		}
		if !containsString(row.InstructionSelection, "call") {
			t.Fatalf("recursion instruction selection = %+v, want call evidence", row.InstructionSelection)
		}
		if row.Validation.MachineVerifier != "pass" ||
			row.Validation.AllocationVerifier != "pass" ||
			row.Validation.CallClobbers != "validated" ||
			row.Validation.StackChurnOps != 0 {
			t.Fatalf("recursion validation = %+v, want verifier/allocation/clobber pass and zero stack churn", row.Validation)
		}
	}
	for name := range wantFunctions {
		if !seenMachine[name] {
			t.Fatalf("recursion machine report missing for %s: %+v", name, report.MachineFunctions)
		}
	}
}

func TestCollectTetraMetadataFiltersResolvedLocalCallHeapFallbackPerfBlocker(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "hash_table_tetra")
	mustWrite(t, binary+".alloc.json", `{
  "totals": {"heap": 0},
  "summary": {
    "bytes_requested": 2048,
    "bytes_reserved": 2048,
    "domains": []
  }
}`)
	mustWrite(t, binary+".bounds.json", `{"totals":{"left":0}}`)
	mustWrite(t, binary+".backend.json", `{"summary":{"register_path":0,"stack_fallback":1,"categories":{"unsupported_control_flow":1}}}`)
	mustWrite(t, binary+".perf.json", `{
  "benchmarks": [{
    "benchmark": "hash_table_tetra",
    "reason_codes": ["allocation.local_call_heap_fallback", "inline.code_size_budget"]
  }]
}`)

	metadata := collectTetraMetadata("hash_table_tetra", binary, filepath.Join(dir, "optimizer.json"), nil, nil)
	if containsString(metadata.PerfBlockers, "allocation.local_call_heap_fallback") {
		t.Fatalf("resolved heap fallback blocker was retained: %#v", metadata.PerfBlockers)
	}
	if !containsString(metadata.PerfBlockers, "inline.code_size_budget") {
		t.Fatalf("non-allocation perf blocker was filtered: %#v", metadata.PerfBlockers)
	}
}

func TestClassifyCategoryPrefersEvidenceBlockers(t *testing.T) {
	tetra := benchmarkRow{Language: "tetra", Status: "measured", MedianRuntimeMS: 1, TetraMetadata: &tetraMetadata{BackendPath: "fallback"}}
	rows := []benchmarkRow{
		tetra,
		{Language: "c", Status: "measured", MedianRuntimeMS: 10},
		{Language: "cpp", Status: "measured", MedianRuntimeMS: 10},
		{Language: "rust", Status: "measured", MedianRuntimeMS: 10},
	}
	classification, _ := classifyCategory("integer loops", rows, 0.20)
	if classification != "blocked by fallback backend" {
		t.Fatalf("fallback classification = %q", classification)
	}

	rows[0].TetraMetadata = &tetraMetadata{BackendPath: "register", HeapAllocations: 1}
	classification, _ = classifyCategory("allocation", rows, 0.20)
	if classification != "blocked by heap allocation" {
		t.Fatalf("heap classification = %q", classification)
	}

	rows[0].TetraMetadata = &tetraMetadata{BackendPath: "register", BoundsLeft: 1}
	classification, _ = classifyCategory("slice sum", rows, 0.20)
	if classification != "blocked by bounds check" {
		t.Fatalf("bounds classification = %q", classification)
	}

	rows[0].TetraMetadata = &tetraMetadata{BackendPath: "register"}
	classification, _ = classifyCategory("actor ping-pong", rows, 0.20)
	if classification != "blocked by actor/runtime limitation" {
		t.Fatalf("actor classification = %q", classification)
	}

	rows[0].TetraMetadata = &tetraMetadata{BackendPath: "fallback", BackendBlockers: []string{"unsupported_call_abi"}}
	classification, reason := classifyCategory("parallel map/reduce", rows, 0.20)
	if classification != "blocked by actor/runtime limitation" || !containsAll(reason, "bounded", "unsupported_call_abi") {
		t.Fatalf("parallel classification/reason = %q/%q, want actor limitation with backend ABI evidence", classification, reason)
	}
}

func TestClassifyCategoryActorPingPongUnblocksWithActorRuntimeReadiness(t *testing.T) {
	rows := []benchmarkRow{
		{
			Language:           "tetra",
			Status:             "measured",
			MedianRuntimeMS:    10,
			RawOutputArtifacts: []string{"artifacts/raw/actor_ping_pong_tetra.run.stdout.txt"},
			TetraMetadata: &tetraMetadata{
				BackendPath:     "register",
				BoundsLeft:      0,
				HeapAllocations: 0,
				MemoryEvidence: &memoryEvidence{
					DomainBytesEvidence: memoryMetric{
						EvidenceClass:  "allocation_report_estimate",
						Method:         "actor_domain_accounting_v1",
						SourceArtifact: "artifacts/bin/actor_ping_pong_tetra.alloc.json",
					},
					DomainBytes: []memoryDomainByte{
						{
							DomainID:       "actor:pong",
							Kind:           "actor",
							CurrentBytes:   128,
							EvidenceClass:  "allocation_report_estimate",
							Method:         "actor_domain_accounting_v1",
							SourceArtifact: "artifacts/bin/actor_ping_pong_tetra.alloc.json",
						},
					},
				},
			},
		},
		{Language: "c", Status: "measured", MedianRuntimeMS: 11},
		{Language: "cpp", Status: "measured", MedianRuntimeMS: 12},
		{Language: "rust", Status: "measured", MedianRuntimeMS: 13},
	}
	classification, reason := classifyCategory("actor ping-pong", rows, 0.20)
	if classification != "comparable" || !containsAll(reason, "within", "fastest local competitor") {
		t.Fatalf("actor readiness classification/reason = %q/%q, want comparable local classification", classification, reason)
	}
}

func TestClassifyCategoryActorPingPongRequiresActorDomainEvidence(t *testing.T) {
	rows := []benchmarkRow{
		{
			Language:           "tetra",
			Status:             "measured",
			MedianRuntimeMS:    10,
			RawOutputArtifacts: []string{"artifacts/raw/actor_ping_pong_tetra.run.stdout.txt"},
			TetraMetadata: &tetraMetadata{
				BackendPath:     "register",
				BoundsLeft:      0,
				HeapAllocations: 0,
			},
		},
		{Language: "c", Status: "measured", MedianRuntimeMS: 11},
		{Language: "cpp", Status: "measured", MedianRuntimeMS: 12},
		{Language: "rust", Status: "measured", MedianRuntimeMS: 13},
	}
	classification, reason := classifyCategory("actor ping-pong", rows, 0.20)
	if classification != "blocked by actor/runtime limitation" || !containsAll(reason, "Actor-domain memory evidence", "missing or unsupported") {
		t.Fatalf("actor missing-domain classification/reason = %q/%q, want actor-domain evidence block", classification, reason)
	}
}

func TestClassifySpecialMetricCategoriesUseMetricSpecificEvidence(t *testing.T) {
	rows := []benchmarkRow{
		{Language: "tetra", Status: "measured", MedianRuntimeMS: 100, CompileTimeMS: 10, BinarySizeBytes: 10, TetraMetadata: &tetraMetadata{BackendPath: "register"}},
		{Language: "c", Status: "measured", MedianRuntimeMS: 1, CompileTimeMS: 100, BinarySizeBytes: 100},
		{Language: "cpp", Status: "measured", MedianRuntimeMS: 1, CompileTimeMS: 100, BinarySizeBytes: 100},
		{Language: "rust", Status: "measured", MedianRuntimeMS: 1, CompileTimeMS: 100, BinarySizeBytes: 100},
	}
	classification, reason := classifyCategory("binary size", rows, 0.20)
	if classification != "comparable" || !containsAll(reason, "binary_size_bytes", "10") {
		t.Fatalf("binary size classification/reason = %q/%q", classification, reason)
	}
	classification, reason = classifyCategory("compile time", rows, 0.20)
	if classification != "faster than C/C++/Rust locally" || !containsAll(reason, "compile_time_ms", "10") {
		t.Fatalf("compile time classification/reason = %q/%q", classification, reason)
	}
}

func TestCollectTetraMetadataAttachesMemoryEvidenceFromAllocationReport(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "allocation_tetra")
	mustWrite(t, binary+".proof.json", `{"kind":"proof"}`)
	mustWrite(t, binary+".bounds.json", `{"totals":{"left":0}}`)
	mustWrite(t, binary+".backend.json", `{"summary":{"register_path":1,"stack_fallback":0}}`)
	mustWrite(t, binary+".perf.json", `{"benchmarks":[]}`)
	mustWrite(t, binary+".alloc.json", `{
  "summary": {
    "bytes_requested": 128,
    "bytes_reserved": 256,
    "bytes_committed": 256,
    "bytes_released": 256,
    "domains": [
      {
        "domain_id": "domain:process",
        "kind": "process",
        "requested_bytes": 128,
        "reserved_bytes": 256,
        "committed_bytes": 256,
        "released_bytes": 256,
        "bytes_copied": 64
      }
    ]
  }
}`)
	heapArtifact := filepath.Join(dir, "allocation_tetra.heap.json")
	rssArtifact := filepath.Join(dir, "allocation_tetra.rss.json")
	metadata := collectTetraMetadata("allocation_tetra", binary, filepath.Join(dir, "optimizer.json"), &runtimeHeapEvidence{
		SourceArtifact: heapArtifact,
		Sample: heaptelemetry.Sample{
			Schema:              heaptelemetry.Schema,
			Target:              heaptelemetry.TargetLinuxX64,
			Method:              heaptelemetry.MethodLinuxX64HeapTelemetryV1,
			Program:             "allocation_tetra",
			HeapCurrentBytes:    32,
			HeapPeakBytes:       64,
			HeapTotalAllocBytes: 96,
			HeapAllocationCount: 3,
		},
	}, &runtimeRSSEvidence{
		SourceArtifact: rssArtifact,
		Sample: rsstelemetry.Sample{
			Schema:          rsstelemetry.Schema,
			Method:          rsstelemetry.MethodLinuxProcfsWait4RSSSamplerV1,
			Program:         "allocation_tetra",
			TargetOS:        rsstelemetry.TargetOSLinux,
			TargetArch:      "amd64",
			ExitStatus:      0,
			SampleCount:     2,
			RSSCurrentBytes: 4096,
			RSSPeakBytes:    8192,
			RSSPeakSource:   rsstelemetry.PeakSourceWait4RusageMaxRSS,
			RUMaxRSSRaw:     8,
			RUMaxRSSUnit:    rsstelemetry.UnitKilobytes,
		},
	})
	if metadata.MemoryEvidence == nil {
		t.Fatalf("memory evidence missing")
	}
	if metric := metadata.MemoryEvidence.HeapAllocBytes; metric.EvidenceClass != "runtime_measured" || metric.Method != heaptelemetry.MethodLinuxX64HeapTelemetryV1 || metric.PeakBytes != 64 || metric.TotalAllocBytes != 96 || metric.AllocationCount != 3 || metric.SourceArtifact != heapArtifact {
		t.Fatalf("heap_alloc_bytes evidence = %#v, want runtime sidecar metric", metric)
	}
	if metadata.MemoryEvidence.BytesRequested.Bytes != 128 || metadata.MemoryEvidence.BytesRequested.EvidenceClass != "allocation_report_estimate" {
		t.Fatalf("bytes_requested evidence = %#v", metadata.MemoryEvidence.BytesRequested)
	}
	if metadata.MemoryEvidence.BytesReserved.Bytes != 256 {
		t.Fatalf("bytes_reserved = %d, want 256", metadata.MemoryEvidence.BytesReserved.Bytes)
	}
	if metadata.MemoryEvidence.BytesCommitted.Bytes != 256 || metadata.MemoryEvidence.BytesCommitted.EvidenceClass != "allocation_report_estimate" {
		t.Fatalf("bytes_committed evidence = %#v, want allocation report estimate 256", metadata.MemoryEvidence.BytesCommitted)
	}
	if metadata.MemoryEvidence.BytesReleased.Bytes != 256 || metadata.MemoryEvidence.BytesReleased.EvidenceClass != "allocation_report_estimate" {
		t.Fatalf("bytes_released evidence = %#v, want allocation report estimate 256", metadata.MemoryEvidence.BytesReleased)
	}
	if metadata.MemoryEvidence.BytesCopied.Bytes != 64 {
		t.Fatalf("bytes_copied = %d, want 64", metadata.MemoryEvidence.BytesCopied.Bytes)
	}
	if metric := metadata.MemoryEvidence.RSSCurrent; metric.EvidenceClass != "runtime_measured" || metric.Method != rsstelemetry.MethodLinuxProcfsStatusVmRSSV1 || metric.CurrentBytes != 4096 || metric.SourceArtifact != rssArtifact {
		t.Fatalf("rss_current evidence = %#v, want runtime sidecar metric", metric)
	}
	if metric := metadata.MemoryEvidence.RSSPeak; metric.EvidenceClass != "runtime_measured" || metric.Method != rsstelemetry.MethodLinuxWait4RusageMaxRSSV1 || metric.PeakBytes != 8192 || metric.SourceArtifact != rssArtifact {
		t.Fatalf("rss_peak evidence = %#v, want runtime sidecar metric", metric)
	}
	if len(metadata.MemoryEvidence.DomainBytes) != 1 || metadata.MemoryEvidence.DomainBytes[0].DomainID != "domain:process" {
		t.Fatalf("domain bytes = %#v, want process domain", metadata.MemoryEvidence.DomainBytes)
	}
}

func TestCollectTetraMetadataAttachesHeapReasonCodes(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "allocation_tetra")
	mustWrite(t, binary+".proof.json", `{"kind":"proof"}`)
	mustWrite(t, binary+".bounds.json", `{"totals":{"left":0}}`)
	mustWrite(t, binary+".backend.json", `{"summary":{"register_path":1,"stack_fallback":0}}`)
	mustWrite(t, binary+".perf.json", `{"benchmarks":[]}`)
	mustWrite(t, binary+".alloc.json", `{
  "totals": {"heap": 1},
  "summary": {
    "heap_reason_codes": {
      "heap.required_unknown_call": 1,
      "heap.required_escape_return": 2
    },
    "bytes_requested": 128,
    "bytes_reserved": 128,
    "domains": []
  }
}`)

	metadata := collectTetraMetadata("allocation_tetra", binary, filepath.Join(dir, "optimizer.json"), nil, nil)
	assertStringSlice(t, "heap_reason_codes", metadata.HeapReasonCodes, []string{"heap.required_escape_return", "heap.required_unknown_call"})
}

func TestCollectTetraMetadataAttachesBackendBlockerCategories(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "parallel_map_reduce_tetra")
	mustWrite(t, binary+".proof.json", `{"kind":"proof"}`)
	mustWrite(t, binary+".bounds.json", `{"totals":{"left":0}}`)
	mustWrite(t, binary+".alloc.json", `{"totals":{"heap":0},"summary":{"domains":[]}}`)
	mustWrite(t, binary+".perf.json", `{"benchmarks":[]}`)
	mustWrite(t, binary+".backend.json", `{
  "summary": {
    "register_path": 3,
    "stack_fallback": 1,
    "categories": {
      "register_path": 3,
      "unsupported_call_abi": 1
    }
  }
}`)
	metadata := collectTetraMetadata("parallel_map_reduce_tetra", binary, filepath.Join(dir, "optimizer.json"), nil, nil)
	if metadata.BackendPath != "fallback" {
		t.Fatalf("backend path = %q, want fallback", metadata.BackendPath)
	}
	if len(metadata.BackendBlockers) != 1 || metadata.BackendBlockers[0] != "unsupported_call_abi" {
		t.Fatalf("backend blockers = %#v, want unsupported_call_abi", metadata.BackendBlockers)
	}
}

func TestCollectTetraMetadataAttachesRuntimeFeatureEvidence(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "actor_task_tetra")
	mustWrite(t, binary+".proof.json", `{"kind":"proof"}`)
	mustWrite(t, binary+".bounds.json", `{"totals":{"left":0}}`)
	mustWrite(t, binary+".alloc.json", `{"totals":{"heap":0},"summary":{"domains":[]}}`)
	mustWrite(t, binary+".perf.json", `{"benchmarks":[]}`)
	mustWrite(t, binary+".backend.json", `{
  "summary": {
    "register_path": 1,
    "stack_fallback": 0,
    "runtime_features_required": ["task_runtime", "actor_runtime"],
    "runtime_features_linked": ["task_runtime", "actor_runtime"],
    "runtime_features_initialized": ["task_runtime", "actor_runtime"],
    "runtime_lazy_init_blockers": ["unknown_runtime_call:__tetra_future_runtime_probe"],
    "runtime_feature_evidence_class": "lowered_ir_static_plan",
    "runtime_feature_evidence_method": "backend_report_lowered_ir_scan_v1",
    "runtime_object_plan": {
      "evidence_class": "native_runtime_object_plan",
      "evidence_method": "native_link_runtime_object_plan_v1",
      "runtime_used": true,
      "runtime_object_linked": true,
      "runtime_object_initialized": true,
      "runtime_object_features_required": ["task_runtime", "actor_runtime"],
      "runtime_object_features_linked": ["task_runtime", "actor_runtime"],
      "runtime_object_features_initialized": ["task_runtime", "actor_runtime"],
      "runtime_object_lazy_init_blockers": []
    }
  }
}`)

	metadata := collectTetraMetadata("actor_task_tetra", binary, filepath.Join(dir, "optimizer.json"), nil, nil)
	assertStringSlice(t, "runtime_features_required", metadata.RuntimeFeaturesRequired, []string{"actor_runtime", "task_runtime"})
	assertStringSlice(t, "runtime_features_linked", metadata.RuntimeFeaturesLinked, []string{"actor_runtime", "task_runtime"})
	assertStringSlice(t, "runtime_features_initialized", metadata.RuntimeFeaturesInitialized, []string{"actor_runtime", "task_runtime"})
	assertStringSlice(t, "runtime_lazy_init_blockers", metadata.RuntimeLazyInitBlockers, []string{"unknown_runtime_call:__tetra_future_runtime_probe"})
	if metadata.RuntimeFeatureEvidence.EvidenceClass != "lowered_ir_static_plan" ||
		metadata.RuntimeFeatureEvidence.Method != "backend_report_lowered_ir_scan_v1" ||
		metadata.RuntimeFeatureEvidence.SourceArtifact != binary+".backend.json" {
		t.Fatalf("runtime feature evidence = %#v, want backend report lowered IR evidence", metadata.RuntimeFeatureEvidence)
	}
	if metadata.RuntimeObjectPlan.EvidenceClass != "native_runtime_object_plan" ||
		metadata.RuntimeObjectPlan.EvidenceMethod != "native_link_runtime_object_plan_v1" ||
		!metadata.RuntimeObjectPlan.RuntimeUsed ||
		!metadata.RuntimeObjectPlan.RuntimeObjectLinked ||
		!metadata.RuntimeObjectPlan.RuntimeObjectInitialized {
		t.Fatalf("runtime object plan = %#v, want linked native runtime object plan", metadata.RuntimeObjectPlan)
	}
	assertStringSlice(t, "runtime_object_features_required", metadata.RuntimeObjectPlan.RuntimeObjectFeaturesRequired, []string{"actor_runtime", "task_runtime"})
}

func TestRunCommandWithRSSCollectsLiveSampleAndPeak(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux procfs/wait4 RSS sampler")
	}
	stdout, stderr, exitCode, elapsed, sample, err := runCommandWithRSS(
		2*time.Second,
		[]string{"/bin/sh", "-c", "printf rss-ok; sleep 0.05"},
		os.Environ(),
		"rss_probe_tetra",
		500*time.Microsecond,
	)
	if err != nil {
		t.Fatalf("runCommandWithRSS: %v stderr=%s", err, string(stderr))
	}
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0 stderr=%s", exitCode, string(stderr))
	}
	if strings.TrimSpace(string(stdout)) != "rss-ok" {
		t.Fatalf("stdout = %q, want rss-ok", string(stdout))
	}
	if elapsed <= 0 {
		t.Fatalf("elapsed = %v, want positive", elapsed)
	}
	if sample.Program != "rss_probe_tetra" || sample.TargetOS != rsstelemetry.TargetOSLinux {
		t.Fatalf("sample identity = %+v", sample)
	}
	if sample.SampleCount == 0 || sample.RSSCurrentBytes == 0 {
		t.Fatalf("sample current RSS = %+v, want live RSS sample", sample)
	}
	if sample.RSSPeakBytes == 0 || sample.RSSPeakBytes < sample.RSSCurrentBytes {
		t.Fatalf("sample peak RSS = %+v, want peak >= current", sample)
	}
	if err := rsstelemetry.Validate(sample); err != nil {
		t.Fatalf("RSS sample did not validate: %v\n%+v", err, sample)
	}
}

func TestMissingTetraMetadataAttachesBlockedMemoryEvidence(t *testing.T) {
	metadata := missingTetraMetadata("missing-binary", "optimizer.json")
	if metadata.RuntimeFeatureEvidence.EvidenceClass != "blocked" || metadata.RuntimeFeatureEvidence.BlockedReason == "" {
		t.Fatalf("runtime feature evidence = %#v, want blocked reason", metadata.RuntimeFeatureEvidence)
	}
	if metadata.RuntimeObjectPlan.EvidenceClass != "blocked" || metadata.RuntimeObjectPlan.BlockedReason == "" {
		t.Fatalf("runtime object plan = %#v, want blocked reason", metadata.RuntimeObjectPlan)
	}
	assertStringSlice(t, "runtime_features_required", metadata.RuntimeFeaturesRequired, []string{})
	assertStringSlice(t, "runtime_features_linked", metadata.RuntimeFeaturesLinked, []string{})
	assertStringSlice(t, "runtime_features_initialized", metadata.RuntimeFeaturesInitialized, []string{})
	assertStringSlice(t, "runtime_lazy_init_blockers", metadata.RuntimeLazyInitBlockers, []string{})
	if metadata.MemoryEvidence == nil {
		t.Fatalf("memory evidence missing")
	}
	for name, metric := range map[string]memoryMetric{
		"heap_alloc_bytes": metadata.MemoryEvidence.HeapAllocBytes,
		"bytes_requested":  metadata.MemoryEvidence.BytesRequested,
		"bytes_reserved":   metadata.MemoryEvidence.BytesReserved,
		"bytes_committed":  metadata.MemoryEvidence.BytesCommitted,
		"bytes_released":   metadata.MemoryEvidence.BytesReleased,
		"bytes_copied":     metadata.MemoryEvidence.BytesCopied,
		"rss_current":      metadata.MemoryEvidence.RSSCurrent,
		"rss_peak":         metadata.MemoryEvidence.RSSPeak,
	} {
		if metric.EvidenceClass != "blocked" || metric.BlockedReason == "" {
			t.Fatalf("%s metric = %#v, want blocked reason", name, metric)
		}
	}
	if len(metadata.MemoryEvidence.DomainBytes) != 0 || metadata.MemoryEvidence.DomainBytesEvidence.EvidenceClass != "blocked" {
		t.Fatalf("domain evidence = %#v/%#v, want blocked empty domains", metadata.MemoryEvidence.DomainBytes, metadata.MemoryEvidence.DomainBytesEvidence)
	}
}

func containsSequence(items []string, want ...string) bool {
	if len(want) == 0 || len(want) > len(items) {
		return false
	}
	for i := 0; i <= len(items)-len(want); i++ {
		ok := true
		for j := range want {
			if items[i+j] != want[j] {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func assertStringSlice(t *testing.T, label string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", label, got, want)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func containsAll(text string, wants ...string) bool {
	for _, want := range wants {
		if !containsSubstring(text, want) {
			return false
		}
	}
	return true
}

func containsSubstring(text string, want string) bool {
	for i := 0; i+len(want) <= len(text); i++ {
		if text[i:i+len(want)] == want {
			return true
		}
	}
	return false
}

func mustWrite(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
