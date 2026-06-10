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

func TestUnsafeExternalRootTrustedStorageVocabulary(t *testing.T) {
	for _, tc := range []struct {
		name       string
		provenance string
		unsafe     string
	}{
		{name: "unsafe unknown", provenance: ProvenanceUnsafeUnknown, unsafe: UnsafeUnknown},
		{name: "unsafe checked", provenance: ProvenanceUnsafeChecked, unsafe: UnsafeChecked},
		{name: "unsafe verified root", provenance: ProvenanceUnsafeVerifiedRoot, unsafe: UnsafeVerifiedRoot},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !UnsafeExternalRootTrustedStorage(tc.provenance, tc.unsafe, StorageStack, StorageStack) {
				t.Fatalf("UnsafeExternalRootTrustedStorage(%q, %q, Stack, Stack) = false, want true", tc.provenance, tc.unsafe)
			}
			if UnsafeExternalRootTrustedStorage(tc.provenance, tc.unsafe, StorageHeap, StorageHeap) {
				t.Fatalf("UnsafeExternalRootTrustedStorage(%q, %q, Heap, Heap) = true, want false", tc.provenance, tc.unsafe)
			}
		})
	}
	if UnsafeExternalRootTrustedStorage(ProvenanceSafeOwned, UnsafeSafe, StorageStack, StorageStack) {
		t.Fatalf("safe owned storage should not be classified as unsafe external root trusted storage")
	}
}
