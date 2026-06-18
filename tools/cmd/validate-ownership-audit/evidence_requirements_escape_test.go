package main

import (
	"strings"
	"testing"
)

func TestValidateOwnershipAuditRejectsMissingNestedSliceEnumPayloadEscapeEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module nested slice enum-payload return/inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence", "nested slice enum payload escape", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing nested slice enum-payload escape evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module nested slice enum-payload return/inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want nested slice enum-payload escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingNestedSliceStructEscapeEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module nested slice struct return/inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence", "nested slice struct escape", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing nested slice struct escape evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module nested slice struct return/inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want nested slice struct escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingOptionalMixedAggregateBranchEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "optional mixed safe/provenance aggregate branch merges", "optional aggregate branch merges", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing optional mixed aggregate branch evidence failure")
	}
	if !strings.Contains(err.Error(), "optional mixed safe/provenance aggregate branch merges") {
		t.Fatalf("error = %v, want optional mixed aggregate branch evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingMixedAggregateBranchAndMatchEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "mixed safe/provenance aggregate branch and match returns", "mixed aggregate returns", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing mixed aggregate branch/match evidence failure")
	}
	if !strings.Contains(err.Error(), "mixed safe/provenance aggregate branch and match returns") {
		t.Fatalf("error = %v, want mixed aggregate branch/match evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingInterproceduralPerFieldRegionSummaryEvidence(t *testing.T) {
	want := "same-module and interface-only cross-module per-field interprocedural region summaries for aggregate returns from multiple island parameters, including optional aggregate wrappers, enum payload wrappers, branch aggregate wrappers, match aggregate wrappers, if-let aggregate wrappers"
	audit := strings.Replace(validBlockedOwnershipAudit(), want, "interprocedural aggregate summaries", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing interprocedural per-field region summary evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want interprocedural per-field region summary evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingInterproceduralResourceSummaryEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "Local return-resource summaries, typed-error throw-resource summaries including rethrow-through-`try`", "Local resource summaries", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing interprocedural resource summary evidence failure")
	}
	if !strings.Contains(err.Error(), "Local return-resource summaries, typed-error throw-resource summaries including rethrow-through-`try`") {
		t.Fatalf("error = %v, want interprocedural resource summary evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingInterproceduralGeneratedT4IResourceProvenanceEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "generated `.t4i` direct/local/aggregate-local-alias/aggregate-field-access/aggregate-field-local-alias resource return, assignment/let/direct-if-let/direct-match/field-local/if-let/match optional and nested/field-local nested optional resource return, typed-error direct/field-local-alias throw, and rethrow-through-`try` direct/field-local-alias provenance stubs", "generated interprocedural provenance stubs", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing interprocedural generated .t4i resource provenance evidence failure")
	}
	if !strings.Contains(err.Error(), "generated `.t4i` direct/local/aggregate-local-alias/aggregate-field-access/aggregate-field-local-alias resource return, assignment/let/direct-if-let/direct-match/field-local/if-let/match optional and nested/field-local nested optional resource return, typed-error direct/field-local-alias throw, and rethrow-through-`try` direct/field-local-alias provenance stubs") {
		t.Fatalf("error = %v, want interprocedural generated .t4i resource provenance evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingSelectedTransitiveInterproceduralResourceCasesEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "selected same-module/cross-module transitive interprocedural resource cases, including task-handle, task-group, island, struct-field, enum-payload, enum-constructor return, same-module throw/catch enum-payload, if-let/match optional-payload, and nested struct/enum optional-payload return resource aliases", "selected resource cases", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing selected transitive interprocedural resource cases evidence failure")
	}
	if !strings.Contains(err.Error(), "selected same-module/cross-module transitive interprocedural resource cases, including task-handle, task-group, island, struct-field, enum-payload, enum-constructor return, same-module throw/catch enum-payload, if-let/match optional-payload, and nested struct/enum optional-payload return resource aliases") {
		t.Fatalf("error = %v, want selected transitive interprocedural resource cases evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingEnumWholeValueGlobalAssignmentEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module ptr-containing enum whole-value global assignment with stable TETRA2102 JSON diagnostic evidence", "enum global assignment", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing enum whole-value global assignment evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module ptr-containing enum whole-value global assignment with stable TETRA2102 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want enum whole-value global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingGlobalFieldTargetAssignmentEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module global field target assignment with stable TETRA2102 JSON diagnostic evidence", "global field assignment", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing global field target assignment evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module global field target assignment with stable TETRA2102 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want global field target assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingAggregateNestedGlobalFieldEscapeEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module aggregate and nested-aggregate global field escapes with stable TETRA2102 JSON diagnostic evidence", "nested global field escapes", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing aggregate/nested-aggregate global field escape evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module aggregate and nested-aggregate global field escapes with stable TETRA2102 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want aggregate/nested-aggregate global field escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingPtrContainingNestedAggregateCallTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module ptr-containing/nested aggregate owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence", "ptr aggregate call rejections", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing ptr-containing/nested aggregate call TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module ptr-containing/nested aggregate owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want ptr-containing/nested aggregate call TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingPtrEnumPayloadCallTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module ptr enum-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence", "ptr enum-payload call rejections", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing ptr enum-payload call TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module ptr enum-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want ptr enum-payload call TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingPtrOptionalPayloadCallTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module ptr optional-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence", "ptr optional-payload call rejections", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing ptr optional-payload call TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module ptr optional-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want ptr optional-payload call TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingSliceOptionalPayloadCallTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module slice optional-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence", "slice optional-payload call rejections", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing slice optional-payload call TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module slice optional-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want slice optional-payload call TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingImportedDirectPtrAggregateCallTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "imported direct ptr-containing/nested aggregate owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence", "imported direct ptr aggregate call rejections", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing imported direct ptr-containing/nested aggregate call TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "imported direct ptr-containing/nested aggregate owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want imported direct ptr-containing/nested aggregate call TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingFunctionTypedSliceAggregateCallbackTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "function-typed value/struct-field/enum-payload callback slice-containing struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence", "function-typed slice callback rejections", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing function-typed slice aggregate callback TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "function-typed value/struct-field/enum-payload callback slice-containing struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want function-typed slice aggregate callback TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingFunctionTypedOptionalPtrCallbackTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module function-typed value/struct-field/enum-payload optional-ptr owned/consume/inout callback diagnostics with stable TETRA2101 CLI JSON evidence", "function-typed optional-ptr callback diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing function-typed optional-ptr callback TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module function-typed value/struct-field/enum-payload optional-ptr owned/consume/inout callback diagnostics with stable TETRA2101 CLI JSON evidence") {
		t.Fatalf("error = %v, want function-typed optional-ptr callback TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingGenericAggregateOptionalPtrTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module generic aggregate and optional-ptr owned/consume/inout instantiations including slice-containing struct/enum aggregate instantiations with stable TETRA2101 CLI JSON evidence", "generic aggregate optional-ptr diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing generic aggregate optional-ptr TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module generic aggregate and optional-ptr owned/consume/inout instantiations including slice-containing struct/enum aggregate instantiations with stable TETRA2101 CLI JSON evidence") {
		t.Fatalf("error = %v, want generic aggregate optional-ptr TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingGenericBorrowReturnTETRA2102Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module generic borrow-aggregate/optional-ptr return diagnostics with stable TETRA2102 CLI JSON evidence", "generic borrow-return diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing generic borrow-return TETRA2102 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module generic borrow-aggregate/optional-ptr return diagnostics with stable TETRA2102 CLI JSON evidence") {
		t.Fatalf("error = %v, want generic borrow-return TETRA2102 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingGenericFunctionTypedGlobalOwnershipMarkerEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "generic function-typed global consume-marker preservation and ownership mismatch diagnostics", "generic function markers", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing generic function-typed global ownership-marker evidence failure")
	}
	if !strings.Contains(err.Error(), "generic function-typed global consume-marker preservation and ownership mismatch diagnostics") {
		t.Fatalf("error = %v, want generic function-typed global ownership-marker evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingGenericResourceAliasTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module monomorphized generic struct task-handle/task-group/island resource aliases with stable TETRA2101 CLI JSON evidence", "generic resource aliases", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing generic resource alias TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module monomorphized generic struct task-handle/task-group/island resource aliases with stable TETRA2101 CLI JSON evidence") {
		t.Fatalf("error = %v, want generic resource alias TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingGenericActorAndResourceAliasTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module monomorphized generic struct actor consume alias diagnostics plus same-module/cross-module monomorphized generic struct task-handle/task-group/island resource aliases with stable TETRA2101 CLI JSON evidence", "generic actor/resource aliases", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing generic actor/resource alias TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module monomorphized generic struct actor consume alias diagnostics plus same-module/cross-module monomorphized generic struct task-handle/task-group/island resource aliases with stable TETRA2101 CLI JSON evidence") {
		t.Fatalf("error = %v, want generic actor/resource alias TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingAliasRowGenericActorResourceAliasEvidence(t *testing.T) {
	want := "same-module/cross-module monomorphized generic struct actor consume aliases, same-module/cross-module monomorphized generic struct task-handle/task-group/island resource aliases with stable TETRA2101 CLI JSON evidence"
	audit := strings.Replace(validBlockedOwnershipAudit(), want, "generic actor/resource aliases", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing alias-row generic actor/resource alias evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want alias-row generic actor/resource alias evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingTransitiveResourceAliasTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module transitive interprocedural task-handle/task-group/island resource aliases with stable TETRA2101 CLI JSON evidence", "transitive resource aliases", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing transitive resource alias TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module transitive interprocedural task-handle/task-group/island resource aliases with stable TETRA2101 CLI JSON evidence") {
		t.Fatalf("error = %v, want transitive resource alias TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingEnumConstructorReturnResourceAliasTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module enum-constructor return resource aliases with stable TETRA2101 CLI JSON evidence", "enum-constructor resource aliases", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing enum-constructor return resource alias TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module enum-constructor return resource aliases with stable TETRA2101 CLI JSON evidence") {
		t.Fatalf("error = %v, want enum-constructor return resource alias TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingGenericProtocolRequirementOwnershipMismatchTETRA2001Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module generic protocol requirement parameter ownership mismatch diagnostics with stable TETRA2001 JSON diagnostic evidence", "generic protocol ownership mismatch diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing generic protocol requirement ownership mismatch TETRA2001 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module generic protocol requirement parameter ownership mismatch diagnostics with stable TETRA2001 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want generic protocol requirement ownership mismatch TETRA2001 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingProtocolImplOwnershipMismatchTETRA2001Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module protocol impl parameter ownership mismatch diagnostics with stable TETRA2001 CLI JSON evidence", "protocol impl ownership mismatch diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing protocol impl ownership mismatch TETRA2001 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module protocol impl parameter ownership mismatch diagnostics with stable TETRA2001 CLI JSON evidence") {
		t.Fatalf("error = %v, want protocol impl ownership mismatch TETRA2001 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingProtocolParameterOwnershipMatchingEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module protocol parameter ownership matching plus ", "", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing protocol parameter ownership matching evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module protocol parameter ownership matching plus same-module/cross-module protocol impl parameter ownership mismatch diagnostics with stable TETRA2001 CLI JSON evidence") {
		t.Fatalf("error = %v, want protocol parameter ownership matching evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingT4IFunctionTypedLocalAliasGlobalStorageEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "generated `.t4i` function-typed parameter local-alias return metadata for interface-only global-storage diagnostics", "generated callable metadata", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing .t4i function-typed local-alias global-storage evidence failure")
	}
	if !strings.Contains(err.Error(), "generated `.t4i` function-typed parameter local-alias return metadata for interface-only global-storage diagnostics") {
		t.Fatalf("error = %v, want .t4i function-typed local-alias global-storage evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingFunctionTypeOwnershipMarkerEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "function type ownership markers parse/format plus function-typed callable ownership-marker diagnostics", "callable ownership metadata", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing function type ownership-marker evidence failure")
	}
	if !strings.Contains(err.Error(), "function type ownership markers parse/format plus function-typed callable ownership-marker diagnostics") {
		t.Fatalf("error = %v, want function type ownership-marker evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingMoveAndDropDiagnosticEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "use-after-move/use-after-consume", "use-after-consume", 1)
	audit = strings.Replace(audit, "double-drop/double-finalization", "double-finalization", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing move/drop diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "use-after-move/use-after-consume") {
		t.Fatalf("error = %v, want use-after-move diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingPartialStructEnumWholeCopyEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "partial struct/enum whole-copy rejection", "partial copy rejection", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing partial struct/enum whole-copy evidence failure")
	}
	if !strings.Contains(err.Error(), "partial struct/enum whole-copy rejection") {
		t.Fatalf("error = %v, want partial struct/enum whole-copy evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingPartialStructEnumEnumConstructorEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "partial struct/enum enum-constructor rejection", "partial enum-constructor rejection", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing partial struct/enum enum-constructor evidence failure")
	}
	if !strings.Contains(err.Error(), "partial struct/enum enum-constructor rejection") {
		t.Fatalf("error = %v, want partial struct/enum enum-constructor evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableOptionalPayloadWholeValueEvidence(t *testing.T) {
	want := "optional payload consume/free whole-value rejection"
	audit := strings.Replace(validBlockedOwnershipAudit(), "partial struct/enum enum-constructor rejection, borrow escape", "partial struct/enum enum-constructor rejection, "+want+", borrow escape", 1)
	audit = strings.Replace(audit, want, "optional payload diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable optional payload whole-value evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable optional payload whole-value evidence failure", err)
	}
}
