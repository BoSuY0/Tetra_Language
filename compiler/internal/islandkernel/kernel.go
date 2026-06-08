package islandkernel

type Decision string

const (
	Accept       Decision = "accept"
	Reject       Decision = "reject"
	Conservative Decision = "conservative"
)

type Reason struct {
	Code    string
	Message string
}

type Result struct {
	Decision      Decision
	Reason        Reason
	ConsumesToken bool
	NextEpoch     uint64
}

type Bounds struct {
	Known    bool
	InBounds bool
}

type MemoryRef struct {
	BaseID      string
	IslandID    string
	Epoch       uint64
	Provenance  string
	Bounds      Bounds
	AliasState  string
	UnsafeClass string
}

type Token struct {
	IslandID string
	Epoch    uint64
	OwnerID  string
	Consumed bool
}

type Proof struct {
	ID            string
	Kind          string
	SubjectBaseID string
	IslandID      string
	Epoch         uint64
	Operation     string
	Verified      bool
}

type BorrowRequest struct {
	Ref   MemoryRef
	Token Token
}

type EscapeRequest struct {
	Ref MemoryRef
}

type BoundaryRequest struct {
	Ref      MemoryRef
	Transfer string
}

type TokenRequest struct {
	Token       Token
	LiveBorrows int
}

type NoAliasRequest struct {
	Left  MemoryRef
	Right MemoryRef
	Proof Proof
}

type ProofRequest struct {
	Ref       MemoryRef
	Proof     Proof
	Operation string
}

type StorageRequest struct {
	Ref             MemoryRef
	PlannedStorage  string
	ActualStorage   string
	Proof           Proof
	EscapesLifetime bool
}

type UnsafeRequest struct {
	Ref MemoryRef
}

const (
	ProvenanceOwned         = "owned_value"
	ProvenanceBorrowedView  = "borrowed_view"
	ProvenanceUnsafeUnknown = "unsafe_unknown"

	AliasUniqueLocal = "unique_local"

	UnsafeUnknown      = "unsafe_unknown"
	UnsafeVerifiedRoot = "unsafe_verified_root"

	ExternalUnsafeIsland = "external_unsafe_island"

	ProofBounds  = "bounds"
	ProofNoAlias = "noalias"
	ProofStorage = "storage"

	OperationIndexLoad             = "index_load"
	OperationNoAlias               = "noalias"
	OperationExplicitIslandStorage = "explicit_island_storage"

	TransferBorrowedView = "borrowed_view"
	TransferOwned        = "owned"
	TransferMoved        = "moved"
	TransferSerialized   = "serialized"

	StorageHeap           = "Heap"
	StorageExplicitIsland = "ExplicitIsland"
)

func CanBorrow(req BorrowRequest) Result {
	if req.Token.Consumed {
		return reject("borrow.consumed_token", "cannot borrow through a consumed island token")
	}
	if req.Ref.IslandID == "" || req.Token.IslandID == "" || req.Ref.IslandID != req.Token.IslandID {
		return reject("borrow.island_mismatch", "borrow requires matching island identity")
	}
	if req.Ref.Epoch != req.Token.Epoch {
		return reject("borrow.stale_epoch", "borrow epoch must match the current island token epoch")
	}
	return accept("borrow.live_epoch", "borrow stays inside the live island epoch")
}

func CanReturn(req EscapeRequest) Result {
	if isBorrowed(req.Ref) {
		return reject("escape.return_borrow", "borrowed island reference cannot escape by return")
	}
	return accept("escape.return_owned", "owned value may return without borrowing island-local memory")
}

func CanStoreGlobal(req EscapeRequest) Result {
	if isBorrowed(req.Ref) {
		return reject("escape.global_borrow", "borrowed island reference cannot be stored globally")
	}
	return accept("escape.global_owned", "owned value may be stored globally")
}

func CanCaptureClosure(req EscapeRequest) Result {
	if isBorrowed(req.Ref) {
		return reject("escape.closure_borrow", "borrowed island reference cannot be captured by an escaping closure")
	}
	return accept("escape.closure_owned", "owned value may be captured")
}

func CanSendToActor(req BoundaryRequest) Result {
	return boundaryDecision(req, "actor")
}

func CanSendToTask(req BoundaryRequest) Result {
	return boundaryDecision(req, "task")
}

func CanMoveIsland(req TokenRequest) Result {
	if req.Token.Consumed {
		return reject("token.move_consumed", "cannot move an already consumed island token")
	}
	if req.Token.IslandID == "" || req.Token.OwnerID == "" {
		return conservative("token.move_missing_identity", "moving an island requires token identity and owner evidence")
	}
	res := accept("token.move_consumes_source", "moving an island consumes the source owner token")
	res.ConsumesToken = true
	return res
}

func CanFreeIsland(req TokenRequest) Result {
	if req.Token.Consumed {
		return reject("token.free_consumed", "cannot free an already consumed island token")
	}
	if req.LiveBorrows > 0 {
		return reject("token.free_live_borrows", "cannot free an island while live borrows exist")
	}
	res := accept("token.free_consumes_source", "free consumes the island token")
	res.ConsumesToken = true
	return res
}

func CanResetIsland(req TokenRequest) Result {
	if req.Token.Consumed {
		return reject("token.reset_consumed", "cannot reset an already consumed island token")
	}
	if req.LiveBorrows > 0 {
		return reject("token.reset_live_borrows", "cannot reset an island while live borrows exist")
	}
	res := accept("token.reset_epoch_advanced", "reset advances the island epoch and invalidates old references")
	res.NextEpoch = req.Token.Epoch + 1
	return res
}

func CanClaimNoAlias(req NoAliasRequest) Result {
	if unsafeOrExternal(req.Left) || unsafeOrExternal(req.Right) {
		return reject("noalias.unsafe_external", "unsafe or external memory cannot authorize noalias")
	}
	if req.Left.IslandID != "" && req.Right.IslandID != "" && req.Left.IslandID != req.Right.IslandID && proofMatches(req.Proof, req.Left, ProofNoAlias, OperationNoAlias) {
		return accept("noalias.distinct_proven_islands", "distinct live islands with verified proof may claim narrow noalias")
	}
	return conservative("noalias.proof_required", "noalias requires distinct island identity and a verified proof")
}

func CanEliminateBoundsCheck(req ProofRequest) Result {
	if proofMatches(req.Proof, req.Ref, ProofBounds, req.Operation) {
		return accept("bounds.proof_verified", "bounds check elimination has a verified proof for this ref and operation")
	}
	return reject("bounds.missing_proof", "bounds check elimination requires a verified proof")
}

func CanLowerAsExplicitIsland(req StorageRequest) Result {
	if req.EscapesLifetime {
		return reject("storage.explicit_island_escape", "explicit island storage cannot be trusted for escaping values")
	}
	if req.PlannedStorage != StorageExplicitIsland || req.ActualStorage != StorageExplicitIsland {
		return reject("storage.explicit_island_mismatch", "explicit island storage requires planned and actual island lowering")
	}
	if proofMatches(req.Proof, storageProofRef(req), ProofStorage, OperationExplicitIslandStorage) {
		return accept("storage.explicit_island_trusted", "explicit island storage matches plan, lowering, and proof")
	}
	return conservative("storage.explicit_island_proof_required", "explicit island storage requires a verified storage proof")
}

func CanPromoteUnsafeRoot(req UnsafeRequest) Result {
	if req.Ref.Provenance == ProvenanceUnsafeUnknown || req.Ref.UnsafeClass == UnsafeUnknown || req.Ref.IslandID == ExternalUnsafeIsland {
		return reject("unsafe.unknown_promotion", "unsafe_unknown and external roots cannot promote to safe memory")
	}
	if req.Ref.UnsafeClass == UnsafeVerifiedRoot && req.Ref.Bounds.Known && req.Ref.Bounds.InBounds {
		return accept("unsafe.verified_root_bounded", "bounded unsafe verified root may remain checked")
	}
	return conservative("unsafe.runtime_contract_required", "unsafe roots require a bounded runtime contract")
}

func CanTrustStorage(req StorageRequest) Result {
	if req.ActualStorage == StorageHeap && req.PlannedStorage != StorageHeap {
		return reject("storage.heap_fallback_not_trusted", "heap fallback cannot satisfy a trusted storage claim")
	}
	if req.EscapesLifetime {
		return reject("storage.escape_not_trusted", "escaping values cannot satisfy trusted storage")
	}
	if req.PlannedStorage == req.ActualStorage && proofMatches(req.Proof, storageProofRef(req), ProofStorage, req.Proof.Operation) {
		return accept("storage.trusted_with_proof", "storage claim matches plan, actual lowering, and proof")
	}
	return conservative("storage.proof_required", "trusted storage requires matching lowering and proof")
}

func CanEraseRuntimeCheck(req ProofRequest) Result {
	if proofMatches(req.Proof, req.Ref, req.Proof.Kind, req.Operation) {
		return accept("runtime_check.erase_verified", "runtime check may be erased only with verified proof")
	}
	return conservative("runtime_check.keep", "runtime check must remain without verified proof")
}

func boundaryDecision(req BoundaryRequest, boundary string) Result {
	if req.Transfer == TransferBorrowedView || isBorrowed(req.Ref) {
		return reject("boundary."+boundary+"_borrow", "borrowed island views cannot cross "+boundary+" boundaries")
	}
	switch req.Transfer {
	case TransferOwned, TransferMoved, TransferSerialized:
		return accept("boundary."+boundary+"_owned", "owned, moved, or serialized values may cross "+boundary+" boundaries")
	default:
		return conservative("boundary."+boundary+"_unknown_transfer", "unknown transfer kind stays conservative")
	}
}

func proofMatches(proof Proof, ref MemoryRef, kind string, operation string) bool {
	return proof.ID != "" &&
		proof.Verified &&
		proof.Kind == kind &&
		proof.Operation == operation &&
		proof.SubjectBaseID == ref.BaseID &&
		proof.IslandID == ref.IslandID &&
		proof.Epoch == ref.Epoch
}

func storageProofRef(req StorageRequest) MemoryRef {
	ref := req.Ref
	if req.Proof.SubjectBaseID != "" {
		ref.BaseID = req.Proof.SubjectBaseID
	}
	return ref
}

func isBorrowed(ref MemoryRef) bool {
	return ref.Provenance == ProvenanceBorrowedView
}

func unsafeOrExternal(ref MemoryRef) bool {
	return ref.Provenance == ProvenanceUnsafeUnknown ||
		ref.UnsafeClass == UnsafeUnknown ||
		ref.IslandID == ExternalUnsafeIsland
}

func accept(code, message string) Result {
	return Result{Decision: Accept, Reason: Reason{Code: code, Message: message}}
}

func reject(code, message string) Result {
	return Result{Decision: Reject, Reason: Reason{Code: code, Message: message}}
}

func conservative(code, message string) Result {
	return Result{Decision: Conservative, Reason: Reason{Code: code, Message: message}}
}
