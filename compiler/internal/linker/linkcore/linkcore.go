package linkcore

import (
	"fmt"
	"sort"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/format/tobj"
)

type DataDisp32Reloc struct {
	At        int
	TargetOff int
}

type IATDisp32Reloc struct {
	At   int
	Name string
}

type Result struct {
	Text []byte
	Data []byte

	Symbols     map[string]int
	DataRelocs  []DataDisp32Reloc
	IATRelocs   []IATDisp32Reloc
	EntryOffset int
}

func LinkX64Objects(objects []*tobj.Object, mainName string, entryStub []byte, entryStubCallAt int, entryOffset int) (*Result, error) {
	if mainName == "" {
		return nil, fmt.Errorf("missing main entry name")
	}
	if len(objects) == 0 {
		return nil, fmt.Errorf("no objects to link")
	}
	if entryStubCallAt < 0 || entryStubCallAt+4 > len(entryStub) {
		return nil, fmt.Errorf("entry stub call patch out of range")
	}
	if entryOffset < 0 || entryOffset > len(entryStub) {
		return nil, fmt.Errorf("entry offset out of range")
	}

	objs := make([]*tobj.Object, 0, len(objects))
	var target string
	for _, obj := range objects {
		if obj == nil {
			return nil, fmt.Errorf("nil object")
		}
		if obj.Target == "" {
			return nil, fmt.Errorf("missing object target for module '%s'", obj.Module)
		}
		if target == "" {
			target = obj.Target
		} else if obj.Target != target {
			return nil, fmt.Errorf("mixed object targets: '%s' vs '%s' (module '%s')", target, obj.Target, obj.Module)
		}
		objs = append(objs, obj)
	}
	sort.Slice(objs, func(i, j int) bool {
		return objs[i].Module < objs[j].Module
	})

	textSize := len(entryStub)
	dataSize := 0
	for _, obj := range objs {
		textSize += len(obj.Code)
		dataSize += len(obj.Data)
	}

	text := make([]byte, 0, textSize)
	data := make([]byte, 0, dataSize)
	text = append(text, entryStub...)

	symbols := make(map[string]int)
	symbolSigs := make(map[string]tobj.Symbol)
	objTextBases := make(map[*tobj.Object]int)
	objDataBases := make(map[*tobj.Object]int)
	textOffset := len(entryStub)
	dataOffset := 0
	for _, obj := range objs {
		textBase := textOffset
		dataBase := dataOffset
		objTextBases[obj] = textBase
		objDataBases[obj] = dataBase
		for _, sym := range obj.Symbols {
			if sym.Name == "" {
				return nil, fmt.Errorf("empty symbol name in module '%s'", obj.Module)
			}
			if uint64(sym.Offset) >= uint64(len(obj.Code)) {
				return nil, fmt.Errorf("symbol offset out of range for '%s' in module '%s'", sym.Name, obj.Module)
			}
			if _, exists := symbols[sym.Name]; exists {
				return nil, fmt.Errorf("duplicate symbol '%s'", sym.Name)
			}
			symbols[sym.Name] = textBase + int(sym.Offset)
			if sym.HasSignature {
				symbolSigs[sym.Name] = sym
			}
		}
		text = append(text, obj.Code...)
		data = append(data, obj.Data...)
		textOffset += len(obj.Code)
		dataOffset += len(obj.Data)
	}

	mainOffset, ok := symbols[mainName]
	if !ok {
		return nil, fmt.Errorf("missing entry symbol '%s'", mainName)
	}
	if sig, ok := symbolSigs[mainName]; ok && (sig.ParamSlots != 0 || sig.ReturnSlots != 1) {
		return nil, fmt.Errorf("entry symbol '%s' has incompatible signature params=%d returns=%d", mainName, sig.ParamSlots, sig.ReturnSlots)
	}
	if err := x64.PatchRel32(text, entryStubCallAt, mainOffset); err != nil {
		return nil, err
	}

	var dataRelocs []DataDisp32Reloc
	var iatRelocs []IATDisp32Reloc

	for _, obj := range objs {
		textBase := objTextBases[obj]
		dataBase := objDataBases[obj]
		for _, reloc := range obj.Relocs {
			switch reloc.Kind {
			case tobj.RelocCallRel32:
				if reloc.Name == "" {
					return nil, fmt.Errorf("call relocation with empty symbol name in module '%s'", obj.Module)
				}
				if reloc.Addend != 0 {
					return nil, fmt.Errorf("call relocation addend must be zero in module '%s'", obj.Module)
				}
				target, ok := symbols[reloc.Name]
				if !ok {
					return nil, fmt.Errorf("unresolved symbol '%s'", reloc.Name)
				}
				at := textBase + int(reloc.At)
				if at < 0 || at+4 > len(text) {
					return nil, fmt.Errorf("relocation out of range for '%s'", reloc.Name)
				}
				if err := x64.PatchRel32(text, at, target); err != nil {
					return nil, err
				}
			case tobj.RelocDataDisp32:
				if reloc.Name != "" {
					return nil, fmt.Errorf("data relocation symbol name must be empty in module '%s'", obj.Module)
				}
				if len(obj.Data) == 0 {
					return nil, fmt.Errorf("data relocation in empty data section for '%s'", obj.Module)
				}
				if reloc.Addend >= uint32(len(obj.Data)) {
					return nil, fmt.Errorf("data relocation out of range in '%s'", obj.Module)
				}
				at := textBase + int(reloc.At)
				if at < 0 || at+4 > len(text) {
					return nil, fmt.Errorf("data relocation out of range for '%s'", obj.Module)
				}
				dataRelocs = append(dataRelocs, DataDisp32Reloc{
					At:        at,
					TargetOff: dataBase + int(reloc.Addend),
				})
			case tobj.RelocIATDisp32:
				if reloc.Name == "" {
					return nil, fmt.Errorf("IAT relocation with empty symbol name in module '%s'", obj.Module)
				}
				if reloc.Addend != 0 {
					return nil, fmt.Errorf("IAT relocation addend must be zero in module '%s'", obj.Module)
				}
				at := textBase + int(reloc.At)
				if at < 0 || at+4 > len(text) {
					return nil, fmt.Errorf("IAT relocation out of range for '%s'", reloc.Name)
				}
				iatRelocs = append(iatRelocs, IATDisp32Reloc{At: at, Name: reloc.Name})
			default:
				return nil, fmt.Errorf("unsupported relocation kind %d", reloc.Kind)
			}
		}
	}

	return &Result{
		Text:        text,
		Data:        data,
		Symbols:     symbols,
		DataRelocs:  dataRelocs,
		IATRelocs:   iatRelocs,
		EntryOffset: entryOffset,
	}, nil
}

func CollectImports(iatRelocs []IATDisp32Reloc, extra []string) []string {
	importSet := make(map[string]struct{}, len(iatRelocs)+len(extra))
	for _, name := range extra {
		importSet[name] = struct{}{}
	}
	for _, reloc := range iatRelocs {
		importSet[reloc.Name] = struct{}{}
	}
	imports := make([]string, 0, len(importSet))
	for name := range importSet {
		imports = append(imports, name)
	}
	sort.Strings(imports)
	return imports
}
