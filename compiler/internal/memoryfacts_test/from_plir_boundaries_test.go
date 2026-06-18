package memoryfacts_test

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/allocplan"
	. "tetra_language/compiler/internal/memoryfacts"
	. "tetra_language/compiler/internal/memoryfacts/fromplir"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/runtimeabi"
)

func TestMemoryIdealV10ProjectsAsyncCancellationBoundaryFacts(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "borrowCarrierV10",
		Values: []plir.Value{
			{
				ID:     "view:pre.await",
				Kind:   plir.ValueView,
				Type:   "[]u8",
				Source: "test.tetra:5:17",
				Region: "fn:borrowCarrierV10",
				Provenance: plir.Provenance{
					Kind: plir.ProvenanceParam,
					Root: "derived:xs.pre_await",
				},
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:5:17",
					Death: "before_await",
					Owner: "xs.pre_await",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeNoEscape,
			},
			{
				ID:     "view:post.await",
				Kind:   plir.ValueView,
				Type:   "[]u8",
				Source: "test.tetra:9:17",
				Region: "fn:borrowCarrierV10",
				Provenance: plir.Provenance{
					Kind: plir.ProvenanceParam,
					Root: "derived:ys.post_await",
				},
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:9:17",
					Death: "after_await",
					Owner: "ys.post_await",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeConservative,
			},
			{
				ID:     "view:cancel",
				Kind:   plir.ValueView,
				Type:   "[]u8",
				Source: "test.tetra:13:17",
				Region: "fn:borrowCarrierV10",
				Provenance: plir.Provenance{
					Kind: plir.ProvenanceParam,
					Root: "derived:task_owned.cancel",
				},
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:13:17",
					Death: "cancel",
					Owner: "task_owned.cancel",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeTask,
			},
			{
				ID:         "param:task_group_dst",
				Kind:       plir.ValueParam,
				Type:       "[]u8",
				Source:     "test.tetra:17:13",
				Region:     "fn:borrowCarrierV10",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "task_group_dst"},
				Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "task_group_dst"},
				Borrow:     plir.BorrowMut,
				Escape:     plir.EscapeNoEscape,
			},
			{
				ID:     "view:actor.reentrant",
				Kind:   plir.ValueView,
				Type:   "[]u8",
				Source: "test.tetra:21:17",
				Region: "fn:borrowCarrierV10",
				Provenance: plir.Provenance{
					Kind: plir.ProvenanceParam,
					Root: "derived:actor_state.reentrant_callback",
				},
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:21:17",
					Death: "actor_reentrant_callback",
					Owner: "actor_state.reentrant_callback",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeActor,
			},
		},
		Ops: []plir.Operation{
			{
				ID:      "op_pre_await",
				Kind:    plir.OpCall,
				Source:  "test.tetra:5:17",
				Inputs:  []string{"xs"},
				Outputs: []string{"view:pre.await"},
				Note:    "pre-await local borrow before suspension has compiler-owned no-escape proof",
			},
			{
				ID:      "op_post_await",
				Kind:    plir.OpCall,
				Source:  "test.tetra:9:17",
				Inputs:  []string{"ys"},
				Outputs: []string{"view:post.await"},
				Note:    "borrow used after await suspension remains conservative",
			},
			{
				ID:      "op_cancel",
				Kind:    plir.OpCall,
				Source:  "test.tetra:13:17",
				Inputs:  []string{"task_owned"},
				Outputs: []string{"view:cancel"},
				Note:    "cancellation path invalidates task-owned borrowed lifetime",
			},
			{
				ID:      "op_actor_reentrant",
				Kind:    plir.OpCall,
				Source:  "test.tetra:21:17",
				Inputs:  []string{"actor_state"},
				Outputs: []string{"view:actor.reentrant"},
				Note:    "actor reentrant callback captures borrowed state and remains conservative",
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "f_pre_await_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:pre.await",
				Source:  "test.tetra:5:17",
				Reason:  "pre-await local borrow before suspension no_escape_proof",
			},
			{
				ID:      "f_post_await_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:post.await",
				Source:  "test.tetra:9:17",
				Reason:  "borrow used after await suspension",
			},
			{
				ID:      "f_cancel_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:cancel",
				Source:  "test.tetra:13:17",
				Reason:  "cancellation path invalidates task-owned borrowed lifetime",
			},
			{
				ID:      "f_task_group_noalias",
				Kind:    plir.FactNoAlias,
				ValueID: "param:task_group_dst",
				Source:  "test.tetra:17:13",
				Reason:  "task group structured concurrency boundary cannot validate broad noalias",
			},
			{
				ID:      "f_actor_reentrant_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:actor.reentrant",
				Source:  "test.tetra:21:17",
				Reason:  "actor reentrant callback captures borrowed state",
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
		level     ClaimLevel
		cost      CostClass
	}{
		{
			claim:     "pre_await_local_borrow_validated",
			validator: "pre_await_local_borrow_validator",
			level:     ClaimValidated,
			cost:      CostZeroCostProven,
		},
		{
			claim:     "post_await_borrow_conservative",
			validator: "post_await_borrow_conservative_validator",
			level:     ClaimConservative,
			cost:      CostConservativeFallback,
		},
		{
			claim:     "cancellation_borrow_lifetime_invalidated",
			validator: "cancellation_lifetime_invalidation_validator",
			level:     ClaimRejected,
			cost:      CostUnsupportedRejected,
		},
		{
			claim:     "task_group_noalias_conservative",
			validator: "task_group_boundary_conservative_validator",
			level:     ClaimConservative,
			cost:      CostConservativeFallback,
		},
		{
			claim:     "actor_reentrant_callback_conservative",
			validator: "actor_reentrant_callback_boundary_validator",
			level:     ClaimConservative,
			cost:      CostConservativeFallback,
		},
	} {
		row, ok := reportRowByClaim(report, tc.claim)
		if !ok {
			t.Fatalf("memory report missing claim %q:\n%+v", tc.claim, report.Rows)
		}
		if row.ParentFactID == "" || row.OwnerID == "" || row.ValidatorName != tc.validator ||
			row.ClaimLevel != tc.level ||
			row.CostClass != tc.cost {
			t.Fatalf(
				"%s row = %+v, want parent, owner, validator %q, level %s, cost %s",
				tc.claim,
				row,
				tc.validator,
				tc.level,
				tc.cost,
			)
		}
	}
}

func TestMemoryIdealV11ProjectsDynamicProtocolWitnessFacts(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name: "borrowCarrierV11",
		Values: []plir.Value{
			{
				ID:     "view:dynamic.existential",
				Kind:   plir.ValueView,
				Type:   "dyn Drawable",
				Source: "test.tetra:5:17",
				Region: "fn:borrowCarrierV11",
				Provenance: plir.Provenance{
					Kind: plir.ProvenanceParam,
					Root: "derived:xs.dynamic_existential",
				},
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:5:17",
					Death: "dynamic_dispatch",
					Owner: "xs.dynamic_existential",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeConservative,
			},
			{
				ID:     "view:static.witness",
				Kind:   plir.ValueView,
				Type:   "Witness<Drawable>",
				Source: "test.tetra:9:17",
				Region: "fn:borrowCarrierV11",
				Provenance: plir.Provenance{
					Kind: plir.ProvenanceParam,
					Root: "derived:ys.static_witness",
				},
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:9:17",
					Death: "return",
					Owner: "ys.static_witness",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeNoEscape,
			},
			{
				ID:         "param:dispatch_dst",
				Kind:       plir.ValueParam,
				Type:       "[]u8",
				Source:     "test.tetra:13:13",
				Region:     "fn:borrowCarrierV11",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "dispatch_dst"},
				Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "dispatch_dst"},
				Borrow:     plir.BorrowMut,
				Escape:     plir.EscapeNoEscape,
			},
			{
				ID:         "view:witness.lookup",
				Kind:       plir.ValueView,
				Type:       "dyn Drawable",
				Source:     "test.tetra:17:17",
				Region:     "fn:borrowCarrierV11",
				Provenance: plir.Provenance{Kind: plir.ProvenanceUnknown, Root: "witness_lookup"},
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:17:17",
					Death: "return",
					Owner: "witness_lookup",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeConservative,
			},
			{
				ID:     "view:report.integrity",
				Kind:   plir.ValueView,
				Type:   "dyn Drawable",
				Source: "test.tetra:21:17",
				Region: "fn:borrowCarrierV11",
				Provenance: plir.Provenance{
					Kind: plir.ProvenanceParam,
					Root: "derived:zs.protocol_dispatch_report",
				},
				Lifetime: plir.Lifetime{
					Birth: "test.tetra:21:17",
					Death: "dynamic_dispatch",
					Owner: "zs.protocol_dispatch_report",
				},
				Borrow: plir.BorrowImm,
				Escape: plir.EscapeConservative,
			},
		},
		Ops: []plir.Operation{
			{
				ID:      "op_dynamic_existential",
				Kind:    plir.OpCall,
				Source:  "test.tetra:5:17",
				Inputs:  []string{"xs"},
				Outputs: []string{"view:dynamic.existential"},
				Note: ("dynamic existential protocol borrow carrier remains " +
					"conservative unless statically resolved"),
			},
			{
				ID:      "op_static_witness",
				Kind:    plir.OpCall,
				Source:  "test.tetra:9:17",
				Inputs:  []string{"ys"},
				Outputs: []string{"view:static.witness"},
				Note: ("static witness conformance proof carries borrow facts only with " +
					"compiler-owned parent fact"),
			},
			{
				ID:      "op_witness_lookup",
				Kind:    plir.OpCall,
				Source:  "test.tetra:17:17",
				Inputs:  []string{"unknown"},
				Outputs: []string{"view:witness.lookup"},
				Note:    "witness table lookup cannot promote unknown provenance to safe_known",
			},
			{
				ID:      "op_report_integrity",
				Kind:    plir.OpCall,
				Source:  "test.tetra:21:17",
				Inputs:  []string{"zs"},
				Outputs: []string{"view:report.integrity"},
				Note: ("protocol existential dispatch report rows preserve source_fact_" +
					"id cost_class normal_build_check"),
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "f_dynamic_existential_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:dynamic.existential",
				Source:  "test.tetra:5:17",
				Reason:  "dynamic existential protocol borrow carrier remains conservative",
			},
			{
				ID:      "f_static_witness_borrow",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:static.witness",
				Source:  "test.tetra:9:17",
				Reason:  "static witness conformance proof has compiler-owned parent fact",
			},
			{
				ID:      "f_dynamic_protocol_noalias",
				Kind:    plir.FactNoAlias,
				ValueID: "param:dispatch_dst",
				Source:  "test.tetra:13:13",
				Reason:  "dynamic protocol dispatch cannot validate broad noalias",
			},
			{
				ID:      "f_witness_lookup_unknown",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:witness.lookup",
				Source:  "test.tetra:17:17",
				Reason:  "witness table lookup cannot promote unknown provenance to safe_known",
			},
			{
				ID:      "f_report_integrity",
				Kind:    plir.FactBorrowedImm,
				ValueID: "view:report.integrity",
				Source:  "test.tetra:21:17",
				Reason: ("protocol existential dispatch report rows preserve source_fact_" +
					"id cost_class normal_build_check"),
			},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, tc := range []struct {
		claim            string
		validator        string
		level            ClaimLevel
		cost             CostClass
		normalBuildCheck bool
	}{
		{
			claim:     "dynamic_existential_borrow_conservative",
			validator: "dynamic_existential_borrow_conservative_validator",
			level:     ClaimConservative,
			cost:      CostConservativeFallback,
		},
		{
			claim:     "static_witness_borrow_parent_validated",
			validator: "static_witness_parent_fact_validator",
			level:     ClaimValidated,
			cost:      CostZeroCostProven,
		},
		{
			claim:     "dynamic_protocol_noalias_rejected",
			validator: "dynamic_protocol_noalias_rejection_validator",
			level:     ClaimRejected,
			cost:      CostUnsupportedRejected,
		},
		{
			claim:     "witness_provenance_promotion_rejected",
			validator: "witness_provenance_promotion_validator",
			level:     ClaimRejected,
			cost:      CostUnsupportedRejected,
		},
		{
			claim:            "protocol_dispatch_report_integrity",
			validator:        "protocol_dispatch_report_integrity_validator",
			level:            ClaimValidated,
			cost:             CostDynamicCheckRequired,
			normalBuildCheck: true,
		},
	} {
		row, ok := reportRowByClaim(report, tc.claim)
		if !ok {
			t.Fatalf("memory report missing claim %q:\n%+v", tc.claim, report.Rows)
		}
		if row.ParentFactID == "" || row.ValidatorName != tc.validator ||
			row.ClaimLevel != tc.level ||
			row.CostClass != tc.cost ||
			row.NormalBuildCheck != tc.normalBuildCheck {
			t.Fatalf(
				"%s row = %+v, want parent, validator %q, level %s, cost %s, normal_build_check %v",
				tc.claim,
				row,
				tc.validator,
				tc.level,
				tc.cost,
				tc.normalBuildCheck,
			)
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
				Reason:  "unique local sequential inout interval",
			},
		},
	}}}

	graph, err := FromPLIRAndAllocPlan("program", prog, nil)
	if err != nil {
		t.Fatalf("FromPLIRAndAllocPlan: %v", err)
	}
	report := BuildReportFromGraph(graph)
	for _, want := range []string{
		"no_alias_validated_narrow_unique_local",
		"no_alias_validated_narrow_sequential_inout",
	} {
		row, ok := reportRowByClaim(report, want)
		if !ok {
			t.Fatalf("memory report missing claim %q:\n%+v", want, report.Rows)
		}
		if row.ParentFactID == "" || row.AliasState != AliasMutableExclusive ||
			row.ValidatorName != "alias_interval_validator" ||
			row.ClaimLevel != ClaimValidated {
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
					Lifetime: plir.Lifetime{
						Birth: "test.tetra:3:12",
						Death: "return",
						Owner: "borrowed",
					},
					Borrow: plir.BorrowImm,
					Escape: plir.EscapeReturn,
				},
			},
			Ops: []plir.Operation{
				{
					ID:     "op_return_borrow",
					Kind:   plir.OpReturn,
					Source: "test.tetra:3:5",
					Inputs: []string{"borrowed"},
				},
			},
			Facts: []plir.Fact{
				{
					ID:      "f_borrow",
					Kind:    plir.FactBorrowedImm,
					ValueID: "view:borrowed",
					Source:  "test.tetra:3:12",
					Reason:  "borrowed return view",
				},
				{
					ID:      "f_prov",
					Kind:    plir.FactProvenanceKnown,
					ValueID: "view:borrowed",
					Source:  "test.tetra:3:12",
					Reason:  "borrow preserves source provenance",
				},
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
			Ops: []plir.Operation{
				{
					ID:     "op_return_copy",
					Kind:   plir.OpReturn,
					Source: "test.tetra:7:5",
					Inputs: []string{"copied"},
				},
			},
			Facts: []plir.Fact{
				{
					ID:      "f_owned",
					Kind:    plir.FactOwned,
					ValueID: "alloc_intent:copied",
					Source:  "test.tetra:7:12",
					Reason:  "copy result owns new storage",
				},
			},
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
			Ops: []plir.Operation{
				{
					ID:     "op_return_raw",
					Kind:   plir.OpReturn,
					Source: "test.tetra:11:5",
					Inputs: []string{"raw"},
				},
			},
			Facts: []plir.Fact{
				{
					ID:      "f_unknown",
					Kind:    plir.FactProvenanceUnknown,
					ValueID: "view:raw",
					Source:  "test.tetra:11:12",
					Reason:  "raw slice external provenance",
				},
			},
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
				{
					ID:      "op_store",
					Kind:    plir.OpGlobalStore,
					Source:  "test.tetra:15:5",
					Inputs:  []string{"dst"},
					Outputs: []string{"G"},
					Note:    "global store",
				},
				{
					ID:     "op_actor",
					Kind:   plir.OpActorSend,
					Source: "test.tetra:16:5",
					Inputs: []string{"peer", "owned"},
					Note:   "core.send_typed typed actor ownership transfer",
				},
				{
					ID:     "op_task",
					Kind:   plir.OpCall,
					Source: "test.tetra:17:5",
					Inputs: []string{"worker"},
					Note:   "core.task_spawn_i32",
				},
				{
					ID:      "op_closure",
					Kind:    plir.OpClosure,
					Source:  "test.tetra:18:12",
					Inputs:  []string{"dst"},
					Outputs: []string{"cb"},
					Note:    "closure captures environment",
				},
				{
					ID:     "op_unknown",
					Kind:   plir.OpCall,
					Source: "test.tetra:19:5",
					Inputs: []string{"p"},
					Note:   "ffi.unknown external call",
				},
			},
			Facts: []plir.Fact{
				{
					ID:      "f_borrow_mut",
					Kind:    plir.FactBorrowedMut,
					ValueID: "param:dst",
					Source:  "test.tetra:14:17",
					Reason:  "inout parameter",
				},
				{
					ID:      "f_no_alias",
					Kind:    plir.FactNoAlias,
					ValueID: "param:dst",
					Source:  "test.tetra:14:17",
					Reason:  "inout parameter has exclusive mutable access for call duration",
				},
				{
					ID:      "f_moved",
					Kind:    plir.FactMoved,
					ValueID: "param:owned",
					Source:  "test.tetra:16:5",
					Reason:  "typed actor ownership transfer moved payload",
				},
				{
					ID:      "f_ptr_unknown",
					Kind:    plir.FactProvenanceUnknown,
					ValueID: "param:p",
					Source:  "test.tetra:14:49",
					Reason:  "pointer retained across unknown unsafe boundary",
				},
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
		"cap_mem_authorization_only",
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
		if row.Claim == "cap_mem_authorization_only" &&
			(row.ProvenanceClass != ProvenanceUnsafeChecked ||
				row.UnsafeClass != UnsafeChecked ||
				row.ClaimLevel != ClaimEvidenceOnly ||
				row.CostClass != CostInstrumentationOnly ||
				row.ValidatorStatus != ValidatorNotRun ||
				strings.Contains(row.Claim, "provenance_known") ||
				strings.Contains(row.Claim, "no_alias")) {
			t.Fatalf(
				("cap_mem_authorization_only row = %+v, want evidence-only " +
					"authorization without safe provenance/noalias proof"),
				row,
			)
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
			{
				ID:      "f_moved_unknown",
				Kind:    plir.FactMoved,
				ValueID: "param:rawOwner",
				Source:  "test.tetra:3:5",
				Reason:  "unknown pointer moved",
			},
			{
				ID:      "f_mut_unknown",
				Kind:    plir.FactBorrowedMut,
				ValueID: "param:rawDst",
				Source:  "test.tetra:3:18",
				Reason:  "unknown external inout",
			},
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
			if row.ProvenanceClass != ProvenanceUnsafeUnknown || row.UnsafeClass != UnsafeUnknown ||
				row.ClaimLevel != ClaimConservative {
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
			{
				ID:      "op_alloc",
				Kind:    plir.OpAllocIntent,
				Source:  "test.tetra:4:17",
				Inputs:  []string{"16"},
				Outputs: []string{"alloc_intent:p"},
				Note:    "alloc_bytes raw allocation-base metadata",
			},
			{
				ID:          "op_ptr_verified",
				Kind:        plir.OpUnsafe,
				Source:      "test.tetra:5:17",
				Inputs:      []string{"p", "4", "mem"},
				Outputs:     []string{"q"},
				UnsafeClass: plir.UnsafeChecked,
				Note:        "core.ptr_add raw_pointer_bounds: derived_allocation_offset base:p offset:4",
			},
			{
				ID:          "op_ptr_unknown",
				Kind:        plir.OpUnsafe,
				Source:      "test.tetra:6:17",
				Inputs:      []string{"external", "4", "mem"},
				Outputs:     []string{"r"},
				UnsafeClass: plir.UnsafeUnknown,
				Note:        "core.ptr_add raw_pointer_bounds: checked_external_unknown base:external offset:4",
			},
			{
				ID:          "op_raw_load",
				Kind:        plir.OpUnsafe,
				Source:      "test.tetra:6:21",
				Inputs:      []string{"q", "mem"},
				UnsafeClass: plir.UnsafeChecked,
				Note:        "core.load_u8 raw memory gateway: derived_allocation_offset pointer:q",
			},
			{
				ID:          "op_raw_store_unknown",
				Kind:        plir.OpUnsafe,
				Source:      "test.tetra:6:41",
				Inputs:      []string{"external", "0", "mem"},
				UnsafeClass: plir.UnsafeUnknown,
				Note:        "core.store_u8 raw memory gateway: checked_external_unknown pointer:external",
			},
			{
				ID:          "op_ptr_negative",
				Kind:        plir.OpUnsafe,
				Source:      "test.tetra:8:17",
				Inputs:      []string{"p", "-1", "mem"},
				Outputs:     []string{"neg"},
				UnsafeClass: plir.UnsafeChecked,
				Note:        "core.ptr_add raw_pointer_bounds: rejected_negative_offset base:p offset:-1",
			},
			{
				ID:          "op_ptr_upper",
				Kind:        plir.OpUnsafe,
				Source:      "test.tetra:9:17",
				Inputs:      []string{"p", "16", "mem"},
				Outputs:     []string{"upper"},
				UnsafeClass: plir.UnsafeChecked,
				Note:        "core.ptr_add raw_pointer_bounds: rejected_upper_bound base:p offset:16",
			},
			{
				ID:          "op_raw_load_width",
				Kind:        plir.OpUnsafe,
				Source:      "test.tetra:10:21",
				Inputs:      []string{"q", "mem"},
				UnsafeClass: plir.UnsafeChecked,
				Note: ("core.load_i32 raw memory gateway: rejected_access_width_" +
					"overflow pointer:q offset:14 width:4"),
			},
			{
				ID:          "op_raw_slice",
				Kind:        plir.OpUnsafe,
				Source:      "test.tetra:7:25",
				Outputs:     []string{"view:raw"},
				UnsafeClass: plir.UnsafeUnknown,
				Note:        "core.raw_slice_u8_from_parts creates a conservative external-provenance view",
			},
			{
				ID:          "op_raw_slice_verified",
				Kind:        plir.OpUnsafe,
				Source:      "test.tetra:7:35",
				Outputs:     []string{"view:raw_checked"},
				UnsafeClass: plir.UnsafeChecked,
				Note: ("core.raw_slice_u8_from_parts raw_slice_bounds: verified_" +
					"allocation_root base:p length_bytes:4"),
			},
			{
				ID:          "op_raw_slice_negative",
				Kind:        plir.OpUnsafe,
				Source:      "test.tetra:8:35",
				Outputs:     []string{"view:raw_negative"},
				UnsafeClass: plir.UnsafeChecked,
				Note: ("core.raw_slice_u8_from_parts raw_slice_bounds: rejected_" +
					"negative_length base:p length:-1"),
			},
			{
				ID:          "op_raw_slice_length_overflow",
				Kind:        plir.OpUnsafe,
				Source:      "test.tetra:9:35",
				Outputs:     []string{"view:raw_overflow"},
				UnsafeClass: plir.UnsafeChecked,
				Note: ("core.raw_slice_i32_from_parts raw_slice_bounds: rejected_length_" +
					"overflow base:p length_bytes:overflow elem_size:4"),
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "f_alloc_prov",
				Kind:    plir.FactProvenanceKnown,
				ValueID: "alloc_intent:p",
				Source:  "test.tetra:4:17",
				Reason:  "core.alloc_bytes allocation-base metadata",
			},
			{
				ID:      "f_alloc_region",
				Kind:    plir.FactRegionAlive,
				ValueID: "alloc_intent:p",
				Region:  "raw_allocation:p",
				Source:  "test.tetra:4:17",
				Reason:  "raw allocation root alive",
			},
			{
				ID:      "f_raw_unknown",
				Kind:    plir.FactProvenanceUnknown,
				ValueID: "view:raw",
				Source:  "test.tetra:7:25",
				Reason:  "raw slice gateway has external provenance",
			},
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
		{
			claim:      "allocation_base_metadata",
			provenance: ProvenanceUnsafeVerifiedRoot,
			unsafe:     UnsafeVerifiedRoot,
			level:      ClaimValidated,
			cost:       CostZeroCostProven,
		},
		{
			claim:      "derived_allocation_offset",
			provenance: ProvenanceUnsafeChecked,
			unsafe:     UnsafeChecked,
			level:      ClaimEvidenceOnly,
			cost:       CostDynamicCheckRequired,
		},
		{
			claim:      "checked_external_unknown",
			provenance: ProvenanceUnsafeUnknown,
			unsafe:     UnsafeUnknown,
			level:      ClaimConservative,
			cost:       CostConservativeFallback,
		},
		{
			claim:      "raw_memory_access_checked",
			provenance: ProvenanceUnsafeChecked,
			unsafe:     UnsafeChecked,
			level:      ClaimEvidenceOnly,
			cost:       CostDynamicCheckRequired,
		},
		{
			claim:      "raw_memory_access_unknown",
			provenance: ProvenanceUnsafeUnknown,
			unsafe:     UnsafeUnknown,
			level:      ClaimConservative,
			cost:       CostConservativeFallback,
		},
		{
			claim:      "rejected_negative_offset",
			provenance: ProvenanceUnsafeChecked,
			unsafe:     UnsafeChecked,
			level:      ClaimEvidenceOnly,
			cost:       CostUnsupportedRejected,
		},
		{
			claim:      "rejected_upper_bound",
			provenance: ProvenanceUnsafeChecked,
			unsafe:     UnsafeChecked,
			level:      ClaimEvidenceOnly,
			cost:       CostUnsupportedRejected,
		},
		{
			claim:      "rejected_access_width_overflow",
			provenance: ProvenanceUnsafeChecked,
			unsafe:     UnsafeChecked,
			level:      ClaimEvidenceOnly,
			cost:       CostUnsupportedRejected,
		},
		{
			claim:      "external_unknown",
			provenance: ProvenanceUnsafeUnknown,
			unsafe:     UnsafeUnknown,
			level:      ClaimConservative,
			cost:       CostConservativeFallback,
		},
		{
			claim:      "raw_slice_verified_allocation_root",
			provenance: ProvenanceUnsafeChecked,
			unsafe:     UnsafeChecked,
			level:      ClaimEvidenceOnly,
			cost:       CostDynamicCheckRequired,
		},
		{
			claim:      "rejected_negative_length",
			provenance: ProvenanceUnsafeChecked,
			unsafe:     UnsafeChecked,
			level:      ClaimEvidenceOnly,
			cost:       CostUnsupportedRejected,
		},
		{
			claim:      "rejected_length_overflow",
			provenance: ProvenanceUnsafeChecked,
			unsafe:     UnsafeChecked,
			level:      ClaimEvidenceOnly,
			cost:       CostUnsupportedRejected,
		},
	} {
		row, ok := reportRowByClaim(report, tc.claim)
		if !ok {
			t.Fatalf("memory report missing unsafe gateway claim %q:\n%+v", tc.claim, report.Rows)
		}
		if row.ProvenanceClass != tc.provenance || row.UnsafeClass != tc.unsafe ||
			row.ClaimLevel != tc.level ||
			row.CostClass != tc.cost {
			t.Fatalf(
				"%s row = %+v, want provenance %s unsafe %s level %s cost %s",
				tc.claim,
				row,
				tc.provenance,
				tc.unsafe,
				tc.level,
				tc.cost,
			)
		}
		if row.CostClass == CostDynamicCheckRequired && !row.NormalBuildCheck {
			t.Fatalf(
				"%s row = %+v, dynamic_check_required must keep normal_build_check",
				tc.claim,
				row,
			)
		}
	}
	for _, row := range report.Rows {
		if row.ValueID == "alloc_intent:p" &&
			(row.Claim == "provenance_known" || row.Claim == "region_alive") {
			t.Fatalf(
				"core.alloc_bytes verified root emitted generic unsafe_verified_root row: %+v",
				row,
			)
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
		t.Fatalf(
			"raw bounds normal-build row = %+v, want validated dynamic check with parent fact",
			rawCheck,
		)
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
			{
				ID:          "op_ptr_verified",
				Kind:        plir.OpUnsafe,
				Source:      "test.tetra:5:17",
				Inputs:      []string{"p", "4", "mem"},
				Outputs:     []string{"q"},
				UnsafeClass: plir.UnsafeChecked,
				Note:        "core.ptr_add raw_pointer_bounds: derived_allocation_offset base:p offset:4",
			},
			{
				ID:          "op_ptr_unknown",
				Kind:        plir.OpUnsafe,
				Source:      "test.tetra:6:17",
				Inputs:      []string{"external", "4", "mem"},
				Outputs:     []string{"r"},
				UnsafeClass: plir.UnsafeUnknown,
				Note:        "core.ptr_add raw_pointer_bounds: checked_external_unknown base:external offset:4",
			},
			{
				ID:          "op_runtime_contract",
				Kind:        plir.OpUnsafe,
				Source:      "test.tetra:7:17",
				Inputs:      []string{"q", "mem"},
				UnsafeClass: plir.UnsafeChecked,
				Note:        "unsafe contract runtime_checkable: nonnull alignment length pointer:q",
			},
			{
				ID:          "op_static_contract",
				Kind:        plir.OpUnsafe,
				Source:      "test.tetra:8:17",
				Inputs:      []string{"external", "mem"},
				UnsafeClass: plir.UnsafeUnknown,
				Note:        "unsafe contract static_untrusted: noalias lifetime region pointer:external",
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "f_alloc_prov",
				Kind:    plir.FactProvenanceKnown,
				ValueID: "alloc_intent:p",
				Source:  "test.tetra:4:17",
				Reason:  "core.alloc_bytes allocation-base metadata",
			},
			{
				ID:      "f_external_unknown",
				Kind:    plir.FactProvenanceUnknown,
				ValueID: "param:external",
				Source:  "test.tetra:2:17",
				Reason:  "external raw pointer remains unsafe_unknown",
			},
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
		{
			claim:      "unsafe_verified_root_allocation_base",
			validator:  "unsafe_verified_root_bounds_validator",
			provenance: ProvenanceUnsafeVerifiedRoot,
			unsafe:     UnsafeVerifiedRoot,
			level:      ClaimValidated,
			cost:       CostZeroCostProven,
		},
		{
			claim:      "unsafe_unknown_rejected_safe_facts",
			validator:  "unsafe_unknown_fact_validator",
			provenance: ProvenanceUnsafeUnknown,
			unsafe:     UnsafeUnknown,
			level:      ClaimRejected,
			cost:       CostUnsupportedRejected,
		},
		{
			claim:      "unsafe_contract_runtime_checkable",
			validator:  "unsafe_runtime_contract_validator",
			provenance: ProvenanceUnsafeChecked,
			unsafe:     UnsafeChecked,
			level:      ClaimValidated,
			cost:       CostDynamicCheckRequired,
		},
		{
			claim:      "unsafe_contract_static_untrusted",
			validator:  "unsafe_static_contract_validator",
			provenance: ProvenanceUnsafeUnknown,
			unsafe:     UnsafeUnknown,
			level:      ClaimConservative,
			cost:       CostConservativeFallback,
		},
	} {
		row, ok := reportRowByClaim(report, tc.claim)
		if !ok {
			t.Fatalf("memory report missing v5 claim %q:\n%+v", tc.claim, report.Rows)
		}
		if row.ValidatorName != tc.validator || row.ProvenanceClass != tc.provenance ||
			row.UnsafeClass != tc.unsafe ||
			row.ClaimLevel != tc.level ||
			row.CostClass != tc.cost {
			t.Fatalf(
				"%s row = %+v, want validator %s provenance %s unsafe %s level %s cost %s",
				tc.claim,
				row,
				tc.validator,
				tc.provenance,
				tc.unsafe,
				tc.level,
				tc.cost,
			)
		}
		if row.CostClass == CostDynamicCheckRequired && !row.NormalBuildCheck {
			t.Fatalf(
				"%s row = %+v, dynamic_check_required must keep normal_build_check",
				tc.claim,
				row,
			)
		}
	}
	staticRow, _ := reportRowByClaim(report, "unsafe_contract_static_untrusted")
	if staticRow.AliasState != AliasInvalidatedByCall {
		t.Fatalf(
			"unsafe_contract_static_untrusted row = %+v, want invalidated_by_call alias state",
			staticRow,
		)
	}
	for _, forbidden := range []string{"safe_known", "provenance_known", "no_alias"} {
		for _, row := range report.Rows {
			if row.UnsafeClass == UnsafeUnknown && row.Claim == forbidden {
				t.Fatalf(
					"unsafe_unknown emitted forbidden safe/noalias claim %q: %+v",
					forbidden,
					row,
				)
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
			{
				ID:     "op_ffi",
				Kind:   plir.OpCall,
				Source: "test.tetra:4:5",
				Inputs: []string{"external", "borrowed", "dst"},
				Note:   "ffi.external call may retain borrowed pointer and invalidates noalias",
			},
			{
				ID:          "op_safe_wrapper",
				Kind:        plir.OpUnsafe,
				Source:      "test.tetra:5:17",
				Inputs:      []string{"external", "mem"},
				Outputs:     []string{"wrapped"},
				UnsafeClass: plir.UnsafeUnknown,
				Note:        "safe wrapper promotion from external pointer without compiler-owned contract",
			},
		},
		Facts: []plir.Fact{
			{
				ID:      "f_external_unknown",
				Kind:    plir.FactProvenanceUnknown,
				ValueID: "param:external",
				Source:  "test.tetra:2:17",
				Reason:  "external pointer remains unsafe_unknown",
			},
			{
				ID:      "f_borrowed",
				Kind:    plir.FactBorrowedImm,
				ValueID: "param:borrowed",
				Source:  "test.tetra:2:34",
				Reason:  "borrowed pointer argument",
			},
			{
				ID:      "f_noalias",
				Kind:    plir.FactNoAlias,
				ValueID: "param:dst",
				Source:  "test.tetra:2:52",
				Reason:  "exclusive pointer before external call",
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
		level     ClaimLevel
		cost      CostClass
		alias     AliasState
		parent    bool
	}{
		{
			claim:     "ffi_pointer_external_unknown",
			validator: "external_pointer_provenance_validator",
			level:     ClaimConservative,
			cost:      CostConservativeFallback,
		},
		{
			claim:     "ffi_call_may_retain_borrow",
			validator: "ffi_lifetime_conservative_validator",
			level:     ClaimConservative,
			cost:      CostConservativeFallback,
			parent:    true,
		},
		{
			claim:     "ffi_noalias_invalidated_by_external_call",
			validator: "ffi_noalias_conservative_validator",
			level:     ClaimConservative,
			cost:      CostConservativeFallback,
			alias:     AliasInvalidatedByCall,
			parent:    true,
		},
		{
			claim:     "safe_wrapper_promotion_rejected_without_contract",
			validator: "safe_wrapper_promotion_validator",
			level:     ClaimRejected,
			cost:      CostUnsupportedRejected,
			parent:    true,
		},
		{
			claim:     "external_pointer_provenance_rejected",
			validator: "external_pointer_provenance_validator",
			level:     ClaimRejected,
			cost:      CostUnsupportedRejected,
			parent:    true,
		},
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
			t.Fatalf(
				"%s row = %+v, want validator %s unsafe_unknown level %s cost %s",
				tc.claim,
				row,
				tc.validator,
				tc.level,
				tc.cost,
			)
		}
		if tc.alias != "" && row.AliasState != tc.alias {
			t.Fatalf("%s row = %+v, want alias %s", tc.claim, row, tc.alias)
		}
		if tc.parent && row.ParentFactID == "" {
			t.Fatalf("%s row = %+v, want parent_fact_id", tc.claim, row)
		}
	}
	for _, forbidden := range []string{
		"safe_known",
		"provenance_known",
		"no_alias",
		"bounds_check_eliminated",
		"index_in_range",
	} {
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

func reportRowByClaimAndSource(report Report, claim string, sourceSpan string) (ReportRow, bool) {
	for _, row := range report.Rows {
		if row.Claim == claim && row.SourceSpan == sourceSpan {
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
