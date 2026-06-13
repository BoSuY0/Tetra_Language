package parallelprod

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateReportAcceptsLinuxX64ParallelProductionEvidence(t *testing.T) {
	raw := []byte(validParallelProductionReport())
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsPaperOnlyParallelEvidence(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.parallel.production.v1",
  "status": "pass",
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "parallel-linux-x64",
  "source": "docs-only-placeholder.md",
  "processes": [],
  "contracts": [],
  "cases": [],
  "audit": []
}`)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected paper-only parallel evidence to fail")
	}
	for _, want := range []string{"placeholder", "process", "contract", "case", "completion audit"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsMissingRequiredLifecycleCases(t *testing.T) {
	raw := strings.Replace(validParallelProductionReport(), `    {"name":"task cancellation","kind":"negative","ran":true,"pass":true,"expected_error":"cancelled"},
    {"name":"deadline timeout","kind":"negative","ran":true,"pass":true,"expected_error":"deadline"},
    {"name":"select readiness","kind":"positive","ran":true,"pass":true},
    {"name":"task group lifecycle","kind":"positive","ran":true,"pass":true},
`, "", 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing lifecycle cases to fail")
	}
	for _, want := range []string{"task cancellation", "deadline timeout", "select readiness", "task group lifecycle"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsMissingStableParallelDiagnosticsAudit(t *testing.T) {
	raw := strings.Replace(validParallelProductionReport(), `    {"requirement":"stable parallel diagnostics","artifact":"compiler/task_runtime_test.go; compiler/actors_test.go; compiler/tests/ownership/actor_task_ownership_test.go; cli/cmd/tetra/check_diagnostics_resource_actor_test.go","evidence":"negative parallel cases require stable expected_error evidence for cancellation, deadline, backpressure, invalid handle, double join, use-after-close, transfer, and shared mutable rejection diagnostics","result":"pass"},
`, "", 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing stable parallel diagnostics audit to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "stable parallel diagnostics") {
		t.Fatalf("error missing stable parallel diagnostics:\n%v", err)
	}
}

func TestValidateReportRejectsNegativeCasesMissingMachineReadableDiagnostics(t *testing.T) {
	var report Report
	if err := json.Unmarshal([]byte(validParallelProductionReport()), &report); err != nil {
		t.Fatalf("fixture unmarshal failed: %v", err)
	}
	report.Diagnostics = nil
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("fixture marshal failed: %v", err)
	}
	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected negative cases without machine-readable diagnostics to fail")
	}
	for _, want := range []string{"task cancellation", "diagnostic", "code", "severity", "position"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsGenericMachineReadableDiagnosticCode(t *testing.T) {
	var report Report
	if err := json.Unmarshal([]byte(validParallelProductionReport()), &report); err != nil {
		t.Fatalf("fixture unmarshal failed: %v", err)
	}
	report.Diagnostics[0].Code = "GENERIC_BACKEND_ERROR"
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("fixture marshal failed: %v", err)
	}
	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected generic diagnostic code to fail")
	}
	for _, want := range []string{"task cancellation", "diagnostic", "code", "stable"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsMissingActorBenchmarkPrepRows(t *testing.T) {
	raw := strings.Replace(validParallelProductionReport(), `  "benchmarks": [
    {"name":"actor ping-pong benchmark prep","kind":"actor_benchmark_prep","metric":"messages_round_trip","unit":"prep_only","baseline_value":0,"measured_value":0,"improvement_ratio":0.0,"evidence":"compiler/actors_test.go::TestActorsPingPongBuildAndRun and examples/actors_pingpong.tetra define the local Linux-x64 actor ping-pong workload candidate","claim_tier":"tier0_local_smoke_only","claim":"Actor ping-pong benchmark prep row exists as Tier 0 local smoke only; no measured result is published and cross-runtime comparison is out of scope.","raw_output_artifacts":["reports/actor-runtime-foundation/P15/parallelrt-evidence.raw.json"],"ran":false,"pass":true},
    {"name":"actor fanout/fanin benchmark prep","kind":"actor_benchmark_prep","metric":"fanout_fanin_messages","unit":"prep_only","baseline_value":0,"measured_value":0,"improvement_ratio":0.0,"evidence":"compiler/internal/parallelrt two-core work stealing model checks actor fanout/fanin scheduling shape without publishing throughput","claim_tier":"tier0_local_smoke_only","claim":"Actor fanout/fanin benchmark prep row exists as Tier 0 local smoke only; it records local workload shape and leaves public benchmark publication out of scope.","raw_output_artifacts":["reports/actor-runtime-foundation/P15/parallelrt-evidence.raw.json"],"ran":false,"pass":true},
    {"name":"actor mailbox throughput benchmark prep","kind":"actor_benchmark_prep","metric":"mailbox_messages","unit":"prep_only","baseline_value":0,"measured_value":0,"improvement_ratio":0.0,"evidence":"compiler/internal/parallelrt TypedMailbox and parallel production actor mailbox cases define the local mailbox throughput workload candidate","claim_tier":"tier0_local_smoke_only","claim":"Actor mailbox throughput benchmark prep row exists as Tier 0 local smoke only; it publishes no measured result and no throughput guarantee.","raw_output_artifacts":["reports/actor-runtime-foundation/P15/parallelrt-evidence.raw.json"],"ran":false,"pass":true},
    {"name":"actor backpressure latency benchmark prep","kind":"actor_benchmark_prep","metric":"backpressure_wait","unit":"prep_only","baseline_value":0,"measured_value":0,"improvement_ratio":0.0,"evidence":"compiler/internal/parallelrt ErrMailboxFull and blocking_recv_yield metadata define the local backpressure latency diagnostic candidate","claim_tier":"tier0_local_smoke_only","claim":"Actor backpressure latency benchmark prep row exists as Tier 0 local smoke only; no real-world SLA or latency advantage is claimed.","raw_output_artifacts":["reports/actor-runtime-foundation/P15/parallelrt-evidence.raw.json"],"ran":false,"pass":true},
    {"name":"zero_copy_move local typed mailbox benchmark prep","kind":"actor_transfer_prep","metric":"owned_region_transfer","unit":"prep_only","baseline_value":0,"measured_value":0,"improvement_ratio":0.0,"evidence":"compiler/internal/parallelrt owned-region transfer report emits zero_copy_move for local typed mailbox metadata only","claim_tier":"tier0_local_smoke_only","claim":"zero_copy_move local typed mailbox benchmark prep row exists as Tier 0 local smoke only; it records local owned-region metadata and leaves distributed or network transfer behavior out of scope.","raw_output_artifacts":["reports/actor-runtime-foundation/P15/parallelrt-evidence.raw.json"],"ran":false,"pass":true}
  ],
`, "", 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing actor benchmark prep rows to fail")
	}
	for _, want := range []string{"actor ping-pong benchmark prep", "zero_copy_move local typed mailbox benchmark prep"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsSchedulerPrototypeProductionPromotion(t *testing.T) {
	var report map[string]any
	if err := json.Unmarshal([]byte(validParallelProductionReport()), &report); err != nil {
		t.Fatal(err)
	}
	benchmarks := report["benchmarks"].([]any)
	benchmarks = append(benchmarks, map[string]any{
		"name":                 "per-core scheduler prototype benchmark",
		"kind":                 "scheduler",
		"metric":               "scheduler_fairness",
		"unit":                 "local_model",
		"baseline_value":       float64(1),
		"measured_value":       float64(1),
		"improvement_ratio":    float64(0),
		"evidence":             "per-core scheduler prototype row from compiler/internal/parallelrt",
		"claim_tier":           "tier1_local_benchmark_evidence",
		"claim":                "Per-core scheduler prototype benchmark proves production runtime scheduler readiness.",
		"raw_output_artifacts": []any{"reports/actor-final-production/parallel-production-linux-x64/parallelrt-evidence.raw.json"},
		"ran":                  true,
		"pass":                 true,
	})
	report["benchmarks"] = benchmarks
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected scheduler prototype production promotion to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "scheduler prototype") {
		t.Fatalf("error = %v, want scheduler prototype rejection", err)
	}
}

func TestValidateReportRejectsMissingSafeUnsafeForbiddenBoundaryCoverageCase(t *testing.T) {
	raw := strings.Replace(validParallelProductionReport(), `    {"name":"safe unsafe forbidden boundary coverage","kind":"positive","ran":true,"pass":true},
`, "", 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing safe unsafe forbidden boundary coverage case to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "safe unsafe forbidden boundary coverage") {
		t.Fatalf("error missing safe unsafe forbidden boundary coverage:\n%v", err)
	}
}

func TestValidateReportRejectsMissingParallelEdgeCases(t *testing.T) {
	for _, tc := range []struct {
		name string
		row  string
	}{
		{name: "task group cancel wakes deadline join", row: `    {"name":"task group cancel wakes deadline join","kind":"negative","ran":true,"pass":true,"expected_error":"cancelled before deadline"},
`},
		{name: "actor recv cancel wake", row: `    {"name":"actor recv cancel wake","kind":"negative","ran":true,"pass":true,"expected_error":"actor recv cancel wake"},
`},
		{name: "nested cancellation propagation", row: `    {"name":"nested cancellation propagation","kind":"positive","ran":true,"pass":true},
`},
		{name: "task actor mailbox handoff", row: `    {"name":"task actor mailbox handoff","kind":"positive","ran":true,"pass":true},
`},
		{name: "message pool exhaustion", row: `    {"name":"message pool exhaustion","kind":"negative","ran":true,"pass":true,"expected_error":"message pool exhausted"},
`},
		{name: "invalid actor handle send", row: `    {"name":"invalid actor handle send","kind":"negative","ran":true,"pass":true,"expected_error":"invalid actor handle"},
`},
		{name: "done actor send", row: `    {"name":"done actor send","kind":"negative","ran":true,"pass":true,"expected_error":"done actor"},
`},
		{name: "race-safety rejection matrix", row: `    {"name":"race-safety rejection matrix","kind":"positive","ran":true,"pass":true},
`},
		{name: "actor island boundary proof", row: `    {"name":"actor island boundary proof","kind":"positive","ran":true,"pass":true},
`},
		{name: "actor broker leak cleanup", row: `    {"name":"actor broker leak cleanup","kind":"positive","ran":true,"pass":true},
`},
		{name: "resource double join diagnostic", row: `    {"name":"resource double join diagnostic","kind":"negative","ran":true,"pass":true,"expected_error":"joined"},
`},
		{name: "task group use-after-close diagnostic", row: `    {"name":"task group use-after-close diagnostic","kind":"negative","ran":true,"pass":true,"expected_error":"closed"},
`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := strings.Replace(validParallelProductionReport(), tc.row, "", 1)
			err := ValidateReport([]byte(raw))
			if err == nil {
				t.Fatalf("expected missing %s case to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.name) {
				t.Fatalf("error = %v, want %s rejection", err, tc.name)
			}
		})
	}
}

func TestValidateReportRejectsMissingCompletionAudit(t *testing.T) {
	raw := strings.Replace(validParallelProductionReport(), `,
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
  ]`, "", 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing completion audit to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "completion audit") {
		t.Fatalf("error missing completion audit:\n%v", err)
	}
}

func validParallelProductionReport() string {
	return `{
  "schema": "tetra.parallel.production.v1",
  "status": "pass",
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "parallel-linux-x64",
  "source": "tools/cmd/parallel-production-smoke",
  "processes": [
    {"name":"tetra build","kind":"build","path":"/tmp/tetra","ran":true,"pass":true,"exit_code":0},
    {"name":"parallel smoke app","kind":"app","path":"/tmp/parallel-smoke","ran":true,"pass":true,"exit_code":0},
    {"name":"parallel stress","kind":"stress","path":"/tmp/parallel-stress","ran":true,"pass":true,"exit_code":0},
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
}
