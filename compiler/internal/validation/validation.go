package validation

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/ir"
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
