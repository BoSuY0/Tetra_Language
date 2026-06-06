package memorymodel

import "testing"

func TestMiniMemoryModelV0RequiredOutcomes(t *testing.T) {
	tests := []struct {
		name    string
		input   Scenario
		outcome Outcome
		valid   bool
	}{
		{
			name:    "valid_borrow_local",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperStructField, Escape: EscapeLocalUse, BranchOwners: []string{"xs"}},
			outcome: OutcomeValidBorrowLocal,
			valid:   true,
		},
		{
			name:    "invalid_borrow_return_escape",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperStructField, Escape: EscapeReturn, BranchOwners: []string{"xs"}},
			outcome: OutcomeInvalidBorrowReturnEscape,
		},
		{
			name:    "valid_copy_escape",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperOptionalPayload, Escape: EscapeStore, Copied: true, BranchOwners: []string{"xs"}},
			outcome: OutcomeValidCopyEscape,
			valid:   true,
		},
		{
			name:    "invalid_branch_owner_mix",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperOptionalPayload, Escape: EscapeLocalUse, BranchOwners: []string{"left", "right"}},
			outcome: OutcomeInvalidBranchOwnerMix,
		},
		{
			name:    "invalid_unsafe_unknown_borrow",
			input:   Scenario{Source: SourceUnsafeUnknown, Wrapper: WrapperStructField, Escape: EscapeLocalUse},
			outcome: OutcomeInvalidUnsafeUnknownBorrow,
		},
		{
			name:    "valid_sequential_inout",
			input:   Scenario{InoutEvents: []InoutEvent{EventStartInout, EventEndInout, EventStartInout, EventEndInout}},
			outcome: OutcomeValidSequentialInout,
			valid:   true,
		},
		{
			name:    "invalid_alias_read_during_inout",
			input:   Scenario{InoutEvents: []InoutEvent{EventStartInout, EventAliasRead, EventEndInout}},
			outcome: OutcomeInvalidAliasReadDuringInout,
		},
		{
			name:    "invalid_alias_write_during_inout",
			input:   Scenario{InoutEvents: []InoutEvent{EventStartInout, EventAliasWrite, EventEndInout}},
			outcome: OutcomeInvalidAliasWriteDuringInout,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Evaluate(test.input)
			if got.Outcome != test.outcome || got.Valid != test.valid {
				t.Fatalf("Evaluate() = %+v, want outcome %s valid %v", got, test.outcome, test.valid)
			}
			if got.Reason == "" {
				t.Fatalf("Evaluate() = %+v, want reviewable reason", got)
			}
		})
	}
}

func TestMiniMemoryModelV0KeepsUnknownCallAndBranchMergeConservative(t *testing.T) {
	for _, test := range []struct {
		name    string
		events  []InoutEvent
		outcome Outcome
	}{
		{
			name:    "unknown_call",
			events:  []InoutEvent{EventStartInout, EventUnknownCall, EventEndInout},
			outcome: OutcomeInvalidUnknownCallDuringInout,
		},
		{
			name:    "branch_merge",
			events:  []InoutEvent{EventBranchMerge},
			outcome: OutcomeInvalidBranchMergedExclusive,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := Evaluate(Scenario{InoutEvents: test.events})
			if got.Outcome != test.outcome || got.Valid {
				t.Fatalf("Evaluate() = %+v, want invalid %s", got, test.outcome)
			}
		})
	}
}

func TestMiniMemoryModelV1EnumAndGenericWrapperCases(t *testing.T) {
	tests := []struct {
		name    string
		input   Scenario
		outcome Outcome
		valid   bool
	}{
		{
			name:    "enum_payload_local_use",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperEnumPayload, Escape: EscapeLocalUse, BranchOwners: []string{"xs"}},
			outcome: OutcomeValidBorrowLocal,
			valid:   true,
		},
		{
			name:    "enum_payload_return_rejected",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperEnumPayload, Escape: EscapeReturn, BranchOwners: []string{"xs"}},
			outcome: OutcomeInvalidBorrowReturnEscape,
		},
		{
			name:    "generic_wrapper_store_rejected",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperGenericWrapper, Escape: EscapeStore, BranchOwners: []string{"xs"}},
			outcome: OutcomeInvalidBorrowReturnEscape,
		},
		{
			name:    "generic_wrapper_copy_return_allowed",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperGenericWrapper, Escape: EscapeReturn, Copied: true, BranchOwners: []string{"xs"}},
			outcome: OutcomeValidCopyEscape,
			valid:   true,
		},
		{
			name:    "enum_payload_mixed_branch_owners_rejected",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperEnumPayload, Escape: EscapeLocalUse, BranchOwners: []string{"left", "right"}},
			outcome: OutcomeInvalidBranchOwnerMix,
		},
		{
			name:    "generic_wrapper_unsafe_unknown_rejected",
			input:   Scenario{Source: SourceUnsafeUnknown, Wrapper: WrapperGenericWrapper, Escape: EscapeLocalUse},
			outcome: OutcomeInvalidUnsafeUnknownBorrow,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Evaluate(test.input)
			if got.Outcome != test.outcome || got.Valid != test.valid {
				t.Fatalf("Evaluate() = %+v, want outcome %s valid %v", got, test.outcome, test.valid)
			}
			if got.Reason == "" {
				t.Fatalf("Evaluate() = %+v, want reviewable reason", got)
			}
		})
	}
}

func TestMiniMemoryModelV2FunctionTypedAndCallbackCases(t *testing.T) {
	tests := []struct {
		name    string
		input   Scenario
		outcome Outcome
		valid   bool
	}{
		{
			name:    "function_value_local_use",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperFunctionValue, Escape: EscapeLocalUse, BranchOwners: []string{"xs"}, CallbackTargetKnown: true},
			outcome: OutcomeValidBorrowLocal,
			valid:   true,
		},
		{
			name:    "known_callback_local_use",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperCallbackArg, Escape: EscapeLocalUse, BranchOwners: []string{"xs"}, CallbackTargetKnown: true},
			outcome: OutcomeValidBorrowLocal,
			valid:   true,
		},
		{
			name:    "borrowed_callback_escape_rejected",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperCallbackArg, Escape: EscapeReturn, BranchOwners: []string{"xs"}, CallbackTargetKnown: true},
			outcome: OutcomeInvalidBorrowReturnEscape,
		},
		{
			name:    "copied_callback_escape_allowed",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperCallbackArg, Escape: EscapeStore, Copied: true, BranchOwners: []string{"xs"}, CallbackTargetKnown: true},
			outcome: OutcomeValidCopyEscape,
			valid:   true,
		},
		{
			name:    "unknown_callback_target_conservative",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperCallbackArg, Escape: EscapeLocalUse, BranchOwners: []string{"xs"}},
			outcome: OutcomeConservativeUnknownCallbackTarget,
		},
		{
			name:    "callback_reentrant_inout_conservative",
			input:   Scenario{InoutEvents: []InoutEvent{EventStartInout, EventCallbackReentrantCall, EventEndInout}},
			outcome: OutcomeInvalidCallbackInoutAlias,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Evaluate(test.input)
			if got.Outcome != test.outcome || got.Valid != test.valid {
				t.Fatalf("Evaluate() = %+v, want outcome %s valid %v", got, test.outcome, test.valid)
			}
			if got.Reason == "" {
				t.Fatalf("Evaluate() = %+v, want reviewable reason", got)
			}
		})
	}
}

func TestMiniMemoryModelV3InterfaceProtocolCases(t *testing.T) {
	tests := []struct {
		name    string
		input   Scenario
		outcome Outcome
		valid   bool
	}{
		{
			name:    "known_static_protocol_target_local_use",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperInterfaceValue, Escape: EscapeLocalUse, BranchOwners: []string{"xs"}, DispatchTargetKnown: true},
			outcome: OutcomeValidBorrowLocal,
			valid:   true,
		},
		{
			name:    "borrowed_interface_escape_rejected",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperInterfaceValue, Escape: EscapeReturn, BranchOwners: []string{"xs"}, DispatchTargetKnown: true},
			outcome: OutcomeInvalidBorrowReturnEscape,
		},
		{
			name:    "unknown_protocol_dispatch_conservative",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperProtocolDispatch, Escape: EscapeLocalUse, BranchOwners: []string{"xs"}},
			outcome: OutcomeConservativeUnknownProtocolDispatch,
		},
		{
			name:    "protocol_dispatch_noalias_conservative",
			input:   Scenario{InoutEvents: []InoutEvent{EventStartInout, EventProtocolDispatchCall, EventEndInout}},
			outcome: OutcomeInvalidProtocolDispatchNoAlias,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Evaluate(test.input)
			if got.Outcome != test.outcome || got.Valid != test.valid {
				t.Fatalf("Evaluate() = %+v, want outcome %s valid %v", got, test.outcome, test.valid)
			}
			if got.Reason == "" {
				t.Fatalf("Evaluate() = %+v, want reviewable reason", got)
			}
		})
	}
}

func TestMiniMemoryModelV4AsyncTaskActorBoundaryCases(t *testing.T) {
	tests := []struct {
		name    string
		input   Scenario
		outcome Outcome
		valid   bool
	}{
		{
			name:    "local_async_use_before_suspension",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperAsyncBoundary, Escape: EscapeBeforeSuspension, BranchOwners: []string{"xs"}},
			outcome: OutcomeValidBorrowLocal,
			valid:   true,
		},
		{
			name:    "borrow_crossing_await_conservative",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperAsyncBoundary, Escape: EscapeAcrossAwait, BranchOwners: []string{"xs"}},
			outcome: OutcomeConservativeAsyncBoundaryBorrow,
		},
		{
			name:    "borrow_crossing_task_rejected",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperTaskBoundary, Escape: EscapeTaskBoundary, BranchOwners: []string{"xs"}},
			outcome: OutcomeInvalidTaskBoundaryBorrow,
		},
		{
			name:    "borrow_crossing_actor_rejected",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperActorBoundary, Escape: EscapeActorBoundary, BranchOwners: []string{"xs"}},
			outcome: OutcomeInvalidActorBoundaryBorrow,
		},
		{
			name:    "copied_value_crossing_task_allowed",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperTaskBoundary, Escape: EscapeTaskBoundary, Copied: true, BranchOwners: []string{"xs"}},
			outcome: OutcomeValidCopyEscape,
			valid:   true,
		},
		{
			name:    "owned_value_crossing_actor_allowed",
			input:   Scenario{Source: SourceOwned, Wrapper: WrapperActorBoundary, Escape: EscapeActorBoundary, BranchOwners: []string{"xs"}},
			outcome: OutcomeValidBorrowLocal,
			valid:   true,
		},
		{
			name:    "task_boundary_noalias_conservative",
			input:   Scenario{InoutEvents: []InoutEvent{EventStartInout, EventTaskBoundaryCall, EventEndInout}},
			outcome: OutcomeInvalidBoundaryNoAlias,
		},
		{
			name:    "actor_boundary_noalias_conservative",
			input:   Scenario{InoutEvents: []InoutEvent{EventStartInout, EventActorBoundaryCall, EventEndInout}},
			outcome: OutcomeInvalidBoundaryNoAlias,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Evaluate(test.input)
			if got.Outcome != test.outcome || got.Valid != test.valid {
				t.Fatalf("Evaluate() = %+v, want outcome %s valid %v", got, test.outcome, test.valid)
			}
			if got.Reason == "" {
				t.Fatalf("Evaluate() = %+v, want reviewable reason", got)
			}
		})
	}
}

func TestMiniMemoryModelV5RawPointerUnsafeContractCases(t *testing.T) {
	tests := []struct {
		name    string
		input   Scenario
		outcome Outcome
		valid   bool
	}{
		{
			name:    "alloc_bytes_root_ptr_add_in_bounds",
			input:   Scenario{Source: SourceUnsafeVerifiedRoot, Wrapper: WrapperRawPointer, Escape: EscapeRawPtrAdd, PointerInBounds: true},
			outcome: OutcomeValidUnsafeVerifiedRootBounds,
			valid:   true,
		},
		{
			name:    "runtime_checkable_nonnull_alignment_length",
			input:   Scenario{Source: SourceUnsafeVerifiedRoot, UnsafeContract: ContractNonNullAlignmentLength, RuntimeCheckable: true},
			outcome: OutcomeValidUnsafeRuntimeContract,
			valid:   true,
		},
		{
			name:    "unknown_pointer_cannot_become_safe_known",
			input:   Scenario{Source: SourceUnsafeUnknown, Wrapper: WrapperRawPointer, Escape: EscapeRawPtrAdd},
			outcome: OutcomeInvalidUnsafeUnknownSafeFacts,
		},
		{
			name:    "unknown_pointer_cannot_emit_noalias",
			input:   Scenario{Source: SourceUnsafeUnknown, UnsafeContract: ContractNoAlias},
			outcome: OutcomeInvalidUnsafeUnknownNoAlias,
		},
		{
			name:    "unsafe_noalias_static_untrusted",
			input:   Scenario{Source: SourceUnsafeVerifiedRoot, UnsafeContract: ContractNoAlias},
			outcome: OutcomeConservativeUnsafeStaticContract,
		},
		{
			name:    "unsafe_lifetime_region_static_untrusted",
			input:   Scenario{Source: SourceUnsafeVerifiedRoot, UnsafeContract: ContractLifetimeRegion},
			outcome: OutcomeConservativeUnsafeStaticContract,
		},
		{
			name:    "raw_slice_unknown_pointer_external_unknown",
			input:   Scenario{Source: SourceUnsafeUnknown, Wrapper: WrapperRawSliceFromParts, Escape: EscapeRawSliceFromParts, RawSliceLengthFits: true},
			outcome: OutcomeConservativeRawSliceExternalUnknown,
		},
		{
			name:    "raw_slice_verified_root_too_large_rejected",
			input:   Scenario{Source: SourceUnsafeVerifiedRoot, Wrapper: WrapperRawSliceFromParts, Escape: EscapeRawSliceFromParts},
			outcome: OutcomeInvalidRawSliceTooLarge,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Evaluate(test.input)
			if got.Outcome != test.outcome || got.Valid != test.valid {
				t.Fatalf("Evaluate() = %+v, want outcome %s valid %v", got, test.outcome, test.valid)
			}
			if got.Reason == "" {
				t.Fatalf("Evaluate() = %+v, want reviewable reason", got)
			}
		})
	}
}

func TestMiniMemoryModelV6BoundsProofCases(t *testing.T) {
	tests := []struct {
		name    string
		input   Scenario
		outcome Outcome
		valid   bool
	}{
		{
			name:    "proof_tagged_removed_check_valid",
			input:   Scenario{BoundsProof: BoundsProofPresent, ProofIDMatches: true},
			outcome: OutcomeValidBoundsCheckRemovedWithProofID,
			valid:   true,
		},
		{
			name:    "missing_proof_rejected",
			input:   Scenario{BoundsProof: BoundsProofMissing},
			outcome: OutcomeInvalidBoundsCheckMissingProofID,
		},
		{
			name:    "mismatched_proof_rejected",
			input:   Scenario{BoundsProof: BoundsProofPresent},
			outcome: OutcomeInvalidBoundsCheckMismatchedProofID,
		},
		{
			name:    "unsafe_unknown_cannot_eliminate_bounds_check",
			input:   Scenario{Source: SourceUnsafeUnknown, BoundsProof: BoundsProofUnsafeUnknown},
			outcome: OutcomeInvalidUnsafeUnknownBoundsElimination,
		},
		{
			name:    "retained_dynamic_check_normal_build",
			input:   Scenario{BoundsProof: BoundsProofRetainedDynamic, NormalBuildCheck: true},
			outcome: OutcomeValidBoundsCheckRetainedDynamic,
			valid:   true,
		},
		{
			name:    "raw_overflow_keeps_check_or_trap",
			input:   Scenario{BoundsProof: BoundsProofRawOverflow, BoundsOverflow: true, NormalBuildCheck: true},
			outcome: OutcomeConservativeRawBoundsRuntimeCheck,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Evaluate(test.input)
			if got.Outcome != test.outcome || got.Valid != test.valid {
				t.Fatalf("Evaluate() = %+v, want outcome %s valid %v", got, test.outcome, test.valid)
			}
			if got.Reason == "" {
				t.Fatalf("Evaluate() = %+v, want reviewable reason", got)
			}
		})
	}
}

func TestMiniMemoryModelV7FFICases(t *testing.T) {
	tests := []struct {
		name    string
		input   Scenario
		outcome Outcome
		valid   bool
	}{
		{
			name:    "external_pointer_remains_unknown",
			input:   Scenario{Source: SourceExternalPointer, Wrapper: WrapperRawPointer},
			outcome: OutcomeConservativeExternalPointerUnknown,
		},
		{
			name:    "ffi_call_may_retain_borrow",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperFFICall, Escape: EscapeFFIBoundary, BranchOwners: []string{"xs"}},
			outcome: OutcomeConservativeFFICallMayRetainBorrow,
		},
		{
			name:    "safe_wrapper_promotion_rejected",
			input:   Scenario{Source: SourceExternalPointer, Wrapper: WrapperSafeWrapperPromotion},
			outcome: OutcomeInvalidSafeWrapperPromotion,
		},
		{
			name:    "external_call_invalidates_noalias",
			input:   Scenario{InoutEvents: []InoutEvent{EventStartInout, EventExternalCall, EventEndInout}},
			outcome: OutcomeInvalidExternalCallNoAlias,
		},
		{
			name:    "owned_copy_crossing_ffi_allowed",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperFFICall, Escape: EscapeFFIBoundary, Copied: true, BranchOwners: []string{"xs"}},
			outcome: OutcomeValidCopyEscape,
			valid:   true,
		},
		{
			name:    "external_pointer_cannot_eliminate_bounds_check",
			input:   Scenario{Source: SourceExternalPointer, BoundsProof: BoundsProofUnsafeUnknown},
			outcome: OutcomeInvalidUnsafeUnknownBoundsElimination,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Evaluate(test.input)
			if got.Outcome != test.outcome || got.Valid != test.valid {
				t.Fatalf("Evaluate() = %+v, want outcome %s valid %v", got, test.outcome, test.valid)
			}
			if got.Reason == "" {
				t.Fatalf("Evaluate() = %+v, want reviewable reason", got)
			}
		})
	}
}

func TestMiniMemoryModelV9StorageCases(t *testing.T) {
	tests := []struct {
		name    string
		input   Scenario
		outcome Outcome
		valid   bool
	}{
		{
			name:    "escaped_return_cannot_use_trusted_stack",
			input:   Scenario{Source: SourceOwned, Escape: EscapeReturn, StoragePlan: StoragePlanTrustedStack},
			outcome: OutcomeInvalidEscapedTrustedStorage,
		},
		{
			name:    "trusted_stack_requires_no_escape_proof",
			input:   Scenario{Source: SourceOwned, Escape: EscapeLocalUse, StoragePlan: StoragePlanTrustedStack},
			outcome: OutcomeInvalidTrustedStorageMissingNoEscapeProof,
		},
		{
			name:    "heap_fallback_preserves_source_fact_and_reason",
			input:   Scenario{Source: SourceOwned, Escape: EscapeReturn, StoragePlan: StoragePlanHeapFallback, SourceFactIDPresent: true, StorageFallbackReasonPresent: true},
			outcome: OutcomeValidHeapFallbackReasonPreserved,
			valid:   true,
		},
		{
			name:    "task_boundary_storage_remains_conservative",
			input:   Scenario{Source: SourceOwned, Escape: EscapeTaskBoundary, StoragePlan: StoragePlanHeapFallback, SourceFactIDPresent: true, StorageFallbackReasonPresent: true},
			outcome: OutcomeConservativeBoundaryStorage,
		},
		{
			name:    "ffi_boundary_storage_remains_conservative",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperFFICall, Escape: EscapeFFIBoundary, StoragePlan: StoragePlanHeapFallback, SourceFactIDPresent: true, StorageFallbackReasonPresent: true},
			outcome: OutcomeConservativeBoundaryStorage,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Evaluate(test.input)
			if got.Outcome != test.outcome || got.Valid != test.valid {
				t.Fatalf("Evaluate() = %+v, want outcome %s valid %v", got, test.outcome, test.valid)
			}
			if got.Reason == "" {
				t.Fatalf("Evaluate() = %+v, want reviewable reason", got)
			}
		})
	}
}

func TestMiniMemoryModelV10AsyncCancellationStructuredBoundaryCases(t *testing.T) {
	tests := []struct {
		name    string
		input   Scenario
		outcome Outcome
		valid   bool
	}{
		{
			name:    "pre_await_local_non_escaping_borrow_validated",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperAsyncBoundary, Escape: EscapeBeforeSuspension, BranchOwners: []string{"xs"}, NoEscapeProof: true},
			outcome: OutcomeValidPreAwaitLocalBorrow,
			valid:   true,
		},
		{
			name:    "post_await_borrow_use_conservative",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperAsyncBoundary, Escape: EscapeAfterCancellation, BranchOwners: []string{"xs"}},
			outcome: OutcomeConservativePostAwaitBorrow,
		},
		{
			name:    "cancellation_invalidates_task_owned_borrow",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperTaskBoundary, Escape: EscapeCancellation, BranchOwners: []string{"task-owned"}},
			outcome: OutcomeInvalidCancellationBorrowLifetime,
		},
		{
			name:    "task_group_boundary_noalias_conservative",
			input:   Scenario{InoutEvents: []InoutEvent{EventStartInout, EventTaskGroupBoundaryCall, EventEndInout}},
			outcome: OutcomeConservativeTaskGroupNoAlias,
		},
		{
			name:    "actor_reentrant_callback_conservative",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperActorReentrantCallback, Escape: EscapeActorBoundary, BranchOwners: []string{"actor-state"}, StoragePlan: StoragePlanHeapFallback, SourceFactIDPresent: true, StorageFallbackReasonPresent: true},
			outcome: OutcomeConservativeActorReentrantCallback,
		},
		{
			name:    "async_boundary_trusted_storage_rejected",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperAsyncBoundary, Escape: EscapeAcrossAwait, StoragePlan: StoragePlanTrustedStack},
			outcome: OutcomeInvalidEscapedTrustedStorage,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Evaluate(test.input)
			if got.Outcome != test.outcome || got.Valid != test.valid {
				t.Fatalf("Evaluate() = %+v, want outcome %s valid %v", got, test.outcome, test.valid)
			}
			if got.Reason == "" {
				t.Fatalf("Evaluate() = %+v, want reviewable reason", got)
			}
		})
	}
}

func TestMiniMemoryModelV11DynamicProtocolWitnessCases(t *testing.T) {
	tests := []struct {
		name    string
		input   Scenario
		outcome Outcome
		valid   bool
	}{
		{
			name:    "dynamic_existential_borrow_carrier_conservative",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperDynamicExistential, Escape: EscapeStore, BranchOwners: []string{"xs"}},
			outcome: OutcomeConservativeDynamicExistentialBorrow,
		},
		{
			name:    "static_witness_parent_fact_validated",
			input:   Scenario{Source: SourceBorrowedView, Wrapper: WrapperStaticWitness, Escape: EscapeLocalUse, BranchOwners: []string{"xs"}, SourceFactIDPresent: true},
			outcome: OutcomeValidStaticWitnessBorrowFact,
			valid:   true,
		},
		{
			name:    "dynamic_protocol_dispatch_noalias_rejected",
			input:   Scenario{InoutEvents: []InoutEvent{EventStartInout, EventDynamicProtocolDispatchCall, EventEndInout}},
			outcome: OutcomeInvalidDynamicProtocolNoAlias,
		},
		{
			name:    "witness_lookup_unknown_provenance_promotion_rejected",
			input:   Scenario{Source: SourceUnsafeUnknown, Wrapper: WrapperWitnessTableLookup, Escape: EscapeLocalUse},
			outcome: OutcomeInvalidWitnessProvenancePromotion,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Evaluate(test.input)
			if got.Outcome != test.outcome || got.Valid != test.valid {
				t.Fatalf("Evaluate() = %+v, want outcome %s valid %v", got, test.outcome, test.valid)
			}
			if got.Reason == "" {
				t.Fatalf("Evaluate() = %+v, want reviewable reason", got)
			}
		})
	}
}
