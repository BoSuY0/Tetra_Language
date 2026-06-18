package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateMemoryCorrelationAcceptsRequiredRows(t *testing.T) {
	path := writeCorrelationFixture(t, validCorrelationMarkdown())
	if err := validateCorrelationFile(path); err != nil {
		t.Fatalf("validateCorrelationFile failed: %v", err)
	}
}

func TestValidateMemoryCorrelationRejectsMissingRequiredRow(t *testing.T) {
	raw := strings.Replace(
		validCorrelationMarkdown(),
		("| MEM-ALIAS-001 | minimal inout exclusivity | fact:alias:" +
			"inout | alias_interval_validator | mem-alias-001 | " +
			"alias_read_during_inout | linux-x64:narrow | validated |\n"),
		"",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-ALIAS-001") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-ALIAS-001", err)
	}
}

func TestValidateMemoryCorrelationRejectsExtraRow(t *testing.T) {
	raw := strings.Replace(
		validCorrelationMarkdown(),
		"| MEM-ALIAS-001 |",
		("| MEM-FUTURE-999 | future row | fact:future | " +
			"future_validator | future-row | future_negative | future | " +
			"future |\n| MEM-ALIAS-001 |"),
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "unexpected requirement_id") {
		t.Fatalf("validateCorrelationFile error = %v, want unexpected row rejection", err)
	}
}

func TestValidateMemoryCorrelationRejectsMissingSourceFact(t *testing.T) {
	raw := strings.Replace(
		validCorrelationMarkdown(),
		"| MEM-REP-001 | safe metadata not user assignable | fact:rep:metadata |",
		"| MEM-REP-001 | safe metadata not user assignable |  |",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "source_fact_id") {
		t.Fatalf("validateCorrelationFile error = %v, want source_fact_id rejection", err)
	}
}

func TestValidateMemoryCorrelationRejectsUnknownStatus(t *testing.T) {
	raw := strings.Replace(
		validCorrelationMarkdown(),
		"| linux-x64:narrow | validated |",
		"| linux-x64:narrow | broad_claim |",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "unknown status") {
		t.Fatalf("validateCorrelationFile error = %v, want unknown status rejection", err)
	}
}

func TestValidateMemoryCorrelationAcceptsV1BorrowRows(t *testing.T) {
	path := writeCorrelationFixture(t, validV1CorrelationMarkdown())
	if err := validateCorrelationFile(path); err != nil {
		t.Fatalf("validateCorrelationFile v1 failed: %v", err)
	}
}

func TestValidateMemoryCorrelationRejectsV1MissingBorrowRow(t *testing.T) {
	raw := strings.Replace(
		validV1CorrelationMarkdown(),
		("| MEM-BORROW-003 | borrowed view through monomorphized " +
			"generic wrapper cannot escape owner | plir:borrowAggregate:" +
			"f_generic_borrow:generic_wrapper_contains_borrow | " +
			"borrow_aggregate_escape_validator | " +
			"generic_wrapper_contains_borrow | " +
			"TestMemoryIdealV1BorrowGenericWrapperGlobalStorageRejected " +
			"| linux-x64:narrow | validated_narrow |\n"),
		"",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-BORROW-003") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-BORROW-003", err)
	}
}

func TestValidateMemoryCorrelationRejectsMixedV0AndV1Rows(t *testing.T) {
	raw := strings.Replace(
		validV1CorrelationMarkdown(),
		"| MEM-BORROW-003 |",
		("| MEM-BORROW-001 | borrowed slice through struct optional " +
			"cannot escape source owner | plir:borrowAggregate:" +
			"f_struct_borrow:aggregate_contains_borrow | " +
			"borrow_aggregate_escape_validator | " +
			"aggregate_contains_borrow | " +
			"TestBorrowedAggregateEscapeDiagnostics | linux-x64:narrow | " +
			"validated |\n| MEM-BORROW-003 |"),
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-BORROW-001") {
		t.Fatalf("validateCorrelationFile error = %v, want mixed row rejection", err)
	}
}

func TestValidateMemoryCorrelationAcceptsV2FunctionCallbackRows(t *testing.T) {
	path := writeCorrelationFixture(t, validV2CorrelationMarkdown())
	if err := validateCorrelationFile(path); err != nil {
		t.Fatalf("validateCorrelationFile v2 failed: %v", err)
	}
}

func TestValidateMemoryCorrelationRejectsV2MissingCallbackRow(t *testing.T) {
	raw := strings.Replace(
		validV2CorrelationMarkdown(),
		("| MEM-BORROW-005 | borrowed view passed through callback " +
			"parameter cannot escape owner | plir:borrowCarrierV2:" +
			"f_callback_arg_borrow:callback_arg_contains_borrow | " +
			"callback_borrow_escape_validator | " +
			"callback_arg_contains_borrow | " +
			"TestMemoryIdealV2BorrowedCallbackGlobalStorageRejected | " +
			"linux-x64:narrow | validated_narrow |\n"),
		"",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-BORROW-005") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-BORROW-005", err)
	}
}

func TestValidateMemoryCorrelationRejectsMixedV1AndV2Rows(t *testing.T) {
	raw := strings.Replace(
		validV2CorrelationMarkdown(),
		"| MEM-ALIAS-002 |",
		("| MEM-BORROW-003 | borrowed view through monomorphized " +
			"generic wrapper cannot escape owner | plir:borrowCarrierV1:" +
			"f_generic_borrow:generic_wrapper_contains_borrow | " +
			"borrow_aggregate_escape_validator | " +
			"generic_wrapper_contains_borrow | " +
			"TestMemoryIdealV1BorrowGenericWrapperGlobalStorageRejected " +
			"| linux-x64:narrow | validated_narrow |\n| MEM-ALIAS-002 |"),
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-BORROW-003") {
		t.Fatalf("validateCorrelationFile error = %v, want mixed v1/v2 row rejection", err)
	}
}

func TestValidateMemoryCorrelationAcceptsV3InterfaceProtocolRows(t *testing.T) {
	path := writeCorrelationFixture(t, validV3CorrelationMarkdown())
	if err := validateCorrelationFile(path); err != nil {
		t.Fatalf("validateCorrelationFile v3 failed: %v", err)
	}
}

func TestValidateMemoryCorrelationRejectsV3MissingProtocolDispatchRow(t *testing.T) {
	raw := strings.Replace(
		validV3CorrelationMarkdown(),
		("| MEM-BORROW-007 | borrowed view passed through dynamic " +
			"dispatch remains conservative unless target is statically " +
			"known | plir:borrowCarrierV3:f_protocol_dispatch_borrow:" +
			"protocol_dispatch_borrow_conservative | " +
			"protocol_dispatch_borrow_validator | " +
			"protocol_dispatch_borrow_conservative | " +
			"TestMemoryIdealV3UnknownDynamicDispatchConservativeRejected " +
			"| linux-x64:narrow | conservative |\n"),
		"",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-BORROW-007") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-BORROW-007", err)
	}
}

func TestValidateMemoryCorrelationRejectsMixedV2AndV3Rows(t *testing.T) {
	raw := strings.Replace(
		validV3CorrelationMarkdown(),
		"| MEM-ALIAS-003 |",
		("| MEM-ALIAS-002 | callback/reentrant inout cannot produce " +
			"broad noalias | plir:borrowCarrierV2:f_callback_inout:" +
			"callback_inout_conservative | " +
			"callback_alias_conservative_validator | " +
			"callback_inout_conservative | " +
			"TestMemoryIdealV2CallbackAliasesInoutArgumentRejected | " +
			"linux-x64:narrow | conservative |\n| MEM-ALIAS-003 |"),
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-ALIAS-002") {
		t.Fatalf("validateCorrelationFile error = %v, want mixed v2/v3 row rejection", err)
	}
}

func TestValidateMemoryCorrelationAcceptsV4AsyncTaskActorRows(t *testing.T) {
	path := writeCorrelationFixture(t, validV4CorrelationMarkdown())
	if err := validateCorrelationFile(path); err != nil {
		t.Fatalf("validateCorrelationFile v4 failed: %v", err)
	}
}

func TestValidateMemoryCorrelationRejectsV4MissingActorBoundaryRow(t *testing.T) {
	raw := strings.Replace(
		validV4CorrelationMarkdown(),
		("| MEM-BORROW-010 | borrowed view cannot cross actor " +
			"boundary without explicit copy | plir:borrowCarrierV4:" +
			"f_actor_boundary_borrow:actor_boundary_borrow_rejected | " +
			"actor_boundary_borrow_validator | " +
			"actor_boundary_borrow_rejected | " +
			"TestMemoryIdealV4BorrowedViewSentToActorRejected | " +
			"linux-x64:narrow | validated_narrow |\n"),
		"",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-BORROW-010") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-BORROW-010", err)
	}
}

func TestValidateMemoryCorrelationRejectsMixedV3AndV4Rows(t *testing.T) {
	raw := strings.Replace(
		validV4CorrelationMarkdown(),
		"| MEM-ALIAS-004 |",
		("| MEM-ALIAS-003 | interface/protocol dispatch cannot " +
			"produce broad noalias | plir:borrowCarrierV3:" +
			"f_protocol_dispatch_noalias:" +
			"protocol_dispatch_noalias_conservative | " +
			"protocol_dispatch_alias_conservative_validator | " +
			"protocol_dispatch_noalias_conservative | " +
			"TestMemoryIdealV3ProtocolDispatchBroadNoAliasRejected | " +
			"linux-x64:narrow | conservative |\n| MEM-ALIAS-004 |"),
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-ALIAS-003") {
		t.Fatalf("validateCorrelationFile error = %v, want mixed v3/v4 row rejection", err)
	}
}

func TestValidateMemoryCorrelationAcceptsV5RawPointerUnsafeRows(t *testing.T) {
	path := writeCorrelationFixture(t, validV5CorrelationMarkdown())
	if err := validateCorrelationFile(path); err != nil {
		t.Fatalf("validateCorrelationFile v5 failed: %v", err)
	}
}

func TestValidateMemoryCorrelationAcceptsV6BoundsRows(t *testing.T) {
	path := writeCorrelationFixture(t, validV6CorrelationMarkdown())
	if err := validateCorrelationFile(path); err != nil {
		t.Fatalf("validateCorrelationFile v6 failed: %v", err)
	}
}

func TestValidateMemoryCorrelationAcceptsV7FFIRows(t *testing.T) {
	path := writeCorrelationFixture(t, validV7CorrelationMarkdown())
	if err := validateCorrelationFile(path); err != nil {
		t.Fatalf("validateCorrelationFile v7 failed: %v", err)
	}
}

func TestValidateMemoryCorrelationAcceptsV8ReportRows(t *testing.T) {
	path := writeCorrelationFixture(t, validV8CorrelationMarkdown())
	if err := validateCorrelationFile(path); err != nil {
		t.Fatalf("validateCorrelationFile v8 failed: %v", err)
	}
}

func TestValidateMemoryCorrelationAcceptsV9StorageRows(t *testing.T) {
	path := writeCorrelationFixture(t, validV9CorrelationMarkdown())
	if err := validateCorrelationFile(path); err != nil {
		t.Fatalf("validateCorrelationFile v9 failed: %v", err)
	}
}

func TestValidateMemoryCorrelationAcceptsV10AsyncCancelRows(t *testing.T) {
	path := writeCorrelationFixture(t, validV10CorrelationMarkdown())
	if err := validateCorrelationFile(path); err != nil {
		t.Fatalf("validateCorrelationFile v10 failed: %v", err)
	}
}

func TestValidateMemoryCorrelationRejectsV10MissingCancellationRow(t *testing.T) {
	raw := strings.Replace(
		validV10CorrelationMarkdown(),
		("| MEM-ASYNC-003 | cancellation path invalidates borrowed " +
			"task-owned lifetime assumptions | memorymodel:asyncV10:" +
			"cancel:borrow_lifetime_invalidated | " +
			"cancellation_lifetime_invalidation_validator | " +
			"cancellation_borrow_lifetime_invalidated | " +
			"TestMiniMemoryModelV10AsyncCancellationStructuredBoundaryCas" +
			"es | linux-x64:narrow | rejected |\n"),
		"",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-ASYNC-003") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-ASYNC-003", err)
	}
}

func TestValidateMemoryCorrelationRejectsV10WidenedBoundaryStatus(t *testing.T) {
	raw := strings.Replace(
		validV10CorrelationMarkdown(),
		"| linux-x64:narrow | conservative |\n",
		"| linux-x64:narrow | validated_narrow |\n",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "widened v10 status") {
		t.Fatalf("validateCorrelationFile error = %v, want widened v10 status rejection", err)
	}
}

func TestValidateMemoryCorrelationAcceptsV11DynamicProtocolRows(t *testing.T) {
	path := writeCorrelationFixture(t, validV11CorrelationMarkdown())
	if err := validateCorrelationFile(path); err != nil {
		t.Fatalf("validateCorrelationFile v11 failed: %v", err)
	}
}

func TestValidateMemoryCorrelationRejectsV11MissingWitnessRow(t *testing.T) {
	raw := strings.Replace(
		validV11CorrelationMarkdown(),
		("| MEM-DYNPROTO-002 | static witness or conformance proof " +
			"may carry borrow facts only with compiler-owned parent fact " +
			"| memorymodel:dynprotoV11:witness:" +
			"static_witness_parent_fact | " +
			"static_witness_parent_fact_validator | " +
			"static_witness_borrow_parent_validated | " +
			"TestMiniMemoryModelV11DynamicProtocolWitnessCases," +
			"TestMemoryIdealV11ProjectsDynamicProtocolWitnessFacts | " +
			"linux-x64:narrow | validated_narrow |\n"),
		"",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-DYNPROTO-002") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-DYNPROTO-002", err)
	}
}

func TestValidateMemoryCorrelationRejectsV11WidenedDynamicRow(t *testing.T) {
	raw := strings.Replace(
		validV11CorrelationMarkdown(),
		"| linux-x64:narrow | conservative |\n",
		"| linux-x64:narrow | validated_narrow |\n",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "widened v11 status") {
		t.Fatalf("validateCorrelationFile error = %v, want widened v11 status rejection", err)
	}
}

func TestValidateMemoryCorrelationRejectsV8MissingClaimDriftRow(t *testing.T) {
	raw := strings.Replace(
		validV8CorrelationMarkdown(),
		("| MEM-REPORT-005 | memory release or audit docs cannot " +
			"claim broad safety from conservative or rejected rows | " +
			"report:v8:claim-drift | memory_claim_drift_validator | " +
			"memory_claim_drift | " +
			"TestValidateMemoryCorrelationRejectsV8BroadSafetyClaimDrift " +
			"| all:docs | rejected |\n"),
		"",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-REPORT-005") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-REPORT-005", err)
	}
}

func TestValidateMemoryCorrelationRejectsV8MissingSafetyMutationRow(t *testing.T) {
	raw := strings.Replace(
		validV8CorrelationMarkdown(),
		("| MEM-REPORT-006 | projection preserves provenance island " +
			"epoch storage noalias and fake normal-build safety fields | " +
			"report:v8:safety-mutation-corpus | " +
			"report_projection_safety_mutation_validator | " +
			"report_projection_safety_field_preservation | " +
			"TestValidateReportProjectionRejectsSafetyFieldMutationCorpus" +
			" | all:report | validated_narrow |\n"),
		"",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-REPORT-006") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-REPORT-006", err)
	}
}

func TestValidateMemoryCorrelationRejectsV8ExtraRow(t *testing.T) {
	raw := strings.Replace(
		validV8CorrelationMarkdown(),
		"| MEM-REPORT-005 |",
		("| MEM-REPORT-999 | future report row | report:v8:future | " +
			"future_validator | future_report_row | future_negative | " +
			"future | future |\n| MEM-REPORT-005 |"),
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "unexpected requirement_id") {
		t.Fatalf("validateCorrelationFile error = %v, want unexpected v8 row rejection", err)
	}
}

func TestValidateMemoryCorrelationRejectsV8BroadSafetyClaimDrift(t *testing.T) {
	raw := strings.Replace(
		validV8CorrelationMarkdown(),
		"memory release or audit docs cannot claim broad safety from conservative or rejected rows",
		"Memory 100% complete and broad safety proven from conservative and rejected rows",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "claim drift") {
		t.Fatalf("validateCorrelationFile error = %v, want claim drift rejection", err)
	}
}

func TestValidateMemoryCorrelationRejectsV9MissingBoundaryRow(t *testing.T) {
	raw := strings.Replace(
		validV9CorrelationMarkdown(),
		("| MEM-STORAGE-004 | async task actor FFI or unknown-call " +
			"escape keeps storage conservative | allocplan:storageV9:" +
			"boundary:boundary_storage_conservative | " +
			"boundary_storage_conservative_validator | " +
			"boundary_storage_conservative | " +
			"TestVerifyPlanRejectsEscapedActualTrustedLowering," +
			"TestMiniMemoryModelV9StorageCases | linux-x64:narrow | " +
			"conservative |\n"),
		"",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-STORAGE-004") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-STORAGE-004", err)
	}
}

func TestValidateMemoryCorrelationRejectsV9ExtraRow(t *testing.T) {
	raw := strings.Replace(
		validV9CorrelationMarkdown(),
		"| MEM-STORAGE-004 |",
		("| MEM-STORAGE-999 | future storage row | allocplan:" +
			"storageV9:future | future_validator | future_storage | " +
			"future_negative | future | future |\n| MEM-STORAGE-004 |"),
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "unexpected requirement_id") {
		t.Fatalf("validateCorrelationFile error = %v, want unexpected v9 row rejection", err)
	}
}

func TestValidateMemoryCorrelationRejectsV9WidenedBoundaryStatus(t *testing.T) {
	raw := strings.Replace(
		validV9CorrelationMarkdown(),
		"| linux-x64:narrow | conservative |\n",
		"| linux-x64:narrow | validated_narrow |\n",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "widened v9 status") {
		t.Fatalf("validateCorrelationFile error = %v, want widened v9 status rejection", err)
	}
}

func TestValidateMemoryCorrelationRejectsV9WrongNegativeTestEvidence(t *testing.T) {
	raw := strings.Replace(
		validV9CorrelationMarkdown(),
		"TestVerifyPlanRejectsTrustedStorageWithoutNoEscapeProof",
		"UnrelatedTest",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "negative_test") ||
		!strings.Contains(err.Error(), "TestVerifyPlanRejectsTrustedStorageWithoutNoEscapeProof") {
		t.Fatalf(
			"validateCorrelationFile error = %v, want missing exact v9 negative test rejection",
			err,
		)
	}
}

func TestValidateMemoryCorrelationRejectsV5MissingStaticContractRow(t *testing.T) {
	raw := strings.Replace(
		validV5CorrelationMarkdown(),
		("| MEM-UNSAFE-004 | unsafe noalias/lifetime/region contracts " +
			"remain static-untrusted unless separately proven | plir:" +
			"rawUnsafeV5:op_static_contract:" +
			"unsafe_contract_static_untrusted | " +
			"unsafe_static_contract_validator | " +
			"unsafe_contract_static_untrusted | " +
			"TestMiniMemoryModelV5RawPointerUnsafeContractCases," +
			"TestValidateMemoryReportRejectsValidatedNoAliasWithUnknownAl" +
			"iasState,TestValidateMemoryReportRejectsBroadNoAliasClaim | " +
			"linux-x64:narrow | conservative |\n"),
		"",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-UNSAFE-004") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-UNSAFE-004", err)
	}
}

func TestValidateMemoryCorrelationRejectsV6MissingRawBoundsRow(t *testing.T) {
	raw := strings.Replace(
		validV6CorrelationMarkdown(),
		("| MEM-BOUNDS-004 | raw bounds target-width or overflow " +
			"uncertainty keeps normal-build check or trap | plir:" +
			"rawBoundsV6:op_raw_width:" +
			"raw_bounds_runtime_check_normal_build | " +
			"raw_bounds_width_validator | " +
			"raw_bounds_runtime_check_normal_build | " +
			"TestMiniMemoryModelV6BoundsProofCases," +
			"TestMemoryIdealV6ProjectsRawBoundsNormalBuildCheck | " +
			"linux-x64:narrow | conservative |\n"),
		"",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-BOUNDS-004") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-BOUNDS-004", err)
	}
}

func TestValidateMemoryCorrelationRejectsV7MissingNoAliasInvalidationRow(t *testing.T) {
	raw := strings.Replace(
		validV7CorrelationMarkdown(),
		("| MEM-FFI-004 | external calls invalidate broad noalias " +
			"unless narrow validator proves otherwise | plir:ffiV7:" +
			"op_ffi:ffi_noalias_invalidated_by_external_call | " +
			"ffi_noalias_conservative_validator | " +
			"ffi_noalias_invalidated_by_external_call | " +
			"TestMiniMemoryModelV7FFICases," +
			"TestValidateMemoryReportRejectsBroadNoAliasClaim | " +
			"linux-x64:narrow | conservative |\n"),
		"",
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-FFI-004") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-FFI-004", err)
	}
}

func TestValidateMemoryCorrelationRejectsMixedV4AndV5Rows(t *testing.T) {
	raw := strings.Replace(
		validV5CorrelationMarkdown(),
		"| MEM-UNSAFE-004 |",
		("| MEM-ALIAS-004 | task/actor boundary cannot produce broad " +
			"noalias | plir:borrowCarrierV4:f_boundary_noalias:" +
			"boundary_noalias_conservative | " +
			"boundary_alias_conservative_validator | " +
			"boundary_noalias_conservative | " +
			"TestMemoryIdealV4TaskActorBroadNoAliasRejected | linux-x64:" +
			"narrow | conservative |\n| MEM-UNSAFE-004 |"),
		1,
	)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-ALIAS-004") {
		t.Fatalf("validateCorrelationFile error = %v, want mixed v4/v5 row rejection", err)
	}
}

func writeCorrelationFixture(t *testing.T, raw string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "correlation.md")
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func validCorrelationMarkdown() string {
	return correlationMarkdown(
		"Memory Ideal Vertical Slice v0 Correlation",
		correlationRow{
			id:           "MEM-REP-001",
			claim:        "safe metadata not user assignable",
			sourceFactID: "fact:rep:metadata",
			validator:    "representation_namespace_validator",
			reportRow:    "mem-rep-001",
			negativeTest: "metadata_assignment_rejected",
			targetLevel:  "all:semantics",
			status:       "validated",
		},
		correlationRow{
			id:           "MEM-BORROW-001",
			claim:        "borrow through struct optional cannot escape owner",
			sourceFactID: "fact:borrow:aggregate",
			validator:    "borrow_aggregate_escape_validator",
			reportRow:    "mem-borrow-001",
			negativeTest: "borrow_aggregate_escape_rejected",
			targetLevel:  "linux-x64:narrow",
			status:       "conservative",
		},
		correlationRow{
			id:           "MEM-ALIAS-001",
			claim:        "minimal inout exclusivity",
			sourceFactID: "fact:alias:inout",
			validator:    "alias_interval_validator",
			reportRow:    "mem-alias-001",
			negativeTest: "alias_read_during_inout",
			targetLevel:  "linux-x64:narrow",
			status:       "validated",
		},
	)
}

func validV1CorrelationMarkdown() string {
	return correlationMarkdown(
		"Memory Ideal Vertical Slice v1 Correlation",
		correlationRow{
			id:           "MEM-BORROW-002",
			claim:        "borrowed view through enum payload cannot escape owner",
			sourceFactID: "plir:borrowAggregate:f_enum_borrow:enum_payload_contains_borrow",
			validator:    "borrow_aggregate_escape_validator",
			reportRow:    "enum_payload_contains_borrow",
			negativeTest: "TestMemoryIdealV1BorrowEnumPayloadGlobalStorageRejected",
			targetLevel:  "linux-x64:narrow",
			status:       "validated_narrow",
		},
		correlationRow{
			id:    "MEM-BORROW-003",
			claim: "borrowed view through monomorphized generic wrapper cannot escape owner",
			sourceFactID: ("plir:borrowAggregate:f_generic_borrow:" +
				"generic_wrapper_contains_borrow"),
			validator:    "borrow_aggregate_escape_validator",
			reportRow:    "generic_wrapper_contains_borrow",
			negativeTest: "TestMemoryIdealV1BorrowGenericWrapperGlobalStorageRejected",
			targetLevel:  "linux-x64:narrow",
			status:       "validated_narrow",
		},
	)
}

func validV2CorrelationMarkdown() string {
	return correlationMarkdown(
		"Memory Ideal Vertical Slice v2 Correlation",
		correlationRow{
			id:    "MEM-BORROW-004",
			claim: "borrowed view passed through function-typed value cannot escape owner",
			sourceFactID: ("plir:borrowCarrierV2:f_function_value_borrow:" +
				"function_value_contains_borrow"),
			validator:    "function_value_borrow_escape_validator",
			reportRow:    "function_value_contains_borrow",
			negativeTest: "TestMemoryIdealV2BorrowedCallbackReturnAsOwnedRejected",
			targetLevel:  "linux-x64:narrow",
			status:       "validated_narrow",
		},
		correlationRow{
			id:           "MEM-BORROW-005",
			claim:        "borrowed view passed through callback parameter cannot escape owner",
			sourceFactID: "plir:borrowCarrierV2:f_callback_arg_borrow:callback_arg_contains_borrow",
			validator:    "callback_borrow_escape_validator",
			reportRow:    "callback_arg_contains_borrow",
			negativeTest: "TestMemoryIdealV2BorrowedCallbackGlobalStorageRejected",
			targetLevel:  "linux-x64:narrow",
			status:       "validated_narrow",
		},
		correlationRow{
			id:           "MEM-ALIAS-002",
			claim:        "callback/reentrant inout cannot produce broad noalias",
			sourceFactID: "plir:borrowCarrierV2:f_callback_inout:callback_inout_conservative",
			validator:    "callback_alias_conservative_validator",
			reportRow:    "callback_inout_conservative",
			negativeTest: "TestMemoryIdealV2CallbackAliasesInoutArgumentRejected",
			targetLevel:  "linux-x64:narrow",
			status:       "conservative",
		},
	)
}

func validV3CorrelationMarkdown() string {
	return correlationMarkdown(
		"Memory Ideal Vertical Slice v3 Correlation",
		correlationRow{
			id:           "MEM-BORROW-006",
			claim:        "borrowed view through interface/protocol value cannot escape owner",
			sourceFactID: "plir:borrowCarrierV3:f_interface_borrow:interface_value_contains_borrow",
			validator:    "interface_borrow_escape_validator",
			reportRow:    "interface_value_contains_borrow",
			negativeTest: "TestMemoryIdealV3BorrowedInterfaceReturnAsOwnedRejected",
			targetLevel:  "linux-x64:narrow",
			status:       "validated_narrow",
		},
		correlationRow{
			id: "MEM-BORROW-007",
			claim: ("borrowed view passed through dynamic dispatch remains conservative " +
				"unless target is statically known"),
			sourceFactID: ("plir:borrowCarrierV3:f_protocol_dispatch_borrow:" +
				"protocol_dispatch_borrow_conservative"),
			validator:    "protocol_dispatch_borrow_validator",
			reportRow:    "protocol_dispatch_borrow_conservative",
			negativeTest: "TestMemoryIdealV3UnknownDynamicDispatchConservativeRejected",
			targetLevel:  "linux-x64:narrow",
			status:       "conservative",
		},
		correlationRow{
			id:    "MEM-ALIAS-003",
			claim: "interface/protocol dispatch cannot produce broad noalias",
			sourceFactID: ("plir:borrowCarrierV3:f_protocol_dispatch_noalias:" +
				"protocol_dispatch_noalias_conservative"),
			validator:    "protocol_dispatch_alias_conservative_validator",
			reportRow:    "protocol_dispatch_noalias_conservative",
			negativeTest: "TestMemoryIdealV3ProtocolDispatchBroadNoAliasRejected",
			targetLevel:  "linux-x64:narrow",
			status:       "conservative",
		},
	)
}

func validV4CorrelationMarkdown() string {
	return correlationMarkdown(
		"Memory Ideal Vertical Slice v4 Correlation",
		correlationRow{
			id: "MEM-BORROW-008",
			claim: ("borrowed view cannot cross async/await suspension boundary unless " +
				"proven local and non-escaping"),
			sourceFactID: ("plir:borrowCarrierV4:f_async_boundary_borrow:" +
				"async_boundary_borrow_conservative"),
			validator:    "async_boundary_borrow_validator",
			reportRow:    "async_boundary_borrow_conservative",
			negativeTest: "TestMemoryIdealV4BorrowedAsyncResultRejected",
			targetLevel:  "linux-x64:narrow",
			status:       "conservative",
		},
		correlationRow{
			id:    "MEM-BORROW-009",
			claim: "borrowed view cannot cross task boundary without explicit copy",
			sourceFactID: ("plir:borrowCarrierV4:f_task_boundary_borrow:" +
				"task_boundary_borrow_rejected"),
			validator:    "task_boundary_borrow_validator",
			reportRow:    "task_boundary_borrow_rejected",
			negativeTest: "TestMemoryIdealV4BorrowedViewSentToTaskRejected",
			targetLevel:  "linux-x64:narrow",
			status:       "validated_narrow",
		},
		correlationRow{
			id:    "MEM-BORROW-010",
			claim: "borrowed view cannot cross actor boundary without explicit copy",
			sourceFactID: ("plir:borrowCarrierV4:f_actor_boundary_borrow:" +
				"actor_boundary_borrow_rejected"),
			validator:    "actor_boundary_borrow_validator",
			reportRow:    "actor_boundary_borrow_rejected",
			negativeTest: "TestMemoryIdealV4BorrowedViewSentToActorRejected",
			targetLevel:  "linux-x64:narrow",
			status:       "validated_narrow",
		},
		correlationRow{
			id:           "MEM-ALIAS-004",
			claim:        "task/actor boundary cannot produce broad noalias",
			sourceFactID: "plir:borrowCarrierV4:f_boundary_noalias:boundary_noalias_conservative",
			validator:    "boundary_alias_conservative_validator",
			reportRow:    "boundary_noalias_conservative",
			negativeTest: "TestMemoryIdealV4TaskActorBroadNoAliasRejected",
			targetLevel:  "linux-x64:narrow",
			status:       "conservative",
		},
	)
}

func validV5CorrelationMarkdown() string {
	return correlationMarkdown(
		"Memory Ideal Vertical Slice v5 Correlation",
		correlationRow{
			id:    "MEM-UNSAFE-001",
			claim: "unsafe_unknown raw pointer cannot produce safe_known/provenance_known/noalias facts",
			sourceFactID: ("plir:rawUnsafeV5:op_ptr_unknown:unsafe:" +
				"unsafe_unknown_rejected_safe_facts"),
			validator: "unsafe_unknown_fact_validator",
			reportRow: "unsafe_unknown_rejected_safe_facts",
			negativeTest: ("TestMiniMemoryModelV5RawPointerUnsafeContractCases," +
				"TestValidateMemoryReportRejectsSafeKnownFromUnsafeUnknown," +
				"TestValidateMemoryReportRejectsUnsafeUnknownOptimizationClaims"),
			targetLevel: "linux-x64:narrow",
			status:      "rejected",
		},
		correlationRow{
			id: "MEM-UNSAFE-002",
			claim: ("unsafe_verified_root from core.alloc_bytes may produce bounded " +
				"allocation-base facts"),
			sourceFactID: "allocplan:rawUnsafeV5:p:unsafe_verified_root_allocation_base",
			validator:    "unsafe_verified_root_bounds_validator",
			reportRow:    "unsafe_verified_root_allocation_base",
			negativeTest: ("TestMiniMemoryModelV5RawPointerUnsafeContractCases," +
				"TestMemoryIdealV5ProjectsRawPointerUnsafeContractFacts"),
			targetLevel: "linux-x64:narrow",
			status:      "validated_narrow",
		},
		correlationRow{
			id:    "MEM-UNSAFE-003",
			claim: "runtime-checkable unsafe contracts may validate nonnull/alignment/length only",
			sourceFactID: ("plir:rawUnsafeV5:op_runtime_contract:" +
				"unsafe_contract_runtime_checkable"),
			validator: "unsafe_runtime_contract_validator",
			reportRow: "unsafe_contract_runtime_checkable",
			negativeTest: ("TestMiniMemoryModelV5RawPointerUnsafeContractCases," +
				"TestMemoryIdealV5ProjectsRawPointerUnsafeContractFacts"),
			targetLevel: "linux-x64:narrow",
			status:      "validated_narrow",
		},
		correlationRow{
			id: "MEM-UNSAFE-004",
			claim: ("unsafe noalias/lifetime/region contracts remain static-untrusted " +
				"unless separately proven"),
			sourceFactID: ("plir:rawUnsafeV5:op_static_contract:" +
				"unsafe_contract_static_untrusted"),
			validator: "unsafe_static_contract_validator",
			reportRow: "unsafe_contract_static_untrusted",
			negativeTest: ("TestMiniMemoryModelV5RawPointerUnsafeContractCases," +
				"TestValidateMemoryReportRejectsValidatedNoAliasWithUnknownAliasState," +
				"TestValidateMemoryReportRejectsBroadNoAliasClaim"),
			targetLevel: "linux-x64:narrow",
			status:      "conservative",
		},
	)
}

func validV6CorrelationMarkdown() string {
	return correlationMarkdown(
		"Memory Ideal Vertical Slice v6 Bounds Correlation",
		correlationRow{
			id: "MEM-BOUNDS-001",
			claim: ("retained dynamic bounds checks remain normal-build checks when no " +
				"proof exists"),
			sourceFactID: "validation:boundsV6:sum:retained_dynamic",
			validator:    "normal_build_bounds_check_validator",
			reportRow:    "bounds_check_retained_dynamic",
			negativeTest: ("TestMiniMemoryModelV6BoundsProofCases," +
				"TestValidateMemoryReportRejectsDynamicOptimizationClaimWithoutNormalBuildCheck"),
			targetLevel: "linux-x64:narrow",
			status:      "validated_narrow",
		},
		correlationRow{
			id:           "MEM-BOUNDS-002",
			claim:        "removed bounds check requires compiler-owned proof id",
			sourceFactID: "validation:boundsV6:sum:removed:bounds_check_removed_with_proof_id",
			validator:    "bounds_proof_id_validator",
			reportRow:    "bounds_check_removed_with_proof_id",
			negativeTest: ("TestCheckBoundsProofsRejectsRemovedCheckWithoutProofID," +
				"TestCheckBoundsProofsWithPLIRRejectsUnknownLiveProof"),
			targetLevel: "linux-x64:narrow",
			status:      "validated_narrow",
		},
		correlationRow{
			id:           "MEM-BOUNDS-003",
			claim:        "unsafe_unknown cannot authorize eliminated bounds checks",
			sourceFactID: "validation:boundsV6:unsafe:bounds_check_removal_rejected_missing_proof_id",
			validator:    "bounds_proof_id_validator",
			reportRow:    "bounds_check_removal_rejected_missing_proof_id",
			negativeTest: ("TestMiniMemoryModelV6BoundsProofCases," +
				"TestValidateMemoryReportRejectsUnsafeUnknownOptimizationClaims"),
			targetLevel: "linux-x64:narrow",
			status:      "rejected",
		},
		correlationRow{
			id:           "MEM-BOUNDS-004",
			claim:        "raw bounds target-width or overflow uncertainty keeps normal-build check or trap",
			sourceFactID: "plir:rawBoundsV6:op_raw_width:raw_bounds_runtime_check_normal_build",
			validator:    "raw_bounds_width_validator",
			reportRow:    "raw_bounds_runtime_check_normal_build",
			negativeTest: ("TestMiniMemoryModelV6BoundsProofCases," +
				"TestMemoryIdealV6ProjectsRawBoundsNormalBuildCheck"),
			targetLevel: "linux-x64:narrow",
			status:      "conservative",
		},
	)
}

func validV7CorrelationMarkdown() string {
	return correlationMarkdown(
		"Memory Ideal Vertical Slice v7 FFI Correlation",
		correlationRow{
			id: "MEM-FFI-001",
			claim: ("external pointers remain unsafe_unknown or external_unknown unless " +
				"compiler-owned provenance exists"),
			sourceFactID: "plir:ffiV7:f_external_unknown:ffi_pointer_external_unknown",
			validator:    "external_pointer_provenance_validator",
			reportRow:    "ffi_pointer_external_unknown",
			negativeTest: ("TestMemoryIdealV7ProjectsFFICallExternalFacts," +
				"TestValidateMemoryReportRejectsUnsafeUnknownProvenanceKnownClaim"),
			targetLevel: "linux-x64:narrow",
			status:      "conservative",
		},
		correlationRow{
			id: "MEM-FFI-002",
			claim: ("external calls may retain borrowed pointers unless compiler-owned " +
				"contract proves otherwise"),
			sourceFactID: "plir:ffiV7:op_ffi:ffi_call_may_retain_borrow",
			validator:    "ffi_lifetime_conservative_validator",
			reportRow:    "ffi_call_may_retain_borrow",
			negativeTest: ("TestMiniMemoryModelV7FFICases," +
				"TestMemoryIdealV7ProjectsFFICallExternalFacts"),
			targetLevel: "linux-x64:narrow",
			status:      "conservative",
		},
		correlationRow{
			id: "MEM-FFI-003",
			claim: ("safe wrapper promotion from raw or external pointer rejects " +
				"without compiler-owned proof"),
			sourceFactID: ("plir:ffiV7:op_safe_wrapper:" +
				"safe_wrapper_promotion_rejected_without_contract"),
			validator: "safe_wrapper_promotion_validator",
			reportRow: "safe_wrapper_promotion_rejected_without_contract",
			negativeTest: ("TestMiniMemoryModelV7FFICases," +
				"TestValidateMemoryReportRejectsSafeKnownFromUnsafeUnknown"),
			targetLevel: "linux-x64:narrow",
			status:      "rejected",
		},
		correlationRow{
			id:           "MEM-FFI-004",
			claim:        "external calls invalidate broad noalias unless narrow validator proves otherwise",
			sourceFactID: "plir:ffiV7:op_ffi:ffi_noalias_invalidated_by_external_call",
			validator:    "ffi_noalias_conservative_validator",
			reportRow:    "ffi_noalias_invalidated_by_external_call",
			negativeTest: ("TestMiniMemoryModelV7FFICases," +
				"TestValidateMemoryReportRejectsBroadNoAliasClaim"),
			targetLevel: "linux-x64:narrow",
			status:      "conservative",
		},
	)
}

func validV8CorrelationMarkdown() string {
	return correlationMarkdown(
		"Memory Ideal Vertical Slice v8 Report Integrity Correlation",
		correlationRow{
			id:           "MEM-REPORT-001",
			claim:        "every report row maps to a MemoryFactGraph source fact",
			sourceFactID: "report:v8:source-map",
			validator:    "report_graph_projection_validator",
			reportRow:    "report_graph_projection",
			negativeTest: "TestValidateReportProjectionRejectsUnknownSourceFactID",
			targetLevel:  "all:report",
			status:       "validated_narrow",
		},
		correlationRow{
			id:           "MEM-REPORT-002",
			claim:        "every graph fact requiring projection appears in the report",
			sourceFactID: "report:v8:projection-complete",
			validator:    "report_projection_completeness_validator",
			reportRow:    "report_projection_completeness",
			negativeTest: "TestValidateReportProjectionRejectsMissingProjectedGraphFact",
			targetLevel:  "all:report",
			status:       "validated_narrow",
		},
		correlationRow{
			id:           "MEM-REPORT-003",
			claim:        "report projection preserves cost_class and normal_build_check",
			sourceFactID: "report:v8:projection-fields",
			validator:    "cost_class_preservation_validator,normal_build_check_preservation_validator",
			reportRow:    "report_projection_field_preservation",
			negativeTest: ("TestValidateReportProjectionRejectsAlteredCostClass," +
				"TestValidateReportProjectionRejectsDroppedNormalBuildCheck"),
			targetLevel: "all:report",
			status:      "validated_narrow",
		},
		correlationRow{
			id:           "MEM-REPORT-004",
			claim:        "correlation docs reject extra missing or widened rows",
			sourceFactID: "report:v8:correlation-exact",
			validator:    "correlation_exact_row_validator",
			reportRow:    "correlation_exact_row_set",
			negativeTest: ("TestValidateMemoryCorrelationRejectsV8MissingClaimDriftRow," +
				"TestValidateMemoryCorrelationRejectsV8ExtraRow"),
			targetLevel: "all:docs",
			status:      "validated_narrow",
		},
		correlationRow{
			id: "MEM-REPORT-005",
			claim: ("memory release or audit docs cannot claim broad safety from " +
				"conservative or rejected rows"),
			sourceFactID: "report:v8:claim-drift",
			validator:    "memory_claim_drift_validator",
			reportRow:    "memory_claim_drift",
			negativeTest: "TestValidateMemoryCorrelationRejectsV8BroadSafetyClaimDrift",
			targetLevel:  "all:docs",
			status:       "rejected",
		},
		correlationRow{
			id: "MEM-REPORT-006",
			claim: ("projection preserves provenance island epoch storage noalias and " +
				"fake normal-build safety fields"),
			sourceFactID: "report:v8:safety-mutation-corpus",
			validator:    "report_projection_safety_mutation_validator",
			reportRow:    "report_projection_safety_field_preservation",
			negativeTest: "TestValidateReportProjectionRejectsSafetyFieldMutationCorpus",
			targetLevel:  "all:report",
			status:       "validated_narrow",
		},
	)
}

func validV9CorrelationMarkdown() string {
	return correlationMarkdown(
		"Memory Ideal Vertical Slice v9 Storage Correlation",
		correlationRow{
			id:           "MEM-STORAGE-001",
			claim:        "escaped value cannot lower as trusted stack region task actor or island storage",
			sourceFactID: "allocplan:storageV9:escape:storage_escape_rejected",
			validator:    "storage_escape_validator",
			reportRow:    "storage_escape_rejected",
			negativeTest: "TestVerifyPlanRejectsEscapedActualTrustedLowering",
			targetLevel:  "linux-x64:narrow",
			status:       "rejected",
		},
		correlationRow{
			id:           "MEM-STORAGE-002",
			claim:        "trusted stack or region storage requires compiler-owned no-escape proof",
			sourceFactID: "allocplan:storageV9:noescape:trusted_storage_requires_no_escape_proof",
			validator:    "storage_no_escape_proof_validator",
			reportRow:    "trusted_storage_requires_no_escape_proof",
			negativeTest: "TestVerifyPlanRejectsTrustedStorageWithoutNoEscapeProof",
			targetLevel:  "linux-x64:narrow",
			status:       "validated_narrow",
		},
		correlationRow{
			id:           "MEM-STORAGE-003",
			claim:        "heap or conservative fallback preserves source_fact_id and reason",
			sourceFactID: "allocplan:storageV9:fallback:heap_fallback_reason_preserved",
			validator:    "heap_fallback_reason_validator",
			reportRow:    "heap_fallback_reason_preserved",
			negativeTest: ("TestFromPLIRAndAllocPlanRejectsHeapFallbackWithoutReason," +
				"TestValidateMemoryReportRejectsHeapFallbackWithoutReason"),
			targetLevel: "linux-x64:narrow",
			status:      "validated_narrow",
		},
		correlationRow{
			id:           "MEM-STORAGE-004",
			claim:        "async task actor FFI or unknown-call escape keeps storage conservative",
			sourceFactID: "allocplan:storageV9:boundary:boundary_storage_conservative",
			validator:    "boundary_storage_conservative_validator",
			reportRow:    "boundary_storage_conservative",
			negativeTest: ("TestVerifyPlanRejectsEscapedActualTrustedLowering," +
				"TestMiniMemoryModelV9StorageCases"),
			targetLevel: "linux-x64:narrow",
			status:      "conservative",
		},
	)
}

func validV10CorrelationMarkdown() string {
	return correlationMarkdown(
		"Memory Ideal Vertical Slice v10 Async Cancellation Correlation",
		correlationRow{
			id: "MEM-ASYNC-001",
			claim: ("borrowed value may be used only before suspension when proven " +
				"local and non-escaping"),
			sourceFactID: "memorymodel:asyncV10:preawait:local_borrow_before_suspension",
			validator:    "pre_await_local_borrow_validator",
			reportRow:    "pre_await_local_borrow_validated",
			negativeTest: "TestMiniMemoryModelV10AsyncCancellationStructuredBoundaryCases",
			targetLevel:  "linux-x64:narrow",
			status:       "validated_narrow",
		},
		correlationRow{
			id:           "MEM-ASYNC-002",
			claim:        "borrowed value crossing await or suspend remains conservative or rejected",
			sourceFactID: "memorymodel:asyncV10:postawait:borrow_after_suspension_conservative",
			validator:    "post_await_borrow_conservative_validator",
			reportRow:    "post_await_borrow_conservative",
			negativeTest: "TestMiniMemoryModelV10AsyncCancellationStructuredBoundaryCases",
			targetLevel:  "linux-x64:narrow",
			status:       "conservative",
		},
		correlationRow{
			id:           "MEM-ASYNC-003",
			claim:        "cancellation path invalidates borrowed task-owned lifetime assumptions",
			sourceFactID: "memorymodel:asyncV10:cancel:borrow_lifetime_invalidated",
			validator:    "cancellation_lifetime_invalidation_validator",
			reportRow:    "cancellation_borrow_lifetime_invalidated",
			negativeTest: "TestMiniMemoryModelV10AsyncCancellationStructuredBoundaryCases",
			targetLevel:  "linux-x64:narrow",
			status:       "rejected",
		},
		correlationRow{
			id:           "MEM-ASYNC-004",
			claim:        "task group structured concurrency boundary cannot validate broad noalias",
			sourceFactID: "memorymodel:asyncV10:taskgroup:task_group_noalias_conservative",
			validator:    "task_group_boundary_conservative_validator",
			reportRow:    "task_group_noalias_conservative",
			negativeTest: ("TestMiniMemoryModelV10AsyncCancellationStructuredBoundaryCases," +
				"TestValidateMemoryReportRejectsBroadNoAliasClaim"),
			targetLevel: "linux-x64:narrow",
			status:      "conservative",
		},
		correlationRow{
			id: "MEM-ASYNC-005",
			claim: ("actor reentrant callback boundary keeps borrow and storage conservative " +
				"unless separately proven"),
			sourceFactID: "memorymodel:asyncV10:actor:actor_reentrant_callback_conservative",
			validator:    "actor_reentrant_callback_boundary_validator",
			reportRow:    "actor_reentrant_callback_conservative",
			negativeTest: "TestMiniMemoryModelV10AsyncCancellationStructuredBoundaryCases",
			targetLevel:  "linux-x64:narrow",
			status:       "conservative",
		},
	)
}

func validV11CorrelationMarkdown() string {
	return correlationMarkdown(
		"Memory Ideal Vertical Slice v11 Dynamic Protocol Correlation",
		correlationRow{
			id: "MEM-DYNPROTO-001",
			claim: ("dynamic existential or protocol borrow carriers remain conservative " +
				"unless statically resolved"),
			sourceFactID: "memorymodel:dynprotoV11:dynamic:existential_borrow_conservative",
			validator:    "dynamic_existential_borrow_conservative_validator",
			reportRow:    "dynamic_existential_borrow_conservative",
			negativeTest: ("TestMiniMemoryModelV11DynamicProtocolWitnessCases," +
				"TestMemoryIdealV11ProjectsDynamicProtocolWitnessFacts"),
			targetLevel: "linux-x64:narrow",
			status:      "conservative",
		},
		correlationRow{
			id: "MEM-DYNPROTO-002",
			claim: ("static witness or conformance proof may carry borrow facts only " +
				"with compiler-owned parent fact"),
			sourceFactID: "memorymodel:dynprotoV11:witness:static_witness_parent_fact",
			validator:    "static_witness_parent_fact_validator",
			reportRow:    "static_witness_borrow_parent_validated",
			negativeTest: ("TestMiniMemoryModelV11DynamicProtocolWitnessCases," +
				"TestMemoryIdealV11ProjectsDynamicProtocolWitnessFacts"),
			targetLevel: "linux-x64:narrow",
			status:      "validated_narrow",
		},
		correlationRow{
			id:           "MEM-DYNPROTO-003",
			claim:        "dynamic protocol dispatch cannot validate broad noalias",
			sourceFactID: "memorymodel:dynprotoV11:dispatch:dynamic_protocol_noalias_rejected",
			validator:    "dynamic_protocol_noalias_rejection_validator",
			reportRow:    "dynamic_protocol_noalias_rejected",
			negativeTest: ("TestMiniMemoryModelV11DynamicProtocolWitnessCases," +
				"TestValidateMemoryReportRejectsBroadNoAliasClaim"),
			targetLevel: "linux-x64:narrow",
			status:      "rejected",
		},
		correlationRow{
			id: "MEM-DYNPROTO-004",
			claim: ("witness or conformance table lookup cannot promote unsafe dynamic " +
				"unknown provenance to safe_known"),
			sourceFactID: "memorymodel:dynprotoV11:witness:witness_provenance_promotion_rejected",
			validator:    "witness_provenance_promotion_validator",
			reportRow:    "witness_provenance_promotion_rejected",
			negativeTest: ("TestMiniMemoryModelV11DynamicProtocolWitnessCases," +
				"TestValidateMemoryReportRejectsSafeKnownFromUnsafeUnknown"),
			targetLevel: "linux-x64:narrow",
			status:      "rejected",
		},
		correlationRow{
			id: "MEM-DYNPROTO-005",
			claim: ("protocol or existential dispatch report rows preserve source_fact_id " +
				"cost_class and normal_build_check"),
			sourceFactID: "report:v11:dynproto:protocol_dispatch_report_integrity",
			validator:    "protocol_dispatch_report_integrity_validator",
			reportRow:    "protocol_dispatch_report_integrity",
			negativeTest: ("TestMemoryIdealV11ProjectsDynamicProtocolWitnessFacts," +
				"TestValidateReportProjectionRejectsAlteredCostClass," +
				"TestValidateReportProjectionRejectsDroppedNormalBuildCheck"),
			targetLevel: "linux-x64:narrow",
			status:      "validated_narrow",
		},
	)
}

type correlationRow struct {
	id           string
	claim        string
	sourceFactID string
	validator    string
	reportRow    string
	negativeTest string
	targetLevel  string
	status       string
}

func correlationMarkdown(title string, rows ...correlationRow) string {
	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	builder.WriteString(
		"| requirement_id | claim | source_fact_id | validator | report_row | " +
			"negative_test | target_level | status |\n",
	)
	builder.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, row := range rows {
		builder.WriteString("| ")
		builder.WriteString(strings.Join([]string{
			row.id,
			row.claim,
			row.sourceFactID,
			row.validator,
			row.reportRow,
			row.negativeTest,
			row.targetLevel,
			row.status,
		}, " | "))
		builder.WriteString(" |\n")
	}
	return builder.String()
}
