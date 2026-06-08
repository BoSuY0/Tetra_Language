package memoryfacts

import "tetra_language/compiler/memoryvocab"

func isUnsafeUnknown(f Fact) bool {
	return f.ProvenanceClass == ProvenanceUnsafeUnknown || f.UnsafeClass == UnsafeUnknown
}

func hasUnsafeUnknownClass(provenance ProvenanceClass, unsafe UnsafeClass) bool {
	return provenance == ProvenanceUnsafeUnknown || unsafe == UnsafeUnknown
}

func isSafeProvenance(p ProvenanceClass) bool {
	return memoryvocab.SafeProvenanceClass(string(p))
}

func hasSafeProvenanceFromUnsafeUnknown(provenance ProvenanceClass, unsafe UnsafeClass) bool {
	return isSafeProvenance(provenance) && unsafe == UnsafeUnknown
}

func requiresLoweredArtifact(f Fact) bool {
	return memoryvocab.RowRequiresArtifact(string(f.StoragePlan), string(f.ActualLoweringStorage), f.Claim)
}

func unsafeUnknownOptimizationClaim(claim string, alias AliasState) bool {
	return memoryvocab.UnsafeUnknownOptimizationClaim(claim, string(alias))
}

func memoryOptimizationClaim(claim string, alias AliasState) bool {
	return memoryvocab.MemoryOptimizationClaim(claim, string(alias))
}

func bareBoundsCheckEliminatedClaim(claim string) bool {
	return memoryvocab.BareBoundsCheckEliminatedClaim(claim)
}

func dynamicRawRuntimeCheckCostDisallowed(claim string, cost CostClass) bool {
	return memoryvocab.DynamicRawRuntimeCheckCostDisallowed(claim, string(cost))
}

func unsafeCheckedDisallowedClaim(provenance ProvenanceClass, unsafe UnsafeClass, claim string) bool {
	return memoryvocab.UnsafeCheckedDisallowedClaim(string(provenance), string(unsafe), claim)
}

func knownCostClass(class CostClass) bool {
	return memoryvocab.KnownCostClass(string(class))
}

func inferCostClass(f Fact) CostClass {
	if memoryvocab.ZeroCostValidationRequiredClaim(f.Claim) && f.ValidationState != ValidationPass {
		if isUnsafeUnknown(f) {
			return CostConservativeFallback
		}
		return CostInstrumentationOnly
	}
	return CostClass(memoryvocab.InferredCostClass(
		f.Claim,
		string(f.StoragePlan),
		string(f.ActualLoweringStorage),
		f.ClaimLevelRejected(),
		isUnsafeUnknown(f) || f.ProvenanceClass == ProvenanceUnsafeUnknown || f.UnsafeClass == UnsafeUnknown,
		string(f.EscapeState),
		string(f.AliasState),
	))
}

func (f Fact) ClaimLevelRejected() bool {
	return f.ValidationState == ValidationFail || f.ValidationState == ValidationInvalidated
}

func unsafeVerifiedRootDisallowedClaim(provenance ProvenanceClass, unsafe UnsafeClass, claim string) bool {
	return memoryvocab.UnsafeVerifiedRootDisallowedClaim(string(provenance), string(unsafe), claim)
}

func capMemDisallowedProofClaim(claim string, validatorName string, reason string) bool {
	return memoryvocab.CapMemDisallowedProofClaim(claim, validatorName, reason)
}

func zeroCostProvenClaimDisallowed(f Fact) bool {
	return memoryvocab.ZeroCostProvenClaimDisallowed(
		f.Claim,
		string(f.CostClass),
		factClaimLevelForCost(f),
		string(f.StoragePlan),
		string(f.ActualLoweringStorage),
	)
}

func zeroCostValidationRequiredClaim(claim string) bool {
	return memoryvocab.ZeroCostValidationRequiredClaim(claim)
}

func factClaimLevelForCost(f Fact) string {
	switch f.ValidationState {
	case ValidationPass:
		return string(ClaimValidated)
	case ValidationFail, ValidationInvalidated:
		return string(ClaimRejected)
	default:
		if isUnsafeUnknown(f) {
			return string(ClaimConservative)
		}
		return string(ClaimEvidenceOnly)
	}
}

func unsafeUnknownTrustedStorage(planned, actual StorageClass) bool {
	return memoryvocab.UnsafeUnknownTrustedStorage(string(planned), string(actual))
}

func validatedTrustedStorageHeapFallback(planned, actual StorageClass) bool {
	return memoryvocab.ValidatedTrustedStorageHeapFallback(string(planned), string(actual))
}

func runtimeProofRequiredStorage(planned, actual StorageClass) bool {
	return memoryvocab.RuntimeProofRequiredStorage(string(planned), string(actual))
}

func trustedStorageHeapFallback(planned, actual StorageClass) bool {
	return memoryvocab.ValidatedTrustedStorageHeapFallback(string(planned), string(actual))
}

func trustedStorageForUnsafeUnknown(class StorageClass) bool {
	return memoryvocab.UnsafeUnknownTrustedStorage(string(class), "")
}

func knownSourceStage(stage SourceStage) bool {
	return memoryvocab.KnownSourceStage(string(stage))
}

func knownProvenanceClass(class ProvenanceClass) bool {
	return memoryvocab.KnownProvenanceClass(string(class))
}

func knownUnsafeClass(class UnsafeClass) bool {
	return memoryvocab.KnownUnsafeClass(string(class))
}

func knownStorageClass(class StorageClass) bool {
	return memoryvocab.KnownStorageClass(string(class))
}

func knownAliasState(state AliasState) bool {
	return memoryvocab.KnownAliasState(string(state))
}

func validatedNoAliasState(state AliasState) bool {
	return memoryvocab.ValidatedNoAliasState(string(state))
}

func broadNoAliasClaim(claim string) bool {
	return memoryvocab.BroadNoAliasClaim(claim)
}

func conservativeNoAliasBoundaryClaim(claim string) bool {
	return memoryvocab.ConservativeNoAliasBoundaryClaim(claim)
}

func claimRequiresParentFactID(claim string) bool {
	return memoryvocab.ClaimRequiresParentFactID(claim)
}

func knownClaimLevel(level ClaimLevel) bool {
	return memoryvocab.KnownClaimLevel(string(level))
}

func knownValidatorStatus(status ValidatorStatus) bool {
	return memoryvocab.KnownValidatorStatus(string(status))
}

func knownReportClaim(claim string) bool {
	return memoryvocab.KnownReportClaim(claim)
}
