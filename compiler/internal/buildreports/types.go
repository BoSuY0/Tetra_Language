package buildreports

import (
	"fmt"
	"reflect"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
	"tetra_language/compiler/internal/plir"
	ctarget "tetra_language/compiler/target"
)

type ReportEnvelope struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	Target        string `json:"target,omitempty"`
}

type BoundsReport struct {
	ReportEnvelope
	Totals    BoundsTotals        `json:"totals"`
	Functions []BoundsFunctionRow `json:"functions"`
}

type BoundsTotals struct {
	Removed int `json:"removed"`
	Left    int `json:"left"`
}

type BoundsFunctionRow struct {
	Function string            `json:"function"`
	Removed  int               `json:"removed"`
	Left     int               `json:"left"`
	Sites    []BoundsCheckSite `json:"sites,omitempty"`
}

type BoundsCheckSite struct {
	Site    string `json:"site,omitempty"`
	Kind    string `json:"kind"`
	Removed bool   `json:"removed"`
	ProofID string `json:"proof_id,omitempty"`
	Reason  string `json:"reason"`
}

type ProofReport struct {
	ReportEnvelope
	Bounds BoundsReport    `json:"bounds"`
	Proofs []ProofEvidence `json:"proofs,omitempty"`
	PLIR   *plir.Program   `json:"plir,omitempty"`
}

type ProofEvidence struct {
	ProofID            string `json:"proof_id"`
	Kind               string `json:"kind"`
	Guard              string `json:"guard,omitempty"`
	Dominates          string `json:"dominates,omitempty"`
	Fact               string `json:"fact,omitempty"`
	Reason             string `json:"reason,omitempty"`
	RemovedBoundsCheck bool   `json:"removed_bounds_check"`
}

type AllocReport struct {
	ReportEnvelope
	Totals    AllocTotals        `json:"totals"`
	Functions []AllocFunctionRow `json:"functions"`
}

type AllocTotals struct {
	Heap           int `json:"heap"`
	Stack          int `json:"stack"`
	ExplicitIsland int `json:"explicit_island"`
	External       int `json:"external"`
	Unknown        int `json:"unknown"`
}

type AllocFunctionRow struct {
	Function    string               `json:"function"`
	Allocations []AllocationDecision `json:"allocations,omitempty"`
}

type AllocationDecision struct {
	Site            string   `json:"site,omitempty"`
	Kind            string   `json:"kind"`
	Storage         string   `json:"storage"`
	Reason          string   `json:"reason"`
	ReasonCodes     []string `json:"reason_codes,omitempty"`
	HeapReasonCodes []string `json:"heap_reason_codes,omitempty"`
}

type AllocationPlanReport struct {
	ReportEnvelope
	TargetMemoryClaimLevel string                   `json:"target_memory_claim_level"`
	StorageEvidenceScope   string                   `json:"storage_evidence_scope"`
	Summary                allocplan.ReportSummary  `json:"summary"`
	Totals                 allocplan.Totals         `json:"totals"`
	Functions              []allocplan.FunctionPlan `json:"functions,omitempty"`
}

func BuildAllocReport(prog *ir.IRProgram, target string) AllocReport {
	report := AllocReport{
		ReportEnvelope: ReportEnvelope{SchemaVersion: 1, Kind: "allocation", Target: target},
	}
	if prog == nil {
		return report
	}
	for _, fn := range prog.Funcs {
		row := AllocFunctionRow{Function: fn.Name}
		for _, instr := range fn.Instrs {
			switch instr.Kind {
			case ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32, ir.IRAllocBytes:
				report.Totals.Heap++
				row.Allocations = append(row.Allocations, AllocationDecision{
					Site:    reportPos(instr.Pos),
					Kind:    irAllocKind(instr.Kind),
					Storage: "Heap",
					Reason: ("allocation planner v0 keeps conservative heap storage until " +
						"escape facts select a narrower class"),
					ReasonCodes:     []string{allocplan.HeapReasonDynamicLifetime},
					HeapReasonCodes: []string{allocplan.HeapReasonDynamicLifetime},
				})
			case ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32:
				report.Totals.Stack++
				row.Allocations = append(row.Allocations, AllocationDecision{
					Site:    reportPos(instr.Pos),
					Kind:    irAllocKind(instr.Kind),
					Storage: "Stack",
					Reason:  "fixed small no-escape allocation lowers to stack frame storage",
				})
			case ir.IRIslandMakeSliceU8,
				ir.IRIslandMakeSliceU16,
				ir.IRIslandMakeSliceI32,
				ir.IRIslandNew:
				report.Totals.ExplicitIsland++
				row.Allocations = append(row.Allocations, AllocationDecision{
					Site:    reportPos(instr.Pos),
					Kind:    irAllocKind(instr.Kind),
					Storage: "ExplicitIsland",
					Reason:  "user-written island scope selects explicit region storage",
				})
			}
		}
		if len(row.Allocations) > 0 {
			report.Functions = append(report.Functions, row)
		}
	}
	return report
}

func WrapAllocationPlanReport(plan *allocplan.Plan, target string) AllocationPlanReport {
	claimLevel, evidenceScope, _ := AllocationPlanTargetStorageScope(target)
	if plan == nil {
		return AllocationPlanReport{
			ReportEnvelope: ReportEnvelope{
				SchemaVersion: 2,
				Kind:          "allocation_plan",
				Target:        target,
			},
			TargetMemoryClaimLevel: claimLevel,
			StorageEvidenceScope:   evidenceScope,
			Summary:                allocplan.Summarize(nil),
		}
	}
	return AllocationPlanReport{
		ReportEnvelope: ReportEnvelope{
			SchemaVersion: 2,
			Kind:          "allocation_plan",
			Target:        target,
		},
		TargetMemoryClaimLevel: claimLevel,
		StorageEvidenceScope:   evidenceScope,
		Summary:                allocplan.Summarize(plan),
		Totals:                 plan.Totals,
		Functions:              plan.Functions,
	}
}

func ValidateAllocationPlanReport(plan *allocplan.Plan, report AllocationPlanReport) error {
	if report.SchemaVersion != 2 || report.Kind != "allocation_plan" {
		return fmt.Errorf(
			"allocation report mismatch: invalid envelope schema=%d kind=%q",
			report.SchemaVersion,
			report.Kind,
		)
	}
	expectedClaimLevel, expectedEvidenceScope, err := AllocationPlanTargetStorageScope(
		report.Target,
	)
	if err != nil {
		return fmt.Errorf("allocation report mismatch: target memory scope: %w", err)
	}
	if report.TargetMemoryClaimLevel != expectedClaimLevel {
		return fmt.Errorf(
			"allocation report mismatch: target_memory_claim_level=%q want %q",
			report.TargetMemoryClaimLevel,
			expectedClaimLevel,
		)
	}
	if report.StorageEvidenceScope != expectedEvidenceScope {
		return fmt.Errorf(
			"allocation report mismatch: storage_evidence_scope=%q want %q",
			report.StorageEvidenceScope,
			expectedEvidenceScope,
		)
	}
	expectedSummary := allocplan.Summarize(plan)
	if !reflect.DeepEqual(report.Summary, expectedSummary) {
		return fmt.Errorf("allocation report mismatch: summary does not match plan")
	}
	if plan == nil {
		if !reflect.DeepEqual(report.Totals, allocplan.Totals{}) || len(report.Functions) != 0 {
			return fmt.Errorf("allocation report mismatch: non-empty report for nil plan")
		}
		return nil
	}
	if !reflect.DeepEqual(report.Totals, plan.Totals) {
		return fmt.Errorf("allocation report mismatch: totals do not match plan")
	}
	if !reflect.DeepEqual(report.Functions, plan.Functions) {
		return fmt.Errorf("allocation report mismatch: functions do not match plan")
	}
	return nil
}

func AllocationPlanTargetStorageScope(triple string) (string, string, error) {
	tgt, err := ctarget.Parse(triple)
	if err != nil {
		return "", "", err
	}
	switch tgt.MemoryClaimLevel {
	case "production/host_runtime":
		return tgt.MemoryClaimLevel, "host_runtime_verified", nil
	case "build_lower_only unless run":
		return tgt.MemoryClaimLevel, "build_lower_only_target_host_required", nil
	case "artifact/runtime tiered":
		return tgt.MemoryClaimLevel, "artifact_runtime_tiered_safe_limited", nil
	case "build_lower_only":
		return tgt.MemoryClaimLevel, "build_lower_only", nil
	default:
		return tgt.MemoryClaimLevel, "target_capability_matrix", nil
	}
}

func irAllocKind(kind ir.IRInstrKind) string {
	switch kind {
	case ir.IRMakeSliceU8:
		return "make_u8"
	case ir.IRMakeSliceU16:
		return "make_u16"
	case ir.IRMakeSliceI32:
		return "make_i32_or_bool"
	case ir.IRStackSliceU8:
		return "stack_make_u8"
	case ir.IRStackSliceU16:
		return "stack_make_u16"
	case ir.IRStackSliceI32:
		return "stack_make_i32_or_bool"
	case ir.IRAllocBytes:
		return "alloc_bytes"
	case ir.IRIslandNew:
		return "island_new"
	case ir.IRIslandMakeSliceU8:
		return "island_make_u8"
	case ir.IRIslandMakeSliceU16:
		return "island_make_u16"
	case ir.IRIslandMakeSliceI32:
		return "island_make_i32_or_bool"
	default:
		return fmt.Sprintf("ir.%d", kind)
	}
}

func reportPos(pos frontend.Position) string {
	if pos.Line == 0 && pos.Col == 0 && pos.File == "" {
		return ""
	}
	return frontend.FormatPos(pos)
}

type BackendReport struct {
	ReportEnvelope
	Backend          string                         `json:"backend"`
	Mode             string                         `json:"mode"`
	Summary          BackendCoverageSummary         `json:"summary"`
	Functions        []BackendFunctionPathReport    `json:"functions,omitempty"`
	MachineFunctions []MachineBackendFunctionReport `json:"machine_functions,omitempty"`
}

type BackendCoverageSummary struct {
	FunctionCount                 int                          `json:"function_count"`
	RegisterPath                  int                          `json:"register_path"`
	StackFallback                 int                          `json:"stack_fallback"`
	MachineRegisterNoStackChurn   int                          `json:"machine_register_no_stack_churn"`
	MachineRegisterWithStackChurn int                          `json:"machine_register_with_stack_churn"`
	Categories                    map[string]int               `json:"categories"`
	OrdinaryCorpus                BackendOrdinaryCorpusSummary `json:"ordinary_corpus"`
	ABIBoundaries                 BackendABIBoundarySummary    `json:"abi_boundaries"`
	HotnessSource                 string                       `json:"hotness_source"`
	RuntimeFeaturesRequired       []string                     `json:"runtime_features_required"`
	RuntimeFeaturesLinked         []string                     `json:"runtime_features_linked"`
	RuntimeFeaturesInitialized    []string                     `json:"runtime_features_initialized"`
	RuntimeLazyInitBlockers       []string                     `json:"runtime_lazy_init_blockers"`
	RuntimeFeatureEvidenceClass   string                       `json:"runtime_feature_evidence_class"`
	RuntimeFeatureEvidenceMethod  string                       `json:"runtime_feature_evidence_method"`
	RuntimeObjectPlan             BackendRuntimeObjectPlan     `json:"runtime_object_plan"`
}

type BackendOrdinaryCorpusSummary struct {
	FunctionCount                int            `json:"function_count"`
	RegisterPath                 int            `json:"register_path"`
	RegisterNoStackChurn         int            `json:"register_no_stack_churn"`
	RegisterWithStackChurn       int            `json:"register_with_stack_churn"`
	RegisterNoStackChurnMajority bool           `json:"register_no_stack_churn_majority"`
	StackFallback                int            `json:"stack_fallback"`
	StackFallbackReasons         map[string]int `json:"stack_fallback_reasons"`
	EvidenceSource               string         `json:"evidence_source"`
}

type BackendABIBoundarySummary struct {
	SingleSlotRegisterReturn         int            `json:"single_slot_register_return"`
	SingleSlotStackFallback          int            `json:"single_slot_stack_fallback"`
	MultiSlotReturnStackFallback     int            `json:"multi_slot_return_stack_fallback"`
	CallMultiSlotReturnStackFallback int            `json:"call_multi_slot_return_stack_fallback"`
	ValueClasses                     map[string]int `json:"value_classes"`
}

type BackendRuntimeObjectPlan struct {
	EvidenceClass                    string   `json:"evidence_class"`
	EvidenceMethod                   string   `json:"evidence_method"`
	RuntimeUsed                      bool     `json:"runtime_used"`
	RuntimeObjectLinked              bool     `json:"runtime_object_linked"`
	RuntimeObjectInitialized         bool     `json:"runtime_object_initialized"`
	RuntimeObjectOverride            bool     `json:"runtime_object_override"`
	TimeOnlyRuntime                  bool     `json:"time_only_runtime"`
	LinuxMinimalRuntime              bool     `json:"linux_minimal_runtime"`
	RuntimeObjectFeaturesRequired    []string `json:"runtime_object_features_required"`
	RuntimeObjectFeaturesLinked      []string `json:"runtime_object_features_linked"`
	RuntimeObjectFeaturesInitialized []string `json:"runtime_object_features_initialized"`
	RuntimeObjectLazyInitBlockers    []string `json:"runtime_object_lazy_init_blockers"`
}

type BackendFunctionPathReport struct {
	Function                     string                   `json:"function"`
	BackendPath                  string                   `json:"backend_path"`
	Category                     string                   `json:"category"`
	ABI                          BackendABIBoundaryReport `json:"abi"`
	Detail                       string                   `json:"detail,omitempty"`
	Reason                       string                   `json:"reason,omitempty"`
	HotnessRank                  int                      `json:"hotness_rank"`
	HotnessSource                string                   `json:"hotness_source"`
	RuntimeFeaturesRequired      []string                 `json:"runtime_features_required"`
	RuntimeFeaturesLinked        []string                 `json:"runtime_features_linked"`
	RuntimeFeaturesInitialized   []string                 `json:"runtime_features_initialized"`
	RuntimeLazyInitBlockers      []string                 `json:"runtime_lazy_init_blockers"`
	RuntimeFeatureEvidenceClass  string                   `json:"runtime_feature_evidence_class"`
	RuntimeFeatureEvidenceMethod string                   `json:"runtime_feature_evidence_method"`
}

type BackendABIBoundaryReport struct {
	ReturnSlots            int    `json:"return_slots"`
	MaxRegisterReturnSlots int    `json:"max_register_return_slots"`
	MultiSlotReturnPolicy  string `json:"multi_slot_return_policy"`
	ValueClass             string `json:"value_class"`
	BoundaryStatus         string `json:"boundary_status"`
}

type MachineBackendFunctionReport struct {
	Function             string                  `json:"function"`
	Path                 string                  `json:"path"`
	SSAPath              string                  `json:"ssa_path,omitempty"`
	SSAVerified          bool                    `json:"ssa_verified"`
	InstructionSelection []string                `json:"instruction_selection,omitempty"`
	Validation           MachineValidationReport `json:"validation"`
	Dump                 string                  `json:"dump"`
	Liveness             machine.Liveness        `json:"liveness"`
	Intervals            []machine.Interval      `json:"intervals"`
	Allocation           MachineAllocationReport `json:"allocation"`
	SpillSlots           int                     `json:"spill_slots"`
}

type MachineAllocationReport struct {
	Assignments map[machine.VReg]machine.PhysReg `json:"assignments"`
	Spills      map[machine.VReg]int             `json:"spills"`
}

type MachineValidationReport struct {
	MachineVerifier    string `json:"machine_verifier"`
	AllocationVerifier string `json:"allocation_verifier"`
	SpillReload        string `json:"spill_reload"`
	CallClobbers       string `json:"call_clobbers"`
	StackChurnOps      int    `json:"stack_churn_ops"`
}

type LayoutReport struct {
	ReportEnvelope
	Policy    string              `json:"policy"`
	Summary   LayoutSummary       `json:"summary"`
	Decisions []LayoutDecisionRow `json:"decisions"`
	Claims    []string            `json:"claims"`
}

type LayoutSummary struct {
	Structs              int `json:"structs"`
	DefaultCompilerOwned int `json:"default_compiler_owned"`
	ReprCABILocked       int `json:"repr_c_abi_locked"`
	ExportedPublicABI    int `json:"exported_public_abi"`
}

type LayoutDecisionRow struct {
	Type               string           `json:"type"`
	Module             string           `json:"module,omitempty"`
	Repr               string           `json:"repr"`
	Public             bool             `json:"public"`
	ABILocked          bool             `json:"abi_locked"`
	PublicABI          string           `json:"public_abi"`
	Decision           string           `json:"decision"`
	SourceFieldOrder   []string         `json:"source_field_order,omitempty"`
	CurrentFieldLayout []LayoutFieldRow `json:"current_field_layout,omitempty"`
	AllowedTransforms  []string         `json:"allowed_transforms,omitempty"`
	DeniedTransforms   []string         `json:"denied_transforms,omitempty"`
	Reason             string           `json:"reason"`
}

type LayoutFieldRow struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Offset    int    `json:"offset"`
	SlotCount int    `json:"slot_count"`
}

type PerfReport struct {
	ReportEnvelope
	MatrixScope  string                            `json:"matrix_scope,omitempty"`
	MatrixReport string                            `json:"matrix_report,omitempty"`
	Claims       []string                          `json:"claims"`
	Blockers     []PerformanceBlockerRow           `json:"blockers"`
	Benchmarks   []PerformanceBenchmarkExplanation `json:"benchmarks,omitempty"`
}

type PerformanceBlockerRow struct {
	Code      string `json:"code"`
	Component string `json:"component"`
	Message   string `json:"message"`
	CostClass string `json:"cost_class"`
	Evidence  string `json:"evidence"`
	NextStep  string `json:"next_step"`
}

type PerformanceBenchmarkExplanation struct {
	Benchmark    string   `json:"benchmark"`
	Category     string   `json:"category"`
	MatrixScope  string   `json:"matrix_scope"`
	MatrixReport string   `json:"matrix_report"`
	ReasonCodes  []string `json:"reason_codes"`
	Artifacts    []string `json:"artifacts"`
	Explanation  string   `json:"explanation"`
	NextStep     string   `json:"next_step"`
}

type ActorTransferReport struct {
	ReportEnvelope
	Totals    ActorTransferTotals `json:"totals"`
	Mailboxes []ActorMailboxRow   `json:"mailboxes,omitempty"`
	Sends     []ActorTransferRow  `json:"sends,omitempty"`
}

type ActorTransferTotals struct {
	Copy         int `json:"copy"`
	Move         int `json:"move"`
	ZeroCopyMove int `json:"zero_copy_move"`
	BytesCopied  int `json:"bytes_copied"`
}

type ActorTransferRow struct {
	Function                   string `json:"function,omitempty"`
	Site                       string `json:"site,omitempty"`
	MessageType                string `json:"message_type,omitempty"`
	Case                       string `json:"case,omitempty"`
	PayloadIndex               int    `json:"payload_index,omitempty"`
	PayloadType                string `json:"payload_type,omitempty"`
	Ownership                  string `json:"ownership,omitempty"`
	Owner                      string `json:"owner,omitempty"`
	TransferMode               string `json:"transfer_mode"`
	RuntimePath                string `json:"runtime_path"`
	BytesCopied                int    `json:"bytes_copied"`
	ZeroCopy                   bool   `json:"zero_copy"`
	ClaimLevel                 string `json:"claim_level"`
	BoundaryScope              string `json:"boundary_scope,omitempty"`
	ProductionRuntimeValidated bool   `json:"production_runtime_validated"`
	Reason                     string `json:"reason,omitempty"`
}

type ActorMailboxRow struct {
	Name              string `json:"name"`
	MessageSchema     string `json:"message_schema"`
	Capacity          int    `json:"capacity"`
	CapacityUnit      string `json:"capacity_unit"`
	Backpressure      string `json:"backpressure"`
	OverflowPolicy    string `json:"overflow_policy"`
	MaxPayloadSlots   int    `json:"max_payload_slots"`
	PayloadSlots      int    `json:"payload_slots"`
	SlotWidthBytes    int    `json:"slot_width_bytes"`
	RuntimePath       string `json:"runtime_path"`
	OwnershipMetadata bool   `json:"ownership_metadata"`
}
