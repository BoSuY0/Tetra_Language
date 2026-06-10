package validation

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/plir"
)

type Stage string

const (
	StageTypedAST       Stage = "typed_ast"
	StagePLIR           Stage = "plir"
	StageProofFacts     Stage = "proof_facts"
	StageOptimizedIR    Stage = "optimized_ir"
	StageAllocationPlan Stage = "allocation_plan"
	StageMachineIR      Stage = "machine_ir"
	StageABI            Stage = "abi"
	StageObjectSmoke    Stage = "object_smoke"
)

type StageVerifier struct {
	Stage       Stage  `json:"stage"`
	Verifier    string `json:"verifier"`
	Implemented bool   `json:"implemented"`
}

type ProofReport struct {
	RemovedChecks []RemovedCheck `json:"removed_checks,omitempty"`
	LeftChecks    int            `json:"left_checks"`
}

type RemovedCheck struct {
	Function  string     `json:"function"`
	Site      int        `json:"site"`
	Kind      string     `json:"kind"`
	ProofID   string     `json:"proof_id"`
	ProofTerm *ProofTerm `json:"proof_term,omitempty"`
	FactsUsed []string   `json:"facts_used,omitempty"`
}

type ProofTerm struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	SubjectBaseID string   `json:"subject_base_id,omitempty"`
	IndexValueID  string   `json:"index_value_id,omitempty"`
	Operation     string   `json:"operation,omitempty"`
	Range         string   `json:"range,omitempty"`
	IslandID      string   `json:"island_id,omitempty"`
	Epoch         int      `json:"epoch,omitempty"`
	BaseID        string   `json:"base_id,omitempty"`
	Source        string   `json:"source,omitempty"`
	FactsUsed     []string `json:"facts_used,omitempty"`
}

type plirProofUseEvidence struct {
	ProofID string
	OpID    string
	OpKind  plir.OperationKind
	OpNote  string
}

type TranslationReport struct {
	FunctionsCompared   int      `json:"functions_compared"`
	Functions           []string `json:"functions,omitempty"`
	ProofFactsCompared  int      `json:"proof_facts_compared,omitempty"`
	SemanticLocalChecks int      `json:"semantic_local_checks,omitempty"`
	DifferentialSamples int      `json:"differential_samples,omitempty"`
}

type OptimizationMetadataOptions struct {
	PassName                  string
	InputKind                 string
	OutputKind                string
	InputVerifier             string
	OutputVerifier            string
	ValidationStrategy        string
	RequiredFacts             []string
	PreservedFacts            []string
	InvalidatedFacts          []string
	ProofRule                 string
	TranslationValidationHook string
	ReportRows                []string
	NegativeTestMarker        string
	ProfileInputPolicy        string
	ProfileInputDigest        string
	ProfileInputSchemaVersion string
}

type OptimizationValidationMetadata struct {
	SchemaVersion             string            `json:"schema_version"`
	PassName                  string            `json:"pass_name"`
	InputKind                 string            `json:"input_ir_kind"`
	OutputKind                string            `json:"output_ir_kind"`
	InputVerifier             string            `json:"input_verifier"`
	OutputVerifier            string            `json:"output_verifier"`
	ValidationStrategy        string            `json:"validation_strategy"`
	RequiredFacts             []string          `json:"required_facts,omitempty"`
	PreservedFacts            []string          `json:"preserved_facts,omitempty"`
	InvalidatedFacts          []string          `json:"invalidated_facts,omitempty"`
	ProofRule                 string            `json:"proof_rule"`
	TranslationValidationHook string            `json:"translation_validation_hook"`
	ReportRows                []string          `json:"report_rows"`
	NegativeTestMarker        string            `json:"negative_test_marker"`
	ProfileInputPolicy        string            `json:"profile_input_policy"`
	ProfileInputDigest        string            `json:"profile_input_digest,omitempty"`
	ProfileInputSchemaVersion string            `json:"profile_input_schema_version,omitempty"`
	BeforeHash                string            `json:"before_hash"`
	AfterHash                 string            `json:"after_hash"`
	Functions                 []string          `json:"functions"`
	Translation               TranslationReport `json:"translation"`
}

func VerifierMap() []StageVerifier {
	return []StageVerifier{
		{Stage: StageTypedAST, Verifier: "semantics.Check/CheckWorldOpt", Implemented: true},
		{Stage: StagePLIR, Verifier: "plir.VerifyProgram", Implemented: true},
		{Stage: StageProofFacts, Verifier: "validation.CheckBoundsProofsWithPLIR", Implemented: true},
		{Stage: StageOptimizedIR, Verifier: "lower.VerifyProgram", Implemented: true},
		{Stage: StageAllocationPlan, Verifier: "allocplan.VerifyPlan + validation.ValidateAllocationLowering", Implemented: true},
		{Stage: StageMachineIR, Verifier: "machine.VerifyFunction", Implemented: true},
		{Stage: StageABI, Verifier: "x64abi/x86abi classifiers and target tests", Implemented: true},
		{Stage: StageObjectSmoke, Verifier: "target smoke validators", Implemented: true},
	}
}

func CheckBoundsProofs(prog *ir.IRProgram) (ProofReport, error) {
	if prog == nil {
		return ProofReport{}, fmt.Errorf("proof checker: missing IR program")
	}
	report := ProofReport{}
	for _, fn := range prog.Funcs {
		for i, instr := range fn.Instrs {
			switch instr.Kind {
			case ir.IRIndexLoadI32Unchecked, ir.IRIndexLoadU8Unchecked, ir.IRIndexLoadU16Unchecked:
				if instr.ProofID == "" {
					return report, fmt.Errorf("proof checker: %s instruction %d removes bounds check without proof id", fn.Name, i)
				}
				report.RemovedChecks = append(report.RemovedChecks, RemovedCheck{
					Function:  fn.Name,
					Site:      i,
					Kind:      boundsKind(instr.Kind),
					ProofID:   instr.ProofID,
					FactsUsed: []string{"index_in_range", "len_stable"},
				})
			case ir.IRIndexLoadI32, ir.IRIndexLoadU8, ir.IRIndexLoadU16,
				ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
				report.LeftChecks++
			}
		}
	}
	return report, nil
}

func CheckBoundsProofsWithPLIR(prog *ir.IRProgram, proofProg *plir.Program) (ProofReport, error) {
	report, err := CheckBoundsProofs(prog)
	if err != nil {
		return report, err
	}
	if proofProg == nil {
		if len(report.RemovedChecks) == 0 {
			return report, nil
		}
		return report, fmt.Errorf("proof checker: missing PLIR program for removed bounds checks")
	}
	if err := plir.VerifyProgram(proofProg); err != nil {
		return report, fmt.Errorf("proof checker: PLIR proof verification failed: %w", err)
	}
	guards := map[string]map[string]bool{}
	terms := map[string]map[string]plir.ProofTerm{}
	ops := map[string]map[string]plir.Operation{}
	uses := map[string]map[string][]plirProofUseEvidence{}
	for _, fn := range proofProg.Funcs {
		if guards[fn.Name] == nil {
			guards[fn.Name] = map[string]bool{}
		}
		if terms[fn.Name] == nil {
			terms[fn.Name] = map[string]plir.ProofTerm{}
		}
		if ops[fn.Name] == nil {
			ops[fn.Name] = map[string]plir.Operation{}
		}
		if uses[fn.Name] == nil {
			uses[fn.Name] = map[string][]plirProofUseEvidence{}
		}
		for _, guard := range fn.ProofGuards {
			guards[fn.Name][guard.ID] = true
		}
		for _, op := range fn.Ops {
			ops[fn.Name][op.ID] = op
		}
		for _, term := range fn.ProofTerms {
			terms[fn.Name][term.ID] = term
		}
		for _, use := range fn.ProofUses {
			if use.UseKind == "bounds_check" {
				evidence := plirProofUseEvidence{ProofID: use.ProofID, OpID: use.OpID}
				if use.OpID != "" {
					op := ops[fn.Name][use.OpID]
					evidence.OpKind = op.Kind
					evidence.OpNote = op.Note
				}
				uses[fn.Name][use.ProofID] = append(uses[fn.Name][use.ProofID], evidence)
			}
		}
	}
	for i := range report.RemovedChecks {
		removed := &report.RemovedChecks[i]
		if !guards[removed.Function][removed.ProofID] {
			return report, fmt.Errorf("proof checker: %s removed bounds check proof id %q not found in PLIR proof guards", removed.Function, removed.ProofID)
		}
		proofUses := uses[removed.Function][removed.ProofID]
		if len(proofUses) == 0 {
			return report, fmt.Errorf("proof checker: %s removed bounds check proof id %q has no dominated PLIR proof use", removed.Function, removed.ProofID)
		}
		term, ok := terms[removed.Function][removed.ProofID]
		if !ok {
			return report, fmt.Errorf("proof checker: %s removed bounds check proof id %q has no typed proof term", removed.Function, removed.ProofID)
		}
		if want := expectedProofOperationForRemovedCheck(*removed); want != "" && term.Operation != want {
			return report, fmt.Errorf("proof checker: %s removed bounds check proof id %q operation %q does not match %q", removed.Function, removed.ProofID, term.Operation, want)
		}
		if want := expectedProofOperationForRemovedCheck(*removed); want != "" && !proofUsesContainOperation(proofUses, want) {
			return report, fmt.Errorf("proof checker: %s removed bounds check proof id %q proof use operation does not match %q", removed.Function, removed.ProofID, want)
		}
		removed.ProofTerm = validationProofTermFromPLIR(term)
	}
	return report, nil
}

func expectedProofOperationForRemovedCheck(removed RemovedCheck) string {
	switch {
	case strings.HasSuffix(removed.Kind, ".load"):
		return "index_load"
	case strings.HasSuffix(removed.Kind, ".store"):
		return "index_store"
	default:
		return ""
	}
}

func proofUsesContainOperation(uses []plirProofUseEvidence, operation string) bool {
	for _, use := range uses {
		if proofUseOperationMatches(use, operation) {
			return true
		}
	}
	return false
}

func proofUseOperationMatches(use plirProofUseEvidence, operation string) bool {
	switch operation {
	case "index_load":
		switch use.OpKind {
		case plir.OpIndexLoad:
			return true
		case plir.OpForSlice:
			return strings.HasPrefix(use.ProofID, "proof:for-collection")
		case plir.OpAllocIntent, plir.OpCall:
			return strings.HasPrefix(use.ProofID, "proof:copy-loop:") && strings.Contains(use.OpNote, "copy")
		default:
			return false
		}
	case "index_store":
		return use.OpKind == plir.OpIndexStore
	default:
		return false
	}
}

func validationProofTermFromPLIR(term plir.ProofTerm) *ProofTerm {
	return &ProofTerm{
		ID:            term.ID,
		Kind:          term.Kind,
		SubjectBaseID: term.SubjectBaseID,
		IndexValueID:  term.IndexValueID,
		Operation:     term.Operation,
		Range:         term.Range,
		IslandID:      term.IslandID,
		Epoch:         term.Epoch,
		BaseID:        term.BaseID,
		Source:        term.Source,
		FactsUsed:     append([]string(nil), term.FactsUsed...),
	}
}

func ValidateAllocationPlan(plan *allocplan.Plan) error {
	if err := allocplan.VerifyPlan(plan); err != nil {
		return fmt.Errorf("allocation validation: %w", err)
	}
	return nil
}

func ValidateAllocationLowering(plan *allocplan.Plan, prog *ir.IRProgram) error {
	return validateAllocationLowering(plan, prog, prog)
}

func ValidateAllocationLoweringWithSummaryProgram(plan *allocplan.Plan, prog *ir.IRProgram, summaryProg *ir.IRProgram) error {
	return validateAllocationLowering(plan, prog, summaryProg)
}

func validateAllocationLowering(plan *allocplan.Plan, prog *ir.IRProgram, summaryProg *ir.IRProgram) error {
	if err := ValidateAllocationPlan(plan); err != nil {
		return err
	}
	if prog == nil {
		return fmt.Errorf("allocation lowering validation: missing IR program")
	}
	if summaryProg == nil {
		summaryProg = prog
	}
	if prog.MainName == "" {
		for _, fn := range prog.Funcs {
			if err := lower.VerifyFunc(fn); err != nil {
				return fmt.Errorf("allocation lowering validation: IR invalid: %w", err)
			}
		}
	} else if err := lower.VerifyProgram(prog); err != nil {
		return fmt.Errorf("allocation lowering validation: IR invalid: %w", err)
	}
	expected := map[string]map[string]allocplan.StorageClass{}
	stackAllocs := map[string]map[string]bool{}
	expectedRegion := map[string]map[string]ir.IRInstrKind{}
	regionAllocs := map[string]map[string]bool{}
	expectedIsland := map[string]map[string]islandExpectation{}
	islandAllocs := map[string]map[string]bool{}
	for _, fn := range plan.Functions {
		for _, alloc := range fn.Allocations {
			switch alloc.ActualLoweringStorage {
			case allocplan.StorageStack:
				if expected[fn.Name] == nil {
					expected[fn.Name] = map[string]allocplan.StorageClass{}
				}
				expected[fn.Name][alloc.ID] = alloc.ActualLoweringStorage
				if stackAllocs[fn.Name] == nil {
					stackAllocs[fn.Name] = map[string]bool{}
				}
				stackAllocs[fn.Name][alloc.ID] = true
			case allocplan.StorageEliminated:
				if alloc.LoweringStatus == "eliminated_no_backing_storage" {
					if expected[fn.Name] == nil {
						expected[fn.Name] = map[string]allocplan.StorageClass{}
					}
					expected[fn.Name][alloc.ID] = alloc.ActualLoweringStorage
				}
			case allocplan.StorageFunctionTempRegion:
				kind, ok := regionIRKind(alloc)
				if !ok {
					return fmt.Errorf("allocation lowering validation: %s allocation %q reports actual FunctionTempRegion for unsupported builtin %q", fn.Name, alloc.ID, alloc.Builtin)
				}
				if expectedRegion[fn.Name] == nil {
					expectedRegion[fn.Name] = map[string]ir.IRInstrKind{}
				}
				expectedRegion[fn.Name][alloc.ID] = kind
				if regionAllocs[fn.Name] == nil {
					regionAllocs[fn.Name] = map[string]bool{}
				}
				regionAllocs[fn.Name][alloc.ID] = true
			case allocplan.StorageExplicitIsland:
				kind, ok := explicitIslandIRKind(alloc)
				if !ok {
					return fmt.Errorf("allocation lowering validation: %s allocation %q reports actual ExplicitIsland for unsupported builtin %q", fn.Name, alloc.ID, alloc.Builtin)
				}
				if strings.TrimSpace(alloc.RegionID) == "" {
					return fmt.Errorf("allocation lowering validation: %s allocation %q reports actual ExplicitIsland without region id", fn.Name, alloc.ID)
				}
				if strings.TrimSpace(alloc.Lifetime) == "" {
					return fmt.Errorf("allocation lowering validation: %s allocation %q reports actual ExplicitIsland without lifetime", fn.Name, alloc.ID)
				}
				if expectedIsland[fn.Name] == nil {
					expectedIsland[fn.Name] = map[string]islandExpectation{}
				}
				expectedIsland[fn.Name][alloc.ID] = islandExpectation{
					kind:                 kind,
					regionID:             alloc.RegionID,
					lifetime:             alloc.Lifetime,
					handleParamSlotKnown: alloc.ExplicitIslandHandleParamSlotKnown,
					handleParamSlot:      alloc.ExplicitIslandHandleParamSlot,
				}
				if islandAllocs[fn.Name] == nil {
					islandAllocs[fn.Name] = map[string]bool{}
				}
				islandAllocs[fn.Name][alloc.ID] = true
			}
		}
	}
	seen := map[string]map[string]bool{}
	seenRegion := map[string]map[string]bool{}
	seenIsland := map[string]map[string]bool{}
	for _, fn := range prog.Funcs {
		for i, instr := range fn.Instrs {
			switch instr.Kind {
			case ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32:
				want, ok := expected[fn.Name][instr.Name]
				if !ok {
					return fmt.Errorf("allocation lowering validation: %s instruction %d stack-lowers %q without matching allocation plan", fn.Name, i, instr.Name)
				}
				if want == allocplan.StorageEliminated && instr.ArgSlots != 0 {
					return fmt.Errorf("allocation lowering validation: %s eliminated allocation %q has %d stack backing slots", fn.Name, instr.Name, instr.ArgSlots)
				}
				if want == allocplan.StorageStack && instr.ArgSlots <= 0 {
					return fmt.Errorf("allocation lowering validation: %s stack allocation %q has no backing slots", fn.Name, instr.Name)
				}
				if seen[fn.Name] == nil {
					seen[fn.Name] = map[string]bool{}
				}
				seen[fn.Name][instr.Name] = true
			case ir.IRRegionMakeSliceU8, ir.IRRegionMakeSliceU16, ir.IRRegionMakeSliceI32:
				want, ok := expectedRegion[fn.Name][instr.Name]
				if !ok {
					return fmt.Errorf("allocation lowering validation: %s instruction %d emits %s for %q without matching function-temp region allocation plan", fn.Name, i, regionIRKindName(instr.Kind), instr.Name)
				}
				if want != instr.Kind {
					return fmt.Errorf("allocation lowering validation: %s allocation %q reports actual FunctionTempRegion for %s but IR emitted %s", fn.Name, instr.Name, regionIRKindName(want), regionIRKindName(instr.Kind))
				}
				if seenRegion[fn.Name] == nil {
					seenRegion[fn.Name] = map[string]bool{}
				}
				if seenRegion[fn.Name][instr.Name] {
					return fmt.Errorf("allocation lowering validation: %s emits duplicate function-temp region slice for %q", fn.Name, instr.Name)
				}
				seenRegion[fn.Name][instr.Name] = true
			case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
				if strings.TrimSpace(instr.Name) == "" {
					return fmt.Errorf("allocation lowering validation: %s instruction %d emits %s without allocation name; no matching IR island slice", fn.Name, i, islandIRKindName(instr.Kind))
				}
				want, ok := expectedIsland[fn.Name][instr.Name]
				if !ok {
					return fmt.Errorf("allocation lowering validation: %s instruction %d emits %s for %q without matching explicit island allocation plan", fn.Name, i, islandIRKindName(instr.Kind), instr.Name)
				}
				if want.kind != instr.Kind {
					return fmt.Errorf("allocation lowering validation: %s allocation %q reports actual ExplicitIsland for %s but IR emitted %s", fn.Name, instr.Name, islandIRKindName(want.kind), islandIRKindName(instr.Kind))
				}
				if seenIsland[fn.Name] == nil {
					seenIsland[fn.Name] = map[string]bool{}
				}
				if seenIsland[fn.Name][instr.Name] {
					return fmt.Errorf("allocation lowering validation: %s emits duplicate explicit island slice for %q", fn.Name, instr.Name)
				}
				seenIsland[fn.Name][instr.Name] = true
			}
		}
	}
	if err := validateStackAllocationsDoNotEscape(prog, stackAllocs); err != nil {
		return err
	}
	if err := validateStackAllocationsDoNotEscape(prog, regionAllocs); err != nil {
		return err
	}
	if err := validateFunctionTempRegionResets(prog, regionAllocs); err != nil {
		return err
	}
	if err := validateExplicitIslandLifetimes(prog, summaryProg, expectedIsland); err != nil {
		return err
	}
	for fn, allocs := range expected {
		for id, storage := range allocs {
			if !seen[fn][id] {
				return fmt.Errorf("allocation lowering validation: %s allocation %q reports actual %s but no matching IR stack slice was emitted", fn, id, storage)
			}
		}
	}
	for fn, allocs := range expectedRegion {
		for id, kind := range allocs {
			if !seenRegion[fn][id] {
				return fmt.Errorf("allocation lowering validation: %s allocation %q reports actual FunctionTempRegion for %s but no matching IR function-temp region slice was emitted", fn, id, regionIRKindName(kind))
			}
		}
	}
	for fn, allocs := range expectedIsland {
		for id, want := range allocs {
			if !seenIsland[fn][id] {
				return fmt.Errorf("allocation lowering validation: %s allocation %q reports actual ExplicitIsland for %s but no matching IR island slice was emitted", fn, id, islandIRKindName(want.kind))
			}
		}
	}
	return nil
}

type stackEscapeState struct {
	idx    int
	stack  []string
	locals []string
}

type functionTempRegionResetState struct {
	idx     int
	entered bool
	active  bool
}

type islandExpectation struct {
	kind                 ir.IRInstrKind
	regionID             string
	lifetime             string
	handleParamSlotKnown bool
	handleParamSlot      int
}

type islandLifetimeState struct {
	idx    int
	stack  []string
	locals []string
	freed  map[string]bool
}

type islandReturnSummary struct {
	retTags []string
}

func validateStackAllocationsDoNotEscape(prog *ir.IRProgram, stackAllocs map[string]map[string]bool) error {
	for _, fn := range prog.Funcs {
		tracked := stackAllocs[fn.Name]
		if len(tracked) == 0 {
			continue
		}
		if err := validateFunctionStackAllocationsDoNotEscape(fn, tracked); err != nil {
			return err
		}
	}
	return nil
}

func validateFunctionTempRegionResets(prog *ir.IRProgram, regionAllocs map[string]map[string]bool) error {
	for _, fn := range prog.Funcs {
		tracked := regionAllocs[fn.Name]
		if len(tracked) == 0 {
			continue
		}
		if err := validateFunctionTempRegionResetsInFunc(fn, tracked); err != nil {
			return err
		}
	}
	return nil
}

func validateFunctionTempRegionResetsInFunc(fn ir.IRFunc, tracked map[string]bool) error {
	labels := map[int]int{}
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel {
			labels[instr.Label] = i
		}
	}
	work := []functionTempRegionResetState{{idx: 0}}
	seen := map[functionTempRegionResetState]bool{}
	for len(work) > 0 {
		cur := work[len(work)-1]
		work = work[:len(work)-1]
		if cur.idx < 0 || cur.idx >= len(fn.Instrs) {
			if cur.active {
				return fmt.Errorf("allocation lowering validation: %s function-temp region reset does not dominate function exit", fn.Name)
			}
			continue
		}
		if seen[cur] {
			continue
		}
		seen[cur] = true
		next, err := stepFunctionTempRegionResetState(fn, cur, labels, tracked)
		if err != nil {
			return err
		}
		work = append(work, next...)
	}
	return nil
}

func stepFunctionTempRegionResetState(fn ir.IRFunc, cur functionTempRegionResetState, labels map[int]int, tracked map[string]bool) ([]functionTempRegionResetState, error) {
	instr := fn.Instrs[cur.idx]
	entered := cur.entered
	active := cur.active
	switch instr.Kind {
	case ir.IRRegionEnter:
		entered = true
	case ir.IRRegionMakeSliceU8, ir.IRRegionMakeSliceU16, ir.IRRegionMakeSliceI32:
		if tracked[instr.Name] {
			if !entered {
				return nil, fmt.Errorf("allocation lowering validation: %s instruction %d function-temp region enter does not dominate make for %q", fn.Name, cur.idx, instr.Name)
			}
			active = true
		}
	case ir.IRRegionReset:
		active = false
	case ir.IRReturn:
		if active {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d function-temp region reset does not dominate return", fn.Name, cur.idx)
		}
		return nil, nil
	}

	next := functionTempRegionResetState{idx: cur.idx + 1, entered: entered, active: active}
	switch instr.Kind {
	case ir.IRJmp:
		labelIdx, ok := labels[instr.Label]
		if !ok {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d jumps to unknown label %d", fn.Name, cur.idx, instr.Label)
		}
		next.idx = labelIdx
		return []functionTempRegionResetState{next}, nil
	case ir.IRJmpIfZero:
		labelIdx, ok := labels[instr.Label]
		if !ok {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d jumps to unknown label %d", fn.Name, cur.idx, instr.Label)
		}
		branch := next
		branch.idx = labelIdx
		return []functionTempRegionResetState{next, branch}, nil
	default:
		return []functionTempRegionResetState{next}, nil
	}
}

func validateExplicitIslandLifetimes(prog *ir.IRProgram, summaryProg *ir.IRProgram, expected map[string]map[string]islandExpectation) error {
	summaries := inferIslandReturnSummaries(summaryProg)
	for _, fn := range prog.Funcs {
		tracked := expected[fn.Name]
		if len(tracked) == 0 {
			continue
		}
		locals, err := initialExplicitIslandLifetimeLocals(fn, tracked)
		if err != nil {
			return err
		}
		labels := map[int]int{}
		for i, instr := range fn.Instrs {
			if instr.Kind == ir.IRLabel {
				labels[instr.Label] = i
			}
		}
		work := []islandLifetimeState{{
			idx:    0,
			locals: locals,
			freed:  map[string]bool{},
		}}
		seen := map[string]bool{}
		for len(work) > 0 {
			cur := work[len(work)-1]
			work = work[:len(work)-1]
			if cur.idx < 0 || cur.idx >= len(fn.Instrs) {
				continue
			}
			key := islandLifetimeStateKey(cur)
			if seen[key] {
				continue
			}
			seen[key] = true
			next, err := stepExplicitIslandLifetimeState(fn, cur, labels, tracked, summaries)
			if err != nil {
				return err
			}
			work = append(work, next...)
		}
	}
	return nil
}

func initialExplicitIslandLifetimeLocals(fn ir.IRFunc, tracked map[string]islandExpectation) ([]string, error) {
	locals := make([]string, fn.LocalSlots)
	for alloc, expectation := range tracked {
		if !expectation.handleParamSlotKnown {
			continue
		}
		slot := expectation.handleParamSlot
		if slot < 0 || slot >= fn.ParamSlots || slot >= len(locals) {
			return nil, fmt.Errorf("allocation lowering validation: %s allocation %q explicit island handle parameter slot %d is outside params=%d locals=%d", fn.Name, alloc, slot, fn.ParamSlots, fn.LocalSlots)
		}
		locals[slot] = islandParamTag(slot)
	}
	return locals, nil
}

func stepExplicitIslandLifetimeState(fn ir.IRFunc, cur islandLifetimeState, labels map[int]int, expected map[string]islandExpectation, summaries map[string]islandReturnSummary) ([]islandLifetimeState, error) {
	instr := fn.Instrs[cur.idx]
	stack := append([]string(nil), cur.stack...)
	locals := append([]string(nil), cur.locals...)
	freed := cloneBoolMap(cur.freed)
	pop, push, ok := validationStackEffect(instr)
	if !ok {
		return nil, fmt.Errorf("allocation lowering validation: %s instruction %d has unknown IR kind %d", fn.Name, cur.idx, instr.Kind)
	}

	switch instr.Kind {
	case ir.IRReturn:
		if name := firstEscapingTrackedExplicitIslandSlice(stack, expected); name != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island allocation %q escapes via return", fn.Name, cur.idx, name)
		}
		if tag := firstFreedIslandUse(stack, freed); tag != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island use after free via return of %s", fn.Name, cur.idx, tag)
		}
		return nil, nil
	case ir.IRIslandNew:
		_, rest := popStackTags(stack, pop)
		stack = append(rest, explicitIslandHandleTag(fn.Name, cur.idx))
	case ir.IRIslandFree:
		popped, rest := popStackTags(stack, pop)
		tag := firstExplicitIslandHandleTag(popped)
		if tag == "" {
			stack = rest
			break
		}
		if freed[tag] {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island double free for %s", fn.Name, cur.idx, tag)
		}
		freed[tag] = true
		stack = rest
	case ir.IRIslandReset:
		popped, rest := popStackTags(stack, pop)
		tag := firstExplicitIslandHandleTag(popped)
		if tag == "" {
			stack = pushEmptyTags(rest, push)
			break
		}
		if freed[tag] {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island reset after free for %s", fn.Name, cur.idx, tag)
		}
		if live := firstLiveIslandSliceForHandle(stack, locals, tag); live != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island reset while live slice %s still references %s", fn.Name, cur.idx, live, tag)
		}
		freed[tag] = true
		stack = append(rest, explicitIslandHandleTag(fn.Name, cur.idx))
	case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
		popped, rest := popStackTags(stack, pop)
		if _, ok := expected[instr.Name]; ok {
			if tag := firstFreedIslandUse(popped, freed); tag != "" {
				return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island allocation %q use after free via operands of %s", fn.Name, cur.idx, instr.Name, tag)
			}
			tag := explicitIslandMakeHandleOperandTag(popped)
			if tag == "" {
				return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island allocation %q has no active island handle operand", fn.Name, cur.idx, instr.Name)
			}
			if freed[tag] {
				return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island allocation %q use after free for %s", fn.Name, cur.idx, instr.Name, tag)
			}
			stack = append(rest, explicitIslandSliceTag(instr.Name, tag), "")
		} else {
			stack = pushEmptyTags(rest, push)
		}
	case ir.IRStoreLocal:
		popped, rest := popStackTags(stack, pop)
		stack = rest
		if instr.Local >= 0 && instr.Local < len(locals) {
			locals[instr.Local] = firstExplicitIslandValueTag(popped)
		}
	case ir.IRLoadLocal:
		if instr.Local >= 0 && instr.Local < len(locals) {
			stack = append(stack, locals[instr.Local])
		} else {
			stack = append(stack, "")
		}
	case ir.IRSliceWindow, ir.IRSlicePrefix, ir.IRSliceSuffix:
		popped, rest := popStackTags(stack, pop)
		if tag := firstFreedIslandUse(popped, freed); tag != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island use after free via slice view of %s", fn.Name, cur.idx, tag)
		}
		stack = append(rest, firstExplicitIslandValueTag(popped), "")
	case ir.IRCall:
		popped, rest := popStackTags(stack, pop)
		if tag := firstFreedIslandUse(popped, freed); tag != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island use after free via %s", fn.Name, cur.idx, tag)
		}
		stack = append(rest, islandCallReturnTags(instr, popped, summaries)...)
	case ir.IRIndexLoadI32, ir.IRIndexLoadU8, ir.IRIndexLoadU16,
		ir.IRIndexLoadI32Unchecked, ir.IRIndexLoadU8Unchecked, ir.IRIndexLoadU16Unchecked,
		ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16,
		ir.IRStoreGlobal,
		ir.IRMemReadI32, ir.IRMemReadU8, ir.IRMemReadPtr,
		ir.IRMemWriteI32, ir.IRMemWriteU8, ir.IRMemWritePtr, ir.IRMemWriteArchPtr,
		ir.IRMemReadI32Offset, ir.IRMemReadU8Offset, ir.IRMemReadPtrOffset,
		ir.IRMemWriteI32Offset, ir.IRMemWriteU8Offset, ir.IRMemWritePtrOffset, ir.IRMemWriteArchPtrOffset:
		popped, rest := popStackTags(stack, pop)
		if tag := firstFreedIslandUse(popped, freed); tag != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island use after free via %s", fn.Name, cur.idx, tag)
		}
		stack = pushEmptyTags(rest, push)
	default:
		popped, rest := popStackTags(stack, pop)
		if tag := firstFreedIslandUse(popped, freed); tag != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island use after free via %s", fn.Name, cur.idx, tag)
		}
		stack = pushEmptyTags(rest, push)
	}

	next := islandLifetimeState{idx: cur.idx + 1, stack: stack, locals: locals, freed: freed}
	switch instr.Kind {
	case ir.IRJmp:
		next.idx = labels[instr.Label]
		return []islandLifetimeState{next}, nil
	case ir.IRJmpIfZero:
		branch := cloneIslandLifetimeState(next)
		branch.idx = labels[instr.Label]
		return []islandLifetimeState{next, branch}, nil
	default:
		return []islandLifetimeState{next}, nil
	}
}

func inferIslandReturnSummaries(prog *ir.IRProgram) map[string]islandReturnSummary {
	summaries := map[string]islandReturnSummary{}
	for _, fn := range prog.Funcs {
		summary, ok := inferIslandReturnSummary(fn)
		if ok {
			summaries[fn.Name] = summary
		}
	}
	return summaries
}

func inferIslandReturnSummary(fn ir.IRFunc) (islandReturnSummary, bool) {
	if fn.ParamSlots <= 0 || fn.ReturnSlots <= 0 || fn.LocalSlots <= 0 {
		return islandReturnSummary{}, false
	}
	labels := map[int]int{}
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel {
			labels[instr.Label] = i
		}
	}
	locals := make([]string, fn.LocalSlots)
	for i := 0; i < fn.ParamSlots && i < len(locals); i++ {
		locals[i] = islandParamTag(i)
	}
	work := []stackEscapeState{{idx: 0, locals: locals}}
	seen := map[string]bool{}
	var merged []string
	for len(work) > 0 {
		cur := work[len(work)-1]
		work = work[:len(work)-1]
		if cur.idx < 0 || cur.idx >= len(fn.Instrs) {
			continue
		}
		key := stackEscapeStateKey(cur)
		if seen[key] {
			continue
		}
		seen[key] = true
		next, ret, ok := stepIslandReturnSummaryState(fn, cur, labels)
		if !ok {
			return islandReturnSummary{}, false
		}
		if ret != nil {
			merged = mergeIslandReturnSources(merged, ret)
			continue
		}
		work = append(work, next...)
	}
	if merged == nil {
		return islandReturnSummary{}, false
	}
	return islandReturnSummary{retTags: merged}, true
}

func stepIslandReturnSummaryState(fn ir.IRFunc, cur stackEscapeState, labels map[int]int) ([]stackEscapeState, []string, bool) {
	instr := fn.Instrs[cur.idx]
	stack := append([]string(nil), cur.stack...)
	locals := append([]string(nil), cur.locals...)
	pop, push, ok := validationStackEffect(instr)
	if !ok {
		return nil, nil, false
	}

	switch instr.Kind {
	case ir.IRReturn:
		return nil, islandReturnTags(stack, fn.ReturnSlots), true
	case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
		popped, rest := popStackTags(stack, pop)
		if tag := explicitIslandMakeHandleOperandTag(popped); strings.HasPrefix(tag, "island-param:") {
			stack = append(rest, explicitIslandSliceTag(instr.Name, tag), "")
		} else {
			stack = pushEmptyTags(rest, push)
		}
	case ir.IRStoreLocal:
		popped, rest := popStackTags(stack, pop)
		stack = rest
		if instr.Local >= 0 && instr.Local < len(locals) {
			locals[instr.Local] = firstIslandParamTag(popped)
		}
	case ir.IRLoadLocal:
		if instr.Local >= 0 && instr.Local < len(locals) {
			stack = append(stack, locals[instr.Local])
		} else {
			stack = append(stack, "")
		}
	default:
		_, rest := popStackTags(stack, pop)
		stack = pushEmptyTags(rest, push)
	}

	next := stackEscapeState{idx: cur.idx + 1, stack: stack, locals: locals}
	switch instr.Kind {
	case ir.IRJmp:
		next.idx = labels[instr.Label]
		return []stackEscapeState{next}, nil, true
	case ir.IRJmpIfZero:
		branch := cloneStackEscapeState(next)
		branch.idx = labels[instr.Label]
		return []stackEscapeState{next, branch}, nil, true
	default:
		return []stackEscapeState{next}, nil, true
	}
}

func islandReturnTags(stack []string, slots int) []string {
	tags := make([]string, slots)
	if slots <= 0 || len(stack) < slots {
		return tags
	}
	start := len(stack) - slots
	for i := 0; i < slots; i++ {
		if islandParamSlotFromValueTag(stack[start+i]) >= 0 {
			tags[i] = stack[start+i]
		}
	}
	return tags
}

func mergeIslandReturnSources(merged []string, ret []string) []string {
	if merged == nil {
		return append([]string(nil), ret...)
	}
	for i := range merged {
		if i >= len(ret) || merged[i] != ret[i] {
			merged[i] = ""
		}
	}
	return merged
}

func islandCallReturnTags(instr ir.IRInstr, popped []string, summaries map[string]islandReturnSummary) []string {
	tags := make([]string, instr.RetSlots)
	summary, ok := summaries[instr.Name]
	if !ok {
		return tags
	}
	for i := range tags {
		if i >= len(summary.retTags) {
			continue
		}
		tags[i] = substituteIslandSummaryTag(summary.retTags[i], instr.ArgSlots, popped)
	}
	return tags
}

func substituteIslandSummaryTag(tag string, argSlots int, popped []string) string {
	if tag == "" {
		return ""
	}
	if slot := islandParamSlot(tag); slot >= 0 {
		return islandCallArgTag(slot, argSlots, popped)
	}
	name, handleTag, ok := islandSliceTagParts(tag)
	if !ok {
		return ""
	}
	slot := islandParamSlot(handleTag)
	if slot < 0 {
		return ""
	}
	replacement := islandCallArgTag(slot, argSlots, popped)
	if replacement == "" {
		return ""
	}
	return explicitIslandSliceTag(name, replacement)
}

func islandCallArgTag(slot int, argSlots int, popped []string) string {
	if slot < 0 || slot >= argSlots || slot >= len(popped) {
		return ""
	}
	return popped[len(popped)-1-slot]
}

func islandParamTag(slot int) string {
	return fmt.Sprintf("island-param:%d", slot)
}

func islandParamSlot(tag string) int {
	const prefix = "island-param:"
	if !strings.HasPrefix(tag, prefix) {
		return -1
	}
	slot, err := strconv.Atoi(strings.TrimPrefix(tag, prefix))
	if err != nil {
		return -1
	}
	return slot
}

func firstIslandParamTag(tags []string) string {
	for _, tag := range tags {
		if islandParamSlotFromValueTag(tag) >= 0 {
			return tag
		}
	}
	return ""
}

func islandParamSlotFromValueTag(tag string) int {
	if slot := islandParamSlot(tag); slot >= 0 {
		return slot
	}
	_, handleTag, ok := islandSliceTagParts(tag)
	if !ok {
		return -1
	}
	return islandParamSlot(handleTag)
}

func explicitIslandHandleTag(function string, idx int) string {
	return fmt.Sprintf("island:%s:%d", function, idx)
}

func explicitIslandSliceTag(name string, islandTag string) string {
	return "island-slice:" + name + "@" + islandTag
}

func firstExplicitIslandHandleTag(tags []string) string {
	for _, tag := range tags {
		if strings.HasPrefix(tag, "island:") || strings.HasPrefix(tag, "island-param:") {
			return tag
		}
	}
	return ""
}

func explicitIslandMakeHandleOperandTag(popped []string) string {
	if len(popped) < 2 {
		return ""
	}
	tag := popped[1]
	if strings.HasPrefix(tag, "island:") || strings.HasPrefix(tag, "island-param:") {
		return tag
	}
	return ""
}

func firstExplicitIslandValueTag(tags []string) string {
	for _, tag := range tags {
		if strings.HasPrefix(tag, "island:") || strings.HasPrefix(tag, "island-param:") || strings.HasPrefix(tag, "island-slice:") {
			return tag
		}
	}
	return ""
}

func firstFreedIslandUse(tags []string, freed map[string]bool) string {
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if strings.HasPrefix(tag, "island:") || strings.HasPrefix(tag, "island-param:") {
			if freed[tag] {
				return tag
			}
			continue
		}
		if _, islandTag, ok := islandSliceTagParts(tag); ok {
			if freed[islandTag] {
				return tag
			}
		}
	}
	return ""
}

func firstEscapingTrackedExplicitIslandSlice(tags []string, expected map[string]islandExpectation) string {
	for _, tag := range tags {
		name, handleTag, ok := islandSliceTagParts(tag)
		if !ok {
			continue
		}
		expectation, ok := expected[name]
		if !ok {
			continue
		}
		if expectation.handleParamSlotKnown && islandParamSlot(handleTag) == expectation.handleParamSlot {
			continue
		}
		return name
	}
	return ""
}

func firstLiveIslandSliceForHandle(stack []string, locals []string, handleTag string) string {
	for _, tags := range [][]string{stack, locals} {
		for _, tag := range tags {
			name, tagHandle, ok := islandSliceTagParts(tag)
			if ok && tagHandle == handleTag {
				return name
			}
		}
	}
	return ""
}

func islandSliceTagParts(tag string) (name string, handleTag string, ok bool) {
	const prefix = "island-slice:"
	if !strings.HasPrefix(tag, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(tag, prefix)
	at := strings.LastIndex(rest, "@")
	if at < 0 || at == 0 || at+1 >= len(rest) {
		return "", "", false
	}
	return rest[:at], rest[at+1:], true
}

func cloneBoolMap(in map[string]bool) map[string]bool {
	out := map[string]bool{}
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneIslandLifetimeState(state islandLifetimeState) islandLifetimeState {
	return islandLifetimeState{
		idx:    state.idx,
		stack:  append([]string(nil), state.stack...),
		locals: append([]string(nil), state.locals...),
		freed:  cloneBoolMap(state.freed),
	}
}

func islandLifetimeStateKey(state islandLifetimeState) string {
	var b strings.Builder
	b.WriteString(strconv.Itoa(state.idx))
	b.WriteByte('|')
	for _, tag := range state.stack {
		b.WriteString(tag)
		b.WriteByte(',')
	}
	b.WriteByte('|')
	for i, tag := range state.locals {
		if tag == "" {
			continue
		}
		b.WriteString(strconv.Itoa(i))
		b.WriteByte('=')
		b.WriteString(tag)
		b.WriteByte(',')
	}
	b.WriteByte('|')
	freed := make([]string, 0, len(state.freed))
	for tag, value := range state.freed {
		if value {
			freed = append(freed, tag)
		}
	}
	sort.Strings(freed)
	for _, tag := range freed {
		b.WriteString(tag)
		b.WriteByte(',')
	}
	return b.String()
}

func validateFunctionStackAllocationsDoNotEscape(fn ir.IRFunc, tracked map[string]bool) error {
	labels := map[int]int{}
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel {
			labels[instr.Label] = i
		}
	}
	work := []stackEscapeState{{
		idx:    0,
		locals: make([]string, fn.LocalSlots),
	}}
	seen := map[string]bool{}
	for len(work) > 0 {
		cur := work[len(work)-1]
		work = work[:len(work)-1]
		if cur.idx < 0 || cur.idx >= len(fn.Instrs) {
			continue
		}
		key := stackEscapeStateKey(cur)
		if seen[key] {
			continue
		}
		seen[key] = true
		next, err := stepStackEscapeState(fn, cur, labels, tracked)
		if err != nil {
			return err
		}
		work = append(work, next...)
	}
	return nil
}

func stepStackEscapeState(fn ir.IRFunc, cur stackEscapeState, labels map[int]int, tracked map[string]bool) ([]stackEscapeState, error) {
	instr := fn.Instrs[cur.idx]
	stack := append([]string(nil), cur.stack...)
	locals := append([]string(nil), cur.locals...)
	pop, push, ok := validationStackEffect(instr)
	if !ok {
		return nil, fmt.Errorf("allocation lowering validation: %s instruction %d has unknown IR kind %d", fn.Name, cur.idx, instr.Kind)
	}

	switch instr.Kind {
	case ir.IRReturn:
		if tag := firstStackTag(stack); tag != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d stack allocation %q escapes via return", fn.Name, cur.idx, tag)
		}
		return nil, nil
	case ir.IRCall:
		popped, rest := popStackTags(stack, pop)
		if tag := firstStackTag(popped); tag != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d stack allocation %q escapes via call %q", fn.Name, cur.idx, tag, instr.Name)
		}
		stack = pushEmptyTags(rest, push)
	case ir.IRStoreGlobal:
		popped, rest := popStackTags(stack, pop)
		if tag := firstStackTag(popped); tag != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d stack allocation %q escapes via global store", fn.Name, cur.idx, tag)
		}
		stack = rest
	case ir.IRStoreLocal:
		popped, rest := popStackTags(stack, pop)
		stack = rest
		if instr.Local >= 0 && instr.Local < len(locals) {
			locals[instr.Local] = firstStackTag(popped)
		}
	case ir.IRLoadLocal:
		if instr.Local >= 0 && instr.Local < len(locals) {
			stack = append(stack, locals[instr.Local])
		} else {
			stack = append(stack, "")
		}
	case ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32,
		ir.IRRegionMakeSliceU8, ir.IRRegionMakeSliceU16, ir.IRRegionMakeSliceI32:
		_, rest := popStackTags(stack, pop)
		if tracked[instr.Name] {
			stack = append(rest, instr.Name, "")
		} else {
			stack = pushEmptyTags(rest, push)
		}
	case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
		_, rest := popStackTags(stack, pop)
		if tracked[instr.Name] {
			stack = append(rest, instr.Name, "")
		} else {
			stack = pushEmptyTags(rest, push)
		}
	case ir.IRRawSliceFromParts, ir.IRSliceWindow, ir.IRSlicePrefix, ir.IRSliceSuffix:
		popped, rest := popStackTags(stack, pop)
		tag := firstStackTag(popped)
		stack = append(rest, tag, "")
	case ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
		popped, rest := popStackTags(stack, pop)
		if len(popped) > 0 && popped[0] != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d stack allocation %q escapes via indexed store value", fn.Name, cur.idx, popped[0])
		}
		stack = rest
	case ir.IRMemWriteI32, ir.IRMemWriteU8, ir.IRMemWritePtr, ir.IRMemWriteArchPtr,
		ir.IRMemWriteI32Offset, ir.IRMemWriteU8Offset, ir.IRMemWritePtrOffset, ir.IRMemWriteArchPtrOffset,
		ir.IRMmioWriteI32, ir.IRAtomicStorePtr, ir.IRAtomicExchangePtr,
		ir.IRAtomicFetchAddPtr, ir.IRAtomicFetchSubPtr, ir.IRAtomicFetchAndPtr,
		ir.IRAtomicFetchOrPtr, ir.IRAtomicFetchXorPtr, ir.IRAtomicStoreI32,
		ir.IRAtomicExchangeI32, ir.IRAtomicFetchAddI32, ir.IRAtomicFetchSubI32,
		ir.IRAtomicFetchAndI32, ir.IRAtomicFetchOrI32, ir.IRAtomicFetchXorI32,
		ir.IRAtomicStoreI64, ir.IRAtomicExchangeI64, ir.IRAtomicFetchAddI64,
		ir.IRAtomicFetchSubI64, ir.IRAtomicFetchAndI64, ir.IRAtomicFetchOrI64,
		ir.IRAtomicFetchXorI64, ir.IRAtomicStoreI8, ir.IRAtomicExchangeI8,
		ir.IRAtomicFetchAddI8, ir.IRAtomicFetchSubI8, ir.IRAtomicFetchAndI8,
		ir.IRAtomicFetchOrI8, ir.IRAtomicFetchXorI8, ir.IRAtomicStoreI16,
		ir.IRAtomicExchangeI16, ir.IRAtomicFetchAddI16, ir.IRAtomicFetchSubI16,
		ir.IRAtomicFetchAndI16, ir.IRAtomicFetchOrI16, ir.IRAtomicFetchXorI16,
		ir.IRAtomicCompareExchangePtr, ir.IRAtomicCompareExchangeI32,
		ir.IRAtomicCompareExchangeI64, ir.IRAtomicCompareExchangeI8,
		ir.IRAtomicCompareExchangeI16:
		popped, rest := popStackTags(stack, pop)
		if len(popped) > 0 && popped[0] != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d stack allocation %q escapes via memory store value", fn.Name, cur.idx, popped[0])
		}
		stack = pushEmptyTags(rest, push)
	case ir.IRPtrAdd:
		popped, rest := popStackTags(stack, pop)
		tag := firstStackTag(popped)
		stack = append(rest, tag)
	default:
		_, rest := popStackTags(stack, pop)
		stack = pushEmptyTags(rest, push)
	}

	next := stackEscapeState{idx: cur.idx + 1, stack: stack, locals: locals}
	switch instr.Kind {
	case ir.IRJmp:
		next.idx = labels[instr.Label]
		return []stackEscapeState{next}, nil
	case ir.IRJmpIfZero:
		branch := cloneStackEscapeState(next)
		branch.idx = labels[instr.Label]
		return []stackEscapeState{next, branch}, nil
	default:
		return []stackEscapeState{next}, nil
	}
}

func popStackTags(stack []string, count int) ([]string, []string) {
	if count <= 0 {
		return nil, stack
	}
	if count > len(stack) {
		count = len(stack)
	}
	popped := make([]string, count)
	for i := 0; i < count; i++ {
		popped[i] = stack[len(stack)-1-i]
	}
	return popped, stack[:len(stack)-count]
}

func pushEmptyTags(stack []string, count int) []string {
	for i := 0; i < count; i++ {
		stack = append(stack, "")
	}
	return stack
}

func firstStackTag(tags []string) string {
	for _, tag := range tags {
		if tag != "" {
			return tag
		}
	}
	return ""
}

func cloneStackEscapeState(state stackEscapeState) stackEscapeState {
	return stackEscapeState{
		idx:    state.idx,
		stack:  append([]string(nil), state.stack...),
		locals: append([]string(nil), state.locals...),
	}
}

func stackEscapeStateKey(state stackEscapeState) string {
	var b strings.Builder
	b.WriteString(strconv.Itoa(state.idx))
	b.WriteByte('|')
	for _, tag := range state.stack {
		b.WriteString(tag)
		b.WriteByte(',')
	}
	b.WriteByte('|')
	for i, tag := range state.locals {
		if tag == "" {
			continue
		}
		b.WriteString(strconv.Itoa(i))
		b.WriteByte('=')
		b.WriteString(tag)
		b.WriteByte(',')
	}
	return b.String()
}

func validationStackEffect(instr ir.IRInstr) (pop int, push int, known bool) {
	switch instr.Kind {
	case ir.IRWrite:
		return 2, 0, true
	case ir.IRStrLit:
		return 0, 2, true
	case ir.IRConstI32, ir.IRLoadLocal, ir.IRLoadGlobal:
		return 0, 1, true
	case ir.IRStoreLocal, ir.IRStoreGlobal:
		return 1, 0, true
	case ir.IRAddI32, ir.IRSubI32, ir.IRCmpEqI32, ir.IRCmpLtI32,
		ir.IRMulI32, ir.IRDivI32, ir.IRModI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return 2, 1, true
	case ir.IRNegI32:
		return 1, 1, true
	case ir.IRCall:
		return instr.ArgSlots, instr.RetSlots, true
	case ir.IRLabel, ir.IRJmp:
		return 0, 0, true
	case ir.IRJmpIfZero:
		return 1, 0, true
	case ir.IRReturn:
		return 0, 0, true
	case ir.IRAllocBytes, ir.IRIslandNew:
		return 1, 1, true
	case ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32,
		ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32,
		ir.IRRegionMakeSliceU8, ir.IRRegionMakeSliceU16, ir.IRRegionMakeSliceI32:
		return 1, 2, true
	case ir.IRRegionEnter, ir.IRRegionReset:
		return 0, 0, true
	case ir.IRRawSliceFromParts:
		return 3, 2, true
	case ir.IRSliceWindow:
		return 4, 2, true
	case ir.IRSlicePrefix, ir.IRSliceSuffix:
		return 3, 2, true
	case ir.IRIndexLoadI32, ir.IRIndexLoadU8, ir.IRIndexLoadU16,
		ir.IRIndexLoadI32Unchecked, ir.IRIndexLoadU8Unchecked, ir.IRIndexLoadU16Unchecked:
		return 3, 1, true
	case ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
		return 4, 0, true
	case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
		return 2, 2, true
	case ir.IRIslandReset:
		return 1, 1, true
	case ir.IRIslandFree:
		return 1, 0, true
	case ir.IRCapIO, ir.IRCapMem, ir.IRSymAddr:
		return 0, 1, true
	case ir.IRMemReadI32, ir.IRMemReadU8, ir.IRMemReadPtr, ir.IRMmioReadI32,
		ir.IRAtomicLoadPtr, ir.IRAtomicLoadI32, ir.IRAtomicLoadI64,
		ir.IRAtomicLoadI8, ir.IRAtomicLoadI16:
		return 2, 1, true
	case ir.IRMemWriteI32, ir.IRMemWriteU8, ir.IRMemWritePtr, ir.IRMemWriteArchPtr, ir.IRPtrAdd,
		ir.IRMmioWriteI32, ir.IRCtxSwitch, ir.IRAtomicStorePtr,
		ir.IRAtomicExchangePtr, ir.IRAtomicFetchAddPtr, ir.IRAtomicFetchSubPtr,
		ir.IRAtomicFetchAndPtr, ir.IRAtomicFetchOrPtr, ir.IRAtomicFetchXorPtr,
		ir.IRAtomicStoreI32, ir.IRAtomicExchangeI32, ir.IRAtomicFetchAddI32,
		ir.IRAtomicFetchSubI32, ir.IRAtomicFetchAndI32, ir.IRAtomicFetchOrI32,
		ir.IRAtomicFetchXorI32, ir.IRAtomicStoreI64, ir.IRAtomicExchangeI64,
		ir.IRAtomicFetchAddI64, ir.IRAtomicFetchSubI64, ir.IRAtomicFetchAndI64,
		ir.IRAtomicFetchOrI64, ir.IRAtomicFetchXorI64, ir.IRAtomicStoreI8,
		ir.IRAtomicExchangeI8, ir.IRAtomicFetchAddI8, ir.IRAtomicFetchSubI8,
		ir.IRAtomicFetchAndI8, ir.IRAtomicFetchOrI8, ir.IRAtomicFetchXorI8,
		ir.IRAtomicStoreI16, ir.IRAtomicExchangeI16, ir.IRAtomicFetchAddI16,
		ir.IRAtomicFetchSubI16, ir.IRAtomicFetchAndI16, ir.IRAtomicFetchOrI16,
		ir.IRAtomicFetchXorI16:
		return 3, 1, true
	case ir.IRAtomicCompareExchangePtr, ir.IRAtomicCompareExchangeI32, ir.IRAtomicCompareExchangeI64,
		ir.IRAtomicCompareExchangeI8, ir.IRAtomicCompareExchangeI16:
		return 4, 1, true
	case ir.IRMemReadI32Offset, ir.IRMemReadU8Offset, ir.IRMemReadPtrOffset:
		return 3, 1, true
	case ir.IRMemWriteI32Offset, ir.IRMemWriteU8Offset, ir.IRMemWritePtrOffset, ir.IRMemWriteArchPtrOffset:
		return 4, 1, true
	case ir.IRAtomicFenceSeqCst, ir.IRAtomicFenceRelaxed, ir.IRAtomicFenceAcquire,
		ir.IRAtomicFenceRelease, ir.IRAtomicFenceAcqRel:
		return 0, 0, true
	default:
		return 0, 0, false
	}
}

func explicitIslandIRKind(alloc allocplan.Allocation) (ir.IRInstrKind, bool) {
	switch alloc.Builtin {
	case "core.island_make_u8":
		return ir.IRIslandMakeSliceU8, true
	case "core.island_make_u16":
		return ir.IRIslandMakeSliceU16, true
	case "core.island_make_i32", "core.island_make_bool":
		return ir.IRIslandMakeSliceI32, true
	default:
		return 0, false
	}
}

func islandIRKindName(kind ir.IRInstrKind) string {
	switch kind {
	case ir.IRIslandMakeSliceU8:
		return "IRIslandMakeSliceU8"
	case ir.IRIslandMakeSliceU16:
		return "IRIslandMakeSliceU16"
	case ir.IRIslandMakeSliceI32:
		return "IRIslandMakeSliceI32"
	case ir.IRIslandReset:
		return "IRIslandReset"
	default:
		return fmt.Sprintf("ir.%d", kind)
	}
}

func regionIRKind(alloc allocplan.Allocation) (ir.IRInstrKind, bool) {
	switch alloc.Builtin {
	case "core.slice_copy_u8", "core.string_copy":
		return ir.IRRegionMakeSliceU8, true
	case "core.slice_copy_u16":
		return ir.IRRegionMakeSliceU16, true
	case "core.slice_copy_i32", "core.slice_copy_bool":
		return ir.IRRegionMakeSliceI32, true
	default:
		return 0, false
	}
}

func regionIRKindName(kind ir.IRInstrKind) string {
	switch kind {
	case ir.IRRegionMakeSliceU8:
		return "IRRegionMakeSliceU8"
	case ir.IRRegionMakeSliceU16:
		return "IRRegionMakeSliceU16"
	case ir.IRRegionMakeSliceI32:
		return "IRRegionMakeSliceI32"
	default:
		return fmt.Sprintf("ir.%d", kind)
	}
}

func ValidateTranslation(before *ir.IRProgram, after *ir.IRProgram) (TranslationReport, error) {
	if err := lower.VerifyProgram(before); err != nil {
		return TranslationReport{}, fmt.Errorf("translation validation: input IR invalid: %w", err)
	}
	if err := lower.VerifyProgram(after); err != nil {
		return TranslationReport{}, fmt.Errorf("translation validation: output IR invalid: %w", err)
	}
	beforeNames := functionNames(before)
	afterNames := functionNames(after)
	if len(beforeNames) != len(afterNames) {
		return TranslationReport{}, fmt.Errorf("translation validation: function count changed from %d to %d", len(beforeNames), len(afterNames))
	}
	for i := range beforeNames {
		if beforeNames[i] != afterNames[i] {
			return TranslationReport{}, fmt.Errorf("translation validation: function set changed: before=%v after=%v", beforeNames, afterNames)
		}
	}
	beforeFuncs := functionsByName(before)
	afterFuncs := functionsByName(after)
	for _, name := range beforeNames {
		if err := validateTranslationFunctionShape(beforeFuncs[name], afterFuncs[name]); err != nil {
			return TranslationReport{}, err
		}
	}
	beforeProofs, err := CheckBoundsProofs(before)
	if err != nil {
		return TranslationReport{}, fmt.Errorf("translation validation: input proof validation failed: %w", err)
	}
	afterProofs, err := CheckBoundsProofs(after)
	if err != nil {
		return TranslationReport{}, fmt.Errorf("translation validation: output proof validation failed: %w", err)
	}
	proofFactsCompared, err := validateProofFactMultiset(beforeProofs, afterProofs)
	if err != nil {
		return TranslationReport{}, err
	}
	semanticChecks, err := validateSemanticLocalEquivalence(beforeFuncs, afterFuncs, beforeNames)
	if err != nil {
		return TranslationReport{}, err
	}
	differentialSamples, err := validateDifferentialSamples(beforeFuncs, afterFuncs, beforeNames)
	if err != nil {
		return TranslationReport{}, err
	}
	return TranslationReport{
		FunctionsCompared:   len(beforeNames),
		Functions:           beforeNames,
		ProofFactsCompared:  proofFactsCompared,
		SemanticLocalChecks: semanticChecks,
		DifferentialSamples: differentialSamples,
	}, nil
}

func BuildOptimizationValidationMetadata(before *ir.IRProgram, after *ir.IRProgram, options OptimizationMetadataOptions) (OptimizationValidationMetadata, error) {
	report, err := ValidateTranslation(before, after)
	if err != nil {
		return OptimizationValidationMetadata{}, err
	}
	meta := OptimizationValidationMetadata{
		SchemaVersion:             "tetra.translation.validation.metadata.v1",
		PassName:                  options.PassName,
		InputKind:                 options.InputKind,
		OutputKind:                options.OutputKind,
		InputVerifier:             options.InputVerifier,
		OutputVerifier:            options.OutputVerifier,
		ValidationStrategy:        options.ValidationStrategy,
		RequiredFacts:             append([]string(nil), options.RequiredFacts...),
		PreservedFacts:            append([]string(nil), options.PreservedFacts...),
		InvalidatedFacts:          append([]string(nil), options.InvalidatedFacts...),
		ProofRule:                 options.ProofRule,
		TranslationValidationHook: options.TranslationValidationHook,
		ReportRows:                append([]string(nil), options.ReportRows...),
		NegativeTestMarker:        options.NegativeTestMarker,
		ProfileInputPolicy:        options.ProfileInputPolicy,
		ProfileInputDigest:        options.ProfileInputDigest,
		ProfileInputSchemaVersion: options.ProfileInputSchemaVersion,
		BeforeHash:                stableIRHash(before),
		AfterHash:                 stableIRHash(after),
		Functions:                 append([]string(nil), report.Functions...),
		Translation:               report,
	}
	if err := ValidateOptimizationValidationMetadata(meta); err != nil {
		return OptimizationValidationMetadata{}, err
	}
	return meta, nil
}

func ValidateOptimizationValidationMetadata(meta OptimizationValidationMetadata) error {
	if meta.SchemaVersion != "tetra.translation.validation.metadata.v1" {
		return fmt.Errorf("translation validation metadata: schema_version is %q", meta.SchemaVersion)
	}
	if strings.TrimSpace(meta.PassName) == "" {
		return fmt.Errorf("translation validation metadata: missing pass_name")
	}
	if strings.TrimSpace(meta.InputKind) == "" || strings.TrimSpace(meta.OutputKind) == "" {
		return fmt.Errorf("translation validation metadata: missing IR kind")
	}
	if strings.TrimSpace(meta.InputVerifier) == "" || strings.TrimSpace(meta.OutputVerifier) == "" {
		return fmt.Errorf("translation validation metadata: missing input/output verifier")
	}
	if strings.TrimSpace(meta.ValidationStrategy) == "" {
		return fmt.Errorf("translation validation metadata: missing validation strategy")
	}
	if strings.TrimSpace(meta.ProofRule) == "" {
		return fmt.Errorf("translation validation metadata: missing proof rule")
	}
	if strings.TrimSpace(meta.TranslationValidationHook) == "" {
		return fmt.Errorf("translation validation metadata: missing translation validation hook")
	}
	if len(meta.ReportRows) == 0 {
		return fmt.Errorf("translation validation metadata: missing report rows")
	}
	if strings.TrimSpace(meta.NegativeTestMarker) == "" {
		return fmt.Errorf("translation validation metadata: missing negative-test marker")
	}
	if strings.TrimSpace(meta.ProfileInputPolicy) == "" {
		return fmt.Errorf("translation validation metadata: missing profile_input_policy")
	}
	if meta.ProfileInputDigest != "" {
		if !isStableHash(meta.ProfileInputDigest) {
			return fmt.Errorf("translation validation metadata: profile input digest must be sha256")
		}
		if strings.TrimSpace(meta.ProfileInputSchemaVersion) == "" {
			return fmt.Errorf("translation validation metadata: missing profile input schema version")
		}
	}
	if meta.ProfileInputSchemaVersion != "" && meta.ProfileInputDigest == "" {
		return fmt.Errorf("translation validation metadata: profile input schema version requires digest")
	}
	if !isStableHash(meta.BeforeHash) || !isStableHash(meta.AfterHash) {
		return fmt.Errorf("translation validation metadata: before/after hashes must be sha256")
	}
	if meta.Translation.FunctionsCompared == 0 || len(meta.Functions) == 0 {
		return fmt.Errorf("translation validation metadata: missing compared functions")
	}
	return nil
}

func stableIRHash(prog *ir.IRProgram) string {
	sum := sha256.Sum256([]byte(stableIRText(prog)))
	return fmt.Sprintf("sha256:%x", sum)
}

func isStableHash(value string) bool {
	return strings.HasPrefix(value, "sha256:") && len(value) == len("sha256:")+64
}

func stableIRText(prog *ir.IRProgram) string {
	if prog == nil {
		return "<nil>\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "main:%s index:%d funcs:%d\n", prog.MainName, prog.MainIndex, len(prog.Funcs))
	for _, fn := range prog.Funcs {
		fmt.Fprintf(&b, "func:%s export:%s params:%d locals:%d returns:%d budget:%t consent:%t\n", fn.Name, fn.ExportName, fn.ParamSlots, fn.LocalSlots, fn.ReturnSlots, fn.Policy.HasBudget, fn.Policy.HasConsent)
		for _, instr := range fn.Instrs {
			fmt.Fprintf(&b, "  kind:%d imm:%d local:%d label:%d name:%s args:%d rets:%d proof:%s str:%x\n", instr.Kind, instr.Imm, instr.Local, instr.Label, instr.Name, instr.ArgSlots, instr.RetSlots, instr.ProofID, instr.Str)
		}
	}
	return b.String()
}

func functionNames(prog *ir.IRProgram) []string {
	if prog == nil {
		return nil
	}
	out := make([]string, 0, len(prog.Funcs))
	for _, fn := range prog.Funcs {
		out = append(out, fn.Name)
	}
	sort.Strings(out)
	return out
}

func functionsByName(prog *ir.IRProgram) map[string]ir.IRFunc {
	out := map[string]ir.IRFunc{}
	if prog == nil {
		return out
	}
	for _, fn := range prog.Funcs {
		out[fn.Name] = fn
	}
	return out
}

func validateTranslationFunctionShape(before ir.IRFunc, after ir.IRFunc) error {
	if before.ParamSlots != after.ParamSlots {
		return fmt.Errorf("translation validation: %s param slot count changed from %d to %d", before.Name, before.ParamSlots, after.ParamSlots)
	}
	if before.ReturnSlots != after.ReturnSlots {
		return fmt.Errorf("translation validation: %s return slot count changed from %d to %d", before.Name, before.ReturnSlots, after.ReturnSlots)
	}
	if before.ExportName != after.ExportName {
		return fmt.Errorf("translation validation: %s export name changed from %q to %q", before.Name, before.ExportName, after.ExportName)
	}
	if before.Policy != after.Policy {
		return fmt.Errorf("translation validation: %s policy changed", before.Name)
	}
	return nil
}

func validateProofFactMultiset(before ProofReport, after ProofReport) (int, error) {
	beforeSet := proofFactMultiset(before)
	afterSet := proofFactMultiset(after)
	if !sameStringIntMap(beforeSet, afterSet) {
		return 0, fmt.Errorf("translation validation: proof facts changed: before=%s after=%s", formatStringIntMap(beforeSet), formatStringIntMap(afterSet))
	}
	total := 0
	for _, count := range beforeSet {
		total += count
	}
	return total, nil
}

func proofFactMultiset(report ProofReport) map[string]int {
	out := map[string]int{}
	for _, removed := range report.RemovedChecks {
		key := removed.Function + "\x00" + removed.Kind + "\x00" + removed.ProofID
		out[key]++
	}
	return out
}

func sameStringIntMap(left map[string]int, right map[string]int) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func formatStringIntMap(values map[string]int) string {
	if len(values) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteByte('{')
	for i, key := range keys {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(strings.ReplaceAll(key, "\x00", "/"))
		b.WriteByte(':')
		b.WriteString(strconv.Itoa(values[key]))
	}
	b.WriteByte('}')
	return b.String()
}

type symbolicValue struct {
	expr    string
	isConst bool
	value   int32
}

func validateSemanticLocalEquivalence(before map[string]ir.IRFunc, after map[string]ir.IRFunc, names []string) (int, error) {
	checks := 0
	for _, name := range names {
		beforeExpr, beforeOK := symbolicReturnExpr(before[name])
		afterExpr, afterOK := symbolicReturnExpr(after[name])
		if !beforeOK || !afterOK {
			continue
		}
		checks++
		if beforeExpr != afterExpr {
			return checks, fmt.Errorf("translation validation: semantic local equivalence failed for %s: before=%s after=%s", name, beforeExpr, afterExpr)
		}
	}
	return checks, nil
}

func symbolicReturnExpr(fn ir.IRFunc) (string, bool) {
	if fn.ReturnSlots > 1 {
		return "", false
	}
	stack := []symbolicValue{}
	locals := map[int]symbolicValue{}
	for i := 0; i < fn.ParamSlots; i++ {
		locals[i] = symbolicExpr(fmt.Sprintf("param%d", i))
	}
	for _, instr := range fn.Instrs {
		switch instr.Kind {
		case ir.IRConstI32:
			stack = append(stack, symbolicConst(instr.Imm))
		case ir.IRLoadLocal:
			value, ok := locals[instr.Local]
			if !ok {
				value = symbolicExpr(fmt.Sprintf("local%d", instr.Local))
			}
			stack = append(stack, value)
		case ir.IRStoreLocal:
			value, ok := popSymbolic(&stack)
			if !ok {
				return "", false
			}
			locals[instr.Local] = value
		case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32,
			ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
			ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
			right, left, ok := pop2Symbolic(&stack)
			if !ok {
				return "", false
			}
			stack = append(stack, symbolicBinary(instr.Kind, left, right))
		case ir.IRNegI32:
			value, ok := popSymbolic(&stack)
			if !ok {
				return "", false
			}
			stack = append(stack, symbolicNeg(value))
		case ir.IRReturn:
			if fn.ReturnSlots == 0 {
				return "void", len(stack) == 0
			}
			value, ok := popSymbolic(&stack)
			if !ok || len(stack) != 0 {
				return "", false
			}
			return value.expr, true
		default:
			return "", false
		}
	}
	return "", false
}

func symbolicConst(value int32) symbolicValue {
	return symbolicValue{expr: strconv.FormatInt(int64(value), 10), isConst: true, value: value}
}

func symbolicExpr(expr string) symbolicValue {
	return symbolicValue{expr: expr}
}

func popSymbolic(stack *[]symbolicValue) (symbolicValue, bool) {
	values := *stack
	if len(values) == 0 {
		return symbolicValue{}, false
	}
	value := values[len(values)-1]
	*stack = values[:len(values)-1]
	return value, true
}

func pop2Symbolic(stack *[]symbolicValue) (right symbolicValue, left symbolicValue, ok bool) {
	right, ok = popSymbolic(stack)
	if !ok {
		return symbolicValue{}, symbolicValue{}, false
	}
	left, ok = popSymbolic(stack)
	if !ok {
		return symbolicValue{}, symbolicValue{}, false
	}
	return right, left, true
}

func symbolicBinary(kind ir.IRInstrKind, left symbolicValue, right symbolicValue) symbolicValue {
	if left.isConst && right.isConst {
		if value, ok := evalBinaryI32(kind, left.value, right.value); ok {
			return symbolicConst(value)
		}
	}
	switch kind {
	case ir.IRAddI32:
		if isSymbolicConst(right, 0) {
			return left
		}
		if isSymbolicConst(left, 0) {
			return right
		}
	case ir.IRSubI32:
		if isSymbolicConst(right, 0) {
			return left
		}
	case ir.IRMulI32:
		if isSymbolicConst(right, 1) {
			return left
		}
		if isSymbolicConst(left, 1) {
			return right
		}
		if isSymbolicConst(left, 0) || isSymbolicConst(right, 0) {
			return symbolicConst(0)
		}
	}
	if left.expr == right.expr {
		switch kind {
		case ir.IRCmpEqI32, ir.IRCmpGeI32, ir.IRCmpLeI32:
			return symbolicConst(1)
		case ir.IRCmpNeI32, ir.IRCmpLtI32, ir.IRCmpGtI32:
			return symbolicConst(0)
		}
	}
	leftExpr, rightExpr := left.expr, right.expr
	if isCommutativeSymbolicBinaryOp(kind) && rightExpr < leftExpr {
		leftExpr, rightExpr = rightExpr, leftExpr
	}
	if isMirroredComparisonSymbolicBinaryOp(kind) && rightExpr < leftExpr {
		leftExpr, rightExpr = rightExpr, leftExpr
		kind = mirroredComparisonSymbolicBinaryOp(kind)
	}
	return symbolicExpr(fmt.Sprintf("%s(%s,%s)", symbolicOpName(kind), leftExpr, rightExpr))
}

func symbolicNeg(value symbolicValue) symbolicValue {
	if value.isConst {
		return symbolicConst(-value.value)
	}
	return symbolicExpr("neg(" + value.expr + ")")
}

func isSymbolicConst(value symbolicValue, want int32) bool {
	return value.isConst && value.value == want
}

func isCommutativeSymbolicBinaryOp(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRAddI32, ir.IRMulI32, ir.IRCmpEqI32, ir.IRCmpNeI32:
		return true
	default:
		return false
	}
}

func isMirroredComparisonSymbolicBinaryOp(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRCmpLtI32, ir.IRCmpGtI32, ir.IRCmpLeI32, ir.IRCmpGeI32:
		return true
	default:
		return false
	}
}

func mirroredComparisonSymbolicBinaryOp(kind ir.IRInstrKind) ir.IRInstrKind {
	switch kind {
	case ir.IRCmpLtI32:
		return ir.IRCmpGtI32
	case ir.IRCmpGtI32:
		return ir.IRCmpLtI32
	case ir.IRCmpLeI32:
		return ir.IRCmpGeI32
	case ir.IRCmpGeI32:
		return ir.IRCmpLeI32
	default:
		return kind
	}
}

func symbolicOpName(kind ir.IRInstrKind) string {
	switch kind {
	case ir.IRAddI32:
		return "add"
	case ir.IRSubI32:
		return "sub"
	case ir.IRMulI32:
		return "mul"
	case ir.IRCmpEqI32:
		return "eq"
	case ir.IRCmpLtI32:
		return "lt"
	case ir.IRCmpGtI32:
		return "gt"
	case ir.IRCmpGeI32:
		return "ge"
	case ir.IRCmpLeI32:
		return "le"
	case ir.IRCmpNeI32:
		return "ne"
	default:
		return fmt.Sprintf("ir%d", kind)
	}
}

func validateDifferentialSamples(before map[string]ir.IRFunc, after map[string]ir.IRFunc, names []string) (int, error) {
	samples := 0
	for _, name := range names {
		beforeFn := before[name]
		afterFn := after[name]
		if beforeFn.ReturnSlots != 1 || beforeFn.ParamSlots > 2 {
			continue
		}
		for _, args := range translationSampleArgs(beforeFn.ParamSlots) {
			beforeValue, beforeOK := evalStraightLineReturn(beforeFn, args)
			afterValue, afterOK := evalStraightLineReturn(afterFn, args)
			if !beforeOK || !afterOK {
				continue
			}
			samples++
			if beforeValue != afterValue {
				return samples, fmt.Errorf("translation validation: differential mismatch for %s args=%v: before=%d after=%d", name, args, beforeValue, afterValue)
			}
		}
	}
	return samples, nil
}

func translationSampleArgs(params int) [][]int32 {
	values := []int32{-2, -1, 0, 1, 2, 7}
	switch params {
	case 0:
		return [][]int32{{}}
	case 1:
		out := make([][]int32, 0, len(values))
		for _, value := range values {
			out = append(out, []int32{value})
		}
		return out
	case 2:
		out := make([][]int32, 0, len(values)*len(values))
		for _, left := range values {
			for _, right := range values {
				out = append(out, []int32{left, right})
			}
		}
		return out
	default:
		return nil
	}
}

func evalStraightLineReturn(fn ir.IRFunc, args []int32) (int32, bool) {
	if len(args) != fn.ParamSlots || fn.ReturnSlots != 1 {
		return 0, false
	}
	stack := []int32{}
	locals := map[int]int32{}
	for i, value := range args {
		locals[i] = value
	}
	for _, instr := range fn.Instrs {
		switch instr.Kind {
		case ir.IRConstI32:
			stack = append(stack, instr.Imm)
		case ir.IRLoadLocal:
			stack = append(stack, locals[instr.Local])
		case ir.IRStoreLocal:
			value, ok := popI32(&stack)
			if !ok {
				return 0, false
			}
			locals[instr.Local] = value
		case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32, ir.IRDivI32, ir.IRModI32,
			ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
			ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
			right, left, ok := pop2I32(&stack)
			if !ok {
				return 0, false
			}
			value, ok := evalBinaryI32(instr.Kind, left, right)
			if !ok {
				return 0, false
			}
			stack = append(stack, value)
		case ir.IRNegI32:
			value, ok := popI32(&stack)
			if !ok {
				return 0, false
			}
			stack = append(stack, -value)
		case ir.IRReturn:
			value, ok := popI32(&stack)
			if !ok || len(stack) != 0 {
				return 0, false
			}
			return value, true
		default:
			return 0, false
		}
	}
	return 0, false
}

func popI32(stack *[]int32) (int32, bool) {
	values := *stack
	if len(values) == 0 {
		return 0, false
	}
	value := values[len(values)-1]
	*stack = values[:len(values)-1]
	return value, true
}

func pop2I32(stack *[]int32) (right int32, left int32, ok bool) {
	right, ok = popI32(stack)
	if !ok {
		return 0, 0, false
	}
	left, ok = popI32(stack)
	if !ok {
		return 0, 0, false
	}
	return right, left, true
}

func evalBinaryI32(kind ir.IRInstrKind, left int32, right int32) (int32, bool) {
	switch kind {
	case ir.IRAddI32:
		return left + right, true
	case ir.IRSubI32:
		return left - right, true
	case ir.IRMulI32:
		return left * right, true
	case ir.IRDivI32:
		if right == 0 {
			return 0, false
		}
		return left / right, true
	case ir.IRModI32:
		if right == 0 {
			return 0, false
		}
		return left % right, true
	case ir.IRCmpEqI32:
		return boolToI32(left == right), true
	case ir.IRCmpLtI32:
		return boolToI32(left < right), true
	case ir.IRCmpGtI32:
		return boolToI32(left > right), true
	case ir.IRCmpGeI32:
		return boolToI32(left >= right), true
	case ir.IRCmpLeI32:
		return boolToI32(left <= right), true
	case ir.IRCmpNeI32:
		return boolToI32(left != right), true
	default:
		return 0, false
	}
}

func boolToI32(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

func boundsKind(kind ir.IRInstrKind) string {
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
