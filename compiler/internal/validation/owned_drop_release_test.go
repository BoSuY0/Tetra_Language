package validation

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestValidateOwnedDropReleaseIRAcceptsTypedDropReleasePair(t *testing.T) {
	fn := ir.IRFunc{
		Name: "drop_once",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRAllocBytes, Name: "buf"},
			ownedDropInstr(),
			ownedReleaseInstr(),
			{Kind: ir.IRReturn},
		},
	}
	if err := ValidateOwnedDropReleaseIR(fn); err != nil {
		t.Fatalf("ValidateOwnedDropReleaseIR: %v", err)
	}
}

func TestValidateOwnedDropReleaseIRRejectsMissingTypedDropMetadata(t *testing.T) {
	fn := ir.IRFunc{
		Name: "missing_metadata",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRAllocBytes, Name: "buf"},
			{Kind: ir.IRDropOwned},
			{Kind: ir.IRReturn},
		},
	}
	err := ValidateOwnedDropReleaseIR(fn)
	if err == nil || !strings.Contains(err.Error(), "typed drop metadata") {
		t.Fatalf("ValidateOwnedDropReleaseIR error = %v, want typed metadata rejection", err)
	}
}

func TestValidateOwnedDropReleaseIRRejectsReleaseWithoutDrop(t *testing.T) {
	fn := ir.IRFunc{
		Name: "release_without_drop",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRAllocBytes, Name: "buf"},
			ownedReleaseInstr(),
			{Kind: ir.IRReturn},
		},
	}
	err := ValidateOwnedDropReleaseIR(fn)
	if err == nil || !strings.Contains(err.Error(), "release without drop") {
		t.Fatalf("ValidateOwnedDropReleaseIR error = %v, want release-without-drop rejection", err)
	}
}

func TestValidateOwnedDropReleaseIRRejectsDropWithoutReleaseBeforeReturn(t *testing.T) {
	fn := ir.IRFunc{
		Name: "drop_without_release",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRAllocBytes, Name: "buf"},
			ownedDropInstr(),
			{Kind: ir.IRReturn},
		},
	}
	err := ValidateOwnedDropReleaseIR(fn)
	if err == nil || !strings.Contains(err.Error(), "release token") {
		t.Fatalf("ValidateOwnedDropReleaseIR error = %v, want missing release rejection", err)
	}
}

func TestValidateOwnedDropReleaseIRRejectsDoubleDropThroughLocal(t *testing.T) {
	fn := ir.IRFunc{
		Name:       "double_drop",
		LocalSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRAllocBytes, Name: "buf"},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			ownedDropInstr(),
			ownedReleaseInstr(),
			{Kind: ir.IRLoadLocal, Local: 0},
			ownedDropInstr(),
			ownedReleaseInstr(),
			{Kind: ir.IRReturn},
		},
	}
	err := ValidateOwnedDropReleaseIR(fn)
	if err == nil || !strings.Contains(err.Error(), "use after drop") {
		t.Fatalf("ValidateOwnedDropReleaseIR error = %v, want double-drop rejection", err)
	}
}

func TestValidateOwnedDropReleaseIRRejectsBranchPathWithoutDrop(t *testing.T) {
	fn := ir.IRFunc{
		Name:       "branch_missing_drop",
		LocalSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRAllocBytes, Name: "buf"},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			ownedDropInstr(),
			ownedReleaseInstr(),
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRReturn},
		},
	}
	err := ValidateOwnedDropReleaseIR(fn)
	if err == nil || !strings.Contains(err.Error(), "not dropped") {
		t.Fatalf("ValidateOwnedDropReleaseIR error = %v, want branch missing-drop rejection", err)
	}
}

func TestValidateOwnedDropReleaseIRAcceptsDropReleaseOnBothBranchReturns(t *testing.T) {
	fn := ir.IRFunc{
		Name:       "branch_both_drop",
		LocalSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRAllocBytes, Name: "buf"},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			ownedDropInstr(),
			ownedReleaseInstr(),
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			ownedDropInstr(),
			ownedReleaseInstr(),
			{Kind: ir.IRReturn},
		},
	}
	if err := ValidateOwnedDropReleaseIR(fn); err != nil {
		t.Fatalf("ValidateOwnedDropReleaseIR: %v", err)
	}
}

func TestValidateOwnedDropReleaseIRAcceptsIncomingOwnedParamDropRelease(t *testing.T) {
	fn := ir.IRFunc{
		Name:       "consume_param",
		LocalSlots: 1,
		OwnedParams: []ir.IROwnedParam{
			{
				Local:           0,
				LayoutID:        "layout:consume_param:consume_param:raw",
				OwnershipDomain: ir.IROwnershipDomainHeap,
				ReleaseKind:     ir.IRReleaseKindLinuxMmap,
			},
		},
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			ownedDropInstrWithLayout("layout:consume_param:consume_param:raw"),
			ownedReleaseInstrWithLayout("layout:consume_param:consume_param:raw"),
			{Kind: ir.IRReturn},
		},
	}
	if err := ValidateOwnedDropReleaseIR(fn); err != nil {
		t.Fatalf("ValidateOwnedDropReleaseIR: %v", err)
	}
}

func TestValidateOwnedDropReleaseIRAcceptsOwnedCallReturnDropRelease(t *testing.T) {
	fn := ir.IRFunc{
		Name:       "caller",
		LocalSlots: 1,
		Instrs: []ir.IRInstr{
			{
				Kind:            ir.IRCall,
				Name:            "make",
				RetSlots:        1,
				LayoutID:        "layout:return:make:p",
				OwnershipDomain: ir.IROwnershipDomainHeap,
				ReleaseKind:     ir.IRReleaseKindLinuxMmap,
			},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			ownedDropInstrWithLayout("layout:return:make:p"),
			ownedReleaseInstrWithLayout("layout:return:make:p"),
			{Kind: ir.IRReturn},
		},
	}
	if err := ValidateOwnedDropReleaseIR(fn); err != nil {
		t.Fatalf("ValidateOwnedDropReleaseIR: %v", err)
	}
}

func TestValidateOwnedDropReleaseIRAcceptsOwnedMultiSlotCallReturnDropRelease(t *testing.T) {
	fn := ir.IRFunc{
		Name:       "caller",
		LocalSlots: 2,
		Instrs: []ir.IRInstr{
			{
				Kind:            ir.IRCall,
				Name:            "make_box",
				RetSlots:        2,
				OwnedReturnSlot: 1,
				LayoutID:        "layout:return:make_box:box",
				OwnershipDomain: ir.IROwnershipDomainHeap,
				ReleaseKind:     ir.IRReleaseKindLinuxMmap,
			},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			ownedDropInstrWithLayout("layout:return:make_box:box"),
			ownedReleaseInstrWithLayout("layout:return:make_box:box"),
			{Kind: ir.IRReturn},
		},
	}
	if err := ValidateOwnedDropReleaseIR(fn); err != nil {
		t.Fatalf("ValidateOwnedDropReleaseIR: %v", err)
	}
}

func TestValidateOwnedDropReleaseIRAcceptsConditionalOwnedCallReturnDropOnPresentPath(t *testing.T) {
	fn := ir.IRFunc{
		Name:       "caller",
		LocalSlots: 2,
		Instrs: []ir.IRInstr{
			{
				Kind:                   ir.IRCall,
				Name:                   "maybe",
				RetSlots:               2,
				OwnedReturnSlot:        0,
				OwnedReturnConditional: true,
				LayoutID:               "layout:return:maybe:p",
				OwnershipDomain:        ir.IROwnershipDomainHeap,
				ReleaseKind:            ir.IRReleaseKindLinuxMmap,
			},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			ownedDropInstrWithLayout("layout:return:maybe:p"),
			ownedReleaseInstrWithLayout("layout:return:maybe:p"),
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRReturn},
		},
	}
	if err := ValidateOwnedDropReleaseIR(fn); err != nil {
		t.Fatalf("ValidateOwnedDropReleaseIR: %v", err)
	}
}

func TestValidateOwnedDropReleaseIRAcceptsOwnedThrowingCallErrorSlotDropOnErrorPath(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "catcher",
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{
				Kind:            ir.IRCall,
				Name:            "fail_box",
				RetSlots:        3,
				OwnsErrorSlot:   true,
				OwnedErrorSlot:  1,
				LayoutID:        "layout:throw:fail_box:err",
				OwnershipDomain: ir.IROwnershipDomainHeap,
				ReleaseKind:     ir.IRReleaseKindLinuxMmap,
			},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			ownedDropInstrWithLayout("layout:throw:fail_box:err"),
			ownedReleaseInstrWithLayout("layout:throw:fail_box:err"),
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRReturn},
		},
	}
	if err := ValidateOwnedDropReleaseIR(fn); err != nil {
		t.Fatalf("ValidateOwnedDropReleaseIR: %v", err)
	}
}

func TestValidateOwnedDropReleaseIRRejectsOwnedThrowingCallErrorSlotLeakOnErrorPath(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "catcher",
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{
				Kind:            ir.IRCall,
				Name:            "fail_box",
				RetSlots:        3,
				OwnsErrorSlot:   true,
				OwnedErrorSlot:  1,
				LayoutID:        "layout:throw:fail_box:err",
				OwnershipDomain: ir.IROwnershipDomainHeap,
				ReleaseKind:     ir.IRReleaseKindLinuxMmap,
			},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRReturn},
		},
	}
	err := ValidateOwnedDropReleaseIR(fn)
	if err == nil || !strings.Contains(err.Error(), "not dropped") {
		t.Fatalf("ValidateOwnedDropReleaseIR error = %v, want owned error leak rejection", err)
	}
}

func TestValidateOwnedDropReleaseIRRejectsOwnedCallReturnLeak(t *testing.T) {
	fn := ir.IRFunc{
		Name:       "caller",
		LocalSlots: 1,
		Instrs: []ir.IRInstr{
			{
				Kind:            ir.IRCall,
				Name:            "make",
				RetSlots:        1,
				LayoutID:        "layout:return:make:p",
				OwnershipDomain: ir.IROwnershipDomainHeap,
				ReleaseKind:     ir.IRReleaseKindLinuxMmap,
			},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRReturn},
		},
	}
	err := ValidateOwnedDropReleaseIR(fn)
	if err == nil || !strings.Contains(err.Error(), "not dropped") {
		t.Fatalf("ValidateOwnedDropReleaseIR error = %v, want call-return leak rejection", err)
	}
}

func TestValidateOwnedDropReleaseIRAcceptsOwnedStoreGlobalTransfer(t *testing.T) {
	fn := ir.IRFunc{
		Name: "global_transfer",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRAllocBytes, Name: "buf"},
			{Kind: ir.IRStoreGlobal, Local: 0},
			{Kind: ir.IRReturn},
		},
	}
	if err := ValidateOwnedDropReleaseIR(fn); err != nil {
		t.Fatalf("ValidateOwnedDropReleaseIR: %v", err)
	}
}

func TestValidateOwnedDropReleaseIRRejectsUseAfterStoreGlobalTransfer(t *testing.T) {
	fn := ir.IRFunc{
		Name:       "use_after_global_transfer",
		LocalSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRAllocBytes, Name: "buf"},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRStoreGlobal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			ownedDropInstr(),
			ownedReleaseInstr(),
			{Kind: ir.IRReturn},
		},
	}
	err := ValidateOwnedDropReleaseIR(fn)
	if err == nil || !strings.Contains(err.Error(), "use after transfer") {
		t.Fatalf("ValidateOwnedDropReleaseIR error = %v, want use-after-transfer rejection", err)
	}
}

func TestValidateOwnedDropReleaseIRRejectsSameNameAllocationLeak(t *testing.T) {
	fn := ir.IRFunc{
		Name:       "same_name_allocs",
		LocalSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRAllocBytes, Name: "buf"},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRAllocBytes, Name: "buf"},
			ownedDropInstr(),
			ownedReleaseInstr(),
			{Kind: ir.IRReturn},
		},
	}
	err := ValidateOwnedDropReleaseIR(fn)
	if err == nil || !strings.Contains(err.Error(), "not dropped") {
		t.Fatalf("ValidateOwnedDropReleaseIR error = %v, want first allocation leak rejection", err)
	}
}

func ownedDropInstr() ir.IRInstr {
	return ownedDropInstrWithLayout("layout:bytes64")
}

func ownedDropInstrWithLayout(layoutID string) ir.IRInstr {
	return ir.IRInstr{
		Kind:            ir.IRDropOwned,
		LayoutID:        layoutID,
		OwnershipDomain: ir.IROwnershipDomainHeap,
		ReleaseKind:     ir.IRReleaseKindLinuxMmap,
	}
}

func ownedReleaseInstr() ir.IRInstr {
	return ownedReleaseInstrWithLayout("layout:bytes64")
}

func ownedReleaseInstrWithLayout(layoutID string) ir.IRInstr {
	return ir.IRInstr{
		Kind:            ir.IRReleaseAllocation,
		LayoutID:        layoutID,
		OwnershipDomain: ir.IROwnershipDomainHeap,
		ReleaseKind:     ir.IRReleaseKindLinuxMmap,
	}
}
