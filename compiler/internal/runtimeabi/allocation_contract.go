package runtimeabi

import (
	"fmt"
	"sort"
)

type AllocationRuntimePath string

const (
	AllocationPathPlannerSelected        AllocationRuntimePath = "planner_selected"
	AllocationPathHeap                   AllocationRuntimePath = "heap"
	AllocationPathSmallHeapBump          AllocationRuntimePath = "small_heap_bump"
	AllocationPathProcessBumpSmallHeapV0 AllocationRuntimePath = "process_bump_small_heap_v0"
	AllocationPathPerCoreSmallHeap       AllocationRuntimePath = "per_core_small_heap"
	AllocationPathLargeMmap              AllocationRuntimePath = "large_mmap"
	AllocationPathExplicitIsland         AllocationRuntimePath = "explicit_island"
	AllocationPathScopedSingleMappingV0  AllocationRuntimePath = "scoped_single_mapping_v0"
	AllocationPathRegion                 AllocationRuntimePath = "region"
	AllocationPathStackFrame             AllocationRuntimePath = "stack_frame"
	AllocationPathEliminated             AllocationRuntimePath = "eliminated"
	AllocationPathExternal               AllocationRuntimePath = "external"
	AllocationPathUnknown                AllocationRuntimePath = "unknown_conservative"
)

type AllocationZeroSizeBehavior string

const (
	AllocationZeroInvalidPrecondition      AllocationZeroSizeBehavior = "invalid_precondition"
	AllocationZeroCanonicalEmpty           AllocationZeroSizeBehavior = "canonical_empty_no_allocator"
	AllocationZeroCanonicalEmptyNoMetadata AllocationZeroSizeBehavior = "canonical_empty_no_metadata_access"
	AllocationZeroRegionHeaderOnly         AllocationZeroSizeBehavior = "region_header_only"
)

type AllocationGuardBehavior string

const (
	AllocationRejectBeforeAllocator AllocationGuardBehavior = "reject_before_allocator"
	AllocationRejectBeforeMetadata  AllocationGuardBehavior = "reject_before_metadata_access"
)

type AllocationFailureBehavior string

const (
	AllocationFailureTrapOrStatus AllocationFailureBehavior = "trap_or_stable_status"
)

type AllocationDebugInstrumentation string

const (
	AllocationDebugBoundsHeader AllocationDebugInstrumentation = "bounds_header"
	AllocationDebugDoubleFree   AllocationDebugInstrumentation = "double_free"
	AllocationDebugUseAfterFree AllocationDebugInstrumentation = "use_after_free"
	AllocationDebugRegionReset  AllocationDebugInstrumentation = "region_reset"
)

type AllocationReportHook string

const (
	AllocationReportStorageClass          AllocationReportHook = "storage_class"
	AllocationReportRuntimePath           AllocationReportHook = "runtime_path"
	AllocationReportBytesRequested        AllocationReportHook = "bytes_requested"
	AllocationReportBytesReserved         AllocationReportHook = "bytes_reserved"
	AllocationReportAllocatorClass        AllocationReportHook = "allocator_class"
	AllocationReportAllocatorScope        AllocationReportHook = "allocator_scope"
	AllocationReportReusePolicy           AllocationReportHook = "allocator_reuse_policy"
	AllocationReportRegionID              AllocationReportHook = "region_id"
	AllocationReportLifetime              AllocationReportHook = "lifetime"
	AllocationReportDomainID              AllocationReportHook = "domain_id"
	AllocationReportDomainKind            AllocationReportHook = "domain_kind"
	AllocationReportDomainOwner           AllocationReportHook = "domain_owner"
	AllocationReportDomainLifetime        AllocationReportHook = "domain_lifetime"
	AllocationReportDebugMode             AllocationReportHook = "debug_mode"
	AllocationReportRawPointerBounds      AllocationReportHook = "raw_pointer_bounds"
	AllocationReportRawPointerBase        AllocationReportHook = "raw_pointer_base"
	AllocationReportRawPointerSlicePolicy AllocationReportHook = "raw_pointer_slice_policy"
)

type RuntimeAllocationContract struct {
	API                  string                           `json:"api"`
	RuntimePath          AllocationRuntimePath            `json:"runtime_path"`
	AlignmentBytes       int                              `json:"alignment_bytes"`
	ZeroSizeBehavior     AllocationZeroSizeBehavior       `json:"zero_size_behavior"`
	NegativeBehavior     AllocationGuardBehavior          `json:"negative_behavior"`
	OverflowBehavior     AllocationGuardBehavior          `json:"overflow_behavior"`
	FailureBehavior      AllocationFailureBehavior        `json:"failure_behavior"`
	DebugInstrumentation []AllocationDebugInstrumentation `json:"debug_instrumentation,omitempty"`
	ReportHooks          []AllocationReportHook           `json:"report_hooks"`
}

func RuntimeAllocationContracts() []RuntimeAllocationContract {
	contracts := []RuntimeAllocationContract{
		allocBytesContract(),
		makeSliceContract("make_u8"),
		makeSliceContract("make_u16"),
		makeSliceContract("make_i32"),
		makeSliceContract("make_bool"),
		{
			API:              "core.island_new",
			RuntimePath:      AllocationPathExplicitIsland,
			AlignmentBytes:   16,
			ZeroSizeBehavior: AllocationZeroRegionHeaderOnly,
			NegativeBehavior: AllocationRejectBeforeAllocator,
			OverflowBehavior: AllocationRejectBeforeAllocator,
			FailureBehavior:  AllocationFailureTrapOrStatus,
			DebugInstrumentation: []AllocationDebugInstrumentation{
				AllocationDebugDoubleFree,
				AllocationDebugUseAfterFree,
			},
			ReportHooks: RequiredAllocationReportHooks(),
		},
		islandMakeContract("core.island_make_u8"),
		islandMakeContract("core.island_make_u16"),
		islandMakeContract("core.island_make_i32"),
		islandMakeContract("core.island_make_bool"),
		{
			API:              "region.temp",
			RuntimePath:      AllocationPathRegion,
			AlignmentBytes:   16,
			ZeroSizeBehavior: AllocationZeroCanonicalEmpty,
			NegativeBehavior: AllocationRejectBeforeAllocator,
			OverflowBehavior: AllocationRejectBeforeAllocator,
			FailureBehavior:  AllocationFailureTrapOrStatus,
			DebugInstrumentation: []AllocationDebugInstrumentation{
				AllocationDebugRegionReset,
				AllocationDebugUseAfterFree,
			},
			ReportHooks: RequiredAllocationReportHooks(),
		},
	}
	sort.Slice(contracts, func(i, j int) bool { return contracts[i].API < contracts[j].API })
	return contracts
}

func RuntimeAllocationContractForAPI(api string) (RuntimeAllocationContract, bool) {
	for _, contract := range RuntimeAllocationContracts() {
		if contract.API == api {
			return contract, true
		}
	}
	return RuntimeAllocationContract{}, false
}

func RequiredAllocationReportHooks() []AllocationReportHook {
	return []AllocationReportHook{
		AllocationReportStorageClass,
		AllocationReportRuntimePath,
		AllocationReportBytesRequested,
		AllocationReportBytesReserved,
		AllocationReportAllocatorClass,
		AllocationReportAllocatorScope,
		AllocationReportReusePolicy,
		AllocationReportRegionID,
		AllocationReportLifetime,
		AllocationReportDomainID,
		AllocationReportDomainKind,
		AllocationReportDomainOwner,
		AllocationReportDomainLifetime,
		AllocationReportDebugMode,
	}
}

func ValidateRuntimeAllocationContract(contract RuntimeAllocationContract) error {
	if contract.API == "" {
		return fmt.Errorf("runtime allocation contract: api is required")
	}
	if contract.RuntimePath == "" {
		return fmt.Errorf("runtime allocation contract %s: runtime_path is required", contract.API)
	}
	if contract.AlignmentBytes <= 0 || !isPowerOfTwo(contract.AlignmentBytes) {
		return fmt.Errorf(
			"runtime allocation contract %s: alignment must be a positive power-of-two",
			contract.API,
		)
	}
	if contract.ZeroSizeBehavior == "" {
		return fmt.Errorf(
			"runtime allocation contract %s: zero-size behavior is required",
			contract.API,
		)
	}
	if contract.NegativeBehavior == "" {
		return fmt.Errorf(
			"runtime allocation contract %s: negative guard behavior is required",
			contract.API,
		)
	}
	if contract.OverflowBehavior == "" {
		return fmt.Errorf(
			"runtime allocation contract %s: overflow guard behavior is required",
			contract.API,
		)
	}
	if contract.FailureBehavior == "" {
		return fmt.Errorf(
			"runtime allocation contract %s: failure behavior is required",
			contract.API,
		)
	}
	for _, hook := range RequiredAllocationReportHooks() {
		if !contract.HasReportHook(hook) {
			return fmt.Errorf(
				"runtime allocation contract %s: missing report hook %s",
				contract.API,
				hook,
			)
		}
	}
	return nil
}

func (contract RuntimeAllocationContract) HasReportHook(hook AllocationReportHook) bool {
	for _, candidate := range contract.ReportHooks {
		if candidate == hook {
			return true
		}
	}
	return false
}

func (contract RuntimeAllocationContract) HasDebugInstrumentation(
	hook AllocationDebugInstrumentation,
) bool {
	for _, candidate := range contract.DebugInstrumentation {
		if candidate == hook {
			return true
		}
	}
	return false
}

func heapContract(api string, zero AllocationZeroSizeBehavior) RuntimeAllocationContract {
	return RuntimeAllocationContract{
		API:                  api,
		RuntimePath:          AllocationPathHeap,
		AlignmentBytes:       16,
		ZeroSizeBehavior:     zero,
		NegativeBehavior:     AllocationRejectBeforeAllocator,
		OverflowBehavior:     AllocationRejectBeforeAllocator,
		FailureBehavior:      AllocationFailureTrapOrStatus,
		DebugInstrumentation: []AllocationDebugInstrumentation{AllocationDebugBoundsHeader},
		ReportHooks:          RequiredAllocationReportHooks(),
	}
}

func allocBytesContract() RuntimeAllocationContract {
	contract := heapContract("core.alloc_bytes", AllocationZeroInvalidPrecondition)
	contract.ReportHooks = append(contract.ReportHooks,
		AllocationReportRawPointerBounds,
		AllocationReportRawPointerBase,
		AllocationReportRawPointerSlicePolicy,
	)
	return contract
}

func makeSliceContract(api string) RuntimeAllocationContract {
	contract := heapContract(api, AllocationZeroCanonicalEmpty)
	contract.RuntimePath = AllocationPathPlannerSelected
	return contract
}

func islandMakeContract(api string) RuntimeAllocationContract {
	return RuntimeAllocationContract{
		API:              api,
		RuntimePath:      AllocationPathExplicitIsland,
		AlignmentBytes:   16,
		ZeroSizeBehavior: AllocationZeroCanonicalEmptyNoMetadata,
		NegativeBehavior: AllocationRejectBeforeMetadata,
		OverflowBehavior: AllocationRejectBeforeMetadata,
		FailureBehavior:  AllocationFailureTrapOrStatus,
		DebugInstrumentation: []AllocationDebugInstrumentation{
			AllocationDebugDoubleFree,
			AllocationDebugUseAfterFree,
		},
		ReportHooks: RequiredAllocationReportHooks(),
	}
}

func isPowerOfTwo(value int) bool {
	return value > 0 && value&(value-1) == 0
}
