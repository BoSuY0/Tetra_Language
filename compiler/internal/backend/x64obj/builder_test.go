package x64obj

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
)

func TestObjectBuildCollectsRelocsAndSymbols(t *testing.T) {
	emit := func(
		e *x64.Emitter,
		fn ir.IRFunc,
		dataBlobs *[][]byte,
		leaPatches *[]LeaPatch,
		callPatches *[]CallPatch,
		importPatches *[]ImportPatch,
		opt x64.CodegenOptions,
	) error {
		e.Emit(0x90, 0xC3)
		*dataBlobs = append(*dataBlobs, []byte("blob"))
		*leaPatches = append(*leaPatches, LeaPatch{At: 0, DataIndex: 0})
		*callPatches = append(*callPatches, CallPatch{At: 0, Name: "ext.call"})
		if importPatches != nil {
			*importPatches = append(*importPatches, ImportPatch{At: 0, Name: "kernel32.ExitProcess"})
		}
		return nil
	}

	obj, err := BuildObject([]ir.IRFunc{{
		Name:        "main",
		ExportName:  "entry",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs:      []ir.IRInstr{{Kind: ir.IRConstI32, Imm: 0}, {Kind: ir.IRReturn}},
	}}, emit, x64.CodegenOptions{}, Options{CollectImports: true})
	if err != nil {
		t.Fatalf("BuildObject: %v", err)
	}
	if len(obj.Code) != 2 {
		t.Fatalf("code len = %d, want 2", len(obj.Code))
	}
	if string(obj.Data) != "blob" {
		t.Fatalf("data = %q, want blob", string(obj.Data))
	}
	hasKind := func(kind tobj.RelocKind) bool {
		for _, r := range obj.Relocs {
			if r.Kind == kind {
				return true
			}
		}
		return false
	}
	if !hasKind(tobj.RelocCallRel32) || !hasKind(tobj.RelocIATDisp32) || !hasKind(tobj.RelocDataDisp32) {
		t.Fatalf("missing expected reloc kinds: %#v", obj.Relocs)
	}
	if len(obj.Symbols) != 2 || obj.Symbols[0].Name != "entry" || obj.Symbols[1].Name != "main" {
		t.Fatalf("unexpected symbols ordering/content: %#v", obj.Symbols)
	}
}

func TestObjectBuildRejectsInvalidDataPatchIndex(t *testing.T) {
	emit := func(
		e *x64.Emitter,
		fn ir.IRFunc,
		dataBlobs *[][]byte,
		leaPatches *[]LeaPatch,
		callPatches *[]CallPatch,
		importPatches *[]ImportPatch,
		opt x64.CodegenOptions,
	) error {
		e.Emit(0xC3)
		*leaPatches = append(*leaPatches, LeaPatch{At: 0, DataIndex: 9})
		return nil
	}

	_, err := BuildObject([]ir.IRFunc{{
		Name:        "main",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs:      []ir.IRInstr{{Kind: ir.IRConstI32, Imm: 0}, {Kind: ir.IRReturn}},
	}}, emit, x64.CodegenOptions{}, Options{})
	if err == nil {
		t.Fatalf("expected invalid data patch error")
	}
	if !strings.Contains(err.Error(), "invalid data patch index") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestObjectBuildRejectsDuplicateSymbols(t *testing.T) {
	emit := func(
		e *x64.Emitter,
		fn ir.IRFunc,
		dataBlobs *[][]byte,
		leaPatches *[]LeaPatch,
		callPatches *[]CallPatch,
		importPatches *[]ImportPatch,
		opt x64.CodegenOptions,
	) error {
		e.Emit(0xC3)
		return nil
	}

	_, err := BuildObject([]ir.IRFunc{
		{Name: "main", ExportName: "dup"},
		{Name: "second", ExportName: "dup"},
	}, emit, x64.CodegenOptions{}, Options{})
	if err == nil {
		t.Fatalf("expected duplicate symbol error")
	}
	if !strings.Contains(err.Error(), "duplicate exported symbol") {
		t.Fatalf("unexpected error: %v", err)
	}
}
