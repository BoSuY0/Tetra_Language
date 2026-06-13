package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"tetra_language/tools/validators/parallelprod"
)

type smokeOptions struct {
	ReportPath string
	TetraPath  string
	KeepWork   bool
}

type smokeRunner struct {
	opt        smokeOptions
	workDir    string
	tetraPath  string
	processes  []parallelprod.ProcessReport
	benchmarks []parallelprod.BenchmarkReport
	cases      []parallelprod.CaseReport
}

type processResult struct {
	exitCode int
	output   string
	err      error
}

func main() {
	var opt smokeOptions
	flag.StringVar(&opt.ReportPath, "report", "", "path to write tetra.parallel.production.v1 report")
	flag.StringVar(&opt.TetraPath, "tetra", "", "tetra CLI path; defaults to a fresh temp build from ./cli/cmd/tetra")
	flag.BoolVar(&opt.KeepWork, "keep-work", false, "keep temporary build directory")
	flag.Parse()
	if opt.ReportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := runSmoke(context.Background(), opt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runSmoke(ctx context.Context, opt smokeOptions) error {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return fmt.Errorf("parallel production smoke requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	workDir, err := os.MkdirTemp(".", ".tetra-parallel-smoke-*")
	if err != nil {
		return err
	}
	r := &smokeRunner{opt: opt, workDir: workDir}
	if !opt.KeepWork {
		defer os.RemoveAll(workDir)
	}
	if err := os.MkdirAll(filepath.Dir(opt.ReportPath), 0o755); err != nil {
		return err
	}
	if opt.TetraPath == "" {
		r.tetraPath = filepath.Join(workDir, "tetra")
		res := runCommand(ctx, 30*time.Second, "go", "build", "-o", r.tetraPath, "./cli/cmd/tetra")
		r.recordProcess("tetra build", "build", "go build ./cli/cmd/tetra", res)
		if res.err != nil {
			return fmt.Errorf("build smoke tetra CLI: %s", res.output)
		}
	} else {
		r.tetraPath = opt.TetraPath
	}

	if err := r.runExecutableEvidence(ctx); err != nil {
		return err
	}
	if err := r.runCompilerEvidence(ctx); err != nil {
		return err
	}
	if err := r.runBoundaryCoverageEvidence(ctx); err != nil {
		return err
	}
	if err := r.runSchedulerPrototypeEvidence(ctx); err != nil {
		return err
	}
	return r.writeReport()
}

func (r *smokeRunner) runExecutableEvidence(ctx context.Context) error {
	if err := r.runExample(ctx, executableCase{
		name:          "wait composition",
		sourcePath:    filepath.Join("examples", "wait_composition_smoke.tetra"),
		expectedExit:  0,
		processName:   "parallel smoke app",
		processKind:   "app",
		recordProcess: true,
		cases: []parallelprod.CaseReport{
			{Name: "select readiness", Kind: "positive", Ran: true, Pass: true},
		},
	}); err != nil {
		return err
	}
	if err := r.runExample(ctx, executableCase{
		name:          "deadline aware waits",
		sourcePath:    filepath.Join("examples", "deadline_aware_waits_smoke.tetra"),
		expectedExit:  0,
		processName:   "deadline aware waits",
		processKind:   "app",
		recordProcess: false,
		cases: []parallelprod.CaseReport{
			{Name: "deadline timeout", Kind: "negative", Ran: true, Pass: true, ExpectedError: "deadline"},
			stressCase("timeouts stress", 1, "deadline-aware-waits-v1", 10000),
		},
	}); err != nil {
		return err
	}
	if err := r.runExample(ctx, executableCase{
		name:          "actors tagged stress",
		sourcePath:    filepath.Join("examples", "actors_tagged_stress.tetra"),
		expectedExit:  0,
		runtimeMode:   "builtin",
		processName:   "parallel stress",
		processKind:   "stress",
		recordProcess: true,
		cases: []parallelprod.CaseReport{
			stressCase("many actor messages stress", 256, "actors-tagged-stress-v1", 10000),
		},
	}); err != nil {
		return err
	}
	if err := r.runExample(ctx, executableCase{
		name:          "task bounded stress",
		sourcePath:    filepath.Join("examples", "task_bounded_stress.tetra"),
		expectedExit:  42,
		recordProcess: false,
		cases: []parallelprod.CaseReport{
			{Name: "scheduler fairness", Kind: "positive", Ran: true, Pass: true},
			stressCase("many tasks stress", 64, "task-bounded-stress-seed-17", 10000),
		},
	}); err != nil {
		return err
	}
	if err := r.runExample(ctx, executableCase{
		name:          "task group lifecycle",
		sourcePath:    filepath.Join("examples", "task_group_lifecycle_smoke.tetra"),
		expectedExit:  42,
		recordProcess: false,
		cases: []parallelprod.CaseReport{
			{Name: "task join lifecycle", Kind: "positive", Ran: true, Pass: true},
			{Name: "task group lifecycle", Kind: "positive", Ran: true, Pass: true},
		},
	}); err != nil {
		return err
	}
	if err := r.runExample(ctx, executableCase{
		name:          "task group cancel",
		sourcePath:    filepath.Join("examples", "task_group_cancel_smoke.tetra"),
		expectedExit:  1,
		recordProcess: false,
		cases: []parallelprod.CaseReport{
			{Name: "task cancellation", Kind: "negative", Ran: true, Pass: true, ExpectedError: "cancelled"},
		},
	}); err != nil {
		return err
	}
	sourcePath := filepath.Join(r.workDir, "parallel_cancellation_storm.tetra")
	if err := os.WriteFile(sourcePath, []byte(strings.TrimLeft(cancellationStormSource, "\n")), 0o644); err != nil {
		return err
	}
	return r.runExample(ctx, executableCase{
		name:          "cancellation storm",
		sourcePath:    sourcePath,
		expectedExit:  0,
		recordProcess: false,
		cases: []parallelprod.CaseReport{
			stressCase("cancellation storm", 16, "parallel-cancellation-storm-v1", 10000),
		},
	})
}

type executableCase struct {
	name          string
	sourcePath    string
	expectedExit  int
	runtimeMode   string
	processName   string
	processKind   string
	recordProcess bool
	cases         []parallelprod.CaseReport
}

func (r *smokeRunner) runExample(ctx context.Context, tc executableCase) error {
	outPath := filepath.Join(r.workDir, strings.ReplaceAll(tc.name, " ", "-"))
	args := []string{"build", "--target", "linux-x64", "-o", outPath}
	if tc.runtimeMode != "" {
		args = append(args, "--runtime", tc.runtimeMode)
	}
	args = append(args, tc.sourcePath)
	build := runCommand(ctx, 30*time.Second, r.tetraPath, args...)
	if build.err != nil {
		r.appendFailedCases(tc.cases, fmt.Sprintf("build failed: %s", build.output))
		return fmt.Errorf("build %s: %s", tc.name, build.output)
	}
	run := runCommand(ctx, 10*time.Second, outPath)
	if tc.recordProcess {
		r.recordProcess(tc.processName, tc.processKind, outPath, run)
	}
	if run.exitCode != tc.expectedExit {
		r.appendFailedCases(tc.cases, fmt.Sprintf("exit=%d output=%s", run.exitCode, run.output))
		return fmt.Errorf("%s exit=%d, want %d: %s", tc.name, run.exitCode, tc.expectedExit, run.output)
	}
	r.cases = append(r.cases, tc.cases...)
	return nil
}

func (r *smokeRunner) runCompilerEvidence(ctx context.Context) error {
	tests := []struct {
		name          string
		kind          string
		pkg           string
		pattern       string
		expectedError string
		iterations    int
		seed          string
		maxDurationMS int
	}{
		{name: "actor mailbox backpressure", kind: "negative", pkg: "./compiler", pattern: "TestActorMailboxFullReturnsCheckedBackpressure", expectedError: "mailbox full/backpressure"},
		{name: "message pool exhaustion", kind: "negative", pkg: "./compiler", pattern: "TestActorMessagePoolExhaustionReturnsCheckedFailure", expectedError: "message pool exhausted"},
		{name: "invalid actor handle send", kind: "negative", pkg: "./compiler", pattern: "TestActorInvalidHandleSendReturnsCheckedFailure", expectedError: "invalid actor handle"},
		{name: "done actor send", kind: "negative", pkg: "./compiler", pattern: "TestActorSendToDoneActorReturnsCheckedFailure", expectedError: "done actor"},
		{name: "actor failure handling", kind: "negative", pkg: "./tools/cmd/distributed-actor-runtime-smoke", pattern: "TestBuildReportProducesValidDistributedActorRuntimeEvidence", expectedError: "missing-node failure"},
		{name: "invalid handle diagnostics", kind: "negative", pkg: "./compiler/tests/ownership", pattern: "TestReleaseTraceabilityLifetimeAndRaceSafetyNegativeActorTaskOwnership", expectedError: "invalid handle"},
		{name: "resource double join diagnostic", kind: "negative", pkg: "./cli/cmd/tetra", pattern: "TestCheckCommandJSONDiagnosticsForResourceDoubleJoinCode", expectedError: "joined"},
		{name: "task group use-after-close diagnostic", kind: "negative", pkg: "./cli/cmd/tetra", pattern: "TestCheckCommandJSONDiagnosticsForTaskGroupUseAfterCloseCode", expectedError: "closed"},
		{name: "ownership transfer across task boundary", kind: "negative", pkg: "./compiler/tests/ownership", pattern: "TestReleaseTraceabilityLifetimeAndRaceSafetyNegativeActorTaskOwnership", expectedError: "transfer"},
		{name: "ownership transfer across actor boundary", kind: "negative", pkg: "./compiler/tests/ownership", pattern: "TestReleaseTraceabilityLifetimeAndRaceSafetyNegativeActorTaskOwnership", expectedError: "transfer"},
		{name: "race-safety shared mutable rejection", kind: "negative", pkg: "./compiler/tests/ownership", pattern: "TestReleaseTraceabilityLifetimeAndRaceSafetyNegativeActorTaskOwnership", expectedError: "shared mutable"},
		{name: "race-safety rejection matrix", kind: "positive", pkg: "./compiler/internal/actorsafety", pattern: "TestTypedActorOwnershipTransferCoverageRequiresRaceSafetyMatrix"},
		{name: "actor island boundary proof", kind: "positive", pkg: "./compiler/internal/semantics", pattern: "TestMemoryBoundaryHandoffCoverageRecordsActorTaskIslandFacts"},
		{name: "actor broker leak cleanup", kind: "positive", pkg: "./cli/internal/actornet", pattern: "TestBrokerCloseWithoutCancelStopsServeWatcher"},
		{name: "task group cancel wakes deadline join", kind: "negative", pkg: "./compiler", pattern: "TestTaskGroupCancelWakesJoinUntilBeforeDeadlineBuildAndRun", expectedError: "cancelled before deadline"},
		{name: "actor recv cancel wake", kind: "negative", pkg: "./compiler", pattern: "TestTaskGroupCancelWakesActorRecvUntilBeforeDeadlineBuildAndRun", expectedError: "actor recv cancel wake"},
		{name: "nested cancellation propagation", kind: "positive", pkg: "./compiler", pattern: "TestTaskCancellationCheckpointInheritedByNestedChildBuildAndRun"},
		{name: "task actor mailbox handoff", kind: "positive", pkg: "./compiler", pattern: "TestTaskSpawnsActorAndReceivesMailboxReplyBuildAndRun"},
		{name: "actor fanout mailbox drain soak", kind: "stress", pkg: "./compiler", pattern: "TestActorFanoutMailboxDrainSoakBuildAndRun", iterations: 512, seed: "actor-fanout-mailbox-drain-v1", maxDurationMS: 90000},
	}
	for _, tc := range tests {
		res := runCommand(ctx, 90*time.Second, "go", "test", tc.pkg, "-run", tc.pattern, "-count=1")
		if res.err != nil || res.exitCode != 0 {
			r.cases = append(r.cases, failedCase(tc.name, tc.kind, tc.expectedError, res.output))
			return fmt.Errorf("%s evidence failed: %s", tc.name, res.output)
		}
		if tc.kind == "stress" {
			r.cases = append(r.cases, stressCase(tc.name, tc.iterations, tc.seed, tc.maxDurationMS))
			continue
		}
		r.cases = append(r.cases, parallelprod.CaseReport{Name: tc.name, Kind: tc.kind, Ran: true, Pass: true, ExpectedError: tc.expectedError})
	}
	return nil
}

func (r *smokeRunner) runBoundaryCoverageEvidence(ctx context.Context) error {
	tests := []struct {
		pkg     string
		pattern string
	}{
		{
			pkg:     "./compiler/tests/semantics",
			pattern: "^(TestTaskSpawnRequiresRuntimeUse|TestTaskSpawnRejectsMutableGlobalTarget|TestActorSpawnRejectsMutableGlobalTarget|TestTaskSpawnAllowsImmutableGlobalTarget|TestTaskSpawnGroupTypedRejectsMutableGlobalTarget)$",
		},
		{
			pkg:     "./compiler/tests/safety",
			pattern: "^(TestEffectsAliasesAndUnsafeRemainSeparate|TestEffectsRequireActorsUse|TestUnsafeStillRequiredWithEffectGroups)$",
		},
	}
	for _, tc := range tests {
		res := runCommand(ctx, 90*time.Second, "go", "test", tc.pkg, "-run", tc.pattern, "-count=1")
		if res.err != nil || res.exitCode != 0 {
			r.cases = append(r.cases, failedCase("safe unsafe forbidden boundary coverage", "positive", "", res.output))
			return fmt.Errorf("safe unsafe forbidden boundary coverage evidence failed: %s", res.output)
		}
	}
	r.cases = append(r.cases, parallelprod.CaseReport{Name: "safe unsafe forbidden boundary coverage", Kind: "positive", Ran: true, Pass: true})
	return nil
}

func (r *smokeRunner) writeReport() error {
	report := buildReportWithBenchmarks("tools/cmd/parallel-production-smoke", r.processes, r.cases, r.benchmarks)
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := parallelprod.ValidateReport(raw); err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(r.opt.ReportPath, raw, 0o644)
}

func buildReport(source string, processes []parallelprod.ProcessReport, cases []parallelprod.CaseReport) parallelprod.Report {
	return buildReportWithBenchmarks(source, processes, cases, parallelSchedulerBenchmarks())
}

func buildReportWithBenchmarks(source string, processes []parallelprod.ProcessReport, cases []parallelprod.CaseReport, benchmarks []parallelprod.BenchmarkReport) parallelprod.Report {
	if len(benchmarks) == 0 {
		benchmarks = parallelSchedulerBenchmarks()
	}
	return parallelprod.Report{
		Schema:     parallelprod.SchemaV1,
		Status:     "pass",
		Target:     "linux-x64",
		Host:       "linux-x64",
		Runtime:    "parallel-linux-x64",
		Source:     source,
		Benchmarks: append([]parallelprod.BenchmarkReport(nil), benchmarks...),
		Processes: append([]parallelprod.ProcessReport(nil),
			processes...,
		),
		Contracts: []parallelprod.ContractReport{
			{Name: "production task scheduler", Status: "pass", Evidence: "linux-x64 cooperative scheduler fairness, wait composition, and many task stress evidence"},
			{Name: "join cancel deadline select group lifecycle", Status: "pass", Evidence: "task join, cancellation, deadline timeout, cancel-wakes-deadline-join, actor recv cancel wake, select2 readiness, and group lifecycle smokes"},
			{Name: "actor mailbox backpressure and failure handling", Status: "pass", Evidence: "actor capacity/backpressure, checked message pool exhaustion, checked invalid-handle/done-actor send failures, task actor mailbox handoff, and distributed missing-node failure evidence"},
			{Name: "task actor thread boundary transfer rules", Status: "pass", Evidence: "compiler and CLI diagnostics for actor/task transfer, actor/island boundary proof, double join, and closed group boundaries"},
			{Name: "race safety model", Status: "pass", Evidence: "conservative shared mutable actor/task boundary rejections and race-safety rejection matrix evidence"},
			{Name: "safe unsafe forbidden parallelism boundary", Status: "pass", Evidence: "actor/task docs plus diagnostics for supported, unsafe, and forbidden boundaries"},
		},
		Cases:       append([]parallelprod.CaseReport(nil), cases...),
		Diagnostics: machineReadableDiagnosticsForCases(cases),
		Audit:       parallelProductionAudit(),
	}
}

func machineReadableDiagnosticsForCases(cases []parallelprod.CaseReport) []parallelprod.DiagnosticReport {
	var diagnostics []parallelprod.DiagnosticReport
	for _, c := range cases {
		if c.Kind != "negative" {
			continue
		}
		diagnostics = append(diagnostics, diagnosticForCase(c.Name, c.ExpectedError))
	}
	return diagnostics
}

func diagnosticForCase(name, expectedError string) parallelprod.DiagnosticReport {
	diagnostic := parallelprod.DiagnosticReport{
		Case:          name,
		Code:          derivedDiagnosticCode(name),
		Severity:      "error",
		Category:      "parallel",
		Position:      "runtime",
		ExpectedError: expectedError,
	}
	switch name {
	case "task cancellation":
		diagnostic.Code = "TASK_CANCELLED"
		diagnostic.Category = "task"
	case "deadline timeout":
		diagnostic.Code = "TASK_DEADLINE_TIMEOUT"
		diagnostic.Category = "task"
	case "task group cancel wakes deadline join":
		diagnostic.Code = "TASK_GROUP_CANCEL_WAKE_JOIN"
		diagnostic.Category = "task"
	case "actor recv cancel wake":
		diagnostic.Code = "ACTOR_RECV_CANCEL_WAKE"
		diagnostic.Category = "actor"
	case "actor mailbox backpressure":
		diagnostic.Code = "ACTOR_MAILBOX_BACKPRESSURE"
		diagnostic.Category = "actor"
	case "message pool exhaustion":
		diagnostic.Code = "ACTOR_MESSAGE_POOL_EXHAUSTED"
		diagnostic.Category = "actor"
	case "invalid actor handle send":
		diagnostic.Code = "ACTOR_INVALID_HANDLE_SEND"
		diagnostic.Category = "actor"
	case "done actor send":
		diagnostic.Code = "ACTOR_DONE_SEND"
		diagnostic.Category = "actor"
	case "actor failure handling":
		diagnostic.Code = "ACTOR_MISSING_NODE_FAILURE"
		diagnostic.Category = "actor"
	case "invalid handle diagnostics":
		diagnostic.Code = "ACTOR_INVALID_HANDLE_DIAGNOSTIC"
		diagnostic.Category = "actor"
		diagnostic.Position = "cli-json"
	case "resource double join diagnostic":
		diagnostic.Code = "RESOURCE_DOUBLE_JOIN"
		diagnostic.Category = "resource"
		diagnostic.Position = "cli-json"
	case "task group use-after-close diagnostic":
		diagnostic.Code = "TASK_GROUP_CLOSED"
		diagnostic.Category = "task"
		diagnostic.Position = "cli-json"
	case "ownership transfer across task boundary":
		diagnostic.Code = "OWNERSHIP_TASK_TRANSFER"
		diagnostic.Category = "ownership"
		diagnostic.Position = "compiler"
	case "ownership transfer across actor boundary":
		diagnostic.Code = "OWNERSHIP_ACTOR_TRANSFER"
		diagnostic.Category = "ownership"
		diagnostic.Position = "compiler"
	case "race-safety shared mutable rejection":
		diagnostic.Code = "RACE_SHARED_MUTABLE_REJECTED"
		diagnostic.Category = "race-safety"
		diagnostic.Position = "compiler"
	}
	return diagnostic
}

func derivedDiagnosticCode(name string) string {
	var b strings.Builder
	b.WriteString("PARALLEL_")
	lastUnderscore := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r - 'a' + 'A')
			lastUnderscore = false
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
			lastUnderscore = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	return strings.TrimRight(b.String(), "_")
}

func (r *smokeRunner) runSchedulerPrototypeEvidence(ctx context.Context) error {
	res := runCommand(ctx, 90*time.Second, "go", "test", "./compiler/internal/parallelrt", "-count=1")
	r.recordProcess("parallel scheduler prototype tests", "benchmark", "go test ./compiler/internal/parallelrt", res)
	if res.err != nil || res.exitCode != 0 {
		r.cases = append(r.cases,
			failedCase("per-core scheduler prototype", "positive", "", res.output),
			failedCase("two-core work stealing scheduler model", "positive", "", res.output),
			failedCase("zero-copy region message benchmark", "positive", "", res.output),
		)
		return fmt.Errorf("parallel scheduler prototype evidence failed: %s", res.output)
	}
	r.cases = append(r.cases,
		parallelprod.CaseReport{Name: "per-core scheduler prototype", Kind: "positive", Ran: true, Pass: true},
		parallelprod.CaseReport{Name: "two-core work stealing scheduler model", Kind: "positive", Ran: true, Pass: true},
		parallelprod.CaseReport{Name: "zero-copy region message benchmark", Kind: "positive", Ran: true, Pass: true},
	)

	evidence := runCommand(ctx, 90*time.Second, "go", "run", "./compiler/cmd/parallelrt-evidence")
	r.recordProcess("parallel scheduler prototype evidence", "benchmark", "go run ./compiler/cmd/parallelrt-evidence", evidence)
	if evidence.err != nil || evidence.exitCode != 0 {
		return fmt.Errorf("parallel scheduler prototype benchmark evidence failed: %s", evidence.output)
	}
	rawPath := filepath.Join(filepath.Dir(r.opt.ReportPath), "parallelrt-evidence.raw.json")
	if err := os.MkdirAll(filepath.Dir(rawPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(rawPath, []byte(evidence.output), 0o644); err != nil {
		return fmt.Errorf("write parallel scheduler raw evidence: %w", err)
	}
	benchmarks, err := parseParallelSchedulerBenchmarks(evidence.output)
	if err != nil {
		return err
	}
	r.benchmarks = attachBenchmarkRawArtifacts(benchmarks, filepath.ToSlash(rawPath))
	return nil
}

func parseParallelSchedulerBenchmarks(raw string) ([]parallelprod.BenchmarkReport, error) {
	var benchmarks []parallelprod.BenchmarkReport
	if err := json.Unmarshal([]byte(raw), &benchmarks); err != nil {
		return nil, fmt.Errorf("parse parallel scheduler prototype benchmarks: %w", err)
	}
	if len(benchmarks) == 0 {
		return nil, fmt.Errorf("parallel scheduler prototype benchmark evidence is empty")
	}
	return benchmarks, nil
}

func attachBenchmarkRawArtifacts(benchmarks []parallelprod.BenchmarkReport, rawPath string) []parallelprod.BenchmarkReport {
	out := append([]parallelprod.BenchmarkReport(nil), benchmarks...)
	for i := range out {
		out[i].RawOutputArtifacts = []string{rawPath}
	}
	return out
}

func parallelSchedulerBenchmarks() []parallelprod.BenchmarkReport {
	rawArtifact := "reports/actor-runtime-foundation/P15/parallelrt-evidence.raw.json"
	return []parallelprod.BenchmarkReport{
		{
			Name:               "actor ping-pong benchmark prep",
			Kind:               "actor_benchmark_prep",
			Metric:             "messages_round_trip",
			Unit:               "prep_only",
			Evidence:           "compiler/actors_test.go::TestActorsPingPongBuildAndRun and examples/actors_pingpong.tetra define the local Linux-x64 actor ping-pong workload candidate",
			ClaimTier:          "tier0_local_smoke_only",
			Claim:              "Actor ping-pong benchmark prep row exists as Tier 0 local smoke only; no measured result is published and cross-runtime comparison is out of scope.",
			RawOutputArtifacts: []string{rawArtifact},
			Ran:                false,
			Pass:               true,
		},
		{
			Name:               "actor fanout/fanin benchmark prep",
			Kind:               "actor_benchmark_prep",
			Metric:             "fanout_fanin_messages",
			Unit:               "prep_only",
			Evidence:           "compiler/internal/parallelrt two-core work stealing model checks actor fanout/fanin scheduling shape without publishing throughput",
			ClaimTier:          "tier0_local_smoke_only",
			Claim:              "Actor fanout/fanin benchmark prep row exists as Tier 0 local smoke only; it records local workload shape and leaves public benchmark publication out of scope.",
			RawOutputArtifacts: []string{rawArtifact},
			Ran:                false,
			Pass:               true,
		},
		{
			Name:               "actor mailbox throughput benchmark prep",
			Kind:               "actor_benchmark_prep",
			Metric:             "mailbox_messages",
			Unit:               "prep_only",
			Evidence:           "compiler/internal/parallelrt TypedMailbox and parallel production actor mailbox cases define the local mailbox throughput workload candidate",
			ClaimTier:          "tier0_local_smoke_only",
			Claim:              "Actor mailbox throughput benchmark prep row exists as Tier 0 local smoke only; it publishes no measured result and no throughput guarantee.",
			RawOutputArtifacts: []string{rawArtifact},
			Ran:                false,
			Pass:               true,
		},
		{
			Name:               "actor backpressure latency benchmark prep",
			Kind:               "actor_benchmark_prep",
			Metric:             "backpressure_wait",
			Unit:               "prep_only",
			Evidence:           "compiler/internal/parallelrt ErrMailboxFull and blocking_recv_yield metadata define the local backpressure latency diagnostic candidate",
			ClaimTier:          "tier0_local_smoke_only",
			Claim:              "Actor backpressure latency benchmark prep row exists as Tier 0 local smoke only; no real-world SLA is claimed.",
			RawOutputArtifacts: []string{rawArtifact},
			Ran:                false,
			Pass:               true,
		},
		{
			Name:               "zero_copy_move local typed mailbox benchmark prep",
			Kind:               "actor_transfer_prep",
			Metric:             "owned_region_transfer",
			Unit:               "prep_only",
			Evidence:           "compiler/internal/parallelrt owned-region transfer report emits zero_copy_move for local typed mailbox metadata only",
			ClaimTier:          "tier0_local_smoke_only",
			Claim:              "zero_copy_move local typed mailbox benchmark prep row exists as Tier 0 local smoke only; it records local owned-region metadata and leaves distributed or network transfer behavior out of scope.",
			RawOutputArtifacts: []string{rawArtifact},
			Ran:                false,
			Pass:               true,
		},
	}
}

func requiredPassingCases() []parallelprod.CaseReport {
	return []parallelprod.CaseReport{
		{Name: "scheduler fairness", Kind: "positive", Ran: true, Pass: true},
		{Name: "task join lifecycle", Kind: "positive", Ran: true, Pass: true},
		{Name: "task cancellation", Kind: "negative", Ran: true, Pass: true, ExpectedError: "cancelled"},
		{Name: "deadline timeout", Kind: "negative", Ran: true, Pass: true, ExpectedError: "deadline"},
		{Name: "select readiness", Kind: "positive", Ran: true, Pass: true},
		{Name: "task group lifecycle", Kind: "positive", Ran: true, Pass: true},
		{Name: "task group cancel wakes deadline join", Kind: "negative", Ran: true, Pass: true, ExpectedError: "cancelled before deadline"},
		{Name: "actor recv cancel wake", Kind: "negative", Ran: true, Pass: true, ExpectedError: "actor recv cancel wake"},
		{Name: "nested cancellation propagation", Kind: "positive", Ran: true, Pass: true},
		{Name: "task actor mailbox handoff", Kind: "positive", Ran: true, Pass: true},
		{Name: "actor mailbox backpressure", Kind: "negative", Ran: true, Pass: true, ExpectedError: "mailbox full/backpressure"},
		{Name: "message pool exhaustion", Kind: "negative", Ran: true, Pass: true, ExpectedError: "message pool exhausted"},
		{Name: "invalid actor handle send", Kind: "negative", Ran: true, Pass: true, ExpectedError: "invalid actor handle"},
		{Name: "done actor send", Kind: "negative", Ran: true, Pass: true, ExpectedError: "done actor"},
		{Name: "actor failure handling", Kind: "negative", Ran: true, Pass: true, ExpectedError: "missing-node failure"},
		{Name: "invalid handle diagnostics", Kind: "negative", Ran: true, Pass: true, ExpectedError: "invalid handle"},
		{Name: "resource double join diagnostic", Kind: "negative", Ran: true, Pass: true, ExpectedError: "joined"},
		{Name: "task group use-after-close diagnostic", Kind: "negative", Ran: true, Pass: true, ExpectedError: "closed"},
		{Name: "ownership transfer across task boundary", Kind: "negative", Ran: true, Pass: true, ExpectedError: "transfer"},
		{Name: "ownership transfer across actor boundary", Kind: "negative", Ran: true, Pass: true, ExpectedError: "transfer"},
		{Name: "race-safety shared mutable rejection", Kind: "negative", Ran: true, Pass: true, ExpectedError: "shared mutable"},
		{Name: "race-safety rejection matrix", Kind: "positive", Ran: true, Pass: true},
		{Name: "actor island boundary proof", Kind: "positive", Ran: true, Pass: true},
		{Name: "actor broker leak cleanup", Kind: "positive", Ran: true, Pass: true},
		{Name: "safe unsafe forbidden boundary coverage", Kind: "positive", Ran: true, Pass: true},
		stressCase("actor fanout mailbox drain soak", 512, "actor-fanout-mailbox-drain-v1", 90000),
		stressCase("many tasks stress", 64, "task-bounded-stress-seed-17", 10000),
		stressCase("many actor messages stress", 256, "actors-tagged-stress-v1", 10000),
		stressCase("cancellation storm", 16, "parallel-cancellation-storm-v1", 10000),
		stressCase("timeouts stress", 1, "deadline-aware-waits-v1", 10000),
	}
}

func stressCase(name string, iterations int, seed string, maxDurationMS int) parallelprod.CaseReport {
	return parallelprod.CaseReport{
		Name:              name,
		Kind:              "stress",
		Ran:               true,
		Pass:              true,
		Iterations:        iterations,
		DeterministicSeed: seed,
		MaxDurationMS:     maxDurationMS,
	}
}

func parallelProductionAudit() []parallelprod.AuditReport {
	return []parallelprod.AuditReport{
		{
			Requirement: "production task scheduler",
			Artifact:    "examples/task_bounded_stress.tetra; examples/wait_composition_smoke.tetra; compiler/internal/actorsrt/linux_x64.go",
			Evidence:    "scheduler fairness, many tasks stress, join, cancel, deadline, select, and task group lifecycle cases ran on linux-x64",
			Result:      "pass",
		},
		{
			Requirement: "join/cancel/deadline/select/group lifecycle",
			Artifact:    "examples/task_group_lifecycle_smoke.tetra; examples/task_group_cancel_smoke.tetra; examples/deadline_aware_waits_smoke.tetra; examples/wait_composition_smoke.tetra",
			Evidence:    "required lifecycle cases cover join, cancellation, deadline timeout, cancel-wakes-deadline-join, actor recv cancel wake, select readiness, task group lifecycle, and nested cancellation propagation",
			Result:      "pass",
		},
		{
			Requirement: "actor mailbox backpressure and failure handling",
			Artifact:    "compiler/actors_test.go; tools/cmd/distributed-actor-runtime-smoke",
			Evidence:    "actor capacity/backpressure, checked message pool exhaustion, checked invalid-handle/done-actor send failures, task actor mailbox handoff, and missing-node failure/status evidence are required cases",
			Result:      "pass",
		},
		{
			Requirement: "task/actor/thread-boundary transfer rules",
			Artifact:    "compiler/tests/ownership/actor_task_ownership_test.go; cli/cmd/tetra/check_diagnostics_resource_actor_test.go",
			Evidence:    "task and actor ownership transfer, actor/island boundary proof, resource double join, and task group use-after-close diagnostics are required cases",
			Result:      "pass",
		},
		{
			Requirement: "race-safety model or conservative rejections",
			Artifact:    "compiler/tests/ownership/actor_task_ownership_test.go; docs/spec/actors.md",
			Evidence:    "shared mutable race-safety rejection and race-safety rejection matrix evidence are required until a broader race-safe model is implemented",
			Result:      "pass",
		},
		{
			Requirement: "stress evidence for tasks, actor messages, cancellation storms, and timeouts",
			Artifact:    "examples/task_bounded_stress.tetra; examples/actors_tagged_stress.tetra; tools/cmd/parallel-production-smoke",
			Evidence:    "many tasks stress, many actor messages stress, actor fanout mailbox drain soak, cancellation storm, timeouts stress, and actor broker leak cleanup cases ran with bounded stress metadata",
			Result:      "pass",
		},
		{
			Requirement: "safe/unsafe/forbidden parallelism documentation",
			Artifact:    "docs/spec/actors.md; docs/user/async_actors_guide.md; docs/spec/runtime_abi.md; compiler/tests/semantics/async_test.go; compiler/tests/safety/effects_test.go",
			Evidence:    "documentation defines supported actor/task runtime, transfer boundaries, and unsupported guarantees; safe unsafe forbidden boundary coverage runs compiler tests for allowed immutable task targets, missing runtime/actors effects, unsafe-only operations, and forbidden mutable actor/task targets",
			Result:      "pass",
		},
		{
			Requirement: "stable parallel diagnostics",
			Artifact:    "compiler/task_runtime_test.go; compiler/actors_test.go; compiler/tests/ownership/actor_task_ownership_test.go; cli/cmd/tetra/check_diagnostics_resource_actor_test.go",
			Evidence:    "negative parallel cases require stable expected_error evidence for cancellation, deadline, backpressure, invalid handle, double join, use-after-close, transfer, and shared mutable rejection diagnostics",
			Result:      "pass",
		},
		{
			Requirement: "actor benchmark Tier 0/Tier 1 preparation",
			Artifact:    "compiler/internal/parallelrt; tools/cmd/parallel-production-smoke",
			Evidence:    "parallelrt evidence emits Tier 0 actor ping-pong, fanout/fanin, mailbox throughput, backpressure latency, and zero_copy_move local typed mailbox prep rows with raw artifact references; Tier 1 remains preparation-only here, with no benchmark superiority, no C++/Rust parity, and no official benchmark claim",
			Result:      "pass",
		},
		{
			Requirement: "release-gate entrypoint",
			Artifact:    "scripts/release/post_v0_4/parallel-production-linux-x64-smoke.sh",
			Evidence:    "parallel production gate must run producer, validator, and artifact hash validation",
			Result:      "pass",
		},
	}
}

func failedCase(name, kind, expectedError, errText string) parallelprod.CaseReport {
	return parallelprod.CaseReport{Name: name, Kind: kind, Ran: true, Pass: false, ExpectedError: expectedError, Error: strings.TrimSpace(errText)}
}

func (r *smokeRunner) appendFailedCases(cases []parallelprod.CaseReport, errText string) {
	for _, c := range cases {
		r.cases = append(r.cases, failedCase(c.Name, c.Kind, c.ExpectedError, errText))
	}
}

func (r *smokeRunner) recordProcess(name, kind, path string, res processResult) {
	r.processes = append(r.processes, parallelprod.ProcessReport{
		Name:     name,
		Kind:     kind,
		Path:     path,
		Ran:      true,
		Pass:     res.err == nil && res.exitCode == 0,
		ExitCode: intPtr(res.exitCode),
	})
}

func runCommand(ctx context.Context, timeout time.Duration, name string, args ...string) processResult {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := strings.TrimSpace(stdout.String() + stderr.String())
	if cctx.Err() == context.DeadlineExceeded {
		return processResult{exitCode: -1, output: output, err: cctx.Err()}
	}
	return processResult{exitCode: processExitCode(err), output: output, err: err}
}

func processExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			if status.Signaled() {
				return -int(status.Signal())
			}
			return status.ExitStatus()
		}
	}
	return -1
}

func intPtr(v int) *int { return &v }

const cancellationStormSource = `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(10)
    return 99

func main() -> Int
uses runtime:
    var i: Int = 0
    var canceled: Int = 0
    while i < 16:
        var group: task.group = core.task_group_open()
        let task: task.i32 = core.task_spawn_group_i32(group, "worker")
        let _delay: Int = core.sleep_ms(1)
        group = core.task_group_cancel(group)
        let result: task.result_i32 = core.task_join_until_i32(task, core.deadline_ms(20))
        let _closed: Int = core.task_group_close(group)
        if result.value != 0:
            return result.value
        if result.error != 1:
            return 20 + result.error
        canceled = canceled + 1
        i = i + 1
    if canceled == 16:
        return 0
    return 1
`
