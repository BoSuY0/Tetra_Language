package compiler_test

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	compiler "tetra_language/compiler"

	"tetra_language/compiler/internal/format/macho"
	"tetra_language/compiler/internal/testkit"
)

func TestBuildMachOHeaders(t *testing.T) {
	tmp := t.TempDir()
	src := "fun main(): i32 uses io {\n  print(\"Hi\")\n  return 0\n}\n"
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := compiler.BuildFile(srcPath, outPath, "macos-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read macho: %v", err)
	}
	info := parseMachOInfo(t, data)
	if info.magic != macho.MachOMagic64 {
		t.Fatalf("bad magic: 0x%x", info.magic)
	}
	if info.cpuType != macho.MachOCpuTypeX86_64 {
		t.Fatalf("bad cpu type: 0x%x", info.cpuType)
	}
	if info.fileType != macho.MachOFiletypeExecute {
		t.Fatalf("bad file type: 0x%x", info.fileType)
	}
	if info.entryOff == 0 {
		t.Fatalf("missing entry offset")
	}
	if info.ncmds != 3 {
		t.Fatalf("load command count = %d, want 3", info.ncmds)
	}
	findMachOSection(t, info.sections, "__TEXT", "__text")
	findMachOSection(t, info.sections, "__DATA", "__cstring")
	if len(info.sections) != 2 {
		t.Fatalf("section count = %d, want 2", len(info.sections))
	}
}

func TestMachOCStringInSection(t *testing.T) {
	tmp := t.TempDir()
	marker := "HELLO_CSTRING"
	src := "fun main(): i32 uses io {\n  print(\"" + marker + "\")\n  return 0\n}\n"
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := compiler.BuildFile(srcPath, outPath, "macos-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read macho: %v", err)
	}
	info := parseMachOInfo(t, data)
	text := machoSectionData(t, data, info.sections, "__TEXT", "__text")
	cstring := machoSectionData(t, data, info.sections, "__DATA", "__cstring")
	if !bytes.Contains(cstring, []byte(marker)) {
		t.Fatalf("missing marker in __cstring")
	}
	if bytes.Contains(text, []byte(marker)) {
		t.Fatalf("marker should not be in __text")
	}
}

func TestMachOCStringRelocPointsToCstring(t *testing.T) {
	tmp := t.TempDir()
	marker := "CSTRING_RELOC"
	src := "fun main(): i32 uses io {\n  print(\"" + marker + "\")\n  return 0\n}\n"
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := compiler.BuildFile(srcPath, outPath, "macos-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read macho: %v", err)
	}
	info := parseMachOInfo(t, data)
	textSec := findMachOSection(t, info.sections, "__TEXT", "__text")
	cstringSec := findMachOSection(t, info.sections, "__DATA", "__cstring")
	text := machoSectionData(t, data, info.sections, "__TEXT", "__text")

	found := false
	for i := 0; i+7 <= len(text); i++ {
		if text[i] != 0x48 || text[i+1] != 0x8D {
			continue
		}
		if text[i+2] != 0x05 && text[i+2] != 0x15 && text[i+2] != 0x35 {
			continue
		}
		disp := int32(binary.LittleEndian.Uint32(text[i+3 : i+7]))
		next := int64(textSec.addr) + int64(i+7)
		target := uint64(next + int64(disp))
		if target < cstringSec.addr || target >= cstringSec.addr+cstringSec.size {
			continue
		}
		targetOff := cstringSec.offset + uint32(target-cstringSec.addr)
		if int(targetOff) >= len(data) {
			continue
		}
		if bytes.HasPrefix(data[targetOff:], []byte(marker)) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing lea relocation to __cstring marker")
	}
}

func TestBuildMachOMmio(t *testing.T) {
	tmp := t.TempDir()
	srcPath := testkit.RepoPath(t, "examples", "mmio_smoke.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}
	outPath := filepath.Join(tmp, "mmio")
	if err := compiler.BuildFile(srcPath, outPath, "macos-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("missing output: %v", err)
	}
}

func TestBuildMachOActors(t *testing.T) {
	tmp := t.TempDir()
	srcPath := testkit.RepoPath(t, "examples", "actors_pingpong.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}
	outPath := filepath.Join(tmp, "actors")
	if err := compiler.BuildFile(srcPath, outPath, "macos-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("missing output: %v", err)
	}
}

func TestMachORuntimeOnDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	tmp := t.TempDir()
	src := "fun main(): i32 uses io {\n  print(\"OK\\n\")\n  return 7\n}\n"
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := compiler.BuildFile(srcPath, outPath, "macos-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	stdout, exitCode := testkit.RunBinary(t, outPath)
	if stdout != "OK\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 7 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestMachOIslandsRuntimeOnDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	tmp := t.TempDir()
	srcPath := testkit.RepoPath(t, "examples", "islands_i32.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}
	outPath := filepath.Join(tmp, "app")
	if err := compiler.BuildFile(srcPath, outPath, "macos-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	stdout, exitCode := testkit.RunBinary(t, outPath)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 55 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestMachOBuildsHighArityCallSurface(t *testing.T) {
	tmp := t.TempDir()
	src := `
fun f8(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32, h: i32): i32 {
  return a + b + c + d + e + f + g + h
}
fun main(): i32 {
  return f8(1, 2, 3, 4, 5, 6, 7, 8)
}
`
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := compiler.BuildFile(srcPath, outPath, "macos-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read macho: %v", err)
	}
	info := parseMachOInfo(t, data)
	if info.entryOff == 0 {
		t.Fatalf("missing entry offset")
	}
}

func TestMachOLinkRejectsNonMacOSObjectTarget(t *testing.T) {
	_, err := compiler.LinkMacOSX64([]*compiler.Object{{
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

type machoSection struct {
	segname  string
	sectname string
	addr     uint64
	size     uint64
	offset   uint32
}

type machoInfo struct {
	magic    uint32
	cpuType  uint32
	fileType uint32
	ncmds    uint32
	entryOff uint64
	sections []machoSection
}

func parseMachOInfo(t *testing.T, data []byte) machoInfo {
	t.Helper()
	if len(data) < 32 {
		t.Fatalf("file too small")
	}
	magic := binary.LittleEndian.Uint32(data[0:])
	cpuType := binary.LittleEndian.Uint32(data[4:])
	fileType := binary.LittleEndian.Uint32(data[12:])
	ncmds := binary.LittleEndian.Uint32(data[16:])
	sizeofcmds := binary.LittleEndian.Uint32(data[20:])
	if len(data) < 32+int(sizeofcmds) {
		t.Fatalf("truncated load commands")
	}
	off := 32
	var entryOff uint64
	var sections []machoSection

	for i := 0; i < int(ncmds); i++ {
		if off+8 > len(data) {
			t.Fatalf("truncated load command")
		}
		cmd := binary.LittleEndian.Uint32(data[off:])
		cmdsize := binary.LittleEndian.Uint32(data[off+4:])
		if cmdsize == 0 || off+int(cmdsize) > len(data) {
			t.Fatalf("invalid load command size")
		}
		switch cmd {
		case macho.MachOCmdSegment64:
			segOff := off
			if segOff+72 > len(data) {
				t.Fatalf("truncated segment command")
			}
			segname := readMachOName(data[segOff+8 : segOff+24])
			nsects := binary.LittleEndian.Uint32(data[segOff+64:])
			sectOff := segOff + 72
			for s := 0; s < int(nsects); s++ {
				if sectOff+80 > len(data) {
					t.Fatalf("truncated section")
				}
				sectname := readMachOName(data[sectOff : sectOff+16])
				sectSeg := readMachOName(data[sectOff+16 : sectOff+32])
				addr := binary.LittleEndian.Uint64(data[sectOff+32:])
				size := binary.LittleEndian.Uint64(data[sectOff+40:])
				offset := binary.LittleEndian.Uint32(data[sectOff+48:])
				sections = append(sections, machoSection{
					segname:  sectSeg,
					sectname: sectname,
					addr:     addr,
					size:     size,
					offset:   offset,
				})
				sectOff += 80
			}
			if segname == "" {
				t.Fatalf("missing segment name")
			}
		case macho.MachOCmdMain:
			entryOff = binary.LittleEndian.Uint64(data[off+8:])
		}
		off += int(cmdsize)
	}

	return machoInfo{
		magic:    magic,
		cpuType:  cpuType,
		fileType: fileType,
		ncmds:    ncmds,
		entryOff: entryOff,
		sections: sections,
	}
}

func readMachOName(b []byte) string {
	for i, ch := range b {
		if ch == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func findMachOSection(t *testing.T, sections []machoSection, seg, sect string) machoSection {
	t.Helper()
	for _, sec := range sections {
		if sec.segname == seg && sec.sectname == sect {
			return sec
		}
	}
	t.Fatalf("section %s,%s not found", seg, sect)
	return machoSection{}
}

func machoSectionData(t *testing.T, data []byte, sections []machoSection, seg, sect string) []byte {
	t.Helper()
	sec := findMachOSection(t, sections, seg, sect)
	start := int(sec.offset)
	end := start + int(sec.size)
	if start < 0 || end > len(data) || start > end {
		t.Fatalf("section %s,%s out of range", seg, sect)
	}
	return data[start:end]
}
