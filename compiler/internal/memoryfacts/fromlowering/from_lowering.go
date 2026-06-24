package fromlowering

import (
	"fmt"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/loweringevidence"
	"tetra_language/compiler/internal/memoryfacts"
)

func AddFacts(graph *memoryfacts.Graph, evidence loweringevidence.Evidence) error {
	if graph == nil {
		return fmt.Errorf("memoryfacts/fromlowering: nil graph")
	}
	delta, err := delta(graph, evidence)
	if err != nil {
		return err
	}
	if err := graph.Apply(delta); err != nil {
		return err
	}
	return graph.Validate()
}

func Delta(evidence loweringevidence.Evidence) (memoryfacts.Delta, error) {
	return delta(nil, evidence)
}

func delta(
	graph *memoryfacts.Graph,
	evidence loweringevidence.Evidence,
) (memoryfacts.Delta, error) {
	facts := make([]memoryfacts.Fact, 0, len(evidence.Allocations))
	for _, row := range evidence.Allocations {
		parentID := firstSourceFactID(row.SourceFactIDs)
		if parentID == "" {
			return memoryfacts.Delta{}, fmt.Errorf(
				"memoryfacts/fromlowering: allocation %s/%s missing source fact parent",
				row.Function,
				row.AllocationID,
			)
		}
		fact := memoryfacts.Fact{
			ID:                    memoryfacts.FactID("lowering:" + row.Function + ":" + row.AllocationID),
			FunctionID:            row.Function,
			ValueID:               row.ValueID,
			SiteID:                row.ArtifactID,
			AllocationSiteID:      row.AllocationID,
			ProvenanceClass:       memoryfacts.ProvenanceUnsafeUnknown,
			UnsafeClass:           memoryfacts.UnsafeUnknown,
			StoragePlan:           storageClass(row.PlannedStorage),
			ActualLoweringStorage: storageClass(row.ActualStorage),
			LoweredArtifactID:     row.ArtifactID,
			Claim:                 memoryfacts.ClaimStorageLowering,
			DecisionCode:          row.DecisionCode,
			Reason:                row.Reason,
			SourceStage:           memoryfacts.StageLowering,
			ParentFactID:          memoryfacts.FactID(parentID),
			ValidationState:       memoryfacts.ValidationNotRun,
			CostClass:             loweringCostClass(row),
		}
		inheritIslandIdentity(graph, parentID, &fact)
		facts = append(facts, fact)
	}
	return memoryfacts.Delta{
		Stage: memoryfacts.StageLowering,
		Add:   facts,
	}, nil
}

func inheritIslandIdentity(graph *memoryfacts.Graph, parentID string, fact *memoryfacts.Fact) {
	if graph == nil || fact == nil {
		return
	}
	parent, ok := graph.Fact(memoryfacts.FactID(parentID))
	if !ok {
		return
	}
	fact.IslandID = parent.IslandID
	fact.Epoch = parent.Epoch
	fact.BaseID = parent.BaseID
}

func firstSourceFactID(ids []string) string {
	for _, id := range ids {
		if id != "" {
			return id
		}
	}
	return ""
}

func storageClass(storage allocplan.StorageClass) memoryfacts.StorageClass {
	if storage == "" {
		return memoryfacts.StorageUnknownConservative
	}
	return memoryfacts.StorageClass(storage)
}

func loweringCostClass(row loweringevidence.Allocation) memoryfacts.CostClass {
	if row.ActualStorage == allocplan.StorageUnknownConservative ||
		row.PlannedStorage != row.ActualStorage {
		return memoryfacts.CostConservativeFallback
	}
	return memoryfacts.CostInstrumentationOnly
}
