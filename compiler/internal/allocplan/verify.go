package allocplan

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/runtimeabi"
)

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
				return fmt.Errorf(
					"allocplan verifier: %s allocation with empty allocation id",
					fn.Name,
				)
			}
			if alloc.SiteID == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q missing stable site id",
					fn.Name,
					alloc.ValueID,
				)
			}
			if alloc.Builtin == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q missing builtin",
					fn.Name,
					alloc.ValueID,
				)
			}
			key := fn.Name + "\x00" + alloc.ValueID
			if seen[key] {
				return fmt.Errorf(
					"allocplan verifier: %s duplicate allocation %q",
					fn.Name,
					alloc.ValueID,
				)
			}
			seen[key] = true
			if alloc.Storage == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q has empty storage",
					fn.Name,
					alloc.ValueID,
				)
			}
			if alloc.PlannedStorage == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q has empty planned storage",
					fn.Name,
					alloc.ValueID,
				)
			}
			if alloc.PlannedStorage != alloc.Storage {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q planned storage %s does not match storage %s",
					fn.Name,
					alloc.ValueID,
					alloc.PlannedStorage,
					alloc.Storage,
				)
			}
			if alloc.ActualLoweringStorage == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q has empty actual lowering storage",
					fn.Name,
					alloc.ValueID,
				)
			}
			if alloc.ValidationStatus == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q missing validation status",
					fn.Name,
					alloc.ValueID,
				)
			}
			if alloc.LoweringStatus == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q missing lowering status",
					fn.Name,
					alloc.ValueID,
				)
			}
			if alloc.LengthStatus == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q has empty length status",
					fn.Name,
					alloc.ValueID,
				)
			}
			if strings.TrimSpace(alloc.Reason) == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q missing storage reason",
					fn.Name,
					alloc.ValueID,
				)
			}
			for _, observed := range []struct {
				name    string
				storage StorageClass
			}{
				{name: "storage", storage: alloc.Storage},
				{name: "planned storage", storage: alloc.PlannedStorage},
				{name: "actual lowering storage", storage: alloc.ActualLoweringStorage},
			} {
				if alloc.Escape != EscapeNoEscape &&
					trustedStorageRequiresNoEscape(observed.storage, alloc.LengthStatus) &&
					!trustedBoundaryStorageForEscape(alloc.Escape, observed.storage) {
					return fmt.Errorf(
						"allocplan verifier: %s escaping allocation %q cannot use %s %s",
						fn.Name,
						alloc.ValueID,
						observed.storage,
						observed.name,
					)
				}
				if alloc.Escape == EscapeNoEscape &&
					trustedStorageRequiresNoEscape(observed.storage, alloc.LengthStatus) &&
					!storageHasCompilerOwnedNoEscapeProof(
						observed.storage,
						alloc.ValidationStatus,
						alloc.LengthStatus,
					) {
					return fmt.Errorf(
						"allocplan verifier: %s allocation %q uses %s %s without compiler-owned no-escape proof",
						fn.Name,
						alloc.ValueID,
						observed.storage,
						observed.name,
					)
				}
			}
			if alloc.Storage == StorageExplicitIsland && alloc.Escape != EscapeNoEscape {
				return fmt.Errorf(
					"allocplan verifier: %s island allocation %q cannot escape its island scope",
					fn.Name,
					alloc.ValueID,
				)
			}
			if trustedBoundaryStorageForEscape(alloc.Escape, alloc.ActualLoweringStorage) &&
				alloc.PlannedStorage != alloc.ActualLoweringStorage {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q actual lowering storage %s requires matching proof-carrying planned storage",
					fn.Name,
					alloc.ValueID,
					alloc.ActualLoweringStorage,
				)
			}
			if err := validateAllocationMemoryBackendEvidence(fn.Name, alloc); err != nil {
				return err
			}
			if err := validateAllocationReasonCodes(fn.Name, alloc); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateAllocationMemoryBackendEvidence(function string, alloc Allocation) error {
	if alloc.MemoryBackend == nil {
		return nil
	}
	if err := runtimeabi.ValidateMemoryBackendAllocationEvidence(*alloc.MemoryBackend); err != nil {
		return fmt.Errorf(
			"allocplan verifier: %s allocation %q invalid memory backend evidence: %w",
			function,
			alloc.ValueID,
			err,
		)
	}
	runtimePath := RuntimePathForAllocation(alloc)
	if alloc.MemoryBackend.RuntimePath != runtimePath {
		return fmt.Errorf(
			("allocplan verifier: %s allocation %q memory backend runtime_" +
				"path %q does not match allocation runtime_path %q"),
			function,
			alloc.ValueID,
			alloc.MemoryBackend.RuntimePath,
			runtimePath,
		)
	}
	switch alloc.MemoryBackend.EvidenceClass {
	case runtimeabi.MemoryFootprintEstimated, runtimeabi.MemoryFootprintMeasured:
		if int64(alloc.BytesCommitted) != alloc.MemoryBackend.CommitBytes {
			return fmt.Errorf(
				("allocplan verifier: %s allocation %q bytes_committed %d does " +
					"not match memory_backend commit_bytes %d"),
				function,
				alloc.ValueID,
				alloc.BytesCommitted,
				alloc.MemoryBackend.CommitBytes,
			)
		}
		if int64(alloc.BytesReleased) != alloc.MemoryBackend.ReleaseBytes {
			return fmt.Errorf(
				("allocplan verifier: %s allocation %q bytes_released %d does not " +
					"match memory_backend release_bytes %d"),
				function,
				alloc.ValueID,
				alloc.BytesReleased,
				alloc.MemoryBackend.ReleaseBytes,
			)
		}
	default:
		if alloc.BytesCommitted != 0 || alloc.BytesReleased != 0 {
			return fmt.Errorf(
				("allocplan verifier: %s allocation %q unsupported/blocked memory " +
					"backend evidence must not report committed or released bytes"),
				function,
				alloc.ValueID,
			)
		}
	}
	return nil
}

func validateAllocationReasonCodes(function string, alloc Allocation) error {
	if allocationUsesHeap(alloc) {
		if len(alloc.HeapReasonCodes) == 0 {
			return fmt.Errorf(
				"allocplan verifier: %s allocation %q missing heap reason code",
				function,
				alloc.ValueID,
			)
		}
	} else if len(alloc.HeapReasonCodes) > 0 {
		return fmt.Errorf(
			"allocplan verifier: %s allocation %q has heap reason codes without heap storage",
			function,
			alloc.ValueID,
		)
	}
	for _, group := range []struct {
		label string
		codes []string
		heap  bool
	}{
		{label: "reason_codes", codes: alloc.ReasonCodes},
		{label: "heap_reason_codes", codes: alloc.HeapReasonCodes, heap: true},
	} {
		seen := map[string]bool{}
		for _, code := range group.codes {
			if strings.TrimSpace(code) == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q has empty %s entry",
					function,
					alloc.ValueID,
					group.label,
				)
			}
			if strings.TrimSpace(code) != code {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q has untrimmed %s entry %q",
					function,
					alloc.ValueID,
					group.label,
					code,
				)
			}
			if seen[code] {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q has duplicate %s entry %q",
					function,
					alloc.ValueID,
					group.label,
					code,
				)
			}
			seen[code] = true
			if group.heap && !isKnownHeapReasonCode(code) {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q has unknown heap reason code %q",
					function,
					alloc.ValueID,
					code,
				)
			}
		}
	}
	for _, code := range alloc.HeapReasonCodes {
		if !contains(alloc.ReasonCodes, code) {
			return fmt.Errorf(
				"allocplan verifier: %s allocation %q heap reason code %q missing from reason_codes",
				function,
				alloc.ValueID,
				code,
			)
		}
	}
	return nil
}

func isKnownHeapReasonCode(code string) bool {
	switch code {
	case HeapReasonEscapeReturn,
		HeapReasonUnknownCall,
		HeapReasonActorBoundary,
		HeapReasonTaskBoundary,
		HeapReasonActorMoveUnproven,
		HeapReasonTaskMoveUnproven,
		HeapReasonRequestOwnerUnproven,
		HeapReasonDynamicLifetime,
		HeapReasonLargeObject,
		HeapReasonFFIExternal,
		HeapReasonBackendLoweringUnavailable,
		HeapReasonRegionLoweringUnavailable:
		return true
	default:
		return false
	}
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

func trustedBoundaryStorageForEscape(escape EscapeClass, storage StorageClass) bool {
	switch storage {
	case StorageTaskRegion:
		return escape == EscapeTask
	case StorageActorMoveRegion:
		return escape == EscapeActor
	default:
		return false
	}
}

func storageHasCompilerOwnedNoEscapeProof(
	storage StorageClass,
	status string,
	lengthStatus LengthStatus,
) bool {
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
