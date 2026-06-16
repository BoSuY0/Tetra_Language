package compiler

import "tetra_language/compiler/internal/abisuite"

const (
	abiVerificationSchemaV1  = abisuite.VerificationSchemaV1
	abiVerificationScopeP211 = abisuite.VerificationScopeP211
)

const (
	abiVerificationTaskCorpus           = abisuite.VerificationTaskCorpus
	abiVerificationTaskAggregateReturns = abisuite.VerificationTaskAggregateReturns
	abiVerificationTaskCallBoundary     = abisuite.VerificationTaskCallBoundary
	abiVerificationTaskFFIReprC         = abisuite.VerificationTaskFFIReprC
)

type ABIVerificationReport = abisuite.VerificationReport
type ABIVerificationTargetRow = abisuite.VerificationTargetRow
type ABIVerificationTaskRow = abisuite.VerificationTaskRow

func BuildP21ABIVerificationReport() ABIVerificationReport {
	return abisuite.BuildP21VerificationReport()
}

func ValidateP21ABIVerificationReport(report ABIVerificationReport) error {
	return abisuite.ValidateP21VerificationReport(report)
}

func p21ABIVerificationTargets() []string {
	return abisuite.P21VerificationTargets()
}

func p21ABIVerificationTaskIDs() []string {
	return abisuite.P21VerificationTaskIDs()
}

func p21ABIVerificationNonClaims() []string {
	return abisuite.P21VerificationNonClaims()
}
