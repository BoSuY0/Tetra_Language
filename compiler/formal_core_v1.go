package compiler

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/differential"
	"tetra_language/compiler/internal/formalcore"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/runtimeabi"
	"tetra_language/compiler/internal/validation"
)

const (
	formalCoreV1Schema    = "tetra.formal_core.v1"
	formalCoreV1ScopeP232 = "p23.2_formal_core_v1"

	p23FormalCoreSpecWitnessID       = "formal_core_spec_inventory"
	p23FormalCoreValuesWitnessID     = "stable_value_differential_subset"
	p23FormalCorePLIRWitnessID       = "plir_borrow_copy_provenance_regions"
	p23FormalCoreProofWitnessID      = "bounds_proof_check_elimination"
	p23FormalCoreAllocationWitnessID = "allocation_length_intent_lowering"
	p23FormalCoreRawPointerWitnessID = "raw_pointer_bounds_metadata"
)

type FormalCoreV1ID string

const (
	FormalCoreV1Values                   FormalCoreV1ID = "values"
	FormalCoreV1BorrowsOwnedCopy         FormalCoreV1ID = "borrows_owned_copy"
	FormalCoreV1ProvenanceRegions        FormalCoreV1ID = "provenance_regions"
	FormalCoreV1BoundsProofIDSemantics   FormalCoreV1ID = "bounds_proof_id_semantics"
	FormalCoreV1AllocationLengthContract FormalCoreV1ID = "allocation_length_contract"
	FormalCoreV1AllocationIntentLowering FormalCoreV1ID = "allocation_intent_lowering"
	FormalCoreV1RawPointerBoundsMetadata FormalCoreV1ID = "raw_pointer_bounds_metadata"
	FormalCoreV1CheckEliminationValidity FormalCoreV1ID = "check_elimination_validity"
)

type FormalCoreV1Report struct {
	SchemaVersion                     string                `json:"schema_version"`
	Scope                             string                `json:"scope"`
	Rows                              []FormalCoreV1Row     `json:"rows"`
	Witnesses                         []FormalCoreV1Witness `json:"witnesses"`
	NonClaims                         []string              `json:"non_claims"`
	FormalSpecValid                   bool                  `json:"formal_spec_valid"`
	FormalConcepts                    int                   `json:"formal_concepts"`
	FormalRules                       int                   `json:"formal_rules"`
	ValueSamples                      int                   `json:"value_samples"`
	DifferentialLanes                 int                   `json:"differential_lanes"`
	BorrowCopyFacts                   bool                  `json:"borrow_copy_facts"`
	ProvenanceRegionFacts             bool                  `json:"provenance_region_facts"`
	BoundsProofIDsChecked             bool                  `json:"bounds_proof_ids_checked"`
	MissingProofRejected              bool                  `json:"missing_proof_rejected"`
	CheckEliminationValidated         bool                  `json:"check_elimination_validated"`
	AllocationLengthContractsChecked  bool                  `json:"allocation_length_contracts_checked"`
	InvalidAllocationLengthRejected   bool                  `json:"invalid_allocation_length_rejected"`
	AllocationIntentLoweringValidated bool                  `json:"allocation_intent_lowering_validated"`
	AllocationIntentDriftRejected     bool                  `json:"allocation_intent_drift_rejected"`
	RawPointerBoundsCases             int                   `json:"raw_pointer_bounds_cases"`
	RawPointerImpossibleAddRejected   bool                  `json:"raw_pointer_impossible_add_rejected"`
	RawPointerUnknownStayedChecked    bool                  `json:"raw_pointer_unknown_stayed_checked"`
	FullFormalProofClaimed            bool                  `json:"full_formal_proof_claimed"`
	BroadLanguageProofClaimed         bool                  `json:"broad_language_proof_claimed"`
	UnsafePolicyChanged               bool                  `json:"unsafe_policy_changed"`
	RuntimeBehaviorChanged            bool                  `json:"runtime_behavior_changed"`
	SafeSemanticsChanged              bool                  `json:"safe_semantics_changed"`
	PerformanceClaimed                bool                  `json:"performance_claimed"`
}

type FormalCoreV1Row struct {
	ID         FormalCoreV1ID `json:"id"`
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	Evidence   []string       `json:"evidence"`
	Tests      []string       `json:"tests"`
	Boundaries []string       `json:"boundaries"`
	WitnessIDs []string       `json:"witness_ids"`
}

type FormalCoreV1Witness struct {
	ID                                string `json:"id"`
	Kind                              string `json:"kind"`
	FormalSpecValid                   bool   `json:"formal_spec_valid,omitempty"`
	FormalConcepts                    int    `json:"formal_concepts,omitempty"`
	FormalRules                       int    `json:"formal_rules,omitempty"`
	ValueSamples                      int    `json:"value_samples,omitempty"`
	DifferentialLanes                 int    `json:"differential_lanes,omitempty"`
	BorrowCopyFacts                   bool   `json:"borrow_copy_facts,omitempty"`
	ProvenanceRegionFacts             bool   `json:"provenance_region_facts,omitempty"`
	BoundsProofIDsChecked             bool   `json:"bounds_proof_ids_checked,omitempty"`
	MissingProofRejected              bool   `json:"missing_proof_rejected,omitempty"`
	CheckEliminationValidated         bool   `json:"check_elimination_validated,omitempty"`
	AllocationLengthContractsChecked  bool   `json:"allocation_length_contracts_checked,omitempty"`
	InvalidAllocationLengthRejected   bool   `json:"invalid_allocation_length_rejected,omitempty"`
	AllocationIntentLoweringValidated bool   `json:"allocation_intent_lowering_validated,omitempty"`
	AllocationIntentDriftRejected     bool   `json:"allocation_intent_drift_rejected,omitempty"`
	RawPointerBoundsCases             int    `json:"raw_pointer_bounds_cases,omitempty"`
	RawPointerImpossibleAddRejected   bool   `json:"raw_pointer_impossible_add_rejected,omitempty"`
	RawPointerUnknownStayedChecked    bool   `json:"raw_pointer_unknown_stayed_checked,omitempty"`
}

func BuildP23FormalCoreV1Report() (FormalCoreV1Report, error) {
	spec, err := buildP23FormalCoreSpecWitness()
	if err != nil {
		return FormalCoreV1Report{}, err
	}
	values, err := buildP23FormalCoreValuesWitness()
	if err != nil {
		return FormalCoreV1Report{}, err
	}
	plirWitness, err := buildP23FormalCorePLIRWitness()
	if err != nil {
		return FormalCoreV1Report{}, err
	}
	proof, err := buildP23FormalCoreProofWitness()
	if err != nil {
		return FormalCoreV1Report{}, err
	}
	allocation, err := buildP23FormalCoreAllocationWitness()
	if err != nil {
		return FormalCoreV1Report{}, err
	}
	raw, err := buildP23FormalCoreRawPointerWitness()
	if err != nil {
		return FormalCoreV1Report{}, err
	}

	report := FormalCoreV1Report{
		SchemaVersion: formalCoreV1Schema,
		Scope:         formalCoreV1ScopeP232,
		Witnesses: []FormalCoreV1Witness{
			spec,
			values,
			plirWitness,
			proof,
			allocation,
			raw,
		},
		Rows: []FormalCoreV1Row{
			p23FormalCoreRow(FormalCoreV1Values, "Values", "current_supported_subset",
				[]string{
					"differential.CheckBackendMatrix confirms stable observable i32 values across supported source, Stack IR, optimized Stack IR, SSA, and Machine IR lanes.",
					"The P23.2 value witness reuses the loop-sum IR sample so values are checked by execution-equivalence evidence rather than prose.",
				},
				[]string{
					"go test ./compiler -run 'P23FormalCoreV1'",
					"go test ./compiler/internal/differential -run 'CheckBackendMatrix'",
				},
				[]string{
					"values evidence is limited to the current supported scalar i32 subset",
					"no public source interpreter mode is introduced",
				},
				[]string{p23FormalCoreValuesWitnessID}),
			p23FormalCoreRow(FormalCoreV1BorrowsOwnedCopy, "Borrows and owned/copy", "current_supported_subset",
				[]string{
					"plir.VerifyProgram accepts a real window().borrow().copy() program with borrowed_imm/no_escape facts for the borrow and owned/provenance_known facts for the copy.",
					"Borrow/copy evidence comes from compiler.Parse, compiler.Check, BuildPLIR, and PLIR fact inspection on supported source.",
				},
				[]string{
					"go test ./compiler/internal/plir -run 'BorrowCopy|PreservesIslandView'",
					"go test ./compiler -run 'P23FormalCoreV1'",
				},
				[]string{
					"borrow/copy evidence is bounded to current PLIR source facts",
					"unsafe lifetime relaxation is not claimed",
				},
				[]string{p23FormalCorePLIRWitnessID}),
			p23FormalCoreRow(FormalCoreV1ProvenanceRegions, "Provenance and regions", "current_supported_subset",
				[]string{
					"PLIR records island provenance and explicit regions for core.island_make_u8, derived window views, and borrowed views.",
					"plir.VerifyProgram rejects contradictory provenance and invalid region/borrow facts in nearby tests.",
				},
				[]string{
					"go test ./compiler/internal/plir -run 'Provenance|Region|PreservesIslandView'",
					"go test ./compiler -run 'P23FormalCoreV1'",
				},
				[]string{
					"region evidence is internal PLIR evidence, not a full region calculus",
					"external/unknown provenance remains conservative",
				},
				[]string{p23FormalCorePLIRWitnessID}),
			p23FormalCoreRow(FormalCoreV1BoundsProofIDSemantics, "Bounds proof id semantics", "current_supported_subset",
				[]string{
					"validation.CheckBoundsProofsWithPLIR accepts removed checks only when the unchecked IR proof id exists in PLIR proof guards.",
					"The proof witness rejects an unchecked load when the proof id is missing, preserving proof id semantics.",
				},
				[]string{
					"go test ./compiler/internal/validation -run 'CheckBoundsProofsWithPLIR'",
					"go test ./compiler -run 'P23FormalCoreV1'",
				},
				[]string{
					"bounds proof evidence covers current proof-tagged removed checks",
					"no broad theorem prover is claimed",
				},
				[]string{p23FormalCoreProofWitnessID}),
			p23FormalCoreRow(FormalCoreV1AllocationLengthContract, "Allocation length contract", "current_supported_subset",
				[]string{
					"allocplan.FromPLIR classifies zero, normal, negative, and overflow allocation length contract rows before storage evidence is trusted.",
					"The allocation witness requires rejected_negative_length and rejected_byte_size_overflow statuses from the real PLIR-to-allocplan path.",
				},
				[]string{
					"go test ./compiler/internal/allocplan -run 'Length'",
					"go test ./compiler -run 'P23FormalCoreV1'",
				},
				[]string{
					"length contract evidence is planner evidence, not a platform build/run claim",
					"runtime behavior does not change",
				},
				[]string{p23FormalCoreAllocationWitnessID}),
			p23FormalCoreRow(FormalCoreV1AllocationIntentLowering, "Allocation intent lowering", "current_supported_subset",
				[]string{
					"validation.ValidateAllocationLowering validates allocation intent rows against lowered IR stack and region allocation operations.",
					"The allocation witness also rejects a drifted IR program with missing matching stack allocation.",
				},
				[]string{
					"go test ./compiler/internal/validation -run 'ValidateAllocationLowering'",
					"go test ./compiler -run 'P23FormalCoreV1'",
				},
				[]string{
					"allocation intent evidence is bounded to current allocplan/lowering validators",
					"no broad allocation optimizer is claimed",
				},
				[]string{p23FormalCoreAllocationWitnessID}),
			p23FormalCoreRow(FormalCoreV1RawPointerBoundsMetadata, "Raw pointer bounds metadata", "current_supported_subset",
				[]string{
					"runtimeabi.NewRawAllocationBounds, DeriveRawPointerBounds, and RawSliceBoundsFromParts cover raw pointer bounds allocation-base metadata, derived offsets, checked external/unknown metadata, and impossible ptr_add rejection.",
					"Unknown raw pointer provenance remains checked external/unknown instead of forging an allocation root.",
				},
				[]string{
					"go test ./compiler/internal/runtimeabi -run 'RawPointerBounds'",
					"go test ./compiler -run 'P23FormalCoreV1'",
				},
				[]string{
					"raw pointer metadata is internal runtime ABI evidence",
					"unsafe policy does not change",
				},
				[]string{p23FormalCoreRawPointerWitnessID}),
			p23FormalCoreRow(FormalCoreV1CheckEliminationValidity, "Check-elimination validity", "current_supported_subset",
				[]string{
					"An unchecked lowered index operation is accepted only when validation.CheckBoundsProofsWithPLIR can match its proof id to PLIR proof guards.",
					"Missing proof ids are rejected, so check elimination stays proof-bound and safe-semantics preserving.",
				},
				[]string{
					"go test ./compiler/internal/validation -run 'CheckBoundsProofsWithPLIR|ValidateTranslationRejectsProof'",
					"go test ./compiler -run 'P23FormalCoreV1'",
				},
				[]string{
					"check-elimination validity is limited to current proof-tagged index operations",
					"safe-program semantics do not change",
				},
				[]string{p23FormalCoreProofWitnessID}),
		},
		NonClaims: []string{
			"no full formal proof of Tetra is claimed",
			"no broad language theorem prover is claimed",
			"no public source interpreter or backend selector is introduced",
			"unsafe policy does not change",
			"runtime behavior does not change",
			"safe-program semantics do not change",
			"no performance claim is made",
		},
		FormalSpecValid:                   spec.FormalSpecValid,
		FormalConcepts:                    spec.FormalConcepts,
		FormalRules:                       spec.FormalRules,
		ValueSamples:                      values.ValueSamples,
		DifferentialLanes:                 values.DifferentialLanes,
		BorrowCopyFacts:                   plirWitness.BorrowCopyFacts,
		ProvenanceRegionFacts:             plirWitness.ProvenanceRegionFacts,
		BoundsProofIDsChecked:             proof.BoundsProofIDsChecked,
		MissingProofRejected:              proof.MissingProofRejected,
		CheckEliminationValidated:         proof.CheckEliminationValidated,
		AllocationLengthContractsChecked:  allocation.AllocationLengthContractsChecked,
		InvalidAllocationLengthRejected:   allocation.InvalidAllocationLengthRejected,
		AllocationIntentLoweringValidated: allocation.AllocationIntentLoweringValidated,
		AllocationIntentDriftRejected:     allocation.AllocationIntentDriftRejected,
		RawPointerBoundsCases:             raw.RawPointerBoundsCases,
		RawPointerImpossibleAddRejected:   raw.RawPointerImpossibleAddRejected,
		RawPointerUnknownStayedChecked:    raw.RawPointerUnknownStayedChecked,
	}
	if err := ValidateP23FormalCoreV1Report(report); err != nil {
		return FormalCoreV1Report{}, err
	}
	return report, nil
}

func ValidateP23FormalCoreV1Report(report FormalCoreV1Report) error {
	if report.SchemaVersion != formalCoreV1Schema {
		return fmt.Errorf("formal core v1: schema_version is %q", report.SchemaVersion)
	}
	if report.Scope != formalCoreV1ScopeP232 {
		return fmt.Errorf("formal core v1: scope is %q", report.Scope)
	}
	if report.FullFormalProofClaimed {
		return fmt.Errorf("formal core v1: full formal proof claim is forbidden")
	}
	if report.BroadLanguageProofClaimed {
		return fmt.Errorf("formal core v1: broad language proof claim is forbidden")
	}
	if report.UnsafePolicyChanged {
		return fmt.Errorf("formal core v1: unsafe policy change claim is forbidden")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf("formal core v1: runtime behavior change claim is forbidden")
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf("formal core v1: safe semantics change claim is forbidden")
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("formal core v1: performance claim is forbidden")
	}
	if !report.FormalSpecValid || report.FormalConcepts < 9 || report.FormalRules < 7 {
		return fmt.Errorf("formal core v1: formal spec evidence incomplete")
	}
	if report.ValueSamples == 0 || report.DifferentialLanes < 5 {
		return fmt.Errorf("formal core v1: values evidence missing")
	}
	if !report.BorrowCopyFacts {
		return fmt.Errorf("formal core v1: borrow/copy facts missing")
	}
	if !report.ProvenanceRegionFacts {
		return fmt.Errorf("formal core v1: provenance/regions facts missing")
	}
	if !report.BoundsProofIDsChecked || !report.MissingProofRejected {
		return fmt.Errorf("formal core v1: bounds proof id evidence missing")
	}
	if !report.CheckEliminationValidated {
		return fmt.Errorf("formal core v1: check-elimination validity evidence missing")
	}
	if !report.AllocationLengthContractsChecked || !report.InvalidAllocationLengthRejected {
		return fmt.Errorf("formal core v1: allocation length contract evidence missing")
	}
	if !report.AllocationIntentLoweringValidated || !report.AllocationIntentDriftRejected {
		return fmt.Errorf("formal core v1: allocation intent lowering evidence missing")
	}
	if report.RawPointerBoundsCases < 4 || !report.RawPointerImpossibleAddRejected || !report.RawPointerUnknownStayedChecked {
		return fmt.Errorf("formal core v1: raw pointer bounds metadata evidence missing")
	}
	for _, want := range []string{
		"no full formal proof of Tetra is claimed",
		"no broad language theorem prover is claimed",
		"unsafe policy does not change",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p23FormalCoreHasString(report.NonClaims, want) {
			return fmt.Errorf("formal core v1: missing non-claim %q", want)
		}
	}
	if err := p23FormalCoreValidateRowsAndWitnesses(report.Rows, report.Witnesses); err != nil {
		return err
	}
	return nil
}

func buildP23FormalCoreSpecWitness() (FormalCoreV1Witness, error) {
	spec := formalcore.MinimumSpec()
	if err := formalcore.ValidateSpec(spec); err != nil {
		return FormalCoreV1Witness{}, err
	}
	return FormalCoreV1Witness{
		ID:              p23FormalCoreSpecWitnessID,
		Kind:            "formal_core_spec_inventory",
		FormalSpecValid: true,
		FormalConcepts:  len(spec.Concepts),
		FormalRules:     len(spec.Rules),
	}, nil
}

func buildP23FormalCoreValuesWitness() (FormalCoreV1Witness, error) {
	matrix, err := differential.CheckBackendMatrix(differential.BackendMatrixCase{
		Name:      "p23.2-formal-core-values-loop",
		Functions: []ir.IRFunc{p23LoopSumFunc()},
		Entry:     "sum_n",
		Samples: []differential.MatrixSample{
			{Name: "zero", Args: []int32{0}},
			{Name: "six", Args: []int32{6}},
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
		return FormalCoreV1Witness{}, err
	}
	return FormalCoreV1Witness{
		ID:                p23FormalCoreValuesWitnessID,
		Kind:              "stable_value_differential_subset",
		ValueSamples:      len(matrix.Samples),
		DifferentialLanes: len(matrix.Lanes),
	}, nil
}

func buildP23FormalCorePLIRWitness() (FormalCoreV1Witness, error) {
	src := []byte(`
func main() -> Int
uses alloc, islands, mem:
    island(64) as isl:
        var xs: []u8 = core.island_make_u8(isl, 4)
        let view: []u8 = xs.window(0, 2)
        let borrowed: []u8 = view.borrow()
        let copied: []u8 = borrowed.copy()
        return copied.len
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	checked, err := Check(prog)
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	plirProg, err := BuildPLIR(checked)
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	if err := plir.VerifyProgram(plirProg); err != nil {
		return FormalCoreV1Witness{}, err
	}
	var borrow, owned, provenance, region bool
	for _, fn := range plirProg.Funcs {
		for _, value := range fn.Values {
			if value.Provenance.Kind == plir.ProvenanceIsland && value.Region != "" {
				provenance = true
				region = true
			}
			if value.Provenance.Kind == plir.ProvenanceAllocation && value.Kind == plir.ValueAllocIntent {
				provenance = true
			}
		}
		for _, fact := range fn.Facts {
			switch fact.Kind {
			case plir.FactBorrowedImm:
				borrow = true
			case plir.FactOwned:
				owned = true
			case plir.FactRegionAlive:
				region = true
			case plir.FactProvenanceKnown:
				provenance = true
			}
		}
	}
	return FormalCoreV1Witness{
		ID:                    p23FormalCorePLIRWitnessID,
		Kind:                  "plir_borrow_copy_provenance_regions",
		BorrowCopyFacts:       borrow && owned,
		ProvenanceRegionFacts: provenance && region,
	}, nil
}

func buildP23FormalCoreProofWitness() (FormalCoreV1Witness, error) {
	proofID := "proof:while:i:xs:1:1"
	report, err := validation.CheckBoundsProofsWithPLIR(p23ProofProgram("main", proofID), p23FormalCoreProofPLIR(proofID))
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	_, badErr := validation.CheckBoundsProofsWithPLIR(p23ProofProgram("main", ""), p23FormalCoreProofPLIR(proofID))
	return FormalCoreV1Witness{
		ID:                        p23FormalCoreProofWitnessID,
		Kind:                      "bounds_proof_check_elimination",
		BoundsProofIDsChecked:     len(report.RemovedChecks) > 0,
		MissingProofRejected:      badErr != nil && strings.Contains(badErr.Error(), "without proof id"),
		CheckEliminationValidated: len(report.RemovedChecks) > 0,
	}, nil
}

func buildP23FormalCoreAllocationWitness() (FormalCoreV1Witness, error) {
	src := []byte(`
func empty() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(0)
    return xs.len

func normal() -> Int
uses alloc, mem:
    var xs: []u16 = make_u16(3)
    return xs.len

func negative() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(0 - 1)
    return xs.len

func overflow() -> Int
uses alloc, mem:
    var xs: []bool = make_bool(536870912)
    return xs.len

func main() -> Int
uses alloc, mem:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	checked, err := Check(prog)
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	plirProg, err := BuildPLIR(checked)
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	plan, err := allocplan.FromPLIR(plirProg)
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	var empty, normal, negative, overflow bool
	for _, fn := range plan.Functions {
		for _, allocation := range fn.Allocations {
			switch allocation.LengthStatus {
			case allocplan.LengthStatusValidEmpty:
				empty = true
			case allocplan.LengthStatusNormal:
				normal = true
			case allocplan.LengthStatusRejectedNegative:
				negative = true
			case allocplan.LengthStatusRejectedOverflow:
				overflow = true
			}
		}
	}
	allocationWitness, err := buildP23AllocationWitness()
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	return FormalCoreV1Witness{
		ID:                                p23FormalCoreAllocationWitnessID,
		Kind:                              "allocation_length_intent_lowering",
		AllocationLengthContractsChecked:  empty && normal,
		InvalidAllocationLengthRejected:   negative && overflow,
		AllocationIntentLoweringValidated: allocationWitness.AllocationPlanValidated,
		AllocationIntentDriftRejected:     allocationWitness.AllocationDriftRejected,
	}, nil
}

func buildP23FormalCoreRawPointerWitness() (FormalCoreV1Witness, error) {
	root, err := runtimeabi.NewRawAllocationBounds("p23.2-root", 16)
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	derived, diag := runtimeabi.DeriveRawPointerBounds(root, 4, 4)
	if diag != nil {
		return FormalCoreV1Witness{}, fmt.Errorf("formal core v1: unexpected raw pointer diagnostic: %+v", diag)
	}
	rejected, rejectedDiag := runtimeabi.DeriveRawPointerBounds(root, 16, 1)
	unknown := runtimeabi.UnknownRawPointerBounds("ffi pointer")
	unknownDerived, unknownDiag := runtimeabi.DeriveRawPointerBounds(unknown, 4, 1)
	if unknownDiag != nil {
		return FormalCoreV1Witness{}, fmt.Errorf("formal core v1: unknown raw pointer returned diagnostic: %+v", unknownDiag)
	}
	verifiedSlice := runtimeabi.RawSliceBoundsFromParts(derived, 2, 4)
	unknownSlice := runtimeabi.RawSliceBoundsFromParts(unknownDerived, 2, 4)
	cases := 0
	for _, ok := range []bool{
		root.Status == runtimeabi.RawPointerBoundsAllocationBase && root.VerifiedAllocationRoot,
		derived.Status == runtimeabi.RawPointerBoundsDerivedOffset && derived.VerifiedAllocationRoot,
		rejected.Status == runtimeabi.RawPointerBoundsRejected && rejectedDiag != nil,
		unknownDerived.Status == runtimeabi.RawPointerBoundsCheckedExternalUnknown && !unknownDerived.VerifiedAllocationRoot,
		verifiedSlice.Status == runtimeabi.RawSliceBoundsVerifiedAllocationRoot && verifiedSlice.VerifiedAllocationRoot,
		unknownSlice.Status == runtimeabi.RawSliceBoundsExternalUnknown && !unknownSlice.VerifiedAllocationRoot,
	} {
		if ok {
			cases++
		}
	}
	return FormalCoreV1Witness{
		ID:                              p23FormalCoreRawPointerWitnessID,
		Kind:                            "raw_pointer_bounds_metadata",
		RawPointerBoundsCases:           cases,
		RawPointerImpossibleAddRejected: rejected.Status == runtimeabi.RawPointerBoundsRejected && rejectedDiag != nil,
		RawPointerUnknownStayedChecked:  unknownDerived.Status == runtimeabi.RawPointerBoundsCheckedExternalUnknown && unknownSlice.Status == runtimeabi.RawSliceBoundsExternalUnknown,
	}, nil
}

func p23FormalCoreProofPLIR(proofID string) *plir.Program {
	return &plir.Program{Funcs: []plir.Function{{
		Name: "main",
		Values: []plir.Value{{
			ID:         "param:xs",
			Kind:       plir.ValueParam,
			Type:       "[]i32",
			Region:     "fn:main",
			Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "xs"},
			Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "xs"},
			Borrow:     plir.BorrowImm,
			Escape:     plir.EscapeNoEscape,
		}},
		Blocks: []plir.BasicBlock{
			{ID: "entry", Kind: "entry", Entry: true, Succs: []string{"body"}},
			{ID: "body", Kind: "while_body", Preds: []string{"entry"}, Ops: []string{"op0"}, Exit: true},
		},
		Ops: []plir.Operation{
			{ID: "op0", Kind: plir.OpIndexLoad, Block: "body"},
		},
		Facts: []plir.Fact{
			{ID: "known", Kind: plir.FactProvenanceKnown, ValueID: "param:xs"},
			{ID: "len", Kind: plir.FactLenStable, ValueID: "param:xs"},
			{ID: "range", Kind: plir.FactIndexInRange, ValueID: "param:xs", Range: "0..xs.len", ProofID: proofID, Source: "formal-core:1:1"},
		},
		ProofGuards: []plir.ProofGuard{{
			ID:        proofID,
			Kind:      "range",
			Block:     "body",
			OpID:      "op0",
			Condition: "0 <= i < xs.len",
			Reason:    "formal core proof witness",
		}},
		ProofUses: []plir.ProofUse{{
			ProofID: proofID,
			Block:   "body",
			OpID:    "op0",
			UseKind: "bounds_check",
			Source:  "formal-core:1:1",
		}},
	}}}
}

func p23FormalCoreValidateRowsAndWitnesses(rows []FormalCoreV1Row, witnesses []FormalCoreV1Witness) error {
	witnessIDs := map[string]bool{}
	for _, witness := range witnesses {
		if strings.TrimSpace(witness.ID) == "" {
			return fmt.Errorf("formal core v1: witness missing id")
		}
		witnessIDs[witness.ID] = true
	}
	seen := map[FormalCoreV1ID]bool{}
	expected := map[FormalCoreV1ID]bool{}
	for _, id := range p23FormalCoreV1IDs() {
		expected[id] = true
	}
	for _, row := range rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 || len(row.WitnessIDs) == 0 {
			return fmt.Errorf("formal core v1: row %q missing required metadata", row.ID)
		}
		if !expected[row.ID] {
			return fmt.Errorf("formal core v1: unexpected row %s", row.ID)
		}
		if seen[row.ID] {
			return fmt.Errorf("formal core v1: duplicate row %s", row.ID)
		}
		seen[row.ID] = true
		if p23ContainsPlaceholder(row.Evidence) || p23ContainsPlaceholder(row.Boundaries) {
			return fmt.Errorf("formal core v1: row %s contains placeholder evidence", row.ID)
		}
		for _, witnessID := range row.WitnessIDs {
			if !witnessIDs[witnessID] {
				return fmt.Errorf("formal core v1: row %s references missing witness %q", row.ID, witnessID)
			}
		}
	}
	for _, id := range p23FormalCoreV1IDs() {
		if !seen[id] {
			return fmt.Errorf("formal core v1: missing row %s", id)
		}
	}
	return nil
}

func p23FormalCoreV1IDs() []FormalCoreV1ID {
	return []FormalCoreV1ID{
		FormalCoreV1Values,
		FormalCoreV1BorrowsOwnedCopy,
		FormalCoreV1ProvenanceRegions,
		FormalCoreV1BoundsProofIDSemantics,
		FormalCoreV1AllocationLengthContract,
		FormalCoreV1AllocationIntentLowering,
		FormalCoreV1RawPointerBoundsMetadata,
		FormalCoreV1CheckEliminationValidity,
	}
}

func p23FormalCoreRow(id FormalCoreV1ID, name string, status string, evidence []string, tests []string, boundaries []string, witnessIDs []string) FormalCoreV1Row {
	return FormalCoreV1Row{
		ID:         id,
		Name:       name,
		Status:     status,
		Evidence:   append([]string(nil), evidence...),
		Tests:      append([]string(nil), tests...),
		Boundaries: append([]string(nil), boundaries...),
		WitnessIDs: append([]string(nil), witnessIDs...),
	}
}

func p23FormalCoreHasString(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
