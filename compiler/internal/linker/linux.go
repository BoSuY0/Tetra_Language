package linker

import (
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/format/elf"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/linker/linkcore"
)

func LinkLinuxX64(objects []*tobj.Object, mainName string) (*elf.Image, error) {
	stub, stubCallAt := emitEntryStubSysVUnixX64(60)
	res, err := linkcore.LinkX64Objects(objects, mainName, stub, stubCallAt, 0)
	if err != nil {
		return nil, err
	}

	layout := elf.LinuxX64Layout(len(res.Text), len(res.Data))
	dataStartRel := layout.DataOffset - layout.CodeOffset

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
