package compiler

import (
	"testing"

	"tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/memorypipeline"
)

func TestT13ReleaseOptimizeAdvancesCanonicalMemoryStateThroughOptimizer(t *testing.T) {
	checked := t13CheckedProgram(t, "fun main(): i32 { return 1 }\n")
	state, err := buildMemoryStateForTarget(checked, "linux-x64")
	if err != nil {
		t.Fatalf("buildMemoryStateForTarget: %v", err)
	}

	result, err := lowerMemoryStateForBuild(
		checked,
		state,
		"linux-x64",
		BuildOptions{ReleaseOptimize: true},
		nil,
	)
	if err != nil {
		t.Fatalf("lowerMemoryStateForBuild: %v", err)
	}
	if result == nil || result.Program == nil {
		t.Fatalf("missing optimized lowering result: %#v", result)
	}
	if state.Phase != memorypipeline.PhaseOptimized {
		t.Fatalf("state phase = %q, want optimized", state.Phase)
	}
	if !t13HasOptimizerFact(t, state, memoryfacts.ClaimOptimizerPass) {
		t.Fatalf("release optimize did not record optimizer pass facts")
	}
}

func TestT13NonReleaseAdvancesWithoutFabricatedOptimizerFacts(t *testing.T) {
	checked := t13CheckedProgram(t, "fun main(): i32 { return 1 }\n")
	state, err := buildMemoryStateForTarget(checked, "linux-x64")
	if err != nil {
		t.Fatalf("buildMemoryStateForTarget: %v", err)
	}

	if _, err := lowerMemoryStateForBuild(checked, state, "linux-x64", BuildOptions{}, nil); err != nil {
		t.Fatalf("lowerMemoryStateForBuild: %v", err)
	}
	if state.Phase != memorypipeline.PhaseOptimized {
		t.Fatalf("state phase = %q, want optimized skip marker", state.Phase)
	}
	if t13HasOptimizerFact(t, state, memoryfacts.ClaimOptimizerPass) ||
		t13HasOptimizerFact(t, state, memoryfacts.ClaimOptimizerDecision) {
		t.Fatalf("non-release build fabricated optimizer facts")
	}
}

func t13CheckedProgram(t *testing.T, src string) *CheckedProgram {
	t.Helper()
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return checked
}

func t13HasOptimizerFact(
	t *testing.T,
	state *memorypipeline.State,
	claim memoryfacts.Claim,
) bool {
	t.Helper()
	snapshot, err := state.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	for _, fact := range snapshot.Facts() {
		if fact.SourceStage == memoryfacts.StageOptimization && fact.Claim == claim {
			return true
		}
	}
	return false
}
