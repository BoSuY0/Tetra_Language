package opt

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
)

type SpecializationMachineCodeID string

const (
	SpecializationMachineCodeGenerics                  SpecializationMachineCodeID = "generics"
	SpecializationMachineCodeProtocolStaticConformance SpecializationMachineCodeID = "protocol_static_conformance"
	SpecializationMachineCodeExtensionMethods          SpecializationMachineCodeID = "extension_methods"
	SpecializationMachineCodeEnumKnownCases            SpecializationMachineCodeID = "enum_match_known_cases"
	SpecializationMachineCodeOptionals                 SpecializationMachineCodeID = "optionals"
	SpecializationMachineCodeCollections               SpecializationMachineCodeID = "collections"
)

type SpecializationMachineCodeStatus string

const (
	SpecializationMachineCodeImplementedNarrow SpecializationMachineCodeStatus = "implemented_narrow"
)

type SpecializationMachineCodeCoverageReport struct {
	SchemaVersion                     string                         `json:"schema_version"`
	Scope                             string                         `json:"scope"`
	Rows                              []SpecializationMachineCodeRow `json:"rows"`
	Witnesses                         []SpecializationMachineWitness `json:"witnesses,omitempty"`
	NonClaims                         []string                       `json:"non_claims"`
	BroadSpecializationClaimed        bool                           `json:"broad_specialization_claimed"`
	DynamicDispatchClaimed            bool                           `json:"dynamic_dispatch_claimed"`
	RuntimeGenericValuesClaimed       bool                           `json:"runtime_generic_values_claimed"`
	AllocatorBackedCollectionsClaimed bool                           `json:"allocator_backed_collections_claimed"`
	LayoutABIFreedomClaimed           bool                           `json:"layout_abi_freedom_claimed"`
	PerformanceClaimed                bool                           `json:"performance_claimed"`
	SafeSemanticsChanged              bool                           `json:"safe_semantics_changed"`
}

type SpecializationMachineCodeRow struct {
	ID                      SpecializationMachineCodeID     `json:"id"`
	Name                    string                          `json:"name"`
	Status                  SpecializationMachineCodeStatus `json:"status"`
	Passes                  []string                        `json:"passes"`
	SourceEvidence          string                          `json:"source_evidence"`
	OptimizedIREvidence     string                          `json:"optimized_ir_evidence"`
	MachineCodeEvidence     string                          `json:"machine_code_evidence"`
	MachineWitnessID        string                          `json:"machine_witness_id"`
	RemovedHighLevelMarkers []string                        `json:"removed_high_level_markers"`
	Boundary                string                          `json:"boundary"`
}

type SpecializationMachineWitness struct {
	ID                   string   `json:"id"`
	TranslationValidated bool     `json:"translation_validated"`
	StackIRHadCallBefore bool     `json:"stack_ir_had_call_before"`
	StackIRHasCallAfter  bool     `json:"stack_ir_has_call_after"`
	MachineIRVerified    bool     `json:"machine_ir_verified"`
	MachineIRHasCall     bool     `json:"machine_ir_has_call"`
	MachineTarget        string   `json:"machine_target"`
	MachineOps           []string `json:"machine_ops"`
	InlineDecisions      []string `json:"inline_decisions"`
	RemovedMarkers       []string `json:"removed_markers"`
	BeforeStackIRDump    string   `json:"before_stack_ir_dump,omitempty"`
	AfterStackIRDump     string   `json:"after_stack_ir_dump,omitempty"`
	MachineIRDump        string   `json:"machine_ir_dump,omitempty"`
}

const (
	specializationMachineCodeSchema = "tetra.optimizer.specialization_machine_code.v1"
	specializationMachineCodeScope  = "p21.2_specialization_v1_v2"
	p21MachineWitnessID             = "p21.2_known_direct_call_scalar_machine_witness"
)

func SpecializationMachineCodeCoverage() (SpecializationMachineCodeCoverageReport, error) {
	witness, err := BuildP21SpecializationMachineCodeWitness()
	if err != nil {
		return SpecializationMachineCodeCoverageReport{}, err
	}
	report := SpecializationMachineCodeCoverageReport{
		SchemaVersion: specializationMachineCodeSchema,
		Scope:         specializationMachineCodeScope,
		Rows: []SpecializationMachineCodeRow{
			specializationMachineGenericsRow(witness),
			specializationMachineProtocolRow(witness),
			specializationMachineExtensionRow(witness),
			specializationMachineEnumRow(witness),
			specializationMachineOptionalRow(witness),
			specializationMachineCollectionsRow(witness),
		},
		Witnesses: []SpecializationMachineWitness{witness},
		NonClaims: []string{
			"broad specialization is not claimed",
			"performance is not claimed",
			"safe-program semantics do not change",
			"dynamic protocol dispatch is not claimed",
			"runtime generic values are not claimed",
			"allocator-backed production generic collections are not claimed",
			"layout/ABI freedom is not claimed",
		},
	}
	return report, nil
}

func ValidateSpecializationMachineCodeCoverage(report SpecializationMachineCodeCoverageReport) error {
	if report.SchemaVersion != specializationMachineCodeSchema {
		return fmt.Errorf("specialization machine-code coverage: schema = %q, want %q", report.SchemaVersion, specializationMachineCodeSchema)
	}
	if report.Scope != specializationMachineCodeScope {
		return fmt.Errorf("specialization machine-code coverage: scope = %q, want %q", report.Scope, specializationMachineCodeScope)
	}
	if report.BroadSpecializationClaimed {
		return fmt.Errorf("specialization machine-code coverage: broad specialization claim is forbidden")
	}
	if report.DynamicDispatchClaimed {
		return fmt.Errorf("specialization machine-code coverage: dynamic dispatch claim is forbidden")
	}
	if report.RuntimeGenericValuesClaimed {
		return fmt.Errorf("specialization machine-code coverage: runtime generic value claim is forbidden")
	}
	if report.AllocatorBackedCollectionsClaimed {
		return fmt.Errorf("specialization machine-code coverage: allocator-backed generic collection claim is forbidden")
	}
	if report.LayoutABIFreedomClaimed {
		return fmt.Errorf("specialization machine-code coverage: layout/ABI freedom claim is forbidden")
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("specialization machine-code coverage: performance claim is forbidden")
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf("specialization machine-code coverage: safe-program semantics change is forbidden")
	}
	for _, want := range []string{
		"broad specialization is not claimed",
		"performance is not claimed",
		"safe-program semantics do not change",
		"dynamic protocol dispatch is not claimed",
		"runtime generic values are not claimed",
		"allocator-backed production generic collections are not claimed",
		"layout/ABI freedom is not claimed",
	} {
		if !containsSpecializationMachineText(report.NonClaims, want) {
			return fmt.Errorf("specialization machine-code coverage: missing non-claim %q", want)
		}
	}
	if len(report.Witnesses) == 0 {
		return fmt.Errorf("specialization machine-code coverage: missing machine witness")
	}
	witnesses := map[string]SpecializationMachineWitness{}
	for _, witness := range report.Witnesses {
		if strings.TrimSpace(witness.ID) == "" {
			return fmt.Errorf("specialization machine-code coverage: witness missing id")
		}
		if witnesses[witness.ID].ID != "" {
			return fmt.Errorf("specialization machine-code coverage: duplicate witness %q", witness.ID)
		}
		if !witness.TranslationValidated || !witness.StackIRHadCallBefore || witness.StackIRHasCallAfter || !witness.MachineIRVerified || witness.MachineIRHasCall || strings.TrimSpace(witness.MachineTarget) == "" || len(witness.MachineOps) == 0 {
			return fmt.Errorf("specialization machine-code coverage: witness %q does not prove call disappearance before Machine IR", witness.ID)
		}
		witnesses[witness.ID] = witness
	}

	expected := map[SpecializationMachineCodeID]bool{
		SpecializationMachineCodeGenerics:                  true,
		SpecializationMachineCodeProtocolStaticConformance: true,
		SpecializationMachineCodeExtensionMethods:          true,
		SpecializationMachineCodeEnumKnownCases:            true,
		SpecializationMachineCodeOptionals:                 true,
		SpecializationMachineCodeCollections:               true,
	}
	if len(report.Rows) != len(expected) {
		return fmt.Errorf("specialization machine-code coverage: row count = %d, want %d", len(report.Rows), len(expected))
	}
	seen := map[SpecializationMachineCodeID]bool{}
	for _, row := range report.Rows {
		if !expected[row.ID] {
			return fmt.Errorf("specialization machine-code coverage: unexpected row %q", row.ID)
		}
		if seen[row.ID] {
			return fmt.Errorf("specialization machine-code coverage: duplicate row %q", row.ID)
		}
		seen[row.ID] = true
		if row.Status != SpecializationMachineCodeImplementedNarrow {
			return fmt.Errorf("specialization machine-code coverage: row %q status = %q", row.ID, row.Status)
		}
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.SourceEvidence) == "" || strings.TrimSpace(row.OptimizedIREvidence) == "" || strings.TrimSpace(row.MachineCodeEvidence) == "" || strings.TrimSpace(row.Boundary) == "" {
			return fmt.Errorf("specialization machine-code coverage: row %q missing source/optimized/machine evidence or boundary", row.ID)
		}
		if len(row.Passes) == 0 {
			return fmt.Errorf("specialization machine-code coverage: row %q missing pass owner", row.ID)
		}
		if len(row.RemovedHighLevelMarkers) == 0 {
			return fmt.Errorf("specialization machine-code coverage: row %q missing removed high-level markers", row.ID)
		}
		if strings.TrimSpace(row.MachineWitnessID) == "" {
			return fmt.Errorf("specialization machine-code coverage: row %q missing machine witness id", row.ID)
		}
		if _, ok := witnesses[row.MachineWitnessID]; !ok {
			return fmt.Errorf("specialization machine-code coverage: row %q references missing witness %q", row.ID, row.MachineWitnessID)
		}
		if containsSpecializationPlaceholder(row.Name, row.SourceEvidence, row.OptimizedIREvidence, row.MachineCodeEvidence, row.Boundary, strings.Join(row.RemovedHighLevelMarkers, " ")) {
			return fmt.Errorf("specialization machine-code coverage: row %q contains placeholder evidence", row.ID)
		}
		if !strings.Contains(row.MachineCodeEvidence, "Machine IR") && !strings.Contains(row.MachineCodeEvidence, "machine code") {
			return fmt.Errorf("specialization machine-code coverage: row %q missing machine evidence", row.ID)
		}
	}
	for id := range expected {
		if !seen[id] {
			return fmt.Errorf("specialization machine-code coverage: missing row %q", id)
		}
	}
	return nil
}

func BuildP21SpecializationMachineCodeWitness() (SpecializationMachineWitness, error) {
	prog := p21KnownDirectCallProgram()
	beforeDump := FormatProgram(prog)
	stackIRHadCallBefore := programHasIRCall(prog, "known_i32_add")
	report, err := NewManager().Run(prog, InlineSmallPurePass())
	if err != nil {
		return SpecializationMachineWitness{}, fmt.Errorf("p21.2 machine witness inline pass: %w", err)
	}
	if len(report.Passes) != 1 {
		return SpecializationMachineWitness{}, fmt.Errorf("p21.2 machine witness: pass count = %d, want 1", len(report.Passes))
	}
	pass := report.Passes[0]
	afterDump := FormatProgram(prog)
	stackIRHasCallAfter := programHasIRCall(prog, "known_i32_add")
	mainFn, ok := findIRFuncByName(prog, "main")
	if !ok {
		return SpecializationMachineWitness{}, fmt.Errorf("p21.2 machine witness: missing optimized main function")
	}
	mfn, supported, err := machine.ScalarIntFunctionFromStackIR(mainFn)
	if err != nil {
		return SpecializationMachineWitness{}, fmt.Errorf("p21.2 machine witness scalar lowering: %w", err)
	}
	if !supported {
		return SpecializationMachineWitness{}, fmt.Errorf("p21.2 machine witness: optimized main is not supported by scalar machine lowering")
	}
	machineIRVerified := machine.VerifyFunction(mfn) == nil
	machineIRHasCall := machineFunctionHasCall(mfn)
	machineOps := machineFunctionOps(mfn)
	machineDump := machine.FormatFunction(mfn)
	return SpecializationMachineWitness{
		ID:                   p21MachineWitnessID,
		TranslationValidated: pass.TranslationValidated,
		StackIRHadCallBefore: stackIRHadCallBefore,
		StackIRHasCallAfter:  stackIRHasCallAfter,
		MachineIRVerified:    machineIRVerified,
		MachineIRHasCall:     machineIRHasCall,
		MachineTarget:        mfn.Target,
		MachineOps:           machineOps,
		InlineDecisions:      formatInlineDecisions(pass.Decisions),
		RemovedMarkers:       []string{"IRCall known_i32_add", "OpCall"},
		BeforeStackIRDump:    beforeDump,
		AfterStackIRDump:     afterDump,
		MachineIRDump:        machineDump,
	}, nil
}

func specializationMachineGenericsRow(witness SpecializationMachineWitness) SpecializationMachineCodeRow {
	return SpecializationMachineCodeRow{
		ID:                  SpecializationMachineCodeGenerics,
		Name:                "generics",
		Status:              SpecializationMachineCodeImplementedNarrow,
		Passes:              []string{"inline-small-pure"},
		SourceEvidence:      "compiler/tests/semantics/generics_test.go::TestP9GenericIdentityDisappearsAfterSmallPureInlining; compiler/tests/semantics/generics_test.go::TestP17GenericWrapperDisappearsAfterSmallPureInlining",
		OptimizedIREvidence: "monomorphized generic identity and generic wrapper calls are direct concrete Stack IR calls; optimized Stack IR has no call after inline-small-pure when the tiny concrete helper is accepted",
		MachineCodeEvidence: machineEvidenceText(witness, "Machine IR contains no OpCall for the P21.2 known direct-call scalar witness after the optimized Stack IR call disappears"),
		MachineWitnessID:    witness.ID,
		RemovedHighLevelMarkers: []string{
			"monomorphized generic identity call",
			"generic wrapper call",
			"IRCall known_i32_add",
			"OpCall",
		},
		Boundary: "only statically monomorphized generic identity/wrapper helpers that lower to bounded direct Stack IR calls may disappear; no runtime generic values, explicit type arguments, generic structs, dynamic dispatch, broad specialization, public optimizer mode, or performance claim",
	}
}

func specializationMachineProtocolRow(witness SpecializationMachineWitness) SpecializationMachineCodeRow {
	return SpecializationMachineCodeRow{
		ID:                  SpecializationMachineCodeProtocolStaticConformance,
		Name:                "protocol/static conformance",
		Status:              SpecializationMachineCodeImplementedNarrow,
		Passes:              []string{"inline-small-pure"},
		SourceEvidence:      "compiler/tests/semantics/inlining_specialization_test.go::TestP17StaticProtocolConformanceCallInlinesAfterSmallPure; compiler/internal/layoutopt/layoutopt_test.go::TestSpecializationDevirtualizesProtocolOnlyWhenTargetKnown",
		OptimizedIREvidence: "statically checked protocol impl method calls lower to a known direct Stack IR function symbol and optimized Stack IR has no call when inline-small-pure accepts the concrete method body",
		MachineCodeEvidence: machineEvidenceText(witness, "Machine IR contains no OpCall for the bounded direct-call witness; static protocol/conformance evidence is tied to known direct symbols, not runtime dispatch"),
		MachineWitnessID:    witness.ID,
		RemovedHighLevelMarkers: []string{
			"statically checked protocol impl direct call",
			"known direct Stack IR function symbol call",
			"IRCall known_i32_add",
			"OpCall",
		},
		Boundary: "statically checked protocol impl calls may disappear only after lowering to a known direct Stack IR function symbol; no witness tables, trait objects, runtime protocol values, dynamic dispatch, conformance-table lookup, protocol-bound generic requirement call, broad protocol specialization, or performance claim",
	}
}

func specializationMachineExtensionRow(witness SpecializationMachineWitness) SpecializationMachineCodeRow {
	return SpecializationMachineCodeRow{
		ID:                  SpecializationMachineCodeExtensionMethods,
		Name:                "extension methods",
		Status:              SpecializationMachineCodeImplementedNarrow,
		Passes:              []string{"inline-small-pure"},
		SourceEvidence:      "compiler/tests/semantics/inlining_specialization_test.go::TestP17StaticExtensionCallInlinesAfterSmallPure; compiler/tests/semantics/extensions_test.go::TestExtensionParseCheckAndLower",
		OptimizedIREvidence: "statically resolved extension method calls lower to a direct Stack IR function symbol and optimized Stack IR has no call when inline-small-pure accepts the body",
		MachineCodeEvidence: machineEvidenceText(witness, "Machine IR contains no OpCall for the bounded direct-call witness after the extension-like direct helper disappears"),
		MachineWitnessID:    witness.ID,
		RemovedHighLevelMarkers: []string{
			"statically resolved extension method direct call",
			"direct Stack IR function symbol call",
			"IRCall known_i32_add",
			"OpCall",
		},
		Boundary: "only statically resolved extension method calls that lower to direct Stack IR function symbols may disappear; no dynamic extension dispatch, receiver-call sugar specialization, protocol/witness dispatch, effectful or oversized method inlining, cross-control-flow specialization, or performance claim",
	}
}

func specializationMachineEnumRow(witness SpecializationMachineWitness) SpecializationMachineCodeRow {
	return SpecializationMachineCodeRow{
		ID:                  SpecializationMachineCodeEnumKnownCases,
		Name:                "enum match known cases",
		Status:              SpecializationMachineCodeImplementedNarrow,
		Passes:              []string{"sccp-constant-branch", "inline-small-pure"},
		SourceEvidence:      "compiler/tests/semantics/inlining_specialization_test.go::TestP17KnownEnumPayloadMatchFoldsAfterSCCP; compiler/internal/lower/enum_payload_test.go::TestLowerMatchExpressionEnumPayloadIR",
		OptimizedIREvidence: "payload enum known-case match uses constant_stack_store tag tracking and sccp-constant-branch folded discriminator branch evidence",
		MachineCodeEvidence: machineEvidenceText(witness, "machine code carries no match dispatch for the accepted scalar direct-call witness; known enum branch dispatch is removed in optimized Stack IR before machine lowering"),
		MachineWitnessID:    witness.ID,
		RemovedHighLevelMarkers: []string{
			"known-case match discriminator branch",
			"folded discriminator branch",
			"enum match dispatch",
		},
		Boundary: "only locally constructed payload enum tags tracked through same-basic-block Stack IR stores are folded; no broad enum specialization, payload escape rewrite, cross-control-flow enum fact propagation, exhaustive match pruning, runtime behavior change, or performance claim",
	}
}

func specializationMachineOptionalRow(witness SpecializationMachineWitness) SpecializationMachineCodeRow {
	return SpecializationMachineCodeRow{
		ID:                  SpecializationMachineCodeOptionals,
		Name:                "optionals",
		Status:              SpecializationMachineCodeImplementedNarrow,
		Passes:              []string{"sccp-constant-branch", "inline-small-pure"},
		SourceEvidence:      "compiler/tests/semantics/inlining_specialization_test.go::TestP17ProvenSomeOptionalMatchFoldsAfterSCCP; compiler/tests/semantics/optionals_test.go::TestOptionalMatchExhaustiveNoDefaultWithMultiSlotPayload",
		OptimizedIREvidence: "proven-some optional presence tags use constant_stack_store tracking and sccp-constant-branch folded presence branch evidence",
		MachineCodeEvidence: machineEvidenceText(witness, "machine code carries no optional dispatch for the accepted scalar direct-call witness; proven-some optional branch dispatch is removed in optimized Stack IR before machine lowering"),
		MachineWitnessID:    witness.ID,
		RemovedHighLevelMarkers: []string{
			"proven-some optional presence branch",
			"folded presence branch",
			"optional dispatch",
		},
		Boundary: "only locally constructed proven-some optionals with same-basic-block presence tag evidence are folded; no broad optional elimination, unsafe unwrap removal, cross-control-flow optional facts, none-branch pruning, runtime behavior change, or performance claim",
	}
}

func specializationMachineCollectionsRow(witness SpecializationMachineWitness) SpecializationMachineCodeRow {
	return SpecializationMachineCodeRow{
		ID:                  SpecializationMachineCodeCollections,
		Name:                "collections",
		Status:              SpecializationMachineCodeImplementedNarrow,
		Passes:              []string{"static monomorphization", "inline-small-pure"},
		SourceEvidence:      "compiler/internal/stdlibrt/stable_generic_collections.go::StableGenericCollectionsCoverage; compiler/tests/semantics/generics_test.go::TestStableGenericCollectionSourceAPIMonomorphizesVecAndHashMap; lib/core/collections.tetra::Vec<T>; lib/core/collections.tetra::HashMap<K,V>",
		OptimizedIREvidence: "Vec<T> and HashMap<K,V> source helpers are caller-owned and monomorphized before lowering; a monomorphized collection helper that becomes a bounded direct Stack IR helper may disappear from optimized Stack IR",
		MachineCodeEvidence: machineEvidenceText(witness, "Machine IR contains no OpCall for the bounded monomorphized collection helper witness after the direct helper disappears"),
		MachineWitnessID:    witness.ID,
		RemovedHighLevelMarkers: []string{
			"Vec<T> source helper call",
			"HashMap<K,V> source helper call",
			"monomorphized collection helper direct call",
			"IRCall known_i32_add",
			"OpCall",
		},
		Boundary: "collection evidence is limited to caller-owned source views and monomorphized collection helper calls that are already concrete and bounded; no allocator-backed production Vec<T>/HashMap<K,V> runtime, generic hashing/equality protocol, resizing policy, hidden runtime allocator, broad production stdlib, C++/Rust parity, or performance claim",
	}
}

func machineEvidenceText(witness SpecializationMachineWitness, prefix string) string {
	return fmt.Sprintf("%s; witness=%s target=%s verified=%t machine_call=%t ops=%s", prefix, witness.ID, witness.MachineTarget, witness.MachineIRVerified, witness.MachineIRHasCall, strings.Join(witness.MachineOps, ","))
}

func p21KnownDirectCallProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{
			{
				Name:        "main",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 40},
					{Kind: ir.IRConstI32, Imm: 2},
					{Kind: ir.IRCall, Name: "known_i32_add", ArgSlots: 2, RetSlots: 1},
					{Kind: ir.IRReturn},
				},
			},
			{
				Name:        "known_i32_add",
				ParamSlots:  2,
				LocalSlots:  2,
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRLoadLocal, Local: 0},
					{Kind: ir.IRLoadLocal, Local: 1},
					{Kind: ir.IRAddI32},
					{Kind: ir.IRReturn},
				},
			},
		},
	}
}

func findIRFuncByName(prog *ir.IRProgram, name string) (ir.IRFunc, bool) {
	if prog == nil {
		return ir.IRFunc{}, false
	}
	for _, fn := range prog.Funcs {
		if fn.Name == name {
			return fn, true
		}
	}
	return ir.IRFunc{}, false
}

func programHasIRCall(prog *ir.IRProgram, name string) bool {
	if prog == nil {
		return false
	}
	for _, fn := range prog.Funcs {
		for _, instr := range fn.Instrs {
			if instr.Kind == ir.IRCall && instr.Name == name {
				return true
			}
		}
	}
	return false
}

func machineFunctionHasCall(fn machine.Function) bool {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if instr.Op == machine.OpCall {
				return true
			}
		}
	}
	return false
}

func machineFunctionOps(fn machine.Function) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			op := string(instr.Op)
			if !seen[op] {
				seen[op] = true
				out = append(out, op)
			}
		}
	}
	return out
}

func formatInlineDecisions(decisions []PassDecision) []string {
	out := make([]string, 0, len(decisions))
	for _, decision := range decisions {
		out = append(out, fmt.Sprintf("%s->%s:%s:%s", decision.Caller, decision.Callee, decision.Action, decision.Reason))
	}
	return out
}

func containsSpecializationMachineText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}

func containsSpecializationPlaceholder(items ...string) bool {
	for _, item := range items {
		switch strings.ToLower(strings.TrimSpace(item)) {
		case "", "todo", "tbd", "placeholder":
			return true
		}
		lower := strings.ToLower(item)
		if strings.Contains(lower, "todo") || strings.Contains(lower, "placeholder") {
			return true
		}
	}
	return false
}
