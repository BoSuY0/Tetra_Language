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

func TestProofStoreRejectsStaleStableHashForSemanticFields(t *testing.T) {
	base := func() Term {
		return NewTerm(Term{
			ID:                 "proof:alloc:stack",
			Kind:               KindAllocationPlacement,
			Subject:            Subject{Kind: "allocation", ID: "alloc:main:1"},
			Assumptions:        []string{"escape_status=no_escape"},
			DerivationRule:     "validated_no_escape",
			SourceSpan:         "main.tetra:10:3",
			ASTID:              "ast:alloc:1",
			PLIROpID:           "plir:alloc:1",
			IROpID:             "ir:alloc:1",
			DominanceScope:     "dom:entry->alloc",
			LifetimeScope:      "lifetime:frame:main",
			MutationEpoch:      "mutation:0",
			AliasEpoch:         "alias:0",
			InvalidationPolicy: "invalidate_on_escape_or_mutation",
			ProducerPass:       "allocplan",
			ConsumerPasses:     []string{"ram-contract", "validation"},
			Status:             StatusProven,
		})
	}

	tests := []struct {
		name   string
		mutate func(*Term)
	}{
		{name: "dominance_scope", mutate: func(term *Term) { term.DominanceScope = "dom:changed" }},
		{name: "lifetime_scope", mutate: func(term *Term) { term.LifetimeScope = "lifetime:changed" }},
		{name: "mutation_epoch", mutate: func(term *Term) { term.MutationEpoch = "mutation:1" }},
		{name: "alias_epoch", mutate: func(term *Term) { term.AliasEpoch = "alias:1" }},
		{name: "invalidation_policy", mutate: func(term *Term) { term.InvalidationPolicy = "never_invalidate" }},
		{name: "consumer_passes", mutate: func(term *Term) { term.ConsumerPasses = append(term.ConsumerPasses, "lowering") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			term := base()
			tt.mutate(&term)
			store := NewStore(term)
			err := store.Validate()
			if err == nil || !strings.Contains(err.Error(), "stale stable_hash") {
				t.Fatalf("Validate error = %v, want stale stable_hash", err)
			}
		})
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
