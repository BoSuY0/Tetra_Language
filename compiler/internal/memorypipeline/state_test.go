package memorypipeline

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/semantics"
)

var (
	programIDPattern      = regexp.MustCompile(`^program:sha256:[0-9a-f]{64}$`)
	memoryPlanIDPattern   = regexp.MustCompile(`^memory-plan:sha256:[0-9a-f]{64}$`)
	memoryPipelineFixture = `
func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    xs[0] = 1
    return 0
`
)

func TestBuildCreatesPlannedStateWithoutReportOptions(t *testing.T) {
	state, err := Build(checkedProgram(t, memoryPipelineFixture), Options{
		Target: "linux-x64",
		AllocPlan: allocplan.Options{
			EnableStackLowering: true,
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if state.Phase != PhasePlanned {
		t.Fatalf("Phase = %q, want %q", state.Phase, PhasePlanned)
	}
	if !programIDPattern.MatchString(state.ProgramID) {
		t.Fatalf("ProgramID = %q, want program sha256 digest", state.ProgramID)
	}
	if state.Target != "linux-x64" || state.PLIR == nil || state.Graph == nil || state.Plan == nil {
		t.Fatalf("incomplete state: %#v", state)
	}
	if got := state.Graph.CurrentStage(); got != memoryfacts.StageAllocPlan {
		t.Fatalf("graph stage = %q, want %q", got, memoryfacts.StageAllocPlan)
	}
	snapshot, err := state.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snapshot.ProgramID() != state.ProgramID {
		t.Fatalf("snapshot ProgramID = %q, want %q", snapshot.ProgramID(), state.ProgramID)
	}
	assertOptionsDoNotExposeReportFlags(t)
}

func TestProgramIDDeterministicPermutationInvariantAndOptionSensitive(t *testing.T) {
	base := permutationProgram()
	permuted := permutedProgram()
	before := mustJSON(t, base)

	id1, err := programIDForPLIR("linux-x64", allocplan.Options{
		EnableStackLowering: true,
	}, base)
	if err != nil {
		t.Fatalf("programIDForPLIR base: %v", err)
	}
	id2, err := programIDForPLIR("linux-x64", allocplan.Options{
		EnableStackLowering: true,
	}, permuted)
	if err != nil {
		t.Fatalf("programIDForPLIR permuted: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("ProgramID depends on PLIR insertion order: %s != %s", id1, id2)
	}
	if after := mustJSON(t, base); after != before {
		t.Fatalf("programIDForPLIR mutated original PLIR\nbefore: %s\nafter:  %s", before, after)
	}
	id3, err := programIDForPLIR("linux-x64", allocplan.Options{
		EnableStackLowering:  true,
		EnableRegionPlanning: true,
	}, base)
	if err != nil {
		t.Fatalf("programIDForPLIR option-sensitive: %v", err)
	}
	if id3 == id1 {
		t.Fatalf("ProgramID ignored allocation options: %s", id1)
	}
}

func TestModulePlanDigestDeterministicOptionSensitiveAndPhaseGuarded(t *testing.T) {
	state, err := Build(checkedProgram(t, memoryPipelineFixture), Options{
		Target: "linux-x64",
		AllocPlan: allocplan.Options{
			EnableStackLowering: true,
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	module := state.PLIR.Funcs[0].Module
	digest1, err := state.ModulePlanDigest(module)
	if err != nil {
		t.Fatalf("ModulePlanDigest: %v", err)
	}
	digest2, err := state.ModulePlanDigest(module)
	if err != nil {
		t.Fatalf("ModulePlanDigest second call: %v", err)
	}
	if digest1 != digest2 || !memoryPlanIDPattern.MatchString(digest1) {
		t.Fatalf("ModulePlanDigest = %q/%q, want stable memory-plan sha256", digest1, digest2)
	}

	changed, err := Build(checkedProgram(t, memoryPipelineFixture), Options{
		Target: "linux-x64",
		AllocPlan: allocplan.Options{
			EnableStackLowering:  true,
			EnableRegionPlanning: true,
		},
	})
	if err != nil {
		t.Fatalf("Build changed options: %v", err)
	}
	changedDigest, err := changed.ModulePlanDigest(module)
	if err != nil {
		t.Fatalf("ModulePlanDigest changed options: %v", err)
	}
	if changedDigest == digest1 {
		t.Fatalf("ModulePlanDigest ignored allocation options: %s", digest1)
	}

	unplanned := *state
	unplanned.Phase = PhasePLIR
	if _, err := unplanned.ModulePlanDigest(module); err == nil ||
		!strings.Contains(err.Error(), "phase") {
		t.Fatalf("ModulePlanDigest before PhasePlanned error = %v, want phase guard", err)
	}
}

func checkedProgram(t *testing.T, src string) *semantics.CheckedProgram {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return checked
}

func assertOptionsDoNotExposeReportFlags(t *testing.T) {
	t.Helper()
	typ := reflect.TypeOf(Options{})
	for i := 0; i < typ.NumField(); i++ {
		name := strings.ToLower(typ.Field(i).Name)
		if strings.Contains(name, "report") {
			t.Fatalf("memorypipeline.Options exposes report flag field %q", typ.Field(i).Name)
		}
	}
}

func permutationProgram() *plir.Program {
	return &plir.Program{Funcs: []plir.Function{{
		Name:   "b",
		Module: "app",
		Values: []plir.Value{{
			ID:   "value:b",
			Kind: plir.ValueLocal,
			Type: "Int",
			Provenance: plir.Provenance{
				Kind: plir.ProvenanceStack,
				Root: "local:b",
			},
		}},
		Ops: []plir.Operation{{
			ID:   "op:b",
			Kind: plir.OpAssign,
		}},
		Facts: []plir.Fact{{
			ID:      "fact:b",
			Kind:    plir.FactProvenanceKnown,
			ValueID: "value:b",
			Source:  "source:b",
		}},
		Blocks: []plir.BasicBlock{{
			ID:    "block:b",
			Kind:  "plain",
			Entry: true,
			Exit:  true,
			Ops:   []string{"op:b"},
		}},
		ProofGuards: []plir.ProofGuard{{
			ID:        "proof:b",
			Kind:      "bounds",
			Block:     "block:b",
			OpID:      "op:b",
			Condition: "i < n",
		}},
		ProofUses: []plir.ProofUse{{
			ProofID: "proof:b",
			Block:   "block:b",
			OpID:    "op:b",
			UseKind: "bounds_check",
			Source:  "source:b",
		}},
		ProofTerms: []plir.ProofTerm{{
			ID:            "term:b",
			Kind:          "bounds",
			SubjectBaseID: "xs",
			IndexValueID:  "i",
			Operation:     "index_load",
			Range:         "0 <= i < n",
			FactsUsed:     []string{"fact:b"},
		}},
		RangeFacts: []plir.RangeFact{{
			Value:          "i",
			Lower:          plir.Bound{Kind: plir.BoundConst, Const: 0},
			Upper:          plir.Bound{Kind: plir.BoundSymbol, Symbol: "n"},
			InclusiveLower: true,
			Source:         "source:b",
			ProofID:        "proof:b",
			Reason:         "test",
			Derivation:     []string{"less_than_n", "non_negative"},
		}},
	}, {
		Name:   "a",
		Module: "app",
		Values: []plir.Value{{
			ID:   "value:a",
			Kind: plir.ValueLocal,
			Type: "Int",
			Provenance: plir.Provenance{
				Kind: plir.ProvenanceStack,
				Root: "local:a",
			},
		}},
		Ops: []plir.Operation{{
			ID:   "op:a",
			Kind: plir.OpAssign,
		}},
		Facts: []plir.Fact{{
			ID:      "fact:a",
			Kind:    plir.FactProvenanceKnown,
			ValueID: "value:a",
			Source:  "source:a",
		}},
		Blocks: []plir.BasicBlock{{
			ID:    "block:a",
			Kind:  "plain",
			Entry: true,
			Exit:  true,
			Ops:   []string{"op:a"},
		}},
	}}}
}

func permutedProgram() *plir.Program {
	prog := permutationProgram()
	prog.Funcs[0], prog.Funcs[1] = prog.Funcs[1], prog.Funcs[0]
	prog.Funcs[1].RangeFacts[0].Derivation[0], prog.Funcs[1].RangeFacts[0].Derivation[1] =
		prog.Funcs[1].RangeFacts[0].Derivation[1], prog.Funcs[1].RangeFacts[0].Derivation[0]
	return prog
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
