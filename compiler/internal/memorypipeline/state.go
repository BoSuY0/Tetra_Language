package memorypipeline

import (
	"fmt"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/memoryfacts/fromallocplan"
	"tetra_language/compiler/internal/memoryfacts/fromplir"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/semantics"
)

type Phase string

const (
	PhasePLIR      Phase = "plir"
	PhasePlanned   Phase = "planned"
	PhaseLowered   Phase = "lowered"
	PhaseOptimized Phase = "optimized"
	PhaseValidated Phase = "validated"
	PhaseFinal     Phase = "final"
)

type Options struct {
	Target    string
	AllocPlan allocplan.Options
}

type State struct {
	ProgramID string
	Target    string
	PLIR      *plir.Program
	Graph     *memoryfacts.Graph
	Plan      *allocplan.Plan
	Phase     Phase

	allocOptions allocplan.Options
}

func Build(checked *semantics.CheckedProgram, opt Options) (*State, error) {
	if checked == nil {
		return nil, fmt.Errorf("memorypipeline: missing checked program")
	}
	prog, err := plir.FromCheckedProgram(checked)
	if err != nil {
		return nil, err
	}
	if err := plir.VerifyProgram(prog); err != nil {
		return nil, err
	}
	programID, err := programIDForPLIR(opt.Target, opt.AllocPlan, prog)
	if err != nil {
		return nil, err
	}
	graph, err := fromplir.Build(programID, prog)
	if err != nil {
		return nil, err
	}
	if err := graph.AdvanceTo(memoryfacts.StagePLIR); err != nil {
		return nil, err
	}
	snapshot, err := graph.Snapshot()
	if err != nil {
		return nil, err
	}
	plan, err := allocplan.Build(allocplan.Input{
		Program:  prog,
		Snapshot: snapshot,
		Options:  opt.AllocPlan,
	})
	if err != nil {
		return nil, err
	}
	delta, err := fromallocplan.Delta(prog, plan)
	if err != nil {
		return nil, err
	}
	if err := graph.Apply(delta); err != nil {
		return nil, err
	}
	return &State{
		ProgramID:    programID,
		Target:       opt.Target,
		PLIR:         prog,
		Graph:        graph,
		Plan:         plan,
		Phase:        PhasePlanned,
		allocOptions: opt.AllocPlan,
	}, nil
}

func (s *State) Snapshot() (memoryfacts.Snapshot, error) {
	if s == nil {
		return memoryfacts.Snapshot{}, fmt.Errorf("memorypipeline: nil state")
	}
	if err := s.requirePhaseAtLeast(PhasePLIR); err != nil {
		return memoryfacts.Snapshot{}, err
	}
	if s.Graph == nil {
		return memoryfacts.Snapshot{}, fmt.Errorf("memorypipeline: missing graph")
	}
	return s.Graph.Snapshot()
}

func (s *State) requirePhaseAtLeast(phase Phase) error {
	if s == nil {
		return fmt.Errorf("memorypipeline: nil state")
	}
	if !phaseAtLeast(s.Phase, phase) {
		return fmt.Errorf("memorypipeline: phase %q is before required phase %q", s.Phase, phase)
	}
	return nil
}

func phaseAtLeast(current Phase, required Phase) bool {
	return phaseRank(current) >= phaseRank(required)
}

func phaseRank(phase Phase) int {
	for i, candidate := range []Phase{
		PhasePLIR,
		PhasePlanned,
		PhaseLowered,
		PhaseOptimized,
		PhaseValidated,
		PhaseFinal,
	} {
		if phase == candidate {
			return i
		}
	}
	return -1
}
