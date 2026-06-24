package memoryfacts

import "strings"

type Claim = string
type ProofKind = string

type ValueKey struct {
	FunctionID string
	ValueID    string
}

type AllocationKey struct {
	FunctionID       string
	AllocationSiteID string
}

type ProofKey struct {
	FunctionID string
	ProofID    string
}

const (
	ProofBounds      ProofKind = "bounds"
	ProofNoAlias     ProofKind = "noalias"
	ProofStorage     ProofKind = "storage"
	ProofNoEscape    ProofKind = "no_escape"
	ProofRegionAlive ProofKind = "region_alive"
	ProofBorrow      ProofKind = "borrow"
	ProofDomainMove  ProofKind = "domain_move"
)

const (
	ClaimSafeRepresentationMetadataNotUserAssignable = ("safe_representation_metadata: not_" +
		"user_assignable")
	ClaimLenStable           = "len_stable"
	ClaimIndexInRange        = "index_in_range"
	ClaimRegionAlive         = "region_alive"
	ClaimNoEscape            = "no_escape"
	ClaimNoAlias             = "no_alias"
	ClaimNonNull             = "non_null"
	ClaimMaybeNull           = "maybe_null"
	ClaimAligned             = "aligned"
	ClaimProvenanceKnown     = "provenance_known"
	ClaimProvenanceUnknown   = "provenance_unknown"
	ClaimOwned               = "owned"
	ClaimBorrowedImm         = "borrowed_imm"
	ClaimBorrowedMut         = "borrowed_mut"
	ClaimMoved               = "moved"
	ClaimPureCall            = "pure_call"
	ClaimNoHeapAllocation    = "no_heap_allocation"
	ClaimNoMemWrite          = "no_mem_write"
	ClaimNoActorSend         = "no_actor_send"
	ClaimNoUnknownEscape     = "no_unknown_escape"
	ClaimDerivedWindow       = "derived_window"
	ClaimIslandEpochAdvanced = "island_epoch_advanced"

	ClaimMayReturnRegion                          = "may_return_region"
	ClaimMayConsumeParam                          = "may_consume_param"
	ClaimMayMutateInout                           = "may_mutate_inout"
	ClaimTrustedStorage                           = "trusted_storage"
	ClaimStorageLowering                          = "storage_lowering"
	ClaimBoundsProofID                            = "bounds_proof_id"
	ClaimBoundsCheckEliminated                    = "bounds_check_eliminated"
	ClaimBoundsCheckRemovedWithProofID            = "bounds_check_removed_with_proof_id"
	ClaimBoundsCheckRetainedDynamic               = "bounds_check_retained_dynamic"
	ClaimBoundsCheckRemovalRejectedMissingProofID = "bounds_check_removal_rejected_missing_proof_id"
	ClaimNormalBuildBoundsCheckGuard              = "normal_build_bounds_check_guard"

	ClaimAllocationBaseMetadata           = "allocation_base_metadata"
	ClaimUnsafeVerifiedRootAllocationBase = "unsafe_verified_root_allocation_base"
	ClaimUnsafeContractStaticUntrusted    = "unsafe_contract_static_untrusted"
	ClaimUnsafeContractRuntimeCheckable   = "unsafe_contract_runtime_checkable"
	ClaimRawMemoryAccessChecked           = "raw_memory_access_checked"
	ClaimRawMemoryAccessUnknown           = "raw_memory_access_unknown"
	ClaimRawSliceVerifiedAllocationRoot   = "raw_slice_verified_allocation_root"
	ClaimRawBoundsRuntimeCheckNormalBuild = "raw_bounds_runtime_check_normal_build"
	ClaimDerivedAllocationOffset          = "derived_allocation_offset"
	ClaimRejectedNegativeOffset           = "rejected_negative_offset"
	ClaimRejectedUpperBound               = "rejected_upper_bound"
	ClaimRejectedAccessWidthOverflow      = "rejected_access_width_overflow"
	ClaimRejectedNegativeLength           = "rejected_negative_length"
	ClaimRejectedLengthOverflow           = "rejected_length_overflow"
	ClaimCheckedExternalUnknown           = "checked_external_unknown"
	ClaimExternalUnknown                  = "external_unknown"
	ClaimFFIPointerExternalUnknown        = "ffi_pointer_external_unknown"
	ClaimReturnsUnknownUnsafe             = "returns_unknown_unsafe"
	ClaimReturnsOwnedNewAllocation        = "returns_owned_new_allocation"
	ClaimReturnsBorrowFromParam           = "returns_borrow_from_param"
	ClaimMayReturnResource                = "may_return_resource"
	ClaimMayThrowResource                 = "may_throw_resource"
	ClaimRequiresEffects                  = "requires_effects"
	ClaimRequiresCapabilities             = "requires_capabilities"
	ClaimCapMemAuthorizationOnly          = "cap_mem_authorization_only"
	ClaimMayStoreGlobal                   = "may_store_global"
	ClaimMayEscapeToActor                 = "may_escape_to_actor"
	ClaimMayCaptureInClosure              = "may_capture_in_closure"
	ClaimMayEscapeToTask                  = "may_escape_to_task"
	ClaimUnknownExternalCallConservative  = "unknown_external_call_conservative"
	ClaimMayRetainPointer                 = "may_retain_pointer"
	ClaimFunctionContract                 = "function_contract"

	ClaimBorrowOwner                           = "borrow_owner"
	ClaimBorrowSourceFactID                    = "borrow_source_fact_id"
	ClaimAggregateContainsBorrow               = "aggregate_contains_borrow"
	ClaimOptionalContainsBorrow                = "optional_contains_borrow"
	ClaimEnumPayloadContainsBorrow             = "enum_payload_contains_borrow"
	ClaimGenericWrapperContainsBorrow          = "generic_wrapper_contains_borrow"
	ClaimFunctionValueContainsBorrow           = "function_value_contains_borrow"
	ClaimCallbackArgContainsBorrow             = "callback_arg_contains_borrow"
	ClaimCallbackInoutConservative             = "callback_inout_conservative"
	ClaimInterfaceValueContainsBorrow          = "interface_value_contains_borrow"
	ClaimProtocolDispatchBorrowConservative    = "protocol_dispatch_borrow_conservative"
	ClaimProtocolDispatchNoaliasConservative   = "protocol_dispatch_noalias_conservative"
	ClaimDynamicExistentialBorrowConservative  = "dynamic_existential_borrow_conservative"
	ClaimStaticWitnessBorrowParentValidated    = "static_witness_borrow_parent_validated"
	ClaimDynamicProtocolNoaliasRejected        = "dynamic_protocol_noalias_rejected"
	ClaimWitnessProvenancePromotionRejected    = "witness_provenance_promotion_rejected"
	ClaimProtocolDispatchReportIntegrity       = "protocol_dispatch_report_integrity"
	ClaimAsyncBoundaryBorrowConservative       = "async_boundary_borrow_conservative"
	ClaimTaskBoundaryBorrowRejected            = "task_boundary_borrow_rejected"
	ClaimActorBoundaryBorrowRejected           = "actor_boundary_borrow_rejected"
	ClaimBoundaryNoaliasConservative           = "boundary_noalias_conservative"
	ClaimPreAwaitLocalBorrowValidated          = "pre_await_local_borrow_validated"
	ClaimPostAwaitBorrowConservative           = "post_await_borrow_conservative"
	ClaimCancellationBorrowLifetimeInvalidated = "cancellation_borrow_lifetime_invalidated"
	ClaimTaskGroupNoaliasConservative          = "task_group_noalias_conservative"
	ClaimActorReentrantCallbackConservative    = "actor_reentrant_callback_conservative"

	ClaimFFICallMayRetainBorrow                      = "ffi_call_may_retain_borrow"
	ClaimFFINoaliasInvalidatedByExternalCall         = "ffi_noalias_invalidated_by_external_call"
	ClaimSafeWrapperPromotionRejectedWithoutContract = ("safe_wrapper_promotion_rejected_" +
		"without_contract")
	ClaimExternalPointerProvenanceRejected = "external_pointer_provenance_rejected"

	ClaimCopyOwned                      = "copy_owned"
	ClaimCopySourceFactID               = "copy_source_fact_id"
	ClaimCopyIntoOperation              = "copy_into_operation"
	ClaimCopyIntoDestinationLengthCheck = "copy_into_destination_length_check"
	ClaimCopyIntoDestinationFactID      = "copy_into_destination_fact_id"
	ClaimCopyIntoOverlapRejected        = "copy_into_overlap_rejected"
	ClaimCopyIntoOverlapConservative    = "copy_into_overlap_conservative"

	ClaimMutableExclusive                      = "mutable_exclusive"
	ClaimStartInoutExclusive                   = "start_inout_exclusive"
	ClaimEndInoutExclusive                     = "end_inout_exclusive"
	ClaimNoAliasValidatedNarrowUniqueLocal     = "no_alias_validated_narrow_unique_local"
	ClaimNoAliasValidatedNarrowSequentialInout = "no_alias_validated_narrow_sequential_inout"

	ClaimUnsafeUnknownRejectedSafeFacts = "unsafe_unknown_rejected_safe_facts"
	ClaimBroadNoAlias                   = "broad_noalias"
	ClaimUniversalNoAlias               = "universal_noalias"
	ClaimFullNoAliasModel               = "full_noalias_model"

	ClaimIslandKernelModelOnly        = "island_kernel_model_only"
	ClaimIslandEpochValidated         = "island_epoch_validated"
	ClaimIslandSanitizeRuntimeChecked = "island_sanitize_runtime_checked"
	ClaimIslandProofVerified          = "island_proof_verified"

	ClaimOptimizerPass     = "optimizer_pass"
	ClaimOptimizerDecision = "optimizer_decision"
)

const (
	FuzzStatusCovered          = "covered"
	FuzzStatusValidatedNarrow  = "validated_narrow"
	FuzzStatusBoundaryRecorded = "boundary_recorded"
	FuzzStatusBlocksRelease    = "blocks_release"
	FuzzStatusReleaseBlocking  = "release_blocking"
	FuzzStatusFuture           = "future"
)

var sourceStages = []string{
	string(StageSemantics),
	string(StageUnsafeGatewayLowering),
	string(StagePLIR),
	string(StageAllocPlan),
	string(StageLowering),
	string(StageOptimization),
	string(StageValidation),
}

var provenanceClasses = []string{
	string(ProvenanceSafeKnown),
	string(ProvenanceSafeBorrowed),
	string(ProvenanceSafeOwned),
	string(ProvenanceUnsafeUnknown),
	string(ProvenanceUnsafeChecked),
	string(ProvenanceUnsafeVerifiedRoot),
}

var unsafeClasses = []string{
	string(UnsafeSafe),
	string(UnsafeUnknown),
	string(UnsafeChecked),
	string(UnsafeVerifiedRoot),
}

var aliasStates = []string{
	string(AliasUnknown),
	string(AliasUnique),
	string(AliasSharedReadonly),
	string(AliasMutableExclusive),
	string(AliasMaybe),
	string(AliasUnknownConservative),
	string(AliasInvalidatedByCall),
}

var storageClasses = []string{
	string(StorageUnknownConservative),
	string(StorageEliminated),
	string(StorageRegister),
	string(StorageHeap),
	string(StorageStack),
	string(StorageRegion),
	string(StorageExplicitIsland),
	string(StorageFunctionTempRegion),
	string(StorageTaskRegion),
	string(StorageActorMoveRegion),
	string(StorageLargeMmap),
	string(StorageExternal),
}

var domainKinds = []string{
	string(DomainProcess),
	string(DomainTask),
	string(DomainActor),
	string(DomainIsland),
	string(DomainRequest),
	string(DomainExternal),
}

var transferKinds = []string{
	string(TransferMove),
	string(TransferCopy),
	string(TransferBorrowed),
	string(TransferUnsafe),
}

var claimLevels = []string{
	string(ClaimValidated),
	string(ClaimEvidenceOnly),
	string(ClaimConservative),
	string(ClaimRejected),
	string(ClaimFuture),
}

var validatorStatuses = []string{
	string(ValidatorPass),
	string(ValidatorFail),
	string(ValidatorNotApplicable),
	string(ValidatorNotRun),
}

var costClasses = []string{
	string(CostZeroCostProven),
	string(CostDynamicCheckRequired),
	string(CostInstrumentationOnly),
	string(CostUnsupportedRejected),
	string(CostConservativeFallback),
}

var reportClaims = []string{
	ClaimSafeRepresentationMetadataNotUserAssignable,
	ClaimLenStable,
	ClaimIndexInRange,
	ClaimRegionAlive,
	ClaimNoEscape,
	ClaimNoAlias,
	ClaimNonNull,
	ClaimMaybeNull,
	ClaimAligned,
	ClaimProvenanceKnown,
	ClaimProvenanceUnknown,
	ClaimOwned,
	ClaimBorrowedImm,
	ClaimBorrowedMut,
	ClaimMoved,
	ClaimPureCall,
	ClaimNoHeapAllocation,
	ClaimNoMemWrite,
	ClaimNoActorSend,
	ClaimNoUnknownEscape,
	ClaimDerivedWindow,
	ClaimIslandEpochAdvanced,
	ClaimMayReturnRegion,
	ClaimMayConsumeParam,
	ClaimMayMutateInout,
	ClaimTrustedStorage,
	ClaimStorageLowering,
	ClaimBoundsProofID,
	ClaimBoundsCheckEliminated,
	ClaimBoundsCheckRemovedWithProofID,
	ClaimBoundsCheckRetainedDynamic,
	ClaimBoundsCheckRemovalRejectedMissingProofID,
	ClaimNormalBuildBoundsCheckGuard,
	ClaimAllocationBaseMetadata,
	ClaimUnsafeVerifiedRootAllocationBase,
	ClaimUnsafeContractStaticUntrusted,
	ClaimUnsafeContractRuntimeCheckable,
	ClaimRawMemoryAccessChecked,
	ClaimRawMemoryAccessUnknown,
	ClaimRawSliceVerifiedAllocationRoot,
	ClaimRawBoundsRuntimeCheckNormalBuild,
	ClaimDerivedAllocationOffset,
	ClaimRejectedNegativeOffset,
	ClaimRejectedUpperBound,
	ClaimRejectedAccessWidthOverflow,
	ClaimRejectedNegativeLength,
	ClaimRejectedLengthOverflow,
	ClaimCheckedExternalUnknown,
	ClaimExternalUnknown,
	ClaimFFIPointerExternalUnknown,
	ClaimReturnsUnknownUnsafe,
	ClaimReturnsOwnedNewAllocation,
	ClaimReturnsBorrowFromParam,
	ClaimMayReturnResource,
	ClaimMayThrowResource,
	ClaimRequiresEffects,
	ClaimRequiresCapabilities,
	ClaimCapMemAuthorizationOnly,
	ClaimMayStoreGlobal,
	ClaimMayEscapeToActor,
	ClaimMayCaptureInClosure,
	ClaimMayEscapeToTask,
	ClaimUnknownExternalCallConservative,
	ClaimMayRetainPointer,
	ClaimFunctionContract,
	ClaimBorrowOwner,
	ClaimBorrowSourceFactID,
	ClaimAggregateContainsBorrow,
	ClaimOptionalContainsBorrow,
	ClaimEnumPayloadContainsBorrow,
	ClaimGenericWrapperContainsBorrow,
	ClaimFunctionValueContainsBorrow,
	ClaimCallbackArgContainsBorrow,
	ClaimCallbackInoutConservative,
	ClaimInterfaceValueContainsBorrow,
	ClaimProtocolDispatchBorrowConservative,
	ClaimProtocolDispatchNoaliasConservative,
	ClaimDynamicExistentialBorrowConservative,
	ClaimStaticWitnessBorrowParentValidated,
	ClaimDynamicProtocolNoaliasRejected,
	ClaimWitnessProvenancePromotionRejected,
	ClaimProtocolDispatchReportIntegrity,
	ClaimAsyncBoundaryBorrowConservative,
	ClaimTaskBoundaryBorrowRejected,
	ClaimActorBoundaryBorrowRejected,
	ClaimBoundaryNoaliasConservative,
	ClaimPreAwaitLocalBorrowValidated,
	ClaimPostAwaitBorrowConservative,
	ClaimCancellationBorrowLifetimeInvalidated,
	ClaimTaskGroupNoaliasConservative,
	ClaimActorReentrantCallbackConservative,
	ClaimFFICallMayRetainBorrow,
	ClaimFFINoaliasInvalidatedByExternalCall,
	ClaimSafeWrapperPromotionRejectedWithoutContract,
	ClaimExternalPointerProvenanceRejected,
	ClaimCopyOwned,
	ClaimCopySourceFactID,
	ClaimCopyIntoOperation,
	ClaimCopyIntoDestinationLengthCheck,
	ClaimCopyIntoDestinationFactID,
	ClaimCopyIntoOverlapRejected,
	ClaimCopyIntoOverlapConservative,
	ClaimMutableExclusive,
	ClaimStartInoutExclusive,
	ClaimEndInoutExclusive,
	ClaimNoAliasValidatedNarrowUniqueLocal,
	ClaimNoAliasValidatedNarrowSequentialInout,
	ClaimUnsafeUnknownRejectedSafeFacts,
	ClaimBroadNoAlias,
	ClaimUniversalNoAlias,
	ClaimFullNoAliasModel,
	ClaimIslandKernelModelOnly,
	ClaimIslandEpochValidated,
	ClaimIslandSanitizeRuntimeChecked,
	ClaimIslandProofVerified,
	ClaimOptimizerPass,
	ClaimOptimizerDecision,
}

var islandKernelEvidenceClaims = []string{
	ClaimIslandKernelModelOnly,
	ClaimIslandEpochValidated,
	ClaimIslandSanitizeRuntimeChecked,
	ClaimIslandProofVerified,
}

var requiredIslandKernelClaimValidators = map[string]string{
	ClaimIslandEpochValidated:         "island_epoch_validator",
	ClaimIslandSanitizeRuntimeChecked: "island_sanitize_runtime",
	ClaimIslandProofVerified:          "validate-island-proof",
}

var parentRequiredClaims = []string{
	ClaimBorrowOwner,
	ClaimBorrowSourceFactID,
	ClaimAggregateContainsBorrow,
	ClaimOptionalContainsBorrow,
	ClaimEnumPayloadContainsBorrow,
	ClaimGenericWrapperContainsBorrow,
	ClaimFunctionValueContainsBorrow,
	ClaimCallbackArgContainsBorrow,
	ClaimCallbackInoutConservative,
	ClaimInterfaceValueContainsBorrow,
	ClaimProtocolDispatchBorrowConservative,
	ClaimProtocolDispatchNoaliasConservative,
	ClaimDynamicExistentialBorrowConservative,
	ClaimStaticWitnessBorrowParentValidated,
	ClaimDynamicProtocolNoaliasRejected,
	ClaimWitnessProvenancePromotionRejected,
	ClaimProtocolDispatchReportIntegrity,
	ClaimAsyncBoundaryBorrowConservative,
	ClaimTaskBoundaryBorrowRejected,
	ClaimActorBoundaryBorrowRejected,
	ClaimBoundaryNoaliasConservative,
	ClaimPreAwaitLocalBorrowValidated,
	ClaimPostAwaitBorrowConservative,
	ClaimCancellationBorrowLifetimeInvalidated,
	ClaimTaskGroupNoaliasConservative,
	ClaimActorReentrantCallbackConservative,
	ClaimUnsafeUnknownRejectedSafeFacts,
	ClaimUnsafeVerifiedRootAllocationBase,
	ClaimBoundsCheckRetainedDynamic,
	ClaimBoundsCheckRemovedWithProofID,
	ClaimRawBoundsRuntimeCheckNormalBuild,
	ClaimFFICallMayRetainBorrow,
	ClaimFFINoaliasInvalidatedByExternalCall,
	ClaimSafeWrapperPromotionRejectedWithoutContract,
	ClaimExternalPointerProvenanceRejected,
	ClaimCopyOwned,
	ClaimCopySourceFactID,
	ClaimCopyIntoDestinationFactID,
	ClaimCopyIntoDestinationLengthCheck,
	ClaimCopyIntoOverlapRejected,
	ClaimCopyIntoOverlapConservative,
	ClaimMutableExclusive,
	ClaimStartInoutExclusive,
	ClaimEndInoutExclusive,
	ClaimNoAliasValidatedNarrowUniqueLocal,
	ClaimNoAliasValidatedNarrowSequentialInout,
}

var dynamicCheckRequiredClaims = []string{
	ClaimDerivedAllocationOffset,
	ClaimRawMemoryAccessChecked,
	ClaimRawSliceVerifiedAllocationRoot,
	ClaimUnsafeContractRuntimeCheckable,
	ClaimBoundsCheckRetainedDynamic,
	ClaimRawBoundsRuntimeCheckNormalBuild,
	ClaimProtocolDispatchReportIntegrity,
	ClaimCopyIntoDestinationLengthCheck,
}

var dynamicRawRuntimeCheckClaims = []string{
	ClaimDerivedAllocationOffset,
	ClaimRawMemoryAccessChecked,
	ClaimRawSliceVerifiedAllocationRoot,
	ClaimRawBoundsRuntimeCheckNormalBuild,
}

var zeroCostProvenClaims = []string{
	ClaimAllocationBaseMetadata,
	ClaimUnsafeVerifiedRootAllocationBase,
	ClaimProvenanceKnown,
	ClaimRegionAlive,
	ClaimLenStable,
	ClaimIndexInRange,
	ClaimBoundsCheckEliminated,
	ClaimBoundsCheckRemovedWithProofID,
	ClaimNonNull,
	ClaimMaybeNull,
	ClaimAligned,
	ClaimOwned,
	ClaimBorrowedImm,
	ClaimBorrowedMut,
	ClaimMoved,
	ClaimBorrowOwner,
	ClaimBorrowSourceFactID,
	ClaimAggregateContainsBorrow,
	ClaimOptionalContainsBorrow,
	ClaimEnumPayloadContainsBorrow,
	ClaimGenericWrapperContainsBorrow,
	ClaimFunctionValueContainsBorrow,
	ClaimCallbackArgContainsBorrow,
	ClaimInterfaceValueContainsBorrow,
	ClaimStaticWitnessBorrowParentValidated,
	ClaimPreAwaitLocalBorrowValidated,
	ClaimCopyOwned,
	ClaimCopySourceFactID,
	ClaimCopyIntoDestinationFactID,
	ClaimNoAlias,
	ClaimMutableExclusive,
	ClaimStartInoutExclusive,
	ClaimEndInoutExclusive,
	ClaimNoAliasValidatedNarrowUniqueLocal,
	ClaimNoAliasValidatedNarrowSequentialInout,
}

var conservativeFallbackClaims = []string{
	ClaimCallbackInoutConservative,
	ClaimProtocolDispatchBorrowConservative,
	ClaimProtocolDispatchNoaliasConservative,
	ClaimDynamicExistentialBorrowConservative,
	ClaimAsyncBoundaryBorrowConservative,
	ClaimBoundaryNoaliasConservative,
	ClaimUnsafeContractStaticUntrusted,
	ClaimCopyIntoOverlapConservative,
}

var unsupportedRejectedClaims = []string{
	ClaimTaskBoundaryBorrowRejected,
	ClaimActorBoundaryBorrowRejected,
	ClaimDynamicProtocolNoaliasRejected,
	ClaimWitnessProvenancePromotionRejected,
	ClaimUnsafeUnknownRejectedSafeFacts,
	ClaimBoundsCheckRemovalRejectedMissingProofID,
	ClaimCopyIntoOverlapRejected,
}

var unsafeCheckedAllowedClaims = []string{
	ClaimCapMemAuthorizationOnly,
	ClaimDerivedAllocationOffset,
	ClaimRawMemoryAccessChecked,
	ClaimRawSliceVerifiedAllocationRoot,
	ClaimUnsafeContractRuntimeCheckable,
	ClaimRawBoundsRuntimeCheckNormalBuild,
	ClaimRejectedNegativeOffset,
	ClaimRejectedUpperBound,
	ClaimRejectedAccessWidthOverflow,
	ClaimRejectedNegativeLength,
	ClaimRejectedLengthOverflow,
}

var capMemProofPromotionClaims = []string{
	string(ProvenanceSafeKnown),
	ClaimProvenanceKnown,
	ClaimNoAlias,
	ClaimIndexInRange,
	ClaimBoundsProofID,
	ClaimBoundsCheckEliminated,
	ClaimBoundsCheckRemovedWithProofID,
}

var safeProvenanceClasses = []string{
	string(ProvenanceSafeKnown),
	string(ProvenanceSafeBorrowed),
	string(ProvenanceSafeOwned),
}

var unsafeUnknownOptimizationClaims = []string{
	ClaimProvenanceKnown,
	ClaimNoAlias,
	ClaimIndexInRange,
	ClaimBoundsCheckEliminated,
	ClaimTrustedStorage,
	string(ProvenanceSafeKnown),
	string(ProvenanceSafeBorrowed),
	string(ProvenanceSafeOwned),
}

var unsafeVerifiedRootAllowedClaims = []string{
	ClaimAllocationBaseMetadata,
	ClaimUnsafeVerifiedRootAllocationBase,
}

var conservativeNoaliasBoundaryClaims = []string{
	ClaimCallbackInoutConservative,
	ClaimProtocolDispatchNoaliasConservative,
	ClaimBoundaryNoaliasConservative,
	ClaimTaskGroupNoaliasConservative,
	ClaimFFINoaliasInvalidatedByExternalCall,
}

var trustedStorageClasses = []string{
	string(StorageEliminated),
	string(StorageRegister),
	string(StorageStack),
	string(StorageRegion),
	string(StorageExplicitIsland),
	string(StorageFunctionTempRegion),
	string(StorageTaskRegion),
	string(StorageActorMoveRegion),
}

var runtimeProofRequiredStorageClasses = []string{
	string(StorageTaskRegion),
	string(StorageActorMoveRegion),
}

var memoryFuzzStatuses = []string{
	FuzzStatusCovered,
	FuzzStatusValidatedNarrow,
	FuzzStatusBoundaryRecorded,
	FuzzStatusBlocksRelease,
	FuzzStatusReleaseBlocking,
	FuzzStatusFuture,
}

func SourceStages() []string         { return copyStrings(sourceStages) }
func ProvenanceClasses() []string    { return copyStrings(provenanceClasses) }
func UnsafeClasses() []string        { return copyStrings(unsafeClasses) }
func AliasStates() []string          { return copyStrings(aliasStates) }
func StorageClasses() []string       { return copyStrings(storageClasses) }
func ClaimLevels() []string          { return copyStrings(claimLevels) }
func ValidatorStatuses() []string    { return copyStrings(validatorStatuses) }
func CostClasses() []string          { return copyStrings(costClasses) }
func ReportClaims() []string         { return copyStrings(reportClaims) }
func ParentRequiredClaims() []string { return copyStrings(parentRequiredClaims) }
func MemoryFuzzStatuses() []string   { return copyStrings(memoryFuzzStatuses) }
func IslandKernelEvidenceClaims() []string {
	return copyStrings(islandKernelEvidenceClaims)
}

func KnownSourceStage(value string) bool      { return contains(sourceStages, value) }
func KnownProvenanceClass(value string) bool  { return contains(provenanceClasses, value) }
func KnownUnsafeClass(value string) bool      { return contains(unsafeClasses, value) }
func KnownAliasState(value string) bool       { return contains(aliasStates, value) }
func KnownStorageClass(value string) bool     { return contains(storageClasses, value) }
func KnownDomainKind(value string) bool       { return value == "" || contains(domainKinds, value) }
func KnownTransferKind(value string) bool     { return value == "" || contains(transferKinds, value) }
func KnownClaimLevel(value string) bool       { return contains(claimLevels, value) }
func KnownValidatorStatus(value string) bool  { return contains(validatorStatuses, value) }
func KnownCostClass(value string) bool        { return contains(costClasses, value) }
func KnownMemoryFuzzStatus(value string) bool { return contains(memoryFuzzStatuses, value) }

func KnownReportClaim(value string) bool {
	return containsClaim(reportClaims, value)
}

func IslandKernelEvidenceClaim(value string) bool {
	return containsClaim(islandKernelEvidenceClaims, value)
}

func RequiredIslandKernelClaimValidator(claim string) string {
	return requiredIslandKernelClaimValidators[normalizeClaim(claim)]
}

func IslandKernelClaimValidatorMismatch(claim string, validatorName string) bool {
	required := RequiredIslandKernelClaimValidator(claim)
	if required == "" {
		return false
	}
	return strings.ToLower(strings.TrimSpace(validatorName)) != required
}

func SafeProvenanceClass(value string) bool {
	return contains(safeProvenanceClasses, value)
}

func UnsafeUnknownRow(provenanceClass string, unsafeClass string) bool {
	return provenanceClass == string(ProvenanceUnsafeUnknown) || unsafeClass == string(UnsafeUnknown)
}

func UnsafeExternalRoot(provenanceClass string, unsafeClass string) bool {
	switch provenanceClass {
	case string(ProvenanceUnsafeUnknown), string(ProvenanceUnsafeChecked), string(ProvenanceUnsafeVerifiedRoot):
		return true
	}
	switch unsafeClass {
	case string(UnsafeUnknown), string(UnsafeChecked), string(UnsafeVerifiedRoot):
		return true
	default:
		return false
	}
}

func UnsafeUnknownOptimizationClaim(claim string, aliasState string) bool {
	claim = normalizeClaim(claim)
	if containsClaim(unsafeUnknownOptimizationClaims, claim) {
		return true
	}
	return aliasState == string(AliasUnique) || aliasState == string(AliasMutableExclusive)
}

func MemoryOptimizationClaim(claim string, aliasState string) bool {
	if UnsafeUnknownOptimizationClaim(claim, aliasState) {
		return true
	}
	claim = normalizeClaim(claim)
	return strings.Contains(claim, "eliminated") || strings.Contains(claim, "zero_cost")
}

func BareBoundsCheckEliminatedClaim(claim string) bool {
	return normalizeClaim(claim) == ClaimBoundsCheckEliminated
}

func DynamicRawRuntimeCheckClaim(claim string) bool {
	return containsClaim(dynamicRawRuntimeCheckClaims, claim)
}

func DynamicRawRuntimeCheckCostDisallowed(claim string, costClass string) bool {
	return DynamicRawRuntimeCheckClaim(claim) && costClass != string(CostDynamicCheckRequired)
}

func UnsafeCheckedDisallowedClaim(provenanceClass string, unsafeClass string, claim string) bool {
	if provenanceClass != string(ProvenanceUnsafeChecked) && unsafeClass != string(UnsafeChecked) {
		return false
	}
	if provenanceClass != string(ProvenanceUnsafeChecked) || unsafeClass != string(UnsafeChecked) {
		return true
	}
	return !containsClaim(unsafeCheckedAllowedClaims, claim)
}

func UnsafeVerifiedRootDisallowedClaim(
	provenanceClass string,
	unsafeClass string,
	claim string,
) bool {
	if provenanceClass != string(ProvenanceUnsafeVerifiedRoot) && unsafeClass != string(UnsafeVerifiedRoot) {
		return false
	}
	return !containsClaim(unsafeVerifiedRootAllowedClaims, claim)
}

func CapMemDisallowedProofClaim(claim string, validatorName string, reason string) bool {
	if !capMemAuthorizationContext(validatorName, reason) {
		return false
	}
	return containsClaim(capMemProofPromotionClaims, claim)
}

func CapMemAuthorizationContext(validatorName string, reason string) bool {
	return capMemAuthorizationContext(validatorName, reason)
}

func capMemAuthorizationContext(validatorName string, reason string) bool {
	context := strings.ToLower(strings.TrimSpace(validatorName) + " " + strings.TrimSpace(reason))
	return strings.Contains(context, "cap.mem") || strings.Contains(context, "cap_mem")
}

func BroadNoAliasClaim(claim string) bool {
	switch normalizeClaim(claim) {
	case ClaimBroadNoAlias, ClaimUniversalNoAlias, ClaimFullNoAliasModel:
		return true
	default:
		return false
	}
}

func ConservativeNoAliasBoundaryClaim(claim string) bool {
	return containsClaim(conservativeNoaliasBoundaryClaims, claim)
}

func ClaimRequiresParentFactID(claim string) bool {
	return containsClaim(parentRequiredClaims, claim)
}

func ValidatedNoAliasState(value string) bool {
	return value == string(AliasUnique) || value == string(AliasMutableExclusive)
}

func UnsafeUnknownTrustedStorage(planned string, actual string) bool {
	return contains(trustedStorageClasses, planned) || contains(trustedStorageClasses, actual)
}

func UnsafeExternalRootTrustedStorage(
	provenanceClass string,
	unsafeClass string,
	planned string,
	actual string,
) bool {
	return UnsafeExternalRoot(provenanceClass, unsafeClass) &&
		UnsafeUnknownTrustedStorage(planned, actual)
}

func ValidatedTrustedStorageHeapFallback(planned string, actual string) bool {
	return actual == string(StorageHeap) && contains(trustedStorageClasses, planned)
}

func RuntimeProofRequiredStorage(planned string, actual string) bool {
	return contains(runtimeProofRequiredStorageClasses, planned) ||
		contains(runtimeProofRequiredStorageClasses, actual)
}

func ZeroCostValidationRequiredClaim(claim string) bool {
	switch normalizeClaim(claim) {
	case ClaimAllocationBaseMetadata, ClaimUnsafeVerifiedRootAllocationBase, ClaimStorageLowering:
		return true
	case ClaimPreAwaitLocalBorrowValidated:
		return true
	default:
		return false
	}
}

func ZeroCostProvenClaimDisallowed(
	claim string,
	costClass string,
	claimLevel string,
	plannedStorage string,
	loweringStorageActual string,
) bool {
	if costClass != string(CostZeroCostProven) {
		return false
	}
	switch claimLevel {
	case string(ClaimConservative), string(ClaimRejected), string(ClaimFuture):
		return true
	}
	claim = normalizeClaim(claim)
	if ZeroCostValidationRequiredClaim(claim) && claimLevel != string(ClaimValidated) {
		return true
	}
	if claim == ClaimStorageLowering {
		if RuntimeProofRequiredStorage(plannedStorage, loweringStorageActual) {
			return true
		}
		return loweringStorageActual == string(StorageHeap) && plannedStorage != "" &&
			plannedStorage != string(StorageHeap)
	}
	return !containsClaim(zeroCostProvenClaims, claim)
}

func RowRequiresArtifact(plannedStorage string, loweringStorageActual string, claim string) bool {
	if plannedStorage != "" || loweringStorageActual != "" {
		return true
	}
	claim = normalizeClaim(claim)
	return strings.Contains(claim, "lowering") || strings.Contains(claim, "storage")
}

func InferredCostClass(
	claim string,
	plannedStorage string,
	loweringStorageActual string,
	claimLevelRejected bool,
	unsafeUnknown bool,
	escapeState string,
	aliasState string,
) string {
	claim = normalizeClaim(claim)
	if strings.HasPrefix(claim, "rejected_") || claimLevelRejected {
		return string(CostUnsupportedRejected)
	}
	if unsafeUnknown {
		return string(CostConservativeFallback)
	}
	if RuntimeProofRequiredStorage(plannedStorage, loweringStorageActual) {
		return string(CostConservativeFallback)
	}
	if containsClaim(dynamicCheckRequiredClaims, claim) {
		return string(CostDynamicCheckRequired)
	}
	if containsClaim(zeroCostProvenClaims, claim) {
		return string(CostZeroCostProven)
	}
	if containsClaim(conservativeFallbackClaims, claim) {
		return string(CostConservativeFallback)
	}
	if containsClaim(unsupportedRejectedClaims, claim) {
		return string(CostUnsupportedRejected)
	}
	if claim == ClaimStorageLowering {
		if loweringStorageActual == string(StorageHeap) && plannedStorage != "" &&
			plannedStorage != string(StorageHeap) {
			return string(CostConservativeFallback)
		}
		return string(CostZeroCostProven)
	}
	if strings.Contains(claim, "unknown") || escapeState == "unknown" ||
		aliasState == string(AliasUnknownConservative) ||
		plannedStorage == string(StorageUnknownConservative) {
		return string(CostConservativeFallback)
	}
	return string(CostInstrumentationOnly)
}

func copyStrings(values []string) []string {
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func containsClaim(values []string, value string) bool {
	value = normalizeClaim(value)
	for _, candidate := range values {
		if normalizeClaim(candidate) == value {
			return true
		}
	}
	return false
}

func normalizeClaim(claim string) string {
	return strings.ToLower(strings.TrimSpace(claim))
}
