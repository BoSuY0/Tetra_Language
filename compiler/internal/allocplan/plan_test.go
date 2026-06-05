package allocplan

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/runtimeabi"
	"tetra_language/compiler/internal/semantics"
)

func checkedProgram(t *testing.T, src string) *semantics.CheckedProgram {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return checked
}

func allocationPlan(t *testing.T, src string) *Plan {
	t.Helper()
	plirProg, err := plir.FromCheckedProgram(checkedProgram(t, src))
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := plir.VerifyProgram(plirProg); err != nil {
		t.Fatalf("PLIR VerifyProgram: %v", err)
	}
	plan, err := FromPLIR(plirProg)
	if err != nil {
		t.Fatalf("FromPLIR: %v", err)
	}
	return plan
}

func allocationPlanWithOptions(t *testing.T, src string, opt Options) *Plan {
	t.Helper()
	plirProg, err := plir.FromCheckedProgram(checkedProgram(t, src))
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := plir.VerifyProgram(plirProg); err != nil {
		t.Fatalf("PLIR VerifyProgram: %v", err)
	}
	plan, err := FromPLIRWithOptions(plirProg, opt)
	if err != nil {
		t.Fatalf("FromPLIRWithOptions: %v", err)
	}
	return plan
}

func allocationPlanFile(t *testing.T, src string) *Plan {
	t.Helper()
	file, err := frontend.ParseFile([]byte(src), "allocplan_test.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	checked, err := semantics.CheckWorld(&module.World{Files: []*frontend.FileAST{file}})
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	plirProg, err := plir.FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := plir.VerifyProgram(plirProg); err != nil {
		t.Fatalf("PLIR VerifyProgram: %v", err)
	}
	plan, err := FromPLIR(plirProg)
	if err != nil {
		t.Fatalf("FromPLIR: %v", err)
	}
	return plan
}

func TestPlannerClassifiesLocalAndReturnedAllocations(t *testing.T) {
	plan := allocationPlan(t, `
func local() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    xs[0] = 1
    return 0

func ret() -> []u8
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    return xs

func main() -> Int
uses alloc, mem:
    return local()
`)

	local := findAllocation(t, plan, "local", "xs")
	if local.Escape != EscapeNoEscape || local.Storage != StorageStack {
		t.Fatalf("local allocation = %+v, want NoEscape/Stack", local)
	}
	if local.SiteID == "" || !strings.HasPrefix(local.SiteID, "allocsite:local:xs:") {
		t.Fatalf("local site id = %q, want stable allocsite prefix", local.SiteID)
	}
	if local.Builtin != "core.make_u8" {
		t.Fatalf("local builtin = %q, want core.make_u8", local.Builtin)
	}
	if local.PlannedStorage != StorageStack || local.ActualLoweringStorage != StorageHeap {
		t.Fatalf("local planned/actual storage = %q/%q, want Stack/Heap", local.PlannedStorage, local.ActualLoweringStorage)
	}
	if local.ValidationStatus == "" || local.LoweringStatus == "" {
		t.Fatalf("local validation/lowering status missing: %+v", local)
	}
	if local.BackendStorage != StorageHeap {
		t.Fatalf("local backend storage = %q, want conservative heap note", local.BackendStorage)
	}
	ret := findAllocation(t, plan, "ret", "xs")
	if ret.Escape != EscapeReturn || ret.Storage != StorageHeap {
		t.Fatalf("returned allocation = %+v, want EscapesReturn/Heap", ret)
	}
}

func TestPlannerReportsActualStackLoweringWhenEnabled(t *testing.T) {
	plirProg, err := plir.FromCheckedProgram(checkedProgram(t, `
func local() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(4)
    xs[0] = 10
    return xs[0]

func ret() -> []i32
uses alloc, mem:
    var xs: []i32 = make_i32(4)
    return xs

func main() -> Int
uses alloc, mem:
    return local()
`))
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	plan, err := FromPLIRWithOptions(plirProg, Options{EnableStackLowering: true})
	if err != nil {
		t.Fatalf("FromPLIRWithOptions: %v", err)
	}

	local := findAllocation(t, plan, "local", "xs")
	if local.PlannedStorage != StorageStack || local.ActualLoweringStorage != StorageStack {
		t.Fatalf("local planned/actual storage = %q/%q, want Stack/Stack", local.PlannedStorage, local.ActualLoweringStorage)
	}
	if local.LoweringStatus != "stack_lowering" || !strings.Contains(local.Reason, "fixed_small_no_escape") {
		t.Fatalf("local lowering status/reason = %q/%q, want stack lowering evidence", local.LoweringStatus, local.Reason)
	}

	ret := findAllocation(t, plan, "ret", "xs")
	if ret.PlannedStorage != StorageHeap || ret.ActualLoweringStorage != StorageHeap {
		t.Fatalf("returned planned/actual storage = %q/%q, want Heap/Heap", ret.PlannedStorage, ret.ActualLoweringStorage)
	}
}

func TestPlannerReportsSmallHeapRuntimeAllocatorClass(t *testing.T) {
	plan := allocationPlanWithOptions(t, `
func ret_small() -> []u8
uses alloc, mem:
    var xs: []u8 = make_u8(17)
    return xs

func ret_large() -> []u8
uses alloc, mem:
    var xs: []u8 = make_u8(5000)
    return xs

func main() -> Int
uses alloc, mem:
    return 0
`, Options{EnableSmallHeapRuntime: true})

	small := findAllocation(t, plan, "ret_small", "xs")
	if small.RuntimePath != runtimeabi.AllocationPathPerCoreSmallHeap || small.AllocatorClass != "small_32" {
		t.Fatalf("small allocation runtime evidence = %+v, want per_core_small_heap/small_32", small)
	}
	if small.BytesRequested != 17 || small.BytesReserved != 32 {
		t.Fatalf("small allocation bytes = requested %d reserved %d, want 17/32", small.BytesRequested, small.BytesReserved)
	}
	large := findAllocation(t, plan, "ret_large", "xs")
	if large.RuntimePath != runtimeabi.AllocationPathLargeMmap || large.AllocatorClass != "large_mmap" {
		t.Fatalf("large allocation runtime evidence = %+v, want large_mmap", large)
	}
	if large.BytesRequested != 5000 || large.BytesReserved != 5000 {
		t.Fatalf("large allocation bytes = requested %d reserved %d, want 5000/5000", large.BytesRequested, large.BytesReserved)
	}
	if !strings.Contains(FormatText(plan), "allocator_class: small_32") {
		t.Fatalf("FormatText missing small heap allocator class:\n%s", FormatText(plan))
	}
}

func TestPlannerReportsPerCoreSmallHeapAllocatorEvidence(t *testing.T) {
	plan := allocationPlanWithOptions(t, `
func ret_small() -> []u8
uses alloc, mem:
    var xs: []u8 = make_u8(17)
    return xs

func main() -> Int
uses alloc, mem:
    return 0
`, Options{EnableSmallHeapRuntime: true})

	small := findAllocation(t, plan, "ret_small", "xs")
	if small.RuntimePath != runtimeabi.AllocationPathPerCoreSmallHeap {
		t.Fatalf("small allocation runtime_path = %q, want %q: %+v", small.RuntimePath, runtimeabi.AllocationPathPerCoreSmallHeap, small)
	}
	if small.AllocatorClass != "small_32" {
		t.Fatalf("small allocation allocator_class = %q, want small_32: %+v", small.AllocatorClass, small)
	}
	if small.AllocatorScope != "core:0" {
		t.Fatalf("small allocation allocator_scope = %q, want core:0: %+v", small.AllocatorScope, small)
	}
	if small.AllocatorReusePolicy != "same_core_same_size_class_free_list" {
		t.Fatalf("small allocation reuse policy = %q, want same-core free list: %+v", small.AllocatorReusePolicy, small)
	}
	if small.AllocatorChunkBytes != runtimeabi.SmallHeapChunkBytes {
		t.Fatalf("small allocation chunk bytes = %d, want %d: %+v", small.AllocatorChunkBytes, runtimeabi.SmallHeapChunkBytes, small)
	}

	summary := Summarize(plan)
	if summary.RuntimePaths[string(runtimeabi.AllocationPathPerCoreSmallHeap)] != 1 {
		t.Fatalf("runtime path summary = %+v, want per_core_small_heap count", summary.RuntimePaths)
	}
	if summary.AllocatorClasses["small_32"] != 1 {
		t.Fatalf("allocator class summary = %+v, want small_32 count", summary.AllocatorClasses)
	}
	if summary.AllocatorScopes["core:0"] != 1 {
		t.Fatalf("allocator scope summary = %+v, want core:0 count", summary.AllocatorScopes)
	}
	if summary.AllocatorReusePolicies["same_core_same_size_class_free_list"] != 1 {
		t.Fatalf("allocator reuse summary = %+v, want same-core reuse policy count", summary.AllocatorReusePolicies)
	}
	text := FormatText(plan)
	for _, want := range []string{
		"runtime_path: per_core_small_heap",
		"allocator_class: small_32",
		"allocator_scope: core:0",
		"allocator_reuse_policy: same_core_same_size_class_free_list",
		"allocator_chunk_bytes: 65536",
		"allocator_reuse_policies:same_core_same_size_class_free_list=1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("FormatText missing %q:\n%s", want, text)
		}
	}
}

func TestPlannerReportsRawAllocBytesPointerBoundsMetadata(t *testing.T) {
	plan := allocationPlanWithOptions(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(24)
        let q: ptr = core.ptr_add(p, 8, mem)
        let _: UInt8 = core.store_u8(q, 7, mem)
        return core.load_u8(q, mem)
    return 0
`, Options{EnableSmallHeapRuntime: true})

	raw := findAllocation(t, plan, "main", "p")
	if raw.Builtin != "core.alloc_bytes" {
		t.Fatalf("raw allocation builtin = %q, want core.alloc_bytes: %+v", raw.Builtin, raw)
	}
	if raw.RawPointerBoundsStatus != string(runtimeabi.RawPointerBoundsAllocationBase) {
		t.Fatalf("raw pointer bounds status = %q, want allocation-base metadata: %+v", raw.RawPointerBoundsStatus, raw)
	}
	if raw.RawPointerBaseID != "p" || raw.RawPointerBaseBytes != 24 || raw.RawPointerOffsetBytes != 0 {
		t.Fatalf("raw pointer base metadata = id %q bytes %d offset %d, want p/24/0: %+v", raw.RawPointerBaseID, raw.RawPointerBaseBytes, raw.RawPointerOffsetBytes, raw)
	}
	if raw.RawSlicePolicy != string(runtimeabi.RawSliceBoundsExternalUnknown) {
		t.Fatalf("raw slice policy = %q, want external unknown until verified root construction: %+v", raw.RawSlicePolicy, raw)
	}
	if raw.RuntimePath != runtimeabi.AllocationPathPerCoreSmallHeap {
		t.Fatalf("raw allocation runtime path = %q, want small heap optimization to be visible without trusting arbitrary pointers: %+v", raw.RuntimePath, raw)
	}

	summary := Summarize(plan)
	if summary.RawPointerBoundsStatuses[string(runtimeabi.RawPointerBoundsAllocationBase)] != 1 {
		t.Fatalf("raw pointer bounds summary = %+v, want allocation_base_metadata count", summary.RawPointerBoundsStatuses)
	}
	if summary.RawSlicePolicies[string(runtimeabi.RawSliceBoundsExternalUnknown)] != 1 {
		t.Fatalf("raw slice policy summary = %+v, want external_unknown count", summary.RawSlicePolicies)
	}
	text := FormatText(plan)
	for _, want := range []string{
		"raw_pointer_bounds: allocation_base_metadata",
		"raw_pointer_base: p",
		"raw_pointer_base_bytes: 24",
		"raw_slice_policy: external_unknown",
		"raw_pointer_bounds:allocation_base_metadata=1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("FormatText missing %q:\n%s", want, text)
		}
	}
}

func TestPlannerReportsExplicitIslandRuntimeAllocatorClass(t *testing.T) {
	plan := allocationPlan(t, `
func main() -> Int
uses alloc, islands, mem:
    island(64) as isl:
        var xs: []u8 = core.island_make_u8(isl, 17)
        xs[0] = 1
        return xs[0]
    return 0
`)

	island := findAllocation(t, plan, "main", "xs")
	if island.Storage != StorageExplicitIsland || island.ActualLoweringStorage != StorageExplicitIsland {
		t.Fatalf("island storage = %s/%s, want explicit island lowering", island.Storage, island.ActualLoweringStorage)
	}
	if island.RuntimePath != runtimeabi.AllocationPathExplicitIsland || island.AllocatorClass != "region_bump_16" {
		t.Fatalf("island runtime evidence = %+v, want explicit_island/region_bump_16", island)
	}
	if island.BytesRequested != 17 || island.BytesReserved != 32 {
		t.Fatalf("island bytes = requested %d reserved %d, want 17/32", island.BytesRequested, island.BytesReserved)
	}
	if island.RegionID != "island:isl" || island.Lifetime == "" || island.DebugMode == "" {
		t.Fatalf("island report hooks missing region/lifetime/debug evidence: %+v", island)
	}
	text := FormatText(plan)
	for _, want := range []string{"runtime_path: explicit_island", "allocator_class: region_bump_16", "region_id: island:isl", "bytes_reserved: 32"} {
		if !strings.Contains(text, want) {
			t.Fatalf("FormatText missing %q:\n%s", want, text)
		}
	}
}

func TestPlannerReportsExplicitIslandRegionForGuardedLengths(t *testing.T) {
	plan := allocationPlan(t, `
func empty() -> Int
uses alloc, islands, mem:
    island(64) as isl:
        var xs: []u8 = core.island_make_u8(isl, 0)
        return xs.len
    return 0

func negative() -> Int
uses alloc, islands, mem:
    island(64) as isl:
        var xs: []u8 = core.island_make_u8(isl, 0 - 1)
        return xs.len
    return 0

func overflow() -> Int
uses alloc, islands, mem:
    island(64) as isl:
        var xs: []u16 = core.island_make_u16(isl, 1073741824)
        return xs.len
    return 0

func main() -> Int
uses alloc, mem:
    return 0
`)

	for _, tc := range []struct {
		fn     string
		status LengthStatus
	}{
		{fn: "empty", status: LengthStatusValidEmpty},
		{fn: "negative", status: LengthStatusRejectedNegative},
		{fn: "overflow", status: LengthStatusRejectedOverflow},
	} {
		alloc := findAllocation(t, plan, tc.fn, "xs")
		if alloc.LengthStatus != tc.status || alloc.ActualLoweringStorage != StorageExplicitIsland {
			t.Fatalf("%s allocation = %+v, want %s actual ExplicitIsland", tc.fn, alloc, tc.status)
		}
		if alloc.RuntimePath != runtimeabi.AllocationPathExplicitIsland || alloc.RegionID != "island:isl" || alloc.Lifetime == "" {
			t.Fatalf("%s explicit island metadata = %+v, want runtime path, region, and lifetime", tc.fn, alloc)
		}
	}
}

func TestPlannerSelectsFunctionTempRegionForTemporaryCopyWhenEnabled(t *testing.T) {
	plan := allocationPlanWithOptions(t, `
func local_copy(n: Int) -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(8)
    xs[0] = 1
    let copied: []u8 = xs.window(0, n).copy()
    return copied.len

func main() -> Int
uses alloc, mem:
    return local_copy(2)
`, Options{EnableRegionPlanning: true})

	copied := findAllocation(t, plan, "local_copy", "copied")
	want := StorageFunctionTempRegion
	if copied.PlannedStorage != want || copied.ActualLoweringStorage != StorageHeap {
		t.Fatalf("temporary copy planned/actual storage = %q/%q, want FunctionTempRegion/Heap fallback: %+v", copied.PlannedStorage, copied.ActualLoweringStorage, copied)
	}
	if copied.RuntimePath != runtimeabi.AllocationPathHeap || copied.RegionID != "region:local_copy:temp" {
		t.Fatalf("temporary copy fallback evidence = %+v, want heap runtime path with planned function temp id", copied)
	}
	if copied.AllocatorClass != "" {
		t.Fatalf("temporary copy fallback allocator class = %q, want no region allocator claim: %+v", copied.AllocatorClass, copied)
	}
	if copied.Lifetime != "function:local_copy" || copied.DebugMode == "" {
		t.Fatalf("temporary copy lifetime/debug evidence = %+v", copied)
	}
	text := FormatText(plan)
	for _, want := range []string{"planned_storage: FunctionTempRegion", "actual_lowering_storage: Heap", "runtime_path: heap", "backend_storage: Heap", "region_id: region:local_copy:temp", "reason: function-local temporary copy"} {
		if !strings.Contains(text, want) {
			t.Fatalf("FormatText missing %q:\n%s", want, text)
		}
	}
}

func TestPlannerReportsActualFunctionTempRegionLoweringWhenEnabled(t *testing.T) {
	plan := allocationPlanWithOptions(t, `
func local_copy(n: Int) -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(8)
    xs[0] = 1
    let copied: []u8 = xs.window(0, n).copy()
    return copied.len

func main() -> Int
uses alloc, mem:
    return local_copy(2)
`, Options{EnableRegionPlanning: true, EnableRegionLowering: true})

	copied := findAllocation(t, plan, "local_copy", "copied")
	want := StorageFunctionTempRegion
	if copied.PlannedStorage != want || copied.ActualLoweringStorage != want {
		t.Fatalf("temporary copy planned/actual storage = %q/%q, want FunctionTempRegion/FunctionTempRegion: %+v", copied.PlannedStorage, copied.ActualLoweringStorage, copied)
	}
	if copied.LoweringStatus != "function_temp_region_lowering" {
		t.Fatalf("temporary copy lowering status = %q, want function_temp_region_lowering: %+v", copied.LoweringStatus, copied)
	}
	if copied.RuntimePath != runtimeabi.AllocationPathRegion || copied.AllocatorClass != "function_temp_region" || copied.RegionID != "region:local_copy:temp" {
		t.Fatalf("temporary copy region report evidence = %+v, want function temp region", copied)
	}
	text := FormatText(plan)
	for _, want := range []string{"planned_storage: FunctionTempRegion", "actual_lowering_storage: FunctionTempRegion", "runtime_path: region", "allocator_class: function_temp_region", "function_temp_region:1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("FormatText missing %q:\n%s", want, text)
		}
	}
}

func TestPlannerKeepsFunctionTempRegionFallbackAsHeapWhenLoweringDisabled(t *testing.T) {
	plan := allocationPlanWithOptions(t, `
func local_copy(n: Int) -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(8)
    let copied: []u8 = xs.window(0, n).copy()
    return copied.len

func main() -> Int
uses alloc, mem:
    return local_copy(2)
`, Options{EnableRegionPlanning: true})

	copied := findAllocation(t, plan, "local_copy", "copied")
	want := StorageFunctionTempRegion
	if copied.PlannedStorage != want || copied.ActualLoweringStorage != StorageHeap {
		t.Fatalf("temporary copy planned/actual storage = %q/%q, want FunctionTempRegion/Heap fallback: %+v", copied.PlannedStorage, copied.ActualLoweringStorage, copied)
	}
	if copied.RuntimePath != runtimeabi.AllocationPathHeap || copied.AllocatorClass != "" {
		t.Fatalf("temporary copy heap fallback evidence = %+v, want heap runtime path without region allocator class", copied)
	}
}

func TestPlannerLimitsFunctionTempRegionToOneAllocationPerFunction(t *testing.T) {
	plan := allocationPlanWithOptions(t, `
func local_copy(n: Int) -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(8)
    xs[0] = 3
    let copied_a: []u8 = xs.window(0, n).copy()
    let copied_b: []u8 = xs.window(0, n).copy()
    return copied_a[0] + copied_b[0]

func main() -> Int
uses alloc, mem:
    return local_copy(2)
`, Options{EnableRegionPlanning: true, EnableRegionLowering: true})

	copiedA := findAllocation(t, plan, "local_copy", "copied_a")
	copiedB := findAllocation(t, plan, "local_copy", "copied_b")
	if copiedA.PlannedStorage != StorageFunctionTempRegion || copiedA.ActualLoweringStorage != StorageFunctionTempRegion {
		t.Fatalf("first temporary copy = %+v, want FunctionTempRegion/FunctionTempRegion", copiedA)
	}
	if copiedB.PlannedStorage != StorageHeap || copiedB.ActualLoweringStorage != StorageHeap {
		t.Fatalf("second temporary copy = %+v, want Heap/Heap conservative fallback", copiedB)
	}
}

func TestPlannerDoesNotSelectDeadRegionForReturnedCopy(t *testing.T) {
	plan := allocationPlanWithOptions(t, `
func returned_copy() -> []u8
uses alloc, mem:
    var xs: []u8 = make_u8(8)
    return xs.window(0, 2).copy()

func main() -> Int
uses alloc, mem:
    return 0
`, Options{EnableRegionPlanning: true})

	copied := findAllocation(t, plan, "returned_copy", "$return")
	if copied.PlannedStorage == StorageFunctionTempRegion || copied.ActualLoweringStorage == StorageFunctionTempRegion {
		t.Fatalf("returned copy used dead function-temp region: %+v", copied)
	}
	if copied.PlannedStorage != StorageHeap || copied.Escape != EscapeReturn {
		t.Fatalf("returned copy = %+v, want heap return escape", copied)
	}
}

func TestPlannerDoesNotUseFunctionTempRegionForActorSend(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "send_msg",
		Values: []plir.Value{{
			ID:     "alloc_intent:msg",
			Kind:   plir.ValueAllocIntent,
			Type:   "[]u8",
			Region: "allocation:msg",
			Alloc: &plir.AllocIntent{
				ElementType:         "u8",
				ElementSize:         1,
				LengthExpr:          "16",
				LengthConstKnown:    true,
				LengthConst:         16,
				ZeroGuardStatus:     "valid_empty_no_allocator",
				NegativeGuardStatus: "reject_before_allocation",
				OverflowGuardStatus: "reject_before_allocation",
				Builtin:             "core.slice_copy_u8",
			},
			Provenance: plir.Provenance{Kind: plir.ProvenanceAllocation, Root: "msg"},
		}},
		Ops: []plir.Operation{{
			Kind:    plir.OpCall,
			Inputs:  []string{"msg"},
			Outputs: []string{"send_status"},
			Note:    "core.send actor boundary",
		}},
	}}}
	plan, err := FromPLIRWithOptions(prog, Options{EnableRegionPlanning: true})
	if err != nil {
		t.Fatalf("FromPLIRWithOptions: %v", err)
	}
	msg := findAllocation(t, plan, "send_msg", "msg")
	if msg.PlannedStorage == StorageRegion || msg.PlannedStorage == StorageFunctionTempRegion || msg.Escape != EscapeActor {
		t.Fatalf("actor send allocation = %+v, want non-region actor escape", msg)
	}
	if msg.PlannedStorage != StorageHeap {
		t.Fatalf("actor send storage = %s, want heap until transfer regions exist", msg.PlannedStorage)
	}
}

func TestPlannerDoesNotUseFunctionTempRegionForUnknownCallRetainedCopy(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "unknown_retained_copy",
		Values: []plir.Value{{
			ID:     "alloc_intent:copied",
			Kind:   plir.ValueAllocIntent,
			Type:   "[]u8",
			Region: "allocation:copied",
			Alloc: &plir.AllocIntent{
				ElementType:         "u8",
				ElementSize:         1,
				LengthExpr:          "n",
				ZeroGuardStatus:     "valid_empty_no_allocator",
				NegativeGuardStatus: "reject_before_allocation",
				OverflowGuardStatus: "reject_before_allocation",
				Builtin:             "core.slice_copy_u8",
			},
			Provenance: plir.Provenance{Kind: plir.ProvenanceAllocation, Root: "copied"},
		}},
		Ops: []plir.Operation{{
			Kind:   plir.OpCall,
			Inputs: []string{"copied"},
			Note:   "unknown external call may retain argument",
		}},
	}}}
	plan, err := FromPLIRWithOptions(prog, Options{EnableRegionPlanning: true, EnableRegionLowering: true})
	if err != nil {
		t.Fatalf("FromPLIRWithOptions: %v", err)
	}
	copied := findAllocation(t, plan, "unknown_retained_copy", "copied")
	if copied.Escape != EscapeCallUnknown || copied.PlannedStorage != StorageHeap || copied.ActualLoweringStorage != StorageHeap {
		t.Fatalf("unknown-call retained copy = %+v, want EscapesCallUnknown Heap/Heap fallback", copied)
	}
}

func TestPlannerStackLowersNonEscapingCopyOfFixedLocalView(t *testing.T) {
	plirProg, err := plir.FromCheckedProgram(checkedProgram(t, `
func local_copy() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    xs[0] = 20
    xs[1] = 22
    let copied: []u8 = xs.window(0, 2).copy()
    return copied[0] + copied[1]

func escaping_copy() -> []u8
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    return xs.window(0, 2).copy()

func dynamic_copy(i: Int) -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    xs[0] = 20
    xs[1] = 22
    let copied: []u8 = xs.window(0, 2).copy()
    return copied[i]

func aliased_copy() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    xs[0] = 20
    xs[1] = 22
    let copied: []u8 = xs.window(0, 2).copy()
    let alias: []u8 = copied
    return alias[0]

func raw_exposed_copy() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    xs[0] = 20
    xs[1] = 22
    let copied: []u8 = xs.window(0, 2).copy()
    let p: ptr = copied.ptr
    return copied[0]

func main() -> Int
uses alloc, mem:
    return local_copy()
`))
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	plan, err := FromPLIRWithOptions(plirProg, Options{EnableStackLowering: true})
	if err != nil {
		t.Fatalf("FromPLIRWithOptions: %v", err)
	}

	copied := findAllocation(t, plan, "local_copy", "copied")
	if copied.PlannedStorage != StorageEliminated || copied.ActualLoweringStorage != StorageEliminated {
		t.Fatalf("local copied planned/actual storage = %q/%q, want Eliminated/Eliminated: %+v", copied.PlannedStorage, copied.ActualLoweringStorage, copied)
	}
	if copied.LengthStatus != LengthStatusNormal || copied.ByteSize != 2 {
		t.Fatalf("local copied length/bytes = %q/%d, want normal bytes=2: %+v", copied.LengthStatus, copied.ByteSize, copied)
	}
	if copied.LoweringStatus != "scalar_replacement" || !strings.Contains(copied.Reason, "scalar_replacement_copy_fixed_constant_indices") {
		t.Fatalf("local copied lowering/reason = %q/%q, want scalar replacement copy evidence", copied.LoweringStatus, copied.Reason)
	}
	source := findAllocation(t, plan, "local_copy", "xs")
	if source.PlannedStorage != StorageStack || source.ActualLoweringStorage != StorageStack {
		t.Fatalf("local source planned/actual storage = %q/%q, want Stack/Stack:\n%s\nPLIR:\n%s", source.PlannedStorage, source.ActualLoweringStorage, FormatText(plan), plir.FormatText(plirProg))
	}

	ret := findAllocation(t, plan, "escaping_copy", "$return")
	if ret.PlannedStorage != StorageHeap || ret.ActualLoweringStorage != StorageHeap {
		t.Fatalf("escaping copy planned/actual storage = %q/%q, want Heap/Heap: %+v", ret.PlannedStorage, ret.ActualLoweringStorage, ret)
	}

	for _, fnName := range []string{"dynamic_copy", "aliased_copy", "raw_exposed_copy"} {
		copied := findAllocation(t, plan, fnName, "copied")
		if copied.PlannedStorage == StorageEliminated || copied.ActualLoweringStorage == StorageEliminated {
			t.Fatalf("%s copied allocation was scalar-eliminated despite dynamic/alias/raw exposure: %+v", fnName, copied)
		}
		if copied.PlannedStorage != StorageStack || copied.ActualLoweringStorage != StorageStack {
			t.Fatalf("%s copied planned/actual storage = %q/%q, want Stack/Stack fallback: %+v", fnName, copied.PlannedStorage, copied.ActualLoweringStorage, copied)
		}
	}
}

func TestPlannerKeepsEscapingCopyFromIslandAsOwnedHeap(t *testing.T) {
	plirProg, err := plir.FromCheckedProgram(checkedProgram(t, `
func copy_from_island(flag: Int) -> []u8
uses alloc, islands, mem:
    if flag:
        island(64) as isl:
            var xs: []u8 = core.island_make_u8(isl, 4)
            xs[0] = 7
            return xs.window(0, 2).copy()
    var fallback: []u8 = make_u8(1)
    return fallback

func main() -> Int
uses alloc, islands, mem:
    return 0
`))
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	plan, err := FromPLIRWithOptions(plirProg, Options{EnableStackLowering: true})
	if err != nil {
		t.Fatalf("FromPLIRWithOptions: %v", err)
	}

	island := findAllocation(t, plan, "copy_from_island", "xs")
	if island.PlannedStorage != StorageExplicitIsland || island.ActualLoweringStorage != StorageExplicitIsland {
		t.Fatalf("island planned/actual storage = %q/%q, want ExplicitIsland/ExplicitIsland: %+v", island.PlannedStorage, island.ActualLoweringStorage, island)
	}
	if island.ValidationStatus != "validated_explicit_island_scope" || island.LoweringStatus != "explicit_island_lowering" {
		t.Fatalf("island validation/lowering status = %q/%q, want explicit island evidence", island.ValidationStatus, island.LoweringStatus)
	}
	copied := findAllocation(t, plan, "copy_from_island", "$return")
	if copied.PlannedStorage != StorageHeap || copied.ActualLoweringStorage != StorageHeap {
		t.Fatalf("copy planned/actual storage = %q/%q, want Heap/Heap: %+v", copied.PlannedStorage, copied.ActualLoweringStorage, copied)
	}
	if copied.Builtin != "core.slice_copy_u8" || copied.Escape != EscapeReturn {
		t.Fatalf("copy allocation = %+v, want returned owned slice copy", copied)
	}
}

func TestPlannerEliminatesScalarReplacedTinyConstantIndexSlice(t *testing.T) {
	plirProg, err := plir.FromCheckedProgram(checkedProgram(t, `
func scalar() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 20
    xs[1] = 22
    return xs[0] + xs[1]

func dynamic(i: Int) -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[i] = 20
    return xs[0]

func printed() -> Int
uses alloc, io, mem:
    var xs: []u8 = make_u8(2)
    xs[0] = 65
    xs[1] = 66
    print(xs)
    return 0

func main() -> Int
uses alloc, mem:
    return scalar()
`))
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	plan, err := FromPLIRWithOptions(plirProg, Options{EnableStackLowering: true})
	if err != nil {
		t.Fatalf("FromPLIRWithOptions: %v", err)
	}

	scalar := findAllocation(t, plan, "scalar", "xs")
	if scalar.PlannedStorage != StorageEliminated || scalar.ActualLoweringStorage != StorageEliminated {
		t.Fatalf("scalar planned/actual storage = %q/%q, want Eliminated/Eliminated: %+v", scalar.PlannedStorage, scalar.ActualLoweringStorage, scalar)
	}
	if scalar.LoweringStatus != "scalar_replacement" || !strings.Contains(scalar.Reason, "scalar_replacement_fixed_constant_indices") {
		t.Fatalf("scalar lowering/reason = %q/%q, want scalar replacement evidence", scalar.LoweringStatus, scalar.Reason)
	}

	dynamic := findAllocation(t, plan, "dynamic", "xs")
	if dynamic.PlannedStorage == StorageEliminated || dynamic.ActualLoweringStorage == StorageEliminated {
		t.Fatalf("dynamic-index allocation was scalar-eliminated: %+v", dynamic)
	}
	if dynamic.PlannedStorage != StorageStack || dynamic.ActualLoweringStorage != StorageStack {
		t.Fatalf("dynamic planned/actual storage = %q/%q, want Stack/Stack fallback:\n%s", dynamic.PlannedStorage, dynamic.ActualLoweringStorage, FormatText(plan))
	}

	printed := findAllocation(t, plan, "printed", "xs")
	if printed.PlannedStorage == StorageEliminated || printed.ActualLoweringStorage == StorageEliminated {
		t.Fatalf("printed slice was scalar-eliminated despite observable slice use: %+v", printed)
	}
	if printed.PlannedStorage != StorageStack || printed.ActualLoweringStorage != StorageStack {
		t.Fatalf("printed planned/actual storage = %q/%q, want Stack/Stack fallback:\n%s", printed.PlannedStorage, printed.ActualLoweringStorage, FormatText(plan))
	}
}

func TestPlannerClassifiesUnknownCallUnsafeAndIslandAllocations(t *testing.T) {
	plan := allocationPlan(t, `
func consume(xs: []u8) -> Int
uses mem:
    return xs.len

func call_unknown() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    return consume(xs)

func unsafe_boundary() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    unsafe:
        var y = 1
    return xs[0]

func islanded() -> Int
uses alloc, islands, mem:
    island(64) as isl:
        var xs: []u8 = core.island_make_u8(isl, 4)
        xs[0] = 1
    return 0

func main() -> Int
uses alloc, islands, mem:
    return call_unknown() + unsafe_boundary() + islanded()
`)

	call := findAllocation(t, plan, "call_unknown", "xs")
	if call.Escape != EscapeCallUnknown || call.Storage != StorageHeap {
		t.Fatalf("unknown-call allocation = %+v, want EscapesCallUnknown/Heap", call)
	}
	unsafeAlloc := findAllocation(t, plan, "unsafe_boundary", "xs")
	if unsafeAlloc.Escape != EscapeUnsafe || unsafeAlloc.Storage != StorageHeap {
		t.Fatalf("unsafe allocation = %+v, want EscapesUnsafe/Heap", unsafeAlloc)
	}
	island := findAllocation(t, plan, "islanded", "xs")
	if island.Escape != EscapeNoEscape || island.Storage != StorageExplicitIsland {
		t.Fatalf("island allocation = %+v, want NoEscape/ExplicitIsland", island)
	}
	dump := FormatText(plan)
	for _, want := range []string{"escape: EscapesCallUnknown", "planned_storage: ExplicitIsland", "actual_lowering_storage: ExplicitIsland"} {
		if !strings.Contains(dump, want) {
			t.Fatalf("FormatText missing %q:\n%s", want, dump)
		}
	}
}

func TestPlannerClassifiesExpandedEscapeKinds(t *testing.T) {
	plan := allocationPlanFile(t, `
struct Box:
    buf: []u8

var stored: []u8

func global_store() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    stored = xs
    return 0

func local_box() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    let box: Box = Box(buf: xs)
    return box.buf.len

func main() -> Int
uses alloc, mem:
    return global_store() + local_box()
`)

	global := findAllocation(t, plan, "global_store", "xs")
	if global.Escape != EscapeGlobal || global.Storage != StorageHeap {
		t.Fatalf("global store allocation = %+v, want EscapesGlobal/Heap", global)
	}
	aggregate := findAllocation(t, plan, "local_box", "xs")
	if aggregate.Escape != EscapeAggregate || aggregate.Storage != StorageHeap {
		t.Fatalf("aggregate allocation = %+v, want EscapesAggregate/Heap", aggregate)
	}
}

func TestPlannerClassifiesSyntheticBoundaryEscapeKinds(t *testing.T) {
	plan, err := FromPLIR(&plir.Program{Funcs: []plir.Function{
		syntheticEscapeFunction("closure", plir.Operation{Kind: plir.OpClosure, Inputs: []string{"xs"}, Outputs: []string{"f"}, Note: "closure captures environment"}),
		syntheticEscapeFunction("task", plir.Operation{Kind: plir.OpCall, Inputs: []string{"xs"}, Note: "core.task_spawn_i32_typed captures payload"}),
		syntheticEscapeFunction("actor", plir.Operation{Kind: plir.OpCall, Inputs: []string{"xs"}, Note: "core.send_typed sends actor payload"}),
	}})
	if err != nil {
		t.Fatalf("FromPLIR: %v", err)
	}

	closure := findAllocation(t, plan, "closure", "xs")
	if closure.Escape != EscapeClosure || closure.Storage != StorageHeap {
		t.Fatalf("closure allocation = %+v, want EscapesClosure/Heap", closure)
	}
	task := findAllocation(t, plan, "task", "xs")
	if task.Escape != EscapeTask || task.Storage != StorageHeap {
		t.Fatalf("task allocation = %+v, want EscapesTask/Heap", task)
	}
	actor := findAllocation(t, plan, "actor", "xs")
	if actor.Escape != EscapeActor || actor.Storage != StorageHeap {
		t.Fatalf("actor allocation = %+v, want EscapesActor/Heap", actor)
	}
}

func syntheticEscapeFunction(name string, op plir.Operation) plir.Function {
	op.ID = "op1"
	op.Block = "entry"
	return plir.Function{
		Name: name,
		Values: []plir.Value{{
			ID:     "alloc_intent:xs",
			Kind:   plir.ValueAllocIntent,
			Type:   "[]u8",
			Source: "test:1:1",
			Alloc: &plir.AllocIntent{
				ElementType:         "u8",
				ElementSize:         1,
				LengthExpr:          "4",
				ZeroGuardStatus:     "valid_empty_no_allocator",
				NegativeGuardStatus: "reject_before_allocation",
				OverflowGuardStatus: "reject_before_allocation",
				Builtin:             "core.make_u8",
			},
			Provenance: plir.Provenance{Kind: plir.ProvenanceAllocation, Root: "xs"},
		}},
		Ops: []plir.Operation{
			{ID: "op0", Kind: plir.OpAllocIntent, Block: "entry", Outputs: []string{"alloc_intent:xs"}, Note: "make<u8>"},
			op,
		},
		Blocks: []plir.BasicBlock{{ID: "entry", Kind: "entry", Entry: true, Ops: []string{"op0", "op1"}, Exit: true}},
	}
}

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
				t.Fatalf("VerifyPlan error = %v, want actual lowering storage escape rejection", err)
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
		{name: "function_temp_region", storage: StorageFunctionTempRegion, status: "validated_heap_fallback"},
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

	err := verifySingleAllocation(alloc)
	if err == nil || !strings.Contains(err.Error(), "reason") {
		t.Fatalf("VerifyPlan error = %v, want missing reason rejection", err)
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

func TestPlannerReportsAllocationLengthContractStatuses(t *testing.T) {
	plan := allocationPlan(t, `
func empty() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(0)
    return xs.len

func normal() -> Int
uses alloc, mem:
    var xs: []u16 = make_u16(3)
    return xs.len

func negative() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(0 - 1)
    return xs.len

func overflow() -> Int
uses alloc, mem:
    var xs: []bool = make_bool(536870912)
    return xs.len

func main() -> Int
uses alloc, mem:
    return 0
`)

	empty := findAllocation(t, plan, "empty", "xs")
	if empty.LengthStatus != LengthStatusValidEmpty || empty.Storage != StorageEliminated {
		t.Fatalf("empty allocation = %+v, want valid empty eliminated allocation", empty)
	}
	if empty.ZeroGuardStatus != "valid_empty_no_allocator" || empty.NegativeGuardStatus != "reject_before_allocation" || empty.OverflowGuardStatus != "reject_before_allocation" {
		t.Fatalf("empty guard status = %+v", empty)
	}
	normal := findAllocation(t, plan, "normal", "xs")
	if normal.LengthStatus != LengthStatusNormal || normal.ByteSize != 6 {
		t.Fatalf("normal allocation = %+v, want normal bytes=6", normal)
	}
	negative := findAllocation(t, plan, "negative", "xs")
	if negative.LengthStatus != LengthStatusRejectedNegative {
		t.Fatalf("negative allocation = %+v, want rejected negative length", negative)
	}
	overflow := findAllocation(t, plan, "overflow", "xs")
	if overflow.LengthStatus != LengthStatusRejectedOverflow {
		t.Fatalf("overflow allocation = %+v, want rejected byte-size overflow", overflow)
	}

	dump := FormatText(plan)
	for _, want := range []string{"length_status: valid_empty_allocation", "length_status: rejected_negative_length", "length_status: rejected_byte_size_overflow", "zero_guard: valid_empty_no_allocator"} {
		if !strings.Contains(dump, want) {
			t.Fatalf("FormatText missing %q:\n%s", want, dump)
		}
	}
}

func TestPlannerElidesUnusedCopyAllocationIntent(t *testing.T) {
	plirProg, err := plir.FromCheckedProgram(checkedProgram(t, `
func unused_copy(xs: []u8) -> Int
uses alloc, mem:
    let unused: []u8 = xs.copy()
    return xs.len

func used_copy(xs: []u8) -> Int
uses alloc, mem:
    let copied: []u8 = xs.copy()
    return copied.len

func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(1)
    return unused_copy(xs) + used_copy(xs)
`))
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	plan, err := FromPLIRWithOptions(plirProg, Options{EnableStackLowering: true})
	if err != nil {
		t.Fatalf("FromPLIRWithOptions: %v", err)
	}

	unused := findAllocation(t, plan, "unused_copy", "unused")
	if unused.Escape != EscapeNoEscape || unused.Storage != StorageEliminated || unused.ActualLoweringStorage != StorageEliminated {
		t.Fatalf("unused copy allocation = %+v, want NoEscape/Eliminated actual Eliminated", unused)
	}
	if unused.LoweringStatus != "eliminated_unused_copy" {
		t.Fatalf("unused copy lowering status = %q, want eliminated_unused_copy", unused.LoweringStatus)
	}
	if !strings.Contains(unused.Reason, "copy result is unused") {
		t.Fatalf("unused copy reason = %q", unused.Reason)
	}
	used := findAllocation(t, plan, "used_copy", "copied")
	if used.Storage == StorageEliminated || used.ActualLoweringStorage == StorageEliminated {
		t.Fatalf("used copy allocation = %+v, must not be eliminated", used)
	}
}

func findAllocation(t *testing.T, plan *Plan, fnName string, allocID string) Allocation {
	t.Helper()
	for _, fn := range plan.Functions {
		if fn.Name != fnName {
			continue
		}
		for _, alloc := range fn.Allocations {
			if alloc.ID == allocID {
				return alloc
			}
		}
	}
	t.Fatalf("missing allocation %s in function %s: %+v", allocID, fnName, plan)
	return Allocation{}
}
