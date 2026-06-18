package memoryfacts_test

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/allocplan"
	. "tetra_language/compiler/internal/memoryfacts"
	. "tetra_language/compiler/internal/memoryfacts/fromplir"
	"tetra_language/compiler/internal/plir"
)

func TestFromPLIRAndAllocPlanProjectsRepresentationMetadataFact(t *testing.T) {
	graph, err := FromPLIRAndAllocPlan("program", nil, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan representation metadata: %v", err)
	}
	report := BuildReportFromGraph(graph)
	row, ok := reportRowByClaim(report, "safe_representation_metadata: not_user_assignable")
	if !ok {
		t.Fatalf("report missing safe representation metadata row: %#v", report.Rows)
	}
	if row.SourceFactID != "semantics:representation-metadata:not-user-assignable" ||
		row.SourceStage != StageSemantics ||
		row.ProvenanceClass != ProvenanceSafeKnown ||
		row.UnsafeClass != UnsafeSafe ||
		row.ValidatorName != "representation_namespace_validator" ||
		row.ValidatorStatus != ValidatorPass {
		t.Fatalf("representation metadata projection row = %#v", row)
	}
	if err := ValidateReportProjection(graph, report); err != nil {
		t.Fatalf("ValidateReportProjection representation metadata: %v", err)
	}
}

func TestFromPLIRAndAllocPlanProjectsIslandMemoryRefFields(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "main",
		Values: []plir.Value{{
			ID:         "alloc_intent:xs",
			Kind:       plir.ValueAllocIntent,
			Type:       "[]u8",
			Source:     "main.tetra:4:23",
			Region:     "island:isl",
			Provenance: plir.Provenance{Kind: plir.ProvenanceIsland, Root: "isl"},
			Lifetime:   plir.Lifetime{Birth: "main.tetra:4:23", Owner: "xs"},
		}},
		Facts: []plir.Fact{{
			ID:       "f_island_known",
			Kind:     plir.FactProvenanceKnown,
			ValueID:  "alloc_intent:xs",
			IslandID: "island:isl",
			Epoch:    1,
			BaseID:   "alloc_intent:xs",
			Source:   "main.tetra:4:23",
			Reason:   "island allocation provenance",
		}},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	row, ok := reportRowByClaim(report, "provenance_known")
	if !ok {
		t.Fatalf("report missing provenance_known row: %#v", report.Rows)
	}
	if row.IslandID != "island:isl" || row.Epoch != 1 || row.BaseID != "alloc_intent:xs" {
		t.Fatalf(
			"island memory ref fields = island_id:%q epoch:%d base_id:%q",
			row.IslandID,
			row.Epoch,
			row.BaseID,
		)
	}
	if err := ValidateReportProjection(graph, report); err != nil {
		t.Fatalf("ValidateReportProjection island memory ref: %v", err)
	}
}

func TestFromPLIRAndAllocPlanProjectsIslandEpochAdvancedFact(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "main",
		Facts: []plir.Fact{{
			ID:       "f_epoch",
			Kind:     plir.FactIslandEpochAdvanced,
			IslandID: "island:isl",
			Epoch:    2,
			BaseID:   "token:isl",
			Source:   "main.tetra:5:21",
			Reason:   "island reset advances epoch and invalidates previous references",
		}},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	row, ok := reportRowByClaim(report, "island_epoch_advanced")
	if !ok {
		t.Fatalf("report missing island_epoch_advanced row: %#v", report.Rows)
	}
	if row.IslandID != "island:isl" || row.Epoch != 2 || row.BaseID != "token:isl" {
		t.Fatalf(
			"island epoch advancement fields = island_id:%q epoch:%d base_id:%q",
			row.IslandID,
			row.Epoch,
			row.BaseID,
		)
	}
	if err := ValidateReportProjection(graph, report); err != nil {
		t.Fatalf("ValidateReportProjection island epoch advancement: %v", err)
	}
}

func TestFromPLIRAndAllocPlanRejectsModuleBoundaryMissingFunctionSummary(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name:   "externBridge",
		Module: "ffi",
		Values: []plir.Value{{
			ID:         "param:p",
			Kind:       plir.ValueParam,
			Type:       "ptr",
			Source:     "ffi.tetra:2:19",
			Region:     "fn:externBridge",
			Provenance: plir.Provenance{Kind: plir.ProvenanceUnknown, Root: "p"},
			Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "p"},
			Escape:     plir.EscapeUnsafe,
		}},
		Ops: []plir.Operation{
			{
				ID:     "extern_call",
				Kind:   plir.OpCall,
				Source: "ffi.tetra:3:5",
				Inputs: []string{"p"},
				Note:   "ffi.unknown external call",
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "unknown_p",
				Kind:    plir.FactProvenanceUnknown,
				ValueID: "param:p",
				Source:  "ffi.tetra:2:19",
				Reason:  "external pointer provenance unknown",
			},
		},
	}}}

	_, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err == nil || !strings.Contains(err.Error(), "summary completeness") ||
		!strings.Contains(err.Error(), "FunctionSummary") {
		t.Fatalf(
			"FromPLIRAndAllocPlan error = %v, want missing FunctionSummary completeness rejection",
			err,
		)
	}
}

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
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:4:28",
					Death: "return",
					Owner: "borrowed",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeNoEscape,
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
			{
				ID:      "op_borrow",
				Kind:    plir.OpCall,
				Source:  "test.tetra:4:28",
				Inputs:  []string{"xs"},
				Outputs: []string{"view:borrowed"},
				Note:    "core.slice_borrow_u8 creates borrowed view without allocation",
			},
			{
				ID:      "op_copy",
				Kind:    plir.OpAllocIntent,
				Source:  "test.tetra:5:27",
				Inputs:  []string{"borrowed"},
				Outputs: []string{"alloc_intent:copied"},
				Note:    "core.slice_copy_u8 creates owned copy with new provenance",
			},
			{
				ID:     "op_copy_into",
				Kind:   plir.OpCall,
				Source: "test.tetra:6:31",
				Inputs: []string{"copied", "dst"},
				Note: ("core.slice_copy_into_u8 copies into caller-owned destination " +
					"without allocation source:copied destination:dst dest_capacity_" +
					"check:normal_build overlap:distinct_roots"),
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "f_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:borrowed",
				Source:  "test.tetra:4:28",
				Reason:  "explicit borrow view",
			},
			{
				ID:      "f_no_escape",
				Kind:    plir.FactNoEscape,
				ValueID: "view:borrowed",
				Source:  "test.tetra:4:28",
				Reason:  "explicit borrowed view may not escape owner",
			},
			{
				ID:      "f_owned",
				Kind:    plir.FactOwned,
				ValueID: "alloc_intent:copied",
				Source:  "test.tetra:5:27",
				Reason:  "copy result owns new storage",
			},
			{
				ID:      "f_prov",
				Kind:    plir.FactProvenanceKnown,
				ValueID: "alloc_intent:copied",
				Source:  "test.tetra:5:27",
				Reason:  "copy creates owned value with new provenance",
			},
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

func TestMemoryIdealB03ProjectsCopyIntoOverlapAndCapacityFacts(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "copyIntoOverlap",
		Values: []plir.Value{
			{
				ID:         "view:src",
				Kind:       plir.ValueView,
				Type:       "[]u8",
				Source:     "test.tetra:4:18",
				Region:     "fn:copyIntoOverlap",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "derived:xs"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:4:18", Owner: "src"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeNoEscape,
			},
			{
				ID:         "view:dst",
				Kind:       plir.ValueView,
				Type:       "[]u8",
				Source:     "test.tetra:5:18",
				Region:     "fn:copyIntoOverlap",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "derived:xs"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:5:18", Owner: "dst"},
				Borrow:     plir.BorrowMut,
				Escape:     plir.EscapeNoEscape,
			},
		},
		Ops: []plir.Operation{
			{
				ID:     "op_copy_into_overlap",
				Kind:   plir.OpCall,
				Source: "test.tetra:6:12",
				Inputs: []string{"src", "dst"},
				Note: ("core.slice_copy_into_u8 copies into caller-owned destination " +
					"without allocation source:src destination:dst dest_capacity_" +
					"check:normal_build overlap:known_overlap"),
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "f_src_window",
				Kind:    plir.FactDerivedWindow,
				ValueID: "view:src",
				Range:   "xs[0..3]",
				Source:  "test.tetra:4:18",
				Reason:  "safe slice view range is checked before construction",
			},
			{
				ID:      "f_dst_window",
				Kind:    plir.FactDerivedWindow,
				ValueID: "view:dst",
				Range:   "xs[1..4]",
				Source:  "test.tetra:5:18",
				Reason:  "safe slice view range is checked before construction",
			},
			{
				ID:      "f_src_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:src",
				Source:  "test.tetra:4:18",
				Reason:  "source view borrowed",
			},
			{
				ID:      "f_dst_borrow",
				Kind:    plir.FactBorrowedMut,
				ValueID: "view:dst",
				Source:  "test.tetra:5:18",
				Reason:  "destination view borrowed mutably",
			},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	lengthRow, ok := reportRowByClaim(report, "copy_into_destination_length_check")
	if !ok {
		t.Fatalf("memory report missing copy_into destination length check:\n%+v", report.Rows)
	}
	if lengthRow.CostClass != CostDynamicCheckRequired || !lengthRow.NormalBuildCheck ||
		lengthRow.ParentFactID == "" {
		t.Fatalf("copy_into length row = %+v, want parented normal-build dynamic check", lengthRow)
	}
	overlapRow, ok := reportRowByClaim(report, "copy_into_overlap_rejected")
	if !ok {
		t.Fatalf("memory report missing copy_into overlap rejection:\n%+v", report.Rows)
	}
	if overlapRow.ClaimLevel != ClaimRejected || overlapRow.CostClass != CostUnsupportedRejected ||
		overlapRow.ParentFactID == "" {
		t.Fatalf("copy_into overlap row = %+v, want parented rejected unsupported fact", overlapRow)
	}
	for _, row := range report.Rows {
		if row.SourceSpan == "test.tetra:6:12" && row.Claim == "copy_into_destination_fact_id" {
			t.Fatalf("overlapping copy_into emitted zero-cost destination fact: %+v", row)
		}
		if row.SourceSpan == "test.tetra:6:12" && strings.Contains(row.Claim, "no_alias") {
			t.Fatalf("overlapping copy_into emitted noalias claim: %+v", row)
		}
	}
}

func TestMemoryIdealB03CopyIntoDistinctOwnedAndUnknownBuffers(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "copyIntoB03",
		Values: []plir.Value{
			{
				ID:         "alloc_intent:src",
				Kind:       plir.ValueAllocIntent,
				Type:       "[]u8",
				Source:     "test.tetra:3:17",
				Region:     "allocation:src",
				Provenance: plir.Provenance{Kind: plir.ProvenanceAllocation, Root: "src"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:3:17", Owner: "src"},
				Escape:     plir.EscapeNoEscape,
			},
			{
				ID:         "alloc_intent:dst",
				Kind:       plir.ValueAllocIntent,
				Type:       "[]u8",
				Source:     "test.tetra:4:17",
				Region:     "allocation:dst",
				Provenance: plir.Provenance{Kind: plir.ProvenanceAllocation, Root: "dst"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:4:17", Owner: "dst"},
				Escape:     plir.EscapeNoEscape,
			},
			{
				ID:          "view:raw",
				Kind:        plir.ValueView,
				Type:        "[]u8",
				Source:      "test.tetra:8:17",
				Region:      "external:raw",
				Provenance:  plir.Provenance{Kind: plir.ProvenanceExternal, Root: "raw_parts"},
				UnsafeClass: plir.UnsafeUnknown,
				Lifetime:    plir.Lifetime{Birth: "test.tetra:8:17", Owner: "raw"},
				Escape:      plir.EscapeConservative,
			},
		},
		Ops: []plir.Operation{
			{
				ID:     "op_copy_into_distinct",
				Kind:   plir.OpCall,
				Source: "test.tetra:5:12",
				Inputs: []string{"src", "dst"},
				Note: ("core.slice_copy_into_u8 copies into caller-owned destination " +
					"without allocation source:src destination:dst dest_capacity_" +
					"check:normal_build overlap:distinct_roots"),
			},
			{
				ID:     "op_copy_into_unknown",
				Kind:   plir.OpCall,
				Source: "test.tetra:9:12",
				Inputs: []string{"raw", "dst"},
				Note: ("core.slice_copy_into_u8 copies into caller-owned destination " +
					"without allocation source:raw destination:dst dest_capacity_" +
					"check:normal_build overlap:unknown_conservative"),
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "f_src_owned",
				Kind:    plir.FactOwned,
				ValueID: "alloc_intent:src",
				Source:  "test.tetra:3:17",
				Reason:  "source owns allocation",
			},
			{
				ID:      "f_dst_owned",
				Kind:    plir.FactOwned,
				ValueID: "alloc_intent:dst",
				Source:  "test.tetra:4:17",
				Reason:  "destination owns allocation",
			},
			{
				ID:      "f_raw_unknown",
				Kind:    plir.FactProvenanceUnknown,
				ValueID: "view:raw",
				Source:  "test.tetra:8:17",
				Reason:  "raw slice gateway has external provenance",
			},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	distinctRow, ok := reportRowByClaimAndSource(
		report,
		"copy_into_destination_fact_id",
		"test.tetra:5:12",
	)
	if !ok {
		t.Fatalf("distinct owned copy_into missing destination fact:\n%+v", report.Rows)
	}
	if distinctRow.CostClass != CostZeroCostProven || distinctRow.ParentFactID == "" {
		t.Fatalf(
			"distinct copy_into destination row = %+v, want parented zero-cost destination fact",
			distinctRow,
		)
	}
	unknownRow, ok := reportRowByClaimAndSource(
		report,
		"copy_into_overlap_conservative",
		"test.tetra:9:12",
	)
	if !ok {
		t.Fatalf("unknown copy_into missing conservative overlap row:\n%+v", report.Rows)
	}
	if unknownRow.ClaimLevel != ClaimConservative ||
		unknownRow.CostClass != CostConservativeFallback ||
		unknownRow.ParentFactID == "" {
		t.Fatalf("unknown copy_into row = %+v, want parented conservative fallback", unknownRow)
	}
	if _, ok := reportRowByClaimAndSource(
		report,
		"copy_into_destination_fact_id",
		"test.tetra:9:12",
	); ok {
		t.Fatalf("unknown copy_into emitted destination zero-cost fact:\n%+v", report.Rows)
	}
	for _, row := range report.Rows {
		if strings.Contains(row.Claim, "no_alias") {
			t.Fatalf("copy_into B03 emitted broad noalias claim: %+v", row)
		}
	}
}

func TestFromPLIRAndAllocPlanProjectsSliceViewDynamicBoundsChecks(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "main",
		Values: []plir.Value{
			{
				ID:         "param:xs",
				Kind:       plir.ValueParam,
				Type:       "[]u16",
				Source:     "test.tetra:2:11",
				Region:     "fn:main",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "xs"},
				Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "xs"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeNoEscape,
			},
			{
				ID:         "view:mid",
				Kind:       plir.ValueView,
				Type:       "[]u16",
				Source:     "test.tetra:4:22",
				Region:     "fn:main",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "derived:xs"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:4:22", Death: "return", Owner: "mid"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeNoEscape,
			},
			{
				ID:         "view:text",
				Kind:       plir.ValueView,
				Type:       "str",
				Source:     "test.tetra:5:27",
				Region:     "fn:main",
				Provenance: plir.Provenance{Kind: plir.ProvenanceLiteral, Root: "string:abcdef"},
				Lifetime:   plir.Lifetime{Birth: "test.tetra:5:27", Death: "return", Owner: "text"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeNoEscape,
			},
		},
		Ops: []plir.Operation{
			{
				ID:      "op_window_u16",
				Kind:    plir.OpSliceWindow,
				Source:  "test.tetra:4:22",
				Inputs:  []string{"xs", "1", "2"},
				Outputs: []string{"view:mid"},
				Note: ("core.slice_window_u16 range xs[1..3] elem_width:2 elem_shift:1 " +
					"bounds_check:normal_build"),
			},
			{
				ID:      "op_string_prefix",
				Kind:    plir.OpSliceWindow,
				Source:  "test.tetra:5:27",
				Inputs:  []string{"text", "3"},
				Outputs: []string{"view:text"},
				Note: ("core.string_prefix range text[0..3] elem_width:1 elem_shift:0 " +
					"bounds_check:normal_build"),
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "f_mid_window",
				Kind:    plir.FactDerivedWindow,
				ValueID: "view:mid",
				Range:   "xs[1..3]",
				Source:  "test.tetra:4:22",
				Reason:  "safe slice view range is checked before construction",
			},
			{
				ID:      "f_mid_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:mid",
				Source:  "test.tetra:4:22",
				Reason:  "safe slice view",
			},
			{
				ID:      "f_mid_region",
				Kind:    plir.FactRegionAlive,
				ValueID: "view:mid",
				Source:  "test.tetra:4:22",
				Region:  "fn:main",
			},
			{
				ID:      "f_mid_no_escape",
				Kind:    plir.FactNoEscape,
				ValueID: "view:mid",
				Source:  "test.tetra:4:22",
				Reason:  "slice view may not escape its owner",
			},
			{
				ID:      "f_text_window",
				Kind:    plir.FactDerivedWindow,
				ValueID: "view:text",
				Range:   "text[0..3]",
				Source:  "test.tetra:5:27",
				Reason:  "safe String view range is checked before construction",
			},
			{
				ID:      "f_text_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:text",
				Source:  "test.tetra:5:27",
				Reason:  "safe String view",
			},
			{
				ID:      "f_text_region",
				Kind:    plir.FactRegionAlive,
				ValueID: "view:text",
				Source:  "test.tetra:5:27",
				Region:  "fn:main",
			},
			{
				ID:      "f_text_no_escape",
				Kind:    plir.FactNoEscape,
				ValueID: "view:text",
				Source:  "test.tetra:5:27",
				Reason:  "String view may not escape its owner",
			},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	if got := countReportClaim(report, "bounds_check_retained_dynamic"); got != 2 {
		t.Fatalf("bounds_check_retained_dynamic rows = %d, want 2:\n%+v", got, report.Rows)
	}
	for _, row := range report.Rows {
		if row.Claim != "bounds_check_retained_dynamic" {
			continue
		}
		if row.ParentFactID == "" ||
			row.CostClass != CostDynamicCheckRequired ||
			!row.NormalBuildCheck ||
			row.ValidatorName != "safe_view_bounds_validator" ||
			row.ValidatorStatus != ValidatorPass ||
			row.ProvenanceClass != ProvenanceSafeBorrowed ||
			row.UnsafeClass != UnsafeSafe {
			t.Fatalf("safe view retained-bounds row = %+v", row)
		}
		for _, want := range []string{"elem_width:", "elem_shift:", "normal-build"} {
			if !strings.Contains(row.Reason, want) {
				t.Fatalf("safe view retained-bounds reason %q missing %q", row.Reason, want)
			}
		}
	}
	if err := ValidateReportProjection(graph, report); err != nil {
		t.Fatalf("ValidateReportProjection: %v", err)
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
		if row.PlannedStorage != StorageStack || row.ActualLoweringStorage != StorageHeap ||
			row.LoweredArtifactID == "" {
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
		if row.PlannedStorage != StorageFunctionTempRegion ||
			row.ActualLoweringStorage != StorageHeap ||
			row.LoweredArtifactID == "" {
			t.Fatalf("FunctionTempRegion heap-fallback row lost storage truth fields: %+v", row)
		}
		return
	}
	t.Fatalf("missing allocplan FunctionTempRegion heap-fallback row: %+v", report.Rows)
}

func TestFromPLIRAndAllocPlanKeepsTaskActorRegionStorageEvidenceOnly(t *testing.T) {
	tests := []struct {
		name              string
		storage           allocplan.StorageClass
		validationStatus  string
		loweringStatus    string
		wantReportStorage StorageClass
	}{
		{
			name:              "task_region",
			storage:           allocplan.StorageTaskRegion,
			validationStatus:  "validated_task_region_scope",
			loweringStatus:    "task_region_lowering",
			wantReportStorage: StorageTaskRegion,
		},
		{
			name:              "actor_move_region",
			storage:           allocplan.StorageActorMoveRegion,
			validationStatus:  "validated_actor_move_region_scope",
			loweringStatus:    "actor_move_region_lowering",
			wantReportStorage: StorageActorMoveRegion,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &allocplan.Plan{Functions: []allocplan.FunctionPlan{{
				Name: "main",
				Allocations: []allocplan.Allocation{{
					ID:                    "payload",
					SiteID:                "alloc:main:payload",
					ValueID:               "alloc_intent:payload",
					Source:                "test.tetra:7:17",
					Builtin:               "core.make_u8",
					ElementType:           "u8",
					PlannedStorage:        tt.storage,
					ActualLoweringStorage: tt.storage,
					Reason: ("boundary storage remains evidence-only until production " +
						"runtime validation exists"),
					ValidationStatus: tt.validationStatus,
					LoweringStatus:   tt.loweringStatus,
				}},
			}}}

			graph, err := FromPLIRAndAllocPlan("program", nil, plan)
			if err != nil {
				t.Fatalf("FromPLIRAndAllocPlan: %v", err)
			}
			report := BuildReportFromGraph(graph)
			for _, row := range report.Rows {
				if row.SourceFactID != "allocplan:main:payload" {
					continue
				}
				if row.ClaimLevel == ClaimValidated || row.ValidatorStatus == ValidatorPass {
					t.Fatalf(
						"%s row was validated without production runtime proof: %+v",
						tt.name,
						row,
					)
				}
				if row.PlannedStorage != tt.wantReportStorage ||
					row.ActualLoweringStorage != tt.wantReportStorage ||
					row.LoweredArtifactID == "" {
					t.Fatalf("%s row lost storage truth fields: %+v", tt.name, row)
				}
				return
			}
			t.Fatalf("missing allocplan %s row: %+v", tt.name, report.Rows)
		})
	}
}

func TestFromPLIRAndAllocPlanKeepsUnvalidatedAllocPlanCostsNonZero(t *testing.T) {
	plan := &allocplan.Plan{Functions: []allocplan.FunctionPlan{{
		Name: "main",
		Allocations: []allocplan.Allocation{
			{
				ID:                    "stack-temp",
				SiteID:                "alloc:main:stack-temp",
				ValueID:               "alloc_intent:stack_temp",
				Source:                "test.tetra:5:17",
				Builtin:               "core.make_u8",
				ElementType:           "u8",
				PlannedStorage:        allocplan.StorageStack,
				ActualLoweringStorage: allocplan.StorageStack,
				Reason:                "unvalidated allocation lowering evidence fixture",
			},
			{
				ID:                    "raw-root",
				SiteID:                "alloc:main:raw-root",
				ValueID:               "alloc_intent:raw_root",
				Source:                "test.tetra:6:17",
				Builtin:               "core.alloc_bytes",
				ElementType:           "u8",
				PlannedStorage:        allocplan.StorageHeap,
				ActualLoweringStorage: allocplan.StorageHeap,
				Reason:                "unvalidated raw allocation metadata fixture",
			},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", nil, plan)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, sourceFactID := range []FactID{"allocplan:main:raw-root", "allocplan:main:stack-temp"} {
		var row ReportRow
		found := false
		for _, candidate := range report.Rows {
			if candidate.SourceFactID == sourceFactID {
				row = candidate
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing allocplan row %q:\n%+v", sourceFactID, report.Rows)
		}
		if row.ClaimLevel == ClaimValidated || row.ValidatorStatus == ValidatorPass {
			t.Fatalf("unvalidated allocplan row %q was validated: %+v", sourceFactID, row)
		}
		if row.CostClass == CostZeroCostProven {
			t.Fatalf("unvalidated allocplan row %q claimed zero cost: %+v", sourceFactID, row)
		}
	}
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
			{
				ID:      "f_unknown",
				Kind:    plir.FactProvenanceUnknown,
				ValueID: "view:raw",
				Source:  "test.tetra:3:17",
				Reason:  "borrow source provenance is external or unknown",
			},
			{
				ID:      "f_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:raw",
				Source:  "test.tetra:3:17",
				Reason:  "explicit borrow view",
			},
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
		if row.Claim == "borrowed_imm" &&
			(row.ProvenanceClass != ProvenanceUnsafeUnknown || row.ClaimLevel != ClaimConservative) {
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
			{
				ID:      "f_borrow_mut",
				Kind:    plir.FactBorrowedMut,
				ValueID: "param:xs",
				Source:  "test.tetra:2:13",
				Reason:  "inout parameter",
			},
			{
				ID:      "f_region",
				Kind:    plir.FactRegionAlive,
				ValueID: "param:xs",
				Region:  "fn:mutate",
				Source:  "test.tetra:2:13",
				Reason:  "function region alive",
			},
			{
				ID:      "f_prov",
				Kind:    plir.FactProvenanceKnown,
				ValueID: "param:xs",
				Source:  "test.tetra:2:13",
				Reason:  "parameter provenance",
			},
			{
				ID:      "f_no_alias",
				Kind:    plir.FactNoAlias,
				ValueID: "param:xs",
				Region:  "fn:mutate",
				Source:  "test.tetra:2:13",
				Reason:  "inout parameter has exclusive mutable access for call duration",
			},
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
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:5:17",
					Death: "return",
					Owner: "xs.view",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeNoEscape,
			},
			{
				ID:     "view:maybe.payload",
				Kind:   plir.ValueView,
				Type:   "str",
				Source: "test.tetra:9:17",
				Region: "fn:borrowAggregate",
				Provenance: plir.Provenance{
					Kind: plir.ProvenanceParam,
					Root: "derived:text.payload",
				},
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:9:17",
					Death: "return",
					Owner: "text.payload",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeNoEscape,
			},
		},
		Ops: []plir.Operation{
			{
				ID:      "op_struct",
				Kind:    plir.OpAggregate,
				Source:  "test.tetra:5:17",
				Inputs:  []string{"xs"},
				Outputs: []string{"view:holder.view"},
				Note:    "struct field carries borrowed view",
			},
			{
				ID:      "op_optional",
				Kind:    plir.OpAggregate,
				Source:  "test.tetra:9:17",
				Inputs:  []string{"text"},
				Outputs: []string{"view:maybe.payload"},
				Note:    "optional payload carries borrowed String view",
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "f_struct_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:holder.view",
				Source:  "test.tetra:5:17",
				Reason:  "borrow through struct field",
			},
			{
				ID:      "f_optional_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:maybe.payload",
				Source:  "test.tetra:9:17",
				Reason:  "borrow through optional payload",
			},
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
		if row.ParentFactID == "" || row.OwnerID == "" ||
			row.ValidatorName != "borrow_aggregate_escape_validator" ||
			row.ClaimLevel != ClaimValidated {
			t.Fatalf("%s row = %+v, want validated row with parent fact and owner", want, row)
		}
	}
}

func TestMemoryIdealV1ProjectsEnumPayloadAndGenericWrapperFacts(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "borrowCarrierV1",
		Values: []plir.Value{
			{
				ID:     "view:msg.payload",
				Kind:   plir.ValueView,
				Type:   "[]u8",
				Source: "test.tetra:5:17",
				Region: "fn:borrowCarrierV1",
				Provenance: plir.Provenance{
					Kind: plir.ProvenanceParam,
					Root: "derived:xs.enum_payload",
				},
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:5:17",
					Death: "return",
					Owner: "xs.enum_payload",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeNoEscape,
			},
			{
				ID:     "view:box.value",
				Kind:   plir.ValueView,
				Type:   "[]u8",
				Source: "test.tetra:9:17",
				Region: "fn:borrowCarrierV1",
				Provenance: plir.Provenance{
					Kind: plir.ProvenanceParam,
					Root: "derived:ys.generic_wrapper",
				},
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:9:17",
					Death: "return",
					Owner: "ys.generic_wrapper.value",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeNoEscape,
			},
		},
		Ops: []plir.Operation{
			{
				ID:      "op_enum",
				Kind:    plir.OpAggregate,
				Source:  "test.tetra:5:17",
				Inputs:  []string{"xs"},
				Outputs: []string{"view:msg.payload"},
				Note:    "enum payload contains borrowed view",
			},
			{
				ID:      "op_generic",
				Kind:    plir.OpAggregate,
				Source:  "test.tetra:9:17",
				Inputs:  []string{"ys"},
				Outputs: []string{"view:box.value"},
				Note:    "monomorphized generic wrapper Box<[]u8>.value contains borrowed view",
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "f_enum_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:msg.payload",
				Source:  "test.tetra:5:17",
				Reason:  "borrow through enum payload",
			},
			{
				ID:      "f_generic_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:box.value",
				Source:  "test.tetra:9:17",
				Reason:  "borrow through monomorphized generic wrapper",
			},
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
		if row.ParentFactID == "" || row.OwnerID == "" ||
			row.ValidatorName != "borrow_aggregate_escape_validator" ||
			row.ClaimLevel != ClaimValidated {
			t.Fatalf("%s row = %+v, want validated row with parent fact and owner", want, row)
		}
	}
}

func TestMemoryIdealV2ProjectsFunctionValueAndCallbackFacts(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "borrowCarrierV2",
		Values: []plir.Value{
			{
				ID:     "view:fn_value.arg",
				Kind:   plir.ValueView,
				Type:   "[]u8",
				Source: "test.tetra:5:17",
				Region: "fn:borrowCarrierV2",
				Provenance: plir.Provenance{
					Kind: plir.ProvenanceParam,
					Root: "derived:xs.function_value",
				},
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:5:17",
					Death: "return",
					Owner: "xs.function_value",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeNoEscape,
			},
			{
				ID:     "view:callback.arg",
				Kind:   plir.ValueView,
				Type:   "[]u8",
				Source: "test.tetra:9:17",
				Region: "fn:borrowCarrierV2",
				Provenance: plir.Provenance{
					Kind: plir.ProvenanceParam,
					Root: "derived:ys.callback_arg",
				},
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:9:17",
					Death: "return",
					Owner: "ys.callback_arg",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeNoEscape,
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
			{
				ID:      "op_function_value",
				Kind:    plir.OpCall,
				Source:  "test.tetra:5:17",
				Inputs:  []string{"xs"},
				Outputs: []string{"view:fn_value.arg"},
				Note:    "function-typed value contains borrowed view argument",
			},
			{
				ID:      "op_callback_arg",
				Kind:    plir.OpCall,
				Source:  "test.tetra:9:17",
				Inputs:  []string{"ys"},
				Outputs: []string{"view:callback.arg"},
				Note:    "known direct callback parameter contains borrowed view argument",
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "f_function_value_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:fn_value.arg",
				Source:  "test.tetra:5:17",
				Reason:  "borrow through function-typed value",
			},
			{
				ID:      "f_callback_arg_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:callback.arg",
				Source:  "test.tetra:9:17",
				Reason:  "borrow through callback parameter",
			},
			{
				ID:      "f_callback_inout",
				Kind:    plir.FactNoAlias,
				ValueID: "param:dst",
				Source:  "test.tetra:12:13",
				Reason:  "callback/reentrant inout cannot produce broad noalias",
			},
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
	if inout.AliasState != AliasInvalidatedByCall || inout.CostClass != CostConservativeFallback ||
		inout.ClaimLevel == ClaimValidated {
		t.Fatalf(
			("callback_inout_conservative row = %+v, want invalidated-by-call " +
				"conservative fallback and not validated"),
			inout,
		)
	}
}

func TestFromPLIRAndAllocPlanProjectsCallbackInoutBoundaryWithoutNoAlias(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "callbackBoundary",
		Values: []plir.Value{{
			ID:         "param:dst",
			Kind:       plir.ValueParam,
			Type:       "[]i32",
			Source:     "test.tetra:4:19",
			Region:     "fn:callbackBoundary",
			Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "param:dst"},
			Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "dst"},
			Borrow:     plir.BorrowMut,
			Escape:     plir.EscapeNoEscape,
		}},
		Ops: []plir.Operation{
			{
				ID:     "op_callback_inout",
				Kind:   plir.OpCall,
				Source: "test.tetra:5:5",
				Inputs: []string{"dst"},
				Note:   "cb alias_boundary:function_typed_inout",
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "f_dst_borrow",
				Kind:    plir.FactBorrowedMut,
				ValueID: "param:dst",
				Source:  "test.tetra:4:19",
				Reason:  "inout parameter borrowed mutably",
			},
			{
				ID:      "f_dst_region",
				Kind:    plir.FactRegionAlive,
				ValueID: "param:dst",
				Source:  "test.tetra:4:19",
				Region:  "fn:callbackBoundary",
				Reason:  "function region is alive",
			},
			{
				ID:      "f_dst_provenance",
				Kind:    plir.FactProvenanceKnown,
				ValueID: "param:dst",
				Source:  "test.tetra:4:19",
				Reason:  "parameter provenance known before callback boundary",
			},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	if got := countReportClaim(report, "no_alias"); got != 0 {
		t.Fatalf("callback boundary report has %d no_alias rows, want none:\n%+v", got, report.Rows)
	}
	row, ok := reportRowByClaim(report, "callback_inout_conservative")
	if !ok {
		t.Fatalf("missing callback_inout_conservative row:\n%+v", report.Rows)
	}
	if row.ParentFactID == "" ||
		row.ValueID != "param:dst" ||
		row.OwnerID != "dst" ||
		row.AliasState != AliasInvalidatedByCall ||
		row.CostClass != CostConservativeFallback ||
		row.ClaimLevel != ClaimConservative ||
		row.ValidatorStatus != ValidatorNotApplicable {
		t.Fatalf(
			"callback_inout_conservative row = %+v, want parented conservative invalidation",
			row,
		)
	}
}

func TestFromPLIRAndAllocPlanProjectsUnknownExternalNoAliasBoundaryWithoutNoAlias(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "externalBoundary",
		Values: []plir.Value{{
			ID:         "param:dst",
			Kind:       plir.ValueParam,
			Type:       "[]u8",
			Source:     "test.tetra:4:19",
			Region:     "fn:externalBoundary",
			Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "param:dst"},
			Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "dst"},
			Borrow:     plir.BorrowMut,
			Escape:     plir.EscapeNoEscape,
		}},
		Ops: []plir.Operation{
			{
				ID:     "op_unknown_external",
				Kind:   plir.OpCall,
				Source: "test.tetra:5:5",
				Inputs: []string{"dst"},
				Note:   "ffi.external unknown external call alias_boundary:unknown_external_call",
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "f_dst_borrow",
				Kind:    plir.FactBorrowedMut,
				ValueID: "param:dst",
				Source:  "test.tetra:4:19",
				Reason:  "inout parameter borrowed mutably",
			},
			{
				ID:      "f_dst_region",
				Kind:    plir.FactRegionAlive,
				ValueID: "param:dst",
				Source:  "test.tetra:4:19",
				Region:  "fn:externalBoundary",
				Reason:  "function region is alive",
			},
			{
				ID:      "f_dst_provenance",
				Kind:    plir.FactProvenanceKnown,
				ValueID: "param:dst",
				Source:  "test.tetra:4:19",
				Reason:  "parameter provenance known before unknown external boundary",
			},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	if got := countReportClaim(report, "no_alias"); got != 0 {
		t.Fatalf(
			"unknown external boundary report has %d no_alias rows, want none:\n%+v",
			got,
			report.Rows,
		)
	}
	row, ok := reportRowByClaim(report, "ffi_noalias_invalidated_by_external_call")
	if !ok {
		t.Fatalf("missing ffi_noalias_invalidated_by_external_call row:\n%+v", report.Rows)
	}
	if row.ParentFactID == "" ||
		row.ValueID != "param:dst" ||
		row.OwnerID != "dst" ||
		row.AliasState != AliasInvalidatedByCall ||
		row.CostClass != CostConservativeFallback ||
		row.ClaimLevel != ClaimConservative ||
		row.ValidatorStatus != ValidatorNotApplicable {
		t.Fatalf(
			"ffi_noalias_invalidated_by_external_call row = %+v, want parented conservative invalidation",
			row,
		)
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
			Lifetime: plir.Lifetime{
				Birth: "test.tetra:4:17",
				Death: "return",
				Owner: "callback_arg",
			},
			Borrow: plir.BorrowImm,
			Escape: plir.EscapeConservative,
		}},
		Ops: []plir.Operation{
			{
				ID:      "op_callback_arg",
				Kind:    plir.OpCall,
				Source:  "test.tetra:4:17",
				Inputs:  []string{"xs"},
				Outputs: []string{"view:callback.arg"},
				Note:    "unknown callback target contains borrowed view argument",
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "f_unknown",
				Kind:    plir.FactProvenanceUnknown,
				ValueID: "view:callback.arg",
				Source:  "test.tetra:4:17",
				Reason:  "unknown callback target",
			},
			{
				ID:      "f_callback_arg_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:callback.arg",
				Source:  "test.tetra:4:17",
				Reason:  "borrow through unknown callback parameter",
			},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, claim := range []string{"function_value_contains_borrow", "callback_arg_contains_borrow"} {
		if reportHasClaim(report, claim) {
			t.Fatalf(
				"unsafe/unknown callback target emitted trusted claim %q:\n%+v",
				claim,
				report.Rows,
			)
		}
	}
}

func TestMemoryIdealV3ProjectsInterfaceProtocolFacts(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "borrowCarrierV3",
		Values: []plir.Value{
			{
				ID:     "view:interface.value",
				Kind:   plir.ValueView,
				Type:   "[]u8",
				Source: "test.tetra:5:17",
				Region: "fn:borrowCarrierV3",
				Provenance: plir.Provenance{
					Kind: plir.ProvenanceParam,
					Root: "derived:xs.interface_value",
				},
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:5:17",
					Death: "return",
					Owner: "xs.interface_value",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeNoEscape,
			},
			{
				ID:     "view:protocol.dispatch",
				Kind:   plir.ValueView,
				Type:   "[]u8",
				Source: "test.tetra:9:17",
				Region: "fn:borrowCarrierV3",
				Provenance: plir.Provenance{
					Kind: plir.ProvenanceParam,
					Root: "derived:ys.protocol_dispatch",
				},
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:9:17",
					Death: "return",
					Owner: "ys.protocol_dispatch",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeConservative,
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
			{
				ID:      "op_interface_value",
				Kind:    plir.OpCall,
				Source:  "test.tetra:5:17",
				Inputs:  []string{"xs"},
				Outputs: []string{"view:interface.value"},
				Note:    "known static interface/protocol value contains borrowed view argument",
			},
			{
				ID:      "op_protocol_dispatch",
				Kind:    plir.OpCall,
				Source:  "test.tetra:9:17",
				Inputs:  []string{"ys"},
				Outputs: []string{"view:protocol.dispatch"},
				Note:    "unknown dynamic protocol dispatch borrow remains conservative",
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "f_interface_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:interface.value",
				Source:  "test.tetra:5:17",
				Reason:  "borrow through interface/protocol value with statically known target",
			},
			{
				ID:      "f_protocol_dispatch_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:protocol.dispatch",
				Source:  "test.tetra:9:17",
				Reason:  "borrow through unknown dynamic protocol dispatch remains conservative",
			},
			{
				ID:      "f_protocol_dispatch_noalias",
				Kind:    plir.FactNoAlias,
				ValueID: "param:dst",
				Source:  "test.tetra:12:13",
				Reason:  "protocol/interface dispatch cannot produce broad noalias",
			},
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
		{
			claim:     "protocol_dispatch_noalias_conservative",
			validator: "protocol_dispatch_alias_conservative_validator",
		},
	} {
		row, ok := reportRowByClaim(report, tc.claim)
		if !ok {
			t.Fatalf("memory report missing claim %q:\n%+v", tc.claim, report.Rows)
		}
		if row.ParentFactID == "" || row.OwnerID == "" || row.ValidatorName != tc.validator {
			t.Fatalf("%s row = %+v, want parent, owner, validator %q", tc.claim, row, tc.validator)
		}
	}
	for _, claim := range []string{
		"protocol_dispatch_borrow_conservative",
		"protocol_dispatch_noalias_conservative",
	} {
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
			Lifetime: plir.Lifetime{
				Birth: "test.tetra:4:17",
				Death: "return",
				Owner: "protocol_dispatch",
			},
			Borrow: plir.BorrowImm,
			Escape: plir.EscapeConservative,
		}},
		Ops: []plir.Operation{
			{
				ID:      "op_protocol_dispatch",
				Kind:    plir.OpCall,
				Source:  "test.tetra:4:17",
				Inputs:  []string{"xs"},
				Outputs: []string{"view:protocol.dispatch"},
				Note:    "unknown dynamic protocol dispatch contains borrowed view argument",
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "f_unknown",
				Kind:    plir.FactProvenanceUnknown,
				ValueID: "view:protocol.dispatch",
				Source:  "test.tetra:4:17",
				Reason:  "unknown dynamic protocol dispatch",
			},
			{
				ID:      "f_protocol_dispatch_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:protocol.dispatch",
				Source:  "test.tetra:4:17",
				Reason:  "borrow through unknown dynamic protocol dispatch",
			},
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
				ID:     "view:async.boundary",
				Kind:   plir.ValueView,
				Type:   "[]u8",
				Source: "test.tetra:5:17",
				Region: "fn:borrowCarrierV4",
				Provenance: plir.Provenance{
					Kind: plir.ProvenanceParam,
					Root: "derived:xs.async_boundary",
				},
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:5:17",
					Death: "await",
					Owner: "xs.async_boundary",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeConservative,
			},
			{
				ID:     "view:task.boundary",
				Kind:   plir.ValueView,
				Type:   "[]u8",
				Source: "test.tetra:9:17",
				Region: "fn:borrowCarrierV4",
				Provenance: plir.Provenance{
					Kind: plir.ProvenanceParam,
					Root: "derived:ys.task_boundary",
				},
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:9:17",
					Death: "task_spawn",
					Owner: "ys.task_boundary",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeTask,
			},
			{
				ID:     "view:actor.boundary",
				Kind:   plir.ValueView,
				Type:   "[]u8",
				Source: "test.tetra:13:17",
				Region: "fn:borrowCarrierV4",
				Provenance: plir.Provenance{
					Kind: plir.ProvenanceParam,
					Root: "derived:zs.actor_boundary",
				},
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:13:17",
					Death: "send_typed",
					Owner: "zs.actor_boundary",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeActor,
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
			{
				ID:      "op_async_boundary",
				Kind:    plir.OpCall,
				Source:  "test.tetra:5:17",
				Inputs:  []string{"xs"},
				Outputs: []string{"view:async.boundary"},
				Note:    "borrow crosses async/await suspension boundary and remains conservative",
			},
			{
				ID:      "op_task_boundary",
				Kind:    plir.OpCall,
				Source:  "test.tetra:9:17",
				Inputs:  []string{"ys"},
				Outputs: []string{"view:task.boundary"},
				Note:    "borrow crosses task boundary without explicit copy and is rejected",
			},
			{
				ID:      "op_actor_boundary",
				Kind:    plir.OpCall,
				Source:  "test.tetra:13:17",
				Inputs:  []string{"zs"},
				Outputs: []string{"view:actor.boundary"},
				Note:    "borrow crosses actor boundary without explicit copy and is rejected",
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "f_async_boundary_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:async.boundary",
				Source:  "test.tetra:5:17",
				Reason:  "borrow crossing async/await suspension boundary",
			},
			{
				ID:      "f_task_boundary_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:task.boundary",
				Source:  "test.tetra:9:17",
				Reason:  "borrow crossing task boundary without explicit copy",
			},
			{
				ID:      "f_actor_boundary_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:actor.boundary",
				Source:  "test.tetra:13:17",
				Reason:  "borrow crossing actor boundary without explicit copy",
			},
			{
				ID:      "f_boundary_noalias",
				Kind:    plir.FactNoAlias,
				ValueID: "param:dst",
				Source:  "test.tetra:17:13",
				Reason:  "task/actor boundary cannot produce broad noalias",
			},
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
		t.Fatalf(
			"async_boundary_borrow_conservative row = %+v, want conservative fallback and not validated",
			asyncRow,
		)
	}
	for _, claim := range []string{"task_boundary_borrow_rejected", "actor_boundary_borrow_rejected"} {
		row, _ := reportRowByClaim(report, claim)
		if row.CostClass != CostUnsupportedRejected || row.ClaimLevel != ClaimRejected {
			t.Fatalf("%s row = %+v, want rejected unsupported boundary fact", claim, row)
		}
	}
	noalias, _ := reportRowByClaim(report, "boundary_noalias_conservative")
	if noalias.AliasState != AliasInvalidatedByCall ||
		noalias.CostClass != CostConservativeFallback ||
		noalias.ClaimLevel == ClaimValidated {
		t.Fatalf(
			("boundary_noalias_conservative row = %+v, want invalidated-by-" +
				"call conservative fallback and not validated"),
			noalias,
		)
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
			Lifetime: plir.Lifetime{
				Birth: "test.tetra:4:17",
				Death: "task_spawn",
				Owner: "task_boundary",
			},
			Borrow: plir.BorrowImm,
			Escape: plir.EscapeConservative,
		}},
		Ops: []plir.Operation{
			{
				ID:      "op_task_boundary",
				Kind:    plir.OpCall,
				Source:  "test.tetra:4:17",
				Inputs:  []string{"xs"},
				Outputs: []string{"view:task.boundary"},
				Note:    "unknown task target contains borrowed view argument",
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "f_unknown",
				Kind:    plir.FactProvenanceUnknown,
				ValueID: "view:task.boundary",
				Source:  "test.tetra:4:17",
				Reason:  "unknown task target",
			},
			{
				ID:      "f_task_boundary_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:task.boundary",
				Source:  "test.tetra:4:17",
				Reason:  "borrow through unknown task target",
			},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, claim := range []string{
		"async_boundary_borrow_conservative",
		"task_boundary_borrow_rejected",
		"actor_boundary_borrow_rejected",
	} {
		if reportHasClaim(report, claim) {
			t.Fatalf(
				"unknown task/actor target emitted boundary claim %q:\n%+v",
				claim,
				report.Rows,
			)
		}
	}
}
