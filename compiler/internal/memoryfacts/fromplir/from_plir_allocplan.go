package fromplir

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/allocplan"
	. "tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/memoryvocab"
)

type plirFactKey struct {
	kind    plir.FactKind
	valueID string
}

func plirFactRequiresValue(kind plir.FactKind) bool {
	switch kind {
	case plir.FactLenStable, plir.FactIndexInRange, plir.FactRegionAlive,
		plir.FactNoEscape, plir.FactNoAlias, plir.FactNonNull,
		plir.FactMaybeNull, plir.FactAligned, plir.FactProvenanceKnown,
		plir.FactProvenanceUnknown, plir.FactOwned, plir.FactBorrowedImm,
		plir.FactBorrowedMut, plir.FactMoved, plir.FactDerivedWindow:
		return true
	default:
		return false
	}
}

func suppressRawVerifiedRootGenericFact(fact plir.Fact, value plir.Value) bool {
	if value.Alloc == nil || value.Alloc.Builtin != "core.alloc_bytes" {
		return false
	}
	switch fact.Kind {
	case plir.FactProvenanceKnown, plir.FactRegionAlive:
		return true
	default:
		return false
	}
}

type allocMemoryRef struct {
	IslandID string
	Epoch    int
	BaseID   string
}

func allocMemoryRefsFromPLIR(prog *plir.Program) map[string]allocMemoryRef {
	refs := map[string]allocMemoryRef{}
	if prog == nil {
		return refs
	}
	for _, fn := range prog.Funcs {
		values := map[string]plir.Value{}
		for _, value := range fn.Values {
			values[value.ID] = value
			if value.Provenance.Kind != plir.ProvenanceIsland {
				continue
			}
			refs[allocMemoryRefKey(fn.Name, value.ID)] = allocMemoryRef{
				IslandID: islandIDForPLIRFact(plir.Fact{}, value),
				Epoch:    epochForPLIRFact(plir.Fact{}, value),
				BaseID:   baseIDForPLIRFact(plir.Fact{}, value),
			}
		}
		for _, fact := range fn.Facts {
			value := values[fact.ValueID]
			ref := allocMemoryRef{
				IslandID: islandIDForPLIRFact(fact, value),
				Epoch:    epochForPLIRFact(fact, value),
				BaseID:   baseIDForPLIRFact(fact, value),
			}
			if ref.IslandID == "" {
				continue
			}
			refs[allocMemoryRefKey(fn.Name, fact.ValueID)] = ref
		}
	}
	return refs
}

func addAllocPlanFacts(graph *Graph, plan *allocplan.Plan, refs map[string]allocMemoryRef) error {
	for _, fn := range plan.Functions {
		for _, alloc := range fn.Allocations {
			if alloc.RawPointerBoundsStatus == "" && alloc.PlannedStorage == "" &&
				alloc.ActualLoweringStorage == "" {
				continue
			}
			if allocPlanTrustedStorageHeapFallback(
				alloc.PlannedStorage,
				alloc.ActualLoweringStorage,
			) &&
				strings.TrimSpace(alloc.Reason) == "" {
				return fmt.Errorf(
					"allocplan %s:%s heap fallback missing storage reason",
					fn.Name,
					alloc.ID,
				)
			}
			claim := alloc.RawPointerBoundsStatus
			provenance := ProvenanceSafeOwned
			unsafeClass := UnsafeSafe
			if alloc.Builtin == "core.alloc_bytes" {
				provenance = ProvenanceUnsafeVerifiedRoot
				unsafeClass = UnsafeVerifiedRoot
				if claim == "" {
					claim = "allocation_base_metadata"
				}
			}
			if claim == "" {
				claim = "storage_lowering"
			}
			ref := allocPlanMemoryRef(fn.Name, alloc, refs)
			loweredArtifactID := fmt.Sprintf(
				"ir:%s:%s:%s",
				fn.Name,
				alloc.ID,
				alloc.ActualLoweringStorage,
			)
			validated := allocPlanValidationPasses(alloc)
			validationState := ValidationNotRun
			validatorName := ""
			if validated {
				validationState = ValidationPass
				validatorName = "allocation_lowering_validator"
			}
			id, err := graph.AddFact(Fact{
				ID:         FactID(fmt.Sprintf("allocplan:%s:%s", fn.Name, alloc.ID)),
				FunctionID: fn.Name,
				ValueID:    alloc.ValueID,
				IslandID:   ref.IslandID,
				Epoch:      ref.Epoch,
				BaseID:     ref.BaseID,
				SiteID: nonEmpty(
					alloc.SiteID,
					fmt.Sprintf("%s:%s", fn.Name, alloc.ID),
				),
				SourceSpan:            alloc.Source,
				SourceStage:           StageAllocPlan,
				TypeName:              alloc.ElementType,
				ProvenanceClass:       provenance,
				UnsafeClass:           unsafeClass,
				AllocationSiteID:      alloc.ID,
				StoragePlan:           StorageClass(alloc.PlannedStorage),
				ActualLoweringStorage: StorageClass(alloc.ActualLoweringStorage),
				ValidationState:       validationState,
				Claim:                 claim,
				LoweredArtifactID:     loweredArtifactID,
				ValidatorName:         validatorName,
				CostClass:             costClassForAllocFact(claim, alloc),
				Reason:                alloc.Reason,
			})
			if err != nil {
				return err
			}
			if validated && alloc.Builtin == "core.alloc_bytes" &&
				claim == "allocation_base_metadata" {
				if err := addUnsafeVerifiedRootAllocationBaseFact(graph, id, fn.Name, alloc); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func allocMemoryRefKey(functionName string, valueID string) string {
	return functionName + "\x00" + valueID
}

func allocPlanMemoryRef(
	functionName string,
	alloc allocplan.Allocation,
	refs map[string]allocMemoryRef,
) allocMemoryRef {
	if ref, ok := refs[allocMemoryRefKey(functionName, alloc.ValueID)]; ok && ref.IslandID != "" {
		return ref
	}
	if alloc.PlannedStorage != allocplan.StorageExplicitIsland &&
		alloc.ActualLoweringStorage != allocplan.StorageExplicitIsland {
		return allocMemoryRef{}
	}
	if strings.TrimSpace(alloc.RegionID) == "" {
		return allocMemoryRef{}
	}
	return allocMemoryRef{
		IslandID: alloc.RegionID,
		Epoch:    1,
		BaseID:   nonEmpty(alloc.ValueID, alloc.ID),
	}
}

func addUnsafeVerifiedRootAllocationBaseFact(
	graph *Graph,
	parentID FactID,
	functionID string,
	alloc allocplan.Allocation,
) error {
	id, err := graph.DeriveFact(parentID, Fact{
		ID:               derivedFactID(parentID, "unsafe_verified_root_allocation_base"),
		FunctionID:       functionID,
		ValueID:          alloc.ValueID,
		SiteID:           nonEmpty(alloc.SiteID, fmt.Sprintf("%s:%s", functionID, alloc.ID)),
		SourceSpan:       alloc.Source,
		SourceStage:      StageAllocPlan,
		TypeName:         alloc.ElementType,
		ProvenanceClass:  ProvenanceUnsafeVerifiedRoot,
		UnsafeClass:      UnsafeVerifiedRoot,
		AllocationSiteID: alloc.ID,
		Claim:            "unsafe_verified_root_allocation_base",
		ValidationState:  ValidationPass,
		ValidatorName:    "unsafe_verified_root_bounds_validator",
		CostClass:        CostZeroCostProven,
		Reason: ("Memory Ideal v5 accepts bounded core.alloc_bytes allocation-" +
			"base metadata without safe-fact promotion"),
	})
	if err != nil {
		return err
	}
	return graph.MarkValidated(id, "unsafe_verified_root_bounds_validator")
}

func allocPlanValidationPasses(alloc allocplan.Allocation) bool {
	if !strings.HasPrefix(alloc.ValidationStatus, "validated") {
		return false
	}
	if allocPlanRuntimeProofRequiredStorage(alloc.PlannedStorage, alloc.ActualLoweringStorage) {
		return false
	}
	if allocPlanTrustedStorageHeapFallback(alloc.PlannedStorage, alloc.ActualLoweringStorage) {
		return false
	}
	return true
}

func allocPlanRuntimeProofRequiredStorage(
	planned allocplan.StorageClass,
	actual allocplan.StorageClass,
) bool {
	return memoryvocab.RuntimeProofRequiredStorage(string(planned), string(actual))
}

func allocPlanTrustedStorageHeapFallback(
	planned allocplan.StorageClass,
	actual allocplan.StorageClass,
) bool {
	if actual != allocplan.StorageHeap {
		return false
	}
	switch planned {
	case allocplan.StorageEliminated, allocplan.StorageRegister, allocplan.StorageStack,
		allocplan.StorageRegion, allocplan.StorageFunctionTempRegion, allocplan.StorageExplicitIsland,
		allocplan.StorageTaskRegion, allocplan.StorageActorMoveRegion:
		return true
	default:
		return false
	}
}
