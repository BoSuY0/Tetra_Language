package runtimeabi

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/runtimeabi/smallheap"
)

const MemoryBackendContractSchemaV1 = "tetra.memory.backend-contract.v1"
const MemoryBackendAllocationEvidenceSchemaV1 = "tetra.memory.backend-allocation.v1"

const (
	MemoryBackendReserveSymbol   = "__tetra_memory_reserve_v1"
	MemoryBackendCommitSymbol    = "__tetra_memory_commit_v1"
	MemoryBackendDecommitSymbol  = "__tetra_memory_decommit_v1"
	MemoryBackendReleaseSymbol   = "__tetra_memory_release_v1"
	MemoryBackendTrimSymbol      = "__tetra_memory_trim_v1"
	MemoryBackendFootprintSymbol = "__tetra_memory_footprint_v1"
)

type MemoryBackendOperation string

const (
	MemoryBackendReserve   MemoryBackendOperation = "reserve"
	MemoryBackendCommit    MemoryBackendOperation = "commit"
	MemoryBackendDecommit  MemoryBackendOperation = "decommit"
	MemoryBackendRelease   MemoryBackendOperation = "release"
	MemoryBackendTrim      MemoryBackendOperation = "trim"
	MemoryBackendFootprint MemoryBackendOperation = "footprint"
)

type MemoryFootprintEvidenceClass string

const (
	MemoryFootprintMeasured    MemoryFootprintEvidenceClass = "runtime_measured"
	MemoryFootprintEstimated   MemoryFootprintEvidenceClass = "allocation_report_estimate"
	MemoryFootprintUnsupported MemoryFootprintEvidenceClass = "unsupported"
	MemoryFootprintBlocked     MemoryFootprintEvidenceClass = "blocked"
)

type MemoryBackendClass string

const (
	MemoryBackendClassNone             MemoryBackendClass = "none"
	MemoryBackendClassSmallHeap        MemoryBackendClass = "small_heap"
	MemoryBackendClassRegion           MemoryBackendClass = "region"
	MemoryBackendClassLargeBackend     MemoryBackendClass = "large_backend"
	MemoryBackendClassExternal         MemoryBackendClass = "external"
	MemoryBackendClassConservativeHeap MemoryBackendClass = "conservative_heap"
	MemoryBackendClassUnknown          MemoryBackendClass = "unknown"
)

const (
	SmallHeapAlignment     = smallheap.SmallHeapAlignment
	SmallHeapChunkBytes    = smallheap.SmallHeapChunkBytes
	SmallHeapMaxSmallBytes = smallheap.SmallHeapMaxSmallBytes
)

type SmallHeapClass = smallheap.SmallHeapClass
type SmallHeapConfig = smallheap.SmallHeapConfig

func RuntimeSmallHeapConfig() SmallHeapConfig {
	return smallheap.RuntimeSmallHeapConfig()
}

func SmallHeapClassForBytes(bytes int64) (SmallHeapClass, bool) {
	return smallheap.SmallHeapClassForBytes(bytes)
}

func AlignSmallHeapBytes(bytes int64) (int64, bool) {
	return smallheap.AlignSmallHeapBytes(bytes)
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
	allocator *smallheap.PerCoreSmallHeapAllocator
	backend   *MemoryBackendRuntime
	chunks    map[string]struct{}
}

func RuntimePerCoreSmallHeapABI(coreCount int) PerCoreSmallHeapABI {
	return perCoreSmallHeapABIFromLeaf(smallheap.RuntimePerCoreSmallHeapABI(coreCount))
}

func NewPerCoreSmallHeapAllocator(abi PerCoreSmallHeapABI) (*PerCoreSmallHeapAllocator, error) {
	allocator, err := smallheap.NewPerCoreSmallHeapAllocator(perCoreSmallHeapABIToLeaf(abi))
	if err != nil {
		return nil, err
	}
	ledger, err := NewMemoryDomainLedger(DefaultProcessMemoryDomain(0, 0))
	if err != nil {
		return nil, err
	}
	backend, err := NewMemoryBackendRuntime(MemoryBackendRuntimeOptions{
		Target:   "linux-x64",
		Ledger:   ledger,
		DomainID: "domain:process",
	})
	if err != nil {
		return nil, err
	}
	return &PerCoreSmallHeapAllocator{
		allocator: allocator,
		backend:   backend,
		chunks:    map[string]struct{}{},
	}, nil
}

func (allocator *PerCoreSmallHeapAllocator) Alloc(
	coreID int,
	bytes int64,
) (PerCoreSmallHeapHandle, error) {
	if allocator == nil || allocator.allocator == nil {
		return PerCoreSmallHeapHandle{}, fmt.Errorf(
			"per-core small heap allocator: allocator is nil",
		)
	}
	handle, err := allocator.allocator.Alloc(coreID, bytes)
	if err != nil {
		return PerCoreSmallHeapHandle{}, err
	}
	out := perCoreSmallHeapHandleFromLeaf(handle)
	if err := allocator.recordSmallHeapAlloc(out); err != nil {
		return PerCoreSmallHeapHandle{}, err
	}
	return out, nil
}

func (allocator *PerCoreSmallHeapAllocator) Free(handle PerCoreSmallHeapHandle) error {
	if allocator == nil || allocator.allocator == nil {
		return fmt.Errorf("per-core small heap allocator: allocator is nil")
	}
	if err := allocator.allocator.Free(perCoreSmallHeapHandleToLeaf(handle)); err != nil {
		return err
	}
	if allocator.backend != nil {
		if err := allocator.backend.RecordAllocationFree(handle.Domain.DomainID, int64(handle.RequestedBytes)); err != nil {
			return err
		}
	}
	return nil
}

func (allocator *PerCoreSmallHeapAllocator) Report() PerCoreSmallHeapReport {
	if allocator == nil || allocator.allocator == nil {
		return PerCoreSmallHeapReport{RuntimePath: AllocationPathPerCoreSmallHeap}
	}
	return perCoreSmallHeapReportFromLeaf(allocator.allocator.Report())
}

func (allocator *PerCoreSmallHeapAllocator) LedgerSnapshot() []MemoryDomain {
	if allocator == nil || allocator.backend == nil || allocator.backend.ledger == nil {
		return nil
	}
	return allocator.backend.ledger.Snapshot()
}

func (allocator *PerCoreSmallHeapAllocator) MemoryBackendEvents() []MemoryBackendRuntimeEvent {
	if allocator == nil || allocator.backend == nil {
		return nil
	}
	return allocator.backend.Events()
}

func (allocator *PerCoreSmallHeapAllocator) recordSmallHeapAlloc(
	handle PerCoreSmallHeapHandle,
) error {
	if allocator == nil || allocator.backend == nil {
		return nil
	}
	chunkKey := fmt.Sprintf("%d:%d", handle.CoreID, handle.ChunkID)
	if _, seen := allocator.chunks[chunkKey]; !seen {
		if err := allocator.backend.Reserve(handle.Domain.DomainID, int64(SmallHeapChunkBytes)); err != nil {
			return err
		}
		if err := allocator.backend.Commit(handle.Domain.DomainID, int64(SmallHeapChunkBytes)); err != nil {
			return err
		}
		allocator.chunks[chunkKey] = struct{}{}
	}
	return allocator.backend.RecordAllocation(handle.Domain.DomainID, int64(handle.RequestedBytes))
}

func perCoreSmallHeapABIToLeaf(abi PerCoreSmallHeapABI) smallheap.PerCoreSmallHeapABI {
	return smallheap.PerCoreSmallHeapABI{
		RuntimePath:          smallheap.AllocationRuntimePath(abi.RuntimePath),
		CoreCount:            abi.CoreCount,
		DefaultDomain:        smallHeapDomainToLeaf(abi.DefaultDomain),
		ChunkBytes:           abi.ChunkBytes,
		MaxSmallBytes:        abi.MaxSmallBytes,
		Alignment:            abi.Alignment,
		MetadataBytesPerCore: abi.MetadataBytesPerCore,
		MetadataFields:       abi.MetadataFields,
		Classes:              abi.Classes,
		ReusePolicy:          abi.ReusePolicy,
		LargeRuntimePath:     smallheap.AllocationRuntimePath(abi.LargeRuntimePath),
	}
}

func perCoreSmallHeapABIFromLeaf(abi smallheap.PerCoreSmallHeapABI) PerCoreSmallHeapABI {
	return PerCoreSmallHeapABI{
		RuntimePath:          AllocationRuntimePath(abi.RuntimePath),
		CoreCount:            abi.CoreCount,
		DefaultDomain:        smallHeapDomainFromLeaf(abi.DefaultDomain),
		ChunkBytes:           abi.ChunkBytes,
		MaxSmallBytes:        abi.MaxSmallBytes,
		Alignment:            abi.Alignment,
		MetadataBytesPerCore: abi.MetadataBytesPerCore,
		MetadataFields:       abi.MetadataFields,
		Classes:              abi.Classes,
		ReusePolicy:          abi.ReusePolicy,
		LargeRuntimePath:     AllocationRuntimePath(abi.LargeRuntimePath),
	}
}

func perCoreSmallHeapHandleToLeaf(handle PerCoreSmallHeapHandle) smallheap.PerCoreSmallHeapHandle {
	return smallheap.PerCoreSmallHeapHandle{
		BlockID:        handle.BlockID,
		Generation:     handle.Generation,
		CoreID:         handle.CoreID,
		ChunkID:        handle.ChunkID,
		Offset:         handle.Offset,
		RequestedBytes: handle.RequestedBytes,
		ReservedBytes:  handle.ReservedBytes,
		ClassName:      handle.ClassName,
		Reused:         handle.Reused,
		Domain:         smallHeapDomainToLeaf(handle.Domain),
	}
}

func perCoreSmallHeapHandleFromLeaf(
	handle smallheap.PerCoreSmallHeapHandle,
) PerCoreSmallHeapHandle {
	return PerCoreSmallHeapHandle{
		BlockID:        handle.BlockID,
		Generation:     handle.Generation,
		CoreID:         handle.CoreID,
		ChunkID:        handle.ChunkID,
		Offset:         handle.Offset,
		RequestedBytes: handle.RequestedBytes,
		ReservedBytes:  handle.ReservedBytes,
		ClassName:      handle.ClassName,
		Reused:         handle.Reused,
		Domain:         smallHeapDomainFromLeaf(handle.Domain),
	}
}

func perCoreSmallHeapReportFromLeaf(
	report smallheap.PerCoreSmallHeapReport,
) PerCoreSmallHeapReport {
	cores := make([]PerCoreSmallHeapCoreReport, 0, len(report.Cores))
	for _, core := range report.Cores {
		cores = append(cores, PerCoreSmallHeapCoreReport{
			CoreID:             core.CoreID,
			AllocationCount:    core.AllocationCount,
			FreeCount:          core.FreeCount,
			ReuseCount:         core.ReuseCount,
			ChunkRefills:       core.ChunkRefills,
			BumpOffset:         core.BumpOffset,
			BytesRequested:     core.BytesRequested,
			BytesReserved:      core.BytesReserved,
			FragmentationBytes: core.FragmentationBytes,
			Domain:             smallHeapDomainFromLeaf(core.Domain),
			FreeListBlocks:     core.FreeListBlocks,
		})
	}
	return PerCoreSmallHeapReport{
		RuntimePath:                AllocationRuntimePath(report.RuntimePath),
		CoreCount:                  report.CoreCount,
		TotalAllocations:           report.TotalAllocations,
		TotalFrees:                 report.TotalFrees,
		TotalReuses:                report.TotalReuses,
		TotalChunkRefills:          report.TotalChunkRefills,
		TotalMmapCalls:             report.TotalMmapCalls,
		BytesRequested:             report.BytesRequested,
		BytesReserved:              report.BytesReserved,
		FragmentationBytes:         report.FragmentationBytes,
		Domain:                     smallHeapDomainFromLeaf(report.Domain),
		EstimatedMmapPerAllocation: report.EstimatedMmapPerAllocation,
		Cores:                      cores,
	}
}

func smallHeapDomainToLeaf(domain MemoryDomain) smallheap.MemoryDomain {
	return smallheap.MemoryDomain{
		DomainID:         domain.DomainID,
		ParentDomainID:   domain.ParentDomainID,
		Kind:             smallheap.MemoryDomainKind(domain.Kind),
		OwnerKind:        domain.OwnerKind,
		OwnerID:          domain.OwnerID,
		Lifetime:         domain.Lifetime,
		BudgetBytes:      domain.BudgetBytes,
		RequestedBytes:   domain.RequestedBytes,
		ReservedBytes:    domain.ReservedBytes,
		CommittedBytes:   domain.CommittedBytes,
		ReleasedBytes:    domain.ReleasedBytes,
		State:            smallheap.MemoryDomainState(domain.State),
		Epoch:            domain.Epoch,
		DecommittedBytes: domain.DecommittedBytes,
		CurrentBytes:     domain.CurrentBytes,
		PeakBytes:        domain.PeakBytes,
		CopyCount:        domain.CopyCount,
		BytesCopied:      domain.BytesCopied,
	}
}

func smallHeapDomainFromLeaf(domain smallheap.MemoryDomain) MemoryDomain {
	return MemoryDomain{
		DomainID:         domain.DomainID,
		ParentDomainID:   domain.ParentDomainID,
		Kind:             MemoryDomainKind(domain.Kind),
		OwnerKind:        domain.OwnerKind,
		OwnerID:          domain.OwnerID,
		Lifetime:         domain.Lifetime,
		BudgetBytes:      domain.BudgetBytes,
		RequestedBytes:   domain.RequestedBytes,
		ReservedBytes:    domain.ReservedBytes,
		CommittedBytes:   domain.CommittedBytes,
		ReleasedBytes:    domain.ReleasedBytes,
		State:            MemoryDomainState(domain.State),
		Epoch:            domain.Epoch,
		DecommittedBytes: domain.DecommittedBytes,
		CurrentBytes:     domain.CurrentBytes,
		PeakBytes:        domain.PeakBytes,
		CopyCount:        domain.CopyCount,
		BytesCopied:      domain.BytesCopied,
	}
}

type MemoryBackendContract struct {
	Schema                 string                          `json:"schema"`
	Target                 string                          `json:"target"`
	Operations             []MemoryBackendOperation        `json:"operations"`
	OperationSupport       []MemoryBackendOperationSupport `json:"operation_support"`
	MinAlignmentBytes      int                             `json:"min_alignment_bytes"`
	ReserveGranularity     int                             `json:"reserve_granularity_bytes"`
	CommitGranularity      int                             `json:"commit_granularity_bytes"`
	FootprintEvidenceClass MemoryFootprintEvidenceClass    `json:"footprint_evidence_class"`
	FootprintMethod        string                          `json:"footprint_method"`
	UnsupportedReason      string                          `json:"unsupported_reason,omitempty"`
	NonClaims              []string                        `json:"non_claims,omitempty"`
}

type MemoryBackendOperationSupport struct {
	Operation         MemoryBackendOperation `json:"operation"`
	Supported         bool                   `json:"supported"`
	Method            string                 `json:"method,omitempty"`
	UnsupportedReason string                 `json:"unsupported_reason,omitempty"`
}

type MemoryBackendAllocationEvidence struct {
	Schema                string                       `json:"schema"`
	BackendClass          MemoryBackendClass           `json:"backend_class"`
	Adapter               string                       `json:"adapter"`
	RuntimePath           AllocationRuntimePath        `json:"runtime_path"`
	Operations            []MemoryBackendOperation     `json:"operations,omitempty"`
	EvidenceClass         MemoryFootprintEvidenceClass `json:"evidence_class"`
	Method                string                       `json:"method"`
	ReserveBytes          int64                        `json:"reserve_bytes,omitempty"`
	CommitBytes           int64                        `json:"commit_bytes,omitempty"`
	DecommitBytes         int64                        `json:"decommit_bytes,omitempty"`
	ReleaseBytes          int64                        `json:"release_bytes,omitempty"`
	FootprintCurrentBytes int64                        `json:"footprint_current_bytes,omitempty"`
	FootprintPeakBytes    int64                        `json:"footprint_peak_bytes,omitempty"`
	UnsupportedReason     string                       `json:"unsupported_reason,omitempty"`
	BlockedReason         string                       `json:"blocked_reason,omitempty"`
}

type MemoryFootprintSample struct {
	Target            string                       `json:"target"`
	EvidenceClass     MemoryFootprintEvidenceClass `json:"evidence_class"`
	Method            string                       `json:"method"`
	CurrentBytes      int64                        `json:"current_bytes,omitempty"`
	PeakBytes         int64                        `json:"peak_bytes,omitempty"`
	UnsupportedReason string                       `json:"unsupported_reason,omitempty"`
	BlockedReason     string                       `json:"blocked_reason,omitempty"`
}

type MemoryBackendRuntimeEvent struct {
	Target    string                 `json:"target"`
	Operation MemoryBackendOperation `json:"operation"`
	DomainID  string                 `json:"domain_id"`
	Bytes     int64                  `json:"bytes,omitempty"`
	Method    string                 `json:"method"`
}

type MemoryBackendTelemetryHook func(MemoryBackendRuntimeEvent) error

type MemoryBackendRuntimeOptions struct {
	Target   string
	Ledger   *MemoryDomainLedger
	DomainID string
	Hook     MemoryBackendTelemetryHook
}

type MemoryBackendRuntime struct {
	contract MemoryBackendContract
	ledger   *MemoryDomainLedger
	domainID string
	hook     MemoryBackendTelemetryHook
	events   []MemoryBackendRuntimeEvent
}

func RuntimeMemoryBackendContract(target string) MemoryBackendContract {
	support := MemoryBackendSupportMatrix(target)
	contract := MemoryBackendContract{
		Schema:             MemoryBackendContractSchemaV1,
		Target:             target,
		Operations:         supportedMemoryBackendOperations(support),
		OperationSupport:   support,
		MinAlignmentBytes:  SmallHeapAlignment,
		ReserveGranularity: 4096,
		CommitGranularity:  4096,
		NonClaims: []string{
			"no zero heap for all programs claim",
			"no all-target RSS parity claim",
			"no performance claim",
		},
	}
	switch target {
	case "linux-x64":
		contract.FootprintEvidenceClass = MemoryFootprintMeasured
		contract.FootprintMethod = "linux_proc_self_status_vmrss_vmhwm"
	case "wasm32-wasi", "wasm32-web":
		contract.FootprintEvidenceClass = MemoryFootprintUnsupported
		contract.FootprintMethod = "unsupported_host_footprint"
		contract.UnsupportedReason = ("host RSS is unavailable for the current linear memory " +
			"target boundary")
	default:
		contract.FootprintEvidenceClass = MemoryFootprintUnsupported
		contract.FootprintMethod = "adapter_not_implemented"
		contract.UnsupportedReason = ("target footprint adapter is not implemented in the " +
			"current memory backend contract")
	}
	return contract
}

func MemoryBackendSupportMatrix(target string) []MemoryBackendOperationSupport {
	switch target {
	case "linux-x64":
		return []MemoryBackendOperationSupport{
			{Operation: MemoryBackendReserve, Supported: true, Method: "linux_mmap_private_anonymous_prot_none"},
			{Operation: MemoryBackendCommit, Supported: true, Method: "linux_mprotect_read_write"},
			{Operation: MemoryBackendDecommit, Supported: true, Method: "linux_madvise_dontneed_mprotect_none"},
			{Operation: MemoryBackendRelease, Supported: true, Method: "linux_munmap"},
			{Operation: MemoryBackendTrim, Supported: true, Method: "linux_allocator_trim_v1"},
			{Operation: MemoryBackendFootprint, Supported: true, Method: "linux_proc_self_status_vmrss_vmhwm"},
		}
	case "wasm32-wasi", "wasm32-web":
		reason := "linear memory supports growth but not host reservation release or RSS"
		return []MemoryBackendOperationSupport{
			{Operation: MemoryBackendReserve, Supported: true, Method: "wasm_memory_grow_combined_reserve_commit"},
			{Operation: MemoryBackendCommit, Supported: true, Method: "wasm_memory_grow_combined_reserve_commit"},
			{Operation: MemoryBackendDecommit, UnsupportedReason: reason},
			{Operation: MemoryBackendRelease, UnsupportedReason: reason},
			{Operation: MemoryBackendTrim, UnsupportedReason: reason},
			{Operation: MemoryBackendFootprint, UnsupportedReason: "host RSS is unavailable across the WASM boundary"},
		}
	default:
		reason := "target memory backend adapter is not implemented"
		return []MemoryBackendOperationSupport{
			{Operation: MemoryBackendReserve, UnsupportedReason: reason},
			{Operation: MemoryBackendCommit, UnsupportedReason: reason},
			{Operation: MemoryBackendDecommit, UnsupportedReason: reason},
			{Operation: MemoryBackendRelease, UnsupportedReason: reason},
			{Operation: MemoryBackendTrim, UnsupportedReason: reason},
			{Operation: MemoryBackendFootprint, UnsupportedReason: reason},
		}
	}
}

func supportedMemoryBackendOperations(
	support []MemoryBackendOperationSupport,
) []MemoryBackendOperation {
	ops := make([]MemoryBackendOperation, 0, len(support))
	for _, row := range support {
		if row.Supported {
			ops = append(ops, row.Operation)
		}
	}
	return ops
}

func RequiredMemoryBackendOperations() []MemoryBackendOperation {
	return []MemoryBackendOperation{
		MemoryBackendReserve,
		MemoryBackendCommit,
		MemoryBackendDecommit,
		MemoryBackendRelease,
		MemoryBackendTrim,
		MemoryBackendFootprint,
	}
}

func RequiredMemoryBackendSymbols() []string {
	return []string{
		MemoryBackendReserveSymbol,
		MemoryBackendCommitSymbol,
		MemoryBackendDecommitSymbol,
		MemoryBackendReleaseSymbol,
		MemoryBackendTrimSymbol,
		MemoryBackendFootprintSymbol,
	}
}

func (contract MemoryBackendContract) SupportsOperation(op MemoryBackendOperation) bool {
	row, ok := contract.OperationSupportFor(op)
	if ok {
		return row.Supported
	}
	for _, candidate := range contract.Operations {
		if candidate == op {
			return true
		}
	}
	return false
}

func (contract MemoryBackendContract) OperationSupportFor(
	op MemoryBackendOperation,
) (MemoryBackendOperationSupport, bool) {
	for _, row := range contract.OperationSupport {
		if row.Operation == op {
			return row, true
		}
	}
	return MemoryBackendOperationSupport{}, false
}

func ValidateMemoryBackendContract(contract MemoryBackendContract) error {
	if contract.Schema != MemoryBackendContractSchemaV1 {
		return fmt.Errorf(
			"memory backend contract: schema = %q, want %s",
			contract.Schema,
			MemoryBackendContractSchemaV1,
		)
	}
	if contract.Target == "" {
		return fmt.Errorf("memory backend contract: target is required")
	}
	if contract.MinAlignmentBytes <= 0 || !isPowerOfTwo(contract.MinAlignmentBytes) {
		return fmt.Errorf(
			"memory backend contract %s: min alignment must be a positive power-of-two",
			contract.Target,
		)
	}
	if contract.ReserveGranularity <= 0 || contract.CommitGranularity <= 0 {
		return fmt.Errorf(
			"memory backend contract %s: reserve and commit granularity must be positive",
			contract.Target,
		)
	}
	if err := validateMemoryBackendSupportRows(contract); err != nil {
		return err
	}
	sample := MemoryFootprintSample{
		Target:            contract.Target,
		EvidenceClass:     contract.FootprintEvidenceClass,
		Method:            contract.FootprintMethod,
		UnsupportedReason: contract.UnsupportedReason,
	}
	if contract.FootprintEvidenceClass == MemoryFootprintMeasured ||
		contract.FootprintEvidenceClass == MemoryFootprintEstimated {
		sample.CurrentBytes = 1
		sample.PeakBytes = 1
	}
	if err := ValidateMemoryFootprintSample(sample); err != nil {
		return fmt.Errorf("memory backend contract %s: footprint policy: %w", contract.Target, err)
	}
	return nil
}

func validateMemoryBackendSupportRows(contract MemoryBackendContract) error {
	if len(contract.OperationSupport) != len(RequiredMemoryBackendOperations()) {
		return fmt.Errorf(
			"memory backend contract %s: operation support rows = %d, want %d",
			contract.Target,
			len(contract.OperationSupport),
			len(RequiredMemoryBackendOperations()),
		)
	}
	seen := map[MemoryBackendOperation]MemoryBackendOperationSupport{}
	for _, row := range contract.OperationSupport {
		if !isKnownMemoryBackendOperation(row.Operation) {
			return fmt.Errorf(
				"memory backend contract %s: unknown operation %q",
				contract.Target,
				row.Operation,
			)
		}
		if _, exists := seen[row.Operation]; exists {
			return fmt.Errorf(
				"memory backend contract %s: duplicate operation support row %s",
				contract.Target,
				row.Operation,
			)
		}
		if row.Supported {
			if strings.TrimSpace(row.Method) == "" {
				return fmt.Errorf(
					"memory backend contract %s: supported operation %s requires method",
					contract.Target,
					row.Operation,
				)
			}
			if strings.TrimSpace(row.UnsupportedReason) != "" {
				return fmt.Errorf(
					"memory backend contract %s: supported operation %s must not include unsupported_reason",
					contract.Target,
					row.Operation,
				)
			}
		} else if strings.TrimSpace(row.UnsupportedReason) == "" {
			return fmt.Errorf(
				"memory backend contract %s: unsupported operation %s requires unsupported_reason",
				contract.Target,
				row.Operation,
			)
		}
		seen[row.Operation] = row
	}
	for _, op := range RequiredMemoryBackendOperations() {
		row, ok := seen[op]
		if !ok {
			return fmt.Errorf(
				"memory backend contract %s: missing operation support row %s",
				contract.Target,
				op,
			)
		}
		if contract.Target == "linux-x64" && !row.Supported {
			return fmt.Errorf(
				"memory backend contract %s: linux-x64 operation %s must be supported",
				contract.Target,
				op,
			)
		}
	}
	if len(contract.Operations) != len(supportedMemoryBackendOperations(contract.OperationSupport)) {
		return fmt.Errorf(
			"memory backend contract %s: operations must mirror supported rows",
			contract.Target,
		)
	}
	for _, op := range contract.Operations {
		row, ok := seen[op]
		if !ok || !row.Supported {
			return fmt.Errorf(
				"memory backend contract %s: operation %s is not a supported row",
				contract.Target,
				op,
			)
		}
	}
	return nil
}

func NewMemoryBackendRuntime(options MemoryBackendRuntimeOptions) (*MemoryBackendRuntime, error) {
	target := strings.TrimSpace(options.Target)
	if target == "" {
		target = "linux-x64"
	}
	domainID := strings.TrimSpace(options.DomainID)
	if domainID == "" {
		domainID = "domain:process"
	}
	contract := RuntimeMemoryBackendContract(target)
	if err := ValidateMemoryBackendContract(contract); err != nil {
		return nil, err
	}
	return &MemoryBackendRuntime{
		contract: contract,
		ledger:   options.Ledger,
		domainID: domainID,
		hook:     options.Hook,
	}, nil
}

func (backend *MemoryBackendRuntime) Contract() MemoryBackendContract {
	if backend == nil {
		return MemoryBackendContract{}
	}
	return backend.contract
}

func (backend *MemoryBackendRuntime) Events() []MemoryBackendRuntimeEvent {
	if backend == nil {
		return nil
	}
	out := make([]MemoryBackendRuntimeEvent, len(backend.events))
	copy(out, backend.events)
	return out
}

func (backend *MemoryBackendRuntime) Reserve(domainID string, bytes int64) error {
	return backend.applyBackendOperation(MemoryBackendReserve, domainID, bytes, DomainEventReserve)
}

func (backend *MemoryBackendRuntime) Commit(domainID string, bytes int64) error {
	return backend.applyBackendOperation(MemoryBackendCommit, domainID, bytes, DomainEventCommit)
}

func (backend *MemoryBackendRuntime) Decommit(domainID string, bytes int64) error {
	return backend.applyBackendOperation(MemoryBackendDecommit, domainID, bytes, DomainEventDecommit)
}

func (backend *MemoryBackendRuntime) Release(domainID string, bytes int64) error {
	return backend.applyBackendOperation(MemoryBackendRelease, domainID, bytes, DomainEventRelease)
}

func (backend *MemoryBackendRuntime) Trim(domainID string, bytes int64) error {
	return backend.applyBackendOperation(MemoryBackendTrim, domainID, bytes, DomainEventTrim)
}

func (backend *MemoryBackendRuntime) RecordAllocation(domainID string, bytes int64) error {
	if backend == nil || backend.ledger == nil {
		return nil
	}
	id := backend.resolveDomainID(domainID)
	if err := backend.ledger.Apply(MemoryDomainEvent{
		Kind:     DomainEventRequest,
		DomainID: id,
		Bytes:    bytes,
	}); err != nil {
		return err
	}
	return backend.ledger.Apply(MemoryDomainEvent{
		Kind:     DomainEventAllocate,
		DomainID: id,
		Bytes:    bytes,
	})
}

func (backend *MemoryBackendRuntime) RecordAllocationFree(domainID string, bytes int64) error {
	if backend == nil || backend.ledger == nil {
		return nil
	}
	return backend.ledger.Apply(MemoryDomainEvent{
		Kind:     DomainEventFree,
		DomainID: backend.resolveDomainID(domainID),
		Bytes:    bytes,
	})
}

func (backend *MemoryBackendRuntime) applyBackendOperation(
	op MemoryBackendOperation,
	domainID string,
	bytes int64,
	ledgerEvent MemoryDomainEventKind,
) error {
	if backend == nil {
		return fmt.Errorf("memory backend runtime: backend is nil")
	}
	if bytes <= 0 {
		return fmt.Errorf("memory backend runtime %s: bytes must be positive", op)
	}
	row, ok := backend.contract.OperationSupportFor(op)
	if !ok {
		return fmt.Errorf("memory backend runtime %s: support row missing", op)
	}
	if !row.Supported {
		return fmt.Errorf(
			"memory backend runtime %s: unsupported: %s",
			op,
			row.UnsupportedReason,
		)
	}
	id := backend.resolveDomainID(domainID)
	event := MemoryBackendRuntimeEvent{
		Target:    backend.contract.Target,
		Operation: op,
		DomainID:  id,
		Bytes:     bytes,
		Method:    row.Method,
	}
	if backend.hook != nil {
		if err := backend.hook(event); err != nil {
			return err
		}
	}
	backend.events = append(backend.events, event)
	if backend.ledger == nil {
		return nil
	}
	return backend.ledger.Apply(MemoryDomainEvent{
		Kind:     ledgerEvent,
		DomainID: id,
		Bytes:    bytes,
	})
}

func (backend *MemoryBackendRuntime) resolveDomainID(domainID string) string {
	if strings.TrimSpace(domainID) != "" {
		return domainID
	}
	if backend == nil || backend.domainID == "" {
		return "domain:process"
	}
	return backend.domainID
}

func MeasureMemoryFootprint(target string) MemoryFootprintSample {
	contract := RuntimeMemoryBackendContract(target)
	switch target {
	case "linux-x64":
		current, peak, err := readLinuxProcSelfStatusFootprint()
		if err != nil {
			return MemoryFootprintSample{
				Target:        target,
				EvidenceClass: MemoryFootprintBlocked,
				Method:        contract.FootprintMethod,
				BlockedReason: err.Error(),
			}
		}
		return MemoryFootprintSample{
			Target:        target,
			EvidenceClass: MemoryFootprintMeasured,
			Method:        contract.FootprintMethod,
			CurrentBytes:  current,
			PeakBytes:     peak,
		}
	case "wasm32-wasi", "wasm32-web":
		return MemoryFootprintSample{
			Target:            target,
			EvidenceClass:     MemoryFootprintUnsupported,
			Method:            contract.FootprintMethod,
			UnsupportedReason: contract.UnsupportedReason,
		}
	default:
		return MemoryFootprintSample{
			Target:            target,
			EvidenceClass:     MemoryFootprintUnsupported,
			Method:            contract.FootprintMethod,
			UnsupportedReason: contract.UnsupportedReason,
		}
	}
}

func readLinuxProcSelfStatusFootprint() (int64, int64, error) {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0, 0, fmt.Errorf("linux footprint: read /proc/self/status: %w", err)
	}
	current, peak, err := parseLinuxProcStatusFootprint(string(data))
	if err != nil {
		return 0, 0, err
	}
	if current <= 0 || peak <= 0 {
		return 0, 0, fmt.Errorf("linux footprint: VmRSS/VmHWM must be positive")
	}
	if peak < current {
		return 0, 0, fmt.Errorf("linux footprint: VmHWM is below VmRSS")
	}
	return current, peak, nil
}

func parseLinuxProcStatusFootprint(status string) (int64, int64, error) {
	var current int64
	var peak int64
	for _, line := range strings.Split(status, "\n") {
		switch {
		case strings.HasPrefix(line, "VmRSS:"):
			bytes, err := parseProcStatusKiBLine(line)
			if err != nil {
				return 0, 0, fmt.Errorf("linux footprint: VmRSS: %w", err)
			}
			current = bytes
		case strings.HasPrefix(line, "VmHWM:"):
			bytes, err := parseProcStatusKiBLine(line)
			if err != nil {
				return 0, 0, fmt.Errorf("linux footprint: VmHWM: %w", err)
			}
			peak = bytes
		}
	}
	if current == 0 || peak == 0 {
		return 0, 0, fmt.Errorf("linux footprint: missing VmRSS or VmHWM in /proc/self/status")
	}
	return current, peak, nil
}

func parseProcStatusKiBLine(line string) (int64, error) {
	fields := strings.Fields(line)
	if len(fields) < 3 || fields[2] != "kB" {
		return 0, fmt.Errorf("expected '<field>: <value> kB', got %q", line)
	}
	kib, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0, err
	}
	if kib <= 0 {
		return 0, fmt.Errorf("kilobytes must be positive")
	}
	return kib * 1024, nil
}

func ValidateMemoryBackendAllocationEvidence(evidence MemoryBackendAllocationEvidence) error {
	if evidence.Schema != MemoryBackendAllocationEvidenceSchemaV1 {
		return fmt.Errorf(
			"memory backend allocation evidence: schema = %q, want %s",
			evidence.Schema,
			MemoryBackendAllocationEvidenceSchemaV1,
		)
	}
	if !isKnownMemoryBackendClass(evidence.BackendClass) {
		return fmt.Errorf(
			"memory backend allocation evidence: backend_class %q is not supported",
			evidence.BackendClass,
		)
	}
	if evidence.Adapter == "" {
		return fmt.Errorf(
			"memory backend allocation evidence %s: adapter is required",
			evidence.BackendClass,
		)
	}
	if evidence.RuntimePath == "" {
		return fmt.Errorf(
			"memory backend allocation evidence %s: runtime_path is required",
			evidence.BackendClass,
		)
	}
	if evidence.Method == "" {
		return fmt.Errorf(
			"memory backend allocation evidence %s: method is required",
			evidence.BackendClass,
		)
	}
	if evidence.ReserveBytes < 0 || evidence.CommitBytes < 0 || evidence.DecommitBytes < 0 ||
		evidence.ReleaseBytes < 0 ||
		evidence.FootprintCurrentBytes < 0 ||
		evidence.FootprintPeakBytes < 0 {
		return fmt.Errorf(
			"memory backend allocation evidence %s: byte counts must be non-negative",
			evidence.BackendClass,
		)
	}
	if evidence.FootprintPeakBytes < evidence.FootprintCurrentBytes {
		return fmt.Errorf(
			"memory backend allocation evidence %s: footprint peak bytes must be >= current bytes",
			evidence.BackendClass,
		)
	}
	switch evidence.EvidenceClass {
	case MemoryFootprintMeasured, MemoryFootprintEstimated:
		if evidence.BlockedReason != "" || evidence.UnsupportedReason != "" {
			return fmt.Errorf(
				("memory backend allocation evidence %s: measured/estimated " +
					"evidence must not include blocked or unsupported reason"),
				evidence.BackendClass,
			)
		}
		if len(evidence.Operations) == 0 {
			return fmt.Errorf(
				("memory backend allocation evidence %s: operations are required " +
					"for measured/estimated evidence"),
				evidence.BackendClass,
			)
		}
		for _, op := range evidence.Operations {
			if !isKnownMemoryBackendOperation(op) {
				return fmt.Errorf(
					"memory backend allocation evidence %s: operation %q is not supported",
					evidence.BackendClass,
					op,
				)
			}
		}
		if evidence.ReserveBytes > 0 && evidence.CommitBytes > evidence.ReserveBytes {
			return fmt.Errorf(
				"memory backend allocation evidence %s: commit bytes must be <= reserve bytes",
				evidence.BackendClass,
			)
		}
		if evidence.ReserveBytes > 0 && evidence.ReleaseBytes > evidence.ReserveBytes {
			return fmt.Errorf(
				"memory backend allocation evidence %s: release bytes must be <= reserve bytes",
				evidence.BackendClass,
			)
		}
	case MemoryFootprintUnsupported:
		if evidence.UnsupportedReason == "" {
			return fmt.Errorf(
				"memory backend allocation evidence %s: unsupported_reason is required",
				evidence.BackendClass,
			)
		}
		if evidence.BlockedReason != "" {
			return fmt.Errorf(
				"memory backend allocation evidence %s: unsupported evidence must not include blocked_reason",
				evidence.BackendClass,
			)
		}
		if evidence.ReserveBytes != 0 || evidence.CommitBytes != 0 || evidence.DecommitBytes != 0 ||
			evidence.ReleaseBytes != 0 ||
			evidence.FootprintCurrentBytes != 0 ||
			evidence.FootprintPeakBytes != 0 {
			return fmt.Errorf(
				"memory backend allocation evidence %s: unsupported evidence must not include byte counts",
				evidence.BackendClass,
			)
		}
	case MemoryFootprintBlocked:
		if evidence.BlockedReason == "" {
			return fmt.Errorf(
				"memory backend allocation evidence %s: blocked_reason is required",
				evidence.BackendClass,
			)
		}
		if evidence.UnsupportedReason != "" {
			return fmt.Errorf(
				"memory backend allocation evidence %s: blocked evidence must not include unsupported_reason",
				evidence.BackendClass,
			)
		}
		if evidence.ReserveBytes != 0 || evidence.CommitBytes != 0 || evidence.DecommitBytes != 0 ||
			evidence.ReleaseBytes != 0 ||
			evidence.FootprintCurrentBytes != 0 ||
			evidence.FootprintPeakBytes != 0 {
			return fmt.Errorf(
				"memory backend allocation evidence %s: blocked evidence must not include byte counts",
				evidence.BackendClass,
			)
		}
	default:
		return fmt.Errorf(
			"memory backend allocation evidence %s: evidence_class %q is not supported",
			evidence.BackendClass,
			evidence.EvidenceClass,
		)
	}
	return nil
}

func isKnownMemoryBackendClass(class MemoryBackendClass) bool {
	switch class {
	case MemoryBackendClassNone,
		MemoryBackendClassSmallHeap,
		MemoryBackendClassRegion,
		MemoryBackendClassLargeBackend,
		MemoryBackendClassExternal,
		MemoryBackendClassConservativeHeap,
		MemoryBackendClassUnknown:
		return true
	default:
		return false
	}
}

func isKnownMemoryBackendOperation(op MemoryBackendOperation) bool {
	switch op {
	case MemoryBackendReserve,
		MemoryBackendCommit,
		MemoryBackendDecommit,
		MemoryBackendRelease,
		MemoryBackendTrim,
		MemoryBackendFootprint:
		return true
	default:
		return false
	}
}

func ValidateMemoryFootprintSample(sample MemoryFootprintSample) error {
	if sample.Target == "" {
		return fmt.Errorf("memory footprint sample: target is required")
	}
	if sample.Method == "" {
		return fmt.Errorf("memory footprint sample %s: method is required", sample.Target)
	}
	switch sample.EvidenceClass {
	case MemoryFootprintMeasured, MemoryFootprintEstimated:
		if sample.BlockedReason != "" || sample.UnsupportedReason != "" {
			return fmt.Errorf(
				("memory footprint sample %s: measured/estimated sample must not " +
					"include blocked or unsupported reason"),
				sample.Target,
			)
		}
		if sample.CurrentBytes < 0 || sample.PeakBytes < 0 {
			return fmt.Errorf(
				"memory footprint sample %s: byte counts must be non-negative",
				sample.Target,
			)
		}
		if sample.PeakBytes < sample.CurrentBytes {
			return fmt.Errorf(
				"memory footprint sample %s: peak bytes must be >= current bytes",
				sample.Target,
			)
		}
	case MemoryFootprintUnsupported:
		if sample.UnsupportedReason == "" {
			return fmt.Errorf(
				"memory footprint sample %s: unsupported_reason is required",
				sample.Target,
			)
		}
		if sample.CurrentBytes != 0 || sample.PeakBytes != 0 {
			return fmt.Errorf(
				"memory footprint sample %s: unsupported sample must not include byte counts",
				sample.Target,
			)
		}
	case MemoryFootprintBlocked:
		if sample.BlockedReason == "" {
			return fmt.Errorf(
				"memory footprint sample %s: blocked_reason is required",
				sample.Target,
			)
		}
		if sample.CurrentBytes != 0 || sample.PeakBytes != 0 {
			return fmt.Errorf(
				"memory footprint sample %s: blocked sample must not include byte counts",
				sample.Target,
			)
		}
	default:
		return fmt.Errorf(
			"memory footprint sample %s: evidence_class %q is not supported",
			sample.Target,
			sample.EvidenceClass,
		)
	}
	return nil
}
