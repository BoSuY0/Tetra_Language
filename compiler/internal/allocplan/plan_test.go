package allocplan

import (
	"strconv"
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
		t.Fatalf(
			"local planned/actual storage = %q/%q, want Stack/Heap",
			local.PlannedStorage,
			local.ActualLoweringStorage,
		)
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
		t.Fatalf(
			"local planned/actual storage = %q/%q, want Stack/Stack",
			local.PlannedStorage,
			local.ActualLoweringStorage,
		)
	}
	if local.LoweringStatus != "stack_lowering" ||
		!strings.Contains(local.Reason, "fixed_i32_no_escape") {
		t.Fatalf(
			"local lowering status/reason = %q/%q, want stack lowering evidence",
			local.LoweringStatus,
			local.Reason,
		)
	}

	ret := findAllocation(t, plan, "ret", "xs")
	if ret.PlannedStorage != StorageHeap || ret.ActualLoweringStorage != StorageHeap {
		t.Fatalf(
			"returned planned/actual storage = %q/%q, want Heap/Heap",
			ret.PlannedStorage,
			ret.ActualLoweringStorage,
		)
	}
}

func TestPlannerLargeNoEscapeI32SliceUsesStackWhenStackLoweringEnabled(t *testing.T) {
	for _, tc := range []struct {
		name string
		make string
	}{
		{name: "unqualified", make: "make_i32(4096)"},
		{name: "core", make: "core.make_i32(4096)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := allocationPlanWithOptions(t, `
func main() -> Int
uses alloc, mem:
    var xs: []i32 = `+tc.make+`
    xs[0] = 1
    return xs[0]
`, Options{EnableStackLowering: true})

			xs := findAllocation(t, plan, "main", "xs")
			if xs.Escape != EscapeNoEscape {
				t.Fatalf("large i32 allocation escape = %q, want NoEscape: %+v", xs.Escape, xs)
			}
			if xs.ByteSize != 16384 {
				t.Fatalf("large i32 allocation bytes = %d, want 16384: %+v", xs.ByteSize, xs)
			}
			if xs.PlannedStorage != StorageStack || xs.ActualLoweringStorage != StorageStack {
				t.Fatalf(
					"large i32 planned/actual storage = %q/%q, want Stack/Stack: %+v",
					xs.PlannedStorage,
					xs.ActualLoweringStorage,
					xs,
				)
			}
			if got := RuntimePathForAllocation(xs); got != runtimeabi.AllocationPathStackFrame {
				t.Fatalf("large i32 runtime path = %q, want stack_frame: %+v", got, xs)
			}
			if contains(xs.HeapReasonCodes, HeapReasonLargeObject) ||
				contains(xs.ReasonCodes, HeapReasonLargeObject) {
				t.Fatalf(
					"large i32 stack allocation reported heap large-object reason: heap=%v reason=%v alloc=%+v",
					xs.HeapReasonCodes,
					xs.ReasonCodes,
					xs,
				)
			}
		})
	}
}

func TestPlannerLargeNoEscapeU8RemainsHeapWithStackLowering(t *testing.T) {
	plan := allocationPlanWithOptions(t, `
func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(5000)
    xs[0] = 1
    return xs[0]
`, Options{EnableStackLowering: true, EnableSmallHeapRuntime: true})

	xs := findAllocation(t, plan, "main", "xs")
	if xs.Escape != EscapeNoEscape {
		t.Fatalf("large u8 allocation escape = %q, want NoEscape: %+v", xs.Escape, xs)
	}
	if xs.PlannedStorage != StorageHeap || xs.ActualLoweringStorage != StorageHeap {
		t.Fatalf(
			"large u8 planned/actual storage = %q/%q, want Heap/Heap: %+v",
			xs.PlannedStorage,
			xs.ActualLoweringStorage,
			xs,
		)
	}
	if got := RuntimePathForAllocation(xs); got != runtimeabi.AllocationPathLargeMmap {
		t.Fatalf("large u8 runtime path = %q, want large_mmap: %+v", got, xs)
	}
	if !contains(xs.HeapReasonCodes, HeapReasonLargeObject) {
		t.Fatalf(
			"large u8 heap reason codes = %v, want %s: %+v",
			xs.HeapReasonCodes,
			HeapReasonLargeObject,
			xs,
		)
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
	if small.RuntimePath != runtimeabi.AllocationPathProcessBumpSmallHeapV0 ||
		small.AllocatorClass != "small_32" {
		t.Fatalf(
			"small allocation runtime evidence = %+v, want process_bump_small_heap_v0/small_32",
			small,
		)
	}
	if small.BytesRequested != 17 || small.BytesReserved != 32 {
		t.Fatalf(
			"small allocation bytes = requested %d reserved %d, want 17/32",
			small.BytesRequested,
			small.BytesReserved,
		)
	}
	if small.BytesCommitted != 32 || small.BytesReleased != 0 {
		t.Fatalf(
			"small allocation backend bytes = committed %d released %d, want 32/0",
			small.BytesCommitted,
			small.BytesReleased,
		)
	}
	if small.MemoryBackend == nil ||
		small.MemoryBackend.BackendClass != runtimeabi.MemoryBackendClassSmallHeap ||
		small.MemoryBackend.Adapter != "runtime.small_heap.process_bump_v0" ||
		small.MemoryBackend.EvidenceClass != runtimeabi.MemoryFootprintEstimated {
		t.Fatalf(
			"small allocation memory backend evidence = %+v, want process bump small heap estimate",
			small.MemoryBackend,
		)
	}
	if small.Domain == nil || small.Domain.DomainID != "domain:process" ||
		small.Domain.Kind != runtimeabi.DomainProcess {
		t.Fatalf("small allocation domain = %+v, want process domain", small.Domain)
	}
	large := findAllocation(t, plan, "ret_large", "xs")
	if large.RuntimePath != runtimeabi.AllocationPathLargeMmap ||
		large.AllocatorClass != "large_mmap" {
		t.Fatalf("large allocation runtime evidence = %+v, want large_mmap", large)
	}
	if large.BytesRequested != 5000 || large.BytesReserved != 5000 {
		t.Fatalf(
			"large allocation bytes = requested %d reserved %d, want 5000/5000",
			large.BytesRequested,
			large.BytesReserved,
		)
	}
	if large.MemoryBackend == nil ||
		large.MemoryBackend.BackendClass != runtimeabi.MemoryBackendClassLargeBackend ||
		large.MemoryBackend.Adapter != "target.large_mmap_v1" ||
		large.MemoryBackend.EvidenceClass != runtimeabi.MemoryFootprintEstimated ||
		large.MemoryBackend.ReleaseBytes != 0 {
		t.Fatalf(
			"large allocation memory backend evidence = %+v, want large mmap reserve/commit estimate without release",
			large.MemoryBackend,
		)
	}
	if !strings.Contains(FormatText(plan), "allocator_class: small_32") {
		t.Fatalf("FormatText missing small heap allocator class:\n%s", FormatText(plan))
	}
}

func TestPlannerReportsProcessBumpSmallHeapAllocatorEvidence(t *testing.T) {
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
	if small.RuntimePath != runtimeabi.AllocationPathProcessBumpSmallHeapV0 {
		t.Fatalf(
			"small allocation runtime_path = %q, want %q: %+v",
			small.RuntimePath,
			runtimeabi.AllocationPathProcessBumpSmallHeapV0,
			small,
		)
	}
	if small.AllocatorClass != "small_32" {
		t.Fatalf(
			"small allocation allocator_class = %q, want small_32: %+v",
			small.AllocatorClass,
			small,
		)
	}
	if small.AllocatorScope != "process" {
		t.Fatalf(
			"small allocation allocator_scope = %q, want process: %+v",
			small.AllocatorScope,
			small,
		)
	}
	if small.AllocatorReusePolicy != "bump_no_reuse_v0" {
		t.Fatalf(
			"small allocation reuse policy = %q, want bump_no_reuse_v0: %+v",
			small.AllocatorReusePolicy,
			small,
		)
	}
	if small.AllocatorChunkBytes != runtimeabi.SmallHeapChunkBytes {
		t.Fatalf(
			"small allocation chunk bytes = %d, want %d: %+v",
			small.AllocatorChunkBytes,
			runtimeabi.SmallHeapChunkBytes,
			small,
		)
	}

	summary := Summarize(plan)
	if summary.RuntimePaths[string(runtimeabi.AllocationPathProcessBumpSmallHeapV0)] != 1 {
		t.Fatalf(
			"runtime path summary = %+v, want process_bump_small_heap_v0 count",
			summary.RuntimePaths,
		)
	}
	if summary.AllocatorClasses["small_32"] != 1 {
		t.Fatalf("allocator class summary = %+v, want small_32 count", summary.AllocatorClasses)
	}
	if summary.AllocatorScopes["process"] != 1 {
		t.Fatalf("allocator scope summary = %+v, want process count", summary.AllocatorScopes)
	}
	if summary.AllocatorReusePolicies["bump_no_reuse_v0"] != 1 {
		t.Fatalf(
			"allocator reuse summary = %+v, want bump_no_reuse_v0 count",
			summary.AllocatorReusePolicies,
		)
	}
	if summary.BytesCommitted != 32 || summary.BytesReleased != 0 {
		t.Fatalf(
			"backend byte summary = committed %d released %d, want 32/0",
			summary.BytesCommitted,
			summary.BytesReleased,
		)
	}
	if summary.MemoryBackendClasses[string(runtimeabi.MemoryBackendClassSmallHeap)] != 1 {
		t.Fatalf(
			"memory backend class summary = %+v, want small_heap count",
			summary.MemoryBackendClasses,
		)
	}
	if summary.MemoryBackendOperations[string(runtimeabi.MemoryBackendCommit)] != 1 ||
		summary.MemoryBackendOperations[string(runtimeabi.MemoryBackendFootprint)] != 1 {
		t.Fatalf(
			"memory backend operation summary = %+v, want commit/footprint counts without release",
			summary.MemoryBackendOperations,
		)
	}
	if len(summary.Domains) != 1 || summary.Domains[0].DomainID != "domain:process" ||
		summary.Domains[0].RequestedBytes != 17 {
		t.Fatalf("domain summary = %+v, want process domain accounting", summary.Domains)
	}
	text := FormatText(plan)
	for _, want := range []string{
		"runtime_path: process_bump_small_heap_v0",
		"allocator_class: small_32",
		"allocator_scope: process",
		"allocator_reuse_policy: bump_no_reuse_v0",
		"allocator_chunk_bytes: 65536",
		"memory_backend: small_heap",
		"memory_backend_ops: commit,footprint,reserve",
		"bytes_committed: 32",
		"domain_id: domain:process",
		"memory_backend_classes:small_heap=1",
		"memory_backend_operations:commit=1,footprint=1,reserve=1",
		"allocator_reuse_policies:bump_no_reuse_v0=1",
		"domains:domain:process=17/32",
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
		t.Fatalf(
			"raw pointer bounds status = %q, want allocation-base metadata: %+v",
			raw.RawPointerBoundsStatus,
			raw,
		)
	}
	if raw.RawPointerBaseID != "p" || raw.RawPointerBaseBytes != 24 ||
		raw.RawPointerOffsetBytes != 0 {
		t.Fatalf(
			"raw pointer base metadata = id %q bytes %d offset %d, want p/24/0: %+v",
			raw.RawPointerBaseID,
			raw.RawPointerBaseBytes,
			raw.RawPointerOffsetBytes,
			raw,
		)
	}
	if raw.RawSlicePolicy != string(runtimeabi.RawSliceBoundsExternalUnknown) {
		t.Fatalf(
			"raw slice policy = %q, want external unknown until verified root construction: %+v",
			raw.RawSlicePolicy,
			raw,
		)
	}
	if raw.RuntimePath != runtimeabi.AllocationPathProcessBumpSmallHeapV0 {
		t.Fatalf(
			("raw allocation runtime path = %q, want small heap optimization " +
				"to be visible without trusting arbitrary pointers: %+v"),
			raw.RuntimePath,
			raw,
		)
	}

	summary := Summarize(plan)
	if summary.RawPointerBoundsStatuses[string(runtimeabi.RawPointerBoundsAllocationBase)] != 1 {
		t.Fatalf(
			"raw pointer bounds summary = %+v, want allocation_base_metadata count",
			summary.RawPointerBoundsStatuses,
		)
	}
	if summary.RawSlicePolicies[string(runtimeabi.RawSliceBoundsExternalUnknown)] != 1 {
		t.Fatalf(
			"raw slice policy summary = %+v, want external_unknown count",
			summary.RawSlicePolicies,
		)
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
	if island.Storage != StorageExplicitIsland ||
		island.ActualLoweringStorage != StorageExplicitIsland {
		t.Fatalf(
			"island storage = %s/%s, want explicit island lowering",
			island.Storage,
			island.ActualLoweringStorage,
		)
	}
	if island.RuntimePath != runtimeabi.AllocationPathExplicitIsland ||
		island.AllocatorClass != "region_bump_16" {
		t.Fatalf("island runtime evidence = %+v, want explicit_island/region_bump_16", island)
	}
	if island.BytesRequested != 17 || island.BytesReserved != 32 {
		t.Fatalf(
			"island bytes = requested %d reserved %d, want 17/32",
			island.BytesRequested,
			island.BytesReserved,
		)
	}
	if island.BytesCommitted != 32 || island.BytesReleased != 32 {
		t.Fatalf(
			"island backend bytes = committed %d released %d, want 32/32",
			island.BytesCommitted,
			island.BytesReleased,
		)
	}
	if island.MemoryBackend == nil ||
		island.MemoryBackend.BackendClass != runtimeabi.MemoryBackendClassRegion ||
		island.MemoryBackend.Adapter != "runtime.region_bump_v1" ||
		island.MemoryBackend.EvidenceClass != runtimeabi.MemoryFootprintEstimated {
		t.Fatalf(
			"island memory backend evidence = %+v, want region allocation estimate",
			island.MemoryBackend,
		)
	}
	if island.RegionID != "island:isl" || island.Lifetime == "" || island.DebugMode == "" {
		t.Fatalf("island report hooks missing region/lifetime/debug evidence: %+v", island)
	}
	if island.Domain == nil || island.Domain.DomainID != "domain:island:isl" ||
		island.Domain.Kind != runtimeabi.DomainIsland {
		t.Fatalf("island domain = %+v, want explicit island domain", island.Domain)
	}
	text := FormatText(plan)
	for _, want := range []string{
		"runtime_path: explicit_island",
		"allocator_class: region_bump_16",
		"memory_backend: region",
		"region_id: island:isl",
		"bytes_reserved: 32",
		"bytes_committed: 32",
		"bytes_released: 32",
		"domain_id: domain:island:isl",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("FormatText missing %q:\n%s", want, text)
		}
	}
}

func TestPlannerRecordsExplicitIslandHandleParamSlot(t *testing.T) {
	plan := allocationPlan(t, `
func make_buf(prefix: []u8, isl: island, n: Int) -> []u8
uses alloc, islands, mem:
    var buf: []u8 = core.island_make_u8(isl, n)
    buf[0] = prefix[0]
    return buf

func main() -> Int
uses alloc, islands, mem:
    var src: []u8 = make_u8(1)
    src[0] = 9
    island(64) as isl:
        var out: []u8 = make_buf(src, isl, 4)
        out[0] = 1
        return out[0]
    return 0
`)

	island := findAllocation(t, plan, "make_buf", "buf")
	if !island.ExplicitIslandHandleParamSlotKnown || island.ExplicitIslandHandleParamSlot != 2 {
		t.Fatalf(
			"island handle param slot = known:%v slot:%d, want known slot 2: %+v",
			island.ExplicitIslandHandleParamSlotKnown,
			island.ExplicitIslandHandleParamSlot,
			island,
		)
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
		if alloc.RuntimePath != runtimeabi.AllocationPathExplicitIsland ||
			alloc.RegionID != "island:isl" ||
			alloc.Lifetime == "" {
			t.Fatalf(
				"%s explicit island metadata = %+v, want runtime path, region, and lifetime",
				tc.fn,
				alloc,
			)
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
		t.Fatalf(
			"temporary copy planned/actual storage = %q/%q, want FunctionTempRegion/Heap fallback: %+v",
			copied.PlannedStorage,
			copied.ActualLoweringStorage,
			copied,
		)
	}
	if copied.RuntimePath != runtimeabi.AllocationPathHeap ||
		copied.RegionID != "region:local_copy:temp" {
		t.Fatalf(
			"temporary copy fallback evidence = %+v, want heap runtime path with planned function temp id",
			copied,
		)
	}
	if copied.AllocatorClass != "" {
		t.Fatalf(
			"temporary copy fallback allocator class = %q, want no region allocator claim: %+v",
			copied.AllocatorClass,
			copied,
		)
	}
	if copied.MemoryBackend == nil ||
		copied.MemoryBackend.BackendClass != runtimeabi.MemoryBackendClassConservativeHeap ||
		copied.MemoryBackend.EvidenceClass != runtimeabi.MemoryFootprintBlocked ||
		copied.MemoryBackend.BlockedReason == "" {
		t.Fatalf(
			"temporary copy fallback memory backend evidence = %+v, want blocked conservative heap",
			copied.MemoryBackend,
		)
	}
	if copied.Lifetime != "function:local_copy" || copied.DebugMode == "" {
		t.Fatalf("temporary copy lifetime/debug evidence = %+v", copied)
	}
	text := FormatText(plan)
	for _, want := range []string{
		"planned_storage: FunctionTempRegion",
		"actual_lowering_storage: Heap",
		"runtime_path: heap",
		"backend_storage: Heap",
		"region_id: region:local_copy:temp",
		"reason: function-local temporary copy",
	} {
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
		t.Fatalf(
			"temporary copy planned/actual storage = %q/%q, want FunctionTempRegion/FunctionTempRegion: %+v",
			copied.PlannedStorage,
			copied.ActualLoweringStorage,
			copied,
		)
	}
	if copied.LoweringStatus != "function_temp_region_lowering" {
		t.Fatalf(
			"temporary copy lowering status = %q, want function_temp_region_lowering: %+v",
			copied.LoweringStatus,
			copied,
		)
	}
	if copied.RuntimePath != runtimeabi.AllocationPathScopedSingleMappingV0 ||
		copied.AllocatorClass != "function_temp_region" ||
		copied.RegionID != "region:local_copy:temp" {
		t.Fatalf("temporary copy region report evidence = %+v, want function temp region", copied)
	}
	if copied.MemoryBackend == nil ||
		copied.MemoryBackend.BackendClass != runtimeabi.MemoryBackendClassRegion ||
		copied.MemoryBackend.Adapter != "runtime.region_bump_v1" ||
		copied.MemoryBackend.EvidenceClass != runtimeabi.MemoryFootprintEstimated {
		t.Fatalf(
			"temporary copy region memory backend evidence = %+v, want region estimate",
			copied.MemoryBackend,
		)
	}
	text := FormatText(plan)
	for _, want := range []string{
		"planned_storage: FunctionTempRegion",
		"actual_lowering_storage: FunctionTempRegion",
		"runtime_path: scoped_single_mapping_v0",
		"allocator_class: function_temp_region",
		"memory_backend: region",
		"function_temp_region:1",
	} {
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
		t.Fatalf(
			"temporary copy planned/actual storage = %q/%q, want FunctionTempRegion/Heap fallback: %+v",
			copied.PlannedStorage,
			copied.ActualLoweringStorage,
			copied,
		)
	}
	if copied.RuntimePath != runtimeabi.AllocationPathHeap || copied.AllocatorClass != "" {
		t.Fatalf(
			("temporary copy heap fallback evidence = %+v, want heap runtime " +
				"path without region allocator class"),
			copied,
		)
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
	if copiedA.PlannedStorage != StorageFunctionTempRegion ||
		copiedA.ActualLoweringStorage != StorageFunctionTempRegion {
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
	if copied.PlannedStorage == StorageFunctionTempRegion ||
		copied.ActualLoweringStorage == StorageFunctionTempRegion {
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
	if msg.PlannedStorage == StorageRegion || msg.PlannedStorage == StorageFunctionTempRegion ||
		msg.Escape != EscapeActor {
		t.Fatalf("actor send allocation = %+v, want non-region actor escape", msg)
	}
	if msg.PlannedStorage != StorageHeap {
		t.Fatalf(
			"actor send storage = %s, want heap until transfer regions exist",
			msg.PlannedStorage,
		)
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
	plan, err := FromPLIRWithOptions(
		prog,
		Options{EnableRegionPlanning: true, EnableRegionLowering: true},
	)
	if err != nil {
		t.Fatalf("FromPLIRWithOptions: %v", err)
	}
	copied := findAllocation(t, plan, "unknown_retained_copy", "copied")
	if copied.Escape != EscapeCallUnknown || copied.PlannedStorage != StorageHeap ||
		copied.ActualLoweringStorage != StorageHeap {
		t.Fatalf(
			"unknown-call retained copy = %+v, want EscapesCallUnknown Heap/Heap fallback",
			copied,
		)
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
	if copied.PlannedStorage != StorageEliminated ||
		copied.ActualLoweringStorage != StorageEliminated {
		t.Fatalf(
			"local copied planned/actual storage = %q/%q, want Eliminated/Eliminated: %+v",
			copied.PlannedStorage,
			copied.ActualLoweringStorage,
			copied,
		)
	}
	if copied.LengthStatus != LengthStatusNormal || copied.ByteSize != 2 {
		t.Fatalf(
			"local copied length/bytes = %q/%d, want normal bytes=2: %+v",
			copied.LengthStatus,
			copied.ByteSize,
			copied,
		)
	}
	if copied.LoweringStatus != "scalar_replacement" ||
		!strings.Contains(copied.Reason, "scalar_replacement_copy_fixed_constant_indices") {
		t.Fatalf(
			"local copied lowering/reason = %q/%q, want scalar replacement copy evidence",
			copied.LoweringStatus,
			copied.Reason,
		)
	}
	source := findAllocation(t, plan, "local_copy", "xs")
	if source.PlannedStorage != StorageStack || source.ActualLoweringStorage != StorageStack {
		t.Fatalf(
			"local source planned/actual storage = %q/%q, want Stack/Stack:\n%s\nPLIR:\n%s",
			source.PlannedStorage,
			source.ActualLoweringStorage,
			FormatText(plan),
			plir.FormatText(plirProg),
		)
	}

	ret := findAllocation(t, plan, "escaping_copy", "$return")
	if ret.PlannedStorage != StorageHeap || ret.ActualLoweringStorage != StorageHeap {
		t.Fatalf(
			"escaping copy planned/actual storage = %q/%q, want Heap/Heap: %+v",
			ret.PlannedStorage,
			ret.ActualLoweringStorage,
			ret,
		)
	}

	for _, fnName := range []string{"dynamic_copy", "aliased_copy", "raw_exposed_copy"} {
		copied := findAllocation(t, plan, fnName, "copied")
		if copied.PlannedStorage == StorageEliminated ||
			copied.ActualLoweringStorage == StorageEliminated {
			t.Fatalf(
				"%s copied allocation was scalar-eliminated despite dynamic/alias/raw exposure: %+v",
				fnName,
				copied,
			)
		}
		if copied.PlannedStorage != StorageStack || copied.ActualLoweringStorage != StorageStack {
			t.Fatalf(
				"%s copied planned/actual storage = %q/%q, want Stack/Stack fallback: %+v",
				fnName,
				copied.PlannedStorage,
				copied.ActualLoweringStorage,
				copied,
			)
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
	if island.PlannedStorage != StorageExplicitIsland ||
		island.ActualLoweringStorage != StorageExplicitIsland {
		t.Fatalf(
			"island planned/actual storage = %q/%q, want ExplicitIsland/ExplicitIsland: %+v",
			island.PlannedStorage,
			island.ActualLoweringStorage,
			island,
		)
	}
	if island.ValidationStatus != "validated_explicit_island_scope" ||
		island.LoweringStatus != "explicit_island_lowering" {
		t.Fatalf(
			"island validation/lowering status = %q/%q, want explicit island evidence",
			island.ValidationStatus,
			island.LoweringStatus,
		)
	}
	copied := findAllocation(t, plan, "copy_from_island", "$return")
	if copied.PlannedStorage != StorageHeap || copied.ActualLoweringStorage != StorageHeap {
		t.Fatalf(
			"copy planned/actual storage = %q/%q, want Heap/Heap: %+v",
			copied.PlannedStorage,
			copied.ActualLoweringStorage,
			copied,
		)
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
	if scalar.PlannedStorage != StorageEliminated ||
		scalar.ActualLoweringStorage != StorageEliminated {
		t.Fatalf(
			"scalar planned/actual storage = %q/%q, want Eliminated/Eliminated: %+v",
			scalar.PlannedStorage,
			scalar.ActualLoweringStorage,
			scalar,
		)
	}
	if scalar.LoweringStatus != "scalar_replacement" ||
		!strings.Contains(scalar.Reason, "scalar_replacement_fixed_constant_indices") {
		t.Fatalf(
			"scalar lowering/reason = %q/%q, want scalar replacement evidence",
			scalar.LoweringStatus,
			scalar.Reason,
		)
	}

	dynamic := findAllocation(t, plan, "dynamic", "xs")
	if dynamic.PlannedStorage == StorageEliminated ||
		dynamic.ActualLoweringStorage == StorageEliminated {
		t.Fatalf("dynamic-index allocation was scalar-eliminated: %+v", dynamic)
	}
	if dynamic.PlannedStorage != StorageStack || dynamic.ActualLoweringStorage != StorageStack {
		t.Fatalf(
			"dynamic planned/actual storage = %q/%q, want Stack/Stack fallback:\n%s",
			dynamic.PlannedStorage,
			dynamic.ActualLoweringStorage,
			FormatText(plan),
		)
	}

	printed := findAllocation(t, plan, "printed", "xs")
	if printed.PlannedStorage == StorageEliminated ||
		printed.ActualLoweringStorage == StorageEliminated {
		t.Fatalf("printed slice was scalar-eliminated despite observable slice use: %+v", printed)
	}
	if printed.PlannedStorage != StorageStack || printed.ActualLoweringStorage != StorageStack {
		t.Fatalf(
			"printed planned/actual storage = %q/%q, want Stack/Stack fallback:\n%s",
			printed.PlannedStorage,
			printed.ActualLoweringStorage,
			FormatText(plan),
		)
	}
}

func TestPlannerDoesNotTreatReadOnlyLookupCallAsEscapesCallUnknown(t *testing.T) {
	plan := allocationPlan(t, `
func lookup(keys: []i32, values: []i32, n: Int, key: Int) -> Int
uses mem:
    var i: Int = 0
    while i < n:
        if keys[i] == key:
            return values[i]
        i = i + 1
    return 0

func main() -> Int
uses alloc, mem:
    let n: Int = 4
    var keys: []i32 = make_i32(n)
    var values: []i32 = make_i32(n)
    return lookup(keys, values, n, 2)
`)

	for _, id := range []string{"keys", "values"} {
		alloc := findAllocation(t, plan, "main", id)
		if alloc.Escape != EscapeNoEscape {
			t.Fatalf(
				"%s allocation = %+v, want NoEscape from read-only local lookup summary",
				id,
				alloc,
			)
		}
		if strings.Contains(alloc.Reason, "unknown call escape") ||
			strings.Contains(alloc.Reason, "without interprocedural escape facts") {
			t.Fatalf("%s allocation reason still reports unknown call escape: %q", id, alloc.Reason)
		}
		if !strings.Contains(alloc.Reason, "read-only local call summary") {
			t.Fatalf(
				"%s allocation reason = %q, want read-only local call summary evidence",
				id,
				alloc.Reason,
			)
		}
	}
}

func TestPlannerTreatsJsonInoutWriterCallAsNoEscape(t *testing.T) {
	plan := allocationPlanWithOptions(t, `
func write_message_object(dst: inout []u8) -> Int
uses mem:
    dst[0] = 123
    dst[1] = 34
    dst[2] = 109
    return 3

func main() -> Int
uses alloc, mem:
    var buf: []u8 = make_u8(128)
    return write_message_object(buf)
`, Options{EnableStackLowering: true})

	buf := findAllocation(t, plan, "main", "buf")
	if buf.Escape != EscapeNoEscape {
		t.Fatalf("buf escape = %q, want NoEscape: %+v", buf.Escape, buf)
	}
	if buf.Storage != StorageStack {
		t.Fatalf("buf storage = %q, want Stack: %+v", buf.Storage, buf)
	}
	if buf.ActualLoweringStorage != StorageStack {
		t.Fatalf(
			"buf actual lowering storage = %q, want Stack: %+v",
			buf.ActualLoweringStorage,
			buf,
		)
	}
	if !strings.Contains(buf.Reason, "inout writer noescape summary") {
		t.Fatalf("buf reason = %q, want local inout writer noescape summary evidence", buf.Reason)
	}
	if contains(buf.ReasonCodes, HeapReasonUnknownCall) ||
		contains(buf.HeapReasonCodes, HeapReasonUnknownCall) {
		t.Fatalf(
			"buf retained unknown-call heap reason: reason=%v heap=%v alloc=%+v",
			buf.ReasonCodes,
			buf.HeapReasonCodes,
			buf,
		)
	}
}

func TestPlannerKeepsEscapingInoutWriterSummariesConservative(t *testing.T) {
	tests := []struct {
		name       string
		ops        []plir.Operation
		wantEscape EscapeClass
		wantReason string
	}{
		{
			name:       "returns_slice",
			wantEscape: EscapeCallUnknown,
			wantReason: HeapReasonUnknownCall,
			ops: []plir.Operation{
				{Kind: plir.OpReturn, Inputs: []string{"dst"}, Note: "returns inout slice"},
			},
		},
		{
			name:       "returns_alias",
			wantEscape: EscapeCallUnknown,
			wantReason: HeapReasonUnknownCall,
			ops: []plir.Operation{
				{
					Kind:    plir.OpAssign,
					Inputs:  []string{"dst"},
					Outputs: []string{"alias"},
					Note:    "local alias",
				},
				{Kind: plir.OpReturn, Inputs: []string{"alias"}, Note: "returns inout alias"},
			},
		},
		{
			name:       "stores_global",
			wantEscape: EscapeCallUnknown,
			wantReason: HeapReasonUnknownCall,
			ops: []plir.Operation{
				{
					Kind:    plir.OpGlobalStore,
					Inputs:  []string{"dst"},
					Outputs: []string{"stored"},
					Note:    "global store",
				},
			},
		},
		{
			name:       "unknown_call",
			wantEscape: EscapeCallUnknown,
			wantReason: HeapReasonUnknownCall,
			ops: []plir.Operation{
				{
					Kind:   plir.OpCall,
					Inputs: []string{"dst"},
					Note:   "external call without escape facts",
				},
			},
		},
		{
			name:       "unsafe_boundary",
			wantEscape: EscapeCallUnknown,
			wantReason: HeapReasonUnknownCall,
			ops: []plir.Operation{
				{
					Kind:        plir.OpUnsafe,
					Inputs:      []string{"dst"},
					UnsafeClass: plir.UnsafeUnknown,
					Note:        "unsafe boundary",
				},
			},
		},
		{
			name:       "actor_send",
			wantEscape: EscapeActor,
			wantReason: HeapReasonActorBoundary,
			ops: []plir.Operation{
				{
					Kind:   plir.OpActorSend,
					Inputs: []string{"mailbox", "dst"},
					Note:   "actor send payload",
				},
			},
		},
		{
			name:       "task_spawn",
			wantEscape: EscapeTask,
			wantReason: HeapReasonTaskBoundary,
			ops: []plir.Operation{
				{
					Kind:   plir.OpCall,
					Inputs: []string{"dst"},
					Note:   "core.task_spawn_i32_typed captures payload",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			callee := "writer_" + tc.name
			caller := "caller_" + tc.name
			plan, err := FromPLIRWithOptions(&plir.Program{Funcs: []plir.Function{
				syntheticInoutWriterCallee(callee, tc.ops),
				syntheticInoutWriterCaller(caller, callee),
			}}, Options{EnableStackLowering: true})
			if err != nil {
				t.Fatalf("FromPLIRWithOptions: %v", err)
			}

			buf := findAllocation(t, plan, caller, "buf")
			if buf.Escape != tc.wantEscape || buf.Storage != StorageHeap ||
				buf.ActualLoweringStorage != StorageHeap {
				t.Fatalf("%s buf allocation = %+v, want %s/Heap/Heap", tc.name, buf, tc.wantEscape)
			}
			if strings.Contains(buf.Reason, "inout writer noescape summary") {
				t.Fatalf("%s buf reason accepted unsafe writer summary: %q", tc.name, buf.Reason)
			}
			if !contains(buf.HeapReasonCodes, tc.wantReason) {
				t.Fatalf(
					"%s buf heap reason codes = %v, want %s: %+v",
					tc.name,
					buf.HeapReasonCodes,
					tc.wantReason,
					buf,
				)
			}
		})
	}
}

func TestPlannerClassifiesReadOnlyCallUnsafeAndIslandAllocations(t *testing.T) {
	plan := allocationPlan(t, `
func consume(xs: []u8) -> Int
uses mem:
    return xs.len

func read_only_call() -> Int
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
    return read_only_call() + unsafe_boundary() + islanded()
`)

	call := findAllocation(t, plan, "read_only_call", "xs")
	if call.Escape != EscapeNoEscape || call.Storage != StorageStack ||
		call.ActualLoweringStorage != StorageHeap {
		t.Fatalf(
			("read-only-call allocation = %+v, want NoEscape planned Stack " +
				"with conservative actual Heap when stack lowering is disabled"),
			call,
		)
	}
	if !strings.Contains(call.Reason, "read-only local call summary") {
		t.Fatalf("read-only-call reason = %q, want summary evidence", call.Reason)
	}
	stackPlan := allocationPlanWithOptions(t, `
func consume(xs: []u8) -> Int
uses mem:
    return xs.len

func read_only_call() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    return consume(xs)

func main() -> Int
uses alloc, mem:
    return read_only_call()
`, Options{EnableStackLowering: true})
	stacked := findAllocation(t, stackPlan, "read_only_call", "xs")
	if stacked.Escape != EscapeNoEscape || stacked.Storage != StorageStack ||
		stacked.ActualLoweringStorage != StorageStack {
		t.Fatalf(
			"read-only-call stack lowering = %+v, want NoEscape Stack/Stack when stack lowering is enabled",
			stacked,
		)
	}
	if stacked.LoweringStatus != "stack_lowering" {
		t.Fatalf(
			"read-only-call lowering status = %q, want stack_lowering: %+v",
			stacked.LoweringStatus,
			stacked,
		)
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
	for _, want := range []string{
		"planned_storage: Stack",
		"read-only local call summary",
		"planned_storage: ExplicitIsland",
		"actual_lowering_storage: ExplicitIsland",
	} {
		if !strings.Contains(dump, want) {
			t.Fatalf("FormatText missing %q:\n%s", want, dump)
		}
	}
}

func TestPlannerKeepsEscapingLocalCallSummariesConservative(t *testing.T) {
	plan := allocationPlanFile(t, `
var stored: []u8

func returns_slice(xs: []u8) -> []u8
uses mem:
    return xs

func stores_global(xs: []u8) -> Int
uses mem:
    stored = xs
    return 0

func touches_unsafe(xs: []u8) -> Int
uses mem:
    unsafe:
        var y = 1
    return xs.len

func call_return() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    let ys: []u8 = returns_slice(xs)
    return ys.len

func call_global() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    return stores_global(xs)

func call_unsafe() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    return touches_unsafe(xs)

func main() -> Int
uses alloc, mem:
    return call_return() + call_global() + call_unsafe()
`)

	for _, tc := range []struct {
		fn string
		id string
	}{
		{fn: "call_return", id: "xs"},
		{fn: "call_global", id: "xs"},
		{fn: "call_unsafe", id: "xs"},
	} {
		alloc := findAllocation(t, plan, tc.fn, tc.id)
		if alloc.Escape == EscapeNoEscape || alloc.Storage != StorageHeap {
			t.Fatalf("%s allocation = %+v, want conservative heap fallback", tc.fn, alloc)
		}
		if !strings.Contains(alloc.Reason, "without interprocedural escape facts") {
			t.Fatalf(
				"%s reason = %q, want conservative interprocedural fallback evidence",
				tc.fn,
				alloc.Reason,
			)
		}
	}
}

func TestPlannerKeepsLocalActorTaskCallSummariesConservative(t *testing.T) {
	for _, tc := range []struct {
		name   string
		callee string
		op     plir.Operation
	}{
		{
			name:   "actor",
			callee: "callee_a",
			op: plir.Operation{
				Kind:   plir.OpActorSend,
				Inputs: []string{"mailbox", "xs"},
				Note:   "actor send payload",
			},
		},
		{
			name:   "task",
			callee: "callee_b",
			op: plir.Operation{
				Kind:   plir.OpCall,
				Inputs: []string{"xs"},
				Note:   "core.task_spawn_i32_typed captures payload",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calleeName := tc.callee
			op := tc.op
			op.ID = "op1"
			prog := &plir.Program{Funcs: []plir.Function{
				{
					Name: calleeName,
					Summary: &plir.FunctionSummary{
						ParamNames: []string{"xs"},
						ParamTypes: []string{"[]u8"},
						ReturnType: "i32",
						Effects:    []string{"mem"},
					},
					Values: []plir.Value{{
						ID:         "param:xs",
						Kind:       plir.ValueParam,
						Type:       "[]u8",
						Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "param:xs"},
						Borrow:     plir.BorrowImm,
					}},
					Ops: []plir.Operation{op},
				},
				{
					Name: "caller_" + tc.name,
					Values: []plir.Value{{
						ID:   "alloc_intent:xs",
						Kind: plir.ValueAllocIntent,
						Type: "[]u8",
						Alloc: &plir.AllocIntent{
							ElementType:         "u8",
							ElementSize:         1,
							LengthExpr:          "4",
							LengthConstKnown:    true,
							LengthConst:         4,
							ZeroGuardStatus:     "valid_empty_no_allocator",
							NegativeGuardStatus: "reject_before_allocation",
							OverflowGuardStatus: "reject_before_allocation",
							Builtin:             "core.make_u8",
						},
						Provenance: plir.Provenance{Kind: plir.ProvenanceAllocation, Root: "xs"},
					}},
					Ops: []plir.Operation{{
						ID:     "op_call",
						Kind:   plir.OpCall,
						Inputs: []string{"xs"},
						Note:   calleeName,
					}},
				},
			}}
			plan, err := FromPLIR(prog)
			if err != nil {
				t.Fatalf("FromPLIR: %v", err)
			}
			alloc := findAllocation(t, plan, "caller_"+tc.name, "xs")
			if alloc.Escape != EscapeCallUnknown || alloc.Storage != StorageHeap {
				t.Fatalf(
					"%s local boundary allocation = %+v, want EscapesCallUnknown/Heap",
					tc.name,
					alloc,
				)
			}
		})
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
		syntheticEscapeFunction(
			"closure",
			plir.Operation{
				Kind:    plir.OpClosure,
				Inputs:  []string{"xs"},
				Outputs: []string{"f"},
				Note:    "closure captures environment",
			},
		),
		syntheticEscapeFunction(
			"task",
			plir.Operation{
				Kind:   plir.OpCall,
				Inputs: []string{"xs"},
				Note:   "core.task_spawn_i32_typed captures payload",
			},
		),
		syntheticEscapeFunction(
			"actor",
			plir.Operation{
				Kind:   plir.OpCall,
				Inputs: []string{"xs"},
				Note:   "core.send_typed sends actor payload",
			},
		),
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
			{
				ID:      "op0",
				Kind:    plir.OpAllocIntent,
				Block:   "entry",
				Outputs: []string{"alloc_intent:xs"},
				Note:    "make<u8>",
			},
			op,
		},
		Blocks: []plir.BasicBlock{
			{ID: "entry", Kind: "entry", Entry: true, Ops: []string{"op0", "op1"}, Exit: true},
		},
	}
}

func syntheticInoutWriterCallee(name string, ops []plir.Operation) plir.Function {
	ops = append([]plir.Operation(nil), ops...)
	blockOps := make([]string, 0, len(ops))
	for i := range ops {
		if ops[i].ID == "" {
			ops[i].ID = "op" + strconv.Itoa(i)
		}
		if ops[i].Block == "" {
			ops[i].Block = "entry"
		}
		blockOps = append(blockOps, ops[i].ID)
	}
	return plir.Function{
		Name: name,
		Summary: &plir.FunctionSummary{
			ParamNames:     []string{"dst"},
			ParamTypes:     []string{"[]u8"},
			ParamOwnership: []string{"inout"},
			ReturnType:     "i32",
			Effects:        []string{"mem"},
		},
		Values: []plir.Value{{
			ID:         "param:dst",
			Kind:       plir.ValueParam,
			Type:       "[]u8",
			Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "param:dst"},
			Borrow:     plir.BorrowMut,
		}},
		Ops: ops,
		Blocks: []plir.BasicBlock{
			{ID: "entry", Kind: "entry", Entry: true, Ops: blockOps, Exit: true},
		},
	}
}

func syntheticInoutWriterCaller(name string, callee string) plir.Function {
	return plir.Function{
		Name: name,
		Values: []plir.Value{{
			ID:     "alloc_intent:buf",
			Kind:   plir.ValueAllocIntent,
			Type:   "[]u8",
			Source: "test:1:1",
			Alloc: &plir.AllocIntent{
				ElementType:         "u8",
				ElementSize:         1,
				LengthExpr:          "128",
				LengthConstKnown:    true,
				LengthConst:         128,
				ZeroGuardStatus:     "valid_empty_no_allocator",
				NegativeGuardStatus: "reject_before_allocation",
				OverflowGuardStatus: "reject_before_allocation",
				Builtin:             "core.make_u8",
			},
			Provenance: plir.Provenance{Kind: plir.ProvenanceAllocation, Root: "buf"},
		}},
		Ops: []plir.Operation{
			{
				ID:      "op0",
				Kind:    plir.OpAllocIntent,
				Block:   "entry",
				Outputs: []string{"alloc_intent:buf"},
				Note:    "make<u8>",
			},
			{ID: "op1", Kind: plir.OpCall, Block: "entry", Inputs: []string{"buf"}, Note: callee},
		},
		Blocks: []plir.BasicBlock{
			{ID: "entry", Kind: "entry", Entry: true, Ops: []string{"op0", "op1"}, Exit: true},
		},
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
	if empty.ZeroGuardStatus != "valid_empty_no_allocator" ||
		empty.NegativeGuardStatus != "reject_before_allocation" ||
		empty.OverflowGuardStatus != "reject_before_allocation" {
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
	for _, want := range []string{
		"length_status: valid_empty_allocation",
		"length_status: rejected_negative_length",
		"length_status: rejected_byte_size_overflow",
		"zero_guard: valid_empty_no_allocator",
	} {
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
	if unused.Escape != EscapeNoEscape || unused.Storage != StorageEliminated ||
		unused.ActualLoweringStorage != StorageEliminated {
		t.Fatalf("unused copy allocation = %+v, want NoEscape/Eliminated actual Eliminated", unused)
	}
	if unused.LoweringStatus != "eliminated_unused_copy" {
		t.Fatalf(
			"unused copy lowering status = %q, want eliminated_unused_copy",
			unused.LoweringStatus,
		)
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
