package fromplir

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/plir"
	semanticsresources "tetra_language/compiler/internal/semantics/resources"
)

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

func addBorrowAggregateV0Facts(
	graph *Graph,
	parent Fact,
	value plir.Value,
	op plir.Operation,
) error {
	ownerPath := ownerPathForPLIRValue(value)
	owner := nonEmpty(ownerFromOperationInput(op, 0), normalizeOwnerID(ownerPath), parent.OwnerID)
	if owner == "" {
		return nil
	}
	paramPath := ""
	if relative, ok := ownerPathRelativeTo(ownerPath, owner); ok && relative != "" {
		paramPath = relative
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
				Reason: fmt.Sprintf(
					"Memory Ideal v11 rejects witness provenance promotion for owner %q",
					owner,
				),
			})
			if err != nil {
				return err
			}
			return graph.InvalidateFact(
				id,
				fmt.Sprintf(
					"Memory Ideal v11 rejects witness provenance promotion for owner %q",
					owner,
				),
			)
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
			Reason: fmt.Sprintf(
				"Memory Ideal v11 keeps dynamic existential/protocol borrow conservative for owner %q",
				owner,
			),
		})
		return err
	}
	if claim == "static_witness_borrow_parent_validated" ||
		claim == "protocol_dispatch_report_integrity" {
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
			Reason: fmt.Sprintf(
				"Memory Ideal v11 validates %s for owner %q with compiler-owned parent fact",
				claim,
				owner,
			),
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
			Reason: fmt.Sprintf(
				("Memory Ideal v10 validates pre-await local borrow for owner %q " +
					"only with compiler-owned no-escape proof"),
				owner,
			),
		})
		if err != nil {
			return err
		}
		return graph.MarkValidated(id, borrowWrapperValidatorName(claim))
	}
	if claim == "post_await_borrow_conservative" ||
		claim == "actor_reentrant_callback_conservative" {
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
			Reason: fmt.Sprintf(
				"Memory Ideal v10 keeps %s conservative for owner %q",
				claim,
				owner,
			),
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
			Reason: fmt.Sprintf(
				"Memory Ideal v10 rejects task-owned borrow lifetime after cancellation for owner %q",
				owner,
			),
		})
		if err != nil {
			return err
		}
		return graph.InvalidateFact(
			id,
			fmt.Sprintf(
				"Memory Ideal v10 rejects task-owned borrow lifetime after cancellation for owner %q",
				owner,
			),
		)
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
			Reason: fmt.Sprintf(
				"Memory Ideal v3 keeps protocol dispatch borrow conservative for owner %q",
				owner,
			),
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
			Reason: fmt.Sprintf(
				"Memory Ideal v4 keeps async boundary borrow conservative for owner %q",
				owner,
			),
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
			Reason: fmt.Sprintf(
				"Memory Ideal v4 rejects %s for owner %q without explicit copy",
				claim,
				owner,
			),
		})
		if err != nil {
			return err
		}
		return graph.InvalidateFact(
			id,
			fmt.Sprintf(
				"Memory Ideal v4 rejects %s for owner %q without explicit copy",
				claim,
				owner,
			),
		)
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

func ownerPathRelativeTo(path string, owner string) (string, bool) {
	relative, ok := semanticsresources.Path(path).RelativeTo(semanticsresources.Path(owner))
	return relative.String(), ok
}

func memoryIdealBorrowWrapperClaim(
	value plir.Value,
	op plir.Operation,
	parent Fact,
	paramPath string,
) (string, bool) {
	context := strings.ToLower(strings.Join([]string{
		value.ID,
		value.Provenance.Root,
		value.Lifetime.Owner,
		op.Note,
		parent.Reason,
		paramPath,
	}, " "))
	if (strings.Contains(
		context,
		"pre-await",
	) || strings.Contains(
		context,
		"pre_await",
	) || strings.Contains(
		context,
		"before suspension",
	)) &&
		(strings.Contains(
			context,
			"no_escape_proof",
		) || strings.Contains(
			context,
			"no-escape proof",
		) || strings.Contains(
			context,
			"no escape proof",
		)) {
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
		(strings.Contains(
			context,
			"normal_build_check",
		) && strings.Contains(
			context,
			"source_fact_id",
		) && strings.Contains(
			context,
			"cost_class",
		)) {
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
	if strings.Contains(context, "callback_arg") || strings.Contains(context, "callback arg") ||
		strings.Contains(context, "callback parameter") {
		return "callback_arg_contains_borrow", true
	}
	if strings.Contains(context, "async_boundary") || strings.Contains(context, "async boundary") ||
		strings.Contains(context, "async.boundary") ||
		strings.Contains(context, "await") ||
		strings.Contains(context, "suspension") {
		return "async_boundary_borrow_conservative", true
	}
	if strings.Contains(context, "task_boundary") || strings.Contains(context, "task boundary") ||
		strings.Contains(context, "task.boundary") ||
		strings.Contains(context, "task_spawn") {
		return "task_boundary_borrow_rejected", true
	}
	if strings.Contains(context, "actor_boundary") || strings.Contains(context, "actor boundary") ||
		strings.Contains(context, "actor.boundary") ||
		strings.Contains(context, "send_typed") {
		return "actor_boundary_borrow_rejected", true
	}
	if strings.Contains(context, "protocol_dispatch") ||
		strings.Contains(context, "protocol dispatch") ||
		strings.Contains(context, "dynamic dispatch") ||
		strings.Contains(context, "dynamic protocol") {
		return "protocol_dispatch_borrow_conservative", true
	}
	if strings.Contains(context, "interface_value") ||
		strings.Contains(context, "interface value") ||
		strings.Contains(context, "protocol_value") ||
		strings.Contains(context, "protocol value") ||
		strings.Contains(context, "interface/protocol value") {
		return "interface_value_contains_borrow", true
	}
	if strings.Contains(context, "function_value") || strings.Contains(context, "function value") ||
		strings.Contains(context, "function-typed value") ||
		strings.Contains(context, "function typed value") {
		return "function_value_contains_borrow", true
	}
	if strings.Contains(context, "generic_wrapper") ||
		strings.Contains(context, "generic wrapper") ||
		strings.Contains(context, "monomorphized generic") ||
		strings.Contains(context, "box<") {
		return "generic_wrapper_contains_borrow", true
	}
	if strings.Contains(context, "enum_payload") || strings.Contains(context, "enum payload") ||
		strings.Contains(context, "enum") {
		return "enum_payload_contains_borrow", true
	}
	if strings.Contains(context, "optional") || strings.Contains(context, "maybe") ||
		strings.Contains(context, "payload") {
		return "optional_contains_borrow", true
	}
	if paramPath != "" || strings.Contains(context, "struct") ||
		strings.Contains(context, "field") ||
		strings.Contains(context, "aggregate") {
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
