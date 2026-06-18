package layoutopt

import "tetra_language/compiler/internal/frontend"

type LayoutPolicy struct {
	Repr                 string
	ABILocked            bool
	MayReorderFields     bool
	MayPackFields        bool
	MaySplitHotCold      bool
	MayScalarReplace     bool
	MayTransformAoSToSoA bool
	ConservativeReason   string
}

func PolicyForStruct(st frontend.StructDecl) LayoutPolicy {
	repr := st.Repr
	if repr == "" {
		repr = frontend.StructReprDefault
	}
	if repr == frontend.StructReprC {
		return LayoutPolicy{
			Repr:               repr,
			ABILocked:          true,
			ConservativeReason: "repr(C) fixes declaration order and C ABI layout",
		}
	}
	return LayoutPolicy{
		Repr:                 repr,
		MayReorderFields:     true,
		MayPackFields:        true,
		MaySplitHotCold:      true,
		MayScalarReplace:     true,
		MayTransformAoSToSoA: true,
	}
}

type SpecializationAction string

const (
	SpecializationKeep          SpecializationAction = "keep"
	SpecializationRemoveWrapper SpecializationAction = "remove_wrapper"
	SpecializationInlineWrapper SpecializationAction = "inline_wrapper"
	SpecializationDevirtualize  SpecializationAction = "devirtualize"
	SpecializationFallback      SpecializationAction = "fallback"
)

type SpecializationInput struct {
	Generic            bool
	KnownTarget        bool
	IdentityBody       bool
	WrapperBody        bool
	SliceProvenance    bool
	ProtocolCall       bool
	ProofErased        bool
	ProvenanceErased   bool
	ABIFactsErased     bool
	UnsupportedEffects bool
}

func (in SpecializationInput) FallbackToken() string {
	switch {
	case in.ProofErased:
		return "proof_erased"
	case in.ProvenanceErased:
		return "provenance_erased"
	case in.ABIFactsErased:
		return "abi_facts_erased"
	case in.UnsupportedEffects:
		return "unsupported_effects"
	case !in.KnownTarget:
		return "unknown_target"
	default:
		return "no_fallback"
	}
}

type SpecializationDecision struct {
	Action              SpecializationAction
	FallbackReason      string
	PreservesProvenance bool
}

func DecideSpecialization(in SpecializationInput) SpecializationDecision {
	if !in.Generic {
		return SpecializationDecision{Action: SpecializationKeep}
	}
	if !in.KnownTarget {
		return specializationFallback("unknown dispatch target")
	}
	if in.ProofErased {
		return specializationFallback("would erase proof facts")
	}
	if in.ProvenanceErased {
		return specializationFallback("would erase provenance facts")
	}
	if in.ABIFactsErased {
		return specializationFallback("would erase ABI facts")
	}
	if in.UnsupportedEffects {
		return specializationFallback("unsupported effect facts")
	}
	if in.ProtocolCall {
		return SpecializationDecision{Action: SpecializationDevirtualize}
	}
	if in.IdentityBody {
		return SpecializationDecision{
			Action:              SpecializationRemoveWrapper,
			PreservesProvenance: true,
		}
	}
	if in.WrapperBody {
		return SpecializationDecision{
			Action:              SpecializationInlineWrapper,
			PreservesProvenance: in.SliceProvenance,
		}
	}
	return SpecializationDecision{Action: SpecializationKeep, PreservesProvenance: true}
}

func specializationFallback(reason string) SpecializationDecision {
	return SpecializationDecision{Action: SpecializationFallback, FallbackReason: reason}
}

type EffectFactOptions struct {
	TouchesMutableGlobals bool
}

func EffectFactsFromEnforcedEffects(effects []string) []string {
	return EffectFactsFromEnforcedEffectsOpt(effects, EffectFactOptions{})
}

func EffectFactsFromEnforcedEffectsOpt(effects []string, opt EffectFactOptions) []string {
	declared := map[string]bool{}
	for _, effect := range effects {
		declared[effect] = true
	}
	out := []string{}
	if len(declared) == 0 && !opt.TouchesMutableGlobals {
		out = append(out, "pure_call")
	}
	if !declared["actors"] {
		out = append(out, "no_actor_send")
	}
	if !declared["alloc"] && !declared["islands"] {
		out = append(out, "no_heap_allocation")
	}
	if !declared["mem"] && !declared["mmio"] {
		out = append(out, "no_mem_write")
	}
	if len(declared) == 0 && !opt.TouchesMutableGlobals {
		out = append(out, "no_unknown_escape")
	}
	return out
}
