package elf

import (
	"encoding/binary"
	"os"
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

func TestWriteELF32LinuxX32HeaderContract(t *testing.T) {
	path := t.TempDir() + "/x32"
	img := &Image{
		Code:        []byte{0xc3},
		Data:        []byte{0x2a},
		EntryOffset: 0,
	}
	if err := WriteELF32LinuxX32(path, img); err != nil {
		t.Fatalf("write x32 ELF: %v", err)
	}

	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat x32 ELF: %v", err)
	}
	if st.Mode()&0o111 == 0 {
		t.Fatalf("x32 ELF output is not executable: mode=%v", st.Mode())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read x32 ELF: %v", err)
	}
	if len(data) < 52 {
		t.Fatalf("x32 ELF too small: %d bytes", len(data))
	}
	if string(data[:4]) != "\x7fELF" {
		t.Fatalf("missing ELF magic")
	}
	if data[4] != 1 {
		t.Fatalf("x32 must use ELFCLASS32, got %d", data[4])
	}
	if data[5] != 1 {
		t.Fatalf("x32 ELF must be little-endian, got %d", data[5])
	}
	if got := binary.LittleEndian.Uint16(data[16:18]); got != 2 {
		t.Fatalf("e_type = %d, want ET_EXEC", got)
	}
	if got := binary.LittleEndian.Uint16(data[18:20]); got != 0x3e {
		t.Fatalf("e_machine = %#x, want EM_X86_64", got)
	}
	layout := LinuxX32Layout(len(img.Code), len(img.Data))
	if got := binary.LittleEndian.Uint32(data[24:28]); got != uint32(LinuxX32BaseVaddr+layout.CodeOffset) {
		t.Fatalf("e_entry = %#x, want %#x", got, LinuxX32BaseVaddr+layout.CodeOffset)
	}
	if got := binary.LittleEndian.Uint32(data[28:32]); got != 52 {
		t.Fatalf("e_phoff = %d, want 52", got)
	}
	if got := binary.LittleEndian.Uint32(data[32:36]); got != 0 {
		t.Fatalf("e_shoff = %#x, want 0", got)
	}
	if got := binary.LittleEndian.Uint32(data[36:40]); got != 0 {
		t.Fatalf("e_flags = %#x, want 0", got)
	}
	if got := binary.LittleEndian.Uint16(data[40:42]); got != 52 {
		t.Fatalf("e_ehsize = %d, want 52", got)
	}
	if got := binary.LittleEndian.Uint16(data[42:44]); got != 32 {
		t.Fatalf("e_phentsize = %d, want 32", got)
	}
	if got := binary.LittleEndian.Uint16(data[44:46]); got != 2 {
		t.Fatalf("e_phnum = %d, want 2", got)
	}
	if layout.CodeOffset != 116 {
		t.Fatalf("x32 code offset = %d, want 116", layout.CodeOffset)
	}
	if layout.DataOffset != 0x1000 {
		t.Fatalf("x32 data offset = %#x, want 0x1000", layout.DataOffset)
	}
	if len(data) != layout.FileSize {
		t.Fatalf("file size = %#x, want %#x", len(data), layout.FileSize)
	}
}

func TestWriteELF32LinuxX86HeaderContract(t *testing.T) {
	path := t.TempDir() + "/x86"
	img := &Image{
		Code:        []byte{0xc3},
		Data:        []byte{0x2a},
		EntryOffset: 0,
	}
	if err := WriteELF32LinuxX86(path, img); err != nil {
		t.Fatalf("write x86 ELF: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read x86 ELF: %v", err)
	}
	if len(data) < 52 {
		t.Fatalf("x86 ELF too small: %d bytes", len(data))
	}
	if string(data[:4]) != "\x7fELF" {
		t.Fatalf("missing ELF magic")
	}
	if data[4] != 1 {
		t.Fatalf("x86 must use ELFCLASS32, got %d", data[4])
	}
	if got := binary.LittleEndian.Uint16(data[18:20]); got != 3 {
		t.Fatalf("e_machine = %#x, want EM_386", got)
	}
	layout := LinuxX86Layout(len(img.Code), len(img.Data))
	if got := binary.LittleEndian.Uint32(data[24:28]); got != uint32(LinuxX86BaseVaddr+layout.CodeOffset) {
		t.Fatalf("e_entry = %#x, want %#x", got, LinuxX86BaseVaddr+layout.CodeOffset)
	}
	if got := binary.LittleEndian.Uint16(data[40:42]); got != 52 {
		t.Fatalf("e_ehsize = %d, want 52", got)
	}
	if got := binary.LittleEndian.Uint16(data[42:44]); got != 32 {
		t.Fatalf("e_phentsize = %d, want 32", got)
	}
	if got := binary.LittleEndian.Uint16(data[44:46]); got != 2 {
		t.Fatalf("e_phnum = %d, want 2", got)
	}
	if len(data) != layout.FileSize {
		t.Fatalf("file size = %#x, want %#x", len(data), layout.FileSize)
	}
}

func TestWriteELF32LinuxX32RejectsInvalidImage(t *testing.T) {
	if err := WriteELF32LinuxX32(t.TempDir()+"/missing", nil); err == nil || !strings.Contains(err.Error(), "missing ELF image") {
		t.Fatalf("missing image error = %v", err)
	}
	err := WriteELF32LinuxX32(t.TempDir()+"/bad-entry", &Image{
		Code:        []byte{0xc3},
		EntryOffset: 2,
	})
	if err == nil || !strings.Contains(err.Error(), "entry offset out of range") {
		t.Fatalf("entry offset error = %v", err)
	}
}
