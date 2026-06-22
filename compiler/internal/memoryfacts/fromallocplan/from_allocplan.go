package fromallocplan

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/plir"
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

func AddFacts(graph *Graph, prog *plir.Program, plan *allocplan.Plan) error {
	if graph == nil {
		return fmt.Errorf("memoryfacts/fromallocplan: nil graph")
	}
	if plan == nil {
		return nil
	}
	delta, err := Delta(prog, plan)
	if err != nil {
		return err
	}
	if err := graph.Apply(delta); err != nil {
		return err
	}
	return graph.Validate()
}

func Delta(prog *plir.Program, plan *allocplan.Plan) (MemoryDelta, error) {
	if plan == nil {
		return MemoryDelta{Stage: StageAllocPlan}, nil
	}
	graph := NewGraph("memoryfacts/fromallocplan:delta")
	if err := addAllocPlanFacts(graph, plan, allocMemoryRefsFromProgram(prog)); err != nil {
		return MemoryDelta{}, err
	}
	facts := graph.Facts()
	for i := range facts {
		facts[i].ProgramID = ""
	}
	return MemoryDelta{
		Stage: StageAllocPlan,
		Add:   facts,
	}, nil
}

func allocMemoryRefsFromProgram(prog *plir.Program) map[string]allocMemoryRef {
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
	return RuntimeProofRequiredStorage(string(planned), string(actual))
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

func costClassForAllocFact(claim string, alloc allocplan.Allocation) CostClass {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(claim)), "rejected_") {
		return CostUnsupportedRejected
	}
	if allocPlanRuntimeProofRequiredStorage(alloc.PlannedStorage, alloc.ActualLoweringStorage) {
		return CostConservativeFallback
	}
	if alloc.ActualLoweringStorage == allocplan.StorageHeap &&
		alloc.PlannedStorage != "" &&
		alloc.PlannedStorage != allocplan.StorageHeap {
		return CostConservativeFallback
	}
	if !allocPlanValidationPasses(alloc) {
		return CostInstrumentationOnly
	}
	if alloc.Builtin == "core.alloc_bytes" && claim == "allocation_base_metadata" {
		return CostZeroCostProven
	}
	switch claim {
	case "storage_lowering":
		return CostZeroCostProven
	default:
		return CostInstrumentationOnly
	}
}

func islandIDForPLIRFact(fact plir.Fact, value plir.Value) string {
	if fact.IslandID != "" {
		return fact.IslandID
	}
	if value.Provenance.Kind != plir.ProvenanceIsland {
		return ""
	}
	root := value.Provenance.Root
	if root == "" {
		root = "unknown"
	}
	return "island:" + root
}

func epochForPLIRFact(fact plir.Fact, value plir.Value) int {
	if fact.Epoch != 0 {
		return fact.Epoch
	}
	if islandIDForPLIRFact(fact, value) != "" {
		return 1
	}
	return 0
}

func baseIDForPLIRFact(fact plir.Fact, value plir.Value) string {
	if fact.BaseID != "" {
		return fact.BaseID
	}
	if islandIDForPLIRFact(fact, value) != "" {
		return value.ID
	}
	return ""
}

func derivedFactID(parentID FactID, suffix string) FactID {
	return FactID(fmt.Sprintf("%s:%s", parentID, suffix))
}

func nonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
