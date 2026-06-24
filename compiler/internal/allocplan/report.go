package allocplan

import (
	"fmt"
	"sort"
	"strings"

	"tetra_language/compiler/internal/runtimeabi"
)

func Summarize(plan *Plan) ReportSummary {
	summary := ReportSummary{
		StorageClasses:               map[string]int{},
		ActualLoweringStorageClasses: map[string]int{},
		RuntimePaths:                 map[string]int{},
		AllocatorClasses:             map[string]int{},
		AllocatorScopes:              map[string]int{},
		AllocatorReusePolicies:       map[string]int{},
		MemoryBackendClasses:         map[string]int{},
		MemoryBackendOperations:      map[string]int{},
		MemoryBackendEvidenceClasses: map[string]int{},
		HeapReasonCodes:              map[string]int{},
		RawPointerBoundsStatuses:     map[string]int{},
		RawSlicePolicies:             map[string]int{},
	}
	if plan == nil {
		return summary
	}
	regions := map[string]RegionReportSummary{}
	var domains []runtimeabi.MemoryDomain
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
			for _, code := range alloc.HeapReasonCodes {
				summary.HeapReasonCodes[code]++
			}
			if alloc.RawPointerBoundsStatus != "" {
				summary.RawPointerBoundsStatuses[alloc.RawPointerBoundsStatus]++
			}
			if alloc.RawSlicePolicy != "" {
				summary.RawSlicePolicies[alloc.RawSlicePolicy]++
			}
			requested := allocationReportBytesRequested(alloc)
			reserved := allocationReportBytesReserved(alloc)
			committed := allocationReportBytesCommitted(alloc)
			released := allocationReportBytesReleased(alloc)
			summary.BytesRequested += requested
			summary.BytesReserved += reserved
			summary.BytesCommitted += committed
			summary.BytesReleased += released
			if alloc.MemoryBackend != nil {
				if alloc.MemoryBackend.BackendClass != "" {
					summary.MemoryBackendClasses[string(alloc.MemoryBackend.BackendClass)]++
				}
				if alloc.MemoryBackend.EvidenceClass != "" {
					summary.MemoryBackendEvidenceClasses[string(alloc.MemoryBackend.EvidenceClass)]++
				}
				seenOps := map[runtimeabi.MemoryBackendOperation]bool{}
				for _, op := range alloc.MemoryBackend.Operations {
					if seenOps[op] {
						continue
					}
					seenOps[op] = true
					summary.MemoryBackendOperations[string(op)]++
				}
			}
			if alloc.Domain != nil {
				domains = append(domains, *alloc.Domain)
			}
			if alloc.RegionID == "" {
				continue
			}
			switch alloc.ActualLoweringStorage {
			case StorageRegion,
				StorageFunctionTempRegion,
				StorageExplicitIsland,
				StorageTaskRegion,
				StorageActorMoveRegion:
			default:
				continue
			}
			key := alloc.RegionID + "\x00" + alloc.Lifetime + "\x00" + string(
				alloc.PlannedStorage,
			) + "\x00" + runtimePath
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
	summary.Domains = runtimeabi.AggregateMemoryDomainSummary(domains)
	return summary
}

func MemoryDomains(plan *Plan) []runtimeabi.MemoryDomain {
	if plan == nil {
		return nil
	}
	byKey := map[string]runtimeabi.MemoryDomain{}
	for _, fn := range plan.Functions {
		for _, alloc := range fn.Allocations {
			if alloc.Domain == nil {
				continue
			}
			domain := *alloc.Domain
			key := string(domain.Kind) + "\x00" + domain.DomainID + "\x00" +
				domain.ParentDomainID + "\x00" + domain.OwnerKind + "\x00" +
				domain.OwnerID + "\x00" + domain.Lifetime
			existing := byKey[key]
			if existing.DomainID == "" {
				byKey[key] = domain
				continue
			}
			existing.BudgetBytes += domain.BudgetBytes
			existing.RequestedBytes += domain.RequestedBytes
			existing.ReservedBytes += domain.ReservedBytes
			existing.CommittedBytes += domain.CommittedBytes
			existing.ReleasedBytes += domain.ReleasedBytes
			existing.CurrentBytes += domain.CurrentBytes
			if domain.PeakBytes > existing.PeakBytes {
				existing.PeakBytes = domain.PeakBytes
			}
			existing.CopyCount += domain.CopyCount
			existing.BytesCopied += domain.BytesCopied
			byKey[key] = existing
		}
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]runtimeabi.MemoryDomain, 0, len(keys))
	for _, key := range keys {
		out = append(out, byKey[key])
	}
	return out
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

func allocationReportBytesCommitted(alloc Allocation) int {
	if alloc.BytesCommitted > 0 {
		return alloc.BytesCommitted
	}
	if alloc.MemoryBackend == nil {
		return 0
	}
	switch alloc.MemoryBackend.EvidenceClass {
	case runtimeabi.MemoryFootprintMeasured, runtimeabi.MemoryFootprintEstimated:
		return int(alloc.MemoryBackend.CommitBytes)
	default:
		return 0
	}
}

func allocationReportBytesReleased(alloc Allocation) int {
	if alloc.BytesReleased > 0 {
		return alloc.BytesReleased
	}
	if alloc.MemoryBackend == nil {
		return 0
	}
	switch alloc.MemoryBackend.EvidenceClass {
	case runtimeabi.MemoryFootprintMeasured, runtimeabi.MemoryFootprintEstimated:
		return int(alloc.MemoryBackend.ReleaseBytes)
	default:
		return 0
	}
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
			fmt.Fprintf(
				&b,
				"  %s: site_id: %s builtin: %s planned_storage: %s actual_lowering_storage: %s escape: %s",
				alloc.ID,
				alloc.SiteID,
				alloc.Builtin,
				alloc.PlannedStorage,
				alloc.ActualLoweringStorage,
				alloc.Escape,
			)
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
			if alloc.MemoryBackend != nil {
				fmt.Fprintf(&b, " memory_backend: %s", alloc.MemoryBackend.BackendClass)
				if alloc.MemoryBackend.Adapter != "" {
					fmt.Fprintf(&b, " memory_backend_adapter: %s", alloc.MemoryBackend.Adapter)
				}
				if len(alloc.MemoryBackend.Operations) > 0 {
					fmt.Fprintf(
						&b,
						" memory_backend_ops: %s",
						formatMemoryBackendOperations(alloc.MemoryBackend.Operations),
					)
				}
				if alloc.MemoryBackend.EvidenceClass != "" {
					fmt.Fprintf(
						&b,
						" memory_backend_evidence: %s",
						alloc.MemoryBackend.EvidenceClass,
					)
				}
				if alloc.MemoryBackend.BlockedReason != "" {
					fmt.Fprintf(
						&b,
						" memory_backend_blocked: %s",
						alloc.MemoryBackend.BlockedReason,
					)
				}
				if alloc.MemoryBackend.UnsupportedReason != "" {
					fmt.Fprintf(
						&b,
						" memory_backend_unsupported: %s",
						alloc.MemoryBackend.UnsupportedReason,
					)
				}
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
			if alloc.BytesCommitted > 0 {
				fmt.Fprintf(&b, " bytes_committed: %d", alloc.BytesCommitted)
			}
			if alloc.BytesReleased > 0 {
				fmt.Fprintf(&b, " bytes_released: %d", alloc.BytesReleased)
			}
			if alloc.RegionID != "" {
				fmt.Fprintf(&b, " region_id: %s", alloc.RegionID)
			}
			if alloc.Lifetime != "" {
				fmt.Fprintf(&b, " lifetime: %s", alloc.Lifetime)
			}
			if alloc.Domain != nil {
				fmt.Fprintf(
					&b,
					" domain_id: %s domain_kind: %s",
					alloc.Domain.DomainID,
					alloc.Domain.Kind,
				)
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
			if len(alloc.HeapReasonCodes) > 0 {
				fmt.Fprintf(&b, " heap_reason_codes: %s", strings.Join(alloc.HeapReasonCodes, ","))
			}
			if len(alloc.ReasonCodes) > 0 {
				fmt.Fprintf(&b, " reason_codes: %s", strings.Join(alloc.ReasonCodes, ","))
			}
			fmt.Fprintln(&b)
		}
	}
	fmt.Fprintf(
		&b,
		"totals allocation_count:%d bytes_requested:%d bytes_reserved:%d bytes_committed:%d bytes_released:%d heap:%d stack:%d region:%d function_temp_region:%d explicit_island:%d eliminated:%d runtime_paths:%s allocator_classes:%s allocator_scopes:%s allocator_reuse_policies:%s memory_backend_classes:%s memory_backend_operations:%s memory_backend_evidence_classes:%s heap_reason_codes:%s raw_pointer_bounds:%s raw_slice_policies:%s domains:%s\n",
		summary.AllocationCount,
		summary.BytesRequested,
		summary.BytesReserved,
		summary.BytesCommitted,
		summary.BytesReleased,
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
		formatSummaryCounts(summary.MemoryBackendClasses),
		formatSummaryCounts(summary.MemoryBackendOperations),
		formatSummaryCounts(summary.MemoryBackendEvidenceClasses),
		formatSummaryCounts(summary.HeapReasonCodes),
		formatSummaryCounts(summary.RawPointerBoundsStatuses),
		formatSummaryCounts(summary.RawSlicePolicies),
		formatDomainSummary(summary.Domains),
	)
	return b.String()
}

func formatMemoryBackendOperations(ops []runtimeabi.MemoryBackendOperation) string {
	if len(ops) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(ops))
	seen := map[string]bool{}
	for _, op := range ops {
		value := string(op)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		parts = append(parts, value)
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func formatDomainSummary(domains []runtimeabi.MemoryDomainSummary) string {
	if len(domains) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(domains))
	for _, domain := range domains {
		parts = append(
			parts,
			fmt.Sprintf("%s=%d/%d", domain.DomainID, domain.RequestedBytes, domain.ReservedBytes),
		)
	}
	return strings.Join(parts, ",")
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
