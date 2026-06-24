package memorypipeline

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/loweringevidence"
	"tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/memoryfacts/fromlowering"
)

func (s *State) ApplyLowering(
	program *ir.IRProgram,
	evidence loweringevidence.Evidence,
) error {
	if s == nil {
		return fmt.Errorf("memorypipeline: nil state")
	}
	if err := s.requirePhaseAtLeast(PhasePlanned); err != nil {
		return err
	}
	if s.Plan == nil {
		return fmt.Errorf("memorypipeline: missing allocation plan")
	}
	if program == nil {
		return fmt.Errorf("memorypipeline: missing lowering result")
	}
	if err := fromlowering.AddFacts(s.Graph, evidence); err != nil {
		return err
	}
	rows := map[string]loweringevidence.Allocation{}
	for _, row := range evidence.Allocations {
		rows[loweringKey(row.Function, row.AllocationID)] = row
	}
	for fnIndex := range s.Plan.Functions {
		fn := &s.Plan.Functions[fnIndex]
		for allocIndex := range fn.Allocations {
			alloc := &fn.Allocations[allocIndex]
			row, ok := rows[loweringKey(fn.Name, alloc.ID)]
			if !ok {
				return fmt.Errorf(
					"memorypipeline: missing lowering evidence for %s/%s",
					fn.Name,
					alloc.ID,
				)
			}
			alloc.ActualLoweringStorage = row.ActualStorage
			alloc.LoweredArtifactID = row.ArtifactID
			alloc.LoweringStatus = loweringStatus(*alloc, row)
			alloc.ValidationStatus = loweringValidationStatus(*alloc, row)
			alloc.RuntimePath = ""
			alloc.DecisionCode = row.DecisionCode
			allocplan.ApplyLoweredAllocationReportHooksWithOptions(alloc, s.allocOptions)
		}
	}
	if err := allocplan.VerifyLowered(s.Plan); err != nil {
		return err
	}
	if err := s.attachLoweringToPlanFacts(rows); err != nil {
		return err
	}
	s.Phase = PhaseLowered
	return nil
}

func (s *State) ApplyOptimization(delta memoryfacts.Delta) error {
	if s == nil {
		return fmt.Errorf("memorypipeline: nil state")
	}
	if s.Phase == PhaseOptimized {
		return nil
	}
	if err := s.requirePhaseAtLeast(PhaseLowered); err != nil {
		return err
	}
	if s.Graph == nil {
		return fmt.Errorf("memorypipeline: missing graph")
	}
	if delta.Stage == "" {
		delta.Stage = memoryfacts.StageOptimization
	}
	if err := s.Graph.Apply(delta); err != nil {
		return err
	}
	s.Phase = PhaseOptimized
	return nil
}

func (s *State) SkipOptimization() error {
	return s.ApplyOptimization(memoryfacts.Delta{Stage: memoryfacts.StageOptimization})
}

func loweringKey(function, allocationID string) string {
	return function + "\x00" + allocationID
}

func (s *State) attachLoweringToPlanFacts(
	rows map[string]loweringevidence.Allocation,
) error {
	for _, fn := range s.Plan.Functions {
		for _, alloc := range fn.Allocations {
			row, ok := rows[loweringKey(fn.Name, alloc.ID)]
			if !ok {
				continue
			}
			factIDs := append(
				[]memoryfacts.FactID{memoryfacts.FactID("allocplan:" + fn.Name + ":" + alloc.ID)},
				allocationSourceFactIDs(alloc.SourceFactIDs)...,
			)
			seen := map[memoryfacts.FactID]struct{}{}
			for _, factID := range factIDs {
				if _, ok := seen[factID]; ok {
					continue
				}
				seen[factID] = struct{}{}
				if err := s.attachLoweringToFact(factID, alloc, row); err != nil {
					return err
				}
			}
		}
	}
	return s.Graph.Validate()
}

func (s *State) attachLoweringToFact(
	factID memoryfacts.FactID,
	alloc allocplan.Allocation,
	row loweringevidence.Allocation,
) error {
	fact, ok := s.Graph.Fact(factID)
	if !ok {
		return nil
	}
	if !allocationFactCarriesLowering(fact) {
		return nil
	}
	if err := s.Graph.AttachLoweringStorage(
		factID,
		memoryfacts.StorageClass(alloc.PlannedStorage),
		memoryfacts.StorageClass(row.ActualStorage),
		row.ArtifactID,
	); err != nil {
		return err
	}
	if allocationFactValidationPasses(fact, alloc) {
		if err := s.Graph.MarkValidated(factID, "allocation_lowering_validator"); err != nil {
			return err
		}
	}
	return nil
}

func allocationSourceFactIDs(ids []string) []memoryfacts.FactID {
	out := make([]memoryfacts.FactID, 0, len(ids))
	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			continue
		}
		out = append(out, memoryfacts.FactID(id))
	}
	return out
}

func allocationFactCarriesLowering(fact memoryfacts.Fact) bool {
	switch fact.Claim {
	case memoryfacts.ClaimAllocationBaseMetadata,
		memoryfacts.ClaimUnsafeVerifiedRootAllocationBase,
		memoryfacts.ClaimTrustedStorage,
		memoryfacts.ClaimStorageLowering:
		return true
	default:
		return false
	}
}

func allocationFactValidationPasses(fact memoryfacts.Fact, alloc allocplan.Allocation) bool {
	if !strings.HasPrefix(alloc.ValidationStatus, "validated") {
		return false
	}
	if memoryfacts.RuntimeProofRequiredStorage(
		string(alloc.PlannedStorage),
		string(alloc.ActualLoweringStorage),
	) {
		return false
	}
	if memoryfacts.ValidatedTrustedStorageHeapFallback(
		string(alloc.PlannedStorage),
		string(alloc.ActualLoweringStorage),
	) {
		return false
	}
	if memoryfacts.UnsafeExternalRootTrustedStorage(
		string(fact.ProvenanceClass),
		string(fact.UnsafeClass),
		string(alloc.PlannedStorage),
		string(alloc.ActualLoweringStorage),
	) {
		return false
	}
	return true
}

func loweringStatus(
	alloc allocplan.Allocation,
	row loweringevidence.Allocation,
) string {
	planned := alloc.PlannedStorage
	if planned == "" {
		planned = alloc.Storage
	}
	if row.ActualStorage == planned {
		switch row.ActualStorage {
		case allocplan.StorageEliminated:
			return "eliminated_lowering"
		case allocplan.StorageRegister:
			return "register_lowering"
		case allocplan.StorageStack:
			return "stack_lowering"
		case allocplan.StorageRegion:
			return "region_lowering"
		case allocplan.StorageFunctionTempRegion:
			return "function_temp_region_lowering"
		case allocplan.StorageExplicitIsland:
			return "explicit_island_lowering"
		case allocplan.StorageTaskRegion:
			return "task_region_lowering"
		case allocplan.StorageActorMoveRegion:
			return "actor_move_region_lowering"
		case allocplan.StorageHeap:
			return "heap_lowering"
		case allocplan.StorageLargeMmap:
			return "large_mmap_lowering"
		case allocplan.StorageExternal:
			return "external_lowering"
		default:
			return "lowered"
		}
	}
	if row.ActualStorage == allocplan.StorageUnknownConservative {
		return "lowered_missing_artifact"
	}
	return "lowered_fallback"
}

func loweringValidationStatus(
	alloc allocplan.Allocation,
	row loweringevidence.Allocation,
) string {
	switch row.ActualStorage {
	case allocplan.StorageEliminated, allocplan.StorageRegister, allocplan.StorageStack:
		return "validated_no_escape"
	case allocplan.StorageRegion:
		return "validated_region_scope"
	case allocplan.StorageFunctionTempRegion:
		return "validated_function_temp_region_scope"
	case allocplan.StorageExplicitIsland:
		if alloc.LengthStatus == allocplan.LengthStatusValidEmpty {
			return "validated_empty_no_backing"
		}
		return "validated_explicit_island_scope"
	case allocplan.StorageTaskRegion:
		return "validated_task_region_scope"
	case allocplan.StorageActorMoveRegion:
		return "validated_actor_move_region_scope"
	case allocplan.StorageHeap, allocplan.StorageLargeMmap, allocplan.StorageExternal:
		return "validated_heap_fallback"
	default:
		return "lowering_unvalidated"
	}
}
