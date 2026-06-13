package surface

import "testing"

func TestSurfaceClaimTierVocabulary(t *testing.T) {
	want := []ClaimTier{
		ClaimTierProdStableScoped,
		ClaimTierBetaTargetHost,
		ClaimTierExperimental,
		ClaimTierUnsupported,
		ClaimTierNonClaim,
	}

	got := SurfaceClaimTiers()
	if len(got) != len(want) {
		t.Fatalf("SurfaceClaimTiers len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SurfaceClaimTiers[%d] = %q, want %q", i, got[i], want[i])
		}
		if !ValidSurfaceClaimTier(string(got[i])) {
			t.Fatalf("ValidSurfaceClaimTier(%q) = false, want true", got[i])
		}
	}
	if ValidSurfaceClaimTier("PRODUCTION_EVERYWHERE") {
		t.Fatalf("ValidSurfaceClaimTier accepted unknown tier")
	}
}
