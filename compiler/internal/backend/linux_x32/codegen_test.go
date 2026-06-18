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

func TestCodegenObjectLinuxX32RawSliceFromPartsBuildsScopedView(t *testing.T) {
	obj, err := CodegenObjectLinuxX32([]ir.IRFunc{{
		Name:        "__test_raw_slice_x32",
		ReturnSlots: 2,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRCapMem},
			{Kind: ir.IRRawSliceFromParts, Imm: 2},
			{Kind: ir.IRReturn},
		},
	}})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX32 raw_slice_from_parts: %v", err)
	}
	assertContainsBytes(
		t,
		"x32 raw slice signed length guard",
		obj.Code,
		[]byte{0x85, 0xC9, 0x0F, 0x8C},
	)
	assertContainsBytes(
		t,
		"x32 raw slice i32 byte-overflow guard",
		obj.Code,
		[]byte{0x48, 0x81, 0xF9, 0xFF, 0xFF, 0xFF, 0x1F, 0x0F, 0x8F},
	)
}

func TestCodegenObjectLinuxX32EmitsCtxSwitchSysVStub(t *testing.T) {
	obj, err := CodegenObjectLinuxX32([]ir.IRFunc{{
		Name:        "__test_ctx_switch_x32",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRCtxSwitch},
			{Kind: ir.IRReturn},
		},
	}})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX32 ctx_switch: %v", err)
	}

	want := expectedCtxSwitchSysVStub()
	if !bytes.Contains(obj.Code, want) {
		t.Fatalf("x32 ctx_switch missing SysV x86_64 stub\nwant=% x\ncode=% x", want, obj.Code)
	}
	shadow := &x64.Emitter{}
	shadow.SubRspImm32(32)
	if bytes.Contains(obj.Code, shadow.Buf) {
		t.Fatalf("x32 ctx_switch unexpectedly emitted Win64 shadow-space adjustment: % x", obj.Code)
	}
	if !bytes.Contains(obj.Code, []byte{0x31, 0xC0, 0x50}) {
		t.Fatalf("x32 ctx_switch continuation did not push zero return status: % x", obj.Code)
	}
}

func expectedCtxSwitchSysVStub() []byte {
	e := &x64.Emitter{}
	e.PushRbx()
	e.PushRbp()
	e.PushR12()
	e.PushR13()
	e.PushR14()
	e.PushR15()
	e.MovMem64RdiDispRsp(0)
	e.MovRdiRsi()
	e.MovRspFromRdiDisp(0)
	e.PopR15()
	e.PopR14()
	e.PopR13()
	e.PopR12()
	e.PopRbp()
	e.PopRbx()
	e.Ret()
	return e.Buf
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
		if sym.Name == name && sym.HasSignature && sym.ParamSlots == params &&
			sym.ReturnSlots == returns {
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
