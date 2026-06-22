package memoryfacts

func isUnsafeUnknown(f Fact) bool {
	return f.ProvenanceClass == ProvenanceUnsafeUnknown || f.UnsafeClass == UnsafeUnknown
}

func hasUnsafeUnknownClass(provenance ProvenanceClass, unsafe UnsafeClass) bool {
	return provenance == ProvenanceUnsafeUnknown || unsafe == UnsafeUnknown
}

func isSafeProvenance(p ProvenanceClass) bool {
	return SafeProvenanceClass(string(p))
}

func hasSafeProvenanceFromUnsafeUnknown(provenance ProvenanceClass, unsafe UnsafeClass) bool {
	return isSafeProvenance(provenance) && unsafe == UnsafeUnknown
}

func requiresLoweredArtifact(f Fact) bool {
	return RowRequiresArtifact(
		string(f.StoragePlan),
		string(f.ActualLoweringStorage),
		f.Claim,
	)
}

func unsafeUnknownOptimizationClaim(claim string, alias AliasState) bool {
	return UnsafeUnknownOptimizationClaim(claim, string(alias))
}

func memoryOptimizationClaim(claim string, alias AliasState) bool {
	return MemoryOptimizationClaim(claim, string(alias))
}

func bareBoundsCheckEliminatedClaim(claim string) bool {
	return BareBoundsCheckEliminatedClaim(claim)
}

func dynamicRawRuntimeCheckCostDisallowed(claim string, cost CostClass) bool {
	return DynamicRawRuntimeCheckCostDisallowed(claim, string(cost))
}

func unsafeCheckedDisallowedClaim(
	provenance ProvenanceClass,
	unsafe UnsafeClass,
	claim string,
) bool {
	return UnsafeCheckedDisallowedClaim(string(provenance), string(unsafe), claim)
}

func knownCostClass(class CostClass) bool {
	return KnownCostClass(string(class))
}

func inferCostClass(f Fact) CostClass {
	if ZeroCostValidationRequiredClaim(f.Claim) && f.ValidationState != ValidationPass {
		if isUnsafeUnknown(f) {
			return CostConservativeFallback
		}
		return CostInstrumentationOnly
	}
	return CostClass(InferredCostClass(
		f.Claim,
		string(f.StoragePlan),
		string(f.ActualLoweringStorage),
		f.ClaimLevelRejected(),
		isUnsafeUnknown(f) || f.ProvenanceClass == ProvenanceUnsafeUnknown ||
			f.UnsafeClass == UnsafeUnknown,
		string(f.EscapeState),
		string(f.AliasState),
	))
}

func (f Fact) ClaimLevelRejected() bool {
	return f.ValidationState == ValidationFail || f.ValidationState == ValidationInvalidated
}

func unsafeVerifiedRootDisallowedClaim(
	provenance ProvenanceClass,
	unsafe UnsafeClass,
	claim string,
) bool {
	return UnsafeVerifiedRootDisallowedClaim(string(provenance), string(unsafe), claim)
}

func capMemDisallowedProofClaim(claim string, validatorName string, reason string) bool {
	return CapMemDisallowedProofClaim(claim, validatorName, reason)
}

func zeroCostProvenClaimDisallowed(f Fact) bool {
	return ZeroCostProvenClaimDisallowed(
		f.Claim,
		string(f.CostClass),
		factClaimLevelForCost(f),
		string(f.StoragePlan),
		string(f.ActualLoweringStorage),
	)
}

func zeroCostValidationRequiredClaim(claim string) bool {
	return ZeroCostValidationRequiredClaim(claim)
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
	return UnsafeUnknownTrustedStorage(string(planned), string(actual))
}

func unsafeExternalRootTrustedStorage(
	provenance ProvenanceClass,
	unsafe UnsafeClass,
	planned, actual StorageClass,
) bool {
	return UnsafeExternalRootTrustedStorage(
		string(provenance),
		string(unsafe),
		string(planned),
		string(actual),
	)
}

func validatedTrustedStorageHeapFallback(planned, actual StorageClass) bool {
	return ValidatedTrustedStorageHeapFallback(string(planned), string(actual))
}

func runtimeProofRequiredStorage(planned, actual StorageClass) bool {
	return RuntimeProofRequiredStorage(string(planned), string(actual))
}

func trustedStorageHeapFallback(planned, actual StorageClass) bool {
	return ValidatedTrustedStorageHeapFallback(string(planned), string(actual))
}

func storageFallbackRequiresReason(planned, actual StorageClass, cost CostClass) bool {
	if planned == "" && actual == "" {
		return false
	}
	if trustedStorageHeapFallback(planned, actual) {
		return true
	}
	return cost == CostConservativeFallback
}

func trustedStorageForUnsafeUnknown(class StorageClass) bool {
	return UnsafeUnknownTrustedStorage(string(class), "")
}

func knownSourceStage(stage SourceStage) bool {
	return KnownSourceStage(string(stage))
}

func knownProvenanceClass(class ProvenanceClass) bool {
	return KnownProvenanceClass(string(class))
}

func knownUnsafeClass(class UnsafeClass) bool {
	return KnownUnsafeClass(string(class))
}

func knownStorageClass(class StorageClass) bool {
	return KnownStorageClass(string(class))
}

func knownDomainKind(kind DomainKind) bool {
	return KnownDomainKind(string(kind))
}

func knownTransferKind(kind TransferKind) bool {
	return KnownTransferKind(string(kind))
}

func knownAliasState(state AliasState) bool {
	return KnownAliasState(string(state))
}

func validatedNoAliasState(state AliasState) bool {
	return ValidatedNoAliasState(string(state))
}

func broadNoAliasClaim(claim string) bool {
	return BroadNoAliasClaim(claim)
}

func conservativeNoAliasBoundaryClaim(claim string) bool {
	return ConservativeNoAliasBoundaryClaim(claim)
}

func claimRequiresParentFactID(claim string) bool {
	return ClaimRequiresParentFactID(claim)
}

func knownClaimLevel(level ClaimLevel) bool {
	return KnownClaimLevel(string(level))
}

func knownValidatorStatus(status ValidatorStatus) bool {
	return KnownValidatorStatus(string(status))
}

func knownReportClaim(claim string) bool {
	return KnownReportClaim(claim)
}
