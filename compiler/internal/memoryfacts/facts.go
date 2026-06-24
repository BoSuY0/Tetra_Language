package memoryfacts

type FactID string

type SourceStage string

const (
	StageSemantics             SourceStage = "semantics"
	StageUnsafeGatewayLowering SourceStage = "unsafe_gateway_lowering"
	StagePLIR                  SourceStage = "plir"
	StageAllocPlan             SourceStage = "allocplan"
	StageLowering              SourceStage = "lowering"
	StageOptimization          SourceStage = "optimization"
	StageValidation            SourceStage = "validation"
)

type ProvenanceClass string

const (
	ProvenanceSafeKnown          ProvenanceClass = "safe_known"
	ProvenanceSafeBorrowed       ProvenanceClass = "safe_borrowed"
	ProvenanceSafeOwned          ProvenanceClass = "safe_owned"
	ProvenanceUnsafeUnknown      ProvenanceClass = "unsafe_unknown"
	ProvenanceUnsafeChecked      ProvenanceClass = "unsafe_checked"
	ProvenanceUnsafeVerifiedRoot ProvenanceClass = "unsafe_verified_root"
)

type UnsafeClass string

const (
	UnsafeSafe         UnsafeClass = "safe"
	UnsafeUnknown      UnsafeClass = "unsafe_unknown"
	UnsafeChecked      UnsafeClass = "unsafe_checked"
	UnsafeVerifiedRoot UnsafeClass = "unsafe_verified_root"
)

type BorrowState string

const (
	BorrowNone      BorrowState = ""
	BorrowImmutable BorrowState = "borrowed_imm"
	BorrowMutable   BorrowState = "borrowed_mut"
	BorrowMoved     BorrowState = "moved"
)

type EscapeState string

const (
	EscapeUnknown      EscapeState = ""
	EscapeNoEscape     EscapeState = "no_escape"
	EscapeReturn       EscapeState = "escapes_return"
	EscapeGlobal       EscapeState = "escapes_global"
	EscapeActor        EscapeState = "escapes_actor"
	EscapeTask         EscapeState = "escapes_task"
	EscapeUnsafe       EscapeState = "escapes_unsafe"
	EscapeConservative EscapeState = "unknown"
)

type AliasState string

const (
	AliasUnknown             AliasState = ""
	AliasUnique              AliasState = "unique"
	AliasSharedReadonly      AliasState = "shared_readonly"
	AliasMutableExclusive    AliasState = "mutable_exclusive"
	AliasMaybe               AliasState = "maybe_alias"
	AliasUnknownConservative AliasState = "unknown_alias"
	AliasInvalidatedByCall   AliasState = "invalidated_by_call"
)

type StorageClass string

const (
	StorageUnknownConservative StorageClass = "UnknownConservative"
	StorageEliminated          StorageClass = "Eliminated"
	StorageRegister            StorageClass = "Register"
	StorageHeap                StorageClass = "Heap"
	StorageStack               StorageClass = "Stack"
	StorageRegion              StorageClass = "Region"
	StorageExplicitIsland      StorageClass = "ExplicitIsland"
	StorageFunctionTempRegion  StorageClass = "FunctionTempRegion"
	StorageTaskRegion          StorageClass = "TaskRegion"
	StorageActorMoveRegion     StorageClass = "ActorMoveRegion"
	StorageLargeMmap           StorageClass = "LargeMmap"
	StorageExternal            StorageClass = "External"
)

type DomainKind string

const (
	DomainProcess  DomainKind = "process"
	DomainTask     DomainKind = "task"
	DomainActor    DomainKind = "actor"
	DomainIsland   DomainKind = "island"
	DomainRequest  DomainKind = "request"
	DomainExternal DomainKind = "external"
)

type TransferKind string

const (
	TransferUnknown  TransferKind = ""
	TransferMove     TransferKind = "move"
	TransferCopy     TransferKind = "copy"
	TransferBorrowed TransferKind = "borrowed"
	TransferUnsafe   TransferKind = "unsafe_contract"
)

type ValidationState string

const (
	ValidationNotRun      ValidationState = "not_run"
	ValidationPass        ValidationState = "pass"
	ValidationFail        ValidationState = "fail"
	ValidationInvalidated ValidationState = "invalidated"
)

type ClaimLevel string

const (
	ClaimValidated    ClaimLevel = "validated"
	ClaimEvidenceOnly ClaimLevel = "evidence_only"
	ClaimConservative ClaimLevel = "conservative"
	ClaimRejected     ClaimLevel = "rejected"
	ClaimFuture       ClaimLevel = "future"
)

type ValidatorStatus string

const (
	ValidatorPass          ValidatorStatus = "pass"
	ValidatorFail          ValidatorStatus = "fail"
	ValidatorNotApplicable ValidatorStatus = "not_applicable"
	ValidatorNotRun        ValidatorStatus = "not_run"
)

type CostClass string

const (
	CostZeroCostProven       CostClass = "zero_cost_proven"
	CostDynamicCheckRequired CostClass = "dynamic_check_required"
	CostInstrumentationOnly  CostClass = "instrumentation_only"
	CostUnsupportedRejected  CostClass = "unsupported_rejected"
	CostConservativeFallback CostClass = "conservative_fallback"
)

type Fact struct {
	ID                    FactID          `json:"fact_id"`
	ProgramID             string          `json:"program_id,omitempty"`
	FunctionID            string          `json:"function_id,omitempty"`
	ContractSchema        string          `json:"contract_schema,omitempty"`
	ContractDigest        string          `json:"contract_digest,omitempty"`
	BlockID               string          `json:"block_id,omitempty"`
	ValueID               string          `json:"value_id,omitempty"`
	IslandID              string          `json:"island_id,omitempty"`
	Epoch                 int             `json:"epoch,omitempty"`
	BaseID                string          `json:"base_id,omitempty"`
	SiteID                string          `json:"site_id,omitempty"`
	SourceSpan            string          `json:"source_span,omitempty"`
	TypeName              string          `json:"type_name,omitempty"`
	ProvenanceClass       ProvenanceClass `json:"provenance_class,omitempty"`
	RegionID              string          `json:"region_id,omitempty"`
	OwnerID               string          `json:"owner_id,omitempty"`
	DomainKind            DomainKind      `json:"domain_kind,omitempty"`
	DomainID              string          `json:"domain_id,omitempty"`
	DomainOwnerID         string          `json:"domain_owner_id,omitempty"`
	TransferKind          TransferKind    `json:"transfer_kind,omitempty"`
	TransferProofID       string          `json:"transfer_proof_id,omitempty"`
	SourceConsumed        bool            `json:"source_consumed,omitempty"`
	LiveBorrowCrossing    bool            `json:"live_borrow_crossing,omitempty"`
	DestinationActive     bool            `json:"destination_active,omitempty"`
	LifetimeBirth         string          `json:"lifetime_birth,omitempty"`
	LifetimeDeath         string          `json:"lifetime_death,omitempty"`
	LifetimeOwner         string          `json:"lifetime_owner,omitempty"`
	ParamIndex            *int            `json:"param_index,omitempty"`
	ParamPath             string          `json:"param_path,omitempty"`
	BorrowState           BorrowState     `json:"borrow_state,omitempty"`
	EscapeState           EscapeState     `json:"escape_state,omitempty"`
	AliasState            AliasState      `json:"alias_state,omitempty"`
	MutabilityState       string          `json:"mutability_state,omitempty"`
	AllocationSiteID      string          `json:"allocation_site_id,omitempty"`
	UnsafeClass           UnsafeClass     `json:"unsafe_class,omitempty"`
	StoragePlan           StorageClass    `json:"storage_plan,omitempty"`
	ActualLoweringStorage StorageClass    `json:"actual_lowering_storage,omitempty"`
	ProofID               string          `json:"proof_id,omitempty"`
	ProofKind             ProofKind       `json:"proof_kind,omitempty"`
	ProofSubjectBaseID    string          `json:"proof_subject_base_id,omitempty"`
	ProofIndexValueID     string          `json:"proof_index_value_id,omitempty"`
	ProofOperation        string          `json:"proof_operation,omitempty"`
	ProofRange            string          `json:"proof_range,omitempty"`
	ValidationState       ValidationState `json:"validation_state,omitempty"`
	SourceStage           SourceStage     `json:"source_stage,omitempty"`
	ParentFactID          FactID          `json:"parent_fact_id,omitempty"`
	DerivedFactIDs        []FactID        `json:"derived_fact_ids,omitempty"`
	LoweredArtifactID     string          `json:"lowered_artifact_id,omitempty"`
	Claim                 Claim           `json:"claim,omitempty"`
	DecisionCode          string          `json:"decision_code,omitempty"`
	Reason                string          `json:"reason,omitempty"`
	ValidatorName         string          `json:"validator_name,omitempty"`
	CostClass             CostClass       `json:"cost_class,omitempty"`
	NormalBuildCheck      bool            `json:"normal_build_check,omitempty"`
}
