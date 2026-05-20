package macho

import (
	"strings"
	"testing"
)

func TestWriteMachO64MacOSX64RejectsMissingImage(t *testing.T) {
	err := WriteMachO64MacOSX64(t.TempDir()+"/missing", nil)
	if err == nil {
		t.Fatalf("expected missing image error")
	}
	if !strings.Contains(err.Error(), "missing Mach-O image") {
		t.Fatalf("error = %v", err)
	}
}

func TestWriteMachO64MacOSX64RejectsEntryOffsetOutOfRange(t *testing.T) {
	err := WriteMachO64MacOSX64(t.TempDir()+"/bad-entry", &MachOImage{
		Text:         []byte{0xc3},
		EntryTextOff: 2,
	})
	if err == nil {
		t.Fatalf("expected entry offset error")
	}
	if !strings.Contains(err.Error(), "entry offset out of range") {
		t.Fatalf("error = %v", err)
	}
}

func TestWriteMachO64MacOSX64RejectsOverflowSizedDataRelocOffset(t *testing.T) {
	err := WriteMachO64MacOSX64(t.TempDir()+"/bad-reloc", &MachOImage{
		Text:       []byte{0x90, 0x90, 0x90, 0x90},
		CString:    []byte("literal"),
		DataRelocs: []DataReloc{{At: int(^uint(0) >> 1)}},
	})
	if err == nil {
		t.Fatalf("expected relocation offset error")
	}
	if !strings.Contains(err.Error(), "relocation out of range") {
		t.Fatalf("error = %v", err)
	}
}

func TestWriteMachO64MacOSX64RejectsDataRelocTargetOutOfRange(t *testing.T) {
	err := WriteMachO64MacOSX64(t.TempDir()+"/bad-target", &MachOImage{
		Text:       []byte{0x90, 0x90, 0x90, 0x90},
		CString:    []byte("x"),
		DataRelocs: []DataReloc{{At: 0, TargetOff: 1}},
	})
	if err == nil {
		t.Fatalf("expected data relocation target error")
	}
	if !strings.Contains(err.Error(), "data relocation target out of range") {
		t.Fatalf("error = %v", err)
	}
}
