package fromvalidation

import memoryfacts "tetra_language/compiler/internal/memoryfacts"

type Graph = memoryfacts.Graph
type Fact = memoryfacts.Fact
type FactID = memoryfacts.FactID
type CostClass = memoryfacts.CostClass

const (
	StageValidation          = memoryfacts.StageValidation
	ProvenanceSafeKnown      = memoryfacts.ProvenanceSafeKnown
	UnsafeSafe               = memoryfacts.UnsafeSafe
	ValidationNotRun         = memoryfacts.ValidationNotRun
	ValidationPass           = memoryfacts.ValidationPass
	ValidationFail           = memoryfacts.ValidationFail
	CostZeroCostProven       = memoryfacts.CostZeroCostProven
	CostDynamicCheckRequired = memoryfacts.CostDynamicCheckRequired
	CostInstrumentationOnly  = memoryfacts.CostInstrumentationOnly
	CostUnsupportedRejected  = memoryfacts.CostUnsupportedRejected
)
