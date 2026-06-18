package runtimeabi

import (
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

func TestRuntimeMemoryBackendContractReportsEvidenceClasses(t *testing.T) {
	linux := RuntimeMemoryBackendContract("linux-x64")
	if linux.FootprintEvidenceClass != MemoryFootprintMeasured || linux.FootprintMethod != "linux_proc_status" {
		t.Fatalf("linux footprint evidence = %q/%q, want measured/linux_proc_status", linux.FootprintEvidenceClass, linux.FootprintMethod)
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
