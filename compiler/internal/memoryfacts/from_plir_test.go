package memoryfacts

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/runtimeabi"
)

func TestFromPLIRAndAllocPlanEmitsBorrowCopyVocabulary(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "main",
		Values: []plir.Value{
			{
				ID:         "param:xs",
				Kind:       plir.ValueParam,
				Type:       "[]u8",
				Source:     "test.tetra:2:11",
				Region:     "fn:main",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "xs"},
				Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "xs"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeNoEscape,
			},
			{
				ID:         "view:borrowed",
				Kind:       plir.ValueView,
				Type:       "[]u8",
				Source:     "test.tetra:4:28",
				Region:     "fn:main",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "derived:xs"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:4:28", Death: "return", Owner: "borrowed"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeNoEscape,
			},
			{
				ID:     "alloc_intent:copied",
				Kind:   plir.ValueAllocIntent,
				Type:   "[]u8",
				Source: "test.tetra:5:27",
				Region: "allocation:copied",
				Alloc: &plir.AllocIntent{
					ElementType: "u8",
					ElementSize: 1,
					LengthExpr:  "borrowed.len",
					Builtin:     "core.slice_copy_u8",
					Source:      "test.tetra:5:27",
				},
				Provenance: plir.Provenance{Kind: plir.ProvenanceAllocation, Root: "copied"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:5:27", Owner: "copied"},
				Escape:     plir.EscapeNoEscape,
			},
		},
		Ops: []plir.Operation{
			{ID: "op_borrow", Kind: plir.OpCall, Source: "test.tetra:4:28", Inputs: []string{"xs"}, Outputs: []string{"view:borrowed"}, Note: "core.slice_borrow_u8 creates borrowed view without allocation"},
			{ID: "op_copy", Kind: plir.OpAllocIntent, Source: "test.tetra:5:27", Inputs: []string{"borrowed"}, Outputs: []string{"alloc_intent:copied"}, Note: "core.slice_copy_u8 creates owned copy with new provenance"},
			{ID: "op_copy_into", Kind: plir.OpCall, Source: "test.tetra:6:31", Inputs: []string{"copied", "dst"}, Note: "core.slice_copy_into_u8 copies into caller-owned destination without allocation"},
		},
		Facts: []plir.Fact{
			{ID: "f_borrow", Kind: plir.FactBorrowedImm, ValueID: "view:borrowed", Source: "test.tetra:4:28", Reason: "explicit borrow view"},
			{ID: "f_no_escape", Kind: plir.FactNoEscape, ValueID: "view:borrowed", Source: "test.tetra:4:28", Reason: "explicit borrowed view may not escape owner"},
			{ID: "f_owned", Kind: plir.FactOwned, ValueID: "alloc_intent:copied", Source: "test.tetra:5:27", Reason: "copy result owns new storage"},
			{ID: "f_prov", Kind: plir.FactProvenanceKnown, ValueID: "alloc_intent:copied", Source: "test.tetra:5:27", Reason: "copy creates owned value with new provenance"},
		},
	}}}
	plan := &allocplan.Plan{Functions: []allocplan.FunctionPlan{{
		Name: "main",
		Allocations: []allocplan.Allocation{{
			ID:                    "copied",
			SiteID:                "alloc:main:copied",
			ValueID:               "alloc_intent:copied",
			Source:                "test.tetra:5:27",
			Builtin:               "core.slice_copy_u8",
			ElementType:           "u8",
			PlannedStorage:        allocplan.StorageStack,
			ActualLoweringStorage: allocplan.StorageStack,
			Reason:                "fixed small no-escape copy",
			ValidationStatus:      "validated_no_escape",
		}},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, plan)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, want := range []string{
		"borrowed_imm",
		"no_escape",
		"borrow_owner",
		"borrow_source_fact_id",
		"copy_owned",
		"copy_source_fact_id",
		"copy_into_destination_fact_id",
	} {
		if !reportHasClaim(report, want) {
			t.Fatalf("memory report missing claim %q:\n%+v", want, report.Rows)
		}
	}
	if !reportHasOwner(report, "borrow_owner", "xs") {
		t.Fatalf("borrow_owner row did not preserve owner xs:\n%+v", report.Rows)
	}
}

func TestFromPLIRAndAllocPlanDoesNotValidateStackHeapFallback(t *testing.T) {
	plan := &allocplan.Plan{Functions: []allocplan.FunctionPlan{{
		Name: "main",
		Allocations: []allocplan.Allocation{{
			ID:                    "xs",
			SiteID:                "alloc:main:xs",
			ValueID:               "alloc_intent:xs",
			Source:                "test.tetra:3:17",
			Builtin:               "core.make_u8",
			ElementType:           "u8",
			PlannedStorage:        allocplan.StorageStack,
			ActualLoweringStorage: allocplan.StorageHeap,
			Reason:                "planner chose stack but backend lowered heap fallback",
			ValidationStatus:      "validated_no_escape",
			LoweringStatus:        "conservative_heap_fallback",
		}},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", nil, plan)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, row := range report.Rows {
		if row.SourceFactID != "allocplan:main:xs" {
			continue
		}
		if row.ClaimLevel == ClaimValidated || row.ValidatorStatus == ValidatorPass {
			t.Fatalf("stack heap-fallback row was validated: %+v", row)
		}
		if row.PlannedStorage != StorageStack || row.ActualLoweringStorage != StorageHeap || row.LoweredArtifactID == "" {
			t.Fatalf("stack heap-fallback row lost storage truth fields: %+v", row)
		}
		return
	}
	t.Fatalf("missing allocplan stack heap-fallback row: %+v", report.Rows)
}

func TestFromPLIRAndAllocPlanDoesNotValidateFunctionTempRegionHeapFallback(t *testing.T) {
	plan := &allocplan.Plan{Functions: []allocplan.FunctionPlan{{
		Name: "main",
		Allocations: []allocplan.Allocation{{
			ID:                    "copied",
			SiteID:                "alloc:main:copied",
			ValueID:               "alloc_intent:copied",
			Source:                "test.tetra:5:27",
			Builtin:               "core.slice_copy_u8",
			ElementType:           "u8",
			PlannedStorage:        allocplan.StorageFunctionTempRegion,
			ActualLoweringStorage: allocplan.StorageHeap,
			Reason:                "planned function-temp region but backend lowered heap fallback",
			ValidationStatus:      "validated_function_temp_region_scope",
			LoweringStatus:        "conservative_heap_fallback",
		}},
	}}}
	if allocPlanValidationPasses(plan.Functions[0].Allocations[0]) {
		t.Fatalf("FunctionTempRegion heap fallback was treated as a validated allocation lowering")
	}

	graph, err := FromPLIRAndAllocPlan("program", nil, plan)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, row := range report.Rows {
		if row.SourceFactID != "allocplan:main:copied" {
			continue
		}
		if row.ClaimLevel == ClaimValidated || row.ValidatorStatus == ValidatorPass {
			t.Fatalf("FunctionTempRegion heap-fallback row was validated: %+v", row)
		}
		if row.PlannedStorage != StorageFunctionTempRegion || row.ActualLoweringStorage != StorageHeap || row.LoweredArtifactID == "" {
			t.Fatalf("FunctionTempRegion heap-fallback row lost storage truth fields: %+v", row)
		}
		return
	}
	t.Fatalf("missing allocplan FunctionTempRegion heap-fallback row: %+v", report.Rows)
}

func TestFromPLIRAndAllocPlanRejectsHeapFallbackWithoutReason(t *testing.T) {
	plan := &allocplan.Plan{Functions: []allocplan.FunctionPlan{{
		Name: "main",
		Allocations: []allocplan.Allocation{{
			ID:                    "xs",
			SiteID:                "alloc:main:xs",
			ValueID:               "alloc_intent:xs",
			Source:                "test.tetra:3:17",
			Builtin:               "core.make_u8",
			ElementType:           "u8",
			PlannedStorage:        allocplan.StorageStack,
			ActualLoweringStorage: allocplan.StorageHeap,
			Reason:                "",
			ValidationStatus:      "validated_no_escape",
			LoweringStatus:        "conservative_heap_fallback",
		}},
	}}}

	_, err := FromPLIRAndAllocPlan("program", nil, plan)
	if err == nil || !strings.Contains(err.Error(), "reason") {
		t.Fatalf("FromPLIRAndAllocPlan error = %v, want heap fallback reason rejection", err)
	}
}

func TestFromPLIRAndAllocPlanKeepsUnsafeUnknownBorrowConservative(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "main",
		Values: []plir.Value{{
			ID:         "view:raw",
			Kind:       plir.ValueView,
			Type:       "[]u8",
			Source:     "test.tetra:3:17",
			Region:     "fn:main",
			Provenance: plir.Provenance{Kind: plir.ProvenanceUnknown, Root: "raw"},
			Lifetime:   plir.Lifetime{Birth: "test.tetra:3:17", Owner: "raw"},
			Borrow:     plir.BorrowImm,
			Escape:     plir.EscapeNoEscape,
		}},
		Facts: []plir.Fact{
			{ID: "f_unknown", Kind: plir.FactProvenanceUnknown, ValueID: "view:raw", Source: "test.tetra:3:17", Reason: "borrow source provenance is external or unknown"},
			{ID: "f_borrow", Kind: plir.FactBorrowedImm, ValueID: "view:raw", Source: "test.tetra:3:17", Reason: "explicit borrow view"},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	if reportHasClaim(report, "borrow_owner") || reportHasClaim(report, "borrow_source_fact_id") {
		t.Fatalf("unsafe_unknown borrow emitted safe owner/source metadata:\n%+v", report.Rows)
	}
	for _, row := range report.Rows {
		if row.Claim == "borrowed_imm" && (row.ProvenanceClass != ProvenanceUnsafeUnknown || row.ClaimLevel != ClaimConservative) {
			t.Fatalf("unsafe_unknown borrowed_imm row = %+v, want unsafe_unknown/conservative", row)
		}
	}
}

func TestFromPLIRAndAllocPlanEmitsInoutAliasVocabulary(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "mutate",
		Values: []plir.Value{{
			ID:         "param:xs",
			Kind:       plir.ValueParam,
			Type:       "[]u8",
			Source:     "test.tetra:2:13",
			Region:     "fn:mutate",
			Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "xs"},
			Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "xs"},
			Borrow:     plir.BorrowMut,
			Escape:     plir.EscapeNoEscape,
		}},
		Facts: []plir.Fact{
			{ID: "f_borrow_mut", Kind: plir.FactBorrowedMut, ValueID: "param:xs", Source: "test.tetra:2:13", Reason: "inout parameter"},
			{ID: "f_region", Kind: plir.FactRegionAlive, ValueID: "param:xs", Region: "fn:mutate", Source: "test.tetra:2:13", Reason: "function region alive"},
			{ID: "f_prov", Kind: plir.FactProvenanceKnown, ValueID: "param:xs", Source: "test.tetra:2:13", Reason: "parameter provenance"},
			{ID: "f_no_alias", Kind: plir.FactNoAlias, ValueID: "param:xs", Region: "fn:mutate", Source: "test.tetra:2:13", Reason: "inout parameter has exclusive mutable access for call duration"},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, want := range []string{
		"no_alias",
		"mutable_exclusive",
		"start_inout_exclusive",
		"end_inout_exclusive",
	} {
		if !reportHasClaim(report, want) {
			t.Fatalf("memory report missing claim %q:\n%+v", want, report.Rows)
		}
	}
	if !reportHasAliasState(report, "no_alias", AliasMutableExclusive) {
		t.Fatalf("no_alias row did not carry mutable_exclusive alias state:\n%+v", report.Rows)
	}
}

func TestMemoryIdealV0ProjectsBorrowAggregateAndOptionalFacts(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "borrowAggregate",
		Values: []plir.Value{
			{
				ID:         "view:holder.view",
				Kind:       plir.ValueView,
				Type:       "[]u8",
				Source:     "test.tetra:5:17",
				Region:     "fn:borrowAggregate",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "derived:xs.view"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:5:17", Death: "return", Owner: "xs.view"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeNoEscape,
			},
			{
				ID:         "view:maybe.payload",
				Kind:       plir.ValueView,
				Type:       "str",
				Source:     "test.tetra:9:17",
				Region:     "fn:borrowAggregate",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "derived:text.payload"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:9:17", Death: "return", Owner: "text.payload"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeNoEscape,
			},
		},
		Ops: []plir.Operation{
			{ID: "op_struct", Kind: plir.OpAggregate, Source: "test.tetra:5:17", Inputs: []string{"xs"}, Outputs: []string{"view:holder.view"}, Note: "struct field carries borrowed view"},
			{ID: "op_optional", Kind: plir.OpAggregate, Source: "test.tetra:9:17", Inputs: []string{"text"}, Outputs: []string{"view:maybe.payload"}, Note: "optional payload carries borrowed String view"},
		},
		Facts: []plir.Fact{
			{ID: "f_struct_borrow", Kind: plir.FactBorrowedImm, ValueID: "view:holder.view", Source: "test.tetra:5:17", Reason: "borrow through struct field"},
			{ID: "f_optional_borrow", Kind: plir.FactBorrowedImm, ValueID: "view:maybe.payload", Source: "test.tetra:9:17", Reason: "borrow through optional payload"},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, want := range []string{"aggregate_contains_borrow", "optional_contains_borrow"} {
		row, ok := reportRowByClaim(report, want)
		if !ok {
			t.Fatalf("memory report missing claim %q:\n%+v", want, report.Rows)
		}
		if row.ParentFactID == "" || row.OwnerID == "" || row.ValidatorName != "borrow_aggregate_escape_validator" || row.ClaimLevel != ClaimValidated {
			t.Fatalf("%s row = %+v, want validated row with parent fact and owner", want, row)
		}
	}
}

func TestMemoryIdealV1ProjectsEnumPayloadAndGenericWrapperFacts(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "borrowCarrierV1",
		Values: []plir.Value{
			{
				ID:         "view:msg.payload",
				Kind:       plir.ValueView,
				Type:       "[]u8",
				Source:     "test.tetra:5:17",
				Region:     "fn:borrowCarrierV1",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "derived:xs.enum_payload"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:5:17", Death: "return", Owner: "xs.enum_payload"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeNoEscape,
			},
			{
				ID:         "view:box.value",
				Kind:       plir.ValueView,
				Type:       "[]u8",
				Source:     "test.tetra:9:17",
				Region:     "fn:borrowCarrierV1",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "derived:ys.generic_wrapper"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:9:17", Death: "return", Owner: "ys.generic_wrapper.value"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeNoEscape,
			},
		},
		Ops: []plir.Operation{
			{ID: "op_enum", Kind: plir.OpAggregate, Source: "test.tetra:5:17", Inputs: []string{"xs"}, Outputs: []string{"view:msg.payload"}, Note: "enum payload contains borrowed view"},
			{ID: "op_generic", Kind: plir.OpAggregate, Source: "test.tetra:9:17", Inputs: []string{"ys"}, Outputs: []string{"view:box.value"}, Note: "monomorphized generic wrapper Box<[]u8>.value contains borrowed view"},
		},
		Facts: []plir.Fact{
			{ID: "f_enum_borrow", Kind: plir.FactBorrowedImm, ValueID: "view:msg.payload", Source: "test.tetra:5:17", Reason: "borrow through enum payload"},
			{ID: "f_generic_borrow", Kind: plir.FactBorrowedImm, ValueID: "view:box.value", Source: "test.tetra:9:17", Reason: "borrow through monomorphized generic wrapper"},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, want := range []string{"enum_payload_contains_borrow", "generic_wrapper_contains_borrow"} {
		row, ok := reportRowByClaim(report, want)
		if !ok {
			t.Fatalf("memory report missing claim %q:\n%+v", want, report.Rows)
		}
		if row.ParentFactID == "" || row.OwnerID == "" || row.ValidatorName != "borrow_aggregate_escape_validator" || row.ClaimLevel != ClaimValidated {
			t.Fatalf("%s row = %+v, want validated row with parent fact and owner", want, row)
		}
	}
}

func TestMemoryIdealV2ProjectsFunctionValueAndCallbackFacts(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "borrowCarrierV2",
		Values: []plir.Value{
			{
				ID:         "view:fn_value.arg",
				Kind:       plir.ValueView,
				Type:       "[]u8",
				Source:     "test.tetra:5:17",
				Region:     "fn:borrowCarrierV2",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "derived:xs.function_value"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:5:17", Death: "return", Owner: "xs.function_value"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeNoEscape,
			},
			{
				ID:         "view:callback.arg",
				Kind:       plir.ValueView,
				Type:       "[]u8",
				Source:     "test.tetra:9:17",
				Region:     "fn:borrowCarrierV2",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "derived:ys.callback_arg"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:9:17", Death: "return", Owner: "ys.callback_arg"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeNoEscape,
			},
			{
				ID:         "param:dst",
				Kind:       plir.ValueParam,
				Type:       "[]u8",
				Source:     "test.tetra:12:13",
				Region:     "fn:borrowCarrierV2",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "dst"},
				Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "dst"},
				Borrow:     plir.BorrowMut,
				Escape:     plir.EscapeNoEscape,
			},
		},
		Ops: []plir.Operation{
			{ID: "op_function_value", Kind: plir.OpCall, Source: "test.tetra:5:17", Inputs: []string{"xs"}, Outputs: []string{"view:fn_value.arg"}, Note: "function-typed value contains borrowed view argument"},
			{ID: "op_callback_arg", Kind: plir.OpCall, Source: "test.tetra:9:17", Inputs: []string{"ys"}, Outputs: []string{"view:callback.arg"}, Note: "known direct callback parameter contains borrowed view argument"},
		},
		Facts: []plir.Fact{
			{ID: "f_function_value_borrow", Kind: plir.FactBorrowedImm, ValueID: "view:fn_value.arg", Source: "test.tetra:5:17", Reason: "borrow through function-typed value"},
			{ID: "f_callback_arg_borrow", Kind: plir.FactBorrowedImm, ValueID: "view:callback.arg", Source: "test.tetra:9:17", Reason: "borrow through callback parameter"},
			{ID: "f_callback_inout", Kind: plir.FactNoAlias, ValueID: "param:dst", Source: "test.tetra:12:13", Reason: "callback/reentrant inout cannot produce broad noalias"},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, tc := range []struct {
		claim     string
		validator string
	}{
		{claim: "function_value_contains_borrow", validator: "function_value_borrow_escape_validator"},
		{claim: "callback_arg_contains_borrow", validator: "callback_borrow_escape_validator"},
		{claim: "callback_inout_conservative", validator: "callback_alias_conservative_validator"},
	} {
		row, ok := reportRowByClaim(report, tc.claim)
		if !ok {
			t.Fatalf("memory report missing claim %q:\n%+v", tc.claim, report.Rows)
		}
		if row.ParentFactID == "" || row.OwnerID == "" || row.ValidatorName != tc.validator {
			t.Fatalf("%s row = %+v, want parent, owner, validator %q", tc.claim, row, tc.validator)
		}
	}
	inout, ok := reportRowByClaim(report, "callback_inout_conservative")
	if !ok {
		t.Fatalf("missing callback_inout_conservative row")
	}
	if inout.AliasState != AliasInvalidatedByCall || inout.CostClass != CostConservativeFallback || inout.ClaimLevel == ClaimValidated {
		t.Fatalf("callback_inout_conservative row = %+v, want invalidated-by-call conservative fallback and not validated", inout)
	}
}

func TestMemoryIdealV2UnknownCallbackTargetDoesNotEmitTrustedBorrowFacts(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "unknownCallback",
		Values: []plir.Value{{
			ID:         "view:callback.arg",
			Kind:       plir.ValueView,
			Type:       "[]u8",
			Source:     "test.tetra:4:17",
			Region:     "fn:unknownCallback",
			Provenance: plir.Provenance{Kind: plir.ProvenanceUnknown, Root: "callback_arg"},
			Lifetime:   plir.Lifetime{Birth: "test.tetra:4:17", Death: "return", Owner: "callback_arg"},
			Borrow:     plir.BorrowImm,
			Escape:     plir.EscapeConservative,
		}},
		Ops: []plir.Operation{
			{ID: "op_callback_arg", Kind: plir.OpCall, Source: "test.tetra:4:17", Inputs: []string{"xs"}, Outputs: []string{"view:callback.arg"}, Note: "unknown callback target contains borrowed view argument"},
		},
		Facts: []plir.Fact{
			{ID: "f_unknown", Kind: plir.FactProvenanceUnknown, ValueID: "view:callback.arg", Source: "test.tetra:4:17", Reason: "unknown callback target"},
			{ID: "f_callback_arg_borrow", Kind: plir.FactBorrowedImm, ValueID: "view:callback.arg", Source: "test.tetra:4:17", Reason: "borrow through unknown callback parameter"},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, claim := range []string{"function_value_contains_borrow", "callback_arg_contains_borrow"} {
		if reportHasClaim(report, claim) {
			t.Fatalf("unsafe/unknown callback target emitted trusted claim %q:\n%+v", claim, report.Rows)
		}
	}
}

func TestMemoryIdealV3ProjectsInterfaceProtocolFacts(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "borrowCarrierV3",
		Values: []plir.Value{
			{
				ID:         "view:interface.value",
				Kind:       plir.ValueView,
				Type:       "[]u8",
				Source:     "test.tetra:5:17",
				Region:     "fn:borrowCarrierV3",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "derived:xs.interface_value"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:5:17", Death: "return", Owner: "xs.interface_value"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeNoEscape,
			},
			{
				ID:         "view:protocol.dispatch",
				Kind:       plir.ValueView,
				Type:       "[]u8",
				Source:     "test.tetra:9:17",
				Region:     "fn:borrowCarrierV3",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "derived:ys.protocol_dispatch"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:9:17", Death: "return", Owner: "ys.protocol_dispatch"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeConservative,
			},
			{
				ID:         "param:dst",
				Kind:       plir.ValueParam,
				Type:       "[]u8",
				Source:     "test.tetra:12:13",
				Region:     "fn:borrowCarrierV3",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "dst"},
				Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "dst"},
				Borrow:     plir.BorrowMut,
				Escape:     plir.EscapeNoEscape,
			},
		},
		Ops: []plir.Operation{
			{ID: "op_interface_value", Kind: plir.OpCall, Source: "test.tetra:5:17", Inputs: []string{"xs"}, Outputs: []string{"view:interface.value"}, Note: "known static interface/protocol value contains borrowed view argument"},
			{ID: "op_protocol_dispatch", Kind: plir.OpCall, Source: "test.tetra:9:17", Inputs: []string{"ys"}, Outputs: []string{"view:protocol.dispatch"}, Note: "unknown dynamic protocol dispatch borrow remains conservative"},
		},
		Facts: []plir.Fact{
			{ID: "f_interface_borrow", Kind: plir.FactBorrowedImm, ValueID: "view:interface.value", Source: "test.tetra:5:17", Reason: "borrow through interface/protocol value with statically known target"},
			{ID: "f_protocol_dispatch_borrow", Kind: plir.FactBorrowedImm, ValueID: "view:protocol.dispatch", Source: "test.tetra:9:17", Reason: "borrow through unknown dynamic protocol dispatch remains conservative"},
			{ID: "f_protocol_dispatch_noalias", Kind: plir.FactNoAlias, ValueID: "param:dst", Source: "test.tetra:12:13", Reason: "protocol/interface dispatch cannot produce broad noalias"},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, tc := range []struct {
		claim     string
		validator string
	}{
		{claim: "interface_value_contains_borrow", validator: "interface_borrow_escape_validator"},
		{claim: "protocol_dispatch_borrow_conservative", validator: "protocol_dispatch_borrow_validator"},
		{claim: "protocol_dispatch_noalias_conservative", validator: "protocol_dispatch_alias_conservative_validator"},
	} {
		row, ok := reportRowByClaim(report, tc.claim)
		if !ok {
			t.Fatalf("memory report missing claim %q:\n%+v", tc.claim, report.Rows)
		}
		if row.ParentFactID == "" || row.OwnerID == "" || row.ValidatorName != tc.validator {
			t.Fatalf("%s row = %+v, want parent, owner, validator %q", tc.claim, row, tc.validator)
		}
	}
	for _, claim := range []string{"protocol_dispatch_borrow_conservative", "protocol_dispatch_noalias_conservative"} {
		row, ok := reportRowByClaim(report, claim)
		if !ok {
			t.Fatalf("missing conservative protocol row %q", claim)
		}
		if row.CostClass != CostConservativeFallback || row.ClaimLevel == ClaimValidated {
			t.Fatalf("%s row = %+v, want conservative fallback and not validated", claim, row)
		}
	}
}

func TestMemoryIdealV3UnknownDynamicDispatchDoesNotEmitTrustedInterfaceFacts(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "unknownProtocolDispatch",
		Values: []plir.Value{{
			ID:         "view:protocol.dispatch",
			Kind:       plir.ValueView,
			Type:       "[]u8",
			Source:     "test.tetra:4:17",
			Region:     "fn:unknownProtocolDispatch",
			Provenance: plir.Provenance{Kind: plir.ProvenanceUnknown, Root: "protocol_dispatch"},
			Lifetime:   plir.Lifetime{Birth: "test.tetra:4:17", Death: "return", Owner: "protocol_dispatch"},
			Borrow:     plir.BorrowImm,
			Escape:     plir.EscapeConservative,
		}},
		Ops: []plir.Operation{
			{ID: "op_protocol_dispatch", Kind: plir.OpCall, Source: "test.tetra:4:17", Inputs: []string{"xs"}, Outputs: []string{"view:protocol.dispatch"}, Note: "unknown dynamic protocol dispatch contains borrowed view argument"},
		},
		Facts: []plir.Fact{
			{ID: "f_unknown", Kind: plir.FactProvenanceUnknown, ValueID: "view:protocol.dispatch", Source: "test.tetra:4:17", Reason: "unknown dynamic protocol dispatch"},
			{ID: "f_protocol_dispatch_borrow", Kind: plir.FactBorrowedImm, ValueID: "view:protocol.dispatch", Source: "test.tetra:4:17", Reason: "borrow through unknown dynamic protocol dispatch"},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	if reportHasClaim(report, "interface_value_contains_borrow") {
		t.Fatalf("unknown dynamic dispatch emitted trusted interface claim:\n%+v", report.Rows)
	}
}

func TestMemoryIdealV4ProjectsAsyncTaskActorBoundaryFacts(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "borrowCarrierV4",
		Values: []plir.Value{
			{
				ID:         "view:async.boundary",
				Kind:       plir.ValueView,
				Type:       "[]u8",
				Source:     "test.tetra:5:17",
				Region:     "fn:borrowCarrierV4",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "derived:xs.async_boundary"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:5:17", Death: "await", Owner: "xs.async_boundary"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeConservative,
			},
			{
				ID:         "view:task.boundary",
				Kind:       plir.ValueView,
				Type:       "[]u8",
				Source:     "test.tetra:9:17",
				Region:     "fn:borrowCarrierV4",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "derived:ys.task_boundary"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:9:17", Death: "task_spawn", Owner: "ys.task_boundary"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeTask,
			},
			{
				ID:         "view:actor.boundary",
				Kind:       plir.ValueView,
				Type:       "[]u8",
				Source:     "test.tetra:13:17",
				Region:     "fn:borrowCarrierV4",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "derived:zs.actor_boundary"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:13:17", Death: "send_typed", Owner: "zs.actor_boundary"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeActor,
			},
			{
				ID:         "param:dst",
				Kind:       plir.ValueParam,
				Type:       "[]u8",
				Source:     "test.tetra:17:13",
				Region:     "fn:borrowCarrierV4",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "dst"},
				Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "dst"},
				Borrow:     plir.BorrowMut,
				Escape:     plir.EscapeNoEscape,
			},
		},
		Ops: []plir.Operation{
			{ID: "op_async_boundary", Kind: plir.OpCall, Source: "test.tetra:5:17", Inputs: []string{"xs"}, Outputs: []string{"view:async.boundary"}, Note: "borrow crosses async/await suspension boundary and remains conservative"},
			{ID: "op_task_boundary", Kind: plir.OpCall, Source: "test.tetra:9:17", Inputs: []string{"ys"}, Outputs: []string{"view:task.boundary"}, Note: "borrow crosses task boundary without explicit copy and is rejected"},
			{ID: "op_actor_boundary", Kind: plir.OpCall, Source: "test.tetra:13:17", Inputs: []string{"zs"}, Outputs: []string{"view:actor.boundary"}, Note: "borrow crosses actor boundary without explicit copy and is rejected"},
		},
		Facts: []plir.Fact{
			{ID: "f_async_boundary_borrow", Kind: plir.FactBorrowedImm, ValueID: "view:async.boundary", Source: "test.tetra:5:17", Reason: "borrow crossing async/await suspension boundary"},
			{ID: "f_task_boundary_borrow", Kind: plir.FactBorrowedImm, ValueID: "view:task.boundary", Source: "test.tetra:9:17", Reason: "borrow crossing task boundary without explicit copy"},
			{ID: "f_actor_boundary_borrow", Kind: plir.FactBorrowedImm, ValueID: "view:actor.boundary", Source: "test.tetra:13:17", Reason: "borrow crossing actor boundary without explicit copy"},
			{ID: "f_boundary_noalias", Kind: plir.FactNoAlias, ValueID: "param:dst", Source: "test.tetra:17:13", Reason: "task/actor boundary cannot produce broad noalias"},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, tc := range []struct {
		claim     string
		validator string
	}{
		{claim: "async_boundary_borrow_conservative", validator: "async_boundary_borrow_validator"},
		{claim: "task_boundary_borrow_rejected", validator: "task_boundary_borrow_validator"},
		{claim: "actor_boundary_borrow_rejected", validator: "actor_boundary_borrow_validator"},
		{claim: "boundary_noalias_conservative", validator: "boundary_alias_conservative_validator"},
	} {
		row, ok := reportRowByClaim(report, tc.claim)
		if !ok {
			t.Fatalf("memory report missing claim %q:\n%+v", tc.claim, report.Rows)
		}
		if row.ParentFactID == "" || row.OwnerID == "" || row.ValidatorName != tc.validator {
			t.Fatalf("%s row = %+v, want parent, owner, validator %q", tc.claim, row, tc.validator)
		}
	}
	asyncRow, _ := reportRowByClaim(report, "async_boundary_borrow_conservative")
	if asyncRow.CostClass != CostConservativeFallback || asyncRow.ClaimLevel == ClaimValidated {
		t.Fatalf("async_boundary_borrow_conservative row = %+v, want conservative fallback and not validated", asyncRow)
	}
	for _, claim := range []string{"task_boundary_borrow_rejected", "actor_boundary_borrow_rejected"} {
		row, _ := reportRowByClaim(report, claim)
		if row.CostClass != CostUnsupportedRejected || row.ClaimLevel != ClaimRejected {
			t.Fatalf("%s row = %+v, want rejected unsupported boundary fact", claim, row)
		}
	}
	noalias, _ := reportRowByClaim(report, "boundary_noalias_conservative")
	if noalias.AliasState != AliasInvalidatedByCall || noalias.CostClass != CostConservativeFallback || noalias.ClaimLevel == ClaimValidated {
		t.Fatalf("boundary_noalias_conservative row = %+v, want invalidated-by-call conservative fallback and not validated", noalias)
	}
}

func TestMemoryIdealV4UnknownTaskActorTargetDoesNotEmitTrustedBoundaryFacts(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "unknownTaskActorBoundary",
		Values: []plir.Value{{
			ID:         "view:task.boundary",
			Kind:       plir.ValueView,
			Type:       "[]u8",
			Source:     "test.tetra:4:17",
			Region:     "fn:unknownTaskActorBoundary",
			Provenance: plir.Provenance{Kind: plir.ProvenanceUnknown, Root: "task_boundary"},
			Lifetime:   plir.Lifetime{Birth: "test.tetra:4:17", Death: "task_spawn", Owner: "task_boundary"},
			Borrow:     plir.BorrowImm,
			Escape:     plir.EscapeConservative,
		}},
		Ops: []plir.Operation{
			{ID: "op_task_boundary", Kind: plir.OpCall, Source: "test.tetra:4:17", Inputs: []string{"xs"}, Outputs: []string{"view:task.boundary"}, Note: "unknown task target contains borrowed view argument"},
		},
		Facts: []plir.Fact{
			{ID: "f_unknown", Kind: plir.FactProvenanceUnknown, ValueID: "view:task.boundary", Source: "test.tetra:4:17", Reason: "unknown task target"},
			{ID: "f_task_boundary_borrow", Kind: plir.FactBorrowedImm, ValueID: "view:task.boundary", Source: "test.tetra:4:17", Reason: "borrow through unknown task target"},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, claim := range []string{"async_boundary_borrow_conservative", "task_boundary_borrow_rejected", "actor_boundary_borrow_rejected"} {
		if reportHasClaim(report, claim) {
			t.Fatalf("unknown task/actor target emitted boundary claim %q:\n%+v", claim, report.Rows)
		}
	}
}

func TestMemoryIdealV0ProjectsNarrowInoutNoAliasFacts(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "mutate",
		Values: []plir.Value{{
			ID:         "param:xs",
			Kind:       plir.ValueParam,
			Type:       "[]u8",
			Source:     "test.tetra:2:13",
			Region:     "fn:mutate",
			Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "xs"},
			Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "xs"},
			Borrow:     plir.BorrowMut,
			Escape:     plir.EscapeNoEscape,
		}},
		Facts: []plir.Fact{
			{ID: "f_borrow_mut", Kind: plir.FactBorrowedMut, ValueID: "param:xs", Source: "test.tetra:2:13", Reason: "inout parameter"},
			{ID: "f_region", Kind: plir.FactRegionAlive, ValueID: "param:xs", Region: "fn:mutate", Source: "test.tetra:2:13", Reason: "function region alive"},
			{ID: "f_prov", Kind: plir.FactProvenanceKnown, ValueID: "param:xs", Source: "test.tetra:2:13", Reason: "parameter provenance"},
			{ID: "f_no_alias", Kind: plir.FactNoAlias, ValueID: "param:xs", Region: "fn:mutate", Source: "test.tetra:2:13", Reason: "unique local sequential inout interval"},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, want := range []string{"no_alias_validated_narrow_unique_local", "no_alias_validated_narrow_sequential_inout"} {
		row, ok := reportRowByClaim(report, want)
		if !ok {
			t.Fatalf("memory report missing claim %q:\n%+v", want, report.Rows)
		}
		if row.ParentFactID == "" || row.AliasState != AliasMutableExclusive || row.ValidatorName != "alias_interval_validator" || row.ClaimLevel != ClaimValidated {
			t.Fatalf("%s row = %+v, want validated narrow noalias row", want, row)
		}
	}
}

func TestFromPLIRAndAllocPlanEmitsFunctionSummaryVocabulary(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{
		{
			Name: "returnBorrow",
			Summary: &plir.FunctionSummary{
				ReturnOwnership:     "borrow",
				ReturnRegionSummary: map[string]int{"": 0},
			},
			Values: []plir.Value{
				{
					ID:         "param:xs",
					Kind:       plir.ValueParam,
					Type:       "[]u8",
					Source:     "test.tetra:2:18",
					Region:     "fn:returnBorrow",
					Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "xs"},
					Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "xs"},
					Borrow:     plir.BorrowImm,
					Escape:     plir.EscapeNoEscape,
				},
				{
					ID:         "view:borrowed",
					Kind:       plir.ValueView,
					Type:       "[]u8",
					Source:     "test.tetra:3:12",
					Region:     "fn:returnBorrow",
					Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "derived:xs"},
					Lifetime:   plir.Lifetime{Birth: "test.tetra:3:12", Death: "return", Owner: "borrowed"},
					Borrow:     plir.BorrowImm,
					Escape:     plir.EscapeReturn,
				},
			},
			Ops: []plir.Operation{
				{ID: "op_return_borrow", Kind: plir.OpReturn, Source: "test.tetra:3:5", Inputs: []string{"borrowed"}},
			},
			Facts: []plir.Fact{
				{ID: "f_borrow", Kind: plir.FactBorrowedImm, ValueID: "view:borrowed", Source: "test.tetra:3:12", Reason: "borrowed return view"},
				{ID: "f_prov", Kind: plir.FactProvenanceKnown, ValueID: "view:borrowed", Source: "test.tetra:3:12", Reason: "borrow preserves source provenance"},
			},
		},
		{
			Name: "returnCopy",
			Summary: &plir.FunctionSummary{
				ReturnOwnership: "owned",
			},
			Values: []plir.Value{{
				ID:     "alloc_intent:copied",
				Kind:   plir.ValueAllocIntent,
				Type:   "[]u8",
				Source: "test.tetra:7:12",
				Region: "allocation:copied",
				Alloc: &plir.AllocIntent{
					ElementType: "u8",
					ElementSize: 1,
					Builtin:     "core.slice_copy_u8",
					Source:      "test.tetra:7:12",
				},
				Provenance: plir.Provenance{Kind: plir.ProvenanceAllocation, Root: "copied"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:7:12", Owner: "copied"},
				Escape:     plir.EscapeReturn,
			}},
			Ops:   []plir.Operation{{ID: "op_return_copy", Kind: plir.OpReturn, Source: "test.tetra:7:5", Inputs: []string{"copied"}}},
			Facts: []plir.Fact{{ID: "f_owned", Kind: plir.FactOwned, ValueID: "alloc_intent:copied", Source: "test.tetra:7:12", Reason: "copy result owns new storage"}},
		},
		{
			Name: "unsafeReturn",
			Values: []plir.Value{{
				ID:         "view:raw",
				Kind:       plir.ValueView,
				Type:       "[]u8",
				Source:     "test.tetra:11:12",
				Region:     "external:raw",
				Provenance: plir.Provenance{Kind: plir.ProvenanceExternal, Root: "raw_parts"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:11:12", Owner: "raw"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeReturn,
			}},
			Ops:   []plir.Operation{{ID: "op_return_raw", Kind: plir.OpReturn, Source: "test.tetra:11:5", Inputs: []string{"raw"}}},
			Facts: []plir.Fact{{ID: "f_unknown", Kind: plir.FactProvenanceUnknown, ValueID: "view:raw", Source: "test.tetra:11:12", Reason: "raw slice external provenance"}},
		},
		{
			Name: "sideEffects",
			Summary: &plir.FunctionSummary{
				Effects:               []string{"actors", "alloc", "capability", "mem"},
				TouchesMutableGlobals: true,
				ReturnResourceSummary: map[string][]plir.ResourceProvenance{
					"ok.handle": {{ParamIndex: 0, ParamPath: "handle"}},
				},
				ThrowResourceSummary: map[string][]plir.ResourceProvenance{
					"": {{ParamIndex: 0, ParamPath: "handle"}},
				},
			},
			Values: []plir.Value{
				{
					ID:         "param:dst",
					Kind:       plir.ValueParam,
					Type:       "[]u8",
					Source:     "test.tetra:14:17",
					Region:     "fn:sideEffects",
					Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "dst"},
					Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "dst"},
					Borrow:     plir.BorrowMut,
					Escape:     plir.EscapeNoEscape,
				},
				{
					ID:         "param:owned",
					Kind:       plir.ValueParam,
					Type:       "island",
					Source:     "test.tetra:14:34",
					Region:     "fn:sideEffects",
					Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "owned"},
					Lifetime:   plir.Lifetime{Birth: "entry", Death: "actor", Owner: "owned"},
					Borrow:     plir.BorrowMove,
					Escape:     plir.EscapeActor,
				},
				{
					ID:         "param:p",
					Kind:       plir.ValueParam,
					Type:       "ptr",
					Source:     "test.tetra:14:49",
					Region:     "fn:sideEffects",
					Provenance: plir.Provenance{Kind: plir.ProvenanceUnknown, Root: "p"},
					Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "p"},
					Escape:     plir.EscapeUnsafe,
				},
			},
			Ops: []plir.Operation{
				{ID: "op_store", Kind: plir.OpGlobalStore, Source: "test.tetra:15:5", Inputs: []string{"dst"}, Outputs: []string{"G"}, Note: "global store"},
				{ID: "op_actor", Kind: plir.OpActorSend, Source: "test.tetra:16:5", Inputs: []string{"peer", "owned"}, Note: "core.send_typed typed actor ownership transfer"},
				{ID: "op_task", Kind: plir.OpCall, Source: "test.tetra:17:5", Inputs: []string{"worker"}, Note: "core.task_spawn_i32"},
				{ID: "op_closure", Kind: plir.OpClosure, Source: "test.tetra:18:12", Inputs: []string{"dst"}, Outputs: []string{"cb"}, Note: "closure captures environment"},
				{ID: "op_unknown", Kind: plir.OpCall, Source: "test.tetra:19:5", Inputs: []string{"p"}, Note: "ffi.unknown external call"},
			},
			Facts: []plir.Fact{
				{ID: "f_borrow_mut", Kind: plir.FactBorrowedMut, ValueID: "param:dst", Source: "test.tetra:14:17", Reason: "inout parameter"},
				{ID: "f_no_alias", Kind: plir.FactNoAlias, ValueID: "param:dst", Source: "test.tetra:14:17", Reason: "inout parameter has exclusive mutable access for call duration"},
				{ID: "f_moved", Kind: plir.FactMoved, ValueID: "param:owned", Source: "test.tetra:16:5", Reason: "typed actor ownership transfer moved payload"},
				{ID: "f_ptr_unknown", Kind: plir.FactProvenanceUnknown, ValueID: "param:p", Source: "test.tetra:14:49", Reason: "pointer retained across unknown unsafe boundary"},
			},
		},
	}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, want := range []string{
		"returns_borrow_from_param",
		"may_return_region",
		"returns_owned_new_allocation",
		"returns_unknown_unsafe",
		"may_store_global",
		"may_escape_to_actor",
		"may_escape_to_task",
		"may_capture_in_closure",
		"may_retain_pointer",
		"may_throw_resource",
		"may_return_resource",
		"may_consume_param",
		"may_mutate_inout",
		"requires_effects",
		"requires_capabilities",
		"unknown_external_call_conservative",
	} {
		if !reportHasClaim(report, want) {
			t.Fatalf("memory report missing summary claim %q:\n%+v", want, report.Rows)
		}
	}
	if !reportHasOwner(report, "returns_borrow_from_param", "xs") {
		t.Fatalf("returns_borrow_from_param row did not preserve owner xs:\n%+v", report.Rows)
	}
	for _, row := range report.Rows {
		if row.Claim == "returns_unknown_unsafe" && row.ClaimLevel != ClaimConservative {
			t.Fatalf("returns_unknown_unsafe row = %+v, want conservative", row)
		}
	}
}

func TestFromPLIRAndAllocPlanKeepsUnsafeUnknownSummaryFactsConservative(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "unsafeSummary",
		Values: []plir.Value{
			{
				ID:         "param:rawOwner",
				Kind:       plir.ValueParam,
				Type:       "ptr",
				Source:     "test.tetra:2:19",
				Region:     "fn:unsafeSummary",
				Provenance: plir.Provenance{Kind: plir.ProvenanceUnknown, Root: "rawOwner"},
				Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "rawOwner"},
				Borrow:     plir.BorrowMove,
				Escape:     plir.EscapeUnsafe,
			},
			{
				ID:         "param:rawDst",
				Kind:       plir.ValueParam,
				Type:       "[]u8",
				Source:     "test.tetra:2:39",
				Region:     "fn:unsafeSummary",
				Provenance: plir.Provenance{Kind: plir.ProvenanceExternal, Root: "rawDst"},
				Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "rawDst"},
				Borrow:     plir.BorrowMut,
				Escape:     plir.EscapeUnsafe,
			},
		},
		Facts: []plir.Fact{
			{ID: "f_moved_unknown", Kind: plir.FactMoved, ValueID: "param:rawOwner", Source: "test.tetra:3:5", Reason: "unknown pointer moved"},
			{ID: "f_mut_unknown", Kind: plir.FactBorrowedMut, ValueID: "param:rawDst", Source: "test.tetra:3:18", Reason: "unknown external inout"},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, claim := range []string{"may_consume_param", "may_mutate_inout"} {
		for _, row := range report.Rows {
			if row.Claim != claim {
				continue
			}
			if row.ProvenanceClass != ProvenanceUnsafeUnknown || row.UnsafeClass != UnsafeUnknown || row.ClaimLevel != ClaimConservative {
				t.Fatalf("%s row = %+v, want unsafe_unknown/conservative", claim, row)
			}
		}
	}
}

func TestFromPLIRAndAllocPlanSummaryFactIDsDoNotCollideForDistinctPaths(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "pathSummary",
		Summary: &plir.FunctionSummary{
			ParamNames: []string{"box"},
			ReturnRegionSummary: map[string]int{
				"a.b": 0,
				"a_b": 0,
			},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	if got := countReportClaim(BuildReportFromGraph(graph), "may_return_region"); got != 2 {
		t.Fatalf("may_return_region rows = %d, want 2", got)
	}
}

func TestFromPLIRAndAllocPlanEmitsUnsafeGatewayVocabulary(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "main",
		Values: []plir.Value{
			{
				ID:     "alloc_intent:p",
				Kind:   plir.ValueAllocIntent,
				Type:   "ptr",
				Source: "test.tetra:4:17",
				Region: "raw_allocation:p",
				Alloc: &plir.AllocIntent{
					ElementType:            "raw_bytes",
					ElementSize:            1,
					LengthExpr:             "16",
					LengthConstKnown:       true,
					LengthConst:            16,
					Builtin:                "core.alloc_bytes",
					Source:                 "test.tetra:4:17",
					RawPointerBoundsStatus: string(runtimeabi.RawPointerBoundsAllocationBase),
					RawPointerBaseID:       "p",
					RawPointerBaseBytes:    16,
					RawPointerOffsetBytes:  0,
				},
				Provenance: plir.Provenance{Kind: plir.ProvenanceAllocation, Root: "p"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:4:17", Owner: "p"},
				Escape:     plir.EscapeConservative,
			},
			{
				ID:         "view:raw",
				Kind:       plir.ValueView,
				Type:       "[]u8",
				Source:     "test.tetra:7:25",
				Region:     "external:raw",
				Provenance: plir.Provenance{Kind: plir.ProvenanceExternal, Root: "raw_parts"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:7:25", Owner: "raw"},
				Escape:     plir.EscapeConservative,
			},
		},
		Ops: []plir.Operation{
			{ID: "op_alloc", Kind: plir.OpAllocIntent, Source: "test.tetra:4:17", Inputs: []string{"16"}, Outputs: []string{"alloc_intent:p"}, Note: "alloc_bytes raw allocation-base metadata"},
			{ID: "op_ptr_verified", Kind: plir.OpUnsafe, Source: "test.tetra:5:17", Inputs: []string{"p", "4", "mem"}, Outputs: []string{"q"}, UnsafeClass: plir.UnsafeChecked, Note: "core.ptr_add raw_pointer_bounds: derived_allocation_offset base:p offset:4"},
			{ID: "op_ptr_unknown", Kind: plir.OpUnsafe, Source: "test.tetra:6:17", Inputs: []string{"external", "4", "mem"}, Outputs: []string{"r"}, UnsafeClass: plir.UnsafeUnknown, Note: "core.ptr_add raw_pointer_bounds: checked_external_unknown base:external offset:4"},
			{ID: "op_raw_load", Kind: plir.OpUnsafe, Source: "test.tetra:6:21", Inputs: []string{"q", "mem"}, UnsafeClass: plir.UnsafeChecked, Note: "core.load_u8 raw memory gateway: derived_allocation_offset pointer:q"},
			{ID: "op_raw_store_unknown", Kind: plir.OpUnsafe, Source: "test.tetra:6:41", Inputs: []string{"external", "0", "mem"}, UnsafeClass: plir.UnsafeUnknown, Note: "core.store_u8 raw memory gateway: checked_external_unknown pointer:external"},
			{ID: "op_ptr_negative", Kind: plir.OpUnsafe, Source: "test.tetra:8:17", Inputs: []string{"p", "-1", "mem"}, Outputs: []string{"neg"}, UnsafeClass: plir.UnsafeChecked, Note: "core.ptr_add raw_pointer_bounds: rejected_negative_offset base:p offset:-1"},
			{ID: "op_ptr_upper", Kind: plir.OpUnsafe, Source: "test.tetra:9:17", Inputs: []string{"p", "16", "mem"}, Outputs: []string{"upper"}, UnsafeClass: plir.UnsafeChecked, Note: "core.ptr_add raw_pointer_bounds: rejected_upper_bound base:p offset:16"},
			{ID: "op_raw_load_width", Kind: plir.OpUnsafe, Source: "test.tetra:10:21", Inputs: []string{"q", "mem"}, UnsafeClass: plir.UnsafeChecked, Note: "core.load_i32 raw memory gateway: rejected_access_width_overflow pointer:q offset:14 width:4"},
			{ID: "op_raw_slice", Kind: plir.OpUnsafe, Source: "test.tetra:7:25", Outputs: []string{"view:raw"}, UnsafeClass: plir.UnsafeUnknown, Note: "core.raw_slice_u8_from_parts creates a conservative external-provenance view"},
			{ID: "op_raw_slice_verified", Kind: plir.OpUnsafe, Source: "test.tetra:7:35", Outputs: []string{"view:raw_checked"}, UnsafeClass: plir.UnsafeChecked, Note: "core.raw_slice_u8_from_parts raw_slice_bounds: verified_allocation_root base:p length_bytes:4"},
			{ID: "op_raw_slice_negative", Kind: plir.OpUnsafe, Source: "test.tetra:8:35", Outputs: []string{"view:raw_negative"}, UnsafeClass: plir.UnsafeChecked, Note: "core.raw_slice_u8_from_parts raw_slice_bounds: rejected_negative_length base:p length:-1"},
			{ID: "op_raw_slice_length_overflow", Kind: plir.OpUnsafe, Source: "test.tetra:9:35", Outputs: []string{"view:raw_overflow"}, UnsafeClass: plir.UnsafeChecked, Note: "core.raw_slice_i32_from_parts raw_slice_bounds: rejected_length_overflow base:p length_bytes:overflow elem_size:4"},
		},
		Facts: []plir.Fact{
			{ID: "f_alloc_prov", Kind: plir.FactProvenanceKnown, ValueID: "alloc_intent:p", Source: "test.tetra:4:17", Reason: "core.alloc_bytes allocation-base metadata"},
			{ID: "f_alloc_region", Kind: plir.FactRegionAlive, ValueID: "alloc_intent:p", Region: "raw_allocation:p", Source: "test.tetra:4:17", Reason: "raw allocation root alive"},
			{ID: "f_raw_unknown", Kind: plir.FactProvenanceUnknown, ValueID: "view:raw", Source: "test.tetra:7:25", Reason: "raw slice gateway has external provenance"},
		},
	}}}
	plan := &allocplan.Plan{Functions: []allocplan.FunctionPlan{{
		Name: "main",
		Allocations: []allocplan.Allocation{{
			ID:                     "p",
			SiteID:                 "alloc:main:p",
			ValueID:                "alloc_intent:p",
			Source:                 "test.tetra:4:17",
			Builtin:                "core.alloc_bytes",
			ElementType:            "raw_bytes",
			PlannedStorage:         allocplan.StorageHeap,
			ActualLoweringStorage:  allocplan.StorageHeap,
			ValidationStatus:       "validated_heap",
			RawPointerBoundsStatus: string(runtimeabi.RawPointerBoundsAllocationBase),
			RawPointerBaseID:       "p",
			RawPointerBaseBytes:    16,
			Reason:                 "raw allocation root remains unsafe-origin",
		}},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, plan)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, tc := range []struct {
		claim      string
		provenance ProvenanceClass
		unsafe     UnsafeClass
		level      ClaimLevel
		cost       CostClass
	}{
		{claim: "allocation_base_metadata", provenance: ProvenanceUnsafeVerifiedRoot, unsafe: UnsafeVerifiedRoot, level: ClaimValidated, cost: CostZeroCostProven},
		{claim: "derived_allocation_offset", provenance: ProvenanceUnsafeChecked, unsafe: UnsafeChecked, level: ClaimEvidenceOnly, cost: CostDynamicCheckRequired},
		{claim: "checked_external_unknown", provenance: ProvenanceUnsafeUnknown, unsafe: UnsafeUnknown, level: ClaimConservative, cost: CostConservativeFallback},
		{claim: "raw_memory_access_checked", provenance: ProvenanceUnsafeChecked, unsafe: UnsafeChecked, level: ClaimEvidenceOnly, cost: CostDynamicCheckRequired},
		{claim: "raw_memory_access_unknown", provenance: ProvenanceUnsafeUnknown, unsafe: UnsafeUnknown, level: ClaimConservative, cost: CostConservativeFallback},
		{claim: "rejected_negative_offset", provenance: ProvenanceUnsafeChecked, unsafe: UnsafeChecked, level: ClaimEvidenceOnly, cost: CostUnsupportedRejected},
		{claim: "rejected_upper_bound", provenance: ProvenanceUnsafeChecked, unsafe: UnsafeChecked, level: ClaimEvidenceOnly, cost: CostUnsupportedRejected},
		{claim: "rejected_access_width_overflow", provenance: ProvenanceUnsafeChecked, unsafe: UnsafeChecked, level: ClaimEvidenceOnly, cost: CostUnsupportedRejected},
		{claim: "external_unknown", provenance: ProvenanceUnsafeUnknown, unsafe: UnsafeUnknown, level: ClaimConservative, cost: CostConservativeFallback},
		{claim: "raw_slice_verified_allocation_root", provenance: ProvenanceUnsafeChecked, unsafe: UnsafeChecked, level: ClaimEvidenceOnly, cost: CostDynamicCheckRequired},
		{claim: "rejected_negative_length", provenance: ProvenanceUnsafeChecked, unsafe: UnsafeChecked, level: ClaimEvidenceOnly, cost: CostUnsupportedRejected},
		{claim: "rejected_length_overflow", provenance: ProvenanceUnsafeChecked, unsafe: UnsafeChecked, level: ClaimEvidenceOnly, cost: CostUnsupportedRejected},
	} {
		row, ok := reportRowByClaim(report, tc.claim)
		if !ok {
			t.Fatalf("memory report missing unsafe gateway claim %q:\n%+v", tc.claim, report.Rows)
		}
		if row.ProvenanceClass != tc.provenance || row.UnsafeClass != tc.unsafe || row.ClaimLevel != tc.level || row.CostClass != tc.cost {
			t.Fatalf("%s row = %+v, want provenance %s unsafe %s level %s cost %s", tc.claim, row, tc.provenance, tc.unsafe, tc.level, tc.cost)
		}
		if row.CostClass == CostDynamicCheckRequired && !row.NormalBuildCheck {
			t.Fatalf("%s row = %+v, dynamic_check_required must keep normal_build_check", tc.claim, row)
		}
	}
	for _, row := range report.Rows {
		if row.ValueID == "alloc_intent:p" && (row.Claim == "provenance_known" || row.Claim == "region_alive") {
			t.Fatalf("core.alloc_bytes verified root emitted generic unsafe_verified_root row: %+v", row)
		}
	}
	rawCheck, ok := reportRowByClaim(report, "raw_bounds_runtime_check_normal_build")
	if !ok {
		t.Fatalf("memory report missing v6 raw bounds normal-build check row:\n%+v", report.Rows)
	}
	if rawCheck.ClaimLevel != ClaimValidated ||
		rawCheck.CostClass != CostDynamicCheckRequired ||
		!rawCheck.NormalBuildCheck ||
		rawCheck.ValidatorName != "raw_bounds_width_validator" ||
		rawCheck.ParentFactID == "" {
		t.Fatalf("raw bounds normal-build row = %+v, want validated dynamic check with parent fact", rawCheck)
	}
}

func TestMemoryIdealV5ProjectsRawPointerUnsafeContractFacts(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "rawUnsafeV5",
		Values: []plir.Value{
			{
				ID:     "alloc_intent:p",
				Kind:   plir.ValueAllocIntent,
				Type:   "ptr",
				Source: "test.tetra:4:17",
				Region: "raw_allocation:p",
				Alloc: &plir.AllocIntent{
					ElementType:            "raw_bytes",
					ElementSize:            1,
					LengthExpr:             "16",
					LengthConstKnown:       true,
					LengthConst:            16,
					Builtin:                "core.alloc_bytes",
					Source:                 "test.tetra:4:17",
					RawPointerBoundsStatus: string(runtimeabi.RawPointerBoundsAllocationBase),
					RawPointerBaseID:       "p",
					RawPointerBaseBytes:    16,
					RawPointerOffsetBytes:  0,
				},
				Provenance: plir.Provenance{Kind: plir.ProvenanceAllocation, Root: "p"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:4:17", Owner: "p"},
				Escape:     plir.EscapeConservative,
			},
			{
				ID:         "param:external",
				Kind:       plir.ValueParam,
				Type:       "ptr",
				Source:     "test.tetra:2:17",
				Region:     "external:raw",
				Provenance: plir.Provenance{Kind: plir.ProvenanceExternal, Root: "external"},
				Lifetime:   plir.Lifetime{Birth: "entry", Owner: "external"},
				Escape:     plir.EscapeUnsafe,
			},
		},
		Ops: []plir.Operation{
			{ID: "op_ptr_verified", Kind: plir.OpUnsafe, Source: "test.tetra:5:17", Inputs: []string{"p", "4", "mem"}, Outputs: []string{"q"}, UnsafeClass: plir.UnsafeChecked, Note: "core.ptr_add raw_pointer_bounds: derived_allocation_offset base:p offset:4"},
			{ID: "op_ptr_unknown", Kind: plir.OpUnsafe, Source: "test.tetra:6:17", Inputs: []string{"external", "4", "mem"}, Outputs: []string{"r"}, UnsafeClass: plir.UnsafeUnknown, Note: "core.ptr_add raw_pointer_bounds: checked_external_unknown base:external offset:4"},
			{ID: "op_runtime_contract", Kind: plir.OpUnsafe, Source: "test.tetra:7:17", Inputs: []string{"q", "mem"}, UnsafeClass: plir.UnsafeChecked, Note: "unsafe contract runtime_checkable: nonnull alignment length pointer:q"},
			{ID: "op_static_contract", Kind: plir.OpUnsafe, Source: "test.tetra:8:17", Inputs: []string{"external", "mem"}, UnsafeClass: plir.UnsafeUnknown, Note: "unsafe contract static_untrusted: noalias lifetime region pointer:external"},
		},
		Facts: []plir.Fact{
			{ID: "f_alloc_prov", Kind: plir.FactProvenanceKnown, ValueID: "alloc_intent:p", Source: "test.tetra:4:17", Reason: "core.alloc_bytes allocation-base metadata"},
			{ID: "f_external_unknown", Kind: plir.FactProvenanceUnknown, ValueID: "param:external", Source: "test.tetra:2:17", Reason: "external raw pointer remains unsafe_unknown"},
		},
	}}}
	plan := &allocplan.Plan{Functions: []allocplan.FunctionPlan{{
		Name: "rawUnsafeV5",
		Allocations: []allocplan.Allocation{{
			ID:                     "p",
			SiteID:                 "alloc:rawUnsafeV5:p",
			ValueID:                "alloc_intent:p",
			Source:                 "test.tetra:4:17",
			Builtin:                "core.alloc_bytes",
			ElementType:            "raw_bytes",
			PlannedStorage:         allocplan.StorageHeap,
			ActualLoweringStorage:  allocplan.StorageHeap,
			ValidationStatus:       "validated_heap",
			RawPointerBoundsStatus: string(runtimeabi.RawPointerBoundsAllocationBase),
			RawPointerBaseID:       "p",
			RawPointerBaseBytes:    16,
			Reason:                 "raw allocation root remains unsafe-origin",
		}},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, plan)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, tc := range []struct {
		claim      string
		validator  string
		provenance ProvenanceClass
		unsafe     UnsafeClass
		level      ClaimLevel
		cost       CostClass
	}{
		{claim: "unsafe_verified_root_allocation_base", validator: "unsafe_verified_root_bounds_validator", provenance: ProvenanceUnsafeVerifiedRoot, unsafe: UnsafeVerifiedRoot, level: ClaimValidated, cost: CostZeroCostProven},
		{claim: "unsafe_unknown_rejected_safe_facts", validator: "unsafe_unknown_fact_validator", provenance: ProvenanceUnsafeUnknown, unsafe: UnsafeUnknown, level: ClaimRejected, cost: CostUnsupportedRejected},
		{claim: "unsafe_contract_runtime_checkable", validator: "unsafe_runtime_contract_validator", provenance: ProvenanceUnsafeChecked, unsafe: UnsafeChecked, level: ClaimValidated, cost: CostDynamicCheckRequired},
		{claim: "unsafe_contract_static_untrusted", validator: "unsafe_static_contract_validator", provenance: ProvenanceUnsafeUnknown, unsafe: UnsafeUnknown, level: ClaimConservative, cost: CostConservativeFallback},
	} {
		row, ok := reportRowByClaim(report, tc.claim)
		if !ok {
			t.Fatalf("memory report missing v5 claim %q:\n%+v", tc.claim, report.Rows)
		}
		if row.ValidatorName != tc.validator || row.ProvenanceClass != tc.provenance || row.UnsafeClass != tc.unsafe || row.ClaimLevel != tc.level || row.CostClass != tc.cost {
			t.Fatalf("%s row = %+v, want validator %s provenance %s unsafe %s level %s cost %s", tc.claim, row, tc.validator, tc.provenance, tc.unsafe, tc.level, tc.cost)
		}
		if row.CostClass == CostDynamicCheckRequired && !row.NormalBuildCheck {
			t.Fatalf("%s row = %+v, dynamic_check_required must keep normal_build_check", tc.claim, row)
		}
	}
	staticRow, _ := reportRowByClaim(report, "unsafe_contract_static_untrusted")
	if staticRow.AliasState != AliasInvalidatedByCall {
		t.Fatalf("unsafe_contract_static_untrusted row = %+v, want invalidated_by_call alias state", staticRow)
	}
	for _, forbidden := range []string{"safe_known", "provenance_known", "no_alias"} {
		for _, row := range report.Rows {
			if row.UnsafeClass == UnsafeUnknown && row.Claim == forbidden {
				t.Fatalf("unsafe_unknown emitted forbidden safe/noalias claim %q: %+v", forbidden, row)
			}
		}
	}
}

func TestMemoryIdealV7ProjectsFFICallExternalFacts(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "ffiV7",
		Values: []plir.Value{
			{
				ID:         "param:external",
				Kind:       plir.ValueParam,
				Type:       "ptr",
				Source:     "test.tetra:2:17",
				Region:     "external:raw",
				Provenance: plir.Provenance{Kind: plir.ProvenanceExternal, Root: "external"},
				Lifetime:   plir.Lifetime{Birth: "entry", Owner: "external"},
				Escape:     plir.EscapeCallUnknown,
			},
			{
				ID:         "param:borrowed",
				Kind:       plir.ValueParam,
				Type:       "ptr",
				Source:     "test.tetra:2:34",
				Region:     "param:xs",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "xs"},
				Lifetime:   plir.Lifetime{Birth: "entry", Owner: "xs"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeCallUnknown,
			},
			{
				ID:         "param:dst",
				Kind:       plir.ValueParam,
				Type:       "ptr",
				Source:     "test.tetra:2:52",
				Region:     "param:dst",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "dst"},
				Lifetime:   plir.Lifetime{Birth: "entry", Owner: "dst"},
				Borrow:     plir.BorrowMut,
				Escape:     plir.EscapeCallUnknown,
			},
		},
		Ops: []plir.Operation{
			{ID: "op_ffi", Kind: plir.OpCall, Source: "test.tetra:4:5", Inputs: []string{"external", "borrowed", "dst"}, Note: "ffi.external call may retain borrowed pointer and invalidates noalias"},
			{ID: "op_safe_wrapper", Kind: plir.OpUnsafe, Source: "test.tetra:5:17", Inputs: []string{"external", "mem"}, Outputs: []string{"wrapped"}, UnsafeClass: plir.UnsafeUnknown, Note: "safe wrapper promotion from external pointer without compiler-owned contract"},
		},
		Facts: []plir.Fact{
			{ID: "f_external_unknown", Kind: plir.FactProvenanceUnknown, ValueID: "param:external", Source: "test.tetra:2:17", Reason: "external pointer remains unsafe_unknown"},
			{ID: "f_borrowed", Kind: plir.FactBorrowedImm, ValueID: "param:borrowed", Source: "test.tetra:2:34", Reason: "borrowed pointer argument"},
			{ID: "f_noalias", Kind: plir.FactNoAlias, ValueID: "param:dst", Source: "test.tetra:2:52", Reason: "exclusive pointer before external call"},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, tc := range []struct {
		claim     string
		validator string
		level     ClaimLevel
		cost      CostClass
		alias     AliasState
		parent    bool
	}{
		{claim: "ffi_pointer_external_unknown", validator: "external_pointer_provenance_validator", level: ClaimConservative, cost: CostConservativeFallback},
		{claim: "ffi_call_may_retain_borrow", validator: "ffi_lifetime_conservative_validator", level: ClaimConservative, cost: CostConservativeFallback, parent: true},
		{claim: "ffi_noalias_invalidated_by_external_call", validator: "ffi_noalias_conservative_validator", level: ClaimConservative, cost: CostConservativeFallback, alias: AliasInvalidatedByCall, parent: true},
		{claim: "safe_wrapper_promotion_rejected_without_contract", validator: "safe_wrapper_promotion_validator", level: ClaimRejected, cost: CostUnsupportedRejected, parent: true},
		{claim: "external_pointer_provenance_rejected", validator: "external_pointer_provenance_validator", level: ClaimRejected, cost: CostUnsupportedRejected, parent: true},
	} {
		row, ok := reportRowByClaim(report, tc.claim)
		if !ok {
			t.Fatalf("memory report missing v7 claim %q:\n%+v", tc.claim, report.Rows)
		}
		if row.ValidatorName != tc.validator ||
			row.ProvenanceClass != ProvenanceUnsafeUnknown ||
			row.UnsafeClass != UnsafeUnknown ||
			row.ClaimLevel != tc.level ||
			row.CostClass != tc.cost {
			t.Fatalf("%s row = %+v, want validator %s unsafe_unknown level %s cost %s", tc.claim, row, tc.validator, tc.level, tc.cost)
		}
		if tc.alias != "" && row.AliasState != tc.alias {
			t.Fatalf("%s row = %+v, want alias %s", tc.claim, row, tc.alias)
		}
		if tc.parent && row.ParentFactID == "" {
			t.Fatalf("%s row = %+v, want parent_fact_id", tc.claim, row)
		}
	}
	for _, forbidden := range []string{"safe_known", "provenance_known", "no_alias", "bounds_check_eliminated", "index_in_range"} {
		for _, row := range report.Rows {
			if row.UnsafeClass == UnsafeUnknown && row.Claim == forbidden {
				t.Fatalf("external/unsafe pointer emitted forbidden claim %q: %+v", forbidden, row)
			}
		}
	}
}

func TestFromPLIRAndAllocPlanRejectsFactForMissingValue(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name:  "main",
		Facts: []plir.Fact{{ID: "f_missing", Kind: plir.FactBorrowedImm, ValueID: "view:missing"}},
	}}}

	_, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err == nil || !strings.Contains(err.Error(), "missing value_id") {
		t.Fatalf("FromPLIRAndAllocPlan error = %v, want missing value_id rejection", err)
	}
}

func reportHasClaim(report Report, claim string) bool {
	for _, row := range report.Rows {
		if row.Claim == claim {
			return true
		}
	}
	return false
}

func reportRowByClaim(report Report, claim string) (ReportRow, bool) {
	for _, row := range report.Rows {
		if row.Claim == claim {
			return row, true
		}
	}
	return ReportRow{}, false
}

func reportHasOwner(report Report, claim string, owner string) bool {
	for _, row := range report.Rows {
		if row.Claim == claim && row.OwnerID == owner {
			return true
		}
	}
	return false
}

func reportHasAliasState(report Report, claim string, state AliasState) bool {
	for _, row := range report.Rows {
		if row.Claim == claim && row.AliasState == state {
			return true
		}
	}
	return false
}

func countReportClaim(report Report, claim string) int {
	count := 0
	for _, row := range report.Rows {
		if row.Claim == claim {
			count++
		}
	}
	return count
}
