package elf

import (
	"strings"
	"testing"
)

func TestWriteELF64LinuxX64RejectsMissingImage(t *testing.T) {
	err := WriteELF64LinuxX64(t.TempDir()+"/missing", nil)
	if err == nil {
		t.Fatalf("expected missing image error")
	}
	if !strings.Contains(err.Error(), "missing ELF image") {
		t.Fatalf("error = %v", err)
	}
}

func TestWriteELF64LinuxX64RejectsEntryOffsetOutOfRange(t *testing.T) {
	err := WriteELF64LinuxX64(t.TempDir()+"/bad-entry", &Image{
		Code:        []byte{0xc3},
		EntryOffset: 2,
	})
	if err == nil {
		t.Fatalf("expected entry offset error")
	}
	if !strings.Contains(err.Error(), "entry offset out of range") {
		t.Fatalf("error = %v", err)
	}
}

func TestLinuxX64LayoutKeepsDataPageAligned(t *testing.T) {
	layout := LinuxX64Layout(3, 5)
	if layout.CodeOffset != 176 {
		t.Fatalf("code offset = %d, want 176", layout.CodeOffset)
	}
	if layout.DataOffset != 0x1000 {
		t.Fatalf("data offset = %#x, want 0x1000", layout.DataOffset)
	}
	if layout.FileSize != 0x1005 {
		t.Fatalf("file size = %#x, want 0x1005", layout.FileSize)
	}
}
