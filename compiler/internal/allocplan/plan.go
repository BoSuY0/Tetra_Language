package allocplan

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/runtimeabi"
	semanticsresources "tetra_language/compiler/internal/semantics/resources"
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
	SourceFactIDs         []string               `json:"source_fact_ids,omitempty"`
	ProofIDs              []string               `json:"proof_ids,omitempty"`
	DecisionCode          string                 `json:"decision_code,omitempty"`
	PlanDigest            string                 `json:"plan_digest,omitempty"`
	LoweredArtifactID     string                 `json:"lowered_artifact_id,omitempty"`
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
	HeapReasonActorMoveUnproven          = "domain.actor_move_unproven"
	HeapReasonTaskMoveUnproven           = "domain.task_move_unproven"
	HeapReasonRequestOwnerUnproven       = "domain.request_owner_unproven"
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
	codes = append(codes, domainUnprovenReasonCodes(alloc.Reason)...)
	if len(codes) == 0 {
		codes = append(codes, HeapReasonDynamicLifetime)
	}
	return appendReasonCodes(nil, codes...)
}

func domainUnprovenReasonCodes(reason string) []string {
	var codes []string
	for _, code := range []string{
		HeapReasonActorMoveUnproven,
		HeapReasonTaskMoveUnproven,
		HeapReasonRequestOwnerUnproven,
	} {
		if strings.Contains(reason, code) {
			codes = append(codes, code)
		}
	}
	return codes
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

func ApplyLoweredAllocationReportHooks(alloc *Allocation) {
	ApplyLoweredAllocationReportHooksWithOptions(alloc, Options{})
}

func ApplyLoweredAllocationReportHooksWithOptions(alloc *Allocation, opt Options) {
	if alloc == nil {
		return
	}
	alloc.RuntimePath = ""
	if alloc.ActualLoweringStorage == StorageFunctionTempRegion {
		alloc.RuntimePath = runtimeabi.AllocationPathScopedSingleMappingV0
		alloc.AllocatorClass = "function_temp_region"
		if alloc.ByteSize > 0 {
			if reserved, ok := runtimeabi.AlignRegionBytes(int64(alloc.ByteSize)); ok {
				alloc.BytesRequested = alloc.ByteSize
				alloc.BytesReserved = int(reserved)
			}
		}
	} else if opt.EnableSmallHeapRuntime && alloc.ActualLoweringStorage == StorageHeap {
		applyRuntimeAllocatorEvidence(alloc, alloc.ByteSize)
	}
	applyDefaultAllocationReportHooks(alloc)
	applyHeapReasonCodeEvidence(alloc)
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
		domain = runtimeabi.DefaultProcessMemoryDomain(0, 0)
	}
	domain.RequestedBytes = requested
	domain.ReservedBytes = reserved
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
	allocPath := semanticsresources.Path(allocName)
	inputPath := semanticsresources.Path(input)
	return allocPath == inputPath || allocPath.IsAncestorOf(inputPath)
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
