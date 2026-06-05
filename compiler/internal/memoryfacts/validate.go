package memoryfacts

import "strings"

func isUnsafeUnknown(f Fact) bool {
	return f.ProvenanceClass == ProvenanceUnsafeUnknown || f.UnsafeClass == UnsafeUnknown
}

func hasUnsafeUnknownClass(provenance ProvenanceClass, unsafe UnsafeClass) bool {
	return provenance == ProvenanceUnsafeUnknown || unsafe == UnsafeUnknown
}

func isSafeProvenance(p ProvenanceClass) bool {
	return p == ProvenanceSafeKnown || p == ProvenanceSafeBorrowed || p == ProvenanceSafeOwned
}

func hasSafeProvenanceFromUnsafeUnknown(provenance ProvenanceClass, unsafe UnsafeClass) bool {
	return isSafeProvenance(provenance) && unsafe == UnsafeUnknown
}

func requiresLoweredArtifact(f Fact) bool {
	if f.StoragePlan != "" || f.ActualLoweringStorage != "" {
		return true
	}
	claim := strings.ToLower(f.Claim)
	return strings.Contains(claim, "lowering") || strings.Contains(claim, "storage")
}

func unsafeUnknownOptimizationClaim(claim string, alias AliasState) bool {
	claim = strings.ToLower(strings.TrimSpace(claim))
	switch claim {
	case "provenance_known", "no_alias", "index_in_range", "bounds_check_eliminated", "trusted_storage":
		return true
	case "safe_known", "safe_borrowed", "safe_owned":
		return true
	}
	if alias == AliasUnique || alias == AliasMutableExclusive {
		return true
	}
	return false
}

func memoryOptimizationClaim(claim string, alias AliasState) bool {
	if unsafeUnknownOptimizationClaim(claim, alias) {
		return true
	}
	claim = strings.ToLower(strings.TrimSpace(claim))
	return strings.Contains(claim, "eliminated") || strings.Contains(claim, "zero_cost")
}

func knownCostClass(class CostClass) bool {
	switch class {
	case CostZeroCostProven, CostDynamicCheckRequired, CostInstrumentationOnly,
		CostUnsupportedRejected, CostConservativeFallback:
		return true
	default:
		return false
	}
}

func inferCostClass(f Fact) CostClass {
	claim := strings.ToLower(strings.TrimSpace(f.Claim))
	if strings.HasPrefix(claim, "rejected_") || f.ClaimLevelRejected() {
		return CostUnsupportedRejected
	}
	if isUnsafeUnknown(f) || f.ProvenanceClass == ProvenanceUnsafeUnknown || f.UnsafeClass == UnsafeUnknown {
		return CostConservativeFallback
	}
	switch claim {
	case "derived_allocation_offset", "raw_memory_access_checked", "raw_slice_verified_allocation_root", "unsafe_contract_runtime_checkable", "bounds_check_retained_dynamic", "raw_bounds_runtime_check_normal_build":
		return CostDynamicCheckRequired
	case "allocation_base_metadata", "unsafe_verified_root_allocation_base", "provenance_known", "region_alive", "len_stable", "index_in_range", "bounds_check_eliminated", "bounds_check_removed_with_proof_id", "non_null", "maybe_null", "aligned", "owned", "borrowed_imm", "borrowed_mut", "moved", "borrow_owner", "borrow_source_fact_id", "aggregate_contains_borrow", "optional_contains_borrow", "enum_payload_contains_borrow", "generic_wrapper_contains_borrow", "function_value_contains_borrow", "callback_arg_contains_borrow", "interface_value_contains_borrow", "copy_owned", "copy_source_fact_id", "copy_into_destination_fact_id", "no_alias", "mutable_exclusive", "start_inout_exclusive", "end_inout_exclusive", "no_alias_validated_narrow_unique_local", "no_alias_validated_narrow_sequential_inout":
		return CostZeroCostProven
	case "callback_inout_conservative", "protocol_dispatch_borrow_conservative", "protocol_dispatch_noalias_conservative", "async_boundary_borrow_conservative", "boundary_noalias_conservative", "unsafe_contract_static_untrusted":
		return CostConservativeFallback
	case "task_boundary_borrow_rejected", "actor_boundary_borrow_rejected", "unsafe_unknown_rejected_safe_facts", "bounds_check_removal_rejected_missing_proof_id":
		return CostUnsupportedRejected
	case "storage_lowering":
		if f.ActualLoweringStorage == StorageHeap && f.StoragePlan != "" && f.StoragePlan != StorageHeap {
			return CostConservativeFallback
		}
		return CostZeroCostProven
	}
	if strings.Contains(claim, "unknown") || f.EscapeState == EscapeConservative || f.AliasState == AliasUnknownConservative || f.StoragePlan == StorageUnknownConservative {
		return CostConservativeFallback
	}
	return CostInstrumentationOnly
}

func (f Fact) ClaimLevelRejected() bool {
	return f.ValidationState == ValidationFail || f.ValidationState == ValidationInvalidated
}

func unsafeVerifiedRootDisallowedClaim(provenance ProvenanceClass, unsafe UnsafeClass, claim string) bool {
	if provenance != ProvenanceUnsafeVerifiedRoot && unsafe != UnsafeVerifiedRoot {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(claim)) {
	case "allocation_base_metadata", "unsafe_verified_root_allocation_base":
		return false
	default:
		return true
	}
}

func unsafeUnknownTrustedStorage(planned, actual StorageClass) bool {
	return trustedStorageForUnsafeUnknown(planned) || trustedStorageForUnsafeUnknown(actual)
}

func validatedTrustedStorageHeapFallback(planned, actual StorageClass) bool {
	return trustedStorageHeapFallback(planned, actual)
}

func trustedStorageHeapFallback(planned, actual StorageClass) bool {
	if actual != StorageHeap {
		return false
	}
	switch planned {
	case StorageEliminated, StorageRegister, StorageStack, StorageRegion,
		StorageExplicitIsland, StorageFunctionTempRegion, StorageTaskRegion,
		StorageActorMoveRegion:
		return true
	default:
		return false
	}
}

func trustedStorageForUnsafeUnknown(class StorageClass) bool {
	switch class {
	case StorageEliminated, StorageRegister, StorageStack, StorageRegion,
		StorageExplicitIsland, StorageFunctionTempRegion, StorageTaskRegion,
		StorageActorMoveRegion:
		return true
	default:
		return false
	}
}

func knownSourceStage(stage SourceStage) bool {
	switch stage {
	case StageSemantics, StageUnsafeGatewayLowering, StagePLIR, StageAllocPlan, StageLowering, StageValidation:
		return true
	default:
		return false
	}
}

func knownProvenanceClass(class ProvenanceClass) bool {
	switch class {
	case ProvenanceSafeKnown, ProvenanceSafeBorrowed, ProvenanceSafeOwned,
		ProvenanceUnsafeUnknown, ProvenanceUnsafeChecked, ProvenanceUnsafeVerifiedRoot:
		return true
	default:
		return false
	}
}

func knownUnsafeClass(class UnsafeClass) bool {
	switch class {
	case UnsafeSafe, UnsafeUnknown, UnsafeChecked, UnsafeVerifiedRoot:
		return true
	default:
		return false
	}
}

func knownStorageClass(class StorageClass) bool {
	switch class {
	case StorageUnknownConservative, StorageEliminated, StorageRegister, StorageHeap,
		StorageStack, StorageRegion, StorageExplicitIsland, StorageFunctionTempRegion,
		StorageTaskRegion, StorageActorMoveRegion, StorageLargeMmap, StorageExternal:
		return true
	default:
		return false
	}
}

func knownAliasState(state AliasState) bool {
	switch state {
	case AliasUnknown, AliasUnique, AliasSharedReadonly, AliasMutableExclusive,
		AliasMaybe, AliasUnknownConservative, AliasInvalidatedByCall:
		return true
	default:
		return false
	}
}

func validatedNoAliasState(state AliasState) bool {
	return state == AliasUnique || state == AliasMutableExclusive
}

func broadNoAliasClaim(claim string) bool {
	claim = strings.ToLower(strings.TrimSpace(claim))
	return claim == "broad_noalias" || claim == "universal_noalias" || claim == "full_noalias_model"
}

func claimRequiresParentFactID(claim string) bool {
	switch strings.ToLower(strings.TrimSpace(claim)) {
	case "borrow_owner", "borrow_source_fact_id", "aggregate_contains_borrow",
		"optional_contains_borrow", "enum_payload_contains_borrow",
		"generic_wrapper_contains_borrow", "function_value_contains_borrow",
		"callback_arg_contains_borrow", "callback_inout_conservative",
		"interface_value_contains_borrow", "protocol_dispatch_borrow_conservative",
		"protocol_dispatch_noalias_conservative",
		"async_boundary_borrow_conservative", "task_boundary_borrow_rejected",
		"actor_boundary_borrow_rejected", "boundary_noalias_conservative",
		"unsafe_unknown_rejected_safe_facts",
		"unsafe_verified_root_allocation_base",
		"bounds_check_removed_with_proof_id",
		"raw_bounds_runtime_check_normal_build",
		"ffi_call_may_retain_borrow",
		"ffi_noalias_invalidated_by_external_call",
		"safe_wrapper_promotion_rejected_without_contract",
		"external_pointer_provenance_rejected",
		"copy_owned", "copy_source_fact_id",
		"mutable_exclusive", "start_inout_exclusive", "end_inout_exclusive",
		"no_alias_validated_narrow_unique_local",
		"no_alias_validated_narrow_sequential_inout":
		return true
	default:
		return false
	}
}

func knownClaimLevel(level ClaimLevel) bool {
	switch level {
	case ClaimValidated, ClaimEvidenceOnly, ClaimConservative, ClaimRejected, ClaimFuture:
		return true
	default:
		return false
	}
}

func knownValidatorStatus(status ValidatorStatus) bool {
	switch status {
	case ValidatorPass, ValidatorFail, ValidatorNotApplicable, ValidatorNotRun:
		return true
	default:
		return false
	}
}
