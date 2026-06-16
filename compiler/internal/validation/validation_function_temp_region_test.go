package validation

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestValidateAllocationLoweringRejectsMissingFunctionTempRegionIR(t *testing.T) {
	plan := functionTempRegionValidationPlan("bad")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "bad",
		LocalSlots: 2,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRMakeSliceU8},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRReturn},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "no matching IR function-temp region slice") {
		t.Fatalf("ValidateAllocationLowering error = %v, want missing function-temp region IR rejection", err)
	}
}

func TestValidateAllocationLoweringRejectsReturnedFunctionTempRegionAllocation(t *testing.T) {
	plan := functionTempRegionValidationPlan("bad")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:        "bad",
		LocalSlots:  2,
		ReturnSlots: 2,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRRegionEnter},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRRegionMakeSliceU8, Name: "copied"},
			{Kind: ir.IRReturn},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "escapes via return") {
		t.Fatalf("ValidateAllocationLowering error = %v, want returned region allocation rejection", err)
	}
}

func TestValidateAllocationLoweringRejectsFunctionTempRegionResetThatDoesNotDominateBranchReturn(t *testing.T) {
	plan := functionTempRegionValidationPlan("branchy")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:        "branchy",
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRRegionEnter},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRRegionMakeSliceU8, Name: "copied"},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRRegionReset},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "function-temp region reset does not dominate return") {
		t.Fatalf("ValidateAllocationLowering error = %v, want branch reset-dominance rejection", err)
	}
}

func TestValidateAllocationLoweringRejectsFunctionTempRegionMakeWithoutEnter(t *testing.T) {
	plan := functionTempRegionValidationPlan("no_enter")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:        "no_enter",
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRRegionMakeSliceU8, Name: "copied"},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRRegionReset},
			{Kind: ir.IRReturn},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "function-temp region enter does not dominate make") {
		t.Fatalf("ValidateAllocationLowering error = %v, want missing region-enter rejection", err)
	}
}

func TestValidateAllocationLoweringRejectsFunctionTempRegionLoopExitWithoutReset(t *testing.T) {
	plan := functionTempRegionValidationPlan("loop_exit")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:        "loop_exit",
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRRegionEnter},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRRegionMakeSliceU8, Name: "copied"},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "function-temp region reset does not dominate return") {
		t.Fatalf("ValidateAllocationLowering error = %v, want loop-exit reset-dominance rejection", err)
	}
}

func TestValidateAllocationLoweringAcceptsFunctionTempRegionResetAfterLoopExit(t *testing.T) {
	plan := functionTempRegionValidationPlan("loop_exit")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:        "loop_exit",
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRRegionEnter},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRRegionMakeSliceU8, Name: "copied"},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRRegionReset},
			{Kind: ir.IRReturn},
		},
	}}}
	if err := ValidateAllocationLowering(plan, prog); err != nil {
		t.Fatalf("ValidateAllocationLowering: %v", err)
	}
}
