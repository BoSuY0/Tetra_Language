package validation

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/ir"
)

func TestValidateAllocationLoweringAcceptsFunctionTempRegionIR(t *testing.T) {
	plan := functionTempRegionValidationPlan("main")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:        "main",
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRRegionEnter},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRRegionMakeSliceU8, Name: "copied"},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRRegionReset},
			{Kind: ir.IRReturn},
		},
	}}}
	if err := ValidateAllocationLowering(plan, prog); err != nil {
		t.Fatalf("ValidateAllocationLowering: %v", err)
	}
}

func functionTempRegionValidationPlan(functionName string) *allocplan.Plan {
	return &allocplan.Plan{Functions: []allocplan.FunctionPlan{{
		Name: functionName,
		Allocations: []allocplan.Allocation{{
			ID:                    "copied",
			SiteID:                "allocsite:" + functionName + ":copied:line_1_1",
			ValueID:               "alloc_intent:copied",
			Builtin:               "core.slice_copy_u8",
			ElementType:           "u8",
			ElementSize:           1,
			LengthExpr:            "4",
			LengthStatus:          allocplan.LengthStatusNormal,
			ZeroGuardStatus:       "valid_empty_no_allocator",
			NegativeGuardStatus:   "reject_before_allocation",
			OverflowGuardStatus:   "reject_before_allocation",
			ByteSize:              4,
			Escape:                allocplan.EscapeNoEscape,
			Storage:               allocplan.StorageFunctionTempRegion,
			PlannedStorage:        allocplan.StorageFunctionTempRegion,
			ActualLoweringStorage: allocplan.StorageFunctionTempRegion,
			ValidationStatus:      "validated_function_temp_region_scope",
			LoweringStatus:        "function_temp_region_lowering",
			Reason:                "test",
			RuntimePath:           "region",
			AllocatorClass:        "function_temp_region",
			RegionID:              "region:" + functionName + ":temp",
			Lifetime:              "function:" + functionName,
		}},
	}}}
}

func TestValidateAllocationLoweringAcceptsScalarReplacementWithoutStackIR(t *testing.T) {
	plan := &allocplan.Plan{Functions: []allocplan.FunctionPlan{{
		Name: "main",
		Allocations: []allocplan.Allocation{{
			ID:                    "xs",
			SiteID:                "allocsite:main:xs:line_1_1",
			ValueID:               "alloc_intent:xs",
			Builtin:               "core.make_i32",
			ElementType:           "i32",
			ElementSize:           4,
			LengthExpr:            "2",
			LengthStatus:          allocplan.LengthStatusNormal,
			ZeroGuardStatus:       "valid_empty_no_allocator",
			NegativeGuardStatus:   "reject_before_allocation",
			OverflowGuardStatus:   "reject_before_allocation",
			ByteSize:              8,
			Escape:                allocplan.EscapeNoEscape,
			Storage:               allocplan.StorageEliminated,
			PlannedStorage:        allocplan.StorageEliminated,
			ActualLoweringStorage: allocplan.StorageEliminated,
			ValidationStatus:      "validated_no_escape",
			LoweringStatus:        "scalar_replacement",
			Reason:                "scalar_replacement_fixed_constant_indices",
		}},
	}}}
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:        "main",
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 20},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}}}
	if err := ValidateAllocationLowering(plan, prog); err != nil {
		t.Fatalf("ValidateAllocationLowering: %v", err)
	}
}

func TestValidateTranslationRequiresSameVerifiedFunctionSet(t *testing.T) {
	before := validEmptyIR("main")
	after := validEmptyIR("main")
	report, err := ValidateTranslation(before, after)
	if err != nil {
		t.Fatalf("ValidateTranslation: %v", err)
	}
	if report.FunctionsCompared != 1 || report.Functions[0] != "main" {
		t.Fatalf("translation report = %+v", report)
	}
	badAfter := validEmptyIR("other")
	_, err = ValidateTranslation(before, badAfter)
	if err == nil || !strings.Contains(err.Error(), "function set changed") {
		t.Fatalf("ValidateTranslation mismatch error = %v, want function set changed", err)
	}
}

func TestValidateTranslationAcceptsLocalAlgebraEquivalentRewrite(t *testing.T) {
	before := singleReturnIR("main", 1,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 0},
		ir.IRInstr{Kind: ir.IRAddI32},
	)
	after := singleReturnIR("main", 1,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
	)
	report, err := ValidateTranslation(before, after)
	if err != nil {
		t.Fatalf("ValidateTranslation equivalent algebra: %v", err)
	}
	if report.SemanticLocalChecks == 0 || report.DifferentialSamples == 0 {
		t.Fatalf("translation report missing semantic/differential evidence: %+v", report)
	}
}

func TestValidateTranslationAcceptsCommutativeLocalAlgebraRewrite(t *testing.T) {
	before := singleReturnIR("main", 2,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRAddI32},
	)
	after := singleReturnIR("main", 2,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRAddI32},
	)
	report, err := ValidateTranslation(before, after)
	if err != nil {
		t.Fatalf("ValidateTranslation commutative algebra: %v", err)
	}
	if report.SemanticLocalChecks == 0 || report.DifferentialSamples == 0 {
		t.Fatalf("translation report missing semantic/differential evidence: %+v", report)
	}
}

func TestValidateTranslationAcceptsMirroredComparisonAlgebraRewrite(t *testing.T) {
	tests := []struct {
		name   string
		before ir.IRInstrKind
		after  ir.IRInstrKind
	}{
		{name: "lt-to-gt", before: ir.IRCmpLtI32, after: ir.IRCmpGtI32},
		{name: "gt-to-lt", before: ir.IRCmpGtI32, after: ir.IRCmpLtI32},
		{name: "le-to-ge", before: ir.IRCmpLeI32, after: ir.IRCmpGeI32},
		{name: "ge-to-le", before: ir.IRCmpGeI32, after: ir.IRCmpLeI32},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			before := singleReturnIR("main", 2,
				ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
				ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
				ir.IRInstr{Kind: tc.before},
			)
			after := singleReturnIR("main", 2,
				ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
				ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
				ir.IRInstr{Kind: tc.after},
			)
			report, err := ValidateTranslation(before, after)
			if err != nil {
				t.Fatalf("ValidateTranslation mirrored comparison algebra: %v", err)
			}
			if report.SemanticLocalChecks == 0 || report.DifferentialSamples == 0 {
				t.Fatalf("translation report missing semantic/differential evidence: %+v", report)
			}
		})
	}
}

func TestValidateTranslationAcceptsSameLocalComparisonAlgebraRewrite(t *testing.T) {
	tests := []struct {
		name  string
		kind  ir.IRInstrKind
		value int32
	}{
		{name: "eq", kind: ir.IRCmpEqI32, value: 1},
		{name: "le", kind: ir.IRCmpLeI32, value: 1},
		{name: "ge", kind: ir.IRCmpGeI32, value: 1},
		{name: "ne", kind: ir.IRCmpNeI32, value: 0},
		{name: "lt", kind: ir.IRCmpLtI32, value: 0},
		{name: "gt", kind: ir.IRCmpGtI32, value: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			before := singleReturnIR("main", 1,
				ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
				ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
				ir.IRInstr{Kind: tc.kind},
			)
			after := singleReturnIR("main", 1,
				ir.IRInstr{Kind: ir.IRConstI32, Imm: tc.value},
			)
			report, err := ValidateTranslation(before, after)
			if err != nil {
				t.Fatalf("ValidateTranslation same-local comparison algebra: %v", err)
			}
			if report.SemanticLocalChecks == 0 || report.DifferentialSamples == 0 {
				t.Fatalf("translation report missing semantic/differential evidence: %+v", report)
			}
		})
	}
}

func TestValidateTranslationRejectsNonCommutativeLocalAlgebraRewrite(t *testing.T) {
	before := singleReturnIR("main", 2,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRSubI32},
	)
	after := singleReturnIR("main", 2,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRSubI32},
	)
	_, err := ValidateTranslation(before, after)
	if err == nil || !strings.Contains(err.Error(), "semantic local equivalence") {
		t.Fatalf(
			"ValidateTranslation non-commutative algebra error = %v, want semantic mismatch",
			err,
		)
	}
}

func TestValidateTranslationRejectsOppositeComparisonRewrite(t *testing.T) {
	before := singleReturnIR("main", 2,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRCmpLtI32},
	)
	after := singleReturnIR("main", 2,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRCmpGeI32},
	)
	_, err := ValidateTranslation(before, after)
	if err == nil || !strings.Contains(err.Error(), "semantic local equivalence") {
		t.Fatalf("ValidateTranslation opposite comparison error = %v, want semantic mismatch", err)
	}
}

func TestValidateTranslationRejectsBadLocalAlgebraRewrite(t *testing.T) {
	before := singleReturnIR("main", 1,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 0},
		ir.IRInstr{Kind: ir.IRAddI32},
	)
	after := singleReturnIR("main", 1,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 1},
		ir.IRInstr{Kind: ir.IRAddI32},
	)
	_, err := ValidateTranslation(before, after)
	if err == nil || !strings.Contains(err.Error(), "semantic local equivalence") {
		t.Fatalf("ValidateTranslation bad algebra error = %v, want semantic mismatch", err)
	}
}

func TestValidateTranslationRejectsProofFactDrift(t *testing.T) {
	before := proofLoadIR("main", "proof:while:i:xs:1:1", ir.IRIndexLoadI32Unchecked)
	after := proofLoadIR("main", "proof:while:i:ys:1:1", ir.IRIndexLoadI32Unchecked)
	_, err := ValidateTranslation(before, after)
	if err == nil || !strings.Contains(err.Error(), "proof facts changed") {
		t.Fatalf("ValidateTranslation proof drift error = %v, want proof facts changed", err)
	}
}

func TestValidateTranslationRejectsMissingProofIDAfterTransform(t *testing.T) {
	before := proofLoadIR("main", "proof:while:i:xs:1:1", ir.IRIndexLoadI32Unchecked)
	after := proofLoadIR("main", "", ir.IRIndexLoadI32Unchecked)
	_, err := ValidateTranslation(before, after)
	if err == nil || !strings.Contains(err.Error(), "missing proof id") {
		t.Fatalf("ValidateTranslation missing proof error = %v, want missing proof id", err)
	}
}

func TestValidateTranslationRejectsDifferentialMismatch(t *testing.T) {
	before := singleReturnIR("main", 2,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRDivI32},
	)
	after := singleReturnIR("main", 2,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRAddI32},
	)
	_, err := ValidateTranslation(before, after)
	if err == nil || !strings.Contains(err.Error(), "differential mismatch") {
		t.Fatalf("ValidateTranslation differential error = %v, want differential mismatch", err)
	}
}

func TestVerifierMapNamesRequiredStages(t *testing.T) {
	stages := map[Stage]bool{}
	for _, item := range VerifierMap() {
		stages[item.Stage] = item.Implemented
	}
	for _, stage := range []Stage{
		StageTypedAST,
		StagePLIR,
		StageProofFacts,
		StageOptimizedIR,
		StageAllocationPlan,
		StageMachineIR,
		StageABI,
		StageObjectSmoke,
	} {
		if !stages[stage] {
			t.Fatalf("VerifierMap missing implemented stage %s: %+v", stage, VerifierMap())
		}
	}
}

func validEmptyIR(name string) *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  name,
		Funcs: []ir.IRFunc{{
			Name:       name,
			LocalSlots: 0,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func singleReturnIR(name string, params int, body ...ir.IRInstr) *ir.IRProgram {
	instrs := append([]ir.IRInstr(nil), body...)
	instrs = append(instrs, ir.IRInstr{Kind: ir.IRReturn})
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  name,
		Funcs: []ir.IRFunc{{
			Name:        name,
			ParamSlots:  params,
			LocalSlots:  params,
			ReturnSlots: 1,
			Instrs:      instrs,
		}},
	}
}

func proofLoadIR(name string, proofID string, kind ir.IRInstrKind) *ir.IRProgram {
	return singleReturnIR(name, 3,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 2},
		ir.IRInstr{Kind: kind, ProofID: proofID},
	)
}
