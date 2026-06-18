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
	raw := []byte(strings.Join([]string{
		"{",
		"  \"schema\": \"tetra.parallel.production.v1\",",
		"  \"status\": \"pass\",",
		"  \"target\": \"linux-x64\",",
		"  \"host\": \"linux-x64\",",
		"  \"runtime\": \"parallel-linux-x64\",",
		"  \"source\": \"docs-only-placeholder.md\",",
		"  \"processes\": [],",
		"  \"contracts\": [],",
		"  \"cases\": [],",
		"  \"audit\": []",
		"}",
	}, "\n"))
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
	raw := strings.Replace(
		validParallelProductionReport(),
		("    {\"name\":\"task cancellation\",\"kind\":\"negative\",\"ran\":" +
			"true,\"pass\":true,\"expected_error\":\"cancelled\"},\n    {\"name\":" +
			"\"deadline timeout\",\"kind\":\"negative\",\"ran\":true,\"pass\":true," +
			"\"expected_error\":\"deadline\"},\n    {\"name\":\"select " +
			"readiness\",\"kind\":\"positive\",\"ran\":true,\"pass\":true},\n    " +
			"{\"name\":\"task group lifecycle\",\"kind\":\"positive\",\"ran\":true," +
			"\"pass\":true},\n"),
		"",
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing lifecycle cases to fail")
	}
	for _, want := range []string{
		"task cancellation",
		"deadline timeout",
		"select readiness",
		"task group lifecycle",
	} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsMissingStableParallelDiagnosticsAudit(t *testing.T) {
	raw := strings.Replace(
		validParallelProductionReport(),
		("    {\"requirement\":\"stable parallel diagnostics\",\"artifact\":" +
			"\"compiler/compiler_suite_test.go; " +
			"compiler/compiler_suite_test.go; " +
			"compiler/tests/ownership/actor_task/actor_task_ownership_tes" +
			"t.go; cli/cmd/tetra/tetra_suite_test.go\",\"evidence\":" +
			"\"negative parallel cases require stable expected_error " +
			"evidence for cancellation, deadline, backpressure, invalid " +
			"handle, double join, use-after-close, transfer, and shared " +
			"mutable rejection diagnostics\",\"result\":\"pass\"},\n"),
		"",
		1,
	)
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
	var report Report
	if err := json.Unmarshal([]byte(validParallelProductionReport()), &report); err != nil {
		t.Fatalf("fixture unmarshal failed: %v", err)
	}
	report.Benchmarks = nil
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("fixture marshal failed: %v", err)
	}
	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing actor benchmark prep rows to fail")
	}
	for _, want := range []string{
		"actor ping-pong benchmark prep",
		"zero_copy_move local typed mailbox benchmark prep",
	} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsMissingActorMemoryDomainEvidence(t *testing.T) {
	var report Report
	if err := json.Unmarshal([]byte(validParallelProductionReport()), &report); err != nil {
		t.Fatalf("fixture unmarshal failed: %v", err)
	}
	report.ActorMemoryDomains = nil
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("fixture marshal failed: %v", err)
	}
	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing actor memory domain evidence to fail")
	}
	for _, want := range []string{"actor_memory_domains", "required"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsActorMemoryDomainOverclaims(t *testing.T) {
	var report Report
	if err := json.Unmarshal([]byte(validParallelProductionReport()), &report); err != nil {
		t.Fatalf("fixture unmarshal failed: %v", err)
	}
	report.ActorMemoryDomains[0].ProductionRuntimeClaimed = true
	report.ActorMemoryDomains[1].DistributedZeroCopyClaimed = true
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("fixture marshal failed: %v", err)
	}
	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected actor memory domain overclaims to fail")
	}
	for _, want := range []string{"production actor runtime", "distributed actor zero-copy"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsActorMemoryDomainByteInconsistency(t *testing.T) {
	var report Report
	if err := json.Unmarshal([]byte(validParallelProductionReport()), &report); err != nil {
		t.Fatalf("fixture unmarshal failed: %v", err)
	}
	report.ActorMemoryDomains[1].Domain.CurrentBytes = 999
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("fixture marshal failed: %v", err)
	}
	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected actor memory domain byte inconsistency to fail")
	}
	for _, want := range []string{"current_bytes", "queued_bytes"} {
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
		"name":              "per-core scheduler prototype benchmark",
		"kind":              "scheduler",
		"metric":            "scheduler_fairness",
		"unit":              "local_model",
		"baseline_value":    float64(1),
		"measured_value":    float64(1),
		"improvement_ratio": float64(0),
		"evidence":          "per-core scheduler prototype row from compiler/internal/parallelrt",
		"claim_tier":        "tier1_local_benchmark_evidence",
		"claim": ("Per-core scheduler prototype benchmark proves production " +
			"runtime scheduler readiness."),
		"raw_output_artifacts": []any{
			"reports/actor-final-production/parallel-production-linux-x64/parallelrt-evidence.raw.json",
		},
		"ran":  true,
		"pass": true,
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

func TestValidateReportRejectsZeroCopyMoveDistributedSpellingVariants(t *testing.T) {
	for _, claim := range []string{
		"zero_copy_move local typed mailbox benchmark prep proves distributed zero copy actor transfer.",
		"zero_copy_move local typed mailbox benchmark prep proves cross-node zero-copy actor transfer.",
		"zero_copy_move local typed mailbox benchmark prep proves actor transfer across nodes.",
	} {
		t.Run(claim, func(t *testing.T) {
			var report Report
			if err := json.Unmarshal([]byte(validParallelProductionReport()), &report); err != nil {
				t.Fatalf("fixture unmarshal failed: %v", err)
			}
			found := false
			for i := range report.Benchmarks {
				if report.Benchmarks[i].Name == "zero_copy_move local typed mailbox benchmark prep" {
					report.Benchmarks[i].Claim = claim
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("fixture missing zero_copy_move benchmark prep row")
			}
			raw, err := json.Marshal(report)
			if err != nil {
				t.Fatalf("fixture marshal failed: %v", err)
			}
			err = ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected zero_copy_move distributed spelling variant to fail")
			}
			if !strings.Contains(strings.ToLower(err.Error()), "zero") {
				t.Fatalf("error = %v, want zero-copy rejection", err)
			}
		})
	}
}

func TestValidateReportRejectsActorBenchmarkSuperiorityClaimWithoutReproducibleArtifacts(
	t *testing.T,
) {
	var report Report
	if err := json.Unmarshal([]byte(validParallelProductionReport()), &report); err != nil {
		t.Fatalf("fixture unmarshal failed: %v", err)
	}
	report.Benchmarks[0].ClaimTier = "tier2_reproducible_cross_machine"
	report.Benchmarks[0].Ran = true
	report.Benchmarks[0].BaselineValue = 100
	report.Benchmarks[0].MeasuredValue = 90
	report.Benchmarks[0].Claim = ("Actor benchmark proves Tetra actors are faster than " +
		"Rust/C++ actor runtimes.")
	report.Benchmarks[0].Evidence = ("Tier 2 cross-machine actor throughput result without " +
		"reproduction artifacts")
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("fixture marshal failed: %v", err)
	}
	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf(
			"expected actor benchmark superiority claim without reproducible artifacts to fail",
		)
	}
	for _, want := range []string{"faster than rust/c++", "tier 2", "reproduction_artifacts"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsBenchmarkPromotionBeyondTierOneWithoutEnvironmentMetadata(
	t *testing.T,
) {
	var report Report
	if err := json.Unmarshal([]byte(validParallelProductionReport()), &report); err != nil {
		t.Fatalf("fixture unmarshal failed: %v", err)
	}
	report.Benchmarks[1].ClaimTier = "tier3_independent_reproduction"
	report.Benchmarks[1].Ran = true
	report.Benchmarks[1].BaselineValue = 100
	report.Benchmarks[1].MeasuredValue = 100
	report.Benchmarks[1].Claim = ("Actor fanout benchmark prep is an independently reproduced " +
		"actor benchmark result.")
	report.Benchmarks[1].Evidence = ("Tier 3 independent actor benchmark result with no " +
		"environment metadata")
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("fixture marshal failed: %v", err)
	}
	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected Tier 3 benchmark promotion without environment metadata to fail")
	}
	for _, want := range []string{"tier 3", "environment metadata", "reproduction_artifacts"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsMissingActorBenchmarkNonClaimAudit(t *testing.T) {
	var report Report
	if err := json.Unmarshal([]byte(validParallelProductionReport()), &report); err != nil {
		t.Fatalf("fixture unmarshal failed: %v", err)
	}
	found := false
	for i := range report.Audit {
		if report.Audit[i].Requirement == "actor benchmark Tier 0/Tier 1 preparation" {
			report.Audit[i].Evidence = ("parallelrt evidence emits Tier 0 actor benchmark prep rows " +
				"with raw artifact references")
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("fixture missing actor benchmark audit row")
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("fixture marshal failed: %v", err)
	}
	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing actor benchmark nonclaim audit to fail")
	}
	for _, want := range []string{"benchmark superiority", "parity", "official benchmark"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsMissingSafeUnsafeForbiddenBoundaryCoverageCase(t *testing.T) {
	var report Report
	if err := json.Unmarshal([]byte(validParallelProductionReport()), &report); err != nil {
		t.Fatalf("fixture unmarshal failed: %v", err)
	}
	var kept []CaseReport
	for _, c := range report.Cases {
		if c.Name != "safe unsafe forbidden boundary coverage" {
			kept = append(kept, c)
		}
	}
	if len(kept) == len(report.Cases) {
		t.Fatalf("fixture missing safe unsafe forbidden boundary coverage case")
	}
	report.Cases = kept
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("fixture marshal failed: %v", err)
	}
	err = ValidateReport(raw)
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
		{name: "task group cancel wakes deadline join", row: (("    {\"name\":\"task group cancel " +
			"wakes deadline join\",\"kind\":") +
			"\"negative\",\"ran\":true,\"pass\":true,\"expected_error\":" +
			"\"cancelled before deadline\"},\n")},
		{name: "actor recv cancel wake", row: (("    {\"name\":\"actor recv cancel wake\"," +
			"\"kind\":\"negative\",") +
			"\"ran\":true,\"pass\":true,\"expected_error\":\"actor recv cancel " +
			"wake\"},\n")},
		{name: "nested cancellation propagation", row: (("    {\"name\":\"nested cancellation " +
			"propagation\",\"kind\":") +
			"\"positive\",\"ran\":true,\"pass\":true},\n")},
		{name: "task actor mailbox handoff", row: (("    {\"name\":\"task actor mailbox handoff\"," +
			"\"kind\":\"positive\",") +
			"\"ran\":true,\"pass\":true},\n")},
		{name: "message pool exhaustion", row: (("    {\"name\":\"message pool exhaustion\"," +
			"\"kind\":\"negative\",") +
			"\"ran\":true,\"pass\":true,\"expected_error\":\"message pool " +
			"exhausted\"},\n")},
		{name: "invalid actor handle send", row: (("    {\"name\":\"invalid actor handle send\"," +
			"\"kind\":\"negative\",") +
			"\"ran\":true,\"pass\":true,\"expected_error\":\"invalid actor " +
			"handle\"},\n")},
		{name: "done actor send", row: (("    {\"name\":\"done actor send\",\"kind\":\"negative\"," +
			"\"ran\":true,") +
			"\"pass\":true,\"expected_error\":\"done actor\"},\n")},
		{name: "race-safety rejection matrix", row: (("    {\"name\":\"race-safety rejection " +
			"matrix\",\"kind\":") +
			"\"positive\",\"ran\":true,\"pass\":true},\n")},
		{name: "actor island boundary proof", row: (("    {\"name\":\"actor island boundary proof\"," +
			"\"kind\":\"positive\",") +
			"\"ran\":true,\"pass\":true},\n")},
		{name: "actor broker leak cleanup", row: (("    {\"name\":\"actor broker leak cleanup\"," +
			"\"kind\":\"positive\",") +
			"\"ran\":true,\"pass\":true},\n")},
		{name: "actor fanout mailbox drain soak", row: (("    {\"name\":\"actor fanout mailbox " +
			"drain soak\",\"kind\":") +
			"\"stress\",\"ran\":true,\"pass\":true,\"iterations\":512," +
			"\"deterministic_seed\":\"actor-fanout-mailbox-drain-v1\"," +
			"\"max_duration_ms\":90000},\n")},
		{name: "resource double join diagnostic", row: (("    {\"name\":\"resource double join " +
			"diagnostic\",\"kind\":") +
			"\"negative\",\"ran\":true,\"pass\":true,\"expected_error\":" +
			"\"joined\"},\n")},
		{name: "task group use-after-close diagnostic", row: (("    {\"name\":\"task group use-" +
			"after-close diagnostic\",\"kind\":") +
			"\"negative\",\"ran\":true,\"pass\":true,\"expected_error\":" +
			"\"closed\"},\n")},
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

func TestValidateReportRejectsStressCasesMissingBoundedMetadata(t *testing.T) {
	var report map[string]any
	if err := json.Unmarshal([]byte(validParallelProductionReport()), &report); err != nil {
		t.Fatalf("fixture unmarshal failed: %v", err)
	}
	cases, ok := report["cases"].([]any)
	if !ok {
		t.Fatalf("fixture cases have type %T, want []any", report["cases"])
	}
	found := false
	for _, rawCase := range cases {
		c, ok := rawCase.(map[string]any)
		if !ok {
			t.Fatalf("fixture case has type %T, want map[string]any", rawCase)
		}
		if c["name"] == "many tasks stress" {
			delete(c, "iterations")
			delete(c, "deterministic_seed")
			delete(c, "max_duration_ms")
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("fixture missing many tasks stress case")
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("fixture marshal failed: %v", err)
	}
	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected stress case without bounded metadata to fail")
	}
	for _, want := range []string{
		"many tasks stress",
		"iterations",
		"deterministic_seed",
		"max_duration_ms",
	} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsMissingCompletionAudit(t *testing.T) {
	raw := strings.Replace(validParallelProductionReport(), `,
  "audit": [
    {"requirement":"production task scheduler","artifact":"compiler/compiler_suite_test.go; compiler/internal/actorsrt/actorsrt_core.go","evidence":"scheduler fairness, many tasks stress, join, cancel, deadline, select, and task group lifecycle cases ran","result":"pass"},
    {"requirement":"join/cancel/deadline/select/group lifecycle","artifact":"compiler/compiler_suite_test.go; examples/tasks/task_bounded_stress.tetra","evidence":"required lifecycle cases cover join, cancellation, deadline timeout, cancel-wakes-deadline-join, actor recv cancel wake, select readiness, task group lifecycle, and nested cancellation propagation","result":"pass"},
    {"requirement":"actor mailbox backpressure and failure handling","artifact":"compiler/compiler_suite_test.go; compiler/compiler_suite_test.go","evidence":"actor mailbox backpressure, checked message pool exhaustion, invalid actor handle send, done actor send, task actor mailbox handoff, and actor failure handling cases are required","result":"pass"},
    {"requirement":"task/actor/thread-boundary transfer rules","artifact":"compiler/tests/ownership; cli/cmd/tetra/tetra_suite_test.go","evidence":"task and actor ownership transfer, actor/island boundary proof, resource double join, and task group use-after-close diagnostics are required cases","result":"pass"},
    {"requirement":"race-safety model or conservative rejections","artifact":"compiler/tests/ownership; docs/spec/runtime/actors.md","evidence":"shared mutable race-safety rejection and race-safety rejection matrix evidence are required until a broader race-safe model is implemented","result":"pass"},
    {"requirement":"stress evidence for tasks, actor messages, cancellation storms, and timeouts","artifact":"tools/cmd/parallel-production-smoke","evidence":"many tasks stress, many actor messages stress, actor fanout mailbox drain soak, cancellation storm, timeouts stress, and actor broker leak cleanup cases are required with bounded metadata","result":"pass"},
    {"requirement":"safe/unsafe/forbidden parallelism documentation","artifact":"docs/spec/runtime/actors.md; docs/user/platform/async_actors_guide.md; docs/spec/runtime/runtime_abi.md; compiler/tests/semantics/semantics_async_ownership_test.go; compiler/tests/safety/effects/effects_test.go","evidence":"documentation defines supported actor/task runtime, transfer boundaries, and unsupported guarantees; safe unsafe forbidden boundary coverage runs compiler tests for allowed immutable task targets, missing runtime/actors effects, unsafe-only operations, and forbidden mutable actor/task targets","result":"pass"},
    {"requirement":"stable parallel diagnostics","artifact":"compiler/compiler_suite_test.go; compiler/compiler_suite_test.go; compiler/tests/ownership/actor_task/actor_task_ownership_test.go; cli/cmd/tetra/tetra_suite_test.go","evidence":"negative parallel cases require stable expected_error evidence for cancellation, deadline, backpressure, invalid handle, double join, use-after-close, transfer, and shared mutable rejection diagnostics","result":"pass"},
    {"requirement":"actor benchmark Tier 0/Tier 1 preparation","artifact":"compiler/internal/parallelrt; tools/cmd/parallel-production-smoke","evidence":"parallelrt evidence emits Tier 0 actor ping-pong, fanout/fanin, mailbox throughput, backpressure latency, and zero_copy_move local typed mailbox prep rows with raw artifact references; Tier 1 remains preparation-only here, with no benchmark superiority, no C++/Rust parity, and no official benchmark claim","result":"pass"},
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
	return strings.Join([]string{
		"",
		"{",
		"  \"schema\": \"tetra.parallel.production.v1\",",
		"  \"status\": \"pass\",",
		"  \"target\": \"linux-x64\",",
		"  \"host\": \"linux-x64\",",
		"  \"runtime\": \"parallel-linux-x64\",",
		"  \"source\": \"tools/cmd/parallel-production-smoke\",",
		"  \"processes\": [",
		("    {\"name\":\"tetra build\",\"kind\":\"build\",\"path\":\"/tmp/tetra\"," +
			"\"ran\":true,\"pass\":true,\"exit_code\":0},"),
		("    {\"name\":\"parallel smoke app\",\"kind\":\"app\",\"path\":\"/tmp/para" +
			"llel-smoke\",\"ran\":true,\"pass\":true,\"exit_code\":0},"),
		("    {\"name\":\"parallel stress\",\"kind\":\"stress\",\"path\":\"/tmp/para" +
			"llel-stress\",\"ran\":true,\"pass\":true,\"exit_code\":0},"),
		("    {\"name\":\"parallel scheduler prototype\",\"kind\":\"benchmark\",\"pa" +
			"th\":\"compiler/internal/parallelrt\",\"ran\":true,\"pass\":true,\"exit_co" +
			"de\":0}"),
		"  ],",
		"  \"benchmarks\": [",
		("    {\"name\":\"actor ping-pong benchmark prep\",\"kind\":\"actor_benchmar" +
			"k_prep\",\"metric\":\"messages_round_trip\",\"unit\":\"prep_only\",\"basel" +
			"ine_value\":0,\"measured_value\":0,\"improvement_ratio\":0.0,\"evidence\":" +
			"\"compiler/compiler_suite_test.go::TestActorsPingPongBuildAndRun and " +
			"examples/actors/actors_pingpong.tetra define the local Linux-x64 act" +
			"or ping-pong workload candidate\",\"claim_tier\":\"tier0_local_smoke_onl" +
			"y\",\"claim\":\"Actor ping-pong benchmark prep row exists as Tier 0 loca" +
			"l smoke only; no measured result is published and cross-runtime comp" +
			"arison is out of scope.\",\"raw_output_artifacts\":[\"reports/actor-runt" +
			"ime-foundation/P15/parallelrt-evidence.raw.json\"],\"ran\":false,\"pass\"" +
			":true},"),
		("    {\"name\":\"actor fanout/fanin benchmark prep\",\"kind\":\"actor_bench" +
			"mark_prep\",\"metric\":\"fanout_fanin_messages\",\"unit\":\"prep_only\",\"" +
			"baseline_value\":0,\"measured_value\":0,\"improvement_ratio\":0.0,\"eviden" +
			"ce\":\"compiler/internal/parallelrt two-core work stealing model check" +
			"s actor fanout/fanin scheduling shape without publishing throughput\"" +
			",\"claim_tier\":\"tier0_local_smoke_only\",\"claim\":\"Actor fanout/fanin " +
			"benchmark prep row exists as Tier 0 local smoke only; it records loc" +
			"al workload shape and leaves public benchmark publication out of sco" +
			"pe.\",\"raw_output_artifacts\":[\"reports/actor-runtime-foundation/P15/p" +
			"arallelrt-evidence.raw.json\"],\"ran\":false,\"pass\":true},"),
		("    {\"name\":\"actor mailbox throughput benchmark prep\",\"kind\":\"actor" +
			"_benchmark_prep\",\"metric\":\"mailbox_messages\",\"unit\":\"prep_only\"," +
			"\"baseline_value\":0,\"measured_value\":0,\"improvement_ratio\":0.0,\"evid" +
			"ence\":\"compiler/internal/parallelrt TypedMailbox and parallel produc" +
			"tion actor mailbox cases define the local mailbox throughput workloa" +
			"d candidate\",\"claim_tier\":\"tier0_local_smoke_only\",\"claim\":\"Actor " +
			"mailbox throughput benchmark prep row exists as Tier 0 local smoke o" +
			"nly; it publishes no measured result and no throughput guarantee.\",\"" +
			"raw_output_artifacts\":[\"reports/actor-runtime-foundation/P15/paralle" +
			"lrt-evidence.raw.json\"],\"ran\":false,\"pass\":true},"),
		("    {\"name\":\"actor backpressure latency benchmark prep\",\"kind\":\"act" +
			"or_benchmark_prep\",\"metric\":\"backpressure_wait\",\"unit\":\"prep_only" +
			"\",\"baseline_value\":0,\"measured_value\":0,\"improvement_ratio\":0.0,\"e" +
			"vidence\":\"compiler/internal/parallelrt ErrMailboxFull and blocking_r" +
			"ecv_yield metadata define the local backpressure latency diagnostic " +
			"candidate\",\"claim_tier\":\"tier0_local_smoke_only\",\"claim\":\"Actor ba" +
			"ckpressure latency benchmark prep row exists as Tier 0 local smoke o" +
			"nly; no real-world SLA is claimed.\",\"raw_output_artifacts\":[\"reports" +
			"/actor-runtime-foundation/P15/parallelrt-evidence.raw.json\"],\"ran\":f" +
			"alse,\"pass\":true},"),
		("    {\"name\":\"zero_copy_move local typed mailbox benchmark prep\",\"kin" +
			"d\":\"actor_transfer_prep\",\"metric\":\"owned_region_transfer\",\"unit\":" +
			"\"prep_only\",\"baseline_value\":0,\"measured_value\":0,\"improvement_rati" +
			"o\":0.0,\"evidence\":\"compiler/internal/parallelrt owned-region transfe" +
			"r report emits zero_copy_move for local typed mailbox metadata only\"" +
			",\"claim_tier\":\"tier0_local_smoke_only\",\"claim\":\"zero_copy_move loca" +
			"l typed mailbox benchmark prep row exists as Tier 0 local smoke only" +
			"; it records local owned-region metadata and leaves distributed or n" +
			"etwork transfer behavior out of scope.\",\"raw_output_artifacts\":[\"rep" +
			"orts/actor-runtime-foundation/P15/parallelrt-evidence.raw.json\"],\"ra" +
			"n\":false,\"pass\":true}"),
		"  ],",
		"  \"actor_memory_domains\": [",
		("    {\"schema_version\":\"tetra.actors.memory-domain.v1\",\"actor_id\":\"a" +
			"ctor-mailbox-copy\",\"evidence_class\":\"local_parallelrt_model\",\"eviden" +
			"ce_method\":\"parallelrt_typed_mailbox_memory_domain_v1\",\"runtime_meas" +
			"ured\":false,\"runtime_blocked_reason\":\"production actor runtime per-a" +
			"ctor byte sampler is not implemented; this is local parallelrt model" +
			" evidence\",\"domain\":{\"domain_id\":\"domain:actor:actor-mailbox-copy\"," +
			"\"kind\":\"actor\",\"owner_kind\":\"actor\",\"owner_id\":\"actor-mailbox-c" +
			"opy\",\"lifetime\":\"actor:actor-mailbox-copy\",\"budget_bytes\":256,\"req" +
			"uested_bytes\":48,\"reserved_bytes\":256,\"committed_bytes\":256,\"current" +
			"_bytes\":48,\"peak_bytes\":48,\"copy_count\":1,\"bytes_copied\":32},\"mail" +
			"box\":{\"capacity_messages\":4,\"queued_messages\":1,\"capacity_bytes\":25" +
			"6,\"queued_bytes\":48,\"peak_queued_bytes\":48,\"message_bytes\":16,\"back" +
			"pressure_mode\":\"blocking_recv_yield\"},\"message_pool\":{\"slab_bytes\":" +
			"64,\"live_bytes\":48,\"capacity_bytes\":256,\"message_slots_live\":1,\"mes" +
			"sage_slots_limit\":4},\"backpressure\":{\"mode\":\"blocking_recv_yield\"," +
			"\"status\":\"available\"},\"non_claims\":[\"full production actor runtime " +
			"is not claimed\",\"distributed actor zero-copy is not claimed\",\"actor " +
			"memory domain bytes are model/report evidence unless paired with run" +
			"time measurement\"],\"production_runtime_claimed\":false,\"distributed_z" +
			"ero_copy_claimed\":false},"),
		("    {\"schema_version\":\"tetra.actors.memory-domain.v1\",\"actor_id\":\"a" +
			"ctor-frame\",\"evidence_class\":\"local_parallelrt_model\",\"evidence_meth" +
			"od\":\"parallelrt_typed_mailbox_memory_domain_v1\",\"runtime_measured\":f" +
			"alse,\"runtime_blocked_reason\":\"production actor runtime per-actor by" +
			"te sampler is not implemented; this is local parallelrt model eviden" +
			"ce\",\"domain\":{\"domain_id\":\"domain:actor:actor-frame\",\"kind\":\"act" +
			"or\",\"owner_kind\":\"actor\",\"owner_id\":\"actor-frame\",\"lifetime\":\"" +
			"actor:actor-frame\",\"budget_bytes\":512,\"requested_bytes\":272,\"reserve" +
			"d_bytes\":512,\"committed_bytes\":512,\"current_bytes\":272,\"peak_bytes\"" +
			":272},\"mailbox\":{\"capacity_messages\":2,\"queued_messages\":1,\"capacit" +
			"y_bytes\":512,\"queued_bytes\":272,\"peak_queued_bytes\":272,\"message_byt" +
			"es\":16,\"backpressure_mode\":\"blocking_recv_yield\"},\"message_pool\":{" +
			"\"slab_bytes\":32,\"live_bytes\":272,\"capacity_bytes\":512,\"message_slot" +
			"s_live\":1,\"message_slots_limit\":2},\"owned_regions\":[{\"region_name\":" +
			"\"frame\",\"domain_id\":\"domain:actor:actor-frame\",\"owner_id\":\"actor-" +
			"frame\",\"bytes\":256}],\"backpressure\":{\"mode\":\"blocking_recv_yield\"" +
			",\"status\":\"byte_limit_reached\",\"reason\":\"mailbox byte capacity reac" +
			"hed\"},\"non_claims\":[\"full production actor runtime is not claimed\",\"" +
			"distributed actor zero-copy is not claimed\",\"actor memory domain byt" +
			"es are model/report evidence unless paired with runtime measurement\"" +
			"],\"production_runtime_claimed\":false,\"distributed_zero_copy_claimed\"" +
			":false}"),
		"  ],",
		"  \"contracts\": [",
		("    {\"name\":\"production task scheduler\",\"status\":\"pass\",\"evidence" +
			"\":\"scheduler fairness and lifecycle cases ran on linux-x64\"},"),
		("    {\"name\":\"join cancel deadline select group lifecycle\",\"status\":" +
			"\"pass\",\"evidence\":\"join, cancel, deadline, select, and group lifecyc" +
			"le diagnostics are stable\"},"),
		("    {\"name\":\"actor mailbox backpressure and failure handling\",\"statu" +
			"s\":\"pass\",\"evidence\":\"mailbox capacity, message pool exhaustion, and" +
			" actor failure cases are covered\"},"),
		("    {\"name\":\"task actor thread boundary transfer rules\",\"status\":\"p" +
			"ass\",\"evidence\":\"ownership transfer diagnostics and actor/island bou" +
			"ndary proof protect task, actor, and thread boundaries\"},"),
		("    {\"name\":\"race safety model\",\"status\":\"pass\",\"evidence\":\"sha" +
			"red mutable state crossing parallel boundaries is rejected conservat" +
			"ively with matrix evidence\"},"),
		("    {\"name\":\"safe unsafe forbidden parallelism boundary\",\"status\":\"" +
			"pass\",\"evidence\":\"docs and diagnostics define safe, unsafe, and forb" +
			"idden parallel behavior\"}"),
		"  ],",
		"  \"cases\": [",
		("    {\"name\":\"scheduler fairness\",\"kind\":\"positive\",\"ran\":true,\"" +
			"pass\":true},"),
		("    {\"name\":\"task join lifecycle\",\"kind\":\"positive\",\"ran\":true," +
			"\"pass\":true},"),
		("    {\"name\":\"task cancellation\",\"kind\":\"negative\",\"ran\":true,\"p" +
			"ass\":true,\"expected_error\":\"cancelled\"},"),
		("    {\"name\":\"deadline timeout\",\"kind\":\"negative\",\"ran\":true,\"pa" +
			"ss\":true,\"expected_error\":\"deadline\"},"),
		("    {\"name\":\"select readiness\",\"kind\":\"positive\",\"ran\":true,\"pa" +
			"ss\":true},"),
		("    {\"name\":\"task group lifecycle\",\"kind\":\"positive\",\"ran\":true," +
			"\"pass\":true},"),
		("    {\"name\":\"task group cancel wakes deadline join\",\"kind\":\"negativ" +
			"e\",\"ran\":true,\"pass\":true,\"expected_error\":\"cancelled before deadl" +
			"ine\"},"),
		("    {\"name\":\"actor recv cancel wake\",\"kind\":\"negative\",\"ran\":tru" +
			"e,\"pass\":true,\"expected_error\":\"actor recv cancel wake\"},"),
		("    {\"name\":\"nested cancellation propagation\",\"kind\":\"positive\",\"" +
			"ran\":true,\"pass\":true},"),
		("    {\"name\":\"task actor mailbox handoff\",\"kind\":\"positive\",\"ran\"" +
			":true,\"pass\":true},"),
		("    {\"name\":\"actor mailbox backpressure\",\"kind\":\"negative\",\"ran\"" +
			":true,\"pass\":true,\"expected_error\":\"backpressure\"},"),
		("    {\"name\":\"message pool exhaustion\",\"kind\":\"negative\",\"ran\":tr" +
			"ue,\"pass\":true,\"expected_error\":\"message pool exhausted\"},"),
		("    {\"name\":\"invalid actor handle send\",\"kind\":\"negative\",\"ran\":" +
			"true,\"pass\":true,\"expected_error\":\"invalid actor handle\"},"),
		("    {\"name\":\"done actor send\",\"kind\":\"negative\",\"ran\":true,\"pas" +
			"s\":true,\"expected_error\":\"done actor\"},"),
		("    {\"name\":\"actor failure handling\",\"kind\":\"negative\",\"ran\":tru" +
			"e,\"pass\":true,\"expected_error\":\"actor failed\"},"),
		("    {\"name\":\"invalid handle diagnostics\",\"kind\":\"negative\",\"ran\"" +
			":true,\"pass\":true,\"expected_error\":\"invalid handle\"},"),
		("    {\"name\":\"resource double join diagnostic\",\"kind\":\"negative\",\"" +
			"ran\":true,\"pass\":true,\"expected_error\":\"joined\"},"),
		("    {\"name\":\"task group use-after-close diagnostic\",\"kind\":\"negativ" +
			"e\",\"ran\":true,\"pass\":true,\"expected_error\":\"closed\"},"),
		("    {\"name\":\"ownership transfer across task boundary\",\"kind\":\"negat" +
			"ive\",\"ran\":true,\"pass\":true,\"expected_error\":\"transfer\"},"),
		("    {\"name\":\"ownership transfer across actor boundary\",\"kind\":\"nega" +
			"tive\",\"ran\":true,\"pass\":true,\"expected_error\":\"transfer\"},"),
		("    {\"name\":\"race-safety shared mutable rejection\",\"kind\":\"negative" +
			"\",\"ran\":true,\"pass\":true,\"expected_error\":\"shared mutable\"},"),
		("    {\"name\":\"race-safety rejection matrix\",\"kind\":\"positive\",\"ran" +
			"\":true,\"pass\":true},"),
		("    {\"name\":\"actor island boundary proof\",\"kind\":\"positive\",\"ran" +
			"\":true,\"pass\":true},"),
		("    {\"name\":\"actor broker leak cleanup\",\"kind\":\"positive\",\"ran\":" +
			"true,\"pass\":true},"),
		("    {\"name\":\"safe unsafe forbidden boundary coverage\",\"kind\":\"posit" +
			"ive\",\"ran\":true,\"pass\":true},"),
		("    {\"name\":\"actor fanout mailbox drain soak\",\"kind\":\"stress\",\"ra" +
			"n\":true,\"pass\":true,\"iterations\":512,\"deterministic_seed\":\"actor-f" +
			"anout-mailbox-drain-v1\",\"max_duration_ms\":90000},"),
		("    {\"name\":\"many tasks stress\",\"kind\":\"stress\",\"ran\":true,\"pas" +
			"s\":true,\"iterations\":64,\"deterministic_seed\":\"task-bounded-stress-se" +
			"ed-17\",\"max_duration_ms\":10000},"),
		("    {\"name\":\"many actor messages stress\",\"kind\":\"stress\",\"ran\":t" +
			"rue,\"pass\":true,\"iterations\":256,\"deterministic_seed\":\"actors-tagge" +
			"d-stress-v1\",\"max_duration_ms\":10000},"),
		("    {\"name\":\"cancellation storm\",\"kind\":\"stress\",\"ran\":true,\"pa" +
			"ss\":true,\"iterations\":16,\"deterministic_seed\":\"parallel-cancellation" +
			"-storm-v1\",\"max_duration_ms\":10000},"),
		("    {\"name\":\"timeouts stress\",\"kind\":\"stress\",\"ran\":true,\"pass" +
			"\":true,\"iterations\":1,\"deterministic_seed\":\"deadline-aware-waits-v1" +
			"\",\"max_duration_ms\":10000}"),
		"  ],",
		"  \"diagnostics\": [",
		("    {\"case\":\"task cancellation\",\"code\":\"TASK_CANCELLED\",\"severity" +
			"\":\"error\",\"category\":\"task\",\"position\":\"runtime\",\"expected_err" +
			"or\":\"cancelled\"},"),
		("    {\"case\":\"deadline timeout\",\"code\":\"TASK_DEADLINE_TIMEOUT\",\"se" +
			"verity\":\"error\",\"category\":\"task\",\"position\":\"runtime\",\"expect" +
			"ed_error\":\"deadline\"},"),
		("    {\"case\":\"task group cancel wakes deadline join\",\"code\":\"TASK_GR" +
			"OUP_CANCEL_WAKE_JOIN\",\"severity\":\"error\",\"category\":\"task\",\"posi" +
			"tion\":\"runtime\",\"expected_error\":\"cancelled before deadline\"},"),
		("    {\"case\":\"actor recv cancel wake\",\"code\":\"ACTOR_RECV_CANCEL_WAKE" +
			"\",\"severity\":\"error\",\"category\":\"actor\",\"position\":\"runtime\"," +
			"\"expected_error\":\"actor recv cancel wake\"},"),
		("    {\"case\":\"actor mailbox backpressure\",\"code\":\"ACTOR_MAILBOX_BACK" +
			"PRESSURE\",\"severity\":\"error\",\"category\":\"actor\",\"position\":\"ru" +
			"ntime\",\"expected_error\":\"backpressure\"},"),
		("    {\"case\":\"message pool exhaustion\",\"code\":\"ACTOR_MESSAGE_POOL_EX" +
			"HAUSTED\",\"severity\":\"error\",\"category\":\"actor\",\"position\":\"run" +
			"time\",\"expected_error\":\"message pool exhausted\"},"),
		("    {\"case\":\"invalid actor handle send\",\"code\":\"ACTOR_INVALID_HANDL" +
			"E_SEND\",\"severity\":\"error\",\"category\":\"actor\",\"position\":\"runt" +
			"ime\",\"expected_error\":\"invalid actor handle\"},"),
		("    {\"case\":\"done actor send\",\"code\":\"ACTOR_DONE_SEND\",\"severity" +
			"\":\"error\",\"category\":\"actor\",\"position\":\"runtime\",\"expected_er" +
			"ror\":\"done actor\"},"),
		("    {\"case\":\"actor failure handling\",\"code\":\"ACTOR_MISSING_NODE_FAI" +
			"LURE\",\"severity\":\"error\",\"category\":\"actor\",\"position\":\"runtim" +
			"e\",\"expected_error\":\"actor failed\"},"),
		("    {\"case\":\"invalid handle diagnostics\",\"code\":\"ACTOR_INVALID_HAND" +
			"LE_DIAGNOSTIC\",\"severity\":\"error\",\"category\":\"actor\",\"position\"" +
			":\"cli-json\",\"expected_error\":\"invalid handle\"},"),
		("    {\"case\":\"resource double join diagnostic\",\"code\":\"RESOURCE_DOUB" +
			"LE_JOIN\",\"severity\":\"error\",\"category\":\"resource\",\"position\":\"" +
			"cli-json\",\"expected_error\":\"joined\"},"),
		("    {\"case\":\"task group use-after-close diagnostic\",\"code\":\"TASK_GR" +
			"OUP_CLOSED\",\"severity\":\"error\",\"category\":\"task\",\"position\":\"c" +
			"li-json\",\"expected_error\":\"closed\"},"),
		("    {\"case\":\"ownership transfer across task boundary\",\"code\":\"OWNER" +
			"SHIP_TASK_TRANSFER\",\"severity\":\"error\",\"category\":\"ownership\",\"p" +
			"osition\":\"compiler\",\"expected_error\":\"transfer\"},"),
		("    {\"case\":\"ownership transfer across actor boundary\",\"code\":\"OWNE" +
			"RSHIP_ACTOR_TRANSFER\",\"severity\":\"error\",\"category\":\"ownership\"," +
			"\"position\":\"compiler\",\"expected_error\":\"transfer\"},"),
		("    {\"case\":\"race-safety shared mutable rejection\",\"code\":\"RACE_SHA" +
			"RED_MUTABLE_REJECTED\",\"severity\":\"error\",\"category\":\"race-safety\"" +
			",\"position\":\"compiler\",\"expected_error\":\"shared mutable\"}"),
		"  ],",
		"  \"audit\": [",
		("    {\"requirement\":\"production task scheduler\",\"artifact\":\"compiler" +
			"/compiler_suite_test.go; compiler/internal/actorsrt/actorsrt_core.go" +
			"\",\"evidence\":\"scheduler fairness, many tasks stress, join, cancel, d" +
			"eadline, select, and task group lifecycle cases ran\",\"result\":\"pass\"" +
			"},"),
		("    {\"requirement\":\"join/cancel/deadline/select/group lifecycle\",\"ar" +
			"tifact\":\"compiler/compiler_suite_test.go; examples/tasks/task_bounde" +
			"d_stress.tetra\",\"evidence\":\"required lifecycle cases cover join, can" +
			"cellation, deadline timeout, cancel-wakes-deadline-join, actor recv " +
			"cancel wake, select readiness, task group lifecycle, and nested canc" +
			"ellation propagation\",\"result\":\"pass\"},"),
		("    {\"requirement\":\"actor mailbox backpressure and failure handling\"" +
			",\"artifact\":\"compiler/compiler_suite_test.go; compiler/compiler_suit" +
			"e_test.go\",\"evidence\":\"actor mailbox backpressure, checked message p" +
			"ool exhaustion, invalid actor handle send, done actor send, task act" +
			"or mailbox handoff, and actor failure handling cases are required\",\"" +
			"result\":\"pass\"},"),
		("    {\"requirement\":\"task/actor/thread-boundary transfer rules\",\"arti" +
			"fact\":\"compiler/tests/ownership; cli/cmd/tetra/tetra_suite_test.go\"," +
			"\"evidence\":\"task and actor ownership transfer, actor/island boundary" +
			" proof, resource double join, and task group use-after-close diagnos" +
			"tics are required cases\",\"result\":\"pass\"},"),
		("    {\"requirement\":\"race-safety model or conservative rejections\",\"a" +
			"rtifact\":\"compiler/tests/ownership; docs/spec/runtime/actors.md\",\"ev" +
			"idence\":\"shared mutable race-safety rejection and race-safety reject" +
			"ion matrix evidence are required until a broader race-safe model is " +
			"implemented\",\"result\":\"pass\"},"),
		("    {\"requirement\":\"stress evidence for tasks, actor messages, cance" +
			"llation storms, and timeouts\",\"artifact\":\"tools/cmd/parallel-product" +
			"ion-smoke\",\"evidence\":\"many tasks stress, many actor messages stress" +
			", actor fanout mailbox drain soak, cancellation storm, timeouts stre" +
			"ss, and actor broker leak cleanup cases are required with bounded me" +
			"tadata\",\"result\":\"pass\"},"),
		("    {\"requirement\":\"safe/unsafe/forbidden parallelism documentation\"" +
			",\"artifact\":\"docs/spec/runtime/actors.md; docs/user/platform/async_a" +
			"ctors_guide.md; docs/spec/runtime/runtime_abi.md; compiler/tests/sem" +
			"antics/semantics_async_ownership_test.go; compiler/tests/safety/effe" +
			"cts/effects_test.go\",\"evidence\":\"documentation defines supported act" +
			"or/task runtime, transfer boundaries, and unsupported guarantees; sa" +
			"fe unsafe forbidden boundary coverage runs compiler tests for allowe" +
			"d immutable task targets, missing runtime/actors effects, unsafe-onl" +
			"y operations, and forbidden mutable actor/task targets\",\"result\":\"pa" +
			"ss\"},"),
		("    {\"requirement\":\"stable parallel diagnostics\",\"artifact\":\"compil" +
			"er/compiler_suite_test.go; compiler/compiler_suite_test.go; compiler" +
			"/tests/ownership/actor_task/actor_task_ownership_test.go; cli/cmd/te" +
			"tra/tetra_suite_test.go\",\"evidence\":\"negative parallel cases require" +
			" stable expected_error evidence for cancellation, deadline, backpres" +
			"sure, invalid handle, double join, use-after-close, transfer, and sh" +
			"ared mutable rejection diagnostics\",\"result\":\"pass\"},"),
		("    {\"requirement\":\"actor benchmark Tier 0/Tier 1 preparation\",\"arti" +
			"fact\":\"compiler/internal/parallelrt; tools/cmd/parallel-production-s" +
			"moke\",\"evidence\":\"parallelrt evidence emits Tier 0 actor ping-pong, " +
			"fanout/fanin, mailbox throughput, backpressure latency, and zero_cop" +
			"y_move local typed mailbox prep rows with raw artifact references; T" +
			"ier 1 remains preparation-only here, with no benchmark superiority, " +
			"no C++/Rust parity, and no official benchmark claim\",\"result\":\"pass\"" +
			"},"),
		("    {\"requirement\":\"release-gate entrypoint\",\"artifact\":\"scripts/re" +
			"lease/post_v0_4/parallel-production-linux-x64-smoke.sh\",\"evidence\":\"" +
			"parallel production gate must run producer, validator, and artifact " +
			"hash validation\",\"result\":\"pass\"}"),
		"  ]",
		"}",
		"",
	}, "\n")
}
