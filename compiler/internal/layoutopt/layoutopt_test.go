package layoutopt

import (
	"reflect"
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestLayoutPolicyDefaultStructAllowsOptimizerFreedom(t *testing.T) {
	policy := PolicyForStruct(frontend.StructDecl{Name: "Point", Repr: frontend.StructReprDefault})
	if !policy.MayReorderFields || !policy.MayPackFields || !policy.MaySplitHotCold ||
		!policy.MayScalarReplace ||
		!policy.MayTransformAoSToSoA {
		t.Fatalf("default layout policy = %#v, want optimizer freedoms", policy)
	}
	if policy.ABILocked {
		t.Fatalf("default layout policy unexpectedly locked ABI: %#v", policy)
	}
}

func TestLayoutPolicyReprCNeverAllowsLayoutFreedom(t *testing.T) {
	policy := PolicyForStruct(frontend.StructDecl{Name: "Header", Repr: frontend.StructReprC})
	if !policy.ABILocked {
		t.Fatalf("repr(C) policy = %#v, want ABI locked", policy)
	}
	if policy.MayReorderFields || policy.MayPackFields || policy.MaySplitHotCold ||
		policy.MayScalarReplace ||
		policy.MayTransformAoSToSoA {
		t.Fatalf("repr(C) policy = %#v, want no layout freedoms", policy)
	}
}

func TestSpecializationDecisionRemovesGenericIdentityButPreservesSliceProvenance(t *testing.T) {
	identity := DecideSpecialization(SpecializationInput{
		Generic:      true,
		KnownTarget:  true,
		IdentityBody: true,
	})
	if identity.Action != SpecializationRemoveWrapper || identity.FallbackReason != "" {
		t.Fatalf("identity specialization = %#v, want remove-wrapper", identity)
	}

	wrapper := DecideSpecialization(SpecializationInput{
		Generic:          true,
		KnownTarget:      true,
		WrapperBody:      true,
		SliceProvenance:  true,
		ProvenanceErased: false,
	})
	if wrapper.Action != SpecializationInlineWrapper || !wrapper.PreservesProvenance {
		t.Fatalf("slice wrapper specialization = %#v, want inline preserving provenance", wrapper)
	}
}

func TestSpecializationDevirtualizesProtocolOnlyWhenTargetKnown(t *testing.T) {
	known := DecideSpecialization(SpecializationInput{
		Generic:      true,
		ProtocolCall: true,
		KnownTarget:  true,
	})
	if known.Action != SpecializationDevirtualize || known.FallbackReason != "" {
		t.Fatalf("known protocol specialization = %#v, want devirtualize", known)
	}

	unknown := DecideSpecialization(SpecializationInput{
		Generic:      true,
		ProtocolCall: true,
		KnownTarget:  false,
	})
	if unknown.Action != SpecializationFallback || unknown.FallbackReason == "" {
		t.Fatalf("unknown protocol specialization = %#v, want fallback reason", unknown)
	}
}

func TestSpecializationFallsBackBeforeErasingProofProvenanceOrABIFacts(t *testing.T) {
	for _, tc := range []SpecializationInput{
		{Generic: true, KnownTarget: true, ProofErased: true},
		{Generic: true, KnownTarget: true, ProvenanceErased: true},
		{Generic: true, KnownTarget: true, ABIFactsErased: true},
		{Generic: true, KnownTarget: true, UnsupportedEffects: true},
	} {
		t.Run(tc.FallbackToken(), func(t *testing.T) {
			decision := DecideSpecialization(tc)
			if decision.Action != SpecializationFallback || decision.FallbackReason == "" {
				t.Fatalf("decision = %#v, want fallback", decision)
			}
		})
	}
}

func TestEffectOptimizationFactsComeOnlyFromCheckerEnforcedEffects(t *testing.T) {
	facts := EffectFactsFromEnforcedEffects([]string{"io"})
	want := []string{"no_actor_send", "no_heap_allocation", "no_mem_write"}
	if !reflect.DeepEqual(facts, want) {
		t.Fatalf("facts = %#v, want %#v", facts, want)
	}
	if facts := EffectFactsFromEnforcedEffects([]string{"alloc", "actors", "mem"}); len(
		facts,
	) != 0 {
		t.Fatalf("effects with alloc/actors/mem yielded optimizer facts: %#v", facts)
	}
	if facts := EffectFactsFromEnforcedEffects(nil); len(facts) != 5 {
		t.Fatalf("pure no-uses function facts = %#v, want five enforced facts", facts)
	}
}
