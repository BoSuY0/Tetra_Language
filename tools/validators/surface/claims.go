package surface

import "strings"

type ClaimTier string

const (
	ClaimTierProdStableScoped ClaimTier = "PROD_STABLE_SCOPED"
	ClaimTierBetaTargetHost   ClaimTier = "BETA_TARGET_HOST"
	ClaimTierExperimental     ClaimTier = "EXPERIMENTAL"
	ClaimTierUnsupported      ClaimTier = "UNSUPPORTED"
	ClaimTierNonClaim         ClaimTier = "NONCLAIM"
)

var surfaceClaimTiers = []ClaimTier{
	ClaimTierProdStableScoped,
	ClaimTierBetaTargetHost,
	ClaimTierExperimental,
	ClaimTierUnsupported,
	ClaimTierNonClaim,
}

func SurfaceClaimTiers() []ClaimTier {
	return append([]ClaimTier(nil), surfaceClaimTiers...)
}

func ValidSurfaceClaimTier(value string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	for _, tier := range surfaceClaimTiers {
		if normalized == string(tier) {
			return true
		}
	}
	return false
}
