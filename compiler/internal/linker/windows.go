package linker

import (
	"fmt"

	"tetra_language/compiler/internal/format/pe"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/linker/linkcore"
)

const winImportExitProcess = "kernel32.ExitProcess"

func LinkWindowsX64(objects []*tobj.Object, mainName string) (*pe.PEImage, error) {
	const expectedTarget = "windows-x64"
	for _, obj := range objects {
		if obj == nil {
			return nil, fmt.Errorf("nil object")
		}
		if obj.Target != expectedTarget {
			return nil, fmt.Errorf(
				"linker target mismatch: windows-x64 expects '%s' object, got '%s' (module '%s')",
				expectedTarget,
				obj.Target,
				obj.Module,
			)
		}
	}

	stub, stubCallAt, stubExitAt := emitEntryStubWin64X64()
	res, err := linkcore.LinkX64Objects(objects, mainName, stub, stubCallAt, 0)
	if err != nil {
		return nil, err
	}

	iatRelocs := make([]pe.IATReloc, 0, len(res.IATRelocs)+1)
	iatRelocs = append(iatRelocs, pe.IATReloc{At: stubExitAt, Name: winImportExitProcess})
	for _, reloc := range res.IATRelocs {
		iatRelocs = append(iatRelocs, pe.IATReloc{At: reloc.At, Name: reloc.Name})
	}

	rdataRelocs := make([]pe.RDataReloc, 0, len(res.DataRelocs))
	for _, reloc := range res.DataRelocs {
		rdataRelocs = append(rdataRelocs, pe.RDataReloc{
			At:        reloc.At,
			TargetOff: uint32(reloc.TargetOff),
		})
	}

	imports := linkcore.CollectImports(res.IATRelocs, []string{winImportExitProcess})

	return &pe.PEImage{
		Text:        res.Text,
		RData:       res.Data,
		EntryOffset: uint32(res.EntryOffset),
		Imports:     imports,
		IATRelocs:   iatRelocs,
		RDataRelocs: rdataRelocs,
	}, nil
}
