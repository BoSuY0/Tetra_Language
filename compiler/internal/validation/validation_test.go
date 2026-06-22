package validation

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/semantics"
)

func TestCheckBoundsProofsRejectsRemovedCheckWithoutProofID(t *testing.T) {
	prog := &ir.IRProgram{MainName: "bad", Funcs: []ir.IRFunc{{
		Name: "main",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRIndexLoadI32Unchecked},
		},
	}}}
	_, err := CheckBoundsProofs(prog)
	if err == nil || !strings.Contains(err.Error(), "without proof id") {
		t.Fatalf("CheckBoundsProofs error = %v, want missing proof id", err)
	}
}

func TestCheckBoundsProofsReportsRemovedAndLeftChecks(t *testing.T) {
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name: "main",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:loop"},
			{Kind: ir.IRIndexLoadI32},
		},
	}}}
	report, err := CheckBoundsProofs(prog)
	if err != nil {
		t.Fatalf("CheckBoundsProofs: %v", err)
	}
	if len(report.RemovedChecks) != 1 || report.LeftChecks != 1 {
		t.Fatalf("proof report = %+v, want one removed and one left", report)
	}
	if got := report.RemovedChecks[0].FactsUsed; len(got) != 2 {
		t.Fatalf("removed check facts = %v, want proof fact names", got)
	}
}

func TestCheckBoundsProofsWithPLIRAcceptsLiveDominatingProof(t *testing.T) {
	irProg := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name: "main",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:while:i:xs:1:1"},
		},
	}}}
	plirProg := &plir.Program{Funcs: []plir.Function{{
		Name: "main",
		Values: []plir.Value{{
			ID:         "local:i",
			Type:       "i32",
			Provenance: plir.Provenance{Kind: plir.ProvenanceStack, Root: "i"},
		}},
		Blocks: []plir.BasicBlock{
			{ID: "entry", Kind: "entry", Entry: true, Succs: []string{"body"}},
			{
				ID:    "body",
				Kind:  "while_body",
				Preds: []string{"entry"},
				Ops:   []string{"op0"},
				Exit:  true,
			},
		},
		Ops: []plir.Operation{
			{ID: "op0", Kind: plir.OpIndexLoad, Block: "body"},
		},
		Facts: []plir.Fact{{
			ID:      "f0",
			Kind:    plir.FactIndexInRange,
			ValueID: "local:i",
			Range:   "0..xs.len",
			ProofID: "proof:while:i:xs:1:1",
			Source:  "test:1:1",
		}},
		ProofGuards: []plir.ProofGuard{{
			ID:        "proof:while:i:xs:1:1",
			Kind:      "range",
			Block:     "body",
			OpID:      "op0",
			Condition: "i < xs.len",
		}},
		ProofUses: []plir.ProofUse{{
			ProofID: "proof:while:i:xs:1:1",
			Block:   "body",
			OpID:    "op0",
			UseKind: "bounds_check",
		}},
		ProofTerms: []plir.ProofTerm{{
			ID:            "proof:while:i:xs:1:1",
			Kind:          "bounds_check",
			SubjectBaseID: "xs",
			IndexValueID:  "local:i",
			Operation:     "index_load",
			Range:         "0..xs.len",
		}},
	}}}
	report, err := CheckBoundsProofsWithPLIR(irProg, plirProg)
	if err != nil {
		t.Fatalf("CheckBoundsProofsWithPLIR: %v", err)
	}
	if len(report.RemovedChecks) != 1 {
		t.Fatalf("report = %+v, want one removed check", report)
	}
}

func TestCheckBoundsProofsWithPLIRRejectsUnknownLiveProof(t *testing.T) {
	irProg := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name: "main",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:missing"},
		},
	}}}
	plirProg := &plir.Program{Funcs: []plir.Function{{Name: "main"}}}
	_, err := CheckBoundsProofsWithPLIR(irProg, plirProg)
	if err == nil || !strings.Contains(err.Error(), "not found in PLIR proof guards") {
		t.Fatalf("CheckBoundsProofsWithPLIR error = %v, want missing live proof", err)
	}
}

func TestCheckBoundsProofsWithPLIRRejectsGuardWithoutTypedProofTerm(t *testing.T) {
	irProg := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name: "main",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:while:i:xs:1:1"},
		},
	}}}
	plirProg := &plir.Program{Funcs: []plir.Function{{
		Name: "main",
		Values: []plir.Value{{
			ID:         "local:i",
			Type:       "i32",
			Provenance: plir.Provenance{Kind: plir.ProvenanceStack, Root: "i"},
		}},
		Blocks: []plir.BasicBlock{
			{ID: "body", Kind: "while_body", Entry: true, Exit: true, Ops: []string{"op0"}},
		},
		Ops: []plir.Operation{{ID: "op0", Kind: plir.OpIndexLoad, Block: "body"}},
		Facts: []plir.Fact{{
			ID:      "f0",
			Kind:    plir.FactIndexInRange,
			ValueID: "local:i",
			Range:   "0..xs.len",
			ProofID: "proof:while:i:xs:1:1",
			Source:  "test:1:1",
		}},
		ProofGuards: []plir.ProofGuard{{
			ID:        "proof:while:i:xs:1:1",
			Kind:      "range",
			Block:     "body",
			OpID:      "op0",
			Condition: "i < xs.len",
		}},
		ProofUses: []plir.ProofUse{{
			ProofID: "proof:while:i:xs:1:1",
			Block:   "body",
			OpID:    "op0",
			UseKind: "bounds_check",
		}},
	}}}
	_, err := CheckBoundsProofsWithPLIR(irProg, plirProg)
	if err == nil || !strings.Contains(err.Error(), "typed proof term") {
		t.Fatalf(
			"CheckBoundsProofsWithPLIR error = %v, want missing typed proof term rejection",
			err,
		)
	}
}

func TestCheckBoundsProofsWithPLIRRejectsTypedProofBaseMismatch(t *testing.T) {
	irProg := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name: "main",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:while:i:xs:1:1"},
		},
	}}}
	plirProg := &plir.Program{Funcs: []plir.Function{{
		Name: "main",
		Values: []plir.Value{{
			ID:         "local:i",
			Type:       "i32",
			Provenance: plir.Provenance{Kind: plir.ProvenanceStack, Root: "i"},
		}},
		Blocks: []plir.BasicBlock{
			{ID: "body", Kind: "while_body", Entry: true, Exit: true, Ops: []string{"op0"}},
		},
		Ops: []plir.Operation{{ID: "op0", Kind: plir.OpIndexLoad, Block: "body"}},
		Facts: []plir.Fact{{
			ID:      "f0",
			Kind:    plir.FactIndexInRange,
			ValueID: "local:i",
			Range:   "0..xs.len",
			ProofID: "proof:while:i:xs:1:1",
			Source:  "test:1:1",
		}},
		ProofGuards: []plir.ProofGuard{{
			ID:        "proof:while:i:xs:1:1",
			Kind:      "range",
			Block:     "body",
			OpID:      "op0",
			Condition: "i < xs.len",
		}},
		ProofUses: []plir.ProofUse{{
			ProofID: "proof:while:i:xs:1:1",
			Block:   "body",
			OpID:    "op0",
			UseKind: "bounds_check",
		}},
		ProofTerms: []plir.ProofTerm{{
			ID:            "proof:while:i:xs:1:1",
			Kind:          "bounds_check",
			SubjectBaseID: "ys",
			IndexValueID:  "local:i",
			Operation:     "index_load",
			Range:         "0..ys.len",
		}},
	}}}
	_, err := CheckBoundsProofsWithPLIR(irProg, plirProg)
	if err == nil || !strings.Contains(err.Error(), "subject base") {
		t.Fatalf("CheckBoundsProofsWithPLIR error = %v, want subject-base mismatch rejection", err)
	}
}

func TestCheckBoundsProofsWithPLIRRejectsProofWithoutUse(t *testing.T) {
	irProg := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name: "main",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:while:i:xs:1:1"},
		},
	}}}
	plirProg := &plir.Program{Funcs: []plir.Function{{
		Name: "main",
		Values: []plir.Value{{
			ID:         "local:i",
			Type:       "i32",
			Provenance: plir.Provenance{Kind: plir.ProvenanceStack, Root: "i"},
		}},
		Blocks: []plir.BasicBlock{
			{ID: "body", Kind: "while_body", Entry: true, Exit: true, Ops: []string{"op0"}},
		},
		Ops: []plir.Operation{{ID: "op0", Kind: plir.OpIndexLoad, Block: "body"}},
		Facts: []plir.Fact{{
			ID:      "f0",
			Kind:    plir.FactIndexInRange,
			ValueID: "local:i",
			Range:   "0..xs.len",
			ProofID: "proof:while:i:xs:1:1",
			Source:  "test:1:1",
		}},
		ProofGuards: []plir.ProofGuard{{
			ID:        "proof:while:i:xs:1:1",
			Kind:      "range",
			Block:     "body",
			OpID:      "op0",
			Condition: "i < xs.len",
		}},
		ProofTerms: []plir.ProofTerm{{
			ID:            "proof:while:i:xs:1:1",
			Kind:          "bounds_check",
			SubjectBaseID: "xs",
			IndexValueID:  "local:i",
			Operation:     "index_load",
			Range:         "0..xs.len",
		}},
	}}}
	_, err := CheckBoundsProofsWithPLIR(irProg, plirProg)
	if err == nil || !strings.Contains(err.Error(), "proof use") {
		t.Fatalf("CheckBoundsProofsWithPLIR error = %v, want missing proof use rejection", err)
	}
}

func TestCheckBoundsProofsWithPLIRRejectsTypedProofOperationMismatch(t *testing.T) {
	irProg := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name: "main",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:while:i:xs:1:1"},
		},
	}}}
	plirProg := &plir.Program{Funcs: []plir.Function{{
		Name: "main",
		Values: []plir.Value{{
			ID:         "local:i",
			Type:       "i32",
			Provenance: plir.Provenance{Kind: plir.ProvenanceStack, Root: "i"},
		}},
		Blocks: []plir.BasicBlock{
			{ID: "body", Kind: "while_body", Entry: true, Exit: true, Ops: []string{"op0"}},
		},
		Ops: []plir.Operation{{ID: "op0", Kind: plir.OpIndexLoad, Block: "body"}},
		Facts: []plir.Fact{{
			ID:      "f0",
			Kind:    plir.FactIndexInRange,
			ValueID: "local:i",
			Range:   "0..xs.len",
			ProofID: "proof:while:i:xs:1:1",
			Source:  "test:1:1",
		}},
		ProofGuards: []plir.ProofGuard{{
			ID:        "proof:while:i:xs:1:1",
			Kind:      "range",
			Block:     "body",
			OpID:      "op0",
			Condition: "i < xs.len",
		}},
		ProofUses: []plir.ProofUse{{
			ProofID: "proof:while:i:xs:1:1",
			Block:   "body",
			OpID:    "op0",
			UseKind: "bounds_check",
		}},
		ProofTerms: []plir.ProofTerm{{
			ID:            "proof:while:i:xs:1:1",
			Kind:          "bounds_check",
			SubjectBaseID: "xs",
			IndexValueID:  "local:i",
			Operation:     "index_store",
			Range:         "0..xs.len",
		}},
	}}}
	_, err := CheckBoundsProofsWithPLIR(irProg, plirProg)
	if err == nil || !strings.Contains(err.Error(), "operation") {
		t.Fatalf("CheckBoundsProofsWithPLIR error = %v, want operation mismatch rejection", err)
	}
}

func TestCheckBoundsProofsWithPLIRRejectsProofUseOperationMismatch(t *testing.T) {
	irProg := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name: "main",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:while:i:xs:1:1"},
		},
	}}}
	plirProg := &plir.Program{Funcs: []plir.Function{{
		Name: "main",
		Values: []plir.Value{{
			ID:         "local:i",
			Type:       "i32",
			Provenance: plir.Provenance{Kind: plir.ProvenanceStack, Root: "i"},
		}},
		Blocks: []plir.BasicBlock{
			{ID: "body", Kind: "while_body", Entry: true, Exit: true, Ops: []string{"op0"}},
		},
		Ops: []plir.Operation{{ID: "op0", Kind: plir.OpIndexStore, Block: "body"}},
		Facts: []plir.Fact{{
			ID:      "f0",
			Kind:    plir.FactIndexInRange,
			ValueID: "local:i",
			Range:   "0..xs.len",
			ProofID: "proof:while:i:xs:1:1",
			Source:  "test:1:1",
		}},
		ProofGuards: []plir.ProofGuard{{
			ID:        "proof:while:i:xs:1:1",
			Kind:      "range",
			Block:     "body",
			OpID:      "op0",
			Condition: "i < xs.len",
		}},
		ProofUses: []plir.ProofUse{{
			ProofID: "proof:while:i:xs:1:1",
			Block:   "body",
			OpID:    "op0",
			UseKind: "bounds_check",
		}},
		ProofTerms: []plir.ProofTerm{{
			ID:            "proof:while:i:xs:1:1",
			Kind:          "bounds_check",
			SubjectBaseID: "xs",
			IndexValueID:  "local:i",
			Operation:     "index_load",
			Range:         "0..xs.len",
		}},
	}}}
	_, err := CheckBoundsProofsWithPLIR(irProg, plirProg)
	if err == nil || !strings.Contains(err.Error(), "proof use operation") {
		t.Fatalf(
			"CheckBoundsProofsWithPLIR error = %v, want proof use operation mismatch rejection",
			err,
		)
	}
}

func TestCheckBoundsProofsWithPLIRAcceptsExpandedBCEProofIDs(t *testing.T) {
	irProg, proofProg := lowerAndPLIRForProofValidation(t, `
func guarded(xs: []i32, i: Int) -> Int
uses mem:
    if i >= 0 && i < xs.len:
        return xs[i]
    return 0

func copied_len(xs: []i32) -> Int
uses alloc, mem:
    let copied: []i32 = xs.copy()
    return copied.len

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 7
    return guarded(xs, 0) + copied_len(xs)
`)
	report, err := CheckBoundsProofsWithPLIR(irProg, proofProg)
	if err != nil {
		t.Fatalf("CheckBoundsProofsWithPLIR: %v", err)
	}
	if !proofReportHasPrefix(report, "proof:if:") {
		t.Fatalf("proof report missing if proof: %+v", report)
	}
	if !proofReportHasPrefix(report, "proof:copy-loop:") {
		t.Fatalf("proof report missing copy-loop proof: %+v", report)
	}
}

func lowerAndPLIRForProofValidation(t *testing.T, src string) (*ir.IRProgram, *plir.Program) {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	proofProg, err := plir.FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("PLIR: %v", err)
	}
	irProg, err := lower.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	return irProg, proofProg
}

func proofReportHasPrefix(report ProofReport, prefix string) bool {
	for _, removed := range report.RemovedChecks {
		if strings.HasPrefix(removed.ProofID, prefix) {
			return true
		}
	}
	return false
}

func TestValidateAllocationPlanRejectsEscapingStackAllocation(t *testing.T) {
	err := ValidateAllocationPlan(&allocplan.Plan{Functions: []allocplan.FunctionPlan{{
		Name: "bad",
		Allocations: []allocplan.Allocation{{
			ID:                    "xs",
			SiteID:                "allocsite:bad:xs:line_1_1",
			ValueID:               "alloc_intent:xs",
			Builtin:               "core.make_u8",
			ElementType:           "u8",
			ElementSize:           1,
			LengthExpr:            "4",
			LengthStatus:          allocplan.LengthStatusNormal,
			ZeroGuardStatus:       "valid_empty_no_allocator",
			NegativeGuardStatus:   "reject_before_allocation",
			OverflowGuardStatus:   "reject_before_allocation",
			Escape:                allocplan.EscapeReturn,
			Storage:               allocplan.StorageStack,
			PlannedStorage:        allocplan.StorageStack,
			ActualLoweringStorage: allocplan.StorageHeap,
			ValidationStatus:      "validated_no_escape",
			LoweringStatus:        "conservative_heap_fallback",
			Reason:                "test",
		}},
	}}})
	if err == nil || !strings.Contains(err.Error(), "escaping allocation") {
		t.Fatalf("ValidateAllocationPlan error = %v, want escaping stack rejection", err)
	}
}

func TestValidateAllocationPlanRejectsMissingLoweringStatus(t *testing.T) {
	err := ValidateAllocationPlan(&allocplan.Plan{Functions: []allocplan.FunctionPlan{{
		Name: "bad",
		Allocations: []allocplan.Allocation{{
			ID:                    "xs",
			SiteID:                "allocsite:bad:xs:line_1_1",
			ValueID:               "alloc_intent:xs",
			Builtin:               "core.make_u8",
			ElementType:           "u8",
			ElementSize:           1,
			LengthExpr:            "4",
			LengthStatus:          allocplan.LengthStatusNormal,
			ZeroGuardStatus:       "valid_empty_no_allocator",
			NegativeGuardStatus:   "reject_before_allocation",
			OverflowGuardStatus:   "reject_before_allocation",
			Escape:                allocplan.EscapeNoEscape,
			Storage:               allocplan.StorageStack,
			PlannedStorage:        allocplan.StorageStack,
			ActualLoweringStorage: allocplan.StorageHeap,
			ValidationStatus:      "validated_no_escape",
			Reason:                "test",
		}},
	}}})
	if err == nil || !strings.Contains(err.Error(), "missing lowering status") {
		t.Fatalf("ValidateAllocationPlan error = %v, want missing lowering status", err)
	}
}

func TestValidateAllocationLoweringRejectsMissingStackIR(t *testing.T) {
	plan := &allocplan.Plan{Functions: []allocplan.FunctionPlan{{
		Name: "bad",
		Allocations: []allocplan.Allocation{{
			ID:                    "xs",
			SiteID:                "allocsite:bad:xs:line_1_1",
			ValueID:               "alloc_intent:xs",
			Builtin:               "core.make_i32",
			ElementType:           "i32",
			ElementSize:           4,
			LengthExpr:            "4",
			LengthStatus:          allocplan.LengthStatusNormal,
			ZeroGuardStatus:       "valid_empty_no_allocator",
			NegativeGuardStatus:   "reject_before_allocation",
			OverflowGuardStatus:   "reject_before_allocation",
			ByteSize:              16,
			Escape:                allocplan.EscapeNoEscape,
			Storage:               allocplan.StorageStack,
			PlannedStorage:        allocplan.StorageStack,
			ActualLoweringStorage: allocplan.StorageStack,
			ValidationStatus:      "validated_no_escape",
			LoweringStatus:        "stack_lowering",
			Reason:                "test",
		}},
	}}}
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "bad",
		LocalSlots: 2,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRMakeSliceI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "no matching IR stack slice") {
		t.Fatalf("ValidateAllocationLowering error = %v, want missing stack IR rejection", err)
	}
}

func TestValidateAllocationLoweringRejectsReturnedStackAllocation(t *testing.T) {
	plan := stackU8ValidationPlan("bad")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:        "bad",
		LocalSlots:  4,
		ReturnSlots: 2,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRStackSliceU8, Local: 0, ArgSlots: 4, Imm: 4, Name: "xs"},
			{Kind: ir.IRReturn},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "escapes via return") {
		t.Fatalf(
			"ValidateAllocationLowering error = %v, want returned stack allocation rejection",
			err,
		)
	}
}

func TestValidateAllocationLoweringRejectsCalledStackAllocation(t *testing.T) {
	plan := stackU8ValidationPlan("bad")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "bad",
		LocalSlots: 4,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRStackSliceU8, Local: 0, ArgSlots: 4, Imm: 4, Name: "xs"},
			{Kind: ir.IRCall, Name: "unknown_sink", ArgSlots: 2, RetSlots: 0},
			{Kind: ir.IRReturn},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "escapes via call") {
		t.Fatalf(
			"ValidateAllocationLowering error = %v, want called stack allocation rejection",
			err,
		)
	}
}

func TestValidateAllocationLoweringAcceptsStackAllocationPassedToKnownNoEscapeCall(t *testing.T) {
	plan := stackU8ValidationPlan("main")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "len_slice",
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "main",
			LocalSlots:  4,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 4},
				{Kind: ir.IRStackSliceU8, Local: 0, ArgSlots: 4, Imm: 4, Name: "xs"},
				{Kind: ir.IRCall, Name: "len_slice", ArgSlots: 2, RetSlots: 1},
				{Kind: ir.IRReturn},
			},
		},
	}}
	if err := ValidateAllocationLowering(plan, prog); err != nil {
		t.Fatalf("ValidateAllocationLowering: %v", err)
	}
}

func TestValidateAllocationLoweringAcceptsNestedNoEscapeWriterCallSummary(t *testing.T) {
	plan := stackU8ValidationPlan("main")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "len_slice",
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "touch_slice",
			ParamSlots:  2,
			LocalSlots:  3,
			ReturnSlots: 0,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRCall, Name: "len_slice", ArgSlots: 2, RetSlots: 1},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "write_count",
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRCall, Name: "touch_slice", ArgSlots: 2, RetSlots: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRIndexStoreU8},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "main",
			LocalSlots:  4,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 4},
				{Kind: ir.IRStackSliceU8, Local: 0, ArgSlots: 4, Imm: 4, Name: "xs"},
				{Kind: ir.IRCall, Name: "write_count", ArgSlots: 2, RetSlots: 1},
				{Kind: ir.IRReturn},
			},
		},
	}}
	if err := ValidateAllocationLowering(plan, prog); err != nil {
		t.Fatalf("ValidateAllocationLowering: %v", err)
	}
}

func TestValidateAllocationLoweringRejectsReturnedStackAllocationThroughCallSummary(t *testing.T) {
	plan := stackU8ValidationPlan("bad")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "identity_slice",
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 2,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "bad",
			LocalSlots:  4,
			ReturnSlots: 2,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 4},
				{Kind: ir.IRStackSliceU8, Local: 0, ArgSlots: 4, Imm: 4, Name: "xs"},
				{Kind: ir.IRCall, Name: "identity_slice", ArgSlots: 2, RetSlots: 2},
				{Kind: ir.IRReturn},
			},
		},
	}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "escapes via return") {
		t.Fatalf(
			"ValidateAllocationLowering error = %v, want call-summary return escape rejection",
			err,
		)
	}
}

func TestValidateAllocationLoweringRejectsGlobalStoredStackAllocation(t *testing.T) {
	plan := stackU8ValidationPlan("bad")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "bad",
		LocalSlots: 4,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRStackSliceU8, Local: 0, ArgSlots: 4, Imm: 4, Name: "xs"},
			{Kind: ir.IRStoreGlobal, Local: 1},
			{Kind: ir.IRStoreGlobal, Local: 0},
			{Kind: ir.IRReturn},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "escapes via global store") {
		t.Fatalf(
			"ValidateAllocationLowering error = %v, want global-stored stack allocation rejection",
			err,
		)
	}
}

func TestValidateAllocationLoweringAllowsReturningStackViewLength(t *testing.T) {
	plan := stackU8ValidationPlan("main")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:        "main",
		LocalSlots:  6,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRStackSliceU8, Local: 0, ArgSlots: 4, Imm: 4, Name: "xs"},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRSliceWindow},
			{Kind: ir.IRStoreLocal, Local: 5},
			{Kind: ir.IRStoreLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRReturn},
		},
	}}}
	if err := ValidateAllocationLowering(plan, prog); err != nil {
		t.Fatalf("ValidateAllocationLowering: %v", err)
	}
}

func stackU8ValidationPlan(functionName string) *allocplan.Plan {
	return &allocplan.Plan{Functions: []allocplan.FunctionPlan{{
		Name: functionName,
		Allocations: []allocplan.Allocation{{
			ID:                    "xs",
			SiteID:                "allocsite:" + functionName + ":xs:line_1_1",
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
	}}}
}

func TestValidateAllocationLoweringRejectsMissingExplicitIslandIR(t *testing.T) {
	plan := &allocplan.Plan{Functions: []allocplan.FunctionPlan{{
		Name: "bad",
		Allocations: []allocplan.Allocation{{
			ID:                    "xs",
			SiteID:                "allocsite:bad:xs:line_1_1",
			ValueID:               "alloc_intent:xs",
			Builtin:               "core.island_make_i32",
			ElementType:           "i32",
			ElementSize:           4,
			LengthExpr:            "4",
			LengthStatus:          allocplan.LengthStatusNormal,
			ZeroGuardStatus:       "valid_empty_no_allocator",
			NegativeGuardStatus:   "reject_before_allocation",
			OverflowGuardStatus:   "reject_before_allocation",
			ByteSize:              16,
			Escape:                allocplan.EscapeNoEscape,
			Storage:               allocplan.StorageExplicitIsland,
			PlannedStorage:        allocplan.StorageExplicitIsland,
			ActualLoweringStorage: allocplan.StorageExplicitIsland,
			ValidationStatus:      "validated_explicit_island_scope",
			LoweringStatus:        "explicit_island_lowering",
			RegionID:              "island:isl",
			Lifetime:              "island:isl:scope",
			Reason:                "test",
		}},
	}}}
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "bad",
		LocalSlots: 2,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRMakeSliceI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "no matching IR island slice") {
		t.Fatalf(
			"ValidateAllocationLowering error = %v, want missing explicit island IR rejection",
			err,
		)
	}
}

func TestValidateAllocationLoweringAcceptsExplicitIslandIR(t *testing.T) {
	plan := &allocplan.Plan{Functions: []allocplan.FunctionPlan{{
		Name: "main",
		Allocations: []allocplan.Allocation{{
			ID:                    "xs",
			SiteID:                "allocsite:main:xs:line_1_1",
			ValueID:               "alloc_intent:xs",
			Builtin:               "core.island_make_i32",
			ElementType:           "i32",
			ElementSize:           4,
			LengthExpr:            "4",
			LengthStatus:          allocplan.LengthStatusNormal,
			ZeroGuardStatus:       "valid_empty_no_allocator",
			NegativeGuardStatus:   "reject_before_allocation",
			OverflowGuardStatus:   "reject_before_allocation",
			ByteSize:              16,
			Escape:                allocplan.EscapeNoEscape,
			Storage:               allocplan.StorageExplicitIsland,
			PlannedStorage:        allocplan.StorageExplicitIsland,
			ActualLoweringStorage: allocplan.StorageExplicitIsland,
			ValidationStatus:      "validated_explicit_island_scope",
			LoweringStatus:        "explicit_island_lowering",
			RegionID:              "island:isl",
			Lifetime:              "island:isl:scope",
			Reason:                "test",
		}},
	}}}
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "main",
		LocalSlots: 2,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
		},
	}}}
	if err := ValidateAllocationLowering(plan, prog); err != nil {
		t.Fatalf("ValidateAllocationLowering: %v", err)
	}
}

func TestValidateAllocationLoweringAcceptsExplicitIslandParamDerivedReturn(t *testing.T) {
	plan := &allocplan.Plan{Functions: []allocplan.FunctionPlan{{
		Name: "make_buf",
		Allocations: []allocplan.Allocation{{
			ID:                                 "buf",
			SiteID:                             "allocsite:make_buf:buf:line_1_1",
			ValueID:                            "alloc_intent:buf",
			Builtin:                            "core.island_make_i32",
			ElementType:                        "i32",
			ElementSize:                        4,
			LengthExpr:                         "n",
			LengthStatus:                       allocplan.LengthStatusNormal,
			Escape:                             allocplan.EscapeNoEscape,
			Storage:                            allocplan.StorageExplicitIsland,
			PlannedStorage:                     allocplan.StorageExplicitIsland,
			ActualLoweringStorage:              allocplan.StorageExplicitIsland,
			ValidationStatus:                   "validated_explicit_island_scope",
			LoweringStatus:                     "explicit_island_lowering",
			RegionID:                           "island:isl",
			Lifetime:                           "island:isl:scope",
			ExplicitIslandHandleParamSlotKnown: true,
			ExplicitIslandHandleParamSlot:      0,
			Reason:                             "test",
		}},
	}}}
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:        "make_buf",
		ParamSlots:  2,
		LocalSlots:  2,
		ReturnSlots: 2,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRIslandMakeSliceI32, Name: "buf"},
			{Kind: ir.IRReturn},
		},
	}}}
	if err := ValidateAllocationLowering(plan, prog); err != nil {
		t.Fatalf("ValidateAllocationLowering: %v", err)
	}
}

func TestValidateAllocationLoweringAcceptsExplicitIslandHandleReturnedFromCall(t *testing.T) {
	plan := explicitIslandValidationPlan("main")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "wrap",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:       "main",
			LocalSlots: 2,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRIslandNew},
				{Kind: ir.IRCall, Name: "wrap", ArgSlots: 1, RetSlots: 1},
				{Kind: ir.IRConstI32, Imm: 4},
				{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRReturn},
			},
		},
	}}
	if err := ValidateAllocationLowering(plan, prog); err != nil {
		t.Fatalf("ValidateAllocationLowering: %v", err)
	}
}

func TestValidateAllocationLoweringAcceptsExplicitIslandHandleReturnedFromSummaryProgram(
	t *testing.T,
) {
	plan := explicitIslandValidationPlan("main")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "main",
		LocalSlots: 2,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRCall, Name: "lib.regions.pass_region", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRReturn},
		},
	}}}
	summaryProg := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:        "lib.regions.pass_region",
		ParamSlots:  1,
		LocalSlots:  1,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRReturn},
		},
	}}}
	if err := ValidateAllocationLoweringWithSummaryProgram(plan, prog, summaryProg); err != nil {
		t.Fatalf("ValidateAllocationLoweringWithSummaryProgram: %v", err)
	}
}

func TestValidateAllocationLoweringAcceptsExplicitIslandSliceMovedByCall(t *testing.T) {
	plan := explicitIslandValidationPlan("main")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "main",
		LocalSlots: 3,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRCall, Name: "__tetra_actor_send_slot", ArgSlots: 2, RetSlots: 1},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRReturn},
		},
	}}}
	if err := ValidateAllocationLowering(plan, prog); err != nil {
		t.Fatalf("ValidateAllocationLowering: %v", err)
	}
}

func TestValidateAllocationLoweringAcceptsExplicitIslandRecreatedAtLoopSite(t *testing.T) {
	plan := explicitIslandValidationPlan("main")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "main",
		LocalSlots: 3,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRIslandFree},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRReturn},
		},
	}}}
	if err := ValidateAllocationLowering(plan, prog); err != nil {
		t.Fatalf("ValidateAllocationLowering: %v", err)
	}
}

func TestValidateAllocationLoweringAllowsFreeOfUntrackedIslandHandle(t *testing.T) {
	plan := explicitIslandValidationPlan("main")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "main",
		LocalSlots: 3,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRCall, Name: "__tetra_actor_recv", ArgSlots: 0, RetSlots: 1},
			{Kind: ir.IRIslandFree},
			{Kind: ir.IRReturn},
		},
	}}}
	if err := ValidateAllocationLowering(plan, prog); err != nil {
		t.Fatalf("ValidateAllocationLowering: %v", err)
	}
}

func TestValidateAllocationLoweringRejectsUnnamedExplicitIslandIR(t *testing.T) {
	plan := explicitIslandValidationPlan("bad")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "bad",
		LocalSlots: 2,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRIslandMakeSliceI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRReturn},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "no matching IR island slice") {
		t.Fatalf(
			"ValidateAllocationLowering error = %v, want unnamed explicit island rejection",
			err,
		)
	}
}

func TestValidateAllocationLoweringRejectsReturnedExplicitIslandAllocation(t *testing.T) {
	plan := explicitIslandValidationPlan("bad")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:        "bad",
		LocalSlots:  2,
		ReturnSlots: 2,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
			{Kind: ir.IRReturn},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "escapes via return") {
		t.Fatalf(
			"ValidateAllocationLowering error = %v, want returned explicit island allocation rejection",
			err,
		)
	}
}

func TestValidateAllocationLoweringRejectsUseAfterExplicitIslandFree(t *testing.T) {
	plan := explicitIslandValidationPlan("bad")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "bad",
		LocalSlots: 2,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRIslandFree},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRReturn},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "use after free") {
		t.Fatalf(
			"ValidateAllocationLowering error = %v, want explicit island use-after-free rejection",
			err,
		)
	}
}

func TestValidateAllocationLoweringRejectsExplicitIslandMakeWithHandleInLengthOperand(
	t *testing.T,
) {
	plan := explicitIslandValidationPlan("bad")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "bad",
		LocalSlots: 2,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRReturn},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "no active island handle operand") {
		t.Fatalf("ValidateAllocationLowering error = %v, want island handle operand rejection", err)
	}
}

func TestValidateAllocationLoweringRejectsExplicitIslandDoubleFree(t *testing.T) {
	plan := explicitIslandValidationPlan("bad")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "bad",
		LocalSlots: 3,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRIslandFree},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRIslandFree},
			{Kind: ir.IRReturn},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "double free") {
		t.Fatalf(
			"ValidateAllocationLowering error = %v, want explicit island double-free rejection",
			err,
		)
	}
}

func TestValidateAllocationLoweringRejectsExplicitIslandDoubleFreeOnBranchMerge(t *testing.T) {
	plan := explicitIslandValidationPlan("bad")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "bad",
		LocalSlots: 3,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRIslandFree},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRIslandFree},
			{Kind: ir.IRReturn},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "double free") {
		t.Fatalf(
			"ValidateAllocationLowering error = %v, want branch-merge explicit island double-free rejection",
			err,
		)
	}
}

func TestValidateAllocationLoweringRejectsExplicitIslandUseAfterFreeThroughIndex(t *testing.T) {
	plan := explicitIslandValidationPlan("bad")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:        "bad",
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRIslandFree},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexLoadI32},
			{Kind: ir.IRReturn},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "use after free") {
		t.Fatalf("ValidateAllocationLowering error = %v, want index use-after-free rejection", err)
	}
}

func TestValidateAllocationLoweringAcceptsExplicitIslandResetReturnedHandle(t *testing.T) {
	plan := explicitIslandValidationPlan("main")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "main",
		LocalSlots: 3,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRIslandReset},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRIslandFree},
			{Kind: ir.IRReturn},
		},
	}}}
	if err := ValidateAllocationLowering(plan, prog); err != nil {
		t.Fatalf("ValidateAllocationLowering: %v", err)
	}
}

func TestValidateAllocationLoweringRejectsExplicitIslandUseAfterReset(t *testing.T) {
	plan := explicitIslandValidationPlan("bad")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "bad",
		LocalSlots: 2,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRIslandReset},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRReturn},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "use after free") {
		t.Fatalf("ValidateAllocationLowering error = %v, want reset invalidation rejection", err)
	}
}

func TestValidateAllocationLoweringRejectsExplicitIslandResetWhileSliceLive(t *testing.T) {
	plan := explicitIslandValidationPlan("bad")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "bad",
		LocalSlots: 3,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRIslandReset},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRIslandFree},
			{Kind: ir.IRReturn},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "reset while live slice") {
		t.Fatalf(
			"ValidateAllocationLowering error = %v, want reset while live slice rejection",
			err,
		)
	}
}

func TestValidateAllocationLoweringAcceptsExplicitIslandResetAfterSliceLocalCleared(t *testing.T) {
	plan := explicitIslandValidationPlan("main")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "main",
		LocalSlots: 3,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRIslandReset},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRIslandFree},
			{Kind: ir.IRReturn},
		},
	}}}
	if err := ValidateAllocationLowering(plan, prog); err != nil {
		t.Fatalf("ValidateAllocationLowering: %v", err)
	}
}

func TestValidateAllocationLoweringRejectsExplicitIslandUnknownCallReturnedHandle(t *testing.T) {
	plan := explicitIslandValidationPlan("bad")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       "bad",
		LocalSlots: 2,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRCall, Name: "__unknown_island_identity", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRReturn},
		},
	}}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "no active island handle operand") {
		t.Fatalf(
			"ValidateAllocationLowering error = %v, want unknown-call conservative handle rejection",
			err,
		)
	}
}

func TestValidateAllocationLoweringRejectsReturnedExplicitIslandAllocationThroughCallSummary(
	t *testing.T,
) {
	plan := explicitIslandValidationPlan("bad")
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "identity_slice",
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 2,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "bad",
			LocalSlots:  3,
			ReturnSlots: 2,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 64},
				{Kind: ir.IRIslandNew},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 4},
				{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRCall, Name: "identity_slice", ArgSlots: 2, RetSlots: 2},
				{Kind: ir.IRReturn},
			},
		},
	}}
	err := ValidateAllocationLowering(plan, prog)
	if err == nil || !strings.Contains(err.Error(), "escapes via return") {
		t.Fatalf(
			"ValidateAllocationLowering error = %v, want call-summary return escape rejection",
			err,
		)
	}
}

func explicitIslandValidationPlan(functionName string) *allocplan.Plan {
	return &allocplan.Plan{Functions: []allocplan.FunctionPlan{{
		Name: functionName,
		Allocations: []allocplan.Allocation{{
			ID:                    "xs",
			SiteID:                "allocsite:" + functionName + ":xs:line_1_1",
			ValueID:               "alloc_intent:xs",
			Builtin:               "core.island_make_i32",
			ElementType:           "i32",
			ElementSize:           4,
			LengthExpr:            "4",
			LengthStatus:          allocplan.LengthStatusNormal,
			ZeroGuardStatus:       "valid_empty_no_metadata_access",
			NegativeGuardStatus:   "reject_before_metadata_access",
			OverflowGuardStatus:   "reject_before_metadata_access",
			ByteSize:              16,
			Escape:                allocplan.EscapeNoEscape,
			Storage:               allocplan.StorageExplicitIsland,
			PlannedStorage:        allocplan.StorageExplicitIsland,
			ActualLoweringStorage: allocplan.StorageExplicitIsland,
			ValidationStatus:      "validated_explicit_island_scope",
			LoweringStatus:        "explicit_island_lowering",
			RegionID:              "island:isl",
			Lifetime:              "island:isl:scope",
			Reason:                "test",
		}},
	}}}
}

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
		t.Fatalf(
			"ValidateAllocationLowering error = %v, want missing function-temp region IR rejection",
			err,
		)
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
		t.Fatalf(
			"ValidateAllocationLowering error = %v, want returned region allocation rejection",
			err,
		)
	}
}

func TestValidateAllocationLoweringRejectsFunctionTempRegionResetThatDoesNotDominateBranchReturn(
	t *testing.T,
) {
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
	if err == nil ||
		!strings.Contains(err.Error(), "function-temp region reset does not dominate return") {
		t.Fatalf(
			"ValidateAllocationLowering error = %v, want branch reset-dominance rejection",
			err,
		)
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
	if err == nil ||
		!strings.Contains(err.Error(), "function-temp region enter does not dominate make") {
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
	if err == nil ||
		!strings.Contains(err.Error(), "function-temp region reset does not dominate return") {
		t.Fatalf(
			"ValidateAllocationLowering error = %v, want loop-exit reset-dominance rejection",
			err,
		)
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
