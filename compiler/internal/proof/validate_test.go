package proof

import (
	"strings"
	"testing"
)

func TestProofStoreRejectsMissingProofID(t *testing.T) {
	store := NewStore()
	err := store.ValidateReferences([]Reference{{ID: "proof:missing", Subject: Subject{Kind: "allocation", ID: "alloc:main:1"}}})
	if err == nil || !strings.Contains(err.Error(), "missing proof id") {
		t.Fatalf("ValidateReferences error = %v, want missing proof id", err)
	}
}

func TestProofStoreRejectsWrongSubject(t *testing.T) {
	term := NewTerm(Term{
		ID:             "proof:alloc:stack",
		Kind:           KindAllocationPlacement,
		Subject:        Subject{Kind: "allocation", ID: "alloc:main:1"},
		DerivationRule: "validated_no_escape",
		ProducerPass:   "allocplan",
		Status:         StatusProven,
	})
	store := NewStore(term)
	err := store.ValidateReferences([]Reference{{ID: term.ID, Subject: Subject{Kind: "allocation", ID: "alloc:other"}}})
	if err == nil || !strings.Contains(err.Error(), "subject mismatch") {
		t.Fatalf("ValidateReferences error = %v, want subject mismatch", err)
	}
}

func TestProofStoreRejectsStaleStableHash(t *testing.T) {
	term := NewTerm(Term{
		ID:             "proof:alloc:stack",
		Kind:           KindAllocationPlacement,
		Subject:        Subject{Kind: "allocation", ID: "alloc:main:1"},
		DerivationRule: "validated_no_escape",
		ProducerPass:   "allocplan",
		Status:         StatusProven,
	})
	term.DerivationRule = "changed after hash"
	store := NewStore(term)
	err := store.Validate()
	if err == nil || !strings.Contains(err.Error(), "stale stable_hash") {
		t.Fatalf("Validate error = %v, want stale stable_hash", err)
	}
}

func TestProofStoreRejectsUnsafeUnknownPromotion(t *testing.T) {
	term := NewTerm(Term{
		ID:             "proof:unsafe",
		Kind:           KindAllocationPlacement,
		Subject:        Subject{Kind: "allocation", ID: "alloc:main:unsafe"},
		Assumptions:    []string{"unsafe_unknown"},
		DerivationRule: "unsafe_unknown_promoted_to_stack",
		ProducerPass:   "allocplan",
		Status:         StatusProven,
	})
	store := NewStore(term)
	err := store.Validate()
	if err == nil || !strings.Contains(err.Error(), "unsafe_unknown") {
		t.Fatalf("Validate error = %v, want unsafe_unknown promotion rejection", err)
	}
}
