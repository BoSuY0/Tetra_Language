package policy

import "tetra_language/compiler/internal/semantics/model"

var NoblockForbiddenCallEffects = []string{"actors", "control", "io", "link", "mmio", "runtime"}

var RealtimeForbiddenCallEffects = []string{"actors", "alloc", "control", "io", "link", "mmio", "runtime"}

func FirstForbiddenEffect(have map[string]struct{}, forbidden []string) string {
	for _, effect := range forbidden {
		if _, ok := have[effect]; ok {
			return effect
		}
	}
	return ""
}

func FuncSigHasEffect(sig model.FuncSig, effect string) bool {
	for _, name := range sig.Effects {
		if name == effect {
			return true
		}
	}
	return false
}

func ActorTaskWorkerBoundaryEffect(sig model.FuncSig) string {
	for _, effect := range sig.Effects {
		switch effect {
		case "alloc", "capability", "control", "islands", "link", "mem", "mmio", "privacy":
			return effect
		}
	}
	return ""
}

func FirstFuncSigForbiddenEffect(sig model.FuncSig, forbidden []string) string {
	effects := EffectSet(sig.Effects)
	return FirstForbiddenEffect(effects, forbidden)
}

func HasStrictSemanticCallClauses(sig model.FuncSig) bool {
	return sig.HasNoAlloc || sig.HasNoBlock || sig.HasRealtime || sig.HasBudget
}

func FirstStrictSemanticCallClause(sig model.FuncSig) string {
	if sig.HasRealtime {
		return "realtime"
	}
	if sig.HasNoAlloc {
		return "noalloc"
	}
	if sig.HasNoBlock {
		return "noblock"
	}
	if sig.HasBudget {
		return "budget"
	}
	return ""
}
