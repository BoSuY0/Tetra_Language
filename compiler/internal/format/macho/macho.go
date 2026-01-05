package macho

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

const (
	machoMagic64          = 0xFEEDFACF
	machoCpuTypeX86_64    = 0x01000007
	machoCpuSubtypeX86_64 = 3
	machoFiletypeExecute  = 2
	machoCmdSegment64     = 0x19
	machoCmdMain          = 0x80000028
	machoPageSize         = 0x1000
	machoBaseAddr         = 0x100000000
)

const (
	MachOMagic64          = machoMagic64
	MachOCpuTypeX86_64    = machoCpuTypeX86_64
	MachOCpuSubtypeX86_64 = machoCpuSubtypeX86_64
	MachOFiletypeExecute  = machoFiletypeExecute
	MachOCmdSegment64     = machoCmdSegment64
	MachOCmdMain          = machoCmdMain
)

type DataReloc struct {
	At        int
	TargetOff uint32
}

type MachOImage struct {
	Text         []byte
	CString      []byte
	EntryTextOff uint32
	DataRelocs   []DataReloc
}

func WriteMachO64MacOSX64(path string, img *MachOImage) error {
	if img == nil {
		return fmt.Errorf("missing Mach-O image")
	}
	if img.EntryTextOff > uint32(len(img.Text)) {
		return fmt.Errorf("entry offset out of range")
	}

	textData := append([]byte(nil), img.Text...)
	cstringData := append([]byte(nil), img.CString...)

	segmentCmdSize := 72
	sectionSize := 80
	textCmdSize := segmentCmdSize + sectionSize
	dataCmdSize := segmentCmdSize + sectionSize
	mainCmdSize := 24
	sizeofcmds := textCmdSize + dataCmdSize + mainCmdSize

	headerSize := 32 + sizeofcmds
	textFileOff := alignUp(headerSize, machoPageSize)
	textFileSize := alignUp(len(textData), machoPageSize)
	cstringFileOff := textFileOff + textFileSize
	cstringFileSize := alignUp(len(cstringData), machoPageSize)

	textVMAddr := machoBaseAddr + uint64(textFileOff)
	cstringVMAddr := machoBaseAddr + uint64(cstringFileOff)

	for _, reloc := range img.DataRelocs {
		if reloc.At < 0 || reloc.At+4 > len(textData) {
			return fmt.Errorf("rdata relocation out of range")
		}
		target := cstringVMAddr + uint64(reloc.TargetOff)
		if err := patchRipDisp32From64(textData, reloc.At, textVMAddr, target); err != nil {
			return err
		}
	}

	entryOff := uint64(textFileOff) + uint64(img.EntryTextOff)

	var buf bytes.Buffer
	if err := writeMachOU32(&buf, machoMagic64); err != nil {
		return err
	}
	if err := writeMachOU32(&buf, machoCpuTypeX86_64); err != nil {
		return err
	}
	if err := writeMachOU32(&buf, machoCpuSubtypeX86_64); err != nil {
		return err
	}
	if err := writeMachOU32(&buf, machoFiletypeExecute); err != nil {
		return err
	}
	if err := writeMachOU32(&buf, 3); err != nil {
		return err
	}
	if err := writeMachOU32(&buf, uint32(sizeofcmds)); err != nil {
		return err
	}
	if err := writeMachOU32(&buf, 0); err != nil {
		return err
	}
	if err := writeMachOU32(&buf, 0); err != nil {
		return err
	}

	if err := writeSegment64(&buf, "__TEXT", textCmdSize, textVMAddr, uint64(textFileSize), uint64(textFileOff), uint64(textFileSize), 0x5, 0x5, 1); err != nil {
		return err
	}
	if err := writeSection64(&buf, "__text", "__TEXT", textVMAddr, uint64(len(textData)), uint32(textFileOff), 4, 0x80000400); err != nil {
		return err
	}

	if err := writeSegment64(&buf, "__DATA", dataCmdSize, cstringVMAddr, uint64(cstringFileSize), uint64(cstringFileOff), uint64(cstringFileSize), 0x1, 0x1, 1); err != nil {
		return err
	}
	if err := writeSection64(&buf, "__cstring", "__DATA", cstringVMAddr, uint64(len(cstringData)), uint32(cstringFileOff), 0, 0x2); err != nil {
		return err
	}

	if err := writeMachOU32(&buf, machoCmdMain); err != nil {
		return err
	}
	if err := writeMachOU32(&buf, uint32(mainCmdSize)); err != nil {
		return err
	}
	if err := writeMachOU64(&buf, entryOff); err != nil {
		return err
	}
	if err := writeMachOU64(&buf, 0); err != nil {
		return err
	}

	if buf.Len() > textFileOff {
		return fmt.Errorf("unexpected header size")
	}
	if err := writePadding(&buf, textFileOff-buf.Len()); err != nil {
		return err
	}
	if _, err := buf.Write(textData); err != nil {
		return err
	}
	if err := writePadding(&buf, textFileSize-len(textData)); err != nil {
		return err
	}
	if _, err := buf.Write(cstringData); err != nil {
		return err
	}
	if err := writePadding(&buf, cstringFileSize-len(cstringData)); err != nil {
		return err
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o755); err != nil {
		return err
	}
	return nil
}

func writeSegment64(w *bytes.Buffer, name string, cmdsize int, vmaddr, vmsize, fileoff, filesize uint64, maxprot, initprot uint32, nsects uint32) error {
	if err := writeMachOU32(w, machoCmdSegment64); err != nil {
		return err
	}
	if err := writeMachOU32(w, uint32(cmdsize)); err != nil {
		return err
	}
	if err := writeMachOName(w, name); err != nil {
		return err
	}
	if err := writeMachOU64(w, vmaddr); err != nil {
		return err
	}
	if err := writeMachOU64(w, vmsize); err != nil {
		return err
	}
	if err := writeMachOU64(w, fileoff); err != nil {
		return err
	}
	if err := writeMachOU64(w, filesize); err != nil {
		return err
	}
	if err := writeMachOU32(w, maxprot); err != nil {
		return err
	}
	if err := writeMachOU32(w, initprot); err != nil {
		return err
	}
	if err := writeMachOU32(w, nsects); err != nil {
		return err
	}
	if err := writeMachOU32(w, 0); err != nil {
		return err
	}
	return nil
}

func writeSection64(w *bytes.Buffer, sectName, segName string, addr, size uint64, offset uint32, align uint32, flags uint32) error {
	if err := writeMachOName(w, sectName); err != nil {
		return err
	}
	if err := writeMachOName(w, segName); err != nil {
		return err
	}
	if err := writeMachOU64(w, addr); err != nil {
		return err
	}
	if err := writeMachOU64(w, size); err != nil {
		return err
	}
	if err := writeMachOU32(w, offset); err != nil {
		return err
	}
	if err := writeMachOU32(w, align); err != nil {
		return err
	}
	if err := writeMachOU32(w, 0); err != nil {
		return err
	}
	if err := writeMachOU32(w, 0); err != nil {
		return err
	}
	if err := writeMachOU32(w, flags); err != nil {
		return err
	}
	if err := writeMachOU32(w, 0); err != nil {
		return err
	}
	if err := writeMachOU32(w, 0); err != nil {
		return err
	}
	if err := writeMachOU32(w, 0); err != nil {
		return err
	}
	return nil
}

func writeMachOName(w *bytes.Buffer, name string) error {
	var buf [16]byte
	copy(buf[:], name)
	_, err := w.Write(buf[:])
	return err
}

func writeMachOU32(w *bytes.Buffer, v uint32) error {
	return binary.Write(w, binary.LittleEndian, v)
}

func writeMachOU64(w *bytes.Buffer, v uint64) error {
	return binary.Write(w, binary.LittleEndian, v)
}

func alignUp(size, align int) int {
	if align <= 0 {
		return size
	}
	rem := size % align
	if rem == 0 {
		return size
	}
	return size + (align - rem)
}

func writePadding(w *bytes.Buffer, size int) error {
	if size < 0 {
		return fmt.Errorf("negative padding size")
	}
	if size == 0 {
		return nil
	}
	_, err := w.Write(make([]byte, size))
	return err
}

func patchRipDisp32From64(code []byte, at int, srcVA, targetVA uint64) error {
	next := srcVA + uint64(at+4)
	disp := int64(targetVA) - int64(next)
	if disp < -2147483648 || disp > 2147483647 {
		return fmt.Errorf("rel32 target out of range")
	}
	binary.LittleEndian.PutUint32(code[at:at+4], uint32(int32(disp)))
	return nil
}
