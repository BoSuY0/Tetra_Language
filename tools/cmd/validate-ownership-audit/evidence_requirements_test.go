package main

import (
	"strings"
	"testing"
)

func joinAuditEvidence(parts ...string) string {
	return strings.Join(parts, ", ")
}

func TestValidateOwnershipAuditRejectsMissingOwnershipSmokeExampleEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"examples/memory/ownership/ownership_smoke.tetra",
		"examples/other.tetra",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing ownership smoke example evidence failure")
	}
	if !strings.Contains(err.Error(), "examples/memory/ownership/ownership_smoke.tetra") {
		t.Fatalf("error = %v, want ownership smoke example evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingFeatureRegistryCommandEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"./tetra features --format=json",
		"./tetra features",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing feature registry command evidence failure")
	}
	if !strings.Contains(err.Error(), "./tetra features --format=json") {
		t.Fatalf("error = %v, want feature registry command evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingResourceFinalizationMergeTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence",
		"resource finalization merge diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing resource finalization merge TETRA2101 evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		"branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence",
	) {
		t.Fatalf("error = %v, want resource finalization merge TETRA2101 evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingSSATaskHandleGroupIslandMergeEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"branch/match/loop task-handle maybe-joined, task-group maybe-closed, island maybe-freed",
		"resource state merge diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing SSA task-handle/task-group/island merge evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		"branch/match/loop task-handle maybe-joined, task-group maybe-closed, island maybe-freed",
	) {
		t.Fatalf("error = %v, want SSA task-handle/task-group/island merge evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingReleaseGateCommandEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"bash scripts/ci/test-all.sh",
		"go test ./...",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
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
		("Structured test-all report " +
			"`docs/generated/v1_0/test-all/summary.json` records `status:" +
			" pass`, `failed_count: 0`, per-step `exit_code: 0`, and is " +
			"checked by `validate-test-all-summary`; this prevents " +
			"accepting the release-gate command name as proxy evidence."),
		"Full local gate passes but does not prove full ownership objective.",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected release gate phrase-only evidence failure")
	}
	if !strings.Contains(err.Error(), "docs/generated/v1_0/test-all/summary.json") {
		t.Fatalf("error = %v, want structured release gate summary evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingResourceAndTransferJSONDiagnosticEvidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("resource use-after-free/double-join/ambiguous-provenance, " +
			"island transfer non-local-payload, callable mutable-capture " +
			"global/heap-escape, callable pointer/resource capture " +
			"escape, function-typed storage/return unsupported capture " +
			"rejection, callable global-storage escape, unsupported " +
			"function-value escape, unsupported function-value call, " +
			"capturing closure raw-ptr escape, captured closure explicit " +
			"type-arg rejection, function-typed explicit type-arg " +
			"rejection, generic closure/generic callback-closure capture," +
			" generic closure pointer/direct-call, and imported mutable " +
			"function-typed global boundary JSON diagnostics"),
		"resource finalization diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing resource and transfer JSON diagnostic evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("resource use-after-free/double-join/ambiguous-provenance, " +
			"island transfer non-local-payload, callable mutable-capture " +
			"global/heap-escape, callable pointer/resource capture " +
			"escape, function-typed storage/return unsupported capture " +
			"rejection, callable global-storage escape, unsupported " +
			"function-value escape, unsupported function-value call, " +
			"capturing closure raw-ptr escape, captured closure explicit " +
			"type-arg rejection, function-typed explicit type-arg " +
			"rejection, generic closure/generic callback-closure capture," +
			" generic closure pointer/direct-call, and imported mutable " +
			"function-typed global boundary JSON diagnostics"),
	) {
		t.Fatalf("error = %v, want resource and transfer JSON diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingResourceAliasTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module struct-field and enum-payload " +
			"alias use-after-free with stable TETRA2101 JSON diagnostic " +
			"evidence"),
		"resource alias diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing resource alias TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module struct-field and enum-payload " +
			"alias use-after-free with stable TETRA2101 JSON diagnostic " +
			"evidence"),
	) {
		t.Fatalf("error = %v, want resource alias TETRA2101 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingResourceAliasCLIJSONEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"same-module/cross-module struct-field/enum-payload alias use-after-free CLI JSON diagnostics",
		"resource alias CLI diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing resource alias CLI JSON evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		"same-module/cross-module struct-field/enum-payload alias use-after-free CLI JSON diagnostics",
	) {
		t.Fatalf("error = %v, want resource alias CLI JSON evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingMoveTaskGroupCancelReturnProvenanceEvidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module struct-field/enum-payload alias " +
			"use-after-free CLI JSON diagnostics, " +
			"same-module/cross-module task_group_cancel return " +
			"provenance diagnostics with stable TETRA2101 CLI JSON " +
			"evidence, and same-module/cross-module struct-field and " +
			"enum-payload alias use-after-free with stable TETRA2101 " +
			"JSON diagnostic evidence"),
		("same-module/cross-module struct-field/enum-payload alias " +
			"use-after-free CLI JSON diagnostics, and " +
			"same-module/cross-module struct-field and enum-payload " +
			"alias use-after-free with stable TETRA2101 JSON diagnostic " +
			"evidence"),
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing move task_group_cancel return provenance evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module task_group_cancel return " +
			"provenance diagnostics with stable TETRA2101 CLI JSON " +
			"evidence"),
	) {
		t.Fatalf("error = %v, want move task_group_cancel return provenance evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingOptionalPayloadWholeValueTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module optional-payload whole-value " +
			"rejection after payload consume/free with stable TETRA2101 " +
			"JSON diagnostic evidence"),
		"optional payload whole-value diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing optional-payload whole-value TETRA2101 diagnostic evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module optional-payload whole-value " +
			"rejection after payload consume/free with stable TETRA2101 " +
			"JSON diagnostic evidence"),
	) {
		t.Fatalf(
			"error = %v, want optional-payload whole-value TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingMoveWholeValuePartialConsumeEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module struct/enum whole-value " +
			"call/let/return rejection after partial consume"),
		"partial consume diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing move whole-value partial-consume evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module struct/enum whole-value " +
			"call/let/return rejection after partial consume"),
	) {
		t.Fatalf("error = %v, want move whole-value partial-consume evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingMoveEnumWrapperConstructorEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"same-module/cross-module enum wrapper-constructor rejection after partial field/payload consume",
		"enum constructor diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing enum wrapper-constructor partial-consume evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		"same-module/cross-module enum wrapper-constructor rejection after partial field/payload consume",
	) {
		t.Fatalf("error = %v, want enum wrapper-constructor partial-consume evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingMoveMutableReinitializationEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"mutable reinitialization",
		"partial reinit",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing mutable reinitialization evidence failure")
	}
	if !strings.Contains(err.Error(), "mutable reinitialization") {
		t.Fatalf("error = %v, want mutable reinitialization evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingMoveTaskIslandGroupFinalizationEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"task/island/task-group finalization including stable `TETRA2101` task-group use-after-close",
		"resource finalization",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing task/island/task-group finalization evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		"task/island/task-group finalization including stable `TETRA2101` task-group use-after-close",
	) {
		t.Fatalf("error = %v, want task/island/task-group finalization evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingTaskHandleAggregateAliasJoinTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module task-handle " +
			"struct-field/enum-payload alias join diagnostics with " +
			"stable TETRA2101 JSON diagnostic evidence"),
		"task-handle aggregate alias join diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing task-handle aggregate alias join TETRA2101 diagnostic evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module task-handle " +
			"struct-field/enum-payload alias join diagnostics with " +
			"stable TETRA2101 JSON diagnostic evidence"),
	) {
		t.Fatalf(
			"error = %v, want task-handle aggregate alias join TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingTaskHandleAggregateAliasTransferTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module task-handle " +
			"struct-field/enum-payload alias transfer diagnostics with " +
			"stable TETRA2101 JSON diagnostic evidence"),
		"task-handle aggregate alias transfer diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing task-handle aggregate alias transfer TETRA2101 diagnostic evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module task-handle " +
			"struct-field/enum-payload alias transfer diagnostics with " +
			"stable TETRA2101 JSON diagnostic evidence"),
	) {
		t.Fatalf(
			"error = %v, want task-handle aggregate alias transfer TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingTaskGroupAggregateAliasCloseTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module task-group " +
			"struct-field/enum-payload alias close diagnostics with " +
			"stable TETRA2101 JSON diagnostic evidence"),
		"task-group aggregate alias close diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing task-group aggregate alias close TETRA2101 diagnostic evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module task-group " +
			"struct-field/enum-payload alias close diagnostics with " +
			"stable TETRA2101 JSON diagnostic evidence"),
	) {
		t.Fatalf(
			"error = %v, want task-group aggregate alias close TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingTaskHandleGroupOptionalPayloadTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module task-handle/task-group " +
			"if-let/match optional-payload join/close aliases with " +
			"stable TETRA2101 CLI JSON evidence"),
		"task-handle/task-group optional payload aliases",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			("expected missing task-handle/task-group optional payload " +
				"join/close TETRA2101 diagnostic evidence failure"),
		)
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module task-handle/task-group " +
			"if-let/match optional-payload join/close aliases with " +
			"stable TETRA2101 CLI JSON evidence"),
	) {
		t.Fatalf(
			("error = %v, want task-handle/task-group optional payload " +
				"join/close TETRA2101 diagnostic evidence failure"),
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingNestedOptionalResourceWrapperUseAfterFreeEvidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module nested optional resource wrapper " +
			"alias use-after-free CLI JSON diagnostics"),
		"nested optional resource wrapper diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing nested optional resource wrapper use-after-free evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module nested optional resource wrapper " +
			"alias use-after-free CLI JSON diagnostics"),
	) {
		t.Fatalf(
			"error = %v, want nested optional resource wrapper use-after-free evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingBaseAliasProvenanceEvidence(t *testing.T) {
	want := ("Ownership paths, enum payload aliases, borrowed ptr-leaf " +
		"aliases for ptr-containing aggregate parameters, borrowed " +
		"scalar `ptr` assignment into optional `ptr?` payloads, " +
		"borrowed region-bearing slice assignment into optional `[]" +
		"u8?` payloads, and pattern-bound enum/optional payloads")
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		want,
		"Ownership paths and alias basics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
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
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing optional payload alias evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want optional payload alias evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingTypedErrorResourceAliasTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("typed-error throw/catch and rethrow-through-try " +
			"enum-payload resource aliases with stable TETRA2101 JSON " +
			"diagnostic evidence"),
		"typed-error resource aliases",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing typed-error resource alias TETRA2101 diagnostic evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("typed-error throw/catch and rethrow-through-try " +
			"enum-payload resource aliases with stable TETRA2101 JSON " +
			"diagnostic evidence"),
	) {
		t.Fatalf(
			"error = %v, want typed-error resource alias TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingOptionalResourceWrapperAliasEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"optional resource wrapper aliases including nested struct/enum wrappers",
		"optional resource wrapper aliases",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing optional resource wrapper alias evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		"optional resource wrapper aliases including nested struct/enum wrappers",
	) {
		t.Fatalf("error = %v, want optional resource wrapper alias evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingActorConsumeAliasEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module actor if-let/match " +
			"optional-payload, struct-field, enum-payload, and " +
			"transitive interprocedural consume aliases"),
		"actor consume aliases",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing actor consume alias evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module actor if-let/match " +
			"optional-payload, struct-field, enum-payload, and " +
			"transitive interprocedural consume aliases"),
	) {
		t.Fatalf("error = %v, want actor consume alias evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingActorAggregateTransferAliasRowEvidence(t *testing.T) {
	want := ("same-module/cross-module actor if-let/match " +
		"optional-payload, struct-field, enum-payload, and " +
		"transitive interprocedural consume aliases including " +
		"same-module/cross-module actor struct-field/enum-payload " +
		"alias transfer diagnostics with stable TETRA2101 JSON " +
		"diagnostic evidence")
	audit := strings.Replace(validBlockedOwnershipAudit(), want, "actor consume aliases", 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing alias-row actor aggregate transfer evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want alias-row actor aggregate transfer evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingTaskHandleGroupAggregateAliasEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module task-handle/task-group " +
			"struct-field/enum-payload transfer/join/close aliases"),
		"task-handle task-group aggregate aliases",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing task-handle/task-group aggregate alias evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module task-handle/task-group " +
			"struct-field/enum-payload transfer/join/close aliases"),
	) {
		t.Fatalf("error = %v, want task-handle/task-group aggregate alias evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingTaskHandleGroupDetailedAggregateAliasRowEvidence(
	t *testing.T,
) {
	want := ("same-module/cross-module task-handle/task-group " +
		"struct-field/enum-payload transfer/join/close aliases " +
		"including same-module/cross-module task-handle " +
		"struct-field/enum-payload alias transfer diagnostics with " +
		"stable TETRA2101 JSON diagnostic evidence, " +
		"same-module/cross-module task-handle " +
		"struct-field/enum-payload alias join diagnostics with " +
		"stable TETRA2101 JSON diagnostic evidence and " +
		"same-module/cross-module task-group " +
		"struct-field/enum-payload alias close diagnostics with " +
		"stable TETRA2101 JSON diagnostic evidence")
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		want,
		"task-handle task-group aggregate aliases",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing alias-row task-handle/task-group detailed aggregate evidence failure",
		)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf(
			"error = %v, want alias-row task-handle/task-group detailed aggregate evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingMonomorphizedGenericActorAliasEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"same-module/cross-module monomorphized generic struct actor consume aliases",
		"generic actor aliases",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing monomorphized generic actor alias evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		"same-module/cross-module monomorphized generic struct actor consume aliases",
	) {
		t.Fatalf("error = %v, want monomorphized generic actor alias evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingAmbiguousResourceProvenanceEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"ambiguous resource provenance",
		"ambiguous provenance",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing ambiguous resource provenance evidence failure")
	}
	if !strings.Contains(err.Error(), "ambiguous resource provenance") {
		t.Fatalf("error = %v, want ambiguous resource provenance evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingGeneratedT4IResourceProvenanceEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("generated `.t4i` " +
			"direct/local/aggregate-local-alias/aggregate-field-access/ag" +
			"gregate-field-local-alias resource return, " +
			"assignment/let/direct-if-let/direct-match/field-local/if-let" +
			"/match optional and nested/field-local nested optional " +
			"resource return, typed-error direct/field-local-alias throw," +
			" and rethrow-through-`try` direct/field-local-alias " +
			"provenance stubs"),
		"generated resource provenance stubs",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing generated .t4i resource provenance evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("generated `.t4i` " +
			"direct/local/aggregate-local-alias/aggregate-field-access/ag" +
			"gregate-field-local-alias resource return, " +
			"assignment/let/direct-if-let/direct-match/field-local/if-let" +
			"/match optional and nested/field-local nested optional " +
			"resource return, typed-error direct/field-local-alias throw," +
			" and rethrow-through-`try` direct/field-local-alias " +
			"provenance stubs"),
	) {
		t.Fatalf("error = %v, want generated .t4i resource provenance evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingActorAggregateAliasTransferTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module actor struct-field/enum-payload " +
			"alias transfer diagnostics with stable TETRA2101 JSON " +
			"diagnostic evidence"),
		"actor aggregate alias transfer diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing actor aggregate alias transfer TETRA2101 diagnostic evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module actor struct-field/enum-payload " +
			"alias transfer diagnostics with stable TETRA2101 JSON " +
			"diagnostic evidence"),
	) {
		t.Fatalf(
			"error = %v, want actor aggregate alias transfer TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingActorBranchMatchLoopConsumeReuseTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"branch/match/loop actor consume reuse diagnostics with stable TETRA2101 CLI JSON evidence",
		"actor branch consume reuse diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing actor branch/match/loop consume reuse TETRA2101 diagnostic evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		"branch/match/loop actor consume reuse diagnostics with stable TETRA2101 CLI JSON evidence",
	) {
		t.Fatalf(
			"error = %v, want actor branch/match/loop consume reuse TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingActorTaskUseAfterTransferTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"actor/task use-after-transfer diagnostics with stable TETRA2101 CLI JSON evidence",
		"actor task use-after-transfer diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing actor/task use-after-transfer TETRA2101 diagnostic evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		"actor/task use-after-transfer diagnostics with stable TETRA2101 CLI JSON evidence",
	) {
		t.Fatalf(
			"error = %v, want actor/task use-after-transfer TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingIslandTransferNonLocalPayloadTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"island transfer non-local-payload rejection with stable TETRA2101 CLI JSON evidence",
		"island transfer diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing island transfer non-local-payload TETRA2101 diagnostic evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		"island transfer non-local-payload rejection with stable TETRA2101 CLI JSON evidence",
	) {
		t.Fatalf(
			"error = %v, want island transfer non-local-payload TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingGenericStructActorAliasTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module monomorphized generic struct actor " +
			"consume alias diagnostics with stable TETRA2101 CLI JSON " +
			"evidence"),
		"generic actor alias diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing generic struct actor alias TETRA2101 diagnostic evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module monomorphized generic struct actor " +
			"consume alias diagnostics with stable TETRA2101 CLI JSON " +
			"evidence"),
	) {
		t.Fatalf(
			"error = %v, want generic struct actor alias TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingTransitiveActorAliasTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module transitive actor consume alias " +
			"diagnostics with stable TETRA2101 CLI JSON evidence"),
		"transitive actor alias diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing transitive actor alias TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module transitive actor consume alias " +
			"diagnostics with stable TETRA2101 CLI JSON evidence"),
	) {
		t.Fatalf(
			"error = %v, want transitive actor alias TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingTaskGroupCancelReturnProvenanceTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module task_group_cancel return " +
			"provenance diagnostics with stable TETRA2101 CLI JSON " +
			"evidence"),
		"task_group_cancel return provenance diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing task_group_cancel return provenance TETRA2101 diagnostic evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module task_group_cancel return " +
			"provenance diagnostics with stable TETRA2101 CLI JSON " +
			"evidence"),
	) {
		t.Fatalf(
			"error = %v, want task_group_cancel return provenance TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingActorTaskOptionalPayloadAliasTransferTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module actor/task if-let/match " +
			"optional-payload alias transfer diagnostics with stable " +
			"TETRA2101 JSON diagnostic evidence"),
		"actor task optional payload alias transfer diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			("expected missing actor/task optional-payload alias transfer " +
				"TETRA2101 diagnostic evidence failure"),
		)
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module actor/task if-let/match " +
			"optional-payload alias transfer diagnostics with stable " +
			"TETRA2101 JSON diagnostic evidence"),
	) {
		t.Fatalf(
			("error = %v, want actor/task optional-payload alias transfer " +
				"TETRA2101 diagnostic evidence failure"),
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingWholeAggregateGlobalAssignmentEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module whole-aggregate global assignment " +
			"with stable TETRA2102 JSON diagnostic evidence"),
		"global field assignment",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing whole-aggregate global assignment evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module whole-aggregate global assignment " +
			"with stable TETRA2102 JSON diagnostic evidence"),
	) {
		t.Fatalf("error = %v, want whole-aggregate global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingDirectSliceGlobalAssignmentEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module direct slice global assignment " +
			"with stable TETRA2102 JSON diagnostic evidence"),
		"slice assignment",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing direct slice global assignment evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module direct slice global assignment " +
			"with stable TETRA2102 JSON diagnostic evidence"),
	) {
		t.Fatalf("error = %v, want direct slice global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingFixedArrayAliasReturnEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"same-module/cross-module fixed-array alias return",
		"array return",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing fixed-array alias return evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module fixed-array alias return") {
		t.Fatalf("error = %v, want fixed-array alias return evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingFixedArrayTETRA2102Evidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"fixed-array escapes including inout assignment with stable TETRA2102 diagnostic evidence",
		"fixed-array escapes",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing fixed-array TETRA2102 diagnostic evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		"fixed-array escapes including inout assignment with stable TETRA2102 diagnostic evidence",
	) {
		t.Fatalf("error = %v, want fixed-array TETRA2102 diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingFixedArrayInoutAssignmentEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"same-module/cross-module fixed-array inout assignment",
		"fixed-array assignment",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing fixed-array inout assignment evidence failure")
	}
	if !strings.Contains(err.Error(), "same-module/cross-module fixed-array inout assignment") {
		t.Fatalf("error = %v, want fixed-array inout assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingFixedArrayGlobalAssignmentEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"same-module/cross-module direct fixed-array global assignment",
		"array global assignment",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing fixed-array global assignment evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		"same-module/cross-module direct fixed-array global assignment",
	) {
		t.Fatalf("error = %v, want fixed-array global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingOptionalFixedArrayGlobalAssignmentEvidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"same-module/cross-module optional fixed-array global assignment",
		"optional array global assignment",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing optional fixed-array global assignment evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		"same-module/cross-module optional fixed-array global assignment",
	) {
		t.Fatalf("error = %v, want optional fixed-array global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingBorrowedStringEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"borrowed string alias return/global assignment",
		"string escape",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing borrowed string evidence failure")
	}
	if !strings.Contains(err.Error(), "borrowed string alias return/global assignment") {
		t.Fatalf("error = %v, want borrowed string evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingOptionalPtrGlobalAssignmentEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module optional ptr global assignment " +
			"with stable TETRA2102 JSON diagnostic evidence"),
		"optional assignment",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing optional ptr global assignment evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module optional ptr global assignment " +
			"with stable TETRA2102 JSON diagnostic evidence"),
	) {
		t.Fatalf("error = %v, want optional ptr global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingOptionalAggregateGlobalAssignmentEvidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module optional aggregate global " +
			"assignment with stable TETRA2102 JSON diagnostic evidence"),
		"optional assignment",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing optional aggregate global assignment evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module optional aggregate global " +
			"assignment with stable TETRA2102 JSON diagnostic evidence"),
	) {
		t.Fatalf("error = %v, want optional aggregate global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingPtrOptionalAssignmentIfLetMatchGlobalEscapeEvidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module ptr optional assignment " +
			"if-let/match global escape with stable TETRA2102 JSON " +
			"diagnostic evidence"),
		"optional if-let escape",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing ptr optional assignment if-let/match global escape evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module ptr optional assignment " +
			"if-let/match global escape with stable TETRA2102 JSON " +
			"diagnostic evidence"),
	) {
		t.Fatalf(
			"error = %v, want ptr optional assignment if-let/match global escape evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingPtrEnumAliasReturnEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module ptr enum alias return escape with " +
			"stable TETRA2102 JSON diagnostic evidence"),
		"enum alias return",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing ptr enum alias return evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module ptr enum alias return escape with " +
			"stable TETRA2102 JSON diagnostic evidence"),
	) {
		t.Fatalf("error = %v, want ptr enum alias return evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingPtrAggregateReturnEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module ptr-containing aggregate " +
			"whole/field/alias/nested-field return escapes with stable " +
			"TETRA2102 JSON diagnostic evidence"),
		"aggregate return",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing ptr aggregate return evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module ptr-containing aggregate " +
			"whole/field/alias/nested-field return escapes with stable " +
			"TETRA2102 JSON diagnostic evidence"),
	) {
		t.Fatalf("error = %v, want ptr aggregate return evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingPtrEnumPayloadEscapeEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module ptr enum-payload " +
			"return/global/inout assignment escapes with stable " +
			"TETRA2102 JSON diagnostic evidence"),
		"enum payload escape",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing ptr enum-payload escape evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module ptr enum-payload " +
			"return/global/inout assignment escapes with stable " +
			"TETRA2102 JSON diagnostic evidence"),
	) {
		t.Fatalf("error = %v, want ptr enum-payload escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingPtrOptionalPayloadEscapeEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module ptr optional-payload " +
			"return/global/inout assignment escapes with stable " +
			"TETRA2102 JSON diagnostic evidence"),
		"optional payload escape",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing ptr optional-payload escape evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module ptr optional-payload " +
			"return/global/inout assignment escapes with stable " +
			"TETRA2102 JSON diagnostic evidence"),
	) {
		t.Fatalf("error = %v, want ptr optional-payload escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingSliceOptionalPayloadInoutGlobalEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module slice optional-payload " +
			"inout/global assignment escapes with stable TETRA2102 JSON " +
			"diagnostic evidence"),
		"slice optional payload escape",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing slice optional-payload inout/global evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module slice optional-payload " +
			"inout/global assignment escapes with stable TETRA2102 JSON " +
			"diagnostic evidence"),
	) {
		t.Fatalf("error = %v, want slice optional-payload inout/global evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingNestedSliceEnumPayloadEscapeEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module nested slice enum-payload " +
			"return/inout/global assignment escapes with stable " +
			"TETRA2102 JSON diagnostic evidence"),
		"nested slice enum payload escape",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing nested slice enum-payload escape evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module nested slice enum-payload " +
			"return/inout/global assignment escapes with stable " +
			"TETRA2102 JSON diagnostic evidence"),
	) {
		t.Fatalf("error = %v, want nested slice enum-payload escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingNestedSliceStructEscapeEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module nested slice struct " +
			"return/inout/global assignment escapes with stable " +
			"TETRA2102 JSON diagnostic evidence"),
		"nested slice struct escape",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing nested slice struct escape evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module nested slice struct " +
			"return/inout/global assignment escapes with stable " +
			"TETRA2102 JSON diagnostic evidence"),
	) {
		t.Fatalf("error = %v, want nested slice struct escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingOptionalMixedAggregateBranchEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"optional mixed safe/provenance aggregate branch merges",
		"optional aggregate branch merges",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing optional mixed aggregate branch evidence failure")
	}
	if !strings.Contains(err.Error(), "optional mixed safe/provenance aggregate branch merges") {
		t.Fatalf("error = %v, want optional mixed aggregate branch evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingMixedAggregateBranchAndMatchEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"mixed safe/provenance aggregate branch and match returns",
		"mixed aggregate returns",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing mixed aggregate branch/match evidence failure")
	}
	if !strings.Contains(err.Error(), "mixed safe/provenance aggregate branch and match returns") {
		t.Fatalf("error = %v, want mixed aggregate branch/match evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingInterproceduralPerFieldRegionSummaryEvidence(
	t *testing.T,
) {
	want := ("same-module and interface-only cross-module per-field " +
		"interprocedural region summaries for aggregate returns from " +
		"multiple island parameters, including optional aggregate " +
		"wrappers, enum payload wrappers, branch aggregate wrappers, " +
		"match aggregate wrappers, if-let aggregate wrappers")
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		want,
		"interprocedural aggregate summaries",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing interprocedural per-field region summary evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want interprocedural per-field region summary evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingInterproceduralResourceSummaryEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("Local return-resource summaries, typed-error throw-resource " +
			"summaries including rethrow-through-`try`"),
		"Local resource summaries",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing interprocedural resource summary evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("Local return-resource summaries, typed-error throw-resource " +
			"summaries including rethrow-through-`try`"),
	) {
		t.Fatalf("error = %v, want interprocedural resource summary evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingInterproceduralGeneratedT4IResourceProvenanceEvidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("generated `.t4i` " +
			"direct/local/aggregate-local-alias/aggregate-field-access/ag" +
			"gregate-field-local-alias resource return, " +
			"assignment/let/direct-if-let/direct-match/field-local/if-let" +
			"/match optional and nested/field-local nested optional " +
			"resource return, typed-error direct/field-local-alias throw," +
			" and rethrow-through-`try` direct/field-local-alias " +
			"provenance stubs"),
		"generated interprocedural provenance stubs",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing interprocedural generated .t4i resource provenance evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("generated `.t4i` " +
			"direct/local/aggregate-local-alias/aggregate-field-access/ag" +
			"gregate-field-local-alias resource return, " +
			"assignment/let/direct-if-let/direct-match/field-local/if-let" +
			"/match optional and nested/field-local nested optional " +
			"resource return, typed-error direct/field-local-alias throw," +
			" and rethrow-through-`try` direct/field-local-alias " +
			"provenance stubs"),
	) {
		t.Fatalf(
			"error = %v, want interprocedural generated .t4i resource provenance evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingSelectedTransitiveInterproceduralResourceCasesEvidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("selected same-module/cross-module transitive " +
			"interprocedural resource cases, including task-handle, " +
			"task-group, island, struct-field, enum-payload, " +
			"enum-constructor return, same-module throw/catch " +
			"enum-payload, if-let/match optional-payload, and nested " +
			"struct/enum optional-payload return resource aliases"),
		"selected resource cases",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing selected transitive interprocedural resource cases evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("selected same-module/cross-module transitive " +
			"interprocedural resource cases, including task-handle, " +
			"task-group, island, struct-field, enum-payload, " +
			"enum-constructor return, same-module throw/catch " +
			"enum-payload, if-let/match optional-payload, and nested " +
			"struct/enum optional-payload return resource aliases"),
	) {
		t.Fatalf(
			"error = %v, want selected transitive interprocedural resource cases evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingEnumWholeValueGlobalAssignmentEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module ptr-containing enum whole-value " +
			"global assignment with stable TETRA2102 JSON diagnostic " +
			"evidence"),
		"enum global assignment",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing enum whole-value global assignment evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module ptr-containing enum whole-value " +
			"global assignment with stable TETRA2102 JSON diagnostic " +
			"evidence"),
	) {
		t.Fatalf("error = %v, want enum whole-value global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingGlobalFieldTargetAssignmentEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module global field target assignment " +
			"with stable TETRA2102 JSON diagnostic evidence"),
		"global field assignment",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing global field target assignment evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module global field target assignment " +
			"with stable TETRA2102 JSON diagnostic evidence"),
	) {
		t.Fatalf("error = %v, want global field target assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingAggregateNestedGlobalFieldEscapeEvidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module aggregate and nested-aggregate " +
			"global field escapes with stable TETRA2102 JSON diagnostic " +
			"evidence"),
		"nested global field escapes",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing aggregate/nested-aggregate global field escape evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module aggregate and nested-aggregate " +
			"global field escapes with stable TETRA2102 JSON diagnostic " +
			"evidence"),
	) {
		t.Fatalf(
			"error = %v, want aggregate/nested-aggregate global field escape evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingPtrContainingNestedAggregateCallTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module ptr-containing/nested aggregate " +
			"owned/consume/inout call rejections with stable TETRA2101 " +
			"JSON diagnostic evidence"),
		"ptr aggregate call rejections",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing ptr-containing/nested aggregate call TETRA2101 diagnostic evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module ptr-containing/nested aggregate " +
			"owned/consume/inout call rejections with stable TETRA2101 " +
			"JSON diagnostic evidence"),
	) {
		t.Fatalf(
			"error = %v, want ptr-containing/nested aggregate call TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingPtrEnumPayloadCallTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module ptr enum-payload " +
			"owned/consume/inout call rejections with stable TETRA2101 " +
			"JSON diagnostic evidence"),
		"ptr enum-payload call rejections",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing ptr enum-payload call TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module ptr enum-payload " +
			"owned/consume/inout call rejections with stable TETRA2101 " +
			"JSON diagnostic evidence"),
	) {
		t.Fatalf(
			"error = %v, want ptr enum-payload call TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingPtrOptionalPayloadCallTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module ptr optional-payload " +
			"owned/consume/inout call rejections with stable TETRA2101 " +
			"JSON diagnostic evidence"),
		"ptr optional-payload call rejections",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing ptr optional-payload call TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module ptr optional-payload " +
			"owned/consume/inout call rejections with stable TETRA2101 " +
			"JSON diagnostic evidence"),
	) {
		t.Fatalf(
			"error = %v, want ptr optional-payload call TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingSliceOptionalPayloadCallTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module slice optional-payload " +
			"owned/consume/inout call rejections with stable TETRA2101 " +
			"JSON diagnostic evidence"),
		"slice optional-payload call rejections",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing slice optional-payload call TETRA2101 diagnostic evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module slice optional-payload " +
			"owned/consume/inout call rejections with stable TETRA2101 " +
			"JSON diagnostic evidence"),
	) {
		t.Fatalf(
			"error = %v, want slice optional-payload call TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingImportedDirectPtrAggregateCallTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("imported direct ptr-containing/nested aggregate " +
			"owned/consume/inout call rejections with stable TETRA2101 " +
			"JSON diagnostic evidence"),
		"imported direct ptr aggregate call rejections",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			("expected missing imported direct ptr-containing/nested " +
				"aggregate call TETRA2101 diagnostic evidence failure"),
		)
	}
	if !strings.Contains(
		err.Error(),
		("imported direct ptr-containing/nested aggregate " +
			"owned/consume/inout call rejections with stable TETRA2101 " +
			"JSON diagnostic evidence"),
	) {
		t.Fatalf(
			("error = %v, want imported direct ptr-containing/nested " +
				"aggregate call TETRA2101 diagnostic evidence failure"),
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingFunctionTypedSliceAggregateCallbackTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("function-typed value/struct-field/enum-payload callback " +
			"slice-containing struct/enum owned/consume/inout call " +
			"rejections with stable TETRA2101 JSON diagnostic evidence"),
		"function-typed slice callback rejections",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing function-typed slice aggregate callback TETRA2101 diagnostic evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("function-typed value/struct-field/enum-payload callback " +
			"slice-containing struct/enum owned/consume/inout call " +
			"rejections with stable TETRA2101 JSON diagnostic evidence"),
	) {
		t.Fatalf(
			"error = %v, want function-typed slice aggregate callback TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingFunctionTypedOptionalPtrCallbackTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module function-typed " +
			"value/struct-field/enum-payload optional-ptr " +
			"owned/consume/inout callback diagnostics with stable " +
			"TETRA2101 CLI JSON evidence"),
		"function-typed optional-ptr callback diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing function-typed optional-ptr callback TETRA2101 diagnostic evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module function-typed " +
			"value/struct-field/enum-payload optional-ptr " +
			"owned/consume/inout callback diagnostics with stable " +
			"TETRA2101 CLI JSON evidence"),
	) {
		t.Fatalf(
			"error = %v, want function-typed optional-ptr callback TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingGenericAggregateOptionalPtrTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module generic aggregate and optional-ptr " +
			"owned/consume/inout instantiations including " +
			"slice-containing struct/enum aggregate instantiations with " +
			"stable TETRA2101 CLI JSON evidence"),
		"generic aggregate optional-ptr diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing generic aggregate optional-ptr TETRA2101 diagnostic evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module generic aggregate and optional-ptr " +
			"owned/consume/inout instantiations including " +
			"slice-containing struct/enum aggregate instantiations with " +
			"stable TETRA2101 CLI JSON evidence"),
	) {
		t.Fatalf(
			"error = %v, want generic aggregate optional-ptr TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingGenericBorrowReturnTETRA2102Evidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module generic " +
			"borrow-aggregate/optional-ptr return diagnostics with " +
			"stable TETRA2102 CLI JSON evidence"),
		"generic borrow-return diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing generic borrow-return TETRA2102 diagnostic evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module generic " +
			"borrow-aggregate/optional-ptr return diagnostics with " +
			"stable TETRA2102 CLI JSON evidence"),
	) {
		t.Fatalf(
			"error = %v, want generic borrow-return TETRA2102 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingGenericFunctionTypedGlobalOwnershipMarkerEvidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"generic function-typed global consume-marker preservation and ownership mismatch diagnostics",
		"generic function markers",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing generic function-typed global ownership-marker evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		"generic function-typed global consume-marker preservation and ownership mismatch diagnostics",
	) {
		t.Fatalf(
			"error = %v, want generic function-typed global ownership-marker evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingGenericResourceAliasTETRA2101Evidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module monomorphized generic struct " +
			"task-handle/task-group/island resource aliases with stable " +
			"TETRA2101 CLI JSON evidence"),
		"generic resource aliases",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing generic resource alias TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module monomorphized generic struct " +
			"task-handle/task-group/island resource aliases with stable " +
			"TETRA2101 CLI JSON evidence"),
	) {
		t.Fatalf(
			"error = %v, want generic resource alias TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingGenericActorAndResourceAliasTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module monomorphized generic struct actor " +
			"consume alias diagnostics plus same-module/cross-module " +
			"monomorphized generic struct task-handle/task-group/island " +
			"resource aliases with stable TETRA2101 CLI JSON evidence"),
		"generic actor/resource aliases",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing generic actor/resource alias TETRA2101 diagnostic evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module monomorphized generic struct actor " +
			"consume alias diagnostics plus same-module/cross-module " +
			"monomorphized generic struct task-handle/task-group/island " +
			"resource aliases with stable TETRA2101 CLI JSON evidence"),
	) {
		t.Fatalf(
			"error = %v, want generic actor/resource alias TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingAliasRowGenericActorResourceAliasEvidence(
	t *testing.T,
) {
	want := ("same-module/cross-module monomorphized generic struct actor " +
		"consume aliases, same-module/cross-module monomorphized " +
		"generic struct task-handle/task-group/island resource " +
		"aliases with stable TETRA2101 CLI JSON evidence")
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		want,
		"generic actor/resource aliases",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing alias-row generic actor/resource alias evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want alias-row generic actor/resource alias evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingTransitiveResourceAliasTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module transitive interprocedural " +
			"task-handle/task-group/island resource aliases with stable " +
			"TETRA2101 CLI JSON evidence"),
		"transitive resource aliases",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing transitive resource alias TETRA2101 diagnostic evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module transitive interprocedural " +
			"task-handle/task-group/island resource aliases with stable " +
			"TETRA2101 CLI JSON evidence"),
	) {
		t.Fatalf(
			"error = %v, want transitive resource alias TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingEnumConstructorReturnResourceAliasTETRA2101Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module enum-constructor return resource " +
			"aliases with stable TETRA2101 CLI JSON evidence"),
		"enum-constructor resource aliases",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing enum-constructor return resource alias TETRA2101 diagnostic evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module enum-constructor return resource " +
			"aliases with stable TETRA2101 CLI JSON evidence"),
	) {
		t.Fatalf(
			"error = %v, want enum-constructor return resource alias TETRA2101 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingGenericProtocolMismatchTETRA2001Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module generic protocol requirement " +
			"parameter ownership mismatch diagnostics with stable " +
			"TETRA2001 JSON diagnostic evidence"),
		"generic protocol ownership mismatch diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			("expected missing generic protocol requirement ownership " +
				"mismatch TETRA2001 diagnostic evidence failure"),
		)
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module generic protocol requirement " +
			"parameter ownership mismatch diagnostics with stable " +
			"TETRA2001 JSON diagnostic evidence"),
	) {
		t.Fatalf(
			("error = %v, want generic protocol requirement ownership " +
				"mismatch TETRA2001 diagnostic evidence failure"),
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingProtocolImplOwnershipMismatchTETRA2001Evidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("same-module/cross-module protocol impl parameter ownership " +
			"mismatch diagnostics with stable TETRA2001 CLI JSON evidence"),
		"protocol impl ownership mismatch diagnostics",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing protocol impl ownership mismatch TETRA2001 diagnostic evidence failure",
		)
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module protocol impl parameter ownership " +
			"mismatch diagnostics with stable TETRA2001 CLI JSON evidence"),
	) {
		t.Fatalf(
			"error = %v, want protocol impl ownership mismatch TETRA2001 diagnostic evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingProtocolParameterOwnershipMatchingEvidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"same-module/cross-module protocol parameter ownership matching plus ",
		"",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing protocol parameter ownership matching evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("same-module/cross-module protocol parameter ownership " +
			"matching plus same-module/cross-module protocol impl " +
			"parameter ownership mismatch diagnostics with stable " +
			"TETRA2001 CLI JSON evidence"),
	) {
		t.Fatalf("error = %v, want protocol parameter ownership matching evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingT4IFunctionTypedLocalAliasGlobalStorageEvidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("generated `.t4i` function-typed parameter local-alias " +
			"return metadata for interface-only global-storage " +
			"diagnostics"),
		"generated callable metadata",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing .t4i function-typed local-alias global-storage evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("generated `.t4i` function-typed parameter local-alias " +
			"return metadata for interface-only global-storage " +
			"diagnostics"),
	) {
		t.Fatalf(
			"error = %v, want .t4i function-typed local-alias global-storage evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingFunctionTypeOwnershipMarkerEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("function type ownership markers parse/format plus " +
			"function-typed callable ownership-marker diagnostics"),
		"callable ownership metadata",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing function type ownership-marker evidence failure")
	}
	if !strings.Contains(
		err.Error(),
		("function type ownership markers parse/format plus " +
			"function-typed callable ownership-marker diagnostics"),
	) {
		t.Fatalf("error = %v, want function type ownership-marker evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingMoveAndDropDiagnosticEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"use-after-move/use-after-consume",
		"use-after-consume",
		1,
	)
	audit = strings.Replace(audit, "double-drop/double-finalization", "double-finalization", 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing move/drop diagnostic evidence failure")
	}
	if !strings.Contains(err.Error(), "use-after-move/use-after-consume") {
		t.Fatalf("error = %v, want use-after-move diagnostic evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingPartialStructEnumWholeCopyEvidence(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"partial struct/enum whole-copy rejection",
		"partial copy rejection",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing partial struct/enum whole-copy evidence failure")
	}
	if !strings.Contains(err.Error(), "partial struct/enum whole-copy rejection") {
		t.Fatalf("error = %v, want partial struct/enum whole-copy evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingPartialStructEnumEnumConstructorEvidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"partial struct/enum enum-constructor rejection",
		"partial enum-constructor rejection",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing partial struct/enum enum-constructor evidence failure")
	}
	if !strings.Contains(err.Error(), "partial struct/enum enum-constructor rejection") {
		t.Fatalf("error = %v, want partial struct/enum enum-constructor evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableOptionalPayloadWholeValueEvidence(t *testing.T) {
	want := "optional payload consume/free whole-value rejection"
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"partial struct/enum enum-constructor rejection, borrow escape",
		"partial struct/enum enum-constructor rejection, "+want+", borrow escape",
		1,
	)
	audit = strings.Replace(audit, want, "optional payload diagnostics", 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable optional payload whole-value evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable optional payload whole-value evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableActorTaskUseAfterTransferEvidence(t *testing.T) {
	want := "actor/task use-after-transfer"
	actorAggregateEvidence := ("same-module/cross-module actor struct-field/enum-payload " +
		"alias transfer diagnostics with stable TETRA2101 JSON " +
		"diagnostic evidence")
	taskHandleAggregateEvidence := ("same-module/cross-module task-handle " +
		"struct-field/enum-payload alias transfer diagnostics with " +
		"stable TETRA2101 JSON diagnostic evidence")
	taskHandleUseAfterTransferJoinEvidence := ("task-handle struct-field/enum-payload alias " +
		"use-after-transfer/join")
	taskGroupUseAfterCloseEvidence := "task-group use-after-close"
	branchActorConsumeReuseEvidence := ("branch/match/loop actor consume reuse with stable branch " +
		"actor CLI JSON")
	maybeConsumedJoinsEvidence := "maybe-consumed joins"
	resourceMergeDiagnosticsEvidence := ("branch/match/loop task-handle maybe-joined, task-group " +
		"maybe-closed, and island maybe-freed merge diagnostics")
	resourceFinalizationMergeEvidence := (("branch/match/loop resource finalization merge " +
		"diagnostics ") +
		"with stable TETRA2101 JSON evidence")
	stableWithWant := joinAuditEvidence(
		"partial struct/enum enum-constructor rejection",
		"optional payload consume/free whole-value rejection",
		"actor/task use-after-transfer",
		actorAggregateEvidence,
		taskHandleAggregateEvidence,
		taskHandleUseAfterTransferJoinEvidence,
		taskGroupUseAfterCloseEvidence,
		branchActorConsumeReuseEvidence,
		maybeConsumedJoinsEvidence,
		resourceMergeDiagnosticsEvidence,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	stableWithoutWant := joinAuditEvidence(
		"partial struct/enum enum-constructor rejection",
		"optional payload consume/free whole-value rejection",
		actorAggregateEvidence,
		taskHandleAggregateEvidence,
		taskHandleUseAfterTransferJoinEvidence,
		taskGroupUseAfterCloseEvidence,
		branchActorConsumeReuseEvidence,
		maybeConsumedJoinsEvidence,
		resourceMergeDiagnosticsEvidence,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit := validBlockedOwnershipAudit()
	updated := strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	if updated == audit {
		previousStableWithWant := ("partial struct/enum enum-constructor rejection, optional " +
			"payload consume/free whole-value rejection, actor/task " +
			"use-after-transfer, ") + actorAggregateEvidence + ", borrow escape"
		previousStableWithoutWant := ("partial struct/enum enum-constructor rejection, optional " +
			"payload consume/free whole-value rejection, ") + actorAggregateEvidence + ", borrow escape"
		updated = strings.Replace(audit, previousStableWithWant, previousStableWithoutWant, 1)
	}
	if updated == audit {
		oldStableWithWant := ("partial struct/enum enum-constructor rejection, optional " +
			"payload consume/free whole-value rejection, actor/task " +
			"use-after-transfer, borrow escape")
		oldStableWithoutWant := ("partial struct/enum enum-constructor rejection, optional " +
			"payload consume/free whole-value rejection, borrow escape")
		updated = strings.Replace(audit, oldStableWithWant, oldStableWithoutWant, 1)
	}
	err := validateOwnershipAudit(
		[]byte(updated),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable actor/task use-after-transfer evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable actor/task use-after-transfer evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableActorAggregateTransferEvidence(t *testing.T) {
	want := ("same-module/cross-module actor struct-field/enum-payload " +
		"alias transfer diagnostics with stable TETRA2101 JSON " +
		"diagnostic evidence")
	taskHandleAggregateEvidence := ("same-module/cross-module task-handle " +
		"struct-field/enum-payload alias transfer diagnostics with " +
		"stable TETRA2101 JSON diagnostic evidence")
	taskHandleUseAfterTransferJoinEvidence := ("task-handle struct-field/enum-payload alias " +
		"use-after-transfer/join")
	taskGroupUseAfterCloseEvidence := "task-group use-after-close"
	branchActorConsumeReuseEvidence := ("branch/match/loop actor consume reuse with stable branch " +
		"actor CLI JSON")
	maybeConsumedJoinsEvidence := "maybe-consumed joins"
	resourceMergeDiagnosticsEvidence := ("branch/match/loop task-handle maybe-joined, task-group " +
		"maybe-closed, and island maybe-freed merge diagnostics")
	resourceFinalizationMergeEvidence := (("branch/match/loop resource finalization merge " +
		"diagnostics ") +
		"with stable TETRA2101 JSON evidence")
	stableWithWant := joinAuditEvidence(
		"optional payload consume/free whole-value rejection",
		"actor/task use-after-transfer",
		want,
		taskHandleAggregateEvidence,
		taskHandleUseAfterTransferJoinEvidence,
		taskGroupUseAfterCloseEvidence,
		branchActorConsumeReuseEvidence,
		maybeConsumedJoinsEvidence,
		resourceMergeDiagnosticsEvidence,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		oldStableWithoutWant := ("optional payload consume/free whole-value rejection, " +
			"actor/task use-after-transfer, borrow escape")
		oldStableWithWant := ("optional payload consume/free whole-value rejection, " +
			"actor/task use-after-transfer, ") + want + ", borrow escape"
		audit = strings.Replace(audit, oldStableWithoutWant, oldStableWithWant, 1)
	}
	replacement := joinAuditEvidence(
		"optional payload consume/free whole-value rejection",
		"actor/task use-after-transfer",
		"actor aggregate transfer diagnostics",
		taskHandleAggregateEvidence,
		taskHandleUseAfterTransferJoinEvidence,
		taskGroupUseAfterCloseEvidence,
		branchActorConsumeReuseEvidence,
		maybeConsumedJoinsEvidence,
		resourceMergeDiagnosticsEvidence,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit = strings.Replace(audit, stableWithWant, replacement, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable actor aggregate transfer evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable actor aggregate transfer evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableTaskHandleAggregateTransferEvidence(
	t *testing.T,
) {
	actorAggregateEvidence := ("same-module/cross-module actor struct-field/enum-payload " +
		"alias transfer diagnostics with stable TETRA2101 JSON " +
		"diagnostic evidence")
	want := ("same-module/cross-module task-handle " +
		"struct-field/enum-payload alias transfer diagnostics with " +
		"stable TETRA2101 JSON diagnostic evidence")
	taskHandleUseAfterTransferJoinEvidence := ("task-handle struct-field/enum-payload alias " +
		"use-after-transfer/join")
	taskGroupUseAfterCloseEvidence := "task-group use-after-close"
	branchActorConsumeReuseEvidence := ("branch/match/loop actor consume reuse with stable branch " +
		"actor CLI JSON")
	maybeConsumedJoinsEvidence := "maybe-consumed joins"
	resourceMergeDiagnosticsEvidence := ("branch/match/loop task-handle maybe-joined, task-group " +
		"maybe-closed, and island maybe-freed merge diagnostics")
	resourceFinalizationMergeEvidence := (("branch/match/loop resource finalization merge " +
		"diagnostics ") +
		"with stable TETRA2101 JSON evidence")
	stableWithoutWant := joinAuditEvidence(
		"actor/task use-after-transfer",
		actorAggregateEvidence,
		taskHandleUseAfterTransferJoinEvidence,
		taskGroupUseAfterCloseEvidence,
		branchActorConsumeReuseEvidence,
		maybeConsumedJoinsEvidence,
		resourceMergeDiagnosticsEvidence,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	stableWithWant := joinAuditEvidence(
		"actor/task use-after-transfer",
		actorAggregateEvidence,
		want,
		taskHandleUseAfterTransferJoinEvidence,
		taskGroupUseAfterCloseEvidence,
		branchActorConsumeReuseEvidence,
		maybeConsumedJoinsEvidence,
		resourceMergeDiagnosticsEvidence,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	replacement := joinAuditEvidence(
		"actor/task use-after-transfer",
		actorAggregateEvidence,
		"task-handle aggregate transfer diagnostics",
		taskHandleUseAfterTransferJoinEvidence,
		taskGroupUseAfterCloseEvidence,
		branchActorConsumeReuseEvidence,
		maybeConsumedJoinsEvidence,
		resourceMergeDiagnosticsEvidence,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit = strings.Replace(audit, stableWithWant, replacement, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable task-handle aggregate transfer evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable task-handle aggregate transfer evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableTaskHandleUseAfterTransferJoinEvidence(
	t *testing.T,
) {
	actorAggregateEvidence := ("same-module/cross-module actor struct-field/enum-payload " +
		"alias transfer diagnostics with stable TETRA2101 JSON " +
		"diagnostic evidence")
	taskHandleAggregateEvidence := ("same-module/cross-module task-handle " +
		"struct-field/enum-payload alias transfer diagnostics with " +
		"stable TETRA2101 JSON diagnostic evidence")
	want := "task-handle struct-field/enum-payload alias use-after-transfer/join"
	taskGroupUseAfterCloseEvidence := "task-group use-after-close"
	branchActorConsumeReuseEvidence := ("branch/match/loop actor consume reuse with stable branch " +
		"actor CLI JSON")
	maybeConsumedJoinsEvidence := "maybe-consumed joins"
	resourceMergeDiagnosticsEvidence := ("branch/match/loop task-handle maybe-joined, task-group " +
		"maybe-closed, and island maybe-freed merge diagnostics")
	resourceFinalizationMergeEvidence := (("branch/match/loop resource finalization merge " +
		"diagnostics ") +
		"with stable TETRA2101 JSON evidence")
	stableWithoutWant := joinAuditEvidence(
		"actor/task use-after-transfer",
		actorAggregateEvidence,
		taskHandleAggregateEvidence,
		taskGroupUseAfterCloseEvidence,
		branchActorConsumeReuseEvidence,
		maybeConsumedJoinsEvidence,
		resourceMergeDiagnosticsEvidence,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	stableWithWant := joinAuditEvidence(
		"actor/task use-after-transfer",
		actorAggregateEvidence,
		taskHandleAggregateEvidence,
		want,
		taskGroupUseAfterCloseEvidence,
		branchActorConsumeReuseEvidence,
		maybeConsumedJoinsEvidence,
		resourceMergeDiagnosticsEvidence,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	replacement := joinAuditEvidence(
		"actor/task use-after-transfer",
		actorAggregateEvidence,
		taskHandleAggregateEvidence,
		"task-handle alias diagnostics",
		taskGroupUseAfterCloseEvidence,
		branchActorConsumeReuseEvidence,
		maybeConsumedJoinsEvidence,
		resourceMergeDiagnosticsEvidence,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit = strings.Replace(audit, stableWithWant, replacement, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable task-handle use-after-transfer/join evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf(
			"error = %v, want stable task-handle use-after-transfer/join evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableTaskGroupUseAfterCloseEvidence(t *testing.T) {
	taskHandleUseAfterTransferJoinEvidence := ("task-handle struct-field/enum-payload alias " +
		"use-after-transfer/join")
	want := "task-group use-after-close"
	branchActorConsumeReuseEvidence := ("branch/match/loop actor consume reuse with stable branch " +
		"actor CLI JSON")
	maybeConsumedJoinsEvidence := "maybe-consumed joins"
	resourceMergeDiagnosticsEvidence := ("branch/match/loop task-handle maybe-joined, task-group " +
		"maybe-closed, and island maybe-freed merge diagnostics")
	resourceFinalizationMergeEvidence := (("branch/match/loop resource finalization merge " +
		"diagnostics ") +
		"with stable TETRA2101 JSON evidence")
	stableWithoutWant := joinAuditEvidence(
		taskHandleUseAfterTransferJoinEvidence,
		branchActorConsumeReuseEvidence,
		maybeConsumedJoinsEvidence,
		resourceMergeDiagnosticsEvidence,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	stableWithWant := joinAuditEvidence(
		taskHandleUseAfterTransferJoinEvidence,
		want,
		branchActorConsumeReuseEvidence,
		maybeConsumedJoinsEvidence,
		resourceMergeDiagnosticsEvidence,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	replacement := joinAuditEvidence(
		taskHandleUseAfterTransferJoinEvidence,
		"task-group close diagnostics",
		branchActorConsumeReuseEvidence,
		maybeConsumedJoinsEvidence,
		resourceMergeDiagnosticsEvidence,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit = strings.Replace(audit, stableWithWant, replacement, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable task-group use-after-close evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable task-group use-after-close evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableActorBranchMatchLoopConsumeReuseEvidence(
	t *testing.T,
) {
	taskGroupUseAfterCloseEvidence := "task-group use-after-close"
	want := "branch/match/loop actor consume reuse with stable branch actor CLI JSON"
	maybeConsumedJoinsEvidence := "maybe-consumed joins"
	resourceMergeDiagnosticsEvidence := ("branch/match/loop task-handle maybe-joined, task-group " +
		"maybe-closed, and island maybe-freed merge diagnostics")
	resourceFinalizationMergeEvidence := (("branch/match/loop resource finalization merge " +
		"diagnostics ") +
		"with stable TETRA2101 JSON evidence")
	stableWithoutWant := joinAuditEvidence(
		taskGroupUseAfterCloseEvidence,
		maybeConsumedJoinsEvidence,
		resourceMergeDiagnosticsEvidence,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	stableWithWant := joinAuditEvidence(
		taskGroupUseAfterCloseEvidence,
		want,
		maybeConsumedJoinsEvidence,
		resourceMergeDiagnosticsEvidence,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	replacement := joinAuditEvidence(
		taskGroupUseAfterCloseEvidence,
		"branch actor diagnostics",
		maybeConsumedJoinsEvidence,
		resourceMergeDiagnosticsEvidence,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit = strings.Replace(audit, stableWithWant, replacement, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable actor branch/match/loop consume reuse evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf(
			"error = %v, want stable actor branch/match/loop consume reuse evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableMaybeConsumedJoinsEvidence(t *testing.T) {
	branchActorConsumeReuseEvidence := ("branch/match/loop actor consume reuse with stable branch " +
		"actor CLI JSON")
	want := "maybe-consumed joins"
	resourceMergeDiagnosticsEvidence := ("branch/match/loop task-handle maybe-joined, task-group " +
		"maybe-closed, and island maybe-freed merge diagnostics")
	resourceFinalizationMergeEvidence := (("branch/match/loop resource finalization merge " +
		"diagnostics ") +
		"with stable TETRA2101 JSON evidence")
	stableWithoutWant := joinAuditEvidence(
		branchActorConsumeReuseEvidence,
		resourceMergeDiagnosticsEvidence,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	stableWithWant := joinAuditEvidence(
		branchActorConsumeReuseEvidence,
		want,
		resourceMergeDiagnosticsEvidence,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	replacement := joinAuditEvidence(
		branchActorConsumeReuseEvidence,
		"maybe-consumed diagnostics",
		resourceMergeDiagnosticsEvidence,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit = strings.Replace(audit, stableWithWant, replacement, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable maybe-consumed joins evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable maybe-consumed joins evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableResourceMergeDiagnosticsEvidence(t *testing.T) {
	maybeConsumedJoinsEvidence := "maybe-consumed joins"
	want := ("branch/match/loop task-handle maybe-joined, task-group " +
		"maybe-closed, and island maybe-freed merge diagnostics")
	resourceFinalizationMergeEvidence := (("branch/match/loop resource finalization merge " +
		"diagnostics ") +
		"with stable TETRA2101 JSON evidence")
	stableWithoutWant := joinAuditEvidence(
		maybeConsumedJoinsEvidence,
		resourceFinalizationMergeEvidence,
		"borrow escape",
	)
	stableWithWant := joinAuditEvidence(
		maybeConsumedJoinsEvidence,
		want,
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	replacement := joinAuditEvidence(
		maybeConsumedJoinsEvidence,
		"resource merge diagnostics",
	) + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit = strings.Replace(audit, stableWithWant, replacement, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable resource merge diagnostics evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable resource merge diagnostics evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableResourceFinalizationMergeTETRA2101Evidence(
	t *testing.T,
) {
	resourceMergeDiagnosticsEvidence := ("branch/match/loop task-handle maybe-joined, task-group " +
		"maybe-closed, and island maybe-freed merge diagnostics")
	want := ("branch/match/loop resource finalization merge diagnostics " +
		"with stable TETRA2101 JSON evidence")
	stableWithoutWant := resourceMergeDiagnosticsEvidence + ", borrow escape"
	stableWithWant := resourceMergeDiagnosticsEvidence + "; " + want + ", borrow escape"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(
		audit,
		stableWithWant,
		resourceMergeDiagnosticsEvidence+"; resource finalization merge diagnostics, borrow escape",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable resource finalization merge TETRA2101 evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf(
			"error = %v, want stable resource finalization merge TETRA2101 evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableBorrowEscapeEvidence(t *testing.T) {
	resourceFinalizationMergeEvidence := (("branch/match/loop resource finalization merge " +
		"diagnostics ") +
		"with stable TETRA2101 JSON evidence")
	want := "borrow escape"
	aliasConflictsEvidence := "alias conflicts"
	stableWithoutWant := resourceFinalizationMergeEvidence + ", " + aliasConflictsEvidence
	stableWithWant := resourceFinalizationMergeEvidence + ", " + want + ", " + aliasConflictsEvidence
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(
		audit,
		stableWithWant,
		resourceFinalizationMergeEvidence+", borrow diagnostics, "+aliasConflictsEvidence,
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable borrow escape evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable borrow escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableAliasConflictsEvidence(t *testing.T) {
	want := "alias conflicts"
	resourceLifecycleEvidence := "use-after-free/join/close"
	stableWithoutWant := "borrow escape, resource use-after-free/double-join/ambiguous-provenance"
	stableWithWant := joinAuditEvidence(
		"borrow escape",
		want,
		resourceLifecycleEvidence,
	) + ", resource use-after-free/double-join/ambiguous-provenance"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(
		audit,
		stableWithWant,
		"borrow escape, alias diagnostics, "+resourceLifecycleEvidence+(", resource use-after-"+
			"free/double-join/ambiguous-provenance"),
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable alias conflicts evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable alias conflicts evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableUseAfterFreeJoinCloseEvidence(t *testing.T) {
	want := "use-after-free/join/close"
	stableWithoutWant := "alias conflicts, resource use-after-free/double-join/ambiguous-provenance"
	stableWithWant := "alias conflicts, " + want + (", resource use-after-free/double-join/" +
		"ambiguous-provenance")
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(
		audit,
		stableWithWant,
		("alias conflicts, resource lifecycle diagnostics, resource " +
			"use-after-free/double-join/ambiguous-provenance"),
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable use-after-free/join/close evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable use-after-free/join/close evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableCallableEscapeDiagnosticsEvidence(t *testing.T) {
	want := "callable escape diagnostics"
	cliJSONEvidence := "CLI JSON ownership/lifetime safety codes"
	fixedArrayBorrowEscapeEvidence := ("borrow-escape including fixed-array alias return/global " +
		"assignment/optional global assignment/inout assignment")
	borrowedStringEvidence := "borrowed string alias return/global assignment"
	sliceStructReturnInoutEvidence := ("slice-containing struct literal/alias/nested " +
		"struct/enum-payload return and inout assignment escapes")
	sliceEnumReturnEvidence := "slice-containing enum direct/alias return escape CLI JSON evidence"
	sliceStructEnumCallEvidence := ("slice-containing struct/enum owned/consume/inout call " +
		"escape CLI JSON evidence including imported direct cases")
	genericBorrowReturnEvidence := ("same-module/cross-module generic " +
		"borrow-aggregate/optional-ptr return diagnostics with " +
		"stable TETRA2102 CLI JSON evidence")
	functionTypedOptionalPtrCallbackEvidence := ("same-module/cross-module function-typed " +
		"value/struct-field/enum-payload optional-ptr " +
		"owned/consume/inout callback diagnostics with stable " +
		"TETRA2101 CLI JSON evidence")
	functionTypedSliceCallbackEvidence := ("function-typed value/struct-field/enum-payload callback " +
		"slice-containing struct/enum owned/consume/inout call " +
		"rejections with stable TETRA2101 JSON diagnostic evidence")
	optionalAssignmentEvidence := ("ptr/slice optional assignment return/owned/consume/inout " +
		"escape with stable same-module/cross-module slice optional " +
		"assignment return/owned/consume/inout CLI JSON evidence")
	stableWithoutWant := joinAuditEvidence(
		"double-drop/double-finalization",
		cliJSONEvidence+" for "+fixedArrayBorrowEscapeEvidence+" and "+
			borrowedStringEvidence,
		sliceStructReturnInoutEvidence+" plus "+sliceEnumReturnEvidence,
		sliceStructEnumCallEvidence,
		genericBorrowReturnEvidence,
		functionTypedOptionalPtrCallbackEvidence,
		functionTypedSliceCallbackEvidence,
		optionalAssignmentEvidence+" exist",
	)
	stableWithWant := joinAuditEvidence(
		"double-drop/double-finalization",
		want,
		"and "+cliJSONEvidence+" for "+fixedArrayBorrowEscapeEvidence+" and "+
			borrowedStringEvidence,
		sliceStructReturnInoutEvidence+" plus "+sliceEnumReturnEvidence,
		sliceStructEnumCallEvidence,
		genericBorrowReturnEvidence,
		functionTypedOptionalPtrCallbackEvidence,
		functionTypedSliceCallbackEvidence,
		optionalAssignmentEvidence+" exist",
	)
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		oldStableWithoutWant := "double-drop/double-finalization diagnostics exist"
		oldStableWithWant := "double-drop/double-finalization, and " + want + " exist"
		audit = strings.Replace(audit, oldStableWithoutWant, oldStableWithWant, 1)
		stableWithoutWant = oldStableWithoutWant
		stableWithWant = oldStableWithWant
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable callable escape diagnostics evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable callable escape diagnostics evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableCLIJSONOwnershipLifetimeSafetyCodesEvidence(
	t *testing.T,
) {
	want := "CLI JSON ownership/lifetime safety codes"
	fixedArrayBorrowEscapeEvidence := ("borrow-escape including fixed-array alias return/global " +
		"assignment/optional global assignment/inout assignment")
	borrowedStringEvidence := "borrowed string alias return/global assignment"
	sliceStructReturnInoutEvidence := ("slice-containing struct literal/alias/nested " +
		"struct/enum-payload return and inout assignment escapes")
	sliceEnumReturnEvidence := "slice-containing enum direct/alias return escape CLI JSON evidence"
	sliceStructEnumCallEvidence := ("slice-containing struct/enum owned/consume/inout call " +
		"escape CLI JSON evidence including imported direct cases")
	genericBorrowReturnEvidence := ("same-module/cross-module generic " +
		"borrow-aggregate/optional-ptr return diagnostics with " +
		"stable TETRA2102 CLI JSON evidence")
	functionTypedOptionalPtrCallbackEvidence := ("same-module/cross-module function-typed " +
		"value/struct-field/enum-payload optional-ptr " +
		"owned/consume/inout callback diagnostics with stable " +
		"TETRA2101 CLI JSON evidence")
	functionTypedSliceCallbackEvidence := ("function-typed value/struct-field/enum-payload callback " +
		"slice-containing struct/enum owned/consume/inout call " +
		"rejections with stable TETRA2101 JSON diagnostic evidence")
	optionalAssignmentEvidence := ("ptr/slice optional assignment return/owned/consume/inout " +
		"escape with stable same-module/cross-module slice optional " +
		"assignment return/owned/consume/inout CLI JSON evidence")
	stableWithoutWant := joinAuditEvidence(
		"callable escape diagnostics",
		fixedArrayBorrowEscapeEvidence+" and "+borrowedStringEvidence,
		sliceStructReturnInoutEvidence+" plus "+sliceEnumReturnEvidence,
		sliceStructEnumCallEvidence,
		genericBorrowReturnEvidence,
		functionTypedOptionalPtrCallbackEvidence,
		functionTypedSliceCallbackEvidence,
		optionalAssignmentEvidence+" exist",
	)
	stableWithWant := joinAuditEvidence(
		"callable escape diagnostics",
		"and "+want+" for "+fixedArrayBorrowEscapeEvidence+" and "+
			borrowedStringEvidence,
		sliceStructReturnInoutEvidence+" plus "+sliceEnumReturnEvidence,
		sliceStructEnumCallEvidence,
		genericBorrowReturnEvidence,
		functionTypedOptionalPtrCallbackEvidence,
		functionTypedSliceCallbackEvidence,
		optionalAssignmentEvidence+" exist",
	)
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		oldStableWithoutWant := "callable escape diagnostics exist"
		oldStableWithWant := "callable escape diagnostics, and " + want + " exist"
		audit = strings.Replace(audit, oldStableWithoutWant, oldStableWithWant, 1)
		stableWithoutWant = oldStableWithoutWant
		stableWithWant = oldStableWithWant
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing stable CLI JSON ownership/lifetime safety codes evidence failure",
		)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf(
			"error = %v, want stable CLI JSON ownership/lifetime safety codes evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableFixedArrayBorrowEscapeCLIJSONEvidence(
	t *testing.T,
) {
	want := ("borrow-escape including fixed-array alias return/global " +
		"assignment/optional global assignment/inout assignment")
	borrowedStringEvidence := "borrowed string alias return/global assignment"
	sliceStructReturnInoutEvidence := ("slice-containing struct literal/alias/nested " +
		"struct/enum-payload return and inout assignment escapes")
	sliceEnumReturnEvidence := "slice-containing enum direct/alias return escape CLI JSON evidence"
	sliceStructEnumCallEvidence := ("slice-containing struct/enum owned/consume/inout call " +
		"escape CLI JSON evidence including imported direct cases")
	genericBorrowReturnEvidence := ("same-module/cross-module generic " +
		"borrow-aggregate/optional-ptr return diagnostics with " +
		"stable TETRA2102 CLI JSON evidence")
	functionTypedOptionalPtrCallbackEvidence := ("same-module/cross-module function-typed " +
		"value/struct-field/enum-payload optional-ptr " +
		"owned/consume/inout callback diagnostics with stable " +
		"TETRA2101 CLI JSON evidence")
	functionTypedSliceCallbackEvidence := ("function-typed value/struct-field/enum-payload callback " +
		"slice-containing struct/enum owned/consume/inout call " +
		"rejections with stable TETRA2101 JSON diagnostic evidence")
	optionalAssignmentEvidence := ("ptr/slice optional assignment return/owned/consume/inout " +
		"escape with stable same-module/cross-module slice optional " +
		"assignment return/owned/consume/inout CLI JSON evidence")
	stableWithoutWant := joinAuditEvidence(
		"CLI JSON ownership/lifetime safety codes for "+borrowedStringEvidence,
		sliceStructReturnInoutEvidence+" plus "+sliceEnumReturnEvidence,
		sliceStructEnumCallEvidence,
		genericBorrowReturnEvidence,
		functionTypedOptionalPtrCallbackEvidence,
		functionTypedSliceCallbackEvidence,
		optionalAssignmentEvidence+" exist",
	)
	stableWithWant := joinAuditEvidence(
		"CLI JSON ownership/lifetime safety codes for "+want+" and "+
			borrowedStringEvidence,
		sliceStructReturnInoutEvidence+" plus "+sliceEnumReturnEvidence,
		sliceStructEnumCallEvidence,
		genericBorrowReturnEvidence,
		functionTypedOptionalPtrCallbackEvidence,
		functionTypedSliceCallbackEvidence,
		optionalAssignmentEvidence+" exist",
	)
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		oldStableWithoutWant := "CLI JSON ownership/lifetime safety codes exist"
		oldStableWithWant := "CLI JSON ownership/lifetime safety codes for " + want + " exist"
		audit = strings.Replace(audit, oldStableWithoutWant, oldStableWithWant, 1)
		stableWithoutWant = oldStableWithoutWant
		stableWithWant = oldStableWithWant
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable fixed-array borrow-escape CLI JSON evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable fixed-array borrow-escape CLI JSON evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableBorrowedStringEvidence(t *testing.T) {
	fixedArrayBorrowEscapeEvidence := ("borrow-escape including fixed-array alias return/global " +
		"assignment/optional global assignment/inout assignment")
	want := "borrowed string alias return/global assignment"
	sliceStructReturnInoutEvidence := ("slice-containing struct literal/alias/nested " +
		"struct/enum-payload return and inout assignment escapes")
	sliceEnumReturnEvidence := "slice-containing enum direct/alias return escape CLI JSON evidence"
	sliceStructEnumCallEvidence := ("slice-containing struct/enum owned/consume/inout call " +
		"escape CLI JSON evidence including imported direct cases")
	genericBorrowReturnEvidence := ("same-module/cross-module generic " +
		"borrow-aggregate/optional-ptr return diagnostics with " +
		"stable TETRA2102 CLI JSON evidence")
	functionTypedOptionalPtrCallbackEvidence := ("same-module/cross-module function-typed " +
		"value/struct-field/enum-payload optional-ptr " +
		"owned/consume/inout callback diagnostics with stable " +
		"TETRA2101 CLI JSON evidence")
	functionTypedSliceCallbackEvidence := ("function-typed value/struct-field/enum-payload callback " +
		"slice-containing struct/enum owned/consume/inout call " +
		"rejections with stable TETRA2101 JSON diagnostic evidence")
	optionalAssignmentEvidence := ("ptr/slice optional assignment return/owned/consume/inout " +
		"escape with stable same-module/cross-module slice optional " +
		"assignment return/owned/consume/inout CLI JSON evidence")
	stableWithoutWant := joinAuditEvidence(
		fixedArrayBorrowEscapeEvidence,
		sliceStructReturnInoutEvidence+" plus "+sliceEnumReturnEvidence,
		sliceStructEnumCallEvidence,
		genericBorrowReturnEvidence,
		functionTypedOptionalPtrCallbackEvidence,
		functionTypedSliceCallbackEvidence,
		optionalAssignmentEvidence+" exist",
	)
	stableWithWant := joinAuditEvidence(
		fixedArrayBorrowEscapeEvidence+" and "+want,
		sliceStructReturnInoutEvidence+" plus "+sliceEnumReturnEvidence,
		sliceStructEnumCallEvidence,
		genericBorrowReturnEvidence,
		functionTypedOptionalPtrCallbackEvidence,
		functionTypedSliceCallbackEvidence,
		optionalAssignmentEvidence+" exist",
	)
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable borrowed string evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable borrowed string evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableSliceEnumReturnEscapeCLIJSONEvidence(
	t *testing.T,
) {
	borrowedStringEvidence := "borrowed string alias return/global assignment"
	sliceStructReturnInoutEvidence := ("slice-containing struct literal/alias/nested " +
		"struct/enum-payload return and inout assignment escapes")
	want := "slice-containing enum direct/alias return escape CLI JSON evidence"
	sliceStructEnumCallEvidence := ("slice-containing struct/enum owned/consume/inout call " +
		"escape CLI JSON evidence including imported direct cases")
	genericBorrowReturnEvidence := ("same-module/cross-module generic " +
		"borrow-aggregate/optional-ptr return diagnostics with " +
		"stable TETRA2102 CLI JSON evidence")
	functionTypedOptionalPtrCallbackEvidence := ("same-module/cross-module function-typed " +
		"value/struct-field/enum-payload optional-ptr " +
		"owned/consume/inout callback diagnostics with stable " +
		"TETRA2101 CLI JSON evidence")
	functionTypedSliceCallbackEvidence := ("function-typed value/struct-field/enum-payload callback " +
		"slice-containing struct/enum owned/consume/inout call " +
		"rejections with stable TETRA2101 JSON diagnostic evidence")
	optionalAssignmentEvidence := ("ptr/slice optional assignment return/owned/consume/inout " +
		"escape with stable same-module/cross-module slice optional " +
		"assignment return/owned/consume/inout CLI JSON evidence")
	stableWithoutWant := joinAuditEvidence(
		borrowedStringEvidence,
		sliceStructReturnInoutEvidence,
		sliceStructEnumCallEvidence,
		genericBorrowReturnEvidence,
		functionTypedOptionalPtrCallbackEvidence,
		functionTypedSliceCallbackEvidence,
		optionalAssignmentEvidence+" exist",
	)
	stableWithWant := joinAuditEvidence(
		borrowedStringEvidence,
		sliceStructReturnInoutEvidence+" plus "+want,
		sliceStructEnumCallEvidence,
		genericBorrowReturnEvidence,
		functionTypedOptionalPtrCallbackEvidence,
		functionTypedSliceCallbackEvidence,
		optionalAssignmentEvidence+" exist",
	)
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable slice enum return escape CLI JSON evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable slice enum return escape CLI JSON evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableSliceStructReturnInoutCLIJSONEvidence(
	t *testing.T,
) {
	borrowedStringEvidence := "borrowed string alias return/global assignment"
	want := ("slice-containing struct literal/alias/nested " +
		"struct/enum-payload return and inout assignment escapes")
	sliceEnumReturnEvidence := "slice-containing enum direct/alias return escape CLI JSON evidence"
	sliceStructEnumCallEvidence := ("slice-containing struct/enum owned/consume/inout call " +
		"escape CLI JSON evidence including imported direct cases")
	genericBorrowReturnEvidence := ("same-module/cross-module generic " +
		"borrow-aggregate/optional-ptr return diagnostics with " +
		"stable TETRA2102 CLI JSON evidence")
	functionTypedOptionalPtrCallbackEvidence := ("same-module/cross-module function-typed " +
		"value/struct-field/enum-payload optional-ptr " +
		"owned/consume/inout callback diagnostics with stable " +
		"TETRA2101 CLI JSON evidence")
	functionTypedSliceCallbackEvidence := ("function-typed value/struct-field/enum-payload callback " +
		"slice-containing struct/enum owned/consume/inout call " +
		"rejections with stable TETRA2101 JSON diagnostic evidence")
	optionalAssignmentEvidence := ("ptr/slice optional assignment return/owned/consume/inout " +
		"escape with stable same-module/cross-module slice optional " +
		"assignment return/owned/consume/inout CLI JSON evidence")
	stableWithoutWant := joinAuditEvidence(
		borrowedStringEvidence+" plus "+sliceEnumReturnEvidence,
		sliceStructEnumCallEvidence,
		genericBorrowReturnEvidence,
		functionTypedOptionalPtrCallbackEvidence,
		functionTypedSliceCallbackEvidence,
		optionalAssignmentEvidence+" exist",
	)
	stableWithWant := joinAuditEvidence(
		borrowedStringEvidence,
		want+" plus "+sliceEnumReturnEvidence,
		sliceStructEnumCallEvidence,
		genericBorrowReturnEvidence,
		functionTypedOptionalPtrCallbackEvidence,
		functionTypedSliceCallbackEvidence,
		optionalAssignmentEvidence+" exist",
	)
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable slice struct return/inout CLI JSON evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable slice struct return/inout CLI JSON evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableSliceStructEnumCallCLIJSONEvidence(
	t *testing.T,
) {
	sliceEnumReturnEvidence := "slice-containing enum direct/alias return escape CLI JSON evidence"
	want := ("slice-containing struct/enum owned/consume/inout call " +
		"escape CLI JSON evidence including imported direct cases")
	genericBorrowReturnEvidence := ("same-module/cross-module generic " +
		"borrow-aggregate/optional-ptr return diagnostics with " +
		"stable TETRA2102 CLI JSON evidence")
	functionTypedOptionalPtrCallbackEvidence := ("same-module/cross-module function-typed " +
		"value/struct-field/enum-payload optional-ptr " +
		"owned/consume/inout callback diagnostics with stable " +
		"TETRA2101 CLI JSON evidence")
	functionTypedSliceCallbackEvidence := ("function-typed value/struct-field/enum-payload callback " +
		"slice-containing struct/enum owned/consume/inout call " +
		"rejections with stable TETRA2101 JSON diagnostic evidence")
	optionalAssignmentEvidence := ("ptr/slice optional assignment return/owned/consume/inout " +
		"escape with stable same-module/cross-module slice optional " +
		"assignment return/owned/consume/inout CLI JSON evidence")
	stableWithoutWant := joinAuditEvidence(
		sliceEnumReturnEvidence,
		genericBorrowReturnEvidence,
		functionTypedOptionalPtrCallbackEvidence,
		functionTypedSliceCallbackEvidence,
		optionalAssignmentEvidence+" exist",
	)
	stableWithWant := joinAuditEvidence(
		sliceEnumReturnEvidence,
		want,
		genericBorrowReturnEvidence,
		functionTypedOptionalPtrCallbackEvidence,
		functionTypedSliceCallbackEvidence,
		optionalAssignmentEvidence+" exist",
	)
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing stable slice struct/enum owned/consume/inout call CLI JSON evidence failure",
		)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf(
			"error = %v, want stable slice struct/enum owned/consume/inout call CLI JSON evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableGenericBorrowReturnCLIJSONEvidence(
	t *testing.T,
) {
	sliceStructEnumCallEvidence := ("slice-containing struct/enum owned/consume/inout call " +
		"escape CLI JSON evidence including imported direct cases")
	want := ("same-module/cross-module generic " +
		"borrow-aggregate/optional-ptr return diagnostics with " +
		"stable TETRA2102 CLI JSON evidence")
	functionTypedOptionalPtrCallbackEvidence := ("same-module/cross-module function-typed " +
		"value/struct-field/enum-payload optional-ptr " +
		"owned/consume/inout callback diagnostics with stable " +
		"TETRA2101 CLI JSON evidence")
	functionTypedSliceCallbackEvidence := ("function-typed value/struct-field/enum-payload callback " +
		"slice-containing struct/enum owned/consume/inout call " +
		"rejections with stable TETRA2101 JSON diagnostic evidence")
	optionalAssignmentEvidence := ("ptr/slice optional assignment return/owned/consume/inout " +
		"escape with stable same-module/cross-module slice optional " +
		"assignment return/owned/consume/inout CLI JSON evidence")
	stableWithoutWant := joinAuditEvidence(
		sliceStructEnumCallEvidence,
		functionTypedOptionalPtrCallbackEvidence,
		functionTypedSliceCallbackEvidence,
		optionalAssignmentEvidence+" exist",
	)
	stableWithWant := joinAuditEvidence(
		sliceStructEnumCallEvidence,
		want,
		functionTypedOptionalPtrCallbackEvidence,
		functionTypedSliceCallbackEvidence,
		optionalAssignmentEvidence+" exist",
	)
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing stable generic borrow aggregate/optional-ptr return CLI JSON evidence failure",
		)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf(
			"error = %v, want stable generic borrow aggregate/optional-ptr return CLI JSON evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableFunctionTypedOptionalPtrCallbackCLIJSONEvidence(
	t *testing.T,
) {
	genericBorrowReturnEvidence := ("same-module/cross-module generic " +
		"borrow-aggregate/optional-ptr return diagnostics with " +
		"stable TETRA2102 CLI JSON evidence")
	want := ("same-module/cross-module function-typed " +
		"value/struct-field/enum-payload optional-ptr " +
		"owned/consume/inout callback diagnostics with stable " +
		"TETRA2101 CLI JSON evidence")
	functionTypedSliceCallbackEvidence := ("function-typed value/struct-field/enum-payload callback " +
		"slice-containing struct/enum owned/consume/inout call " +
		"rejections with stable TETRA2101 JSON diagnostic evidence")
	optionalAssignmentEvidence := ("ptr/slice optional assignment return/owned/consume/inout " +
		"escape with stable same-module/cross-module slice optional " +
		"assignment return/owned/consume/inout CLI JSON evidence")
	stableWithoutWant := joinAuditEvidence(
		genericBorrowReturnEvidence,
		functionTypedSliceCallbackEvidence,
		optionalAssignmentEvidence+" exist",
	)
	stableWithWant := joinAuditEvidence(
		genericBorrowReturnEvidence,
		want,
		functionTypedSliceCallbackEvidence,
		optionalAssignmentEvidence+" exist",
	)
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing stable function-typed optional-ptr callback CLI JSON evidence failure",
		)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf(
			"error = %v, want stable function-typed optional-ptr callback CLI JSON evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableFunctionTypedSliceCallbackCLIJSONEvidence(
	t *testing.T,
) {
	functionTypedOptionalPtrCallbackEvidence := ("same-module/cross-module function-typed " +
		"value/struct-field/enum-payload optional-ptr " +
		"owned/consume/inout callback diagnostics with stable " +
		"TETRA2101 CLI JSON evidence")
	want := ("function-typed value/struct-field/enum-payload callback " +
		"slice-containing struct/enum owned/consume/inout call " +
		"rejections with stable TETRA2101 JSON diagnostic evidence")
	optionalAssignmentEvidence := ("ptr/slice optional assignment return/owned/consume/inout " +
		"escape with stable same-module/cross-module slice optional " +
		"assignment return/owned/consume/inout CLI JSON evidence")
	stableWithoutWant := joinAuditEvidence(
		functionTypedOptionalPtrCallbackEvidence,
		optionalAssignmentEvidence+" exist",
	)
	stableWithWant := joinAuditEvidence(
		functionTypedOptionalPtrCallbackEvidence,
		want,
		optionalAssignmentEvidence+" exist",
	)
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing stable function-typed slice aggregate callback CLI JSON evidence failure",
		)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf(
			"error = %v, want stable function-typed slice aggregate callback CLI JSON evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableOptionalAssignmentCLIJSONEvidence(t *testing.T) {
	functionTypedSliceCallbackEvidence := ("function-typed value/struct-field/enum-payload callback " +
		"slice-containing struct/enum owned/consume/inout call " +
		"rejections with stable TETRA2101 JSON diagnostic evidence")
	want := ("ptr/slice optional assignment return/owned/consume/inout " +
		"escape with stable same-module/cross-module slice optional " +
		"assignment return/owned/consume/inout CLI JSON evidence")
	stableWithoutWant := functionTypedSliceCallbackEvidence + " exist"
	stableWithWant := functionTypedSliceCallbackEvidence + ", " + want + " exist"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable optional assignment CLI JSON evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable optional assignment CLI JSON evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableSliceOptionalPayloadBindingCLIJSONEvidence(
	t *testing.T,
) {
	want := ("same-module/cross-module slice optional payload binding " +
		"owned/consume/inout call, `inout` assignment, and global " +
		"assignment CLI JSON evidence")
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		want,
		"slice optional-payload binding CLI JSON evidence",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable slice optional-payload binding CLI JSON evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf(
			"error = %v, want stable slice optional-payload binding CLI JSON evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableDirectSliceGlobalAssignmentEvidence(
	t *testing.T,
) {
	sliceOptionalPayloadBindingEvidence := (("same-module/cross-module slice optional " +
		"payload binding ") +
		"owned/consume/inout call, `inout` assignment, and global " +
		"assignment CLI JSON evidence")
	want := ("same-module/cross-module direct slice global assignment " +
		"with stable TETRA2102 JSON diagnostic evidence")
	stableWithoutWant := sliceOptionalPayloadBindingEvidence + " exists"
	stableWithWant := sliceOptionalPayloadBindingEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable direct slice global assignment evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable direct slice global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableOptionalPtrGlobalAssignmentEvidence(
	t *testing.T,
) {
	directSliceGlobalEvidence := ("same-module/cross-module direct slice global assignment " +
		"with stable TETRA2102 JSON diagnostic evidence")
	want := ("same-module/cross-module optional ptr global assignment " +
		"with stable TETRA2102 JSON diagnostic evidence")
	stableWithoutWant := directSliceGlobalEvidence + " exists"
	stableWithWant := directSliceGlobalEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable optional ptr global assignment evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable optional ptr global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableOptionalAggregateGlobalAssignmentEvidence(
	t *testing.T,
) {
	optionalPtrGlobalEvidence := ("same-module/cross-module optional ptr global assignment " +
		"with stable TETRA2102 JSON diagnostic evidence")
	want := ("same-module/cross-module optional aggregate global " +
		"assignment with stable TETRA2102 JSON diagnostic evidence")
	stableWithoutWant := optionalPtrGlobalEvidence + " exists"
	stableWithWant := optionalPtrGlobalEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable optional aggregate global assignment evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf(
			"error = %v, want stable optional aggregate global assignment evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingStablePtrOptionalIfLetMatchGlobalEscapeEvidence(
	t *testing.T,
) {
	optionalAggregateGlobalEvidence := ("same-module/cross-module optional aggregate global " +
		"assignment with stable TETRA2102 JSON diagnostic evidence")
	want := ("same-module/cross-module ptr optional assignment " +
		"if-let/match global escape with stable TETRA2102 JSON " +
		"diagnostic evidence")
	stableWithoutWant := optionalAggregateGlobalEvidence + " exists"
	stableWithWant := optionalAggregateGlobalEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing stable ptr optional assignment if-let/match global escape evidence failure",
		)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf(
			"error = %v, want stable ptr optional assignment if-let/match global escape evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingStablePtrEnumAliasReturnEvidence(t *testing.T) {
	ptrOptionalGlobalEvidence := ("same-module/cross-module ptr optional assignment " +
		"if-let/match global escape with stable TETRA2102 JSON " +
		"diagnostic evidence")
	want := ("same-module/cross-module ptr enum alias return escape with " +
		"stable TETRA2102 JSON diagnostic evidence")
	stableWithoutWant := ptrOptionalGlobalEvidence + " exists"
	stableWithWant := ptrOptionalGlobalEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable ptr enum alias return evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable ptr enum alias return evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStablePtrAggregateReturnEvidence(t *testing.T) {
	ptrEnumAliasReturnEvidence := ("same-module/cross-module ptr enum alias return escape with " +
		"stable TETRA2102 JSON diagnostic evidence")
	want := ("same-module/cross-module ptr-containing aggregate " +
		"whole/field/alias/nested-field return escapes with stable " +
		"TETRA2102 JSON diagnostic evidence")
	stableWithoutWant := ptrEnumAliasReturnEvidence + " exists"
	stableWithWant := ptrEnumAliasReturnEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable ptr aggregate return evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable ptr aggregate return evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableWholeAggregateGlobalAssignmentEvidence(
	t *testing.T,
) {
	ptrAggregateReturnEvidence := ("same-module/cross-module ptr-containing aggregate " +
		"whole/field/alias/nested-field return escapes with stable " +
		"TETRA2102 JSON diagnostic evidence")
	want := ("same-module/cross-module whole-aggregate global assignment " +
		"with stable TETRA2102 JSON diagnostic evidence")
	stableWithoutWant := ptrAggregateReturnEvidence + " exists"
	stableWithWant := ptrAggregateReturnEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable whole-aggregate global assignment evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable whole-aggregate global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStablePtrEnumWholeValueGlobalAssignmentEvidence(
	t *testing.T,
) {
	wholeAggregateGlobalEvidence := ("same-module/cross-module whole-aggregate global assignment " +
		"with stable TETRA2102 JSON diagnostic evidence")
	want := ("same-module/cross-module ptr-containing enum whole-value " +
		"global assignment with stable TETRA2102 JSON diagnostic " +
		"evidence")
	stableWithoutWant := wholeAggregateGlobalEvidence + " exists"
	stableWithWant := wholeAggregateGlobalEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable ptr enum whole-value global assignment evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf(
			"error = %v, want stable ptr enum whole-value global assignment evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableGlobalFieldTargetAssignmentEvidence(
	t *testing.T,
) {
	ptrEnumWholeValueGlobalEvidence := ("same-module/cross-module ptr-containing enum whole-value " +
		"global assignment with stable TETRA2102 JSON diagnostic " +
		"evidence")
	want := ("same-module/cross-module global field target assignment " +
		"with stable TETRA2102 JSON diagnostic evidence")
	stableWithoutWant := ptrEnumWholeValueGlobalEvidence + " exists"
	stableWithWant := ptrEnumWholeValueGlobalEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable global field target assignment evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable global field target assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableAggregateNestedGlobalFieldEvidence(
	t *testing.T,
) {
	globalFieldTargetEvidence := ("same-module/cross-module global field target assignment " +
		"with stable TETRA2102 JSON diagnostic evidence")
	want := ("same-module/cross-module aggregate and nested-aggregate " +
		"global field escapes with stable TETRA2102 JSON diagnostic " +
		"evidence")
	stableWithoutWant := globalFieldTargetEvidence + " exists"
	stableWithWant := globalFieldTargetEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable aggregate/nested global field evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable aggregate/nested global field evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStablePtrEnumPayloadEscapeEvidence(t *testing.T) {
	aggregateNestedEvidence := ("same-module/cross-module aggregate and nested-aggregate " +
		"global field escapes with stable TETRA2102 JSON diagnostic " +
		"evidence")
	want := ("same-module/cross-module ptr enum-payload " +
		"return/global/inout assignment escapes with stable " +
		"TETRA2102 JSON diagnostic evidence")
	stableWithoutWant := aggregateNestedEvidence + " exists"
	stableWithWant := aggregateNestedEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable ptr enum-payload escape evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable ptr enum-payload escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStablePtrOptionalPayloadEscapeEvidence(t *testing.T) {
	enumPayloadEvidence := ("same-module/cross-module ptr enum-payload " +
		"return/global/inout assignment escapes with stable " +
		"TETRA2102 JSON diagnostic evidence")
	want := ("same-module/cross-module ptr optional-payload " +
		"return/global/inout assignment escapes with stable " +
		"TETRA2102 JSON diagnostic evidence")
	stableWithoutWant := enumPayloadEvidence + " exists"
	stableWithWant := enumPayloadEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable ptr optional-payload escape evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable ptr optional-payload escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableSliceOptionalPayloadEscapeEvidence(
	t *testing.T,
) {
	ptrOptionalPayloadEvidence := ("same-module/cross-module ptr optional-payload " +
		"return/global/inout assignment escapes with stable " +
		"TETRA2102 JSON diagnostic evidence")
	want := ("same-module/cross-module slice optional-payload " +
		"inout/global assignment escapes with stable TETRA2102 JSON " +
		"diagnostic evidence")
	stableWithoutWant := ptrOptionalPayloadEvidence + " exists"
	stableWithWant := ptrOptionalPayloadEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable slice optional-payload escape evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable slice optional-payload escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableNestedSliceEnumPayloadEscapeEvidence(
	t *testing.T,
) {
	sliceOptionalPayloadEvidence := ("same-module/cross-module slice optional-payload " +
		"inout/global assignment escapes with stable TETRA2102 JSON " +
		"diagnostic evidence")
	want := ("same-module/cross-module nested slice enum-payload " +
		"return/inout/global assignment escapes with stable " +
		"TETRA2102 JSON diagnostic evidence")
	stableWithoutWant := sliceOptionalPayloadEvidence + " exists"
	stableWithWant := sliceOptionalPayloadEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable nested slice enum-payload escape evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable nested slice enum-payload escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableNestedSliceStructEscapeEvidence(t *testing.T) {
	nestedSliceEnumPayloadEvidence := ("same-module/cross-module nested slice enum-payload " +
		"return/inout/global assignment escapes with stable " +
		"TETRA2102 JSON diagnostic evidence")
	want := ("same-module/cross-module nested slice struct " +
		"return/inout/global assignment escapes with stable " +
		"TETRA2102 JSON diagnostic evidence")
	stableWithoutWant := nestedSliceEnumPayloadEvidence + " exists"
	stableWithWant := nestedSliceEnumPayloadEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable nested slice struct escape evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable nested slice struct escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStablePtrNestedAggregateCallTETRA2101Evidence(
	t *testing.T,
) {
	nestedSliceStructEvidence := ("same-module/cross-module nested slice struct " +
		"return/inout/global assignment escapes with stable " +
		"TETRA2102 JSON diagnostic evidence")
	want := ("same-module/cross-module ptr-containing/nested aggregate " +
		"owned/consume/inout call rejections with stable TETRA2101 " +
		"JSON diagnostic evidence")
	stableWithoutWant := nestedSliceStructEvidence + " exists"
	stableWithWant := nestedSliceStructEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf(
			"expected missing stable ptr-containing/nested aggregate call TETRA2101 evidence failure",
		)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf(
			"error = %v, want stable ptr-containing/nested aggregate call TETRA2101 evidence failure",
			err,
		)
	}
}

func TestValidateOwnershipAuditRejectsMissingStablePtrEnumPayloadCallTETRA2101Evidence(
	t *testing.T,
) {
	ptrAggregateCallEvidence := ("same-module/cross-module ptr-containing/nested aggregate " +
		"owned/consume/inout call rejections with stable TETRA2101 " +
		"JSON diagnostic evidence")
	want := ("same-module/cross-module ptr enum-payload " +
		"owned/consume/inout call rejections with stable TETRA2101 " +
		"JSON diagnostic evidence")
	stableWithoutWant := ptrAggregateCallEvidence + " exists"
	stableWithWant := ptrAggregateCallEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing stable ptr enum-payload call TETRA2101 evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable ptr enum-payload call TETRA2101 evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingPartialStructEnumConsumeWholeValueEvidence(
	t *testing.T,
) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"partial struct/enum consume whole-value rejection",
		"partial whole-value rejection",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing partial struct/enum consume whole-value evidence failure")
	}
	if !strings.Contains(err.Error(), "partial struct/enum consume whole-value rejection") {
		t.Fatalf("error = %v, want partial struct/enum consume whole-value evidence failure", err)
	}
}
