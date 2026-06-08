package memoryfacts

import (
	"fmt"
	"strings"

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

func addFunctionSummaryFacts(graph *Graph, fn plir.Function, values map[string]plir.Value, factIDs map[plirFactKey]FactID) error {
	if err := addReturnSummaryFacts(graph, fn, values, factIDs); err != nil {
		return err
	}
	if err := addDeclaredSummaryFacts(graph, fn); err != nil {
		return err
	}
	if err := addOperationSummaryFacts(graph, fn, values, factIDs); err != nil {
		return err
	}
	return addFactKindSummaryFacts(graph, fn, values, factIDs)
}

func addReturnSummaryFacts(graph *Graph, fn plir.Function, values map[string]plir.Value, factIDs map[plirFactKey]FactID) error {
	for _, op := range fn.Ops {
		if op.Kind != plir.OpReturn || len(op.Inputs) == 0 {
			continue
		}
		path := op.Inputs[0]
		returnValues := returnSummaryValues(path, values)
		if len(returnValues) == 0 {
			continue
		}
		for _, value := range returnValues {
			anchor := op.ID
			if len(returnValues) > 1 {
				anchor += ":" + value.ID
			}
			if plirValueIsUnsafeUnknown(value, factIDs) {
				if _, err := graph.AddFact(functionSummaryFact(fn.Name, anchor, "returns_unknown_unsafe", op.Source, value, ProvenanceUnsafeUnknown, UnsafeUnknown, EscapeConservative, AliasUnknown, "", "returned value has unknown or external unsafe provenance")); err != nil {
					return err
				}
				continue
			}
			if returnValueIsOwnedAllocation(value, factIDs) {
				if _, err := graph.AddFact(functionSummaryFact(fn.Name, anchor, "returns_owned_new_allocation", op.Source, value, ProvenanceSafeOwned, UnsafeSafe, EscapeReturn, AliasUnknown, ownerForPLIRValue(value), "return value owns fresh compiler-visible allocation provenance")); err != nil {
					return err
				}
			}
			if source, ok := returnedBorrowSource(fn, value, factIDs); ok {
				reason := fmt.Sprintf("return value borrows from parameter %q", source.Owner)
				if source.ParamPath != "" {
					reason = fmt.Sprintf("return value borrows from parameter path %q", source.Owner+"."+source.ParamPath)
				}
				if index, indexOK := source.ParamIndexValue(); indexOK {
					reason = fmt.Sprintf("return value borrows from parameter #%d %q", index, source.Owner)
					if source.ParamPath != "" {
						reason = fmt.Sprintf("return value borrows from parameter #%d %q path %q", index, source.Owner, source.ParamPath)
					}
				}
				fact := functionSummaryFact(fn.Name, anchor, "returns_borrow_from_param", op.Source, value, ProvenanceSafeBorrowed, UnsafeSafe, EscapeReturn, AliasUnknown, source.Owner, reason)
				fact.ParamIndex = source.ParamIndex
				fact.ParamPath = source.ParamPath
				if _, err := graph.AddFact(fact); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func addDeclaredSummaryFacts(graph *Graph, fn plir.Function) error {
	summary := fn.Summary
	if summary == nil {
		return nil
	}
	for leaf, paramIndex := range summary.ReturnRegionSummary {
		owner := summaryParamOwner(summary, paramIndex)
		reason := fmt.Sprintf("return%s region provenance is parameter #%d", formatSummaryLeaf(leaf), paramIndex)
		if owner != "" {
			reason = fmt.Sprintf("%s (%s)", reason, owner)
		}
		fact := functionSummaryFact(fn.Name, "return_region:"+leaf, "may_return_region", summarySite(fn), plir.Value{}, ProvenanceSafeBorrowed, UnsafeSafe, EscapeReturn, AliasUnknown, owner, reason)
		fact.ParamIndex = paramIndexPtr(paramIndex)
		fact.ParamPath = leaf
		if _, err := graph.AddFact(fact); err != nil {
			return err
		}
	}
	if summary.ReturnRegionUnknown {
		if _, err := graph.AddFact(functionSummaryFact(fn.Name, "return_region_unknown", "may_return_region", summarySite(fn), plir.Value{}, ProvenanceUnsafeUnknown, UnsafeUnknown, EscapeConservative, AliasUnknown, "", "return region summary is unknown and remains conservative")); err != nil {
			return err
		}
	}
	for leaf, provenances := range summary.ReturnResourceSummary {
		for _, provenance := range provenances {
			owner := summaryParamOwner(summary, provenance.ParamIndex)
			reason := fmt.Sprintf("returned resource%s provenance is parameter #%d%s", formatSummaryLeaf(leaf), provenance.ParamIndex, formatSummaryLeaf(provenance.ParamPath))
			if owner != "" {
				reason = fmt.Sprintf("%s (%s)", reason, owner)
			}
			fact := functionSummaryFact(fn.Name, fmt.Sprintf("return_resource:%s:%d:%s", leaf, provenance.ParamIndex, provenance.ParamPath), "may_return_resource", summarySite(fn), plir.Value{}, ProvenanceSafeKnown, UnsafeSafe, EscapeReturn, AliasUnknown, owner, reason)
			fact.ParamIndex = paramIndexPtr(provenance.ParamIndex)
			fact.ParamPath = provenance.ParamPath
			if _, err := graph.AddFact(fact); err != nil {
				return err
			}
		}
	}
	if summary.ReturnResourceUnknown {
		if _, err := graph.AddFact(functionSummaryFact(fn.Name, "return_resource_unknown", "returns_unknown_unsafe", summarySite(fn), plir.Value{}, ProvenanceUnsafeUnknown, UnsafeUnknown, EscapeConservative, AliasUnknown, "", "returned resource provenance is unknown and remains conservative")); err != nil {
			return err
		}
	}
	for leaf, provenances := range summary.ThrowResourceSummary {
		for _, provenance := range provenances {
			owner := summaryParamOwner(summary, provenance.ParamIndex)
			reason := fmt.Sprintf("thrown resource%s provenance is parameter #%d%s", formatSummaryLeaf(leaf), provenance.ParamIndex, formatSummaryLeaf(provenance.ParamPath))
			if owner != "" {
				reason = fmt.Sprintf("%s (%s)", reason, owner)
			}
			fact := functionSummaryFact(fn.Name, fmt.Sprintf("throw_resource:%s:%d:%s", leaf, provenance.ParamIndex, provenance.ParamPath), "may_throw_resource", summarySite(fn), plir.Value{}, ProvenanceSafeKnown, UnsafeSafe, EscapeReturn, AliasUnknown, owner, reason)
			fact.ParamIndex = paramIndexPtr(provenance.ParamIndex)
			fact.ParamPath = provenance.ParamPath
			if _, err := graph.AddFact(fact); err != nil {
				return err
			}
		}
	}
	if len(summary.Effects) > 0 {
		if _, err := graph.AddFact(functionSummaryFact(fn.Name, "effects", "requires_effects", summarySite(fn), plir.Value{}, ProvenanceSafeKnown, UnsafeSafe, EscapeUnknown, AliasUnknown, "", "function declares effects: "+strings.Join(summary.Effects, ", "))); err != nil {
			return err
		}
	}
	if summaryRequiresCapabilities(summary) {
		if _, err := graph.AddFact(functionSummaryFact(fn.Name, "capabilities", "requires_capabilities", summarySite(fn), plir.Value{}, ProvenanceSafeKnown, UnsafeSafe, EscapeUnknown, AliasUnknown, "", "function effects require capability-gated operations: "+strings.Join(summary.Effects, ", "))); err != nil {
			return err
		}
	}
	if summaryRequiresCapMemAuthorization(summary) {
		if _, err := graph.AddFact(functionSummaryFact(fn.Name, "cap_mem_authorization", "cap_mem_authorization_only", summarySite(fn), plir.Value{}, ProvenanceUnsafeChecked, UnsafeChecked, EscapeUnknown, AliasUnknown, "", "cap.mem authorizes raw operations only; it does not prove pointer validity, bounds, ownership, noalias, or safe provenance")); err != nil {
			return err
		}
	}
	if summary.TouchesMutableGlobals {
		if _, err := graph.AddFact(functionSummaryFact(fn.Name, "touches_mutable_globals", "may_store_global", summarySite(fn), plir.Value{}, ProvenanceSafeKnown, UnsafeSafe, EscapeGlobal, AliasUnknown, "", "semantics summary records mutable global access")); err != nil {
			return err
		}
	}
	return nil
}

func addOperationSummaryFacts(graph *Graph, fn plir.Function, values map[string]plir.Value, factIDs map[plirFactKey]FactID) error {
	for _, op := range fn.Ops {
		switch op.Kind {
		case plir.OpGlobalStore:
			owner := ownerFromOperationInput(op, 0)
			if _, err := graph.AddFact(functionSummaryFact(fn.Name, op.ID, "may_store_global", op.Source, plir.Value{}, ProvenanceSafeKnown, UnsafeSafe, EscapeGlobal, AliasUnknown, owner, "operation stores into global state")); err != nil {
				return err
			}
		case plir.OpActorSend:
			owner := ownerFromOperationInput(op, 1)
			if _, err := graph.AddFact(functionSummaryFact(fn.Name, op.ID, "may_escape_to_actor", op.Source, plir.Value{}, ProvenanceSafeKnown, UnsafeSafe, EscapeActor, AliasUnknown, owner, "operation transfers payload across actor boundary")); err != nil {
				return err
			}
		case plir.OpClosure:
			owner := strings.Join(op.Inputs, ",")
			if _, err := graph.AddFact(functionSummaryFact(fn.Name, op.ID, "may_capture_in_closure", op.Source, plir.Value{}, ProvenanceSafeKnown, UnsafeSafe, EscapeConservative, AliasUnknown, owner, "closure captures visible environment values")); err != nil {
				return err
			}
		case plir.OpCall:
			if isTaskEscapeOperation(op) {
				if _, err := graph.AddFact(functionSummaryFact(fn.Name, op.ID, "may_escape_to_task", op.Source, plir.Value{}, ProvenanceSafeKnown, UnsafeSafe, EscapeTask, AliasUnknown, strings.Join(op.Inputs, ","), "operation may transfer work or handles across task boundary")); err != nil {
					return err
				}
			}
			if err := addNoAliasCallBoundaryFactsForOperation(graph, fn.Name, op, values, factIDs); err != nil {
				return err
			}
			if isUnknownExternalCallOperation(op) {
				if _, err := graph.AddFact(functionSummaryFact(fn.Name, op.ID, "unknown_external_call_conservative", op.Source, plir.Value{}, ProvenanceUnsafeUnknown, UnsafeUnknown, EscapeConservative, AliasUnknown, strings.Join(op.Inputs, ","), "callee summary is unknown, so memory/resource effects remain conservative")); err != nil {
					return err
				}
				if err := addFFIExternalFactsForOperation(graph, fn.Name, op, values, factIDs); err != nil {
					return err
				}
				if err := addPointerRetentionFactsForOperation(graph, fn.Name, op, values); err != nil {
					return err
				}
			}
		}
	}
	return addPointerRetentionFactsForValues(graph, fn, values)
}

func addNoAliasCallBoundaryFactsForOperation(graph *Graph, functionID string, op plir.Operation, values map[string]plir.Value, factIDs map[plirFactKey]FactID) error {
	note := strings.ToLower(op.Note)
	callbackBoundary := strings.Contains(note, "alias_boundary:function_typed_inout")
	unknownExternalBoundary := strings.Contains(note, "alias_boundary:unknown_external_call") || isUnknownExternalCallOperation(op)
	if !callbackBoundary && !unknownExternalBoundary {
		return nil
	}
	for _, input := range op.Inputs {
		value, ok := plirValueForPath(input, values)
		if !ok {
			continue
		}
		if callbackBoundary {
			parentID := factIDs[plirFactKey{kind: plir.FactBorrowedMut, valueID: value.ID}]
			if parentID != "" {
				if err := addCallbackNoAliasInvalidationFact(graph, parentID, functionID, op, value); err != nil {
					return err
				}
			}
		}
		if unknownExternalBoundary {
			if factIDs[plirFactKey{kind: plir.FactNoAlias, valueID: value.ID}] != "" {
				continue
			}
			parentID := factIDs[plirFactKey{kind: plir.FactBorrowedMut, valueID: value.ID}]
			if parentID != "" {
				if err := addFFINoAliasInvalidationFact(graph, parentID, functionID, op, value); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func addFFIExternalFactsForOperation(graph *Graph, functionID string, op plir.Operation, values map[string]plir.Value, factIDs map[plirFactKey]FactID) error {
	for _, input := range op.Inputs {
		value, ok := plirValueForPath(input, values)
		if !ok {
			continue
		}
		if parentID := factIDs[plirFactKey{kind: plir.FactProvenanceUnknown, valueID: value.ID}]; parentID != "" && value.Type == "ptr" {
			externalID, err := addFFIExternalPointerUnknownFact(graph, parentID, functionID, op, value)
			if err != nil {
				return err
			}
			if err := addExternalPointerProvenanceRejectedFact(graph, externalID, functionID, op, value); err != nil {
				return err
			}
		}
		if parentID := firstFactID(factIDs,
			plirFactKey{kind: plir.FactBorrowedImm, valueID: value.ID},
			plirFactKey{kind: plir.FactBorrowedMut, valueID: value.ID},
		); parentID != "" && value.Type == "ptr" {
			if err := addFFICallMayRetainBorrowFact(graph, parentID, functionID, op, value); err != nil {
				return err
			}
		}
		if parentID := factIDs[plirFactKey{kind: plir.FactNoAlias, valueID: value.ID}]; parentID != "" {
			if err := addFFINoAliasInvalidationFact(graph, parentID, functionID, op, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func addFFIExternalPointerUnknownFact(graph *Graph, parentID FactID, functionID string, op plir.Operation, value plir.Value) (FactID, error) {
	return graph.DeriveFact(parentID, Fact{
		ID:              ffiDerivedFactID(parentID, op.ID, "ffi_pointer_external_unknown"),
		FunctionID:      functionID,
		ValueID:         value.ID,
		SiteID:          nonEmpty(op.Source, value.Source, functionID+":ffi"),
		SourceSpan:      nonEmpty(op.Source, value.Source),
		TypeName:        value.Type,
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceUnsafeUnknown,
		UnsafeClass:     UnsafeUnknown,
		RegionID:        value.Region,
		OwnerID:         ownerForPLIRValue(value),
		BorrowState:     borrowStateForPLIRValue(value),
		EscapeState:     EscapeConservative,
		AliasState:      AliasUnknownConservative,
		Claim:           "ffi_pointer_external_unknown",
		ValidatorName:   "external_pointer_provenance_validator",
		CostClass:       CostConservativeFallback,
		Reason:          "Memory Ideal v7 keeps external pointer provenance unsafe_unknown at FFI boundary",
	})
}

func addExternalPointerProvenanceRejectedFact(graph *Graph, parentID FactID, functionID string, op plir.Operation, value plir.Value) error {
	id, err := graph.DeriveFact(parentID, Fact{
		ID:              ffiDerivedFactID(parentID, op.ID, "external_pointer_provenance_rejected"),
		FunctionID:      functionID,
		ValueID:         value.ID,
		SiteID:          nonEmpty(op.Source, value.Source, functionID+":ffi"),
		SourceSpan:      nonEmpty(op.Source, value.Source),
		TypeName:        value.Type,
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceUnsafeUnknown,
		UnsafeClass:     UnsafeUnknown,
		RegionID:        value.Region,
		OwnerID:         ownerForPLIRValue(value),
		EscapeState:     EscapeConservative,
		AliasState:      AliasUnknownConservative,
		Claim:           "external_pointer_provenance_rejected",
		ValidatorName:   "external_pointer_provenance_validator",
		CostClass:       CostUnsupportedRejected,
		Reason:          "Memory Ideal v7 rejects provenance_known promotion from external pointer without compiler-owned proof",
	})
	if err != nil {
		return err
	}
	return graph.InvalidateFact(id, "external pointer cannot become provenance_known without compiler-owned proof")
}

func addFFICallMayRetainBorrowFact(graph *Graph, parentID FactID, functionID string, op plir.Operation, value plir.Value) error {
	_, err := graph.DeriveFact(parentID, Fact{
		ID:              ffiDerivedFactID(parentID, op.ID, "ffi_call_may_retain_borrow"),
		FunctionID:      functionID,
		ValueID:         value.ID,
		SiteID:          nonEmpty(op.Source, value.Source, functionID+":ffi"),
		SourceSpan:      nonEmpty(op.Source, value.Source),
		TypeName:        value.Type,
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceUnsafeUnknown,
		UnsafeClass:     UnsafeUnknown,
		RegionID:        value.Region,
		OwnerID:         ownerForPLIRValue(value),
		BorrowState:     borrowStateForPLIRValue(value),
		EscapeState:     EscapeConservative,
		AliasState:      AliasUnknownConservative,
		Claim:           "ffi_call_may_retain_borrow",
		ValidatorName:   "ffi_lifetime_conservative_validator",
		CostClass:       CostConservativeFallback,
		Reason:          "Memory Ideal v7 keeps borrowed pointer passed to FFI conservative because external call may retain it",
	})
	return err
}

func addCallbackNoAliasInvalidationFact(graph *Graph, parentID FactID, functionID string, op plir.Operation, value plir.Value) error {
	_, err := graph.DeriveFact(parentID, Fact{
		ID:              aliasBoundaryDerivedFactID(parentID, op.ID, "callback_inout_conservative"),
		FunctionID:      functionID,
		ValueID:         value.ID,
		SiteID:          nonEmpty(op.Source, value.Source, functionID+":callback"),
		SourceSpan:      nonEmpty(op.Source, value.Source),
		TypeName:        value.Type,
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceUnsafeUnknown,
		UnsafeClass:     UnsafeUnknown,
		RegionID:        value.Region,
		OwnerID:         ownerForPLIRValue(value),
		BorrowState:     borrowStateForPLIRValue(value),
		EscapeState:     EscapeConservative,
		AliasState:      AliasInvalidatedByCall,
		Claim:           "callback_inout_conservative",
		ValidatorName:   "callback_alias_conservative_validator",
		CostClass:       CostConservativeFallback,
		Reason:          "callback or reentrant inout boundary invalidates broad noalias evidence",
	})
	return err
}

func addFFINoAliasInvalidationFact(graph *Graph, parentID FactID, functionID string, op plir.Operation, value plir.Value) error {
	_, err := graph.DeriveFact(parentID, Fact{
		ID:              ffiDerivedFactID(parentID, op.ID, "ffi_noalias_invalidated_by_external_call"),
		FunctionID:      functionID,
		ValueID:         value.ID,
		SiteID:          nonEmpty(op.Source, value.Source, functionID+":ffi"),
		SourceSpan:      nonEmpty(op.Source, value.Source),
		TypeName:        value.Type,
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceUnsafeUnknown,
		UnsafeClass:     UnsafeUnknown,
		RegionID:        value.Region,
		OwnerID:         ownerForPLIRValue(value),
		BorrowState:     borrowStateForPLIRValue(value),
		EscapeState:     EscapeConservative,
		AliasState:      AliasInvalidatedByCall,
		Claim:           "ffi_noalias_invalidated_by_external_call",
		ValidatorName:   "ffi_noalias_conservative_validator",
		CostClass:       CostConservativeFallback,
		Reason:          "Memory Ideal v7 keeps noalias conservative across external call",
	})
	return err
}

func addSafeWrapperPromotionRejectionFact(graph *Graph, functionID string, op plir.Operation, values map[string]plir.Value, factIDs map[plirFactKey]FactID) error {
	for _, input := range op.Inputs {
		value, ok := plirValueForPath(input, values)
		if !ok {
			continue
		}
		parentID := factIDs[plirFactKey{kind: plir.FactProvenanceUnknown, valueID: value.ID}]
		if parentID == "" {
			continue
		}
		id, err := graph.DeriveFact(parentID, Fact{
			ID:              ffiDerivedFactID(parentID, op.ID, "safe_wrapper_promotion_rejected_without_contract"),
			FunctionID:      functionID,
			ValueID:         value.ID,
			SiteID:          nonEmpty(op.Source, value.Source, functionID+":ffi"),
			SourceSpan:      nonEmpty(op.Source, value.Source),
			TypeName:        value.Type,
			SourceStage:     StagePLIR,
			ProvenanceClass: ProvenanceUnsafeUnknown,
			UnsafeClass:     UnsafeUnknown,
			RegionID:        value.Region,
			OwnerID:         ownerForPLIRValue(value),
			EscapeState:     EscapeConservative,
			AliasState:      AliasUnknownConservative,
			Claim:           "safe_wrapper_promotion_rejected_without_contract",
			ValidatorName:   "safe_wrapper_promotion_validator",
			CostClass:       CostUnsupportedRejected,
			Reason:          "Memory Ideal v7 rejects safe wrapper promotion from external pointer without compiler-owned contract",
		})
		if err != nil {
			return err
		}
		return graph.InvalidateFact(id, "safe wrapper promotion from external pointer requires compiler-owned contract")
	}
	return nil
}

func firstFactID(factIDs map[plirFactKey]FactID, keys ...plirFactKey) FactID {
	for _, key := range keys {
		if id := factIDs[key]; id != "" {
			return id
		}
	}
	return ""
}

func addFactKindSummaryFacts(graph *Graph, fn plir.Function, values map[string]plir.Value, factIDs map[plirFactKey]FactID) error {
	for _, value := range values {
		if value.Kind == plir.ValueParam && value.Borrow == plir.BorrowMove {
			provenance, unsafeClass := summaryClassesForPLIRValue(value, factIDs, ProvenanceSafeOwned)
			if _, err := graph.AddFact(functionSummaryFact(fn.Name, "consume:"+value.ID, "may_consume_param", nonEmpty(value.Source, summarySite(fn)), value, provenance, unsafeClass, escapeStateForPLIRValue(value), AliasUnknown, ownerForPLIRValue(value), "consume parameter may be moved by this function")); err != nil {
				return err
			}
		}
	}
	for _, pf := range fn.Facts {
		value, ok := values[pf.ValueID]
		if !ok {
			continue
		}
		switch pf.Kind {
		case plir.FactMoved:
			if value.Kind == plir.ValueParam {
				provenance, unsafeClass := summaryClassesForPLIRValue(value, factIDs, ProvenanceSafeOwned)
				if _, err := graph.AddFact(functionSummaryFact(fn.Name, pf.ID, "may_consume_param", pf.Source, value, provenance, unsafeClass, escapeStateForPLIRValue(value), AliasUnknown, ownerForPLIRValue(value), "parameter may be consumed or moved by this function")); err != nil {
					return err
				}
			}
		case plir.FactBorrowedMut, plir.FactNoAlias:
			if value.Kind == plir.ValueParam || value.Borrow == plir.BorrowMut {
				provenance, unsafeClass := summaryClassesForPLIRValue(value, factIDs, ProvenanceSafeKnown)
				alias := AliasMutableExclusive
				if unsafeClass == UnsafeUnknown {
					alias = AliasUnknownConservative
				}
				if _, err := graph.AddFact(functionSummaryFact(fn.Name, pf.ID, "may_mutate_inout", pf.Source, value, provenance, unsafeClass, escapeStateForPLIRValue(value), alias, ownerForPLIRValue(value), "inout parameter may be mutated under exclusive-borrow evidence")); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func functionSummaryFact(functionID, anchor, claim, site string, value plir.Value, provenance ProvenanceClass, unsafeClass UnsafeClass, escape EscapeState, alias AliasState, owner string, reason string) Fact {
	if site == "" {
		site = functionID + ":summary"
	}
	if owner == "" {
		owner = ownerForPLIRValue(value)
	}
	return Fact{
		ID:               summaryFactID(functionID, anchor, claim),
		FunctionID:       functionID,
		ValueID:          value.ID,
		SiteID:           site,
		SourceSpan:       site,
		TypeName:         value.Type,
		SourceStage:      StagePLIR,
		ProvenanceClass:  provenance,
		UnsafeClass:      unsafeClass,
		RegionID:         value.Region,
		OwnerID:          owner,
		BorrowState:      borrowStateForPLIRValue(value),
		EscapeState:      nonEmptyEscape(escape, escapeStateForPLIRValue(value)),
		AliasState:       alias,
		AllocationSiteID: allocationSiteIDForPLIRValue(value),
		Claim:            claim,
		Reason:           reason,
	}
}

func summaryFactID(functionID, anchor, claim string) FactID {
	return FactID(fmt.Sprintf("plir:%s:summary:%s:%s", safeFactIDPart(functionID), safeFactIDPart(anchor), safeFactIDPart(claim)))
}

func safeFactIDPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "function"
	}
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			b.WriteByte(c)
			continue
		}
		fmt.Fprintf(&b, "_x%02x", c)
	}
	return b.String()
}

func summarySite(fn plir.Function) string {
	if len(fn.Values) > 0 && fn.Values[0].Source != "" {
		return fn.Values[0].Source
	}
	return fn.Name + ":summary"
}

func formatSummaryLeaf(leaf string) string {
	if strings.TrimSpace(leaf) == "" {
		return ""
	}
	return "." + strings.TrimPrefix(strings.TrimSpace(leaf), ".")
}

func summaryParamOwner(summary *plir.FunctionSummary, index int) string {
	if summary == nil || index < 0 || index >= len(summary.ParamNames) {
		return ""
	}
	return summary.ParamNames[index]
}

func summaryParamIndex(summary *plir.FunctionSummary, owner string) (int, bool) {
	if summary == nil {
		return -1, false
	}
	owner = normalizeOwnerID(owner)
	for index, name := range summary.ParamNames {
		if normalizeOwnerID(name) == owner {
			return index, true
		}
	}
	return -1, false
}

func summaryRequiresCapabilities(summary *plir.FunctionSummary) bool {
	if summary == nil {
		return false
	}
	for _, effect := range summary.Effects {
		if effect == "capability" {
			return true
		}
	}
	return false
}

func summaryRequiresCapMemAuthorization(summary *plir.FunctionSummary) bool {
	if summary == nil {
		return false
	}
	hasCapability := false
	hasMem := false
	for _, effect := range summary.Effects {
		switch effect {
		case "capability":
			hasCapability = true
		case "mem":
			hasMem = true
		}
	}
	return hasCapability && hasMem
}

func plirValueForPath(path string, values map[string]plir.Value) (plir.Value, bool) {
	valueID := valueIDForPath(path, values)
	if valueID == "" {
		return plir.Value{}, false
	}
	value, ok := values[valueID]
	return value, ok
}

func returnSummaryValues(path string, values map[string]plir.Value) []plir.Value {
	if value, ok := plirValueForPath(path, values); ok {
		return []plir.Value{value}
	}
	var out []plir.Value
	for _, candidate := range []string{"view:$return", "alloc_intent:$return", "local:$return", "param:$return"} {
		if value, ok := values[candidate]; ok {
			out = append(out, value)
		}
	}
	for _, value := range values {
		if value.ID == "" || !strings.Contains(value.ID, ":$return") {
			continue
		}
		if containsPLIRValue(out, value.ID) {
			continue
		}
		out = append(out, value)
	}
	return out
}

func containsPLIRValue(values []plir.Value, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}

func plirValueIsUnsafeUnknown(value plir.Value, factIDs map[plirFactKey]FactID) bool {
	if value.Provenance.Kind == plir.ProvenanceExternal || value.Provenance.Kind == plir.ProvenanceUnknown {
		return true
	}
	return valueHasAnyFact(value.ID, factIDs, plir.FactProvenanceUnknown)
}

func returnValueIsOwnedAllocation(value plir.Value, factIDs map[plirFactKey]FactID) bool {
	if value.Alloc != nil && value.Alloc.Builtin != "core.alloc_bytes" {
		return true
	}
	return valueHasAnyFact(value.ID, factIDs, plir.FactOwned)
}

type returnedBorrowSourceInfo struct {
	Owner      string
	ParamIndex *int
	ParamPath  string
}

func (info returnedBorrowSourceInfo) ParamIndexValue() (int, bool) {
	if info.ParamIndex == nil {
		return -1, false
	}
	return *info.ParamIndex, true
}

func returnedBorrowSource(fn plir.Function, value plir.Value, factIDs map[plirFactKey]FactID) (returnedBorrowSourceInfo, bool) {
	if value.Provenance.Kind != plir.ProvenanceParam {
		return returnedBorrowSourceInfo{}, false
	}
	ownerPath := ownerPathForPLIRValue(value)
	if ownerPath == "" {
		return returnedBorrowSourceInfo{}, false
	}
	owner := normalizeOwnerID(ownerPath)
	if owner == "" {
		return returnedBorrowSourceInfo{}, false
	}
	if !(value.Borrow != plir.BorrowNone ||
		valueHasAnyFact(value.ID, factIDs, plir.FactBorrowedImm, plir.FactBorrowedMut) ||
		(fn.Summary != nil && fn.Summary.ReturnOwnership == "borrow")) {
		return returnedBorrowSourceInfo{}, false
	}
	info := returnedBorrowSourceInfo{Owner: owner}
	if strings.HasPrefix(ownerPath, owner+".") {
		info.ParamPath = strings.TrimPrefix(ownerPath, owner+".")
	}
	if index, ok := summaryParamIndex(fn.Summary, owner); ok {
		info.ParamIndex = paramIndexPtr(index)
	}
	return info, true
}

func ownerPathForPLIRValue(value plir.Value) string {
	return normalizeOwnerPath(nonEmpty(value.Provenance.Root, value.Lifetime.Owner))
}

func normalizeOwnerPath(owner string) string {
	owner = strings.TrimSpace(owner)
	for strings.HasPrefix(owner, "derived:") {
		owner = strings.TrimPrefix(owner, "derived:")
	}
	owner = strings.TrimPrefix(owner, "param:")
	owner = strings.TrimPrefix(owner, "local:")
	owner = strings.TrimPrefix(owner, "view:")
	owner = strings.TrimPrefix(owner, "alloc_intent:")
	return owner
}

func paramIndexPtr(index int) *int {
	if index < 0 {
		return nil
	}
	return &index
}

func summaryClassesForPLIRValue(value plir.Value, factIDs map[plirFactKey]FactID, safe ProvenanceClass) (ProvenanceClass, UnsafeClass) {
	if plirValueIsUnsafeUnknown(value, factIDs) {
		return ProvenanceUnsafeUnknown, UnsafeUnknown
	}
	return safe, UnsafeSafe
}

func valueHasAnyFact(valueID string, factIDs map[plirFactKey]FactID, kinds ...plir.FactKind) bool {
	if valueID == "" {
		return false
	}
	for _, kind := range kinds {
		if factIDs[plirFactKey{kind: kind, valueID: valueID}] != "" {
			return true
		}
	}
	return false
}

func borrowStateForPLIRValue(value plir.Value) BorrowState {
	switch value.Borrow {
	case plir.BorrowImm:
		return BorrowImmutable
	case plir.BorrowMut:
		return BorrowMutable
	case plir.BorrowMove:
		return BorrowMoved
	default:
		return BorrowNone
	}
}

func escapeStateForPLIRValue(value plir.Value) EscapeState {
	switch value.Escape {
	case plir.EscapeNoEscape:
		return EscapeNoEscape
	case plir.EscapeReturn:
		return EscapeReturn
	case plir.EscapeGlobal:
		return EscapeGlobal
	case plir.EscapeActor:
		return EscapeActor
	case plir.EscapeTask:
		return EscapeTask
	case plir.EscapeUnsafe, plir.EscapeCallUnknown, plir.EscapeClosure, plir.EscapeAggregate:
		return EscapeUnsafe
	case plir.EscapeConservative:
		return EscapeConservative
	default:
		return EscapeUnknown
	}
}

func nonEmptyEscape(values ...EscapeState) EscapeState {
	for _, value := range values {
		if value != EscapeUnknown {
			return value
		}
	}
	return EscapeUnknown
}

func isTaskEscapeOperation(op plir.Operation) bool {
	if op.Kind != plir.OpCall {
		return false
	}
	note := strings.ToLower(strings.TrimSpace(op.Note))
	return strings.Contains(note, "task_spawn") || strings.Contains(note, "task_group")
}

func isUnknownExternalCallOperation(op plir.Operation) bool {
	if op.Kind != plir.OpCall {
		return false
	}
	note := strings.ToLower(strings.TrimSpace(op.Note))
	if strings.Contains(note, "unknown external") || strings.Contains(note, "external call") {
		return true
	}
	return strings.HasPrefix(note, "ffi.") || strings.Contains(note, " extern")
}

func addPointerRetentionFactsForOperation(graph *Graph, functionID string, op plir.Operation, values map[string]plir.Value) error {
	for _, input := range op.Inputs {
		value, ok := plirValueForPath(input, values)
		if !ok || !plirValueMayRetainPointer(value) {
			continue
		}
		if _, err := graph.AddFact(functionSummaryFact(functionID, op.ID+":"+value.ID, "may_retain_pointer", op.Source, value, ProvenanceUnsafeUnknown, UnsafeUnknown, EscapeConservative, AliasUnknown, ownerForPLIRValue(value), "unknown external call may retain pointer argument")); err != nil {
			return err
		}
	}
	return nil
}

func addPointerRetentionFactsForValues(graph *Graph, fn plir.Function, values map[string]plir.Value) error {
	for _, value := range values {
		if !plirValueMayRetainPointer(value) {
			continue
		}
		if _, err := graph.AddFact(functionSummaryFact(fn.Name, "retain:"+value.ID, "may_retain_pointer", nonEmpty(value.Source, summarySite(fn)), value, ProvenanceUnsafeUnknown, UnsafeUnknown, EscapeConservative, AliasUnknown, ownerForPLIRValue(value), "pointer value may escape or be retained outside a proven safe owner")); err != nil {
			return err
		}
	}
	return nil
}

func plirValueMayRetainPointer(value plir.Value) bool {
	if value.Type != "ptr" {
		return false
	}
	switch value.Escape {
	case plir.EscapeNoEscape:
		return false
	default:
		return true
	}
}

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
			if alloc.RawPointerBoundsStatus == "" && alloc.PlannedStorage == "" && alloc.ActualLoweringStorage == "" {
				continue
			}
			if allocPlanTrustedStorageHeapFallback(alloc.PlannedStorage, alloc.ActualLoweringStorage) && strings.TrimSpace(alloc.Reason) == "" {
				return fmt.Errorf("allocplan %s:%s heap fallback missing storage reason", fn.Name, alloc.ID)
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
			loweredArtifactID := fmt.Sprintf("ir:%s:%s:%s", fn.Name, alloc.ID, alloc.ActualLoweringStorage)
			validated := allocPlanValidationPasses(alloc)
			validationState := ValidationNotRun
			validatorName := ""
			if validated {
				validationState = ValidationPass
				validatorName = "allocation_lowering_validator"
			}
			id, err := graph.AddFact(Fact{
				ID:                    FactID(fmt.Sprintf("allocplan:%s:%s", fn.Name, alloc.ID)),
				FunctionID:            fn.Name,
				ValueID:               alloc.ValueID,
				IslandID:              ref.IslandID,
				Epoch:                 ref.Epoch,
				BaseID:                ref.BaseID,
				SiteID:                nonEmpty(alloc.SiteID, fmt.Sprintf("%s:%s", fn.Name, alloc.ID)),
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
			if validated && alloc.Builtin == "core.alloc_bytes" && claim == "allocation_base_metadata" {
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

func allocPlanMemoryRef(functionName string, alloc allocplan.Allocation, refs map[string]allocMemoryRef) allocMemoryRef {
	if ref, ok := refs[allocMemoryRefKey(functionName, alloc.ValueID)]; ok && ref.IslandID != "" {
		return ref
	}
	if alloc.PlannedStorage != allocplan.StorageExplicitIsland && alloc.ActualLoweringStorage != allocplan.StorageExplicitIsland {
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

func addUnsafeVerifiedRootAllocationBaseFact(graph *Graph, parentID FactID, functionID string, alloc allocplan.Allocation) error {
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
		Reason:           "Memory Ideal v5 accepts bounded core.alloc_bytes allocation-base metadata without safe-fact promotion",
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

func allocPlanRuntimeProofRequiredStorage(planned allocplan.StorageClass, actual allocplan.StorageClass) bool {
	return runtimeProofRequiredStorage(StorageClass(planned), StorageClass(actual))
}

func allocPlanTrustedStorageHeapFallback(planned allocplan.StorageClass, actual allocplan.StorageClass) bool {
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

func addBorrowMetadataFacts(graph *Graph, parent Fact, value plir.Value, op plir.Operation) error {
	if isUnsafeUnknown(parent) {
		return nil
	}
	owner := nonEmpty(ownerFromOperationInput(op, 0), ownerForPLIRValue(value))
	if owner == "" {
		return nil
	}
	if _, err := graph.DeriveFact(parent.ID, Fact{
		ID:              derivedFactID(parent.ID, "borrow_owner"),
		FunctionID:      parent.FunctionID,
		ValueID:         parent.ValueID,
		SiteID:          parent.SiteID,
		SourceSpan:      parent.SourceSpan,
		TypeName:        parent.TypeName,
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceSafeBorrowed,
		UnsafeClass:     UnsafeSafe,
		RegionID:        parent.RegionID,
		OwnerID:         owner,
		BorrowState:     parent.BorrowState,
		EscapeState:     EscapeNoEscape,
		Claim:           "borrow_owner",
		Reason:          fmt.Sprintf("borrowed view owner %q is visible in PLIR provenance", owner),
	}); err != nil {
		return err
	}
	_, err := graph.DeriveFact(parent.ID, Fact{
		ID:              derivedFactID(parent.ID, "borrow_source_fact_id"),
		FunctionID:      parent.FunctionID,
		ValueID:         parent.ValueID,
		SiteID:          parent.SiteID,
		SourceSpan:      parent.SourceSpan,
		TypeName:        parent.TypeName,
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceSafeBorrowed,
		UnsafeClass:     UnsafeSafe,
		RegionID:        parent.RegionID,
		OwnerID:         owner,
		BorrowState:     parent.BorrowState,
		EscapeState:     EscapeNoEscape,
		Claim:           "borrow_source_fact_id",
		Reason:          fmt.Sprintf("borrowed view source fact_id is %s", parent.ID),
	})
	return err
}

func addBorrowAggregateV0Facts(graph *Graph, parent Fact, value plir.Value, op plir.Operation) error {
	ownerPath := ownerPathForPLIRValue(value)
	owner := nonEmpty(ownerFromOperationInput(op, 0), normalizeOwnerID(ownerPath), parent.OwnerID)
	if owner == "" {
		return nil
	}
	paramPath := ""
	if strings.HasPrefix(ownerPath, owner+".") {
		paramPath = strings.TrimPrefix(ownerPath, owner+".")
	}
	claim, ok := memoryIdealBorrowWrapperClaim(value, op, parent, paramPath)
	if !ok {
		return nil
	}
	if isUnsafeUnknown(parent) {
		if claim == "witness_provenance_promotion_rejected" {
			id, err := graph.DeriveFact(parent.ID, Fact{
				ID:              derivedFactID(parent.ID, claim),
				FunctionID:      parent.FunctionID,
				ValueID:         parent.ValueID,
				SiteID:          parent.SiteID,
				SourceSpan:      parent.SourceSpan,
				TypeName:        parent.TypeName,
				SourceStage:     StagePLIR,
				ProvenanceClass: ProvenanceUnsafeUnknown,
				UnsafeClass:     UnsafeUnknown,
				RegionID:        parent.RegionID,
				OwnerID:         owner,
				ParamPath:       paramPath,
				BorrowState:     parent.BorrowState,
				EscapeState:     EscapeConservative,
				Claim:           claim,
				ValidatorName:   borrowWrapperValidatorName(claim),
				CostClass:       CostUnsupportedRejected,
				Reason:          fmt.Sprintf("Memory Ideal v11 rejects witness provenance promotion for owner %q", owner),
			})
			if err != nil {
				return err
			}
			return graph.InvalidateFact(id, fmt.Sprintf("Memory Ideal v11 rejects witness provenance promotion for owner %q", owner))
		}
		return nil
	}
	if claim == "dynamic_existential_borrow_conservative" {
		_, err := graph.DeriveFact(parent.ID, Fact{
			ID:              derivedFactID(parent.ID, claim),
			FunctionID:      parent.FunctionID,
			ValueID:         parent.ValueID,
			SiteID:          parent.SiteID,
			SourceSpan:      parent.SourceSpan,
			TypeName:        parent.TypeName,
			SourceStage:     StagePLIR,
			ProvenanceClass: ProvenanceUnsafeUnknown,
			UnsafeClass:     UnsafeUnknown,
			RegionID:        parent.RegionID,
			OwnerID:         owner,
			ParamPath:       paramPath,
			BorrowState:     parent.BorrowState,
			EscapeState:     EscapeConservative,
			AliasState:      AliasUnknownConservative,
			Claim:           claim,
			ValidatorName:   borrowWrapperValidatorName(claim),
			CostClass:       CostConservativeFallback,
			Reason:          fmt.Sprintf("Memory Ideal v11 keeps dynamic existential/protocol borrow conservative for owner %q", owner),
		})
		return err
	}
	if claim == "static_witness_borrow_parent_validated" || claim == "protocol_dispatch_report_integrity" {
		cost := CostZeroCostProven
		normalBuildCheck := false
		if claim == "protocol_dispatch_report_integrity" {
			cost = CostDynamicCheckRequired
			normalBuildCheck = true
		}
		id, err := graph.DeriveFact(parent.ID, Fact{
			ID:               derivedFactID(parent.ID, claim),
			FunctionID:       parent.FunctionID,
			ValueID:          parent.ValueID,
			SiteID:           parent.SiteID,
			SourceSpan:       parent.SourceSpan,
			TypeName:         parent.TypeName,
			SourceStage:      StagePLIR,
			ProvenanceClass:  ProvenanceSafeBorrowed,
			UnsafeClass:      UnsafeSafe,
			RegionID:         parent.RegionID,
			OwnerID:          owner,
			ParamPath:        paramPath,
			BorrowState:      parent.BorrowState,
			EscapeState:      EscapeNoEscape,
			Claim:            claim,
			CostClass:        cost,
			NormalBuildCheck: normalBuildCheck,
			Reason:           fmt.Sprintf("Memory Ideal v11 validates %s for owner %q with compiler-owned parent fact", claim, owner),
		})
		if err != nil {
			return err
		}
		return graph.MarkValidated(id, borrowWrapperValidatorName(claim))
	}
	if claim == "pre_await_local_borrow_validated" {
		id, err := graph.DeriveFact(parent.ID, Fact{
			ID:              derivedFactID(parent.ID, claim),
			FunctionID:      parent.FunctionID,
			ValueID:         parent.ValueID,
			SiteID:          parent.SiteID,
			SourceSpan:      parent.SourceSpan,
			TypeName:        parent.TypeName,
			SourceStage:     StagePLIR,
			ProvenanceClass: ProvenanceSafeBorrowed,
			UnsafeClass:     UnsafeSafe,
			RegionID:        parent.RegionID,
			OwnerID:         owner,
			ParamPath:       paramPath,
			BorrowState:     parent.BorrowState,
			EscapeState:     EscapeNoEscape,
			Claim:           claim,
			ValidationState: ValidationPass,
			ValidatorName:   borrowWrapperValidatorName(claim),
			CostClass:       CostZeroCostProven,
			Reason:          fmt.Sprintf("Memory Ideal v10 validates pre-await local borrow for owner %q only with compiler-owned no-escape proof", owner),
		})
		if err != nil {
			return err
		}
		return graph.MarkValidated(id, borrowWrapperValidatorName(claim))
	}
	if claim == "post_await_borrow_conservative" || claim == "actor_reentrant_callback_conservative" {
		_, err := graph.DeriveFact(parent.ID, Fact{
			ID:              derivedFactID(parent.ID, claim),
			FunctionID:      parent.FunctionID,
			ValueID:         parent.ValueID,
			SiteID:          parent.SiteID,
			SourceSpan:      parent.SourceSpan,
			TypeName:        parent.TypeName,
			SourceStage:     StagePLIR,
			ProvenanceClass: ProvenanceUnsafeUnknown,
			UnsafeClass:     UnsafeUnknown,
			RegionID:        parent.RegionID,
			OwnerID:         owner,
			ParamPath:       paramPath,
			BorrowState:     parent.BorrowState,
			EscapeState:     EscapeConservative,
			AliasState:      AliasUnknownConservative,
			Claim:           claim,
			ValidatorName:   borrowWrapperValidatorName(claim),
			CostClass:       CostConservativeFallback,
			Reason:          fmt.Sprintf("Memory Ideal v10 keeps %s conservative for owner %q", claim, owner),
		})
		return err
	}
	if claim == "cancellation_borrow_lifetime_invalidated" {
		id, err := graph.DeriveFact(parent.ID, Fact{
			ID:              derivedFactID(parent.ID, claim),
			FunctionID:      parent.FunctionID,
			ValueID:         parent.ValueID,
			SiteID:          parent.SiteID,
			SourceSpan:      parent.SourceSpan,
			TypeName:        parent.TypeName,
			SourceStage:     StagePLIR,
			ProvenanceClass: ProvenanceSafeBorrowed,
			UnsafeClass:     UnsafeSafe,
			RegionID:        parent.RegionID,
			OwnerID:         owner,
			ParamPath:       paramPath,
			BorrowState:     parent.BorrowState,
			EscapeState:     EscapeTask,
			Claim:           claim,
			ValidatorName:   borrowWrapperValidatorName(claim),
			CostClass:       CostUnsupportedRejected,
			Reason:          fmt.Sprintf("Memory Ideal v10 rejects task-owned borrow lifetime after cancellation for owner %q", owner),
		})
		if err != nil {
			return err
		}
		return graph.InvalidateFact(id, fmt.Sprintf("Memory Ideal v10 rejects task-owned borrow lifetime after cancellation for owner %q", owner))
	}
	if claim == "protocol_dispatch_borrow_conservative" {
		_, err := graph.DeriveFact(parent.ID, Fact{
			ID:              derivedFactID(parent.ID, claim),
			FunctionID:      parent.FunctionID,
			ValueID:         parent.ValueID,
			SiteID:          parent.SiteID,
			SourceSpan:      parent.SourceSpan,
			TypeName:        parent.TypeName,
			SourceStage:     StagePLIR,
			ProvenanceClass: ProvenanceUnsafeUnknown,
			UnsafeClass:     UnsafeUnknown,
			RegionID:        parent.RegionID,
			OwnerID:         owner,
			ParamPath:       paramPath,
			BorrowState:     parent.BorrowState,
			EscapeState:     EscapeConservative,
			Claim:           claim,
			ValidatorName:   borrowWrapperValidatorName(claim),
			CostClass:       CostConservativeFallback,
			Reason:          fmt.Sprintf("Memory Ideal v3 keeps protocol dispatch borrow conservative for owner %q", owner),
		})
		return err
	}
	if claim == "async_boundary_borrow_conservative" {
		_, err := graph.DeriveFact(parent.ID, Fact{
			ID:              derivedFactID(parent.ID, claim),
			FunctionID:      parent.FunctionID,
			ValueID:         parent.ValueID,
			SiteID:          parent.SiteID,
			SourceSpan:      parent.SourceSpan,
			TypeName:        parent.TypeName,
			SourceStage:     StagePLIR,
			ProvenanceClass: ProvenanceUnsafeUnknown,
			UnsafeClass:     UnsafeUnknown,
			RegionID:        parent.RegionID,
			OwnerID:         owner,
			ParamPath:       paramPath,
			BorrowState:     parent.BorrowState,
			EscapeState:     EscapeConservative,
			Claim:           claim,
			ValidatorName:   borrowWrapperValidatorName(claim),
			CostClass:       CostConservativeFallback,
			Reason:          fmt.Sprintf("Memory Ideal v4 keeps async boundary borrow conservative for owner %q", owner),
		})
		return err
	}
	if claim == "task_boundary_borrow_rejected" || claim == "actor_boundary_borrow_rejected" {
		escape := EscapeTask
		if claim == "actor_boundary_borrow_rejected" {
			escape = EscapeActor
		}
		id, err := graph.DeriveFact(parent.ID, Fact{
			ID:              derivedFactID(parent.ID, claim),
			FunctionID:      parent.FunctionID,
			ValueID:         parent.ValueID,
			SiteID:          parent.SiteID,
			SourceSpan:      parent.SourceSpan,
			TypeName:        parent.TypeName,
			SourceStage:     StagePLIR,
			ProvenanceClass: ProvenanceSafeBorrowed,
			UnsafeClass:     UnsafeSafe,
			RegionID:        parent.RegionID,
			OwnerID:         owner,
			ParamPath:       paramPath,
			BorrowState:     parent.BorrowState,
			EscapeState:     escape,
			Claim:           claim,
			ValidatorName:   borrowWrapperValidatorName(claim),
			CostClass:       CostUnsupportedRejected,
			Reason:          fmt.Sprintf("Memory Ideal v4 rejects %s for owner %q without explicit copy", claim, owner),
		})
		if err != nil {
			return err
		}
		return graph.InvalidateFact(id, fmt.Sprintf("Memory Ideal v4 rejects %s for owner %q without explicit copy", claim, owner))
	}
	id, err := graph.DeriveFact(parent.ID, Fact{
		ID:              derivedFactID(parent.ID, claim),
		FunctionID:      parent.FunctionID,
		ValueID:         parent.ValueID,
		SiteID:          parent.SiteID,
		SourceSpan:      parent.SourceSpan,
		TypeName:        parent.TypeName,
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceSafeBorrowed,
		UnsafeClass:     UnsafeSafe,
		RegionID:        parent.RegionID,
		OwnerID:         owner,
		ParamPath:       paramPath,
		BorrowState:     parent.BorrowState,
		EscapeState:     EscapeNoEscape,
		Claim:           claim,
		Reason:          fmt.Sprintf("Memory Ideal v0 proves %s for owner %q", claim, owner),
	})
	if err != nil {
		return err
	}
	return graph.MarkValidated(id, borrowWrapperValidatorName(claim))
}

func memoryIdealBorrowWrapperClaim(value plir.Value, op plir.Operation, parent Fact, paramPath string) (string, bool) {
	context := strings.ToLower(strings.Join([]string{
		value.ID,
		value.Provenance.Root,
		value.Lifetime.Owner,
		op.Note,
		parent.Reason,
		paramPath,
	}, " "))
	if (strings.Contains(context, "pre-await") || strings.Contains(context, "pre_await") || strings.Contains(context, "before suspension")) &&
		(strings.Contains(context, "no_escape_proof") || strings.Contains(context, "no-escape proof") || strings.Contains(context, "no escape proof")) {
		return "pre_await_local_borrow_validated", true
	}
	if strings.Contains(context, "post-await") ||
		strings.Contains(context, "post_await") ||
		strings.Contains(context, "after await") ||
		strings.Contains(context, "after suspension") {
		return "post_await_borrow_conservative", true
	}
	if strings.Contains(context, "cancellation") || strings.Contains(context, "cancel") {
		return "cancellation_borrow_lifetime_invalidated", true
	}
	if strings.Contains(context, "actor reentrant") ||
		strings.Contains(context, "actor_reentrant") ||
		strings.Contains(context, "reentrant_callback") {
		return "actor_reentrant_callback_conservative", true
	}
	if strings.Contains(context, "witness table lookup") ||
		strings.Contains(context, "witness_table_lookup") ||
		strings.Contains(context, "witness lookup") {
		return "witness_provenance_promotion_rejected", true
	}
	if strings.Contains(context, "report integrity") ||
		(strings.Contains(context, "normal_build_check") && strings.Contains(context, "source_fact_id") && strings.Contains(context, "cost_class")) {
		return "protocol_dispatch_report_integrity", true
	}
	if strings.Contains(context, "static witness") ||
		strings.Contains(context, "static_witness") ||
		strings.Contains(context, "conformance proof") ||
		strings.Contains(context, "compiler-owned parent fact") {
		return "static_witness_borrow_parent_validated", true
	}
	if strings.Contains(context, "dynamic existential") ||
		strings.Contains(context, "dynamic_existential") ||
		strings.Contains(context, "existential protocol") {
		return "dynamic_existential_borrow_conservative", true
	}
	if strings.Contains(context, "callback_arg") || strings.Contains(context, "callback arg") || strings.Contains(context, "callback parameter") {
		return "callback_arg_contains_borrow", true
	}
	if strings.Contains(context, "async_boundary") || strings.Contains(context, "async boundary") || strings.Contains(context, "async.boundary") || strings.Contains(context, "await") || strings.Contains(context, "suspension") {
		return "async_boundary_borrow_conservative", true
	}
	if strings.Contains(context, "task_boundary") || strings.Contains(context, "task boundary") || strings.Contains(context, "task.boundary") || strings.Contains(context, "task_spawn") {
		return "task_boundary_borrow_rejected", true
	}
	if strings.Contains(context, "actor_boundary") || strings.Contains(context, "actor boundary") || strings.Contains(context, "actor.boundary") || strings.Contains(context, "send_typed") {
		return "actor_boundary_borrow_rejected", true
	}
	if strings.Contains(context, "protocol_dispatch") || strings.Contains(context, "protocol dispatch") || strings.Contains(context, "dynamic dispatch") || strings.Contains(context, "dynamic protocol") {
		return "protocol_dispatch_borrow_conservative", true
	}
	if strings.Contains(context, "interface_value") || strings.Contains(context, "interface value") || strings.Contains(context, "protocol_value") || strings.Contains(context, "protocol value") || strings.Contains(context, "interface/protocol value") {
		return "interface_value_contains_borrow", true
	}
	if strings.Contains(context, "function_value") || strings.Contains(context, "function value") || strings.Contains(context, "function-typed value") || strings.Contains(context, "function typed value") {
		return "function_value_contains_borrow", true
	}
	if strings.Contains(context, "generic_wrapper") || strings.Contains(context, "generic wrapper") || strings.Contains(context, "monomorphized generic") || strings.Contains(context, "box<") {
		return "generic_wrapper_contains_borrow", true
	}
	if strings.Contains(context, "enum_payload") || strings.Contains(context, "enum payload") || strings.Contains(context, "enum") {
		return "enum_payload_contains_borrow", true
	}
	if strings.Contains(context, "optional") || strings.Contains(context, "maybe") || strings.Contains(context, "payload") {
		return "optional_contains_borrow", true
	}
	if paramPath != "" || strings.Contains(context, "struct") || strings.Contains(context, "field") || strings.Contains(context, "aggregate") {
		return "aggregate_contains_borrow", true
	}
	return "", false
}

func borrowWrapperValidatorName(claim string) string {
	switch claim {
	case "function_value_contains_borrow":
		return "function_value_borrow_escape_validator"
	case "callback_arg_contains_borrow":
		return "callback_borrow_escape_validator"
	case "interface_value_contains_borrow":
		return "interface_borrow_escape_validator"
	case "protocol_dispatch_borrow_conservative":
		return "protocol_dispatch_borrow_validator"
	case "dynamic_existential_borrow_conservative":
		return "dynamic_existential_borrow_conservative_validator"
	case "static_witness_borrow_parent_validated":
		return "static_witness_parent_fact_validator"
	case "witness_provenance_promotion_rejected":
		return "witness_provenance_promotion_validator"
	case "protocol_dispatch_report_integrity":
		return "protocol_dispatch_report_integrity_validator"
	case "async_boundary_borrow_conservative":
		return "async_boundary_borrow_validator"
	case "pre_await_local_borrow_validated":
		return "pre_await_local_borrow_validator"
	case "post_await_borrow_conservative":
		return "post_await_borrow_conservative_validator"
	case "cancellation_borrow_lifetime_invalidated":
		return "cancellation_lifetime_invalidation_validator"
	case "actor_reentrant_callback_conservative":
		return "actor_reentrant_callback_boundary_validator"
	case "task_boundary_borrow_rejected":
		return "task_boundary_borrow_validator"
	case "actor_boundary_borrow_rejected":
		return "actor_boundary_borrow_validator"
	default:
		return "borrow_aggregate_escape_validator"
	}
}

func addCopyMetadataFacts(graph *Graph, parent Fact, value plir.Value, op plir.Operation, factIDs map[plirFactKey]FactID, values map[string]plir.Value) error {
	if isUnsafeUnknown(parent) {
		return nil
	}
	owner := ownerForPLIRValue(value)
	source := ownerFromOperationInput(op, 0)
	sourceFactID := sourceFactIDForPath(source, factIDs, values)
	if _, err := graph.DeriveFact(parent.ID, Fact{
		ID:               derivedFactID(parent.ID, "copy_owned"),
		FunctionID:       parent.FunctionID,
		ValueID:          parent.ValueID,
		SiteID:           parent.SiteID,
		SourceSpan:       parent.SourceSpan,
		TypeName:         parent.TypeName,
		SourceStage:      StagePLIR,
		ProvenanceClass:  ProvenanceSafeOwned,
		UnsafeClass:      UnsafeSafe,
		RegionID:         parent.RegionID,
		OwnerID:          owner,
		AllocationSiteID: parent.AllocationSiteID,
		Claim:            "copy_owned",
		Reason:           "copy result owns new storage and provenance",
	}); err != nil {
		return err
	}
	reason := fmt.Sprintf("copy source value %q is recorded in PLIR", source)
	if sourceFactID != "" {
		reason = fmt.Sprintf("copy source fact_id is %s", sourceFactID)
	}
	_, err := graph.DeriveFact(parent.ID, Fact{
		ID:               derivedFactID(parent.ID, "copy_source_fact_id"),
		FunctionID:       parent.FunctionID,
		ValueID:          parent.ValueID,
		SiteID:           parent.SiteID,
		SourceSpan:       parent.SourceSpan,
		TypeName:         parent.TypeName,
		SourceStage:      StagePLIR,
		ProvenanceClass:  ProvenanceSafeOwned,
		UnsafeClass:      UnsafeSafe,
		RegionID:         parent.RegionID,
		OwnerID:          source,
		AllocationSiteID: parent.AllocationSiteID,
		Claim:            "copy_source_fact_id",
		Reason:           reason,
	})
	return err
}

func addNoAliasMetadataFacts(graph *Graph, parent Fact, value plir.Value) error {
	if isUnsafeUnknown(parent) {
		return nil
	}
	owner := nonEmpty(ownerForPLIRValue(value), parent.OwnerID)
	if dynamicProtocolNoAliasRejectedContext(parent, value) {
		id, err := graph.DeriveFact(parent.ID, Fact{
			ID:              derivedFactID(parent.ID, "dynamic_protocol_noalias_rejected"),
			FunctionID:      parent.FunctionID,
			ValueID:         parent.ValueID,
			SiteID:          parent.SiteID,
			SourceSpan:      parent.SourceSpan,
			TypeName:        parent.TypeName,
			SourceStage:     StagePLIR,
			ProvenanceClass: ProvenanceSafeKnown,
			UnsafeClass:     UnsafeSafe,
			RegionID:        parent.RegionID,
			OwnerID:         owner,
			AliasState:      AliasInvalidatedByCall,
			Claim:           "dynamic_protocol_noalias_rejected",
			ValidatorName:   "dynamic_protocol_noalias_rejection_validator",
			CostClass:       CostUnsupportedRejected,
			Reason:          "dynamic protocol dispatch cannot validate broad noalias evidence",
		})
		if err != nil {
			return err
		}
		return graph.InvalidateFact(id, "dynamic protocol dispatch cannot validate broad noalias evidence")
	}
	if taskGroupNoAliasConservativeContext(parent, value) {
		_, err := graph.DeriveFact(parent.ID, Fact{
			ID:              derivedFactID(parent.ID, "task_group_noalias_conservative"),
			FunctionID:      parent.FunctionID,
			ValueID:         parent.ValueID,
			SiteID:          parent.SiteID,
			SourceSpan:      parent.SourceSpan,
			TypeName:        parent.TypeName,
			SourceStage:     StagePLIR,
			ProvenanceClass: ProvenanceUnsafeUnknown,
			UnsafeClass:     UnsafeUnknown,
			RegionID:        parent.RegionID,
			OwnerID:         owner,
			AliasState:      AliasInvalidatedByCall,
			Claim:           "task_group_noalias_conservative",
			ValidatorName:   "task_group_boundary_conservative_validator",
			CostClass:       CostConservativeFallback,
			Reason:          "task group or structured concurrency boundary invalidates broad noalias evidence",
		})
		return err
	}
	if boundaryNoAliasConservativeContext(parent, value) {
		_, err := graph.DeriveFact(parent.ID, Fact{
			ID:              derivedFactID(parent.ID, "boundary_noalias_conservative"),
			FunctionID:      parent.FunctionID,
			ValueID:         parent.ValueID,
			SiteID:          parent.SiteID,
			SourceSpan:      parent.SourceSpan,
			TypeName:        parent.TypeName,
			SourceStage:     StagePLIR,
			ProvenanceClass: ProvenanceUnsafeUnknown,
			UnsafeClass:     UnsafeUnknown,
			RegionID:        parent.RegionID,
			OwnerID:         owner,
			AliasState:      AliasInvalidatedByCall,
			Claim:           "boundary_noalias_conservative",
			ValidatorName:   "boundary_alias_conservative_validator",
			CostClass:       CostConservativeFallback,
			Reason:          "task or actor boundary invalidates broad noalias evidence",
		})
		return err
	}
	if protocolDispatchNoAliasConservativeContext(parent, value) {
		_, err := graph.DeriveFact(parent.ID, Fact{
			ID:              derivedFactID(parent.ID, "protocol_dispatch_noalias_conservative"),
			FunctionID:      parent.FunctionID,
			ValueID:         parent.ValueID,
			SiteID:          parent.SiteID,
			SourceSpan:      parent.SourceSpan,
			TypeName:        parent.TypeName,
			SourceStage:     StagePLIR,
			ProvenanceClass: ProvenanceUnsafeUnknown,
			UnsafeClass:     UnsafeUnknown,
			RegionID:        parent.RegionID,
			OwnerID:         owner,
			AliasState:      AliasInvalidatedByCall,
			Claim:           "protocol_dispatch_noalias_conservative",
			ValidatorName:   "protocol_dispatch_alias_conservative_validator",
			CostClass:       CostConservativeFallback,
			Reason:          "interface or protocol dispatch invalidates broad noalias evidence",
		})
		return err
	}
	if callbackInoutConservativeContext(parent, value) {
		_, err := graph.DeriveFact(parent.ID, Fact{
			ID:              derivedFactID(parent.ID, "callback_inout_conservative"),
			FunctionID:      parent.FunctionID,
			ValueID:         parent.ValueID,
			SiteID:          parent.SiteID,
			SourceSpan:      parent.SourceSpan,
			TypeName:        parent.TypeName,
			SourceStage:     StagePLIR,
			ProvenanceClass: ProvenanceUnsafeUnknown,
			UnsafeClass:     UnsafeUnknown,
			RegionID:        parent.RegionID,
			OwnerID:         owner,
			AliasState:      AliasInvalidatedByCall,
			Claim:           "callback_inout_conservative",
			ValidatorName:   "callback_alias_conservative_validator",
			CostClass:       CostConservativeFallback,
			Reason:          "callback or reentrant inout boundary invalidates broad noalias evidence",
		})
		return err
	}
	for _, item := range []struct {
		suffix string
		claim  string
		reason string
	}{
		{"mutable_exclusive", "mutable_exclusive", "inout parameter has exclusive mutable access for the call duration"},
		{"start_inout_exclusive", "start_inout_exclusive", "exclusive inout scope starts at function entry for the parameter"},
		{"end_inout_exclusive", "end_inout_exclusive", "exclusive inout scope ends at function return for the parameter"},
	} {
		if _, err := graph.DeriveFact(parent.ID, Fact{
			ID:              derivedFactID(parent.ID, item.suffix),
			FunctionID:      parent.FunctionID,
			ValueID:         parent.ValueID,
			SiteID:          parent.SiteID,
			SourceSpan:      parent.SourceSpan,
			TypeName:        parent.TypeName,
			SourceStage:     StagePLIR,
			ProvenanceClass: ProvenanceSafeKnown,
			UnsafeClass:     UnsafeSafe,
			RegionID:        parent.RegionID,
			OwnerID:         owner,
			AliasState:      AliasMutableExclusive,
			Claim:           item.claim,
			Reason:          item.reason,
		}); err != nil {
			return err
		}
	}
	for _, item := range []struct {
		suffix string
		claim  string
		reason string
	}{
		{"no_alias_validated_narrow_unique_local", "no_alias_validated_narrow_unique_local", "unique local value has narrow noalias evidence only for this inout interval"},
		{"no_alias_validated_narrow_sequential_inout", "no_alias_validated_narrow_sequential_inout", "sequential inout calls are valid only after the prior exclusive interval ends"},
	} {
		id, err := graph.DeriveFact(parent.ID, Fact{
			ID:              derivedFactID(parent.ID, item.suffix),
			FunctionID:      parent.FunctionID,
			ValueID:         parent.ValueID,
			SiteID:          parent.SiteID,
			SourceSpan:      parent.SourceSpan,
			TypeName:        parent.TypeName,
			SourceStage:     StagePLIR,
			ProvenanceClass: ProvenanceSafeKnown,
			UnsafeClass:     UnsafeSafe,
			RegionID:        parent.RegionID,
			OwnerID:         owner,
			AliasState:      AliasMutableExclusive,
			Claim:           item.claim,
			Reason:          item.reason,
		})
		if err != nil {
			return err
		}
		if err := graph.MarkValidated(id, "alias_interval_validator"); err != nil {
			return err
		}
	}
	return nil
}

func dynamicProtocolNoAliasRejectedContext(parent Fact, value plir.Value) bool {
	context := strings.ToLower(strings.Join([]string{
		parent.Claim,
		parent.Reason,
		parent.ValueID,
		parent.SiteID,
		value.ID,
		value.Provenance.Root,
		value.Lifetime.Owner,
	}, " "))
	return strings.Contains(context, "dynamic protocol dispatch") &&
		(strings.Contains(context, "broad noalias") || strings.Contains(context, "cannot validate"))
}

func taskGroupNoAliasConservativeContext(parent Fact, value plir.Value) bool {
	context := strings.ToLower(strings.Join([]string{
		parent.Claim,
		parent.Reason,
		parent.ValueID,
		parent.SiteID,
		value.ID,
		value.Provenance.Root,
		value.Lifetime.Owner,
	}, " "))
	return strings.Contains(context, "task group") ||
		strings.Contains(context, "task_group") ||
		strings.Contains(context, "structured concurrency")
}

func callbackInoutConservativeContext(parent Fact, value plir.Value) bool {
	context := strings.ToLower(strings.Join([]string{
		parent.Claim,
		parent.Reason,
		parent.ValueID,
		parent.SiteID,
		value.ID,
		value.Provenance.Root,
		value.Lifetime.Owner,
	}, " "))
	return strings.Contains(context, "callback") || strings.Contains(context, "reentrant")
}

func protocolDispatchNoAliasConservativeContext(parent Fact, value plir.Value) bool {
	context := strings.ToLower(strings.Join([]string{
		parent.Claim,
		parent.Reason,
		parent.ValueID,
		parent.SiteID,
		value.ID,
		value.Provenance.Root,
		value.Lifetime.Owner,
	}, " "))
	return strings.Contains(context, "protocol dispatch") ||
		strings.Contains(context, "protocol_dispatch") ||
		strings.Contains(context, "interface dispatch") ||
		strings.Contains(context, "interface/protocol dispatch") ||
		strings.Contains(context, "dynamic dispatch")
}

func boundaryNoAliasConservativeContext(parent Fact, value plir.Value) bool {
	context := strings.ToLower(strings.Join([]string{
		parent.Claim,
		parent.Reason,
		parent.ValueID,
		parent.SiteID,
		value.ID,
		value.Provenance.Root,
		value.Lifetime.Owner,
	}, " "))
	return strings.Contains(context, "task/actor boundary") ||
		strings.Contains(context, "task actor boundary") ||
		strings.Contains(context, "task_boundary") ||
		strings.Contains(context, "task boundary") ||
		strings.Contains(context, "actor_boundary") ||
		strings.Contains(context, "actor boundary") ||
		strings.Contains(context, "task_spawn") ||
		strings.Contains(context, "send_typed")
}

func addCopyIntoFacts(graph *Graph, functionID string, op plir.Operation, values map[string]plir.Value) error {
	parentID, err := graph.AddFact(copyIntoOperationFact(functionID, op, values))
	if err != nil {
		return err
	}
	parent, ok := graph.Fact(parentID)
	if !ok {
		return fmt.Errorf("memoryfacts: copy_into operation fact %q was not recorded", parentID)
	}
	if strings.Contains(op.Note, "dest_capacity_check:normal_build") {
		if err := addCopyIntoDestinationLengthCheckFact(graph, parent, op, values); err != nil {
			return err
		}
	}
	switch copyIntoOverlapStatusFromNote(op.Note) {
	case "distinct_roots", "known_disjoint":
		_, err = graph.DeriveFact(parent.ID, copyIntoDestinationFact(parent, op, values))
	case "known_overlap":
		err = addCopyIntoOverlapRejectedFact(graph, parent, op)
	default:
		err = addCopyIntoOverlapConservativeFact(graph, parent, op)
	}
	return err
}

func copyIntoOperationFact(functionID string, op plir.Operation, values map[string]plir.Value) Fact {
	source := ownerFromOperationInput(op, 0)
	destination := ownerFromOperationInput(op, 1)
	destinationValueID := valueIDForPath(destination, values)
	value := values[destinationValueID]
	return Fact{
		ID:              FactID(fmt.Sprintf("plir:%s:%s:copy_into_operation", functionID, op.ID)),
		FunctionID:      functionID,
		ValueID:         nonEmpty(destinationValueID, destination),
		SiteID:          nonEmpty(op.Source, fmt.Sprintf("%s:%s", functionID, op.ID)),
		SourceSpan:      op.Source,
		TypeName:        value.Type,
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceSafeKnown,
		UnsafeClass:     UnsafeSafe,
		OwnerID:         destination,
		EscapeState:     EscapeNoEscape,
		Claim:           "copy_into_operation",
		CostClass:       CostInstrumentationOnly,
		Reason:          fmt.Sprintf("copy_into operation from %q into %q records destination capacity and overlap contract: %s", source, destination, op.Note),
	}
}

func addCopyIntoDestinationLengthCheckFact(graph *Graph, parent Fact, op plir.Operation, values map[string]plir.Value) error {
	destination := ownerFromOperationInput(op, 1)
	destinationValueID := valueIDForPath(destination, values)
	value := values[destinationValueID]
	_, err := graph.DeriveFact(parent.ID, Fact{
		ID:               derivedFactID(parent.ID, "copy_into_destination_length_check"),
		FunctionID:       parent.FunctionID,
		ValueID:          nonEmpty(destinationValueID, destination),
		SiteID:           parent.SiteID,
		SourceSpan:       parent.SourceSpan,
		TypeName:         nonEmpty(value.Type, parent.TypeName),
		SourceStage:      StagePLIR,
		ProvenanceClass:  ProvenanceSafeKnown,
		UnsafeClass:      UnsafeSafe,
		OwnerID:          destination,
		EscapeState:      parent.EscapeState,
		Claim:            "copy_into_destination_length_check",
		ValidationState:  ValidationPass,
		ValidatorName:    "copy_into_destination_capacity_validator",
		CostClass:        CostDynamicCheckRequired,
		NormalBuildCheck: true,
		Reason:           fmt.Sprintf("copy_into destination %q is sliced to source length in normal builds before the store loop", destination),
	})
	return err
}

func copyIntoDestinationFact(parent Fact, op plir.Operation, values map[string]plir.Value) Fact {
	destination := ownerFromOperationInput(op, 1)
	valueID := valueIDForPath(destination, values)
	value := values[valueID]
	return Fact{
		ID:              derivedFactID(parent.ID, "copy_into_destination_fact_id"),
		FunctionID:      parent.FunctionID,
		ValueID:         nonEmpty(valueID, destination),
		SiteID:          parent.SiteID,
		SourceSpan:      parent.SourceSpan,
		TypeName:        nonEmpty(value.Type, parent.TypeName),
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceSafeOwned,
		UnsafeClass:     UnsafeSafe,
		OwnerID:         destination,
		EscapeState:     EscapeNoEscape,
		Claim:           "copy_into_destination_fact_id",
		Reason:          fmt.Sprintf("copy_into writes into caller-owned destination %q only after parented length check and overlap status %q", destination, copyIntoOverlapStatusFromNote(op.Note)),
	}
}

func addCopyIntoOverlapRejectedFact(graph *Graph, parent Fact, op plir.Operation) error {
	_, err := graph.DeriveFact(parent.ID, Fact{
		ID:              derivedFactID(parent.ID, "copy_into_overlap_rejected"),
		FunctionID:      parent.FunctionID,
		ValueID:         parent.ValueID,
		SiteID:          parent.SiteID,
		SourceSpan:      parent.SourceSpan,
		TypeName:        parent.TypeName,
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceSafeKnown,
		UnsafeClass:     UnsafeSafe,
		OwnerID:         parent.OwnerID,
		EscapeState:     parent.EscapeState,
		AliasState:      AliasMaybe,
		Claim:           "copy_into_overlap_rejected",
		ValidationState: ValidationFail,
		ValidatorName:   "copy_into_overlap_validator",
		CostClass:       CostUnsupportedRejected,
		Reason:          fmt.Sprintf("copy_into overlap status %q rejects zero-cost/noalias destination claim: %s", copyIntoOverlapStatusFromNote(op.Note), op.Note),
	})
	return err
}

func addCopyIntoOverlapConservativeFact(graph *Graph, parent Fact, op plir.Operation) error {
	_, err := graph.DeriveFact(parent.ID, Fact{
		ID:              derivedFactID(parent.ID, "copy_into_overlap_conservative"),
		FunctionID:      parent.FunctionID,
		ValueID:         parent.ValueID,
		SiteID:          parent.SiteID,
		SourceSpan:      parent.SourceSpan,
		TypeName:        parent.TypeName,
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceUnsafeUnknown,
		UnsafeClass:     UnsafeUnknown,
		OwnerID:         parent.OwnerID,
		EscapeState:     EscapeConservative,
		AliasState:      AliasUnknownConservative,
		Claim:           "copy_into_overlap_conservative",
		CostClass:       CostConservativeFallback,
		Reason:          fmt.Sprintf("copy_into overlap status %q remains conservative and cannot authorize zero-cost/noalias destination claim: %s", copyIntoOverlapStatusFromNote(op.Note), op.Note),
	})
	return err
}

func copyIntoOverlapStatusFromNote(note string) string {
	note = strings.TrimSpace(note)
	for _, field := range strings.Fields(note) {
		if strings.HasPrefix(field, "overlap:") {
			return strings.TrimSpace(strings.TrimPrefix(field, "overlap:"))
		}
	}
	return "unknown_conservative"
}

func addSliceViewBoundsCheckFact(graph *Graph, parent Fact, value plir.Value, op plir.Operation) error {
	if isUnsafeUnknown(parent) || op.Kind != plir.OpSliceWindow {
		return nil
	}
	note := strings.TrimSpace(op.Note)
	if !strings.Contains(note, "bounds_check:normal_build") {
		return nil
	}
	_, err := graph.DeriveFact(parent.ID, Fact{
		ID:               derivedFactID(parent.ID, "bounds_check_retained_dynamic"),
		FunctionID:       parent.FunctionID,
		ValueID:          parent.ValueID,
		SiteID:           parent.SiteID,
		SourceSpan:       parent.SourceSpan,
		TypeName:         parent.TypeName,
		SourceStage:      StagePLIR,
		ProvenanceClass:  ProvenanceSafeBorrowed,
		UnsafeClass:      UnsafeSafe,
		RegionID:         parent.RegionID,
		OwnerID:          parent.OwnerID,
		BorrowState:      parent.BorrowState,
		EscapeState:      parent.EscapeState,
		Claim:            "bounds_check_retained_dynamic",
		ValidationState:  ValidationPass,
		ValidatorName:    "safe_view_bounds_validator",
		CostClass:        CostDynamicCheckRequired,
		NormalBuildCheck: true,
		Reason:           fmt.Sprintf("safe view constructor retains dynamic bounds/length check in normal-build: %s", nonEmpty(note, value.ID)),
	})
	return err
}

func provenanceClassForPLIRFact(fact plir.Fact, value plir.Value) ProvenanceClass {
	if value.Alloc != nil && value.Alloc.Builtin == "core.alloc_bytes" {
		return ProvenanceUnsafeVerifiedRoot
	}
	switch fact.Kind {
	case plir.FactProvenanceUnknown:
		return ProvenanceUnsafeUnknown
	case plir.FactBorrowedImm, plir.FactBorrowedMut:
		if value.Provenance.Kind == plir.ProvenanceExternal || value.Provenance.Kind == plir.ProvenanceUnknown {
			return ProvenanceUnsafeUnknown
		}
		return ProvenanceSafeBorrowed
	case plir.FactOwned:
		return ProvenanceSafeOwned
	default:
		if value.Provenance.Kind == plir.ProvenanceExternal || value.Provenance.Kind == plir.ProvenanceUnknown {
			return ProvenanceUnsafeUnknown
		}
		return ProvenanceSafeKnown
	}
}

func unsafeClassForPLIRFact(fact plir.Fact, value plir.Value) UnsafeClass {
	if value.Alloc != nil && value.Alloc.Builtin == "core.alloc_bytes" {
		return UnsafeVerifiedRoot
	}
	switch fact.Kind {
	case plir.FactProvenanceUnknown:
		return UnsafeUnknown
	default:
		if value.Provenance.Kind == plir.ProvenanceExternal || value.Provenance.Kind == plir.ProvenanceUnknown {
			return UnsafeUnknown
		}
		return UnsafeSafe
	}
}

func borrowStateForPLIRFact(fact plir.Fact) BorrowState {
	switch fact.Kind {
	case plir.FactBorrowedImm:
		return BorrowImmutable
	case plir.FactBorrowedMut:
		return BorrowMutable
	case plir.FactMoved:
		return BorrowMoved
	default:
		return BorrowNone
	}
}

func aliasStateForPLIRFact(fact plir.Fact) AliasState {
	switch fact.Kind {
	case plir.FactNoAlias:
		return AliasMutableExclusive
	default:
		return AliasUnknown
	}
}

func escapeStateForPLIRFact(fact plir.Fact, value plir.Value) EscapeState {
	if fact.Kind == plir.FactNoEscape {
		return EscapeNoEscape
	}
	switch value.Escape {
	case plir.EscapeNoEscape:
		return EscapeNoEscape
	case plir.EscapeReturn:
		return EscapeReturn
	case plir.EscapeGlobal:
		return EscapeGlobal
	case plir.EscapeActor:
		return EscapeActor
	case plir.EscapeTask:
		return EscapeTask
	case plir.EscapeUnsafe:
		return EscapeUnsafe
	case plir.EscapeConservative:
		return EscapeConservative
	default:
		return EscapeUnknown
	}
}

func allocationSiteIDForPLIRValue(value plir.Value) string {
	if value.Alloc == nil {
		return ""
	}
	return nonEmpty(value.Alloc.Builtin, value.Provenance.Root, value.ID)
}

func ownerForPLIRValue(value plir.Value) string {
	return normalizeOwnerID(nonEmpty(value.Provenance.Root, value.Lifetime.Owner))
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

func ownerFromOperationInput(op plir.Operation, index int) string {
	if index < 0 || index >= len(op.Inputs) {
		return ""
	}
	return normalizeOwnerID(op.Inputs[index])
}

func normalizeOwnerID(owner string) string {
	owner = strings.TrimSpace(owner)
	for strings.HasPrefix(owner, "derived:") {
		owner = strings.TrimPrefix(owner, "derived:")
	}
	owner = strings.TrimPrefix(owner, "param:")
	owner = strings.TrimPrefix(owner, "local:")
	owner = strings.TrimPrefix(owner, "view:")
	owner = strings.TrimPrefix(owner, "alloc_intent:")
	if dot := strings.Index(owner, "."); dot > 0 {
		owner = owner[:dot]
	}
	return owner
}

func isCopyAllocationValue(value plir.Value) bool {
	return value.Kind == plir.ValueAllocIntent && value.Alloc != nil && copyAllocationBuiltin(value.Alloc.Builtin)
}

func copyAllocationBuiltin(name string) bool {
	return name == "core.string_copy" || (strings.HasPrefix(name, "core.slice_copy_") && !strings.HasPrefix(name, "core.slice_copy_into_"))
}

func isCopyIntoOperation(op plir.Operation) bool {
	return op.Kind == plir.OpCall && strings.Contains(op.Note, "copy_into")
}

func sourceFactIDForPath(path string, factIDs map[plirFactKey]FactID, values map[string]plir.Value) FactID {
	for _, valueID := range candidateValueIDs(path, values) {
		for _, kind := range []plir.FactKind{plir.FactBorrowedImm, plir.FactBorrowedMut, plir.FactProvenanceKnown, plir.FactOwned, plir.FactNoEscape} {
			if id := factIDs[plirFactKey{kind: kind, valueID: valueID}]; id != "" {
				return id
			}
		}
	}
	return ""
}

func valueIDForPath(path string, values map[string]plir.Value) string {
	for _, valueID := range candidateValueIDs(path, values) {
		if _, ok := values[valueID]; ok {
			return valueID
		}
	}
	return ""
}

func candidateValueIDs(path string, values map[string]plir.Value) []string {
	owner := normalizeOwnerID(path)
	if owner == "" {
		return nil
	}
	candidates := []string{owner}
	for _, prefix := range []string{"view:", "alloc_intent:", "local:", "param:"} {
		candidates = append(candidates, prefix+owner)
	}
	if path != owner {
		candidates = append(candidates, path)
		for _, prefix := range []string{"view:", "alloc_intent:", "local:", "param:"} {
			candidates = append(candidates, prefix+path)
		}
	}
	out := candidates[:0]
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if candidate == "" || seen[candidate] {
			continue
		}
		if len(values) > 0 {
			if _, ok := values[candidate]; !ok && !strings.Contains(candidate, ":") {
				continue
			}
		}
		seen[candidate] = true
		out = append(out, candidate)
	}
	return out
}

func derivedFactID(parentID FactID, suffix string) FactID {
	return FactID(fmt.Sprintf("%s:%s", parentID, suffix))
}

func ffiDerivedFactID(parentID FactID, opID string, suffix string) FactID {
	return FactID(fmt.Sprintf("%s:%s:%s", parentID, safeFactIDPart(opID), suffix))
}

func aliasBoundaryDerivedFactID(parentID FactID, opID string, suffix string) FactID {
	return FactID(fmt.Sprintf("%s:%s:%s", parentID, safeFactIDPart(opID), suffix))
}

func claimForPLIRFact(fact plir.Fact) string {
	if fact.Kind == plir.FactProvenanceUnknown && strings.Contains(strings.ToLower(fact.Reason), "raw slice") {
		return "external_unknown"
	}
	return fact.Kind.String()
}

func isSafeWrapperPromotionOperation(op plir.Operation) bool {
	if op.Kind != plir.OpUnsafe {
		return false
	}
	note := strings.ToLower(strings.TrimSpace(op.Note))
	return strings.Contains(note, "safe wrapper") && (strings.Contains(note, "external") || strings.Contains(note, "unsafe_unknown") || strings.Contains(note, "raw"))
}

func unsafeOperationClaim(op plir.Operation) (string, ProvenanceClass, UnsafeClass, bool) {
	note := strings.ToLower(op.Note)
	switch {
	case unsafeStaticContractNote(note):
		return "unsafe_contract_static_untrusted", ProvenanceUnsafeUnknown, UnsafeUnknown, true
	case unsafeRuntimeCheckableContractNote(note):
		return "unsafe_contract_runtime_checkable", ProvenanceUnsafeChecked, UnsafeChecked, true
	case strings.Contains(note, "rejected_negative_offset"):
		return "rejected_negative_offset", ProvenanceUnsafeChecked, UnsafeChecked, true
	case strings.Contains(note, "rejected_upper_bound"):
		return "rejected_upper_bound", ProvenanceUnsafeChecked, UnsafeChecked, true
	case strings.Contains(note, "rejected_access_width_overflow"):
		return "rejected_access_width_overflow", ProvenanceUnsafeChecked, UnsafeChecked, true
	case strings.Contains(note, "rejected_negative_length"):
		return "rejected_negative_length", ProvenanceUnsafeChecked, UnsafeChecked, true
	case strings.Contains(note, "rejected_length_overflow"):
		return "rejected_length_overflow", ProvenanceUnsafeChecked, UnsafeChecked, true
	case strings.Contains(note, "raw_slice_bounds") && strings.Contains(note, "verified_allocation_root"):
		return "raw_slice_verified_allocation_root", ProvenanceUnsafeChecked, UnsafeChecked, true
	case strings.Contains(note, "raw memory gateway"):
		if op.UnsafeClass == plir.UnsafeChecked {
			return "raw_memory_access_checked", ProvenanceUnsafeChecked, UnsafeChecked, true
		}
		return "raw_memory_access_unknown", ProvenanceUnsafeUnknown, UnsafeUnknown, true
	case strings.Contains(note, "derived_allocation_offset"):
		return "derived_allocation_offset", ProvenanceUnsafeChecked, UnsafeChecked, true
	case strings.Contains(note, "checked_external_unknown"):
		return "checked_external_unknown", ProvenanceUnsafeUnknown, UnsafeUnknown, true
	case strings.Contains(note, "external-provenance view"):
		return "external_unknown", ProvenanceUnsafeUnknown, UnsafeUnknown, true
	default:
		return "", "", "", false
	}
}

func unsafeRuntimeCheckableContractNote(note string) bool {
	if !strings.Contains(note, "unsafe contract") || !strings.Contains(note, "runtime_checkable") {
		return false
	}
	return strings.Contains(note, "nonnull") ||
		strings.Contains(note, "non_null") ||
		strings.Contains(note, "alignment") ||
		strings.Contains(note, "aligned") ||
		strings.Contains(note, "length") ||
		strings.Contains(note, "bounds")
}

func unsafeStaticContractNote(note string) bool {
	if !strings.Contains(note, "unsafe contract") {
		return false
	}
	return strings.Contains(note, "static_untrusted") ||
		strings.Contains(note, "noalias") ||
		strings.Contains(note, "no_alias") ||
		strings.Contains(note, "lifetime") ||
		strings.Contains(note, "region")
}

func finalizeUnsafeOperationFact(graph *Graph, id FactID, claim string, provenance ProvenanceClass, unsafeClass UnsafeClass) error {
	switch claim {
	case "unsafe_contract_runtime_checkable":
		return graph.MarkValidated(id, "unsafe_runtime_contract_validator")
	}
	if rawBoundsRuntimeCheckClaim(claim) && provenance == ProvenanceUnsafeChecked && unsafeClass == UnsafeChecked {
		parent, ok := graph.Fact(id)
		if !ok {
			return fmt.Errorf("memoryfacts: unsafe operation fact %q was not recorded", id)
		}
		if err := addRawBoundsRuntimeCheckFact(graph, parent); err != nil {
			return err
		}
	}
	if provenance == ProvenanceUnsafeUnknown || unsafeClass == UnsafeUnknown {
		parent, ok := graph.Fact(id)
		if !ok {
			return fmt.Errorf("memoryfacts: unsafe operation fact %q was not recorded", id)
		}
		if claim == "unsafe_unknown_rejected_safe_facts" {
			return nil
		}
		return addUnsafeUnknownRejectedSafeFacts(graph, parent)
	}
	return nil
}

func rawBoundsRuntimeCheckClaim(claim string) bool {
	switch strings.ToLower(strings.TrimSpace(claim)) {
	case "raw_memory_access_checked", "rejected_access_width_overflow", "rejected_length_overflow":
		return true
	default:
		return false
	}
}

func addRawBoundsRuntimeCheckFact(graph *Graph, parent Fact) error {
	_, err := graph.DeriveFact(parent.ID, Fact{
		ID:               derivedFactID(parent.ID, "raw_bounds_runtime_check_normal_build"),
		FunctionID:       parent.FunctionID,
		ValueID:          parent.ValueID,
		SiteID:           parent.SiteID,
		SourceSpan:       parent.SourceSpan,
		TypeName:         parent.TypeName,
		SourceStage:      parent.SourceStage,
		ProvenanceClass:  ProvenanceUnsafeChecked,
		UnsafeClass:      UnsafeChecked,
		EscapeState:      parent.EscapeState,
		Claim:            "raw_bounds_runtime_check_normal_build",
		ValidationState:  ValidationPass,
		ValidatorName:    "raw_bounds_width_validator",
		CostClass:        CostDynamicCheckRequired,
		NormalBuildCheck: true,
		Reason:           "Memory Ideal v6 keeps raw bounds width/overflow uncertainty as a normal-build check or trap",
	})
	return err
}

func addUnsafeUnknownRejectedSafeFacts(graph *Graph, parent Fact) error {
	_, err := graph.DeriveFact(parent.ID, Fact{
		ID:              derivedFactID(parent.ID, "unsafe_unknown_rejected_safe_facts"),
		FunctionID:      parent.FunctionID,
		ValueID:         parent.ValueID,
		SiteID:          parent.SiteID,
		SourceSpan:      parent.SourceSpan,
		TypeName:        parent.TypeName,
		SourceStage:     parent.SourceStage,
		ProvenanceClass: ProvenanceUnsafeUnknown,
		UnsafeClass:     UnsafeUnknown,
		EscapeState:     EscapeConservative,
		Claim:           "unsafe_unknown_rejected_safe_facts",
		ValidationState: ValidationFail,
		ValidatorName:   "unsafe_unknown_fact_validator",
		CostClass:       CostUnsupportedRejected,
		Reason:          "Memory Ideal v5 rejects unsafe_unknown promotion to safe_known, provenance_known, or noalias facts",
	})
	return err
}

func costClassForUnsafeOperationClaim(claim string, provenance ProvenanceClass, unsafeClass UnsafeClass) CostClass {
	if provenance == ProvenanceUnsafeUnknown || unsafeClass == UnsafeUnknown {
		return CostConservativeFallback
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(claim)), "rejected_") {
		return CostUnsupportedRejected
	}
	switch claim {
	case "derived_allocation_offset", "raw_memory_access_checked", "raw_slice_verified_allocation_root", "unsafe_contract_runtime_checkable":
		return CostDynamicCheckRequired
	default:
		return CostInstrumentationOnly
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

func nonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
