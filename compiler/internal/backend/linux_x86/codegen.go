package linux_x86

import (
	"encoding/binary"
	"fmt"
	"sort"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
)

func CodegenObjectLinuxX86(funcs []ir.IRFunc) (*tobj.Object, error) {
	return CodegenObjectLinuxX86WithDataPrefix(funcs, nil)
}

func CodegenObjectLinuxX86WithOptions(funcs []ir.IRFunc, opt x64.CodegenOptions) (*tobj.Object, error) {
	return CodegenObjectLinuxX86WithOptionsAndDataPrefix(funcs, nil, opt)
}

func CodegenObjectLinuxX86WithDataPrefix(funcs []ir.IRFunc, dataPrefix [][]byte) (*tobj.Object, error) {
	return CodegenObjectLinuxX86WithOptionsAndDataPrefix(funcs, dataPrefix, x64.CodegenOptions{})
}

func CodegenObjectLinuxX86WithOptionsAndDataPrefix(funcs []ir.IRFunc, dataPrefix [][]byte, opt x64.CodegenOptions) (*tobj.Object, error) {
	obj, err := buildObject(funcs, dataPrefix, opt)
	if err != nil {
		return nil, err
	}
	obj.Target = "linux-x86"
	return obj, nil
}

type emitter struct {
	buf []byte
}

type labelPatch struct {
	at    int
	label int
}

type callPatch struct {
	at   int
	name string
}

type dataPatch struct {
	at        int
	dataIndex int
}

type funcAddrPatch struct {
	at   int
	name string
}

func (e *emitter) emit(bs ...byte) {
	e.buf = append(e.buf, bs...)
}

func (e *emitter) imm32(v uint32) {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	e.emit(buf[:]...)
}

func (e *emitter) patchRel32(at int, target int) error {
	if at < 0 || at+4 > len(e.buf) {
		return fmt.Errorf("invalid rel32 patch offset %d", at)
	}
	rel := int32(target - (at + 4))
	binary.LittleEndian.PutUint32(e.buf[at:at+4], uint32(rel))
	return nil
}

func (e *emitter) pushEbp()            { e.emit(0x55) }
func (e *emitter) movEbpEsp()          { e.emit(0x89, 0xE5) }
func (e *emitter) leave()              { e.emit(0xC9) }
func (e *emitter) ret()                { e.emit(0xC3) }
func (e *emitter) pushEax()            { e.emit(0x50) }
func (e *emitter) pushEcx()            { e.emit(0x51) }
func (e *emitter) pushEdx()            { e.emit(0x52) }
func (e *emitter) pushEbx()            { e.emit(0x53) }
func (e *emitter) popEax()             { e.emit(0x58) }
func (e *emitter) popEcx()             { e.emit(0x59) }
func (e *emitter) popEdx()             { e.emit(0x5A) }
func (e *emitter) popEbx()             { e.emit(0x5B) }
func (e *emitter) popEbp()             { e.emit(0x5D) }
func (e *emitter) popEdi()             { e.emit(0x5F) }
func (e *emitter) cdq()                { e.emit(0x99) }
func (e *emitter) int80()              { e.emit(0xCD, 0x80) }
func (e *emitter) mfenceCompat()       { e.emit(0xF0, 0x83, 0x0C, 0x24, 0x00) }
func (e *emitter) movEaxEcx()          { e.emit(0x89, 0xC8) }
func (e *emitter) movEcxEax()          { e.emit(0x89, 0xC1) }
func (e *emitter) movEdxEax()          { e.emit(0x89, 0xC2) }
func (e *emitter) movEdiEax()          { e.emit(0x89, 0xC7) }
func (e *emitter) movEaxEdi()          { e.emit(0x89, 0xF8) }
func (e *emitter) movEdiEdx()          { e.emit(0x89, 0xD7) }
func (e *emitter) movEbxEax()          { e.emit(0x89, 0xC3) }
func (e *emitter) movEbxEcx()          { e.emit(0x89, 0xCB) }
func (e *emitter) movEaxEcxValue()     { e.emit(0x89, 0xC8) }
func (e *emitter) addEaxEcx()          { e.emit(0x01, 0xC8) }
func (e *emitter) addEaxEdx()          { e.emit(0x01, 0xD0) }
func (e *emitter) addEdxEax()          { e.emit(0x01, 0xC2) }
func (e *emitter) addEdiEcx()          { e.emit(0x01, 0xCF) }
func (e *emitter) subEaxEcx()          { e.emit(0x29, 0xC8) }
func (e *emitter) subEaxEdi()          { e.emit(0x29, 0xF8) }
func (e *emitter) imulEaxEcx()         { e.emit(0x0F, 0xAF, 0xC1) }
func (e *emitter) idivEcx()            { e.emit(0xF7, 0xF9) }
func (e *emitter) negEax()             { e.emit(0xF7, 0xD8) }
func (e *emitter) negEcx()             { e.emit(0xF7, 0xD9) }
func (e *emitter) cmpEaxEcx()          { e.emit(0x39, 0xC8) }
func (e *emitter) cmpEdxEcx()          { e.emit(0x39, 0xCA) }
func (e *emitter) cmpEdiEbx()          { e.emit(0x39, 0xDF) }
func (e *emitter) testEaxEax()         { e.emit(0x85, 0xC0) }
func (e *emitter) xorEbxEbx()          { e.emit(0x31, 0xDB) }
func (e *emitter) xorEbpEbp()          { e.emit(0x31, 0xED) }
func (e *emitter) movzxEaxAl()         { e.emit(0x0F, 0xB6, 0xC0) }
func (e *emitter) movzxEaxAx()         { e.emit(0x0F, 0xB7, 0xC0) }
func (e *emitter) movEaxFromEaxPtr()   { e.emit(0x8B, 0x00) }
func (e *emitter) movEdxFromEaxPtr()   { e.emit(0x8B, 0x10) }
func (e *emitter) movEaxFromEdiPtr()   { e.emit(0x8B, 0x07) }
func (e *emitter) movEcxFromEdiPtr()   { e.emit(0x8B, 0x0F) }
func (e *emitter) movzxEaxByteEaxPtr() { e.emit(0x0F, 0xB6, 0x00) }
func (e *emitter) movzxEaxByteEdiPtr() { e.emit(0x0F, 0xB6, 0x07) }
func (e *emitter) movzxEaxWordEaxPtr() { e.emit(0x0F, 0xB7, 0x00) }
func (e *emitter) movzxEaxWordEdiPtr() { e.emit(0x0F, 0xB7, 0x07) }
func (e *emitter) movMemEaxPtrEcx()    { e.emit(0x89, 0x08) }
func (e *emitter) movMemEaxPtrEbx()    { e.emit(0x89, 0x18) }
func (e *emitter) movMemEaxPtrEdi()    { e.emit(0x89, 0x38) }
func (e *emitter) movMem8EaxPtrBl()    { e.emit(0x88, 0x18) }
func (e *emitter) movMem16EaxPtrBx()   { e.emit(0x66, 0x89, 0x18) }
func (e *emitter) xchgMem32EdiEcx()    { e.emit(0x87, 0x0F) }
func (e *emitter) xchgMem8EdiCl()      { e.emit(0x86, 0x0F) }
func (e *emitter) xchgMem16EdiCx()     { e.emit(0x66, 0x87, 0x0F) }
func (e *emitter) lockXaddMem32EdiEcx() {
	e.emit(0xF0, 0x0F, 0xC1, 0x0F)
}
func (e *emitter) lockXaddMem8EdiCl() {
	e.emit(0xF0, 0x0F, 0xC0, 0x0F)
}
func (e *emitter) lockXaddMem16EdiCx() {
	e.emit(0x66, 0xF0, 0x0F, 0xC1, 0x0F)
}
func (e *emitter) lockCmpxchgMem32EdiEdx() {
	e.emit(0xF0, 0x0F, 0xB1, 0x17)
}
func (e *emitter) lockCmpxchgMem32EdiEbx() {
	e.emit(0xF0, 0x0F, 0xB1, 0x1F)
}
func (e *emitter) lockCmpxchgMem8EdiDl() {
	e.emit(0xF0, 0x0F, 0xB0, 0x17)
}
func (e *emitter) lockCmpxchgMem16EdiDx() {
	e.emit(0x66, 0xF0, 0x0F, 0xB1, 0x17)
}
func (e *emitter) andEbxEcx() { e.emit(0x21, 0xCB) }
func (e *emitter) orEbxEcx()  { e.emit(0x09, 0xCB) }
func (e *emitter) xorEbxEcx() { e.emit(0x31, 0xCB) }
func (e *emitter) andDlCl()   { e.emit(0x20, 0xCA) }
func (e *emitter) orDlCl()    { e.emit(0x08, 0xCA) }
func (e *emitter) xorDlCl()   { e.emit(0x30, 0xCA) }
func (e *emitter) andDxCx()   { e.emit(0x66, 0x21, 0xCA) }
func (e *emitter) orDxCx()    { e.emit(0x66, 0x09, 0xCA) }
func (e *emitter) xorDxCx()   { e.emit(0x66, 0x31, 0xCA) }

func (e *emitter) subEspImm32(v int32) {
	e.emit(0x81, 0xEC)
	e.imm32(uint32(v))
}

func (e *emitter) addEspImm32(v int32) {
	e.emit(0x81, 0xC4)
	e.imm32(uint32(v))
}

func (e *emitter) addEaxImm8(v byte) {
	e.emit(0x83, 0xC0, v)
}

func (e *emitter) addEdxImm8(v byte) {
	e.emit(0x83, 0xC2, v)
}

func (e *emitter) subEdxImm8(v byte) {
	e.emit(0x83, 0xEA, v)
}

func (e *emitter) addEdiImm8(v byte) {
	e.emit(0x83, 0xC7, v)
}

func (e *emitter) addEcxImm32(v int32) {
	e.emit(0x81, 0xC1)
	e.imm32(uint32(v))
}

func (e *emitter) subEcxImm32(v int32) {
	e.emit(0x81, 0xE9)
	e.imm32(uint32(v))
}

func (e *emitter) addEbxImm32(v int32) {
	e.emit(0x81, 0xC3)
	e.imm32(uint32(v))
}

func (e *emitter) movEaxImm32(v uint32) {
	e.emit(0xB8)
	e.imm32(v)
}

func (e *emitter) movEbxImm32(v uint32) {
	e.emit(0xBB)
	e.imm32(v)
}

func (e *emitter) movEdxImm32(v uint32) {
	e.emit(0xBA)
	e.imm32(v)
}

func (e *emitter) movEsiImm32(v uint32) {
	e.emit(0xBE)
	e.imm32(v)
}

func (e *emitter) movEdiImm32(v uint32) {
	e.emit(0xBF)
	e.imm32(v)
}

func (e *emitter) movEaxImm32Patch() int {
	e.emit(0xB8, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) movEaxFromEbpDisp(disp int32) {
	e.emit(0x8B, 0x85)
	e.imm32(uint32(disp))
}

func (e *emitter) movMemEbpDispEax(disp int32) {
	e.emit(0x89, 0x85)
	e.imm32(uint32(disp))
}

func (e *emitter) movEaxFromEspDisp(disp int32) {
	e.emit(0x8B, 0x84, 0x24)
	e.imm32(uint32(disp))
}

func (e *emitter) movMemEspDispEax(disp int32) {
	e.emit(0x89, 0x84, 0x24)
	e.imm32(uint32(disp))
}

func (e *emitter) movMemEbpDispImm32(disp int32, v uint32) {
	e.emit(0xC7, 0x85)
	e.imm32(uint32(disp))
	e.imm32(v)
}

func (e *emitter) movMemEaxPtrImm32(v uint32) {
	e.emit(0xC7, 0x00)
	e.imm32(v)
}

func (e *emitter) movMemEaxDispEcx(disp byte) {
	e.emit(0x89, 0x48, disp)
}

func (e *emitter) movMemEaxDispImm32(disp byte, v uint32) {
	e.emit(0xC7, 0x40, disp)
	e.imm32(v)
}

func (e *emitter) movEbxFromEaxDisp(disp byte) {
	e.emit(0x8B, 0x58, disp)
}

func (e *emitter) movEaxFromEbxDisp(disp byte) {
	e.emit(0x8B, 0x43, disp)
}

func (e *emitter) movEcxFromEbxDisp(disp byte) {
	e.emit(0x8B, 0x4B, disp)
}

func (e *emitter) movMemEbxDispImm32(disp byte, v uint32) {
	e.emit(0xC7, 0x43, disp)
	e.imm32(v)
}

func (e *emitter) andEdiImm32(v int32) {
	e.emit(0x81, 0xE7)
	e.imm32(uint32(v))
}

func (e *emitter) cmpEcxImm8(v byte) {
	e.emit(0x83, 0xF9, v)
}

func (e *emitter) cmpEdxImm8(v byte) {
	e.emit(0x83, 0xFA, v)
}

func (e *emitter) cmpEaxImm32(v uint32) {
	e.emit(0x3D)
	e.imm32(v)
}

func (e *emitter) shlEcxImm8(v byte) {
	e.emit(0xC1, 0xE1, v)
}

func (e *emitter) shlEdxImm8(v byte) {
	e.emit(0xC1, 0xE2, v)
}

func (e *emitter) setccAl(op byte) {
	e.emit(0x0F, op, 0xC0)
}

func (e *emitter) jgeRel32() int {
	e.emit(0x0F, 0x8D, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) jaeRel32() int {
	e.emit(0x0F, 0x83, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) jaRel32() int {
	e.emit(0x0F, 0x87, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) jnzRel32() int {
	e.emit(0x0F, 0x85, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) jzRel32() int {
	e.emit(0x0F, 0x84, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) jmpRel32() int {
	e.emit(0xE9, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) callRel32() int {
	e.emit(0xE8, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) movEaxFromAbs32() int {
	e.emit(0xA1, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) movAbs32FromEax() int {
	e.emit(0xA3, 0, 0, 0, 0)
	return len(e.buf) - 4
}

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
		case ir.IRIndexLoadI32, ir.IRIndexLoadU8, ir.IRIndexLoadU16:
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

func emitCall(e *emitter, fn ir.IRFunc, instr ir.IRInstr, pop func(int) error, push func(int), callPatches *[]callPatch) error {
	if instr.Name == "" {
		return fmt.Errorf("x86 backend: empty call target in function '%s'", fn.Name)
	}
	if instr.ArgSlots < 0 || instr.RetSlots < 0 {
		return fmt.Errorf("x86 backend: call %q has negative ABI slots args=%d rets=%d in function '%s'", instr.Name, instr.ArgSlots, instr.RetSlots, fn.Name)
	}
	if instr.RetSlots > 3 {
		return fmt.Errorf("x86 backend: unsupported call return slots %d in function '%s' call %q", instr.RetSlots, fn.Name, instr.Name)
	}
	if err := pop(instr.ArgSlots); err != nil {
		return err
	}
	argBytes := instr.ArgSlots * 4
	if argBytes > 0 {
		e.subEspImm32(int32(argBytes))
		for i := 0; i < instr.ArgSlots; i++ {
			src := int32(argBytes + 4*(instr.ArgSlots-1-i))
			dst := int32(4 * i)
			e.movEaxFromEspDisp(src)
			e.movMemEspDispEax(dst)
		}
	}
	at := e.callRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: instr.Name})
	if argBytes > 0 {
		e.addEspImm32(int32(argBytes * 2))
	}
	switch instr.RetSlots {
	case 0:
	case 1:
		e.pushEax()
		push(1)
	case 2:
		e.pushEax()
		e.pushEdx()
		push(2)
	case 3:
		e.pushEax()
		e.pushEdx()
		e.pushEcx()
		push(3)
	}
	return nil
}

func emitReturn(e *emitter, fn ir.IRFunc, pop func(int) error) error {
	if err := pop(fn.ReturnSlots); err != nil {
		return err
	}
	switch fn.ReturnSlots {
	case 0:
	case 1:
		e.popEax()
	case 2:
		e.popEdx()
		e.popEax()
	case 3:
		e.popEcx()
		e.popEdx()
		e.popEax()
	default:
		return fmt.Errorf("x86 backend: unsupported return slots %d in function '%s'", fn.ReturnSlots, fn.Name)
	}
	e.leave()
	e.ret()
	return nil
}

func emitCmp(e *emitter, kind ir.IRInstrKind, pop func(int) error) error {
	if err := pop(2); err != nil {
		return err
	}
	e.popEcx()
	e.popEax()
	e.cmpEaxEcx()
	switch kind {
	case ir.IRCmpEqI32:
		e.setccAl(0x94)
	case ir.IRCmpLtI32:
		e.setccAl(0x9C)
	case ir.IRCmpGtI32:
		e.setccAl(0x9F)
	case ir.IRCmpGeI32:
		e.setccAl(0x9D)
	case ir.IRCmpLeI32:
		e.setccAl(0x9E)
	case ir.IRCmpNeI32:
		e.setccAl(0x95)
	default:
		return fmt.Errorf("unsupported comparison kind %v", kind)
	}
	e.movzxEaxAl()
	e.pushEax()
	return nil
}

func emitWrite(e *emitter, pop func(int) error) error {
	if err := pop(2); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.movEbxImm32(1)
	e.movEaxImm32(4)
	e.int80()
	return nil
}

func emitAllocBytes(e *emitter, pop func(int) error, push func(int)) error {
	if err := pop(1); err != nil {
		return err
	}
	e.popEcx()
	e.cmpEcxImm8(1)
	sizeOKAt := e.jgeRel32()
	emitExit(e, 2)
	sizeOKOff := len(e.buf)
	if err := e.patchRel32(sizeOKAt, sizeOKOff); err != nil {
		return err
	}

	e.pushEcx()
	e.addEcxImm32(8)
	emitMmap2Anonymous(e)
	failAt := emitMmapFailureBranch(e)
	e.popEcx()
	e.movMemEaxPtrEcx()
	e.addEaxImm8(8)
	e.pushEax()
	push(1)
	return patchExitBranch(e, failAt, 2)
}

func emitMakeSlice(e *emitter, kind ir.IRInstrKind, pop func(int) error, push func(int)) error {
	if err := pop(1); err != nil {
		return err
	}
	e.popEcx()
	e.pushEcx()
	switch kind {
	case ir.IRMakeSliceI32:
		e.shlEcxImm8(2)
	case ir.IRMakeSliceU16:
		e.shlEcxImm8(1)
	case ir.IRMakeSliceU8:
	default:
		return fmt.Errorf("x86 backend: unsupported make_slice kind %v", kind)
	}
	emitMmap2Anonymous(e)
	failAt := emitMmapFailureBranch(e)
	e.popEcx()
	e.pushEax()
	e.pushEcx()
	push(2)
	return patchExitBranch(e, failAt, 2)
}

func emitIslandNew(e *emitter, pop func(int) error, push func(int), opt x64.CodegenOptions) error {
	if err := pop(1); err != nil {
		return err
	}
	e.popEcx()
	headerSize := int32(16)
	if opt.IslandsDebug {
		headerSize = x64.IslandsDebugPageSize
	}
	e.addEcxImm32(headerSize)
	e.pushEcx()
	emitMmap2Anonymous(e)
	failAt := emitMmapFailureBranch(e)
	e.popEcx()
	e.movMemEaxPtrImm32(uint32(headerSize))
	e.movMemEaxDispEcx(4)
	e.movMemEaxDispEcx(8)
	e.movMemEaxDispImm32(12, 0)
	e.pushEax()
	push(1)
	return patchExitBranch(e, failAt, 2)
}

func emitIslandMakeSlice(e *emitter, kind ir.IRInstrKind, pop func(int) error, push func(int)) error {
	if err := pop(2); err != nil {
		return err
	}
	e.popEcx()
	e.popEax()
	e.pushEcx()
	switch kind {
	case ir.IRIslandMakeSliceI32:
		e.shlEcxImm8(2)
	case ir.IRIslandMakeSliceU16:
		e.shlEcxImm8(1)
	case ir.IRIslandMakeSliceU8:
	default:
		return fmt.Errorf("x86 backend: unsupported island_make_slice kind %v", kind)
	}
	e.movEdxFromEaxPtr()
	e.movEbxFromEaxDisp(4)
	e.movEdiEdx()
	e.addEdiEcx()
	e.cmpEdiEbx()
	failAt := e.jaRel32()
	e.movMemEaxPtrEdi()
	e.addEdxEax()
	e.popEcx()
	e.pushEdx()
	e.pushEcx()
	push(2)
	return patchExitBranch(e, failAt, 1)
}

func emitIslandFree(e *emitter, pop func(int) error, opt x64.CodegenOptions) error {
	if err := pop(1); err != nil {
		return err
	}
	e.popEbx()
	if opt.IslandsDebug {
		e.movEaxFromEbxDisp(12)
		e.testEaxEax()
		okAt := e.jzRel32()
		emitExit(e, 2)
		okOff := len(e.buf)
		if err := e.patchRel32(okAt, okOff); err != nil {
			return err
		}
		e.movMemEbxDispImm32(12, 1)
		e.movEcxFromEbxDisp(8)
		e.subEcxImm32(x64.IslandsDebugPageSize)
		e.addEbxImm32(x64.IslandsDebugPageSize)
		e.movEdxImm32(0)
		e.movEaxImm32(125)
		e.int80()
		return nil
	}
	e.movEcxFromEbxDisp(8)
	e.movEaxImm32(91)
	e.int80()
	return nil
}

func emitIndexLoad(e *emitter, kind ir.IRInstrKind, pop func(int) error, push func(int)) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEax()
	e.cmpEdxEcx()
	failAt := e.jaeRel32()
	scaleIndex(e, kind)
	e.addEaxEdx()
	switch kind {
	case ir.IRIndexLoadI32:
		e.movEaxFromEaxPtr()
	case ir.IRIndexLoadU16:
		e.movzxEaxWordEaxPtr()
	case ir.IRIndexLoadU8:
		e.movzxEaxByteEaxPtr()
	default:
		return fmt.Errorf("x86 backend: unsupported index load kind %v", kind)
	}
	e.pushEax()
	push(1)
	return patchExitBranch(e, failAt, 1)
}

func emitIndexStore(e *emitter, kind ir.IRInstrKind, pop func(int) error) error {
	if err := pop(4); err != nil {
		return err
	}
	e.popEbx()
	e.popEdx()
	e.popEcx()
	e.popEax()
	e.cmpEdxEcx()
	failAt := e.jaeRel32()
	scaleIndex(e, kind)
	e.addEaxEdx()
	switch kind {
	case ir.IRIndexStoreI32:
		e.movMemEaxPtrEbx()
	case ir.IRIndexStoreU16:
		e.movMem16EaxPtrBx()
	case ir.IRIndexStoreU8:
		e.movMem8EaxPtrBl()
	default:
		return fmt.Errorf("x86 backend: unsupported index store kind %v", kind)
	}
	return patchExitBranch(e, failAt, 1)
}

func scaleIndex(e *emitter, kind ir.IRInstrKind) {
	switch kind {
	case ir.IRIndexLoadI32, ir.IRIndexStoreI32:
		e.shlEdxImm8(2)
	case ir.IRIndexLoadU16, ir.IRIndexStoreU16:
		e.shlEdxImm8(1)
	}
}

func emitMmap2Anonymous(e *emitter) {
	e.xorEbxEbx()
	e.movEdxImm32(3)
	e.movEsiImm32(0x22)
	e.movEdiImm32(0xFFFFFFFF)
	e.pushEbp()
	e.xorEbpEbp()
	e.movEaxImm32(192)
	e.int80()
	e.popEbp()
}

func emitMmapFailureBranch(e *emitter) int {
	e.cmpEaxImm32(0xFFFFF001)
	return e.jaeRel32()
}

func patchExitBranch(e *emitter, failAt int, code int32) error {
	doneAt := e.jmpRel32()
	failOff := len(e.buf)
	emitExit(e, code)
	doneOff := len(e.buf)
	if err := e.patchRel32(failAt, failOff); err != nil {
		return err
	}
	return e.patchRel32(doneAt, doneOff)
}

func emitExit(e *emitter, code int32) {
	e.movEbxImm32(uint32(code))
	e.movEaxImm32(1)
	e.int80()
}

func emitRawMemoryRead(e *emitter, kind ir.IRInstrKind, pop func(int) error, push func(int)) error {
	if err := pop(2); err != nil {
		return err
	}
	e.popEdx()
	e.popEax()
	if err := guardAllocationBaseRawAccess(e, rawMemoryWidthBytes(kind)); err != nil {
		return err
	}
	emitRawLoad(e, kind)
	e.pushEax()
	push(1)
	return nil
}

func emitRawMemoryWrite(e *emitter, kind ir.IRInstrKind, pop func(int) error, push func(int)) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEbx()
	e.popEax()
	if err := guardAllocationBaseRawAccess(e, rawMemoryWidthBytes(kind)); err != nil {
		return err
	}
	emitRawStore(e, kind)
	e.pushEbx()
	push(1)
	return nil
}

func emitRawMemoryOffsetRead(e *emitter, kind ir.IRInstrKind, pop func(int) error, push func(int)) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEcx()
	e.popEdx()
	e.popEax()
	if err := guardAllocationOffsetRawAccess(e, rawMemoryWidthBytes(kind)); err != nil {
		return err
	}
	emitRawLoad(e, kind)
	e.pushEax()
	push(1)
	return nil
}

func emitRawMemoryOffsetWrite(e *emitter, kind ir.IRInstrKind, pop func(int) error, push func(int)) error {
	if err := pop(4); err != nil {
		return err
	}
	e.popEcx()
	e.popEbx()
	e.popEdx()
	e.popEax()
	if err := guardAllocationOffsetRawAccess(e, rawMemoryWidthBytes(kind)); err != nil {
		return err
	}
	emitRawStore(e, kind)
	e.pushEbx()
	push(1)
	return nil
}

func emitPtrAdd(e *emitter, pop func(int) error, push func(int)) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEcx()
	e.popEdx()
	e.popEax()
	if err := guardAllocationOffsetRawAccess(e, 1); err != nil {
		return err
	}
	e.pushEax()
	push(1)
	return nil
}

func emitMMIOReadI32(e *emitter, pop func(int) error, push func(int)) error {
	if err := pop(2); err != nil {
		return err
	}
	e.popEdx()
	e.popEax()
	e.movEaxFromEaxPtr()
	e.pushEax()
	push(1)
	return nil
}

func emitMMIOWriteI32(e *emitter, pop func(int) error, push func(int)) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEax()
	e.movMemEaxPtrEcx()
	e.pushEcx()
	push(1)
	return nil
}

func rawMemoryWidthBytes(kind ir.IRInstrKind) byte {
	switch kind {
	case ir.IRMemReadU8, ir.IRMemWriteU8, ir.IRMemReadU8Offset, ir.IRMemWriteU8Offset:
		return 1
	default:
		return 4
	}
}

func emitRawLoad(e *emitter, kind ir.IRInstrKind) {
	switch kind {
	case ir.IRMemReadU8, ir.IRMemReadU8Offset:
		e.movzxEaxByteEaxPtr()
	default:
		e.movEaxFromEaxPtr()
	}
}

func emitRawStore(e *emitter, kind ir.IRInstrKind) {
	switch kind {
	case ir.IRMemWriteU8, ir.IRMemWriteU8Offset:
		e.movMem8EaxPtrBl()
	default:
		e.movMemEaxPtrEbx()
	}
}

func guardAllocationBaseRawAccess(e *emitter, width byte) error {
	e.movEdxImm32(0)
	return guardAllocationOffsetRawAccess(e, width)
}

func guardAllocationOffsetRawAccess(e *emitter, width byte) error {
	e.cmpEdxImm8(0)
	offsetOKAt := e.jgeRel32()
	emitExit(e, 2)
	offsetOKOff := len(e.buf)
	if err := e.patchRel32(offsetOKAt, offsetOKOff); err != nil {
		return err
	}
	e.movEdiEax()
	e.andEdiImm32(-4096)
	e.movEcxFromEdiPtr()
	e.addEdiImm8(8)
	e.subEaxEdi()
	e.addEdxEax()
	e.addEdxImm8(width)
	e.cmpEdxEcx()
	failAt := e.jaRel32()
	e.subEdxImm8(width)
	e.movEaxEdi()
	e.addEaxEdx()
	return patchExitBranch(e, failAt, 2)
}

func atomicLoad32(e *emitter, pop func(int) error) error {
	if err := pop(2); err != nil {
		return err
	}
	e.popEdx()
	e.popEdi()
	e.movEaxFromEdiPtr()
	e.pushEax()
	return nil
}

func atomicLoad8(e *emitter, pop func(int) error) error {
	if err := pop(2); err != nil {
		return err
	}
	e.popEdx()
	e.popEdi()
	e.movzxEaxByteEdiPtr()
	e.pushEax()
	return nil
}

func atomicLoad16(e *emitter, pop func(int) error) error {
	if err := pop(2); err != nil {
		return err
	}
	e.popEdx()
	e.popEdi()
	e.movzxEaxWordEdiPtr()
	e.pushEax()
	return nil
}

func atomicStore32(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.movEbxEcx()
	e.xchgMem32EdiEcx()
	e.pushEbx()
	return nil
}

func atomicStore8(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.movEaxEcx()
	e.movzxEaxAl()
	e.xchgMem8EdiCl()
	e.pushEax()
	return nil
}

func atomicStore16(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.movEaxEcx()
	e.movzxEaxAx()
	e.xchgMem16EdiCx()
	e.pushEax()
	return nil
}

func atomicExchange32(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.xchgMem32EdiEcx()
	e.pushEcx()
	return nil
}

func atomicExchange8(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.xchgMem8EdiCl()
	e.movzxEaxAl()
	e.pushEax()
	return nil
}

func atomicExchange16(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.xchgMem16EdiCx()
	e.movzxEaxAx()
	e.pushEax()
	return nil
}

func atomicFetchAdd32(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.lockXaddMem32EdiEcx()
	e.pushEcx()
	return nil
}

func atomicFetchAdd8(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.lockXaddMem8EdiCl()
	e.movzxEaxAl()
	e.pushEax()
	return nil
}

func atomicFetchAdd16(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.lockXaddMem16EdiCx()
	e.movzxEaxAx()
	e.pushEax()
	return nil
}

func atomicFetchSub32(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.negEcx()
	e.lockXaddMem32EdiEcx()
	e.pushEcx()
	return nil
}

func atomicFetchSub8(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.negEcx()
	e.lockXaddMem8EdiCl()
	e.movzxEaxAl()
	e.pushEax()
	return nil
}

func atomicFetchSub16(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.negEcx()
	e.lockXaddMem16EdiCx()
	e.movzxEaxAx()
	e.pushEax()
	return nil
}

func atomicCompareExchange32(e *emitter, pop func(int) error) error {
	if err := pop(4); err != nil {
		return err
	}
	e.popEbx()
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.movEaxEcxValue()
	e.lockCmpxchgMem32EdiEdx()
	e.pushEax()
	return nil
}

func atomicCompareExchange8(e *emitter, pop func(int) error) error {
	if err := pop(4); err != nil {
		return err
	}
	e.popEbx()
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.movEaxEcxValue()
	e.lockCmpxchgMem8EdiDl()
	e.movzxEaxAl()
	e.pushEax()
	return nil
}

func atomicCompareExchange16(e *emitter, pop func(int) error) error {
	if err := pop(4); err != nil {
		return err
	}
	e.popEbx()
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.movEaxEcxValue()
	e.lockCmpxchgMem16EdiDx()
	e.movzxEaxAx()
	e.pushEax()
	return nil
}

func atomicFetchCASLoop32(e *emitter, pop func(int) error, op func()) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.movEaxFromEdiPtr()
	retry := len(e.buf)
	e.movEbxEax()
	op()
	e.lockCmpxchgMem32EdiEbx()
	jnz := e.jnzRel32()
	if err := e.patchRel32(jnz, retry); err != nil {
		return err
	}
	e.pushEax()
	return nil
}

func atomicFetchCASLoop8(e *emitter, pop func(int) error, op func()) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.movzxEaxByteEdiPtr()
	retry := len(e.buf)
	e.movEdxEax()
	op()
	e.lockCmpxchgMem8EdiDl()
	jnz := e.jnzRel32()
	if err := e.patchRel32(jnz, retry); err != nil {
		return err
	}
	e.movzxEaxAl()
	e.pushEax()
	return nil
}

func atomicFetchCASLoop16(e *emitter, pop func(int) error, op func()) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.movzxEaxWordEdiPtr()
	retry := len(e.buf)
	e.movEdxEax()
	op()
	e.lockCmpxchgMem16EdiDx()
	jnz := e.jnzRel32()
	if err := e.patchRel32(jnz, retry); err != nil {
		return err
	}
	e.movzxEaxAx()
	e.pushEax()
	return nil
}

func alignUp(value int, align int) int {
	if align <= 1 {
		return value
	}
	remainder := value % align
	if remainder == 0 {
		return value
	}
	return value + align - remainder
}
