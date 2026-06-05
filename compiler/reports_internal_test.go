package compiler

import (
	"reflect"
	"strings"
	"testing"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
	"tetra_language/compiler/internal/semantics"
)

func TestValidateAllocationPlanReportRejectsMismatch(t *testing.T) {
	plan := &allocplan.Plan{
		Totals: allocplan.Totals{Stack: 1},
		Functions: []allocplan.FunctionPlan{{
			Name: "main",
			Allocations: []allocplan.Allocation{{
				ID:                    "xs",
				SiteID:                "allocsite:main:xs:line_1_1",
				ValueID:               "alloc_intent:xs",
				Builtin:               "core.make_u8",
				ElementType:           "u8",
				ElementSize:           1,
				LengthExpr:            "4",
				LengthStatus:          allocplan.LengthStatusNormal,
				ZeroGuardStatus:       "valid_empty_no_allocator",
				NegativeGuardStatus:   "reject_before_allocation",
				OverflowGuardStatus:   "reject_before_allocation",
				ByteSize:              4,
				Escape:                allocplan.EscapeNoEscape,
				Storage:               allocplan.StorageStack,
				PlannedStorage:        allocplan.StorageStack,
				ActualLoweringStorage: allocplan.StorageStack,
				ValidationStatus:      "validated_no_escape",
				LoweringStatus:        "stack_lowering",
				Reason:                "test",
			}},
		}},
	}
	report := wrapAllocationPlanReport(plan, "linux-x64")
	report.Totals.Stack = 0

	err := validateAllocationPlanReport(plan, report)
	if err == nil || !strings.Contains(err.Error(), "allocation report mismatch") {
		t.Fatalf("validateAllocationPlanReport error = %v, want mismatch rejection", err)
	}
}

func TestBuildLayoutReportRecordsP21DefaultReprCAndExportDecisions(t *testing.T) {
	checked := &semantics.CheckedProgram{
		Structs: []semantics.CheckedStruct{
			{
				Name:   "main.Packet",
				Module: "main",
				Decl: &frontend.StructDecl{
					Name: "Packet",
					Repr: frontend.StructReprDefault,
					Fields: []frontend.FieldDecl{
						{Name: "tag"},
						{Name: "code"},
					},
				},
			},
			{
				Name:   "main.Header",
				Module: "main",
				Decl: &frontend.StructDecl{
					Name: "Header",
					Repr: frontend.StructReprC,
					Fields: []frontend.FieldDecl{
						{Name: "tag"},
						{Name: "code"},
					},
				},
			},
		},
		Types: map[string]*semantics.TypeInfo{
			"main.Packet": {
				Name:      "main.Packet",
				Kind:      semantics.TypeStruct,
				Repr:      frontend.StructReprDefault,
				SlotCount: 2,
				Fields: []semantics.FieldInfo{
					{Name: "tag", TypeName: "c_int", Offset: 0, SlotCount: 1},
					{Name: "code", TypeName: "c_int", Offset: 1, SlotCount: 1},
				},
			},
			"main.Header": {
				Name:      "main.Header",
				Kind:      semantics.TypeStruct,
				Repr:      frontend.StructReprC,
				SlotCount: 2,
				Fields: []semantics.FieldInfo{
					{Name: "tag", TypeName: "c_int", Offset: 0, SlotCount: 1},
					{Name: "code", TypeName: "c_int", Offset: 1, SlotCount: 1},
				},
			},
		},
		Funcs: []semantics.CheckedFunc{
			{
				Name:   "main.ffi_header",
				Module: "main",
				Decl:   &frontend.FuncDecl{Name: "ffi_header", ExportName: "ffi_header_c"},
			},
		},
		FuncSigs: map[string]semantics.FuncSig{
			"main.ffi_header": {
				ParamNames: []string{"header"},
				ParamTypes: []string{"main.Header"},
				ReturnType: "c_int",
			},
		},
	}

	report := buildLayoutReport("linux-x64", checked)
	if report.SchemaVersion != 2 || report.Kind != "layout" || report.Policy != p21LayoutPolicy {
		t.Fatalf("layout report header = %+v", report.reportEnvelope)
	}
	if report.Summary.Structs != 2 || report.Summary.DefaultCompilerOwned != 1 || report.Summary.ReprCABILocked != 1 || report.Summary.ExportedPublicABI != 1 {
		t.Fatalf("layout summary = %+v, want default/reprC/export counts", report.Summary)
	}
	byName := map[string]layoutDecisionRow{}
	for _, row := range report.Decisions {
		byName[row.Type] = row
	}
	packet := byName["main.Packet"]
	if packet.Decision != "compiler_owned_default" || packet.ABILocked || packet.PublicABI != "not_public_abi" {
		t.Fatalf("default packet row = %+v", packet)
	}
	for _, want := range []string{"field_reordering", "padding_removal", "hot_cold_splitting", "scalar_replacement", "aos_to_soa"} {
		if !containsString(packet.AllowedTransforms, want) {
			t.Fatalf("default packet allowed transforms = %+v, want %q", packet.AllowedTransforms, want)
		}
	}
	header := byName["main.Header"]
	if header.Decision != "abi_locked_repr_c" || !header.ABILocked || header.PublicABI != "exported_ffi_explicit_repr_c" {
		t.Fatalf("repr(C) header row = %+v", header)
	}
	if len(header.AllowedTransforms) != 0 || !containsString(header.DeniedTransforms, "field_reordering") {
		t.Fatalf("repr(C) transforms = allowed %+v denied %+v", header.AllowedTransforms, header.DeniedTransforms)
	}
	if err := ValidateLayoutReport(report); err != nil {
		t.Fatalf("ValidateLayoutReport: %v", err)
	}
}

func TestValidateLayoutReportRejectsFakeP21Decisions(t *testing.T) {
	report := layoutReport{
		reportEnvelope: reportEnvelope{SchemaVersion: 2, Kind: "layout", Target: "linux-x64"},
		Policy:         p21LayoutPolicy,
		Summary: layoutSummary{
			Structs:              2,
			DefaultCompilerOwned: 1,
			ReprCABILocked:       1,
			ExportedPublicABI:    1,
		},
		Decisions: []layoutDecisionRow{
			{
				Type:              "main.Packet",
				Repr:              frontend.StructReprDefault,
				Decision:          "compiler_owned_default",
				PublicABI:         "not_public_abi",
				AllowedTransforms: []string{"field_reordering", "padding_removal", "hot_cold_splitting", "scalar_replacement", "aos_to_soa"},
				Reason:            "default struct layout is compiler-owned",
			},
			{
				Type:             "main.Header",
				Repr:             frontend.StructReprC,
				ABILocked:        true,
				Decision:         "abi_locked_repr_c",
				PublicABI:        "exported_ffi_explicit_repr_c",
				DeniedTransforms: []string{"field_reordering", "padding_removal", "hot_cold_splitting", "scalar_replacement", "aos_to_soa"},
				Reason:           "repr(C) locks layout",
			},
		},
	}
	report.Decisions[1].AllowedTransforms = []string{"field_reordering"}
	err := ValidateLayoutReport(report)
	if err == nil || !strings.Contains(err.Error(), "repr(C)") {
		t.Fatalf("ValidateLayoutReport accepted repr(C) layout freedom: %v", err)
	}

	report = buildMinimalValidLayoutReportForTest()
	report.Decisions[0].ABILocked = true
	err = ValidateLayoutReport(report)
	if err == nil || !strings.Contains(err.Error(), "default") {
		t.Fatalf("ValidateLayoutReport accepted default ABI lock: %v", err)
	}

	report = buildMinimalValidLayoutReportForTest()
	report.Decisions[0].PublicABI = "exported_ffi_missing_explicit_repr"
	err = ValidateLayoutReport(report)
	if err == nil || !strings.Contains(err.Error(), "explicit repr(C)") {
		t.Fatalf("ValidateLayoutReport accepted exported default-layout ABI row: %v", err)
	}

	report = buildMinimalValidLayoutReportForTest()
	report.Decisions[0].PublicABI = "exported_ffi_explicit_repr_c"
	err = ValidateLayoutReport(report)
	if err == nil || !strings.Contains(err.Error(), "without repr(C)") {
		t.Fatalf("ValidateLayoutReport accepted spoofed explicit repr(C) ABI row: %v", err)
	}
}

func buildMinimalValidLayoutReportForTest() layoutReport {
	return layoutReport{
		reportEnvelope: reportEnvelope{SchemaVersion: 2, Kind: "layout", Target: "linux-x64"},
		Policy:         p21LayoutPolicy,
		Summary: layoutSummary{
			Structs:              2,
			DefaultCompilerOwned: 1,
			ReprCABILocked:       1,
			ExportedPublicABI:    1,
		},
		Decisions: []layoutDecisionRow{
			{
				Type:              "main.Packet",
				Repr:              frontend.StructReprDefault,
				Decision:          "compiler_owned_default",
				PublicABI:         "not_public_abi",
				AllowedTransforms: []string{"field_reordering", "padding_removal", "hot_cold_splitting", "scalar_replacement", "aos_to_soa"},
				Reason:            "default struct layout is compiler-owned",
			},
			{
				Type:             "main.Header",
				Repr:             frontend.StructReprC,
				ABILocked:        true,
				Decision:         "abi_locked_repr_c",
				PublicABI:        "exported_ffi_explicit_repr_c",
				DeniedTransforms: []string{"field_reordering", "padding_removal", "hot_cold_splitting", "scalar_replacement", "aos_to_soa"},
				Reason:           "repr(C) locks layout",
			},
		},
	}
}

func TestWrapAllocationPlanReportV2IncludesRuntimeSummary(t *testing.T) {
	plan := &allocplan.Plan{
		Totals: allocplan.Totals{Stack: 1, ExplicitIsland: 1},
		Functions: []allocplan.FunctionPlan{{
			Name: "main",
			Allocations: []allocplan.Allocation{
				{
					ID:                    "xs",
					SiteID:                "allocsite:main:xs:line_1_1",
					ValueID:               "alloc_intent:xs",
					Builtin:               "core.make_u8",
					ElementType:           "u8",
					ElementSize:           1,
					LengthExpr:            "32",
					LengthStatus:          allocplan.LengthStatusNormal,
					ByteSize:              32,
					Escape:                allocplan.EscapeNoEscape,
					Storage:               allocplan.StorageStack,
					PlannedStorage:        allocplan.StorageStack,
					ActualLoweringStorage: allocplan.StorageStack,
					ValidationStatus:      "validated_no_escape",
					LoweringStatus:        "stack_lowering",
					RuntimePath:           "stack_frame",
					BytesRequested:        32,
					BytesReserved:         32,
					Reason:                "test stack",
				},
				{
					ID:                    "ys",
					SiteID:                "allocsite:main:ys:line_2_1",
					ValueID:               "alloc_intent:ys",
					Builtin:               "core.island_make_u8",
					ElementType:           "u8",
					ElementSize:           1,
					LengthExpr:            "17",
					LengthStatus:          allocplan.LengthStatusNormal,
					ByteSize:              17,
					Escape:                allocplan.EscapeNoEscape,
					Storage:               allocplan.StorageExplicitIsland,
					PlannedStorage:        allocplan.StorageExplicitIsland,
					ActualLoweringStorage: allocplan.StorageExplicitIsland,
					ValidationStatus:      "validated_explicit_island_scope",
					LoweringStatus:        "explicit_island_lowering",
					RuntimePath:           "explicit_island",
					BytesRequested:        17,
					BytesReserved:         32,
					RegionID:              "island:isl",
					Lifetime:              "island:isl:scope",
					Reason:                "test island",
				},
			},
		}},
	}

	report := wrapAllocationPlanReport(plan, "linux-x64")
	if report.SchemaVersion != 2 {
		t.Fatalf("allocation report schema_version = %d, want 2", report.SchemaVersion)
	}
	if report.Summary.AllocationCount != 2 {
		t.Fatalf("allocation_count = %d, want 2", report.Summary.AllocationCount)
	}
	if report.Summary.StorageClasses[string(allocplan.StorageStack)] != 1 || report.Summary.StorageClasses[string(allocplan.StorageExplicitIsland)] != 1 {
		t.Fatalf("storage class summary = %+v, want Stack and ExplicitIsland counts", report.Summary.StorageClasses)
	}
	if report.Summary.ActualLoweringStorageClasses[string(allocplan.StorageStack)] != 1 || report.Summary.ActualLoweringStorageClasses[string(allocplan.StorageExplicitIsland)] != 1 {
		t.Fatalf("actual storage summary = %+v, want Stack and ExplicitIsland counts", report.Summary.ActualLoweringStorageClasses)
	}
	if report.Summary.RuntimePaths["stack_frame"] != 1 || report.Summary.RuntimePaths["explicit_island"] != 1 {
		t.Fatalf("runtime path summary = %+v, want stack_frame and explicit_island counts", report.Summary.RuntimePaths)
	}
	if report.Summary.BytesRequested != 49 || report.Summary.BytesReserved != 64 {
		t.Fatalf("byte summary = requested %d reserved %d, want 49/64", report.Summary.BytesRequested, report.Summary.BytesReserved)
	}
	if len(report.Summary.Regions) != 1 || report.Summary.Regions[0].RegionID != "island:isl" || report.Summary.Regions[0].Lifetime != "island:isl:scope" {
		t.Fatalf("regions summary = %+v, want island region", report.Summary.Regions)
	}
	if err := validateAllocationPlanReport(plan, report); err != nil {
		t.Fatalf("validateAllocationPlanReport: %v", err)
	}
}

func TestWrapAllocationPlanReportV2IncludesFunctionTempRegionSummary(t *testing.T) {
	plan := &allocplan.Plan{
		Totals: allocplan.Totals{FunctionTempRegion: 1},
		Functions: []allocplan.FunctionPlan{{
			Name: "local_copy",
			Allocations: []allocplan.Allocation{{
				ID:                    "copied",
				SiteID:                "allocsite:local_copy:copied:line_4_5",
				ValueID:               "alloc_intent:copied",
				Builtin:               "core.slice_copy_u8",
				ElementType:           "u8",
				ElementSize:           1,
				LengthExpr:            "n",
				LengthStatus:          allocplan.LengthStatusNormal,
				ByteSize:              0,
				Escape:                allocplan.EscapeNoEscape,
				Storage:               allocplan.StorageFunctionTempRegion,
				PlannedStorage:        allocplan.StorageFunctionTempRegion,
				ActualLoweringStorage: allocplan.StorageFunctionTempRegion,
				ValidationStatus:      "validated_function_temp_region_scope",
				LoweringStatus:        "function_temp_region_lowering",
				RuntimePath:           "region",
				AllocatorClass:        "function_temp_region",
				BytesRequested:        0,
				BytesReserved:         0,
				RegionID:              "region:local_copy:temp",
				Lifetime:              "function:local_copy",
				DebugMode:             "region_reset_when_enabled",
				Reason:                "function-local temporary copy lowers through region enter/reset IR",
			}},
		}},
	}

	report := wrapAllocationPlanReport(plan, "linux-x64")
	if report.Summary.StorageClasses["FunctionTempRegion"] != 1 ||
		report.Summary.ActualLoweringStorageClasses["FunctionTempRegion"] != 1 ||
		report.Summary.RuntimePaths["region"] != 1 {
		t.Fatalf("function-temp region summary missing region counts: %+v", report.Summary)
	}
	if len(report.Summary.Regions) != 1 {
		t.Fatalf("regions summary = %+v, want one function-temp region", report.Summary.Regions)
	}
	region := report.Summary.Regions[0]
	if region.RegionID != "region:local_copy:temp" ||
		region.Lifetime != "function:local_copy" ||
		region.StorageClass != "FunctionTempRegion" ||
		region.RuntimePath != "region" ||
		region.AllocationCount != 1 {
		t.Fatalf("function-temp region summary row = %+v", region)
	}
	if err := validateAllocationPlanReport(plan, report); err != nil {
		t.Fatalf("validateAllocationPlanReport: %v", err)
	}
}

func TestValidateAllocationPlanReportRejectsRuntimeSummaryMismatch(t *testing.T) {
	plan := &allocplan.Plan{
		Totals: allocplan.Totals{Heap: 1},
		Functions: []allocplan.FunctionPlan{{
			Name: "main",
			Allocations: []allocplan.Allocation{{
				ID:                    "xs",
				SiteID:                "allocsite:main:xs:line_1_1",
				ValueID:               "alloc_intent:xs",
				Builtin:               "core.make_u8",
				ElementType:           "u8",
				ElementSize:           1,
				LengthExpr:            "5000",
				LengthStatus:          allocplan.LengthStatusNormal,
				ByteSize:              5000,
				Escape:                allocplan.EscapeReturn,
				Storage:               allocplan.StorageHeap,
				PlannedStorage:        allocplan.StorageHeap,
				ActualLoweringStorage: allocplan.StorageHeap,
				ValidationStatus:      "validated_heap_fallback",
				LoweringStatus:        "large_mmap_runtime",
				RuntimePath:           "large_mmap",
				BytesRequested:        5000,
				BytesReserved:         5000,
				Reason:                "test heap",
			}},
		}},
	}
	report := wrapAllocationPlanReport(plan, "linux-x64")
	report.Summary.RuntimePaths["large_mmap"] = 0

	err := validateAllocationPlanReport(plan, report)
	if err == nil || !strings.Contains(err.Error(), "allocation report mismatch") {
		t.Fatalf("validateAllocationPlanReport error = %v, want summary mismatch rejection", err)
	}
}

func TestBackendReportListsRegisterAndStackFallbackPaths(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "add",
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "checked_get",
			ParamSlots:  3,
			LocalSlots:  3,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRIndexLoadI32},
				{Kind: ir.IRReturn},
			},
		},
	}})
	got := map[string]string{}
	for _, row := range report.Functions {
		got[row.Function] = row.BackendPath
	}
	if got["add"] != "register" {
		t.Fatalf("add backend_path = %q, want register (rows=%+v)", got["add"], report.Functions)
	}
	if got["checked_get"] != "stack" {
		t.Fatalf("checked_get backend_path = %q, want stack fallback (rows=%+v)", got["checked_get"], report.Functions)
	}
}

func TestBackendCoverageAuditClassifiesFallbackReasonsAndHotness(t *testing.T) {
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "response_cost",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "slice_return",
			ReturnSlots: 2,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "aggregate_return",
			ReturnSlots: 3,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "wide_call",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRConstI32, Imm: 3},
				{Kind: ir.IRConstI32, Imm: 4},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 6},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRCall, Name: "callee", ArgSlots: 7, RetSlots: 1},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "alloc_runtime",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 8},
				{Kind: ir.IRAllocBytes},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "branchy",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "checked_get",
			ParamSlots:  3,
			LocalSlots:  3,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRIndexLoadI32},
				{Kind: ir.IRReturn},
			},
		},
	}}
	report := buildBackendReport("linux-x64", prog)
	if len(report.Functions) != len(prog.Funcs) {
		t.Fatalf("backend coverage rows = %d, want one row per %d functions: %+v", len(report.Functions), len(prog.Funcs), report.Functions)
	}
	rows := backendRowsByFunction(report.Functions)
	assertBackendCoverageRow(t, rows["response_cost"], "register", "register_path", "eligible_machine_ir_subset")
	if rows["response_cost"].HotnessRank != 1 || rows["response_cost"].HotnessSource != "examples/benchmarks/techempower_plaintext_kernel.tetra" {
		t.Fatalf("response_cost hotness = rank %d source %q, want rank 1 from TechEmpower plaintext corpus row", rows["response_cost"].HotnessRank, rows["response_cost"].HotnessSource)
	}
	assertBackendCoverageRow(t, rows["slice_return"], "stack", "unsupported_slice_string_return", "unsupported_slice_or_string_return_uses_stack_fallback")
	assertBackendCoverageRow(t, rows["aggregate_return"], "stack", "unsupported_aggregate_return", "unsupported_aggregate_return_uses_stack_fallback")
	assertBackendCoverageRow(t, rows["wide_call"], "stack", "unsupported_call_abi", "unsupported_call_abi_uses_stack_fallback")
	assertBackendCoverageRow(t, rows["alloc_runtime"], "stack", "unsupported_effect_runtime_call", "unsupported_effect_runtime_call_uses_stack_fallback")
	assertBackendCoverageRow(t, rows["branchy"], "stack", "unsupported_control_flow", "unsupported_control_flow_uses_stack_fallback")
	assertBackendCoverageRow(t, rows["checked_get"], "stack", "stack_fallback", "unsupported_or_unproven_subset_uses_stack_fallback")
	if rows["checked_get"].HotnessRank != 0 || rows["checked_get"].HotnessSource != "not_in_benchmark_corpus" {
		t.Fatalf("checked_get hotness = rank %d source %q, want explicit non-corpus marker", rows["checked_get"].HotnessRank, rows["checked_get"].HotnessSource)
	}
}

func TestBackendCoverageSummaryCountsRowsAndCategories(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "add",
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "slice_return",
			ReturnSlots: 2,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRReturn},
			},
		},
	}})
	if report.Summary.FunctionCount != 2 || report.Summary.RegisterPath != 1 || report.Summary.StackFallback != 1 {
		t.Fatalf("backend summary = %+v, want one register row and one stack fallback row", report.Summary)
	}
	if report.Summary.Categories["register_path"] != 1 || report.Summary.Categories["unsupported_slice_string_return"] != 1 {
		t.Fatalf("backend summary categories = %+v, want register_path and unsupported_slice_string_return counts", report.Summary.Categories)
	}
	if report.Summary.HotnessSource != "benchmark-corpus-static-map" {
		t.Fatalf("backend summary hotness source = %q, want benchmark corpus source marker", report.Summary.HotnessSource)
	}
}

func TestBackendMachineReportsRequireSSAVerifiedPath(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "add",
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "sum",
			ParamSlots:  2,
			LocalSlots:  4,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 3},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRCmpLtI32},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 3},
				{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:while:test"},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRLoadLocal, Local: 3},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRStoreLocal, Local: 3},
				{Kind: ir.IRJmp, Label: 1},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRReturn},
			},
		},
	}})
	if len(report.MachineFunctions) != 2 {
		t.Fatalf("machine reports = %d, want add and slice sum paths: %+v", len(report.MachineFunctions), report.MachineFunctions)
	}
	for _, row := range report.MachineFunctions {
		if !row.SSAVerified || row.SSAPath != "value-ssa-v1" {
			t.Fatalf("machine report %s SSA gate = verified:%v path:%q, want value-ssa-v1 verified", row.Function, row.SSAVerified, row.SSAPath)
		}
	}
}

func TestBackendMachineReportIncludesDivModInstructionSelection(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:        "div_mod",
		ParamSlots:  2,
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRDivI32},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRModI32},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}}})
	rows := backendRowsByFunction(report.Functions)
	assertBackendCoverageRow(t, rows["div_mod"], "register", "register_path", "eligible_machine_ir_subset")
	if len(report.MachineFunctions) != 1 {
		t.Fatalf("machine reports = %+v, want div_mod machine report", report.MachineFunctions)
	}
	machineRow := report.MachineFunctions[0]
	for _, want := range []string{"div", "mod"} {
		if !containsReportString(machineRow.InstructionSelection, want) {
			t.Fatalf("instruction selection = %+v, want %s", machineRow.InstructionSelection, want)
		}
	}
	if machineRow.Validation.StackChurnOps != 0 || machineRow.Validation.MachineVerifier != "pass" || machineRow.Validation.AllocationVerifier != "pass" {
		t.Fatalf("machine validation = %+v, want verifier pass and no push/pop stack churn", machineRow.Validation)
	}
}

func TestBackendReportIncludesMultiSlotReturnABIBoundary(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "add",
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "slice_header_return",
			ReturnSlots: 2,
			Instrs:      []ir.IRInstr{{Kind: ir.IRReturn}},
		},
		{
			Name:        "aggregate_return",
			ReturnSlots: 3,
			Instrs:      []ir.IRInstr{{Kind: ir.IRReturn}},
		},
		{
			Name:        "call_returns_header",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRCall, Name: "slice_header_return", ArgSlots: 0, RetSlots: 2},
				{Kind: ir.IRReturn},
			},
		},
	}})
	rows := backendRowsByFunction(report.Functions)
	if rows["add"].ABI.MultiSlotReturnPolicy != "single_slot_register_return" || rows["add"].ABI.ReturnSlots != 1 {
		t.Fatalf("add ABI boundary = %+v, want single-slot register return", rows["add"].ABI)
	}
	if rows["slice_header_return"].ABI.MultiSlotReturnPolicy != "unsupported_multi_slot_return_stack_fallback" || rows["slice_header_return"].ABI.ReturnSlots != 2 {
		t.Fatalf("slice return ABI boundary = %+v, want unsupported multi-slot stack fallback", rows["slice_header_return"].ABI)
	}
	if rows["aggregate_return"].ABI.MultiSlotReturnPolicy != "unsupported_multi_slot_return_stack_fallback" || rows["aggregate_return"].ABI.ReturnSlots != 3 {
		t.Fatalf("aggregate return ABI boundary = %+v, want unsupported aggregate stack fallback", rows["aggregate_return"].ABI)
	}
	if rows["call_returns_header"].ABI.MultiSlotReturnPolicy != "unsupported_call_multi_slot_return_stack_fallback" {
		t.Fatalf("call multi-return ABI boundary = %+v, want unsupported call multi-slot fallback", rows["call_returns_header"].ABI)
	}
}

func TestBackendCoverageSummaryIncludesOrdinaryCorpusNoStackChurn(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{
		scalarCorpusFunc("response_cost", ir.IRAddI32),
		scalarCorpusFunc("flip_count", ir.IRMulI32),
		scalarCorpusFunc("safe_pair", ir.IRSubI32),
		{
			Name:        "branch",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRReturn},
			},
		},
	}})

	corpus := report.Summary.OrdinaryCorpus
	if corpus.FunctionCount != 4 || corpus.RegisterPath != 3 || corpus.RegisterNoStackChurn != 3 || corpus.StackFallback != 1 {
		t.Fatalf("ordinary corpus summary = %+v, want 4 functions, 3 register/no-churn, 1 fallback", corpus)
	}
	if !corpus.RegisterNoStackChurnMajority {
		t.Fatalf("ordinary corpus summary = %+v, want no-stack-churn majority", corpus)
	}
	if corpus.StackFallbackReasons["unsupported_control_flow"] != 1 {
		t.Fatalf("ordinary corpus fallback reasons = %+v, want unsupported_control_flow=1", corpus.StackFallbackReasons)
	}
	if report.Summary.MachineRegisterNoStackChurn != 3 || report.Summary.MachineRegisterWithStackChurn != 0 {
		t.Fatalf("machine no-stack-churn summary = %+v, want three register paths without push/pop churn", report.Summary)
	}
}

func TestBackendReportBoundsMultiSlotHeaderAndAggregateBoundaryEvidence(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "simple_pair_return",
			ReturnSlots: 2,
			Instrs:      []ir.IRInstr{{Kind: ir.IRReturn}},
		},
		{
			Name:        "string_header_return",
			ReturnSlots: 2,
			Instrs:      []ir.IRInstr{{Kind: ir.IRReturn}},
		},
		{
			Name:        "aggregate_return",
			ReturnSlots: 3,
			Instrs:      []ir.IRInstr{{Kind: ir.IRReturn}},
		},
		{
			Name:        "call_returns_header",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRCall, Name: "string_header_return", ArgSlots: 0, RetSlots: 2},
				{Kind: ir.IRReturn},
			},
		},
	}})
	rows := backendRowsByFunction(report.Functions)

	for _, name := range []string{"simple_pair_return", "string_header_return"} {
		row := rows[name]
		assertBackendCoverageRow(t, row, "stack", "unsupported_slice_string_return", "unsupported_slice_or_string_return_uses_stack_fallback")
		if row.ABI.ValueClass != "unverified_header_or_pair" || row.ABI.BoundaryStatus != "stack_fallback_until_multi_slot_abi_verified" {
			t.Fatalf("%s ABI boundary = %+v, want bounded unverified header-or-pair stack fallback", name, row.ABI)
		}
	}
	if rows["aggregate_return"].ABI.ValueClass != "unverified_aggregate" ||
		rows["aggregate_return"].ABI.BoundaryStatus != "stack_fallback_until_multi_slot_abi_verified" {
		t.Fatalf("aggregate ABI boundary = %+v, want bounded aggregate stack fallback", rows["aggregate_return"].ABI)
	}
	if rows["call_returns_header"].ABI.ValueClass != "callee_multi_slot_return_unverified" ||
		rows["call_returns_header"].ABI.BoundaryStatus != "stack_fallback_until_multi_slot_abi_verified" {
		t.Fatalf("call multi-slot ABI boundary = %+v, want bounded callee multi-slot fallback", rows["call_returns_header"].ABI)
	}
	if report.Summary.ABIBoundaries.MultiSlotReturnStackFallback != 3 ||
		report.Summary.ABIBoundaries.CallMultiSlotReturnStackFallback != 1 ||
		report.Summary.ABIBoundaries.ValueClasses["unverified_header_or_pair"] != 2 ||
		report.Summary.ABIBoundaries.ValueClasses["unverified_aggregate"] != 1 {
		t.Fatalf("ABI boundary summary = %+v, want bounded multi-slot/header/aggregate evidence", report.Summary.ABIBoundaries)
	}
}

func TestBackendMachineReportValidatesCallClobbersAndSpillReloadEvidence(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:        "apply",
		ParamSlots:  1,
		LocalSlots:  1,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCall, Name: "callee", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRReturn},
		},
	}}})
	if len(report.MachineFunctions) != 1 {
		t.Fatalf("machine reports = %+v, want call machine report", report.MachineFunctions)
	}
	callRow := report.MachineFunctions[0]
	if callRow.Validation.CallClobbers != "validated" ||
		callRow.Validation.SpillReload != "validated_no_spills" ||
		!containsReportString(callRow.InstructionSelection, "call") {
		t.Fatalf("call validation = %+v selection=%+v, want clobbers validated and no spills", callRow.Validation, callRow.InstructionSelection)
	}

	spillFn := machine.Function{
		Name:   "spill_reload_evidence",
		Target: "test",
		Params: []machine.VReg{"a"},
		Blocks: []machine.Block{{
			Name: "entry",
			Instrs: []machine.Instr{
				{Op: machine.OpSpill, Uses: []machine.VReg{"a"}, Imm: 0},
				{Op: machine.OpReload, Defs: []machine.VReg{"b"}, Imm: 0},
				{Op: machine.OpReturn, Uses: []machine.VReg{"b"}},
			},
		}},
	}
	spillRow, ok := buildMachineBackendFunctionReport(spillFn, "machine-ir-spill-reload-evidence", machine.LinuxX64CallerSaved(), true)
	if !ok {
		t.Fatalf("buildMachineBackendFunctionReport did not accept spill/reload evidence function")
	}
	if spillRow.Validation.SpillReload != "validated_spill_reload_ops" ||
		spillRow.Validation.CallClobbers != "not_applicable" ||
		spillRow.Validation.MachineVerifier != "pass" ||
		spillRow.Validation.AllocationVerifier != "pass" {
		t.Fatalf("spill/reload validation = %+v, want explicit spill/reload validation evidence", spillRow.Validation)
	}
}

func scalarCorpusFunc(name string, op ir.IRInstrKind) ir.IRFunc {
	return ir.IRFunc{
		Name:        name,
		ParamSlots:  2,
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: op},
			{Kind: ir.IRReturn},
		},
	}
}

func backendRowsByFunction(rows []backendFunctionPathReport) map[string]backendFunctionPathReport {
	out := make(map[string]backendFunctionPathReport, len(rows))
	for _, row := range rows {
		out[row.Function] = row
	}
	return out
}

func assertBackendCoverageRow(t *testing.T, row backendFunctionPathReport, path string, category string, reason string) {
	t.Helper()
	if row.BackendPath != path || row.Category != category || row.Reason != reason {
		t.Fatalf("backend row for %s = %+v, want path=%q category=%q reason=%q", row.Function, row, path, category, reason)
	}
}

func containsReportString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestBuildOptionsExposeNoBackendSemanticMode(t *testing.T) {
	buildOptionsType := reflect.TypeOf(BuildOptions{})
	for i := 0; i < buildOptionsType.NumField(); i++ {
		fieldName := strings.ToLower(buildOptionsType.Field(i).Name)
		for _, forbidden := range []string{"backend", "machine", "register", "pgo", "profile", "lto", "targetcpu", "target_cpu", "targetfeature"} {
			if strings.Contains(fieldName, forbidden) {
				t.Fatalf("BuildOptions exposes semantic tuning field %q; backend/profile/LTO/target-cpu selection must remain internal or evidence-only", buildOptionsType.Field(i).Name)
			}
		}
	}
	if nativeCodegenOptions(BuildOptions{}).DisableMachinePaths {
		t.Fatalf("native codegen options should not set DisableMachinePaths from public BuildOptions")
	}
}

func TestPerformanceReportIncludesBlockerDiagnostics(t *testing.T) {
	report := buildPerformanceReport("linux-x64")
	got := map[string]bool{}
	for _, blocker := range report.Blockers {
		got[blocker.Message] = true
		if blocker.Code == "" || blocker.Component == "" || blocker.Evidence == "" || blocker.CostClass == "" {
			t.Fatalf("incomplete blocker row: %+v", blocker)
		}
	}
	for _, want := range []string{
		"left bounds check: missing dominance",
		"heap allocation: escapes through return",
		"heap allocation: unknown call",
		"not vectorized: no noalias proof",
		"not inlined: code-size budget",
		"register spill: live range pressure",
		"stack fallback: unsupported aggregate return",
		"actor copy: borrowed data crosses boundary",
	} {
		if !got[want] {
			t.Fatalf("performance report missing blocker %q: %+v", want, report.Blockers)
		}
	}
	if len(report.Claims) == 0 || strings.Contains(strings.ToLower(strings.Join(report.Claims, " ")), "fastest language") {
		t.Fatalf("performance claims are not claim-disciplined: %+v", report.Claims)
	}
}

func TestPerformanceReportCoversP20BenchmarkBlockers(t *testing.T) {
	report := buildPerformanceReport("linux-x64")
	if report.SchemaVersion != 3 || report.MatrixScope != "p20.0_benchmark_matrix" {
		t.Fatalf("performance report schema/scope = %d/%q, want P20.1 schema over P20.0 matrix", report.SchemaVersion, report.MatrixScope)
	}
	if report.MatrixReport != "reports/benchmark-matrix-hardening-v1/benchmarks/p20-matrix-hardening-report.json" {
		t.Fatalf("matrix report path = %q", report.MatrixReport)
	}
	gotReasons := map[string]performanceBlockerRow{}
	for _, blocker := range report.Blockers {
		gotReasons[blocker.Code] = blocker
		if blocker.Message == "" || blocker.Evidence == "" || blocker.NextStep == "" || blocker.CostClass == "" {
			t.Fatalf("incomplete blocker row: %+v", blocker)
		}
	}
	for code, want := range map[string]struct {
		message   string
		costClass string
	}{
		"bounds.missing_dominance":                    {message: "left bounds check: missing dominance", costClass: "dynamic_check_required"},
		"allocation.return_escape":                    {message: "heap allocation: escapes through return", costClass: "conservative_fallback"},
		"allocation.unknown_call":                     {message: "heap allocation: unknown call", costClass: "conservative_fallback"},
		"vector.no_noalias_proof":                     {message: "not vectorized: no noalias proof", costClass: "dynamic_check_required"},
		"inline.code_size_budget":                     {message: "not inlined: code-size budget", costClass: "instrumentation_only"},
		"register_spill.live_range_pressure":          {message: "register spill: live range pressure", costClass: "instrumentation_only"},
		"stack_fallback.unsupported_aggregate_return": {message: "stack fallback: unsupported aggregate return", costClass: "conservative_fallback"},
		"actor_copy.borrowed_data_boundary":           {message: "actor copy: borrowed data crosses boundary", costClass: "conservative_fallback"},
	} {
		row, ok := gotReasons[code]
		if !ok {
			t.Fatalf("performance report missing P20.1 blocker code %q: %+v", code, report.Blockers)
		}
		if row.Message != want.message || row.CostClass != want.costClass {
			t.Fatalf("blocker %s = %+v, want message=%q cost_class=%q", code, row, want.message, want.costClass)
		}
	}
	gotBenchmarks := map[string]performanceBenchmarkExplanation{}
	for _, row := range report.Benchmarks {
		gotBenchmarks[row.Benchmark] = row
		if row.Category == "" || row.Explanation == "" || row.NextStep == "" {
			t.Fatalf("incomplete benchmark explanation row: %+v", row)
		}
		if row.MatrixScope != report.MatrixScope || row.MatrixReport != report.MatrixReport {
			t.Fatalf("benchmark row %s matrix linkage = %q/%q", row.Benchmark, row.MatrixScope, row.MatrixReport)
		}
		if len(row.ReasonCodes) == 0 {
			t.Fatalf("benchmark row %s missing reason codes", row.Benchmark)
		}
		for _, code := range row.ReasonCodes {
			if _, ok := gotReasons[code]; !ok {
				t.Fatalf("benchmark row %s cites unknown reason code %q", row.Benchmark, code)
			}
		}
	}
	for _, want := range []string{
		"integer_loops_tetra",
		"slice_sum_tetra",
		"bounds_check_loops_tetra",
		"function_calls_tetra",
		"recursion_tetra",
		"matrix_multiply_tetra",
		"hash_table_tetra",
		"allocation_tetra",
		"region_island_allocation_tetra",
		"json_parse_stringify_tetra",
		"http_plaintext_json_tetra",
		"postgresql_single_multiple_update_tetra",
		"actor_ping_pong_tetra",
		"parallel_map_reduce_tetra",
		"startup_time_tetra",
		"binary_size_tetra",
		"compile_time_tetra",
	} {
		if _, ok := gotBenchmarks[want]; !ok {
			t.Fatalf("performance report missing P20.0 benchmark explanation %q", want)
		}
	}
	if err := ValidatePerformanceBlockerReport(report); err != nil {
		t.Fatalf("ValidatePerformanceBlockerReport: %v", err)
	}
}

func TestValidatePerformanceBlockerReportRejectsWeakP20Evidence(t *testing.T) {
	report := buildPerformanceReport("linux-x64")
	report.Blockers = report.Blockers[:len(report.Blockers)-1]
	err := ValidatePerformanceBlockerReport(report)
	if err == nil || !strings.Contains(err.Error(), "actor_copy.borrowed_data_boundary") {
		t.Fatalf("accepted report missing actor-copy blocker: %v", err)
	}

	report = buildPerformanceReport("linux-x64")
	report.Benchmarks = report.Benchmarks[:len(report.Benchmarks)-1]
	err = ValidatePerformanceBlockerReport(report)
	if err == nil || !strings.Contains(err.Error(), "compile_time_tetra") {
		t.Fatalf("accepted report missing compile-time benchmark explanation: %v", err)
	}

	report = buildPerformanceReport("linux-x64")
	report.Benchmarks[0].ReasonCodes = []string{"unknown.reason"}
	err = ValidatePerformanceBlockerReport(report)
	if err == nil || !strings.Contains(err.Error(), "unknown reason code") {
		t.Fatalf("accepted unknown benchmark reason code: %v", err)
	}

	report = buildPerformanceReport("linux-x64")
	report.Benchmarks[0].Explanation = "TODO"
	err = ValidatePerformanceBlockerReport(report)
	if err == nil || !strings.Contains(err.Error(), "placeholder") {
		t.Fatalf("accepted placeholder explanation: %v", err)
	}

	report = buildPerformanceReport("linux-x64")
	report.Claims = []string{"This proves C++/Rust parity and measured speed superiority."}
	err = ValidatePerformanceBlockerReport(report)
	if err == nil || !strings.Contains(err.Error(), "C++/Rust parity") {
		t.Fatalf("accepted fake performance claim: %v", err)
	}

	report = buildPerformanceReport("linux-x64")
	report.Blockers[0].CostClass = "mystery_cost"
	err = ValidatePerformanceBlockerReport(report)
	if err == nil || !strings.Contains(err.Error(), "unknown cost_class") {
		t.Fatalf("accepted unknown cost class: %v", err)
	}

	report = buildPerformanceReport("linux-x64")
	report.Claims = append(report.Claims, "dynamic_check_required rows prove zero-cost bounds_check_eliminated")
	err = ValidatePerformanceBlockerReport(report)
	if err == nil || !strings.Contains(err.Error(), "dynamic_check_required") {
		t.Fatalf("accepted fake dynamic zero-cost claim: %v", err)
	}

	report = buildPerformanceReport("linux-x64")
	report.Claims = append(report.Claims, "unsafe_unknown is optimized as trusted storage")
	err = ValidatePerformanceBlockerReport(report)
	if err == nil || !strings.Contains(err.Error(), "unsafe_unknown") {
		t.Fatalf("accepted trusted unsafe_unknown claim: %v", err)
	}
}
