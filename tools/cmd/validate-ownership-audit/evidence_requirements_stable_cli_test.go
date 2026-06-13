package main

import (
	"strings"
	"testing"
)

func TestValidateOwnershipAuditRejectsMissingStableCLIJSONOwnershipLifetimeSafetyCodesEvidence(t *testing.T) {
	want := "CLI JSON ownership/lifetime safety codes"
	fixedArrayBorrowEscapeEvidence := "borrow-escape including fixed-array alias return/global assignment/optional global assignment/inout assignment"
	borrowedStringEvidence := "borrowed string alias return/global assignment"
	sliceStructReturnInoutEvidence := "slice-containing struct literal/alias/nested struct/enum-payload return and inout assignment escapes"
	sliceEnumReturnEvidence := "slice-containing enum direct/alias return escape CLI JSON evidence"
	sliceStructEnumCallEvidence := "slice-containing struct/enum owned/consume/inout call escape CLI JSON evidence including imported direct cases"
	genericBorrowReturnEvidence := "same-module/cross-module generic borrow-aggregate/optional-ptr return diagnostics with stable TETRA2102 CLI JSON evidence"
	functionTypedOptionalPtrCallbackEvidence := "same-module/cross-module function-typed value/struct-field/enum-payload optional-ptr owned/consume/inout callback diagnostics with stable TETRA2101 CLI JSON evidence"
	functionTypedSliceCallbackEvidence := "function-typed value/struct-field/enum-payload callback slice-containing struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence"
	optionalAssignmentEvidence := "ptr/slice optional assignment return/owned/consume/inout escape with stable same-module/cross-module slice optional assignment return/owned/consume/inout CLI JSON evidence"
	stableWithoutWant := "callable escape diagnostics, " + fixedArrayBorrowEscapeEvidence + " and " + borrowedStringEvidence + ", " + sliceStructReturnInoutEvidence + " plus " + sliceEnumReturnEvidence + ", " + sliceStructEnumCallEvidence + ", " + genericBorrowReturnEvidence + ", " + functionTypedOptionalPtrCallbackEvidence + ", " + functionTypedSliceCallbackEvidence + ", " + optionalAssignmentEvidence + " exist"
	stableWithWant := "callable escape diagnostics, and " + want + " for " + fixedArrayBorrowEscapeEvidence + " and " + borrowedStringEvidence + ", " + sliceStructReturnInoutEvidence + " plus " + sliceEnumReturnEvidence + ", " + sliceStructEnumCallEvidence + ", " + genericBorrowReturnEvidence + ", " + functionTypedOptionalPtrCallbackEvidence + ", " + functionTypedSliceCallbackEvidence + ", " + optionalAssignmentEvidence + " exist"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		oldStableWithoutWant := "callable escape diagnostics exist"
		oldStableWithWant := "callable escape diagnostics, and " + want + " exist"
		audit = strings.Replace(audit, oldStableWithoutWant, oldStableWithWant, 1)
		stableWithoutWant = oldStableWithoutWant
		stableWithWant = oldStableWithWant
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable CLI JSON ownership/lifetime safety codes evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable CLI JSON ownership/lifetime safety codes evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableFixedArrayBorrowEscapeCLIJSONEvidence(t *testing.T) {
	want := "borrow-escape including fixed-array alias return/global assignment/optional global assignment/inout assignment"
	borrowedStringEvidence := "borrowed string alias return/global assignment"
	sliceStructReturnInoutEvidence := "slice-containing struct literal/alias/nested struct/enum-payload return and inout assignment escapes"
	sliceEnumReturnEvidence := "slice-containing enum direct/alias return escape CLI JSON evidence"
	sliceStructEnumCallEvidence := "slice-containing struct/enum owned/consume/inout call escape CLI JSON evidence including imported direct cases"
	genericBorrowReturnEvidence := "same-module/cross-module generic borrow-aggregate/optional-ptr return diagnostics with stable TETRA2102 CLI JSON evidence"
	functionTypedOptionalPtrCallbackEvidence := "same-module/cross-module function-typed value/struct-field/enum-payload optional-ptr owned/consume/inout callback diagnostics with stable TETRA2101 CLI JSON evidence"
	functionTypedSliceCallbackEvidence := "function-typed value/struct-field/enum-payload callback slice-containing struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence"
	optionalAssignmentEvidence := "ptr/slice optional assignment return/owned/consume/inout escape with stable same-module/cross-module slice optional assignment return/owned/consume/inout CLI JSON evidence"
	stableWithoutWant := "CLI JSON ownership/lifetime safety codes for " + borrowedStringEvidence + ", " + sliceStructReturnInoutEvidence + " plus " + sliceEnumReturnEvidence + ", " + sliceStructEnumCallEvidence + ", " + genericBorrowReturnEvidence + ", " + functionTypedOptionalPtrCallbackEvidence + ", " + functionTypedSliceCallbackEvidence + ", " + optionalAssignmentEvidence + " exist"
	stableWithWant := "CLI JSON ownership/lifetime safety codes for " + want + " and " + borrowedStringEvidence + ", " + sliceStructReturnInoutEvidence + " plus " + sliceEnumReturnEvidence + ", " + sliceStructEnumCallEvidence + ", " + genericBorrowReturnEvidence + ", " + functionTypedOptionalPtrCallbackEvidence + ", " + functionTypedSliceCallbackEvidence + ", " + optionalAssignmentEvidence + " exist"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		oldStableWithoutWant := "CLI JSON ownership/lifetime safety codes exist"
		oldStableWithWant := "CLI JSON ownership/lifetime safety codes for " + want + " exist"
		audit = strings.Replace(audit, oldStableWithoutWant, oldStableWithWant, 1)
		stableWithoutWant = oldStableWithoutWant
		stableWithWant = oldStableWithWant
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable fixed-array borrow-escape CLI JSON evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable fixed-array borrow-escape CLI JSON evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableBorrowedStringEvidence(t *testing.T) {
	fixedArrayBorrowEscapeEvidence := "borrow-escape including fixed-array alias return/global assignment/optional global assignment/inout assignment"
	want := "borrowed string alias return/global assignment"
	sliceStructReturnInoutEvidence := "slice-containing struct literal/alias/nested struct/enum-payload return and inout assignment escapes"
	sliceEnumReturnEvidence := "slice-containing enum direct/alias return escape CLI JSON evidence"
	sliceStructEnumCallEvidence := "slice-containing struct/enum owned/consume/inout call escape CLI JSON evidence including imported direct cases"
	genericBorrowReturnEvidence := "same-module/cross-module generic borrow-aggregate/optional-ptr return diagnostics with stable TETRA2102 CLI JSON evidence"
	functionTypedOptionalPtrCallbackEvidence := "same-module/cross-module function-typed value/struct-field/enum-payload optional-ptr owned/consume/inout callback diagnostics with stable TETRA2101 CLI JSON evidence"
	functionTypedSliceCallbackEvidence := "function-typed value/struct-field/enum-payload callback slice-containing struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence"
	optionalAssignmentEvidence := "ptr/slice optional assignment return/owned/consume/inout escape with stable same-module/cross-module slice optional assignment return/owned/consume/inout CLI JSON evidence"
	stableWithoutWant := fixedArrayBorrowEscapeEvidence + ", " + sliceStructReturnInoutEvidence + " plus " + sliceEnumReturnEvidence + ", " + sliceStructEnumCallEvidence + ", " + genericBorrowReturnEvidence + ", " + functionTypedOptionalPtrCallbackEvidence + ", " + functionTypedSliceCallbackEvidence + ", " + optionalAssignmentEvidence + " exist"
	stableWithWant := fixedArrayBorrowEscapeEvidence + " and " + want + ", " + sliceStructReturnInoutEvidence + " plus " + sliceEnumReturnEvidence + ", " + sliceStructEnumCallEvidence + ", " + genericBorrowReturnEvidence + ", " + functionTypedOptionalPtrCallbackEvidence + ", " + functionTypedSliceCallbackEvidence + ", " + optionalAssignmentEvidence + " exist"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable borrowed string evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable borrowed string evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableSliceEnumReturnEscapeCLIJSONEvidence(t *testing.T) {
	borrowedStringEvidence := "borrowed string alias return/global assignment"
	sliceStructReturnInoutEvidence := "slice-containing struct literal/alias/nested struct/enum-payload return and inout assignment escapes"
	want := "slice-containing enum direct/alias return escape CLI JSON evidence"
	sliceStructEnumCallEvidence := "slice-containing struct/enum owned/consume/inout call escape CLI JSON evidence including imported direct cases"
	genericBorrowReturnEvidence := "same-module/cross-module generic borrow-aggregate/optional-ptr return diagnostics with stable TETRA2102 CLI JSON evidence"
	functionTypedOptionalPtrCallbackEvidence := "same-module/cross-module function-typed value/struct-field/enum-payload optional-ptr owned/consume/inout callback diagnostics with stable TETRA2101 CLI JSON evidence"
	functionTypedSliceCallbackEvidence := "function-typed value/struct-field/enum-payload callback slice-containing struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence"
	optionalAssignmentEvidence := "ptr/slice optional assignment return/owned/consume/inout escape with stable same-module/cross-module slice optional assignment return/owned/consume/inout CLI JSON evidence"
	stableWithoutWant := borrowedStringEvidence + ", " + sliceStructReturnInoutEvidence + ", " + sliceStructEnumCallEvidence + ", " + genericBorrowReturnEvidence + ", " + functionTypedOptionalPtrCallbackEvidence + ", " + functionTypedSliceCallbackEvidence + ", " + optionalAssignmentEvidence + " exist"
	stableWithWant := borrowedStringEvidence + ", " + sliceStructReturnInoutEvidence + " plus " + want + ", " + sliceStructEnumCallEvidence + ", " + genericBorrowReturnEvidence + ", " + functionTypedOptionalPtrCallbackEvidence + ", " + functionTypedSliceCallbackEvidence + ", " + optionalAssignmentEvidence + " exist"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable slice enum return escape CLI JSON evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable slice enum return escape CLI JSON evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableSliceStructReturnInoutCLIJSONEvidence(t *testing.T) {
	borrowedStringEvidence := "borrowed string alias return/global assignment"
	want := "slice-containing struct literal/alias/nested struct/enum-payload return and inout assignment escapes"
	sliceEnumReturnEvidence := "slice-containing enum direct/alias return escape CLI JSON evidence"
	sliceStructEnumCallEvidence := "slice-containing struct/enum owned/consume/inout call escape CLI JSON evidence including imported direct cases"
	genericBorrowReturnEvidence := "same-module/cross-module generic borrow-aggregate/optional-ptr return diagnostics with stable TETRA2102 CLI JSON evidence"
	functionTypedOptionalPtrCallbackEvidence := "same-module/cross-module function-typed value/struct-field/enum-payload optional-ptr owned/consume/inout callback diagnostics with stable TETRA2101 CLI JSON evidence"
	functionTypedSliceCallbackEvidence := "function-typed value/struct-field/enum-payload callback slice-containing struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence"
	optionalAssignmentEvidence := "ptr/slice optional assignment return/owned/consume/inout escape with stable same-module/cross-module slice optional assignment return/owned/consume/inout CLI JSON evidence"
	stableWithoutWant := borrowedStringEvidence + " plus " + sliceEnumReturnEvidence + ", " + sliceStructEnumCallEvidence + ", " + genericBorrowReturnEvidence + ", " + functionTypedOptionalPtrCallbackEvidence + ", " + functionTypedSliceCallbackEvidence + ", " + optionalAssignmentEvidence + " exist"
	stableWithWant := borrowedStringEvidence + ", " + want + " plus " + sliceEnumReturnEvidence + ", " + sliceStructEnumCallEvidence + ", " + genericBorrowReturnEvidence + ", " + functionTypedOptionalPtrCallbackEvidence + ", " + functionTypedSliceCallbackEvidence + ", " + optionalAssignmentEvidence + " exist"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable slice struct return/inout CLI JSON evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable slice struct return/inout CLI JSON evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableSliceStructEnumOwnedConsumeInoutCallCLIJSONEvidence(t *testing.T) {
	sliceEnumReturnEvidence := "slice-containing enum direct/alias return escape CLI JSON evidence"
	want := "slice-containing struct/enum owned/consume/inout call escape CLI JSON evidence including imported direct cases"
	genericBorrowReturnEvidence := "same-module/cross-module generic borrow-aggregate/optional-ptr return diagnostics with stable TETRA2102 CLI JSON evidence"
	functionTypedOptionalPtrCallbackEvidence := "same-module/cross-module function-typed value/struct-field/enum-payload optional-ptr owned/consume/inout callback diagnostics with stable TETRA2101 CLI JSON evidence"
	functionTypedSliceCallbackEvidence := "function-typed value/struct-field/enum-payload callback slice-containing struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence"
	optionalAssignmentEvidence := "ptr/slice optional assignment return/owned/consume/inout escape with stable same-module/cross-module slice optional assignment return/owned/consume/inout CLI JSON evidence"
	stableWithoutWant := sliceEnumReturnEvidence + ", " + genericBorrowReturnEvidence + ", " + functionTypedOptionalPtrCallbackEvidence + ", " + functionTypedSliceCallbackEvidence + ", " + optionalAssignmentEvidence + " exist"
	stableWithWant := sliceEnumReturnEvidence + ", " + want + ", " + genericBorrowReturnEvidence + ", " + functionTypedOptionalPtrCallbackEvidence + ", " + functionTypedSliceCallbackEvidence + ", " + optionalAssignmentEvidence + " exist"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable slice struct/enum owned/consume/inout call CLI JSON evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable slice struct/enum owned/consume/inout call CLI JSON evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableGenericBorrowAggregateOptionalPtrReturnCLIJSONEvidence(t *testing.T) {
	sliceStructEnumCallEvidence := "slice-containing struct/enum owned/consume/inout call escape CLI JSON evidence including imported direct cases"
	want := "same-module/cross-module generic borrow-aggregate/optional-ptr return diagnostics with stable TETRA2102 CLI JSON evidence"
	functionTypedOptionalPtrCallbackEvidence := "same-module/cross-module function-typed value/struct-field/enum-payload optional-ptr owned/consume/inout callback diagnostics with stable TETRA2101 CLI JSON evidence"
	functionTypedSliceCallbackEvidence := "function-typed value/struct-field/enum-payload callback slice-containing struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence"
	optionalAssignmentEvidence := "ptr/slice optional assignment return/owned/consume/inout escape with stable same-module/cross-module slice optional assignment return/owned/consume/inout CLI JSON evidence"
	stableWithoutWant := sliceStructEnumCallEvidence + ", " + functionTypedOptionalPtrCallbackEvidence + ", " + functionTypedSliceCallbackEvidence + ", " + optionalAssignmentEvidence + " exist"
	stableWithWant := sliceStructEnumCallEvidence + ", " + want + ", " + functionTypedOptionalPtrCallbackEvidence + ", " + functionTypedSliceCallbackEvidence + ", " + optionalAssignmentEvidence + " exist"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable generic borrow aggregate/optional-ptr return CLI JSON evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable generic borrow aggregate/optional-ptr return CLI JSON evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableFunctionTypedOptionalPtrCallbackCLIJSONEvidence(t *testing.T) {
	genericBorrowReturnEvidence := "same-module/cross-module generic borrow-aggregate/optional-ptr return diagnostics with stable TETRA2102 CLI JSON evidence"
	want := "same-module/cross-module function-typed value/struct-field/enum-payload optional-ptr owned/consume/inout callback diagnostics with stable TETRA2101 CLI JSON evidence"
	functionTypedSliceCallbackEvidence := "function-typed value/struct-field/enum-payload callback slice-containing struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence"
	optionalAssignmentEvidence := "ptr/slice optional assignment return/owned/consume/inout escape with stable same-module/cross-module slice optional assignment return/owned/consume/inout CLI JSON evidence"
	stableWithoutWant := genericBorrowReturnEvidence + ", " + functionTypedSliceCallbackEvidence + ", " + optionalAssignmentEvidence + " exist"
	stableWithWant := genericBorrowReturnEvidence + ", " + want + ", " + functionTypedSliceCallbackEvidence + ", " + optionalAssignmentEvidence + " exist"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable function-typed optional-ptr callback CLI JSON evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable function-typed optional-ptr callback CLI JSON evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableFunctionTypedSliceAggregateCallbackCLIJSONEvidence(t *testing.T) {
	functionTypedOptionalPtrCallbackEvidence := "same-module/cross-module function-typed value/struct-field/enum-payload optional-ptr owned/consume/inout callback diagnostics with stable TETRA2101 CLI JSON evidence"
	want := "function-typed value/struct-field/enum-payload callback slice-containing struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence"
	optionalAssignmentEvidence := "ptr/slice optional assignment return/owned/consume/inout escape with stable same-module/cross-module slice optional assignment return/owned/consume/inout CLI JSON evidence"
	stableWithoutWant := functionTypedOptionalPtrCallbackEvidence + ", " + optionalAssignmentEvidence + " exist"
	stableWithWant := functionTypedOptionalPtrCallbackEvidence + ", " + want + ", " + optionalAssignmentEvidence + " exist"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable function-typed slice aggregate callback CLI JSON evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable function-typed slice aggregate callback CLI JSON evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableOptionalAssignmentCLIJSONEvidence(t *testing.T) {
	functionTypedSliceCallbackEvidence := "function-typed value/struct-field/enum-payload callback slice-containing struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence"
	want := "ptr/slice optional assignment return/owned/consume/inout escape with stable same-module/cross-module slice optional assignment return/owned/consume/inout CLI JSON evidence"
	stableWithoutWant := functionTypedSliceCallbackEvidence + " exist"
	stableWithWant := functionTypedSliceCallbackEvidence + ", " + want + " exist"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable optional assignment CLI JSON evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable optional assignment CLI JSON evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableSliceOptionalPayloadBindingCLIJSONEvidence(t *testing.T) {
	want := "same-module/cross-module slice optional payload binding owned/consume/inout call, `inout` assignment, and global assignment CLI JSON evidence"
	audit := strings.Replace(validBlockedOwnershipAudit(), want, "slice optional-payload binding CLI JSON evidence", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable slice optional-payload binding CLI JSON evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable slice optional-payload binding CLI JSON evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableDirectSliceGlobalAssignmentEvidence(t *testing.T) {
	sliceOptionalPayloadBindingEvidence := "same-module/cross-module slice optional payload binding owned/consume/inout call, `inout` assignment, and global assignment CLI JSON evidence"
	want := "same-module/cross-module direct slice global assignment with stable TETRA2102 JSON diagnostic evidence"
	stableWithoutWant := sliceOptionalPayloadBindingEvidence + " exists"
	stableWithWant := sliceOptionalPayloadBindingEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable direct slice global assignment evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable direct slice global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableOptionalPtrGlobalAssignmentEvidence(t *testing.T) {
	directSliceGlobalEvidence := "same-module/cross-module direct slice global assignment with stable TETRA2102 JSON diagnostic evidence"
	want := "same-module/cross-module optional ptr global assignment with stable TETRA2102 JSON diagnostic evidence"
	stableWithoutWant := directSliceGlobalEvidence + " exists"
	stableWithWant := directSliceGlobalEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable optional ptr global assignment evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable optional ptr global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableOptionalAggregateGlobalAssignmentEvidence(t *testing.T) {
	optionalPtrGlobalEvidence := "same-module/cross-module optional ptr global assignment with stable TETRA2102 JSON diagnostic evidence"
	want := "same-module/cross-module optional aggregate global assignment with stable TETRA2102 JSON diagnostic evidence"
	stableWithoutWant := optionalPtrGlobalEvidence + " exists"
	stableWithWant := optionalPtrGlobalEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable optional aggregate global assignment evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable optional aggregate global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStablePtrOptionalAssignmentIfLetMatchGlobalEscapeEvidence(t *testing.T) {
	optionalAggregateGlobalEvidence := "same-module/cross-module optional aggregate global assignment with stable TETRA2102 JSON diagnostic evidence"
	want := "same-module/cross-module ptr optional assignment if-let/match global escape with stable TETRA2102 JSON diagnostic evidence"
	stableWithoutWant := optionalAggregateGlobalEvidence + " exists"
	stableWithWant := optionalAggregateGlobalEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable ptr optional assignment if-let/match global escape evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable ptr optional assignment if-let/match global escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStablePtrEnumAliasReturnEvidence(t *testing.T) {
	ptrOptionalGlobalEvidence := "same-module/cross-module ptr optional assignment if-let/match global escape with stable TETRA2102 JSON diagnostic evidence"
	want := "same-module/cross-module ptr enum alias return escape with stable TETRA2102 JSON diagnostic evidence"
	stableWithoutWant := ptrOptionalGlobalEvidence + " exists"
	stableWithWant := ptrOptionalGlobalEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable ptr enum alias return evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable ptr enum alias return evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStablePtrAggregateReturnEvidence(t *testing.T) {
	ptrEnumAliasReturnEvidence := "same-module/cross-module ptr enum alias return escape with stable TETRA2102 JSON diagnostic evidence"
	want := "same-module/cross-module ptr-containing aggregate whole/field/alias/nested-field return escapes with stable TETRA2102 JSON diagnostic evidence"
	stableWithoutWant := ptrEnumAliasReturnEvidence + " exists"
	stableWithWant := ptrEnumAliasReturnEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable ptr aggregate return evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable ptr aggregate return evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableWholeAggregateGlobalAssignmentEvidence(t *testing.T) {
	ptrAggregateReturnEvidence := "same-module/cross-module ptr-containing aggregate whole/field/alias/nested-field return escapes with stable TETRA2102 JSON diagnostic evidence"
	want := "same-module/cross-module whole-aggregate global assignment with stable TETRA2102 JSON diagnostic evidence"
	stableWithoutWant := ptrAggregateReturnEvidence + " exists"
	stableWithWant := ptrAggregateReturnEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable whole-aggregate global assignment evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable whole-aggregate global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStablePtrEnumWholeValueGlobalAssignmentEvidence(t *testing.T) {
	wholeAggregateGlobalEvidence := "same-module/cross-module whole-aggregate global assignment with stable TETRA2102 JSON diagnostic evidence"
	want := "same-module/cross-module ptr-containing enum whole-value global assignment with stable TETRA2102 JSON diagnostic evidence"
	stableWithoutWant := wholeAggregateGlobalEvidence + " exists"
	stableWithWant := wholeAggregateGlobalEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable ptr enum whole-value global assignment evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable ptr enum whole-value global assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableGlobalFieldTargetAssignmentEvidence(t *testing.T) {
	ptrEnumWholeValueGlobalEvidence := "same-module/cross-module ptr-containing enum whole-value global assignment with stable TETRA2102 JSON diagnostic evidence"
	want := "same-module/cross-module global field target assignment with stable TETRA2102 JSON diagnostic evidence"
	stableWithoutWant := ptrEnumWholeValueGlobalEvidence + " exists"
	stableWithWant := ptrEnumWholeValueGlobalEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable global field target assignment evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable global field target assignment evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableAggregateNestedGlobalFieldEvidence(t *testing.T) {
	globalFieldTargetEvidence := "same-module/cross-module global field target assignment with stable TETRA2102 JSON diagnostic evidence"
	want := "same-module/cross-module aggregate and nested-aggregate global field escapes with stable TETRA2102 JSON diagnostic evidence"
	stableWithoutWant := globalFieldTargetEvidence + " exists"
	stableWithWant := globalFieldTargetEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable aggregate/nested global field evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable aggregate/nested global field evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStablePtrEnumPayloadEscapeEvidence(t *testing.T) {
	aggregateNestedEvidence := "same-module/cross-module aggregate and nested-aggregate global field escapes with stable TETRA2102 JSON diagnostic evidence"
	want := "same-module/cross-module ptr enum-payload return/global/inout assignment escapes with stable TETRA2102 JSON diagnostic evidence"
	stableWithoutWant := aggregateNestedEvidence + " exists"
	stableWithWant := aggregateNestedEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable ptr enum-payload escape evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable ptr enum-payload escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStablePtrOptionalPayloadEscapeEvidence(t *testing.T) {
	enumPayloadEvidence := "same-module/cross-module ptr enum-payload return/global/inout assignment escapes with stable TETRA2102 JSON diagnostic evidence"
	want := "same-module/cross-module ptr optional-payload return/global/inout assignment escapes with stable TETRA2102 JSON diagnostic evidence"
	stableWithoutWant := enumPayloadEvidence + " exists"
	stableWithWant := enumPayloadEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable ptr optional-payload escape evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable ptr optional-payload escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableSliceOptionalPayloadEscapeEvidence(t *testing.T) {
	ptrOptionalPayloadEvidence := "same-module/cross-module ptr optional-payload return/global/inout assignment escapes with stable TETRA2102 JSON diagnostic evidence"
	want := "same-module/cross-module slice optional-payload inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence"
	stableWithoutWant := ptrOptionalPayloadEvidence + " exists"
	stableWithWant := ptrOptionalPayloadEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable slice optional-payload escape evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable slice optional-payload escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableNestedSliceEnumPayloadEscapeEvidence(t *testing.T) {
	sliceOptionalPayloadEvidence := "same-module/cross-module slice optional-payload inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence"
	want := "same-module/cross-module nested slice enum-payload return/inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence"
	stableWithoutWant := sliceOptionalPayloadEvidence + " exists"
	stableWithWant := sliceOptionalPayloadEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable nested slice enum-payload escape evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable nested slice enum-payload escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStableNestedSliceStructEscapeEvidence(t *testing.T) {
	nestedSliceEnumPayloadEvidence := "same-module/cross-module nested slice enum-payload return/inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence"
	want := "same-module/cross-module nested slice struct return/inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence"
	stableWithoutWant := nestedSliceEnumPayloadEvidence + " exists"
	stableWithWant := nestedSliceEnumPayloadEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable nested slice struct escape evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable nested slice struct escape evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStablePtrContainingNestedAggregateCallTETRA2101Evidence(t *testing.T) {
	nestedSliceStructEvidence := "same-module/cross-module nested slice struct return/inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence"
	want := "same-module/cross-module ptr-containing/nested aggregate owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence"
	stableWithoutWant := nestedSliceStructEvidence + " exists"
	stableWithWant := nestedSliceStructEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable ptr-containing/nested aggregate call TETRA2101 evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable ptr-containing/nested aggregate call TETRA2101 evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingStablePtrEnumPayloadCallTETRA2101Evidence(t *testing.T) {
	ptrAggregateCallEvidence := "same-module/cross-module ptr-containing/nested aggregate owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence"
	want := "same-module/cross-module ptr enum-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence"
	stableWithoutWant := ptrAggregateCallEvidence + " exists"
	stableWithWant := ptrAggregateCallEvidence + " exists. " + want + " exists"
	audit := validBlockedOwnershipAudit()
	if !strings.Contains(audit, stableWithWant) {
		audit = strings.Replace(audit, stableWithoutWant, stableWithWant, 1)
	}
	audit = strings.Replace(audit, stableWithWant, stableWithoutWant, 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing stable ptr enum-payload call TETRA2101 evidence failure")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want stable ptr enum-payload call TETRA2101 evidence failure", err)
	}
}

func TestValidateOwnershipAuditRejectsMissingPartialStructEnumConsumeWholeValueEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "partial struct/enum consume whole-value rejection", "partial whole-value rejection", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing partial struct/enum consume whole-value evidence failure")
	}
	if !strings.Contains(err.Error(), "partial struct/enum consume whole-value rejection") {
		t.Fatalf("error = %v, want partial struct/enum consume whole-value evidence failure", err)
	}
}
