package fromplir

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/plir"
)

func addFunctionSummaryFacts(
	graph *Graph,
	fn plir.Function,
	values map[string]plir.Value,
	factIDs map[plirFactKey]FactID,
) error {
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

func addReturnSummaryFacts(
	graph *Graph,
	fn plir.Function,
	values map[string]plir.Value,
	factIDs map[plirFactKey]FactID,
) error {
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
				if _, err := graph.AddFact(
					functionSummaryFact(fn.Name, anchor, "returns_unknown_unsafe", op.Source, value, ProvenanceUnsafeUnknown, UnsafeUnknown, EscapeConservative, AliasUnknown, "", ("returned value has " +
						"unknown or external unsafe provenance")),
				); err != nil {
					return err
				}
				continue
			}
			if returnValueIsOwnedAllocation(value, factIDs) {
				if _, err := graph.AddFact(
					functionSummaryFact(fn.Name, anchor, "returns_owned_new_allocation", op.Source, value, ProvenanceSafeOwned, UnsafeSafe, EscapeReturn, AliasUnknown, ownerForPLIRValue(value), ("return value owns fresh " +
						"compiler-visible allocation provenance")),
				); err != nil {
					return err
				}
			}
			if source, ok := returnedBorrowSource(fn, value, factIDs); ok {
				reason := fmt.Sprintf("return value borrows from parameter %q", source.Owner)
				if source.ParamPath != "" {
					reason = fmt.Sprintf(
						"return value borrows from parameter path %q",
						source.Owner+"."+source.ParamPath,
					)
				}
				if index, indexOK := source.ParamIndexValue(); indexOK {
					reason = fmt.Sprintf(
						"return value borrows from parameter #%d %q",
						index,
						source.Owner,
					)
					if source.ParamPath != "" {
						reason = fmt.Sprintf(
							"return value borrows from parameter #%d %q path %q",
							index,
							source.Owner,
							source.ParamPath,
						)
					}
				}
				fact := functionSummaryFact(
					fn.Name,
					anchor,
					"returns_borrow_from_param",
					op.Source,
					value,
					ProvenanceSafeBorrowed,
					UnsafeSafe,
					EscapeReturn,
					AliasUnknown,
					source.Owner,
					reason,
				)
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
		reason := fmt.Sprintf(
			"return%s region provenance is parameter #%d",
			formatSummaryLeaf(leaf),
			paramIndex,
		)
		if owner != "" {
			reason = fmt.Sprintf("%s (%s)", reason, owner)
		}
		fact := functionSummaryFact(
			fn.Name,
			"return_region:"+leaf,
			"may_return_region",
			summarySite(fn),
			plir.Value{},
			ProvenanceSafeBorrowed,
			UnsafeSafe,
			EscapeReturn,
			AliasUnknown,
			owner,
			reason,
		)
		fact.ParamIndex = paramIndexPtr(paramIndex)
		fact.ParamPath = leaf
		if _, err := graph.AddFact(fact); err != nil {
			return err
		}
	}
	if summary.ReturnRegionUnknown {
		if _, err := graph.AddFact(
			functionSummaryFact(fn.Name, "return_region_unknown", "may_return_region", summarySite(fn), plir.Value{}, ProvenanceUnsafeUnknown, UnsafeUnknown, EscapeConservative, AliasUnknown, "", ("return region summary " +
				"is unknown and remains conservative")),
		); err != nil {
			return err
		}
	}
	for leaf, provenances := range summary.ReturnResourceSummary {
		for _, provenance := range provenances {
			owner := summaryParamOwner(summary, provenance.ParamIndex)
			reason := fmt.Sprintf(
				"returned resource%s provenance is parameter #%d%s",
				formatSummaryLeaf(leaf),
				provenance.ParamIndex,
				formatSummaryLeaf(provenance.ParamPath),
			)
			if owner != "" {
				reason = fmt.Sprintf("%s (%s)", reason, owner)
			}
			fact := functionSummaryFact(
				fn.Name,
				fmt.Sprintf(
					"return_resource:%s:%d:%s",
					leaf,
					provenance.ParamIndex,
					provenance.ParamPath,
				),
				"may_return_resource",
				summarySite(fn),
				plir.Value{},
				ProvenanceSafeKnown,
				UnsafeSafe,
				EscapeReturn,
				AliasUnknown,
				owner,
				reason,
			)
			fact.ParamIndex = paramIndexPtr(provenance.ParamIndex)
			fact.ParamPath = provenance.ParamPath
			if _, err := graph.AddFact(fact); err != nil {
				return err
			}
		}
	}
	if summary.ReturnResourceUnknown {
		if _, err := graph.AddFact(
			functionSummaryFact(fn.Name, "return_resource_unknown", "returns_unknown_unsafe", summarySite(fn), plir.Value{}, ProvenanceUnsafeUnknown, UnsafeUnknown, EscapeConservative, AliasUnknown, "", ("returned resource " +
				"provenance is unknown and remains conservative")),
		); err != nil {
			return err
		}
	}
	for leaf, provenances := range summary.ThrowResourceSummary {
		for _, provenance := range provenances {
			owner := summaryParamOwner(summary, provenance.ParamIndex)
			reason := fmt.Sprintf(
				"thrown resource%s provenance is parameter #%d%s",
				formatSummaryLeaf(leaf),
				provenance.ParamIndex,
				formatSummaryLeaf(provenance.ParamPath),
			)
			if owner != "" {
				reason = fmt.Sprintf("%s (%s)", reason, owner)
			}
			fact := functionSummaryFact(
				fn.Name,
				fmt.Sprintf(
					"throw_resource:%s:%d:%s",
					leaf,
					provenance.ParamIndex,
					provenance.ParamPath,
				),
				"may_throw_resource",
				summarySite(fn),
				plir.Value{},
				ProvenanceSafeKnown,
				UnsafeSafe,
				EscapeReturn,
				AliasUnknown,
				owner,
				reason,
			)
			fact.ParamIndex = paramIndexPtr(provenance.ParamIndex)
			fact.ParamPath = provenance.ParamPath
			if _, err := graph.AddFact(fact); err != nil {
				return err
			}
		}
	}
	if len(summary.Effects) > 0 {
		if _, err := graph.AddFact(
			functionSummaryFact(fn.Name, "effects", "requires_effects", summarySite(fn), plir.Value{}, ProvenanceSafeKnown, UnsafeSafe, EscapeUnknown, AliasUnknown, "", "function declares effects: "+strings.Join(summary.Effects, ", ")),
		); err != nil {
			return err
		}
	}
	if summaryRequiresCapabilities(summary) {
		if _, err := graph.AddFact(
			functionSummaryFact(fn.Name, "capabilities", "requires_capabilities", summarySite(fn), plir.Value{}, ProvenanceSafeKnown, UnsafeSafe, EscapeUnknown, AliasUnknown, "", ("function effects "+
				"require capability-gated operations: ")+strings.Join(summary.Effects, ", ")),
		); err != nil {
			return err
		}
	}
	if summaryRequiresCapMemAuthorization(summary) {
		if _, err := graph.AddFact(
			functionSummaryFact(fn.Name, "cap_mem_authorization", "cap_mem_authorization_only", summarySite(fn), plir.Value{}, ProvenanceUnsafeChecked, UnsafeChecked, EscapeUnknown, AliasUnknown, "", ("cap.mem authorizes raw " +
				"operations only; it does not prove pointer validity, bounds, ownership, " +
				"noalias, or safe provenance")),
		); err != nil {
			return err
		}
	}
	if summary.TouchesMutableGlobals {
		if _, err := graph.AddFact(
			functionSummaryFact(fn.Name, "touches_mutable_globals", "may_store_global", summarySite(fn), plir.Value{}, ProvenanceSafeKnown, UnsafeSafe, EscapeGlobal, AliasUnknown, "", ("semantics summary " +
				"records mutable global access")),
		); err != nil {
			return err
		}
	}
	return nil
}

func addOperationSummaryFacts(
	graph *Graph,
	fn plir.Function,
	values map[string]plir.Value,
	factIDs map[plirFactKey]FactID,
) error {
	for _, op := range fn.Ops {
		switch op.Kind {
		case plir.OpGlobalStore:
			owner := ownerFromOperationInput(op, 0)
			if _, err := graph.AddFact(
				functionSummaryFact(fn.Name, op.ID, "may_store_global", op.Source, plir.Value{}, ProvenanceSafeKnown, UnsafeSafe, EscapeGlobal, AliasUnknown, owner, "operation stores into global state"),
			); err != nil {
				return err
			}
		case plir.OpActorSend:
			owner := ownerFromOperationInput(op, 0)
			fact := functionSummaryFact(fn.Name, op.ID, "may_escape_to_actor", op.Source, plir.Value{}, ProvenanceSafeKnown, UnsafeSafe, EscapeActor, AliasUnknown, owner, ("operation transfers " +
				"payload across actor boundary"))
			fact.DomainKind = DomainActor
			fact.DomainOwnerID = owner
			fact.DestinationActive = owner != ""
			if _, err := graph.AddFact(fact); err != nil {
				return err
			}
		case plir.OpClosure:
			owner := strings.Join(op.Inputs, ",")
			if _, err := graph.AddFact(
				functionSummaryFact(fn.Name, op.ID, "may_capture_in_closure", op.Source, plir.Value{}, ProvenanceSafeKnown, UnsafeSafe, EscapeConservative, AliasUnknown, owner, ("closure captures " +
					"visible environment values")),
			); err != nil {
				return err
			}
		case plir.OpCall:
			if isTaskEscapeOperation(op) {
				owner := ownerFromTypedTaskOperationInput(op)
				fact := functionSummaryFact(fn.Name, op.ID, "may_escape_to_task", op.Source, plir.Value{}, ProvenanceSafeKnown, UnsafeSafe, EscapeTask, AliasUnknown, owner, ("operation may transfer " +
					"work or handles across task boundary"))
				fact.DomainKind = DomainTask
				fact.DomainOwnerID = owner
				fact.DestinationActive = owner != ""
				if _, err := graph.AddFact(fact); err != nil {
					return err
				}
			}
			if err := addNoAliasCallBoundaryFactsForOperation(
				graph,
				fn.Name,
				op,
				values,
				factIDs,
			); err != nil {
				return err
			}
			if isUnknownExternalCallOperation(op) {
				if _, err := graph.AddFact(
					functionSummaryFact(fn.Name, op.ID, "unknown_external_call_conservative", op.Source, plir.Value{}, ProvenanceUnsafeUnknown, UnsafeUnknown, EscapeConservative, AliasUnknown, strings.Join(op.Inputs, ","), ("callee summary is " +
						"unknown, so memory/resource effects remain conservative")),
				); err != nil {
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

func addNoAliasCallBoundaryFactsForOperation(
	graph *Graph,
	functionID string,
	op plir.Operation,
	values map[string]plir.Value,
	factIDs map[plirFactKey]FactID,
) error {
	note := strings.ToLower(op.Note)
	callbackBoundary := strings.Contains(note, "alias_boundary:function_typed_inout")
	unknownExternalBoundary := strings.Contains(note, "alias_boundary:unknown_external_call") ||
		isUnknownExternalCallOperation(op)
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
				if err := addCallbackNoAliasInvalidationFact(
					graph,
					parentID,
					functionID,
					op,
					value,
				); err != nil {
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

func addFFIExternalFactsForOperation(
	graph *Graph,
	functionID string,
	op plir.Operation,
	values map[string]plir.Value,
	factIDs map[plirFactKey]FactID,
) error {
	for _, input := range op.Inputs {
		value, ok := plirValueForPath(input, values)
		if !ok {
			continue
		}
		if parentID := factIDs[plirFactKey{
			kind:    plir.FactProvenanceUnknown,
			valueID: value.ID,
		}]; parentID != "" &&
			value.Type == "ptr" {
			externalID, err := addFFIExternalPointerUnknownFact(
				graph,
				parentID,
				functionID,
				op,
				value,
			)
			if err != nil {
				return err
			}
			if err := addExternalPointerProvenanceRejectedFact(
				graph,
				externalID,
				functionID,
				op,
				value,
			); err != nil {
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

func addFFIExternalPointerUnknownFact(
	graph *Graph,
	parentID FactID,
	functionID string,
	op plir.Operation,
	value plir.Value,
) (FactID, error) {
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
		Reason: ("Memory Ideal v7 keeps external pointer provenance unsafe_" +
			"unknown at FFI boundary"),
	})
}

func addExternalPointerProvenanceRejectedFact(
	graph *Graph,
	parentID FactID,
	functionID string,
	op plir.Operation,
	value plir.Value,
) error {
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
		Reason: ("Memory Ideal v7 rejects provenance_known promotion from " +
			"external pointer without compiler-owned proof"),
	})
	if err != nil {
		return err
	}
	return graph.InvalidateFact(
		id,
		"external pointer cannot become provenance_known without compiler-owned proof",
	)
}

func addFFICallMayRetainBorrowFact(
	graph *Graph,
	parentID FactID,
	functionID string,
	op plir.Operation,
	value plir.Value,
) error {
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
		Reason: ("Memory Ideal v7 keeps borrowed pointer passed to FFI " +
			"conservative because external call may retain it"),
	})
	return err
}

func addCallbackNoAliasInvalidationFact(
	graph *Graph,
	parentID FactID,
	functionID string,
	op plir.Operation,
	value plir.Value,
) error {
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

func addFFINoAliasInvalidationFact(
	graph *Graph,
	parentID FactID,
	functionID string,
	op plir.Operation,
	value plir.Value,
) error {
	_, err := graph.DeriveFact(parentID, Fact{
		ID: ffiDerivedFactID(
			parentID,
			op.ID,
			"ffi_noalias_invalidated_by_external_call",
		),
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

func addSafeWrapperPromotionRejectionFact(
	graph *Graph,
	functionID string,
	op plir.Operation,
	values map[string]plir.Value,
	factIDs map[plirFactKey]FactID,
) error {
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
			ID: ffiDerivedFactID(
				parentID,
				op.ID,
				"safe_wrapper_promotion_rejected_without_contract",
			),
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
			Reason: ("Memory Ideal v7 rejects safe wrapper promotion from external " +
				"pointer without compiler-owned contract"),
		})
		if err != nil {
			return err
		}
		return graph.InvalidateFact(
			id,
			"safe wrapper promotion from external pointer requires compiler-owned contract",
		)
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

func addFactKindSummaryFacts(
	graph *Graph,
	fn plir.Function,
	values map[string]plir.Value,
	factIDs map[plirFactKey]FactID,
) error {
	for _, value := range values {
		if value.Kind == plir.ValueParam && value.Borrow == plir.BorrowMove {
			provenance, unsafeClass := summaryClassesForPLIRValue(
				value,
				factIDs,
				ProvenanceSafeOwned,
			)
			if _, err := graph.AddFact(
				functionSummaryFact(fn.Name, "consume:"+value.ID, "may_consume_param", nonEmpty(value.Source, summarySite(fn)), value, provenance, unsafeClass, escapeStateForPLIRValue(value), AliasUnknown, ownerForPLIRValue(value), ("consume parameter may " +
					"be moved by this function")),
			); err != nil {
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
				provenance, unsafeClass := summaryClassesForPLIRValue(
					value,
					factIDs,
					ProvenanceSafeOwned,
				)
				if _, err := graph.AddFact(
					functionSummaryFact(fn.Name, pf.ID, "may_consume_param", pf.Source, value, provenance, unsafeClass, escapeStateForPLIRValue(value), AliasUnknown, ownerForPLIRValue(value), ("parameter may be " +
						"consumed or moved by this function")),
				); err != nil {
					return err
				}
			}
		case plir.FactBorrowedMut, plir.FactNoAlias:
			if value.Kind == plir.ValueParam || value.Borrow == plir.BorrowMut {
				provenance, unsafeClass := summaryClassesForPLIRValue(
					value,
					factIDs,
					ProvenanceSafeKnown,
				)
				alias := AliasMutableExclusive
				if unsafeClass == UnsafeUnknown {
					alias = AliasUnknownConservative
				}
				if _, err := graph.AddFact(
					functionSummaryFact(fn.Name, pf.ID, "may_mutate_inout", pf.Source, value, provenance, unsafeClass, escapeStateForPLIRValue(value), alias, ownerForPLIRValue(value), ("inout parameter may be " +
						"mutated under exclusive-borrow evidence")),
				); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func functionSummaryFact(
	functionID, anchor, claim, site string,
	value plir.Value,
	provenance ProvenanceClass,
	unsafeClass UnsafeClass,
	escape EscapeState,
	alias AliasState,
	owner string,
	reason string,
) Fact {
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
	return FactID(
		fmt.Sprintf(
			"plir:%s:summary:%s:%s",
			safeFactIDPart(functionID),
			safeFactIDPart(anchor),
			safeFactIDPart(claim),
		),
	)
}

func safeFactIDPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "function"
	}
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' ||
			c == '-' {
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
	for _, candidate := range []string{
		"view:$return",
		"alloc_intent:$return",
		"local:$return",
		"param:$return",
	} {
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
	if value.Provenance.Kind == plir.ProvenanceExternal ||
		value.Provenance.Kind == plir.ProvenanceUnknown {
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

func returnedBorrowSource(
	fn plir.Function,
	value plir.Value,
	factIDs map[plirFactKey]FactID,
) (returnedBorrowSourceInfo, bool) {
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

func summaryClassesForPLIRValue(
	value plir.Value,
	factIDs map[plirFactKey]FactID,
	safe ProvenanceClass,
) (ProvenanceClass, UnsafeClass) {
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

func addPointerRetentionFactsForOperation(
	graph *Graph,
	functionID string,
	op plir.Operation,
	values map[string]plir.Value,
) error {
	for _, input := range op.Inputs {
		value, ok := plirValueForPath(input, values)
		if !ok || !plirValueMayRetainPointer(value) {
			continue
		}
		if _, err := graph.AddFact(
			functionSummaryFact(functionID, op.ID+":"+value.ID, "may_retain_pointer", op.Source, value, ProvenanceUnsafeUnknown, UnsafeUnknown, EscapeConservative, AliasUnknown, ownerForPLIRValue(value), ("unknown external call " +
				"may retain pointer argument")),
		); err != nil {
			return err
		}
	}
	return nil
}

func addPointerRetentionFactsForValues(
	graph *Graph,
	fn plir.Function,
	values map[string]plir.Value,
) error {
	for _, value := range values {
		if !plirValueMayRetainPointer(value) {
			continue
		}
		if _, err := graph.AddFact(
			functionSummaryFact(fn.Name, "retain:"+value.ID, "may_retain_pointer", nonEmpty(value.Source, summarySite(fn)), value, ProvenanceUnsafeUnknown, UnsafeUnknown, EscapeConservative, AliasUnknown, ownerForPLIRValue(value), ("pointer value may " +
				"escape or be retained outside a proven safe owner")),
		); err != nil {
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
