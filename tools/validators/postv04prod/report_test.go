package postv04prod

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/memoryprod"
	"tetra_language/tools/validators/nativeui"
	"tetra_language/tools/validators/parallelprod"
	"tetra_language/tools/validators/uiprod"
)

func TestBuildReportAcceptsCompletePostV04ProductionReportDir(t *testing.T) {
	dir := writeValidReportDir(t)
	report, err := BuildReport(dir)
	if err != nil {
		t.Fatalf("BuildReport failed: %v", err)
	}
	if err := ValidateReport(report); err != nil {
		t.Fatalf("ValidateReport failed: %v", err)
	}
	if report.Schema != SchemaV1 {
		t.Fatalf("schema = %q, want %q", report.Schema, SchemaV1)
	}
	if len(report.Layers) != 3 {
		t.Fatalf("layers = %d, want 3", len(report.Layers))
	}
	if len(report.Checklist) != len(RequiredChecklist()) {
		t.Fatalf("checklist = %d, want %d", len(report.Checklist), len(RequiredChecklist()))
	}
}

func TestValidateReportRejectsOutOfOrderLayers(t *testing.T) {
	dir := writeValidReportDir(t)
	report, err := BuildReport(dir)
	if err != nil {
		t.Fatalf("BuildReport failed: %v", err)
	}
	report.Layers = []LayerReport{report.Layers[1], report.Layers[0], report.Layers[2]}
	err = ValidateReport(report)
	if err == nil {
		t.Fatalf("expected out-of-order layers to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "ordered") {
		t.Fatalf("error = %v, want layer order failure", err)
	}
}

func TestBuildReportRejectsMissingUIRuntimeArtifact(t *testing.T) {
	dir := writeValidReportDir(t)
	if err := os.Remove(filepath.Join(dir, "ui-production-runtime-linux-x64.json")); err != nil {
		t.Fatal(err)
	}
	_, err := BuildReport(dir)
	if err == nil {
		t.Fatalf("expected missing UI runtime artifact to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "ui-production-runtime-linux-x64.json") {
		t.Fatalf("error = %v, want missing UI artifact", err)
	}
}

func TestBuildReportRejectsMissingChecklistArtifactReference(t *testing.T) {
	dir := writeValidReportDir(t)
	memory := validMemoryReport()
	memory.Audit[8].Artifact = "docs/spec/__missing_safe_memory_doc__.md"
	writeJSON(t, filepath.Join(dir, "memory-production-linux-x64.json"), memory)

	_, err := BuildReport(dir)
	if err == nil {
		t.Fatalf("expected missing checklist artifact reference to fail")
	}
	if !strings.Contains(err.Error(), "docs/spec/__missing_safe_memory_doc__.md") {
		t.Fatalf("error = %v, want missing artifact path", err)
	}
}

func TestValidateReportRejectsMissingPromptChecklistRequirement(t *testing.T) {
	dir := writeValidReportDir(t)
	report, err := BuildReport(dir)
	if err != nil {
		t.Fatalf("BuildReport failed: %v", err)
	}
	var filtered []ChecklistItem
	for _, item := range report.Checklist {
		if item.Requirement == "async commands and timers" {
			continue
		}
		filtered = append(filtered, item)
	}
	report.Checklist = filtered
	err = ValidateReport(report)
	if err == nil {
		t.Fatalf("expected missing checklist requirement to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "async commands and timers") {
		t.Fatalf("error = %v, want missing async commands and timers", err)
	}
}

func TestValidateReportDirRequiresFinalHashManifestToIncludeAudit(t *testing.T) {
	dir := writeValidReportDir(t)
	report, err := BuildReport(dir)
	if err != nil {
		t.Fatalf("BuildReport failed: %v", err)
	}
	writeJSON(t, filepath.Join(dir, DefaultAuditFilename), report)
	err = ValidateReportDir(dir)
	if err == nil {
		t.Fatalf("expected missing audit artifact hash to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), DefaultAuditFilename) {
		t.Fatalf("error = %v, want missing audit artifact hash", err)
	}
	writeHashManifest(t, dir, true)
	if err := ValidateReportDir(dir); err != nil {
		t.Fatalf("ValidateReportDir failed after final hash manifest: %v", err)
	}
}

func writeValidReportDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "memory-production-linux-x64.json"), validMemoryReport())
	writeJSON(t, filepath.Join(dir, "parallel-production-linux-x64.json"), validParallelReport())
	writeJSON(t, filepath.Join(dir, "ui-production-runtime-linux-x64.json"), validUIReport())
	writeJSON(t, filepath.Join(dir, "native-ui-runtime-linux-x64.integration.json"), validNativeUIReport())
	writeHashManifest(t, dir, false)
	return dir
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeHashManifest(t *testing.T, dir string, includeAudit bool) {
	t.Helper()
	artifacts := []map[string]any{
		{"path": "memory-production-linux-x64.json", "sha256": strings.Repeat("0", 64), "size": 1, "schema": memoryprod.SchemaV1},
		{"path": "parallel-production-linux-x64.json", "sha256": strings.Repeat("1", 64), "size": 1, "schema": parallelprod.SchemaV1},
		{"path": "ui-production-runtime-linux-x64.json", "sha256": strings.Repeat("2", 64), "size": 1, "schema": uiprod.SchemaV1},
		{"path": "native-ui-runtime-linux-x64.integration.json", "sha256": strings.Repeat("4", 64), "size": 1, "schema": nativeui.SchemaV1},
	}
	if includeAudit {
		artifacts = append(artifacts, map[string]any{"path": DefaultAuditFilename, "sha256": strings.Repeat("3", 64), "size": 1, "schema": SchemaV1})
	}
	writeJSON(t, filepath.Join(dir, "artifact-hashes.json"), map[string]any{
		"schema":    "tetra.release-artifact-hashes.v1alpha1",
		"root":      ".",
		"artifacts": artifacts,
	})
}

func validMemoryReport() memoryprod.Report {
	exitZero := 0
	return memoryprod.Report{
		Schema:  memoryprod.SchemaV1,
		Status:  "pass",
		Target:  "linux-x64",
		Host:    "linux-x64",
		Runtime: "memory-linux-x64",
		Source:  "tools/cmd/memory-production-smoke",
		Processes: []memoryprod.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "/tmp/tetra", Ran: true, Pass: true, ExitCode: &exitZero},
			{Name: "memory smoke app", Kind: "app", Path: "/tmp/memory-smoke", Ran: true, Pass: true, ExitCode: &exitZero},
			{Name: "memory stress", Kind: "stress", Path: "tools/cmd/memory-production-smoke", Ran: true, Pass: true, ExitCode: &exitZero},
			{Name: "actornet close-without-cancel leak coverage", Kind: "stress", Path: "go test -buildvcs=false ./cli/internal/actornet -run TestBrokerCloseWithoutCancelStopsServeWatcher -count=1", Ran: true, Pass: true, ExitCode: &exitZero},
			{Name: "compiler resource finalization diagnostics", Kind: "stress", Path: "go test -buildvcs=false ./compiler/tests/runtime -run ^(TestTaskHandleFinalization|TestTaskGroupFinalization|TestIslandFinalization) -count=1", Ran: true, Pass: true, ExitCode: &exitZero},
		},
		Benchmarks: []memoryprod.BenchmarkReport{
			{Name: "small heap allocation syscall reduction", Kind: "allocator", Metric: "estimated_os_syscalls", Unit: "syscalls", BaselineValue: 64, MeasuredValue: 1, ImprovementRatio: 64, Evidence: "allocation report schema v2 shows 64 per_core_small_heap rows with same_core_same_size_class_free_list reuse policy inside one 64KiB chunk refill", Ran: true, Pass: true},
		},
		Contracts: []memoryprod.ContractReport{
			{Name: "allocator runtime model", Status: "pass", Evidence: "allocator lifecycle returns deterministic handles and failure status"},
			{Name: "allocator failure semantics", Status: "pass", Evidence: "linux-x64 allocation failure exits deterministically"},
			{Name: "ownership escape model", Status: "pass", Evidence: "heap, slices, structs, and closures preserve borrow/consume diagnostics"},
			{Name: "unsafe cap.mem raw memory rules", Status: "pass", Evidence: "raw memory helpers require unsafe and explicit cap.mem"},
			{Name: "runtime bounds diagnostics", Status: "pass", Evidence: "out-of-bounds memory access reports deterministic runtime diagnostic"},
			{Name: "raw pointer bounds metadata", Status: "pass", Evidence: "allocation_base_metadata, derived_allocation_offset, checked_external_unknown, and external_unknown raw-slice policy"},
			{Name: "host resource leak and finalization checks", Status: "pass", Evidence: "actornet close-without-cancel watcher cleanup plus compiler resource finalization diagnostics ran"},
			{Name: "actor task transfer rules", Status: "pass", Evidence: "memory-bearing values cannot cross actor/task boundaries without checked transfer"},
		},
		Cases: []memoryprod.CaseReport{
			{Name: "allocator alloc/free lifecycle", Kind: "positive", Ran: true, Pass: true},
			{Name: "allocator failure semantics", Kind: "negative", Ran: true, Pass: true, ExpectedError: "allocation failure"},
			{Name: "allocator invalid size precondition", Kind: "negative", Ran: true, Pass: true, ExpectedError: "invalid allocation size"},
			{Name: "cap.mem unsafe boundary", Kind: "negative", Ran: true, Pass: true, ExpectedError: "only allowed in unsafe blocks"},
			{Name: "memcpy/memset capability path", Kind: "positive", Ran: true, Pass: true},
			{Name: "runtime bounds check", Kind: "negative", Ran: true, Pass: true, ExpectedError: "bounds"},
			{Name: "raw ptr_add negative offset bounds", Kind: "negative", Ran: true, Pass: true, ExpectedError: "negative ptr_add offset"},
			{Name: "raw ptr_add allocation upper bound", Kind: "negative", Ran: true, Pass: true, ExpectedError: "allocation upper bound"},
			{Name: "raw allocation-base i32 access width", Kind: "negative", Ran: true, Pass: true, ExpectedError: "i32 access width exceeds allocation"},
			{Name: "raw allocation-base ptr access width", Kind: "negative", Ran: true, Pass: true, ExpectedError: "ptr access width exceeds allocation"},
			{Name: "raw slice negative length", Kind: "negative", Ran: true, Pass: true, ExpectedError: "negative raw slice length"},
			{Name: "raw slice i32 length byte overflow", Kind: "negative", Ran: true, Pass: true, ExpectedError: "raw slice length byte overflow"},
			{Name: "raw pointer bounds metadata report", Kind: "positive", Ran: true, Pass: true},
			{Name: "memcpy/memset negative length", Kind: "negative", Ran: true, Pass: true, ExpectedError: "negative helper length"},
			{Name: "reject use-after-free", Kind: "negative", Ran: true, Pass: true, ExpectedError: "use-after-free"},
			{Name: "reject double-free", Kind: "negative", Ran: true, Pass: true, ExpectedError: "double-free"},
			{Name: "reject borrow escape", Kind: "negative", Ran: true, Pass: true, ExpectedError: "borrow escape"},
			{Name: "reject aliasing violation", Kind: "negative", Ran: true, Pass: true, ExpectedError: "alias"},
			{Name: "callable mutable capture heap escape", Kind: "negative", Ran: true, Pass: true, ExpectedError: "heap-escaped function value captures mutable local"},
			{Name: "reject actor task transfer violation", Kind: "negative", Ran: true, Pass: true, ExpectedError: "transfer"},
			{Name: "heap closure handle coverage", Kind: "positive", Ran: true, Pass: true},
			{Name: "slice struct borrow escape coverage", Kind: "negative", Ran: true, Pass: true, ExpectedError: "borrow escape"},
			{Name: "function-typed slice aggregate borrow escape coverage", Kind: "negative", Ran: true, Pass: true, ExpectedError: "borrow escape"},
			{Name: "actornet broker close-without-cancel leak smoke", Kind: "stress", Ran: true, Pass: true},
			{Name: "compiler resource finalization diagnostics", Kind: "negative", Ran: true, Pass: true, ExpectedError: "resource finalization"},
			{Name: "real memory examples", Kind: "positive", Ran: true, Pass: true},
			{Name: "stress allocator reuse", Kind: "stress", Ran: true, Pass: true},
			{Name: "deterministic memcpy/memset fuzz", Kind: "stress", Ran: true, Pass: true},
		},
		Audit: []memoryprod.AuditReport{
			{Requirement: "stable allocator/runtime memory model", Artifact: "lib/core/memory.tetra; tools/cmd/memory-production-smoke", Evidence: "allocator lifecycle, invalid size, failure, and stress cases ran", Result: "pass"},
			{Requirement: "ownership/borrow/consume escape model", Artifact: "compiler/tests/ownership", Evidence: "borrow escape, use-after-free, double-free, aliasing, callable heap escape, and transfer diagnostics are required", Result: "pass"},
			{Requirement: "heap, slices, structs, and closures memory coverage", Artifact: "docs/spec/ownership_v1.md; compiler/tests/ownership; compiler/tests/semantics/closures_semantic_clauses_test.go", Evidence: "heap closure handle coverage, callable heap escape rejection, slice struct borrow escape coverage, and function-typed slice aggregate borrow escape coverage cases ran", Result: "pass"},
			{Requirement: "unsafe/cap.mem/raw memory/memcpy/memset rules", Artifact: "docs/spec/unsafe.md; lib/core/memory.tetra", Evidence: "cap.mem unsafe boundary plus memcpy/memset capability path and negative length cases require unsafe cap.mem", Result: "pass"},
			{Requirement: "runtime bounds checks and diagnostics", Artifact: "docs/spec/runtime_abi.md", Evidence: "bounds and raw ptr_add diagnostics are required cases", Result: "pass"},
			{Requirement: "raw pointer bounds metadata", Artifact: "compiler/internal/runtimeabi/raw_pointer_bounds.go; compiler/internal/plir/plir.go; compiler/internal/allocplan/plan.go; tools/cmd/memory-production-smoke", Evidence: "core.alloc_bytes allocation reports include allocation_base_metadata and external_unknown raw-slice policy; PLIR records derived_allocation_offset and checked_external_unknown raw pointer paths", Result: "pass"},
			{Requirement: "stress/fuzz evidence", Artifact: "tools/cmd/memory-production-smoke", Evidence: "allocator stress and deterministic memcpy/memset fuzz cases ran", Result: "pass"},
			{Requirement: "measured memory benchmark improvement", Artifact: "tools/cmd/memory-production-smoke; compiler allocation report schema v2", Evidence: "small heap allocation syscall reduction benchmark compares estimated mmap-per-allocation baseline against 64KiB chunk refill calls", Result: "pass"},
			{Requirement: "use-after-free, double-free, borrow escape, and aliasing safety", Artifact: "compiler/tests/safety; compiler/tests/ownership", Evidence: "required cases reject unsafe memory behavior", Result: "pass"},
			{Requirement: "actor/task transfer safety", Artifact: "compiler/tests/ownership", Evidence: "actor/task transfer safety case is required", Result: "pass"},
			{Requirement: "leak/resource finalization evidence", Artifact: "cli/internal/actornet/broker_test.go; compiler/tests/runtime/resource_finalization_test.go; tools/cmd/memory-production-smoke", Evidence: "release smoke runs actornet close-without-cancel watcher leak coverage and compiler resource finalization diagnostics", Result: "pass"},
			{Requirement: "real memory examples", Artifact: "examples/core_memory_smoke.tetra; examples/ownership_smoke.tetra; examples/flow_unsafe_cap_mem_smoke.tetra", Evidence: "checked-in memory examples build and run", Result: "pass"},
			{Requirement: "safe memory documentation", Artifact: "docs/spec/runtime_abi.md; docs/spec/ownership_v1.md", Evidence: "safe memory docs are verified", Result: "pass"},
			{Requirement: "release-gate entrypoint", Artifact: "scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh", Evidence: "memory gate writes and validates production evidence", Result: "pass"},
		},
	}
}

func validParallelReport() parallelprod.Report {
	exitZero := 0
	return parallelprod.Report{
		Schema:  parallelprod.SchemaV1,
		Status:  "pass",
		Target:  "linux-x64",
		Host:    "linux-x64",
		Runtime: "parallel-linux-x64",
		Source:  "tools/cmd/parallel-production-smoke",
		Processes: []parallelprod.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "/tmp/tetra", Ran: true, Pass: true, ExitCode: &exitZero},
			{Name: "parallel smoke app", Kind: "app", Path: "/tmp/parallel-smoke", Ran: true, Pass: true, ExitCode: &exitZero},
			{Name: "parallel stress", Kind: "stress", Path: "/tmp/parallel-stress", Ran: true, Pass: true, ExitCode: &exitZero},
			{Name: "parallel scheduler prototype tests", Kind: "benchmark", Path: "go test ./compiler/internal/parallelrt", Ran: true, Pass: true, ExitCode: &exitZero},
			{Name: "parallel scheduler prototype evidence", Kind: "benchmark", Path: "go run ./compiler/cmd/parallelrt-evidence", Ran: true, Pass: true, ExitCode: &exitZero},
		},
		Benchmarks: []parallelprod.BenchmarkReport{
			{Name: "actor ping-pong benchmark prep", Kind: "actor_benchmark_prep", Metric: "messages_round_trip", Unit: "prep_only", Evidence: "compiler/actors_test.go::TestActorsPingPongBuildAndRun and examples/actors_pingpong.tetra define the local Linux-x64 actor ping-pong workload candidate", ClaimTier: "tier0_local_smoke_only", Claim: "Actor ping-pong benchmark prep row exists as Tier 0 local smoke only; no measured result is published and cross-runtime comparison is out of scope.", RawOutputArtifacts: []string{"reports/actor-runtime-foundation/P15/parallelrt-evidence.raw.json"}, Ran: false, Pass: true},
			{Name: "actor fanout/fanin benchmark prep", Kind: "actor_benchmark_prep", Metric: "fanout_fanin_messages", Unit: "prep_only", Evidence: "compiler/internal/parallelrt two-core work stealing model checks actor fanout/fanin scheduling shape without publishing throughput", ClaimTier: "tier0_local_smoke_only", Claim: "Actor fanout/fanin benchmark prep row exists as Tier 0 local smoke only; it records local workload shape and leaves public benchmark publication out of scope.", RawOutputArtifacts: []string{"reports/actor-runtime-foundation/P15/parallelrt-evidence.raw.json"}, Ran: false, Pass: true},
			{Name: "actor mailbox throughput benchmark prep", Kind: "actor_benchmark_prep", Metric: "mailbox_messages", Unit: "prep_only", Evidence: "compiler/internal/parallelrt TypedMailbox and parallel production actor mailbox cases define the local mailbox throughput workload candidate", ClaimTier: "tier0_local_smoke_only", Claim: "Actor mailbox throughput benchmark prep row exists as Tier 0 local smoke only; it publishes no measured result and no throughput guarantee.", RawOutputArtifacts: []string{"reports/actor-runtime-foundation/P15/parallelrt-evidence.raw.json"}, Ran: false, Pass: true},
			{Name: "actor backpressure latency benchmark prep", Kind: "actor_benchmark_prep", Metric: "backpressure_wait", Unit: "prep_only", Evidence: "compiler/internal/parallelrt ErrMailboxFull and blocking_recv_yield metadata define the local backpressure latency diagnostic candidate", ClaimTier: "tier0_local_smoke_only", Claim: "Actor backpressure latency benchmark prep row exists as Tier 0 local smoke only; no real-world SLA or latency advantage is claimed.", RawOutputArtifacts: []string{"reports/actor-runtime-foundation/P15/parallelrt-evidence.raw.json"}, Ran: false, Pass: true},
			{Name: "zero_copy_move local typed mailbox benchmark prep", Kind: "actor_transfer_prep", Metric: "owned_region_transfer", Unit: "prep_only", Evidence: "compiler/internal/parallelrt owned-region transfer report emits zero_copy_move for local typed mailbox metadata only", ClaimTier: "tier0_local_smoke_only", Claim: "zero_copy_move local typed mailbox benchmark prep row exists as Tier 0 local smoke only; it records local owned-region metadata and leaves distributed or network transfer behavior out of scope.", RawOutputArtifacts: []string{"reports/actor-runtime-foundation/P15/parallelrt-evidence.raw.json"}, Ran: false, Pass: true},
		},
		Contracts: []parallelprod.ContractReport{
			{Name: "production task scheduler", Status: "pass", Evidence: "scheduler fairness and lifecycle cases ran"},
			{Name: "join cancel deadline select group lifecycle", Status: "pass", Evidence: "join, cancel, deadline, select, and group lifecycle diagnostics are stable"},
			{Name: "actor mailbox backpressure and failure handling", Status: "pass", Evidence: "mailbox capacity and actor failure cases are covered"},
			{Name: "task actor thread boundary transfer rules", Status: "pass", Evidence: "ownership transfer diagnostics protect boundaries"},
			{Name: "race safety model", Status: "pass", Evidence: "shared mutable state crossing parallel boundaries is rejected conservatively"},
			{Name: "safe unsafe forbidden parallelism boundary", Status: "pass", Evidence: "docs and diagnostics define safe unsafe forbidden parallel behavior"},
		},
		Cases: []parallelprod.CaseReport{
			{Name: "scheduler fairness", Kind: "positive", Ran: true, Pass: true},
			{Name: "task join lifecycle", Kind: "positive", Ran: true, Pass: true},
			{Name: "task cancellation", Kind: "negative", Ran: true, Pass: true, ExpectedError: "cancelled"},
			{Name: "deadline timeout", Kind: "negative", Ran: true, Pass: true, ExpectedError: "deadline"},
			{Name: "select readiness", Kind: "positive", Ran: true, Pass: true},
			{Name: "task group lifecycle", Kind: "positive", Ran: true, Pass: true},
			{Name: "task group cancel wakes deadline join", Kind: "negative", Ran: true, Pass: true, ExpectedError: "cancelled before deadline"},
			{Name: "nested cancellation propagation", Kind: "positive", Ran: true, Pass: true},
			{Name: "task actor mailbox handoff", Kind: "positive", Ran: true, Pass: true},
			{Name: "actor recv cancel wake", Kind: "negative", Ran: true, Pass: true, ExpectedError: "actor recv cancel wake"},
			{Name: "actor mailbox backpressure", Kind: "negative", Ran: true, Pass: true, ExpectedError: "backpressure"},
			{Name: "message pool exhaustion", Kind: "negative", Ran: true, Pass: true, ExpectedError: "message pool exhausted"},
			{Name: "invalid actor handle send", Kind: "negative", Ran: true, Pass: true, ExpectedError: "invalid actor handle"},
			{Name: "done actor send", Kind: "negative", Ran: true, Pass: true, ExpectedError: "done actor"},
			{Name: "actor failure handling", Kind: "negative", Ran: true, Pass: true, ExpectedError: "actor failed"},
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
			{Name: "many tasks stress", Kind: "stress", Ran: true, Pass: true},
			{Name: "many actor messages stress", Kind: "stress", Ran: true, Pass: true},
			{Name: "cancellation storm", Kind: "stress", Ran: true, Pass: true},
			{Name: "timeouts stress", Kind: "stress", Ran: true, Pass: true},
		},
		Diagnostics: []parallelprod.DiagnosticReport{
			{Case: "task cancellation", Code: "TASK_CANCELLED", Severity: "error", Category: "task", Position: "runtime", ExpectedError: "cancelled"},
			{Case: "deadline timeout", Code: "TASK_DEADLINE_TIMEOUT", Severity: "error", Category: "task", Position: "runtime", ExpectedError: "deadline"},
			{Case: "task group cancel wakes deadline join", Code: "TASK_GROUP_CANCEL_WAKE_JOIN", Severity: "error", Category: "task", Position: "runtime", ExpectedError: "cancelled before deadline"},
			{Case: "actor recv cancel wake", Code: "ACTOR_RECV_CANCEL_WAKE", Severity: "error", Category: "actor", Position: "runtime", ExpectedError: "actor recv cancel wake"},
			{Case: "actor mailbox backpressure", Code: "ACTOR_MAILBOX_BACKPRESSURE", Severity: "error", Category: "actor", Position: "runtime", ExpectedError: "backpressure"},
			{Case: "message pool exhaustion", Code: "ACTOR_MESSAGE_POOL_EXHAUSTED", Severity: "error", Category: "actor", Position: "runtime", ExpectedError: "message pool exhausted"},
			{Case: "invalid actor handle send", Code: "ACTOR_INVALID_HANDLE_SEND", Severity: "error", Category: "actor", Position: "runtime", ExpectedError: "invalid actor handle"},
			{Case: "done actor send", Code: "ACTOR_DONE_SEND", Severity: "error", Category: "actor", Position: "runtime", ExpectedError: "done actor"},
			{Case: "actor failure handling", Code: "ACTOR_MISSING_NODE_FAILURE", Severity: "error", Category: "actor", Position: "runtime", ExpectedError: "actor failed"},
			{Case: "invalid handle diagnostics", Code: "ACTOR_INVALID_HANDLE_DIAGNOSTIC", Severity: "error", Category: "actor", Position: "cli-json", ExpectedError: "invalid handle"},
			{Case: "resource double join diagnostic", Code: "RESOURCE_DOUBLE_JOIN", Severity: "error", Category: "resource", Position: "cli-json", ExpectedError: "joined"},
			{Case: "task group use-after-close diagnostic", Code: "TASK_GROUP_CLOSED", Severity: "error", Category: "task", Position: "cli-json", ExpectedError: "closed"},
			{Case: "ownership transfer across task boundary", Code: "OWNERSHIP_TASK_TRANSFER", Severity: "error", Category: "ownership", Position: "compiler", ExpectedError: "transfer"},
			{Case: "ownership transfer across actor boundary", Code: "OWNERSHIP_ACTOR_TRANSFER", Severity: "error", Category: "ownership", Position: "compiler", ExpectedError: "transfer"},
			{Case: "race-safety shared mutable rejection", Code: "RACE_SHARED_MUTABLE_REJECTED", Severity: "error", Category: "race-safety", Position: "compiler", ExpectedError: "shared mutable"},
		},
		Audit: []parallelprod.AuditReport{
			{Requirement: "production task scheduler", Artifact: "compiler/task_runtime_test.go", Evidence: "scheduler fairness and many tasks stress cases ran", Result: "pass"},
			{Requirement: "join/cancel/deadline/select/group lifecycle", Artifact: "compiler/task_runtime_test.go", Evidence: "join cancellation deadline select task group cancel-wakes-deadline-join and nested cancellation cases are required", Result: "pass"},
			{Requirement: "actor mailbox backpressure and failure handling", Artifact: "compiler/actors_test.go", Evidence: "mailbox backpressure task actor mailbox handoff and failure cases are required", Result: "pass"},
			{Requirement: "task/actor/thread-boundary transfer rules", Artifact: "compiler/tests/ownership; cli/cmd/tetra/check_diagnostics_resource_actor_test.go", Evidence: "task and actor transfer double join and closed group diagnostics are required", Result: "pass"},
			{Requirement: "race-safety model or conservative rejections", Artifact: "compiler/tests/ownership; docs/spec/actors.md", Evidence: "shared mutable crossing is rejected conservatively", Result: "pass"},
			{Requirement: "stress evidence for tasks, actor messages, cancellation storms, and timeouts", Artifact: "tools/cmd/parallel-production-smoke", Evidence: "required stress cases ran", Result: "pass"},
			{Requirement: "safe/unsafe/forbidden parallelism documentation", Artifact: "docs/spec/actors.md; docs/user/async_actors_guide.md; docs/spec/runtime_abi.md; compiler/tests/semantics/async_test.go; compiler/tests/safety/effects_test.go", Evidence: "safe unsafe forbidden boundary coverage case runs compiler tests and docs define boundaries", Result: "pass"},
			{Requirement: "stable parallel diagnostics", Artifact: "compiler/task_runtime_test.go; compiler/actors_test.go; compiler/tests/ownership/actor_task_ownership_test.go; cli/cmd/tetra/check_diagnostics_resource_actor_test.go", Evidence: "negative parallel cases require stable expected_error evidence for cancellation deadline backpressure invalid handle double join use-after-close transfer and shared mutable rejection", Result: "pass"},
			{Requirement: "actor benchmark Tier 0/Tier 1 preparation", Artifact: "compiler/internal/parallelrt; tools/cmd/parallel-production-smoke", Evidence: "parallelrt evidence emits Tier 0 actor ping-pong, fanout/fanin, mailbox throughput, backpressure latency, and zero_copy_move local typed mailbox prep rows with raw artifact references and no performance claim", Result: "pass"},
			{Requirement: "release-gate entrypoint", Artifact: "scripts/release/post_v0_4/parallel-production-linux-x64-smoke.sh", Evidence: "parallel gate writes and validates production evidence", Result: "pass"},
		},
	}
}

func validUIReport() uiprod.Report {
	exitZero := 0
	return uiprod.Report{
		Schema:   uiprod.SchemaV1,
		Status:   "pass",
		Target:   "linux-x64",
		Host:     "linux-x64",
		Runtime:  "desktop-ui-linux-x64",
		UISchema: "tetra.ui.v0.4.0",
		Source:   "tools/cmd/ui-production-runtime-smoke",
		Processes: []uiprod.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "/tmp/tetra", Ran: true, Pass: true, ExitCode: &exitZero},
			{Name: "desktop UI app", Kind: "app", Path: "/tmp/ui-desktop", Ran: true, Pass: true, ExitCode: &exitZero},
			{Name: "desktop UI runtime", Kind: "runtime", Path: "tools/cmd/ui-production-runtime-smoke", Ran: true, Pass: true, ExitCode: &exitZero},
			{Name: "native shell runtime integration", Kind: "runtime", Path: "go run ./tools/cmd/native-ui-runtime-smoke", Ran: true, Pass: true, ExitCode: &exitZero},
			{Name: "native runtime evidence validator", Kind: "runtime", Path: "go run ./tools/cmd/validate-native-ui-runtime", Ran: true, Pass: true, ExitCode: &exitZero},
			{Name: "desktop UI widget stress", Kind: "stress", Path: "/tmp/ui-widget-stress", Ran: true, Pass: true, ExitCode: &exitZero},
		},
		Contracts: []uiprod.ContractReport{
			{Name: "Linux-x64 desktop UI runtime", Status: "pass", Evidence: "desktop UI and sidecar-driven native runtime process evidence ran on linux-x64"},
			{Name: "window lifecycle", Status: "pass", Evidence: "window lifecycle is covered"},
			{Name: "layout system", Status: "pass", Evidence: "layout measure/place and panel nesting cases ran"},
			{Name: "buttons text input lists panels state binding", Status: "pass", Evidence: "button text input focus change list panel and bound state widgets are present"},
			{Name: "event loop", Status: "pass", Evidence: "focus input change select click and timer events ran through runtime"},
			{Name: "async UI commands", Status: "pass", Evidence: "async command completion case runs"},
			{Name: "timers", Status: "pass", Evidence: "timer scheduled redraw case records a tick event and timer_tick operation"},
			{Name: "redraw update model", Status: "pass", Evidence: "redraw lifecycle case records state to redraw"},
			{Name: "error crash handling", Status: "pass", Evidence: "invalid widget command failure and crash handling cases are required"},
			{Name: "real dogfood applications", Status: "pass", Evidence: "dogfood application smoke case uses real Tetra UI source"},
		},
		Widgets: []uiprod.WidgetReport{
			{ID: "AppWindow", Kind: "window", Binding: "app.open", Enabled: true, Visible: true, Bounds: uiprod.Bounds{Width: 640, Height: 480}},
			{ID: "RootPanel", Kind: "panel", Parent: "AppWindow", Binding: "layout.root", Enabled: true, Visible: true, Bounds: uiprod.Bounds{Width: 640, Height: 480}},
			{ID: "TitleText", Kind: "text", Parent: "RootPanel", Binding: "state.title", Value: "Saved after timer", Enabled: true, Visible: true, Bounds: uiprod.Bounds{Width: 608, Height: 32}},
			{ID: "NameInput", Kind: "input", Parent: "RootPanel", Binding: "state.name", Event: "input", Value: "tetra-prod", Enabled: true, Visible: true, Bounds: uiprod.Bounds{Width: 608, Height: 32}},
			{ID: "ItemList", Kind: "list", Parent: "RootPanel", Binding: "state.items", Event: "select", Value: "item-1", Enabled: true, Visible: true, Bounds: uiprod.Bounds{Width: 608, Height: 240}},
			{ID: "SaveButton", Kind: "button", Parent: "RootPanel", Binding: "state.saved", Event: "click", Command: "saveAsync", Enabled: true, Visible: true, Bounds: uiprod.Bounds{Width: 200, Height: 44}},
		},
		Events: []uiprod.EventReport{
			{Order: 1, WidgetID: "NameInput", Event: "focus", Command: "focusName", Pass: true, BeforeState: map[string]string{"AppState.focused": "none"}, AfterState: map[string]string{"AppState.focused": "NameInput"}, Operations: []uiprod.OperationReport{{Kind: "focus", Target: "widget.NameInput", Value: "focused", StateField: "focused", StateValue: "NameInput"}}, WidgetUpdates: []uiprod.WidgetUpdateReport{{ID: "TitleText", Before: "Ready", After: "Editing name"}}},
			{Order: 2, WidgetID: "NameInput", Event: "input", Command: "setName", Pass: true, BeforeState: map[string]string{"AppState.name": "tetra"}, AfterState: map[string]string{"AppState.name": "tetra-lang"}, Operations: []uiprod.OperationReport{{Kind: "state_set", Target: "state.name", Value: "tetra-lang", StateField: "name", StateValue: "tetra-lang"}}, WidgetUpdates: []uiprod.WidgetUpdateReport{{ID: "NameInput", Before: "tetra", After: "tetra-lang"}}},
			{Order: 3, WidgetID: "NameInput", Event: "change", Command: "commitName", Pass: true, BeforeState: map[string]string{"AppState.name": "tetra-lang", "AppState.changed": "false"}, AfterState: map[string]string{"AppState.name": "tetra-prod", "AppState.changed": "true"}, Operations: []uiprod.OperationReport{{Kind: "change", Target: "state.name", Value: "tetra-prod", StateField: "name", StateValue: "tetra-prod"}, {Kind: "state_set", Target: "state.changed", Value: "true", StateField: "changed", StateValue: "true"}}, WidgetUpdates: []uiprod.WidgetUpdateReport{{ID: "NameInput", Before: "tetra-lang", After: "tetra-prod"}}},
			{Order: 4, WidgetID: "ItemList", Event: "select", Command: "selectItem", Pass: true, BeforeState: map[string]string{"AppState.selected": "item-1"}, AfterState: map[string]string{"AppState.selected": "item-2"}, Operations: []uiprod.OperationReport{{Kind: "state_set", Target: "state.selected", Value: "item-2", StateField: "selected", StateValue: "item-2"}}, WidgetUpdates: []uiprod.WidgetUpdateReport{{ID: "ItemList", Before: "item-1", After: "item-2"}}},
			{Order: 5, WidgetID: "SaveButton", Event: "click", Command: "saveAsync", Pass: true, BeforeState: map[string]string{"AppState.saved": "false"}, AfterState: map[string]string{"AppState.saved": "true"}, Operations: []uiprod.OperationReport{{Kind: "async_command", Target: "command.saveAsync", Value: "completed", StateField: "saved", StateValue: "true"}, {Kind: "redraw", Target: "AppWindow", Value: "scheduled", StateField: "dirty", StateValue: "false"}}, WidgetUpdates: []uiprod.WidgetUpdateReport{{ID: "TitleText", Before: "Editing name", After: "Saved"}}},
			{Order: 6, WidgetID: "AppWindow", Event: "tick", Command: "timerTick", Pass: true, BeforeState: map[string]string{"AppState.dirty": "true"}, AfterState: map[string]string{"AppState.dirty": "false"}, Operations: []uiprod.OperationReport{{Kind: "timer_tick", Target: "timer.redraw", Value: "fired", StateField: "dirty", StateValue: "false"}, {Kind: "redraw", Target: "AppWindow", Value: "completed", StateField: "dirty", StateValue: "false"}}, WidgetUpdates: []uiprod.WidgetUpdateReport{{ID: "TitleText", Before: "Saved", After: "Saved after timer"}}},
		},
		Cases: []uiprod.CaseReport{
			{Name: "window lifecycle", Kind: "positive", Ran: true, Pass: true},
			{Name: "layout measure and place", Kind: "positive", Ran: true, Pass: true},
			{Name: "button command dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "text render", Kind: "positive", Ran: true, Pass: true},
			{Name: "input focus traversal", Kind: "positive", Ran: true, Pass: true},
			{Name: "input edit", Kind: "positive", Ran: true, Pass: true},
			{Name: "input change commit", Kind: "positive", Ran: true, Pass: true},
			{Name: "list selection", Kind: "positive", Ran: true, Pass: true},
			{Name: "panel nesting", Kind: "positive", Ran: true, Pass: true},
			{Name: "state binding update", Kind: "positive", Ran: true, Pass: true},
			{Name: "event loop dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "async UI command completion", Kind: "positive", Ran: true, Pass: true},
			{Name: "timer scheduled redraw", Kind: "positive", Ran: true, Pass: true},
			{Name: "redraw update lifecycle", Kind: "positive", Ran: true, Pass: true},
			{Name: "compiler UI bundle runtime load", Kind: "positive", Ran: true, Pass: true},
			{Name: "native shell runtime integration", Kind: "positive", Ran: true, Pass: true},
			{Name: "native runtime sidecar consistency", Kind: "positive", Ran: true, Pass: true},
			{Name: "invalid widget diagnostic", Kind: "negative", Ran: true, Pass: true, ExpectedError: "unknown widget"},
			{Name: "command failure recovery", Kind: "negative", Ran: true, Pass: true, ExpectedError: "command failed"},
			{Name: "crash error handling", Kind: "negative", Ran: true, Pass: true, ExpectedError: "runtime panic recovered"},
			{Name: "dogfood application smoke", Kind: "positive", Ran: true, Pass: true},
			{Name: "widget tree stress", Kind: "stress", Ran: true, Pass: true},
		},
		Audit: []uiprod.AuditReport{
			{Requirement: "Linux-x64 desktop UI runtime", Artifact: "tools/cmd/ui-production-runtime-smoke", Evidence: "build app desktop runtime native runtime and stress processes ran", Result: "pass"},
			{Requirement: "window lifecycle", Artifact: "examples/ui_desktop_runtime_smoke.tetra", Evidence: "window lifecycle case is required", Result: "pass"},
			{Requirement: "layout system", Artifact: "compiler/internal/lower/ui.go; docs/spec/ui_v0.4.0.md", Evidence: "layout measure/place and panel nesting are required", Result: "pass"},
			{Requirement: "buttons/text/input/lists/panels widgets", Artifact: "examples/ui_desktop_runtime_smoke.tetra", Evidence: "widget tree includes required widgets", Result: "pass"},
			{Requirement: "state binding", Artifact: "tools/validators/uiprod", Evidence: "state binding update plus input focus/change widget update evidence are required", Result: "pass"},
			{Requirement: "event loop and redraw/update model", Artifact: "tools/cmd/ui-production-runtime-smoke", Evidence: "focus input change select click timer and redraw lifecycle cases are required", Result: "pass"},
			{Requirement: "async commands and timers", Artifact: "tools/cmd/ui-production-runtime-smoke", Evidence: "async command, timer tick event evidence, and timer scheduled redraw cases are required", Result: "pass"},
			{Requirement: "error/crash handling", Artifact: "tools/validators/uiprod", Evidence: "invalid widget command failure and crash cases are required", Result: "pass"},
			{Requirement: "real examples and dogfood applications", Artifact: "examples/ui_desktop_runtime_smoke.tetra", Evidence: "dogfood smoke and native runtime integration cases are required", Result: "pass"},
			{Requirement: "compiler-emitted UI bundle/native-shell trace load evidence", Artifact: "examples/ui_desktop_runtime_smoke.tetra; <output>.ui.json; <output>.ui.shell.json", Evidence: "UI production smoke loads compiler-emitted UI bundle and native-shell trace before accepting runtime evidence", Result: "pass"},
			{Requirement: "sidecar-driven native UI runtime integration", Artifact: "tools/cmd/native-ui-runtime-smoke; tools/cmd/validate-native-ui-runtime; native-ui-runtime-linux-x64.integration.json", Evidence: "UI production smoke runs and validates tetra.ui.native-runtime.v1 consistency evidence", Result: "pass"},
			{Requirement: "stable UI diagnostics", Artifact: "tools/cmd/ui-production-runtime-smoke; tools/validators/uiprod", Evidence: "negative UI cases require stable expected_error evidence", Result: "pass"},
			{Requirement: "release-gate entrypoint rejecting runtime-less evidence", Artifact: "scripts/release/post_v0_4/ui-production-runtime-linux-x64-smoke.sh", Evidence: "validator rejects non-runtime evidence", Result: "pass"},
		},
	}
}

func validNativeUIReport() nativeui.Report {
	exitZero := 0
	return nativeui.Report{
		Schema:   nativeui.SchemaV1,
		Status:   "pass",
		Target:   "linux-x64",
		Host:     "linux-x64",
		Runtime:  "native-ui-linux-x64",
		UISchema: "tetra.ui.v0.4.0",
		Source:   "examples/ui_native_shell_smoke.tetra",
		Processes: []nativeui.ProcessReport{
			{Name: "tetra build native UI", Kind: "build", Path: "tetra build --target linux-x64", Ran: true, Pass: true, ExitCode: &exitZero},
			{Name: "native app", Kind: "app", Path: "/tmp/ui-native", Ran: true, Pass: true, ExitCode: &exitZero},
			{Name: "native ui runtime", Kind: "runtime", Path: "tools/cmd/native-ui-runtime-smoke", Ran: true, Pass: true, ExitCode: &exitZero},
		},
		Widgets: []nativeui.WidgetReport{
			{ID: "CounterView", Kind: "view", Enabled: true, Visible: true, Bounds: nativeui.Bounds{Width: 320, Height: 96}},
			{ID: "CounterView.count", Kind: "text", Parent: "CounterView", Binding: "count", Value: "0", Enabled: true, Visible: true, Bounds: nativeui.Bounds{Width: 304, Height: 24}},
			{ID: "CounterView.increment", Kind: "action", Parent: "CounterView", Event: "click", Command: "increment", Enabled: true, Visible: true, Bounds: nativeui.Bounds{Width: 304, Height: 24}},
		},
		Events: []nativeui.EventReport{
			{Order: 1, WidgetID: "CounterView.increment", Event: "click", Command: "increment", Pass: true, BeforeState: map[string]string{"CounterState.count": "0"}, AfterState: map[string]string{"CounterState.count": "1"}, Operations: []nativeui.OperationReport{{Kind: "state_add", Target: "state.count", Value: "1", StateField: "count", StateValue: "1"}}, WidgetUpdates: []nativeui.WidgetUpdateReport{{ID: "CounterView.count", Before: "0", After: "1"}}},
		},
		Cases: []nativeui.CaseReport{
			{Name: "load widget tree", Ran: true, Pass: true},
			{Name: "dispatch click command", Ran: true, Pass: true},
			{Name: "propagate state update", Ran: true, Pass: true},
			{Name: "dispatch multiple ordered events", Ran: true, Pass: true},
			{Name: "reject invalid widget id", Ran: true, Pass: true, ExpectedError: "unknown widget"},
			{Name: "reject malformed metadata", Ran: true, Pass: true, ExpectedError: "malformed metadata"},
			{Name: "reject unsupported event kind", Ran: true, Pass: true, ExpectedError: "unsupported event"},
			{Name: "reject command failure", Ran: true, Pass: true, ExpectedError: "unknown command"},
			{Name: "close runtime", Ran: true, Pass: true},
		},
	}
}
