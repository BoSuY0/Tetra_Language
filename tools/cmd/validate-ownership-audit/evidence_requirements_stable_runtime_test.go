package main

import (
	"strings"
	"testing"
)

func TestValidateOwnershipAuditRejectsMissingStableActorTaskUseAfterTransferEvidence(t *testing.T) {
	want := "actor/task use-after-transfer"
	actorAggregateEvidence := "same-module/cross-module actor struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence"
	taskHandleAggregateEvidence := "same-module/cross-module task-handle struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence"
	taskHandleUseAfterTransferJoinEvidence := "task-handle struct-field/enum-payload alias use-after-transfer/join"
	taskGroupUseAfterCloseEvidence := "task-group use-after-close"
	branchActorConsumeReuseEvidence := "branch/match/loop actor consume reuse with stable branch actor CLI JSON"
	maybeConsumedJoinsEvidence := "maybe-consumed joins"
	resourceMergeDiagnosticsEvidence := "branch/match/loop task-handle maybe-joined, task-group maybe-closed, and island maybe-freed merge diagnostics"
	resourceFinalizationMergeEvidence := "branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence"
	stableWithWant := "partial struct/enum enum-constructor rejection, optional payload consume/free whole-value rejection, actor/task use-after-transfer, " + actorAggregateEvidence + ", " + taskHandleAggregateEvidence + ", " + taskHandleUseAfterTransferJoinEvidence + ", " + taskGroupUseAfterCloseEvidence + ", " + branchActorConsumeReuseEvidence + ", " + maybeConsumedJoinsEvidence + ", " + resourceMergeDiagnosticsEvidence + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	stableWithoutWant := "partial struct/enum enum-constructor rejection, optional payload consume/free whole-value rejection, " + actorAggregateEvidence + ", " + taskHandleAggregateEvidence + ", " + taskHandleUseAfterTransferJoinEvidence + ", " + taskGroupUseAfterCloseEvidence + ", " + branchActorConsumeReuseEvidence + ", " + maybeConsumedJoinsEvidence + ", " + resourceMergeDiagnosticsEvidence + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit := validBlockedOwnershipAudit()
	updated := strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	if updated == audit {
		previousStableWithWant := "partial struct/enum enum-constructor rejection, optional payload consume/free whole-value rejection, actor/task use-after-transfer, " + actorAggregateEvidence + ", borrow escape"
		previousStableWithoutWant := "partial struct/enum enum-constructor rejection, optional payload consume/free whole-value rejection, " + actorAggregateEvidence + ", borrow escape"
		updated = strings.Replace(audit, previousStableWithWant, previousStableWithoutWant, 1)
	}
	if updated == audit {
		oldStableWithWant := "partial struct/enum enum-constructor rejection, optional payload consume/free whole-value rejection, actor/task use-after-transfer, borrow escape"
		oldStableWithoutWant := "partial struct/enum enum-constructor rejection, optional payload consume/free whole-value rejection, borrow escape"
		updated = strings.Replace(audit, oldStableWithWant, oldStableWithoutWant, 1)
	}
	err := validateOwnershipAudit([]byte(updated), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable actor/task use-after-transfer evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable actor/task use-after-transfer evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableActorAggregateTransferEvidence(t *testing.T) {
	want := "same-module/cross-module actor struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence"
	taskHandleAggregateEvidence := "same-module/cross-module task-handle struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence"
	taskHandleUseAfterTransferJoinEvidence := "task-handle struct-field/enum-payload alias use-after-transfer/join"
	taskGroupUseAfterCloseEvidence := "task-group use-after-close"
	branchActorConsumeReuseEvidence := "branch/match/loop actor consume reuse with stable branch actor CLI JSON"
	maybeConsumedJoinsEvidence := "maybe-consumed joins"
	resourceMergeDiagnosticsEvidence := "branch/match/loop task-handle maybe-joined, task-group maybe-closed, and island maybe-freed merge diagnostics"
	resourceFinalizationMergeEvidence := "branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence"
	stableWithWant := "optional payload consume/free whole-value rejection, actor/task use-after-transfer, " + want + ", " + taskHandleAggregateEvidence + ", " + taskHandleUseAfterTransferJoinEvidence + ", " + taskGroupUseAfterCloseEvidence + ", " + branchActorConsumeReuseEvidence + ", " + maybeConsumedJoinsEvidence + ", " + resourceMergeDiagnosticsEvidence + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		oldStableWithoutWant := "optional payload consume/free whole-value rejection, actor/task use-after-transfer, borrow escape"
		oldStableWithWant := "optional payload consume/free whole-value rejection, actor/task use-after-transfer, " + want + ", borrow escape"
		audit = strings.Replace(audit, oldStableWithoutWant, oldStableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, "optional payload consume/free whole-value rejection, actor/task use-after-transfer, actor aggregate transfer diagnostics, "+taskHandleAggregateEvidence+", "+taskHandleUseAfterTransferJoinEvidence+", "+taskGroupUseAfterCloseEvidence+", "+branchActorConsumeReuseEvidence+", "+maybeConsumedJoinsEvidence+", "+resourceMergeDiagnosticsEvidence+"; "+resourceFinalizationMergeEvidence+", borrow escape", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable actor aggregate transfer evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable actor aggregate transfer evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableTaskHandleAggregateTransferEvidence(t *testing.T) {
	actorAggregateEvidence := "same-module/cross-module actor struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence"
	want := "same-module/cross-module task-handle struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence"
	taskHandleUseAfterTransferJoinEvidence := "task-handle struct-field/enum-payload alias use-after-transfer/join"
	taskGroupUseAfterCloseEvidence := "task-group use-after-close"
	branchActorConsumeReuseEvidence := "branch/match/loop actor consume reuse with stable branch actor CLI JSON"
	maybeConsumedJoinsEvidence := "maybe-consumed joins"
	resourceMergeDiagnosticsEvidence := "branch/match/loop task-handle maybe-joined, task-group maybe-closed, and island maybe-freed merge diagnostics"
	resourceFinalizationMergeEvidence := "branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence"
	stableWithoutWant := "actor/task use-after-transfer, " + actorAggregateEvidence + ", " + taskHandleUseAfterTransferJoinEvidence + ", " + taskGroupUseAfterCloseEvidence + ", " + branchActorConsumeReuseEvidence + ", " + maybeConsumedJoinsEvidence + ", " + resourceMergeDiagnosticsEvidence + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	stableWithWant := "actor/task use-after-transfer, " + actorAggregateEvidence + ", " + want + ", " + taskHandleUseAfterTransferJoinEvidence + ", " + taskGroupUseAfterCloseEvidence + ", " + branchActorConsumeReuseEvidence + ", " + maybeConsumedJoinsEvidence + ", " + resourceMergeDiagnosticsEvidence + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, "actor/task use-after-transfer, "+actorAggregateEvidence+", task-handle aggregate transfer diagnostics, "+taskHandleUseAfterTransferJoinEvidence+", "+taskGroupUseAfterCloseEvidence+", "+branchActorConsumeReuseEvidence+", "+maybeConsumedJoinsEvidence+", "+resourceMergeDiagnosticsEvidence+"; "+resourceFinalizationMergeEvidence+", borrow escape", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable task-handle aggregate transfer evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable task-handle aggregate transfer evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableTaskHandleUseAfterTransferJoinEvidence(t *testing.T) {
	actorAggregateEvidence := "same-module/cross-module actor struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence"
	taskHandleAggregateEvidence := "same-module/cross-module task-handle struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence"
	want := "task-handle struct-field/enum-payload alias use-after-transfer/join"
	taskGroupUseAfterCloseEvidence := "task-group use-after-close"
	branchActorConsumeReuseEvidence := "branch/match/loop actor consume reuse with stable branch actor CLI JSON"
	maybeConsumedJoinsEvidence := "maybe-consumed joins"
	resourceMergeDiagnosticsEvidence := "branch/match/loop task-handle maybe-joined, task-group maybe-closed, and island maybe-freed merge diagnostics"
	resourceFinalizationMergeEvidence := "branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence"
	stableWithoutWant := "actor/task use-after-transfer, " + actorAggregateEvidence + ", " + taskHandleAggregateEvidence + ", " + taskGroupUseAfterCloseEvidence + ", " + branchActorConsumeReuseEvidence + ", " + maybeConsumedJoinsEvidence + ", " + resourceMergeDiagnosticsEvidence + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	stableWithWant := "actor/task use-after-transfer, " + actorAggregateEvidence + ", " + taskHandleAggregateEvidence + ", " + want + ", " + taskGroupUseAfterCloseEvidence + ", " + branchActorConsumeReuseEvidence + ", " + maybeConsumedJoinsEvidence + ", " + resourceMergeDiagnosticsEvidence + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, "actor/task use-after-transfer, "+actorAggregateEvidence+", "+taskHandleAggregateEvidence+", task-handle alias diagnostics, "+taskGroupUseAfterCloseEvidence+", "+branchActorConsumeReuseEvidence+", "+maybeConsumedJoinsEvidence+", "+resourceMergeDiagnosticsEvidence+"; "+resourceFinalizationMergeEvidence+", borrow escape", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable task-handle use-after-transfer/join evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable task-handle use-after-transfer/join evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableTaskGroupUseAfterCloseEvidence(t *testing.T) {
	taskHandleUseAfterTransferJoinEvidence := "task-handle struct-field/enum-payload alias use-after-transfer/join"
	want := "task-group use-after-close"
	branchActorConsumeReuseEvidence := "branch/match/loop actor consume reuse with stable branch actor CLI JSON"
	maybeConsumedJoinsEvidence := "maybe-consumed joins"
	resourceMergeDiagnosticsEvidence := "branch/match/loop task-handle maybe-joined, task-group maybe-closed, and island maybe-freed merge diagnostics"
	resourceFinalizationMergeEvidence := "branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence"
	stableWithoutWant := taskHandleUseAfterTransferJoinEvidence + ", " + branchActorConsumeReuseEvidence + ", " + maybeConsumedJoinsEvidence + ", " + resourceMergeDiagnosticsEvidence + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	stableWithWant := taskHandleUseAfterTransferJoinEvidence + ", " + want + ", " + branchActorConsumeReuseEvidence + ", " + maybeConsumedJoinsEvidence + ", " + resourceMergeDiagnosticsEvidence + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, taskHandleUseAfterTransferJoinEvidence+", task-group close diagnostics, "+branchActorConsumeReuseEvidence+", "+maybeConsumedJoinsEvidence+", "+resourceMergeDiagnosticsEvidence+"; "+resourceFinalizationMergeEvidence+", borrow escape", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable task-group use-after-close evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable task-group use-after-close evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableActorBranchMatchLoopConsumeReuseEvidence(t *testing.T) {
	taskGroupUseAfterCloseEvidence := "task-group use-after-close"
	want := "branch/match/loop actor consume reuse with stable branch actor CLI JSON"
	maybeConsumedJoinsEvidence := "maybe-consumed joins"
	resourceMergeDiagnosticsEvidence := "branch/match/loop task-handle maybe-joined, task-group maybe-closed, and island maybe-freed merge diagnostics"
	resourceFinalizationMergeEvidence := "branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence"
	stableWithoutWant := taskGroupUseAfterCloseEvidence + ", " + maybeConsumedJoinsEvidence + ", " + resourceMergeDiagnosticsEvidence + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	stableWithWant := taskGroupUseAfterCloseEvidence + ", " + want + ", " + maybeConsumedJoinsEvidence + ", " + resourceMergeDiagnosticsEvidence + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, taskGroupUseAfterCloseEvidence+", branch actor diagnostics, "+maybeConsumedJoinsEvidence+", "+resourceMergeDiagnosticsEvidence+"; "+resourceFinalizationMergeEvidence+", borrow escape", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable actor branch/match/loop consume reuse evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable actor branch/match/loop consume reuse evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableMaybeConsumedJoinsEvidence(t *testing.T) {
	branchActorConsumeReuseEvidence := "branch/match/loop actor consume reuse with stable branch actor CLI JSON"
	want := "maybe-consumed joins"
	resourceMergeDiagnosticsEvidence := "branch/match/loop task-handle maybe-joined, task-group maybe-closed, and island maybe-freed merge diagnostics"
	resourceFinalizationMergeEvidence := "branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence"
	stableWithoutWant := branchActorConsumeReuseEvidence + ", " + resourceMergeDiagnosticsEvidence + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	stableWithWant := branchActorConsumeReuseEvidence + ", " + want + ", " + resourceMergeDiagnosticsEvidence + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, branchActorConsumeReuseEvidence+", maybe-consumed diagnostics, "+resourceMergeDiagnosticsEvidence+"; "+resourceFinalizationMergeEvidence+", borrow escape", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable maybe-consumed joins evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable maybe-consumed joins evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableResourceMergeDiagnosticsEvidence(t *testing.T) {
	maybeConsumedJoinsEvidence := "maybe-consumed joins"
	want := "branch/match/loop task-handle maybe-joined, task-group maybe-closed, and island maybe-freed merge diagnostics"
	resourceFinalizationMergeEvidence := "branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence"
	stableWithoutWant := maybeConsumedJoinsEvidence + ", " + resourceFinalizationMergeEvidence + ", borrow escape"
	stableWithWant := maybeConsumedJoinsEvidence + ", " + want + "; " + resourceFinalizationMergeEvidence + ", borrow escape"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, maybeConsumedJoinsEvidence+", resource merge diagnostics; "+resourceFinalizationMergeEvidence+", borrow escape", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable resource merge diagnostics evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable resource merge diagnostics evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableResourceFinalizationMergeTETRA2101Evidence(t *testing.T) {
	resourceMergeDiagnosticsEvidence := "branch/match/loop task-handle maybe-joined, task-group maybe-closed, and island maybe-freed merge diagnostics"
	want := "branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence"
	stableWithoutWant := resourceMergeDiagnosticsEvidence + ", borrow escape"
	stableWithWant := resourceMergeDiagnosticsEvidence + "; " + want + ", borrow escape"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, resourceMergeDiagnosticsEvidence+"; resource finalization merge diagnostics, borrow escape", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable resource finalization merge TETRA2101 evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable resource finalization merge TETRA2101 evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableBorrowEscapeEvidence(t *testing.T) {
	resourceFinalizationMergeEvidence := "branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence"
	want := "borrow escape"
	aliasConflictsEvidence := "alias conflicts"
	stableWithoutWant := resourceFinalizationMergeEvidence + ", " + aliasConflictsEvidence
	stableWithWant := resourceFinalizationMergeEvidence + ", " + want + ", " + aliasConflictsEvidence
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, resourceFinalizationMergeEvidence+", borrow diagnostics, "+aliasConflictsEvidence, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
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
	stableWithWant := "borrow escape, " + want + ", " + resourceLifecycleEvidence + ", resource use-after-free/double-join/ambiguous-provenance"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, "borrow escape, alias diagnostics, "+resourceLifecycleEvidence+", resource use-after-free/double-join/ambiguous-provenance", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
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
	stableWithWant := "alias conflicts, " + want + ", resource use-after-free/double-join/ambiguous-provenance"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, "alias conflicts, resource lifecycle diagnostics, resource use-after-free/double-join/ambiguous-provenance", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
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
	fixedArrayBorrowEscapeEvidence := "borrow-escape including fixed-array alias return/global assignment/optional global assignment/inout assignment"
	borrowedStringEvidence := "borrowed string alias return/global assignment"
	sliceStructReturnInoutEvidence := "slice-containing struct literal/alias/nested struct/enum-payload return and inout assignment escapes"
	sliceEnumReturnEvidence := "slice-containing enum direct/alias return escape CLI JSON evidence"
	sliceStructEnumCallEvidence := "slice-containing struct/enum owned/consume/inout call escape CLI JSON evidence including imported direct cases"
	genericBorrowReturnEvidence := "same-module/cross-module generic borrow-aggregate/optional-ptr return diagnostics with stable TETRA2102 CLI JSON evidence"
	functionTypedOptionalPtrCallbackEvidence := "same-module/cross-module function-typed value/struct-field/enum-payload optional-ptr owned/consume/inout callback diagnostics with stable TETRA2101 CLI JSON evidence"
	functionTypedSliceCallbackEvidence := "function-typed value/struct-field/enum-payload callback slice-containing struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence"
	optionalAssignmentEvidence := "ptr/slice optional assignment return/owned/consume/inout escape with stable same-module/cross-module slice optional assignment return/owned/consume/inout CLI JSON evidence"
	stableWithoutWant := "double-drop/double-finalization, " + cliJSONEvidence + " for " + fixedArrayBorrowEscapeEvidence + " and " + borrowedStringEvidence + ", " + sliceStructReturnInoutEvidence + " plus " + sliceEnumReturnEvidence + ", " + sliceStructEnumCallEvidence + ", " + genericBorrowReturnEvidence + ", " + functionTypedOptionalPtrCallbackEvidence + ", " + functionTypedSliceCallbackEvidence + ", " + optionalAssignmentEvidence + " exist"
	stableWithWant := "double-drop/double-finalization, " + want + ", and " + cliJSONEvidence + " for " + fixedArrayBorrowEscapeEvidence + " and " + borrowedStringEvidence + ", " + sliceStructReturnInoutEvidence + " plus " + sliceEnumReturnEvidence + ", " + sliceStructEnumCallEvidence + ", " + genericBorrowReturnEvidence + ", " + functionTypedOptionalPtrCallbackEvidence + ", " + functionTypedSliceCallbackEvidence + ", " + optionalAssignmentEvidence + " exist"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		oldStableWithoutWant := "double-drop/double-finalization diagnostics exist"
		oldStableWithWant := "double-drop/double-finalization, and " + want + " exist"
		audit = strings.Replace(audit, oldStableWithoutWant, oldStableWithWant, 1)
		stableWithoutWant = oldStableWithoutWant
		stableWithWant = oldStableWithWant
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable callable escape diagnostics evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable callable escape diagnostics evidence failure", err)
	}
}
