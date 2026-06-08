package memoryvocab

import "testing"

func TestIslandKernelClaimVocabularyRegistered(t *testing.T) {
	for _, claim := range []string{
		ClaimIslandKernelModelOnly,
		ClaimIslandEpochValidated,
		ClaimIslandSanitizeRuntimeChecked,
		ClaimIslandProofVerified,
	} {
		if !KnownReportClaim(claim) {
			t.Fatalf("KnownReportClaim(%q) = false, want true", claim)
		}
		if !IslandKernelEvidenceClaim(claim) {
			t.Fatalf("IslandKernelEvidenceClaim(%q) = false, want true", claim)
		}
	}
	if got := RequiredIslandKernelClaimValidator(ClaimIslandProofVerified); got != "validate-island-proof" {
		t.Fatalf("RequiredIslandKernelClaimValidator(%q) = %q, want validate-island-proof", ClaimIslandProofVerified, got)
	}
	if !IslandKernelClaimValidatorMismatch(ClaimIslandProofVerified, "memory_report_validator") {
		t.Fatalf("IslandKernelClaimValidatorMismatch should reject island proof without validate-island-proof")
	}
	if IslandKernelClaimValidatorMismatch(ClaimIslandProofVerified, "validate-island-proof") {
		t.Fatalf("IslandKernelClaimValidatorMismatch should accept validate-island-proof")
	}
}
