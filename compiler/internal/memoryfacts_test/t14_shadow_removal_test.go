package memoryfacts_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type t14ParityRow struct {
	outcome       string
	oldTests      string
	realTest      string
	canonicalCode string
}

func TestT14MemoryModelParityManifestUsesRealPipelineTests(t *testing.T) {
	rows := t14MemoryModelParityRows()
	if len(rows) != 50 {
		t.Fatalf("parity rows = %d, want 50 memorymodel outcomes", len(rows))
	}

	contents := strings.Builder{}
	for _, file := range t14GoFiles(t) {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		contents.Write(raw)
		contents.WriteByte('\n')
	}
	allGo := contents.String()
	seen := map[string]bool{}
	for _, row := range rows {
		if row.outcome == "" || row.oldTests == "" || row.realTest == "" || row.canonicalCode == "" {
			t.Fatalf("incomplete parity row: %#v", row)
		}
		if strings.Contains(row.realTest, "MiniMemoryModel") {
			t.Fatalf("%s maps to shadow model test %q", row.outcome, row.realTest)
		}
		if seen[row.outcome] {
			t.Fatalf("duplicate parity outcome %q", row.outcome)
		}
		seen[row.outcome] = true
		if !strings.Contains(allGo, "func "+row.realTest+"(") {
			t.Fatalf("%s real-pipeline test %q not found", row.outcome, row.realTest)
		}
	}
}

func TestT14ShadowMemoryPackagesAndImportsAreRemoved(t *testing.T) {
	root := t14RepoRoot(t)
	for _, dir := range []string{
		filepath.Join(root, "compiler", "internal", "memory"+"model"),
		filepath.Join(root, "compiler", "memory"+"vocab"),
	} {
		if _, err := os.Stat(dir); err == nil {
			t.Fatalf("shadow memory package still exists: %s", dir)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", dir, err)
		}
	}

	for _, file := range t14GoFiles(t) {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		text := string(raw)
		for _, forbidden := range []string{
			"compiler/internal/" + "memorymodel",
			"memory" + "model.",
			"compiler/" + "memoryvocab",
			"memory" + "vocab.",
		} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s still contains forbidden shadow reference %q", file, forbidden)
			}
		}
	}
}

func TestT14NoDuplicateMemoryPolicyHelpersOutsideCanonicalPackages(t *testing.T) {
	for _, file := range t14GoFiles(t) {
		if strings.Contains(filepath.ToSlash(file), "/compiler/internal/memoryfacts/") ||
			strings.Contains(filepath.ToSlash(file), "/compiler/internal/islandkernel/") {
			continue
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		text := string(raw)
		for _, forbidden := range []string{
			"func " + "actual" + "Lowering" + "Storage",
			"func " + "looksActor",
			"func " + "looksTask",
			"func " + "classifyStorage",
			"func " + "unsafeUnknownOptimizationClaim",
			"func " + "memoryOptimizationClaim",
			"func " + "bareBoundsCheckEliminatedClaim",
			"func " + "dynamicRawRuntimeCheckCostDisallowed",
			"func " + "unsafeCheckedDisallowedClaim",
			"func " + "capMemDisallowedProofClaim",
			"func " + "broadNoAliasClaim",
			"func " + "conservativeNoAliasBoundaryClaim",
			"func " + "rowRequiresArtifact",
		} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s defines duplicate memory policy helper %q", file, forbidden)
			}
		}
	}
}

func t14MemoryModelParityRows() []t14ParityRow {
	return []t14ParityRow{
		{"valid_borrow_local", "valid_borrow_local; enum_payload_local_use; function_value_local_use; known_callback_local_use; known_static_protocol_target_local_use; local_async_use_before_suspension; owned_value_crossing_actor_allowed", "TestMemoryIdealV0ProjectsBorrowAggregateAndOptionalFacts", "borrowed_imm/no_escape"},
		{"invalid_borrow_return_escape", "invalid_borrow_return_escape; enum_payload_return_rejected; generic_wrapper_store_rejected; borrowed_callback_escape_rejected; borrowed_interface_escape_rejected", "TestMemoryIdealV0BorrowStructOptionalLocalAndCopyEscapes", "semantics.borrow_escape_rejected"},
		{"valid_copy_escape", "valid_copy_escape; generic_wrapper_copy_return_allowed; copied_callback_escape_allowed; copied_value_crossing_task_allowed; owned_copy_crossing_ffi_allowed", "TestMemoryIdealV0BorrowStructOptionalLocalAndCopyEscapes", "copy_owned"},
		{"invalid_branch_owner_mix", "invalid_branch_owner_mix; enum_payload_mixed_branch_owners_rejected", "TestOwnershipRejectsBorrowEscapeViaEnumPayloadBinding", "semantics.branch_owner_mix_rejected"},
		{"invalid_unsafe_unknown_borrow", "invalid_unsafe_unknown_borrow; generic_wrapper_unsafe_unknown_rejected", "TestMemoryFactsRejectsUnsafeUnknownToSafeKnown", "unsafe_unknown_rejected_safe_facts"},
		{"valid_sequential_inout", "valid_sequential_inout", "TestMemoryIdealV0SequentialInoutAndCopyThenInout", "no_alias_validated_narrow_sequential_inout"},
		{"invalid_alias_read_during_inout", "invalid_alias_read_during_inout", "TestOwnershipRejectsBorrowInoutAlias", "mutable_exclusive_alias_rejected"},
		{"invalid_alias_write_during_inout", "invalid_alias_write_during_inout", "TestOwnershipRejectsBorrowInoutAlias", "mutable_exclusive_alias_rejected"},
		{"invalid_unknown_call_during_inout", "unknown_call", "TestFromCheckedProgramDoesNotClaimNoAliasAfterRawInoutExposure", "alias_invalidated_by_call"},
		{"invalid_branch_merged_mutable_exclusive", "branch_merge", "TestMemoryIdealV0ProjectsNarrowInoutNoAliasFacts", "mutable_exclusive_branch_merge_rejected"},
		{"conservative_unknown_callback_target", "unknown_callback_target_conservative", "TestMemoryIdealV2UnknownCallbackTargetDoesNotEmitTrustedBorrowFacts", "callback_arg_contains_borrow"},
		{"invalid_callback_inout_alias", "callback_reentrant_inout_conservative", "TestMemoryIdealV2CallbackAliasesInoutArgumentRejected", "callback_inout_conservative"},
		{"conservative_unknown_protocol_dispatch", "unknown_protocol_dispatch_conservative", "TestMemoryIdealV3UnknownDynamicDispatchDoesNotEmitTrustedInterfaceFacts", "protocol_dispatch_borrow_conservative"},
		{"invalid_protocol_dispatch_noalias", "protocol_dispatch_noalias_conservative", "TestMemoryIdealV3ProtocolDispatchBroadNoAliasRejected", "protocol_dispatch_noalias_conservative"},
		{"conservative_async_boundary_borrow", "borrow_crossing_await_conservative", "TestMemoryIdealV4ProjectsAsyncTaskActorBoundaryFacts", "async_boundary_borrow_conservative"},
		{"invalid_task_boundary_borrow", "borrow_crossing_task_rejected", "TestMemoryIdealV4ProjectsAsyncTaskActorBoundaryFacts", "task_boundary_borrow_rejected"},
		{"invalid_actor_boundary_borrow", "borrow_crossing_actor_rejected", "TestMemoryIdealV4ProjectsAsyncTaskActorBoundaryFacts", "actor_boundary_borrow_rejected"},
		{"invalid_boundary_noalias", "task_boundary_noalias_conservative; actor_boundary_noalias_conservative", "TestMemoryIdealV4TaskActorBroadNoAliasRejected", "boundary_noalias_conservative"},
		{"valid_unsafe_verified_root_bounds", "alloc_bytes_root_ptr_add_in_bounds", "TestMemoryIdealV5ProjectsRawPointerUnsafeContractFacts", "unsafe_verified_root_allocation_base"},
		{"valid_unsafe_runtime_contract", "runtime_checkable_nonnull_alignment_length", "TestValidateMemoryReportAcceptsMemoryIdealV5UnsafeContractRows", "unsafe_contract_runtime_checkable"},
		{"invalid_unsafe_unknown_safe_facts", "unknown_pointer_cannot_become_safe_known", "TestMemoryFactsRejectsUnsafeUnknownToSafeKnown", "unsafe_unknown_rejected_safe_facts"},
		{"invalid_unsafe_unknown_noalias", "unknown_pointer_cannot_emit_noalias", "TestMemoryFactsRejectsUnsafeUnknownNoAliasAndBoundsProofClaims", "no_alias_rejected_for_unsafe_unknown"},
		{"conservative_unsafe_static_contract", "unsafe_noalias_static_untrusted; unsafe_lifetime_region_static_untrusted", "TestMemoryIdealV5ProjectsRawPointerUnsafeContractFacts", "unsafe_contract_static_untrusted"},
		{"conservative_raw_slice_external_unknown", "raw_slice_unknown_pointer_external_unknown", "TestMemoryIdealV5ProjectsRawPointerUnsafeContractFacts", "external_unknown"},
		{"invalid_raw_slice_too_large", "raw_slice_verified_root_too_large_rejected", "TestMemoryIdealV5RawSliceFromPartsUnsafeGatewayTypeChecks", "rejected_length_overflow"},
		{"valid_bounds_check_removed_with_proof_id", "proof_tagged_removed_check_valid", "TestMemoryIdealV6ProjectsBoundsProofFacts", "bounds_check_removed_with_proof_id"},
		{"invalid_bounds_check_missing_proof_id", "missing_proof_rejected", "TestMemoryIdealV6ProjectsMissingProofRejection", "bounds_check_removal_rejected_missing_proof_id"},
		{"invalid_bounds_check_mismatched_proof_id", "mismatched_proof_rejected", "TestCheckBoundsProofsWithPLIRRejectsTypedProofBaseMismatch", "bounds_proof_id_mismatch"},
		{"invalid_unsafe_unknown_bounds_elimination", "unsafe_unknown_cannot_eliminate_bounds_check; external_pointer_cannot_eliminate_bounds_check", "TestT13ProofSensitiveRewriteSkipsInvalidatedAndUnsafeProofs", "optimizer.proof_unsafe_skip"},
		{"valid_bounds_check_retained_dynamic", "retained_dynamic_check_normal_build", "TestMemoryReportRejectsDynamicOptimizationClaimWithoutNormalBuildCheck", "bounds_check_retained_dynamic"},
		{"conservative_raw_bounds_runtime_check", "raw_overflow_keeps_check_or_trap", "TestBuildRawPtrAddNegativeOffsetBoundsDiagnostic", "raw_bounds_runtime_check_normal_build"},
		{"conservative_external_pointer_unknown", "external_pointer_remains_unknown", "TestMemoryIdealV7ProjectsFFICallExternalFacts", "ffi_pointer_external_unknown"},
		{"conservative_ffi_call_may_retain_borrow", "ffi_call_may_retain_borrow", "TestMemoryIdealV7ProjectsFFICallExternalFacts", "ffi_call_may_retain_borrow"},
		{"invalid_safe_wrapper_promotion", "safe_wrapper_promotion_rejected", "TestValidateMemoryReportRejectsUnsafeCheckedGenericPromotions", "safe_wrapper_promotion_rejected_without_contract"},
		{"invalid_external_call_noalias", "external_call_invalidates_noalias", "TestFromCheckedProgramDoesNotClaimNoAliasAfterCallbackInoutBoundary", "ffi_noalias_invalidated_by_external_call"},
		{"invalid_escaped_trusted_storage", "escaped_return_cannot_use_trusted_stack; async_boundary_trusted_storage_rejected", "TestValidateMemoryReportRejectsValidatedTrustedStorageHeapFallback", "trusted_storage_rejected_for_escape"},
		{"invalid_trusted_storage_missing_no_escape_proof", "trusted_stack_requires_no_escape_proof", "TestMemoryFactsRejectsValidatedUnsafeUnknownTrustedStorage", "trusted_storage_requires_no_escape"},
		{"valid_heap_fallback_reason_preserved", "heap_fallback_preserves_source_fact_and_reason", "TestBuildGraphFromPLIRAndPlanDoesNotValidateStackHeapFallback", "heap_fallback_reason_preserved"},
		{"invalid_heap_fallback_evidence", "constant-only: no mini_test case", "TestValidateMemoryReportRejectsHeapFallbackWithoutReason", "heap_fallback_reason_required"},
		{"conservative_boundary_storage", "task_boundary_storage_remains_conservative; ffi_boundary_storage_remains_conservative", "TestMemoryIdealV4UnknownTaskActorTargetDoesNotEmitTrustedBoundaryFacts", "boundary_storage_conservative"},
		{"valid_pre_await_local_borrow", "pre_await_local_non_escaping_borrow_validated", "TestMemoryIdealV10ProjectsAsyncCancellationBoundaryFacts", "pre_await_local_borrow_validated"},
		{"conservative_post_await_borrow", "post_await_borrow_use_conservative", "TestMemoryIdealV10ProjectsAsyncCancellationBoundaryFacts", "post_await_borrow_conservative"},
		{"invalid_cancellation_borrow_lifetime", "cancellation_invalidates_task_owned_borrow", "TestMemoryIdealV10ProjectsAsyncCancellationBoundaryFacts", "cancellation_borrow_lifetime_invalidated"},
		{"conservative_task_group_noalias", "task_group_boundary_noalias_conservative", "TestMemoryIdealV10ProjectsAsyncCancellationBoundaryFacts", "task_group_noalias_conservative"},
		{"conservative_actor_reentrant_callback", "actor_reentrant_callback_conservative", "TestMemoryIdealV10ProjectsAsyncCancellationBoundaryFacts", "actor_reentrant_callback_conservative"},
		{"conservative_dynamic_existential_borrow", "dynamic_existential_borrow_carrier_conservative", "TestMemoryIdealV11ProjectsDynamicProtocolWitnessFacts", "dynamic_existential_borrow_conservative"},
		{"valid_static_witness_borrow_fact", "static_witness_parent_fact_validated", "TestMemoryIdealV11ProjectsDynamicProtocolWitnessFacts", "static_witness_borrow_parent_validated"},
		{"invalid_static_witness_missing_parent", "constant-only: no mini_test case", "TestValidateMemoryReportRejectsV11DerivedRowsWithoutParent", "static_witness_parent_fact_required"},
		{"invalid_dynamic_protocol_noalias", "dynamic_protocol_dispatch_noalias_rejected", "TestMemoryIdealV11ProjectsDynamicProtocolWitnessFacts", "dynamic_protocol_noalias_rejected"},
		{"invalid_witness_provenance_promotion", "witness_lookup_unknown_provenance_promotion_rejected", "TestMemoryIdealV11ProjectsDynamicProtocolWitnessFacts", "witness_provenance_promotion_rejected"},
	}
}

func t14RepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func t14GoFiles(t *testing.T) []string {
	t.Helper()
	root := t14RepoRoot(t)
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".cache", ".workflow", "graphify-out":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo Go files: %v", err)
	}
	return files
}
