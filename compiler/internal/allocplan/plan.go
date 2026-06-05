package allocplan

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/runtimeabi"
)

type Plan struct {
	Functions []FunctionPlan `json:"functions,omitempty"`
	Totals    Totals         `json:"totals"`
}

type ReportSummary struct {
	AllocationCount              int                   `json:"allocation_count"`
	StorageClasses               map[string]int        `json:"storage_classes"`
	ActualLoweringStorageClasses map[string]int        `json:"actual_lowering_storage_classes"`
	RuntimePaths                 map[string]int        `json:"runtime_paths"`
	AllocatorClasses             map[string]int        `json:"allocator_classes,omitempty"`
	AllocatorScopes              map[string]int        `json:"allocator_scopes,omitempty"`
	AllocatorReusePolicies       map[string]int        `json:"allocator_reuse_policies,omitempty"`
	RawPointerBoundsStatuses     map[string]int        `json:"raw_pointer_bounds_statuses,omitempty"`
	RawSlicePolicies             map[string]int        `json:"raw_slice_policies,omitempty"`
	BytesRequested               int                   `json:"bytes_requested"`
	BytesReserved                int                   `json:"bytes_reserved"`
	Regions                      []RegionReportSummary `json:"regions,omitempty"`
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
	ID                     string                           `json:"id"`
	SiteID                 string                           `json:"site_id"`
	ValueID                string                           `json:"value_id"`
	Source                 string                           `json:"source,omitempty"`
	Builtin                string                           `json:"builtin,omitempty"`
	ElementType            string                           `json:"element_type,omitempty"`
	ElementSize            int                              `json:"element_size,omitempty"`
	LengthExpr             string                           `json:"length_expr,omitempty"`
	LengthStatus           LengthStatus                     `json:"length_status,omitempty"`
	ZeroGuardStatus        string                           `json:"zero_guard_status,omitempty"`
	NegativeGuardStatus    string                           `json:"negative_guard_status,omitempty"`
	OverflowGuardStatus    string                           `json:"overflow_guard_status,omitempty"`
	ByteSize               int                              `json:"byte_size,omitempty"`
	Escape                 EscapeClass                      `json:"escape"`
	Storage                StorageClass                     `json:"storage"`
	PlannedStorage         StorageClass                     `json:"planned_storage"`
	ActualLoweringStorage  StorageClass                     `json:"actual_lowering_storage"`
	Reason                 string                           `json:"reason"`
	ValidationStatus       string                           `json:"validation_status,omitempty"`
	LoweringStatus         string                           `json:"lowering_status,omitempty"`
	BackendStorage         StorageClass                     `json:"backend_storage,omitempty"`
	BackendReason          string                           `json:"backend_reason,omitempty"`
	RuntimePath            runtimeabi.AllocationRuntimePath `json:"runtime_path,omitempty"`
	AllocatorClass         string                           `json:"allocator_class,omitempty"`
	AllocatorScope         string                           `json:"allocator_scope,omitempty"`
	AllocatorReusePolicy   string                           `json:"allocator_reuse_policy,omitempty"`
	AllocatorChunkBytes    int                              `json:"allocator_chunk_bytes,omitempty"`
	RawPointerBoundsStatus string                           `json:"raw_pointer_bounds_status,omitempty"`
	RawPointerBaseID       string                           `json:"raw_pointer_base_id,omitempty"`
	RawPointerBaseBytes    int64                            `json:"raw_pointer_base_bytes,omitempty"`
	RawPointerOffsetBytes  int64                            `json:"raw_pointer_offset_bytes,omitempty"`
	RawSlicePolicy         string                           `json:"raw_slice_policy,omitempty"`
	BytesRequested         int                              `json:"bytes_requested"`
	BytesReserved          int                              `json:"bytes_reserved"`
	RegionID               string                           `json:"region_id,omitempty"`
	Lifetime               string                           `json:"lifetime,omitempty"`
	DebugMode              string                           `json:"debug_mode,omitempty"`
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

const smallStackAllocationBytes = 4096
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
			alloc := planAllocation(fn, value, valueOpt)
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

func planAllocation(fn plir.Function, value plir.Value, opt Options) Allocation {
	id := allocationName(value.ID)
	escape, reason := classifyEscape(fn, id, value)
	storage, storageReason := chooseStorage(value, escape, opt)
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
		storageReason = "copy result is unused in the supported v0 intra-function scan; allocation intent can be elided"
	}
	if reason != "" {
		storageReason = storageReason + "; " + reason
	}
	if lengthStatus == LengthStatusValidEmpty {
		storageReason = storageReason + "; valid empty allocation has no allocator access where the backend implements the contract"
	}
	if lengthStatus == LengthStatusRejectedNegative {
		storageReason = storageReason + "; negative length is rejected before allocation"
	}
	if lengthStatus == LengthStatusRejectedOverflow {
		storageReason = storageReason + "; byte-size overflow is rejected before allocation"
	}
	byteSize := constantByteSize(value.Alloc)
	actualStorage, loweringStatus, backendReason := actualLoweringStorage(value, storage, lengthStatus, opt, scalarReplacement, unusedCopy)
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
	}
	if storage == StorageFunctionTempRegion {
		applyPlannedRegionAllocatorEvidence(&alloc, fn, byteSize)
	}
	if opt.EnableSmallHeapRuntime && actualStorage == StorageHeap && storage != StorageFunctionTempRegion && lengthStatus == LengthStatusNormal {
		applyRuntimeAllocatorEvidence(&alloc, byteSize)
	}
	applyDefaultAllocationReportHooks(&alloc)
	return alloc
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
		return runtimeabi.AllocationPathRegion
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
		alloc.RuntimePath = runtimeabi.AllocationPathRegion
		alloc.AllocatorClass = "function_temp_region"
	}
	if alloc.ActualLoweringStorage == StorageFunctionTempRegion && byteSize > 0 {
		if reserved, ok := runtimeabi.AlignRegionBytes(int64(byteSize)); ok {
			alloc.BytesRequested = byteSize
			alloc.BytesReserved = int(reserved)
		}
	}
	if alloc.ActualLoweringStorage == StorageFunctionTempRegion {
		alloc.Reason = alloc.Reason + "; P15.0 function-local temporary region lowers through region enter/reset IR"
	} else {
		alloc.Reason = alloc.Reason + "; P5.3 planned function-local temporary region; current backend still reports heap fallback until implicit region lowering lands"
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
	alloc.Reason = alloc.Reason + "; P5.2 region allocator uses aligned bump allocation with bulk free at island scope exit"
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

func applyRuntimeAllocatorEvidence(alloc *Allocation, byteSize int) {
	if alloc == nil || byteSize <= 0 {
		return
	}
	alloc.BytesRequested = byteSize
	if cls, ok := runtimeabi.SmallHeapClassForBytes(int64(byteSize)); ok {
		abi := runtimeabi.RuntimePerCoreSmallHeapABI(1)
		alloc.RuntimePath = abi.RuntimePath
		alloc.AllocatorClass = cls.Name
		alloc.AllocatorScope = "core:0"
		alloc.AllocatorReusePolicy = abi.ReusePolicy
		alloc.AllocatorChunkBytes = abi.ChunkBytes
		alloc.BytesReserved = cls.MaxBytes
		alloc.LoweringStatus = "per_core_small_heap_runtime"
		alloc.Reason = alloc.Reason + "; P15.2 runtime allocator uses per-core small-heap metadata, 16-byte size classes, chunk refills, and same-core same-class free-list reuse for this byte class"
		return
	}
	alloc.RuntimePath = runtimeabi.AllocationPathLargeMmap
	alloc.AllocatorClass = "large_mmap"
	alloc.BytesReserved = byteSize
	alloc.LoweringStatus = "large_mmap_runtime"
	alloc.Reason = alloc.Reason + "; P5.1 runtime allocator routes this large allocation to mmap fallback"
}

func actualLoweringStorage(value plir.Value, planned StorageClass, lengthStatus LengthStatus, opt Options, scalarReplacement bool, unusedCopy bool) (StorageClass, string, string) {
	if value.Provenance.Kind == plir.ProvenanceIsland || planned == StorageExplicitIsland {
		return StorageExplicitIsland, "explicit_island_lowering", "explicit island allocation remains region-backed in the current backend"
	}
	switch planned {
	case StorageEliminated:
		if opt.EnableStackLowering && unusedCopy {
			return StorageEliminated, "eliminated_unused_copy", "unused copy lowers to source evaluation plus an empty local header with no fresh storage"
		}
		if opt.EnableStackLowering && scalarReplacement {
			return StorageEliminated, "scalar_replacement", "fixed tiny no-escape allocation lowers to scalar locals with no slice backing storage"
		}
		if opt.EnableStackLowering && lengthStatus == LengthStatusValidEmpty {
			return StorageEliminated, "eliminated_no_backing_storage", "valid empty allocation intent lowers without backing storage"
		}
	case StorageStack:
		if opt.EnableStackLowering {
			return StorageStack, "stack_lowering", "fixed small no-escape allocation lowers to stack frame storage"
		}
	case StorageHeap:
		return StorageHeap, "heap_runtime", "planned heap allocation lowers through the conservative heap/runtime path"
	case StorageFunctionTempRegion:
		if opt.EnableRegionLowering {
			return StorageFunctionTempRegion, "function_temp_region_lowering", "function-local temporary copy lowers through function-temp region enter/reset IR"
		}
		return StorageHeap, "region_planned_heap_fallback", "P5.3 models a bounded function-local region, but implicit region runtime lowering is not enabled in the current backend"
	case StorageExternal:
		return StorageExternal, "external_no_lowering", "external storage is supplied outside the allocator"
	case StorageUnknownConservative:
		return StorageUnknownConservative, "unknown_conservative", "unknown storage remains conservative"
	default:
		return StorageHeap, "conservative_heap_fallback", "planner v0 records the narrower storage class; current stack backend still lowers make through the conservative heap/runtime path"
	}
	return StorageHeap, "conservative_heap_fallback", "planner v0 records the narrower storage class; current stack backend still lowers make through the conservative heap/runtime path"
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
	if value.Alloc == nil || !isCopyBuiltin(value.Alloc.Builtin) || allocName == "" || allocName == "$return" {
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
	return name == "core.string_copy" || (strings.HasPrefix(name, "core.slice_copy_") && !strings.HasPrefix(name, "core.slice_copy_into_"))
}

func isScalarReplacementCandidate(fn plir.Function, value plir.Value, allocName string) (bool, string) {
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
		return true, fmt.Sprintf("scalar_replacement_copy_fixed_constant_indices: fixed tiny no-escape copy has %d elements and every indexed use is constant in range", length)
	}
	return true, fmt.Sprintf("scalar_replacement_fixed_constant_indices: fixed tiny no-escape allocation has %d elements and every indexed use is constant in range", length)
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

func classifyEscape(fn plir.Function, allocName string, value plir.Value) (EscapeClass, string) {
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
		return EscapeUnsafe, "function contains an unsafe boundary; v0 conservatively assumes possible raw exposure"
	}
	return EscapeNoEscape, "allocation does not escape in the supported v0 intra-function scan"
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
	return isBorrowViewOperation(note) || strings.Contains(note, "copies into caller-owned destination without allocation")
}

func isBorrowViewOperation(note string) bool {
	return strings.Contains(note, "creates borrowed view without allocation")
}

func chooseStorage(value plir.Value, escape EscapeClass, opt Options) (StorageClass, string) {
	if classifyLengthStatus(value.Alloc) == LengthStatusValidEmpty {
		return StorageEliminated, "zero-length allocation intent needs no backing storage"
	}
	if value.Provenance.Kind == plir.ProvenanceIsland {
		return StorageExplicitIsland, "user-written island scope selects explicit region storage"
	}
	switch escape {
	case EscapeNoEscape:
		if opt.EnableRegionPlanning && isFunctionTempRegionCandidate(value) {
			return StorageFunctionTempRegion, "function-local temporary copy has bounded lifetime and can be planned for a temp region"
		}
		bytes := constantByteSize(value.Alloc)
		if bytes > 0 && bytes <= smallStackAllocationBytes {
			return StorageStack, fmt.Sprintf("fixed_small_no_escape: fixed-size no-escape allocation is %d bytes, within stack threshold", bytes)
		}
		return StorageHeap, "no-escape allocation has non-constant or large size; planner v0 keeps heap fallback"
	case EscapeReturn:
		return StorageHeap, "returned allocation needs caller-owned region support before it can avoid heap fallback"
	case EscapeCallUnknown:
		return StorageHeap, "unknown call escape requires conservative heap fallback"
	case EscapeActor:
		return StorageHeap, "actor transfer region planning is not enabled for this allocation"
	case EscapeTask:
		return StorageHeap, "task transfer region planning is not enabled for this allocation"
	case EscapeUnsafe:
		return StorageHeap, "unsafe exposure requires conservative heap fallback unless explicit proof exists"
	case EscapeClosure:
		return StorageHeap, "closure environment escape requires conservative heap fallback"
	case EscapeAggregate:
		return StorageHeap, "aggregate escape requires conservative heap fallback until field-sensitive storage planning"
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

func VerifyPlan(plan *Plan) error {
	if plan == nil {
		return fmt.Errorf("allocplan verifier: missing plan")
	}
	seen := map[string]bool{}
	for _, fn := range plan.Functions {
		if fn.Name == "" {
			return fmt.Errorf("allocplan verifier: function with empty name")
		}
		for _, alloc := range fn.Allocations {
			if alloc.ValueID == "" {
				return fmt.Errorf("allocplan verifier: %s allocation with empty value id", fn.Name)
			}
			if alloc.ID == "" {
				return fmt.Errorf("allocplan verifier: %s allocation with empty allocation id", fn.Name)
			}
			if alloc.SiteID == "" {
				return fmt.Errorf("allocplan verifier: %s allocation %q missing stable site id", fn.Name, alloc.ValueID)
			}
			if alloc.Builtin == "" {
				return fmt.Errorf("allocplan verifier: %s allocation %q missing builtin", fn.Name, alloc.ValueID)
			}
			key := fn.Name + "\x00" + alloc.ValueID
			if seen[key] {
				return fmt.Errorf("allocplan verifier: %s duplicate allocation %q", fn.Name, alloc.ValueID)
			}
			seen[key] = true
			if alloc.Storage == "" {
				return fmt.Errorf("allocplan verifier: %s allocation %q has empty storage", fn.Name, alloc.ValueID)
			}
			if alloc.PlannedStorage == "" {
				return fmt.Errorf("allocplan verifier: %s allocation %q has empty planned storage", fn.Name, alloc.ValueID)
			}
			if alloc.PlannedStorage != alloc.Storage {
				return fmt.Errorf("allocplan verifier: %s allocation %q planned storage %s does not match storage %s", fn.Name, alloc.ValueID, alloc.PlannedStorage, alloc.Storage)
			}
			if alloc.ActualLoweringStorage == "" {
				return fmt.Errorf("allocplan verifier: %s allocation %q has empty actual lowering storage", fn.Name, alloc.ValueID)
			}
			if alloc.ValidationStatus == "" {
				return fmt.Errorf("allocplan verifier: %s allocation %q missing validation status", fn.Name, alloc.ValueID)
			}
			if alloc.LoweringStatus == "" {
				return fmt.Errorf("allocplan verifier: %s allocation %q missing lowering status", fn.Name, alloc.ValueID)
			}
			if alloc.LengthStatus == "" {
				return fmt.Errorf("allocplan verifier: %s allocation %q has empty length status", fn.Name, alloc.ValueID)
			}
			if strings.TrimSpace(alloc.Reason) == "" {
				return fmt.Errorf("allocplan verifier: %s allocation %q missing storage reason", fn.Name, alloc.ValueID)
			}
			for _, observed := range []struct {
				name    string
				storage StorageClass
			}{
				{name: "storage", storage: alloc.Storage},
				{name: "planned storage", storage: alloc.PlannedStorage},
				{name: "actual lowering storage", storage: alloc.ActualLoweringStorage},
			} {
				if alloc.Escape != EscapeNoEscape && trustedStorageRequiresNoEscape(observed.storage, alloc.LengthStatus) {
					return fmt.Errorf("allocplan verifier: %s escaping allocation %q cannot use %s %s", fn.Name, alloc.ValueID, observed.storage, observed.name)
				}
				if alloc.Escape == EscapeNoEscape && trustedStorageRequiresNoEscape(observed.storage, alloc.LengthStatus) && !storageHasCompilerOwnedNoEscapeProof(observed.storage, alloc.ValidationStatus, alloc.LengthStatus) {
					return fmt.Errorf("allocplan verifier: %s allocation %q uses %s %s without compiler-owned no-escape proof", fn.Name, alloc.ValueID, observed.storage, observed.name)
				}
			}
			if alloc.Storage == StorageExplicitIsland && alloc.Escape != EscapeNoEscape {
				return fmt.Errorf("allocplan verifier: %s island allocation %q cannot escape its island scope", fn.Name, alloc.ValueID)
			}
		}
	}
	return nil
}

func trustedStorageRequiresNoEscape(storage StorageClass, lengthStatus LengthStatus) bool {
	switch storage {
	case StorageEliminated:
		return lengthStatus != LengthStatusValidEmpty
	case StorageRegister, StorageStack, StorageRegion, StorageFunctionTempRegion,
		StorageExplicitIsland, StorageTaskRegion, StorageActorMoveRegion:
		return true
	default:
		return false
	}
}

func storageHasCompilerOwnedNoEscapeProof(storage StorageClass, status string, lengthStatus LengthStatus) bool {
	status = strings.TrimSpace(status)
	switch storage {
	case StorageEliminated:
		return lengthStatus == LengthStatusValidEmpty || status == "validated_no_escape"
	case StorageRegister, StorageStack:
		return status == "validated_no_escape"
	case StorageRegion:
		return status == "validated_region_scope"
	case StorageFunctionTempRegion:
		return status == "validated_function_temp_region_scope"
	case StorageExplicitIsland:
		if lengthStatus == LengthStatusValidEmpty && status == "validated_empty_no_backing" {
			return true
		}
		return status == "validated_explicit_island_scope"
	case StorageTaskRegion:
		return status == "validated_task_region_scope"
	case StorageActorMoveRegion:
		return status == "validated_actor_move_region_scope"
	default:
		return true
	}
}

func Summarize(plan *Plan) ReportSummary {
	summary := ReportSummary{
		StorageClasses:               map[string]int{},
		ActualLoweringStorageClasses: map[string]int{},
		RuntimePaths:                 map[string]int{},
		AllocatorClasses:             map[string]int{},
		AllocatorScopes:              map[string]int{},
		AllocatorReusePolicies:       map[string]int{},
		RawPointerBoundsStatuses:     map[string]int{},
		RawSlicePolicies:             map[string]int{},
	}
	if plan == nil {
		return summary
	}
	regions := map[string]RegionReportSummary{}
	for _, fn := range plan.Functions {
		for _, alloc := range fn.Allocations {
			summary.AllocationCount++
			summary.StorageClasses[string(alloc.PlannedStorage)]++
			summary.ActualLoweringStorageClasses[string(alloc.ActualLoweringStorage)]++
			runtimePath := string(RuntimePathForAllocation(alloc))
			summary.RuntimePaths[runtimePath]++
			if alloc.AllocatorClass != "" {
				summary.AllocatorClasses[alloc.AllocatorClass]++
			}
			if alloc.AllocatorScope != "" {
				summary.AllocatorScopes[alloc.AllocatorScope]++
			}
			if alloc.AllocatorReusePolicy != "" {
				summary.AllocatorReusePolicies[alloc.AllocatorReusePolicy]++
			}
			if alloc.RawPointerBoundsStatus != "" {
				summary.RawPointerBoundsStatuses[alloc.RawPointerBoundsStatus]++
			}
			if alloc.RawSlicePolicy != "" {
				summary.RawSlicePolicies[alloc.RawSlicePolicy]++
			}
			requested := allocationReportBytesRequested(alloc)
			reserved := allocationReportBytesReserved(alloc)
			summary.BytesRequested += requested
			summary.BytesReserved += reserved
			if alloc.RegionID == "" {
				continue
			}
			switch alloc.ActualLoweringStorage {
			case StorageRegion, StorageFunctionTempRegion, StorageExplicitIsland, StorageTaskRegion, StorageActorMoveRegion:
			default:
				continue
			}
			key := alloc.RegionID + "\x00" + alloc.Lifetime + "\x00" + string(alloc.PlannedStorage) + "\x00" + runtimePath
			region := regions[key]
			if region.RegionID == "" {
				region.RegionID = alloc.RegionID
				region.Lifetime = alloc.Lifetime
				region.StorageClass = string(alloc.PlannedStorage)
				region.RuntimePath = runtimePath
			}
			region.AllocationCount++
			region.BytesRequested += requested
			region.BytesReserved += reserved
			regions[key] = region
		}
	}
	if len(regions) > 0 {
		keys := make([]string, 0, len(regions))
		for key := range regions {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			summary.Regions = append(summary.Regions, regions[key])
		}
	}
	return summary
}

func allocationReportBytesRequested(alloc Allocation) int {
	if alloc.BytesRequested > 0 {
		return alloc.BytesRequested
	}
	if alloc.ByteSize > 0 {
		return alloc.ByteSize
	}
	return 0
}

func allocationReportBytesReserved(alloc Allocation) int {
	if alloc.BytesReserved > 0 {
		return alloc.BytesReserved
	}
	if alloc.ActualLoweringStorage == StorageEliminated {
		return 0
	}
	if alloc.ByteSize > 0 {
		return alloc.ByteSize
	}
	return 0
}

func FormatText(plan *Plan) string {
	if plan == nil {
		return ""
	}
	summary := Summarize(plan)
	var b strings.Builder
	for _, fn := range plan.Functions {
		fmt.Fprintf(&b, "func %s\n", fn.Name)
		for _, alloc := range fn.Allocations {
			fmt.Fprintf(&b, "  %s: site_id: %s builtin: %s planned_storage: %s actual_lowering_storage: %s escape: %s", alloc.ID, alloc.SiteID, alloc.Builtin, alloc.PlannedStorage, alloc.ActualLoweringStorage, alloc.Escape)
			if alloc.LengthStatus != "" {
				fmt.Fprintf(&b, " length_status: %s", alloc.LengthStatus)
			}
			if alloc.ValidationStatus != "" {
				fmt.Fprintf(&b, " validation_status: %s", alloc.ValidationStatus)
			}
			if alloc.LoweringStatus != "" {
				fmt.Fprintf(&b, " lowering_status: %s", alloc.LoweringStatus)
			}
			if alloc.ZeroGuardStatus != "" {
				fmt.Fprintf(&b, " zero_guard: %s", alloc.ZeroGuardStatus)
			}
			if alloc.NegativeGuardStatus != "" {
				fmt.Fprintf(&b, " negative_guard: %s", alloc.NegativeGuardStatus)
			}
			if alloc.OverflowGuardStatus != "" {
				fmt.Fprintf(&b, " overflow_guard: %s", alloc.OverflowGuardStatus)
			}
			if alloc.ByteSize > 0 {
				fmt.Fprintf(&b, " bytes: %d", alloc.ByteSize)
			}
			if alloc.RuntimePath != "" {
				fmt.Fprintf(&b, " runtime_path: %s", alloc.RuntimePath)
			}
			if alloc.AllocatorClass != "" {
				fmt.Fprintf(&b, " allocator_class: %s", alloc.AllocatorClass)
			}
			if alloc.AllocatorScope != "" {
				fmt.Fprintf(&b, " allocator_scope: %s", alloc.AllocatorScope)
			}
			if alloc.AllocatorReusePolicy != "" {
				fmt.Fprintf(&b, " allocator_reuse_policy: %s", alloc.AllocatorReusePolicy)
			}
			if alloc.AllocatorChunkBytes > 0 {
				fmt.Fprintf(&b, " allocator_chunk_bytes: %d", alloc.AllocatorChunkBytes)
			}
			if alloc.RawPointerBoundsStatus != "" {
				fmt.Fprintf(&b, " raw_pointer_bounds: %s", alloc.RawPointerBoundsStatus)
			}
			if alloc.RawPointerBaseID != "" {
				fmt.Fprintf(&b, " raw_pointer_base: %s", alloc.RawPointerBaseID)
			}
			if alloc.RawPointerBaseBytes > 0 {
				fmt.Fprintf(&b, " raw_pointer_base_bytes: %d", alloc.RawPointerBaseBytes)
			}
			if alloc.RawPointerOffsetBytes != 0 {
				fmt.Fprintf(&b, " raw_pointer_offset_bytes: %d", alloc.RawPointerOffsetBytes)
			}
			if alloc.RawSlicePolicy != "" {
				fmt.Fprintf(&b, " raw_slice_policy: %s", alloc.RawSlicePolicy)
			}
			if alloc.BytesRequested > 0 {
				fmt.Fprintf(&b, " bytes_requested: %d", alloc.BytesRequested)
			}
			if alloc.BytesReserved > 0 {
				fmt.Fprintf(&b, " bytes_reserved: %d", alloc.BytesReserved)
			}
			if alloc.RegionID != "" {
				fmt.Fprintf(&b, " region_id: %s", alloc.RegionID)
			}
			if alloc.Lifetime != "" {
				fmt.Fprintf(&b, " lifetime: %s", alloc.Lifetime)
			}
			if alloc.DebugMode != "" {
				fmt.Fprintf(&b, " debug_mode: %s", alloc.DebugMode)
			}
			if alloc.Reason != "" {
				fmt.Fprintf(&b, " reason: %s", alloc.Reason)
			}
			if alloc.BackendStorage != "" {
				fmt.Fprintf(&b, " backend_storage: %s", alloc.BackendStorage)
			}
			fmt.Fprintln(&b)
		}
	}
	fmt.Fprintf(&b, "totals allocation_count:%d bytes_requested:%d bytes_reserved:%d heap:%d stack:%d region:%d function_temp_region:%d explicit_island:%d eliminated:%d runtime_paths:%s allocator_classes:%s allocator_scopes:%s allocator_reuse_policies:%s raw_pointer_bounds:%s raw_slice_policies:%s\n",
		summary.AllocationCount,
		summary.BytesRequested,
		summary.BytesReserved,
		plan.Totals.Heap,
		plan.Totals.Stack,
		plan.Totals.Region,
		plan.Totals.FunctionTempRegion,
		plan.Totals.ExplicitIsland,
		plan.Totals.Eliminated,
		formatSummaryCounts(summary.RuntimePaths),
		formatSummaryCounts(summary.AllocatorClasses),
		formatSummaryCounts(summary.AllocatorScopes),
		formatSummaryCounts(summary.AllocatorReusePolicies),
		formatSummaryCounts(summary.RawPointerBoundsStatuses),
		formatSummaryCounts(summary.RawSlicePolicies),
	)
	return b.String()
}

func formatSummaryCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return strings.Join(parts, ",")
}

func (t *Totals) add(storage StorageClass) {
	switch storage {
	case StorageEliminated:
		t.Eliminated++
	case StorageRegister:
		t.Register++
	case StorageStack:
		t.Stack++
	case StorageRegion:
		t.Region++
	case StorageFunctionTempRegion:
		t.FunctionTempRegion++
	case StorageExplicitIsland:
		t.ExplicitIsland++
	case StorageTaskRegion:
		t.TaskRegion++
	case StorageActorMoveRegion:
		t.ActorMoveRegion++
	case StorageHeap:
		t.Heap++
	case StorageMmapLarge:
		t.MmapLarge++
	case StorageExternal:
		t.External++
	default:
		t.Unknown++
	}
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
