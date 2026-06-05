package formalcore

import (
	"fmt"
	"strings"
)

type Concept string

const (
	ConceptValues                   Concept = "values"
	ConceptProvenance               Concept = "provenance"
	ConceptRegions                  Concept = "regions"
	ConceptBorrowCopy               Concept = "borrow_copy"
	ConceptBoundsProof              Concept = "bounds_proof"
	ConceptAllocationIntent         Concept = "allocation_intent"
	ConceptAllocationLengthContract Concept = "allocation_length_contract"
	ConceptRawPointerBoundsMetadata Concept = "raw_pointer_bounds_metadata"
	ConceptCheckEliminationValidity Concept = "check_elimination_validity"
)

type Spec struct {
	SchemaVersion string    `json:"schema_version"`
	Concepts      []Concept `json:"concepts"`
	Rules         []Rule    `json:"rules"`
}

type Rule struct {
	Name         string    `json:"name"`
	Concepts     []Concept `json:"concepts"`
	Statement    string    `json:"statement"`
	MachineCheck string    `json:"machine_check"`
}

func MinimumSpec() Spec {
	return Spec{
		SchemaVersion: "tetra.formal_core.v1",
		Concepts: []Concept{
			ConceptValues,
			ConceptProvenance,
			ConceptRegions,
			ConceptBorrowCopy,
			ConceptBoundsProof,
			ConceptAllocationIntent,
			ConceptAllocationLengthContract,
			ConceptRawPointerBoundsMetadata,
			ConceptCheckEliminationValidity,
		},
		Rules: []Rule{
			{
				Name:         "values-have-stable-observable-result",
				Concepts:     []Concept{ConceptValues},
				Statement:    "A supported stable i32 value has one observable result across source, Stack IR, optimized Stack IR, SSA, Machine IR, and native execution evidence lanes.",
				MachineCheck: "compiler/internal/differential.CheckBackendMatrix",
			},
			{
				Name:         "proof-before-check-elimination",
				Concepts:     []Concept{ConceptBoundsProof, ConceptCheckEliminationValidity},
				Statement:    "An unchecked lowered index load is valid only when it keeps a proof id whose guard dominates the use.",
				MachineCheck: "compiler/internal/validation.CheckBoundsProofsWithPLIR",
			},
			{
				Name:         "allocation-intent-preserves-provenance",
				Concepts:     []Concept{ConceptAllocationIntent, ConceptProvenance, ConceptRegions},
				Statement:    "A storage plan may change representation only when the allocation intent, provenance, and validation evidence still match the lowered IR.",
				MachineCheck: "compiler/internal/validation.ValidateAllocationLowering",
			},
			{
				Name:         "regions-remain-explicit-on-memory-values",
				Concepts:     []Concept{ConceptRegions, ConceptProvenance, ConceptBorrowCopy},
				Statement:    "Region-bearing memory values must keep explicit region identities through provenance-preserving views and borrows.",
				MachineCheck: "compiler/internal/plir.VerifyProgram",
			},
			{
				Name:         "allocation-length-contract-before-storage",
				Concepts:     []Concept{ConceptAllocationIntent, ConceptAllocationLengthContract},
				Statement:    "Allocation length contracts classify valid empty, normal, negative, overflow, and invalid lengths before any storage claim is trusted.",
				MachineCheck: "compiler/internal/allocplan.VerifyPlan",
			},
			{
				Name:         "raw-pointer-bounds-stay-metadata-bound",
				Concepts:     []Concept{ConceptRawPointerBoundsMetadata, ConceptProvenance},
				Statement:    "Raw pointer arithmetic may keep allocation-base metadata, derive checked offsets, reject impossible offsets, or remain checked external/unknown without forging provenance.",
				MachineCheck: "compiler/internal/runtimeabi.RawPointerBoundsMetadata",
			},
			{
				Name:         "borrow-copy-does-not-forge-ownership",
				Concepts:     []Concept{ConceptBorrowCopy, ConceptProvenance, ConceptRegions},
				Statement:    "Borrowed views preserve source provenance, while copy creates owned provenance and cannot erase required escape checks.",
				MachineCheck: "compiler/internal/plir.VerifyProgram",
			},
		},
	}
}

func (s Spec) Covers(concept Concept) bool {
	for _, got := range s.Concepts {
		if got == concept {
			return true
		}
	}
	return false
}

func ValidateSpec(spec Spec) error {
	if spec.SchemaVersion != "tetra.formal_core.v1" {
		return fmt.Errorf("formal core: schema_version is %q, want tetra.formal_core.v1", spec.SchemaVersion)
	}
	for _, concept := range []Concept{
		ConceptValues,
		ConceptProvenance,
		ConceptRegions,
		ConceptBorrowCopy,
		ConceptBoundsProof,
		ConceptAllocationIntent,
		ConceptAllocationLengthContract,
		ConceptRawPointerBoundsMetadata,
		ConceptCheckEliminationValidity,
	} {
		if !spec.Covers(concept) {
			return fmt.Errorf("formal core: missing concept %s", concept)
		}
	}
	if len(spec.Rules) == 0 {
		return fmt.Errorf("formal core: missing rules")
	}
	known := map[Concept]bool{}
	for _, concept := range spec.Concepts {
		known[concept] = true
	}
	for _, rule := range spec.Rules {
		if strings.TrimSpace(rule.Name) == "" {
			return fmt.Errorf("formal core: rule is missing name")
		}
		if strings.TrimSpace(rule.Statement) == "" {
			return fmt.Errorf("formal core: rule %q is missing statement", rule.Name)
		}
		if strings.TrimSpace(rule.MachineCheck) == "" {
			return fmt.Errorf("formal core: rule %q is not machine-checkable", rule.Name)
		}
		if len(rule.Concepts) == 0 {
			return fmt.Errorf("formal core: rule %q has no concepts", rule.Name)
		}
		for _, concept := range rule.Concepts {
			if !known[concept] {
				return fmt.Errorf("formal core: rule %q references unknown concept %s", rule.Name, concept)
			}
		}
	}
	return nil
}
