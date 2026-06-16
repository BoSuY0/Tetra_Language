package linux_x86

import (
	"fmt"
	"sort"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
)

func buildObject(funcs []ir.IRFunc, dataPrefix [][]byte, opt x64.CodegenOptions) (*tobj.Object, error) {
	if len(funcs) == 0 {
		return nil, fmt.Errorf("missing IR functions")
	}
	functionSigs := make(map[string]ir.IRFunc, len(funcs))
	for _, fn := range funcs {
		if fn.Name == "" {
			return nil, fmt.Errorf("function name is empty")
		}
		if fn.ParamSlots < 0 || fn.LocalSlots < 0 || fn.LocalSlots < fn.ParamSlots || fn.ReturnSlots < 0 {
			return nil, fmt.Errorf("function '%s' has invalid slots", fn.Name)
		}
		if _, exists := functionSigs[fn.Name]; exists {
			return nil, fmt.Errorf("duplicate function '%s'", fn.Name)
		}
		functionSigs[fn.Name] = fn
	}

	e := &emitter{}
	symbolOffsets := make(map[string]int)
	symbolSigs := make(map[string]ir.IRFunc)
	var callPatches []callPatch
	var dataPatches []dataPatch
	var funcAddrPatches []funcAddrPatch
	dataBlobs := make([][]byte, 0, len(dataPrefix))
	dataBlobs = append(dataBlobs, dataPrefix...)
	for _, fn := range funcs {
		if err := validateLocalCallSignatures(fn, functionSigs); err != nil {
			return nil, err
		}
		symbolOffsets[fn.Name] = len(e.buf)
		symbolSigs[fn.Name] = fn
		if fn.ExportName != "" {
			if _, exists := symbolOffsets[fn.ExportName]; exists {
				return nil, fmt.Errorf("duplicate exported symbol '%s'", fn.ExportName)
			}
			symbolOffsets[fn.ExportName] = len(e.buf)
			symbolSigs[fn.ExportName] = fn
		}
		if err := emitFunc(e, fn, &callPatches, &dataPatches, &funcAddrPatches, &dataBlobs, opt); err != nil {
			return nil, err
		}
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
	validatePatchOffset := func(kind string, at int) error {
		if at < 0 || at+4 > len(e.buf) {
			return fmt.Errorf("invalid patch offset %d for %s", at, kind)
		}
		return nil
	}
	relocs := make([]tobj.Reloc, 0, len(callPatches)+len(dataPatches)+len(funcAddrPatches))
	for _, patch := range callPatches {
		if err := validatePatchOffset("call", patch.at); err != nil {
			return nil, err
		}
		relocs = append(relocs, tobj.Reloc{Kind: tobj.RelocCallRel32, At: uint32(patch.at), Name: patch.name})
	}
	for _, patch := range dataPatches {
		if err := validatePatchOffset("data", patch.at); err != nil {
			return nil, err
		}
		if patch.dataIndex < 0 || patch.dataIndex >= len(dataOffsets) {
			return nil, fmt.Errorf("invalid data patch index %d", patch.dataIndex)
		}
		relocs = append(relocs, tobj.Reloc{
			Kind:   tobj.RelocDataAbs32,
			At:     uint32(patch.at),
			Name:   "",
			Addend: uint32(dataOffsets[patch.dataIndex]),
		})
	}
	for _, patch := range funcAddrPatches {
		if err := validatePatchOffset("function address", patch.at); err != nil {
			return nil, err
		}
		if patch.name == "" {
			return nil, fmt.Errorf("function address patch name is empty")
		}
		relocs = append(relocs, tobj.Reloc{
			Kind:   tobj.RelocFuncAddrAbs32,
			At:     uint32(patch.at),
			Name:   patch.name,
			Addend: 0,
		})
	}
	return &tobj.Object{Code: e.buf, Data: data, Symbols: symbols, Relocs: relocs}, nil
}

func validateLocalCallSignatures(fn ir.IRFunc, functionSigs map[string]ir.IRFunc) error {
	for _, instr := range fn.Instrs {
		if instr.Kind != ir.IRCall {
			continue
		}
		target, ok := functionSigs[instr.Name]
		if !ok {
			continue
		}
		if instr.ArgSlots != target.ParamSlots || instr.RetSlots != target.ReturnSlots {
			return fmt.Errorf("function '%s' call %q ABI mismatch args=%d rets=%d want args=%d rets=%d", fn.Name, instr.Name, instr.ArgSlots, instr.RetSlots, target.ParamSlots, target.ReturnSlots)
		}
	}
	return nil
}

func emitFunc(e *emitter, fn ir.IRFunc, callPatches *[]callPatch, dataPatches *[]dataPatch, funcAddrPatches *[]funcAddrPatch, dataBlobs *[][]byte, opt x64.CodegenOptions) error {
	if fn.ParamSlots < 0 || fn.LocalSlots < fn.ParamSlots || fn.ReturnSlots < 0 {
		return fmt.Errorf("x86 backend: function '%s' has invalid slots", fn.Name)
	}
	labelOffsets := make(map[int]int)
	var patches []labelPatch
	stackDepth := 0
	pop := func(n int) error {
		if stackDepth < n {
			return fmt.Errorf("stack underflow in function '%s'", fn.Name)
		}
		stackDepth -= n
		return nil
	}
	push := func(n int) { stackDepth += n }
	localSlotOffset := func(slot int) (int32, error) {
		if slot < 0 || slot >= fn.LocalSlots {
			return 0, fmt.Errorf("x86 backend: local slot %d out of bounds in function '%s' (locals=%d)", slot, fn.Name, fn.LocalSlots)
		}
		return -int32((slot + 1) * 8), nil
	}

	e.pushEbp()
	e.movEbpEsp()
	localSize := alignUp(fn.LocalSlots*8, 16)
	if localSize > 0 {
		e.subEspImm32(int32(localSize))
	}
	for i := 0; i < fn.ParamSlots; i++ {
		dst, err := localSlotOffset(i)
		if err != nil {
			return err
		}
		e.movEaxFromEbpDisp(int32(8 + i*4))
		e.movMemEbpDispEax(dst)
		e.movMemEbpDispImm32(dst+4, 0)
	}
	for i := fn.ParamSlots; i < fn.LocalSlots; i++ {
		dst, err := localSlotOffset(i)
		if err != nil {
			return err
		}
		e.movMemEbpDispImm32(dst, 0)
		e.movMemEbpDispImm32(dst+4, 0)
	}

	for _, instr := range fn.Instrs {
		switch instr.Kind {
		case ir.IRWrite:
			if err := emitWrite(e, pop); err != nil {
				return err
			}
		case ir.IRStrLit:
			if dataBlobs == nil {
				return fmt.Errorf("x86 backend: missing data blobs for string literal in function '%s'", fn.Name)
			}
			if len(instr.Str) == 0 {
				e.movEaxImm32(0)
				e.pushEax()
				e.pushEax()
				push(2)
				continue
			}
			at := e.movEaxImm32Patch()
			*dataPatches = append(*dataPatches, dataPatch{at: at, dataIndex: len(*dataBlobs)})
			*dataBlobs = append(*dataBlobs, instr.Str)
			e.pushEax()
			e.movEaxImm32(uint32(len(instr.Str)))
			e.pushEax()
			push(2)
		case ir.IRConstI32:
			e.movEaxImm32(uint32(instr.Imm))
			e.pushEax()
			push(1)
		case ir.IRLoadLocal:
			off, err := localSlotOffset(instr.Local)
			if err != nil {
				return err
			}
			e.movEaxFromEbpDisp(off)
			e.pushEax()
			push(1)
		case ir.IRStoreLocal:
			if err := pop(1); err != nil {
				return err
			}
			off, err := localSlotOffset(instr.Local)
			if err != nil {
				return err
			}
			e.popEax()
			e.movMemEbpDispEax(off)
		case ir.IRLoadGlobal:
			if instr.Local < 0 {
				return fmt.Errorf("x86 backend: global slot %d out of bounds in function '%s'", instr.Local, fn.Name)
			}
			at := e.movEaxFromAbs32()
			*dataPatches = append(*dataPatches, dataPatch{at: at, dataIndex: instr.Local})
			e.pushEax()
			push(1)
		case ir.IRStoreGlobal:
			if instr.Local < 0 {
				return fmt.Errorf("x86 backend: global slot %d out of bounds in function '%s'", instr.Local, fn.Name)
			}
			if err := pop(1); err != nil {
				return err
			}
			e.popEax()
			at := e.movAbs32FromEax()
			*dataPatches = append(*dataPatches, dataPatch{at: at, dataIndex: instr.Local})
		case ir.IRAddI32:
			if err := pop(2); err != nil {
				return err
			}
			e.popEcx()
			e.popEax()
			e.addEaxEcx()
			e.pushEax()
			push(1)
		case ir.IRSubI32:
			if err := pop(2); err != nil {
				return err
			}
			e.popEcx()
			e.popEax()
			e.subEaxEcx()
			e.pushEax()
			push(1)
		case ir.IRNegI32:
			if err := pop(1); err != nil {
				return err
			}
			e.popEax()
			e.negEax()
			e.pushEax()
			push(1)
		case ir.IRMulI32:
			if err := pop(2); err != nil {
				return err
			}
			e.popEcx()
			e.popEax()
			e.imulEaxEcx()
			e.pushEax()
			push(1)
		case ir.IRDivI32:
			if err := pop(2); err != nil {
				return err
			}
			e.popEcx()
			e.popEax()
			e.cdq()
			e.idivEcx()
			e.pushEax()
			push(1)
		case ir.IRModI32:
			if err := pop(2); err != nil {
				return err
			}
			e.popEcx()
			e.popEax()
			e.cdq()
			e.idivEcx()
			e.pushEdx()
			push(1)
		case ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32, ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
			if err := emitCmp(e, instr.Kind, pop); err != nil {
				return err
			}
			push(1)
		case ir.IRLabel:
			if instr.Label < 0 {
				return fmt.Errorf("x86 backend: negative label %d in function '%s'", instr.Label, fn.Name)
			}
			if _, exists := labelOffsets[instr.Label]; exists {
				return fmt.Errorf("x86 backend: duplicate label %d in function '%s'", instr.Label, fn.Name)
			}
			labelOffsets[instr.Label] = len(e.buf)
		case ir.IRJmp:
			if instr.Label < 0 {
				return fmt.Errorf("x86 backend: negative label %d in function '%s'", instr.Label, fn.Name)
			}
			at := e.jmpRel32()
			patches = append(patches, labelPatch{at: at, label: instr.Label})
		case ir.IRJmpIfZero:
			if instr.Label < 0 {
				return fmt.Errorf("x86 backend: negative label %d in function '%s'", instr.Label, fn.Name)
			}
			if err := pop(1); err != nil {
				return err
			}
			e.popEax()
			e.testEaxEax()
			at := e.jzRel32()
			patches = append(patches, labelPatch{at: at, label: instr.Label})
		case ir.IRCall:
			if err := emitCall(e, fn, instr, pop, push, callPatches); err != nil {
				return err
			}
		case ir.IRSymAddr:
			if instr.Name == "" {
				return fmt.Errorf("x86 backend: symbol address is missing name in function '%s'", fn.Name)
			}
			at := e.movEaxImm32Patch()
			*funcAddrPatches = append(*funcAddrPatches, funcAddrPatch{at: at, name: instr.Name})
			e.pushEax()
			push(1)
		case ir.IRAllocBytes:
			if err := emitAllocBytes(e, pop, push); err != nil {
				return err
			}
		case ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32:
			if err := emitMakeSlice(e, instr.Kind, pop, push); err != nil {
				return err
			}
		case ir.IRRawSliceFromParts:
			if err := emitRawSliceFromParts(e, pop, push); err != nil {
				return err
			}
		case ir.IRSliceWindow, ir.IRSlicePrefix, ir.IRSliceSuffix:
			if err := emitSliceView(e, instr.Kind, byte(instr.Imm), pop, push); err != nil {
				return err
			}
		case ir.IRIslandNew:
			if err := emitIslandNew(e, pop, push, opt); err != nil {
				return err
			}
		case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
			if err := emitIslandMakeSlice(e, instr.Kind, pop, push); err != nil {
				return err
			}
		case ir.IRIslandFree:
			if err := emitIslandFree(e, pop, opt); err != nil {
				return err
			}
		case ir.IRIslandReset:
			if err := emitIslandReset(e, pop, push, opt); err != nil {
				return err
			}
		case ir.IRIndexLoadI32, ir.IRIndexLoadU8, ir.IRIndexLoadU16,
			ir.IRIndexLoadI32Unchecked, ir.IRIndexLoadU8Unchecked, ir.IRIndexLoadU16Unchecked:
			if err := emitIndexLoad(e, instr.Kind, pop, push); err != nil {
				return err
			}
		case ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
			if err := emitIndexStore(e, instr.Kind, pop); err != nil {
				return err
			}
		case ir.IRCapMem, ir.IRCapIO:
			e.movEaxImm32(1)
			e.pushEax()
			push(1)
		case ir.IRMemReadI32, ir.IRMemReadU8, ir.IRMemReadPtr:
			if err := emitRawMemoryRead(e, instr.Kind, pop, push); err != nil {
				return err
			}
		case ir.IRMemWriteI32, ir.IRMemWriteU8, ir.IRMemWritePtr, ir.IRMemWriteArchPtr:
			if err := emitRawMemoryWrite(e, instr.Kind, pop, push); err != nil {
				return err
			}
		case ir.IRMemReadI32Offset, ir.IRMemReadU8Offset, ir.IRMemReadPtrOffset:
			if err := emitRawMemoryOffsetRead(e, instr.Kind, pop, push); err != nil {
				return err
			}
		case ir.IRMemWriteI32Offset, ir.IRMemWriteU8Offset, ir.IRMemWritePtrOffset, ir.IRMemWriteArchPtrOffset:
			if err := emitRawMemoryOffsetWrite(e, instr.Kind, pop, push); err != nil {
				return err
			}
		case ir.IRPtrAdd:
			if err := emitPtrAdd(e, pop, push); err != nil {
				return err
			}
		case ir.IRMmioReadI32:
			if err := emitMMIOReadI32(e, pop, push); err != nil {
				return err
			}
		case ir.IRMmioWriteI32:
			if err := emitMMIOWriteI32(e, pop, push); err != nil {
				return err
			}
		case ir.IRCtxSwitch:
			if err := emitCtxSwitch(e, pop, push); err != nil {
				return err
			}
		case ir.IRAtomicFenceSeqCst:
			e.mfenceCompat()
		case ir.IRAtomicFenceRelaxed, ir.IRAtomicFenceAcquire, ir.IRAtomicFenceRelease, ir.IRAtomicFenceAcqRel:
		case ir.IRAtomicLoadPtr, ir.IRAtomicLoadI32:
			if err := atomicLoad32(e, pop); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicStorePtr, ir.IRAtomicStoreI32:
			if err := atomicStore32(e, pop); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicExchangePtr, ir.IRAtomicExchangeI32:
			if err := atomicExchange32(e, pop); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicFetchAddPtr, ir.IRAtomicFetchAddI32:
			if err := atomicFetchAdd32(e, pop); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicFetchSubPtr, ir.IRAtomicFetchSubI32:
			if err := atomicFetchSub32(e, pop); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicCompareExchangePtr, ir.IRAtomicCompareExchangeI32:
			if err := atomicCompareExchange32(e, pop); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicFetchAndPtr, ir.IRAtomicFetchAndI32:
			if err := atomicFetchCASLoop32(e, pop, e.andEbxEcx); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicFetchOrPtr, ir.IRAtomicFetchOrI32:
			if err := atomicFetchCASLoop32(e, pop, e.orEbxEcx); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicFetchXorPtr, ir.IRAtomicFetchXorI32:
			if err := atomicFetchCASLoop32(e, pop, e.xorEbxEcx); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicLoadI8:
			if err := atomicLoad8(e, pop); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicLoadI16:
			if err := atomicLoad16(e, pop); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicStoreI8:
			if err := atomicStore8(e, pop); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicExchangeI8:
			if err := atomicExchange8(e, pop); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicStoreI16:
			if err := atomicStore16(e, pop); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicExchangeI16:
			if err := atomicExchange16(e, pop); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicFetchAddI8:
			if err := atomicFetchAdd8(e, pop); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicFetchAddI16:
			if err := atomicFetchAdd16(e, pop); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicFetchSubI8:
			if err := atomicFetchSub8(e, pop); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicFetchSubI16:
			if err := atomicFetchSub16(e, pop); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicCompareExchangeI8:
			if err := atomicCompareExchange8(e, pop); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicCompareExchangeI16:
			if err := atomicCompareExchange16(e, pop); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicFetchAndI8:
			if err := atomicFetchCASLoop8(e, pop, e.andDlCl); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicFetchOrI8:
			if err := atomicFetchCASLoop8(e, pop, e.orDlCl); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicFetchXorI8:
			if err := atomicFetchCASLoop8(e, pop, e.xorDlCl); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicFetchAndI16:
			if err := atomicFetchCASLoop16(e, pop, e.andDxCx); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicFetchOrI16:
			if err := atomicFetchCASLoop16(e, pop, e.orDxCx); err != nil {
				return err
			}
			push(1)
		case ir.IRAtomicFetchXorI16:
			if err := atomicFetchCASLoop16(e, pop, e.xorDxCx); err != nil {
				return err
			}
			push(1)
		case ir.IRReturn:
			if err := emitReturn(e, fn, pop); err != nil {
				return err
			}
		default:
			return fmt.Errorf("x86 backend: unsupported IR instruction %v in function '%s'", instr.Kind, fn.Name)
		}
	}
	for _, patch := range patches {
		target, ok := labelOffsets[patch.label]
		if !ok {
			return fmt.Errorf("x86 backend: missing label %d in function '%s'", patch.label, fn.Name)
		}
		if err := e.patchRel32(patch.at, target); err != nil {
			return err
		}
	}
	return nil
}
