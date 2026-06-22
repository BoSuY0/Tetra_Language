package fromplir

import memoryfacts "tetra_language/compiler/internal/memoryfacts"
import "tetra_language/compiler/internal/plir"

type Graph = memoryfacts.Graph
type Fact = memoryfacts.Fact
type FactID = memoryfacts.FactID
type ProvenanceClass = memoryfacts.ProvenanceClass
type UnsafeClass = memoryfacts.UnsafeClass
type BorrowState = memoryfacts.BorrowState
type EscapeState = memoryfacts.EscapeState
type AliasState = memoryfacts.AliasState
type StorageClass = memoryfacts.StorageClass
type DomainKind = memoryfacts.DomainKind
type TransferKind = memoryfacts.TransferKind
type ValidationState = memoryfacts.ValidationState
type CostClass = memoryfacts.CostClass

const (
	StageSemantics               = memoryfacts.StageSemantics
	StagePLIR                    = memoryfacts.StagePLIR
	StageAllocPlan               = memoryfacts.StageAllocPlan
	ProvenanceSafeKnown          = memoryfacts.ProvenanceSafeKnown
	ProvenanceSafeBorrowed       = memoryfacts.ProvenanceSafeBorrowed
	ProvenanceSafeOwned          = memoryfacts.ProvenanceSafeOwned
	ProvenanceUnsafeUnknown      = memoryfacts.ProvenanceUnsafeUnknown
	ProvenanceUnsafeChecked      = memoryfacts.ProvenanceUnsafeChecked
	ProvenanceUnsafeVerifiedRoot = memoryfacts.ProvenanceUnsafeVerifiedRoot
	UnsafeSafe                   = memoryfacts.UnsafeSafe
	UnsafeUnknown                = memoryfacts.UnsafeUnknown
	UnsafeChecked                = memoryfacts.UnsafeChecked
	UnsafeVerifiedRoot           = memoryfacts.UnsafeVerifiedRoot
	BorrowNone                   = memoryfacts.BorrowNone
	BorrowImmutable              = memoryfacts.BorrowImmutable
	BorrowMutable                = memoryfacts.BorrowMutable
	BorrowMoved                  = memoryfacts.BorrowMoved
	EscapeUnknown                = memoryfacts.EscapeUnknown
	EscapeNoEscape               = memoryfacts.EscapeNoEscape
	EscapeReturn                 = memoryfacts.EscapeReturn
	EscapeGlobal                 = memoryfacts.EscapeGlobal
	EscapeActor                  = memoryfacts.EscapeActor
	EscapeTask                   = memoryfacts.EscapeTask
	EscapeUnsafe                 = memoryfacts.EscapeUnsafe
	EscapeConservative           = memoryfacts.EscapeConservative
	AliasUnknown                 = memoryfacts.AliasUnknown
	AliasUnique                  = memoryfacts.AliasUnique
	AliasMaybe                   = memoryfacts.AliasMaybe
	AliasMutableExclusive        = memoryfacts.AliasMutableExclusive
	AliasUnknownConservative     = memoryfacts.AliasUnknownConservative
	AliasInvalidatedByCall       = memoryfacts.AliasInvalidatedByCall
	StorageTaskRegion            = memoryfacts.StorageTaskRegion
	StorageActorMoveRegion       = memoryfacts.StorageActorMoveRegion
	StorageExplicitIsland        = memoryfacts.StorageExplicitIsland
	DomainTask                   = memoryfacts.DomainTask
	DomainActor                  = memoryfacts.DomainActor
	DomainRequest                = memoryfacts.DomainRequest
	TransferMove                 = memoryfacts.TransferMove
	TransferCopy                 = memoryfacts.TransferCopy
	TransferBorrowed             = memoryfacts.TransferBorrowed
	TransferUnsafe               = memoryfacts.TransferUnsafe
	ValidationNotRun             = memoryfacts.ValidationNotRun
	ValidationPass               = memoryfacts.ValidationPass
	ValidationFail               = memoryfacts.ValidationFail
	CostZeroCostProven           = memoryfacts.CostZeroCostProven
	CostDynamicCheckRequired     = memoryfacts.CostDynamicCheckRequired
	CostInstrumentationOnly      = memoryfacts.CostInstrumentationOnly
	CostUnsupportedRejected      = memoryfacts.CostUnsupportedRejected
	CostConservativeFallback     = memoryfacts.CostConservativeFallback
	ProofStorage                 = memoryfacts.ProofStorage
	ProofDomainMove              = memoryfacts.ProofDomainMove
	ClaimTrustedStorage          = memoryfacts.ClaimTrustedStorage
)

var NewGraph = memoryfacts.NewGraph
var RuntimeProofRequiredStorage = memoryfacts.RuntimeProofRequiredStorage

type plirFactKey struct {
	kind    plir.FactKind
	valueID string
}
