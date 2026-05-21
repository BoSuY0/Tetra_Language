package linux_x32

import (
	"bytes"
	"encoding/binary"
	"testing"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
)

func TestCodegenObjectLinuxX32SetsTargetAndUsesX32Syscalls(t *testing.T) {
	obj, err := CodegenObjectLinuxX32([]ir.IRFunc{writeHelloMainFunc()})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX32: %v", err)
	}
	if obj.Target != "linux-x32" {
		t.Fatalf("target = %q, want linux-x32", obj.Target)
	}
	if len(obj.Code) == 0 {
		t.Fatalf("expected object code")
	}
	if !bytes.Contains(obj.Data, []byte("hello")) {
		t.Fatalf("data = %q, want hello literal", string(obj.Data))
	}
	if hasRelocKind(obj.Relocs, tobj.RelocIATDisp32) {
		t.Fatalf("linux-x32 object unexpectedly collected Windows IAT relocs: %#v", obj.Relocs)
	}
	if !hasSymbol(obj.Symbols, "main", 0, 1) {
		t.Fatalf("missing main symbol with expected ABI: %#v", obj.Symbols)
	}
	if !containsMovEaxImm32(obj.Code, 0x40000001) {
		t.Fatalf("missing x32 write syscall number in code: % x", obj.Code)
	}
	if containsMovEaxImm32(obj.Code, 1) {
		t.Fatalf("linux-x32 emitted plain x64 write syscall number: % x", obj.Code)
	}
}

func TestCodegenObjectLinuxX32DefaultsToILP32PointerOps(t *testing.T) {
	obj, err := CodegenObjectLinuxX32([]ir.IRFunc{{
		Name:        "__test_ptr_read_x32",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRMemReadPtr},
			{Kind: ir.IRReturn},
		},
	}})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX32: %v", err)
	}
	assertContainsBytes(t, "x32 ptr read guard width", obj.Code, addEdxImm32Bytes(4))
	wantLoad32 := &x64.Emitter{}
	wantLoad32.MovEaxFromRaxPtr()
	assertContainsBytes(t, "x32 ptr read 32-bit load", obj.Code, wantLoad32.Buf)
	forbidLoad64 := &x64.Emitter{}
	forbidLoad64.MovRaxFromRdiDisp(0)
	assertNotContainsBytes(t, "x32 ptr read 64-bit load", obj.Code, forbidLoad64.Buf)
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

func containsMovEaxImm32(buf []byte, imm uint32) bool {
	for i := 0; i+5 <= len(buf); i++ {
		if buf[i] == 0xB8 && binary.LittleEndian.Uint32(buf[i+1:i+5]) == imm {
			return true
		}
	}
	return false
}

func addEdxImm32Bytes(v uint32) []byte {
	e := &x64.Emitter{}
	e.AddEdxImm32(int32(v))
	return e.Buf
}

func assertContainsBytes(t *testing.T, label string, haystack []byte, needle []byte) {
	t.Helper()
	if !bytes.Contains(haystack, needle) {
		t.Fatalf("%s: missing bytes % x in % x", label, needle, haystack)
	}
}

func assertNotContainsBytes(t *testing.T, label string, haystack []byte, needle []byte) {
	t.Helper()
	if bytes.Contains(haystack, needle) {
		t.Fatalf("%s: unexpected bytes % x in % x", label, needle, haystack)
	}
}
