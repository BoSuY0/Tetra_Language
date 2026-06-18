package ramcontract

import "time"

const ReportSchemaV1 = "tetra.ram-contract-report.v1"
const GradeReportSchemaV1 = "tetra.memory-grade-report.v1"
const ProofStoreSummarySchemaV1 = "tetra.proof-store-summary.v1"
const PipelineCoverageSchemaV1 = "tetra.validation-pipeline-coverage.v1"
const BlockerReportSchemaV1 = "tetra.ram-blockers.v1"

type MemoryGrade string

const (
	GradeM0 MemoryGrade = "M0"
	GradeM1 MemoryGrade = "M1"
	GradeM2 MemoryGrade = "M2"
	GradeM3 MemoryGrade = "M3"
	GradeM4 MemoryGrade = "M4"
	GradeM5 MemoryGrade = "M5"
	GradeM6 MemoryGrade = "M6"
)

type Placement string

const (
	PlacementEliminated    Placement = "eliminated"
	PlacementRegister      Placement = "register"
	PlacementStack         Placement = "stack"
	PlacementStatic        Placement = "static"
	PlacementInterned      Placement = "interned"
	PlacementIsland        Placement = "island"
	PlacementRegion        Placement = "region"
	PlacementHeapBounded   Placement = "heap_bounded"
	PlacementHeapUnbounded Placement = "heap_unbounded"
	PlacementExternal      Placement = "external"
	PlacementRejected      Placement = "rejected"
)

type Intent string

const (
	IntentAllocation               Intent = "allocation"
	IntentCopy                     Intent = "copy"
	IntentIntern                   Intent = "intern"
	IntentRegionAlloc              Intent = "region_alloc"
	IntentHeapFallback             Intent = "heap_fallback"
	IntentCopyEliminated           Intent = "copy_eliminated"
	IntentCopyStackBacked          Intent = "copy_stack_backed"
	IntentCopyHeapBounded          Intent = "copy_heap_bounded"
	IntentCopyHeapUnbounded        Intent = "copy_heap_unbounded"
	IntentCopyRequiredBoundary     Intent = "copy_required_boundary"
	IntentCopyRequiredMutableAlias Intent = "copy_required_mutable_alias"
	IntentCopyIntoNoAllocation     Intent = "copy_into_no_allocation"
)

type EscapeStatus string

const (
	EscapeNoEscape        EscapeStatus = "no_escape"
	EscapeReturn          EscapeStatus = "escapes_return"
	EscapeCall            EscapeStatus = "escapes_call"
	EscapeActorCrossing   EscapeStatus = "actor_crossing"
	EscapeTaskCrossing    EscapeStatus = "task_crossing"
	EscapeFFICrossing     EscapeStatus = "ffi_crossing"
	EscapeBrowserCrossing EscapeStatus = "browser_crossing"
	EscapeUnsafe          EscapeStatus = "unsafe_exposure"
	EscapeUnknown         EscapeStatus = "unknown"
)

type ValidationStatus string

const (
	ValidationValidated    ValidationStatus = "validated"
	ValidationConservative ValidationStatus = "conservative"
	ValidationRejected     ValidationStatus = "rejected"
	ValidationUnknown      ValidationStatus = "unknown"
)

type MemoryDomainKind string

const (
	DomainProcess  MemoryDomainKind = "process"
	DomainTask     MemoryDomainKind = "task"
	DomainActor    MemoryDomainKind = "actor"
	DomainIsland   MemoryDomainKind = "island"
	DomainRequest  MemoryDomainKind = "request"
	DomainExternal MemoryDomainKind = "external"
)

type Report struct {
	SchemaVersion string         `json:"schema_version"`
	GitHead       string         `json:"git_head,omitempty"`
	Target        string         `json:"target"`
	GeneratedBy   string         `json:"generated_by"`
	GeneratedAt   string         `json:"generated_at,omitempty"`
	Functions     []FunctionRow  `json:"functions,omitempty"`
	Rows          []Row          `json:"rows"`
	Proofs        []ProofSummary `json:"proofs,omitempty"`
	Summary       Summary        `json:"summary"`
	NonClaims     []string       `json:"non_claims"`
}

type Row struct {
	SiteID           string           `json:"site_id"`
	ValueID          string           `json:"value_id"`
	Function         string           `json:"function"`
	SourceSpan       string           `json:"source_span,omitempty"`
	Intent           Intent           `json:"intent"`
	RequestedBytes   int64            `json:"requested_bytes"`
	Bounded          bool             `json:"bounded"`
	Owner            string           `json:"owner"`
	Lifetime         string           `json:"lifetime"`
	EscapeStatus     EscapeStatus     `json:"escape_status"`
	Placement        Placement        `json:"placement"`
	ProofIDs         []string         `json:"proof_ids"`
	Blockers         []string         `json:"blockers"`
	ReasonCodes      []string         `json:"reason_codes,omitempty"`
	HeapReasonCodes  []string         `json:"heap_reason_codes,omitempty"`
	CopyReason       string           `json:"copy_reason,omitempty"`
	FreePoint        string           `json:"free_point,omitempty"`
	ContractGrade    MemoryGrade      `json:"contract_grade"`
	ValidationStatus ValidationStatus `json:"validation_status"`
	SourceFactID     string           `json:"source_fact_id,omitempty"`
	Domain           *MemoryDomain    `json:"domain,omitempty"`
}

type MemoryDomain struct {
	DomainID       string           `json:"domain_id"`
	ParentDomainID string           `json:"parent_domain_id,omitempty"`
	Kind           MemoryDomainKind `json:"kind"`
	OwnerKind      string           `json:"owner_kind"`
	OwnerID        string           `json:"owner_id"`
	Lifetime       string           `json:"lifetime"`
	BudgetBytes    int64            `json:"budget_bytes,omitempty"`
	RequestedBytes int64            `json:"requested_bytes,omitempty"`
	ReservedBytes  int64            `json:"reserved_bytes,omitempty"`
	CommittedBytes int64            `json:"committed_bytes,omitempty"`
	ReleasedBytes  int64            `json:"released_bytes,omitempty"`
	CurrentBytes   int64            `json:"current_bytes,omitempty"`
	PeakBytes      int64            `json:"peak_bytes,omitempty"`
	CopyCount      int              `json:"copy_count,omitempty"`
	BytesCopied    int64            `json:"bytes_copied,omitempty"`
}

type ProofSummary struct {
	ProofID    string `json:"proof_id"`
	Kind       string `json:"kind"`
	Subject    string `json:"subject"`
	StableHash string `json:"stable_hash"`
	Status     string `json:"status"`
}

type FunctionRow struct {
	Function    string      `json:"function"`
	Grade       MemoryGrade `json:"grade"`
	RowCount    int         `json:"row_count"`
	HeapRows    int         `json:"heap_rows"`
	CopyRows    int         `json:"copy_rows"`
	BudgetBytes int64       `json:"budget_bytes"`
}

type Summary struct {
	RowCount      int                   `json:"row_count"`
	ArtifactGrade MemoryGrade           `json:"artifact_grade"`
	HeapRows      int                   `json:"heap_rows"`
	CopyRows      int                   `json:"copy_rows"`
	UnboundedRows int                   `json:"unbounded_rows"`
	BudgetBytes   int64                 `json:"budget_bytes"`
	Domains       []MemoryDomainSummary `json:"domains,omitempty"`
}

type MemoryDomainSummary struct {
	DomainID       string           `json:"domain_id"`
	ParentDomainID string           `json:"parent_domain_id,omitempty"`
	Kind           MemoryDomainKind `json:"kind"`
	OwnerKind      string           `json:"owner_kind"`
	OwnerID        string           `json:"owner_id"`
	Lifetime       string           `json:"lifetime"`
	RowCount       int              `json:"row_count"`
	BudgetBytes    int64            `json:"budget_bytes,omitempty"`
	RequestedBytes int64            `json:"requested_bytes,omitempty"`
	ReservedBytes  int64            `json:"reserved_bytes,omitempty"`
	CommittedBytes int64            `json:"committed_bytes,omitempty"`
	ReleasedBytes  int64            `json:"released_bytes,omitempty"`
	CurrentBytes   int64            `json:"current_bytes,omitempty"`
	PeakBytes      int64            `json:"peak_bytes,omitempty"`
	CopyCount      int              `json:"copy_count,omitempty"`
	BytesCopied    int64            `json:"bytes_copied,omitempty"`
}

type GradeReport struct {
	SchemaVersion string        `json:"schema_version"`
	GitHead       string        `json:"git_head,omitempty"`
	Target        string        `json:"target"`
	GeneratedBy   string        `json:"generated_by"`
	ArtifactGrade MemoryGrade   `json:"artifact_grade"`
	Functions     []FunctionRow `json:"functions"`
	Summary       Summary       `json:"summary"`
	NonClaims     []string      `json:"non_claims"`
}

type ProofStoreSummary struct {
	SchemaVersion string         `json:"schema_version"`
	GitHead       string         `json:"git_head,omitempty"`
	Target        string         `json:"target"`
	GeneratedBy   string         `json:"generated_by"`
	Proofs        []ProofSummary `json:"proofs"`
	Summary       struct {
		ProofCount   int `json:"proof_count"`
		Proven       int `json:"proven"`
		Conservative int `json:"conservative"`
		Rejected     int `json:"rejected"`
		Unknown      int `json:"unknown"`
	} `json:"summary"`
	NonClaims []string `json:"non_claims"`
}

type PipelineCoverageReport struct {
	SchemaVersion string          `json:"schema_version"`
	GitHead       string          `json:"git_head,omitempty"`
	Target        string          `json:"target"`
	GeneratedBy   string          `json:"generated_by"`
	Entries       []PipelineEntry `json:"entries"`
	NonClaims     []string        `json:"non_claims"`
}

type PipelineEntry struct {
	Entrypoint   string   `json:"entrypoint"`
	ArtifactPath string   `json:"artifact_path,omitempty"`
	Status       string   `json:"status"`
	Validators   []string `json:"validators,omitempty"`
	Exemption    string   `json:"exemption,omitempty"`
}

type BlockerReport struct {
	SchemaVersion string       `json:"schema_version"`
	Kind          string       `json:"kind"`
	GitHead       string       `json:"git_head,omitempty"`
	Target        string       `json:"target"`
	GeneratedBy   string       `json:"generated_by"`
	Rows          []BlockerRow `json:"rows"`
	NonClaims     []string     `json:"non_claims"`
}

type BlockerRow struct {
	SiteID               string      `json:"site_id"`
	Function             string      `json:"function"`
	Intent               Intent      `json:"intent"`
	Placement            Placement   `json:"placement"`
	Blockers             []string    `json:"blockers,omitempty"`
	ReasonCodes          []string    `json:"reason_codes,omitempty"`
	HeapReasonCodes      []string    `json:"heap_reason_codes,omitempty"`
	CopyReason           string      `json:"copy_reason,omitempty"`
	ContractGrade        MemoryGrade `json:"contract_grade"`
	File                 string      `json:"file,omitempty"`
	Line                 int         `json:"line,omitempty"`
	Symbol               string      `json:"symbol,omitempty"`
	SourceLocationStatus string      `json:"source_location_status"`
	Severity             string      `json:"severity"`
	Reason               string      `json:"reason"`
	SuggestedFix         string      `json:"suggested_fix"`
	ProofID              string      `json:"proof_id,omitempty"`
	EvidenceID           string      `json:"evidence_id"`
	SafeToOptimize       bool        `json:"safe_to_optimize"`
	CopyKind             string      `json:"copy_kind,omitempty"`
	SourceValue          string      `json:"source_value,omitempty"`
	DestinationValue     string      `json:"destination_value,omitempty"`
	BytesEstimate        int64       `json:"bytes_estimate,omitempty"`
	SafetyReason         string      `json:"safety_reason,omitempty"`
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
