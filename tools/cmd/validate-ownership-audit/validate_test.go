package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateOwnershipAuditAcceptsBlockedAudit(t *testing.T) {
	audit := validBlockedOwnershipAudit()
	if err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	); err != nil {
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

	if err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	); err != nil {
		t.Fatalf("validateOwnershipAudit failed with detail evidence: %v\n%s", err, audit)
	}
}

func TestValidateOwnershipAuditRejectsMissingRequirement(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"| Alias/provenance tracking |",
		"| Alias tracking note |",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected missing requirement failure")
	}
	if !strings.Contains(err.Error(), "missing required checklist requirement") {
		t.Fatalf("error = %v, want missing required checklist requirement", err)
	}
}

func TestValidateOwnershipAuditRejectsAchievedStatusWithOpenRows(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		"Status: not achieved.",
		"Status: achieved.",
		1,
	)
	err := validateOwnershipAudit([]byte(audit), ownershipAuditOptions{ExpectedStatus: "achieved"})
	if err == nil {
		t.Fatalf("expected achieved audit with open rows to fail")
	}
	if !strings.Contains(err.Error(), "achieved audit has non-passing checklist row") {
		t.Fatalf("error = %v, want non-passing checklist row", err)
	}
}

func TestValidateOwnershipAuditRejectsMockClaim(t *testing.T) {
	audit := strings.Replace(
		validBlockedOwnershipAudit(),
		("Local control-flow solver, branch/match/loop task-handle maybe-" +
			"joined, task-group maybe-closed, island maybe-freed diagnostics, and " +
			"branch/match/loop resource finalization merge diagnostics with stable " +
			"TETRA2101 JSON evidence."),
		"mock ownership evidence.",
		1,
	)
	err := validateOwnershipAudit(
		[]byte(audit),
		ownershipAuditOptions{ExpectedStatus: "not-achieved"},
	)
	if err == nil {
		t.Fatalf("expected mock claim failure")
	}
	if !strings.Contains(err.Error(), "mock") {
		t.Fatalf("error = %v, want mock claim failure", err)
	}
}

func TestOwnershipAuditFixtureSourceAvoidsGiantRows(t *testing.T) {
	paths := []string{"fixture_test.go"}
	evidencePaths, err := filepath.Glob("evidence_requirements*_test.go")
	if err != nil {
		t.Fatalf("glob evidence requirement tests: %v", err)
	}
	paths = append(paths, evidencePaths...)

	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for i, line := range strings.Split(string(raw), "\n") {
			if len(line) > 1000 {
				t.Fatalf(
					"%s line %d has %d bytes; split fixture evidence into readable parts",
					path,
					i+1,
					len(line),
				)
			}
		}
	}
}

func TestOwnershipAuditValidatorPackageIsSplitByResponsibility(t *testing.T) {
	expectedFiles := map[string]bool{
		"requirements.go":               true,
		"parser.go":                     true,
		"validate.go":                   true,
		"fixture_test.go":               true,
		"validate_test.go":              true,
		"evidence_requirements_test.go": true,
	}
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob package Go files: %v", err)
	}
	if len(paths) != len(expectedFiles) {
		t.Fatalf("package has %d Go files, want %d: %v", len(paths), len(expectedFiles), paths)
	}
	for _, path := range paths {
		name := filepath.Base(path)
		if !expectedFiles[name] {
			t.Fatalf("unexpected Go file %s; package should stay at the six-file structure", name)
		}
	}

	for path := range expectedFiles {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if len(strings.TrimSpace(string(raw))) == 0 {
			t.Fatalf("%s must not be empty", path)
		}
	}
}

func TestOwnershipAuditTestsAreSplitByResponsibility(t *testing.T) {
	expected := map[string][]string{
		"validate_test.go": {
			"TestValidateOwnershipAuditAcceptsBlockedAudit",
			"TestValidateOwnershipAuditAcceptsRowEvidenceDetails",
			"TestValidateOwnershipAuditRejectsMissingRequirement",
			"TestValidateOwnershipAuditRejectsAchievedStatusWithOpenRows",
			"TestValidateOwnershipAuditRejectsMockClaim",
			"TestOwnershipAuditFixtureSourceAvoidsGiantRows",
			"TestOwnershipAuditValidatorPackageIsSplitByResponsibility",
			"TestOwnershipAuditTestsAreSplitByResponsibility",
			"TestOwnershipAuditFixtureHelpersLiveInFocusedFile",
			"TestOwnershipAuditDocumentSourceAvoidsGiantRows",
		},
		"evidence_requirements_test.go": {
			"TestValidateOwnershipAuditRejectsMissingOwnershipSmokeExampleEvidence",
			"TestValidateOwnershipAuditRejectsMissingFeatureRegistryCommandEvidence",
			"TestValidateOwnershipAuditRejectsMissingSliceOptionalPayloadInoutGlobalEvidence",
			"TestValidateOwnershipAuditRejectsMissingNestedSliceEnumPayloadEscapeEvidence",
			"TestValidateOwnershipAuditRejectsMissingStableOptionalPayloadWholeValueEvidence",
			"TestValidateOwnershipAuditRejectsMissingStableActorTaskUseAfterTransferEvidence",
			"TestValidateOwnershipAuditRejectsMissingStableCallableEscapeDiagnosticsEvidence",
			"TestValidateOwnershipAuditRejectsMissingStableCLIJSONOwnershipLifetimeSafetyCodesEvidence",
			"TestValidateOwnershipAuditRejectsMissingPartialStructEnumConsumeWholeValueEvidence",
		},
	}

	for path, symbols := range expected {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(raw)
		for _, symbol := range symbols {
			if !strings.Contains(text, "func "+symbol+"(") {
				t.Fatalf("%s must contain %s", path, symbol)
			}
		}
	}
}

func TestOwnershipAuditFixtureHelpersLiveInFocusedFile(t *testing.T) {
	fixtureRaw, err := os.ReadFile("fixture_test.go")
	if err != nil {
		t.Fatalf("read fixture_test.go: %v", err)
	}
	for _, symbol := range []string{
		"validBlockedOwnershipAudit",
		"stableForbiddenCaseDiagnosticsFixtureEvidence",
		"renderBlockedOwnershipAudit",
	} {
		if !strings.Contains(string(fixtureRaw), "func "+symbol) {
			t.Fatalf("fixture_test.go must contain %s", symbol)
		}
	}

	paths := []string{"validate_test.go"}
	evidencePaths, err := filepath.Glob("evidence_requirements*_test.go")
	if err != nil {
		t.Fatalf("glob evidence requirement tests: %v", err)
	}
	paths = append(paths, evidencePaths...)

	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, symbol := range []string{
			"validBlockedOwnershipAudit",
			"stableForbiddenCaseDiagnosticsFixtureEvidence",
			"renderBlockedOwnershipAudit",
		} {
			if strings.Contains(string(raw), "\nfunc "+symbol+"(") {
				t.Fatalf("%s still contains fixture helper %s", path, symbol)
			}
		}
	}
}

func TestOwnershipAuditDocumentSourceAvoidsGiantRows(t *testing.T) {
	path := filepath.Join(
		"..",
		"..",
		"..",
		"docs",
		"release",
		"production",
		"ownership_production_audit.md",
	)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for i, line := range strings.Split(string(raw), "\n") {
		if len(line) > 1000 {
			t.Fatalf(
				"%s line %d has %d bytes; move detailed evidence into readable sections",
				path,
				i+1,
				len(line),
			)
		}
	}
}
