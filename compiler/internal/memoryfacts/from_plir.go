package memoryfacts

import (
	"fmt"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/plir"
)

func FromPLIRAndAllocPlan(programID string, prog *plir.Program, plan *allocplan.Plan) (*Graph, error) {
	graph := NewGraph(programID)
	if err := addRepresentationMetadataFact(graph); err != nil {
		return nil, err
	}
	if prog != nil {
		if err := validatePLIRFunctionSummaryCompleteness(prog); err != nil {
			return nil, err
		}
		if err := addPLIRFacts(graph, prog); err != nil {
			return nil, err
		}
	}
	if plan != nil {
		if err := addAllocPlanFacts(graph, plan, allocMemoryRefsFromPLIR(prog)); err != nil {
			return nil, err
		}
	}
	if err := graph.Validate(); err != nil {
		return nil, err
	}
	return graph, nil
}

func validatePLIRFunctionSummaryCompleteness(prog *plir.Program) error {
	for _, fn := range prog.Funcs {
		if err := plir.VerifyFunctionSummaryCompleteness(fn); err != nil {
			return err
		}
	}
	return nil
}

func addRepresentationMetadataFact(graph *Graph) error {
	id, err := graph.AddFact(Fact{
		ID:              "semantics:representation-metadata:not-user-assignable",
		SiteID:          "semantics:representation-metadata",
		SourceStage:     StageSemantics,
		ProvenanceClass: ProvenanceSafeKnown,
		UnsafeClass:     UnsafeSafe,
		Claim:           "safe_representation_metadata: not_user_assignable",
		Reason:          "slice/String representation metadata is not user-assignable state",
	})
	if err != nil {
		return err
	}
	return graph.MarkValidated(id, "representation_namespace_validator")
}

func addPLIRFacts(graph *Graph, prog *plir.Program) error {
	for _, fn := range prog.Funcs {
		values := map[string]plir.Value{}
		for _, value := range fn.Values {
			values[value.ID] = value
		}
		opsByOutput := map[string]plir.Operation{}
		for _, op := range fn.Ops {
			for _, output := range op.Outputs {
				if _, exists := opsByOutput[output]; !exists {
					opsByOutput[output] = op
				}
			}
		}
		factIDs := map[plirFactKey]FactID{}
		for _, pf := range fn.Facts {
			value, ok := values[pf.ValueID]
			if !ok && (pf.ValueID != "" || plirFactRequiresValue(pf.Kind)) {
				return fmt.Errorf("memoryfacts: plir function %s fact %q references missing value_id %q", fn.Name, pf.ID, pf.ValueID)
			}
			if suppressRawVerifiedRootGenericFact(pf, value) {
				continue
			}
			fact := Fact{
				ID:               FactID(fmt.Sprintf("plir:%s:%s", fn.Name, pf.ID)),
				FunctionID:       fn.Name,
				ValueID:          pf.ValueID,
				IslandID:         islandIDForPLIRFact(pf, value),
				Epoch:            epochForPLIRFact(pf, value),
				BaseID:           baseIDForPLIRFact(pf, value),
				SiteID:           nonEmpty(pf.Source, fmt.Sprintf("%s:%s", fn.Name, pf.ID)),
				SourceSpan:       pf.Source,
				TypeName:         value.Type,
				SourceStage:      StagePLIR,
				ProvenanceClass:  provenanceClassForPLIRFact(pf, value),
				UnsafeClass:      unsafeClassForPLIRFact(pf, value),
				RegionID:         nonEmpty(pf.Region, value.Region),
				OwnerID:          ownerForPLIRValue(value),
				BorrowState:      borrowStateForPLIRFact(pf),
				EscapeState:      escapeStateForPLIRFact(pf, value),
				AliasState:       aliasStateForPLIRFact(pf),
				AllocationSiteID: allocationSiteIDForPLIRValue(value),
				Claim:            claimForPLIRFact(pf),
				Reason:           pf.Reason,
			}
			id, err := graph.AddFact(fact)
			if err != nil {
				return err
			}
			factIDs[plirFactKey{kind: pf.Kind, valueID: pf.ValueID}] = id
		}
		for _, pf := range fn.Facts {
			value := values[pf.ValueID]
			parentID := factIDs[plirFactKey{kind: pf.Kind, valueID: pf.ValueID}]
			parent, ok := graph.Fact(parentID)
			if !ok {
				continue
			}
			switch pf.Kind {
			case plir.FactBorrowedImm, plir.FactBorrowedMut:
				if err := addBorrowMetadataFacts(graph, parent, value, opsByOutput[pf.ValueID]); err != nil {
					return err
				}
				if err := addBorrowAggregateV0Facts(graph, parent, value, opsByOutput[pf.ValueID]); err != nil {
					return err
				}
			case plir.FactOwned:
				if isCopyAllocationValue(value) {
					if err := addCopyMetadataFacts(graph, parent, value, opsByOutput[pf.ValueID], factIDs, values); err != nil {
						return err
					}
				}
			case plir.FactNoAlias:
				if err := addNoAliasMetadataFacts(graph, parent, value); err != nil {
					return err
				}
			case plir.FactDerivedWindow:
				if err := addSliceViewBoundsCheckFact(graph, parent, value, opsByOutput[pf.ValueID]); err != nil {
					return err
				}
			}
		}
		if err := addFunctionSummaryFacts(graph, fn, values, factIDs); err != nil {
			return err
		}
		for _, op := range fn.Ops {
			if isCopyIntoOperation(op) {
				if err := addCopyIntoFacts(graph, fn.Name, op, values); err != nil {
					return err
				}
			}
			if op.Kind != plir.OpUnsafe {
				continue
			}
			if isSafeWrapperPromotionOperation(op) {
				if err := addSafeWrapperPromotionRejectionFact(graph, fn.Name, op, values, factIDs); err != nil {
					return err
				}
				continue
			}
			claim, provenance, unsafeClass, ok := unsafeOperationClaim(op)
			if !ok {
				continue
			}
			valueID := ""
			if len(op.Outputs) > 0 {
				valueID = op.Outputs[0]
			}
			costClass := costClassForUnsafeOperationClaim(claim, provenance, unsafeClass)
			fact := Fact{
				ID:               FactID(fmt.Sprintf("plir:%s:%s:unsafe", fn.Name, op.ID)),
				FunctionID:       fn.Name,
				ValueID:          valueID,
				SiteID:           nonEmpty(op.Source, fmt.Sprintf("%s:%s", fn.Name, op.ID)),
				SourceSpan:       op.Source,
				SourceStage:      StagePLIR,
				ProvenanceClass:  provenance,
				UnsafeClass:      unsafeClass,
				Claim:            claim,
				CostClass:        costClass,
				NormalBuildCheck: costClass == CostDynamicCheckRequired,
				Reason:           op.Note,
			}
			if claim == "unsafe_contract_static_untrusted" {
				fact.AliasState = AliasInvalidatedByCall
				fact.ValidatorName = "unsafe_static_contract_validator"
			}
			if claim == "unsafe_contract_runtime_checkable" {
				fact.ValidatorName = "unsafe_runtime_contract_validator"
			}
			id, err := graph.AddFact(fact)
			if err != nil {
				return err
			}
			if err := finalizeUnsafeOperationFact(graph, id, claim, provenance, unsafeClass); err != nil {
				return err
			}
		}
	}
	return nil
}
