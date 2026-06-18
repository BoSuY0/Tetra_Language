package policy

import (
	"testing"

	"tetra_language/compiler/internal/semantics/model"
)

func TestFirstForbiddenEffectUsesForbiddenOrder(t *testing.T) {
	have := map[string]struct{}{
		"runtime": {},
		"io":      {},
	}

	if got := FirstForbiddenEffect(have, []string{"actors", "io", "runtime"}); got != "io" {
		t.Fatalf("FirstForbiddenEffect = %q, want io", got)
	}
}

func TestFuncSigPolicyHelpers(t *testing.T) {
	sig := model.FuncSig{
		Effects:     []string{"surface", "alloc", "runtime"},
		HasNoAlloc:  true,
		HasNoBlock:  true,
		HasRealtime: true,
		HasBudget:   true,
	}

	if !FuncSigHasEffect(sig, "alloc") {
		t.Fatalf("FuncSigHasEffect did not find alloc")
	}
	if got := ActorTaskWorkerBoundaryEffect(sig); got != "alloc" {
		t.Fatalf("ActorTaskWorkerBoundaryEffect = %q, want alloc", got)
	}
	if !HasStrictSemanticCallClauses(sig) {
		t.Fatalf("HasStrictSemanticCallClauses = false, want true")
	}
	if got := FirstStrictSemanticCallClause(sig); got != "realtime" {
		t.Fatalf("FirstStrictSemanticCallClause = %q, want realtime", got)
	}
	if got := FirstFuncSigForbiddenEffect(sig, RealtimeForbiddenCallEffects); got != "alloc" {
		t.Fatalf("FirstFuncSigForbiddenEffect = %q, want alloc", got)
	}
}
