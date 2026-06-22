package validation

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/islandkernel"
	"tetra_language/compiler/internal/memoryfacts"
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
		{
			Stage:       StageProofFacts,
			Verifier:    "validation.CheckBoundsProofsWithPLIR",
			Implemented: true,
		},
		{Stage: StageOptimizedIR, Verifier: "lower.VerifyProgram", Implemented: true},
		{
			Stage:       StageAllocationPlan,
			Verifier:    "allocplan.VerifyPlan + validation.ValidateAllocationLowering",
			Implemented: true,
		},
		{Stage: StageMachineIR, Verifier: "machine.VerifyFunction", Implemented: true},
		{
			Stage:       StageABI,
			Verifier:    "x64abi/x86abi classifiers and target tests",
			Implemented: true,
		},
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
					return report, fmt.Errorf(
						"proof checker: %s instruction %d removes bounds check without proof id",
						fn.Name,
						i,
					)
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
		want := expectedProofOperationForRemovedCheck(*removed)
		if !guards[removed.Function][removed.ProofID] {
			if proofID, ok := resolveRemovedProofID(
				*removed,
				want,
				guards[removed.Function],
				terms[removed.Function],
				uses[removed.Function],
			); ok {
				removed.ProofID = proofID
			}
		}
		if !guards[removed.Function][removed.ProofID] {
			return report, fmt.Errorf(
				"proof checker: %s removed bounds check proof id %q not found in PLIR proof guards",
				removed.Function,
				removed.ProofID,
			)
		}
		proofUses := uses[removed.Function][removed.ProofID]
		if len(proofUses) == 0 {
			return report, fmt.Errorf(
				"proof checker: %s removed bounds check proof id %q has no dominated PLIR proof use",
				removed.Function,
				removed.ProofID,
			)
		}
		term, ok := terms[removed.Function][removed.ProofID]
		if !ok {
			return report, fmt.Errorf(
				"proof checker: %s removed bounds check proof id %q has no typed proof term",
				removed.Function,
				removed.ProofID,
			)
		}
		if want != "" && term.Operation != want {
			return report, fmt.Errorf(
				"proof checker: %s removed bounds check proof id %q operation %q does not match %q",
				removed.Function,
				removed.ProofID,
				term.Operation,
				want,
			)
		}
		if want != "" && !proofUsesContainOperation(proofUses, want) {
			return report, fmt.Errorf(
				"proof checker: %s removed bounds check proof id %q proof use operation does not match %q",
				removed.Function,
				removed.ProofID,
				want,
			)
		}
		proofReq := islandKernelProofRequest(term, want)
		if decision := islandkernel.CanEliminateBoundsCheck(proofReq); decision.Decision != islandkernel.Accept {
			return report, fmt.Errorf(
				"proof checker: %s removed bounds check proof id %q rejected by islandkernel: %s",
				removed.Function,
				removed.ProofID,
				decision.Reason.Code,
			)
		}
		if decision := islandkernel.CanEraseRuntimeCheck(proofReq); decision.Decision != islandkernel.Accept {
			return report, fmt.Errorf(
				"proof checker: %s removed runtime check proof id %q rejected by islandkernel: %s",
				removed.Function,
				removed.ProofID,
				decision.Reason.Code,
			)
		}
		removed.ProofTerm = validationProofTermFromProgramIR(term)
	}
	return report, nil
}

func resolveRemovedProofID(
	removed RemovedCheck,
	wantOperation string,
	guards map[string]bool,
	terms map[string]plir.ProofTerm,
	uses map[string][]plirProofUseEvidence,
) (string, bool) {
	prefix := removed.ProofID + ":"
	for id, term := range terms {
		if !strings.HasPrefix(id, prefix) || !guards[id] {
			continue
		}
		if wantOperation != "" && term.Operation != wantOperation {
			continue
		}
		if wantOperation != "" && !proofUsesContainOperation(uses[id], wantOperation) {
			continue
		}
		return id, true
	}
	for id := range guards {
		if strings.HasPrefix(id, prefix) {
			return id, true
		}
	}
	return "", false
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
			return strings.HasPrefix(use.ProofID, "proof:copy-loop:") &&
				strings.Contains(use.OpNote, "copy")
		default:
			return false
		}
	case "index_store":
		return use.OpKind == plir.OpIndexStore
	default:
		return false
	}
}

func validationProofTermFromProgramIR(term plir.ProofTerm) *ProofTerm {
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

func islandKernelProofRequest(term plir.ProofTerm, operation string) islandkernel.ProofRequest {
	if operation == "" {
		operation = term.Operation
	}
	baseID := term.SubjectBaseID
	if baseID == "" {
		baseID = term.BaseID
	}
	epoch := uint64(0)
	if term.Epoch > 0 {
		epoch = uint64(term.Epoch)
	}
	return islandkernel.ProofRequest{
		Ref: islandkernel.MemoryRef{
			BaseID:      baseID,
			IslandID:    term.IslandID,
			Epoch:       epoch,
			Provenance:  memoryfacts.ProvenanceSafeOwned,
			UnsafeClass: memoryfacts.UnsafeSafe,
		},
		Proof: islandkernel.Proof{
			ID:            term.ID,
			Kind:          islandKernelProofKind(term),
			SubjectBaseID: baseID,
			IslandID:      term.IslandID,
			Epoch:         epoch,
			Operation:     operation,
			Verified:      true,
		},
		Operation: operation,
	}
}

func islandKernelProofKind(term plir.ProofTerm) memoryfacts.ProofKind {
	switch term.Kind {
	case string(memoryfacts.ProofBounds), "bounds_check":
		return memoryfacts.ProofBounds
	default:
		return memoryfacts.ProofKind(term.Kind)
	}
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
	case ir.IRDropOwned:
		return 1, 1, true
	case ir.IRReleaseAllocation:
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
	case ir.IRAtomicCompareExchangePtr,
		ir.IRAtomicCompareExchangeI32,
		ir.IRAtomicCompareExchangeI64,
		ir.IRAtomicCompareExchangeI8,
		ir.IRAtomicCompareExchangeI16:
		return 4, 1, true
	case ir.IRMemReadI32Offset, ir.IRMemReadU8Offset, ir.IRMemReadPtrOffset:
		return 3, 1, true
	case ir.IRMemWriteI32Offset,
		ir.IRMemWriteU8Offset,
		ir.IRMemWritePtrOffset,
		ir.IRMemWriteArchPtrOffset:
		return 4, 1, true
	case ir.IRAtomicFenceSeqCst, ir.IRAtomicFenceRelaxed, ir.IRAtomicFenceAcquire,
		ir.IRAtomicFenceRelease, ir.IRAtomicFenceAcqRel:
		return 0, 0, true
	default:
		return 0, 0, false
	}
}
