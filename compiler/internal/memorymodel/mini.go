package memorymodel

type SourceKind string

const (
	SourceOwned              SourceKind = "owned_value"
	SourceBorrowedView       SourceKind = "borrowed_view"
	SourceUnsafeUnknown      SourceKind = "unsafe_unknown"
	SourceUnsafeVerifiedRoot SourceKind = "unsafe_verified_root"
	SourceExternalPointer    SourceKind = "external_pointer"
)

type WrapperKind string

const (
	WrapperNone                 WrapperKind = "none"
	WrapperStructField          WrapperKind = "struct_field"
	WrapperOptionalPayload      WrapperKind = "optional_payload"
	WrapperEnumPayload          WrapperKind = "enum_payload"
	WrapperGenericWrapper       WrapperKind = "generic_wrapper"
	WrapperFunctionValue        WrapperKind = "function_value"
	WrapperCallbackArg          WrapperKind = "callback_arg"
	WrapperInterfaceValue       WrapperKind = "interface_value"
	WrapperProtocolDispatch     WrapperKind = "protocol_dispatch"
	WrapperAsyncBoundary        WrapperKind = "async_boundary"
	WrapperTaskBoundary         WrapperKind = "task_boundary"
	WrapperActorBoundary        WrapperKind = "actor_boundary"
	WrapperFFICall              WrapperKind = "ffi_call"
	WrapperSafeWrapperPromotion WrapperKind = "safe_wrapper_promotion"
	WrapperRawPointer           WrapperKind = "raw_pointer"
	WrapperRawSliceFromParts    WrapperKind = "raw_slice_from_parts"
)

type EscapeKind string

const (
	EscapeLocalUse          EscapeKind = "local_use"
	EscapeReturn            EscapeKind = "return"
	EscapeStore             EscapeKind = "store"
	EscapeBeforeSuspension  EscapeKind = "before_suspension"
	EscapeAcrossAwait       EscapeKind = "across_await"
	EscapeTaskBoundary      EscapeKind = "task_boundary"
	EscapeActorBoundary     EscapeKind = "actor_boundary"
	EscapeFFIBoundary       EscapeKind = "ffi_boundary"
	EscapeRawPtrAdd         EscapeKind = "raw_ptr_add"
	EscapeRawSliceFromParts EscapeKind = "raw_slice_from_parts"
)

type UnsafeContractKind string

const (
	ContractNone                   UnsafeContractKind = ""
	ContractNonNullAlignmentLength UnsafeContractKind = "nonnull_alignment_length"
	ContractNoAlias                UnsafeContractKind = "noalias"
	ContractLifetimeRegion         UnsafeContractKind = "lifetime_region"
)

type BoundsProofKind string

const (
	BoundsProofNone            BoundsProofKind = ""
	BoundsProofPresent         BoundsProofKind = "proof_present"
	BoundsProofMissing         BoundsProofKind = "proof_missing"
	BoundsProofUnsafeUnknown   BoundsProofKind = "unsafe_unknown_proof"
	BoundsProofRetainedDynamic BoundsProofKind = "retained_dynamic"
	BoundsProofRawOverflow     BoundsProofKind = "raw_overflow"
)

type StoragePlanKind string

const (
	StoragePlanNone                   StoragePlanKind = ""
	StoragePlanHeapFallback           StoragePlanKind = "heap_fallback"
	StoragePlanTrustedStack           StoragePlanKind = "trusted_stack"
	StoragePlanTrustedRegion          StoragePlanKind = "trusted_region"
	StoragePlanTrustedFunctionRegion  StoragePlanKind = "trusted_function_temp_region"
	StoragePlanTrustedTaskRegion      StoragePlanKind = "trusted_task_region"
	StoragePlanTrustedActorMoveRegion StoragePlanKind = "trusted_actor_move_region"
)

type InoutEvent string

const (
	EventStartInout            InoutEvent = "start_inout"
	EventEndInout              InoutEvent = "end_inout"
	EventAliasRead             InoutEvent = "alias_read"
	EventAliasWrite            InoutEvent = "alias_write"
	EventUnknownCall           InoutEvent = "unknown_call"
	EventBranchMerge           InoutEvent = "branch_merge"
	EventCallbackReentrantCall InoutEvent = "callback_reentrant_call"
	EventProtocolDispatchCall  InoutEvent = "protocol_dispatch_call"
	EventTaskBoundaryCall      InoutEvent = "task_boundary_call"
	EventActorBoundaryCall     InoutEvent = "actor_boundary_call"
	EventExternalCall          InoutEvent = "external_call"
)

type Outcome string

const (
	OutcomeValidBorrowLocal                          Outcome = "valid_borrow_local"
	OutcomeInvalidBorrowReturnEscape                 Outcome = "invalid_borrow_return_escape"
	OutcomeValidCopyEscape                           Outcome = "valid_copy_escape"
	OutcomeInvalidBranchOwnerMix                     Outcome = "invalid_branch_owner_mix"
	OutcomeInvalidUnsafeUnknownBorrow                Outcome = "invalid_unsafe_unknown_borrow"
	OutcomeValidSequentialInout                      Outcome = "valid_sequential_inout"
	OutcomeInvalidAliasReadDuringInout               Outcome = "invalid_alias_read_during_inout"
	OutcomeInvalidAliasWriteDuringInout              Outcome = "invalid_alias_write_during_inout"
	OutcomeInvalidUnknownCallDuringInout             Outcome = "invalid_unknown_call_during_inout"
	OutcomeInvalidBranchMergedExclusive              Outcome = "invalid_branch_merged_mutable_exclusive"
	OutcomeConservativeUnknownCallbackTarget         Outcome = "conservative_unknown_callback_target"
	OutcomeInvalidCallbackInoutAlias                 Outcome = "invalid_callback_inout_alias"
	OutcomeConservativeUnknownProtocolDispatch       Outcome = "conservative_unknown_protocol_dispatch"
	OutcomeInvalidProtocolDispatchNoAlias            Outcome = "invalid_protocol_dispatch_noalias"
	OutcomeConservativeAsyncBoundaryBorrow           Outcome = "conservative_async_boundary_borrow"
	OutcomeInvalidTaskBoundaryBorrow                 Outcome = "invalid_task_boundary_borrow"
	OutcomeInvalidActorBoundaryBorrow                Outcome = "invalid_actor_boundary_borrow"
	OutcomeInvalidBoundaryNoAlias                    Outcome = "invalid_boundary_noalias"
	OutcomeValidUnsafeVerifiedRootBounds             Outcome = "valid_unsafe_verified_root_bounds"
	OutcomeValidUnsafeRuntimeContract                Outcome = "valid_unsafe_runtime_contract"
	OutcomeInvalidUnsafeUnknownSafeFacts             Outcome = "invalid_unsafe_unknown_safe_facts"
	OutcomeInvalidUnsafeUnknownNoAlias               Outcome = "invalid_unsafe_unknown_noalias"
	OutcomeConservativeUnsafeStaticContract          Outcome = "conservative_unsafe_static_contract"
	OutcomeConservativeRawSliceExternalUnknown       Outcome = "conservative_raw_slice_external_unknown"
	OutcomeInvalidRawSliceTooLarge                   Outcome = "invalid_raw_slice_too_large"
	OutcomeValidBoundsCheckRemovedWithProofID        Outcome = "valid_bounds_check_removed_with_proof_id"
	OutcomeInvalidBoundsCheckMissingProofID          Outcome = "invalid_bounds_check_missing_proof_id"
	OutcomeInvalidBoundsCheckMismatchedProofID       Outcome = "invalid_bounds_check_mismatched_proof_id"
	OutcomeInvalidUnsafeUnknownBoundsElimination     Outcome = "invalid_unsafe_unknown_bounds_elimination"
	OutcomeValidBoundsCheckRetainedDynamic           Outcome = "valid_bounds_check_retained_dynamic"
	OutcomeConservativeRawBoundsRuntimeCheck         Outcome = "conservative_raw_bounds_runtime_check"
	OutcomeConservativeExternalPointerUnknown        Outcome = "conservative_external_pointer_unknown"
	OutcomeConservativeFFICallMayRetainBorrow        Outcome = "conservative_ffi_call_may_retain_borrow"
	OutcomeInvalidSafeWrapperPromotion               Outcome = "invalid_safe_wrapper_promotion"
	OutcomeInvalidExternalCallNoAlias                Outcome = "invalid_external_call_noalias"
	OutcomeInvalidEscapedTrustedStorage              Outcome = "invalid_escaped_trusted_storage"
	OutcomeInvalidTrustedStorageMissingNoEscapeProof Outcome = "invalid_trusted_storage_missing_no_escape_proof"
	OutcomeValidHeapFallbackReasonPreserved          Outcome = "valid_heap_fallback_reason_preserved"
	OutcomeInvalidHeapFallbackEvidence               Outcome = "invalid_heap_fallback_evidence"
	OutcomeConservativeBoundaryStorage               Outcome = "conservative_boundary_storage"
)

type Scenario struct {
	Source                       SourceKind
	Wrapper                      WrapperKind
	Escape                       EscapeKind
	Copied                       bool
	BranchOwners                 []string
	InoutEvents                  []InoutEvent
	CallbackTargetKnown          bool
	DispatchTargetKnown          bool
	UnsafeContract               UnsafeContractKind
	PointerInBounds              bool
	RuntimeCheckable             bool
	RawSliceLengthFits           bool
	BoundsProof                  BoundsProofKind
	ProofIDMatches               bool
	NormalBuildCheck             bool
	BoundsOverflow               bool
	StoragePlan                  StoragePlanKind
	NoEscapeProof                bool
	SourceFactIDPresent          bool
	StorageFallbackReasonPresent bool
}

type Result struct {
	Outcome Outcome
	Valid   bool
	Reason  string
}

func Evaluate(s Scenario) Result {
	if result, ok := evaluateStorage(s); ok {
		return result
	}
	if result, ok := evaluateBoundsProof(s); ok {
		return result
	}
	if len(s.InoutEvents) > 0 {
		return evaluateInout(s.InoutEvents)
	}
	if result, ok := evaluateFFI(s); ok {
		return result
	}
	if result, ok := evaluateRawUnsafe(s); ok {
		return result
	}
	if distinctNonEmpty(s.BranchOwners) > 1 {
		return Result{Outcome: OutcomeInvalidBranchOwnerMix, Reason: "borrowed value has multiple visible branch owners"}
	}
	if s.Source == SourceUnsafeUnknown {
		return Result{Outcome: OutcomeInvalidUnsafeUnknownBorrow, Reason: "unsafe_unknown cannot produce trusted borrowed ownership"}
	}
	if s.Wrapper == WrapperCallbackArg && !s.CallbackTargetKnown {
		return Result{Outcome: OutcomeConservativeUnknownCallbackTarget, Reason: "unknown callback target cannot produce trusted borrow facts"}
	}
	if s.Wrapper == WrapperProtocolDispatch && !s.DispatchTargetKnown {
		return Result{Outcome: OutcomeConservativeUnknownProtocolDispatch, Reason: "unknown protocol dispatch target cannot produce trusted borrow facts"}
	}
	if s.Copied && (s.Escape == EscapeReturn || s.Escape == EscapeStore || s.Escape == EscapeTaskBoundary || s.Escape == EscapeActorBoundary) {
		return Result{Outcome: OutcomeValidCopyEscape, Valid: true, Reason: "copy creates owned provenance before escape"}
	}
	if s.Source == SourceBorrowedView && s.Wrapper == WrapperAsyncBoundary && s.Escape == EscapeAcrossAwait {
		return Result{Outcome: OutcomeConservativeAsyncBoundaryBorrow, Reason: "borrowed view crossing async suspension remains conservative unless proven local and non-escaping"}
	}
	if s.Source == SourceBorrowedView && s.Wrapper == WrapperTaskBoundary && s.Escape == EscapeTaskBoundary {
		return Result{Outcome: OutcomeInvalidTaskBoundaryBorrow, Reason: "borrowed view cannot cross task boundary without explicit copy"}
	}
	if s.Source == SourceBorrowedView && s.Wrapper == WrapperActorBoundary && s.Escape == EscapeActorBoundary {
		return Result{Outcome: OutcomeInvalidActorBoundaryBorrow, Reason: "borrowed view cannot cross actor boundary without explicit copy"}
	}
	if s.Source == SourceBorrowedView && s.Wrapper != WrapperNone && (s.Escape == EscapeReturn || s.Escape == EscapeStore) {
		return Result{Outcome: OutcomeInvalidBorrowReturnEscape, Reason: "borrowed aggregate cannot escape its owner"}
	}
	return Result{Outcome: OutcomeValidBorrowLocal, Valid: true, Reason: "borrowed view stays in local scope"}
}

func evaluateStorage(s Scenario) (Result, bool) {
	if s.StoragePlan == StoragePlanNone {
		return Result{}, false
	}
	if storagePlanIsTrusted(s.StoragePlan) && escapeCrossesStorageBoundary(s.Escape) {
		return Result{Outcome: OutcomeInvalidEscapedTrustedStorage, Reason: "escaped value cannot lower as trusted stack, region, task, actor, or island storage"}, true
	}
	if storagePlanIsTrusted(s.StoragePlan) && !s.NoEscapeProof {
		return Result{Outcome: OutcomeInvalidTrustedStorageMissingNoEscapeProof, Reason: "trusted stack or region storage requires compiler-owned no-escape proof"}, true
	}
	if s.StoragePlan == StoragePlanHeapFallback {
		if !s.SourceFactIDPresent || !s.StorageFallbackReasonPresent {
			return Result{Outcome: OutcomeInvalidHeapFallbackEvidence, Reason: "heap fallback must preserve source_fact_id and a reviewable reason"}, true
		}
		if escapeCrossesAsyncTaskActorFFIOrUnknownBoundary(s.Escape) || s.Wrapper == WrapperFFICall {
			return Result{Outcome: OutcomeConservativeBoundaryStorage, Reason: "async, task, actor, FFI, or unknown-call boundary keeps storage conservative"}, true
		}
		return Result{Outcome: OutcomeValidHeapFallbackReasonPreserved, Valid: true, Reason: "heap fallback preserves source_fact_id and reason"}, true
	}
	return Result{}, false
}

func storagePlanIsTrusted(plan StoragePlanKind) bool {
	switch plan {
	case StoragePlanTrustedStack, StoragePlanTrustedRegion, StoragePlanTrustedFunctionRegion,
		StoragePlanTrustedTaskRegion, StoragePlanTrustedActorMoveRegion:
		return true
	default:
		return false
	}
}

func escapeCrossesStorageBoundary(escape EscapeKind) bool {
	switch escape {
	case EscapeReturn, EscapeStore, EscapeAcrossAwait, EscapeTaskBoundary, EscapeActorBoundary, EscapeFFIBoundary:
		return true
	default:
		return false
	}
}

func escapeCrossesAsyncTaskActorFFIOrUnknownBoundary(escape EscapeKind) bool {
	switch escape {
	case EscapeAcrossAwait, EscapeTaskBoundary, EscapeActorBoundary, EscapeFFIBoundary:
		return true
	default:
		return false
	}
}

func evaluateFFI(s Scenario) (Result, bool) {
	if s.Wrapper == WrapperFFICall && s.Copied && (s.Escape == EscapeFFIBoundary || s.Escape == EscapeReturn || s.Escape == EscapeStore) {
		return Result{Outcome: OutcomeValidCopyEscape, Valid: true, Reason: "explicit copy creates owned provenance before crossing FFI"}, true
	}
	if s.Source == SourceExternalPointer && s.Wrapper == WrapperSafeWrapperPromotion {
		return Result{Outcome: OutcomeInvalidSafeWrapperPromotion, Reason: "safe wrapper promotion from external pointer requires compiler-owned proof"}, true
	}
	if s.Source == SourceUnsafeUnknown && s.Wrapper == WrapperSafeWrapperPromotion {
		return Result{Outcome: OutcomeInvalidSafeWrapperPromotion, Reason: "unsafe_unknown cannot be promoted into a safe wrapper without proof"}, true
	}
	if s.Wrapper == WrapperFFICall && s.Source == SourceBorrowedView && s.Escape == EscapeFFIBoundary {
		return Result{Outcome: OutcomeConservativeFFICallMayRetainBorrow, Reason: "external call may retain borrowed pointer unless a compiler-owned contract proves otherwise"}, true
	}
	if s.Source == SourceExternalPointer && (s.Wrapper == WrapperRawPointer || s.Wrapper == WrapperRawSliceFromParts || s.Wrapper == WrapperFFICall) {
		return Result{Outcome: OutcomeConservativeExternalPointerUnknown, Reason: "external pointer remains unsafe_unknown without compiler-owned provenance"}, true
	}
	return Result{}, false
}

func evaluateBoundsProof(s Scenario) (Result, bool) {
	if s.BoundsProof == BoundsProofNone {
		return Result{}, false
	}
	switch s.BoundsProof {
	case BoundsProofPresent:
		if s.ProofIDMatches {
			return Result{Outcome: OutcomeValidBoundsCheckRemovedWithProofID, Valid: true, Reason: "removed bounds check has compiler-owned proof id linked to PLIR proof guards"}, true
		}
		return Result{Outcome: OutcomeInvalidBoundsCheckMismatchedProofID, Reason: "removed bounds check proof id does not match live PLIR proof guards"}, true
	case BoundsProofMissing:
		return Result{Outcome: OutcomeInvalidBoundsCheckMissingProofID, Reason: "removed bounds check without proof id is rejected"}, true
	case BoundsProofUnsafeUnknown:
		return Result{Outcome: OutcomeInvalidUnsafeUnknownBoundsElimination, Reason: "unsafe_unknown cannot authorize eliminated bounds checks"}, true
	case BoundsProofRetainedDynamic:
		if s.NormalBuildCheck {
			return Result{Outcome: OutcomeValidBoundsCheckRetainedDynamic, Valid: true, Reason: "missing proof keeps bounds check in the normal build"}, true
		}
		return Result{Outcome: OutcomeInvalidBoundsCheckMissingProofID, Reason: "retained dynamic bounds check requires normal_build_check evidence"}, true
	case BoundsProofRawOverflow:
		if s.BoundsOverflow || s.NormalBuildCheck {
			return Result{Outcome: OutcomeConservativeRawBoundsRuntimeCheck, Reason: "raw bounds width or overflow uncertainty keeps a normal-build check or trap"}, true
		}
		return Result{Outcome: OutcomeInvalidBoundsCheckMismatchedProofID, Reason: "raw bounds uncertainty cannot become zero-cost eliminated without proof"}, true
	default:
		return Result{}, false
	}
}

func evaluateRawUnsafe(s Scenario) (Result, bool) {
	if !isRawUnsafeScenario(s) {
		return Result{}, false
	}
	if s.Source == SourceUnsafeUnknown && s.UnsafeContract == ContractNoAlias {
		return Result{Outcome: OutcomeInvalidUnsafeUnknownNoAlias, Reason: "unsafe_unknown raw pointer cannot emit noalias facts"}, true
	}
	if s.Wrapper == WrapperRawSliceFromParts && s.Source == SourceUnsafeUnknown {
		return Result{Outcome: OutcomeConservativeRawSliceExternalUnknown, Reason: "raw_slice_from_parts over unknown pointer remains external_unknown"}, true
	}
	if s.UnsafeContract == ContractNoAlias || s.UnsafeContract == ContractLifetimeRegion {
		return Result{Outcome: OutcomeConservativeUnsafeStaticContract, Reason: "unsafe noalias/lifetime/region contracts remain static-untrusted unless separately proven"}, true
	}
	if s.UnsafeContract == ContractNonNullAlignmentLength && s.RuntimeCheckable {
		return Result{Outcome: OutcomeValidUnsafeRuntimeContract, Valid: true, Reason: "nonnull, alignment, and length are runtime-checkable unsafe contracts"}, true
	}
	if s.Wrapper == WrapperRawSliceFromParts && s.Source == SourceUnsafeVerifiedRoot {
		if s.RawSliceLengthFits {
			return Result{Outcome: OutcomeValidUnsafeVerifiedRootBounds, Valid: true, Reason: "raw_slice_from_parts fits verified allocation-root bounds"}, true
		}
		return Result{Outcome: OutcomeInvalidRawSliceTooLarge, Reason: "raw_slice_from_parts length exceeds verified allocation-root bounds"}, true
	}
	if s.Source == SourceUnsafeVerifiedRoot && s.Wrapper == WrapperRawPointer && s.Escape == EscapeRawPtrAdd && s.PointerInBounds {
		return Result{Outcome: OutcomeValidUnsafeVerifiedRootBounds, Valid: true, Reason: "ptr_add stays within core.alloc_bytes verified allocation-root bounds"}, true
	}
	if s.Source == SourceUnsafeUnknown && (s.Wrapper == WrapperRawPointer || s.Escape == EscapeRawPtrAdd || s.UnsafeContract != ContractNone) {
		return Result{Outcome: OutcomeInvalidUnsafeUnknownSafeFacts, Reason: "unsafe_unknown raw pointer cannot produce safe_known or provenance_known facts"}, true
	}
	return Result{}, false
}

func isRawUnsafeScenario(s Scenario) bool {
	return s.Source == SourceUnsafeVerifiedRoot ||
		s.Source == SourceExternalPointer ||
		s.Wrapper == WrapperRawPointer ||
		s.Wrapper == WrapperRawSliceFromParts ||
		s.Escape == EscapeRawPtrAdd ||
		s.Escape == EscapeRawSliceFromParts ||
		s.UnsafeContract != ContractNone
}

func evaluateInout(events []InoutEvent) Result {
	active := false
	for _, event := range events {
		switch event {
		case EventStartInout:
			if active {
				return Result{Outcome: OutcomeInvalidBranchMergedExclusive, Reason: "inout interval started while another exclusive interval is active"}
			}
			active = true
		case EventEndInout:
			active = false
		case EventAliasRead:
			if active {
				return Result{Outcome: OutcomeInvalidAliasReadDuringInout, Reason: "read alias used during active inout interval"}
			}
		case EventAliasWrite:
			if active {
				return Result{Outcome: OutcomeInvalidAliasWriteDuringInout, Reason: "write alias used during active inout interval"}
			}
		case EventUnknownCall:
			if active {
				return Result{Outcome: OutcomeInvalidUnknownCallDuringInout, Reason: "unknown call may retain an exclusive inout pointer"}
			}
		case EventCallbackReentrantCall:
			if active {
				return Result{Outcome: OutcomeInvalidCallbackInoutAlias, Reason: "callback or reentrant call invalidates broad inout noalias"}
			}
		case EventProtocolDispatchCall:
			if active {
				return Result{Outcome: OutcomeInvalidProtocolDispatchNoAlias, Reason: "protocol or interface dispatch invalidates broad inout noalias"}
			}
		case EventTaskBoundaryCall, EventActorBoundaryCall:
			if active {
				return Result{Outcome: OutcomeInvalidBoundaryNoAlias, Reason: "task or actor boundary invalidates broad inout noalias"}
			}
		case EventExternalCall:
			if active {
				return Result{Outcome: OutcomeInvalidExternalCallNoAlias, Reason: "external call invalidates broad inout noalias"}
			}
		case EventBranchMerge:
			return Result{Outcome: OutcomeInvalidBranchMergedExclusive, Reason: "branch-merged mutable_exclusive state is conservative"}
		}
	}
	return Result{Outcome: OutcomeValidSequentialInout, Valid: true, Reason: "all inout intervals ended before the next exclusive use"}
}

func distinctNonEmpty(values []string) int {
	seen := map[string]bool{}
	for _, value := range values {
		if value == "" {
			continue
		}
		seen[value] = true
	}
	return len(seen)
}
