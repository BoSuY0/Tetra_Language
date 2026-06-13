package main

import (
	"strings"
	"testing"
)

func TestValidateOwnershipAuditRejectsMissingOwnershipSmokeExampleEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "examples/ownership_smoke.tetra", "examples/other.tetra", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing ownership smoke example evidence failure")
	}
	if !strings.Contains(err.Error(), "examples/ownership_smoke.tetra") {
		t.Fatalf("error = %v, want ownership smoke example evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingFeatureRegistryCommandEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "./tetra features --format=json", "./tetra features", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing feature registry command evidence failure")
	}
	if !strings.Contains(err.Error(), "./tetra features --format=json") {
		t.Fatalf("error = %v, want feature registry command evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingResourceFinalizationMergeTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence", "resource finalization merge diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing resource finalization merge TETRA2101 evidence failure")
	}
	if !strings.Contains(err.Error(), "branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence") {
		t.Fatalf("error = %v, want resource finalization merge TETRA2101 evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingSSATaskHandleGroupIslandMergeEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "branch/match/loop task-handle maybe-joined, task-group maybe-closed, island maybe-freed", "resource state merge diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing SSA task-handle/task-group/island merge evidence failure")
	}
	if !strings.Contains(err.Error(), "branch/match/loop task-handle maybe-joined, task-group maybe-closed, island maybe-freed") {
		t.Fatalf("error = %v, want SSA task-handle/task-group/island merge evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingReleaseGateCommandEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "bash scripts/ci/test-all.sh", "go test ./...", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing release gate command evidence failure")
	}
	if !strings.Contains(err.Error(), "bash scripts/ci/test-all.sh") {
		t.Fatalf("error = %v, want release gate command evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsReleaseGatePhraseOnlyEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"Structured test-all report `docs/generated/v1_0/test-all/summary.json` records `status: pass`, `failed_count: 0`, per-step `exit_code: 0`, and is checked by `validate-test-all-summary`; this prevents accepting the release-gate command name as proxy evidence.",
		"Full local gate passes but does not prove full ownership objective.",
		1,
	)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected release gate phrase-only evidence failure")
	}
	if !strings.Contains(err.Error(), "docs/generated/v1_0/test-all/summary.json") {
		t.Fatalf("error = %v, want structured release gate summary evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingResourceAndTransferJSONDiagnosticEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "resource use-after-free/double-join/ambiguous-provenance, island transfer non-local-payload, callable mutable-capture global/heap-escape, callable pointer/resource capture escape, function-typed storage/return unsupported capture rejection, callable global-storage escape, unsupported function-value escape, unsupported function-value call, capturing closure raw-ptr escape, captured closure explicit type-arg rejection, function-typed explicit type-arg rejection, generic closure/generic callback-closure capture, generic closure pointer/direct-call, and imported mutable function-typed global boundary JSON diagnostics", "resource finalization diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing resource and transfer JSON diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "resource use-after-free/double-join/ambiguous-provenance, island transfer non-local-payload, callable mutable-capture global/heap-escape, callable pointer/resource capture escape, function-typed storage/return unsupported capture rejection, callable global-storage escape, unsupported function-value escape, unsupported function-value call, capturing closure raw-ptr escape, captured closure explicit type-arg rejection, function-typed explicit type-arg rejection, generic closure/generic callback-closure capture, generic closure pointer/direct-call, and imported mutable function-typed global boundary JSON diagnostics") {
		t.Fatalf("error = %v, want resource and transfer JSON diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingResourceAliasTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module struct-field and enum-payload alias use-after-free with stable TETRA2101 JSON diagnostic evidence", "resource alias diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing resource alias TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module struct-field and enum-payload alias use-after-free with stable TETRA2101 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want resource alias TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingResourceAliasCLIJSONEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module struct-field/enum-payload alias use-after-free CLI JSON diagnostics", "resource alias CLI diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing resource alias CLI JSON evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module struct-field/enum-payload alias use-after-free CLI JSON diagnostics") {
		t.Fatalf("error = %v, want resource alias CLI JSON evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingMoveTaskGroupCancelReturnProvenanceEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module struct-field/enum-payload alias use-after-free CLI JSON diagnostics, same-module/cross-module task_group_cancel return provenance diagnostics with stable TETRA2101 CLI JSON evidence, and same-module/cross-module struct-field and enum-payload alias use-after-free with stable TETRA2101 JSON diagnostic evidence", "same-module/cross-module struct-field/enum-payload alias use-after-free CLI JSON diagnostics, and same-module/cross-module struct-field and enum-payload alias use-after-free with stable TETRA2101 JSON diagnostic evidence", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing move task_group_cancel return provenance evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module task_group_cancel return provenance diagnostics with stable TETRA2101 CLI JSON evidence") {
		t.Fatalf("error = %v, want move task_group_cancel return provenance evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingOptionalPayloadWholeValueTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module optional-payload whole-value rejection after payload consume/free with stable TETRA2101 JSON diagnostic evidence", "optional payload whole-value diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing optional-payload whole-value TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module optional-payload whole-value rejection after payload consume/free with stable TETRA2101 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want optional-payload whole-value TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingMoveWholeValuePartialConsumeEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module struct/enum whole-value call/let/return rejection after partial consume", "partial consume diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing move whole-value partial-consume evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module struct/enum whole-value call/let/return rejection after partial consume") {
		t.Fatalf("error = %v, want move whole-value partial-consume evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingMoveEnumWrapperConstructorEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module enum wrapper-constructor rejection after partial field/payload consume", "enum constructor diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing enum wrapper-constructor partial-consume evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module enum wrapper-constructor rejection after partial field/payload consume") {
		t.Fatalf("error = %v, want enum wrapper-constructor partial-consume evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingMoveMutableReinitializationEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "mutable reinitialization", "partial reinit", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing mutable reinitialization evidence failure")
	}
	if !strings.Contains(err.Error(), "mutable reinitialization") {
		t.Fatalf("error = %v, want mutable reinitialization evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingMoveTaskIslandGroupFinalizationEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "task/island/task-group finalization including stable `TETRA2101` task-group use-after-close", "resource finalization", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing task/island/task-group finalization evidence failure")
	}
	if !strings.Contains(err.Error(), "task/island/task-group finalization including stable `TETRA2101` task-group use-after-close") {
		t.Fatalf("error = %v, want task/island/task-group finalization evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingTaskHandleAggregateAliasJoinTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module task-handle struct-field/enum-payload alias join diagnostics with stable TETRA2101 JSON diagnostic evidence", "task-handle aggregate alias join diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing task-handle aggregate alias join TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module task-handle struct-field/enum-payload alias join diagnostics with stable TETRA2101 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want task-handle aggregate alias join TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingTaskHandleAggregateAliasTransferTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module task-handle struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence", "task-handle aggregate alias transfer diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing task-handle aggregate alias transfer TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module task-handle struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want task-handle aggregate alias transfer TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingTaskGroupAggregateAliasCloseTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module task-group struct-field/enum-payload alias close diagnostics with stable TETRA2101 JSON diagnostic evidence", "task-group aggregate alias close diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing task-group aggregate alias close TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module task-group struct-field/enum-payload alias close diagnostics with stable TETRA2101 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want task-group aggregate alias close TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingTaskHandleGroupOptionalPayloadJoinCloseTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module task-handle/task-group if-let/match optional-payload join/close aliases with stable TETRA2101 CLI JSON evidence", "task-handle/task-group optional payload aliases", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing task-handle/task-group optional payload join/close TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module task-handle/task-group if-let/match optional-payload join/close aliases with stable TETRA2101 CLI JSON evidence") {
		t.Fatalf("error = %v, want task-handle/task-group optional payload join/close TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingNestedOptionalResourceWrapperUseAfterFreeEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module nested optional resource wrapper alias use-after-free CLI JSON diagnostics", "nested optional resource wrapper diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing nested optional resource wrapper use-after-free evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module nested optional resource wrapper alias use-after-free CLI JSON diagnostics") {
		t.Fatalf("error = %v, want nested optional resource wrapper use-after-free evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingBaseAliasProvenanceEvidence(t *testing.T) {
	want := "Ownership paths, enum payload aliases, borrowed ptr-leaf aliases for ptr-containing aggregate parameters, borrowed scalar `ptr` assignment into optional `ptr?` payloads, borrowed region-bearing slice assignment into optional `[]u8?` payloads, and pattern-bound enum/optional payloads"
	audit := strings.Replace(validBlockedOwnershipAudit(), want, "Ownership paths and alias basics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing base alias/provenance evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want base alias/provenance evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingOptionalPayloadAliasEvidence(t *testing.T) {
	want := "optional payload consume aliases, if-let/match optional resource aliases"
	audit := strings.Replace(validBlockedOwnershipAudit(), want, "optional payload aliases", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing optional payload alias evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want optional payload alias evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingTypedErrorResourceAliasTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "typed-error throw/catch and rethrow-through-try enum-payload resource aliases with stable TETRA2101 JSON diagnostic evidence", "typed-error resource aliases", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing typed-error resource alias TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "typed-error throw/catch and rethrow-through-try enum-payload resource aliases with stable TETRA2101 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want typed-error resource alias TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingOptionalResourceWrapperAliasEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "optional resource wrapper aliases including nested struct/enum wrappers", "optional resource wrapper aliases", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing optional resource wrapper alias evidence failure")
	}
	if !strings.Contains(err.Error(), "optional resource wrapper aliases including nested struct/enum wrappers") {
		t.Fatalf("error = %v, want optional resource wrapper alias evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingActorConsumeAliasEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module actor if-let/match optional-payload, struct-field, enum-payload, and transitive interprocedural consume aliases", "actor consume aliases", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing actor consume alias evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module actor if-let/match optional-payload, struct-field, enum-payload, and transitive interprocedural consume aliases") {
		t.Fatalf("error = %v, want actor consume alias evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingActorAggregateTransferAliasRowEvidence(t *testing.T) {
	want := "same-module/cross-module actor if-let/match optional-payload, struct-field, enum-payload, and transitive interprocedural consume aliases including same-module/cross-module actor struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence"
	audit := strings.Replace(validBlockedOwnershipAudit(), want, "actor consume aliases", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing alias-row actor aggregate transfer evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want alias-row actor aggregate transfer evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingTaskHandleGroupAggregateAliasEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module task-handle/task-group struct-field/enum-payload transfer/join/close aliases", "task-handle task-group aggregate aliases", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing task-handle/task-group aggregate alias evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module task-handle/task-group struct-field/enum-payload transfer/join/close aliases") {
		t.Fatalf("error = %v, want task-handle/task-group aggregate alias evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingTaskHandleGroupDetailedAggregateAliasRowEvidence(t *testing.T) {
	want := "same-module/cross-module task-handle/task-group struct-field/enum-payload transfer/join/close aliases including same-module/cross-module task-handle struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence, same-module/cross-module task-handle struct-field/enum-payload alias join diagnostics with stable TETRA2101 JSON diagnostic evidence and same-module/cross-module task-group struct-field/enum-payload alias close diagnostics with stable TETRA2101 JSON diagnostic evidence"
	audit := strings.Replace(validBlockedOwnershipAudit(), want, "task-handle task-group aggregate aliases", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing alias-row task-handle/task-group detailed aggregate evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want alias-row task-handle/task-group detailed aggregate evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingMonomorphizedGenericActorAliasEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module monomorphized generic struct actor consume aliases", "generic actor aliases", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing monomorphized generic actor alias evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module monomorphized generic struct actor consume aliases") {
		t.Fatalf("error = %v, want monomorphized generic actor alias evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingAmbiguousResourceProvenanceEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "ambiguous resource provenance", "ambiguous provenance", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing ambiguous resource provenance evidence failure")
	}
	if !strings.Contains(err.Error(), "ambiguous resource provenance") {
		t.Fatalf("error = %v, want ambiguous resource provenance evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingGeneratedT4IResourceProvenanceEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "generated `.t4i` direct/local/aggregate-local-alias/aggregate-field-access/aggregate-field-local-alias resource return, assignment/let/direct-if-let/direct-match/field-local/if-let/match optional and nested/field-local nested optional resource return, typed-error direct/field-local-alias throw, and rethrow-through-`try` direct/field-local-alias provenance stubs", "generated resource provenance stubs", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing generated .t4i resource provenance evidence failure")
	}
	if !strings.Contains(err.Error(), "generated `.t4i` direct/local/aggregate-local-alias/aggregate-field-access/aggregate-field-local-alias resource return, assignment/let/direct-if-let/direct-match/field-local/if-let/match optional and nested/field-local nested optional resource return, typed-error direct/field-local-alias throw, and rethrow-through-`try` direct/field-local-alias provenance stubs") {
		t.Fatalf("error = %v, want generated .t4i resource provenance evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingActorAggregateAliasTransferTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module actor struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence", "actor aggregate alias transfer diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing actor aggregate alias transfer TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module actor struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want actor aggregate alias transfer TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingActorBranchMatchLoopConsumeReuseTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "branch/match/loop actor consume reuse diagnostics with stable TETRA2101 CLI JSON evidence", "actor branch consume reuse diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing actor branch/match/loop consume reuse TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "branch/match/loop actor consume reuse diagnostics with stable TETRA2101 CLI JSON evidence") {
		t.Fatalf("error = %v, want actor branch/match/loop consume reuse TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingActorTaskUseAfterTransferTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "actor/task use-after-transfer diagnostics with stable TETRA2101 CLI JSON evidence", "actor task use-after-transfer diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing actor/task use-after-transfer TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "actor/task use-after-transfer diagnostics with stable TETRA2101 CLI JSON evidence") {
		t.Fatalf("error = %v, want actor/task use-after-transfer TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingIslandTransferNonLocalPayloadTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "island transfer non-local-payload rejection with stable TETRA2101 CLI JSON evidence", "island transfer diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing island transfer non-local-payload TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "island transfer non-local-payload rejection with stable TETRA2101 CLI JSON evidence") {
		t.Fatalf("error = %v, want island transfer non-local-payload TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingGenericStructActorAliasTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module monomorphized generic struct actor consume alias diagnostics with stable TETRA2101 CLI JSON evidence", "generic actor alias diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing generic struct actor alias TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module monomorphized generic struct actor consume alias diagnostics with stable TETRA2101 CLI JSON evidence") {
		t.Fatalf("error = %v, want generic struct actor alias TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingTransitiveActorAliasTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module transitive actor consume alias diagnostics with stable TETRA2101 CLI JSON evidence", "transitive actor alias diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing transitive actor alias TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module transitive actor consume alias diagnostics with stable TETRA2101 CLI JSON evidence") {
		t.Fatalf("error = %v, want transitive actor alias TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingTaskGroupCancelReturnProvenanceTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module task_group_cancel return provenance diagnostics with stable TETRA2101 CLI JSON evidence", "task_group_cancel return provenance diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing task_group_cancel return provenance TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module task_group_cancel return provenance diagnostics with stable TETRA2101 CLI JSON evidence") {
		t.Fatalf("error = %v, want task_group_cancel return provenance TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingActorTaskOptionalPayloadAliasTransferTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module actor/task if-let/match optional-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence", "actor task optional payload alias transfer diagnostics", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing actor/task optional-payload alias transfer TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module actor/task if-let/match optional-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want actor/task optional-payload alias transfer TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingWholeAggregateGlobalAssignmentEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module whole-aggregate global assignment with stable TETRA2102 JSON diagnostic evidence", "global field assignment", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing whole-aggregate global assignment evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module whole-aggregate global assignment with stable TETRA2102 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want whole-aggregate global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingDirectSliceGlobalAssignmentEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module direct slice global assignment with stable TETRA2102 JSON diagnostic evidence", "slice assignment", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing direct slice global assignment evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module direct slice global assignment with stable TETRA2102 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want direct slice global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingFixedArrayAliasReturnEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module fixed-array alias return", "array return", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing fixed-array alias return evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module fixed-array alias return") {
		t.Fatalf("error = %v, want fixed-array alias return evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingFixedArrayTETRA2102Evidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "fixed-array escapes including inout assignment with stable TETRA2102 diagnostic evidence", "fixed-array escapes", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing fixed-array TETRA2102 diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "fixed-array escapes including inout assignment with stable TETRA2102 diagnostic evidence") {
		t.Fatalf("error = %v, want fixed-array TETRA2102 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingFixedArrayInoutAssignmentEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module fixed-array inout assignment", "fixed-array assignment", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing fixed-array inout assignment evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module fixed-array inout assignment") {
		t.Fatalf("error = %v, want fixed-array inout assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingFixedArrayGlobalAssignmentEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module direct fixed-array global assignment", "array global assignment", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing fixed-array global assignment evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module direct fixed-array global assignment") {
		t.Fatalf("error = %v, want fixed-array global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingOptionalFixedArrayGlobalAssignmentEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module optional fixed-array global assignment", "optional array global assignment", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing optional fixed-array global assignment evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module optional fixed-array global assignment") {
		t.Fatalf("error = %v, want optional fixed-array global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingBorrowedStringEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "borrowed string alias return/global assignment", "string escape", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing borrowed string evidence failure")
	}
	if !strings.Contains(err.Error(), "borrowed string alias return/global assignment") {
		t.Fatalf("error = %v, want borrowed string evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingOptionalPtrGlobalAssignmentEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module optional ptr global assignment with stable TETRA2102 JSON diagnostic evidence", "optional assignment", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing optional ptr global assignment evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module optional ptr global assignment with stable TETRA2102 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want optional ptr global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingOptionalAggregateGlobalAssignmentEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module optional aggregate global assignment with stable TETRA2102 JSON diagnostic evidence", "optional assignment", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing optional aggregate global assignment evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module optional aggregate global assignment with stable TETRA2102 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want optional aggregate global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingPtrOptionalAssignmentIfLetMatchGlobalEscapeEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module ptr optional assignment if-let/match global escape with stable TETRA2102 JSON diagnostic evidence", "optional if-let escape", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing ptr optional assignment if-let/match global escape evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module ptr optional assignment if-let/match global escape with stable TETRA2102 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want ptr optional assignment if-let/match global escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingPtrEnumAliasReturnEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module ptr enum alias return escape with stable TETRA2102 JSON diagnostic evidence", "enum alias return", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing ptr enum alias return evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module ptr enum alias return escape with stable TETRA2102 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want ptr enum alias return evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingPtrAggregateReturnEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module ptr-containing aggregate whole/field/alias/nested-field return escapes with stable TETRA2102 JSON diagnostic evidence", "aggregate return", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing ptr aggregate return evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module ptr-containing aggregate whole/field/alias/nested-field return escapes with stable TETRA2102 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want ptr aggregate return evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingPtrEnumPayloadEscapeEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module ptr enum-payload return/global/inout assignment escapes with stable TETRA2102 JSON diagnostic evidence", "enum payload escape", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing ptr enum-payload escape evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module ptr enum-payload return/global/inout assignment escapes with stable TETRA2102 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want ptr enum-payload escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingPtrOptionalPayloadEscapeEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module ptr optional-payload return/global/inout assignment escapes with stable TETRA2102 JSON diagnostic evidence", "optional payload escape", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing ptr optional-payload escape evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module ptr optional-payload return/global/inout assignment escapes with stable TETRA2102 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want ptr optional-payload escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingSliceOptionalPayloadInoutGlobalEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "same-module/cross-module slice optional-payload inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence", "slice optional payload escape", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing slice optional-payload inout/global evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module slice optional-payload inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence") {
		t.Fatalf("error = %v, want slice optional-payload inout/global evidence failure", err)
	}
}
