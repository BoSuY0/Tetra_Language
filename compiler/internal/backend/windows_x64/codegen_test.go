package windows_x64

import (
	"bytes"
	"testing"

	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
)

func TestCodegenObjectWindowsX64SetsTargetAndCollectsIATRelocs(t *testing.T) {
	obj, err := CodegenObjectWindowsX64([]ir.IRFunc{writeHelloMainFunc()})
	if err != nil {
		t.Fatalf("CodegenObjectWindowsX64: %v", err)
	}
	if obj.Target != "windows-x64" {
		t.Fatalf("target = %q, want windows-x64", obj.Target)
	}
	if len(obj.Code) == 0 {
		t.Fatalf("expected object code")
	}
	if !bytes.Contains(obj.Data, []byte("hello")) {
		t.Fatalf("data = %q, want hello literal", string(obj.Data))
	}
	for _, name := range []string{"kernel32.GetStdHandle", "kernel32.WriteFile"} {
		if !hasIATReloc(obj.Relocs, name) {
			t.Fatalf("missing IAT reloc %q in %#v", name, obj.Relocs)
		}
	}
	if !hasSymbol(obj.Symbols, "main", 0, 1) {
		t.Fatalf("missing main symbol with expected ABI: %#v", obj.Symbols)
	}
}

func writeHelloMainFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRStrLit, Str: []byte("hello")},
			{Kind: ir.IRWrite},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}

func hasIATReloc(relocs []tobj.Reloc, name string) bool {
	for _, reloc := range relocs {
		if reloc.Kind == tobj.RelocIATDisp32 && reloc.Name == name {
			return true
		}
	}
	return false
}

func hasSymbol(symbols []tobj.Symbol, name string, params, returns int) bool {
	for _, sym := range symbols {
		if sym.Name == name && sym.HasSignature && sym.ParamSlots == params &&
			sym.ReturnSlots == returns {
			return true
		}
	}
	return false
}
