package compiler

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/differential"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/opt"
	"tetra_language/compiler/internal/validation"
)

const (
	translationValidationV2Schema    = "tetra.translation.validation.v2"
	translationValidationV2ScopeP230 = "p23.0_translation_validation_v2"

	p23TranslationRegisteredPassesWitnessID = "registered_optimizer_passes"
	p23TranslationScalarWitnessID           = "symbolic_scalar_arithmetic"
	p23TranslationMemoryWitnessID           = "memory_i32_slice_equivalence"
	p23TranslationLoopWitnessID             = "loop_equivalence_samples"
	p23TranslationCallInliningWitnessID     = "call_inlining_equivalence"
	p23TranslationProofWitnessID            = "bounds_proof_preservation"
	p23TranslationAllocationWitnessID       = "allocation_plan_preservation"
	p23TranslationHashWitnessID             = "before_after_hash_metadata"
)

type TranslationValidationV2ID string

const (
	TranslationValidationV2RegisteredPasses           TranslationValidationV2ID = "registered_passes"
	TranslationValidationV2SymbolicScalar             TranslationValidationV2ID = "symbolic_scalar_equivalence"
	TranslationValidationV2MemoryEquivalence          TranslationValidationV2ID = "memory_equivalence"
	TranslationValidationV2BoundsProofPreservation    TranslationValidationV2ID = "bounds_proof_preservation"
	TranslationValidationV2AllocationPlanPreservation TranslationValidationV2ID = "allocation_plan_preservation"
	TranslationValidationV2MachineCheckableHashes     TranslationValidationV2ID = "machine_checkable_hashes"
)

type TranslationValidationV2Report struct {
	SchemaVersion                          string                           `json:"schema_version"`
	Scope                                  string                           `json:"scope"`
	Rows                                   []TranslationValidationV2Row     `json:"rows"`
	Witnesses                              []TranslationValidationV2Witness `json:"witnesses"`
	NonClaims                              []string                         `json:"non_claims"`
	RegisteredPassCoverageComplete         bool                             `json:"registered_pass_coverage_complete"`
	SymbolicScalarEquivalenceSamples       int                              `json:"symbolic_scalar_equivalence_samples"`
	MemoryEquivalenceSamples               int                              `json:"memory_equivalence_samples"`
	LoopEquivalenceSamples                 int                              `json:"loop_equivalence_samples"`
	CallEquivalenceSamples                 int                              `json:"call_equivalence_samples"`
	BoundsProofsPreserved                  bool                             `json:"bounds_proofs_preserved"`
	AllocationPlanValidated                bool                             `json:"allocation_plan_validated"`
	BeforeAfterHashesMachineCheckable      bool                             `json:"before_after_hashes_machine_checkable"`
	FullFormalProofClaimed                 bool                             `json:"full_formal_proof_claimed"`
	ExhaustiveOptimizerCompletenessClaimed bool                             `json:"exhaustive_optimizer_completeness_claimed"`
	BroadMemoryModelClaimed                bool                             `json:"broad_memory_model_claimed"`
	BroadLoopTheoremProverClaimed          bool                             `json:"broad_loop_theorem_prover_claimed"`
	PerformanceClaimed                     bool                             `json:"performance_claimed"`
	RuntimeBehaviorChanged                 bool                             `json:"runtime_behavior_changed"`
	SafeSemanticsChanged                   bool                             `json:"safe_semantics_changed"`
}

type TranslationValidationV2Row struct {
	ID         TranslationValidationV2ID `json:"id"`
	Name       string                    `json:"name"`
	Status     string                    `json:"status"`
	Evidence   []string                  `json:"evidence"`
	Tests      []string                  `json:"tests"`
	Boundaries []string                  `json:"boundaries"`
	WitnessIDs []string                  `json:"witness_ids"`
}

type TranslationValidationV2Witness struct {
	ID                             string `json:"id"`
	Kind                           string `json:"kind"`
	RegisteredPasses               int    `json:"registered_passes,omitempty"`
	RegisteredPassCoverageComplete bool   `json:"registered_pass_coverage_complete,omitempty"`
	TranslationMetadataPresent     bool   `json:"translation_metadata_present,omitempty"`
	SymbolicScalarChecks           int    `json:"symbolic_scalar_checks,omitempty"`
	DifferentialSamples            int    `json:"differential_samples,omitempty"`
	SemanticMismatchRejected       bool   `json:"semantic_mismatch_rejected,omitempty"`
	MemoryEquivalenceSamples       int    `json:"memory_equivalence_samples,omitempty"`
	MemoryMismatchRejected         bool   `json:"memory_mismatch_rejected,omitempty"`
	LoopEquivalenceSamples         int    `json:"loop_equivalence_samples,omitempty"`
	DifferentialLanes              int    `json:"differential_lanes,omitempty"`
	CallEquivalenceSamples         int    `json:"call_equivalence_samples,omitempty"`
	BeforeHadCall                  bool   `json:"before_had_call,omitempty"`
	AfterHadCall                   bool   `json:"after_had_call,omitempty"`
	TranslationValidated           bool   `json:"translation_validated,omitempty"`
	ProofFactsCompared             int    `json:"proof_facts_compared,omitempty"`
	BoundsProofsPreserved          bool   `json:"bounds_proofs_preserved,omitempty"`
	MissingProofRejected           bool   `json:"missing_proof_rejected,omitempty"`
	AllocationPlanValidated        bool   `json:"allocation_plan_validated,omitempty"`
	AllocationDriftRejected        bool   `json:"allocation_drift_rejected,omitempty"`
	BeforeHash                     string `json:"before_hash,omitempty"`
	AfterHash                      string `json:"after_hash,omitempty"`
	HashesMachineCheckable         bool   `json:"hashes_machine_checkable,omitempty"`
	HashesDistinct                 bool   `json:"hashes_distinct,omitempty"`
}

func BuildP23TranslationValidationV2() (TranslationValidationV2Report, error) {
	registered, err := buildP23RegisteredPassesWitness()
	if err != nil {
		return TranslationValidationV2Report{}, err
	}
	scalar, err := buildP23ScalarWitness()
	if err != nil {
		return TranslationValidationV2Report{}, err
	}
	memory, err := buildP23MemoryWitness()
	if err != nil {
		return TranslationValidationV2Report{}, err
	}
	loop, err := buildP23LoopWitness()
	if err != nil {
		return TranslationValidationV2Report{}, err
	}
	call, err := buildP23CallInliningWitness()
	if err != nil {
		return TranslationValidationV2Report{}, err
	}
	proof, err := buildP23ProofWitness()
	if err != nil {
		return TranslationValidationV2Report{}, err
	}
	allocation, err := buildP23AllocationWitness()
	if err != nil {
		return TranslationValidationV2Report{}, err
	}
	hash, err := buildP23HashWitness()
	if err != nil {
		return TranslationValidationV2Report{}, err
	}

	report := TranslationValidationV2Report{
		SchemaVersion: translationValidationV2Schema,
		Scope:         translationValidationV2ScopeP230,
		Witnesses: []TranslationValidationV2Witness{
			registered,
			scalar,
			memory,
			loop,
			call,
			proof,
			allocation,
			hash,
		},
		Rows: []TranslationValidationV2Row{
			p23TranslationRow(TranslationValidationV2RegisteredPasses, "Registered optimizer passes", "current_supported_subset",
				[]string{
					"opt.RegisteredPasses returns every current optimizer pass and opt.ValidatePassContract requires translation_validation plus validation.ValidateTranslation.",
					"RegisteredPasses witness runs NewManager over all registered passes and requires validation metadata on every pass report.",
					"compiler/internal/opt/manager.go stores translation reports, validation metadata, before dumps, after dumps, and profile_input_policy rows.",
				},
				[]string{
					"go test ./compiler -run 'P23TranslationValidationV2|ValidateP23TranslationValidationV2'",
					"go test ./compiler/internal/opt -run 'Manager|BasicScalar|SCCP|Mem2Reg|Inline|Loop|LICM'",
				},
				[]string{
					"registered optimizer pass coverage is limited to opt.RegisteredPasses",
					"translation validation is an internal evidence hook, not a public optimization mode",
					"no exhaustive optimizer completeness is claimed",
				},
				[]string{p23TranslationRegisteredPassesWitnessID}),
			p23TranslationRow(TranslationValidationV2SymbolicScalar, "Symbolic scalar equivalence", "current_supported_subset",
				[]string{
					"validation.ValidateTranslation runs validateSemanticLocalEquivalence over supported straight-line scalar arithmetic and comparison rewrites.",
					"Symbolic scalar witness checks add-zero equivalence and records semantic local equivalence plus differential samples.",
					"Negative witness rejects a semantic local equivalence mismatch when add-zero becomes add-one.",
				},
				[]string{
					"go test ./compiler/internal/validation -run 'ValidateTranslation.*Algebra|ValidateTranslationRejectsBadLocalAlgebraRewrite'",
					"go test ./compiler -run 'P23TranslationValidationV2'",
				},
				[]string{
					"symbolic equivalence is limited to the current scalar i32 local subset",
					"unsupported expressions are skipped rather than trusted",
					"no full scalar theorem prover is claimed",
				},
				[]string{p23TranslationScalarWitnessID}),
			p23TranslationRow(TranslationValidationV2MemoryEquivalence, "Memory equivalence for supported i32 slice samples", "current_supported_subset",
				[]string{
					"differential.CheckBackendMatrix compares source, Stack IR, optimized Stack IR, SSA, and Machine IR lanes for a proof-tagged i32 slice sum memory sample.",
					"compiler/internal/differential/differential.go interprets supported i32 slice memory through loadI32Slice and storeI32Slice.",
					"Memory witness rejects a bad source oracle through the same backend matrix mismatch path.",
				},
				[]string{
					"go test ./compiler/internal/differential -run 'BackendMatrix'",
					"go test ./compiler -run 'P23TranslationValidationV2'",
				},
				[]string{
					"memory equivalence evidence is limited to supported i32 slice samples",
					"no broad memory model or alias model is claimed",
					"region/local slice coverage is evidence-bound to current lowering and allocation reports",
				},
				[]string{p23TranslationMemoryWitnessID}),
			p23TranslationRow(TranslationValidationV2BoundsProofPreservation, "Bounds proof preservation", "current_supported_subset",
				[]string{
					"validation.ValidateTranslation validates input/output proof facts through CheckBoundsProofs and validateProofFactMultiset.",
					"Proof witness preserves a proof-tagged unchecked i32 load and records proof facts compared.",
					"Negative witness rejects an unchecked bounds load whose proof id disappears after transformation.",
				},
				[]string{
					"go test ./compiler/internal/validation -run 'ValidateTranslationRejectsProof|ValidateTranslationRejectsMissingProof'",
					"go test ./compiler -run 'P23TranslationValidationV2'",
				},
				[]string{
					"bounds proof preservation covers proof-tagged removed checks in current IR",
					"changed or missing proof ids are validation failures",
					"no unchecked load may be trusted without proof id evidence",
				},
				[]string{p23TranslationProofWitnessID}),
			p23TranslationRow(TranslationValidationV2AllocationPlanPreservation, "Allocation-plan preservation", "current_supported_subset",
				[]string{
					"validation.ValidateAllocationLowering checks allocation plan rows against emitted Stack IR allocation lowering.",
					"Allocation witness validates a stack-lowered i32 slice against allocplan.VerifyPlan and ValidateAllocationLowering.",
					"Negative witness rejects allocation drift when the matching Stack IR allocation is missing.",
				},
				[]string{
					"go test ./compiler/internal/validation -run 'ValidateAllocationLowering'",
					"go test ./compiler -run 'P23TranslationValidationV2'",
				},
				[]string{
					"allocation plan preservation is evidence-bound to current allocplan and lowering validators",
					"no broad allocation optimizer is claimed",
					"runtime behavior does not change",
				},
				[]string{p23TranslationAllocationWitnessID}),
			p23TranslationRow(TranslationValidationV2MachineCheckableHashes, "Machine-checkable before/after hashes", "current_supported_subset",
				[]string{
					"validation.BuildOptimizationValidationMetadata records sha256 before and after IR hashes for translation-validated optimization evidence.",
					"Hash witness builds metadata for a real add-zero rewrite and records distinct before/after sha256 hashes.",
					"ValidateOptimizationValidationMetadata rejects missing or malformed hash metadata.",
				},
				[]string{
					"go test ./compiler/internal/validation -run 'OptimizationValidationMetadata'",
					"go test ./compiler -run 'P23TranslationValidationV2'",
				},
				[]string{
					"hashes are machine-checkable evidence, not proof by themselves",
					"hash evidence is scoped to the compared IR text and functions",
					"safe-program semantics do not change",
				},
				[]string{p23TranslationHashWitnessID}),
		},
		NonClaims: []string{
			"no full formal proof is claimed",
			"no exhaustive optimizer completeness is claimed",
			"no broad memory model or alias model is claimed",
			"no broad loop theorem prover is claimed",
			"no performance claim is made",
			"runtime behavior does not change",
			"safe-program semantics do not change",
		},
		RegisteredPassCoverageComplete:    registered.RegisteredPassCoverageComplete,
		SymbolicScalarEquivalenceSamples:  scalar.SymbolicScalarChecks,
		MemoryEquivalenceSamples:          memory.MemoryEquivalenceSamples,
		LoopEquivalenceSamples:            loop.LoopEquivalenceSamples,
		CallEquivalenceSamples:            call.CallEquivalenceSamples,
		BoundsProofsPreserved:             proof.BoundsProofsPreserved,
		AllocationPlanValidated:           allocation.AllocationPlanValidated,
		BeforeAfterHashesMachineCheckable: hash.HashesMachineCheckable,
	}
	if err := ValidateP23TranslationValidationV2(report); err != nil {
		return TranslationValidationV2Report{}, err
	}
	return report, nil
}

func ValidateP23TranslationValidationV2(report TranslationValidationV2Report) error {
	if report.SchemaVersion != translationValidationV2Schema {
		return fmt.Errorf("translation validation v2: schema_version is %q", report.SchemaVersion)
	}
	if report.Scope != translationValidationV2ScopeP230 {
		return fmt.Errorf("translation validation v2: scope is %q", report.Scope)
	}
	if report.FullFormalProofClaimed {
		return fmt.Errorf("translation validation v2: full formal proof claim is forbidden")
	}
	if report.ExhaustiveOptimizerCompletenessClaimed {
		return fmt.Errorf("translation validation v2: exhaustive optimizer completeness claim is forbidden")
	}
	if report.BroadMemoryModelClaimed {
		return fmt.Errorf("translation validation v2: broad memory model claim is forbidden")
	}
	if report.BroadLoopTheoremProverClaimed {
		return fmt.Errorf("translation validation v2: broad loop theorem prover claim is forbidden")
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("translation validation v2: performance claim is forbidden")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf("translation validation v2: runtime behavior change claim is forbidden")
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf("translation validation v2: safe semantics change claim is forbidden")
	}
	if !report.RegisteredPassCoverageComplete {
		return fmt.Errorf("translation validation v2: registered pass coverage is incomplete")
	}
	if report.SymbolicScalarEquivalenceSamples == 0 {
		return fmt.Errorf("translation validation v2: missing symbolic scalar equivalence samples")
	}
	if report.MemoryEquivalenceSamples == 0 {
		return fmt.Errorf("translation validation v2: missing memory equivalence samples")
	}
	if report.LoopEquivalenceSamples == 0 {
		return fmt.Errorf("translation validation v2: missing loop equivalence samples")
	}
	if report.CallEquivalenceSamples == 0 {
		return fmt.Errorf("translation validation v2: missing call equivalence samples")
	}
	if !report.BoundsProofsPreserved {
		return fmt.Errorf("translation validation v2: bounds proof preservation evidence missing")
	}
	if !report.AllocationPlanValidated {
		return fmt.Errorf("translation validation v2: allocation plan validation evidence missing")
	}
	if !report.BeforeAfterHashesMachineCheckable {
		return fmt.Errorf("translation validation v2: before/after hash evidence missing")
	}
	if err := p23ValidateRows(report.Rows, report.Witnesses); err != nil {
		return err
	}
	if err := p23ValidateWitnesses(report.Witnesses); err != nil {
		return err
	}
	for _, want := range []string{
		"no full formal proof is claimed",
		"no exhaustive optimizer completeness is claimed",
		"no broad memory model or alias model is claimed",
		"no broad loop theorem prover is claimed",
		"no performance claim is made",
		"runtime behavior does not change",
		"safe-program semantics do not change",
	} {
		if !p23TranslationHasString(report.NonClaims, want) {
			return fmt.Errorf("translation validation v2: missing non-claim %q", want)
		}
	}
	return nil
}

func buildP23RegisteredPassesWitness() (TranslationValidationV2Witness, error) {
	passes := opt.RegisteredPasses()
	for _, pass := range passes {
		if err := opt.ValidatePassContract(pass); err != nil {
			return TranslationValidationV2Witness{}, err
		}
	}
	report, err := opt.NewManager().Run(p23TinyProgram(), passes...)
	if err != nil {
		return TranslationValidationV2Witness{}, err
	}
	metadata := len(report.Passes) == len(passes)
	for _, row := range report.Passes {
		if !row.TranslationValidated || row.TranslationReport == nil || row.ValidationMetadata == nil {
			metadata = false
			break
		}
		if row.ValidationMetadata.BeforeHash == "" || row.ValidationMetadata.AfterHash == "" {
			metadata = false
			break
		}
	}
	return TranslationValidationV2Witness{
		ID:                             p23TranslationRegisteredPassesWitnessID,
		Kind:                           "optimizer_manager_registered_passes",
		RegisteredPasses:               len(passes),
		RegisteredPassCoverageComplete: len(report.Passes) == len(passes) && metadata,
		TranslationMetadataPresent:     metadata,
	}, nil
}

func buildP23ScalarWitness() (TranslationValidationV2Witness, error) {
	before := p23SingleReturnProgram("main", 1,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 0},
		ir.IRInstr{Kind: ir.IRAddI32},
	)
	after := p23SingleReturnProgram("main", 1,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
	)
	report, err := validation.ValidateTranslation(before, after)
	if err != nil {
		return TranslationValidationV2Witness{}, err
	}
	badAfter := p23SingleReturnProgram("main", 1,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 1},
		ir.IRInstr{Kind: ir.IRAddI32},
	)
	_, badErr := validation.ValidateTranslation(before, badAfter)
	return TranslationValidationV2Witness{
		ID:                       p23TranslationScalarWitnessID,
		Kind:                     "symbolic_scalar_arithmetic",
		SymbolicScalarChecks:     report.SemanticLocalChecks,
		DifferentialSamples:      report.DifferentialSamples,
		SemanticMismatchRejected: badErr != nil && strings.Contains(badErr.Error(), "semantic local equivalence"),
	}, nil
}

func buildP23MemoryWitness() (TranslationValidationV2Witness, error) {
	tc := p23SliceMatrixCase(false)
	report, err := differential.CheckBackendMatrix(tc)
	if err != nil {
		return TranslationValidationV2Witness{}, err
	}
	bad := p23SliceMatrixCase(true)
	_, badErr := differential.CheckBackendMatrix(bad)
	return TranslationValidationV2Witness{
		ID:                       p23TranslationMemoryWitnessID,
		Kind:                     "i32_slice_memory_matrix",
		MemoryEquivalenceSamples: len(report.Samples),
		DifferentialLanes:        len(report.Lanes),
		MemoryMismatchRejected:   badErr != nil && strings.Contains(badErr.Error(), "differential mismatch"),
	}, nil
}

func buildP23LoopWitness() (TranslationValidationV2Witness, error) {
	report, err := differential.CheckBackendMatrix(differential.BackendMatrixCase{
		Name:      "p23-loop-sum",
		Functions: []ir.IRFunc{p23LoopSumFunc()},
		Entry:     "sum_n",
		Samples: []differential.MatrixSample{
			{Name: "zero", Args: []int32{0}},
			{Name: "five", Args: []int32{5}},
		},
		Source: func(sample differential.MatrixSample) (int32, bool) {
			n := sample.Args[0]
			var total int32
			for i := int32(0); i < n; i++ {
				total += i
			}
			return total, true
		},
	})
	if err != nil {
		return TranslationValidationV2Witness{}, err
	}
	return TranslationValidationV2Witness{
		ID:                     p23TranslationLoopWitnessID,
		Kind:                   "loop_backend_matrix",
		LoopEquivalenceSamples: len(report.Samples),
		DifferentialLanes:      len(report.Lanes),
	}, nil
}

func buildP23CallInliningWitness() (TranslationValidationV2Witness, error) {
	funcs := []ir.IRFunc{p23HelperAddOneFunc(), p23CallHelperFunc()}
	tc := differential.BackendMatrixCase{
		Name:          "p23-call-inline",
		Functions:     funcs,
		Entry:         "main",
		Samples:       []differential.MatrixSample{{Name: "seven", Args: []int32{7}}},
		Optimizations: []opt.Pass{opt.InlineSmallPurePass()},
		Source: func(sample differential.MatrixSample) (int32, bool) {
			return sample.Args[0] + 1, true
		},
	}
	matrix, err := differential.CheckBackendMatrix(tc)
	if err != nil {
		return TranslationValidationV2Witness{}, err
	}
	prog := &ir.IRProgram{MainIndex: 1, MainName: "main", Funcs: p23CloneFuncs(funcs)}
	runReport, err := opt.NewManager().Run(prog, opt.InlineSmallPurePass())
	if err != nil {
		return TranslationValidationV2Witness{}, err
	}
	return TranslationValidationV2Witness{
		ID:                     p23TranslationCallInliningWitnessID,
		Kind:                   "call_inlining_backend_matrix",
		CallEquivalenceSamples: len(matrix.Samples),
		DifferentialLanes:      len(matrix.Lanes),
		BeforeHadCall:          p23ProgramHasCall(&ir.IRProgram{Funcs: funcs}),
		AfterHadCall:           p23ProgramHasCall(prog),
		TranslationValidated:   len(runReport.Passes) == 1 && runReport.Passes[0].TranslationValidated,
	}, nil
}

func buildP23ProofWitness() (TranslationValidationV2Witness, error) {
	before := p23ProofProgram("main", "proof:while:i:xs:1:1")
	after := p23ProofProgram("main", "proof:while:i:xs:1:1")
	report, err := validation.ValidateTranslation(before, after)
	if err != nil {
		return TranslationValidationV2Witness{}, err
	}
	missingProof := p23ProofProgram("main", "")
	_, badErr := validation.ValidateTranslation(before, missingProof)
	return TranslationValidationV2Witness{
		ID:                    p23TranslationProofWitnessID,
		Kind:                  "bounds_proof_multiset",
		ProofFactsCompared:    report.ProofFactsCompared,
		BoundsProofsPreserved: report.ProofFactsCompared > 0,
		MissingProofRejected:  badErr != nil && strings.Contains(badErr.Error(), "missing proof id"),
	}, nil
}

func buildP23AllocationWitness() (TranslationValidationV2Witness, error) {
	plan := p23AllocationPlan("main")
	prog := p23AllocationProgram("main", true)
	if err := validation.ValidateAllocationLowering(plan, prog); err != nil {
		return TranslationValidationV2Witness{}, err
	}
	badProg := p23AllocationProgram("main", false)
	badErr := validation.ValidateAllocationLowering(plan, badProg)
	return TranslationValidationV2Witness{
		ID:                      p23TranslationAllocationWitnessID,
		Kind:                    "allocation_lowering_validation",
		AllocationPlanValidated: true,
		AllocationDriftRejected: badErr != nil && strings.Contains(badErr.Error(), "no matching IR stack slice"),
	}, nil
}

func buildP23HashWitness() (TranslationValidationV2Witness, error) {
	before := p23SingleReturnProgram("main", 1,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 0},
		ir.IRInstr{Kind: ir.IRAddI32},
	)
	after := p23SingleReturnProgram("main", 1,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
	)
	meta, err := validation.BuildOptimizationValidationMetadata(before, after, validation.OptimizationMetadataOptions{
		PassName:                  "basic-scalar",
		InputKind:                 string(opt.IRKindStack),
		OutputKind:                string(opt.IRKindStack),
		InputVerifier:             opt.VerifierLowerVerifyProgram,
		OutputVerifier:            opt.VerifierLowerVerifyProgram,
		ValidationStrategy:        string(opt.ValidationTranslation),
		RequiredFacts:             []string{string(opt.FactIRVerified)},
		PreservedFacts:            []string{string(opt.FactBoundsProofs)},
		InvalidatedFacts:          []string{string(opt.FactLiveness)},
		ProofRule:                 string(opt.ProofRulePreserveBoundsInvalidateLiveness),
		TranslationValidationHook: opt.TranslationHookValidateTranslation,
		ReportRows:                opt.RequiredP17ReportRows(),
		NegativeTestMarker:        opt.NegativeTestPassContractV1,
		ProfileInputPolicy:        string(opt.ProfileInputUnused),
	})
	if err != nil {
		return TranslationValidationV2Witness{}, err
	}
	return TranslationValidationV2Witness{
		ID:                     p23TranslationHashWitnessID,
		Kind:                   "optimization_validation_metadata_hashes",
		BeforeHash:             meta.BeforeHash,
		AfterHash:              meta.AfterHash,
		HashesMachineCheckable: strings.HasPrefix(meta.BeforeHash, "sha256:") && strings.HasPrefix(meta.AfterHash, "sha256:"),
		HashesDistinct:         meta.BeforeHash != meta.AfterHash,
	}, nil
}

func p23ValidateRows(rows []TranslationValidationV2Row, witnesses []TranslationValidationV2Witness) error {
	witnessByID := map[string]bool{}
	for _, witness := range witnesses {
		witnessByID[witness.ID] = true
	}
	seen := map[TranslationValidationV2ID]bool{}
	for _, row := range rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 || len(row.WitnessIDs) == 0 {
			return fmt.Errorf("translation validation v2: row %q missing required metadata", row.ID)
		}
		if seen[row.ID] {
			return fmt.Errorf("translation validation v2: duplicate row %s", row.ID)
		}
		seen[row.ID] = true
		if p23ContainsPlaceholder(row.Evidence) || p23ContainsPlaceholder(row.Boundaries) {
			return fmt.Errorf("translation validation v2: row %s contains placeholder evidence", row.ID)
		}
		for _, witnessID := range row.WitnessIDs {
			if !witnessByID[witnessID] {
				return fmt.Errorf("translation validation v2: row %s references missing witness %q", row.ID, witnessID)
			}
		}
	}
	for _, id := range p23TranslationValidationV2IDs() {
		if !seen[id] {
			return fmt.Errorf("translation validation v2: missing row %s", id)
		}
	}
	return nil
}

func p23ValidateWitnesses(witnesses []TranslationValidationV2Witness) error {
	byID := map[string]TranslationValidationV2Witness{}
	for _, witness := range witnesses {
		byID[witness.ID] = witness
	}
	registered := byID[p23TranslationRegisteredPassesWitnessID]
	if registered.RegisteredPasses < len(opt.RegisteredPasses()) || !registered.RegisteredPassCoverageComplete || !registered.TranslationMetadataPresent {
		return fmt.Errorf("translation validation v2: registered pass witness incomplete: %+v", registered)
	}
	scalar := byID[p23TranslationScalarWitnessID]
	if scalar.SymbolicScalarChecks == 0 || scalar.DifferentialSamples == 0 || !scalar.SemanticMismatchRejected {
		return fmt.Errorf("translation validation v2: symbolic scalar witness incomplete: %+v", scalar)
	}
	memory := byID[p23TranslationMemoryWitnessID]
	if memory.MemoryEquivalenceSamples == 0 || memory.DifferentialLanes < 5 || !memory.MemoryMismatchRejected {
		return fmt.Errorf("translation validation v2: memory equivalence witness incomplete: %+v", memory)
	}
	loop := byID[p23TranslationLoopWitnessID]
	if loop.LoopEquivalenceSamples == 0 || loop.DifferentialLanes < 5 {
		return fmt.Errorf("translation validation v2: loop equivalence witness incomplete: %+v", loop)
	}
	call := byID[p23TranslationCallInliningWitnessID]
	if call.CallEquivalenceSamples == 0 || !call.BeforeHadCall || call.AfterHadCall || !call.TranslationValidated {
		return fmt.Errorf("translation validation v2: call/inlining witness incomplete: %+v", call)
	}
	proof := byID[p23TranslationProofWitnessID]
	if proof.ProofFactsCompared == 0 || !proof.BoundsProofsPreserved || !proof.MissingProofRejected {
		return fmt.Errorf("translation validation v2: bounds proof witness incomplete: %+v", proof)
	}
	allocation := byID[p23TranslationAllocationWitnessID]
	if !allocation.AllocationPlanValidated || !allocation.AllocationDriftRejected {
		return fmt.Errorf("translation validation v2: allocation plan witness incomplete: %+v", allocation)
	}
	hash := byID[p23TranslationHashWitnessID]
	if !strings.HasPrefix(hash.BeforeHash, "sha256:") || !strings.HasPrefix(hash.AfterHash, "sha256:") || !hash.HashesMachineCheckable || !hash.HashesDistinct {
		return fmt.Errorf("translation validation v2: hash witness incomplete: %+v", hash)
	}
	return nil
}

func p23TranslationValidationV2IDs() []TranslationValidationV2ID {
	return []TranslationValidationV2ID{
		TranslationValidationV2RegisteredPasses,
		TranslationValidationV2SymbolicScalar,
		TranslationValidationV2MemoryEquivalence,
		TranslationValidationV2BoundsProofPreservation,
		TranslationValidationV2AllocationPlanPreservation,
		TranslationValidationV2MachineCheckableHashes,
	}
}

func p23TranslationRow(id TranslationValidationV2ID, name string, status string, evidence []string, tests []string, boundaries []string, witnessIDs []string) TranslationValidationV2Row {
	return TranslationValidationV2Row{
		ID:         id,
		Name:       name,
		Status:     status,
		Evidence:   evidence,
		Tests:      tests,
		Boundaries: boundaries,
		WitnessIDs: witnessIDs,
	}
}

func p23TranslationHasString(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func p23ContainsPlaceholder(values []string) bool {
	for _, value := range values {
		text := strings.ToLower(strings.TrimSpace(value))
		if text == "" || strings.Contains(text, "todo") || strings.Contains(text, "placeholder") || strings.Contains(text, "paper-only") {
			return true
		}
	}
	return false
}

func p23TinyProgram() *ir.IRProgram {
	return p23SingleReturnProgram("main", 0, ir.IRInstr{Kind: ir.IRConstI32, Imm: 1})
}

func p23SingleReturnProgram(name string, params int, instrs ...ir.IRInstr) *ir.IRProgram {
	body := append([]ir.IRInstr(nil), instrs...)
	body = append(body, ir.IRInstr{Kind: ir.IRReturn})
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  name,
		Funcs: []ir.IRFunc{{
			Name:        name,
			ParamSlots:  params,
			LocalSlots:  params,
			ReturnSlots: 1,
			Instrs:      body,
		}},
	}
}

func p23SliceMatrixCase(badSource bool) differential.BackendMatrixCase {
	return differential.BackendMatrixCase{
		Name:      "p23-slice-memory",
		Functions: []ir.IRFunc{p23SliceSumFunc()},
		Entry:     "sum",
		Samples: []differential.MatrixSample{{
			Name:      "four-elements",
			Args:      []int32{1, 4},
			I32Slices: map[int32][]int32{1: {1, 2, 3, 4}},
		}},
		Source: func(sample differential.MatrixSample) (int32, bool) {
			xs := sample.I32Slices[sample.Args[0]]
			var total int32
			for i := int32(0); i < sample.Args[1]; i++ {
				total += xs[i]
			}
			if badSource {
				total++
			}
			return total, true
		},
	}
}

func p23SliceSumFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum",
		ParamSlots:  2,
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:while:i:xs:1:1"},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func p23LoopSumFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum_n",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func p23HelperAddOneFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "inc",
		ParamSlots:  1,
		LocalSlots:  1,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}
}

func p23CallHelperFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		ParamSlots:  1,
		LocalSlots:  1,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCall, Name: "inc", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func p23ProofProgram(name string, proofID string) *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  name,
		Funcs: []ir.IRFunc{{
			Name:        name,
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRIndexLoadI32Unchecked, ProofID: proofID},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func p23AllocationPlan(name string) *allocplan.Plan {
	return &allocplan.Plan{Functions: []allocplan.FunctionPlan{{
		Name: name,
		Allocations: []allocplan.Allocation{{
			ID:                    "xs",
			SiteID:                "allocsite:" + name + ":xs:line_1_1",
			ValueID:               "alloc_intent:xs",
			Builtin:               "core.make_i32",
			ElementType:           "i32",
			ElementSize:           4,
			LengthExpr:            "4",
			LengthStatus:          allocplan.LengthStatusNormal,
			ZeroGuardStatus:       "valid_empty_no_allocator",
			NegativeGuardStatus:   "reject_before_allocation",
			OverflowGuardStatus:   "reject_before_allocation",
			ByteSize:              16,
			Escape:                allocplan.EscapeNoEscape,
			Storage:               allocplan.StorageStack,
			PlannedStorage:        allocplan.StorageStack,
			ActualLoweringStorage: allocplan.StorageStack,
			ValidationStatus:      "validated_no_escape",
			LoweringStatus:        "stack_lowering",
			Reason:                "p23 translation validation allocation witness",
		}},
	}}}
}

func p23AllocationProgram(name string, stackLowered bool) *ir.IRProgram {
	instrs := []ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 4},
	}
	if stackLowered {
		instrs = append(instrs, ir.IRInstr{Kind: ir.IRStackSliceI32, Local: 2, ArgSlots: 4, Imm: 4, Name: "xs"})
	} else {
		instrs = append(instrs, ir.IRInstr{Kind: ir.IRMakeSliceI32, Name: "xs"})
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRStoreLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRStoreLocal, Local: 0},
	)
	return &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       name,
		LocalSlots: 6,
		Instrs:     instrs,
	}}}
}

func p23CloneFuncs(funcs []ir.IRFunc) []ir.IRFunc {
	out := make([]ir.IRFunc, len(funcs))
	for i, fn := range funcs {
		out[i] = fn
		out[i].Instrs = append([]ir.IRInstr(nil), fn.Instrs...)
	}
	return out
}

func p23ProgramHasCall(prog *ir.IRProgram) bool {
	if prog == nil {
		return false
	}
	for _, fn := range prog.Funcs {
		for _, instr := range fn.Instrs {
			if instr.Kind == ir.IRCall {
				return true
			}
		}
	}
	return false
}
