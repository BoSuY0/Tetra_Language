package parallelprod

import (
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
		{name: "nested cancellation propagation", row: `    {"name":"nested cancellation propagation","kind":"positive","ran":true,"pass":true},
`},
		{name: "task actor mailbox handoff", row: `    {"name":"task actor mailbox handoff","kind":"positive","ran":true,"pass":true},
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
    {"requirement":"join/cancel/deadline/select/group lifecycle","artifact":"compiler/task_runtime_test.go; examples/task_bounded_stress.tetra","evidence":"required lifecycle cases cover join, cancellation, deadline timeout, cancel-wakes-deadline-join, select readiness, task group lifecycle, and nested cancellation propagation","result":"pass"},
    {"requirement":"actor mailbox backpressure and failure handling","artifact":"compiler/actors_test.go; compiler/distributed_actor_runtime_test.go","evidence":"actor mailbox backpressure, task actor mailbox handoff, and actor failure handling cases are required","result":"pass"},
    {"requirement":"task/actor/thread-boundary transfer rules","artifact":"compiler/tests/ownership; cli/cmd/tetra/check_diagnostics_resource_actor_test.go","evidence":"task and actor ownership transfer, resource double join, and task group use-after-close diagnostics are required cases","result":"pass"},
    {"requirement":"race-safety model or conservative rejections","artifact":"compiler/tests/ownership; docs/spec/actors.md","evidence":"shared mutable race-safety rejection is required until a broader race-safe model is implemented","result":"pass"},
    {"requirement":"stress evidence for tasks, actor messages, cancellation storms, and timeouts","artifact":"tools/cmd/parallel-production-smoke","evidence":"many tasks stress, many actor messages stress, cancellation storm, and timeouts stress cases are required","result":"pass"},
    {"requirement":"safe/unsafe/forbidden parallelism documentation","artifact":"docs/spec/actors.md; docs/user/async_actors_guide.md; docs/spec/runtime_abi.md; compiler/tests/semantics/async_test.go; compiler/tests/safety/effects_test.go","evidence":"documentation defines supported actor/task runtime, transfer boundaries, and unsupported guarantees; safe unsafe forbidden boundary coverage runs compiler tests for allowed immutable task targets, missing runtime/actors effects, unsafe-only operations, and forbidden mutable actor/task targets","result":"pass"},
    {"requirement":"stable parallel diagnostics","artifact":"compiler/task_runtime_test.go; compiler/actors_test.go; compiler/tests/ownership/actor_task_ownership_test.go; cli/cmd/tetra/check_diagnostics_resource_actor_test.go","evidence":"negative parallel cases require stable expected_error evidence for cancellation, deadline, backpressure, invalid handle, double join, use-after-close, transfer, and shared mutable rejection diagnostics","result":"pass"},
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
    {"name":"parallel stress","kind":"stress","path":"/tmp/parallel-stress","ran":true,"pass":true,"exit_code":0}
  ],
  "contracts": [
    {"name":"production task scheduler","status":"pass","evidence":"scheduler fairness and lifecycle cases ran on linux-x64"},
    {"name":"join cancel deadline select group lifecycle","status":"pass","evidence":"join, cancel, deadline, select, and group lifecycle diagnostics are stable"},
    {"name":"actor mailbox backpressure and failure handling","status":"pass","evidence":"mailbox capacity and actor failure cases are covered"},
    {"name":"task actor thread boundary transfer rules","status":"pass","evidence":"ownership transfer diagnostics protect task, actor, and thread boundaries"},
    {"name":"race safety model","status":"pass","evidence":"shared mutable state crossing parallel boundaries is rejected conservatively"},
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
    {"name":"nested cancellation propagation","kind":"positive","ran":true,"pass":true},
    {"name":"task actor mailbox handoff","kind":"positive","ran":true,"pass":true},
    {"name":"actor mailbox backpressure","kind":"negative","ran":true,"pass":true,"expected_error":"backpressure"},
    {"name":"actor failure handling","kind":"negative","ran":true,"pass":true,"expected_error":"actor failed"},
    {"name":"invalid handle diagnostics","kind":"negative","ran":true,"pass":true,"expected_error":"invalid handle"},
    {"name":"resource double join diagnostic","kind":"negative","ran":true,"pass":true,"expected_error":"joined"},
    {"name":"task group use-after-close diagnostic","kind":"negative","ran":true,"pass":true,"expected_error":"closed"},
    {"name":"ownership transfer across task boundary","kind":"negative","ran":true,"pass":true,"expected_error":"transfer"},
    {"name":"ownership transfer across actor boundary","kind":"negative","ran":true,"pass":true,"expected_error":"transfer"},
    {"name":"race-safety shared mutable rejection","kind":"negative","ran":true,"pass":true,"expected_error":"shared mutable"},
    {"name":"safe unsafe forbidden boundary coverage","kind":"positive","ran":true,"pass":true},
    {"name":"many tasks stress","kind":"stress","ran":true,"pass":true},
    {"name":"many actor messages stress","kind":"stress","ran":true,"pass":true},
    {"name":"cancellation storm","kind":"stress","ran":true,"pass":true},
    {"name":"timeouts stress","kind":"stress","ran":true,"pass":true}
  ],
  "audit": [
    {"requirement":"production task scheduler","artifact":"compiler/task_runtime_test.go; compiler/internal/actorsrt/linux_x64.go","evidence":"scheduler fairness, many tasks stress, join, cancel, deadline, select, and task group lifecycle cases ran","result":"pass"},
    {"requirement":"join/cancel/deadline/select/group lifecycle","artifact":"compiler/task_runtime_test.go; examples/task_bounded_stress.tetra","evidence":"required lifecycle cases cover join, cancellation, deadline timeout, cancel-wakes-deadline-join, select readiness, task group lifecycle, and nested cancellation propagation","result":"pass"},
    {"requirement":"actor mailbox backpressure and failure handling","artifact":"compiler/actors_test.go; compiler/distributed_actor_runtime_test.go","evidence":"actor mailbox backpressure, task actor mailbox handoff, and actor failure handling cases are required","result":"pass"},
    {"requirement":"task/actor/thread-boundary transfer rules","artifact":"compiler/tests/ownership; cli/cmd/tetra/check_diagnostics_resource_actor_test.go","evidence":"task and actor ownership transfer, resource double join, and task group use-after-close diagnostics are required cases","result":"pass"},
    {"requirement":"race-safety model or conservative rejections","artifact":"compiler/tests/ownership; docs/spec/actors.md","evidence":"shared mutable race-safety rejection is required until a broader race-safe model is implemented","result":"pass"},
    {"requirement":"stress evidence for tasks, actor messages, cancellation storms, and timeouts","artifact":"tools/cmd/parallel-production-smoke","evidence":"many tasks stress, many actor messages stress, cancellation storm, and timeouts stress cases are required","result":"pass"},
    {"requirement":"safe/unsafe/forbidden parallelism documentation","artifact":"docs/spec/actors.md; docs/user/async_actors_guide.md; docs/spec/runtime_abi.md; compiler/tests/semantics/async_test.go; compiler/tests/safety/effects_test.go","evidence":"documentation defines supported actor/task runtime, transfer boundaries, and unsupported guarantees; safe unsafe forbidden boundary coverage runs compiler tests for allowed immutable task targets, missing runtime/actors effects, unsafe-only operations, and forbidden mutable actor/task targets","result":"pass"},
    {"requirement":"stable parallel diagnostics","artifact":"compiler/task_runtime_test.go; compiler/actors_test.go; compiler/tests/ownership/actor_task_ownership_test.go; cli/cmd/tetra/check_diagnostics_resource_actor_test.go","evidence":"negative parallel cases require stable expected_error evidence for cancellation, deadline, backpressure, invalid handle, double join, use-after-close, transfer, and shared mutable rejection diagnostics","result":"pass"},
    {"requirement":"release-gate entrypoint","artifact":"scripts/release/post_v0_4/parallel-production-linux-x64-smoke.sh","evidence":"parallel production gate must run producer, validator, and artifact hash validation","result":"pass"}
  ]
}`
}
