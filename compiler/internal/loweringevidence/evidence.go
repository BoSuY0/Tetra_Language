package loweringevidence

import "tetra_language/compiler/internal/allocplan"

type Evidence struct {
	Allocations []Allocation `json:"allocations,omitempty"`
}

type Allocation struct {
	Function         string                 `json:"function"`
	AllocationID     string                 `json:"allocation_id"`
	ValueID          string                 `json:"value_id,omitempty"`
	PlannedStorage   allocplan.StorageClass `json:"planned_storage"`
	ActualStorage    allocplan.StorageClass `json:"actual_storage"`
	ArtifactID       string                 `json:"artifact_id"`
	DecisionCode     string                 `json:"decision_code"`
	Reason           string                 `json:"reason,omitempty"`
	SourceFactIDs    []string               `json:"source_fact_ids,omitempty"`
	ProofIDs         []string               `json:"proof_ids,omitempty"`
	PlanDigest       string                 `json:"plan_digest,omitempty"`
	FirstInstruction int                    `json:"first_instruction"`
	LastInstruction  int                    `json:"last_instruction"`
}
