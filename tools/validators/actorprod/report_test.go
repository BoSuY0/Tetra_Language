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
			{Path: "actor-runtime-foundation-manifest.json", Kind: "foundation_manifest", Schema: SchemaV1},
			{Path: "artifact-hashes.json", Kind: "hash_manifest", Schema: ArtifactHashSchema},
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing subreports to fail")
	}
	for _, want := range []string{"parallel-production-linux-x64.json", "distributed-actors-linux-x64.json"} {
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
			{Name: "distributed-actors-smoke", Command: "bash scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh --report-dir reports/actor-runtime-foundation/final/distributed-actors-linux-x64", Status: "pass", Log: "logs/distributed-actors-smoke.log"},
			{Name: "parallel-production-smoke", Command: "bash scripts/release/post_v0_4/parallel-production-linux-x64-smoke.sh --report-dir reports/actor-runtime-foundation/final/parallel-production-linux-x64", Status: "pass", Log: "logs/parallel-production-smoke.log"},
			{Name: "focused-actor-tests", Command: "go test -buildvcs=false ./cli/cmd/tetra ./compiler/tests/ownership ./compiler -run 'Diagnostic|Actor|Backpressure|Invalid|Closed|Transfer' -count=1", Status: "pass", Log: "logs/focused-actor-tests.log"},
			{Name: "race-actor-slice", Command: "go test -race -buildvcs=false ./compiler ./cli/internal/actornet -run 'Actor|Broker' -count=1", Status: "pass", Log: "logs/race-actor-slice.log"},
			{Name: "validate-manifest", Command: "go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json", Status: "pass", Log: "logs/validate-manifest.log"},
			{Name: "verify-docs", Command: "go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json", Status: "pass", Log: "logs/verify-docs.log"},
			{Name: "artifact-hashes-write", Command: "go run ./tools/cmd/validate-artifact-hashes --write --root reports/actor-runtime-foundation/final --out reports/actor-runtime-foundation/final/artifact-hashes.json", Status: "pass", Log: "logs/artifact-hashes-write.log"},
			{Name: "artifact-hashes-validate", Command: "go run ./tools/cmd/validate-artifact-hashes --manifest reports/actor-runtime-foundation/final/artifact-hashes.json", Status: "pass", Log: "logs/artifact-hashes-validate.log"},
			{Name: "actor-foundation-validator", Command: "go run ./tools/cmd/validate-actor-runtime-foundation --report-dir reports/actor-runtime-foundation/final --current-git-head " + testGitHead, Status: "pass", Log: "logs/actor-foundation-validator.log"},
		},
		Artifacts: []ArtifactReport{
			{Path: "actor-runtime-foundation-manifest.json", Kind: "foundation_manifest", Schema: SchemaV1},
			{Path: "parallel-production-linux-x64/parallel-production-linux-x64.json", Kind: "parallel_production_report", Schema: "tetra.parallel.production.v1"},
			{Path: "parallel-production-linux-x64/artifact-hashes.json", Kind: "parallel_hash_manifest", Schema: ArtifactHashSchema},
			{Path: "distributed-actors-linux-x64/distributed-actors-linux-x64.json", Kind: "distributed_actor_runtime_report", Schema: "tetra.actors.distributed-runtime.v1"},
			{Path: "distributed-actors-linux-x64/artifact-hashes.json", Kind: "distributed_hash_manifest", Schema: ArtifactHashSchema},
			{Path: "artifact-hashes.json", Kind: "foundation_hash_manifest", Schema: ArtifactHashSchema},
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
	writeFile(t, filepath.Join(dir, "actor-runtime-foundation-manifest.json"), string(validActorFoundationReport(t)))
	writeFile(t, filepath.Join(dir, "parallel-production-linux-x64", "parallel-production-linux-x64.json"), validParallelProductionSubreport)
	writeArtifactHashManifest(t, filepath.Join(dir, "parallel-production-linux-x64"), []testArtifact{
		{Path: "parallel-production-linux-x64.json", Schema: "tetra.parallel.production.v1"},
	})
	writeFile(t, filepath.Join(dir, "distributed-actors-linux-x64", "distributed-actors-linux-x64.json"), validDistributedActorSubreport)
	writeArtifactHashManifest(t, filepath.Join(dir, "distributed-actors-linux-x64"), []testArtifact{
		{Path: "distributed-actors-linux-x64.json", Schema: "tetra.actors.distributed-runtime.v1"},
	})
	writeFile(t, filepath.Join(dir, "logs", "focused-actor-tests.log"), "ok\n")
	writeArtifactHashManifest(t, dir, []testArtifact{
		{Path: "actor-runtime-foundation-manifest.json", Schema: SchemaV1},
		{Path: "distributed-actors-linux-x64/artifact-hashes.json", Schema: ArtifactHashSchema},
		{Path: "distributed-actors-linux-x64/distributed-actors-linux-x64.json", Schema: "tetra.actors.distributed-runtime.v1"},
		{Path: "logs/focused-actor-tests.log"},
		{Path: "parallel-production-linux-x64/artifact-hashes.json", Schema: ArtifactHashSchema},
		{Path: "parallel-production-linux-x64/parallel-production-linux-x64.json", Schema: "tetra.parallel.production.v1"},
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

const validParallelProductionSubreport = `{
  "schema": "tetra.parallel.production.v1",
  "status": "pass",
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "parallel-linux-x64",
  "source": "tools/cmd/parallel-production-smoke",
  "processes": [
    {"name":"tetra build","kind":"build","path":"go build ./cli/cmd/tetra","ran":true,"pass":true,"exit_code":0},
    {"name":"parallel smoke app","kind":"app","path":"parallel-smoke","ran":true,"pass":true,"exit_code":0},
    {"name":"parallel stress","kind":"stress","path":"parallel-stress","ran":true,"pass":true,"exit_code":0},
    {"name":"parallel scheduler prototype","kind":"benchmark","path":"compiler/internal/parallelrt","ran":true,"pass":true,"exit_code":0}
  ],
  "benchmarks": [
    {"name":"actor ping-pong benchmark prep","kind":"actor_benchmark_prep","metric":"messages_round_trip","unit":"prep_only","baseline_value":0,"measured_value":0,"improvement_ratio":0.0,"evidence":"compiler/actors_test.go::TestActorsPingPongBuildAndRun and examples/actors_pingpong.tetra define the local Linux-x64 actor ping-pong workload candidate","claim_tier":"tier0_local_smoke_only","claim":"Actor ping-pong benchmark prep row exists as Tier 0 local smoke only; no measured result is published and cross-runtime comparison is out of scope.","raw_output_artifacts":["reports/actor-runtime-foundation/P15/parallelrt-evidence.raw.json"],"ran":false,"pass":true},
    {"name":"actor fanout/fanin benchmark prep","kind":"actor_benchmark_prep","metric":"fanout_fanin_messages","unit":"prep_only","baseline_value":0,"measured_value":0,"improvement_ratio":0.0,"evidence":"compiler/internal/parallelrt two-core work stealing model checks actor fanout/fanin scheduling shape without publishing throughput","claim_tier":"tier0_local_smoke_only","claim":"Actor fanout/fanin benchmark prep row exists as Tier 0 local smoke only; it records local workload shape and leaves public benchmark publication out of scope.","raw_output_artifacts":["reports/actor-runtime-foundation/P15/parallelrt-evidence.raw.json"],"ran":false,"pass":true},
    {"name":"actor mailbox throughput benchmark prep","kind":"actor_benchmark_prep","metric":"mailbox_messages","unit":"prep_only","baseline_value":0,"measured_value":0,"improvement_ratio":0.0,"evidence":"compiler/internal/parallelrt TypedMailbox and parallel production actor mailbox cases define the local mailbox throughput workload candidate","claim_tier":"tier0_local_smoke_only","claim":"Actor mailbox throughput benchmark prep row exists as Tier 0 local smoke only; it publishes no measured result and no throughput guarantee.","raw_output_artifacts":["reports/actor-runtime-foundation/P15/parallelrt-evidence.raw.json"],"ran":false,"pass":true},
    {"name":"actor backpressure latency benchmark prep","kind":"actor_benchmark_prep","metric":"backpressure_wait","unit":"prep_only","baseline_value":0,"measured_value":0,"improvement_ratio":0.0,"evidence":"compiler/internal/parallelrt ErrMailboxFull and blocking_recv_yield metadata define the local backpressure latency diagnostic candidate","claim_tier":"tier0_local_smoke_only","claim":"Actor backpressure latency benchmark prep row exists as Tier 0 local smoke only; no real-world SLA or latency advantage is claimed.","raw_output_artifacts":["reports/actor-runtime-foundation/P15/parallelrt-evidence.raw.json"],"ran":false,"pass":true},
    {"name":"zero_copy_move local typed mailbox benchmark prep","kind":"actor_transfer_prep","metric":"owned_region_transfer","unit":"prep_only","baseline_value":0,"measured_value":0,"improvement_ratio":0.0,"evidence":"compiler/internal/parallelrt owned-region transfer report emits zero_copy_move for local typed mailbox metadata only","claim_tier":"tier0_local_smoke_only","claim":"zero_copy_move local typed mailbox benchmark prep row exists as Tier 0 local smoke only; it records local owned-region metadata and leaves distributed or network transfer behavior out of scope.","raw_output_artifacts":["reports/actor-runtime-foundation/P15/parallelrt-evidence.raw.json"],"ran":false,"pass":true}
  ],
  "contracts": [
    {"name":"production task scheduler","status":"pass","evidence":"scheduler fairness and lifecycle cases ran on linux-x64"},
    {"name":"join cancel deadline select group lifecycle","status":"pass","evidence":"join, cancel, deadline, select, and group lifecycle diagnostics are stable"},
    {"name":"actor mailbox backpressure and failure handling","status":"pass","evidence":"mailbox capacity, message pool exhaustion, and actor failure cases are covered"},
    {"name":"task actor thread boundary transfer rules","status":"pass","evidence":"ownership transfer diagnostics and actor/island boundary proof protect task, actor, and thread boundaries"},
    {"name":"race safety model","status":"pass","evidence":"shared mutable state crossing parallel boundaries is rejected conservatively with matrix evidence"},
    {"name":"safe unsafe forbidden parallelism boundary","status":"pass","evidence":"docs and diagnostics define safe, unsafe, and forbidden parallel behavior"}
  ],
  "cases": [
    {"name":"scheduler fairness","kind":"positive","ran":true,"pass":true},
    {"name":"task join lifecycle","kind":"positive","ran":true,"pass":true},
    {"name":"task cancellation","kind":"negative","ran":true,"pass":true,"expected_error":"cancelled"},
    {"name":"deadline timeout","kind":"negative","ran":true,"pass":true,"expected_error":"deadline"},
    {"name":"select readiness","kind":"positive","ran":true,"pass":true},
    {"name":"task group lifecycle","kind":"positive","ran":true,"pass":true},
    {"name":"task group cancel wakes deadline join","kind":"negative","ran":true,"pass":true,"expected_error":"cancelled before deadline"},
    {"name":"actor recv cancel wake","kind":"negative","ran":true,"pass":true,"expected_error":"actor recv cancel wake"},
    {"name":"nested cancellation propagation","kind":"positive","ran":true,"pass":true},
    {"name":"task actor mailbox handoff","kind":"positive","ran":true,"pass":true},
    {"name":"actor mailbox backpressure","kind":"negative","ran":true,"pass":true,"expected_error":"backpressure"},
    {"name":"message pool exhaustion","kind":"negative","ran":true,"pass":true,"expected_error":"message pool exhausted"},
    {"name":"invalid actor handle send","kind":"negative","ran":true,"pass":true,"expected_error":"invalid actor handle"},
    {"name":"done actor send","kind":"negative","ran":true,"pass":true,"expected_error":"done actor"},
    {"name":"actor failure handling","kind":"negative","ran":true,"pass":true,"expected_error":"actor failed"},
    {"name":"invalid handle diagnostics","kind":"negative","ran":true,"pass":true,"expected_error":"invalid handle"},
    {"name":"resource double join diagnostic","kind":"negative","ran":true,"pass":true,"expected_error":"joined"},
    {"name":"task group use-after-close diagnostic","kind":"negative","ran":true,"pass":true,"expected_error":"closed"},
    {"name":"ownership transfer across task boundary","kind":"negative","ran":true,"pass":true,"expected_error":"transfer"},
    {"name":"ownership transfer across actor boundary","kind":"negative","ran":true,"pass":true,"expected_error":"transfer"},
    {"name":"race-safety shared mutable rejection","kind":"negative","ran":true,"pass":true,"expected_error":"shared mutable"},
    {"name":"race-safety rejection matrix","kind":"positive","ran":true,"pass":true},
    {"name":"actor island boundary proof","kind":"positive","ran":true,"pass":true},
    {"name":"actor broker leak cleanup","kind":"positive","ran":true,"pass":true},
    {"name":"safe unsafe forbidden boundary coverage","kind":"positive","ran":true,"pass":true},
    {"name":"many tasks stress","kind":"stress","ran":true,"pass":true},
    {"name":"many actor messages stress","kind":"stress","ran":true,"pass":true},
    {"name":"cancellation storm","kind":"stress","ran":true,"pass":true},
    {"name":"timeouts stress","kind":"stress","ran":true,"pass":true}
  ],
  "diagnostics": [
    {"case":"task cancellation","code":"TASK_CANCELLED","severity":"error","category":"task","position":"runtime","expected_error":"cancelled"},
    {"case":"deadline timeout","code":"TASK_DEADLINE_TIMEOUT","severity":"error","category":"task","position":"runtime","expected_error":"deadline"},
    {"case":"task group cancel wakes deadline join","code":"TASK_GROUP_CANCEL_WAKE_JOIN","severity":"error","category":"task","position":"runtime","expected_error":"cancelled before deadline"},
    {"case":"actor recv cancel wake","code":"ACTOR_RECV_CANCEL_WAKE","severity":"error","category":"actor","position":"runtime","expected_error":"actor recv cancel wake"},
    {"case":"actor mailbox backpressure","code":"ACTOR_MAILBOX_BACKPRESSURE","severity":"error","category":"actor","position":"runtime","expected_error":"backpressure"},
    {"case":"message pool exhaustion","code":"ACTOR_MESSAGE_POOL_EXHAUSTED","severity":"error","category":"actor","position":"runtime","expected_error":"message pool exhausted"},
    {"case":"invalid actor handle send","code":"ACTOR_INVALID_HANDLE_SEND","severity":"error","category":"actor","position":"runtime","expected_error":"invalid actor handle"},
    {"case":"done actor send","code":"ACTOR_DONE_SEND","severity":"error","category":"actor","position":"runtime","expected_error":"done actor"},
    {"case":"actor failure handling","code":"ACTOR_MISSING_NODE_FAILURE","severity":"error","category":"actor","position":"runtime","expected_error":"actor failed"},
    {"case":"invalid handle diagnostics","code":"ACTOR_INVALID_HANDLE_DIAGNOSTIC","severity":"error","category":"actor","position":"cli-json","expected_error":"invalid handle"},
    {"case":"resource double join diagnostic","code":"RESOURCE_DOUBLE_JOIN","severity":"error","category":"resource","position":"cli-json","expected_error":"joined"},
    {"case":"task group use-after-close diagnostic","code":"TASK_GROUP_CLOSED","severity":"error","category":"task","position":"cli-json","expected_error":"closed"},
    {"case":"ownership transfer across task boundary","code":"OWNERSHIP_TASK_TRANSFER","severity":"error","category":"ownership","position":"compiler","expected_error":"transfer"},
    {"case":"ownership transfer across actor boundary","code":"OWNERSHIP_ACTOR_TRANSFER","severity":"error","category":"ownership","position":"compiler","expected_error":"transfer"},
    {"case":"race-safety shared mutable rejection","code":"RACE_SHARED_MUTABLE_REJECTED","severity":"error","category":"race-safety","position":"compiler","expected_error":"shared mutable"}
  ],
  "audit": [
    {"requirement":"production task scheduler","artifact":"compiler/task_runtime_test.go; compiler/internal/actorsrt/linux_x64.go","evidence":"scheduler fairness, many tasks stress, join, cancel, deadline, select, and task group lifecycle cases ran","result":"pass"},
    {"requirement":"join/cancel/deadline/select/group lifecycle","artifact":"compiler/task_runtime_test.go; examples/task_bounded_stress.tetra","evidence":"required lifecycle cases cover join, cancellation, deadline timeout, cancel-wakes-deadline-join, actor recv cancel wake, select readiness, task group lifecycle, and nested cancellation propagation","result":"pass"},
    {"requirement":"actor mailbox backpressure and failure handling","artifact":"compiler/actors_test.go; compiler/distributed_actor_runtime_test.go","evidence":"actor mailbox backpressure, checked message pool exhaustion, invalid actor handle send, done actor send, task actor mailbox handoff, and actor failure handling cases are required","result":"pass"},
    {"requirement":"task/actor/thread-boundary transfer rules","artifact":"compiler/tests/ownership; cli/cmd/tetra/check_diagnostics_resource_actor_test.go","evidence":"task and actor ownership transfer, actor/island boundary proof, resource double join, and task group use-after-close diagnostics are required cases","result":"pass"},
    {"requirement":"race-safety model or conservative rejections","artifact":"compiler/tests/ownership; docs/spec/actors.md","evidence":"shared mutable race-safety rejection and race-safety rejection matrix evidence are required until a broader race-safe model is implemented","result":"pass"},
    {"requirement":"stress evidence for tasks, actor messages, cancellation storms, and timeouts","artifact":"tools/cmd/parallel-production-smoke","evidence":"many tasks stress, many actor messages stress, cancellation storm, timeouts stress, and actor broker leak cleanup cases are required","result":"pass"},
    {"requirement":"safe/unsafe/forbidden parallelism documentation","artifact":"docs/spec/actors.md; docs/user/async_actors_guide.md; docs/spec/runtime_abi.md; compiler/tests/semantics/async_test.go; compiler/tests/safety/effects_test.go","evidence":"documentation defines supported actor/task runtime, transfer boundaries, and unsupported guarantees; safe unsafe forbidden boundary coverage runs compiler tests for allowed immutable task targets, missing runtime/actors effects, unsafe-only operations, and forbidden mutable actor/task targets","result":"pass"},
    {"requirement":"stable parallel diagnostics","artifact":"compiler/task_runtime_test.go; compiler/actors_test.go; compiler/tests/ownership/actor_task_ownership_test.go; cli/cmd/tetra/check_diagnostics_resource_actor_test.go","evidence":"negative parallel cases require stable expected_error evidence for cancellation, deadline, backpressure, invalid handle, double join, use-after-close, transfer, and shared mutable rejection diagnostics","result":"pass"},
    {"requirement":"actor benchmark Tier 0/Tier 1 preparation","artifact":"compiler/internal/parallelrt; tools/cmd/parallel-production-smoke","evidence":"parallelrt evidence emits Tier 0 actor ping-pong, fanout/fanin, mailbox throughput, backpressure latency, and zero_copy_move local typed mailbox prep rows with raw artifact references and no performance claim","result":"pass"},
    {"requirement":"release-gate entrypoint","artifact":"scripts/release/post_v0_4/parallel-production-linux-x64-smoke.sh","evidence":"parallel production gate must run producer, validator, and artifact hash validation","result":"pass"}
  ]
}`

const validDistributedActorSubreport = `{
  "schema": "tetra.actors.distributed-runtime.v1",
  "status": "pass",
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "actornet",
  "transport": "loopback-tcp",
  "git_head": "e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "artifact_hashes": "artifact-hashes.json",
  "claims": ["linux-x64 loopback tcp distributed actor runtime evidence"],
  "nonclaims": ["no cluster membership", "no reconnect/retry production", "no non-linux distributed actor runtime support"],
  "broker": {"runtime":"actornet","transport":"loopback-tcp","listen_addr":"127.0.0.1:47777","accepted_connections":3,"routed_frames":5,"dropped_frames":1},
  "processes": [
    {"name":"broker","kind":"broker","path":"./tetra actor-net","ran":true,"pass":true,"exit_code":0},
    {"name":"node-a","kind":"node","path":"node-a","ran":true,"pass":true,"exit_code":0},
    {"name":"node-b","kind":"node","path":"node-b","ran":true,"pass":true,"exit_code":0}
  ],
  "frame_counts": {"hello":2,"hello_ack":2,"spawn_req":1,"spawn_ack":1,"send_i32":1,"send_msg":1,"send_typed":1,"node_down":1},
  "frame_order": ["hello","hello_ack","spawn_req","spawn_ack","send_i32","send_msg","send_typed","node_down"],
  "cases": [
    {"name":"cross-node i32 send/receive","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":2},
    {"name":"cross-node tagged send/receive","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":2},
    {"name":"cross-node typed send/receive","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":2},
    {"name":"missing-node failure/status","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":1},
    {"name":"task cancel/join compatibility","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":1}
  ]
}`
