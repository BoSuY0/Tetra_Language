package allocplan

import (
	"reflect"
	"testing"

	"tetra_language/compiler/internal/runtimeabi"
)

func TestReportSummaryUsesActualLoweredPlan(t *testing.T) {
	domain := runtimeabi.DefaultProcessMemoryDomain(0, 0)
	domain.RequestedBytes = 16
	domain.ReservedBytes = 16
	plan := &Plan{
		Functions: []FunctionPlan{{
			Name: "main",
			Allocations: []Allocation{{
				ID:                    "alloc0",
				SiteID:                "site:main:alloc0",
				ValueID:               "alloc0",
				ByteSize:              16,
				BytesRequested:        16,
				BytesReserved:         16,
				PlannedStorage:        StorageStack,
				ActualLoweringStorage: StorageHeap,
				Domain:                &domain,
			}},
		}},
	}

	summary := Summarize(plan)
	if got := summary.StorageClasses[string(StorageStack)]; got != 1 {
		t.Fatalf("planned storage count = %d, want 1", got)
	}
	if got := summary.ActualLoweringStorageClasses[string(StorageHeap)]; got != 1 {
		t.Fatalf("actual lowering storage count = %d, want 1", got)
	}
	if got := summary.BytesReserved; got != 16 {
		t.Fatalf("bytes reserved = %d, want 16", got)
	}
}

func TestLoweredReportHooksRecordSmallHeapRuntimeEvidenceAfterLowering(t *testing.T) {
	alloc := Allocation{
		ID:                    "xs",
		SiteID:                "site:main:xs",
		ValueID:               "alloc_intent:xs",
		Builtin:               "core.make_u8",
		ElementType:           "u8",
		ElementSize:           1,
		LengthExpr:            "32",
		LengthStatus:          LengthStatusNormal,
		ByteSize:              32,
		Escape:                EscapeReturn,
		Storage:               StorageHeap,
		PlannedStorage:        StorageHeap,
		ActualLoweringStorage: StorageHeap,
		Reason:                "returned allocation needs caller-owned region support before it can avoid heap fallback",
	}

	ApplyLoweredAllocationReportHooksWithOptions(
		&alloc,
		Options{EnableSmallHeapRuntime: true},
	)

	if got := RuntimePathForAllocation(alloc); got != runtimeabi.AllocationPathProcessBumpSmallHeapV0 {
		t.Fatalf("runtime path = %q, want %q", got, runtimeabi.AllocationPathProcessBumpSmallHeapV0)
	}
	if alloc.AllocatorClass != "small_32" ||
		alloc.AllocatorScope != "process" ||
		alloc.AllocatorReusePolicy != "bump_no_reuse_v0" {
		t.Fatalf(
			"allocator evidence = class %q scope %q reuse %q",
			alloc.AllocatorClass,
			alloc.AllocatorScope,
			alloc.AllocatorReusePolicy,
		)
	}
	if alloc.BytesRequested != 32 || alloc.BytesReserved != 32 {
		t.Fatalf(
			"bytes = requested %d reserved %d, want 32/32",
			alloc.BytesRequested,
			alloc.BytesReserved,
		)
	}
	if got := alloc.HeapReasonCodes; !reflect.DeepEqual(got, []string{HeapReasonEscapeReturn}) {
		t.Fatalf("heap reason codes = %+v, want return escape", got)
	}
}

func TestMemoryDomainsDeterministicCopyProjection(t *testing.T) {
	firstDomain := runtimeabi.DefaultProcessMemoryDomain(0, 0)
	firstDomain.RequestedBytes = 4
	firstDomain.ReservedBytes = 8
	secondDomain := runtimeabi.DefaultProcessMemoryDomain(0, 0)
	secondDomain.RequestedBytes = 12
	secondDomain.ReservedBytes = 24
	plan := &Plan{
		Functions: []FunctionPlan{{
			Name: "b",
			Allocations: []Allocation{{
				ID:      "b",
				SiteID:  "site:b",
				ValueID: "b",
				Domain:  &secondDomain,
			}},
		}, {
			Name: "a",
			Allocations: []Allocation{{
				ID:      "a",
				SiteID:  "site:a",
				ValueID: "a",
				Domain:  &firstDomain,
			}},
		}},
	}

	domains := MemoryDomains(plan)
	if len(domains) != 1 {
		t.Fatalf("domains = %d, want 1 merged process domain", len(domains))
	}
	if got, want := domains[0].RequestedBytes, int64(16); got != want {
		t.Fatalf("domain requested bytes = %d, want %d", got, want)
	}
	domains[0].DomainID = "domain:mutated"
	if plan.Functions[0].Allocations[0].Domain.DomainID != "domain:process" {
		t.Fatalf("plan domain changed through projection: %+v", plan.Functions[0].Allocations[0].Domain)
	}
	if next := MemoryDomains(plan); reflect.DeepEqual(domains, next) {
		t.Fatalf("mutated projection matched fresh projection: %+v", next)
	}
}
