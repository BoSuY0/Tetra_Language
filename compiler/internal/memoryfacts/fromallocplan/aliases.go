package fromallocplan

import memoryfacts "tetra_language/compiler/internal/memoryfacts"

type Graph = memoryfacts.Graph
type Fact = memoryfacts.Fact
type FactID = memoryfacts.FactID
type ProvenanceClass = memoryfacts.ProvenanceClass
type UnsafeClass = memoryfacts.UnsafeClass
type StorageClass = memoryfacts.StorageClass
type ValidationState = memoryfacts.ValidationState
type CostClass = memoryfacts.CostClass
type MemoryDelta = memoryfacts.Delta

const (
	StageAllocPlan               = memoryfacts.StageAllocPlan
	ProvenanceSafeOwned          = memoryfacts.ProvenanceSafeOwned
	ProvenanceUnsafeVerifiedRoot = memoryfacts.ProvenanceUnsafeVerifiedRoot
	UnsafeSafe                   = memoryfacts.UnsafeSafe
	UnsafeVerifiedRoot           = memoryfacts.UnsafeVerifiedRoot
	ValidationNotRun             = memoryfacts.ValidationNotRun
	ValidationPass               = memoryfacts.ValidationPass
	CostZeroCostProven           = memoryfacts.CostZeroCostProven
	CostInstrumentationOnly      = memoryfacts.CostInstrumentationOnly
	CostUnsupportedRejected      = memoryfacts.CostUnsupportedRejected
	CostConservativeFallback     = memoryfacts.CostConservativeFallback
)

var RuntimeProofRequiredStorage = memoryfacts.RuntimeProofRequiredStorage
var NewGraph = memoryfacts.NewGraph
