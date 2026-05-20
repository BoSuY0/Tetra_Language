package compiler_test

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"

	"tetra_language/compiler/internal/format/elf"
)

type elfProgramHeader struct {
	pType   uint32
	pFlags  uint32
	pOffset uint64
	pVaddr  uint64
	pFilesz uint64
}

func parseELF64ProgramHeaders(t *testing.T, data []byte) []elfProgramHeader {
	t.Helper()

	if len(data) < 64 {
		t.Fatalf("file too small")
	}
	if !bytes.Equal(data[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("missing ELF magic")
	}
	if data[4] != 2 {
		t.Fatalf("expected ELF64")
	}
	if data[5] != 1 {
		t.Fatalf("expected little-endian")
	}
	ePhOff := binary.LittleEndian.Uint64(data[32:40])
	eEhSize := binary.LittleEndian.Uint16(data[52:54])
	ePhEntSize := binary.LittleEndian.Uint16(data[54:56])
	ePhNum := binary.LittleEndian.Uint16(data[56:58])
	if eEhSize != 64 {
		t.Fatalf("unexpected ELF header size: %d", eEhSize)
	}
	if ePhEntSize != 56 {
		t.Fatalf("unexpected program header size: %d", ePhEntSize)
	}
	if ePhNum == 0 {
		t.Fatalf("missing program headers")
	}
	if int(ePhOff)+int(ePhEntSize)*int(ePhNum) > len(data) {
		t.Fatalf("truncated program headers")
	}

	phdrs := make([]elfProgramHeader, 0, int(ePhNum))
	for i := 0; i < int(ePhNum); i++ {
		off := int(ePhOff) + (i * int(ePhEntSize))
		pType := binary.LittleEndian.Uint32(data[off : off+4])
		pFlags := binary.LittleEndian.Uint32(data[off+4 : off+8])
		pOffset := binary.LittleEndian.Uint64(data[off+8 : off+16])
		pVaddr := binary.LittleEndian.Uint64(data[off+16 : off+24])
		pFilesz := binary.LittleEndian.Uint64(data[off+32 : off+40])
		phdrs = append(phdrs, elfProgramHeader{
			pType:   pType,
			pFlags:  pFlags,
			pOffset: pOffset,
			pVaddr:  pVaddr,
			pFilesz: pFilesz,
		})
	}
	return phdrs
}

func findELFLoadSegment(t *testing.T, phdrs []elfProgramHeader, wantFlags uint32) elfProgramHeader {
	t.Helper()
	for _, ph := range phdrs {
		const ptLoad = 1
		if ph.pType != ptLoad {
			continue
		}
		if ph.pFlags == wantFlags {
			return ph
		}
	}
	t.Fatalf("missing PT_LOAD segment with flags=0x%x", wantFlags)
	return elfProgramHeader{}
}

func TestELFHasRWDataSegmentAndStringNotInText(t *testing.T) {
	tmp := t.TempDir()
	marker := "ELF_DATA_MARKER"
	src := "fun main(): i32 uses io {\n  print(\"" + marker + "\")\n  return 0\n}\n"
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := compiler.BuildFile(srcPath, outPath, "linux-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read elf: %v", err)
	}
	phdrs := parseELF64ProgramHeaders(t, data)

	const (
		pfX = 1
		pfW = 2
		pfR = 4
	)
	codeSeg := findELFLoadSegment(t, phdrs, pfR|pfX)
	dataSeg := findELFLoadSegment(t, phdrs, pfR|pfW)
	if codeSeg.pOffset != 0 {
		t.Fatalf("unexpected code segment file offset: %d", codeSeg.pOffset)
	}
	if dataSeg.pOffset == 0 || dataSeg.pFilesz == 0 {
		t.Fatalf("unexpected data segment: off=%d size=%d", dataSeg.pOffset, dataSeg.pFilesz)
	}
	if dataSeg.pOffset+dataSeg.pFilesz > uint64(len(data)) {
		t.Fatalf("truncated data segment")
	}

	codeBytes := data[:dataSeg.pOffset]
	dataBytes := data[dataSeg.pOffset : dataSeg.pOffset+dataSeg.pFilesz]
	if !bytes.Contains(dataBytes, []byte(marker)) {
		t.Fatalf("missing marker in RW data segment")
	}
	if bytes.Contains(codeBytes, []byte(marker)) {
		t.Fatalf("marker should not be in RX segment")
	}
}

func TestELFExecutableModeAndHeaderContract(t *testing.T) {
	tmp := t.TempDir()
	src := "fun main(): i32 uses io {\n  return 0\n}\n"
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := compiler.BuildFile(srcPath, outPath, "linux-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}

	st, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("stat ELF: %v", err)
	}
	if st.Mode()&0o111 == 0 {
		t.Fatalf("ELF output is not executable: mode=%v", st.Mode())
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read elf: %v", err)
	}
	if binary.LittleEndian.Uint64(data[40:48]) != 0 {
		t.Fatalf("expected no ELF section header table in segment-only executable")
	}
	if binary.LittleEndian.Uint16(data[60:62]) != 0 {
		t.Fatalf("expected section header count 0")
	}
	phdrs := parseELF64ProgramHeaders(t, data)
	if len(phdrs) != 2 {
		t.Fatalf("program header count = %d, want 2", len(phdrs))
	}
}

func TestELFDataRelocPointsToDataMarker(t *testing.T) {
	tmp := t.TempDir()
	marker := "ELF_RELOC_MARKER"
	src := "fun main(): i32 uses io {\n  print(\"" + marker + "\")\n  return 0\n}\n"
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := compiler.BuildFile(srcPath, outPath, "linux-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read elf: %v", err)
	}
	phdrs := parseELF64ProgramHeaders(t, data)

	const (
		pfX = 1
		pfW = 2
		pfR = 4
	)
	codeSeg := findELFLoadSegment(t, phdrs, pfR|pfX)
	dataSeg := findELFLoadSegment(t, phdrs, pfR|pfW)
	baseVaddr := codeSeg.pVaddr - codeSeg.pOffset

	ePhOff := binary.LittleEndian.Uint64(data[32:40])
	ePhEntSize := binary.LittleEndian.Uint16(data[54:56])
	ePhNum := binary.LittleEndian.Uint16(data[56:58])
	codeStart := ePhOff + uint64(ePhEntSize)*uint64(ePhNum)
	if codeStart >= dataSeg.pOffset {
		t.Fatalf("invalid code start offset")
	}

	text := data[codeStart:dataSeg.pOffset]
	textVaddr := baseVaddr + codeStart

	found := false
	for i := 0; i+7 <= len(text); i++ {
		if text[i] != 0x48 || text[i+1] != 0x8D {
			continue
		}
		if text[i+2] != 0x05 && text[i+2] != 0x15 && text[i+2] != 0x35 {
			continue
		}
		disp := int32(binary.LittleEndian.Uint32(text[i+3 : i+7]))
		next := int64(textVaddr) + int64(i+7)
		target := uint64(next + int64(disp))
		if target < dataSeg.pVaddr || target >= dataSeg.pVaddr+dataSeg.pFilesz {
			continue
		}
		targetOff := dataSeg.pOffset + (target - dataSeg.pVaddr)
		if targetOff >= uint64(len(data)) {
			continue
		}
		if bytes.HasPrefix(data[targetOff:], []byte(marker)) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing lea relocation to data marker")
	}
}

func TestELFBuildsHighArityCallSurface(t *testing.T) {
	tmp := t.TempDir()
	src := `
fun f7(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32): i32 {
  return a + b + c + d + e + f + g
}
fun main(): i32 {
  return f7(1, 2, 3, 4, 5, 6, 7)
}
`
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := compiler.BuildFile(srcPath, outPath, "linux-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read elf: %v", err)
	}
	phdrs := parseELF64ProgramHeaders(t, data)
	if len(phdrs) != 2 {
		t.Fatalf("program header count = %d, want 2", len(phdrs))
	}
}

func TestELFLinkRejectsNonLinuxObjectTarget(t *testing.T) {
	_, err := compiler.LinkLinuxX64([]*compiler.Object{{
		Target:  "windows-x64",
		Module:  "wrong",
		Code:    []byte{0xC3},
		Symbols: []compiler.Symbol{{Name: "main", Offset: 0}},
	}}, "main")
	if err == nil {
		t.Fatalf("expected mismatch error")
	}
	if !strings.Contains(err.Error(), "linker target mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestELFLinuxLayoutStaysInSyncWithWriter(t *testing.T) {
	layout := elf.LinuxX64Layout(1, 1)
	if layout.CodeOffset == 0 || layout.DataOffset == 0 || layout.FileSize == 0 {
		t.Fatalf("unexpected zero layout")
	}
	if layout.DataOffset <= layout.CodeOffset {
		t.Fatalf("expected data offset > code offset")
	}
	if layout.FileSize != layout.DataOffset+1 {
		t.Fatalf("unexpected file size")
	}
}
