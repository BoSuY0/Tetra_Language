package macos_x64

import (
	"bytes"
	"testing"

	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
)

func TestCodegenObjectMacOSX64SetsTargetAndUsesSysVRelocs(t *testing.T) {
	obj, err := CodegenObjectMacOSX64([]ir.IRFunc{writeHelloMainFunc()})
	if err != nil {
		t.Fatalf("CodegenObjectMacOSX64: %v", err)
	}
	if obj.Target != "macos-x64" {
		t.Fatalf("target = %q, want macos-x64", obj.Target)
	}
	if len(obj.Code) == 0 {
		t.Fatalf("expected object code")
	}
	if !bytes.Contains(obj.Data, []byte("hello")) {
		t.Fatalf("data = %q, want hello literal", string(obj.Data))
	}
	if hasRelocKind(obj.Relocs, tobj.RelocIATDisp32) {
		t.Fatalf("macOS object unexpectedly collected Windows IAT relocs: %#v", obj.Relocs)
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

func hasRelocKind(relocs []tobj.Reloc, kind tobj.RelocKind) bool {
	for _, reloc := range relocs {
		if reloc.Kind == kind {
			return true
		}
	}
	return false
}

func hasSymbol(symbols []tobj.Symbol, name string, params, returns int) bool {
	for _, sym := range symbols {
		if sym.Name == name && sym.HasSignature && sym.ParamSlots == params && sym.ReturnSlots == returns {
			return true
		}
	}
	return false
}
