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
	raw := strings.Replace(validCorrelationMarkdown(), "| MEM-ALIAS-001 | minimal inout exclusivity | fact:alias:inout | alias_interval_validator | mem-alias-001 | alias_read_during_inout | linux-x64:narrow | validated |\n", "", 1)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-ALIAS-001") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-ALIAS-001", err)
	}
}

func TestValidateMemoryCorrelationRejectsExtraRow(t *testing.T) {
	raw := strings.Replace(validCorrelationMarkdown(), "| MEM-ALIAS-001 |", "| MEM-FUTURE-999 | future row | fact:future | future_validator | future-row | future_negative | future | future |\n| MEM-ALIAS-001 |", 1)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "unexpected requirement_id") {
		t.Fatalf("validateCorrelationFile error = %v, want unexpected row rejection", err)
	}
}

func TestValidateMemoryCorrelationRejectsMissingSourceFact(t *testing.T) {
	raw := strings.Replace(validCorrelationMarkdown(), "| MEM-REP-001 | safe metadata not user assignable | fact:rep:metadata |", "| MEM-REP-001 | safe metadata not user assignable |  |", 1)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "source_fact_id") {
		t.Fatalf("validateCorrelationFile error = %v, want source_fact_id rejection", err)
	}
}

func TestValidateMemoryCorrelationRejectsUnknownStatus(t *testing.T) {
	raw := strings.Replace(validCorrelationMarkdown(), "| linux-x64:narrow | validated |", "| linux-x64:narrow | broad_claim |", 1)
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
	raw := strings.Replace(validV1CorrelationMarkdown(), "| MEM-BORROW-003 | borrowed view through monomorphized generic wrapper cannot escape owner | plir:borrowAggregate:f_generic_borrow:generic_wrapper_contains_borrow | borrow_aggregate_escape_validator | generic_wrapper_contains_borrow | TestMemoryIdealV1BorrowGenericWrapperGlobalStorageRejected | linux-x64:narrow | validated_narrow |\n", "", 1)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-BORROW-003") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-BORROW-003", err)
	}
}

func TestValidateMemoryCorrelationRejectsMixedV0AndV1Rows(t *testing.T) {
	raw := strings.Replace(validV1CorrelationMarkdown(), "| MEM-BORROW-003 |", "| MEM-BORROW-001 | borrowed slice through struct optional cannot escape source owner | plir:borrowAggregate:f_struct_borrow:aggregate_contains_borrow | borrow_aggregate_escape_validator | aggregate_contains_borrow | TestBorrowedAggregateEscapeDiagnostics | linux-x64:narrow | validated |\n| MEM-BORROW-003 |", 1)
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
	raw := strings.Replace(validV2CorrelationMarkdown(), "| MEM-BORROW-005 | borrowed view passed through callback parameter cannot escape owner | plir:borrowCarrierV2:f_callback_arg_borrow:callback_arg_contains_borrow | callback_borrow_escape_validator | callback_arg_contains_borrow | TestMemoryIdealV2BorrowedCallbackGlobalStorageRejected | linux-x64:narrow | validated_narrow |\n", "", 1)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-BORROW-005") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-BORROW-005", err)
	}
}

func TestValidateMemoryCorrelationRejectsMixedV1AndV2Rows(t *testing.T) {
	raw := strings.Replace(validV2CorrelationMarkdown(), "| MEM-ALIAS-002 |", "| MEM-BORROW-003 | borrowed view through monomorphized generic wrapper cannot escape owner | plir:borrowCarrierV1:f_generic_borrow:generic_wrapper_contains_borrow | borrow_aggregate_escape_validator | generic_wrapper_contains_borrow | TestMemoryIdealV1BorrowGenericWrapperGlobalStorageRejected | linux-x64:narrow | validated_narrow |\n| MEM-ALIAS-002 |", 1)
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
	raw := strings.Replace(validV3CorrelationMarkdown(), "| MEM-BORROW-007 | borrowed view passed through dynamic dispatch remains conservative unless target is statically known | plir:borrowCarrierV3:f_protocol_dispatch_borrow:protocol_dispatch_borrow_conservative | protocol_dispatch_borrow_validator | protocol_dispatch_borrow_conservative | TestMemoryIdealV3UnknownDynamicDispatchConservativeRejected | linux-x64:narrow | conservative |\n", "", 1)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-BORROW-007") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-BORROW-007", err)
	}
}

func TestValidateMemoryCorrelationRejectsMixedV2AndV3Rows(t *testing.T) {
	raw := strings.Replace(validV3CorrelationMarkdown(), "| MEM-ALIAS-003 |", "| MEM-ALIAS-002 | callback/reentrant inout cannot produce broad noalias | plir:borrowCarrierV2:f_callback_inout:callback_inout_conservative | callback_alias_conservative_validator | callback_inout_conservative | TestMemoryIdealV2CallbackAliasesInoutArgumentRejected | linux-x64:narrow | conservative |\n| MEM-ALIAS-003 |", 1)
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
	raw := strings.Replace(validV4CorrelationMarkdown(), "| MEM-BORROW-010 | borrowed view cannot cross actor boundary without explicit copy | plir:borrowCarrierV4:f_actor_boundary_borrow:actor_boundary_borrow_rejected | actor_boundary_borrow_validator | actor_boundary_borrow_rejected | TestMemoryIdealV4BorrowedViewSentToActorRejected | linux-x64:narrow | validated_narrow |\n", "", 1)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-BORROW-010") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-BORROW-010", err)
	}
}

func TestValidateMemoryCorrelationRejectsMixedV3AndV4Rows(t *testing.T) {
	raw := strings.Replace(validV4CorrelationMarkdown(), "| MEM-ALIAS-004 |", "| MEM-ALIAS-003 | interface/protocol dispatch cannot produce broad noalias | plir:borrowCarrierV3:f_protocol_dispatch_noalias:protocol_dispatch_noalias_conservative | protocol_dispatch_alias_conservative_validator | protocol_dispatch_noalias_conservative | TestMemoryIdealV3ProtocolDispatchBroadNoAliasRejected | linux-x64:narrow | conservative |\n| MEM-ALIAS-004 |", 1)
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
	raw := strings.Replace(validV10CorrelationMarkdown(), "| MEM-ASYNC-003 | cancellation path invalidates borrowed task-owned lifetime assumptions | memorymodel:asyncV10:cancel:borrow_lifetime_invalidated | cancellation_lifetime_invalidation_validator | cancellation_borrow_lifetime_invalidated | TestMiniMemoryModelV10AsyncCancellationStructuredBoundaryCases | linux-x64:narrow | rejected |\n", "", 1)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-ASYNC-003") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-ASYNC-003", err)
	}
}

func TestValidateMemoryCorrelationRejectsV10WidenedBoundaryStatus(t *testing.T) {
	raw := strings.Replace(validV10CorrelationMarkdown(), "| linux-x64:narrow | conservative |\n", "| linux-x64:narrow | validated_narrow |\n", 1)
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
	raw := strings.Replace(validV11CorrelationMarkdown(), "| MEM-DYNPROTO-002 | static witness or conformance proof may carry borrow facts only with compiler-owned parent fact | memorymodel:dynprotoV11:witness:static_witness_parent_fact | static_witness_parent_fact_validator | static_witness_borrow_parent_validated | TestMiniMemoryModelV11DynamicProtocolWitnessCases,TestMemoryIdealV11ProjectsDynamicProtocolWitnessFacts | linux-x64:narrow | validated_narrow |\n", "", 1)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-DYNPROTO-002") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-DYNPROTO-002", err)
	}
}

func TestValidateMemoryCorrelationRejectsV11WidenedDynamicRow(t *testing.T) {
	raw := strings.Replace(validV11CorrelationMarkdown(), "| linux-x64:narrow | conservative |\n", "| linux-x64:narrow | validated_narrow |\n", 1)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "widened v11 status") {
		t.Fatalf("validateCorrelationFile error = %v, want widened v11 status rejection", err)
	}
}

func TestValidateMemoryCorrelationRejectsV8MissingClaimDriftRow(t *testing.T) {
	raw := strings.Replace(validV8CorrelationMarkdown(), "| MEM-REPORT-005 | memory release or audit docs cannot claim broad safety from conservative or rejected rows | report:v8:claim-drift | memory_claim_drift_validator | memory_claim_drift | TestValidateMemoryCorrelationRejectsV8BroadSafetyClaimDrift | all:docs | rejected |\n", "", 1)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-REPORT-005") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-REPORT-005", err)
	}
}

func TestValidateMemoryCorrelationRejectsV8ExtraRow(t *testing.T) {
	raw := strings.Replace(validV8CorrelationMarkdown(), "| MEM-REPORT-005 |", "| MEM-REPORT-999 | future report row | report:v8:future | future_validator | future_report_row | future_negative | future | future |\n| MEM-REPORT-005 |", 1)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "unexpected requirement_id") {
		t.Fatalf("validateCorrelationFile error = %v, want unexpected v8 row rejection", err)
	}
}

func TestValidateMemoryCorrelationRejectsV8BroadSafetyClaimDrift(t *testing.T) {
	raw := strings.Replace(validV8CorrelationMarkdown(), "memory release or audit docs cannot claim broad safety from conservative or rejected rows", "Memory 100% complete and broad safety proven from conservative and rejected rows", 1)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "claim drift") {
		t.Fatalf("validateCorrelationFile error = %v, want claim drift rejection", err)
	}
}

func TestValidateMemoryCorrelationRejectsV9MissingBoundaryRow(t *testing.T) {
	raw := strings.Replace(validV9CorrelationMarkdown(), "| MEM-STORAGE-004 | async task actor FFI or unknown-call escape keeps storage conservative | allocplan:storageV9:boundary:boundary_storage_conservative | boundary_storage_conservative_validator | boundary_storage_conservative | TestVerifyPlanRejectsEscapedActualTrustedLowering,TestMiniMemoryModelV9StorageCases | linux-x64:narrow | conservative |\n", "", 1)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-STORAGE-004") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-STORAGE-004", err)
	}
}

func TestValidateMemoryCorrelationRejectsV9ExtraRow(t *testing.T) {
	raw := strings.Replace(validV9CorrelationMarkdown(), "| MEM-STORAGE-004 |", "| MEM-STORAGE-999 | future storage row | allocplan:storageV9:future | future_validator | future_storage | future_negative | future | future |\n| MEM-STORAGE-004 |", 1)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "unexpected requirement_id") {
		t.Fatalf("validateCorrelationFile error = %v, want unexpected v9 row rejection", err)
	}
}

func TestValidateMemoryCorrelationRejectsV9WidenedBoundaryStatus(t *testing.T) {
	raw := strings.Replace(validV9CorrelationMarkdown(), "| linux-x64:narrow | conservative |\n", "| linux-x64:narrow | validated_narrow |\n", 1)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "widened v9 status") {
		t.Fatalf("validateCorrelationFile error = %v, want widened v9 status rejection", err)
	}
}

func TestValidateMemoryCorrelationRejectsV5MissingStaticContractRow(t *testing.T) {
	raw := strings.Replace(validV5CorrelationMarkdown(), "| MEM-UNSAFE-004 | unsafe noalias/lifetime/region contracts remain static-untrusted unless separately proven | plir:rawUnsafeV5:op_static_contract:unsafe_contract_static_untrusted | unsafe_static_contract_validator | unsafe_contract_static_untrusted | TestMiniMemoryModelV5RawPointerUnsafeContractCases,TestValidateMemoryReportRejectsValidatedNoAliasWithUnknownAliasState,TestValidateMemoryReportRejectsBroadNoAliasClaim | linux-x64:narrow | conservative |\n", "", 1)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-UNSAFE-004") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-UNSAFE-004", err)
	}
}

func TestValidateMemoryCorrelationRejectsV6MissingRawBoundsRow(t *testing.T) {
	raw := strings.Replace(validV6CorrelationMarkdown(), "| MEM-BOUNDS-004 | raw bounds target-width or overflow uncertainty keeps normal-build check or trap | plir:rawBoundsV6:op_raw_width:raw_bounds_runtime_check_normal_build | raw_bounds_width_validator | raw_bounds_runtime_check_normal_build | TestMiniMemoryModelV6BoundsProofCases,TestMemoryIdealV6ProjectsRawBoundsNormalBuildCheck | linux-x64:narrow | conservative |\n", "", 1)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-BOUNDS-004") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-BOUNDS-004", err)
	}
}

func TestValidateMemoryCorrelationRejectsV7MissingNoAliasInvalidationRow(t *testing.T) {
	raw := strings.Replace(validV7CorrelationMarkdown(), "| MEM-FFI-004 | external calls invalidate broad noalias unless narrow validator proves otherwise | plir:ffiV7:op_ffi:ffi_noalias_invalidated_by_external_call | ffi_noalias_conservative_validator | ffi_noalias_invalidated_by_external_call | TestMiniMemoryModelV7FFICases,TestValidateMemoryReportRejectsBroadNoAliasClaim | linux-x64:narrow | conservative |\n", "", 1)
	path := writeCorrelationFixture(t, raw)
	err := validateCorrelationFile(path)
	if err == nil || !strings.Contains(err.Error(), "MEM-FFI-004") {
		t.Fatalf("validateCorrelationFile error = %v, want missing MEM-FFI-004", err)
	}
}

func TestValidateMemoryCorrelationRejectsMixedV4AndV5Rows(t *testing.T) {
	raw := strings.Replace(validV5CorrelationMarkdown(), "| MEM-UNSAFE-004 |", "| MEM-ALIAS-004 | task/actor boundary cannot produce broad noalias | plir:borrowCarrierV4:f_boundary_noalias:boundary_noalias_conservative | boundary_alias_conservative_validator | boundary_noalias_conservative | TestMemoryIdealV4TaskActorBroadNoAliasRejected | linux-x64:narrow | conservative |\n| MEM-UNSAFE-004 |", 1)
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
	return `# Memory Ideal Vertical Slice v0 Correlation

| requirement_id | claim | source_fact_id | validator | report_row | negative_test | target_level | status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| MEM-REP-001 | safe metadata not user assignable | fact:rep:metadata | representation_namespace_validator | mem-rep-001 | metadata_assignment_rejected | all:semantics | validated |
| MEM-BORROW-001 | borrow through struct optional cannot escape owner | fact:borrow:aggregate | borrow_aggregate_escape_validator | mem-borrow-001 | borrow_aggregate_escape_rejected | linux-x64:narrow | conservative |
| MEM-ALIAS-001 | minimal inout exclusivity | fact:alias:inout | alias_interval_validator | mem-alias-001 | alias_read_during_inout | linux-x64:narrow | validated |
`
}

func validV1CorrelationMarkdown() string {
	return `# Memory Ideal Vertical Slice v1 Correlation

| requirement_id | claim | source_fact_id | validator | report_row | negative_test | target_level | status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| MEM-BORROW-002 | borrowed view through enum payload cannot escape owner | plir:borrowAggregate:f_enum_borrow:enum_payload_contains_borrow | borrow_aggregate_escape_validator | enum_payload_contains_borrow | TestMemoryIdealV1BorrowEnumPayloadGlobalStorageRejected | linux-x64:narrow | validated_narrow |
| MEM-BORROW-003 | borrowed view through monomorphized generic wrapper cannot escape owner | plir:borrowAggregate:f_generic_borrow:generic_wrapper_contains_borrow | borrow_aggregate_escape_validator | generic_wrapper_contains_borrow | TestMemoryIdealV1BorrowGenericWrapperGlobalStorageRejected | linux-x64:narrow | validated_narrow |
`
}

func validV2CorrelationMarkdown() string {
	return `# Memory Ideal Vertical Slice v2 Correlation

| requirement_id | claim | source_fact_id | validator | report_row | negative_test | target_level | status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| MEM-BORROW-004 | borrowed view passed through function-typed value cannot escape owner | plir:borrowCarrierV2:f_function_value_borrow:function_value_contains_borrow | function_value_borrow_escape_validator | function_value_contains_borrow | TestMemoryIdealV2BorrowedCallbackReturnAsOwnedRejected | linux-x64:narrow | validated_narrow |
| MEM-BORROW-005 | borrowed view passed through callback parameter cannot escape owner | plir:borrowCarrierV2:f_callback_arg_borrow:callback_arg_contains_borrow | callback_borrow_escape_validator | callback_arg_contains_borrow | TestMemoryIdealV2BorrowedCallbackGlobalStorageRejected | linux-x64:narrow | validated_narrow |
| MEM-ALIAS-002 | callback/reentrant inout cannot produce broad noalias | plir:borrowCarrierV2:f_callback_inout:callback_inout_conservative | callback_alias_conservative_validator | callback_inout_conservative | TestMemoryIdealV2CallbackAliasesInoutArgumentRejected | linux-x64:narrow | conservative |
`
}

func validV3CorrelationMarkdown() string {
	return `# Memory Ideal Vertical Slice v3 Correlation

| requirement_id | claim | source_fact_id | validator | report_row | negative_test | target_level | status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| MEM-BORROW-006 | borrowed view through interface/protocol value cannot escape owner | plir:borrowCarrierV3:f_interface_borrow:interface_value_contains_borrow | interface_borrow_escape_validator | interface_value_contains_borrow | TestMemoryIdealV3BorrowedInterfaceReturnAsOwnedRejected | linux-x64:narrow | validated_narrow |
| MEM-BORROW-007 | borrowed view passed through dynamic dispatch remains conservative unless target is statically known | plir:borrowCarrierV3:f_protocol_dispatch_borrow:protocol_dispatch_borrow_conservative | protocol_dispatch_borrow_validator | protocol_dispatch_borrow_conservative | TestMemoryIdealV3UnknownDynamicDispatchConservativeRejected | linux-x64:narrow | conservative |
| MEM-ALIAS-003 | interface/protocol dispatch cannot produce broad noalias | plir:borrowCarrierV3:f_protocol_dispatch_noalias:protocol_dispatch_noalias_conservative | protocol_dispatch_alias_conservative_validator | protocol_dispatch_noalias_conservative | TestMemoryIdealV3ProtocolDispatchBroadNoAliasRejected | linux-x64:narrow | conservative |
`
}

func validV4CorrelationMarkdown() string {
	return `# Memory Ideal Vertical Slice v4 Correlation

| requirement_id | claim | source_fact_id | validator | report_row | negative_test | target_level | status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| MEM-BORROW-008 | borrowed view cannot cross async/await suspension boundary unless proven local and non-escaping | plir:borrowCarrierV4:f_async_boundary_borrow:async_boundary_borrow_conservative | async_boundary_borrow_validator | async_boundary_borrow_conservative | TestMemoryIdealV4BorrowedAsyncResultRejected | linux-x64:narrow | conservative |
| MEM-BORROW-009 | borrowed view cannot cross task boundary without explicit copy | plir:borrowCarrierV4:f_task_boundary_borrow:task_boundary_borrow_rejected | task_boundary_borrow_validator | task_boundary_borrow_rejected | TestMemoryIdealV4BorrowedViewSentToTaskRejected | linux-x64:narrow | validated_narrow |
| MEM-BORROW-010 | borrowed view cannot cross actor boundary without explicit copy | plir:borrowCarrierV4:f_actor_boundary_borrow:actor_boundary_borrow_rejected | actor_boundary_borrow_validator | actor_boundary_borrow_rejected | TestMemoryIdealV4BorrowedViewSentToActorRejected | linux-x64:narrow | validated_narrow |
| MEM-ALIAS-004 | task/actor boundary cannot produce broad noalias | plir:borrowCarrierV4:f_boundary_noalias:boundary_noalias_conservative | boundary_alias_conservative_validator | boundary_noalias_conservative | TestMemoryIdealV4TaskActorBroadNoAliasRejected | linux-x64:narrow | conservative |
`
}

func validV5CorrelationMarkdown() string {
	return `# Memory Ideal Vertical Slice v5 Correlation

| requirement_id | claim | source_fact_id | validator | report_row | negative_test | target_level | status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| MEM-UNSAFE-001 | unsafe_unknown raw pointer cannot produce safe_known/provenance_known/noalias facts | plir:rawUnsafeV5:op_ptr_unknown:unsafe:unsafe_unknown_rejected_safe_facts | unsafe_unknown_fact_validator | unsafe_unknown_rejected_safe_facts | TestMiniMemoryModelV5RawPointerUnsafeContractCases,TestValidateMemoryReportRejectsSafeKnownFromUnsafeUnknown,TestValidateMemoryReportRejectsUnsafeUnknownOptimizationClaims | linux-x64:narrow | rejected |
| MEM-UNSAFE-002 | unsafe_verified_root from core.alloc_bytes may produce bounded allocation-base facts | allocplan:rawUnsafeV5:p:unsafe_verified_root_allocation_base | unsafe_verified_root_bounds_validator | unsafe_verified_root_allocation_base | TestMiniMemoryModelV5RawPointerUnsafeContractCases,TestMemoryIdealV5ProjectsRawPointerUnsafeContractFacts | linux-x64:narrow | validated_narrow |
| MEM-UNSAFE-003 | runtime-checkable unsafe contracts may validate nonnull/alignment/length only | plir:rawUnsafeV5:op_runtime_contract:unsafe_contract_runtime_checkable | unsafe_runtime_contract_validator | unsafe_contract_runtime_checkable | TestMiniMemoryModelV5RawPointerUnsafeContractCases,TestMemoryIdealV5ProjectsRawPointerUnsafeContractFacts | linux-x64:narrow | validated_narrow |
| MEM-UNSAFE-004 | unsafe noalias/lifetime/region contracts remain static-untrusted unless separately proven | plir:rawUnsafeV5:op_static_contract:unsafe_contract_static_untrusted | unsafe_static_contract_validator | unsafe_contract_static_untrusted | TestMiniMemoryModelV5RawPointerUnsafeContractCases,TestValidateMemoryReportRejectsValidatedNoAliasWithUnknownAliasState,TestValidateMemoryReportRejectsBroadNoAliasClaim | linux-x64:narrow | conservative |
`
}

func validV6CorrelationMarkdown() string {
	return `# Memory Ideal Vertical Slice v6 Bounds Correlation

| requirement_id | claim | source_fact_id | validator | report_row | negative_test | target_level | status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| MEM-BOUNDS-001 | retained dynamic bounds checks remain normal-build checks when no proof exists | validation:boundsV6:sum:retained_dynamic | normal_build_bounds_check_validator | bounds_check_retained_dynamic | TestMiniMemoryModelV6BoundsProofCases,TestValidateMemoryReportRejectsDynamicOptimizationClaimWithoutNormalBuildCheck | linux-x64:narrow | validated_narrow |
| MEM-BOUNDS-002 | removed bounds check requires compiler-owned proof id | validation:boundsV6:sum:removed:bounds_check_removed_with_proof_id | bounds_proof_id_validator | bounds_check_removed_with_proof_id | TestCheckBoundsProofsRejectsRemovedCheckWithoutProofID,TestCheckBoundsProofsWithPLIRRejectsUnknownLiveProof | linux-x64:narrow | validated_narrow |
| MEM-BOUNDS-003 | unsafe_unknown cannot authorize eliminated bounds checks | validation:boundsV6:unsafe:bounds_check_removal_rejected_missing_proof_id | bounds_proof_id_validator | bounds_check_removal_rejected_missing_proof_id | TestMiniMemoryModelV6BoundsProofCases,TestValidateMemoryReportRejectsUnsafeUnknownOptimizationClaims | linux-x64:narrow | rejected |
| MEM-BOUNDS-004 | raw bounds target-width or overflow uncertainty keeps normal-build check or trap | plir:rawBoundsV6:op_raw_width:raw_bounds_runtime_check_normal_build | raw_bounds_width_validator | raw_bounds_runtime_check_normal_build | TestMiniMemoryModelV6BoundsProofCases,TestMemoryIdealV6ProjectsRawBoundsNormalBuildCheck | linux-x64:narrow | conservative |
`
}

func validV7CorrelationMarkdown() string {
	return `# Memory Ideal Vertical Slice v7 FFI Correlation

| requirement_id | claim | source_fact_id | validator | report_row | negative_test | target_level | status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| MEM-FFI-001 | external pointers remain unsafe_unknown or external_unknown unless compiler-owned provenance exists | plir:ffiV7:f_external_unknown:ffi_pointer_external_unknown | external_pointer_provenance_validator | ffi_pointer_external_unknown | TestMemoryIdealV7ProjectsFFICallExternalFacts,TestValidateMemoryReportRejectsUnsafeUnknownProvenanceKnownClaim | linux-x64:narrow | conservative |
| MEM-FFI-002 | external calls may retain borrowed pointers unless compiler-owned contract proves otherwise | plir:ffiV7:op_ffi:ffi_call_may_retain_borrow | ffi_lifetime_conservative_validator | ffi_call_may_retain_borrow | TestMiniMemoryModelV7FFICases,TestMemoryIdealV7ProjectsFFICallExternalFacts | linux-x64:narrow | conservative |
| MEM-FFI-003 | safe wrapper promotion from raw or external pointer rejects without compiler-owned proof | plir:ffiV7:op_safe_wrapper:safe_wrapper_promotion_rejected_without_contract | safe_wrapper_promotion_validator | safe_wrapper_promotion_rejected_without_contract | TestMiniMemoryModelV7FFICases,TestValidateMemoryReportRejectsSafeKnownFromUnsafeUnknown | linux-x64:narrow | rejected |
| MEM-FFI-004 | external calls invalidate broad noalias unless narrow validator proves otherwise | plir:ffiV7:op_ffi:ffi_noalias_invalidated_by_external_call | ffi_noalias_conservative_validator | ffi_noalias_invalidated_by_external_call | TestMiniMemoryModelV7FFICases,TestValidateMemoryReportRejectsBroadNoAliasClaim | linux-x64:narrow | conservative |
`
}

func validV8CorrelationMarkdown() string {
	return `# Memory Ideal Vertical Slice v8 Report Integrity Correlation

| requirement_id | claim | source_fact_id | validator | report_row | negative_test | target_level | status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| MEM-REPORT-001 | every report row maps to a MemoryFactGraph source fact | report:v8:source-map | report_graph_projection_validator | report_graph_projection | TestValidateReportProjectionRejectsUnknownSourceFactID | all:report | validated_narrow |
| MEM-REPORT-002 | every graph fact requiring projection appears in the report | report:v8:projection-complete | report_projection_completeness_validator | report_projection_completeness | TestValidateReportProjectionRejectsMissingProjectedGraphFact | all:report | validated_narrow |
| MEM-REPORT-003 | report projection preserves cost_class and normal_build_check | report:v8:projection-fields | cost_class_preservation_validator,normal_build_check_preservation_validator | report_projection_field_preservation | TestValidateReportProjectionRejectsAlteredCostClass,TestValidateReportProjectionRejectsDroppedNormalBuildCheck | all:report | validated_narrow |
| MEM-REPORT-004 | correlation docs reject extra missing or widened rows | report:v8:correlation-exact | correlation_exact_row_validator | correlation_exact_row_set | TestValidateMemoryCorrelationRejectsV8MissingClaimDriftRow,TestValidateMemoryCorrelationRejectsV8ExtraRow | all:docs | validated_narrow |
| MEM-REPORT-005 | memory release or audit docs cannot claim broad safety from conservative or rejected rows | report:v8:claim-drift | memory_claim_drift_validator | memory_claim_drift | TestValidateMemoryCorrelationRejectsV8BroadSafetyClaimDrift | all:docs | rejected |
`
}

func validV9CorrelationMarkdown() string {
	return `# Memory Ideal Vertical Slice v9 Storage Correlation

| requirement_id | claim | source_fact_id | validator | report_row | negative_test | target_level | status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| MEM-STORAGE-001 | escaped value cannot lower as trusted stack region task actor or island storage | allocplan:storageV9:escape:storage_escape_rejected | storage_escape_validator | storage_escape_rejected | TestVerifyPlanRejectsEscapedActualTrustedLowering | linux-x64:narrow | rejected |
| MEM-STORAGE-002 | trusted stack or region storage requires compiler-owned no-escape proof | allocplan:storageV9:noescape:trusted_storage_requires_no_escape_proof | storage_no_escape_proof_validator | trusted_storage_requires_no_escape_proof | TestVerifyPlanRejectsTrustedStorageWithoutNoEscapeProof | linux-x64:narrow | validated_narrow |
| MEM-STORAGE-003 | heap or conservative fallback preserves source_fact_id and reason | allocplan:storageV9:fallback:heap_fallback_reason_preserved | heap_fallback_reason_validator | heap_fallback_reason_preserved | TestFromPLIRAndAllocPlanRejectsHeapFallbackWithoutReason,TestValidateMemoryReportRejectsHeapFallbackWithoutReason | linux-x64:narrow | validated_narrow |
| MEM-STORAGE-004 | async task actor FFI or unknown-call escape keeps storage conservative | allocplan:storageV9:boundary:boundary_storage_conservative | boundary_storage_conservative_validator | boundary_storage_conservative | TestVerifyPlanRejectsEscapedActualTrustedLowering,TestMiniMemoryModelV9StorageCases | linux-x64:narrow | conservative |
`
}

func validV10CorrelationMarkdown() string {
	return `# Memory Ideal Vertical Slice v10 Async Cancellation Correlation

| requirement_id | claim | source_fact_id | validator | report_row | negative_test | target_level | status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| MEM-ASYNC-001 | borrowed value may be used only before suspension when proven local and non-escaping | memorymodel:asyncV10:preawait:local_borrow_before_suspension | pre_await_local_borrow_validator | pre_await_local_borrow_validated | TestMiniMemoryModelV10AsyncCancellationStructuredBoundaryCases | linux-x64:narrow | validated_narrow |
| MEM-ASYNC-002 | borrowed value crossing await or suspend remains conservative or rejected | memorymodel:asyncV10:postawait:borrow_after_suspension_conservative | post_await_borrow_conservative_validator | post_await_borrow_conservative | TestMiniMemoryModelV10AsyncCancellationStructuredBoundaryCases | linux-x64:narrow | conservative |
| MEM-ASYNC-003 | cancellation path invalidates borrowed task-owned lifetime assumptions | memorymodel:asyncV10:cancel:borrow_lifetime_invalidated | cancellation_lifetime_invalidation_validator | cancellation_borrow_lifetime_invalidated | TestMiniMemoryModelV10AsyncCancellationStructuredBoundaryCases | linux-x64:narrow | rejected |
| MEM-ASYNC-004 | task group structured concurrency boundary cannot validate broad noalias | memorymodel:asyncV10:taskgroup:task_group_noalias_conservative | task_group_boundary_conservative_validator | task_group_noalias_conservative | TestMiniMemoryModelV10AsyncCancellationStructuredBoundaryCases,TestValidateMemoryReportRejectsBroadNoAliasClaim | linux-x64:narrow | conservative |
| MEM-ASYNC-005 | actor reentrant callback boundary keeps borrow and storage conservative unless separately proven | memorymodel:asyncV10:actor:actor_reentrant_callback_conservative | actor_reentrant_callback_boundary_validator | actor_reentrant_callback_conservative | TestMiniMemoryModelV10AsyncCancellationStructuredBoundaryCases | linux-x64:narrow | conservative |
`
}

func validV11CorrelationMarkdown() string {
	return `# Memory Ideal Vertical Slice v11 Dynamic Protocol Correlation

| requirement_id | claim | source_fact_id | validator | report_row | negative_test | target_level | status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| MEM-DYNPROTO-001 | dynamic existential or protocol borrow carriers remain conservative unless statically resolved | memorymodel:dynprotoV11:dynamic:existential_borrow_conservative | dynamic_existential_borrow_conservative_validator | dynamic_existential_borrow_conservative | TestMiniMemoryModelV11DynamicProtocolWitnessCases,TestMemoryIdealV11ProjectsDynamicProtocolWitnessFacts | linux-x64:narrow | conservative |
| MEM-DYNPROTO-002 | static witness or conformance proof may carry borrow facts only with compiler-owned parent fact | memorymodel:dynprotoV11:witness:static_witness_parent_fact | static_witness_parent_fact_validator | static_witness_borrow_parent_validated | TestMiniMemoryModelV11DynamicProtocolWitnessCases,TestMemoryIdealV11ProjectsDynamicProtocolWitnessFacts | linux-x64:narrow | validated_narrow |
| MEM-DYNPROTO-003 | dynamic protocol dispatch cannot validate broad noalias | memorymodel:dynprotoV11:dispatch:dynamic_protocol_noalias_rejected | dynamic_protocol_noalias_rejection_validator | dynamic_protocol_noalias_rejected | TestMiniMemoryModelV11DynamicProtocolWitnessCases,TestValidateMemoryReportRejectsBroadNoAliasClaim | linux-x64:narrow | rejected |
| MEM-DYNPROTO-004 | witness or conformance table lookup cannot promote unsafe dynamic unknown provenance to safe_known | memorymodel:dynprotoV11:witness:witness_provenance_promotion_rejected | witness_provenance_promotion_validator | witness_provenance_promotion_rejected | TestMiniMemoryModelV11DynamicProtocolWitnessCases,TestValidateMemoryReportRejectsSafeKnownFromUnsafeUnknown | linux-x64:narrow | rejected |
| MEM-DYNPROTO-005 | protocol or existential dispatch report rows preserve source_fact_id cost_class and normal_build_check | report:v11:dynproto:protocol_dispatch_report_integrity | protocol_dispatch_report_integrity_validator | protocol_dispatch_report_integrity | TestMemoryIdealV11ProjectsDynamicProtocolWitnessFacts,TestValidateReportProjectionRejectsAlteredCostClass,TestValidateReportProjectionRejectsDroppedNormalBuildCheck | linux-x64:narrow | validated_narrow |
`
}
