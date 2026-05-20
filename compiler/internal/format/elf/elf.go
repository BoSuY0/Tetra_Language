package elf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

type Image struct {
	Code        []byte
	Data        []byte
	EntryOffset uint64
}

const LinuxX64BaseVaddr = 0x400000

type LinuxX64LayoutInfo struct {
	CodeOffset int
	DataOffset int
	FileSize   int
}

func LinuxX64Layout(codeSize, dataSize int) LinuxX64LayoutInfo {
	const (
		elfHeaderSize  = 64
		programHdrSize = 56
		programHdrNum  = 2
		pageAlign      = 0x1000
	)

	codeOffset := elfHeaderSize + (programHdrSize * programHdrNum)
	textEnd := codeOffset + codeSize
	dataOffset := alignUp(textEnd, pageAlign)
	fileSize := dataOffset + dataSize
	return LinuxX64LayoutInfo{CodeOffset: codeOffset, DataOffset: dataOffset, FileSize: fileSize}
}

func WriteELF64LinuxX64(path string, img *Image) error {
	if img == nil {
		return fmt.Errorf("missing ELF image")
	}
	if img.EntryOffset > uint64(len(img.Code)) {
		return fmt.Errorf("entry offset out of range")
	}

	layout := LinuxX64Layout(len(img.Code), len(img.Data))
	entry := uint64(LinuxX64BaseVaddr+layout.CodeOffset) + img.EntryOffset

	var buf bytes.Buffer
	eIdent := [16]byte{0x7f, 'E', 'L', 'F', 2, 1, 1, 0}
	if _, err := buf.Write(eIdent[:]); err != nil {
		return err
	}
	if err := writeLE(&buf, uint16(2)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint16(0x3E)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint32(1)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint64(entry)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint64(64)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint64(0)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint32(0)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint16(64)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint16(56)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint16(2)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint16(0)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint16(0)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint16(0)); err != nil {
		return err
	}

	const (
		ptLoad = 1
		pfX    = 1
		pfW    = 2
		pfR    = 4
	)

	if err := writeLE(&buf, uint32(ptLoad)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint32(pfR|pfX)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint64(0)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint64(LinuxX64BaseVaddr)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint64(LinuxX64BaseVaddr)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint64(layout.DataOffset)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint64(layout.DataOffset)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint64(0x1000)); err != nil {
		return err
	}

	if err := writeLE(&buf, uint32(ptLoad)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint32(pfR|pfW)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint64(layout.DataOffset)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint64(LinuxX64BaseVaddr+layout.DataOffset)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint64(LinuxX64BaseVaddr+layout.DataOffset)); err != nil {
		return err
	}
	if err := writeLE(&buf, uint64(len(img.Data))); err != nil {
		return err
	}
	if err := writeLE(&buf, uint64(len(img.Data))); err != nil {
		return err
	}
	if err := writeLE(&buf, uint64(0x1000)); err != nil {
		return err
	}

	if buf.Len() > layout.CodeOffset {
		return fmt.Errorf("unexpected header size")
	}
	pad := make([]byte, layout.CodeOffset-buf.Len())
	if _, err := buf.Write(pad); err != nil {
		return err
	}
	if _, err := buf.Write(img.Code); err != nil {
		return err
	}
	if buf.Len() > layout.DataOffset {
		return fmt.Errorf("unexpected text size")
	}
	dataPad := make([]byte, layout.DataOffset-buf.Len())
	if _, err := buf.Write(dataPad); err != nil {
		return err
	}
	if len(img.Data) > 0 {
		if _, err := buf.Write(img.Data); err != nil {
			return err
		}
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o755); err != nil {
		return err
	}
	return nil
}

func writeLE(w *bytes.Buffer, v interface{}) error {
	return binary.Write(w, binary.LittleEndian, v)
}

func alignUp(v, align int) int {
	if align <= 0 {
		return v
	}
	rem := v % align
	if rem == 0 {
		return v
	}
	return v + (align - rem)
}
