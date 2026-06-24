package smallheap

import "fmt"

const (
	SmallHeapAlignment     = 16
	SmallHeapChunkBytes    = 64 * 1024
	SmallHeapMaxSmallBytes = 4096
)

type AllocationRuntimePath string

const (
	AllocationPathPerCoreSmallHeap AllocationRuntimePath = "per_core_small_heap"
	AllocationPathLargeMmap        AllocationRuntimePath = "large_mmap"
)

type MemoryDomainKind string

const DomainProcess MemoryDomainKind = "process"

type MemoryDomainState string

const (
	DomainStateActive MemoryDomainState = "active"
	DomainStateClosed MemoryDomainState = "closed"
)

type MemoryDomain struct {
	DomainID         string            `json:"domain_id"`
	ParentDomainID   string            `json:"parent_domain_id,omitempty"`
	Kind             MemoryDomainKind  `json:"kind"`
	OwnerKind        string            `json:"owner_kind"`
	OwnerID          string            `json:"owner_id"`
	Lifetime         string            `json:"lifetime"`
	BudgetBytes      int64             `json:"budget_bytes,omitempty"`
	RequestedBytes   int64             `json:"requested_bytes,omitempty"`
	ReservedBytes    int64             `json:"reserved_bytes,omitempty"`
	CommittedBytes   int64             `json:"committed_bytes,omitempty"`
	ReleasedBytes    int64             `json:"released_bytes,omitempty"`
	State            MemoryDomainState `json:"state,omitempty"`
	Epoch            uint64            `json:"epoch,omitempty"`
	DecommittedBytes int64             `json:"decommitted_bytes,omitempty"`
	CurrentBytes     int64             `json:"current_bytes,omitempty"`
	PeakBytes        int64             `json:"peak_bytes,omitempty"`
	CopyCount        int               `json:"copy_count,omitempty"`
	BytesCopied      int64             `json:"bytes_copied,omitempty"`
}

func DefaultProcessMemoryDomain(requested int64, reserved int64) MemoryDomain {
	return MemoryDomain{
		DomainID:  "domain:process",
		Kind:      DomainProcess,
		OwnerKind: "process",
		OwnerID:   "current",
		Lifetime:  "process",
		State:     DomainStateActive,
	}
}

type SmallHeapClass struct {
	Name      string `json:"name"`
	MaxBytes  int    `json:"max_bytes"`
	Alignment int    `json:"alignment"`
}

type SmallHeapConfig struct {
	ChunkBytes    int              `json:"chunk_bytes"`
	MaxSmallBytes int              `json:"max_small_bytes"`
	Alignment     int              `json:"alignment"`
	Classes       []SmallHeapClass `json:"classes"`
}

func RuntimeSmallHeapConfig() SmallHeapConfig {
	classes := make([]SmallHeapClass, 0, SmallHeapMaxSmallBytes/SmallHeapAlignment)
	for max := SmallHeapAlignment; max <= SmallHeapMaxSmallBytes; max += SmallHeapAlignment {
		classes = append(classes, SmallHeapClass{
			Name:      fmt.Sprintf("small_%d", max),
			MaxBytes:  max,
			Alignment: SmallHeapAlignment,
		})
	}
	return SmallHeapConfig{
		ChunkBytes:    SmallHeapChunkBytes,
		MaxSmallBytes: SmallHeapMaxSmallBytes,
		Alignment:     SmallHeapAlignment,
		Classes:       classes,
	}
}

func SmallHeapClassForBytes(bytes int64) (SmallHeapClass, bool) {
	aligned, ok := AlignSmallHeapBytes(bytes)
	if !ok {
		return SmallHeapClass{}, false
	}
	return SmallHeapClass{
		Name:      fmt.Sprintf("small_%d", aligned),
		MaxBytes:  int(aligned),
		Alignment: SmallHeapAlignment,
	}, true
}

func AlignSmallHeapBytes(bytes int64) (int64, bool) {
	if bytes <= 0 || bytes > SmallHeapMaxSmallBytes {
		return 0, false
	}
	aligned := (bytes + SmallHeapAlignment - 1) & ^int64(SmallHeapAlignment-1)
	if aligned > SmallHeapMaxSmallBytes {
		return 0, false
	}
	return aligned, true
}

type PerCoreSmallHeapABI struct {
	RuntimePath          AllocationRuntimePath `json:"runtime_path"`
	CoreCount            int                   `json:"core_count"`
	DefaultDomain        MemoryDomain          `json:"default_domain"`
	ChunkBytes           int                   `json:"chunk_bytes"`
	MaxSmallBytes        int                   `json:"max_small_bytes"`
	Alignment            int                   `json:"alignment"`
	MetadataBytesPerCore int                   `json:"metadata_bytes_per_core"`
	MetadataFields       []string              `json:"metadata_fields"`
	Classes              []SmallHeapClass      `json:"classes"`
	ReusePolicy          string                `json:"reuse_policy"`
	LargeRuntimePath     AllocationRuntimePath `json:"large_runtime_path"`
}

type PerCoreSmallHeapHandle struct {
	BlockID        int64        `json:"block_id"`
	Generation     int64        `json:"generation"`
	CoreID         int          `json:"core_id"`
	ChunkID        int          `json:"chunk_id"`
	Offset         int          `json:"offset"`
	RequestedBytes int          `json:"requested_bytes"`
	ReservedBytes  int          `json:"reserved_bytes"`
	ClassName      string       `json:"class_name"`
	Reused         bool         `json:"reused"`
	Domain         MemoryDomain `json:"domain"`
}

type PerCoreSmallHeapReport struct {
	RuntimePath                AllocationRuntimePath        `json:"runtime_path"`
	CoreCount                  int                          `json:"core_count"`
	TotalAllocations           int                          `json:"total_allocations"`
	TotalFrees                 int                          `json:"total_frees"`
	TotalReuses                int                          `json:"total_reuses"`
	TotalChunkRefills          int                          `json:"total_chunk_refills"`
	TotalMmapCalls             int                          `json:"total_mmap_calls"`
	BytesRequested             int                          `json:"bytes_requested"`
	BytesReserved              int                          `json:"bytes_reserved"`
	FragmentationBytes         int                          `json:"fragmentation_bytes"`
	Domain                     MemoryDomain                 `json:"domain"`
	EstimatedMmapPerAllocation bool                         `json:"estimated_mmap_per_allocation"`
	Cores                      []PerCoreSmallHeapCoreReport `json:"cores"`
}

type PerCoreSmallHeapCoreReport struct {
	CoreID             int            `json:"core_id"`
	AllocationCount    int            `json:"allocation_count"`
	FreeCount          int            `json:"free_count"`
	ReuseCount         int            `json:"reuse_count"`
	ChunkRefills       int            `json:"chunk_refills"`
	BumpOffset         int            `json:"bump_offset"`
	BytesRequested     int            `json:"bytes_requested"`
	BytesReserved      int            `json:"bytes_reserved"`
	FragmentationBytes int            `json:"fragmentation_bytes"`
	Domain             MemoryDomain   `json:"domain"`
	FreeListBlocks     map[string]int `json:"free_list_blocks"`
}

type PerCoreSmallHeapAllocator struct {
	abi         PerCoreSmallHeapABI
	cores       []perCoreSmallHeapCore
	live        map[int64]int64
	nextBlockID int64
}

type perCoreSmallHeapCore struct {
	id                 int
	currentChunkID     int
	bumpOffset         int
	allocationCount    int
	freeCount          int
	reuseCount         int
	chunkRefills       int
	bytesRequested     int
	bytesReserved      int
	fragmentationBytes int
	freeLists          map[string][]PerCoreSmallHeapHandle
}

func RuntimePerCoreSmallHeapABI(coreCount int) PerCoreSmallHeapABI {
	if coreCount <= 0 {
		coreCount = 1
	}
	cfg := RuntimeSmallHeapConfig()
	return PerCoreSmallHeapABI{
		RuntimePath:          AllocationPathPerCoreSmallHeap,
		CoreCount:            coreCount,
		DefaultDomain:        DefaultProcessMemoryDomain(0, 0),
		ChunkBytes:           cfg.ChunkBytes,
		MaxSmallBytes:        cfg.MaxSmallBytes,
		Alignment:            cfg.Alignment,
		MetadataBytesPerCore: 64,
		MetadataFields: []string{
			"bump_offset",
			"chunk_refills",
			"free_list",
			"allocation_count",
			"free_count",
			"reuse_count",
		},
		Classes:          cfg.Classes,
		ReusePolicy:      "same_core_same_size_class_free_list",
		LargeRuntimePath: AllocationPathLargeMmap,
	}
}

func NewPerCoreSmallHeapAllocator(abi PerCoreSmallHeapABI) (*PerCoreSmallHeapAllocator, error) {
	if abi.CoreCount <= 0 {
		return nil, fmt.Errorf("per-core small heap allocator: core_count must be positive")
	}
	if abi.ChunkBytes <= 0 {
		return nil, fmt.Errorf("per-core small heap allocator: chunk_bytes must be positive")
	}
	if abi.Alignment != SmallHeapAlignment {
		return nil, fmt.Errorf(
			"per-core small heap allocator: alignment = %d, want %d",
			abi.Alignment,
			SmallHeapAlignment,
		)
	}
	cores := make([]perCoreSmallHeapCore, abi.CoreCount)
	for i := range cores {
		cores[i] = perCoreSmallHeapCore{id: i, freeLists: map[string][]PerCoreSmallHeapHandle{}}
	}
	return &PerCoreSmallHeapAllocator{
		abi:   abi,
		cores: cores,
		live:  map[int64]int64{},
	}, nil
}

func (allocator *PerCoreSmallHeapAllocator) Alloc(
	coreID int,
	bytes int64,
) (PerCoreSmallHeapHandle, error) {
	if allocator == nil {
		return PerCoreSmallHeapHandle{}, fmt.Errorf(
			"per-core small heap allocator: allocator is nil",
		)
	}
	if coreID < 0 || coreID >= len(allocator.cores) {
		return PerCoreSmallHeapHandle{}, fmt.Errorf(
			"per-core small heap allocator: core %d out of range",
			coreID,
		)
	}
	cls, ok := SmallHeapClassForBytes(bytes)
	if !ok {
		return PerCoreSmallHeapHandle{}, fmt.Errorf(
			"per-core small heap allocator: %d bytes is outside small size classes",
			bytes,
		)
	}
	core := &allocator.cores[coreID]
	if list := core.freeLists[cls.Name]; len(list) > 0 {
		handle := list[len(list)-1]
		core.freeLists[cls.Name] = list[:len(list)-1]
		handle.Generation++
		handle.RequestedBytes = int(bytes)
		handle.ReservedBytes = cls.MaxBytes
		handle.Domain = DefaultProcessMemoryDomain(0, 0)
		handle.Domain.RequestedBytes = int64(handle.RequestedBytes)
		handle.Domain.ReservedBytes = int64(handle.ReservedBytes)
		handle.Reused = true
		allocator.live[handle.BlockID] = handle.Generation
		core.recordAllocation(handle)
		core.reuseCount++
		return handle, nil
	}
	if core.currentChunkID == 0 || core.bumpOffset+cls.MaxBytes > allocator.abi.ChunkBytes {
		core.chunkRefills++
		core.currentChunkID = core.chunkRefills
		core.bumpOffset = 0
	}
	allocator.nextBlockID++
	handle := PerCoreSmallHeapHandle{
		BlockID:        allocator.nextBlockID,
		Generation:     1,
		CoreID:         coreID,
		ChunkID:        core.currentChunkID,
		Offset:         core.bumpOffset,
		RequestedBytes: int(bytes),
		ReservedBytes:  cls.MaxBytes,
		ClassName:      cls.Name,
		Domain:         DefaultProcessMemoryDomain(0, 0),
	}
	handle.Domain.RequestedBytes = bytes
	handle.Domain.ReservedBytes = int64(cls.MaxBytes)
	core.bumpOffset += cls.MaxBytes
	allocator.live[handle.BlockID] = handle.Generation
	core.recordAllocation(handle)
	return handle, nil
}

func (allocator *PerCoreSmallHeapAllocator) Free(handle PerCoreSmallHeapHandle) error {
	if allocator == nil {
		return fmt.Errorf("per-core small heap allocator: allocator is nil")
	}
	if handle.CoreID < 0 || handle.CoreID >= len(allocator.cores) {
		return fmt.Errorf("per-core small heap allocator: core %d out of range", handle.CoreID)
	}
	generation, ok := allocator.live[handle.BlockID]
	if !ok || generation != handle.Generation {
		return fmt.Errorf(
			"per-core small heap allocator: stale or double free for block %d",
			handle.BlockID,
		)
	}
	delete(allocator.live, handle.BlockID)
	core := &allocator.cores[handle.CoreID]
	core.freeLists[handle.ClassName] = append(core.freeLists[handle.ClassName], handle)
	core.freeCount++
	return nil
}

func (allocator *PerCoreSmallHeapAllocator) Report() PerCoreSmallHeapReport {
	report := PerCoreSmallHeapReport{RuntimePath: AllocationPathPerCoreSmallHeap}
	if allocator == nil {
		return report
	}
	report.CoreCount = len(allocator.cores)
	for _, core := range allocator.cores {
		coreReport := core.report()
		report.TotalAllocations += coreReport.AllocationCount
		report.TotalFrees += coreReport.FreeCount
		report.TotalReuses += coreReport.ReuseCount
		report.TotalChunkRefills += coreReport.ChunkRefills
		report.BytesRequested += coreReport.BytesRequested
		report.BytesReserved += coreReport.BytesReserved
		report.FragmentationBytes += coreReport.FragmentationBytes
		report.Cores = append(report.Cores, coreReport)
	}
	report.TotalMmapCalls = report.TotalChunkRefills
	report.Domain = DefaultProcessMemoryDomain(0, 0)
	report.Domain.RequestedBytes = int64(report.BytesRequested)
	report.Domain.ReservedBytes = int64(report.BytesReserved)
	report.EstimatedMmapPerAllocation = report.TotalAllocations > 0 &&
		report.TotalMmapCalls >= report.TotalAllocations
	return report
}

func (core *perCoreSmallHeapCore) recordAllocation(handle PerCoreSmallHeapHandle) {
	core.allocationCount++
	core.bytesRequested += handle.RequestedBytes
	core.bytesReserved += handle.ReservedBytes
	core.fragmentationBytes += handle.ReservedBytes - handle.RequestedBytes
}

func (core perCoreSmallHeapCore) report() PerCoreSmallHeapCoreReport {
	freeListBlocks := map[string]int{}
	for className, blocks := range core.freeLists {
		freeListBlocks[className] = len(blocks)
	}
	return PerCoreSmallHeapCoreReport{
		CoreID:             core.id,
		AllocationCount:    core.allocationCount,
		FreeCount:          core.freeCount,
		ReuseCount:         core.reuseCount,
		ChunkRefills:       core.chunkRefills,
		BumpOffset:         core.bumpOffset,
		BytesRequested:     core.bytesRequested,
		BytesReserved:      core.bytesReserved,
		FragmentationBytes: core.fragmentationBytes,
		Domain: func() MemoryDomain {
			domain := DefaultProcessMemoryDomain(0, 0)
			domain.RequestedBytes = int64(core.bytesRequested)
			domain.ReservedBytes = int64(core.bytesReserved)
			return domain
		}(),
		FreeListBlocks: freeListBlocks,
	}
}
