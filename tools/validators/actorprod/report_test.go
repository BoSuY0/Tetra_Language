package actorprod

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testGitHead = "e2c19b8ee276158f8eb2c54cf61e11bd84952893"

func TestValidateReportAcceptsActorFoundationEvidence(t *testing.T) {
	raw := validActorFoundationReport(t)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsMissingParallelAndDistributedEvidence(t *testing.T) {
	raw := validActorFoundationReportFrom(t, func(report *Report) {
		report.Artifacts = []ArtifactReport{
			{
				Path:   "actor-runtime-foundation-manifest.json",
				Kind:   "foundation_manifest",
				Schema: SchemaV1,
			},
			{Path: "artifact-hashes.json", Kind: "hash_manifest", Schema: ArtifactHashSchema},
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing subreports to fail")
	}
	for _, want := range []string{
		"parallel-production-linux-x64.json",
		"distributed-actors-linux-x64.json",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsFakeOrBuildOnlyEvidence(t *testing.T) {
	raw := validActorFoundationReportFrom(t, func(report *Report) {
		report.Commands[0].Command = "echo docs-only fake build-only actor evidence"
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected fake/build-only evidence to fail")
	}
	for _, want := range []string{"docs-only", "fake", "build-only"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsCrossTargetDistributedActorClaimWithoutSmoke(t *testing.T) {
	for _, claim := range []string{
		"windows-x64 distributed actor runtime evidence",
		"macos-x64 distributed actor runtime evidence",
	} {
		t.Run(claim, func(t *testing.T) {
			raw := validActorFoundationReportFrom(t, func(report *Report) {
				report.Claims = []string{claim}
			})
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected cross-target distributed actor claim to fail")
			}
			if !strings.Contains(err.Error(), "cross-target distributed actor claim") {
				t.Fatalf("error = %v, want cross-target distributed actor claim rejection", err)
			}
		})
	}
}

func TestValidateReportRejectsDistributedZeroCopySpellingVariants(t *testing.T) {
	for _, claim := range []string{
		"linux-x64 actor foundation proves distributed zero copy actor transfer",
		"linux-x64 actor foundation proves cross-node zero-copy actor transfer",
	} {
		t.Run(claim, func(t *testing.T) {
			raw := validActorFoundationReportFrom(t, func(report *Report) {
				report.Claims = []string{claim}
			})
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected distributed zero-copy spelling variant to fail")
			}
			if !strings.Contains(strings.ToLower(err.Error()), "zero") {
				t.Fatalf("error = %v, want zero-copy rejection", err)
			}
		})
	}
}

func TestValidateReportRejectsMissingArtifactHashes(t *testing.T) {
	raw := validActorFoundationReportFrom(t, func(report *Report) {
		report.ArtifactHashes = ""
		report.Artifacts = removeActorFoundationArtifact(report.Artifacts, "artifact-hashes.json")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing artifact hashes to fail")
	}
	if !strings.Contains(err.Error(), "artifact-hashes.json") {
		t.Fatalf("error = %v, want artifact-hashes rejection", err)
	}
}

func TestValidateReportRejectsGitHeadMismatch(t *testing.T) {
	raw := validActorFoundationReportFrom(t, func(report *Report) {
		report.GitHead = strings.Repeat("a", 40)
	})
	err := ValidateReportWithOptions(raw, Options{CurrentGitHead: testGitHead})
	if err == nil {
		t.Fatalf("expected git head mismatch to fail")
	}
	if !strings.Contains(err.Error(), "git_head") {
		t.Fatalf("error = %v, want git_head mismatch", err)
	}
}

func TestValidateReportDirCrossChecksSubreportsAndArtifactHashes(t *testing.T) {
	dir := t.TempDir()
	writeActorFoundationFixtureDir(t, dir)
	if err := ValidateReportDir(dir, Options{CurrentGitHead: testGitHead}); err != nil {
		t.Fatalf("ValidateReportDir failed: %v", err)
	}
}

func TestValidateReportDirRejectsMissingSubreportsAndHashManifests(t *testing.T) {
	for _, tc := range []struct {
		name      string
		remove    string
		wantError string
	}{
		{
			name:      "missing parallel subreport",
			remove:    "parallel-production-linux-x64/parallel-production-linux-x64.json",
			wantError: "parallel-production-linux-x64.json",
		},
		{
			name:      "missing distributed subreport",
			remove:    "distributed-actors-linux-x64/distributed-actors-linux-x64.json",
			wantError: "distributed-actors-linux-x64.json",
		},
		{
			name:      "missing foundation hash manifest",
			remove:    "artifact-hashes.json",
			wantError: "artifact-hashes.json",
		},
		{
			name:      "missing parallel hash manifest",
			remove:    "parallel-production-linux-x64/artifact-hashes.json",
			wantError: "parallel-production-linux-x64/artifact-hashes.json",
		},
		{
			name:      "missing distributed hash manifest",
			remove:    "distributed-actors-linux-x64/artifact-hashes.json",
			wantError: "distributed-actors-linux-x64/artifact-hashes.json",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeActorFoundationFixtureDir(t, dir)
			if err := os.Remove(filepath.Join(dir, filepath.FromSlash(tc.remove))); err != nil {
				t.Fatalf("remove fixture artifact: %v", err)
			}
			err := ValidateReportDir(dir, Options{CurrentGitHead: testGitHead})
			if err == nil {
				t.Fatalf("expected missing %s to fail", tc.remove)
			}
			if !strings.Contains(filepath.ToSlash(err.Error()), tc.wantError) {
				t.Fatalf("error = %v, want %s", err, tc.wantError)
			}
		})
	}
}

func TestValidateReportDirRejectsDistributedSubreportStaleGitHead(t *testing.T) {
	dir := t.TempDir()
	writeActorFoundationFixtureDir(t, dir)
	stale := strings.Replace(
		validDistributedActorSubreport,
		testGitHead,
		strings.Repeat("a", 40),
		1,
	)
	writeFile(
		t,
		filepath.Join(dir, "distributed-actors-linux-x64", "distributed-actors-linux-x64.json"),
		stale,
	)
	writeArtifactHashManifest(t, filepath.Join(dir, "distributed-actors-linux-x64"), []testArtifact{
		{Path: "distributed-actors-linux-x64.json", Schema: "tetra.actors.distributed-runtime.v1"},
	})
	writeArtifactHashManifest(t, dir, []testArtifact{
		{Path: "actor-runtime-foundation-manifest.json", Schema: SchemaV1},
		{Path: "distributed-actors-linux-x64/artifact-hashes.json", Schema: ArtifactHashSchema},
		{
			Path:   "distributed-actors-linux-x64/distributed-actors-linux-x64.json",
			Schema: "tetra.actors.distributed-runtime.v1",
		},
		{Path: "logs/focused-actor-tests.log"},
		{Path: "parallel-production-linux-x64/artifact-hashes.json", Schema: ArtifactHashSchema},
		{
			Path:   "parallel-production-linux-x64/parallel-production-linux-x64.json",
			Schema: "tetra.parallel.production.v1",
		},
	})
	err := ValidateReportDir(dir, Options{CurrentGitHead: testGitHead})
	if err == nil {
		t.Fatalf("expected stale distributed subreport git_head to fail")
	}
	if !strings.Contains(err.Error(), "distributed actor git_head") {
		t.Fatalf("error = %v, want distributed actor git_head mismatch", err)
	}
}

func validActorFoundationReport(t *testing.T) []byte {
	t.Helper()
	return validActorFoundationReportFrom(t, func(*Report) {})
}

func validActorFoundationReportFrom(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	report := Report{
		Schema:         SchemaV1,
		Status:         "pass",
		Target:         "linux-x64",
		Host:           "linux-x64",
		GitHead:        testGitHead,
		ReportDir:      ".",
		ArtifactHashes: "artifact-hashes.json",
		Claims: []string{
			"linux-x64 scoped actor/task runtime foundation evidence",
		},
		NonClaims: []string{
			"no full Erlang/OTP actor runtime claim",
			"no cluster membership or reconnect/retry production claim",
			"no non-Linux distributed actor runtime support claim",
			"no distributed zero-copy pointer or region transfer claim",
			"no formal race proof claim",
		},
		Commands: []CommandReport{
			{
				Name: "distributed-actors-smoke",
				Command: ("bash " +
					"scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh" +
					" --report-dir " +
					"reports/actor-runtime-foundation/final/distributed-actors-li" +
					"nux-x64"),
				Status: "pass",
				Log:    "logs/distributed-actors-smoke.log",
			},
			{
				Name: "parallel-production-smoke",
				Command: ("bash " +
					"scripts/release/post_v0_4/parallel-production-linux-x64-smok" +
					"e.sh --report-dir " +
					"reports/actor-runtime-foundation/final/parallel-production-l" +
					"inux-x64"),
				Status: "pass",
				Log:    "logs/parallel-production-smoke.log",
			},
			{
				Name: "focused-actor-tests",
				Command: ("go test -buildvcs=false ./cli/cmd/tetra " +
					"./compiler/tests/ownership " +
					"./compiler/tests/ownership/actor_task ./compiler -run " +
					"'Diagnostic|Actor|Backpressure|Invalid|Closed|Transfer' " +
					"-count=1"),
				Status: "pass",
				Log:    "logs/focused-actor-tests.log",
			},
			{
				Name: "race-actor-slice",
				Command: ("go test -race -buildvcs=false ./compiler " +
					"./cli/internal/actornet -run 'Actor|Broker' -count=1"),
				Status: "pass",
				Log:    "logs/race-actor-slice.log",
			},
			{
				Name:    "validate-manifest",
				Command: "go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json",
				Status:  "pass",
				Log:     "logs/validate-manifest.log",
			},
			{
				Name:    "verify-docs",
				Command: "go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
				Status:  "pass",
				Log:     "logs/verify-docs.log",
			},
			{
				Name: "artifact-hashes-write",
				Command: ("go run ./tools/cmd/validate-artifact-hashes --write --root " +
					"reports/actor-runtime-foundation/final --out " +
					"reports/actor-runtime-foundation/final/artifact-hashes.json"),
				Status: "pass",
				Log:    "logs/artifact-hashes-write.log",
			},
			{
				Name: "artifact-hashes-validate",
				Command: ("go run ./tools/cmd/validate-artifact-hashes --manifest " +
					"reports/actor-runtime-foundation/final/artifact-hashes.json"),
				Status: "pass",
				Log:    "logs/artifact-hashes-validate.log",
			},
			{
				Name: "actor-foundation-validator",
				Command: ("go run ./tools/cmd/validate-actor-runtime-foundation " +
					"--report-dir reports/actor-runtime-foundation/final " +
					"--current-git-head ") + testGitHead,
				Status: "pass",
				Log:    "logs/actor-foundation-validator.log",
			},
		},
		Artifacts: []ArtifactReport{
			{
				Path:   "actor-runtime-foundation-manifest.json",
				Kind:   "foundation_manifest",
				Schema: SchemaV1,
			},
			{
				Path:   "parallel-production-linux-x64/parallel-production-linux-x64.json",
				Kind:   "parallel_production_report",
				Schema: "tetra.parallel.production.v1",
			},
			{
				Path:   "parallel-production-linux-x64/artifact-hashes.json",
				Kind:   "parallel_hash_manifest",
				Schema: ArtifactHashSchema,
			},
			{
				Path:   "distributed-actors-linux-x64/distributed-actors-linux-x64.json",
				Kind:   "distributed_actor_runtime_report",
				Schema: "tetra.actors.distributed-runtime.v1",
			},
			{
				Path:   "distributed-actors-linux-x64/artifact-hashes.json",
				Kind:   "distributed_hash_manifest",
				Schema: ArtifactHashSchema,
			},
			{
				Path:   "artifact-hashes.json",
				Kind:   "foundation_hash_manifest",
				Schema: ArtifactHashSchema,
			},
		},
	}
	mutate(&report)
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func removeActorFoundationArtifact(artifacts []ArtifactReport, path string) []ArtifactReport {
	var kept []ArtifactReport
	for _, artifact := range artifacts {
		if artifact.Path != path {
			kept = append(kept, artifact)
		}
	}
	return kept
}

func writeActorFoundationFixtureDir(t *testing.T, dir string) {
	t.Helper()
	writeFile(
		t,
		filepath.Join(dir, "actor-runtime-foundation-manifest.json"),
		string(validActorFoundationReport(t)),
	)
	writeFile(
		t,
		filepath.Join(dir, "parallel-production-linux-x64", "parallel-production-linux-x64.json"),
		validParallelProductionSubreport,
	)
	writeArtifactHashManifest(
		t,
		filepath.Join(dir, "parallel-production-linux-x64"),
		[]testArtifact{
			{Path: "parallel-production-linux-x64.json", Schema: "tetra.parallel.production.v1"},
		},
	)
	writeFile(
		t,
		filepath.Join(dir, "distributed-actors-linux-x64", "distributed-actors-linux-x64.json"),
		validDistributedActorSubreport,
	)
	writeArtifactHashManifest(t, filepath.Join(dir, "distributed-actors-linux-x64"), []testArtifact{
		{Path: "distributed-actors-linux-x64.json", Schema: "tetra.actors.distributed-runtime.v1"},
	})
	writeFile(t, filepath.Join(dir, "logs", "focused-actor-tests.log"), "ok\n")
	writeArtifactHashManifest(t, dir, []testArtifact{
		{Path: "actor-runtime-foundation-manifest.json", Schema: SchemaV1},
		{Path: "distributed-actors-linux-x64/artifact-hashes.json", Schema: ArtifactHashSchema},
		{
			Path:   "distributed-actors-linux-x64/distributed-actors-linux-x64.json",
			Schema: "tetra.actors.distributed-runtime.v1",
		},
		{Path: "logs/focused-actor-tests.log"},
		{Path: "parallel-production-linux-x64/artifact-hashes.json", Schema: ArtifactHashSchema},
		{
			Path:   "parallel-production-linux-x64/parallel-production-linux-x64.json",
			Schema: "tetra.parallel.production.v1",
		},
	})
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

type testArtifact struct {
	Path   string
	Schema string
}

var validParallelProductionActorMemoryDomains = strings.Join([]string{
	("[{\"schema_version\":\"tetra.actors.memory-domain.v1\",\"actor_id\":\"acto" +
		"r-mailbox-copy\",\"evidence_class\":\"local_parallelrt_model\",\"evidence_" +
		"method\":\"parallelrt_typed_mailbox_memory_domain_v1\",\"runtime_measure" +
		"d\":false,\"runtime_blocked_reason\":\"production actor runtime per-acto" +
		"r byte sampler is not implemented; this is local parallelrt model ev" +
		"idence\",\"domain\":{\"domain_id\":\"domain:actor:actor-mailbox-copy\",\"k" +
		"ind\":\"actor\",\"owner_kind\":\"actor\",\"owner_id\":\"actor-mailbox-copy" +
		"\",\"lifetime\":\"actor:actor-mailbox-copy\",\"budget_bytes\":256,\"reques" +
		"ted_bytes\":48,\"reserved_bytes\":256,\"committed_bytes\":256,\"current_by" +
		"tes\":48,\"peak_bytes\":48,\"copy_count\":1,\"bytes_copied\":32},\"mailbox" +
		"\":{\"capacity_messages\":4,\"queued_messages\":1,\"capacity_bytes\":256," +
		"\"queued_bytes\":48,\"peak_queued_bytes\":48,\"message_bytes\":16,\"backpr" +
		"essure_mode\":\"blocking_recv_yield\"},\"message_pool\":{\"slab_bytes\":64" +
		",\"live_bytes\":48,\"capacity_bytes\":256,\"message_slots_live\":1,\"messa" +
		"ge_slots_limit\":4},\"backpressure\":{\"mode\":\"blocking_recv_yield\",\"s" +
		"tatus\":\"available\"},\"non_claims\":[\"full production actor runtime is " +
		"not claimed\",\"distributed actor zero-copy is not claimed\",\"actor mem" +
		"ory domain bytes are model/report evidence unless paired with runtim" +
		"e measurement\"],\"production_runtime_claimed\":false,\"distributed_zero" +
		"_copy_claimed\":false},{\"schema_version\":\"tetra.actors.memory-domain." +
		"v1\",\"actor_id\":\"actor-frame\",\"evidence_class\":\"local_parallelrt_mo" +
		"del\",\"evidence_method\":\"parallelrt_typed_mailbox_memory_domain_v1\",\"" +
		"runtime_measured\":false,\"runtime_blocked_reason\":\"production actor r" +
		"untime per-actor byte sampler is not implemented; this is local para" +
		"llelrt model evidence\",\"domain\":{\"domain_id\":\"domain:actor:actor-fra" +
		"me\",\"kind\":\"actor\",\"owner_kind\":\"actor\",\"owner_id\":\"actor-fram" +
		"e\",\"lifetime\":\"actor:actor-frame\",\"budget_bytes\":512,\"requested_by" +
		"tes\":272,\"reserved_bytes\":512,\"committed_bytes\":512,\"current_bytes\"" +
		":272,\"peak_bytes\":272},\"mailbox\":{\"capacity_messages\":2,\"queued_mes" +
		"sages\":1,\"capacity_bytes\":512,\"queued_bytes\":272,\"peak_queued_bytes" +
		"\":272,\"message_bytes\":16,\"backpressure_mode\":\"blocking_recv_yield\"}" +
		",\"message_pool\":{\"slab_bytes\":32,\"live_bytes\":272,\"capacity_bytes\"" +
		":512,\"message_slots_live\":1,\"message_slots_limit\":2},\"owned_regions\"" +
		":[{\"region_name\":\"frame\",\"domain_id\":\"domain:actor:actor-frame\",\"" +
		"owner_id\":\"actor-frame\",\"bytes\":256}],\"backpressure\":{\"mode\":\"bl" +
		"ocking_recv_yield\",\"status\":\"byte_limit_reached\",\"reason\":\"mailbox" +
		" byte capacity reached\"},\"non_claims\":[\"full production actor runtim" +
		"e is not claimed\",\"distributed actor zero-copy is not claimed\",\"acto" +
		"r memory domain bytes are model/report evidence unless paired with r" +
		"untime measurement\"],\"production_runtime_claimed\":false,\"distributed" +
		"_zero_copy_claimed\":false}]"),
}, "\n")

func writeArtifactHashManifest(t *testing.T, root string, artifacts []testArtifact) {
	t.Helper()
	var rows []map[string]any
	for _, artifact := range artifacts {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(artifact.Path)))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		rows = append(rows, map[string]any{
			"path":   artifact.Path,
			"sha256": fmt.Sprintf("sha256:%x", sum),
			"size":   len(raw),
			"schema": artifact.Schema,
		})
	}
	manifest := map[string]any{
		"schema":    ArtifactHashSchema,
		"root":      ".",
		"artifacts": rows,
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "artifact-hashes.json"), string(raw)+"\n")
}

var validParallelProductionSubreport = mustMarshalJSON(map[string]any{
	"schema":               "tetra.parallel.production.v1",
	"status":               "pass",
	"target":               "linux-x64",
	"host":                 "linux-x64",
	"runtime":              "parallel-linux-x64",
	"source":               "tools/cmd/parallel-production-smoke",
	"processes":            parallelProductionProcesses(),
	"benchmarks":           parallelProductionBenchmarks(),
	"actor_memory_domains": json.RawMessage(validParallelProductionActorMemoryDomains),
	"contracts":            parallelProductionContracts(),
	"cases":                parallelProductionCases(),
	"diagnostics":          parallelProductionDiagnostics(),
	"audit":                parallelProductionAudit(),
})

func mustMarshalJSON(value any) string {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(raw) + "\n"
}

func parallelProductionProcesses() []map[string]any {
	return []map[string]any{
		processRow("tetra build", "build", "go build ./cli/cmd/tetra"),
		processRow("parallel smoke app", "app", "parallel-smoke"),
		processRow("parallel stress", "stress", "parallel-stress"),
		processRow("parallel scheduler prototype", "benchmark", "compiler/internal/parallelrt"),
	}
}

func processRow(name, kind, path string) map[string]any {
	return map[string]any{
		"name":      name,
		"kind":      kind,
		"path":      path,
		"ran":       true,
		"pass":      true,
		"exit_code": 0,
	}
}

func parallelProductionBenchmarks() []map[string]any {
	return []map[string]any{
		benchmarkPrep(
			"actor ping-pong benchmark prep",
			"actor_benchmark_prep",
			"messages_round_trip",
			"compiler/compiler_suite_test.go::TestActorsPingPongBuildAndRun and "+
				"examples/actors/actors_pingpong.tetra define the local Linux-x64 "+
				"actor ping-pong workload candidate",
			"Actor ping-pong benchmark prep row exists as Tier 0 local smoke only; "+
				"no measured result is published and cross-runtime comparison is out of scope.",
		),
		benchmarkPrep(
			"actor fanout/fanin benchmark prep",
			"actor_benchmark_prep",
			"fanout_fanin_messages",
			"compiler/internal/parallelrt two-core work stealing model checks "+
				"actor fanout/fanin scheduling shape without publishing throughput",
			"Actor fanout/fanin benchmark prep row exists as Tier 0 local smoke only; "+
				"it records local workload shape and leaves public benchmark publication "+
				"out of scope.",
		),
		benchmarkPrep(
			"actor mailbox throughput benchmark prep",
			"actor_benchmark_prep",
			"mailbox_messages",
			"compiler/internal/parallelrt TypedMailbox and parallel production "+
				"actor mailbox cases define the local mailbox throughput workload candidate",
			"Actor mailbox throughput benchmark prep row exists as Tier 0 local smoke only; "+
				"it publishes no measured result and no throughput guarantee.",
		),
		benchmarkPrep(
			"actor backpressure latency benchmark prep",
			"actor_benchmark_prep",
			"backpressure_wait",
			"compiler/internal/parallelrt ErrMailboxFull and blocking_recv_yield "+
				"metadata define the local backpressure latency diagnostic candidate",
			"Actor backpressure latency benchmark prep row exists as Tier 0 local smoke only; "+
				"no real-world SLA is claimed.",
		),
		benchmarkPrep(
			"zero_copy_move local typed mailbox benchmark prep",
			"actor_transfer_prep",
			"owned_region_transfer",
			"compiler/internal/parallelrt owned-region transfer report emits "+
				"zero_copy_move for local typed mailbox metadata only",
			"zero_copy_move local typed mailbox benchmark prep row exists as Tier 0 "+
				"local smoke only; it records local owned-region metadata and leaves "+
				"distributed or network transfer behavior out of scope.",
		),
	}
}

func benchmarkPrep(name, kind, metric, evidence, claim string) map[string]any {
	return map[string]any{
		"name":                 name,
		"kind":                 kind,
		"metric":               metric,
		"unit":                 "prep_only",
		"baseline_value":       0,
		"measured_value":       0,
		"improvement_ratio":    0.0,
		"evidence":             evidence,
		"claim_tier":           "tier0_local_smoke_only",
		"claim":                claim,
		"raw_output_artifacts": []string{parallelRTEvidenceArtifact},
		"ran":                  false,
		"pass":                 true,
	}
}

var parallelRTEvidenceArtifact = "reports/actor-runtime-foundation/P15/" +
	"parallelrt-evidence.raw.json"

func parallelProductionContracts() []map[string]any {
	return []map[string]any{
		contractRow(
			"production task scheduler",
			"scheduler fairness and lifecycle cases ran on linux-x64",
		),
		contractRow(
			"join cancel deadline select group lifecycle",
			"join, cancel, deadline, select, and group lifecycle diagnostics are stable",
		),
		contractRow(
			"actor mailbox backpressure and failure handling",
			"mailbox capacity, message pool exhaustion, and actor failure cases are covered",
		),
		contractRow(
			"task actor thread boundary transfer rules",
			"ownership transfer diagnostics and actor/island boundary proof protect "+
				"task, actor, and thread boundaries",
		),
		contractRow(
			"race safety model",
			"shared mutable state crossing parallel boundaries is rejected "+
				"conservatively with matrix evidence",
		),
		contractRow(
			"safe unsafe forbidden parallelism boundary",
			"docs and diagnostics define safe, unsafe, and forbidden parallel behavior",
		),
	}
}

func contractRow(name, evidence string) map[string]any {
	return map[string]any{
		"name":     name,
		"status":   "pass",
		"evidence": evidence,
	}
}

func parallelProductionCases() []map[string]any {
	return []map[string]any{
		caseRow("scheduler fairness", "positive"),
		caseRow("task join lifecycle", "positive"),
		caseError("task cancellation", "negative", "cancelled"),
		caseError("deadline timeout", "negative", "deadline"),
		caseRow("select readiness", "positive"),
		caseRow("task group lifecycle", "positive"),
		caseError("task group cancel wakes deadline join", "negative", "cancelled before deadline"),
		caseError("actor recv cancel wake", "negative", "actor recv cancel wake"),
		caseRow("nested cancellation propagation", "positive"),
		caseRow("task actor mailbox handoff", "positive"),
		caseError("actor mailbox backpressure", "negative", "backpressure"),
		caseError("message pool exhaustion", "negative", "message pool exhausted"),
		caseError("invalid actor handle send", "negative", "invalid actor handle"),
		caseError("done actor send", "negative", "done actor"),
		caseError("actor failure handling", "negative", "actor failed"),
		caseError("invalid handle diagnostics", "negative", "invalid handle"),
		caseError("resource double join diagnostic", "negative", "joined"),
		caseError("task group use-after-close diagnostic", "negative", "closed"),
		caseError("ownership transfer across task boundary", "negative", "transfer"),
		caseError("ownership transfer across actor boundary", "negative", "transfer"),
		caseError("race-safety shared mutable rejection", "negative", "shared mutable"),
		caseRow("race-safety rejection matrix", "positive"),
		caseRow("actor island boundary proof", "positive"),
		caseRow("actor broker leak cleanup", "positive"),
		caseRow("safe unsafe forbidden boundary coverage", "positive"),
		stressCase("actor fanout mailbox drain soak", 512, "actor-fanout-mailbox-drain-v1", 90000),
		stressCase("many tasks stress", 64, "task-bounded-stress-seed-17", 10000),
		stressCase("many actor messages stress", 256, "actors-tagged-stress-v1", 10000),
		stressCase("cancellation storm", 16, "parallel-cancellation-storm-v1", 10000),
		stressCase("timeouts stress", 1, "deadline-aware-waits-v1", 10000),
	}
}

func caseRow(name, kind string) map[string]any {
	return map[string]any{"name": name, "kind": kind, "ran": true, "pass": true}
}

func caseError(name, kind, expected string) map[string]any {
	row := caseRow(name, kind)
	row["expected_error"] = expected
	return row
}

func stressCase(name string, iterations int, seed string, maxDurationMS int) map[string]any {
	row := caseRow(name, "stress")
	row["iterations"] = iterations
	row["deterministic_seed"] = seed
	row["max_duration_ms"] = maxDurationMS
	return row
}

func parallelProductionDiagnostics() []map[string]any {
	return []map[string]any{
		diagnosticRow("task cancellation", "TASK_CANCELLED", "task", "runtime", "cancelled"),
		diagnosticRow("deadline timeout", "TASK_DEADLINE_TIMEOUT", "task", "runtime", "deadline"),
		diagnosticRow(
			"task group cancel wakes deadline join",
			"TASK_GROUP_CANCEL_WAKE_JOIN",
			"task",
			"runtime",
			"cancelled before deadline",
		),
		diagnosticRow(
			"actor recv cancel wake",
			"ACTOR_RECV_CANCEL_WAKE",
			"actor",
			"runtime",
			"actor recv cancel wake",
		),
		diagnosticRow(
			"actor mailbox backpressure",
			"ACTOR_MAILBOX_BACKPRESSURE",
			"actor",
			"runtime",
			"backpressure",
		),
		diagnosticRow(
			"message pool exhaustion",
			"ACTOR_MESSAGE_POOL_EXHAUSTED",
			"actor",
			"runtime",
			"message pool exhausted",
		),
		diagnosticRow(
			"invalid actor handle send",
			"ACTOR_INVALID_HANDLE_SEND",
			"actor",
			"runtime",
			"invalid actor handle",
		),
		diagnosticRow("done actor send", "ACTOR_DONE_SEND", "actor", "runtime", "done actor"),
		diagnosticRow(
			"actor failure handling",
			"ACTOR_MISSING_NODE_FAILURE",
			"actor",
			"runtime",
			"actor failed",
		),
		diagnosticRow(
			"invalid handle diagnostics",
			"ACTOR_INVALID_HANDLE_DIAGNOSTIC",
			"actor",
			"cli-json",
			"invalid handle",
		),
		diagnosticRow(
			"resource double join diagnostic",
			"RESOURCE_DOUBLE_JOIN",
			"resource",
			"cli-json",
			"joined",
		),
		diagnosticRow(
			"task group use-after-close diagnostic",
			"TASK_GROUP_CLOSED",
			"task",
			"cli-json",
			"closed",
		),
		diagnosticRow(
			"ownership transfer across task boundary",
			"OWNERSHIP_TASK_TRANSFER",
			"ownership",
			"compiler",
			"transfer",
		),
		diagnosticRow(
			"ownership transfer across actor boundary",
			"OWNERSHIP_ACTOR_TRANSFER",
			"ownership",
			"compiler",
			"transfer",
		),
		diagnosticRow(
			"race-safety shared mutable rejection",
			"RACE_SHARED_MUTABLE_REJECTED",
			"race-safety",
			"compiler",
			"shared mutable",
		),
	}
}

func diagnosticRow(name, code, category, position, expected string) map[string]any {
	return map[string]any{
		"case":           name,
		"code":           code,
		"severity":       "error",
		"category":       category,
		"position":       position,
		"expected_error": expected,
	}
}

func parallelProductionAudit() []map[string]any {
	return []map[string]any{
		auditRow(
			"production task scheduler",
			"compiler/compiler_suite_test.go; compiler/internal/actorsrt/actorsrt_core.go",
			"scheduler fairness, many tasks stress, join, cancel, deadline, select, "+
				"and task group lifecycle cases ran",
		),
		auditRow(
			"join/cancel/deadline/select/group lifecycle",
			"compiler/compiler_suite_test.go; examples/tasks/task_bounded_stress.tetra",
			"required lifecycle cases cover join, cancellation, deadline timeout, "+
				"cancel-wakes-deadline-join, actor recv cancel wake, select readiness, "+
				"task group lifecycle, and nested cancellation propagation",
		),
		auditRow(
			"actor mailbox backpressure and failure handling",
			"compiler/compiler_suite_test.go; compiler/compiler_suite_test.go",
			"actor mailbox backpressure, checked message pool exhaustion, invalid actor "+
				"handle send, done actor send, task actor mailbox handoff, and actor "+
				"failure handling cases are required",
		),
		auditRow(
			"task/actor/thread-boundary transfer rules",
			"compiler/tests/ownership; cli/cmd/tetra/tetra_suite_test.go",
			"task and actor ownership transfer, actor/island boundary proof, resource "+
				"double join, and task group use-after-close diagnostics are required cases",
		),
		auditRow(
			"race-safety model or conservative rejections",
			"compiler/tests/ownership; docs/spec/runtime/actors.md",
			"shared mutable race-safety rejection and race-safety rejection matrix "+
				"evidence are required until a broader race-safe model is implemented",
		),
		auditRow(
			"stress evidence for tasks, actor messages, cancellation storms, and timeouts",
			"tools/cmd/parallel-production-smoke",
			"many tasks stress, many actor messages stress, actor fanout mailbox drain "+
				"soak, cancellation storm, timeouts stress, and actor broker leak cleanup "+
				"cases are required with bounded metadata",
		),
		auditRow(
			"safe/unsafe/forbidden parallelism documentation",
			"docs/spec/runtime/actors.md; docs/user/platform/async_actors_guide.md; "+
				"docs/spec/runtime/runtime_abi.md; "+
				"compiler/tests/semantics/semantics_async_ownership_test.go; "+
				"compiler/tests/safety/effects/effects_test.go",
			"documentation defines supported actor/task runtime, transfer boundaries, "+
				"and unsupported guarantees; safe unsafe forbidden boundary coverage runs "+
				"compiler tests for allowed immutable task targets, missing runtime/actors "+
				"effects, unsafe-only operations, and forbidden mutable actor/task targets",
		),
		auditRow(
			"stable parallel diagnostics",
			"compiler/compiler_suite_test.go; compiler/compiler_suite_test.go; "+
				"compiler/tests/ownership/actor_task/actor_task_ownership_test.go; "+
				"cli/cmd/tetra/tetra_suite_test.go",
			"negative parallel cases require stable expected_error evidence for "+
				"cancellation, deadline, backpressure, invalid handle, double join, "+
				"use-after-close, transfer, and shared mutable rejection diagnostics",
		),
		auditRow(
			"actor benchmark Tier 0/Tier 1 preparation",
			"compiler/internal/parallelrt; tools/cmd/parallel-production-smoke",
			"parallelrt evidence emits Tier 0 actor ping-pong, fanout/fanin, mailbox "+
				"throughput, backpressure latency, and zero_copy_move local typed mailbox "+
				"prep rows with raw artifact references; Tier 1 remains preparation-only "+
				"here, with no benchmark superiority, no C++/Rust parity, and no official "+
				"benchmark claim",
		),
		auditRow(
			"release-gate entrypoint",
			"scripts/release/post_v0_4/parallel-production-linux-x64-smoke.sh",
			"parallel production gate must run producer, validator, and artifact hash "+
				"validation",
		),
	}
}

func auditRow(requirement, artifact, evidence string) map[string]any {
	return map[string]any{
		"requirement": requirement,
		"artifact":    artifact,
		"evidence":    evidence,
		"result":      "pass",
	}
}

var validDistributedActorSubreport = strings.Join([]string{
	"",
	"{",
	"  \"schema\": \"tetra.actors.distributed-runtime.v1\",",
	"  \"status\": \"pass\",",
	"  \"target\": \"linux-x64\",",
	"  \"host\": \"linux-x64\",",
	"  \"runtime\": \"actornet\",",
	"  \"transport\": \"loopback-tcp\",",
	"  \"git_head\": \"e2c19b8ee276158f8eb2c54cf61e11bd84952893\",",
	"  \"artifact_hashes\": \"artifact-hashes.json\",",
	("  \"claims\": [\"linux-x64 loopback tcp distributed actor runtime evide" +
		"nce\"],"),
	("  \"nonclaims\": [\"no cluster membership\",\"no reconnect/retry producti" +
		"on\",\"no non-linux distributed actor runtime support\"],"),
	("  \"broker\": {\"runtime\":\"actornet\",\"transport\":\"loopback-tcp\",\"l" +
		"isten_addr\":\"127.0.0.1:47777\",\"accepted_connections\":8,\"routed_frame" +
		"s\":5,\"dropped_frames\":3,\"decode_errors\":3,\"expected_decode_errors\":" +
		"3,\"last_error\":\"actor wire: invalid slot count: 9\"},"),
	"  \"processes\": [",
	("    {\"name\":\"broker\",\"kind\":\"broker\",\"path\":\"./tetra actor-net" +
		"\",\"ran\":true,\"pass\":true,\"exit_code\":0},"),
	("    {\"name\":\"node-a\",\"kind\":\"node\",\"path\":\"node-a\",\"ran\":tru" +
		"e,\"pass\":true,\"exit_code\":0},"),
	("    {\"name\":\"node-b\",\"kind\":\"node\",\"path\":\"node-b\",\"ran\":tru" +
		"e,\"pass\":true,\"exit_code\":0}"),
	"  ],",
	("  \"frame_counts\": {\"hello\":2,\"hello_ack\":2,\"spawn_req\":1,\"spawn_a" +
		"ck\":1,\"send_i32\":1,\"send_msg\":1,\"send_typed\":1,\"node_down\":1,\"er" +
		"ror\":2},"),
	("  \"frame_order\": [\"hello\",\"hello_ack\",\"spawn_req\",\"spawn_ack\",\"" +
		"send_i32\",\"send_msg\",\"send_typed\",\"node_down\",\"error\",\"error\"],"),
	"  \"cases\": [",
	("    {\"name\":\"cross-node i32 send/receive\",\"ran\":true,\"pass\":true," +
		"\"expected_exit\":0,\"actual_exit\":0,\"node_processes\":2},"),
	("    {\"name\":\"cross-node tagged send/receive\",\"ran\":true,\"pass\":tru" +
		"e,\"expected_exit\":0,\"actual_exit\":0,\"node_processes\":2},"),
	("    {\"name\":\"cross-node typed send/receive\",\"ran\":true,\"pass\":true" +
		",\"expected_exit\":0,\"actual_exit\":0,\"node_processes\":2},"),
	("    {\"name\":\"missing-node failure/status\",\"ran\":true,\"pass\":true," +
		"\"expected_exit\":0,\"actual_exit\":0,\"node_processes\":1},"),
	("    {\"name\":\"task cancel/join compatibility\",\"ran\":true,\"pass\":tru" +
		"e,\"expected_exit\":0,\"actual_exit\":0,\"node_processes\":1},"),
	("    {\"name\":\"malformed frame length rejected\",\"kind\":\"network_negat" +
		"ive\",\"ran\":true,\"pass\":true,\"expected_exit\":0,\"actual_exit\":0,\"n" +
		"ode_processes\":0},"),
	("    {\"name\":\"duplicate node rejected\",\"kind\":\"network_negative\",\"" +
		"ran\":true,\"pass\":true,\"expected_exit\":0,\"actual_exit\":0,\"node_proc" +
		"esses\":0},"),
	("    {\"name\":\"unknown frame type rejected\",\"kind\":\"network_negative" +
		"\",\"ran\":true,\"pass\":true,\"expected_exit\":0,\"actual_exit\":0,\"node" +
		"_processes\":0},"),
	("    {\"name\":\"bad typed slot count rejected\",\"kind\":\"network_negativ" +
		"e\",\"ran\":true,\"pass\":true,\"expected_exit\":0,\"actual_exit\":0,\"nod" +
		"e_processes\":0},"),
	("    {\"name\":\"missing-node send after broker close\",\"kind\":\"network_" +
		"negative\",\"ran\":true,\"pass\":true,\"expected_exit\":0,\"actual_exit\":" +
		"0,\"node_processes\":0},"),
	("    {\"name\":\"forged source node rejected\",\"kind\":\"network_negative" +
		"\",\"ran\":true,\"pass\":true,\"expected_exit\":0,\"actual_exit\":0,\"node" +
		"_processes\":0}"),
	"  ]",
	"}",
	"",
}, "\n")
