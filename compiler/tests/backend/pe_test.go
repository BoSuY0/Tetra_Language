package compiler_test

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"

	"tetra_language/compiler/internal/format/pe"
	"tetra_language/compiler/internal/testkit"
)

func TestBuildWindowsPEHeaders(t *testing.T) {
	tmp := t.TempDir()
	src := "fun main(): i32 uses io {\n  print(\"Hi\\n\")\n  return 7\n}\n"
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app.exe")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := compiler.BuildFile(srcPath, outPath, "windows-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read exe: %v", err)
	}
	info := parsePEInfo(t, data)
	if info.machine != 0x8664 {
		t.Fatalf("machine mismatch: %x", info.machine)
	}
	if info.optionalMagic != 0x20b {
		t.Fatalf("optional header magic mismatch: %x", info.optionalMagic)
	}
	if info.entry == 0 {
		t.Fatalf("entrypoint is zero")
	}
	if info.importRVA == 0 || info.importSize == 0 {
		t.Fatalf("missing import directory")
	}
	for _, name := range []string{".text", ".rdata", ".idata", ".reloc"} {
		findSection(t, info.sections, name)
	}
	idata := findSection(t, info.sections, ".idata")
	if info.importRVA < idata.virtualAddress ||
		info.importRVA >= idata.virtualAddress+idata.virtualSize {
		t.Fatalf("import directory outside .idata section")
	}
	textSec := findSection(t, info.sections, ".text")
	if info.entry < textSec.virtualAddress ||
		info.entry >= textSec.virtualAddress+textSec.virtualSize {
		t.Fatalf("entrypoint outside .text section")
	}

	imports := readPEImports(t, data, info)
	dll, ok := imports["KERNEL32.dll"]
	if !ok {
		t.Fatalf("missing KERNEL32.dll import")
	}
	assertImportHas(t, dll, "ExitProcess")
	assertImportHas(t, dll, "GetStdHandle")
	assertImportHas(t, dll, "WriteFile")

	textBytes := sectionData(t, data, info.sections, ".text")
	if !bytes.Contains(textBytes, []byte{0xFF, 0x15}) {
		t.Fatalf("expected indirect call pattern in .text")
	}
}

func TestPEImportsIncludeIslands(t *testing.T) {
	tmp := t.TempDir()
	srcPath := testkit.RepoPath(t, "examples", "memory", "islands", "islands_hello.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}
	outPath := filepath.Join(tmp, "islands.exe")
	if err := compiler.BuildFile(srcPath, outPath, "windows-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read exe: %v", err)
	}
	info := parsePEInfo(t, data)
	imports := readPEImports(t, data, info)
	dll, ok := imports["KERNEL32.dll"]
	if !ok {
		t.Fatalf("missing KERNEL32.dll import")
	}
	assertImportHas(t, dll, "VirtualAlloc")
	assertImportHas(t, dll, "VirtualFree")
}

func TestBuildWindowsPEMmio(t *testing.T) {
	tmp := t.TempDir()
	srcPath := testkit.RepoPath(t, "examples", "memory", "raw", "mmio_smoke.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}
	outPath := filepath.Join(tmp, "mmio.exe")
	if err := compiler.BuildFile(srcPath, outPath, "windows-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("missing output: %v", err)
	}
}

func TestBuildWindowsPEActors(t *testing.T) {
	tmp := t.TempDir()
	srcPath := testkit.RepoPath(t, "examples", "actors", "actors_pingpong.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}
	outPath := filepath.Join(tmp, "actors.exe")
	if err := compiler.BuildFile(srcPath, outPath, "windows-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read exe: %v", err)
	}
	info := parsePEInfo(t, data)
	imports := readPEImports(t, data, info)
	dll, ok := imports["KERNEL32.dll"]
	if !ok {
		t.Fatalf("missing KERNEL32.dll import")
	}
	assertImportHas(t, dll, "VirtualAlloc")
}

func TestPEHasRDataSectionAndStringNotInText(t *testing.T) {
	tmp := t.TempDir()
	marker := "HELLO_RDATA"
	src := "fun main(): i32 uses io {\n  print(\"" + marker + "\")\n  return 0\n}\n"
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app.exe")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := compiler.BuildFile(srcPath, outPath, "windows-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read exe: %v", err)
	}
	info := parsePEInfo(t, data)
	text := sectionData(t, data, info.sections, ".text")
	rdata := sectionData(t, data, info.sections, ".rdata")
	if !bytes.Contains(rdata, []byte(marker)) {
		t.Fatalf("missing marker in .rdata")
	}
	if bytes.Contains(text, []byte(marker)) {
		t.Fatalf("marker should not be in .text")
	}
}

func TestPERDataRelocPointsToRData(t *testing.T) {
	tmp := t.TempDir()
	marker := "RDATA_RELOC"
	src := "fun main(): i32 uses io {\n  print(\"" + marker + "\")\n  return 0\n}\n"
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app.exe")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := compiler.BuildFile(srcPath, outPath, "windows-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read exe: %v", err)
	}
	info := parsePEInfo(t, data)
	textSec := findSection(t, info.sections, ".text")
	rdataSec := findSection(t, info.sections, ".rdata")
	text := sectionData(t, data, info.sections, ".text")

	found := false
	for i := 0; i+7 <= len(text); i++ {
		if text[i] != 0x48 || text[i+1] != 0x8D {
			continue
		}
		if text[i+2] != 0x05 && text[i+2] != 0x15 && text[i+2] != 0x35 {
			continue
		}
		disp := int32(binary.LittleEndian.Uint32(text[i+3 : i+7]))
		next := int64(textSec.virtualAddress) + int64(i+7)
		targetRVA := uint32(next + int64(disp))
		if targetRVA < rdataSec.virtualAddress ||
			targetRVA >= rdataSec.virtualAddress+rdataSec.virtualSize {
			continue
		}
		targetOff, ok := rvaToOffset(targetRVA, info.sections)
		if !ok || int(targetOff) >= len(data) {
			continue
		}
		if bytes.HasPrefix(data[targetOff:], []byte(marker)) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing lea relocation to .rdata marker")
	}
}

func TestPEDllCharacteristicsHasNXAndASLR(t *testing.T) {
	tmp := t.TempDir()
	src := "fun main(): i32 uses io {\n  return 0\n}\n"
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app.exe")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := compiler.BuildFile(srcPath, outPath, "windows-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read exe: %v", err)
	}
	info := parsePEInfo(t, data)
	const nxCompat = 0x0100
	const dynamicBase = 0x0040
	if info.dllCharacteristics&nxCompat == 0 || info.dllCharacteristics&dynamicBase == 0 {
		t.Fatalf("dll characteristics missing NX/ASLR: 0x%x", info.dllCharacteristics)
	}
}

func TestPEHasRelocDirectoryAndRelocSection(t *testing.T) {
	tmp := t.TempDir()
	src := "fun main(): i32 uses io {\n  return 0\n}\n"
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app.exe")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := compiler.BuildFile(srcPath, outPath, "windows-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read exe: %v", err)
	}
	info := parsePEInfo(t, data)
	if info.relocRVA == 0 || info.relocSize == 0 {
		t.Fatalf("missing reloc directory")
	}
	relocSec := findSection(t, info.sections, ".reloc")
	if info.relocRVA < relocSec.virtualAddress ||
		info.relocRVA >= relocSec.virtualAddress+relocSec.virtualSize {
		t.Fatalf("reloc directory outside .reloc section")
	}
}

func TestPEMultiDLLImports(t *testing.T) {
	idataRVA := uint32(0x2000)
	idata, _, _, _, _, err := pe.BuildImportSection(idataRVA, []string{
		"kernel32.ExitProcess",
		"user32.MessageBoxA",
	})
	if err != nil {
		t.Fatalf("build import section: %v", err)
	}
	info := peInfo{
		importRVA: idataRVA,
		sections: []peSection{{
			name:           ".idata",
			virtualAddress: idataRVA,
			virtualSize:    uint32(len(idata)),
			rawSize:        uint32(len(idata)),
			rawOffset:      0,
		}},
	}
	imports := readPEImports(t, idata, info)
	kernel, ok := imports["KERNEL32.dll"]
	if !ok {
		t.Fatalf("missing KERNEL32.dll import")
	}
	user, ok := imports["USER32.dll"]
	if !ok {
		t.Fatalf("missing USER32.dll import")
	}
	assertImportHas(t, kernel, "ExitProcess")
	assertImportHas(t, user, "MessageBoxA")
}

func TestPEBuildsHighArityCallSurface(t *testing.T) {
	tmp := t.TempDir()
	src := `
fun f10(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32, h: i32, i: i32, j: i32): i32 {
  return a + b + c + d + e + f + g + h + i + j
}
fun main(): i32 {
  return f10(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
}
`
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app.exe")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := compiler.BuildFile(srcPath, outPath, "windows-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read exe: %v", err)
	}
	info := parsePEInfo(t, data)
	textSec := findSection(t, info.sections, ".text")
	if info.entry < textSec.virtualAddress ||
		info.entry >= textSec.virtualAddress+textSec.virtualSize {
		t.Fatalf("entrypoint outside .text for high-arity sample")
	}
}

func TestPELinkRejectsNonWindowsObjectTarget(t *testing.T) {
	_, err := compiler.LinkWindowsX64([]*compiler.Object{{
		Target:  "linux-x64",
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

type peSection struct {
	name           string
	virtualSize    uint32
	virtualAddress uint32
	rawSize        uint32
	rawOffset      uint32
}

type peInfo struct {
	machine            uint16
	optionalMagic      uint16
	entry              uint32
	importRVA          uint32
	importSize         uint32
	relocRVA           uint32
	relocSize          uint32
	dllCharacteristics uint16
	sections           []peSection
}

func parsePEInfo(t *testing.T, data []byte) peInfo {
	t.Helper()
	if len(data) < 0x40 {
		t.Fatalf("file too small")
	}
	if !bytes.Equal(data[:2], []byte{'M', 'Z'}) {
		t.Fatalf("missing MZ header")
	}
	peOff := int(binary.LittleEndian.Uint32(data[0x3c:0x40]))
	if peOff+4 > len(data) || !bytes.Equal(data[peOff:peOff+4], []byte{'P', 'E', 0, 0}) {
		t.Fatalf("missing PE header")
	}

	coffOff := peOff + 4
	if coffOff+20 > len(data) {
		t.Fatalf("truncated COFF header")
	}
	machine := binary.LittleEndian.Uint16(data[coffOff:])
	numSections := binary.LittleEndian.Uint16(data[coffOff+2:])
	optSize := binary.LittleEndian.Uint16(data[coffOff+16:])
	optOff := coffOff + 20
	if optOff+int(optSize) > len(data) {
		t.Fatalf("truncated optional header")
	}
	optionalMagic := binary.LittleEndian.Uint16(data[optOff:])
	entry := binary.LittleEndian.Uint32(data[optOff+16:])
	dllCharacteristics := binary.LittleEndian.Uint16(data[optOff+70:])
	importRVA := binary.LittleEndian.Uint32(data[optOff+112+8:])
	importSize := binary.LittleEndian.Uint32(data[optOff+112+8+4:])
	relocRVA := binary.LittleEndian.Uint32(data[optOff+112+5*8:])
	relocSize := binary.LittleEndian.Uint32(data[optOff+112+5*8+4:])

	secOff := optOff + int(optSize)
	sections := make([]peSection, 0, numSections)
	for i := 0; i < int(numSections); i++ {
		off := secOff + i*40
		if off+40 > len(data) {
			t.Fatalf("truncated section header")
		}
		name := strings.TrimRight(string(data[off:off+8]), "\x00")
		sections = append(sections, peSection{
			name:           name,
			virtualSize:    binary.LittleEndian.Uint32(data[off+8:]),
			virtualAddress: binary.LittleEndian.Uint32(data[off+12:]),
			rawSize:        binary.LittleEndian.Uint32(data[off+16:]),
			rawOffset:      binary.LittleEndian.Uint32(data[off+20:]),
		})
	}

	return peInfo{
		machine:            machine,
		optionalMagic:      optionalMagic,
		entry:              entry,
		importRVA:          importRVA,
		importSize:         importSize,
		relocRVA:           relocRVA,
		relocSize:          relocSize,
		dllCharacteristics: dllCharacteristics,
		sections:           sections,
	}
}

func readPEImports(t *testing.T, data []byte, info peInfo) map[string][]string {
	t.Helper()
	imports := make(map[string][]string)
	if info.importRVA == 0 {
		return imports
	}
	descOff, ok := rvaToOffset(info.importRVA, info.sections)
	if !ok {
		t.Fatalf("import RVA not in sections")
	}

	for {
		if int(descOff)+20 > len(data) {
			t.Fatalf("truncated import descriptor")
		}
		desc := data[descOff : descOff+20]
		if bytes.Equal(desc, make([]byte, 20)) {
			break
		}
		iltRVA := binary.LittleEndian.Uint32(desc[0:])
		nameRVA := binary.LittleEndian.Uint32(desc[12:])
		nameOff, ok := rvaToOffset(nameRVA, info.sections)
		if !ok {
			t.Fatalf("import name RVA not in sections")
		}
		dll := readCString(data[nameOff:])

		iltOff, ok := rvaToOffset(iltRVA, info.sections)
		if !ok {
			t.Fatalf("ILT RVA not in sections")
		}
		var funcs []string
		for idx := 0; ; idx++ {
			entryOff := int(iltOff) + idx*8
			if entryOff+8 > len(data) {
				t.Fatalf("truncated ILT")
			}
			val := binary.LittleEndian.Uint64(data[entryOff : entryOff+8])
			if val == 0 {
				break
			}
			if val&(1<<63) != 0 {
				continue
			}
			nameOff, ok := rvaToOffset(uint32(val), info.sections)
			if !ok {
				t.Fatalf("hint/name RVA not in sections")
			}
			funcName := readCString(data[nameOff+2:])
			funcs = append(funcs, funcName)
		}
		imports[dll] = funcs
		descOff += 20
	}

	return imports
}

func assertImportHas(t *testing.T, funcs []string, want string) {
	t.Helper()
	for _, name := range funcs {
		if name == want {
			return
		}
	}
	t.Fatalf("missing import: %s", want)
}

func rvaToOffset(rva uint32, sections []peSection) (uint32, bool) {
	for _, sec := range sections {
		size := sec.virtualSize
		if sec.rawSize > size {
			size = sec.rawSize
		}
		if rva >= sec.virtualAddress && rva < sec.virtualAddress+size {
			return sec.rawOffset + (rva - sec.virtualAddress), true
		}
	}
	return 0, false
}

func readCString(b []byte) string {
	for i, ch := range b {
		if ch == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func sectionData(t *testing.T, data []byte, sections []peSection, name string) []byte {
	t.Helper()
	for _, sec := range sections {
		if sec.name != name {
			continue
		}
		start := int(sec.rawOffset)
		end := start + int(sec.rawSize)
		if start < 0 || end > len(data) || start > end {
			t.Fatalf("section %s out of range", name)
		}
		return data[start:end]
	}
	t.Fatalf("section %s not found", name)
	return nil
}

func findSection(t *testing.T, sections []peSection, name string) peSection {
	t.Helper()
	for _, sec := range sections {
		if sec.name == name {
			return sec
		}
	}
	t.Fatalf("section %s not found", name)
	return peSection{}
}
