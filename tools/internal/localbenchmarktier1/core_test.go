package localbenchmarktier1

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"tetra_language/tools/internal/heaptelemetry"
	"tetra_language/tools/internal/rsstelemetry"
	"time"
)

func TestBuildCommandEnablesTetraRuntimeHeapTelemetry(t *testing.T) {
	spec := benchmarkSpec{
		Language:      "tetra",
		SourceRelPath: "src/main.tetra",
		BinaryRelPath: "bin/main",
	}
	cmd := buildCommand(spec, "tetra", "artifacts/heap-telemetry/main/runtime")
	if !containsSequence(
		cmd,
		"--emit-runtime-heap-telemetry",
		"--runtime-heap-telemetry-dir",
		"artifacts/heap-telemetry/main/runtime",
	) {
		t.Fatalf("tetra build command = %#v, want runtime heap telemetry flags", cmd)
	}
}

func TestBuildCommandUsesBuiltinRuntimeForActorPingPongTetraTelemetry(t *testing.T) {
	spec := benchmarkSpec{
		Category:      "actor ping-pong",
		Language:      "tetra",
		SourceRelPath: "src/actor_ping_pong.tetra",
		BinaryRelPath: "bin/actor_ping_pong_tetra",
	}
	cmd := buildCommand(spec, "tetra", "artifacts/heap-telemetry/actor_ping_pong_tetra/runtime")
	if !containsSequence(cmd, "--runtime", "builtin") {
		t.Fatalf("actor ping-pong tetra build command = %#v, want builtin runtime", cmd)
	}
	if !containsSequence(
		cmd,
		"--emit-runtime-heap-telemetry",
		"--runtime-heap-telemetry-dir",
		"artifacts/heap-telemetry/actor_ping_pong_tetra/runtime",
	) {
		t.Fatalf(
			"actor ping-pong tetra build command = %#v, want runtime heap telemetry flags",
			cmd,
		)
	}
}

func TestBuildCommandKeepsNonActorTetraRuntimeDefault(t *testing.T) {
	spec := benchmarkSpec{
		Category:      "integer loops",
		Language:      "tetra",
		SourceRelPath: "src/integer_loops.tetra",
		BinaryRelPath: "bin/integer_loops_tetra",
	}
	cmd := buildCommand(spec, "tetra", "artifacts/heap-telemetry/integer_loops_tetra/runtime")
	if containsSequence(cmd, "--runtime", "builtin") {
		t.Fatalf("non-actor tetra build command = %#v, did not expect builtin runtime", cmd)
	}
	if !containsSequence(
		cmd,
		"--emit-runtime-heap-telemetry",
		"--runtime-heap-telemetry-dir",
		"artifacts/heap-telemetry/integer_loops_tetra/runtime",
	) {
		t.Fatalf("non-actor tetra build command = %#v, want runtime heap telemetry flags", cmd)
	}
}

func TestWriteAuditKeepsLinesWithinLimit(t *testing.T) {
	dir := t.TempDir()
	auditPath := filepath.Join(dir, "audit.md")
	outDir := filepath.Join(
		"reports",
		"benchmark-vnext-memory-baseline",
		"tier1-after-actor-transfer-scalar-send-sidecar-coordinator",
	)
	report := tier1Report{
		Results: []categoryResult{
			{
				Category:       "integer loops",
				Classification: "faster than C/C++/Rust locally",
				ClassificationReason: ("Tetra median 0.811 ms is more than 20% below " +
					"the fastest local competitor median 1.040 ms."),
			},
			{
				Category:       "actor ping-pong",
				Classification: "blocked by actor/runtime limitation",
				ClassificationReason: ("Current local actor/task runtime evidence is bounded " +
					"and not a production parallel benchmark claim. Backend path is " +
					"fallback, not register. Perf blockers: actor_copy.borrowed_data_boundary."),
			},
		},
	}

	if err := writeAudit(auditPath, report, outDir); err != nil {
		t.Fatalf("writeAudit: %v", err)
	}
	raw, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	for i, line := range strings.Split(string(raw), "\n") {
		if len(line) > 100 {
			t.Fatalf("audit line %d has length %d: %q", i+1, len(line), line)
		}
	}
	if !strings.Contains(string(raw), "go run ./tools/cmd/validate-local-benchmark-tier1 \\") {
		t.Fatalf("audit missing wrapped validator command:\n%s", raw)
	}
}

func TestExecuteActorPingPongTetraCollectsBuiltinRuntimeActorDomainBytes(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	outDir := filepath.Join(t.TempDir(), "tier1")
	tetraTool := filepath.Join(outDir, "artifacts", "bin", "tetra")
	if err := os.MkdirAll(filepath.Dir(tetraTool), 0o755); err != nil {
		t.Fatalf("create tetra tool dir: %v", err)
	}
	env := commandEnv(root)
	buildStdout := filepath.Join(outDir, "artifacts", "raw", "tetra_cli_build.stdout.txt")
	buildStderr := filepath.Join(outDir, "artifacts", "raw", "tetra_cli_build.stderr.txt")
	if _, _, err := runCaptured(
		60*time.Second,
		[]string{"go", "build", "-C", root, "-o", tetraTool, "./cli/cmd/tetra"},
		env,
		buildStdout,
		buildStderr,
	); err != nil {
		t.Fatalf(
			"build local tetra CLI: %v\nstdout:\n%s\nstderr:\n%s",
			err,
			readText(buildStdout),
			readText(buildStderr),
		)
	}

	optimizerArtifact, err := writeOptimizerArtifact(outDir)
	if err != nil {
		t.Fatalf("write optimizer artifact: %v", err)
	}
	var spec benchmarkSpec
	for _, candidate := range buildBenchmarkSpecs(outDir) {
		if candidate.Category == "actor ping-pong" && candidate.Language == "tetra" {
			spec = candidate
			break
		}
	}
	if spec.Name == "" {
		t.Fatalf("actor ping-pong tetra spec missing")
	}

	row := executeSpec(
		spec,
		options{OutDir: outDir, Iterations: 1, Timeout: 20 * time.Second},
		env,
		map[string]string{"tetra": "test"},
		tetraTool,
		optimizerArtifact,
	)
	if row.Status != "measured" {
		t.Fatalf(
			"actor ping-pong row status = %q error=%q\nbuild stderr:\n%s\nrun stderr:\n%s",
			row.Status,
			row.Error,
			readText(filepath.Join(outDir, "artifacts", "raw", spec.Name+".build.stderr.txt")),
			readText(filepath.Join(outDir, "artifacts", "raw", spec.Name+".run.stderr.txt")),
		)
	}
	if !containsSequence(row.BuildCommand, "--runtime", "builtin") {
		t.Fatalf("actor ping-pong build command = %#v, want builtin runtime", row.BuildCommand)
	}
	if !containsSequence(
		row.BuildCommand,
		"--emit-runtime-heap-telemetry",
		"--runtime-heap-telemetry-dir",
		filepath.Join(outDir, "artifacts", "heap-telemetry", spec.Name, "runtime"),
	) {
		t.Fatalf(
			"actor ping-pong build command = %#v, want runtime heap telemetry flags",
			row.BuildCommand,
		)
	}
	if row.TetraMetadata == nil || row.TetraMetadata.MemoryEvidence == nil {
		t.Fatalf("actor ping-pong metadata missing: %#v", row.TetraMetadata)
	}
	memory := row.TetraMetadata.MemoryEvidence
	if got := memory.DomainBytesEvidence; got.EvidenceClass != "runtime_measured" ||
		got.Method != heaptelemetry.MethodLinuxX64HeapTelemetryV1 ||
		got.SourceArtifact == "" {
		t.Fatalf("domain_bytes_evidence = %#v, want runtime heap sidecar evidence", got)
	}
	if got := memory.BytesCopied; got.EvidenceClass != "runtime_measured" ||
		got.Method != heaptelemetry.MethodLinuxX64HeapTelemetryV1 ||
		got.Bytes == 0 ||
		got.SourceArtifact != memory.DomainBytesEvidence.SourceArtifact {
		t.Fatalf("bytes_copied = %#v, want runtime aggregate from same heap sidecar", got)
	}
	domain, ok := firstActorDomain(memory.DomainBytes)
	if !ok {
		t.Fatalf("domain bytes = %#v, want at least one actor domain", memory.DomainBytes)
	}
	if !strings.HasPrefix(domain.DomainID, "domain:actor:") || domain.Kind != "actor" ||
		domain.PeakBytes == 0 ||
		domain.BytesCopied == 0 {
		t.Fatalf(
			"actor domain = %#v, want domain:actor:* with non-zero peak_bytes and bytes_copied",
			domain,
		)
	}
	sample, err := heaptelemetry.ReadFile(memory.DomainBytesEvidence.SourceArtifact, outDir)
	if err != nil {
		t.Fatalf("read heap sidecar %s: %v", memory.DomainBytesEvidence.SourceArtifact, err)
	}
	var sidecarActor bool
	for _, sidecarDomain := range sample.DomainBytes {
		if strings.HasPrefix(sidecarDomain.DomainID, "domain:actor:") &&
			sidecarDomain.Kind == "actor" &&
			sidecarDomain.PeakBytes > 0 &&
			sidecarDomain.BytesCopied > 0 {
			sidecarActor = true
			break
		}
	}
	if !sidecarActor {
		t.Fatalf("heap sidecar domains = %#v, want runtime actor domain bytes", sample.DomainBytes)
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
	mustWrite(
		t,
		binary+".backend.json",
		`{"summary":{"register_path":0,"stack_fallback":1,"categories":{"unsupported_control_flow":1}}}`,
	)
	mustWrite(t, binary+".perf.json", `{
  "benchmarks": [{
    "benchmark": "hash_table_tetra",
    "reason_codes": ["allocation.local_call_heap_fallback", "inline.code_size_budget"]
  }]
}`)

	metadata := collectTetraMetadata(
		"hash_table_tetra",
		binary,
		filepath.Join(dir, "optimizer.json"),
		nil,
		nil,
	)
	if containsString(metadata.PerfBlockers, "allocation.local_call_heap_fallback") {
		t.Fatalf("resolved heap fallback blocker was retained: %#v", metadata.PerfBlockers)
	}
	if !containsString(metadata.PerfBlockers, "inline.code_size_budget") {
		t.Fatalf("non-allocation perf blocker was filtered: %#v", metadata.PerfBlockers)
	}
}

func TestCollectTetraMetadataFiltersEvidenceResolvedPerfBlockers(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "json_parse_stringify_tetra")
	mustWrite(t, binary+".alloc.json", `{
  "totals": {"heap": 0},
  "summary": {
    "heap_reason_codes": {},
    "bytes_requested": 2048,
    "bytes_reserved": 2048,
    "domains": []
  }
}`)
	mustWrite(t, binary+".bounds.json", `{"totals":{"left":0}}`)
	mustWrite(t, binary+".backend.json", `{"summary":{"register_path":1,"stack_fallback":0}}`)
	mustWrite(t, binary+".perf.json", `{
  "benchmarks": [{
    "benchmark": "json_parse_stringify_tetra",
    "reason_codes": [
      "allocation.local_call_heap_fallback",
      "allocation.unknown_call",
      "allocation.return_escape",
      "bounds.missing_dominance",
      "inline.code_size_budget",
      "vector.no_noalias_proof"
    ]
  }]
}`)

	metadata := collectTetraMetadata(
		"json_parse_stringify_tetra",
		binary,
		filepath.Join(dir, "optimizer.json"),
		nil,
		nil,
	)
	assertStringSlice(
		t,
		"perf_blockers",
		metadata.PerfBlockers,
		[]string{"inline.code_size_budget", "vector.no_noalias_proof"},
	)
}

func TestCollectTetraMetadataPreservesEvidenceUnresolvedPerfBlockers(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "allocation_tetra")
	mustWrite(t, binary+".alloc.json", `{
  "totals": {"heap": 2},
  "summary": {
    "heap_reason_codes": {
      "heap.required_unknown_call": 1,
      "heap.required_escape_return": 1
    },
    "bytes_requested": 2048,
    "bytes_reserved": 2048,
    "domains": []
  }
}`)
	mustWrite(t, binary+".bounds.json", `{"totals":{"left":3}}`)
	mustWrite(t, binary+".backend.json", `{"summary":{"register_path":1,"stack_fallback":0}}`)
	mustWrite(t, binary+".perf.json", `{
  "benchmarks": [{
    "benchmark": "allocation_tetra",
    "reason_codes": [
      "allocation.unknown_call",
      "allocation.return_escape",
      "bounds.missing_dominance"
    ]
  }]
}`)

	metadata := collectTetraMetadata(
		"allocation_tetra",
		binary,
		filepath.Join(dir, "optimizer.json"),
		nil,
		nil,
	)
	assertStringSlice(t, "perf_blockers", metadata.PerfBlockers, []string{
		"allocation.unknown_call",
		"allocation.return_escape",
		"bounds.missing_dominance",
	})
}

func TestClassifyCategoryPrefersEvidenceBlockers(t *testing.T) {
	tetra := benchmarkRow{
		Language:        "tetra",
		Status:          "measured",
		MedianRuntimeMS: 1,
		TetraMetadata:   &tetraMetadata{BackendPath: "fallback"},
	}
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

	rows[0].TetraMetadata = &tetraMetadata{
		BackendPath:     "fallback",
		BackendBlockers: []string{"unsupported_call_abi"},
	}
	classification, reason := classifyCategory("parallel map/reduce", rows, 0.20)
	if classification != "blocked by actor/runtime limitation" ||
		!containsAll(reason, "bounded", "unsupported_call_abi") {
		t.Fatalf(
			"parallel classification/reason = %q/%q, want actor limitation with backend ABI evidence",
			classification,
			reason,
		)
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
						EvidenceClass:  "runtime_measured",
						Method:         heaptelemetry.MethodLinuxX64HeapTelemetryV1,
						SourceArtifact: "artifacts/heap-telemetry/actor_ping_pong_tetra/iteration-01.heap.json",
					},
					DomainBytes: []memoryDomainByte{
						{
							DomainID:       "actor:pong",
							Kind:           "actor",
							CurrentBytes:   128,
							EvidenceClass:  "runtime_measured",
							Method:         heaptelemetry.MethodLinuxX64HeapTelemetryV1,
							SourceArtifact: "artifacts/heap-telemetry/actor_ping_pong_tetra/iteration-01.heap.json",
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
	if classification != "comparable" ||
		!containsAll(reason, "within", "fastest local competitor") {
		t.Fatalf(
			"actor readiness classification/reason = %q/%q, want comparable local classification",
			classification,
			reason,
		)
	}
}

func TestClassifyCategoryActorPingPongRejectsAllocationEstimateActorDomainEvidence(t *testing.T) {
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
						Method:         "allocation_report_summary",
						SourceArtifact: "artifacts/bin/actor_ping_pong_tetra.alloc.json",
					},
					DomainBytes: []memoryDomainByte{
						{
							DomainID:       "actor:pong",
							Kind:           "actor",
							CurrentBytes:   128,
							EvidenceClass:  "allocation_report_estimate",
							Method:         "allocation_report_summary",
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
	if classification != "blocked by actor/runtime limitation" ||
		!containsAll(reason, "Actor-domain memory evidence", "missing or unsupported") {
		t.Fatalf(
			"actor estimate classification/reason = %q/%q, want actor-domain evidence block",
			classification,
			reason,
		)
	}
}

func TestClassifyCategoryActorPingPongRejectsUnsupportedActorDomainEvidence(t *testing.T) {
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
					DomainBytesEvidence: unsupportedMemoryMetric(
						"not_collected",
						"actor domain bytes were not emitted",
					),
					DomainBytes: []memoryDomainByte{
						{
							DomainID:      "actor:pong",
							Kind:          "actor",
							CurrentBytes:  128,
							EvidenceClass: "unsupported",
							Method:        "not_collected",
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
	if classification != "blocked by actor/runtime limitation" ||
		!containsAll(reason, "Actor-domain memory evidence", "missing or unsupported") {
		t.Fatalf(
			"actor unsupported classification/reason = %q/%q, want actor-domain evidence block",
			classification,
			reason,
		)
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
	if classification != "blocked by actor/runtime limitation" ||
		!containsAll(reason, "Actor-domain memory evidence", "missing or unsupported") {
		t.Fatalf(
			"actor missing-domain classification/reason = %q/%q, want actor-domain evidence block",
			classification,
			reason,
		)
	}
}

func TestClassifySpecialMetricCategoriesUseMetricSpecificEvidence(t *testing.T) {
	rows := []benchmarkRow{
		{
			Language:        "tetra",
			Status:          "measured",
			MedianRuntimeMS: 100,
			CompileTimeMS:   10,
			BinarySizeBytes: 10,
			TetraMetadata:   &tetraMetadata{BackendPath: "register"},
		},
		{
			Language:        "c",
			Status:          "measured",
			MedianRuntimeMS: 1,
			CompileTimeMS:   100,
			BinarySizeBytes: 100,
		},
		{
			Language:        "cpp",
			Status:          "measured",
			MedianRuntimeMS: 1,
			CompileTimeMS:   100,
			BinarySizeBytes: 100,
		},
		{
			Language:        "rust",
			Status:          "measured",
			MedianRuntimeMS: 1,
			CompileTimeMS:   100,
			BinarySizeBytes: 100,
		},
	}
	classification, reason := classifyCategory("binary size", rows, 0.20)
	if classification != "comparable" || !containsAll(reason, "binary_size_bytes", "10") {
		t.Fatalf("binary size classification/reason = %q/%q", classification, reason)
	}
	classification, reason = classifyCategory("compile time", rows, 0.20)
	if classification != "faster than C/C++/Rust locally" ||
		!containsAll(reason, "compile_time_ms", "10") {
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
	metadata := collectTetraMetadata(
		"allocation_tetra",
		binary,
		filepath.Join(dir, "optimizer.json"),
		&runtimeHeapEvidence{
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
		},
		&runtimeRSSEvidence{
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
		},
	)
	if metadata.MemoryEvidence == nil {
		t.Fatalf("memory evidence missing")
	}
	if metric := metadata.MemoryEvidence.HeapAllocBytes; metric.EvidenceClass != "runtime_measured" ||
		metric.Method != heaptelemetry.MethodLinuxX64HeapTelemetryV1 ||
		metric.PeakBytes != 64 ||
		metric.TotalAllocBytes != 96 ||
		metric.AllocationCount != 3 ||
		metric.SourceArtifact != heapArtifact {
		t.Fatalf("heap_alloc_bytes evidence = %#v, want runtime sidecar metric", metric)
	}
	if metadata.MemoryEvidence.BytesRequested.Bytes != 128 ||
		metadata.MemoryEvidence.BytesRequested.EvidenceClass != "allocation_report_estimate" {
		t.Fatalf("bytes_requested evidence = %#v", metadata.MemoryEvidence.BytesRequested)
	}
	if metadata.MemoryEvidence.BytesReserved.Bytes != 256 {
		t.Fatalf("bytes_reserved = %d, want 256", metadata.MemoryEvidence.BytesReserved.Bytes)
	}
	if metadata.MemoryEvidence.BytesCommitted.Bytes != 256 ||
		metadata.MemoryEvidence.BytesCommitted.EvidenceClass != "allocation_report_estimate" {
		t.Fatalf(
			"bytes_committed evidence = %#v, want allocation report estimate 256",
			metadata.MemoryEvidence.BytesCommitted,
		)
	}
	if metadata.MemoryEvidence.BytesReleased.Bytes != 256 ||
		metadata.MemoryEvidence.BytesReleased.EvidenceClass != "allocation_report_estimate" {
		t.Fatalf(
			"bytes_released evidence = %#v, want allocation report estimate 256",
			metadata.MemoryEvidence.BytesReleased,
		)
	}
	if metadata.MemoryEvidence.BytesCopied.Bytes != 64 {
		t.Fatalf("bytes_copied = %d, want 64", metadata.MemoryEvidence.BytesCopied.Bytes)
	}
	if metric := metadata.MemoryEvidence.RSSCurrent; metric.EvidenceClass != "runtime_measured" ||
		metric.Method != rsstelemetry.MethodLinuxProcfsStatusVmRSSV1 ||
		metric.CurrentBytes != 4096 ||
		metric.SourceArtifact != rssArtifact {
		t.Fatalf("rss_current evidence = %#v, want runtime sidecar metric", metric)
	}
	if metric := metadata.MemoryEvidence.RSSPeak; metric.EvidenceClass != "runtime_measured" ||
		metric.Method != rsstelemetry.MethodLinuxWait4RusageMaxRSSV1 ||
		metric.PeakBytes != 8192 ||
		metric.SourceArtifact != rssArtifact {
		t.Fatalf("rss_peak evidence = %#v, want runtime sidecar metric", metric)
	}
	if len(metadata.MemoryEvidence.DomainBytes) != 1 ||
		metadata.MemoryEvidence.DomainBytes[0].DomainID != "domain:process" {
		t.Fatalf("domain bytes = %#v, want process domain", metadata.MemoryEvidence.DomainBytes)
	}
}

func TestCollectTetraMetadataPrefersRuntimeDomainBytesOverAllocationReport(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "actor_ping_pong_tetra")
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
	heapArtifact := filepath.Join(dir, "actor_ping_pong_tetra.heap.json")
	metadata := collectTetraMetadata(
		"actor_ping_pong_tetra",
		binary,
		filepath.Join(dir, "optimizer.json"),
		&runtimeHeapEvidence{
			SourceArtifact: heapArtifact,
			Sample: heaptelemetry.Sample{
				Schema:              heaptelemetry.Schema,
				Target:              heaptelemetry.TargetLinuxX64,
				Method:              heaptelemetry.MethodLinuxX64HeapTelemetryV1,
				Program:             "actor_ping_pong_tetra",
				HeapCurrentBytes:    128,
				HeapPeakBytes:       192,
				HeapTotalAllocBytes: 256,
				HeapAllocationCount: 4,
				DomainBytes: []heaptelemetry.DomainBytes{
					{
						DomainID:             "actor:pong",
						Kind:                 "actor",
						RequestedBytes:       32,
						ReservedBytes:        64,
						CommittedBytes:       64,
						CurrentBytes:         128,
						PeakBytes:            192,
						BytesCopied:          7,
						MailboxCurrentBytes:  64,
						MailboxPeakBytes:     96,
						StackLiveBytes:       64,
						StackReservedBytes:   96,
						StackRetainedBytes:   0,
						StackReleasedBytes:   0,
						ByteBudget:           1088,
						OverBudgetCount:      3,
						BackpressureEvents:   4,
						ActorDomainFieldsSet: true,
					},
				},
			},
		},
		nil,
	)
	if metadata.MemoryEvidence == nil {
		t.Fatalf("memory evidence missing")
	}
	if got := metadata.MemoryEvidence.DomainBytesEvidence; got.EvidenceClass != "runtime_measured" ||
		got.Method != heaptelemetry.MethodLinuxX64HeapTelemetryV1 ||
		got.SourceArtifact != heapArtifact {
		t.Fatalf("domain_bytes_evidence = %#v, want runtime heap sidecar evidence", got)
	}
	if got := metadata.MemoryEvidence.BytesCopied; got.Bytes != 7 ||
		got.EvidenceClass != "runtime_measured" ||
		got.Method != heaptelemetry.MethodLinuxX64HeapTelemetryV1 ||
		got.SourceArtifact != heapArtifact {
		t.Fatalf("bytes_copied = %#v, want runtime domain aggregate from sidecar", got)
	}
	if len(metadata.MemoryEvidence.DomainBytes) != 1 {
		t.Fatalf(
			"domain bytes = %#v, want one runtime actor domain",
			metadata.MemoryEvidence.DomainBytes,
		)
	}
	domain := metadata.MemoryEvidence.DomainBytes[0]
	if domain.DomainID != "actor:pong" || domain.Kind != "actor" ||
		domain.RequestedBytes != 32 || domain.ReservedBytes != 64 ||
		domain.CommittedBytes != 64 || domain.CurrentBytes != 128 ||
		domain.PeakBytes != 192 || domain.BytesCopied != 7 ||
		domain.MailboxCurrentBytes != 64 || domain.MailboxPeakBytes != 96 ||
		domain.StackLiveBytes != 64 || domain.StackReservedBytes != 96 ||
		domain.StackRetainedBytes != 0 || domain.StackReleasedBytes != 0 ||
		domain.ByteBudget != 1088 || domain.OverBudgetCount != 3 ||
		domain.BackpressureEvents != 4 ||
		domain.EvidenceClass != "runtime_measured" ||
		domain.Method != heaptelemetry.MethodLinuxX64HeapTelemetryV1 ||
		domain.SourceArtifact != heapArtifact {
		t.Fatalf("domain bytes = %+v, want runtime sidecar actor domain", domain)
	}
}

func TestCollectTetraMetadataActorFieldsOnlyMarshalForRuntimeActorDomains(t *testing.T) {
	evidence := memoryEvidence{
		Schema: schemaBenchmarkMemoryV1,
		DomainBytes: []memoryDomainByte{
			{
				DomainID:      "domain:process",
				Kind:          "process",
				CurrentBytes:  0,
				PeakBytes:     0,
				EvidenceClass: "runtime_measured",
				Method:        heaptelemetry.MethodLinuxX64HeapTelemetryV1,
			},
			{
				DomainID:             "domain:actor:000",
				Kind:                 "actor",
				CurrentBytes:         0,
				PeakBytes:            88,
				BytesCopied:          88,
				MailboxCurrentBytes:  0,
				MailboxPeakBytes:     88,
				StackLiveBytes:       0,
				StackReservedBytes:   65536,
				StackRetainedBytes:   65536,
				StackReleasedBytes:   0,
				ByteBudget:           22528,
				OverBudgetCount:      0,
				BackpressureEvents:   0,
				ActorDomainFieldsSet: true,
				EvidenceClass:        "runtime_measured",
				Method:               heaptelemetry.MethodLinuxX64HeapTelemetryV1,
			},
		},
	}
	raw, err := json.Marshal(evidence)
	if err != nil {
		t.Fatalf("marshal memory evidence: %v", err)
	}
	var decoded struct {
		DomainBytes []map[string]any `json:"domain_bytes"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal memory evidence: %v\n%s", err, raw)
	}
	if len(decoded.DomainBytes) != 2 {
		t.Fatalf("domain_bytes len = %d, want 2: %s", len(decoded.DomainBytes), raw)
	}
	for _, key := range []string{
		"mailbox_current_bytes",
		"mailbox_peak_bytes",
		"stack_live_bytes",
		"stack_reserved_bytes",
		"stack_retained_bytes",
		"stack_released_bytes",
		"byte_budget",
		"over_budget_count",
		"backpressure_events",
	} {
		if _, ok := decoded.DomainBytes[0][key]; ok {
			t.Fatalf("process domain unexpectedly emitted %s in %s", key, raw)
		}
		if _, ok := decoded.DomainBytes[1][key]; !ok {
			t.Fatalf("actor domain missing %s in %s", key, raw)
		}
	}
	if got := decoded.DomainBytes[1]["mailbox_current_bytes"]; got != float64(0) {
		t.Fatalf("actor mailbox_current_bytes = %#v, want encoded zero", got)
	}
	if got := decoded.DomainBytes[1]["stack_live_bytes"]; got != float64(0) {
		t.Fatalf("actor stack_live_bytes = %#v, want encoded zero", got)
	}
	if got := decoded.DomainBytes[1]["stack_reserved_bytes"]; got != float64(65536) {
		t.Fatalf("actor stack_reserved_bytes = %#v, want encoded 65536", got)
	}
	if got := decoded.DomainBytes[1]["over_budget_count"]; got != float64(0) {
		t.Fatalf("actor over_budget_count = %#v, want encoded zero", got)
	}
	if got := decoded.DomainBytes[1]["backpressure_events"]; got != float64(0) {
		t.Fatalf("actor backpressure_events = %#v, want encoded zero", got)
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

	metadata := collectTetraMetadata(
		"allocation_tetra",
		binary,
		filepath.Join(dir, "optimizer.json"),
		nil,
		nil,
	)
	assertStringSlice(
		t,
		"heap_reason_codes",
		metadata.HeapReasonCodes,
		[]string{"heap.required_escape_return", "heap.required_unknown_call"},
	)
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
	metadata := collectTetraMetadata(
		"parallel_map_reduce_tetra",
		binary,
		filepath.Join(dir, "optimizer.json"),
		nil,
		nil,
	)
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

	metadata := collectTetraMetadata(
		"actor_task_tetra",
		binary,
		filepath.Join(dir, "optimizer.json"),
		nil,
		nil,
	)
	assertStringSlice(
		t,
		"runtime_features_required",
		metadata.RuntimeFeaturesRequired,
		[]string{"actor_runtime", "task_runtime"},
	)
	assertStringSlice(
		t,
		"runtime_features_linked",
		metadata.RuntimeFeaturesLinked,
		[]string{"actor_runtime", "task_runtime"},
	)
	assertStringSlice(
		t,
		"runtime_features_initialized",
		metadata.RuntimeFeaturesInitialized,
		[]string{"actor_runtime", "task_runtime"},
	)
	assertStringSlice(
		t,
		"runtime_lazy_init_blockers",
		metadata.RuntimeLazyInitBlockers,
		[]string{"unknown_runtime_call:__tetra_future_runtime_probe"},
	)
	if metadata.RuntimeFeatureEvidence.EvidenceClass != "lowered_ir_static_plan" ||
		metadata.RuntimeFeatureEvidence.Method != "backend_report_lowered_ir_scan_v1" ||
		metadata.RuntimeFeatureEvidence.SourceArtifact != binary+".backend.json" {
		t.Fatalf(
			"runtime feature evidence = %#v, want backend report lowered IR evidence",
			metadata.RuntimeFeatureEvidence,
		)
	}
	if metadata.RuntimeObjectPlan.EvidenceClass != "native_runtime_object_plan" ||
		metadata.RuntimeObjectPlan.EvidenceMethod != "native_link_runtime_object_plan_v1" ||
		!metadata.RuntimeObjectPlan.RuntimeUsed ||
		!metadata.RuntimeObjectPlan.RuntimeObjectLinked ||
		!metadata.RuntimeObjectPlan.RuntimeObjectInitialized {
		t.Fatalf(
			"runtime object plan = %#v, want linked native runtime object plan",
			metadata.RuntimeObjectPlan,
		)
	}
	assertStringSlice(
		t,
		"runtime_object_features_required",
		metadata.RuntimeObjectPlan.RuntimeObjectFeaturesRequired,
		[]string{"actor_runtime", "task_runtime"},
	)
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

func TestGeneratedRSSBudgetPolicyContainsMeasuredTetraRowsWithRSSPeakEvidence(t *testing.T) {
	report := rssPolicyFixtureReport()
	policy := buildLocalRSSBudgetPolicy(report)

	if policy.Schema != "tetra.local_benchmark.rss_budget_policy.v1" {
		t.Fatalf("schema = %q", policy.Schema)
	}
	if policy.Target != "linux-x64" {
		t.Fatalf("target = %q, want linux-x64", policy.Target)
	}
	if len(policy.Budgets) != 2 {
		t.Fatalf("budgets len = %d, want 2: %#v", len(policy.Budgets), policy.Budgets)
	}
	want := map[string]uint64{
		"integer loops": 111,
		"slice sum":     222,
	}
	for _, budget := range policy.Budgets {
		if budget.Language != "tetra" {
			t.Fatalf("budget language = %q, want tetra: %#v", budget.Language, budget)
		}
		if budget.AllowedVariancePercent != 5 {
			t.Fatalf("budget variance = %v, want 5: %#v", budget.AllowedVariancePercent, budget)
		}
		if !strings.Contains(budget.Reason, "host-pinned") {
			t.Fatalf("budget reason = %q, want host-pinned local reason", budget.Reason)
		}
		wantBytes, ok := want[budget.Category]
		if !ok {
			t.Fatalf("unexpected budget category %q in %#v", budget.Category, policy.Budgets)
		}
		if budget.RSSPeakBudgetBytes != wantBytes {
			t.Fatalf(
				"budget %s rss_peak_budget_bytes = %d, want %d",
				budget.Category,
				budget.RSSPeakBudgetBytes,
				wantBytes,
			)
		}
		delete(want, budget.Category)
	}
	if len(want) != 0 {
		t.Fatalf("missing budget categories: %#v", want)
	}
	for _, forbidden := range []string{"c", "cpp", "rust"} {
		for _, budget := range policy.Budgets {
			if budget.Language == forbidden {
				t.Fatalf("generated policy included non-Tetra budget: %#v", budget)
			}
		}
	}
}

func TestGeneratedRSSBudgetPolicyHostProfileMatchesReportHost(t *testing.T) {
	report := rssPolicyFixtureReport()
	policy := buildLocalRSSBudgetPolicy(report)

	want := localRSSBudgetHost{
		GOOS:      report.Host.GOOS,
		GOARCH:    report.Host.GOARCH,
		CPUs:      report.Host.CPUs,
		TargetCPU: report.Host.TargetCPU,
		GitCommit: report.Host.GitCommit,
	}
	if policy.HostProfile != want {
		t.Fatalf("host_profile = %#v, want %#v", policy.HostProfile, want)
	}
	assertStringSlice(t, "non_claims", policy.NonClaims, []string{
		"local RSS budget only",
		"no cross-machine RSS claim",
		"no official benchmark claim",
	})
}

func TestGeneratedRSSBudgetPolicyValidatesSameReportFixture(t *testing.T) {
	report := rssPolicyFixtureReport()
	policy := buildLocalRSSBudgetPolicy(report)

	for _, budget := range policy.Budgets {
		row, ok := findReportRow(report, budget.Category, budget.Language)
		if !ok {
			t.Fatalf("budget has no matching report row: %#v", budget)
		}
		if row.Status != "measured" {
			t.Fatalf("budget matched non-measured row: %#v", row)
		}
		if row.TetraMetadata == nil || row.TetraMetadata.MemoryEvidence == nil {
			t.Fatalf("budget matched row without memory evidence: %#v", row)
		}
		peak := row.TetraMetadata.MemoryEvidence.RSSPeak
		if peak.EvidenceClass != "runtime_measured" ||
			peak.SourceArtifact == "" ||
			budget.RSSPeakBudgetBytes != peak.PeakBytes {
			t.Fatalf("budget %#v does not match row rss_peak %#v", budget, peak)
		}
	}
}

func TestWriteGeneratedRSSBudgetPolicyReplacesStalePolicy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rss-budget-policy.local.json")
	if err := os.WriteFile(path, []byte(`{"stale":true}`), 0o644); err != nil {
		t.Fatalf("write stale policy: %v", err)
	}
	if err := writeLocalRSSBudgetPolicy(path, rssPolicyFixtureReport()); err != nil {
		t.Fatalf("writeLocalRSSBudgetPolicy: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read policy: %v", err)
	}
	if strings.Contains(string(raw), "stale") {
		t.Fatalf("stale policy content was retained: %s", raw)
	}
	var policy localRSSBudgetPolicy
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatalf("unmarshal policy: %v\n%s", err, raw)
	}
	if len(policy.Budgets) != 2 {
		t.Fatalf("budgets len = %d, want 2: %s", len(policy.Budgets), raw)
	}
}

func TestMissingTetraMetadataAttachesBlockedMemoryEvidence(t *testing.T) {
	metadata := missingTetraMetadata("missing-binary", "optimizer.json")
	if metadata.RuntimeFeatureEvidence.EvidenceClass != "blocked" ||
		metadata.RuntimeFeatureEvidence.BlockedReason == "" {
		t.Fatalf(
			"runtime feature evidence = %#v, want blocked reason",
			metadata.RuntimeFeatureEvidence,
		)
	}
	if metadata.RuntimeObjectPlan.EvidenceClass != "blocked" ||
		metadata.RuntimeObjectPlan.BlockedReason == "" {
		t.Fatalf("runtime object plan = %#v, want blocked reason", metadata.RuntimeObjectPlan)
	}
	assertStringSlice(t, "runtime_features_required", metadata.RuntimeFeaturesRequired, []string{})
	assertStringSlice(t, "runtime_features_linked", metadata.RuntimeFeaturesLinked, []string{})
	assertStringSlice(
		t,
		"runtime_features_initialized",
		metadata.RuntimeFeaturesInitialized,
		[]string{},
	)
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
	if len(metadata.MemoryEvidence.DomainBytes) != 0 ||
		metadata.MemoryEvidence.DomainBytesEvidence.EvidenceClass != "blocked" {
		t.Fatalf(
			"domain evidence = %#v/%#v, want blocked empty domains",
			metadata.MemoryEvidence.DomainBytes,
			metadata.MemoryEvidence.DomainBytesEvidence,
		)
	}
}

func rssPolicyFixtureReport() tier1Report {
	return tier1Report{
		Host: tier1Host{
			GOOS:      "linux",
			GOARCH:    "amd64",
			CPUs:      8,
			TargetCPU: "test cpu",
			GitCommit: "abcdef",
		},
		Results: []categoryResult{
			{
				Category: "integer loops",
				Rows: []benchmarkRow{
					rssPolicyFixtureTetraRow("integer loops", "integer_loops_tetra", 111),
					{
						Name:     "integer_loops_c",
						Category: "integer loops",
						Language: "c",
						Status:   "measured",
					},
				},
			},
			{
				Category: "slice sum",
				Rows: []benchmarkRow{
					rssPolicyFixtureTetraRow("slice sum", "slice_sum_tetra", 222),
				},
			},
			{
				Category: "allocation",
				Rows: []benchmarkRow{
					{
						Name:     "allocation_tetra",
						Category: "allocation",
						Language: "tetra",
						Status:   "run_failed",
						TetraMetadata: &tetraMetadata{MemoryEvidence: &memoryEvidence{
							RSSPeak: rssPolicyFixtureRSSPeak(333),
						}},
					},
				},
			},
			{
				Category: "function calls",
				Rows: []benchmarkRow{
					{
						Name:     "function_calls_tetra",
						Category: "function calls",
						Language: "tetra",
						Status:   "measured",
						TetraMetadata: &tetraMetadata{MemoryEvidence: &memoryEvidence{
							RSSPeak: unsupportedMemoryMetric("not_collected", "fixture lacks RSS"),
						}},
					},
				},
			},
		},
	}
}

func rssPolicyFixtureTetraRow(category string, name string, peak uint64) benchmarkRow {
	return benchmarkRow{
		Name:     name,
		Category: category,
		Language: "tetra",
		Status:   "measured",
		TetraMetadata: &tetraMetadata{MemoryEvidence: &memoryEvidence{
			RSSPeak: rssPolicyFixtureRSSPeak(peak),
		}},
	}
}

func rssPolicyFixtureRSSPeak(peak uint64) memoryMetric {
	return memoryMetric{
		Bytes:          peak,
		PeakBytes:      peak,
		EvidenceClass:  "runtime_measured",
		Method:         rsstelemetry.MethodLinuxWait4RusageMaxRSSV1,
		SourceArtifact: "artifacts/rss-fixture.json",
	}
}

func findReportRow(report tier1Report, category string, language string) (benchmarkRow, bool) {
	for _, result := range report.Results {
		if result.Category != category {
			continue
		}
		return rowForLanguage(result.Rows, language)
	}
	return benchmarkRow{}, false
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

func firstActorDomain(domains []memoryDomainByte) (memoryDomainByte, bool) {
	for _, domain := range domains {
		if domain.Kind == "actor" {
			return domain, true
		}
	}
	return memoryDomainByte{}, false
}

func readText(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "<unavailable: " + err.Error() + ">"
	}
	return string(raw)
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
