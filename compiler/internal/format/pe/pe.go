package pe

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	peImageBase       = 0x140000000
	peSectionAlign    = 0x1000
	peFileAlign       = 0x200
	peOptionalHdrSize = 0xF0
	peDosStubSize     = 0x80
)

type IATReloc struct {
	At   int
	Name string
}

type RDataReloc struct {
	At        int
	TargetOff uint32
}

type PEImage struct {
	Text        []byte
	RData       []byte
	EntryOffset uint32
	Imports     []string
	IATRelocs   []IATReloc
	RDataRelocs []RDataReloc
}

func WritePE64WindowsX64(path string, img *PEImage) error {
	if img == nil {
		return fmt.Errorf("missing PE image")
	}
	if img.EntryOffset > uint32(len(img.Text)) {
		return fmt.Errorf("entry offset out of range")
	}
	if len(img.Imports) == 0 {
		return fmt.Errorf("missing imports")
	}

	textData := append([]byte(nil), img.Text...)
	rdataData := append([]byte(nil), img.RData...)

	headersSize := alignUp(peDosStubSize+4+20+peOptionalHdrSize+4*40, peFileAlign)
	textRVA := alignUp(headersSize, peSectionAlign)
	textRawOffset := headersSize
	textRawSize := alignUp(len(textData), peFileAlign)
	textVirtSize := len(textData)
	textVirtAligned := alignUp(textVirtSize, peSectionAlign)

	rdataRVA := textRVA + textVirtAligned
	rdataRawOffset := textRawOffset + textRawSize
	rdataRawSize := alignUp(len(rdataData), peFileAlign)
	rdataVirtSize := len(rdataData)
	rdataVirtAligned := alignUp(rdataVirtSize, peSectionAlign)

	idataRVA := rdataRVA + rdataVirtAligned
	idataRawOffset := rdataRawOffset + rdataRawSize

	idataData, iatRVAs, importDirSize, iatRVA, iatSize, err := BuildImportSection(uint32(idataRVA), img.Imports)
	if err != nil {
		return err
	}

	for _, reloc := range img.IATRelocs {
		key, err := normalizeImportKey(reloc.Name)
		if err != nil {
			return err
		}
		targetRVA, ok := iatRVAs[key]
		if !ok {
			return fmt.Errorf("missing IAT entry for '%s'", reloc.Name)
		}
		if !validDisp32PatchOffset(textData, reloc.At) {
			return fmt.Errorf("IAT relocation out of range for '%s'", reloc.Name)
		}
		if err := patchRipDisp32(textData, reloc.At, uint32(textRVA), targetRVA); err != nil {
			return err
		}
	}

	for _, reloc := range img.RDataRelocs {
		if !validDisp32PatchOffset(textData, reloc.At) {
			return fmt.Errorf("rdata relocation out of range")
		}
		if reloc.TargetOff >= uint32(len(rdataData)) {
			return fmt.Errorf("rdata relocation target out of range")
		}
		targetRVA := uint32(rdataRVA) + reloc.TargetOff
		if err := patchRipDisp32(textData, reloc.At, uint32(textRVA), targetRVA); err != nil {
			return err
		}
	}

	idataRawSize := alignUp(len(idataData), peFileAlign)
	idataVirtSize := len(idataData)
	idataVirtAligned := alignUp(idataVirtSize, peSectionAlign)

	relocRVA := idataRVA + idataVirtAligned
	relocRawOffset := idataRawOffset + idataRawSize
	relocData := buildRelocSection()
	relocRawSize := alignUp(len(relocData), peFileAlign)
	relocVirtSize := len(relocData)

	sizeOfImage := alignUp(relocRVA+relocVirtSize, peSectionAlign)

	entryRVA := uint32(textRVA) + img.EntryOffset

	sizeOfCode := uint32(textRawSize)
	sizeOfInitData := uint32(rdataRawSize + idataRawSize + relocRawSize)

	var buf bytes.Buffer
	dosStub := make([]byte, peDosStubSize)
	dosStub[0] = 'M'
	dosStub[1] = 'Z'
	binary.LittleEndian.PutUint32(dosStub[0x3c:], peDosStubSize)
	if _, err := buf.Write(dosStub); err != nil {
		return err
	}

	if _, err := buf.Write([]byte{'P', 'E', 0, 0}); err != nil {
		return err
	}
	if err := writePEU16(&buf, 0x8664); err != nil {
		return err
	}
	if err := writePEU16(&buf, 4); err != nil {
		return err
	}
	if err := writePEU32(&buf, 0); err != nil {
		return err
	}
	if err := writePEU32(&buf, 0); err != nil {
		return err
	}
	if err := writePEU32(&buf, 0); err != nil {
		return err
	}
	if err := writePEU16(&buf, peOptionalHdrSize); err != nil {
		return err
	}
	if err := writePEU16(&buf, 0x0022); err != nil {
		return err
	}

	if err := writePEU16(&buf, 0x20b); err != nil {
		return err
	}
	if err := writePEU8(&buf, 0); err != nil {
		return err
	}
	if err := writePEU8(&buf, 0); err != nil {
		return err
	}
	if err := writePEU32(&buf, sizeOfCode); err != nil {
		return err
	}
	if err := writePEU32(&buf, sizeOfInitData); err != nil {
		return err
	}
	if err := writePEU32(&buf, 0); err != nil {
		return err
	}
	if err := writePEU32(&buf, entryRVA); err != nil {
		return err
	}
	if err := writePEU32(&buf, uint32(textRVA)); err != nil {
		return err
	}
	if err := writePEU64(&buf, peImageBase); err != nil {
		return err
	}
	if err := writePEU32(&buf, peSectionAlign); err != nil {
		return err
	}
	if err := writePEU32(&buf, peFileAlign); err != nil {
		return err
	}
	if err := writePEU16(&buf, 4); err != nil {
		return err
	}
	if err := writePEU16(&buf, 0); err != nil {
		return err
	}
	if err := writePEU16(&buf, 0); err != nil {
		return err
	}
	if err := writePEU16(&buf, 0); err != nil {
		return err
	}
	if err := writePEU16(&buf, 4); err != nil {
		return err
	}
	if err := writePEU16(&buf, 0); err != nil {
		return err
	}
	if err := writePEU32(&buf, 0); err != nil {
		return err
	}
	if err := writePEU32(&buf, uint32(sizeOfImage)); err != nil {
		return err
	}
	if err := writePEU32(&buf, uint32(headersSize)); err != nil {
		return err
	}
	if err := writePEU32(&buf, 0); err != nil {
		return err
	}
	if err := writePEU16(&buf, 3); err != nil {
		return err
	}
	if err := writePEU16(&buf, 0x0140); err != nil {
		return err
	}
	if err := writePEU64(&buf, 1<<20); err != nil {
		return err
	}
	if err := writePEU64(&buf, 0x1000); err != nil {
		return err
	}
	if err := writePEU64(&buf, 1<<20); err != nil {
		return err
	}
	if err := writePEU64(&buf, 0x1000); err != nil {
		return err
	}
	if err := writePEU32(&buf, 0); err != nil {
		return err
	}
	if err := writePEU32(&buf, 16); err != nil {
		return err
	}

	for i := 0; i < 16; i++ {
		rva := uint32(0)
		size := uint32(0)
		switch i {
		case 1:
			rva = uint32(idataRVA)
			size = importDirSize
		case 5:
			rva = uint32(relocRVA)
			size = uint32(relocVirtSize)
		case 12:
			rva = iatRVA
			size = iatSize
		}
		if err := writePEU32(&buf, rva); err != nil {
			return err
		}
		if err := writePEU32(&buf, size); err != nil {
			return err
		}
	}

	if err := writeSectionHeader(&buf, ".text", uint32(textVirtSize), uint32(textRVA), uint32(textRawSize), uint32(textRawOffset), 0x60000020); err != nil {
		return err
	}
	if err := writeSectionHeader(&buf, ".rdata", uint32(rdataVirtSize), uint32(rdataRVA), uint32(rdataRawSize), uint32(rdataRawOffset), 0x40000040); err != nil {
		return err
	}
	if err := writeSectionHeader(&buf, ".idata", uint32(idataVirtSize), uint32(idataRVA), uint32(idataRawSize), uint32(idataRawOffset), 0xC0000040); err != nil {
		return err
	}
	if err := writeSectionHeader(&buf, ".reloc", uint32(relocVirtSize), uint32(relocRVA), uint32(relocRawSize), uint32(relocRawOffset), 0x42000040); err != nil {
		return err
	}

	if buf.Len() > headersSize {
		return fmt.Errorf("unexpected header size")
	}
	if err := writePadding(&buf, headersSize-buf.Len()); err != nil {
		return err
	}
	if _, err := buf.Write(textData); err != nil {
		return err
	}
	if err := writePadding(&buf, textRawSize-len(textData)); err != nil {
		return err
	}
	if _, err := buf.Write(rdataData); err != nil {
		return err
	}
	if err := writePadding(&buf, rdataRawSize-len(rdataData)); err != nil {
		return err
	}
	if _, err := buf.Write(idataData); err != nil {
		return err
	}
	if err := writePadding(&buf, idataRawSize-len(idataData)); err != nil {
		return err
	}
	if _, err := buf.Write(relocData); err != nil {
		return err
	}
	if err := writePadding(&buf, relocRawSize-len(relocData)); err != nil {
		return err
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o755); err != nil {
		return err
	}
	return nil
}

func BuildImportSection(idataRVA uint32, imports []string) ([]byte, map[string]uint32, uint32, uint32, uint32, error) {
	type dllGroup struct {
		name    string
		syms    []string
		iltOff  int
		iatOff  int
		nameOff int
	}

	dllSyms := make(map[string]map[string]struct{})
	for _, full := range imports {
		dll, sym, err := parseImportName(full)
		if err != nil {
			return nil, nil, 0, 0, 0, err
		}
		set := dllSyms[dll]
		if set == nil {
			set = make(map[string]struct{})
			dllSyms[dll] = set
		}
		set[sym] = struct{}{}
	}
	dllNames := make([]string, 0, len(dllSyms))
	for dll := range dllSyms {
		dllNames = append(dllNames, dll)
	}
	sort.Strings(dllNames)
	if len(dllNames) == 0 {
		return nil, nil, 0, 0, 0, fmt.Errorf("no imports")
	}

	groups := make([]dllGroup, 0, len(dllNames))
	for _, dll := range dllNames {
		syms := make([]string, 0, len(dllSyms[dll]))
		for sym := range dllSyms[dll] {
			syms = append(syms, sym)
		}
		sort.Strings(syms)
		groups = append(groups, dllGroup{name: dll, syms: syms})
	}

	descSize := (len(groups) + 1) * 20
	cursor := descSize
	for i := range groups {
		iltSize := (len(groups[i].syms) + 1) * 8
		iatSize := (len(groups[i].syms) + 1) * 8
		groups[i].iltOff = cursor
		cursor += iltSize
		groups[i].iatOff = cursor
		cursor += iatSize
	}

	hintStart := cursor
	hintEnd := hintStart
	for _, group := range groups {
		for _, sym := range group.syms {
			hintEnd += 2 + len(sym) + 1
			if hintEnd%2 != 0 {
				hintEnd++
			}
		}
	}
	cursor = hintEnd
	for i := range groups {
		groups[i].nameOff = cursor
		cursor += len(groups[i].name) + 1
	}
	totalSize := cursor

	data := make([]byte, totalSize)
	for i, group := range groups {
		descOff := i * 20
		writeU32At(data, descOff, idataRVA+uint32(group.iltOff))
		writeU32At(data, descOff+12, idataRVA+uint32(group.nameOff))
		writeU32At(data, descOff+16, idataRVA+uint32(group.iatOff))
	}

	iatRVAs := make(map[string]uint32)
	cursor = hintStart
	for _, group := range groups {
		for idx, sym := range group.syms {
			hintRVA := idataRVA + uint32(cursor)
			writeU64At(data, group.iltOff+idx*8, uint64(hintRVA))
			writeU64At(data, group.iatOff+idx*8, uint64(hintRVA))
			iatRVAs[importKey(group.name, sym)] = idataRVA + uint32(group.iatOff+idx*8)

			binary.LittleEndian.PutUint16(data[cursor:], 0)
			copy(data[cursor+2:], []byte(sym))
			data[cursor+2+len(sym)] = 0
			cursor += 2 + len(sym) + 1
			if cursor%2 != 0 {
				cursor++
			}
		}
	}

	for _, group := range groups {
		copy(data[group.nameOff:], []byte(group.name))
		data[group.nameOff+len(group.name)] = 0
	}

	iatStart := groups[0].iatOff
	iatEnd := groups[len(groups)-1].iatOff + (len(groups[len(groups)-1].syms)+1)*8

	return data, iatRVAs, uint32(descSize), idataRVA + uint32(iatStart), uint32(iatEnd - iatStart), nil
}

func parseImportName(full string) (string, string, error) {
	parts := strings.SplitN(full, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid import name '%s'", full)
	}
	dll := strings.ToUpper(parts[0]) + ".dll"
	return dll, parts[1], nil
}

func normalizeImportKey(full string) (string, error) {
	dll, sym, err := parseImportName(full)
	if err != nil {
		return "", err
	}
	return importKey(dll, sym), nil
}

func importKey(dll, sym string) string {
	base := strings.ToLower(strings.TrimSuffix(dll, ".dll"))
	return base + "." + sym
}

func buildRelocSection() []byte {
	data := make([]byte, 12)
	writeU32At(data, 0, 0)
	writeU32At(data, 4, 12)
	return data
}

func patchRipDisp32(code []byte, at int, srcRVA uint32, targetRVA uint32) error {
	if !validDisp32PatchOffset(code, at) {
		return fmt.Errorf("rel32 patch offset out of range")
	}
	next := srcRVA + uint32(at+4)
	disp := int64(targetRVA) - int64(next)
	if disp < -2147483648 || disp > 2147483647 {
		return fmt.Errorf("rel32 target out of range")
	}
	binary.LittleEndian.PutUint32(code[at:at+4], uint32(int32(disp)))
	return nil
}

func validDisp32PatchOffset(code []byte, at int) bool {
	return at >= 0 && at <= len(code)-4
}

func alignUp(v int, align int) int {
	if align <= 0 {
		return v
	}
	rem := v % align
	if rem == 0 {
		return v
	}
	return v + (align - rem)
}

func writePEU8(w *bytes.Buffer, v uint8) error {
	return binary.Write(w, binary.LittleEndian, v)
}

func writePEU16(w *bytes.Buffer, v uint16) error {
	return binary.Write(w, binary.LittleEndian, v)
}

func writePEU32(w *bytes.Buffer, v uint32) error {
	return binary.Write(w, binary.LittleEndian, v)
}

func writePEU64(w *bytes.Buffer, v uint64) error {
	return binary.Write(w, binary.LittleEndian, v)
}

func writeU32At(buf []byte, off int, v uint32) {
	binary.LittleEndian.PutUint32(buf[off:off+4], v)
}

func writeU64At(buf []byte, off int, v uint64) {
	binary.LittleEndian.PutUint64(buf[off:off+8], v)
}

func writeSectionHeader(w *bytes.Buffer, name string, virtSize, virtAddr, rawSize, rawOffset, chars uint32) error {
	var nameBuf [8]byte
	copy(nameBuf[:], name)
	if _, err := w.Write(nameBuf[:]); err != nil {
		return err
	}
	if err := writePEU32(w, virtSize); err != nil {
		return err
	}
	if err := writePEU32(w, virtAddr); err != nil {
		return err
	}
	if err := writePEU32(w, rawSize); err != nil {
		return err
	}
	if err := writePEU32(w, rawOffset); err != nil {
		return err
	}
	if err := writePEU32(w, 0); err != nil {
		return err
	}
	if err := writePEU32(w, 0); err != nil {
		return err
	}
	if err := writePEU16(w, 0); err != nil {
		return err
	}
	if err := writePEU16(w, 0); err != nil {
		return err
	}
	if err := writePEU32(w, chars); err != nil {
		return err
	}
	return nil
}

func writePadding(w *bytes.Buffer, n int) error {
	if n <= 0 {
		return nil
	}
	_, err := w.Write(make([]byte, n))
	return err
}
