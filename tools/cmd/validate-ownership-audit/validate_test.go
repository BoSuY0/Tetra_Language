package main

import (
	"strings"
	"testing"
)

func TestValidateOwnershipAuditAcceptsBlockedAudit(t *testing.T) {
	audit := validBlockedOwnershipAudit()
	if err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"}); err != nil {
		t.Fatalf("validateOwnershipAudit failed: %v\n%s", err, audit)
	}
}

func TestValidateOwnershipAuditAcceptsRowEvidenceDetails(t *testing.T) {
	audit := validBlockedOwnershipAudit()
	audit = replaceOwnershipAuditRowEvidence(
		t,
		audit,
		"Stable forbidden-case diagnostics",
		"See `Evidence Details / Stable forbidden-case diagnostics`.",
	)
	audit += "\n## Evidence Details\n\n### Stable forbidden-case diagnostics\n\n" +
		stableForbiddenCaseDiagnosticsFixtureEvidence() + "\n"

	if err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"}); err != nil {
		t.Fatalf("validateOwnershipAudit failed with detail evidence: %v\n%s", err, audit)
	}
}

func TestValidateOwnershipAuditRejectsMissingRequirement(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "| Alias/provenance tracking |", "| Alias tracking note |", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected missing requirement failure")
	}
	if !strings.Contains(err.Error(), "missing required checklist requirement") {
		t.Fatalf("error = %v, want missing required checklist requirement", err)
	}
}

func TestValidateOwnershipAuditRejectsAchievedStatusWithOpenRows(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "Status: not achieved.", "Status: achieved.", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "achieved"})
	if err == nil {
		t.Fatalf("expected achieved audit with open rows to fail")
	}
	if !strings.Contains(err.Error(), "achieved audit has non-passing checklist row") {
		t.Fatalf("error = %v, want non-passing checklist row", err)
	}
}

func TestValidateOwnershipAuditRejectsMockClaim(t *testing.T) {
	audit := strings.Replace(validBlockedOwnershipAudit(), "Local control-flow solver, branch/match/loop task-handle maybe-joined, task-group maybe-closed, island maybe-freed diagnostics, and branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence.", "mock ownership evidence.", 1)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "not-achieved"})
	if err == nil {
		t.Fatalf("expected mock claim failure")
	}
	if !strings.Contains(err.Error(), "mock") {
		t.Fatalf("error = %v, want mock claim failure", err)
	}
}
