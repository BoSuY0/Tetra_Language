package allocplan

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/runtimeabi"
)

type allocationRuntimePath = runtimeabi.AllocationRuntimePath
type memoryBackendEvidence = runtimeabi.MemoryBackendAllocationEvidence
type memoryDomain = runtimeabi.MemoryDomain
type memoryDomainSummary = runtimeabi.MemoryDomainSummary

type Plan struct {
	Functions []FunctionPlan `json:"functions,omitempty"`
	Totals    Totals         `json:"totals"`
}

type ReportSummary struct {
	AllocationCount              int            `json:"allocation_count"`
	StorageClasses               map[string]int `json:"storage_classes"`
	ActualLoweringStorageClasses map[string]int `json:"actual_lowering_storage_classes"`
	RuntimePaths                 map[string]int `json:"runtime_paths"`
	AllocatorClasses             map[string]int `json:"allocator_classes,omitempty"`
	AllocatorScopes              map[string]int `json:"allocator_scopes,omitempty"`
	AllocatorReusePolicies       map[string]int `json:"allocator_reuse_policies,omitempty"`

	MemoryBackendClasses         map[string]int `json:"memory_backend_classes,omitempty"`
	MemoryBackendOperations      map[string]int `json:"memory_backend_operations,omitempty"`
	MemoryBackendEvidenceClasses map[string]int `json:"memory_backend_evidence_classes,omitempty"`
	HeapReasonCodes              map[string]int `json:"heap_reason_codes,omitempty"`

	RawPointerBoundsStatuses map[string]int `json:"raw_pointer_bounds_statuses,omitempty"`
	RawSlicePolicies         map[string]int `json:"raw_slice_policies,omitempty"`
	BytesRequested           int            `json:"bytes_requested"`
	BytesReserved            int            `json:"bytes_reserved"`
	BytesCommitted           int            `json:"bytes_committed,omitempty"`
	BytesReleased            int            `json:"bytes_released,omitempty"`

	Regions []RegionReportSummary `json:"regions,omitempty"`
	Domains []memoryDomainSummary `json:"domains,omitempty"`
}

type RegionReportSummary struct {
	RegionID        string `json:"region_id"`
	Lifetime        string `json:"lifetime,omitempty"`
	StorageClass    string `json:"storage_class"`
	RuntimePath     string `json:"runtime_path"`
	AllocationCount int    `json:"allocation_count"`
	BytesRequested  int    `json:"bytes_requested"`
	BytesReserved   int    `json:"bytes_reserved"`
}

type Options struct {
	EnableStackLowering    bool
	EnableSmallHeapRuntime bool
	EnableRegionPlanning   bool
	EnableRegionLowering   bool
}

type FunctionPlan struct {
	Name        string       `json:"name"`
	Allocations []Allocation `json:"allocations,omitempty"`
}

type Allocation struct {
	ID                    string                 `json:"id"`
	SiteID                string                 `json:"site_id"`
	ValueID               string                 `json:"value_id"`
	Source                string                 `json:"source,omitempty"`
	Builtin               string                 `json:"builtin,omitempty"`
	ElementType           string                 `json:"element_type,omitempty"`
	ElementSize           int                    `json:"element_size,omitempty"`
	LengthExpr            string                 `json:"length_expr,omitempty"`
	LengthStatus          LengthStatus           `json:"length_status,omitempty"`
	ZeroGuardStatus       string                 `json:"zero_guard_status,omitempty"`
	NegativeGuardStatus   string                 `json:"negative_guard_status,omitempty"`
	OverflowGuardStatus   string                 `json:"overflow_guard_status,omitempty"`
	ByteSize              int                    `json:"byte_size,omitempty"`
	Escape                EscapeClass            `json:"escape"`
	Storage               StorageClass           `json:"storage"`
	PlannedStorage        StorageClass           `json:"planned_storage"`
	ActualLoweringStorage StorageClass           `json:"actual_lowering_storage"`
	Reason                string                 `json:"reason"`
	ValidationStatus      string                 `json:"validation_status,omitempty"`
	LoweringStatus        string                 `json:"lowering_status,omitempty"`
	BackendStorage        StorageClass           `json:"backend_storage,omitempty"`
	BackendReason         string                 `json:"backend_reason,omitempty"`
	ReasonCodes           []string               `json:"reason_codes,omitempty"`
	HeapReasonCodes       []string               `json:"heap_reason_codes,omitempty"`
	RuntimePath           allocationRuntimePath  `json:"runtime_path,omitempty"`
	AllocatorClass        string                 `json:"allocator_class,omitempty"`
	AllocatorScope        string                 `json:"allocator_scope,omitempty"`
	AllocatorReusePolicy  string                 `json:"allocator_reuse_policy,omitempty"`
	AllocatorChunkBytes   int                    `json:"allocator_chunk_bytes,omitempty"`
	MemoryBackend         *memoryBackendEvidence `json:"memory_backend,omitempty"`

	RawPointerBoundsStatus string `json:"raw_pointer_bounds_status,omitempty"`
	RawPointerBaseID       string `json:"raw_pointer_base_id,omitempty"`
	RawPointerBaseBytes    int64  `json:"raw_pointer_base_bytes,omitempty"`
	RawPointerOffsetBytes  int64  `json:"raw_pointer_offset_bytes,omitempty"`
	RawSlicePolicy         string `json:"raw_slice_policy,omitempty"`

	BytesRequested int           `json:"bytes_requested"`
	BytesReserved  int           `json:"bytes_reserved"`
	BytesCommitted int           `json:"bytes_committed,omitempty"`
	BytesReleased  int           `json:"bytes_released,omitempty"`
	RegionID       string        `json:"region_id,omitempty"`
	Lifetime       string        `json:"lifetime,omitempty"`
	DebugMode      string        `json:"debug_mode,omitempty"`
	Domain         *memoryDomain `json:"domain,omitempty"`

	ExplicitIslandHandleParamSlotKnown bool `json:"-"`
	ExplicitIslandHandleParamSlot      int  `json:"-"`
}

type Totals struct {
	Eliminated         int `json:"eliminated"`
	Register           int `json:"register"`
	Stack              int `json:"stack"`
	Region             int `json:"region"`
	FunctionTempRegion int `json:"function_temp_region"`
	ExplicitIsland     int `json:"explicit_island"`
	TaskRegion         int `json:"task_region"`
	ActorMoveRegion    int `json:"actor_move_region"`
	Heap               int `json:"heap"`
	MmapLarge          int `json:"mmap_large"`
	External           int `json:"external"`
	Unknown            int `json:"unknown"`
}

type StorageClass string

const (
	StorageEliminated          StorageClass = "Eliminated"
	StorageRegister            StorageClass = "Register"
	StorageStack               StorageClass = "Stack"
	StorageRegion              StorageClass = "Region"
	StorageFunctionTempRegion  StorageClass = "FunctionTempRegion"
	StorageExplicitIsland      StorageClass = "ExplicitIsland"
	StorageTaskRegion          StorageClass = "TaskRegion"
	StorageActorMoveRegion     StorageClass = "ActorMoveRegion"
	StorageHeap                StorageClass = "Heap"
	StorageLargeMmap           StorageClass = "LargeMmap"
	StorageMmapLarge           StorageClass = StorageLargeMmap
	StorageExternal            StorageClass = "External"
	StorageUnknownConservative StorageClass = "UnknownConservative"
	StorageUnknown             StorageClass = StorageUnknownConservative
)

type EscapeClass string

const (
	EscapeNoEscape    EscapeClass = "NoEscape"
	EscapeReturn      EscapeClass = "EscapesReturn"
	EscapeGlobal      EscapeClass = "EscapesGlobal"
	EscapeCallUnknown EscapeClass = "EscapesCallUnknown"
	EscapeActor       EscapeClass = "EscapesActor"
	EscapeTask        EscapeClass = "EscapesTask"
	EscapeUnsafe      EscapeClass = "EscapesUnsafe"
	EscapeClosure     EscapeClass = "EscapesClosure"
	EscapeAggregate   EscapeClass = "EscapesAggregate"
	EscapeUnknown     EscapeClass = "Unknown"
)

type LengthStatus string

const (
	LengthStatusRuntimeGuarded   LengthStatus = "runtime_guarded"
	LengthStatusValidEmpty       LengthStatus = "valid_empty_allocation"
	LengthStatusNormal           LengthStatus = "normal_allocation"
	LengthStatusRejectedNegative LengthStatus = "rejected_negative_length"
	LengthStatusRejectedOverflow LengthStatus = "rejected_byte_size_overflow"
	LengthStatusInvalidContract  LengthStatus = "invalid_length_contract"
)

const (
	HeapReasonEscapeReturn               = "heap.required_escape_return"
	HeapReasonUnknownCall                = "heap.required_unknown_call"
	HeapReasonActorBoundary              = "heap.required_actor_boundary"
	HeapReasonTaskBoundary               = "heap.required_task_boundary"
	HeapReasonDynamicLifetime            = "heap.required_dynamic_lifetime"
	HeapReasonLargeObject                = "heap.required_large_object"
	HeapReasonFFIExternal                = "heap.required_ffi_external"
	HeapReasonBackendLoweringUnavailable = "heap.required_backend_lowering_unavailable"
	HeapReasonRegionLoweringUnavailable  = "heap.required_region_lowering_unavailable"
)

const smallStackAllocationBytes = 4096
const largeI32StackAllocationBytes = 16 * 1024
const scalarReplacementMaxElements int64 = 2
const maxAllocationByteSize int64 = 1<<31 - 1

func FromPLIR(prog *plir.Program) (*Plan, error) {
	return FromPLIRWithOptions(prog, Options{})
}

func FromPLIRWithOptions(prog *plir.Program, opt Options) (*Plan, error) {
	if prog == nil {
		return nil, fmt.Errorf("allocplan: missing PLIR program")
	}
	plan := &Plan{}
	callSummaries := buildReadOnlyCallSummaries(prog)
	for _, fn := range prog.Funcs {
		row := FunctionPlan{Name: fn.Name}
		values := append([]plir.Value(nil), fn.Values...)
		sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
		functionTempRegionUsed := false
		for _, value := range values {
			if value.Kind != plir.ValueAllocIntent || value.Alloc == nil {
				continue
			}
			valueOpt := opt
			if functionTempRegionUsed {
				valueOpt.EnableRegionPlanning = false
				valueOpt.EnableRegionLowering = false
			}
			alloc := planAllocation(fn, value, valueOpt, callSummaries)
			if alloc.Storage == StorageFunctionTempRegion {
				functionTempRegionUsed = true
			}
			row.Allocations = append(row.Allocations, alloc)
			plan.Totals.add(alloc.Storage)
		}
		if len(row.Allocations) > 0 {
			plan.Functions = append(plan.Functions, row)
		}
	}
	if err := VerifyPlan(plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func planAllocation(
	fn plir.Function,
	value plir.Value,
	opt Options,
	callSummaries map[string]readOnlyCallSummary,
) Allocation {
	id := allocationName(value.ID)
	escape, reason := classifyEscape(fn, id, value, callSummaries)
	storage, storageReason := chooseStorage(value, escape, reason, opt)
	lengthStatus := classifyLengthStatus(value.Alloc)
	scalarReplacement := false
	unusedCopy := false
	if escape == EscapeNoEscape {
		var scalarReason string
		scalarReplacement, scalarReason = isScalarReplacementCandidate(fn, value, id)
		if scalarReplacement {
			storage = StorageEliminated
			storageReason = scalarReason
		}
	}
	if isUnusedCopyAllocation(fn, value, id) {
		unusedCopy = true
		escape = EscapeNoEscape
		storage = StorageEliminated
		storageReason = ("copy result is unused in the supported v0 intra-function " +
			"scan; allocation intent can be elided")
	}
	if reason != "" {
		storageReason = storageReason + "; " + reason
	}
	if lengthStatus == LengthStatusValidEmpty {
		storageReason = storageReason + ("; valid empty allocation has no allocator access where the " +
			"backend implements the contract")
	}
	if lengthStatus == LengthStatusRejectedNegative {
		storageReason = storageReason + "; negative length is rejected before allocation"
	}
	if lengthStatus == LengthStatusRejectedOverflow {
		storageReason = storageReason + "; byte-size overflow is rejected before allocation"
	}
	byteSize := constantByteSize(value.Alloc)
	actualStorage, loweringStatus, backendReason := actualLoweringStorage(
		value,
		storage,
		lengthStatus,
		opt,
		scalarReplacement,
		unusedCopy,
	)
	alloc := Allocation{
		ID:                     id,
		SiteID:                 allocationSiteID(fn.Name, id, value.Source),
		ValueID:                value.ID,
		Source:                 value.Source,
		Builtin:                value.Alloc.Builtin,
		ElementType:            value.Alloc.ElementType,
		ElementSize:            value.Alloc.ElementSize,
		LengthExpr:             value.Alloc.LengthExpr,
		LengthStatus:           lengthStatus,
		ZeroGuardStatus:        value.Alloc.ZeroGuardStatus,
		NegativeGuardStatus:    value.Alloc.NegativeGuardStatus,
		OverflowGuardStatus:    value.Alloc.OverflowGuardStatus,
		ByteSize:               byteSize,
		Escape:                 escape,
		Storage:                storage,
		PlannedStorage:         storage,
		ActualLoweringStorage:  actualStorage,
		Reason:                 storageReason,
		ValidationStatus:       validationStatus(escape, storage, lengthStatus),
		LoweringStatus:         loweringStatus,
		RawPointerBoundsStatus: value.Alloc.RawPointerBoundsStatus,
		RawPointerBaseID:       value.Alloc.RawPointerBaseID,
		RawPointerBaseBytes:    value.Alloc.RawPointerBaseBytes,
		RawPointerOffsetBytes:  value.Alloc.RawPointerOffsetBytes,
		RawSlicePolicy:         value.Alloc.RawSlicePolicy,
	}
	if actualStorage != storage {
		alloc.BackendStorage = actualStorage
		alloc.BackendReason = backendReason
	}
	if actualStorage == StorageExplicitIsland {
		applyRegionAllocatorEvidence(&alloc, value, byteSize)
		if slot, ok := explicitIslandHandleParamSlot(fn, value); ok {
			alloc.ExplicitIslandHandleParamSlotKnown = true
			alloc.ExplicitIslandHandleParamSlot = slot
		}
	}
	if storage == StorageFunctionTempRegion {
		applyPlannedRegionAllocatorEvidence(&alloc, fn, byteSize)
	}
	if opt.EnableSmallHeapRuntime && actualStorage == StorageHeap &&
		storage != StorageFunctionTempRegion &&
		lengthStatus == LengthStatusNormal {
		applyRuntimeAllocatorEvidence(&alloc, byteSize)
	}
	applyDefaultAllocationReportHooks(&alloc)
	applyMemoryBackendEvidence(&alloc)
	applyHeapReasonCodeEvidence(&alloc)
	return alloc
}

func applyHeapReasonCodeEvidence(alloc *Allocation) {
	if alloc == nil || !allocationUsesHeap(*alloc) {
		return
	}
	codes := heapReasonCodesForAllocation(*alloc)
	alloc.HeapReasonCodes = appendReasonCodes(nil, codes...)
	alloc.ReasonCodes = appendReasonCodes(alloc.ReasonCodes, codes...)
}

func allocationUsesHeap(alloc Allocation) bool {
	if alloc.Storage == StorageHeap || alloc.PlannedStorage == StorageHeap ||
		alloc.ActualLoweringStorage == StorageHeap {
		return true
	}
	switch RuntimePathForAllocation(alloc) {
	case runtimeabi.AllocationPathHeap,
		runtimeabi.AllocationPathProcessBumpSmallHeapV0,
		runtimeabi.AllocationPathPerCoreSmallHeap,
		runtimeabi.AllocationPathLargeMmap:
		return true
	default:
		return false
	}
}

func heapReasonCodesForAllocation(alloc Allocation) []string {
	var codes []string
	switch alloc.Escape {
	case EscapeReturn:
		codes = append(codes, HeapReasonEscapeReturn)
	case EscapeCallUnknown:
		codes = append(codes, HeapReasonUnknownCall)
	case EscapeActor:
		codes = append(codes, HeapReasonActorBoundary)
	case EscapeTask:
		codes = append(codes, HeapReasonTaskBoundary)
	case EscapeUnsafe:
		codes = append(codes, HeapReasonFFIExternal)
	case EscapeGlobal, EscapeClosure, EscapeAggregate, EscapeUnknown:
		codes = append(codes, HeapReasonDynamicLifetime)
	}
	if alloc.PlannedStorage != StorageHeap && alloc.ActualLoweringStorage == StorageHeap {
		switch alloc.LoweringStatus {
		case "region_planned_heap_fallback":
			codes = append(codes, HeapReasonRegionLoweringUnavailable)
		case "conservative_heap_fallback":
			codes = append(codes, HeapReasonBackendLoweringUnavailable)
		}
	}
	if alloc.RuntimePath == runtimeabi.AllocationPathLargeMmap ||
		alloc.ByteSize > smallStackAllocationBytes {
		codes = append(codes, HeapReasonLargeObject)
	}
	if len(codes) == 0 {
		codes = append(codes, HeapReasonDynamicLifetime)
	}
	return appendReasonCodes(nil, codes...)
}

func appendReasonCodes(existing []string, codes ...string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(existing)+len(codes))
	for _, code := range existing {
		code = strings.TrimSpace(code)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		out = append(out, code)
	}
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}

func applyDefaultAllocationReportHooks(alloc *Allocation) {
	if alloc == nil {
		return
	}
	if alloc.RuntimePath == "" {
		alloc.RuntimePath = RuntimePathForAllocation(*alloc)
	}
	if alloc.ByteSize > 0 && alloc.BytesRequested == 0 {
		alloc.BytesRequested = alloc.ByteSize
	}
	if alloc.ByteSize > 0 && alloc.BytesReserved == 0 {
		switch alloc.ActualLoweringStorage {
		case StorageEliminated:
			alloc.BytesReserved = 0
		case StorageRegion, StorageFunctionTempRegion, StorageExplicitIsland:
			if reserved, ok := runtimeabi.AlignRegionBytes(int64(alloc.ByteSize)); ok {
				alloc.BytesReserved = int(reserved)
			}
		default:
			alloc.BytesReserved = alloc.ByteSize
		}
	}
	if alloc.Domain == nil {
		alloc.Domain = allocationMemoryDomain(*alloc)
	}
}

func applyMemoryBackendEvidence(alloc *Allocation) {
	if alloc == nil {
		return
	}
	runtimePath := RuntimePathForAllocation(*alloc)
	alloc.RuntimePath = runtimePath
	reserved := int64(allocationReportBytesReserved(*alloc))
	evidence := runtimeabi.MemoryBackendAllocationEvidence{
		Schema:      runtimeabi.MemoryBackendAllocationEvidenceSchemaV1,
		RuntimePath: runtimePath,
	}
	switch runtimePath {
	case runtimeabi.AllocationPathProcessBumpSmallHeapV0:
		evidence.BackendClass = runtimeabi.MemoryBackendClassSmallHeap
		evidence.Adapter = "runtime.small_heap.process_bump_v0"
		evidence.EvidenceClass = runtimeabi.MemoryFootprintEstimated
		evidence.Method = "allocation_report_memory_backend_v1"
		evidence.Operations = retainedMemoryBackendOperations()
		evidence.ReserveBytes = reserved
		evidence.CommitBytes = reserved
		evidence.FootprintCurrentBytes = reserved
		evidence.FootprintPeakBytes = reserved
	case runtimeabi.AllocationPathPerCoreSmallHeap:
		evidence.BackendClass = runtimeabi.MemoryBackendClassSmallHeap
		evidence.Adapter = "runtime.small_heap.per_core_v1"
		evidence.EvidenceClass = runtimeabi.MemoryFootprintEstimated
		evidence.Method = "allocation_report_memory_backend_v1"
		evidence.Operations = estimatedMemoryBackendOperations()
		evidence.ReserveBytes = reserved
		evidence.CommitBytes = reserved
		evidence.ReleaseBytes = reserved
		evidence.FootprintCurrentBytes = reserved
		evidence.FootprintPeakBytes = reserved
	case runtimeabi.AllocationPathLargeMmap:
		evidence.BackendClass = runtimeabi.MemoryBackendClassLargeBackend
		evidence.Adapter = "target.large_mmap_v1"
		evidence.EvidenceClass = runtimeabi.MemoryFootprintEstimated
		evidence.Method = "allocation_report_memory_backend_v1"
		evidence.Operations = retainedMemoryBackendOperations()
		evidence.ReserveBytes = reserved
		evidence.CommitBytes = reserved
		evidence.FootprintCurrentBytes = reserved
		evidence.FootprintPeakBytes = reserved
	case runtimeabi.AllocationPathExplicitIsland,
		runtimeabi.AllocationPathScopedSingleMappingV0,
		runtimeabi.AllocationPathRegion:
		evidence.BackendClass = runtimeabi.MemoryBackendClassRegion
		evidence.Adapter = "runtime.region_bump_v1"
		evidence.EvidenceClass = runtimeabi.MemoryFootprintEstimated
		evidence.Method = "allocation_report_memory_backend_v1"
		evidence.Operations = estimatedMemoryBackendOperations()
		evidence.ReserveBytes = reserved
		evidence.CommitBytes = reserved
		evidence.ReleaseBytes = reserved
		evidence.FootprintCurrentBytes = reserved
		evidence.FootprintPeakBytes = reserved
	case runtimeabi.AllocationPathStackFrame, runtimeabi.AllocationPathEliminated:
		evidence.BackendClass = runtimeabi.MemoryBackendClassNone
		evidence.Adapter = "no_runtime_memory_backend"
		evidence.EvidenceClass = runtimeabi.MemoryFootprintUnsupported
		evidence.Method = "no_runtime_memory_backend"
		evidence.UnsupportedReason = ("stack/register/eliminated storage does not use the runtime " +
			"MemoryBackend")
	case runtimeabi.AllocationPathExternal:
		evidence.BackendClass = runtimeabi.MemoryBackendClassExternal
		evidence.Adapter = "external_owner_memory"
		evidence.EvidenceClass = runtimeabi.MemoryFootprintUnsupported
		evidence.Method = "external_owner_memory"
		evidence.UnsupportedReason = "external storage is owned outside the Tetra runtime MemoryBackend"
	case runtimeabi.AllocationPathHeap:
		evidence.BackendClass = runtimeabi.MemoryBackendClassConservativeHeap
		evidence.Adapter = "runtime.heap_conservative"
		evidence.EvidenceClass = runtimeabi.MemoryFootprintBlocked
		evidence.Method = "allocator_backend_not_enabled"
		evidence.BlockedReason = "heap path has no MemoryBackend adapter evidence in this build"
	default:
		evidence.BackendClass = runtimeabi.MemoryBackendClassUnknown
		evidence.Adapter = "unknown_memory_backend"
		evidence.EvidenceClass = runtimeabi.MemoryFootprintBlocked
		evidence.Method = "unknown_memory_backend"
		evidence.BlockedReason = "runtime path does not map to a MemoryBackend evidence producer"
	}
	alloc.MemoryBackend = &evidence
	if evidence.EvidenceClass == runtimeabi.MemoryFootprintEstimated ||
		evidence.EvidenceClass == runtimeabi.MemoryFootprintMeasured {
		alloc.BytesCommitted = int(evidence.CommitBytes)
		alloc.BytesReleased = int(evidence.ReleaseBytes)
	}
	if alloc.Domain != nil {
		alloc.Domain.CommittedBytes = int64(alloc.BytesCommitted)
		alloc.Domain.ReleasedBytes = int64(alloc.BytesReleased)
		alloc.Domain.CurrentBytes = evidence.FootprintCurrentBytes
		alloc.Domain.PeakBytes = evidence.FootprintPeakBytes
	}
}

func estimatedMemoryBackendOperations() []runtimeabi.MemoryBackendOperation {
	return []runtimeabi.MemoryBackendOperation{
		runtimeabi.MemoryBackendReserve,
		runtimeabi.MemoryBackendCommit,
		runtimeabi.MemoryBackendRelease,
		runtimeabi.MemoryBackendFootprint,
	}
}

func retainedMemoryBackendOperations() []runtimeabi.MemoryBackendOperation {
	return []runtimeabi.MemoryBackendOperation{
		runtimeabi.MemoryBackendReserve,
		runtimeabi.MemoryBackendCommit,
		runtimeabi.MemoryBackendFootprint,
	}
}

func allocationMemoryDomain(alloc Allocation) *runtimeabi.MemoryDomain {
	requested := int64(allocationReportBytesRequested(alloc))
	reserved := int64(allocationReportBytesReserved(alloc))
	var domain runtimeabi.MemoryDomain
	switch alloc.ActualLoweringStorage {
	case StorageExplicitIsland:
		domain = runtimeabi.IslandMemoryDomain(alloc.RegionID, alloc.Lifetime, requested, reserved)
	case StorageExternal:
		domain = runtimeabi.ExternalMemoryDomain(
			alloc.RegionID,
			alloc.Lifetime,
			requested,
			reserved,
		)
	default:
		domain = runtimeabi.DefaultProcessMemoryDomain(requested, reserved)
	}
	return &domain
}

func RuntimePathForAllocation(alloc Allocation) runtimeabi.AllocationRuntimePath {
	if alloc.RuntimePath != "" {
		return alloc.RuntimePath
	}
	switch alloc.ActualLoweringStorage {
	case StorageEliminated:
		return runtimeabi.AllocationPathEliminated
	case StorageStack, StorageRegister:
		return runtimeabi.AllocationPathStackFrame
	case StorageRegion, StorageFunctionTempRegion, StorageTaskRegion, StorageActorMoveRegion:
		return runtimeabi.AllocationPathScopedSingleMappingV0
	case StorageExplicitIsland:
		return runtimeabi.AllocationPathExplicitIsland
	case StorageLargeMmap:
		return runtimeabi.AllocationPathLargeMmap
	case StorageHeap:
		return runtimeabi.AllocationPathHeap
	case StorageExternal:
		return runtimeabi.AllocationPathExternal
	default:
		return runtimeabi.AllocationPathUnknown
	}
}

func applyPlannedRegionAllocatorEvidence(alloc *Allocation, fn plir.Function, byteSize int) {
	if alloc == nil {
		return
	}
	alloc.RegionID = "region:" + fn.Name + ":temp"
	alloc.Lifetime = "function:" + fn.Name
	alloc.DebugMode = "region_reset_when_enabled"
	if alloc.ActualLoweringStorage == StorageFunctionTempRegion {
		alloc.RuntimePath = runtimeabi.AllocationPathScopedSingleMappingV0
		alloc.AllocatorClass = "function_temp_region"
	}
	if alloc.ActualLoweringStorage == StorageFunctionTempRegion && byteSize > 0 {
		if reserved, ok := runtimeabi.AlignRegionBytes(int64(byteSize)); ok {
			alloc.BytesRequested = byteSize
			alloc.BytesReserved = int(reserved)
		}
	}
	if alloc.ActualLoweringStorage == StorageFunctionTempRegion {
		alloc.Reason = alloc.Reason + ("; P15.0 function-local temporary region lowers through " +
			"region enter/reset IR")
	} else {
		alloc.Reason = alloc.Reason + ("; P5.3 planned function-local temporary region; current " +
			"backend still reports heap fallback until implicit region " +
			"lowering lands")
	}
}

func applyRegionAllocatorEvidence(alloc *Allocation, value plir.Value, byteSize int) {
	if alloc == nil {
		return
	}
	alloc.RuntimePath = runtimeabi.AllocationPathExplicitIsland
	alloc.RegionID = value.Region
	alloc.Lifetime = allocationLifetime(value)
	alloc.DebugMode = "double_free_and_use_after_free_when_enabled"
	if byteSize > 0 {
		reserved, ok := runtimeabi.AlignRegionBytes(int64(byteSize))
		if ok {
			alloc.AllocatorClass = "region_bump_16"
			alloc.BytesRequested = byteSize
			alloc.BytesReserved = int(reserved)
		}
	}
	alloc.Reason = alloc.Reason + ("; P5.2 region allocator uses aligned bump allocation with " +
		"bulk free at island scope exit")
}

func allocationLifetime(value plir.Value) string {
	if value.Region != "" && strings.HasPrefix(value.Region, "island:") {
		return value.Region + ":scope"
	}
	if value.Lifetime.Birth != "" && value.Lifetime.Death != "" {
		return value.Lifetime.Birth + ".." + value.Lifetime.Death
	}
	if value.Lifetime.Owner != "" {
		return "owner:" + value.Lifetime.Owner
	}
	return "function_scope"
}

func explicitIslandHandleParamSlot(fn plir.Function, value plir.Value) (int, bool) {
	if value.Provenance.Kind != plir.ProvenanceIsland {
		return 0, false
	}
	root := strings.TrimSpace(strings.TrimPrefix(value.Region, "island:"))
	if root == "" || root == value.Region {
		root = strings.TrimSpace(value.Provenance.Root)
	}
	if root == "" || root == "?" || root == "island" || strings.Contains(root, ".") {
		return 0, false
	}
	if fn.Summary == nil {
		return 0, false
	}
	slot := 0
	for i, name := range fn.Summary.ParamNames {
		typeName := ""
		if i < len(fn.Summary.ParamTypes) {
			typeName = fn.Summary.ParamTypes[i]
		}
		if name == root {
			if typeName != "island" {
				return 0, false
			}
			return slot, true
		}
		slot += paramTypeSlotCount(typeName)
	}
	return 0, false
}

func paramTypeSlotCount(typeName string) int {
	switch {
	case typeName == "":
		return 1
	case typeName == "str", typeName == "String":
		return 2
	case strings.HasPrefix(typeName, "[]"):
		return 2
	case strings.HasPrefix(typeName, "fn("), typeName == "fnptr":
		return 9
	default:
		return 1
	}
}

func applyRuntimeAllocatorEvidence(alloc *Allocation, byteSize int) {
	if alloc == nil || byteSize <= 0 {
		return
	}
	alloc.BytesRequested = byteSize
	if cls, ok := runtimeabi.SmallHeapClassForBytes(int64(byteSize)); ok {
		alloc.RuntimePath = runtimeabi.AllocationPathProcessBumpSmallHeapV0
		alloc.AllocatorClass = cls.Name
		alloc.AllocatorScope = "process"
		alloc.AllocatorReusePolicy = "bump_no_reuse_v0"
		alloc.AllocatorChunkBytes = runtimeabi.SmallHeapChunkBytes
		alloc.BytesReserved = cls.MaxBytes
		alloc.LoweringStatus = "process_bump_small_heap_v0_runtime"
		alloc.Reason = alloc.Reason + ("; current emitted linux-x64 runtime allocator uses a " +
			"process-global bump small heap with 16-byte size classes and 64KiB chunk refills; " +
			"free-list reuse/reclamation is not claimed")
		return
	}
	alloc.RuntimePath = runtimeabi.AllocationPathLargeMmap
	alloc.AllocatorClass = "large_mmap"
	alloc.BytesReserved = byteSize
	alloc.LoweringStatus = "large_mmap_runtime"
	alloc.Reason = alloc.Reason + ("; P5.1 runtime allocator routes this large allocation to " +
		"mmap fallback")
}

func actualLoweringStorage(
	value plir.Value,
	planned StorageClass,
	lengthStatus LengthStatus,
	opt Options,
	scalarReplacement bool,
	unusedCopy bool,
) (StorageClass, string, string) {
	if value.Provenance.Kind == plir.ProvenanceIsland || planned == StorageExplicitIsland {
		return StorageExplicitIsland, "explicit_island_lowering", (("explicit island allocation " +
			"remains region-backed in the ") +
			"current backend")
	}
	switch planned {
	case StorageEliminated:
		if opt.EnableStackLowering && unusedCopy {
			return StorageEliminated, "eliminated_unused_copy", (("unused copy lowers to source " +
				"evaluation plus an empty local ") +
				"header with no fresh storage")
		}
		if opt.EnableStackLowering && scalarReplacement {
			return StorageEliminated, "scalar_replacement", (("fixed tiny no-escape allocation " +
				"lowers to scalar locals ") +
				"with no slice backing storage")
		}
		if opt.EnableStackLowering && lengthStatus == LengthStatusValidEmpty {
			return StorageEliminated, "eliminated_no_backing_storage", ("valid empty allocation " +
				"intent lowers without backing storage")
		}
	case StorageStack:
		if opt.EnableStackLowering {
			return StorageStack, "stack_lowering", (("fixed small no-escape allocation lowers to " +
				"stack frame ") +
				"storage")
		}
	case StorageHeap:
		return StorageHeap, "heap_runtime", ("planned heap allocation lowers through the conservative " +
			"heap/runtime path")
	case StorageFunctionTempRegion:
		if opt.EnableRegionLowering {
			return StorageFunctionTempRegion, "function_temp_region_lowering", (("function-local " +
				"temporary copy lowers through function-temp ") +
				"region enter/reset IR")
		}
		return StorageHeap, "region_planned_heap_fallback", (("P5.3 models a bounded function-" +
			"local region, but implicit ") +
			"region runtime lowering is not enabled in the current " +
			"backend")
	case StorageExternal:
		return StorageExternal, "external_no_lowering", ("external storage is supplied outside " +
			"the allocator")
	case StorageUnknownConservative:
		return StorageUnknownConservative, "unknown_conservative", "unknown storage remains conservative"
	default:
		return StorageHeap, "conservative_heap_fallback", (("planner v0 records the narrower " +
			"storage class; current ") +
			"stack backend still lowers make through the conservative " +
			"heap/runtime path")
	}
	return StorageHeap, "conservative_heap_fallback", (("planner v0 records the narrower " +
		"storage class; current ") +
		"stack backend still lowers make through the conservative " +
		"heap/runtime path")
}

func validationStatus(escape EscapeClass, storage StorageClass, lengthStatus LengthStatus) string {
	switch storage {
	case StorageEliminated:
		if lengthStatus == LengthStatusValidEmpty {
			return "validated_empty_no_backing"
		}
		if escape == EscapeNoEscape {
			return "validated_no_escape"
		}
		return "invalid_escape_for_storage"
	case StorageStack, StorageRegister:
		if escape == EscapeNoEscape {
			return "validated_no_escape"
		}
		return "invalid_escape_for_storage"
	case StorageRegion:
		if escape == EscapeNoEscape {
			return "validated_region_scope"
		}
		return "invalid_region_escape"
	case StorageFunctionTempRegion:
		if escape == EscapeNoEscape {
			return "validated_function_temp_region_scope"
		}
		return "invalid_function_temp_region_escape"
	case StorageExplicitIsland:
		if escape == EscapeNoEscape {
			return "validated_explicit_island_scope"
		}
		return "invalid_island_escape"
	case StorageHeap:
		return "validated_heap_fallback"
	default:
		return "validated_conservative"
	}
}

func isUnusedCopyAllocation(fn plir.Function, value plir.Value, allocName string) bool {
	if value.Alloc == nil || !isCopyBuiltin(value.Alloc.Builtin) || allocName == "" ||
		allocName == "$return" {
		return false
	}
	for _, op := range fn.Ops {
		for _, input := range op.Inputs {
			if allocationInputUses(input, allocName) {
				return false
			}
		}
	}
	return true
}

func isCopyBuiltin(name string) bool {
	return name == "core.string_copy" ||
		(strings.HasPrefix(name, "core.slice_copy_") && !strings.HasPrefix(name, "core.slice_copy_into_"))
}

func isScalarReplacementCandidate(
	fn plir.Function,
	value plir.Value,
	allocName string,
) (bool, string) {
	if value.Alloc == nil || allocName == "" {
		return false, ""
	}
	makeCandidate := isMakeSliceBuiltin(value.Alloc.Builtin)
	copyCandidate := isCopyBuiltin(value.Alloc.Builtin)
	if !makeCandidate && !copyCandidate {
		return false, ""
	}
	length, ok := allocationLengthConst(value.Alloc)
	if !ok || length <= 0 || length > scalarReplacementMaxElements {
		return false, ""
	}
	bytes := constantByteSize(value.Alloc)
	if bytes <= 0 || bytes > smallStackAllocationBytes {
		return false, ""
	}
	hasIndexUse := false
	for _, op := range fn.Ops {
		if !operationInputsUseAllocation(op.Inputs, allocName) {
			continue
		}
		switch op.Kind {
		case plir.OpIndexLoad, plir.OpIndexStore:
			index, ok := constantIndexForAllocationUse(op.Inputs, allocName)
			if !ok || index < 0 || index >= length {
				return false, ""
			}
			hasIndexUse = true
		case plir.OpReturn:
			if !allInputsAreAllocationLen(op.Inputs, allocName) {
				return false, ""
			}
		default:
			return false, ""
		}
	}
	if !hasIndexUse {
		return false, ""
	}
	if copyCandidate {
		return true, fmt.Sprintf(
			("scalar_replacement_copy_fixed_constant_indices: fixed tiny " +
				"no-escape copy has %d elements and every indexed use is " +
				"constant in range"),
			length,
		)
	}
	return true, fmt.Sprintf(
		("scalar_replacement_fixed_constant_indices: fixed tiny " +
			"no-escape allocation has %d elements and every indexed use " +
			"is constant in range"),
		length,
	)
}

func isMakeSliceBuiltin(name string) bool {
	switch name {
	case "core.make_u8", "core.make_u16", "core.make_i32", "core.make_bool":
		return true
	default:
		return false
	}
}

func operationInputsUseAllocation(inputs []string, allocName string) bool {
	for _, input := range inputs {
		if allocationInputUses(input, allocName) {
			return true
		}
	}
	return false
}

func constantIndexForAllocationUse(inputs []string, allocName string) (int64, bool) {
	for i, input := range inputs {
		if !allocationInputUses(input, allocName) {
			continue
		}
		if i+1 >= len(inputs) {
			return 0, false
		}
		index, err := strconv.ParseInt(inputs[i+1], 10, 64)
		return index, err == nil
	}
	return 0, false
}

func allInputsAreAllocationLen(inputs []string, allocName string) bool {
	if len(inputs) == 0 {
		return true
	}
	for _, input := range inputs {
		if input != allocName+".len" {
			return false
		}
	}
	return true
}

func allocationInputUses(input string, allocName string) bool {
	return input == allocName || strings.HasPrefix(input, allocName+".")
}

type readOnlyCallSummary struct {
	Params            map[int]bool
	InoutWriterParams map[int]bool
}

func buildReadOnlyCallSummaries(prog *plir.Program) map[string]readOnlyCallSummary {
	if prog == nil {
		return nil
	}
	out := map[string]readOnlyCallSummary{}
	for _, fn := range prog.Funcs {
		summary := readOnlyCallSummary{Params: map[int]bool{}, InoutWriterParams: map[int]bool{}}
		if fn.Summary == nil || fn.Summary.Async || fn.Summary.TouchesMutableGlobals {
			continue
		}
		for i, name := range fn.Summary.ParamNames {
			if !isMemoryBearingParamType(fn.Summary, i) {
				continue
			}
			if parameterHasReadOnlyNoEscapeUse(fn, name) {
				summary.Params[i] = true
			}
			if parameterHasInoutWriterNoEscapeUse(fn, i, name) {
				summary.InoutWriterParams[i] = true
			}
		}
		if len(summary.Params) > 0 || len(summary.InoutWriterParams) > 0 {
			out[fn.Name] = summary
		}
	}
	return out
}

func isMemoryBearingParamType(summary *plir.FunctionSummary, index int) bool {
	if summary == nil || index < 0 || index >= len(summary.ParamTypes) {
		return false
	}
	typeName := strings.TrimSpace(summary.ParamTypes[index])
	return strings.HasPrefix(typeName, "[]") || typeName == "str" || typeName == "String"
}

func parameterHasReadOnlyNoEscapeUse(fn plir.Function, paramName string) bool {
	if strings.TrimSpace(paramName) == "" {
		return false
	}
	carriers := allocationCarriers(fn, paramName, string(plir.ValueParam)+":"+paramName)
	for _, op := range fn.Ops {
		if op.Kind == plir.OpUnsafe {
			return false
		}
		if !opUsesCarrier(op, carriers) {
			continue
		}
		switch op.Kind {
		case plir.OpAssign, plir.OpGuard, plir.OpIndexLoad, plir.OpSliceWindow:
			continue
		case plir.OpCall:
			if isNonEscapingBuiltinCall(op.Note) {
				continue
			}
			return false
		default:
			return false
		}
	}
	return true
}

func parameterHasInoutWriterNoEscapeUse(fn plir.Function, index int, paramName string) bool {
	if fn.Summary == nil || !isMemoryBearingParamType(fn.Summary, index) ||
		!summaryParamOwnershipIs(fn.Summary, index, "inout") {
		return false
	}
	if strings.TrimSpace(paramName) == "" {
		return false
	}
	carriers := allocationCarriers(fn, paramName, string(plir.ValueParam)+":"+paramName)
	sawCarrierUse := false
	for _, op := range fn.Ops {
		if op.Kind == plir.OpUnsafe {
			return false
		}
		if !opUsesCarrier(op, carriers) {
			continue
		}
		sawCarrierUse = true
		switch op.Kind {
		case plir.OpAssign, plir.OpGuard, plir.OpIndexLoad, plir.OpSliceWindow:
			continue
		case plir.OpIndexStore:
			if indexStoreUsesCarrierOnlyAsBase(op, carriers) {
				continue
			}
			return false
		case plir.OpCall:
			if isNonEscapingBuiltinCall(op.Note) {
				continue
			}
			return false
		default:
			return false
		}
	}
	return sawCarrierUse
}

func summaryParamOwnershipIs(summary *plir.FunctionSummary, index int, want string) bool {
	if summary == nil || index < 0 || index >= len(summary.ParamOwnership) {
		return false
	}
	return strings.TrimSpace(summary.ParamOwnership[index]) == want
}

func indexStoreUsesCarrierOnlyAsBase(op plir.Operation, carriers map[string]bool) bool {
	if len(op.Inputs) == 0 || !inputCarriesAny(op.Inputs[0], carriers) {
		return false
	}
	for _, input := range op.Inputs[1:] {
		if inputCarriesAny(input, carriers) {
			return false
		}
	}
	return true
}

func classifyEscape(
	fn plir.Function,
	allocName string,
	value plir.Value,
	callSummaries map[string]readOnlyCallSummary,
) (EscapeClass, string) {
	if value.Provenance.Kind == plir.ProvenanceIsland {
		return EscapeNoEscape, "explicit island allocation is bounded by the island scope"
	}
	if allocName == "$return" {
		return EscapeReturn, "allocation is returned directly"
	}
	carriers := allocationCarriers(fn, allocName, value.ID)
	unsafeBoundary := false
	aggregateBoundary := false
	closureBoundary := false
	readOnlyCallSummaryProof := false
	inoutWriterCallSummaryProof := false
	for _, op := range fn.Ops {
		if opUsesCarrier(op, carriers) {
			switch op.Kind {
			case plir.OpReturn:
				return EscapeReturn, "allocation value is returned from the function"
			case plir.OpGlobalStore:
				return EscapeGlobal, "allocation value is stored in global state"
			case plir.OpClosure:
				closureBoundary = true
			case plir.OpAggregate:
				if contains(op.Outputs, "$return") {
					return EscapeReturn, "allocation is embedded in a returned aggregate"
				}
				aggregateBoundary = true
			case plir.OpActorSend:
				return EscapeActor, "allocation crosses an actor/send boundary"
			case plir.OpCall:
				if isNonEscapingBuiltinCall(op.Note) {
					continue
				}
				if looksActorSend(op.Note) {
					return EscapeActor, "allocation crosses an actor/send boundary"
				}
				if looksTaskBoundary(op.Note) {
					return EscapeTask, "allocation crosses a task boundary"
				}
				if proof, ok := callInputsCoveredByLocalNoEscapeSummary(op, carriers, callSummaries); ok {
					readOnlyCallSummaryProof = readOnlyCallSummaryProof || proof.ReadOnly
					inoutWriterCallSummaryProof = inoutWriterCallSummaryProof || proof.InoutWriter
					continue
				}
				return EscapeCallUnknown, "allocation is passed to a call without interprocedural escape facts"
			}
		}
		switch op.Kind {
		case plir.OpUnsafe:
			unsafeBoundary = true
		}
	}
	if closureBoundary {
		return EscapeClosure, "allocation is captured by a closure environment"
	}
	if aggregateBoundary {
		return EscapeAggregate, "allocation is stored inside an aggregate value"
	}
	if unsafeBoundary {
		return EscapeUnsafe, ("function contains an unsafe boundary; v0 conservatively " +
			"assumes possible raw exposure")
	}
	if readOnlyCallSummaryProof && inoutWriterCallSummaryProof {
		return EscapeNoEscape, ("allocation is passed only to proven read-only local call " +
			"summary parameters and proven local inout writer noescape " +
			"summary parameters and does not escape in the supported v0 " +
			"scan")
	}
	if inoutWriterCallSummaryProof {
		return EscapeNoEscape, ("allocation is passed only to proven local inout writer " +
			"noescape summary parameters and does not escape in the " +
			"supported v0 scan")
	}
	if readOnlyCallSummaryProof {
		return EscapeNoEscape, ("allocation is passed only to proven read-only local call " +
			"summary parameters and does not escape in the supported v0 " +
			"scan")
	}
	return EscapeNoEscape, "allocation does not escape in the supported v0 intra-function scan"
}

type localCallSummaryProof struct {
	ReadOnly    bool
	InoutWriter bool
}

func callInputsCoveredByLocalNoEscapeSummary(
	op plir.Operation,
	carriers map[string]bool,
	summaries map[string]readOnlyCallSummary,
) (localCallSummaryProof, bool) {
	callee := localCallSummaryName(op.Note)
	if callee == "" {
		return localCallSummaryProof{}, false
	}
	summary, ok := summaries[callee]
	if !ok || (len(summary.Params) == 0 && len(summary.InoutWriterParams) == 0) {
		return localCallSummaryProof{}, false
	}
	proof := localCallSummaryProof{}
	matched := false
	for i, input := range op.Inputs {
		if !inputCarriesAny(input, carriers) {
			continue
		}
		matched = true
		if summary.Params[i] {
			proof.ReadOnly = true
			continue
		}
		if summary.InoutWriterParams[i] {
			proof.InoutWriter = true
			continue
		}
		return localCallSummaryProof{}, false
	}
	return proof, matched
}

func inputCarriesAny(input string, carriers map[string]bool) bool {
	for carrier := range carriers {
		if allocationInputCarriesValue(input, carrier) {
			return true
		}
	}
	return false
}

func localCallSummaryName(note string) string {
	note = strings.TrimSpace(note)
	if note == "" {
		return ""
	}
	lower := strings.ToLower(note)
	if strings.Contains(lower, "unknown external") || strings.Contains(lower, "external call") ||
		strings.Contains(lower, "alias_boundary:") {
		return ""
	}
	fields := strings.Fields(note)
	if len(fields) == 0 {
		return ""
	}
	name := fields[0]
	if strings.HasPrefix(name, "core.") || strings.HasPrefix(name, "ffi.") {
		return ""
	}
	return name
}

func allocationCarriers(fn plir.Function, allocName string, valueID string) map[string]bool {
	carriers := map[string]bool{}
	addCarrier(carriers, allocName)
	addCarrier(carriers, valueID)
	changed := true
	for changed {
		changed = false
		for _, op := range fn.Ops {
			switch op.Kind {
			case plir.OpAssign, plir.OpAggregate, plir.OpSliceWindow:
			case plir.OpCall:
				if !isBorrowViewOperation(op.Note) {
					continue
				}
			default:
				continue
			}
			if !inputsUseCarrier(op.Inputs, carriers) {
				continue
			}
			for _, output := range op.Outputs {
				if addCarrier(carriers, output) {
					changed = true
				}
			}
		}
	}
	return carriers
}

func addCarrier(carriers map[string]bool, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	changed := false
	for _, candidate := range carrierAliases(name) {
		if candidate == "" || carriers[candidate] {
			continue
		}
		carriers[candidate] = true
		changed = true
	}
	return changed
}

func carrierAliases(name string) []string {
	aliases := []string{name}
	if idx := strings.Index(name, ":"); idx >= 0 && idx+1 < len(name) {
		aliases = append(aliases, name[idx+1:])
	}
	return aliases
}

func opUsesCarrier(op plir.Operation, carriers map[string]bool) bool {
	return inputsUseCarrier(op.Inputs, carriers)
}

func inputsUseCarrier(inputs []string, carriers map[string]bool) bool {
	for _, input := range inputs {
		for carrier := range carriers {
			if allocationInputCarriesValue(input, carrier) {
				return true
			}
		}
	}
	return false
}

func allocationInputCarriesValue(input string, carrier string) bool {
	if !allocationInputUses(input, carrier) {
		return false
	}
	return input != carrier+".len" && !strings.HasSuffix(input, ".len")
}

func isNonEscapingBuiltinCall(note string) bool {
	return isBorrowViewOperation(note) ||
		strings.Contains(note, "copies into caller-owned destination without allocation")
}

func isBorrowViewOperation(note string) bool {
	return strings.Contains(note, "creates borrowed view without allocation")
}

func chooseStorage(
	value plir.Value,
	escape EscapeClass,
	escapeReason string,
	opt Options,
) (StorageClass, string) {
	if classifyLengthStatus(value.Alloc) == LengthStatusValidEmpty {
		return StorageEliminated, "zero-length allocation intent needs no backing storage"
	}
	if value.Provenance.Kind == plir.ProvenanceIsland {
		return StorageExplicitIsland, "user-written island scope selects explicit region storage"
	}
	switch escape {
	case EscapeNoEscape:
		if strings.Contains(escapeReason, "read-only local call summary") {
			bytes := constantByteSize(value.Alloc)
			if bytes > 0 && bytes <= smallStackAllocationBytes {
				return StorageStack, fmt.Sprintf(
					("fixed_small_read_only_local_call_no_escape: fixed-size " +
						"no-escape allocation crosses only proven read-only local " +
						"call summaries and is %d bytes, within stack threshold"),
					bytes,
				)
			}
			return StorageHeap, ("no-escape allocation crosses a local call boundary but is " +
				"not fixed-small; planner keeps heap fallback until broader " +
				"call-aware region lowering is proven")
		}
		if opt.EnableRegionPlanning && isFunctionTempRegionCandidate(value) {
			return StorageFunctionTempRegion, (("function-local temporary copy has bounded " +
				"lifetime and can ") +
				"be planned for a temp region")
		}
		bytes := constantByteSize(value.Alloc)
		stackLimit := stackAllocationLimitBytes(value.Alloc)
		if bytes > 0 && bytes <= stackLimit {
			if stackLimit > smallStackAllocationBytes {
				return StorageStack, fmt.Sprintf(
					("fixed_i32_no_escape: fixed-size no-escape i32 allocation is " +
						"%d bytes, within i32 stack threshold"),
					bytes,
				)
			}
			return StorageStack, fmt.Sprintf(
				"fixed_small_no_escape: fixed-size no-escape allocation is %d bytes, within stack threshold",
				bytes,
			)
		}
		return StorageHeap, ("no-escape allocation has non-constant or large size; " +
			"planner v0 keeps heap fallback")
	case EscapeReturn:
		return StorageHeap, ("returned allocation needs caller-owned region support " +
			"before it can avoid heap fallback")
	case EscapeCallUnknown:
		return StorageHeap, "unknown call escape requires conservative heap fallback"
	case EscapeActor:
		return StorageHeap, "actor transfer region planning is not enabled for this allocation"
	case EscapeTask:
		return StorageHeap, "task transfer region planning is not enabled for this allocation"
	case EscapeUnsafe:
		return StorageHeap, ("unsafe exposure requires conservative heap fallback unless " +
			"explicit proof exists")
	case EscapeClosure:
		return StorageHeap, "closure environment escape requires conservative heap fallback"
	case EscapeAggregate:
		return StorageHeap, ("aggregate escape requires conservative heap fallback until " +
			"field-sensitive storage planning")
	default:
		return StorageHeap, "unknown escape state requires conservative heap fallback"
	}
}

func isFunctionTempRegionCandidate(value plir.Value) bool {
	if value.Alloc == nil {
		return false
	}
	if !isCopyBuiltin(value.Alloc.Builtin) {
		return false
	}
	bytes := constantByteSize(value.Alloc)
	return bytes == 0 || bytes > smallStackAllocationBytes
}

func stackAllocationLimitBytes(intent *plir.AllocIntent) int {
	if intent == nil {
		return smallStackAllocationBytes
	}
	if intent.Builtin == "core.make_i32" && intent.ElementType == "i32" && intent.ElementSize == 4 {
		return largeI32StackAllocationBytes
	}
	return smallStackAllocationBytes
}

func allocationName(valueID string) string {
	return strings.TrimPrefix(valueID, string(plir.ValueAllocIntent)+":")
}

func allocationSiteID(functionName string, allocName string, source string) string {
	if functionName == "" {
		functionName = "fn"
	}
	if allocName == "" {
		allocName = "alloc"
	}
	source = strings.TrimSpace(source)
	if source == "" {
		source = "unknown"
	}
	replacer := strings.NewReplacer(" ", "_", ":", "_", "/", "_", "\\", "_", "\t", "_")
	return "allocsite:" + functionName + ":" + allocName + ":" + replacer.Replace(source)
}

func constantByteSize(intent *plir.AllocIntent) int {
	if intent == nil || intent.ElementSize <= 0 || intent.LengthExpr == "" {
		return 0
	}
	n, ok := allocationLengthConst(intent)
	if !ok || n < 0 {
		return 0
	}
	bytes := n * int64(intent.ElementSize)
	if bytes < 0 || bytes > int64(^uint(0)>>1) {
		return 0
	}
	return int(bytes)
}

func classifyLengthStatus(intent *plir.AllocIntent) LengthStatus {
	if intent == nil || intent.ElementSize <= 0 {
		return LengthStatusInvalidContract
	}
	n, ok := allocationLengthConst(intent)
	if !ok {
		return LengthStatusRuntimeGuarded
	}
	if n < 0 {
		return LengthStatusRejectedNegative
	}
	bytes := n * int64(intent.ElementSize)
	if bytes > maxAllocationByteSize {
		return LengthStatusRejectedOverflow
	}
	if n == 0 && intent.Builtin == "core.alloc_bytes" {
		return LengthStatusInvalidContract
	}
	if n == 0 {
		return LengthStatusValidEmpty
	}
	return LengthStatusNormal
}

func allocationLengthConst(intent *plir.AllocIntent) (int64, bool) {
	if intent == nil {
		return 0, false
	}
	if intent.LengthConstKnown {
		return intent.LengthConst, true
	}
	if intent.LengthExpr == "" {
		return 0, false
	}
	n, err := strconv.ParseInt(intent.LengthExpr, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func looksActorSend(name string) bool {
	name = strings.ToLower(name)
	return strings.Contains(name, "actor") || strings.Contains(name, "send")
}

func looksTaskBoundary(name string) bool {
	name = strings.ToLower(name)
	return strings.Contains(name, "task") || strings.Contains(name, "spawn")
}
