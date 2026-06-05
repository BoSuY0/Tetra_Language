package formalcore

import (
	"strings"
	"testing"
)

func TestMinimumSpecCoversP11FormalCoreConcepts(t *testing.T) {
	spec := MinimumSpec()
	if err := ValidateSpec(spec); err != nil {
		t.Fatalf("ValidateSpec(MinimumSpec): %v", err)
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
			t.Fatalf("MinimumSpec missing concept %s: %+v", concept, spec)
		}
	}
}

func TestValidateSpecRejectsMissingCheckEliminationValidity(t *testing.T) {
	spec := MinimumSpec()
	spec.Concepts = withoutConcept(spec.Concepts, ConceptCheckEliminationValidity)
	err := ValidateSpec(spec)
	if err == nil || !strings.Contains(err.Error(), "check_elimination_validity") {
		t.Fatalf("ValidateSpec error = %v, want check elimination validity failure", err)
	}
}

func TestValidateSpecRejectsMissingRegions(t *testing.T) {
	spec := MinimumSpec()
	spec.Concepts = withoutConcept(spec.Concepts, ConceptRegions)
	err := ValidateSpec(spec)
	if err == nil || !strings.Contains(err.Error(), "regions") {
		t.Fatalf("ValidateSpec error = %v, want regions failure", err)
	}
}

func TestValidateSpecRejectsMissingRawPointerBoundsMetadata(t *testing.T) {
	spec := MinimumSpec()
	spec.Concepts = withoutConcept(spec.Concepts, ConceptRawPointerBoundsMetadata)
	err := ValidateSpec(spec)
	if err == nil || !strings.Contains(err.Error(), "raw_pointer_bounds_metadata") {
		t.Fatalf("ValidateSpec error = %v, want raw pointer bounds metadata failure", err)
	}
}

func TestValidateSpecRejectsRuleWithoutMachineCheck(t *testing.T) {
	spec := MinimumSpec()
	spec.Rules[0].MachineCheck = ""
	err := ValidateSpec(spec)
	if err == nil || !strings.Contains(err.Error(), "machine-checkable") {
		t.Fatalf("ValidateSpec error = %v, want machine-checkable failure", err)
	}
}

func withoutConcept(values []Concept, drop Concept) []Concept {
	out := values[:0]
	for _, value := range values {
		if value != drop {
			out = append(out, value)
		}
	}
	return out
}
