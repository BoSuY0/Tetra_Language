package x64obj

import (
	"fmt"
	"sort"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
)

type Options struct {
	CollectImports bool
}

type LeaPatch struct {
	At        int
	DataIndex int
}

type CallPatch struct {
	At   int
	Name string
}

type ImportPatch struct {
	At   int
	Name string
}

type EmitFunc func(
	e *x64.Emitter,
	fn ir.IRFunc,
	dataBlobs *[][]byte,
	leaPatches *[]LeaPatch,
	callPatches *[]CallPatch,
	importPatches *[]ImportPatch,
	opt x64.CodegenOptions,
) error

func BuildObject(funcs []ir.IRFunc, emit EmitFunc, opt x64.CodegenOptions, options Options) (*tobj.Object, error) {
	return BuildObjectWithDataPrefix(funcs, nil, emit, opt, options)
}

func BuildObjectWithDataPrefix(funcs []ir.IRFunc, dataPrefix [][]byte, emit EmitFunc, opt x64.CodegenOptions, options Options) (*tobj.Object, error) {
	if len(funcs) == 0 {
		return nil, fmt.Errorf("missing IR functions")
	}
	if emit == nil {
		return nil, fmt.Errorf("missing emit function")
	}

	e := &x64.Emitter{}
	var leaPatches []LeaPatch
	dataBlobs := append([][]byte(nil), dataPrefix...)
	funcOffsets := make(map[string]int)
	symbolOffsets := make(map[string]int)
	var callPatches []CallPatch
	var importPatches []ImportPatch

	var importPatchesPtr *[]ImportPatch
	if options.CollectImports {
		importPatchesPtr = &importPatches
	}

	for _, fn := range funcs {
		if _, exists := funcOffsets[fn.Name]; exists {
			return nil, fmt.Errorf("duplicate function '%s'", fn.Name)
		}
		funcOffsets[fn.Name] = len(e.Buf)
		symbolOffsets[fn.Name] = len(e.Buf)
		if fn.ExportName != "" {
			if _, exists := symbolOffsets[fn.ExportName]; exists {
				return nil, fmt.Errorf("duplicate exported symbol '%s'", fn.ExportName)
			}
			symbolOffsets[fn.ExportName] = len(e.Buf)
		}
		if err := emit(e, fn, &dataBlobs, &leaPatches, &callPatches, importPatchesPtr, opt); err != nil {
			return nil, err
		}
	}

	code := e.Buf
	var relocs []tobj.Reloc

	for _, patch := range callPatches {
		if target, ok := funcOffsets[patch.Name]; ok {
			if err := x64.PatchRel32(code, patch.At, target); err != nil {
				return nil, err
			}
			continue
		}
		relocs = append(relocs, tobj.Reloc{Kind: tobj.RelocCallRel32, At: uint32(patch.At), Name: patch.Name, Addend: 0})
	}
	for _, patch := range importPatches {
		relocs = append(relocs, tobj.Reloc{Kind: tobj.RelocIATDisp32, At: uint32(patch.At), Name: patch.Name, Addend: 0})
	}

	dataOffsets := make([]int, len(dataBlobs))
	dataSize := 0
	for i, blob := range dataBlobs {
		dataOffsets[i] = dataSize
		dataSize += len(blob)
	}
	data := make([]byte, 0, dataSize)
	for _, blob := range dataBlobs {
		data = append(data, blob...)
	}

	for _, patch := range leaPatches {
		if patch.DataIndex < 0 || patch.DataIndex >= len(dataOffsets) {
			return nil, fmt.Errorf("invalid data patch index %d", patch.DataIndex)
		}
		addend := dataOffsets[patch.DataIndex]
		relocs = append(relocs, tobj.Reloc{
			Kind:   tobj.RelocDataDisp32,
			At:     uint32(patch.At),
			Name:   "",
			Addend: uint32(addend),
		})
	}

	names := make([]string, 0, len(symbolOffsets))
	for name := range symbolOffsets {
		names = append(names, name)
	}
	sort.Strings(names)
	symbols := make([]tobj.Symbol, 0, len(names))
	for _, name := range names {
		symbols = append(symbols, tobj.Symbol{Name: name, Offset: uint32(symbolOffsets[name])})
	}

	return &tobj.Object{Code: code, Data: data, Symbols: symbols, Relocs: relocs}, nil
}
