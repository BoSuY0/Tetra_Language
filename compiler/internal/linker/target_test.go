package linker

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"tetra_language/compiler/internal/format/elf"
	"tetra_language/compiler/internal/format/tobj"
)

func TestLinkLinuxRejectsNonLinuxTargetObject(t *testing.T) {
	_, err := LinkLinuxX64([]*tobj.Object{{
		Target:  "windows-x64",
		Module:  "wrong",
		Code:    []byte{0xC3},
		Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
	}}, "main")
	if err == nil {
		t.Fatalf("expected mismatch error")
	}
	if !strings.Contains(err.Error(), "linker target mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkLinuxX32RejectsNonX32TargetObject(t *testing.T) {
	_, err := LinkLinuxX32([]*tobj.Object{{
		Target:  "linux-x64",
		Module:  "wrong",
		Code:    []byte{0xC3},
		Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
	}}, "main")
	if err == nil {
		t.Fatalf("expected mismatch error")
	}
	if !strings.Contains(err.Error(), "linker target mismatch") ||
		!strings.Contains(err.Error(), "linux-x32") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkLinuxX32UsesX32ExitSyscall(t *testing.T) {
	img, err := LinkLinuxX32([]*tobj.Object{{
		Target:  "linux-x32",
		Module:  "main",
		Code:    []byte{0xC3},
		Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
	}}, "main")
	if err != nil {
		t.Fatalf("link x32: %v", err)
	}
	if !containsMovEaxImm32LinkerTest(img.Code, 0x40000000+60) {
		t.Fatalf("missing x32 exit syscall number in entry stub: % x", img.Code)
	}
	if containsMovEaxImm32LinkerTest(img.Code, 60) {
		t.Fatalf("x32 entry stub emitted plain x64 exit syscall number: % x", img.Code)
	}
}

func TestLinkLinuxX86UsesI386ExitInt80(t *testing.T) {
	img, err := LinkLinuxX86([]*tobj.Object{{
		Target:  "linux-x86",
		Module:  "main",
		Code:    []byte{0xC3},
		Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
	}}, "main")
	if err != nil {
		t.Fatalf("link x86: %v", err)
	}
	if !bytes.Contains(img.Code, []byte{0x89, 0xC3, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80}) {
		t.Fatalf("missing i386 exit int 0x80 stub: % x", img.Code)
	}
	if containsMovEaxImm32LinkerTest(img.Code, 60) {
		t.Fatalf("x86 entry stub emitted x64 exit syscall number: % x", img.Code)
	}
}

func TestLinkLinuxX86PatchesAbsoluteDataRelocs(t *testing.T) {
	img, err := LinkLinuxX86([]*tobj.Object{{
		Target:  "linux-x86",
		Module:  "main",
		Code:    []byte{0xA1, 0, 0, 0, 0, 0xC3}, // mov eax, moffs32; ret
		Data:    []byte{0x2a, 0, 0, 0, 0, 0, 0, 0},
		Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
		Relocs:  []tobj.Reloc{{Kind: tobj.RelocDataAbs32, At: 1}},
	}}, "main")
	if err != nil {
		t.Fatalf("link x86: %v", err)
	}
	loadAt := bytes.IndexByte(img.Code, 0xA1)
	if loadAt < 0 {
		t.Fatalf("missing x86 absolute load in linked code: % x", img.Code)
	}
	got := binary.LittleEndian.Uint32(img.Code[loadAt+1 : loadAt+5])
	layout := elf.LinuxX86Layout(len(img.Code), len(img.Data))
	want := uint32(elf.LinuxX86BaseVaddr + layout.DataOffset)
	if got != want {
		t.Fatalf("x86 absolute data reloc = %#x, want %#x", got, want)
	}
}

func TestLinkLinuxX86PatchesAbsoluteFunctionAddressRelocs(t *testing.T) {
	img, err := LinkLinuxX86([]*tobj.Object{
		{
			Target:  "linux-x86",
			Module:  "a",
			Code:    []byte{0xC3},
			Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
		},
		{
			Target:  "linux-x86",
			Module:  "b",
			Code:    []byte{0xB8, 0, 0, 0, 0, 0xC3}, // mov eax, imm32; ret
			Symbols: []tobj.Symbol{{Name: "addr_user", Offset: 0}},
			Relocs:  []tobj.Reloc{{Kind: tobj.RelocFuncAddrAbs32, At: 1, Name: "main"}},
		},
	}, "main")
	if err != nil {
		t.Fatalf("link x86: %v", err)
	}
	stubLen := len(emitEntryStubForTestX86())
	movAt := stubLen + 1
	if movAt+5 > len(img.Code) || img.Code[movAt] != 0xB8 {
		t.Fatalf("missing x86 mov eax, imm32 in linked code: % x", img.Code)
	}
	got := binary.LittleEndian.Uint32(img.Code[movAt+1 : movAt+5])
	layout := elf.LinuxX86Layout(len(img.Code), len(img.Data))
	want := uint32(elf.LinuxX86BaseVaddr + layout.CodeOffset + stubLen)
	if got != want {
		t.Fatalf("x86 absolute function address reloc = %#x, want %#x", got, want)
	}
}

func TestLinkLinuxX32DataRelocsUseELF32X32Layout(t *testing.T) {
	img, err := LinkLinuxX32([]*tobj.Object{{
		Target: "linux-x32",
		Module: "main",
		Code: []byte{
			0x48, 0x8D, 0x05, 0, 0, 0, 0, // lea rax,[rip+data]
			0xC3,
		},
		Data:    []byte{0x2a},
		Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
		Relocs: []tobj.Reloc{{
			Kind: tobj.RelocDataDisp32,
			At:   3,
		}},
	}}, "main")
	if err != nil {
		t.Fatalf("link x32: %v", err)
	}
	leaAt := bytes.Index(img.Code, []byte{0x48, 0x8D, 0x05})
	if leaAt < 0 {
		t.Fatalf("missing test lea instruction in linked code: % x", img.Code)
	}
	gotDisp := int32(binary.LittleEndian.Uint32(img.Code[leaAt+3 : leaAt+7]))
	layout := elf.LinuxX32Layout(len(img.Code), len(img.Data))
	wantDisp := int32((layout.DataOffset - layout.CodeOffset) - (leaAt + 7))
	if gotDisp != wantDisp {
		t.Fatalf("x32 data relocation disp = %#x, want %#x", gotDisp, wantDisp)
	}
}

func TestLinkLinuxX64RejectsAbsoluteDataRelocs(t *testing.T) {
	_, err := LinkLinuxX64([]*tobj.Object{{
		Target:  "linux-x64",
		Module:  "main",
		Code:    []byte{0xA1, 0, 0, 0, 0, 0xC3},
		Data:    []byte{0x2a},
		Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
		Relocs:  []tobj.Reloc{{Kind: tobj.RelocDataAbs32, At: 1}},
	}}, "main")
	if err == nil {
		t.Fatalf("expected absolute data relocation rejection")
	}
	if !strings.Contains(err.Error(), "absolute data relocation") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkLinuxX64RejectsAbsoluteFunctionAddressRelocs(t *testing.T) {
	_, err := LinkLinuxX64([]*tobj.Object{{
		Target:  "linux-x64",
		Module:  "main",
		Code:    []byte{0xB8, 0, 0, 0, 0, 0xC3},
		Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
		Relocs:  []tobj.Reloc{{Kind: tobj.RelocFuncAddrAbs32, At: 1, Name: "main"}},
	}}, "main")
	if err == nil {
		t.Fatalf("expected absolute function address relocation rejection")
	}
	if !strings.Contains(err.Error(), "absolute function address relocation") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func containsMovEaxImm32LinkerTest(code []byte, imm uint32) bool {
	needle := []byte{0xB8, 0, 0, 0, 0}
	binary.LittleEndian.PutUint32(needle[1:], imm)
	return bytes.Contains(code, needle)
}

func emitEntryStubForTestX86() []byte {
	stub, _ := emitEntryStubSysVLinuxX86()
	return stub
}

func TestLinkWindowsRejectsNonWindowsTargetObject(t *testing.T) {
	_, err := LinkWindowsX64([]*tobj.Object{{
		Target:  "linux-x64",
		Module:  "wrong",
		Code:    []byte{0xC3},
		Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
	}}, "main")
	if err == nil {
		t.Fatalf("expected mismatch error")
	}
	if !strings.Contains(err.Error(), "linker target mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkMachORejectsNonMacTargetObject(t *testing.T) {
	_, err := LinkMacOSX64([]*tobj.Object{{
		Target:  "linux-x64",
		Module:  "wrong",
		Code:    []byte{0xC3},
		Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
	}}, "main")
	if err == nil {
		t.Fatalf("expected mismatch error")
	}
	if !strings.Contains(err.Error(), "linker target mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}
