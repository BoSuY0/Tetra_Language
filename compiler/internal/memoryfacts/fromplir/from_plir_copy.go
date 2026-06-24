package fromplir

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/plir"
)

func addCopyMetadataFacts(
	graph *Graph,
	parent Fact,
	value plir.Value,
	op plir.Operation,
	factIDs map[plirFactKey]FactID,
	values map[string]plir.Value,
) error {
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
		return graph.InvalidateFact(
			id,
			"dynamic protocol dispatch cannot validate broad noalias evidence",
		)
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
			Reason: ("task group or structured concurrency boundary invalidates broad " +
				"noalias evidence"),
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
		{
			"mutable_exclusive",
			"mutable_exclusive",
			"inout parameter has exclusive mutable access for the call duration",
		},
		{
			"start_inout_exclusive",
			"start_inout_exclusive",
			"exclusive inout scope starts at function entry for the parameter",
		},
		{
			"end_inout_exclusive",
			"end_inout_exclusive",
			"exclusive inout scope ends at function return for the parameter",
		},
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
		{
			"no_alias_validated_narrow_unique_local",
			"no_alias_validated_narrow_unique_local",
			"unique local value has narrow noalias evidence only for this inout interval",
		},
		{
			"no_alias_validated_narrow_sequential_inout",
			"no_alias_validated_narrow_sequential_inout",
			"sequential inout calls are valid only after the prior exclusive interval ends",
		},
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

func addCopyIntoFacts(
	graph *Graph,
	functionID string,
	op plir.Operation,
	values map[string]plir.Value,
) error {
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

func copyIntoOperationFact(
	functionID string,
	op plir.Operation,
	values map[string]plir.Value,
) Fact {
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
		Reason: fmt.Sprintf(
			"copy_into operation from %q into %q records destination capacity and overlap contract: %s",
			source,
			destination,
			op.Note,
		),
	}
}

func addCopyIntoDestinationLengthCheckFact(
	graph *Graph,
	parent Fact,
	op plir.Operation,
	values map[string]plir.Value,
) error {
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
		Reason: fmt.Sprintf(
			"copy_into destination %q is sliced to source length in normal builds before the store loop",
			destination,
		),
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
		Reason: fmt.Sprintf(
			("copy_into writes into caller-owned destination %q only after " +
				"parented length check and overlap status %q"),
			destination,
			copyIntoOverlapStatusFromNote(op.Note),
		),
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
		Reason: fmt.Sprintf(
			"copy_into overlap status %q rejects zero-cost/noalias destination claim: %s",
			copyIntoOverlapStatusFromNote(op.Note),
			op.Note,
		),
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
		Reason: fmt.Sprintf(
			("copy_into overlap status %q remains conservative and cannot " +
				"authorize zero-cost/noalias destination claim: %s"),
			copyIntoOverlapStatusFromNote(op.Note),
			op.Note,
		),
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

func addSliceViewBoundsCheckFact(
	graph *Graph,
	parent Fact,
	value plir.Value,
	op plir.Operation,
) error {
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
		Reason: fmt.Sprintf(
			"safe view constructor retains dynamic bounds/length check in normal-build: %s",
			nonEmpty(note, value.ID),
		),
	})
	return err
}
