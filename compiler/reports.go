package compiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"tetra_language/compiler/internal/actorsafety"
	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/layoutopt"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/machine"
	"tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/semantics"
	"tetra_language/compiler/internal/ssair"
	"tetra_language/compiler/internal/validation"
	ctarget "tetra_language/compiler/target"
)

type reportEnvelope struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	Target        string `json:"target,omitempty"`
}

type boundsReport struct {
	reportEnvelope
	Totals    boundsTotals        `json:"totals"`
	Functions []boundsFunctionRow `json:"functions"`
}

type boundsTotals struct {
	Removed int `json:"removed"`
	Left    int `json:"left"`
}

type boundsFunctionRow struct {
	Function string            `json:"function"`
	Removed  int               `json:"removed"`
	Left     int               `json:"left"`
	Sites    []boundsCheckSite `json:"sites,omitempty"`
}

type boundsCheckSite struct {
	Site    string `json:"site,omitempty"`
	Kind    string `json:"kind"`
	Removed bool   `json:"removed"`
	ProofID string `json:"proof_id,omitempty"`
	Reason  string `json:"reason"`
}

type proofReport struct {
	reportEnvelope
	Bounds boundsReport    `json:"bounds"`
	Proofs []proofEvidence `json:"proofs,omitempty"`
	PLIR   *plir.Program   `json:"plir,omitempty"`
}

type proofEvidence struct {
	ProofID            string `json:"proof_id"`
	Kind               string `json:"kind"`
	Guard              string `json:"guard,omitempty"`
	Dominates          string `json:"dominates,omitempty"`
	Fact               string `json:"fact,omitempty"`
	Reason             string `json:"reason,omitempty"`
	RemovedBoundsCheck bool   `json:"removed_bounds_check"`
}

type allocReport struct {
	reportEnvelope
	Totals    allocTotals        `json:"totals"`
	Functions []allocFunctionRow `json:"functions"`
}

type allocTotals struct {
	Heap           int `json:"heap"`
	Stack          int `json:"stack"`
	ExplicitIsland int `json:"explicit_island"`
	External       int `json:"external"`
	Unknown        int `json:"unknown"`
}

type allocFunctionRow struct {
	Function    string               `json:"function"`
	Allocations []allocationDecision `json:"allocations,omitempty"`
}

type allocationDecision struct {
	Site    string `json:"site,omitempty"`
	Kind    string `json:"kind"`
	Storage string `json:"storage"`
	Reason  string `json:"reason"`
}

type allocationPlanReport struct {
	reportEnvelope
	TargetMemoryClaimLevel string                   `json:"target_memory_claim_level"`
	StorageEvidenceScope   string                   `json:"storage_evidence_scope"`
	Summary                allocplan.ReportSummary  `json:"summary"`
	Totals                 allocplan.Totals         `json:"totals"`
	Functions              []allocplan.FunctionPlan `json:"functions,omitempty"`
}

type backendReport struct {
	reportEnvelope
	Backend          string                         `json:"backend"`
	Mode             string                         `json:"mode"`
	Summary          backendCoverageSummary         `json:"summary"`
	Functions        []backendFunctionPathReport    `json:"functions,omitempty"`
	MachineFunctions []machineBackendFunctionReport `json:"machine_functions,omitempty"`
}

type backendCoverageSummary struct {
	FunctionCount                 int                          `json:"function_count"`
	RegisterPath                  int                          `json:"register_path"`
	StackFallback                 int                          `json:"stack_fallback"`
	MachineRegisterNoStackChurn   int                          `json:"machine_register_no_stack_churn"`
	MachineRegisterWithStackChurn int                          `json:"machine_register_with_stack_churn"`
	Categories                    map[string]int               `json:"categories"`
	OrdinaryCorpus                backendOrdinaryCorpusSummary `json:"ordinary_corpus"`
	ABIBoundaries                 backendABIBoundarySummary    `json:"abi_boundaries"`
	HotnessSource                 string                       `json:"hotness_source"`
}

type backendOrdinaryCorpusSummary struct {
	FunctionCount                int            `json:"function_count"`
	RegisterPath                 int            `json:"register_path"`
	RegisterNoStackChurn         int            `json:"register_no_stack_churn"`
	RegisterWithStackChurn       int            `json:"register_with_stack_churn"`
	RegisterNoStackChurnMajority bool           `json:"register_no_stack_churn_majority"`
	StackFallback                int            `json:"stack_fallback"`
	StackFallbackReasons         map[string]int `json:"stack_fallback_reasons"`
	EvidenceSource               string         `json:"evidence_source"`
}

type backendABIBoundarySummary struct {
	SingleSlotRegisterReturn         int            `json:"single_slot_register_return"`
	SingleSlotStackFallback          int            `json:"single_slot_stack_fallback"`
	MultiSlotReturnStackFallback     int            `json:"multi_slot_return_stack_fallback"`
	CallMultiSlotReturnStackFallback int            `json:"call_multi_slot_return_stack_fallback"`
	ValueClasses                     map[string]int `json:"value_classes"`
}

type backendFunctionPathReport struct {
	Function      string                   `json:"function"`
	BackendPath   string                   `json:"backend_path"`
	Category      string                   `json:"category"`
	ABI           backendABIBoundaryReport `json:"abi"`
	Detail        string                   `json:"detail,omitempty"`
	Reason        string                   `json:"reason,omitempty"`
	HotnessRank   int                      `json:"hotness_rank"`
	HotnessSource string                   `json:"hotness_source"`
}

type backendABIBoundaryReport struct {
	ReturnSlots            int    `json:"return_slots"`
	MaxRegisterReturnSlots int    `json:"max_register_return_slots"`
	MultiSlotReturnPolicy  string `json:"multi_slot_return_policy"`
	ValueClass             string `json:"value_class"`
	BoundaryStatus         string `json:"boundary_status"`
}

type machineBackendFunctionReport struct {
	Function             string                  `json:"function"`
	Path                 string                  `json:"path"`
	SSAPath              string                  `json:"ssa_path,omitempty"`
	SSAVerified          bool                    `json:"ssa_verified"`
	InstructionSelection []string                `json:"instruction_selection,omitempty"`
	Validation           machineValidationReport `json:"validation"`
	Dump                 string                  `json:"dump"`
	Liveness             machine.Liveness        `json:"liveness"`
	Intervals            []machine.Interval      `json:"intervals"`
	Allocation           machineAllocationReport `json:"allocation"`
	SpillSlots           int                     `json:"spill_slots"`
}

type machineAllocationReport struct {
	Assignments map[machine.VReg]machine.PhysReg `json:"assignments"`
	Spills      map[machine.VReg]int             `json:"spills"`
}

type machineValidationReport struct {
	MachineVerifier    string `json:"machine_verifier"`
	AllocationVerifier string `json:"allocation_verifier"`
	SpillReload        string `json:"spill_reload"`
	CallClobbers       string `json:"call_clobbers"`
	StackChurnOps      int    `json:"stack_churn_ops"`
}

type layoutReport struct {
	reportEnvelope
	Policy    string              `json:"policy"`
	Summary   layoutSummary       `json:"summary"`
	Decisions []layoutDecisionRow `json:"decisions"`
	Claims    []string            `json:"claims"`
}

type layoutSummary struct {
	Structs              int `json:"structs"`
	DefaultCompilerOwned int `json:"default_compiler_owned"`
	ReprCABILocked       int `json:"repr_c_abi_locked"`
	ExportedPublicABI    int `json:"exported_public_abi"`
}

type layoutDecisionRow struct {
	Type               string           `json:"type"`
	Module             string           `json:"module,omitempty"`
	Repr               string           `json:"repr"`
	Public             bool             `json:"public"`
	ABILocked          bool             `json:"abi_locked"`
	PublicABI          string           `json:"public_abi"`
	Decision           string           `json:"decision"`
	SourceFieldOrder   []string         `json:"source_field_order,omitempty"`
	CurrentFieldLayout []layoutFieldRow `json:"current_field_layout,omitempty"`
	AllowedTransforms  []string         `json:"allowed_transforms,omitempty"`
	DeniedTransforms   []string         `json:"denied_transforms,omitempty"`
	Reason             string           `json:"reason"`
}

type layoutFieldRow struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Offset    int    `json:"offset"`
	SlotCount int    `json:"slot_count"`
}

type perfReport struct {
	reportEnvelope
	MatrixScope  string                            `json:"matrix_scope,omitempty"`
	MatrixReport string                            `json:"matrix_report,omitempty"`
	Claims       []string                          `json:"claims"`
	Blockers     []performanceBlockerRow           `json:"blockers"`
	Benchmarks   []performanceBenchmarkExplanation `json:"benchmarks,omitempty"`
}

type performanceBlockerRow struct {
	Code      string `json:"code"`
	Component string `json:"component"`
	Message   string `json:"message"`
	CostClass string `json:"cost_class"`
	Evidence  string `json:"evidence"`
	NextStep  string `json:"next_step"`
}

type performanceBenchmarkExplanation struct {
	Benchmark    string   `json:"benchmark"`
	Category     string   `json:"category"`
	MatrixScope  string   `json:"matrix_scope"`
	MatrixReport string   `json:"matrix_report"`
	ReasonCodes  []string `json:"reason_codes"`
	Artifacts    []string `json:"artifacts"`
	Explanation  string   `json:"explanation"`
	NextStep     string   `json:"next_step"`
}

type actorTransferReport struct {
	reportEnvelope
	Totals    actorTransferTotals `json:"totals"`
	Mailboxes []actorMailboxRow   `json:"mailboxes,omitempty"`
	Sends     []actorTransferRow  `json:"sends,omitempty"`
}

type actorTransferTotals struct {
	Copy         int `json:"copy"`
	Move         int `json:"move"`
	ZeroCopyMove int `json:"zero_copy_move"`
	BytesCopied  int `json:"bytes_copied"`
}

type actorTransferRow struct {
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

type actorMailboxRow struct {
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

func emitExplainReports(outputPath string, target string, checked *semantics.CheckedProgram, opt BuildOptions) error {
	if !opt.Explain && !opt.EmitPLIR && !opt.EmitProof && !opt.EmitBoundsReport && !opt.EmitAllocReport && !opt.EmitMemoryReport {
		return nil
	}
	plirProg, err := plir.FromCheckedProgram(checked)
	if err != nil {
		return err
	}
	if err := plir.VerifyProgram(plirProg); err != nil {
		return err
	}
	allocPlan, err := allocplan.FromPLIRWithOptions(plirProg, allocationPlanOptionsForTarget(target))
	if err != nil {
		return err
	}
	irProg, err := lower.LowerWithOptions(checked, lowerOptionsForTarget(target))
	if err != nil {
		return err
	}
	if err := validation.ValidateAllocationLowering(allocPlan, irProg); err != nil {
		return err
	}
	if _, err := validation.CheckBoundsProofsWithPLIR(irProg, plirProg); err != nil {
		return err
	}
	bounds := buildBoundsReport(irProg, checked, target)
	if opt.Explain || opt.EmitPLIR {
		if err := writeReport(outputPath+".plir.json", plirProg); err != nil {
			return err
		}
		if err := os.WriteFile(outputPath+".plir.txt", []byte(plir.FormatText(plirProg)), 0o644); err != nil {
			return err
		}
	}
	if opt.Explain || opt.EmitProof {
		if err := writeReport(outputPath+".proof.json", buildProofReport(plirProg, bounds, target)); err != nil {
			return err
		}
	}
	if opt.Explain || opt.EmitBoundsReport {
		if err := writeReport(outputPath+".bounds.json", bounds); err != nil {
			return err
		}
	}
	if opt.Explain || opt.EmitAllocReport {
		allocReport := wrapAllocationPlanReport(allocPlan, target)
		if err := validateAllocationPlanReport(allocPlan, allocReport); err != nil {
			return err
		}
		if err := writeReport(outputPath+".alloc.json", allocReport); err != nil {
			return err
		}
		if err := os.WriteFile(outputPath+".alloc.txt", []byte(allocplan.FormatText(allocPlan)), 0o644); err != nil {
			return err
		}
	}
	if opt.EmitMemoryReport {
		graph, err := memoryfacts.FromPLIRAndAllocPlan(target, plirProg, allocPlan)
		if err != nil {
			return err
		}
		report := memoryfacts.BuildReportFromGraph(graph)
		if err := validateMemoryReportForEmission(graph, report); err != nil {
			return err
		}
		if err := writeReport(outputPath+".memory.json", report); err != nil {
			return err
		}
	}
	if opt.Explain {
		backend := buildBackendReport(target, irProg)
		actorTransfers := buildActorTransferReport(checked, target)
		layout := buildLayoutReport(target, checked)
		if err := ValidateLayoutReport(layout); err != nil {
			return err
		}
		for _, report := range []struct {
			path string
			data any
		}{
			{path: outputPath + ".backend.json", data: backend},
			{path: outputPath + ".actor-transfer.json", data: actorTransfers},
			{path: outputPath + ".layout.json", data: layout},
			{path: outputPath + ".perf.json", data: buildPerformanceReport(target)},
		} {
			if err := writeReport(report.path, report.data); err != nil {
				return err
			}
		}
		if err := os.WriteFile(outputPath+".explain.txt", []byte(formatExplainText(target, bounds, allocPlan, plirProg)), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func validateMemoryReportForEmission(graph *memoryfacts.Graph, report memoryfacts.Report) error {
	if err := memoryfacts.ValidateReportProjection(graph, report); err != nil {
		return fmt.Errorf("validate memory report projection: %w", err)
	}
	return nil
}

const p21LayoutPolicy = "p21.0_default_layout_freedom_v1"

var p21LayoutTransforms = []string{
	"field_reordering",
	"padding_removal",
	"hot_cold_splitting",
	"scalar_replacement",
	"aos_to_soa",
}

func buildLayoutReport(target string, checked *semantics.CheckedProgram) layoutReport {
	report := layoutReport{
		reportEnvelope: reportEnvelope{SchemaVersion: 2, Kind: "layout", Target: target},
		Policy:         p21LayoutPolicy,
		Claims: []string{
			"default struct layout is compiler-owned",
			"repr(C) locks layout",
			"public ABI/exported FFI requires explicit repr(C)",
			"layout reports show decisions",
			"Default struct layout is compiler-owned and does not promise C field order, padding, or public ABI layout.",
			"repr(C) locks layout for ABI-facing code and denies field reordering, padding removal, hot/cold splitting, scalar replacement, and AoS-to-SoA transforms.",
			"public ABI/exported FFI requires explicit repr(C).",
			"No field reordering, padding removal, hot/cold splitting, scalar replacement, AoS-to-SoA transform, performance change, or runtime behavior change is claimed by this report.",
		},
	}
	if checked == nil {
		return report
	}
	exported := exportedLayoutABITypeUses(checked)
	for _, st := range checked.Structs {
		if st.Decl == nil {
			continue
		}
		info := checked.Types[st.Name]
		policy := layoutopt.PolicyForStruct(*st.Decl)
		row := layoutDecisionRow{
			Type:             st.Name,
			Module:           st.Module,
			Repr:             policy.Repr,
			Public:           info != nil && info.Public,
			ABILocked:        policy.ABILocked,
			SourceFieldOrder: sourceFieldOrder(st.Decl),
			PublicABI:        "not_public_abi",
			Reason:           "default struct layout is compiler-owned; public ABI/exported FFI requires explicit repr(C)",
		}
		if info != nil {
			row.CurrentFieldLayout = layoutFieldRows(info.Fields)
		}
		if policy.Repr == frontend.StructReprC {
			row.Decision = "abi_locked_repr_c"
			row.Reason = "repr(C) locks layout; public ABI/exported FFI requires explicit repr(C)"
			row.DeniedTransforms = append([]string(nil), p21LayoutTransforms...)
		} else {
			row.Decision = "compiler_owned_default"
			row.AllowedTransforms = allowedLayoutTransforms(policy)
			row.DeniedTransforms = deniedLayoutTransforms(policy)
		}
		if _, ok := exported[st.Name]; ok {
			report.Summary.ExportedPublicABI++
			if policy.Repr == frontend.StructReprC {
				row.PublicABI = "exported_ffi_explicit_repr_c"
			} else {
				row.PublicABI = "exported_ffi_missing_explicit_repr"
			}
		}
		switch policy.Repr {
		case frontend.StructReprC:
			report.Summary.ReprCABILocked++
		default:
			report.Summary.DefaultCompilerOwned++
		}
		report.Decisions = append(report.Decisions, row)
	}
	sort.Slice(report.Decisions, func(i, j int) bool {
		return report.Decisions[i].Type < report.Decisions[j].Type
	})
	report.Summary.Structs = len(report.Decisions)
	return report
}

func ValidateLayoutReport(report layoutReport) error {
	if report.SchemaVersion != 2 {
		return fmt.Errorf("layout report schema_version = %d, want 2", report.SchemaVersion)
	}
	if report.Kind != "layout" {
		return fmt.Errorf("layout report kind = %q, want layout", report.Kind)
	}
	if strings.TrimSpace(report.Target) == "" {
		return fmt.Errorf("layout report target is required")
	}
	if report.Policy != p21LayoutPolicy {
		return fmt.Errorf("layout report policy = %q, want %q", report.Policy, p21LayoutPolicy)
	}
	if report.Summary.Structs != len(report.Decisions) {
		return fmt.Errorf("layout report summary structs = %d, decisions = %d", report.Summary.Structs, len(report.Decisions))
	}
	counts := layoutSummary{}
	for _, row := range report.Decisions {
		if err := validateLayoutDecisionRow(row); err != nil {
			return err
		}
		counts.Structs++
		switch row.Repr {
		case frontend.StructReprC:
			counts.ReprCABILocked++
		default:
			counts.DefaultCompilerOwned++
		}
		if strings.HasPrefix(row.PublicABI, "exported_ffi") {
			counts.ExportedPublicABI++
		}
	}
	if !reflect.DeepEqual(report.Summary, counts) {
		return fmt.Errorf("layout report summary mismatch: got %+v want %+v", report.Summary, counts)
	}
	for _, claim := range report.Claims {
		if strings.TrimSpace(claim) == "" || containsWeakReportText(claim) {
			return fmt.Errorf("layout report contains weak claim text %q", claim)
		}
	}
	return nil
}

func validateLayoutDecisionRow(row layoutDecisionRow) error {
	if strings.TrimSpace(row.Type) == "" {
		return fmt.Errorf("layout decision row missing type")
	}
	if strings.TrimSpace(row.Repr) == "" {
		return fmt.Errorf("layout decision row %s missing repr", row.Type)
	}
	if strings.TrimSpace(row.Decision) == "" || strings.TrimSpace(row.PublicABI) == "" || containsWeakReportText(row.Reason) {
		return fmt.Errorf("layout decision row %s has incomplete decision evidence", row.Type)
	}
	switch row.PublicABI {
	case "not_public_abi":
	case "exported_ffi_explicit_repr_c":
		if row.Repr != frontend.StructReprC {
			return fmt.Errorf("layout decision row %s claims exported FFI explicit repr(C) without repr(C)", row.Type)
		}
	case "exported_ffi_missing_explicit_repr":
		return fmt.Errorf("layout decision row %s is exported public ABI without explicit repr(C)", row.Type)
	default:
		return fmt.Errorf("layout decision row %s has unknown public ABI state %q", row.Type, row.PublicABI)
	}
	switch row.Repr {
	case frontend.StructReprC:
		if !row.ABILocked {
			return fmt.Errorf("layout decision row %s repr(C) must be ABI locked", row.Type)
		}
		if row.Decision != "abi_locked_repr_c" {
			return fmt.Errorf("layout decision row %s repr(C) decision = %q", row.Type, row.Decision)
		}
		if len(row.AllowedTransforms) != 0 {
			return fmt.Errorf("layout decision row %s repr(C) must not allow layout transforms", row.Type)
		}
		for _, transform := range p21LayoutTransforms {
			if !stringListContains(row.DeniedTransforms, transform) {
				return fmt.Errorf("layout decision row %s repr(C) missing denied transform %q", row.Type, transform)
			}
		}
	default:
		if row.ABILocked {
			return fmt.Errorf("layout decision row %s default struct must not claim ABI lock", row.Type)
		}
		if row.Decision != "compiler_owned_default" {
			return fmt.Errorf("layout decision row %s default decision = %q", row.Type, row.Decision)
		}
		for _, transform := range p21LayoutTransforms {
			if !stringListContains(row.AllowedTransforms, transform) {
				return fmt.Errorf("layout decision row %s default struct missing allowed transform %q", row.Type, transform)
			}
		}
	}
	return nil
}

func exportedLayoutABITypeUses(checked *semantics.CheckedProgram) map[string]struct{} {
	out := map[string]struct{}{}
	if checked == nil {
		return out
	}
	for _, fn := range checked.Funcs {
		if fn.Decl == nil || fn.Decl.ExportName == "" {
			continue
		}
		sig, ok := checked.FuncSigs[fn.Name]
		if !ok {
			continue
		}
		for _, typ := range sig.ParamTypes {
			collectStructLayoutABITypeUse(typ, checked.Types, out, map[string]bool{})
		}
		collectStructLayoutABITypeUse(sig.ReturnType, checked.Types, out, map[string]bool{})
	}
	return out
}

func collectStructLayoutABITypeUse(typeName string, types map[string]*semantics.TypeInfo, out map[string]struct{}, visiting map[string]bool) {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" || typeName == "none" || visiting[typeName] {
		return
	}
	info := types[typeName]
	if info == nil {
		return
	}
	visiting[typeName] = true
	defer delete(visiting, typeName)
	switch info.Kind {
	case semantics.TypeStruct:
		out[typeName] = struct{}{}
		for _, field := range info.Fields {
			collectStructLayoutABITypeUse(field.TypeName, types, out, visiting)
		}
	case semantics.TypeArray, semantics.TypeOptional:
		collectStructLayoutABITypeUse(info.ElemType, types, out, visiting)
	}
}

func sourceFieldOrder(st *frontend.StructDecl) []string {
	if st == nil {
		return nil
	}
	out := make([]string, 0, len(st.Fields))
	for _, field := range st.Fields {
		out = append(out, field.Name)
	}
	return out
}

func layoutFieldRows(fields []semantics.FieldInfo) []layoutFieldRow {
	out := make([]layoutFieldRow, 0, len(fields))
	for _, field := range fields {
		out = append(out, layoutFieldRow{
			Name:      field.Name,
			Type:      field.TypeName,
			Offset:    field.Offset,
			SlotCount: field.SlotCount,
		})
	}
	return out
}

func allowedLayoutTransforms(policy layoutopt.LayoutPolicy) []string {
	out := []string{}
	if policy.MayReorderFields {
		out = append(out, "field_reordering")
	}
	if policy.MayPackFields {
		out = append(out, "padding_removal")
	}
	if policy.MaySplitHotCold {
		out = append(out, "hot_cold_splitting")
	}
	if policy.MayScalarReplace {
		out = append(out, "scalar_replacement")
	}
	if policy.MayTransformAoSToSoA {
		out = append(out, "aos_to_soa")
	}
	return out
}

func deniedLayoutTransforms(policy layoutopt.LayoutPolicy) []string {
	allowed := allowedLayoutTransforms(policy)
	out := []string{}
	for _, transform := range p21LayoutTransforms {
		if !stringListContains(allowed, transform) {
			out = append(out, transform)
		}
	}
	return out
}

func stringListContains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func buildPerformanceReport(target string) perfReport {
	return perfReport{
		reportEnvelope: reportEnvelope{SchemaVersion: 3, Kind: "perf", Target: target},
		MatrixScope:    p20PerformanceMatrixScope,
		MatrixReport:   p20PerformanceMatrixReport,
		Claims: []string{
			"No broad performance claim is made without benchmark evidence.",
			"Allowed claims must cite a benchmark, report artifact, target, and measured comparison row.",
			"No measured speed comparison, C++/Rust parity, official benchmark result, official TechEmpower result, P20.2 claim tier, optimizer behavior change, or runtime behavior change is claimed by this report.",
		},
		Blockers:   p20PerformanceBlockers(),
		Benchmarks: p20PerformanceBenchmarkExplanations(),
	}
}

const (
	p20PerformanceMatrixScope     = "p20.0_benchmark_matrix"
	p20PerformanceMatrixReport    = "reports/benchmark-matrix-hardening-v1/benchmarks/p20-matrix-hardening-report.json"
	p20PerformanceReportArtifact  = "reports/benchmark-matrix-hardening-v1/benchmarks/artifacts/p20-matrix-hardening.perf.json"
	p20PerformanceProofArtifact   = "reports/benchmark-matrix-hardening-v1/benchmarks/artifacts/p20-matrix-hardening.proof.json"
	p20PerformanceAllocArtifact   = "reports/benchmark-matrix-hardening-v1/benchmarks/artifacts/p20-matrix-hardening.allocation.json"
	p20PerformanceBoundsArtifact  = "reports/benchmark-matrix-hardening-v1/benchmarks/artifacts/p20-matrix-hardening.bounds.json"
	p20PerformanceBackendArtifact = "reports/benchmark-matrix-hardening-v1/benchmarks/artifacts/p20-matrix-hardening.backend.json"
	p20PerformanceActorArtifact   = "reports/benchmark-matrix-hardening-v1/benchmarks/artifacts/p20-matrix-hardening.actor-transfer.json"
)

func p20PerformanceBlockers() []performanceBlockerRow {
	rows := []performanceBlockerRow{
		{
			Code:      "bounds.missing_dominance",
			Component: "bounds-check-elimination",
			Message:   "left bounds check: missing dominance",
			CostClass: "dynamic_check_required",
			Evidence:  "bounds/proof reports must show a proof_id, guard, and dominance before the bounds report may mark the check removed",
			NextStep:  "add or preserve a dominating guard for the indexed access, or keep the checked bounds path",
		},
		{
			Code:      "allocation.return_escape",
			Component: "allocation-planning",
			Message:   "heap allocation: escapes through return",
			CostClass: "conservative_fallback",
			Evidence:  "allocation reports must keep heap or region storage when escape analysis classifies a returned value",
			NextStep:  "return a caller-owned view/copy or make the lifetime explicit before expecting stack lowering",
		},
		{
			Code:      "allocation.unknown_call",
			Component: "allocation-planning",
			Message:   "heap allocation: unknown call",
			CostClass: "conservative_fallback",
			Evidence:  "allocation reports must stay conservative when a call boundary lacks escape/lifetime facts",
			NextStep:  "inline or summarize the callee, add lifetime/effect facts, or keep heap/region storage",
		},
		{
			Code:      "vector.no_noalias_proof",
			Component: "vectorization",
			Message:   "not vectorized: no noalias proof",
			CostClass: "dynamic_check_required",
			Evidence:  "vectorization reports require provenance/noalias facts before selecting a vector path that could observe aliasing",
			NextStep:  "prove source/destination disjointness or keep the scalar path selected",
		},
		{
			Code:      "inline.code_size_budget",
			Component: "inlining",
			Message:   "not inlined: code-size budget",
			CostClass: "instrumentation_only",
			Evidence:  "inlining reports must preserve not_inlined reasons when a body exceeds the current budget",
			NextStep:  "reduce the callee body or accept the call boundary until the budget changes with validation",
		},
		{
			Code:      "register_spill.live_range_pressure",
			Component: "register-allocation",
			Message:   "register spill: live range pressure",
			CostClass: "instrumentation_only",
			Evidence:  "backend reports expose machine intervals, allocation decisions, and spill slots for register-path functions",
			NextStep:  "shorten live ranges, split temporaries, or inspect the machine backend allocation row",
		},
		{
			Code:      "stack_fallback.unsupported_aggregate_return",
			Component: "backend-selection",
			Message:   "stack fallback: unsupported aggregate return",
			CostClass: "conservative_fallback",
			Evidence:  "backend reports keep stack fallback rows when the current register ABI cannot return the aggregate shape",
			NextStep:  "use a supported single-slot return shape or wait for aggregate-return register backend evidence",
		},
		{
			Code:      "actor_copy.borrowed_data_boundary",
			Component: "actor-transfer",
			Message:   "actor copy: borrowed data crosses boundary",
			CostClass: "conservative_fallback",
			Evidence:  "actor transfer reports must keep copy rows when borrowed payload data crosses an actor boundary",
			NextStep:  "transfer owned data/region ownership or keep the explicit copy at the actor boundary",
		},
	}
	return append([]performanceBlockerRow(nil), rows...)
}

func p20PerformanceBenchmarkExplanations() []performanceBenchmarkExplanation {
	specs := []struct {
		benchmark string
		category  string
		reasons   []string
	}{
		{benchmark: "integer_loops_tetra", category: "integer loops", reasons: []string{"register_spill.live_range_pressure", "inline.code_size_budget"}},
		{benchmark: "slice_sum_tetra", category: "slice sum", reasons: []string{"bounds.missing_dominance", "vector.no_noalias_proof"}},
		{benchmark: "bounds_check_loops_tetra", category: "bounds-check loops", reasons: []string{"bounds.missing_dominance"}},
		{benchmark: "function_calls_tetra", category: "function calls", reasons: []string{"inline.code_size_budget"}},
		{benchmark: "recursion_tetra", category: "recursion", reasons: []string{"inline.code_size_budget", "register_spill.live_range_pressure"}},
		{benchmark: "matrix_multiply_tetra", category: "matrix multiply", reasons: []string{"vector.no_noalias_proof", "register_spill.live_range_pressure"}},
		{benchmark: "hash_table_tetra", category: "hash table", reasons: []string{"allocation.unknown_call", "inline.code_size_budget"}},
		{benchmark: "allocation_tetra", category: "allocation", reasons: []string{"allocation.return_escape", "allocation.unknown_call"}},
		{benchmark: "region_island_allocation_tetra", category: "region/island allocation", reasons: []string{"allocation.return_escape", "allocation.unknown_call"}},
		{benchmark: "json_parse_stringify_tetra", category: "JSON parse/stringify", reasons: []string{"allocation.unknown_call", "bounds.missing_dominance"}},
		{benchmark: "http_plaintext_json_tetra", category: "HTTP plaintext/json", reasons: []string{"allocation.unknown_call", "inline.code_size_budget"}},
		{benchmark: "postgresql_single_multiple_update_tetra", category: "PostgreSQL single/multiple/update", reasons: []string{"allocation.unknown_call", "stack_fallback.unsupported_aggregate_return"}},
		{benchmark: "actor_ping_pong_tetra", category: "actor ping-pong", reasons: []string{"actor_copy.borrowed_data_boundary"}},
		{benchmark: "parallel_map_reduce_tetra", category: "parallel map/reduce", reasons: []string{"actor_copy.borrowed_data_boundary", "register_spill.live_range_pressure"}},
		{benchmark: "startup_time_tetra", category: "startup time", reasons: []string{"inline.code_size_budget", "allocation.unknown_call"}},
		{benchmark: "binary_size_tetra", category: "binary size", reasons: []string{"inline.code_size_budget"}},
		{benchmark: "compile_time_tetra", category: "compile time", reasons: []string{"inline.code_size_budget", "stack_fallback.unsupported_aggregate_return"}},
	}
	rows := make([]performanceBenchmarkExplanation, 0, len(specs))
	for _, spec := range specs {
		rows = append(rows, performanceBenchmarkExplanation{
			Benchmark:    spec.benchmark,
			Category:     spec.category,
			MatrixScope:  p20PerformanceMatrixScope,
			MatrixReport: p20PerformanceMatrixReport,
			ReasonCodes:  append([]string(nil), spec.reasons...),
			Artifacts: []string{
				p20PerformanceMatrixReport,
				p20PerformanceReportArtifact,
				p20PerformanceProofArtifact,
				p20PerformanceAllocArtifact,
				p20PerformanceBoundsArtifact,
				p20PerformanceBackendArtifact,
				p20PerformanceActorArtifact,
			},
			Explanation: fmt.Sprintf("%s uses the P20.1 blocker map for %s; inspect the cited reason codes and report artifacts before changing source, compiler policy, or benchmark wording.", spec.benchmark, spec.category),
			NextStep:    "open the referenced proof/allocation/bounds/backend/actor-transfer reports, fix the first applicable blocker, then rerun the benchmark report before making any performance claim",
		})
	}
	return rows
}

func ValidatePerformanceBlockerReport(report perfReport) error {
	if report.SchemaVersion != 3 {
		return fmt.Errorf("performance blocker report schema_version = %d, want 3", report.SchemaVersion)
	}
	if report.Kind != "perf" {
		return fmt.Errorf("performance blocker report kind = %q, want perf", report.Kind)
	}
	if strings.TrimSpace(report.Target) == "" {
		return fmt.Errorf("performance blocker report target is required")
	}
	if report.MatrixScope != p20PerformanceMatrixScope {
		return fmt.Errorf("performance blocker report matrix_scope = %q, want %q", report.MatrixScope, p20PerformanceMatrixScope)
	}
	if report.MatrixReport != p20PerformanceMatrixReport {
		return fmt.Errorf("performance blocker report matrix_report = %q, want %q", report.MatrixReport, p20PerformanceMatrixReport)
	}
	if err := validatePerformanceBlockerClaims(report.Claims); err != nil {
		return err
	}
	requiredBlockers := map[string]performanceBlockerRow{}
	for _, row := range p20PerformanceBlockers() {
		requiredBlockers[row.Code] = row
	}
	seenBlockers := map[string]bool{}
	for _, row := range report.Blockers {
		if strings.TrimSpace(row.Code) == "" {
			return fmt.Errorf("performance blocker row missing code")
		}
		if seenBlockers[row.Code] {
			return fmt.Errorf("duplicate performance blocker code %q", row.Code)
		}
		seenBlockers[row.Code] = true
		required, ok := requiredBlockers[row.Code]
		if !ok {
			return fmt.Errorf("unknown performance blocker code %q", row.Code)
		}
		if row.Message != required.Message {
			return fmt.Errorf("performance blocker %s message = %q, want %q", row.Code, row.Message, required.Message)
		}
		if !knownPerformanceCostClass(row.CostClass) {
			return fmt.Errorf("performance blocker %s unknown cost_class %q", row.Code, row.CostClass)
		}
		if row.CostClass != required.CostClass {
			return fmt.Errorf("performance blocker %s cost_class = %q, want %q", row.Code, row.CostClass, required.CostClass)
		}
		if isWeakPerformanceText(row.Component) || isWeakPerformanceText(row.Evidence) || isWeakPerformanceText(row.NextStep) {
			return fmt.Errorf("performance blocker %s has placeholder evidence or next step", row.Code)
		}
	}
	for code := range requiredBlockers {
		if !seenBlockers[code] {
			return fmt.Errorf("performance blocker report missing required blocker %s", code)
		}
	}

	requiredBenchmarks := map[string]string{}
	for _, row := range p20PerformanceBenchmarkExplanations() {
		requiredBenchmarks[row.Benchmark] = row.Category
	}
	knownReasons := map[string]bool{}
	for code := range requiredBlockers {
		knownReasons[code] = true
	}
	seenBenchmarks := map[string]bool{}
	for _, row := range report.Benchmarks {
		if strings.TrimSpace(row.Benchmark) == "" {
			return fmt.Errorf("performance benchmark explanation missing benchmark")
		}
		if seenBenchmarks[row.Benchmark] {
			return fmt.Errorf("duplicate performance benchmark explanation %q", row.Benchmark)
		}
		seenBenchmarks[row.Benchmark] = true
		requiredCategory, ok := requiredBenchmarks[row.Benchmark]
		if !ok {
			return fmt.Errorf("unknown performance benchmark explanation %q", row.Benchmark)
		}
		if row.Category != requiredCategory {
			return fmt.Errorf("performance benchmark %s category = %q, want %q", row.Benchmark, row.Category, requiredCategory)
		}
		if row.MatrixScope != report.MatrixScope || row.MatrixReport != report.MatrixReport {
			return fmt.Errorf("performance benchmark %s matrix linkage = %q/%q", row.Benchmark, row.MatrixScope, row.MatrixReport)
		}
		if len(row.ReasonCodes) == 0 {
			return fmt.Errorf("performance benchmark %s missing reason codes", row.Benchmark)
		}
		for _, code := range row.ReasonCodes {
			if !knownReasons[code] {
				return fmt.Errorf("performance benchmark %s cites unknown reason code %q", row.Benchmark, code)
			}
		}
		if len(row.Artifacts) == 0 {
			return fmt.Errorf("performance benchmark %s missing report artifacts", row.Benchmark)
		}
		if isWeakPerformanceText(row.Explanation) || isWeakPerformanceText(row.NextStep) {
			return fmt.Errorf("performance benchmark %s has placeholder explanation or next step", row.Benchmark)
		}
	}
	for benchmark := range requiredBenchmarks {
		if !seenBenchmarks[benchmark] {
			return fmt.Errorf("performance blocker report missing benchmark explanation %s", benchmark)
		}
	}
	return nil
}

func validatePerformanceBlockerClaims(claims []string) error {
	if len(claims) == 0 {
		return fmt.Errorf("performance blocker report claim policy notes are required")
	}
	for _, claim := range claims {
		lower := strings.ToLower(claim)
		nonClaim := strings.Contains(lower, "no ") || strings.Contains(lower, "not ") || strings.Contains(lower, "without") || strings.Contains(lower, "does not")
		switch {
		case strings.Contains(lower, "fastest language") && !nonClaim:
			return fmt.Errorf("performance blocker report claims fastest language")
		case strings.Contains(lower, "c++/rust parity") && !nonClaim:
			return fmt.Errorf("performance blocker report claims C++/Rust parity")
		case strings.Contains(lower, "official techempower") && !nonClaim:
			return fmt.Errorf("performance blocker report claims official TechEmpower result")
		case strings.Contains(lower, "official benchmark") && !nonClaim:
			return fmt.Errorf("performance blocker report claims official benchmark result")
		case (strings.Contains(lower, "measured speed") || strings.Contains(lower, "speed superiority") || strings.Contains(lower, "throughput advantage") || strings.Contains(lower, "latency advantage")) && !nonClaim:
			return fmt.Errorf("performance blocker report claims measured speed comparison")
		case strings.Contains(lower, "runtime behavior change") && !nonClaim:
			return fmt.Errorf("performance blocker report claims runtime behavior change")
		case (strings.Contains(lower, "zero-cost") || strings.Contains(lower, "zero cost") || strings.Contains(lower, "zero_cost")) && strings.Contains(lower, "dynamic_check_required") && !nonClaim:
			return fmt.Errorf("performance blocker report claims dynamic_check_required as zero-cost")
		case strings.Contains(lower, "unsafe_unknown") && strings.Contains(lower, "trusted") && !nonClaim:
			return fmt.Errorf("performance blocker report claims unsafe_unknown trusted optimization")
		}
	}
	return nil
}

func knownPerformanceCostClass(value string) bool {
	switch value {
	case "zero_cost_proven", "dynamic_check_required", "instrumentation_only", "unsupported_rejected", "conservative_fallback":
		return true
	default:
		return false
	}
}

func isWeakPerformanceText(text string) bool {
	text = strings.TrimSpace(strings.ToLower(text))
	return text == "" || text == "todo" || text == "tbd" || strings.Contains(text, "placeholder")
}

func containsWeakReportText(text string) bool {
	text = strings.TrimSpace(strings.ToLower(text))
	if text == "" {
		return true
	}
	for _, marker := range []string{"todo", "tbd", "placeholder", "fixme"} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func buildBackendReport(target string, irProg *ir.IRProgram) backendReport {
	report := backendReport{
		reportEnvelope: reportEnvelope{SchemaVersion: 2, Kind: "backend", Target: target},
		Backend:        "stack",
		Mode:           "compatibility-with-proof-checked-index-loads",
	}
	if !targetSupportsMachineScalar(target) || irProg == nil {
		report.Functions = buildBackendFunctionPathReports(target, irProg, nil)
		report.Summary = summarizeBackendCoverage(report.Functions, nil)
		return report
	}
	machineReports := buildMachineBackendFunctionReports(target, irProg)
	report.Functions = buildBackendFunctionPathReports(target, irProg, machineReports)
	report.Summary = summarizeBackendCoverage(report.Functions, machineReports)
	if len(machineReports) == 0 {
		return report
	}
	report.Backend = "stack+machine"
	report.MachineFunctions = machineReports
	hasScalar := false
	hasLoop := false
	hasSliceSum := false
	hasCall := false
	for _, fn := range machineReports {
		switch fn.Path {
		case "machine-ir-call", "machine-ir-call-loop":
			hasCall = true
		case "machine-ir-scalar":
			hasScalar = true
		case "machine-ir-loop":
			hasLoop = true
		case "machine-ir-slice-sum":
			hasSliceSum = true
		}
	}
	switch {
	case hasCall && hasSliceSum:
		report.Mode = "machine-ir-scalar-loop-slice-sum-and-calls-for-eligible-functions; stack fallback otherwise"
	case hasCall:
		report.Mode = "machine-ir-calls-for-eligible-functions; stack fallback otherwise"
	case hasSliceSum:
		report.Mode = "machine-ir-scalar-loop-and-slice-sum-for-eligible-functions; stack fallback otherwise"
	case hasScalar && hasLoop:
		report.Mode = "machine-ir-scalar-and-loop-for-eligible-functions; stack fallback otherwise"
	case hasLoop:
		report.Mode = "machine-ir-loop-for-eligible-functions; stack fallback otherwise"
	default:
		report.Mode = "machine-ir-scalar-for-eligible-functions; stack fallback otherwise"
	}
	return report
}

func buildBackendFunctionPathReports(target string, irProg *ir.IRProgram, machineReports []machineBackendFunctionReport) []backendFunctionPathReport {
	if irProg == nil {
		return nil
	}
	machineByFunction := make(map[string]machineBackendFunctionReport, len(machineReports))
	for _, report := range machineReports {
		machineByFunction[report.Function] = report
	}
	rows := make([]backendFunctionPathReport, 0, len(irProg.Funcs))
	for _, fn := range irProg.Funcs {
		if machineReport, ok := machineByFunction[fn.Name]; ok {
			row := backendFunctionPathReport{
				Function:    fn.Name,
				BackendPath: "register",
				Category:    "register_path",
				ABI:         backendABIBoundaryForFunction(target, fn, "register"),
				Detail:      machineReport.Path,
				Reason:      "eligible_machine_ir_subset",
			}
			applyBackendHotness(&row)
			rows = append(rows, row)
			continue
		}
		classification := classifyBackendFallback(target, fn)
		row := backendFunctionPathReport{
			Function:    fn.Name,
			BackendPath: "stack",
			Category:    classification.Category,
			ABI:         backendABIBoundaryForFunction(target, fn, "stack"),
			Detail:      classification.Detail,
			Reason:      classification.Reason,
		}
		applyBackendHotness(&row)
		rows = append(rows, row)
	}
	return rows
}

type backendFallbackClassification struct {
	Category string
	Detail   string
	Reason   string
}

func classifyBackendFallback(target string, fn ir.IRFunc) backendFallbackClassification {
	if fn.ReturnSlots > 2 {
		return backendFallbackClassification{
			Category: "unsupported_aggregate_return",
			Detail:   fmt.Sprintf("return_slots=%d", fn.ReturnSlots),
			Reason:   "unsupported_aggregate_return_uses_stack_fallback",
		}
	}
	if fn.ReturnSlots == 2 {
		return backendFallbackClassification{
			Category: "unsupported_slice_string_return",
			Detail:   "return_slots=2",
			Reason:   "unsupported_slice_or_string_return_uses_stack_fallback",
		}
	}
	callABI := machineCallABIForTarget(target)
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall && backendCallABIUnsupported(instr, callABI) {
			return backendFallbackClassification{
				Category: "unsupported_call_abi",
				Detail:   fmt.Sprintf("call=%s arg_slots=%d ret_slots=%d max_arg_slots=%d max_ret_slots=%d", instr.Name, instr.ArgSlots, instr.RetSlots, callABI.MaxArgSlots, callABI.MaxRetSlots),
				Reason:   "unsupported_call_abi_uses_stack_fallback",
			}
		}
	}
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall && backendCallLooksRuntimeEffect(instr.Name) {
			return backendFallbackClassification{
				Category: "unsupported_effect_runtime_call",
				Detail:   fmt.Sprintf("runtime_call=%s", instr.Name),
				Reason:   "unsupported_effect_runtime_call_uses_stack_fallback",
			}
		}
		if backendIRKindIsEffectRuntime(instr.Kind) {
			return backendFallbackClassification{
				Category: "unsupported_effect_runtime_call",
				Detail:   fmt.Sprintf("ir_kind=%d", instr.Kind),
				Reason:   "unsupported_effect_runtime_call_uses_stack_fallback",
			}
		}
	}
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel || instr.Kind == ir.IRJmp || instr.Kind == ir.IRJmpIfZero {
			return backendFallbackClassification{
				Category: "unsupported_control_flow",
				Detail:   fmt.Sprintf("ir_kind=%d label=%d", instr.Kind, instr.Label),
				Reason:   "unsupported_control_flow_uses_stack_fallback",
			}
		}
	}
	return backendFallbackClassification{
		Category: "stack_fallback",
		Reason:   "unsupported_or_unproven_subset_uses_stack_fallback",
	}
}

func backendCallABIUnsupported(instr ir.IRInstr, callABI machine.CallABIInfo) bool {
	return instr.ArgSlots < 0 || instr.RetSlots < 0 || instr.ArgSlots > callABI.MaxArgSlots || instr.RetSlots > callABI.MaxRetSlots
}

func backendCallLooksRuntimeEffect(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	return strings.HasPrefix(lower, "__tetra_") ||
		strings.HasPrefix(lower, "runtime.") ||
		strings.HasPrefix(lower, "core.")
}

func backendIRKindIsEffectRuntime(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRWrite,
		ir.IRStrLit,
		ir.IRLoadGlobal,
		ir.IRStoreGlobal,
		ir.IRAllocBytes,
		ir.IRMakeSliceU8,
		ir.IRMakeSliceU16,
		ir.IRMakeSliceI32,
		ir.IRStackSliceU8,
		ir.IRStackSliceU16,
		ir.IRStackSliceI32,
		ir.IRRegionEnter,
		ir.IRRegionMakeSliceU8,
		ir.IRRegionMakeSliceU16,
		ir.IRRegionMakeSliceI32,
		ir.IRRegionReset,
		ir.IRRawSliceFromParts,
		ir.IRSliceWindow,
		ir.IRSlicePrefix,
		ir.IRSliceSuffix,
		ir.IRIndexStoreI32,
		ir.IRIndexStoreU8,
		ir.IRIndexStoreU16,
		ir.IRIslandNew,
		ir.IRIslandMakeSliceU8,
		ir.IRIslandMakeSliceU16,
		ir.IRIslandMakeSliceI32,
		ir.IRIslandFree,
		ir.IRCapIO,
		ir.IRCapMem,
		ir.IRMemReadI32,
		ir.IRMemWriteI32,
		ir.IRMemReadU8,
		ir.IRMemWriteU8,
		ir.IRMemReadPtr,
		ir.IRMemWritePtr,
		ir.IRMemWriteArchPtr,
		ir.IRMemReadI32Offset,
		ir.IRMemWriteI32Offset,
		ir.IRMemReadU8Offset,
		ir.IRMemWriteU8Offset,
		ir.IRMemReadPtrOffset,
		ir.IRMemWritePtrOffset,
		ir.IRMemWriteArchPtrOffset,
		ir.IRPtrAdd,
		ir.IRMmioReadI32,
		ir.IRMmioWriteI32,
		ir.IRSymAddr,
		ir.IRCtxSwitch,
		ir.IRAtomicLoadPtr,
		ir.IRAtomicStorePtr,
		ir.IRAtomicExchangePtr,
		ir.IRAtomicFetchAddPtr,
		ir.IRAtomicFetchSubPtr,
		ir.IRAtomicFetchAndPtr,
		ir.IRAtomicFetchOrPtr,
		ir.IRAtomicFetchXorPtr,
		ir.IRAtomicCompareExchangePtr,
		ir.IRAtomicFenceSeqCst,
		ir.IRAtomicFenceRelaxed,
		ir.IRAtomicFenceAcquire,
		ir.IRAtomicFenceRelease,
		ir.IRAtomicFenceAcqRel,
		ir.IRAtomicLoadI32,
		ir.IRAtomicStoreI32,
		ir.IRAtomicExchangeI32,
		ir.IRAtomicCompareExchangeI32,
		ir.IRAtomicFetchAddI32,
		ir.IRAtomicFetchSubI32,
		ir.IRAtomicFetchAndI32,
		ir.IRAtomicFetchOrI32,
		ir.IRAtomicFetchXorI32,
		ir.IRAtomicLoadI64,
		ir.IRAtomicStoreI64,
		ir.IRAtomicExchangeI64,
		ir.IRAtomicCompareExchangeI64,
		ir.IRAtomicFetchAddI64,
		ir.IRAtomicFetchSubI64,
		ir.IRAtomicFetchAndI64,
		ir.IRAtomicFetchOrI64,
		ir.IRAtomicFetchXorI64,
		ir.IRAtomicLoadI8,
		ir.IRAtomicStoreI8,
		ir.IRAtomicExchangeI8,
		ir.IRAtomicCompareExchangeI8,
		ir.IRAtomicFetchAddI8,
		ir.IRAtomicFetchSubI8,
		ir.IRAtomicFetchAndI8,
		ir.IRAtomicFetchOrI8,
		ir.IRAtomicFetchXorI8,
		ir.IRAtomicLoadI16,
		ir.IRAtomicStoreI16,
		ir.IRAtomicExchangeI16,
		ir.IRAtomicCompareExchangeI16,
		ir.IRAtomicFetchAddI16,
		ir.IRAtomicFetchSubI16,
		ir.IRAtomicFetchAndI16,
		ir.IRAtomicFetchOrI16,
		ir.IRAtomicFetchXorI16:
		return true
	default:
		return false
	}
}

func summarizeBackendCoverage(rows []backendFunctionPathReport, machineReports []machineBackendFunctionReport) backendCoverageSummary {
	summary := backendCoverageSummary{
		FunctionCount: len(rows),
		Categories:    map[string]int{},
		HotnessSource: "benchmark-corpus-static-map",
	}
	machineByFunction := make(map[string]machineBackendFunctionReport, len(machineReports))
	for _, report := range machineReports {
		machineByFunction[report.Function] = report
		if report.Validation.StackChurnOps == 0 &&
			report.Validation.MachineVerifier == "pass" &&
			report.Validation.AllocationVerifier == "pass" {
			summary.MachineRegisterNoStackChurn++
			continue
		}
		if report.Validation.StackChurnOps > 0 {
			summary.MachineRegisterWithStackChurn++
		}
	}
	for _, row := range rows {
		if row.BackendPath == "register" {
			summary.RegisterPath++
		}
		if row.BackendPath == "stack" {
			summary.StackFallback++
		}
		if row.Category != "" {
			summary.Categories[row.Category]++
		}
	}
	summary.OrdinaryCorpus = summarizeBackendOrdinaryCorpus(rows, machineByFunction)
	summary.ABIBoundaries = summarizeBackendABIBoundaries(rows)
	return summary
}

func summarizeBackendOrdinaryCorpus(rows []backendFunctionPathReport, machineByFunction map[string]machineBackendFunctionReport) backendOrdinaryCorpusSummary {
	summary := backendOrdinaryCorpusSummary{
		StackFallbackReasons: map[string]int{},
		EvidenceSource:       "benchmark-corpus-static-map/non-runtime-hotness",
	}
	for _, row := range rows {
		if !backendRowInOrdinaryCorpus(row) {
			continue
		}
		summary.FunctionCount++
		switch row.BackendPath {
		case "register":
			summary.RegisterPath++
			machineReport := machineByFunction[row.Function]
			if machineReport.Validation.StackChurnOps == 0 &&
				machineReport.Validation.MachineVerifier == "pass" &&
				machineReport.Validation.AllocationVerifier == "pass" {
				summary.RegisterNoStackChurn++
			} else {
				summary.RegisterWithStackChurn++
			}
		case "stack":
			summary.StackFallback++
			if row.Category != "" {
				summary.StackFallbackReasons[row.Category]++
			}
		}
	}
	summary.RegisterNoStackChurnMajority = summary.FunctionCount > 0 &&
		summary.RegisterNoStackChurn*2 > summary.FunctionCount
	return summary
}

func backendRowInOrdinaryCorpus(row backendFunctionPathReport) bool {
	if row.HotnessRank <= 0 {
		return false
	}
	if backendHotnessSourceIsRuntimeHeavy(row.HotnessSource) {
		return false
	}
	return row.Category != "unsupported_effect_runtime_call"
}

func backendHotnessSourceIsRuntimeHeavy(source string) bool {
	lower := strings.ToLower(source)
	return strings.Contains(lower, "compiler/internal/webrt/") ||
		strings.Contains(lower, "runtime")
}

func summarizeBackendABIBoundaries(rows []backendFunctionPathReport) backendABIBoundarySummary {
	summary := backendABIBoundarySummary{
		ValueClasses: map[string]int{},
	}
	for _, row := range rows {
		switch row.ABI.MultiSlotReturnPolicy {
		case "single_slot_register_return":
			summary.SingleSlotRegisterReturn++
		case "single_slot_stack_fallback":
			summary.SingleSlotStackFallback++
		case "unsupported_multi_slot_return_stack_fallback":
			summary.MultiSlotReturnStackFallback++
		case "unsupported_call_multi_slot_return_stack_fallback":
			summary.CallMultiSlotReturnStackFallback++
		}
		if row.ABI.ValueClass != "" {
			summary.ValueClasses[row.ABI.ValueClass]++
		}
	}
	return summary
}

type backendHotness struct {
	Rank   int
	Source string
}

func applyBackendHotness(row *backendFunctionPathReport) {
	hotness := backendHotnessForFunction(row.Function)
	row.HotnessRank = hotness.Rank
	row.HotnessSource = hotness.Source
}

func backendHotnessForFunction(name string) backendHotness {
	normalized := normalizeBackendHotnessName(name)
	if hotness, ok := backendBenchmarkHotnessByFunction[normalized]; ok {
		return hotness
	}
	return backendHotness{Rank: 0, Source: "not_in_benchmark_corpus"}
}

func normalizeBackendHotnessName(name string) string {
	out := strings.ToLower(strings.TrimSpace(name))
	out = strings.ReplaceAll(out, "-", "_")
	out = strings.ReplaceAll(out, ".", "_")
	out = strings.ReplaceAll(out, ":", "_")
	return out
}

var backendBenchmarkHotnessByFunction = map[string]backendHotness{
	"response_cost":      {Rank: 1, Source: "examples/benchmarks/techempower_plaintext_kernel.tetra"},
	"jsonmessagehandler": {Rank: 2, Source: "compiler/internal/webrt/techempower.go:/json"},
	"dbhandler":          {Rank: 3, Source: "compiler/internal/webrt/techempower.go:/db"},
	"querieshandler":     {Rank: 4, Source: "compiler/internal/webrt/techempower.go:/queries"},
	"updateshandler":     {Rank: 5, Source: "compiler/internal/webrt/techempower.go:/updates"},
	"fortuneshandler":    {Rank: 6, Source: "compiler/internal/webrt/techempower.go:/fortunes"},
	"queryworld":         {Rank: 7, Source: "compiler/internal/webrt/techempower.go:queryWorld"},
	"updateworld":        {Rank: 8, Source: "compiler/internal/webrt/techempower.go:updateWorld"},
	"fetchworld":         {Rank: 9, Source: "compiler/internal/webrt/techempower.go:fetchWorld"},
	"fetchfortunes":      {Rank: 10, Source: "compiler/internal/webrt/techempower.go:fetchFortunes"},
	"flip_count":         {Rank: 20, Source: "examples/benchmarks/clbg_fannkuch_redux.tetra"},
	"escape_iters":       {Rank: 21, Source: "examples/benchmarks/clbg_integer_mandelbrot.tetra"},
	"mix":                {Rank: 22, Source: "examples/benchmarks/energy_languages_checksum.tetra"},
	"transform":          {Rank: 23, Source: "examples/benchmarks/spec_cpu_branch_mix.tetra"},
	"safe_pair":          {Rank: 24, Source: "examples/benchmarks/plb2_nqueen.tetra"},
	"safe_6":             {Rank: 25, Source: "examples/benchmarks/plb2_nqueen.tetra"},
	"abs_i32":            {Rank: 26, Source: "examples/benchmarks/plb2_nqueen.tetra"},
	"cell":               {Rank: 27, Source: "examples/benchmarks/plb2_sudoku_checksum.tetra"},
	"branch":             {Rank: 28, Source: "examples/benchmarks/rustc_perf_frontend_mix.tetra"},
	"apply":              {Rank: 29, Source: "examples/benchmarks/awfy_closure_dispatch.tetra"},
	"f0":                 {Rank: 30, Source: "examples/benchmarks/pyperformance_call_mix.tetra"},
	"f1":                 {Rank: 31, Source: "examples/benchmarks/pyperformance_call_mix.tetra"},
	"f2":                 {Rank: 32, Source: "examples/benchmarks/pyperformance_call_mix.tetra"},
	"score":              {Rank: 33, Source: "examples/benchmarks/jvm_dacapo_object_kernel.tetra"},
}

func backendABIBoundaryForFunction(target string, fn ir.IRFunc, backendPath string) backendABIBoundaryReport {
	maxRegisterReturns := machineCallABIForTarget(target).MaxRetSlots
	hasMultiSlotCall := hasMultiSlotCallReturn(fn, maxRegisterReturns)
	policy := "single_slot_stack_fallback"
	switch {
	case hasMultiSlotCall:
		policy = "unsupported_call_multi_slot_return_stack_fallback"
	case fn.ReturnSlots > maxRegisterReturns:
		policy = "unsupported_multi_slot_return_stack_fallback"
	case backendPath == "register":
		policy = "single_slot_register_return"
	}
	return backendABIBoundaryReport{
		ReturnSlots:            fn.ReturnSlots,
		MaxRegisterReturnSlots: maxRegisterReturns,
		MultiSlotReturnPolicy:  policy,
		ValueClass:             backendABIValueClass(fn.ReturnSlots, maxRegisterReturns, hasMultiSlotCall),
		BoundaryStatus:         backendABIBoundaryStatus(policy),
	}
}

func backendABIValueClass(returnSlots int, maxRegisterReturns int, hasMultiSlotCall bool) string {
	switch {
	case hasMultiSlotCall:
		return "callee_multi_slot_return_unverified"
	case returnSlots == 0:
		return "void_or_no_return"
	case returnSlots <= maxRegisterReturns:
		return "single_register_slot"
	case returnSlots == 2:
		return "unverified_header_or_pair"
	default:
		return "unverified_aggregate"
	}
}

func backendABIBoundaryStatus(policy string) string {
	switch policy {
	case "single_slot_register_return":
		return "register_return_verified"
	case "unsupported_multi_slot_return_stack_fallback", "unsupported_call_multi_slot_return_stack_fallback":
		return "stack_fallback_until_multi_slot_abi_verified"
	default:
		return "stack_fallback_for_unpromoted_single_slot"
	}
}

func hasMultiSlotCallReturn(fn ir.IRFunc, maxRegisterReturns int) bool {
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall && instr.RetSlots > maxRegisterReturns {
			return true
		}
	}
	return false
}

func buildMachineBackendFunctionReports(target string, irProg *ir.IRProgram) []machineBackendFunctionReport {
	if irProg == nil {
		return nil
	}
	callABI := machineCallABIForTarget(target)
	callerSaved := machineCallerSavedForTarget(target)
	var out []machineBackendFunctionReport
	for _, fn := range irProg.Funcs {
		if !stackIRFunctionPassesSSAGate(fn) {
			continue
		}
		if mfn, ok, err := machine.ScalarI32SliceSumLoopFunctionFromStackIR(fn); err == nil && ok {
			if report, ok := buildMachineBackendFunctionReport(mfn, "machine-ir-slice-sum", callerSaved, true); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.ScalarIntCallLoopFunctionFromStackIRWithCallABI(fn, callABI); err == nil && ok {
			if report, ok := buildMachineBackendFunctionReport(mfn, "machine-ir-call-loop", callerSaved, true); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.ScalarIntLoopFunctionFromStackIR(fn); err == nil && ok {
			if report, ok := buildMachineBackendFunctionReport(mfn, "machine-ir-loop", callerSaved, true); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.ScalarIntFunctionFromStackIRWithCallABI(fn, callABI); err == nil && ok {
			path := "machine-ir-scalar"
			if machineFunctionHasOp(mfn, machine.OpCall) {
				path = "machine-ir-call"
			}
			if report, ok := buildMachineBackendFunctionReport(mfn, path, callerSaved, true); ok {
				out = append(out, report)
			}
		}
	}
	return out
}

func stackIRFunctionPassesSSAGate(fn ir.IRFunc) bool {
	ssaFn, ok, err := ssair.FromStackIRFunction(fn)
	if err != nil || !ok {
		return false
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func buildMachineBackendFunctionReport(fn machine.Function, path string, callerSaved []machine.PhysReg, ssaVerified bool) (machineBackendFunctionReport, bool) {
	live, err := machine.AnalyzeLiveness(fn)
	if err != nil {
		return machineBackendFunctionReport{}, false
	}
	intervals, err := machine.BuildIntervals(fn)
	if err != nil {
		return machineBackendFunctionReport{}, false
	}
	alloc, err := machine.LinearScan(intervals, callerSaved)
	if err != nil {
		return machineBackendFunctionReport{}, false
	}
	spillSlots := len(alloc.Spills)
	if alloc.Assignments == nil {
		alloc.Assignments = map[machine.VReg]machine.PhysReg{}
	}
	if alloc.Spills == nil {
		alloc.Spills = map[machine.VReg]int{}
	}
	if err := machine.VerifyAllocation(fn, alloc, callerSaved, spillSlots); err != nil {
		return machineBackendFunctionReport{}, false
	}
	validation := machineValidationReport{
		MachineVerifier:    "pass",
		AllocationVerifier: "pass",
		SpillReload:        machineSpillReloadValidationStatus(fn, spillSlots),
		CallClobbers:       machineCallClobberValidationStatus(fn),
		StackChurnOps:      machineStackChurnOps(fn),
	}
	return machineBackendFunctionReport{
		Function:             fn.Name,
		Path:                 path,
		SSAPath:              "value-ssa-v1",
		SSAVerified:          ssaVerified,
		InstructionSelection: machineInstructionSelection(fn),
		Validation:           validation,
		Dump:                 machine.FormatFunction(fn),
		Liveness:             live,
		Intervals:            intervals,
		Allocation: machineAllocationReport{
			Assignments: alloc.Assignments,
			Spills:      alloc.Spills,
		},
		SpillSlots: spillSlots,
	}, true
}

func machineInstructionSelection(fn machine.Function) []string {
	seen := map[string]bool{}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if instr.Op != "" {
				seen[string(instr.Op)] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for op := range seen {
		out = append(out, op)
	}
	sort.Strings(out)
	return out
}

func machineSpillReloadValidationStatus(fn machine.Function, spillSlots int) string {
	hasSpillReloadOps := false
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if instr.Op == machine.OpSpill || instr.Op == machine.OpReload {
				hasSpillReloadOps = true
			}
		}
	}
	switch {
	case hasSpillReloadOps:
		return "validated_spill_reload_ops"
	case spillSlots > 0:
		return "validated_spills_reported"
	default:
		return "validated_no_spills"
	}
}

func machineCallClobberValidationStatus(fn machine.Function) string {
	hasCall := false
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if instr.Op != machine.OpCall {
				continue
			}
			hasCall = true
			if instr.ABI == "" || len(instr.Clobbers) == 0 {
				return "missing_call_clobber_metadata"
			}
		}
	}
	if !hasCall {
		return "not_applicable"
	}
	return "validated"
}

func machineStackChurnOps(fn machine.Function) int {
	count := 0
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if instr.Op == machine.OpPush || instr.Op == machine.OpPop {
				count++
			}
		}
	}
	return count
}

func machineFunctionHasOp(fn machine.Function, op machine.Opcode) bool {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if instr.Op == op {
				return true
			}
		}
	}
	return false
}

func machineCallABIForTarget(target string) machine.CallABIInfo {
	if target == "windows-x64" {
		return machine.Win64CallABIInfo()
	}
	return machine.SysVCallABIInfo()
}

func machineCallerSavedForTarget(target string) []machine.PhysReg {
	if target == "windows-x64" {
		return machine.Win64CallerSaved()
	}
	return machine.LinuxX64CallerSaved()
}

func targetSupportsMachineScalar(target string) bool {
	switch target {
	case "linux-x64", "macos-x64", "windows-x64":
		return true
	default:
		return false
	}
}

func allocationPlanOptionsForTarget(target string) allocplan.Options {
	return allocplan.Options{
		EnableStackLowering:    targetSupportsStackAllocationLowering(target),
		EnableSmallHeapRuntime: target == "linux-x64",
		EnableRegionPlanning:   target == "linux-x64",
		EnableRegionLowering:   target == "linux-x64",
	}
}

func lowerOptionsForTarget(target string) lower.Options {
	return lower.Options{
		StackAllocationLowering:    targetSupportsStackAllocationLowering(target),
		FunctionTempRegionLowering: target == "linux-x64",
	}
}

func targetSupportsStackAllocationLowering(target string) bool {
	switch target {
	case "linux-x64", "macos-x64", "windows-x64":
		return true
	default:
		return false
	}
}

func buildProofReport(plirProg *plir.Program, bounds boundsReport, target string) proofReport {
	return proofReport{
		reportEnvelope: reportEnvelope{SchemaVersion: 1, Kind: "proof", Target: target},
		Bounds:         bounds,
		Proofs:         buildProofEvidence(plirProg, bounds),
		PLIR:           plirProg,
	}
}

func buildProofEvidence(prog *plir.Program, bounds boundsReport) []proofEvidence {
	if prog == nil {
		return nil
	}
	removed := map[string]bool{}
	for _, fn := range bounds.Functions {
		for _, site := range fn.Sites {
			if site.Removed && site.ProofID != "" {
				removed[site.ProofID] = true
			}
		}
	}
	var out []proofEvidence
	for _, fn := range prog.Funcs {
		facts := proofFactsByID(fn)
		for _, guard := range fn.ProofGuards {
			uses := guard.Dominates
			if len(uses) == 0 {
				for _, use := range fn.ProofUses {
					if use.ProofID == guard.ID {
						uses = append(uses, use)
					}
				}
			}
			if len(uses) == 0 {
				out = append(out, proofEvidence{
					ProofID:            guard.ID,
					Kind:               guard.Kind,
					Guard:              guard.Condition,
					Fact:               facts[guard.ID],
					Reason:             guard.Reason,
					RemovedBoundsCheck: removed[guard.ID],
				})
				continue
			}
			for _, use := range uses {
				out = append(out, proofEvidence{
					ProofID:            guard.ID,
					Kind:               guard.Kind,
					Guard:              guard.Condition,
					Dominates:          use.UseKind + " " + use.OpID,
					Fact:               facts[guard.ID],
					Reason:             guard.Reason,
					RemovedBoundsCheck: removed[guard.ID],
				})
			}
		}
	}
	return out
}

func proofFactsByID(fn plir.Function) map[string]string {
	out := map[string]string{}
	for _, fact := range fn.Facts {
		if fact.ProofID == "" || fact.Kind != plir.FactIndexInRange {
			continue
		}
		out[fact.ProofID] = fact.ValueID
		if fact.Range != "" {
			out[fact.ProofID] = fmt.Sprintf("%s in [%s]", fact.ValueID, fact.Range)
		}
	}
	for _, fact := range fn.RangeFacts {
		if fact.ProofID == "" {
			continue
		}
		out[fact.ProofID] = formatRangeEvidence(fact)
	}
	return out
}

func formatRangeEvidence(fact plir.RangeFact) string {
	lower := formatReportBound(fact.Lower)
	upper := formatReportBound(fact.Upper)
	open := "["
	close := ")"
	if !fact.InclusiveLower {
		open = "("
	}
	if fact.InclusiveUpper {
		close = "]"
	}
	name := fact.Value
	if strings.HasPrefix(name, "local:") {
		name = strings.TrimPrefix(name, "local:")
	}
	if strings.HasPrefix(name, "loop_index:") {
		name = strings.TrimPrefix(name, "loop_index:")
	}
	text := fmt.Sprintf("%s in %s%s, %s%s", name, open, lower, upper, close)
	if len(fact.Derivation) > 0 {
		text += "; derivation: " + strings.Join(fact.Derivation, ", ")
	}
	return text
}

func formatReportBound(bound plir.Bound) string {
	switch bound.Kind {
	case plir.BoundConst:
		return fmt.Sprintf("%d", bound.Const)
	case plir.BoundSymbol:
		return bound.Symbol
	case plir.BoundSymbolMinus:
		return fmt.Sprintf("%s - %d", bound.Symbol, bound.Const)
	default:
		return string(bound.Kind)
	}
}

func wrapAllocationPlanReport(plan *allocplan.Plan, target string) allocationPlanReport {
	claimLevel, evidenceScope, _ := allocationPlanTargetStorageScope(target)
	if plan == nil {
		return allocationPlanReport{
			reportEnvelope:         reportEnvelope{SchemaVersion: 2, Kind: "allocation_plan", Target: target},
			TargetMemoryClaimLevel: claimLevel,
			StorageEvidenceScope:   evidenceScope,
			Summary:                allocplan.Summarize(nil),
		}
	}
	return allocationPlanReport{
		reportEnvelope:         reportEnvelope{SchemaVersion: 2, Kind: "allocation_plan", Target: target},
		TargetMemoryClaimLevel: claimLevel,
		StorageEvidenceScope:   evidenceScope,
		Summary:                allocplan.Summarize(plan),
		Totals:                 plan.Totals,
		Functions:              plan.Functions,
	}
}

func validateAllocationPlanReport(plan *allocplan.Plan, report allocationPlanReport) error {
	if report.SchemaVersion != 2 || report.Kind != "allocation_plan" {
		return fmt.Errorf("allocation report mismatch: invalid envelope schema=%d kind=%q", report.SchemaVersion, report.Kind)
	}
	expectedClaimLevel, expectedEvidenceScope, err := allocationPlanTargetStorageScope(report.Target)
	if err != nil {
		return fmt.Errorf("allocation report mismatch: target memory scope: %w", err)
	}
	if report.TargetMemoryClaimLevel != expectedClaimLevel {
		return fmt.Errorf("allocation report mismatch: target_memory_claim_level=%q want %q", report.TargetMemoryClaimLevel, expectedClaimLevel)
	}
	if report.StorageEvidenceScope != expectedEvidenceScope {
		return fmt.Errorf("allocation report mismatch: storage_evidence_scope=%q want %q", report.StorageEvidenceScope, expectedEvidenceScope)
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

func allocationPlanTargetStorageScope(triple string) (string, string, error) {
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

func buildBoundsReport(prog *ir.IRProgram, checked *semantics.CheckedProgram, target string) boundsReport {
	report := boundsReport{
		reportEnvelope: reportEnvelope{SchemaVersion: 1, Kind: "bounds", Target: target},
	}
	if prog == nil {
		return report
	}
	leftReasons := buildBoundsLeftReasonIndex(checked)
	removedReasons := buildBoundsRemovedReasonIndex(checked)
	for _, fn := range prog.Funcs {
		row := boundsFunctionRow{Function: fn.Name}
		for _, instr := range fn.Instrs {
			switch {
			case isUncheckedIndexLoad(instr.Kind):
				row.Removed++
				report.Totals.Removed++
				row.Sites = append(row.Sites, boundsCheckSite{
					Site:    reportPos(instr.Pos),
					Kind:    irIndexKind(instr.Kind),
					Removed: true,
					ProofID: instr.ProofID,
					Reason:  removedBoundsReasonForSite(fn.Name, instr.Pos, instr.ProofID, removedReasons),
				})
			case isCheckedIndexAccess(instr.Kind):
				row.Left++
				report.Totals.Left++
				row.Sites = append(row.Sites, boundsCheckSite{
					Site:    reportPos(instr.Pos),
					Kind:    irIndexKind(instr.Kind),
					Removed: false,
					Reason:  leftBoundsReason(fn.Name, instr.Pos, leftReasons),
				})
			}
		}
		if row.Removed > 0 || row.Left > 0 {
			report.Functions = append(report.Functions, row)
		}
	}
	return report
}

type boundsLeftReasonKey struct {
	Function string
	File     string
	Line     int
	Col      int
}

type boundsBranchGuard struct {
	Index string
	Base  string
}

type boundsLeftReasonContext struct {
	seenBranchGuards        []boundsBranchGuard
	missingLowerBoundGuards []boundsBranchGuard
	activeProofGuards       []boundsBranchGuard
	mutationInvalidated     []boundsBranchGuard
}

type boundsLeftReasonBuilder struct {
	function string
	funcs    map[string]semantics.FuncSig
	locals   map[string]semantics.LocalInfo
	globals  map[string]semantics.GlobalInfo
	reasons  map[boundsLeftReasonKey]string
}

func buildBoundsLeftReasonIndex(checked *semantics.CheckedProgram) map[boundsLeftReasonKey]string {
	reasons := map[boundsLeftReasonKey]string{}
	if checked == nil {
		return reasons
	}
	for _, fn := range checked.Funcs {
		if fn.Decl == nil {
			continue
		}
		builder := boundsLeftReasonBuilder{
			function: fn.Name,
			funcs:    checked.FuncSigs,
			locals:   fn.Locals,
			globals:  checked.GlobalsByModule[fn.Module],
			reasons:  reasons,
		}
		builder.walkBoundsReasonStmts(fn.Decl.Body, boundsLeftReasonContext{})
	}
	return reasons
}

func leftBoundsReason(function string, pos frontend.Position, reasons map[boundsLeftReasonKey]string) string {
	if reason := reasons[boundsLeftReasonKeyFor(function, pos)]; reason != "" {
		return reason
	}
	return "left_missing_dominance"
}

func removedBoundsReasonForSite(function string, pos frontend.Position, proofID string, reasons map[boundsLeftReasonKey]string) string {
	if reason := reasons[boundsLeftReasonKeyFor(function, pos)]; reason != "" {
		return reason
	}
	return removedBoundsReason(proofID)
}

func boundsLeftReasonKeyFor(function string, pos frontend.Position) boundsLeftReasonKey {
	return boundsLeftReasonKey{Function: function, File: pos.File, Line: pos.Line, Col: pos.Col}
}

func (b *boundsLeftReasonBuilder) walkBoundsReasonStmts(stmts []frontend.Stmt, ctx boundsLeftReasonContext) boundsLeftReasonContext {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			b.walkBoundsReasonExpr(s.Value, ctx, frontend.Position{})
			ctx = b.markCallMutationsInExpr(ctx, s.Value)
		case *frontend.ReturnStmt:
			b.walkBoundsReasonExpr(s.Value, ctx, frontend.Position{})
		case *frontend.ThrowStmt:
			b.walkBoundsReasonExpr(s.Value, ctx, frontend.Position{})
		case *frontend.DeferStmt:
			b.walkBoundsReasonStmts(s.Body, ctx)
		case *frontend.LetStmt:
			b.walkBoundsReasonExpr(s.Value, ctx, frontend.Position{})
			ctx = b.markCallMutationsInExpr(ctx, s.Value)
		case *frontend.AssignStmt:
			if idx, ok := s.Target.(*frontend.IndexExpr); ok && idx != nil {
				b.markBoundsIndexReason(idx, ctx, s.At)
				b.walkBoundsReasonExpr(idx.Base, ctx, frontend.Position{})
				b.walkBoundsReasonExpr(idx.Index, ctx, frontend.Position{})
			} else {
				b.walkBoundsReasonExpr(s.Target, ctx, frontend.Position{})
			}
			b.walkBoundsReasonExpr(s.Value, ctx, frontend.Position{})
			b.walkBoundsReasonExpr(s.CompoundValue, ctx, frontend.Position{})
			ctx = b.markCallMutationsInExpr(ctx, s.Value)
			ctx = b.markCallMutationsInExpr(ctx, s.CompoundValue)
			if id, ok := s.Target.(*frontend.IdentExpr); ok && id != nil {
				ctx = b.markMutationInvalidated(ctx, id.Name)
			}
		case *frontend.IfStmt:
			b.walkBoundsReasonExpr(s.Cond, ctx, frontend.Position{})
			if guard, ok := reportMissingLowerBranchGuard(s.Cond); ok {
				thenCtx := ctx
				thenCtx.missingLowerBoundGuards = appendBoundsBranchGuard(ctx.missingLowerBoundGuards, guard)
				b.walkBoundsReasonStmts(s.Then, thenCtx)
				b.walkBoundsReasonStmts(s.Else, ctx)
				continue
			}
			if guard, ok := reportFullBranchGuard(s.Cond); ok {
				b.walkBoundsReasonStmts(s.Then, ctx)
				b.walkBoundsReasonStmts(s.Else, ctx)
				ctx.seenBranchGuards = append(ctx.seenBranchGuards, guard)
				continue
			}
			b.walkBoundsReasonStmts(s.Then, ctx)
			b.walkBoundsReasonStmts(s.Else, ctx)
		case *frontend.IfLetStmt:
			b.walkBoundsReasonExpr(s.Pattern, ctx, frontend.Position{})
			b.walkBoundsReasonExpr(s.Value, ctx, frontend.Position{})
			b.walkBoundsReasonStmts(s.Then, ctx)
			b.walkBoundsReasonStmts(s.Else, ctx)
		case *frontend.WhileStmt:
			b.walkBoundsReasonExpr(s.Cond, ctx, frontend.Position{})
			bodyCtx := ctx
			if guard, ok := reportUpperBranchGuard(s.Cond); ok {
				bodyCtx.activeProofGuards = appendBoundsBranchGuard(bodyCtx.activeProofGuards, guard)
			}
			b.walkBoundsReasonStmts(s.Body, bodyCtx)
		case *frontend.ForRangeStmt:
			b.walkBoundsReasonExpr(s.Start, ctx, frontend.Position{})
			b.walkBoundsReasonExpr(s.End, ctx, frontend.Position{})
			b.walkBoundsReasonExpr(s.Iterable, ctx, frontend.Position{})
			b.walkBoundsReasonStmts(s.Body, ctx)
		case *frontend.MatchStmt:
			b.walkBoundsReasonExpr(s.Value, ctx, frontend.Position{})
			for _, c := range s.Cases {
				b.walkBoundsReasonExpr(c.Pattern, ctx, frontend.Position{})
				b.walkBoundsReasonExpr(c.Guard, ctx, frontend.Position{})
				b.walkBoundsReasonStmts(c.Body, ctx)
			}
		case *frontend.FreeStmt:
			b.walkBoundsReasonExpr(s.Value, ctx, frontend.Position{})
		case *frontend.UnsafeStmt:
			b.walkBoundsReasonStmts(s.Body, ctx)
		case *frontend.IslandStmt:
			b.walkBoundsReasonExpr(s.Size, ctx, frontend.Position{})
			b.walkBoundsReasonStmts(s.Body, ctx)
		case *frontend.ExprStmt:
			b.walkBoundsReasonExpr(s.Expr, ctx, frontend.Position{})
			ctx = b.markCallMutationsInExpr(ctx, s.Expr)
		case *frontend.ExpectStmt:
			b.walkBoundsReasonExpr(s.Cond, ctx, frontend.Position{})
		}
	}
	return ctx
}

func (b *boundsLeftReasonBuilder) walkBoundsReasonExpr(expr frontend.Expr, ctx boundsLeftReasonContext, siteOverride frontend.Position) {
	switch e := expr.(type) {
	case *frontend.BinaryExpr:
		b.walkBoundsReasonExpr(e.Left, ctx, frontend.Position{})
		b.walkBoundsReasonExpr(e.Right, ctx, frontend.Position{})
	case *frontend.UnaryExpr:
		b.walkBoundsReasonExpr(e.X, ctx, frontend.Position{})
	case *frontend.TryExpr:
		b.walkBoundsReasonExpr(e.X, ctx, frontend.Position{})
	case *frontend.AwaitExpr:
		b.walkBoundsReasonExpr(e.X, ctx, frontend.Position{})
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			b.walkBoundsReasonExpr(arg, ctx, frontend.Position{})
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			b.walkBoundsReasonExpr(field.Value, ctx, frontend.Position{})
		}
	case *frontend.FieldAccessExpr:
		b.walkBoundsReasonExpr(e.Base, ctx, frontend.Position{})
	case *frontend.IndexExpr:
		b.markBoundsIndexReason(e, ctx, siteOverride)
		b.walkBoundsReasonExpr(e.Base, ctx, frontend.Position{})
		b.walkBoundsReasonExpr(e.Index, ctx, frontend.Position{})
	case *frontend.MatchExpr:
		b.walkBoundsReasonExpr(e.Value, ctx, frontend.Position{})
		for _, c := range e.Cases {
			b.walkBoundsReasonExpr(c.Pattern, ctx, frontend.Position{})
			b.walkBoundsReasonExpr(c.Guard, ctx, frontend.Position{})
			b.walkBoundsReasonExpr(c.Value, ctx, frontend.Position{})
		}
	case *frontend.CatchExpr:
		b.walkBoundsReasonExpr(e.Call, ctx, frontend.Position{})
		for _, c := range e.Cases {
			b.walkBoundsReasonExpr(c.Pattern, ctx, frontend.Position{})
			b.walkBoundsReasonExpr(c.Guard, ctx, frontend.Position{})
			b.walkBoundsReasonExpr(c.Value, ctx, frontend.Position{})
		}
	}
}

func (b *boundsLeftReasonBuilder) markBoundsIndexReason(index *frontend.IndexExpr, ctx boundsLeftReasonContext, siteOverride frontend.Position) {
	if index == nil {
		return
	}
	guard := boundsBranchGuard{Base: reportExprPath(index.Base), Index: reportExprPath(index.Index)}
	if guard.Base == "" || guard.Index == "" {
		return
	}
	reason := ""
	if boundsGuardListContains(ctx.mutationInvalidated, guard) {
		reason = "left_proof_invalidated_by_mutation"
	} else if boundsGuardListContains(ctx.missingLowerBoundGuards, guard) {
		reason = "left_missing_non_negative_lower_bound"
	} else if boundsGuardListContains(ctx.seenBranchGuards, guard) {
		reason = "left_guard_not_dominating"
	}
	if reason == "" {
		return
	}
	pos := index.At
	if siteOverride.Line != 0 || siteOverride.Col != 0 || siteOverride.File != "" {
		pos = siteOverride
	}
	b.setBoundsLeftReason(pos, reason)
}

func (b *boundsLeftReasonBuilder) markMutationInvalidated(ctx boundsLeftReasonContext, name string) boundsLeftReasonContext {
	if name == "" {
		return ctx
	}
	for _, guard := range ctx.activeProofGuards {
		if reportProofPathMatchesMutation(guard.Index, name) || reportProofPathMatchesMutation(guard.Base, name) {
			ctx.mutationInvalidated = appendBoundsBranchGuard(ctx.mutationInvalidated, guard)
		}
	}
	return ctx
}

func (b *boundsLeftReasonBuilder) markCallMutationsInExpr(ctx boundsLeftReasonContext, expr frontend.Expr) boundsLeftReasonContext {
	switch e := expr.(type) {
	case *frontend.BinaryExpr:
		ctx = b.markCallMutationsInExpr(ctx, e.Left)
		ctx = b.markCallMutationsInExpr(ctx, e.Right)
	case *frontend.UnaryExpr:
		ctx = b.markCallMutationsInExpr(ctx, e.X)
	case *frontend.TryExpr:
		ctx = b.markCallMutationsInExpr(ctx, e.X)
	case *frontend.AwaitExpr:
		ctx = b.markCallMutationsInExpr(ctx, e.X)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			ctx = b.markCallMutationsInExpr(ctx, arg)
		}
		ctx = b.markCallMutationInvalidated(ctx, e)
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			ctx = b.markCallMutationsInExpr(ctx, field.Value)
		}
	case *frontend.FieldAccessExpr:
		ctx = b.markCallMutationsInExpr(ctx, e.Base)
	case *frontend.IndexExpr:
		ctx = b.markCallMutationsInExpr(ctx, e.Base)
		ctx = b.markCallMutationsInExpr(ctx, e.Index)
	case *frontend.MatchExpr:
		ctx = b.markCallMutationsInExpr(ctx, e.Value)
		for _, c := range e.Cases {
			ctx = b.markCallMutationsInExpr(ctx, c.Pattern)
			ctx = b.markCallMutationsInExpr(ctx, c.Guard)
			ctx = b.markCallMutationsInExpr(ctx, c.Value)
		}
	case *frontend.CatchExpr:
		ctx = b.markCallMutationsInExpr(ctx, e.Call)
		for _, c := range e.Cases {
			ctx = b.markCallMutationsInExpr(ctx, c.Pattern)
			ctx = b.markCallMutationsInExpr(ctx, c.Guard)
			ctx = b.markCallMutationsInExpr(ctx, c.Value)
		}
	}
	return ctx
}

func (b *boundsLeftReasonBuilder) markCallMutationInvalidated(ctx boundsLeftReasonContext, call *frontend.CallExpr) boundsLeftReasonContext {
	if call == nil {
		return ctx
	}
	ownership := b.callParamOwnership(call.Name)
	for i, owner := range ownership {
		if owner != "inout" {
			continue
		}
		if i >= len(call.Args) {
			break
		}
		path := reportExprPath(call.Args[i])
		if path == "" {
			continue
		}
		ctx = b.markMutationInvalidated(ctx, path)
	}
	return ctx
}

func (b *boundsLeftReasonBuilder) callParamOwnership(name string) []string {
	if name == "" {
		return nil
	}
	if target, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = target
	}
	if b.funcs != nil {
		if sig, ok := b.funcs[name]; ok {
			return sig.ParamOwnership
		}
	}
	if local, ok := b.locals[name]; ok && local.FunctionTypeValue {
		return local.FunctionParamOwnership
	}
	if b.globals != nil {
		if global, ok := b.globals[name]; ok && global.FunctionTypeValue {
			return global.FunctionParamOwnership
		}
	}
	return nil
}

func reportProofPathMatchesMutation(proofPath string, mutatedPath string) bool {
	if proofPath == "" || mutatedPath == "" {
		return false
	}
	return proofPath == mutatedPath || strings.HasPrefix(proofPath, mutatedPath+".")
}

func (b *boundsLeftReasonBuilder) setBoundsLeftReason(pos frontend.Position, reason string) {
	if pos.Line == 0 && pos.Col == 0 && pos.File == "" {
		return
	}
	key := boundsLeftReasonKeyFor(b.function, pos)
	if existing := b.reasons[key]; existing == "left_missing_non_negative_lower_bound" {
		return
	}
	b.reasons[key] = reason
}

func boundsGuardListContains(guards []boundsBranchGuard, want boundsBranchGuard) bool {
	for _, guard := range guards {
		if guard == want {
			return true
		}
	}
	return false
}

func appendBoundsBranchGuard(guards []boundsBranchGuard, guard boundsBranchGuard) []boundsBranchGuard {
	out := make([]boundsBranchGuard, 0, len(guards)+1)
	out = append(out, guards...)
	out = append(out, guard)
	return out
}

func reportMissingLowerBranchGuard(cond frontend.Expr) (boundsBranchGuard, bool) {
	if _, ok := reportFullBranchGuard(cond); ok {
		return boundsBranchGuard{}, false
	}
	return reportUpperBranchGuard(cond)
}

func reportFullBranchGuard(cond frontend.Expr) (boundsBranchGuard, bool) {
	bin, ok := cond.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenAmpAmp {
		return boundsBranchGuard{}, false
	}
	if guard, ok := reportFullBranchGuardParts(bin.Left, bin.Right); ok {
		return guard, true
	}
	return reportFullBranchGuardParts(bin.Right, bin.Left)
}

func reportFullBranchGuardParts(lower frontend.Expr, upper frontend.Expr) (boundsBranchGuard, bool) {
	lowerIndex, ok := reportNonNegativeGuardIndex(lower)
	if !ok {
		return boundsBranchGuard{}, false
	}
	upperGuard, ok := reportUpperBranchGuard(upper)
	if !ok || upperGuard.Index != lowerIndex {
		return boundsBranchGuard{}, false
	}
	return upperGuard, true
}

func reportUpperBranchGuard(cond frontend.Expr) (boundsBranchGuard, bool) {
	bin, ok := cond.(*frontend.BinaryExpr)
	if !ok || bin == nil {
		return boundsBranchGuard{}, false
	}
	left, ok := bin.Left.(*frontend.IdentExpr)
	if !ok || left == nil {
		return boundsBranchGuard{}, false
	}
	var base string
	switch bin.Op {
	case frontend.TokenLess:
		base = reportLenFieldBaseName(bin.Right)
	case frontend.TokenLessEq:
		base = reportLenMinusOneBaseName(bin.Right)
	}
	if base == "" {
		return boundsBranchGuard{}, false
	}
	return boundsBranchGuard{Index: left.Name, Base: base}, true
}

func reportNonNegativeGuardIndex(expr frontend.Expr) (string, bool) {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil {
		return "", false
	}
	if left, ok := bin.Left.(*frontend.IdentExpr); ok && left != nil && bin.Op == frontend.TokenGreaterEq && reportIsZeroNumber(bin.Right) {
		return left.Name, true
	}
	if right, ok := bin.Right.(*frontend.IdentExpr); ok && right != nil && bin.Op == frontend.TokenLessEq && reportIsZeroNumber(bin.Left) {
		return right.Name, true
	}
	return "", false
}

func reportLenFieldBaseName(expr frontend.Expr) string {
	field, ok := expr.(*frontend.FieldAccessExpr)
	if !ok || field == nil || field.Field != "len" {
		return ""
	}
	return reportExprPath(field.Base)
}

func reportLenMinusOneBaseName(expr frontend.Expr) string {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenMinus {
		return ""
	}
	right, ok := bin.Right.(*frontend.NumberExpr)
	if !ok || right == nil || right.Value != 1 {
		return ""
	}
	return reportLenFieldBaseName(bin.Left)
}

func reportIsZeroNumber(expr frontend.Expr) bool {
	num, ok := expr.(*frontend.NumberExpr)
	return ok && num != nil && num.Value == 0
}

type boundsReportViewInfo struct {
	isView   bool
	composed bool
	unsafe   bool
}

type boundsRemovedReasonBuilder struct {
	function string
	reasons  map[boundsLeftReasonKey]string
	locals   map[string]boundsReportViewInfo
}

func buildBoundsRemovedReasonIndex(checked *semantics.CheckedProgram) map[boundsLeftReasonKey]string {
	reasons := map[boundsLeftReasonKey]string{}
	if checked == nil {
		return reasons
	}
	for _, fn := range checked.Funcs {
		if fn.Decl == nil {
			continue
		}
		builder := boundsRemovedReasonBuilder{
			function: fn.Name,
			reasons:  reasons,
			locals:   map[string]boundsReportViewInfo{},
		}
		builder.walkRemovedReasonStmts(fn.Decl.Body)
	}
	return reasons
}

func (b *boundsRemovedReasonBuilder) walkRemovedReasonStmts(stmts []frontend.Stmt) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			b.rememberViewLocal(s.Name, s.Value)
		case *frontend.AssignStmt:
			if id, ok := s.Target.(*frontend.IdentExpr); ok && id != nil {
				b.rememberViewLocal(id.Name, s.Value)
			}
		case *frontend.IfStmt:
			thenLocals := cloneBoundsReportViewInfoMap(b.locals)
			elseLocals := cloneBoundsReportViewInfoMap(b.locals)
			thenBuilder := boundsRemovedReasonBuilder{function: b.function, reasons: b.reasons, locals: thenLocals}
			elseBuilder := boundsRemovedReasonBuilder{function: b.function, reasons: b.reasons, locals: elseLocals}
			thenBuilder.walkRemovedReasonStmts(s.Then)
			elseBuilder.walkRemovedReasonStmts(s.Else)
			b.locals = mergeBoundsReportViewInfoMaps(thenBuilder.locals, elseBuilder.locals)
		case *frontend.IfLetStmt:
			b.walkRemovedReasonStmts(s.Then)
			b.walkRemovedReasonStmts(s.Else)
		case *frontend.WhileStmt:
			b.walkRemovedReasonStmts(s.Body)
		case *frontend.ForRangeStmt:
			if info := b.viewChainInfo(s.Iterable); info.composed && !info.unsafe {
				b.setRemovedReason(s.At, "removed_by_view_chain")
			}
			b.walkRemovedReasonStmts(s.Body)
		case *frontend.MatchStmt:
			for _, c := range s.Cases {
				b.walkRemovedReasonStmts(c.Body)
			}
		case *frontend.DeferStmt:
			b.walkRemovedReasonStmts(s.Body)
		case *frontend.UnsafeStmt:
			b.walkRemovedReasonStmts(s.Body)
		case *frontend.IslandStmt:
			b.walkRemovedReasonStmts(s.Body)
		}
	}
}

func (b *boundsRemovedReasonBuilder) rememberViewLocal(name string, expr frontend.Expr) {
	if name == "" {
		return
	}
	info := b.viewChainInfo(expr)
	if !info.isView && !info.unsafe {
		delete(b.locals, name)
		return
	}
	b.locals[name] = info
}

func (b *boundsRemovedReasonBuilder) viewChainInfo(expr frontend.Expr) boundsReportViewInfo {
	switch e := expr.(type) {
	case nil:
		return boundsReportViewInfo{}
	case *frontend.IdentExpr:
		if e == nil {
			return boundsReportViewInfo{}
		}
		return b.locals[e.Name]
	case *frontend.CallExpr:
		if e == nil {
			return boundsReportViewInfo{}
		}
		name := reportResolvedBuiltinName(e.Name)
		if reportRawSliceBuiltinName(name) {
			return boundsReportViewInfo{isView: true, unsafe: true}
		}
		if reportCopyResultBuiltinName(name) {
			return boundsReportViewInfo{}
		}
		if reportBorrowBuiltinName(name) {
			source := b.viewChainInfo(reportCallArg(e, 0))
			return boundsReportViewInfo{isView: true, composed: source.composed, unsafe: source.unsafe}
		}
		if reportViewBuiltinName(name) {
			source := b.viewChainInfo(reportCallArg(e, 0))
			return boundsReportViewInfo{
				isView:   true,
				composed: source.isView || source.composed,
				unsafe:   source.unsafe || reportStaticInvalidStringViewCall(name, e),
			}
		}
	}
	return boundsReportViewInfo{}
}

func (b *boundsRemovedReasonBuilder) setRemovedReason(pos frontend.Position, reason string) {
	if pos.Line == 0 && pos.Col == 0 && pos.File == "" {
		return
	}
	b.reasons[boundsLeftReasonKeyFor(b.function, pos)] = reason
}

func cloneBoundsReportViewInfoMap(in map[string]boundsReportViewInfo) map[string]boundsReportViewInfo {
	out := make(map[string]boundsReportViewInfo, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func mergeBoundsReportViewInfoMaps(left, right map[string]boundsReportViewInfo) map[string]boundsReportViewInfo {
	out := map[string]boundsReportViewInfo{}
	keys := map[string]bool{}
	for key := range left {
		keys[key] = true
	}
	for key := range right {
		keys[key] = true
	}
	for key := range keys {
		l, lok := left[key]
		r, rok := right[key]
		if !lok || !rok {
			continue
		}
		info := boundsReportViewInfo{
			isView:   l.isView && r.isView,
			composed: l.composed && r.composed,
			unsafe:   l.unsafe || r.unsafe,
		}
		if info.isView || info.unsafe {
			out[key] = info
		}
	}
	return out
}

func reportResolvedBuiltinName(name string) string {
	if target, ok := semantics.ResolveBuiltinAlias(name); ok {
		return target
	}
	return name
}

func reportRawSliceBuiltinName(name string) bool {
	switch name {
	case "core.raw_slice_u8_from_parts", "core.raw_slice_u16_from_parts", "core.raw_slice_i32_from_parts", "core.raw_slice_bool_from_parts":
		return true
	default:
		return false
	}
}

func reportCopyResultBuiltinName(name string) bool {
	if name == "core.string_copy" {
		return true
	}
	if !strings.HasPrefix(name, "core.slice_copy_") || strings.HasPrefix(name, "core.slice_copy_into_") {
		return false
	}
	switch strings.TrimPrefix(name, "core.slice_copy_") {
	case "u8", "u16", "i32", "bool":
		return true
	default:
		return false
	}
}

func reportBorrowBuiltinName(name string) bool {
	if name == "core.string_borrow" {
		return true
	}
	if !strings.HasPrefix(name, "core.slice_borrow_") {
		return false
	}
	switch strings.TrimPrefix(name, "core.slice_borrow_") {
	case "u8", "u16", "i32", "bool":
		return true
	default:
		return false
	}
}

func reportViewBuiltinName(name string) bool {
	if name == "core.string_window" || name == "core.string_prefix" || name == "core.string_suffix" {
		return true
	}
	for _, prefix := range []string{"core.slice_window_", "core.slice_prefix_", "core.slice_suffix_"} {
		if strings.HasPrefix(name, prefix) {
			switch strings.TrimPrefix(name, prefix) {
			case "u8", "u16", "i32", "bool":
				return true
			}
		}
	}
	return false
}

func reportStaticInvalidStringViewCall(name string, call *frontend.CallExpr) bool {
	if call == nil || !strings.HasPrefix(name, "core.string_") {
		return false
	}
	sourceLen, knownLen := reportStaticStringByteLen(reportCallArg(call, 0))
	if !knownLen {
		return false
	}
	switch name {
	case "core.string_window":
		start, startKnown := reportEvalConstInt64(reportCallArg(call, 1))
		count, countKnown := reportEvalConstInt64(reportCallArg(call, 2))
		if !startKnown || !countKnown {
			return false
		}
		return start < 0 || count < 0 || start > sourceLen || count > sourceLen-start
	case "core.string_prefix":
		count, known := reportEvalConstInt64(reportCallArg(call, 1))
		return known && (count < 0 || count > sourceLen)
	case "core.string_suffix":
		start, known := reportEvalConstInt64(reportCallArg(call, 1))
		return known && (start < 0 || start > sourceLen)
	default:
		return false
	}
}

func reportCallArg(call *frontend.CallExpr, index int) frontend.Expr {
	if call == nil || index < 0 || index >= len(call.Args) {
		return nil
	}
	return call.Args[index]
}

func reportStaticStringByteLen(expr frontend.Expr) (int64, bool) {
	lit, ok := expr.(*frontend.StringLitExpr)
	if !ok || lit == nil {
		return 0, false
	}
	return int64(len(lit.Value)), true
}

func reportEvalConstInt64(expr frontend.Expr) (int64, bool) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		if e == nil {
			return 0, false
		}
		return int64(e.Value), true
	case *frontend.UnaryExpr:
		if e == nil || e.Op != frontend.TokenMinus {
			return 0, false
		}
		value, ok := reportEvalConstInt64(e.X)
		if !ok {
			return 0, false
		}
		return -value, true
	default:
		return 0, false
	}
}

func removedBoundsReason(proofID string) string {
	switch {
	case strings.HasPrefix(proofID, "proof:while:"):
		return "removed_by_while_range"
	case strings.HasPrefix(proofID, "proof:if:"):
		return "removed_by_branch_guard"
	case strings.HasPrefix(proofID, "proof:copy-loop:"):
		return "removed_by_copy_loop_range"
	case strings.HasPrefix(proofID, "proof:for-collection-view:"):
		return "removed_by_view_constructor"
	case strings.HasPrefix(proofID, "proof:for-collection:"):
		return "removed_by_for_loop_range"
	default:
		return "removed_by_for_loop_range"
	}
}

func buildAllocReport(prog *ir.IRProgram, target string) allocReport {
	report := allocReport{
		reportEnvelope: reportEnvelope{SchemaVersion: 1, Kind: "allocation", Target: target},
	}
	if prog == nil {
		return report
	}
	for _, fn := range prog.Funcs {
		row := allocFunctionRow{Function: fn.Name}
		for _, instr := range fn.Instrs {
			switch instr.Kind {
			case ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32, ir.IRAllocBytes:
				report.Totals.Heap++
				row.Allocations = append(row.Allocations, allocationDecision{
					Site:    reportPos(instr.Pos),
					Kind:    irAllocKind(instr.Kind),
					Storage: "Heap",
					Reason:  "allocation planner v0 keeps conservative heap storage until escape facts select a narrower class",
				})
			case ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32:
				report.Totals.Stack++
				row.Allocations = append(row.Allocations, allocationDecision{
					Site:    reportPos(instr.Pos),
					Kind:    irAllocKind(instr.Kind),
					Storage: "Stack",
					Reason:  "fixed small no-escape allocation lowers to stack frame storage",
				})
			case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32, ir.IRIslandNew:
				report.Totals.ExplicitIsland++
				row.Allocations = append(row.Allocations, allocationDecision{
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

func buildActorTransferReport(checked *semantics.CheckedProgram, target string) actorTransferReport {
	report := actorTransferReport{
		reportEnvelope: reportEnvelope{SchemaVersion: 1, Kind: "actor_transfer", Target: target},
	}
	if checked == nil {
		return report
	}
	mailboxes := map[string]actorMailboxRow{}
	for _, fn := range checked.Funcs {
		if fn.Decl == nil {
			continue
		}
		walkActorTransferStmts(fn.Decl.Body, func(call *frontend.CallExpr) {
			if msgType := actorMailboxMessageTypeFromCall(call, checked.Types); msgType != "" {
				if row, ok := actorMailboxRowForMessage(msgType, checked.Types, target); ok {
					mailboxes[row.Name] = row
				}
			}
			rows := actorTransferRowsForSend(fn.Name, call, checked.Types)
			for _, row := range rows {
				switch row.TransferMode {
				case "copy":
					report.Totals.Copy++
				case "move":
					report.Totals.Move++
				case "zero_copy_move":
					report.Totals.ZeroCopyMove++
				}
				report.Totals.BytesCopied += row.BytesCopied
				report.Sends = append(report.Sends, row)
			}
		})
	}
	if len(mailboxes) > 0 {
		names := make([]string, 0, len(mailboxes))
		for name := range mailboxes {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			report.Mailboxes = append(report.Mailboxes, mailboxes[name])
		}
	}
	return report
}

func actorMailboxMessageTypeFromCall(call *frontend.CallExpr, types map[string]*semantics.TypeInfo) string {
	if call == nil {
		return ""
	}
	switch call.Name {
	case "core.send_typed":
		if len(call.Args) < 2 {
			return ""
		}
		msgCall, ok := call.Args[1].(*frontend.CallExpr)
		if !ok {
			return ""
		}
		msgType, _, ok := reportEnumCaseConstructor(msgCall, types)
		if ok {
			return msgType
		}
	case "core.recv_typed":
		if len(call.TypeArgs) == 1 {
			return call.TypeArgs[0].Name
		}
	}
	return ""
}

func actorMailboxRowForMessage(typeName string, types map[string]*semantics.TypeInfo, target string) (actorMailboxRow, bool) {
	info, ok := types[typeName]
	if !ok || info.Kind != semantics.TypeEnum {
		return actorMailboxRow{}, false
	}
	row := actorMailboxRow{
		Name:              "typed:" + typeName,
		MessageSchema:     typeName,
		Capacity:          actorMailboxCapacityForTarget(target),
		CapacityUnit:      "messages",
		Backpressure:      "blocking_recv_yield",
		OverflowPolicy:    "unchecked_fixed_pool_overflow",
		MaxPayloadSlots:   8,
		PayloadSlots:      info.SlotCount - 1,
		SlotWidthBytes:    actorMailboxSlotWidthBytes(target),
		RuntimePath:       "actor_mailbox_typed_slots",
		OwnershipMetadata: true,
	}
	if err := actorsafety.VerifyMailbox(actorsafety.Mailbox{
		Name:         row.Name,
		Message:      row.MessageSchema,
		Capacity:     row.Capacity,
		Backpressure: row.Backpressure,
	}); err != nil {
		return actorMailboxRow{}, false
	}
	return row, true
}

func actorMailboxCapacityForTarget(target string) int {
	switch target {
	case "linux-x64", "macos-x64", "windows-x64":
		return 64 * 1024 / 88
	default:
		return 64 * 1024 / 88
	}
}

func actorMailboxSlotWidthBytes(target string) int {
	switch target {
	case "linux-x64", "macos-x64", "windows-x64":
		return 8
	default:
		return 8
	}
}

func actorTransferRowsForSend(function string, call *frontend.CallExpr, types map[string]*semantics.TypeInfo) []actorTransferRow {
	if call == nil || call.Name != "core.send_typed" || len(call.Args) < 2 {
		return nil
	}
	msgCall, ok := call.Args[1].(*frontend.CallExpr)
	if !ok {
		return nil
	}
	msgType, caseInfo, ok := reportEnumCaseConstructor(msgCall, types)
	if !ok {
		return nil
	}
	owners := make([]string, 0, len(caseInfo.PayloadTypes))
	for i, payloadType := range caseInfo.PayloadTypes {
		if i >= len(msgCall.Args) {
			break
		}
		if reportTypeKind(payloadType, types) == semantics.TypeIsland {
			if owner := reportExprPath(msgCall.Args[i]); owner != "" {
				owners = append(owners, owner)
			}
		}
	}
	rows := []actorTransferRow{}
	for i, payloadType := range caseInfo.PayloadTypes {
		if i >= len(msgCall.Args) {
			continue
		}
		if row, ok := actorTransferRowForPayload(function, call, msgType, caseInfo, i, payloadType, msgCall.Args[i], owners, types); ok {
			rows = append(rows, row)
		}
	}
	return rows
}

func actorTransferRowForPayload(
	function string,
	call *frontend.CallExpr,
	msgType string,
	caseInfo semantics.EnumCaseInfo,
	index int,
	payloadType string,
	expr frontend.Expr,
	owners []string,
	types map[string]*semantics.TypeInfo,
) (actorTransferRow, bool) {
	base := actorTransferRow{
		Function:                   function,
		Site:                       reportPos(call.At),
		MessageType:                msgType,
		Case:                       caseInfo.Name,
		PayloadIndex:               index,
		PayloadType:                payloadType,
		ClaimLevel:                 "validated",
		BoundaryScope:              "local_typed_mailbox",
		ProductionRuntimeValidated: false,
	}
	slotBytes := reportPayloadSlotCount(caseInfo, index, payloadType, types) * actorMailboxSlotWidthBytes("")
	switch reportTypeKind(payloadType, types) {
	case semantics.TypeI32, semantics.TypeU8, semantics.TypeBool:
		base.Ownership = "copy"
		base.TransferMode = "copy"
		base.RuntimePath = "actor_mailbox_value_slot"
		base.BytesCopied = slotBytes
		base.ZeroCopy = false
		base.Reason = "small scalar payload crosses typed actor mailbox by copy"
		return base, true
	case semantics.TypeIsland:
		base.Ownership = "owned_region"
		base.Owner = reportExprPath(expr)
		base.TransferMode = "move"
		base.RuntimePath = "actor_mailbox_resource_slot"
		base.BytesCopied = 0
		base.ZeroCopy = true
		base.ClaimLevel = "evidence_only"
		base.BoundaryScope = "local_typed_mailbox_owned_region_move"
		base.Reason = "island payload moves ownership across typed actor mailbox"
		return base, true
	case semantics.TypeStr, semantics.TypeSlice:
		if reportExprIsExplicitCopy(expr) {
			base.Ownership = "owned_copy"
			base.TransferMode = "copy"
			base.RuntimePath = "actor_mailbox_copy_region_slot"
			base.BytesCopied = slotBytes
			base.ZeroCopy = false
			base.Reason = "borrowed view crosses actor boundary through explicit copy"
			return base, true
		}
		if reportTypeKind(payloadType, types) == semantics.TypeSlice {
			owner := reportOwnedRegionSliceOwner(expr, owners)
			if owner == "" {
				return actorTransferRow{}, false
			}
			base.Ownership = "owned_region_slice"
			base.Owner = owner
			base.TransferMode = "zero_copy_move"
			base.RuntimePath = "actor_mailbox_zero_copy_region_slot"
			base.BytesCopied = 0
			base.ZeroCopy = true
			base.ClaimLevel = "evidence_only"
			base.BoundaryScope = "local_typed_mailbox_owned_region_slice_move"
			base.Reason = "owned region-backed slice moves with its island owner in the same typed actor payload"
			return base, true
		}
	case semantics.TypeStruct, semantics.TypeEnum:
		base.Ownership = "copy"
		base.TransferMode = "copy"
		base.RuntimePath = "actor_mailbox_aggregate_value_slots"
		base.BytesCopied = slotBytes
		base.ZeroCopy = false
		base.Reason = "value-only aggregate payload crosses typed actor mailbox by slot copy"
		return base, true
	}
	return actorTransferRow{}, false
}

func reportPayloadSlotCount(caseInfo semantics.EnumCaseInfo, index int, payloadType string, types map[string]*semantics.TypeInfo) int {
	if index >= 0 && index < len(caseInfo.PayloadSlots) && caseInfo.PayloadSlots[index] > 0 {
		return caseInfo.PayloadSlots[index]
	}
	if info, ok := types[payloadType]; ok && info.SlotCount > 0 {
		return info.SlotCount
	}
	return 1
}

func reportEnumCaseConstructor(call *frontend.CallExpr, types map[string]*semantics.TypeInfo) (string, semantics.EnumCaseInfo, bool) {
	if call == nil {
		return "", semantics.EnumCaseInfo{}, false
	}
	if call.ResolvedType != "" {
		if info, ok := types[call.ResolvedType]; ok && info.Kind == semantics.TypeEnum {
			caseName := reportCallCaseName(call.Name)
			if caseInfo, ok := info.CaseMap[caseName]; ok {
				return call.ResolvedType, caseInfo, true
			}
		}
	}
	parts := strings.Split(call.Name, ".")
	if len(parts) < 2 {
		return "", semantics.EnumCaseInfo{}, false
	}
	typeName := strings.Join(parts[:len(parts)-1], ".")
	caseName := parts[len(parts)-1]
	info, ok := types[typeName]
	if !ok || info.Kind != semantics.TypeEnum {
		return "", semantics.EnumCaseInfo{}, false
	}
	caseInfo, ok := info.CaseMap[caseName]
	if !ok {
		return "", semantics.EnumCaseInfo{}, false
	}
	return typeName, caseInfo, true
}

func reportCallCaseName(name string) string {
	parts := strings.Split(name, ".")
	if len(parts) == 0 {
		return name
	}
	return parts[len(parts)-1]
}

func reportTypeKind(typeName string, types map[string]*semantics.TypeInfo) semantics.TypeKind {
	if info, ok := types[typeName]; ok {
		return info.Kind
	}
	return 0
}

func reportOwnedRegionSliceOwner(expr frontend.Expr, owners []string) string {
	if len(owners) == 0 || expr == nil || reportExprIsExplicitCopy(expr) {
		return ""
	}
	if call, ok := expr.(*frontend.CallExpr); ok && len(call.Args) > 0 && reportIsIslandMakeCall(call.Name) {
		owner := reportExprPath(call.Args[0])
		if reportStringIn(owner, owners) {
			return owner
		}
		return ""
	}
	return owners[0]
}

func reportIsIslandMakeCall(name string) bool {
	switch name {
	case "core.island_make_u8", "core.island_make_u16", "core.island_make_i32", "core.island_make_bool",
		"island_make_u8", "island_make_u16", "island_make_i32", "island_make_bool":
		return true
	default:
		return false
	}
}

func reportExprIsExplicitCopy(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	name := call.Name
	if name == "core.string_copy" || name == "string_copy" {
		return true
	}
	return (strings.HasPrefix(name, "core.slice_copy_") || strings.HasPrefix(name, "slice_copy_")) &&
		!strings.HasPrefix(name, "core.slice_copy_into_") &&
		!strings.HasPrefix(name, "slice_copy_into_")
}

func reportExprPath(expr frontend.Expr) string {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name
	case *frontend.FieldAccessExpr:
		base := reportExprPath(e.Base)
		if base == "" || e.Field == "" {
			return base
		}
		return base + "." + e.Field
	default:
		return ""
	}
}

func reportStringIn(value string, values []string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func walkActorTransferStmts(stmts []frontend.Stmt, visit func(*frontend.CallExpr)) {
	for _, stmt := range stmts {
		walkActorTransferStmt(stmt, visit)
	}
}

func walkActorTransferStmt(stmt frontend.Stmt, visit func(*frontend.CallExpr)) {
	switch s := stmt.(type) {
	case *frontend.PrintStmt:
		walkActorTransferExpr(s.Value, visit)
	case *frontend.ReturnStmt:
		walkActorTransferExpr(s.Value, visit)
	case *frontend.ThrowStmt:
		walkActorTransferExpr(s.Value, visit)
	case *frontend.DeferStmt:
		walkActorTransferStmts(s.Body, visit)
	case *frontend.LetStmt:
		walkActorTransferExpr(s.Value, visit)
	case *frontend.AssignStmt:
		walkActorTransferExpr(s.Target, visit)
		walkActorTransferExpr(s.Value, visit)
		walkActorTransferExpr(s.CompoundValue, visit)
	case *frontend.IfStmt:
		walkActorTransferExpr(s.Cond, visit)
		walkActorTransferStmts(s.Then, visit)
		walkActorTransferStmts(s.Else, visit)
	case *frontend.IfLetStmt:
		walkActorTransferExpr(s.Pattern, visit)
		walkActorTransferExpr(s.Value, visit)
		walkActorTransferStmts(s.Then, visit)
		walkActorTransferStmts(s.Else, visit)
	case *frontend.WhileStmt:
		walkActorTransferExpr(s.Cond, visit)
		walkActorTransferStmts(s.Body, visit)
	case *frontend.ForRangeStmt:
		walkActorTransferExpr(s.Start, visit)
		walkActorTransferExpr(s.End, visit)
		walkActorTransferExpr(s.Iterable, visit)
		walkActorTransferStmts(s.Body, visit)
	case *frontend.MatchStmt:
		walkActorTransferExpr(s.Value, visit)
		for _, c := range s.Cases {
			walkActorTransferExpr(c.Pattern, visit)
			walkActorTransferExpr(c.Guard, visit)
			walkActorTransferStmts(c.Body, visit)
		}
	case *frontend.FreeStmt:
		walkActorTransferExpr(s.Value, visit)
	case *frontend.UnsafeStmt:
		walkActorTransferStmts(s.Body, visit)
	case *frontend.IslandStmt:
		walkActorTransferExpr(s.Size, visit)
		walkActorTransferStmts(s.Body, visit)
	case *frontend.ExprStmt:
		walkActorTransferExpr(s.Expr, visit)
	case *frontend.ExpectStmt:
		walkActorTransferExpr(s.Cond, visit)
	}
}

func walkActorTransferExpr(expr frontend.Expr, visit func(*frontend.CallExpr)) {
	switch e := expr.(type) {
	case nil:
		return
	case *frontend.BinaryExpr:
		walkActorTransferExpr(e.Left, visit)
		walkActorTransferExpr(e.Right, visit)
	case *frontend.UnaryExpr:
		walkActorTransferExpr(e.X, visit)
	case *frontend.TryExpr:
		walkActorTransferExpr(e.X, visit)
	case *frontend.AwaitExpr:
		walkActorTransferExpr(e.X, visit)
	case *frontend.CallExpr:
		visit(e)
		for _, arg := range e.Args {
			walkActorTransferExpr(arg, visit)
		}
	case *frontend.ClosureExpr:
		if e.Decl != nil {
			walkActorTransferStmts(e.Decl.Body, visit)
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			walkActorTransferExpr(field.Value, visit)
		}
	case *frontend.FieldAccessExpr:
		walkActorTransferExpr(e.Base, visit)
	case *frontend.IndexExpr:
		walkActorTransferExpr(e.Base, visit)
		walkActorTransferExpr(e.Index, visit)
	case *frontend.MatchExpr:
		walkActorTransferExpr(e.Value, visit)
		for _, c := range e.Cases {
			walkActorTransferExpr(c.Pattern, visit)
			walkActorTransferExpr(c.Guard, visit)
			walkActorTransferExpr(c.Value, visit)
		}
	case *frontend.CatchExpr:
		walkActorTransferExpr(e.Call, visit)
		for _, c := range e.Cases {
			walkActorTransferExpr(c.Pattern, visit)
			walkActorTransferExpr(c.Guard, visit)
			walkActorTransferExpr(c.Value, visit)
		}
	}
}

func writeReport(path string, data any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func formatExplainText(target string, bounds boundsReport, alloc *allocplan.Plan, plirProg *plir.Program) string {
	var b strings.Builder
	fmt.Fprintf(&b, "target: %s\n", target)
	fmt.Fprintf(&b, "bounds checks removed: %d\n", bounds.Totals.Removed)
	fmt.Fprintf(&b, "bounds checks left: %d\n", bounds.Totals.Left)
	if alloc != nil {
		fmt.Fprintf(&b, "planned heap allocations: %d\n", alloc.Totals.Heap)
		fmt.Fprintf(&b, "planned stack allocations: %d\n", alloc.Totals.Stack)
		fmt.Fprintf(&b, "explicit island allocations: %d\n\n", alloc.Totals.ExplicitIsland)
		b.WriteString(allocplan.FormatText(alloc))
		b.WriteString("\n")
	}
	b.WriteString(plir.FormatText(plirProg))
	return b.String()
}

func isUncheckedIndexLoad(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRIndexLoadI32Unchecked, ir.IRIndexLoadU8Unchecked, ir.IRIndexLoadU16Unchecked:
		return true
	default:
		return false
	}
}

func isCheckedIndexAccess(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRIndexLoadI32, ir.IRIndexLoadU8, ir.IRIndexLoadU16,
		ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
		return true
	default:
		return false
	}
}

func irIndexKind(kind ir.IRInstrKind) string {
	switch kind {
	case ir.IRIndexLoadI32, ir.IRIndexLoadI32Unchecked:
		return "i32.load"
	case ir.IRIndexLoadU8, ir.IRIndexLoadU8Unchecked:
		return "u8.load"
	case ir.IRIndexLoadU16, ir.IRIndexLoadU16Unchecked:
		return "u16.load"
	case ir.IRIndexStoreI32:
		return "i32.store"
	case ir.IRIndexStoreU8:
		return "u8.store"
	case ir.IRIndexStoreU16:
		return "u16.store"
	default:
		return fmt.Sprintf("ir.%d", kind)
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
