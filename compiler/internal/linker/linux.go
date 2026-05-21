package linker

import (
	"encoding/binary"
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/format/elf"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/linker/linkcore"
)

func LinkLinuxX64(objects []*tobj.Object, mainName string) (*elf.Image, error) {
	return linkLinuxSysV(objects, mainName, "linux-x64", 60, func(codeSize, dataSize int) int {
		layout := elf.LinuxX64Layout(codeSize, dataSize)
		return layout.DataOffset - layout.CodeOffset
	})
}

func LinkLinuxX32(objects []*tobj.Object, mainName string) (*elf.Image, error) {
	const x32SyscallBit = 0x40000000
	return linkLinuxSysV(objects, mainName, "linux-x32", x32SyscallBit+60, func(codeSize, dataSize int) int {
		layout := elf.LinuxX32Layout(codeSize, dataSize)
		return layout.DataOffset - layout.CodeOffset
	})
}

func LinkLinuxX86(objects []*tobj.Object, mainName string) (*elf.Image, error) {
	for _, obj := range objects {
		if obj == nil {
			return nil, fmt.Errorf("nil object")
		}
		if obj.Target != "linux-x86" {
			return nil, fmt.Errorf("linker target mismatch: linux-x86 expects 'linux-x86' object, got '%s' (module '%s')", obj.Target, obj.Module)
		}
	}

	stub, stubCallAt := emitEntryStubSysVLinuxX86()
	res, err := linkcore.LinkX64Objects(objects, mainName, stub, stubCallAt, 0)
	if err != nil {
		return nil, err
	}
	dataStartRel := func(codeSize, dataSize int) int {
		layout := elf.LinuxX86Layout(codeSize, dataSize)
		return layout.DataOffset - layout.CodeOffset
	}(len(res.Text), len(res.Data))
	for _, reloc := range res.DataRelocs {
		if reloc.At < 0 || reloc.At+4 > len(res.Text) {
			return nil, fmt.Errorf("data relocation out of range")
		}
		if reloc.TargetOff < 0 || reloc.TargetOff >= len(res.Data) {
			return nil, fmt.Errorf("data relocation target out of range")
		}
		if err := x64.PatchRel32(res.Text, reloc.At, dataStartRel+reloc.TargetOff); err != nil {
			return nil, err
		}
	}
	layout := elf.LinuxX86Layout(len(res.Text), len(res.Data))
	for _, reloc := range res.FuncAbsRelocs {
		if reloc.At < 0 || reloc.At+4 > len(res.Text) {
			return nil, fmt.Errorf("absolute function address relocation out of range")
		}
		if reloc.TargetOff < 0 || reloc.TargetOff >= len(res.Text) {
			return nil, fmt.Errorf("absolute function address relocation target out of range")
		}
		addr := uint64(elf.LinuxX86BaseVaddr + layout.CodeOffset + reloc.TargetOff)
		if addr > uint64(^uint32(0)) {
			return nil, fmt.Errorf("absolute function address relocation target exceeds 32-bit address range")
		}
		binary.LittleEndian.PutUint32(res.Text[reloc.At:reloc.At+4], uint32(addr))
	}
	for _, reloc := range res.DataAbsRelocs {
		if reloc.At < 0 || reloc.At+4 > len(res.Text) {
			return nil, fmt.Errorf("absolute data relocation out of range")
		}
		if reloc.TargetOff < 0 || reloc.TargetOff >= len(res.Data) {
			return nil, fmt.Errorf("absolute data relocation target out of range")
		}
		addr := uint64(elf.LinuxX86BaseVaddr + layout.DataOffset + reloc.TargetOff)
		if addr > uint64(^uint32(0)) {
			return nil, fmt.Errorf("absolute data relocation target exceeds 32-bit address range")
		}
		binary.LittleEndian.PutUint32(res.Text[reloc.At:reloc.At+4], uint32(addr))
	}
	return &elf.Image{Code: res.Text, Data: res.Data, EntryOffset: uint64(res.EntryOffset)}, nil
}

func linkLinuxSysV(objects []*tobj.Object, mainName string, expectedTarget string, sysExit uint32, dataStartRelFor func(codeSize, dataSize int) int) (*elf.Image, error) {
	for _, obj := range objects {
		if obj == nil {
			return nil, fmt.Errorf("nil object")
		}
		if obj.Target != expectedTarget {
			return nil, fmt.Errorf("linker target mismatch: %s expects '%s' object, got '%s' (module '%s')", expectedTarget, expectedTarget, obj.Target, obj.Module)
		}
	}

	stub, stubCallAt := emitEntryStubSysVUnixX64(sysExit)
	res, err := linkcore.LinkX64Objects(objects, mainName, stub, stubCallAt, 0)
	if err != nil {
		return nil, err
	}

	dataStartRel := dataStartRelFor(len(res.Text), len(res.Data))

	if len(res.FuncAbsRelocs) != 0 {
		return nil, fmt.Errorf("absolute function address relocation is not supported for %s", expectedTarget)
	}
	if len(res.DataAbsRelocs) != 0 {
		return nil, fmt.Errorf("absolute data relocation is not supported for %s", expectedTarget)
	}

	for _, reloc := range res.DataRelocs {
		if reloc.At < 0 || reloc.At+4 > len(res.Text) {
			return nil, fmt.Errorf("data relocation out of range")
		}
		if reloc.TargetOff < 0 || reloc.TargetOff >= len(res.Data) {
			return nil, fmt.Errorf("data relocation target out of range")
		}
		if err := x64.PatchRel32(res.Text, reloc.At, dataStartRel+reloc.TargetOff); err != nil {
			return nil, err
		}
	}

	return &elf.Image{Code: res.Text, Data: res.Data, EntryOffset: uint64(res.EntryOffset)}, nil
}
