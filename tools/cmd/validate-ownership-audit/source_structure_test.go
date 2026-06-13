package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
				t.Fatalf("%s line %d has %d bytes; split fixture evidence into readable parts", path, i+1, len(line))
			}
		}
	}
}

func TestOwnershipAuditValidatorPackageIsSplitByResponsibility(t *testing.T) {
	for _, path := range []string{"requirements.go", "parser.go", "validate.go"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if len(strings.TrimSpace(string(raw))) == 0 {
			t.Fatalf("%s must not be empty", path)
		}
	}

	mainRaw, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	mainText := string(mainRaw)
	for _, symbol := range []string{
		"requiredOwnershipAuditEvidencePhrases",
		"parseOwnershipAuditRows",
		"validateOwnershipAuditRows",
	} {
		if strings.Contains(mainText, symbol) {
			t.Fatalf("main.go still contains %s; move validator internals into focused files", symbol)
		}
	}

	for _, path := range []string{"requirements.go", "parser.go", "validate.go"} {
		if filepath.Dir(path) != "." {
			t.Fatalf("unexpected nested validator file path %s", path)
		}
	}
}

func TestOwnershipAuditTestsAreSplitByResponsibility(t *testing.T) {
	expected := map[string][]string{
		"source_structure_test.go": {
			"TestOwnershipAuditFixtureSourceAvoidsGiantRows",
			"TestOwnershipAuditValidatorPackageIsSplitByResponsibility",
			"TestOwnershipAuditFixtureHelpersLiveInFocusedFile",
			"TestOwnershipAuditDocumentSourceAvoidsGiantRows",
		},
		"validate_test.go": {
			"TestValidateOwnershipAuditAcceptsBlockedAudit",
			"TestValidateOwnershipAuditAcceptsRowEvidenceDetails",
			"TestValidateOwnershipAuditRejectsMissingRequirement",
			"TestValidateOwnershipAuditRejectsAchievedStatusWithOpenRows",
			"TestValidateOwnershipAuditRejectsMockClaim",
		},
		"evidence_requirements_test.go": {
			"TestValidateOwnershipAuditRejectsMissingOwnershipSmokeExampleEvidence",
			"TestValidateOwnershipAuditRejectsMissingFeatureRegistryCommandEvidence",
			"TestValidateOwnershipAuditRejectsMissingSliceOptionalPayloadInoutGlobalEvidence",
		},
		"evidence_requirements_escape_test.go": {
			"TestValidateOwnershipAuditRejectsMissingNestedSliceEnumPayloadEscapeEvidence",
			"TestValidateOwnershipAuditRejectsMissingStableOptionalPayloadWholeValueEvidence",
		},
		"evidence_requirements_stable_runtime_test.go": {
			"TestValidateOwnershipAuditRejectsMissingStableActorTaskUseAfterTransferEvidence",
			"TestValidateOwnershipAuditRejectsMissingStableCallableEscapeDiagnosticsEvidence",
		},
		"evidence_requirements_stable_cli_test.go": {
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

	paths := []string{"source_structure_test.go", "validate_test.go"}
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
	path := filepath.Join("..", "..", "..", "docs", "release", "ownership_production_audit.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for i, line := range strings.Split(string(raw), "\n") {
		if len(line) > 1000 {
			t.Fatalf("%s line %d has %d bytes; move detailed evidence into readable sections", path, i+1, len(line))
		}
	}
}
