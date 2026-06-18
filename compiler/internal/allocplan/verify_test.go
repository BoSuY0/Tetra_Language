package allocplan

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/runtimeabi"
)

func TestVerifyPlanRejectsMissingSiteID(t *testing.T) {
	err := VerifyPlan(&Plan{Functions: []FunctionPlan{{
		Name: "bad",
		Allocations: []Allocation{{
			ID:                    "xs",
			ValueID:               "alloc_intent:xs",
			Builtin:               "core.make_u8",
			ElementType:           "u8",
			ElementSize:           1,
			LengthExpr:            "4",
			LengthStatus:          LengthStatusNormal,
			ZeroGuardStatus:       "valid_empty_no_allocator",
			NegativeGuardStatus:   "reject_before_allocation",
			OverflowGuardStatus:   "reject_before_allocation",
			Escape:                EscapeNoEscape,
			Storage:               StorageStack,
			PlannedStorage:        StorageStack,
			ActualLoweringStorage: StorageHeap,
			ValidationStatus:      "validated_no_escape",
			LoweringStatus:        "conservative_heap_fallback",
			Reason:                "test",
		}},
	}}})
	if err == nil || !strings.Contains(err.Error(), "missing stable site id") {
		t.Fatalf("VerifyPlan error = %v, want missing stable site id", err)
	}
}

func TestVerifyPlanRejectsEscapedActualTrustedLowering(t *testing.T) {
	tests := []struct {
		name   string
		escape EscapeClass
		actual StorageClass
	}{
		{name: "returned_stack", escape: EscapeReturn, actual: StorageStack},
		{name: "global_region", escape: EscapeGlobal, actual: StorageRegion},
		{name: "task_region", escape: EscapeTask, actual: StorageTaskRegion},
		{name: "actor_move_region", escape: EscapeActor, actual: StorageActorMoveRegion},
		{name: "unknown_call_stack", escape: EscapeCallUnknown, actual: StorageStack},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			alloc := validVerifierAllocation()
			alloc.Escape = test.escape
			alloc.Storage = StorageHeap
			alloc.PlannedStorage = StorageHeap
			alloc.ActualLoweringStorage = test.actual
			alloc.ValidationStatus = "validated_heap_fallback"
			alloc.LoweringStatus = "storage_lowering"
			alloc.Reason = "escaped allocation should stay on heap"

			err := verifySingleAllocation(alloc)
			if err == nil || !strings.Contains(err.Error(), "actual lowering storage") {
				t.Fatalf(
					"VerifyPlan error = %v, want actual lowering storage escape rejection",
					err,
				)
			}
		})
	}
}

func TestVerifyPlanRejectsTrustedStorageWithoutNoEscapeProof(t *testing.T) {
	tests := []struct {
		name    string
		storage StorageClass
		status  string
	}{
		{name: "stack", storage: StorageStack, status: "validated_heap_fallback"},
		{name: "region", storage: StorageRegion, status: "validated_conservative"},
		{
			name:    "function_temp_region",
			storage: StorageFunctionTempRegion,
			status:  "validated_heap_fallback",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			alloc := validVerifierAllocation()
			alloc.Storage = test.storage
			alloc.PlannedStorage = test.storage
			alloc.ActualLoweringStorage = test.storage
			alloc.ValidationStatus = test.status
			alloc.LoweringStatus = "trusted_storage_lowering"
			alloc.Reason = "trusted storage fixture without a matching no-escape proof"

			err := verifySingleAllocation(alloc)
			if err == nil || !strings.Contains(err.Error(), "no-escape proof") {
				t.Fatalf("VerifyPlan error = %v, want no-escape proof rejection", err)
			}
		})
	}
}

func TestVerifyPlanRejectsHeapFallbackWithoutReason(t *testing.T) {
	alloc := validVerifierAllocation()
	alloc.Escape = EscapeReturn
	alloc.Storage = StorageHeap
	alloc.PlannedStorage = StorageHeap
	alloc.ActualLoweringStorage = StorageHeap
	alloc.ValidationStatus = "validated_heap_fallback"
	alloc.LoweringStatus = "heap_runtime"
	alloc.Reason = ""
	alloc.HeapReasonCodes = []string{"heap.required_escape_return"}
	alloc.ReasonCodes = []string{"heap.required_escape_return"}

	err := verifySingleAllocation(alloc)
	if err == nil || !strings.Contains(err.Error(), "reason") {
		t.Fatalf("VerifyPlan error = %v, want missing reason rejection", err)
	}
}

func TestVerifyPlanRejectsHeapFallbackWithoutReasonCodes(t *testing.T) {
	alloc := validVerifierAllocation()
	alloc.Escape = EscapeReturn
	alloc.Storage = StorageHeap
	alloc.PlannedStorage = StorageHeap
	alloc.ActualLoweringStorage = StorageHeap
	alloc.ValidationStatus = "validated_heap_fallback"
	alloc.LoweringStatus = "heap_runtime"
	alloc.Reason = "returned allocation should stay on heap"

	err := verifySingleAllocation(alloc)
	if err == nil || !strings.Contains(err.Error(), "heap reason code") {
		t.Fatalf("VerifyPlan error = %v, want missing heap reason code rejection", err)
	}
}

func TestVerifyPlanRejectsInvalidMemoryBackendEvidence(t *testing.T) {
	alloc := validVerifierAllocation()
	alloc.MemoryBackend = &runtimeabi.MemoryBackendAllocationEvidence{
		Schema:       runtimeabi.MemoryBackendAllocationEvidenceSchemaV1,
		BackendClass: runtimeabi.MemoryBackendClassSmallHeap,
		Adapter:      "runtime.small_heap.per_core_v1",
		RuntimePath:  runtimeabi.AllocationPathPerCoreSmallHeap,
		Operations: []runtimeabi.MemoryBackendOperation{
			runtimeabi.MemoryBackendReserve,
			runtimeabi.MemoryBackendCommit,
		},
		EvidenceClass: runtimeabi.MemoryFootprintEstimated,
		Method:        "allocation_report_memory_backend_v1",
		ReserveBytes:  32,
		CommitBytes:   64,
	}

	err := verifySingleAllocation(alloc)
	if err == nil || !strings.Contains(err.Error(), "memory backend") {
		t.Fatalf("VerifyPlan error = %v, want memory backend evidence rejection", err)
	}
}

func verifySingleAllocation(alloc Allocation) error {
	return VerifyPlan(&Plan{Functions: []FunctionPlan{{
		Name:        "main",
		Allocations: []Allocation{alloc},
	}}})
}

func validVerifierAllocation() Allocation {
	return Allocation{
		ID:                    "xs",
		SiteID:                "alloc:main:xs",
		ValueID:               "alloc_intent:xs",
		Builtin:               "core.make_u8",
		ElementType:           "u8",
		ElementSize:           1,
		LengthExpr:            "4",
		LengthStatus:          LengthStatusNormal,
		ZeroGuardStatus:       "valid_empty_no_allocator",
		NegativeGuardStatus:   "reject_before_allocation",
		OverflowGuardStatus:   "reject_before_allocation",
		Escape:                EscapeNoEscape,
		Storage:               StorageStack,
		PlannedStorage:        StorageStack,
		ActualLoweringStorage: StorageStack,
		ValidationStatus:      "validated_no_escape",
		LoweringStatus:        "stack_lowering",
		Reason:                "fixed small no-escape allocation",
	}
}
