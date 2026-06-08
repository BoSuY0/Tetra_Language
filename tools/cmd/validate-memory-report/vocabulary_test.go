package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler/memoryvocab"
)

func TestValidateMemoryReportVocabularyMatchesRegistry(t *testing.T) {
	for _, stage := range memoryvocab.SourceStages() {
		if !knownSourceStage(stage) {
			t.Fatalf("CLI validator rejects registered source stage %q", stage)
		}
	}
	for _, class := range memoryvocab.CostClasses() {
		if !knownCostClass(class) {
			t.Fatalf("CLI validator rejects registered cost class %q", class)
		}
	}
	for _, level := range memoryvocab.ClaimLevels() {
		if !knownClaimLevel(level) {
			t.Fatalf("CLI validator rejects registered claim level %q", level)
		}
	}
	for _, claim := range memoryvocab.ReportClaims() {
		if !knownReportClaim(claim) {
			t.Fatalf("CLI validator rejects registered report claim %q", claim)
		}
	}
	for _, claim := range memoryvocab.ParentRequiredClaims() {
		if !claimRequiresParentFactID(claim) {
			t.Fatalf("CLI validator does not require parent_fact_id for registered derived claim %q", claim)
		}
	}
	if !unsafeUnknownOptimizationClaim(memoryvocab.ClaimBoundsCheckEliminated, "") {
		t.Fatalf("CLI validator should classify bounds_check_eliminated as registered unsafe_unknown optimization")
	}
	if !unsafeCheckedDisallowedClaim(memoryvocab.ProvenanceUnsafeChecked, memoryvocab.UnsafeChecked, memoryvocab.ClaimNoAlias) {
		t.Fatalf("CLI validator should reject unsafe_checked no_alias through shared vocabulary")
	}
	if unsafeCheckedDisallowedClaim(memoryvocab.ProvenanceUnsafeChecked, memoryvocab.UnsafeChecked, memoryvocab.ClaimRawBoundsRuntimeCheckNormalBuild) {
		t.Fatalf("CLI validator should allow checked raw bounds runtime evidence through shared vocabulary")
	}
}

func TestValidateMemoryReportRejectsUnregisteredClaimVocabulary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(validSchemaV1MemoryReport(), `"claim": "allocation_base_metadata"`, `"claim": "new_zero_cost_magic_claim"`, 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	err := validateMemoryReport(path)
	if err == nil {
		t.Fatalf("expected CLI validator to reject unregistered claim")
	}
	if got := err.Error(); !strings.Contains(got, "unknown memory report claim") || !strings.Contains(got, "new_zero_cost_magic_claim") {
		t.Fatalf("validateMemoryReport error = %v, want unknown claim rejection", err)
	}
}

func TestValidateMemoryReportRejectsIslandProofClaimWithoutIslandVerifier(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(validSchemaV1MemoryReport(), `"claim": "allocation_base_metadata"`, `"claim": "`+memoryvocab.ClaimIslandProofVerified+`"`, 1)
	raw = strings.Replace(raw, `"cost_class": "zero_cost_proven"`, `"cost_class": "instrumentation_only"`, 1)
	raw = strings.Replace(raw, `"validator_name": "raw_bounds_validator"`, `"validator_name": "memory_report_validator"`, 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	err := validateMemoryReport(path)
	if err == nil {
		t.Fatalf("expected CLI validator to reject island proof claim without island verifier")
	}
	if got := err.Error(); !strings.Contains(got, memoryvocab.ClaimIslandProofVerified) || !strings.Contains(got, "validate-island-proof") {
		t.Fatalf("validateMemoryReport error = %v, want validate-island-proof rejection", err)
	}
}
