package linker

import (
	"fmt"

	"tetra_language/compiler/internal/format/macho"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/linker/linkcore"
)

const (
	macosSyscallExit = 0x2000001
)

func LinkMacOSX64(objects []*tobj.Object, mainName string) (*macho.MachOImage, error) {
	const expectedTarget = "macos-x64"
	for _, obj := range objects {
		if obj == nil {
			return nil, fmt.Errorf("nil object")
		}
		if obj.Target != expectedTarget {
			return nil, fmt.Errorf("linker target mismatch: macos-x64 expects '%s' object, got '%s' (module '%s')", expectedTarget, obj.Target, obj.Module)
		}
	}

	stub, stubCallAt := emitEntryStubSysVUnixX64(macosSyscallExit)
	res, err := linkcore.LinkX64Objects(objects, mainName, stub, stubCallAt, 0)
	if err != nil {
		return nil, err
	}

	dataRelocs := make([]macho.DataReloc, 0, len(res.DataRelocs))
	for _, reloc := range res.DataRelocs {
		dataRelocs = append(dataRelocs, macho.DataReloc{
			At:        reloc.At,
			TargetOff: uint32(reloc.TargetOff),
		})
	}

	return &macho.MachOImage{
		Text:         res.Text,
		CString:      res.Data,
		EntryTextOff: uint32(res.EntryOffset),
		DataRelocs:   dataRelocs,
	}, nil
}
