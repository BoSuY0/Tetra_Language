package memoryfacts_test

import (
	"tetra_language/compiler/internal/allocplan"
	memoryfacts "tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/memoryfacts/fromallocplan"
	"tetra_language/compiler/internal/memoryfacts/fromplir"
	"tetra_language/compiler/internal/memoryfacts/fromvalidation"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/validation"
)

func BuildGraphFromPLIRAndPlan(
	programID string,
	prog *plir.Program,
	plan *allocplan.Plan,
) (*memoryfacts.Graph, error) {
	graph, err := fromplir.Build(programID, prog)
	if err != nil {
		return nil, err
	}
	if err := fromallocplan.AddFacts(graph, prog, plan); err != nil {
		return nil, err
	}
	if err := graph.Validate(); err != nil {
		return nil, err
	}
	return graph, nil
}

func AddBoundsProofFacts(graph *memoryfacts.Graph, report validation.ProofReport) error {
	return fromvalidation.AddBoundsProofFacts(graph, report)
}

func AddBoundsProofRejectionFact(
	graph *memoryfacts.Graph,
	functionID string,
	siteID string,
	reason string,
) error {
	return fromvalidation.AddBoundsProofRejectionFact(graph, functionID, siteID, reason)
}
