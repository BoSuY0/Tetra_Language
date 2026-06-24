package runtimeabi_test

import (
	"testing"

	. "tetra_language/compiler/internal/runtimeabi"
)

func TestSmallHeapConfigDefinesChunkAndSizeClasses(t *testing.T) {
	cfg := RuntimeSmallHeapConfig()
	if cfg.ChunkBytes < 64*1024 {
		t.Fatalf("chunk bytes = %d, want at least one 64KiB runtime chunk", cfg.ChunkBytes)
	}
	if cfg.Alignment != 16 {
		t.Fatalf("alignment = %d, want 16-byte minimum alignment", cfg.Alignment)
	}
	if cfg.MaxSmallBytes != 4096 {
		t.Fatalf("max small bytes = %d, want 4096", cfg.MaxSmallBytes)
	}
	if len(cfg.Classes) == 0 {
		t.Fatalf("missing small heap size classes")
	}
	for _, cls := range cfg.Classes {
		if cls.MaxBytes <= 0 || cls.MaxBytes%cfg.Alignment != 0 {
			t.Fatalf("bad class %+v for config %+v", cls, cfg)
		}
	}
}

func TestSmallHeapClassForBytes(t *testing.T) {
	cases := []struct {
		bytes int64
		name  string
		ok    bool
	}{
		{bytes: 1, name: "small_16", ok: true},
		{bytes: 16, name: "small_16", ok: true},
		{bytes: 17, name: "small_32", ok: true},
		{bytes: 4096, name: "small_4096", ok: true},
		{bytes: 4097, ok: false},
		{bytes: 0, ok: false},
		{bytes: -1, ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cls, ok := SmallHeapClassForBytes(tc.bytes)
			if ok != tc.ok {
				t.Fatalf(
					"SmallHeapClassForBytes(%d) ok=%v, want %v (class=%+v)",
					tc.bytes,
					ok,
					tc.ok,
					cls,
				)
			}
			if ok && cls.Name != tc.name {
				t.Fatalf("SmallHeapClassForBytes(%d) = %+v, want name %q", tc.bytes, cls, tc.name)
			}
		})
	}
}

func TestAlignSmallHeapBytes(t *testing.T) {
	cases := []struct {
		bytes int64
		want  int64
		ok    bool
	}{
		{bytes: 1, want: 16, ok: true},
		{bytes: 16, want: 16, ok: true},
		{bytes: 17, want: 32, ok: true},
		{bytes: 4096, want: 4096, ok: true},
		{bytes: 4097, ok: false},
	}
	for _, tc := range cases {
		got, ok := AlignSmallHeapBytes(tc.bytes)
		if ok != tc.ok || got != tc.want {
			t.Fatalf(
				"AlignSmallHeapBytes(%d) = %d,%v want %d,%v",
				tc.bytes,
				got,
				ok,
				tc.want,
				tc.ok,
			)
		}
	}
}

func TestPerCoreSmallHeapAllocatorABIDefinesMetadataAndReuse(t *testing.T) {
	abi := RuntimePerCoreSmallHeapABI(4)
	if abi.RuntimePath != AllocationPathPerCoreSmallHeap {
		t.Fatalf("runtime path = %q, want %q", abi.RuntimePath, AllocationPathPerCoreSmallHeap)
	}
	if abi.CoreCount != 4 {
		t.Fatalf("core count = %d, want 4", abi.CoreCount)
	}
	if abi.ChunkBytes != SmallHeapChunkBytes || abi.Alignment != SmallHeapAlignment {
		t.Fatalf(
			"allocator ABI chunk/alignment = %d/%d, want %d/%d",
			abi.ChunkBytes,
			abi.Alignment,
			SmallHeapChunkBytes,
			SmallHeapAlignment,
		)
	}
	if abi.MetadataBytesPerCore <= 0 {
		t.Fatalf(
			"metadata bytes per core = %d, want positive metadata size",
			abi.MetadataBytesPerCore,
		)
	}
	for _, want := range []string{"bump_offset", "chunk_refills", "free_list", "reuse_count"} {
		if !containsString(abi.MetadataFields, want) {
			t.Fatalf("metadata fields = %v, want %q", abi.MetadataFields, want)
		}
	}
	if abi.ReusePolicy != "same_core_same_size_class_free_list" {
		t.Fatalf("reuse policy = %q, want same-core size-class free list", abi.ReusePolicy)
	}
}

func TestPerCoreSmallHeapAllocatorReusesFreedSameClassBlocks(t *testing.T) {
	allocator, err := NewPerCoreSmallHeapAllocator(RuntimePerCoreSmallHeapABI(2))
	if err != nil {
		t.Fatalf("NewPerCoreSmallHeapAllocator: %v", err)
	}

	first, err := allocator.Alloc(0, 17)
	if err != nil {
		t.Fatalf("first alloc: %v", err)
	}
	if first.CoreID != 0 || first.ClassName != "small_32" || first.Reused {
		t.Fatalf("first allocation = %+v, want core 0 fresh small_32 block", first)
	}
	if err := allocator.Free(first); err != nil {
		t.Fatalf("free first: %v", err)
	}
	if err := allocator.Free(first); err == nil {
		t.Fatalf("double free of first handle unexpectedly succeeded")
	}

	reused, err := allocator.Alloc(0, 24)
	if err != nil {
		t.Fatalf("reuse alloc: %v", err)
	}
	if !reused.Reused || reused.BlockID != first.BlockID {
		t.Fatalf("reused allocation = %+v, want same freed block id %d", reused, first.BlockID)
	}
	if reused.CoreID != first.CoreID || reused.ClassName != first.ClassName {
		t.Fatalf("reused allocation core/class = %+v, want same core/class as %+v", reused, first)
	}
	report := allocator.Report()
	if report.TotalAllocations != 2 || report.TotalFrees != 1 || report.TotalReuses != 1 {
		t.Fatalf("report totals = %+v, want allocations=2 frees=1 reuses=1", report)
	}
	if report.Cores[0].FreeListBlocks["small_32"] != 0 {
		t.Fatalf(
			"core 0 free list = %+v, want reused class drained",
			report.Cores[0].FreeListBlocks,
		)
	}
	events := allocator.MemoryBackendEvents()
	var reserveCommit int
	for _, event := range events {
		if event.Operation == MemoryBackendReserve || event.Operation == MemoryBackendCommit {
			reserveCommit++
		}
	}
	if reserveCommit != 2 {
		t.Fatalf("reserve/commit events = %d in %+v, want one chunk reserve+commit despite reuse", reserveCommit, events)
	}
	snapshot := allocator.LedgerSnapshot()
	if len(snapshot) != 1 {
		t.Fatalf("ledger snapshot = %+v, want process domain", snapshot)
	}
	if snapshot[0].ReservedBytes != SmallHeapChunkBytes ||
		snapshot[0].CommittedBytes != SmallHeapChunkBytes {
		t.Fatalf("ledger domain = %+v, want one reserved/committed chunk", snapshot[0])
	}
	if snapshot[0].CurrentBytes != int64(reused.RequestedBytes) {
		t.Fatalf("ledger current bytes = %d, want live reused request %d", snapshot[0].CurrentBytes, reused.RequestedBytes)
	}
}

func TestPerCoreSmallHeapAllocatorStressDoesNotBehaveLikeMmapPerAllocation(t *testing.T) {
	allocator, err := NewPerCoreSmallHeapAllocator(RuntimePerCoreSmallHeapABI(4))
	if err != nil {
		t.Fatalf("NewPerCoreSmallHeapAllocator: %v", err)
	}

	var handles []PerCoreSmallHeapHandle
	for i := 0; i < 128; i++ {
		bytes := int64(17 + (i%5)*16)
		handle, err := allocator.Alloc(i%4, bytes)
		if err != nil {
			t.Fatalf("alloc %d: %v", i, err)
		}
		handles = append(handles, handle)
	}
	for i, handle := range handles {
		if i%2 == 0 {
			if err := allocator.Free(handle); err != nil {
				t.Fatalf("free %d: %v", i, err)
			}
		}
	}
	for i := 0; i < 64; i++ {
		bytes := int64(17 + (i%5)*16)
		if _, err := allocator.Alloc(i%4, bytes); err != nil {
			t.Fatalf("reuse alloc %d: %v", i, err)
		}
	}

	report := allocator.Report()
	if report.TotalReuses == 0 {
		t.Fatalf("stress report = %+v, want reuse evidence", report)
	}
	if report.TotalChunkRefills >= report.TotalAllocations {
		t.Fatalf("stress report = %+v, chunk refills should be fewer than allocations", report)
	}
	if report.EstimatedMmapPerAllocation {
		t.Fatalf("stress report = %+v, should not behave like mmap-per-allocation", report)
	}
	if report.FragmentationBytes <= 0 {
		t.Fatalf(
			"stress report = %+v, want fragmentation accounting from size-class rounding",
			report,
		)
	}
	if report.Domain.DomainID != "domain:process" || report.Domain.Kind != DomainProcess {
		t.Fatalf("stress report domain = %+v, want process domain", report.Domain)
	}
	if report.Domain.RequestedBytes != int64(report.BytesRequested) ||
		report.Domain.ReservedBytes != int64(report.BytesReserved) {
		t.Fatalf(
			"stress report domain bytes = %+v, want requested/reserved from report %+v",
			report.Domain,
			report,
		)
	}
	for _, core := range report.Cores {
		if core.Domain.DomainID != "domain:process" {
			t.Fatalf("core report domain = %+v, want process domain", core.Domain)
		}
	}
	if report.Domain.DomainID != "domain:process" || report.Domain.Kind != DomainProcess {
		t.Fatalf("stress report domain = %+v, want process domain", report.Domain)
	}
	if report.Domain.RequestedBytes != int64(report.BytesRequested) || report.Domain.ReservedBytes != int64(report.BytesReserved) {
		t.Fatalf("stress report domain bytes = %+v, want requested/reserved from report %+v", report.Domain, report)
	}
	for _, core := range report.Cores {
		if core.Domain.DomainID != "domain:process" {
			t.Fatalf("core report domain = %+v, want process domain", core.Domain)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
