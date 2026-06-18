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

type PatchKind uint8

const (
	PatchCallRel32 PatchKind = iota
	PatchFuncAddrRel32
)

type CallPatch struct {
	At   int
	Name string
	Kind PatchKind
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

func BuildObject(
	funcs []ir.IRFunc,
	emit EmitFunc,
	opt x64.CodegenOptions,
	options Options,
) (*tobj.Object, error) {
	return BuildObjectWithDataPrefix(funcs, nil, emit, opt, options)
}

func BuildObjectWithDataPrefix(
	funcs []ir.IRFunc,
	dataPrefix [][]byte,
	emit EmitFunc,
	opt x64.CodegenOptions,
	options Options,
) (*tobj.Object, error) {
	if len(funcs) == 0 {
		return nil, fmt.Errorf("missing IR functions")
	}
	if emit == nil {
		return nil, fmt.Errorf("missing emit function")
	}
	functionSigs := make(map[string]ir.IRFunc, len(funcs))
	for _, fn := range funcs {
		if fn.Name == "" {
			return nil, fmt.Errorf("function name is empty")
		}
		if fn.ParamSlots < 0 || fn.LocalSlots < 0 || fn.LocalSlots < fn.ParamSlots ||
			fn.ReturnSlots < 0 {
			return nil, fmt.Errorf("function '%s' has invalid slots", fn.Name)
		}
		if _, exists := functionSigs[fn.Name]; exists {
			return nil, fmt.Errorf("duplicate function '%s'", fn.Name)
		}
		functionSigs[fn.Name] = fn
	}

	e := &x64.Emitter{}
	var leaPatches []LeaPatch
	dataBlobs := append([][]byte(nil), dataPrefix...)
	funcOffsets := make(map[string]int)
	symbolOffsets := make(map[string]int)
	symbolSigs := make(map[string]ir.IRFunc)
	var callPatches []CallPatch
	var importPatches []ImportPatch

	var importPatchesPtr *[]ImportPatch
	if options.CollectImports {
		importPatchesPtr = &importPatches
	}

	for _, fn := range funcs {
		if fn.Name == "" {
			return nil, fmt.Errorf("function name is empty")
		}
		if fn.ParamSlots < 0 || fn.LocalSlots < 0 || fn.LocalSlots < fn.ParamSlots ||
			fn.ReturnSlots < 0 {
			return nil, fmt.Errorf("function '%s' has invalid slots", fn.Name)
		}
		if _, exists := funcOffsets[fn.Name]; exists {
			return nil, fmt.Errorf("duplicate function '%s'", fn.Name)
		}
		if err := validateObjectLocalCallSignatures(fn, functionSigs); err != nil {
			return nil, err
		}
		funcOffsets[fn.Name] = len(e.Buf)
		symbolOffsets[fn.Name] = len(e.Buf)
		symbolSigs[fn.Name] = fn
		if fn.ExportName != "" {
			if _, exists := symbolOffsets[fn.ExportName]; exists {
				return nil, fmt.Errorf("duplicate exported symbol '%s'", fn.ExportName)
			}
			symbolOffsets[fn.ExportName] = len(e.Buf)
			symbolSigs[fn.ExportName] = fn
		}
		if err := emit(e, fn, &dataBlobs, &leaPatches, &callPatches, importPatchesPtr, opt); err != nil {
			return nil, err
		}
	}

	code := e.Buf
	var relocs []tobj.Reloc
	validatePatchOffset := func(kind string, at int) error {
		if at < 0 || at > len(code)-4 {
			return fmt.Errorf("invalid patch offset %d for %s", at, kind)
		}
		return nil
	}

	for _, patch := range callPatches {
		patchLabel := "call"
		relocKind := tobj.RelocCallRel32
		switch patch.Kind {
		case PatchCallRel32:
		case PatchFuncAddrRel32:
			patchLabel = "function address"
			relocKind = tobj.RelocFuncAddrDisp32
		default:
			return nil, fmt.Errorf(
				"unsupported symbol patch kind %d for %q",
				patch.Kind,
				patch.Name,
			)
		}
		if err := validatePatchOffset(patchLabel, patch.At); err != nil {
			return nil, err
		}
		if patch.Name == "" {
			return nil, fmt.Errorf("%s patch name is empty", patchLabel)
		}
		if target, ok := funcOffsets[patch.Name]; ok {
			if err := x64.PatchRel32(code, patch.At, target); err != nil {
				return nil, err
			}
			continue
		}
		relocs = append(
			relocs,
			tobj.Reloc{Kind: relocKind, At: uint32(patch.At), Name: patch.Name, Addend: 0},
		)
	}
	for _, patch := range importPatches {
		if err := validatePatchOffset("import", patch.At); err != nil {
			return nil, err
		}
		if patch.Name == "" {
			return nil, fmt.Errorf("import patch name is empty")
		}
		relocs = append(
			relocs,
			tobj.Reloc{
				Kind:   tobj.RelocIATDisp32,
				At:     uint32(patch.At),
				Name:   patch.Name,
				Addend: 0,
			},
		)
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
		if err := validatePatchOffset("data", patch.At); err != nil {
			return nil, err
		}
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
		fn := symbolSigs[name]
		symbols = append(symbols, tobj.Symbol{
			Name:         name,
			Offset:       uint32(symbolOffsets[name]),
			HasSignature: true,
			ParamSlots:   fn.ParamSlots,
			ReturnSlots:  fn.ReturnSlots,
		})
	}

	return &tobj.Object{Code: code, Data: data, Symbols: symbols, Relocs: relocs}, nil
}

func validateObjectLocalCallSignatures(fn ir.IRFunc, functionSigs map[string]ir.IRFunc) error {
	for _, instr := range fn.Instrs {
		if instr.Kind != ir.IRCall {
			continue
		}
		target, ok := functionSigs[instr.Name]
		if !ok {
			continue
		}
		if instr.ArgSlots != target.ParamSlots || instr.RetSlots != target.ReturnSlots {
			return fmt.Errorf(
				"function '%s' call %q ABI mismatch args=%d rets=%d want args=%d rets=%d",
				fn.Name,
				instr.Name,
				instr.ArgSlots,
				instr.RetSlots,
				target.ParamSlots,
				target.ReturnSlots,
			)
		}
	}
	return nil
}
