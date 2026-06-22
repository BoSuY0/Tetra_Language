package runtimeabi

import (
	"runtime"
	"strings"
	"testing"
)

func TestRuntimeMemoryBackendContractDefinesTargetNeutralOperations(t *testing.T) {
	contract := RuntimeMemoryBackendContract("linux-x64")
	if contract.Target != "linux-x64" {
		t.Fatalf("target = %q, want linux-x64", contract.Target)
	}
	if contract.Schema != MemoryBackendContractSchemaV1 {
		t.Fatalf("schema = %q, want %q", contract.Schema, MemoryBackendContractSchemaV1)
	}
	for _, op := range []MemoryBackendOperation{
		MemoryBackendReserve,
		MemoryBackendCommit,
		MemoryBackendDecommit,
		MemoryBackendRelease,
		MemoryBackendTrim,
		MemoryBackendFootprint,
	} {
		if !contract.SupportsOperation(op) {
			t.Fatalf("contract operations = %v, missing %s", contract.Operations, op)
		}
	}
	if err := ValidateMemoryBackendContract(contract); err != nil {
		t.Fatalf("ValidateMemoryBackendContract: %v", err)
	}
}

func TestRuntimeMemoryBackendContractUsesOperationSupportRows(t *testing.T) {
	linux := RuntimeMemoryBackendContract("linux-x64")
	if len(linux.OperationSupport) != len(RequiredMemoryBackendOperations()) {
		t.Fatalf("linux support rows = %d, want %d", len(linux.OperationSupport), len(RequiredMemoryBackendOperations()))
	}
	for _, op := range RequiredMemoryBackendOperations() {
		row, ok := linux.OperationSupportFor(op)
		if !ok {
			t.Fatalf("linux missing support row for %s", op)
		}
		if !row.Supported || row.Method == "" {
			t.Fatalf("linux row for %s = %+v, want supported method", op, row)
		}
	}

	wasm := RuntimeMemoryBackendContract("wasm32-wasi")
	for _, op := range []MemoryBackendOperation{MemoryBackendReserve, MemoryBackendCommit} {
		row, ok := wasm.OperationSupportFor(op)
		if !ok || !row.Supported || !strings.Contains(row.Method, "memory_grow") {
			t.Fatalf("wasm %s support row = %+v,%v, want memory.grow support", op, row, ok)
		}
	}
	for _, op := range []MemoryBackendOperation{
		MemoryBackendDecommit,
		MemoryBackendRelease,
		MemoryBackendTrim,
		MemoryBackendFootprint,
	} {
		row, ok := wasm.OperationSupportFor(op)
		if !ok || row.Supported || row.UnsupportedReason == "" {
			t.Fatalf("wasm %s support row = %+v,%v, want explicit unsupported", op, row, ok)
		}
	}
	if wasm.SupportsOperation(MemoryBackendRelease) {
		t.Fatalf("wasm release must be explicitly unsupported")
	}
	if err := ValidateMemoryBackendContract(wasm); err != nil {
		t.Fatalf("wasm contract invalid: %v", err)
	}
}

func TestMemoryBackendContractRejectsMalformedSupportRows(t *testing.T) {
	contract := RuntimeMemoryBackendContract("linux-x64")
	contract.OperationSupport[0].Method = ""
	if err := ValidateMemoryBackendContract(contract); err == nil || !strings.Contains(err.Error(), "requires method") {
		t.Fatalf("missing method validation error = %v", err)
	}

	contract = RuntimeMemoryBackendContract("wasm32-web")
	contract.OperationSupport[2].UnsupportedReason = ""
	if err := ValidateMemoryBackendContract(contract); err == nil || !strings.Contains(err.Error(), "unsupported_reason") {
		t.Fatalf("missing unsupported reason validation error = %v", err)
	}

	contract = RuntimeMemoryBackendContract("linux-x64")
	contract.OperationSupport = append(contract.OperationSupport, contract.OperationSupport[0])
	if err := ValidateMemoryBackendContract(contract); err == nil || !strings.Contains(err.Error(), "operation support rows") {
		t.Fatalf("duplicate row count validation error = %v", err)
	}
}

func TestRuntimeMemoryBackendContractReportsEvidenceClasses(t *testing.T) {
	linux := RuntimeMemoryBackendContract("linux-x64")
	if linux.FootprintEvidenceClass != MemoryFootprintMeasured ||
		linux.FootprintMethod != "linux_proc_self_status_vmrss_vmhwm" {
		t.Fatalf("linux footprint evidence = %q/%q, want measured/linux_proc_self_status_vmrss_vmhwm", linux.FootprintEvidenceClass, linux.FootprintMethod)
	}

	wasm := RuntimeMemoryBackendContract("wasm32-wasi")
	if wasm.FootprintEvidenceClass != MemoryFootprintUnsupported {
		t.Fatalf("wasm evidence class = %q, want unsupported", wasm.FootprintEvidenceClass)
	}
	if !strings.Contains(wasm.UnsupportedReason, "linear memory") {
		t.Fatalf("wasm unsupported reason = %q, want linear memory boundary", wasm.UnsupportedReason)
	}

	blocked := MemoryFootprintSample{
		Target:        "linux-x64",
		EvidenceClass: MemoryFootprintBlocked,
		Method:        "time_v",
		BlockedReason: "/usr/bin/time unavailable",
	}
	if err := ValidateMemoryFootprintSample(blocked); err != nil {
		t.Fatalf("blocked footprint sample invalid: %v", err)
	}
}

func TestRuntimeMemoryBackendContractKeepsNonLinuxTargetsExplicitlyUnsupported(t *testing.T) {
	for _, target := range []string{"macos-x64", "windows-x64", "unknown-target"} {
		contract := RuntimeMemoryBackendContract(target)
		if contract.FootprintEvidenceClass != MemoryFootprintUnsupported {
			t.Fatalf("%s evidence class = %q, want unsupported", target, contract.FootprintEvidenceClass)
		}
		if contract.UnsupportedReason == "" {
			t.Fatalf("%s unsupported reason is missing: %+v", target, contract)
		}
		if err := ValidateMemoryBackendContract(contract); err != nil {
			t.Fatalf("%s contract invalid: %v", target, err)
		}
	}
}

func TestMemoryFootprintSampleSeparatesMeasuredEstimatedUnsupportedAndBlocked(t *testing.T) {
	for _, sample := range []MemoryFootprintSample{
		{Target: "linux-x64", EvidenceClass: MemoryFootprintMeasured, Method: "linux_proc_status", CurrentBytes: 1024, PeakBytes: 2048},
		{Target: "linux-x64", EvidenceClass: MemoryFootprintEstimated, Method: "allocation_report_summary", CurrentBytes: 1024, PeakBytes: 2048},
		{Target: "wasm32-web", EvidenceClass: MemoryFootprintUnsupported, Method: "browser_host_unavailable", UnsupportedReason: "host RSS is unavailable"},
		{Target: "linux-x64", EvidenceClass: MemoryFootprintBlocked, Method: "time_v", BlockedReason: "/usr/bin/time unavailable"},
	} {
		if err := ValidateMemoryFootprintSample(sample); err != nil {
			t.Fatalf("sample %+v invalid: %v", sample, err)
		}
	}

	bad := MemoryFootprintSample{Target: "linux-x64", EvidenceClass: MemoryFootprintMeasured, Method: "linux_proc_status", CurrentBytes: 4096, PeakBytes: 2048}
	if err := ValidateMemoryFootprintSample(bad); err == nil || !strings.Contains(err.Error(), "peak") {
		t.Fatalf("bad measured sample error = %v, want peak diagnostic", err)
	}

	bad = MemoryFootprintSample{Target: "linux-x64", EvidenceClass: MemoryFootprintBlocked, Method: "time_v"}
	if err := ValidateMemoryFootprintSample(bad); err == nil || !strings.Contains(err.Error(), "blocked_reason") {
		t.Fatalf("bad blocked sample error = %v, want blocked_reason diagnostic", err)
	}
}

func TestMemoryBackendAllocationEvidenceSeparatesEstimatedUnsupportedAndBlocked(t *testing.T) {
	estimated := MemoryBackendAllocationEvidence{
		Schema:                MemoryBackendAllocationEvidenceSchemaV1,
		BackendClass:          MemoryBackendClassSmallHeap,
		Adapter:               "runtime.small_heap.per_core_v1",
		RuntimePath:           AllocationPathPerCoreSmallHeap,
		Operations:            []MemoryBackendOperation{MemoryBackendReserve, MemoryBackendCommit, MemoryBackendRelease, MemoryBackendFootprint},
		EvidenceClass:         MemoryFootprintEstimated,
		Method:                "allocation_report_memory_backend_v1",
		ReserveBytes:          32,
		CommitBytes:           32,
		ReleaseBytes:          32,
		FootprintCurrentBytes: 32,
		FootprintPeakBytes:    32,
	}
	if err := ValidateMemoryBackendAllocationEvidence(estimated); err != nil {
		t.Fatalf("estimated allocation evidence invalid: %v", err)
	}

	unsupported := MemoryBackendAllocationEvidence{
		Schema:            MemoryBackendAllocationEvidenceSchemaV1,
		BackendClass:      MemoryBackendClassNone,
		Adapter:           "no_runtime_memory_backend",
		RuntimePath:       AllocationPathStackFrame,
		EvidenceClass:     MemoryFootprintUnsupported,
		Method:            "no_runtime_memory_backend",
		UnsupportedReason: "stack/register/eliminated storage does not use the runtime MemoryBackend",
	}
	if err := ValidateMemoryBackendAllocationEvidence(unsupported); err != nil {
		t.Fatalf("unsupported allocation evidence invalid: %v", err)
	}

	blocked := MemoryBackendAllocationEvidence{
		Schema:        MemoryBackendAllocationEvidenceSchemaV1,
		BackendClass:  MemoryBackendClassConservativeHeap,
		Adapter:       "runtime.heap_conservative",
		RuntimePath:   AllocationPathHeap,
		EvidenceClass: MemoryFootprintBlocked,
		Method:        "allocator_backend_not_enabled",
		BlockedReason: "heap path has no MemoryBackend adapter evidence in this build",
	}
	if err := ValidateMemoryBackendAllocationEvidence(blocked); err != nil {
		t.Fatalf("blocked allocation evidence invalid: %v", err)
	}
}

func TestMemoryBackendAllocationEvidenceRejectsOverclaims(t *testing.T) {
	bad := MemoryBackendAllocationEvidence{
		Schema:        MemoryBackendAllocationEvidenceSchemaV1,
		BackendClass:  MemoryBackendClassSmallHeap,
		Adapter:       "runtime.small_heap.per_core_v1",
		RuntimePath:   AllocationPathPerCoreSmallHeap,
		EvidenceClass: MemoryFootprintEstimated,
		Method:        "allocation_report_memory_backend_v1",
		ReserveBytes:  32,
		CommitBytes:   64,
		Operations:    []MemoryBackendOperation{MemoryBackendReserve, MemoryBackendCommit},
	}
	if err := ValidateMemoryBackendAllocationEvidence(bad); err == nil || !strings.Contains(err.Error(), "commit bytes") {
		t.Fatalf("bad commit evidence error = %v, want commit bytes diagnostic", err)
	}

	bad = MemoryBackendAllocationEvidence{
		Schema:            MemoryBackendAllocationEvidenceSchemaV1,
		BackendClass:      MemoryBackendClassNone,
		Adapter:           "no_runtime_memory_backend",
		RuntimePath:       AllocationPathStackFrame,
		EvidenceClass:     MemoryFootprintUnsupported,
		Method:            "no_runtime_memory_backend",
		UnsupportedReason: "stack storage",
		ReserveBytes:      16,
	}
	if err := ValidateMemoryBackendAllocationEvidence(bad); err == nil || !strings.Contains(err.Error(), "unsupported evidence") {
		t.Fatalf("bad unsupported evidence error = %v, want unsupported byte diagnostic", err)
	}

	bad = MemoryBackendAllocationEvidence{
		Schema:        MemoryBackendAllocationEvidenceSchemaV1,
		BackendClass:  MemoryBackendClassConservativeHeap,
		Adapter:       "runtime.heap_conservative",
		RuntimePath:   AllocationPathHeap,
		EvidenceClass: MemoryFootprintBlocked,
		Method:        "allocator_backend_not_enabled",
	}
	if err := ValidateMemoryBackendAllocationEvidence(bad); err == nil || !strings.Contains(err.Error(), "blocked_reason") {
		t.Fatalf("bad blocked evidence error = %v, want blocked_reason diagnostic", err)
	}
}

func TestMemoryBackendRuntimeRecordsLedgerAndTelemetryEvents(t *testing.T) {
	ledger, err := NewMemoryDomainLedger(DefaultProcessMemoryDomain(0, 0))
	if err != nil {
		t.Fatalf("NewMemoryDomainLedger: %v", err)
	}
	var events []MemoryBackendRuntimeEvent
	backend, err := NewMemoryBackendRuntime(MemoryBackendRuntimeOptions{
		Target:   "linux-x64",
		Ledger:   ledger,
		DomainID: "domain:process",
		Hook: func(event MemoryBackendRuntimeEvent) error {
			events = append(events, event)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("NewMemoryBackendRuntime: %v", err)
	}
	if err := backend.Reserve("", 4096); err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if err := backend.Commit("", 4096); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := backend.RecordAllocation("", 64); err != nil {
		t.Fatalf("RecordAllocation: %v", err)
	}
	if err := backend.RecordAllocationFree("", 64); err != nil {
		t.Fatalf("RecordAllocationFree: %v", err)
	}
	if err := backend.Decommit("", 4096); err != nil {
		t.Fatalf("Decommit: %v", err)
	}
	if err := backend.Release("", 4096); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if len(events) != 4 {
		t.Fatalf("backend telemetry events = %d, want 4: %+v", len(events), events)
	}
	snapshot := ledger.Snapshot()
	if len(snapshot) != 1 {
		t.Fatalf("ledger snapshot = %+v, want process domain", snapshot)
	}
	domain := snapshot[0]
	if domain.ReservedBytes != 0 || domain.CommittedBytes != 0 || domain.CurrentBytes != 0 {
		t.Fatalf("domain after reserve/commit/free/decommit/release = %+v, want zero live backend bytes", domain)
	}
	if domain.DecommittedBytes != 4096 || domain.ReleasedBytes != 4096 {
		t.Fatalf("domain decommit/release = %+v, want 4096/4096", domain)
	}
}

func TestMemoryBackendRuntimeRejectsUnsupportedWASMOperation(t *testing.T) {
	backend, err := NewMemoryBackendRuntime(MemoryBackendRuntimeOptions{Target: "wasm32-web"})
	if err != nil {
		t.Fatalf("NewMemoryBackendRuntime: %v", err)
	}
	if err := backend.Release("domain:process", 4096); err == nil ||
		!strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("wasm release error = %v, want unsupported", err)
	}
}

func TestMeasureMemoryFootprintUsesTruthfulEvidenceClasses(t *testing.T) {
	wasm := MeasureMemoryFootprint("wasm32-wasi")
	if wasm.EvidenceClass != MemoryFootprintUnsupported ||
		wasm.UnsupportedReason == "" ||
		wasm.CurrentBytes != 0 ||
		wasm.PeakBytes != 0 {
		t.Fatalf("wasm footprint sample = %+v, want unsupported without byte evidence", wasm)
	}

	if runtime.GOOS != "linux" {
		t.Skip("linux /proc footprint sample requires linux host")
	}
	linux := MeasureMemoryFootprint("linux-x64")
	if err := ValidateMemoryFootprintSample(linux); err != nil {
		t.Fatalf("linux footprint sample invalid: %+v: %v", linux, err)
	}
	if linux.EvidenceClass == MemoryFootprintMeasured &&
		(linux.CurrentBytes <= 0 || linux.PeakBytes < linux.CurrentBytes) {
		t.Fatalf("linux measured footprint = %+v, want positive current and peak >= current", linux)
	}
	if linux.EvidenceClass == MemoryFootprintBlocked &&
		(linux.BlockedReason == "" || linux.CurrentBytes != 0 || linux.PeakBytes != 0) {
		t.Fatalf("linux blocked footprint = %+v, want blocked reason and no zero measured values", linux)
	}
}

func TestParseLinuxProcStatusFootprintRejectsMissingOrZeroValues(t *testing.T) {
	current, peak, err := parseLinuxProcStatusFootprint("VmHWM:\t8 kB\nVmRSS:\t4 kB\n")
	if err != nil {
		t.Fatalf("parse footprint: %v", err)
	}
	if current != 4096 || peak != 8192 {
		t.Fatalf("footprint = %d/%d, want 4096/8192", current, peak)
	}
	if _, _, err := parseLinuxProcStatusFootprint("VmHWM:\t8 kB\n"); err == nil ||
		!strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing VmRSS error = %v", err)
	}
	if _, _, err := parseLinuxProcStatusFootprint("VmHWM:\t0 kB\nVmRSS:\t4 kB\n"); err == nil ||
		!strings.Contains(err.Error(), "positive") {
		t.Fatalf("zero VmHWM error = %v", err)
	}
}
