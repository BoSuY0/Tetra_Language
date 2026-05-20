package wasm32_wasi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"sort"

	"tetra_language/compiler/internal/ir"
)

const (
	wasmPageSize  = 65536
	wasmHeapAlign = 16

	// Reserved scratch area for fd_write iovec and written-bytes output.
	iovecAddr   = uint32(0x0800)
	nwrittenPtr = uint32(0x0810)
	dataBase    = uint32(0x1000)
)

type Function struct {
	Name        string
	ParamSlots  int
	LocalSlots  int
	ReturnSlots int
	Instrs      []ir.IRInstr
}

type Object struct {
	Functions   []Function
	MainName    string
	GlobalSlots int
	GlobalInits []int32
}

type wasmFunctionSignature struct {
	ParamSlots  int
	ReturnSlots int
}

func CodegenObject(funcs []ir.IRFunc, mainName string) (*Object, error) {
	return CodegenObjectWithDataPrefix(funcs, mainName, nil)
}

func CodegenObjectWithDataPrefix(funcs []ir.IRFunc, mainName string, dataPrefix [][]byte) (*Object, error) {
	if len(funcs) == 0 {
		return nil, fmt.Errorf("wasm backend: no functions to compile")
	}
	out := make([]Function, 0, len(funcs))
	globalSlots := 0
	symbolTokens := make(map[uint32]string)
	functionNames := make(map[string]struct{}, len(funcs))
	functionSigs := make(map[string]wasmFunctionSignature, len(funcs))
	for _, fn := range funcs {
		if err := validateWasmFunctionMetadata(functionNames, fn.Name, fn.ParamSlots, fn.LocalSlots, fn.ReturnSlots); err != nil {
			return nil, err
		}
		if err := validateWasmLabelMetadata(fn.Name, fn.Instrs); err != nil {
			return nil, err
		}
		out = append(out, Function{
			Name:        fn.Name,
			ParamSlots:  fn.ParamSlots,
			LocalSlots:  fn.LocalSlots,
			ReturnSlots: fn.ReturnSlots,
			Instrs:      fn.Instrs,
		})
		functionSigs[fn.Name] = wasmFunctionSignature{ParamSlots: fn.ParamSlots, ReturnSlots: fn.ReturnSlots}
	}
	for _, fn := range funcs {
		for _, instr := range fn.Instrs {
			if instr.Kind == ir.IRLoadGlobal || instr.Kind == ir.IRStoreGlobal {
				if instr.Local < 0 {
					return nil, wasmNegativeGlobalSlotError(fn.Name, instr.Local)
				}
				if instr.Local+1 > globalSlots {
					globalSlots = instr.Local + 1
				}
			} else if instr.Kind == ir.IRSymAddr {
				if err := validateWasmSymbolToken(symbolTokens, instr.Name); err != nil {
					return nil, err
				}
			} else if instr.Kind == ir.IRCall {
				if err := validateWasmCallMetadata(fn.Name, instr, functionSigs); err != nil {
					return nil, err
				}
			} else if instr.Kind == ir.IRLoadLocal || instr.Kind == ir.IRStoreLocal {
				if err := validateWasmLocalSlot(fn.Name, fn.LocalSlots, instr.Local); err != nil {
					return nil, err
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	globalInits, err := wasmGlobalInitializers(globalSlots, dataPrefix)
	if err != nil {
		return nil, err
	}
	return &Object{Functions: out, MainName: mainName, GlobalSlots: globalSlots, GlobalInits: globalInits}, nil
}

func LinkObject(obj *Object) ([]byte, error) {
	if obj == nil {
		return nil, fmt.Errorf("wasm backend: missing object")
	}
	if len(obj.Functions) == 0 {
		return nil, fmt.Errorf("wasm backend: missing functions")
	}
	if err := validateWasmObjectFunctions(obj.Functions); err != nil {
		return nil, err
	}
	if err := validateWasmObjectGlobalSlots(obj); err != nil {
		return nil, err
	}
	if err := validateWasmObjectLabels(obj.Functions); err != nil {
		return nil, err
	}
	if err := validateWasmObjectLocalSlots(obj.Functions); err != nil {
		return nil, err
	}
	if err := validateWasmObjectCalls(obj.Functions); err != nil {
		return nil, err
	}
	if err := validateWasmObjectSymbolTokens(obj.Functions); err != nil {
		return nil, err
	}
	mainName := obj.MainName
	if mainName == "" {
		mainName = "main"
	}

	const (
		fdWriteImport = iota
		procExitImport
		importCount
	)

	typeIdxBySig := map[string]uint32{}
	var typeEntries []wasmFuncType
	typeIndex := func(params int, returns int) uint32 {
		key := fmt.Sprintf("p%d-r%d", params, returns)
		if idx, ok := typeIdxBySig[key]; ok {
			return idx
		}
		idx := uint32(len(typeEntries))
		typeIdxBySig[key] = idx
		typeEntries = append(typeEntries, wasmFuncType{
			paramCount:  params,
			returnCount: returns,
		})
		return idx
	}

	fdWriteType := typeIndex(4, 1)
	procExitType := typeIndex(1, 0)

	funcIndexByName := make(map[string]uint32, len(obj.Functions))
	returnSlotsByName := make(map[string]int, len(obj.Functions))
	funcTypeIdx := make([]uint32, 0, len(obj.Functions)+1)
	for i, fn := range obj.Functions {
		funcIndexByName[fn.Name] = uint32(importCount + i)
		returnSlotsByName[fn.Name] = fn.ReturnSlots
		funcTypeIdx = append(funcTypeIdx, typeIndex(fn.ParamSlots, fn.ReturnSlots))
	}
	mainFuncIdx, ok := funcIndexByName[mainName]
	if !ok {
		return nil, fmt.Errorf("wasm backend: entry function '%s' not found", mainName)
	}
	if returnSlotsByName[mainName] != 1 {
		return nil, fmt.Errorf("wasm backend: entry function '%s' must return exactly 1 slot, got %d", mainName, returnSlotsByName[mainName])
	}
	startTypeIdx := typeIndex(0, 0)
	startFuncIdx := uint32(importCount + len(obj.Functions))
	funcTypeIdx = append(funcTypeIdx, startTypeIdx)

	data := newDataBuilder()
	codeBodies := make([][]byte, 0, len(obj.Functions)+1)
	const heapGlobalIndex = uint32(0)
	for _, fn := range obj.Functions {
		body, err := compileFunction(fn, data, funcIndexByName, fdWriteImport, heapGlobalIndex)
		if err != nil {
			return nil, err
		}
		codeBodies = append(codeBodies, body)
	}
	startBody := compileStartFunction(mainFuncIdx, procExitImport)
	codeBodies = append(codeBodies, startBody)

	maxUsed := data.maxUsed()
	if reserved := nwrittenPtr + 4; reserved > maxUsed {
		maxUsed = reserved
	}
	heapBase, err := alignedWASMHeapBase(maxUsed)
	if err != nil {
		return nil, err
	}
	memoryMinPages := wasmMemoryMinPagesForBytes(heapBase)

	var module bytes.Buffer
	module.Write([]byte{0x00, 0x61, 0x73, 0x6d}) // \0asm
	module.Write([]byte{0x01, 0x00, 0x00, 0x00}) // version 1

	writeSection(&module, 1, func(sec *bytes.Buffer) {
		writeULEB(sec, uint32(len(typeEntries)))
		for _, t := range typeEntries {
			sec.WriteByte(0x60)
			writeULEB(sec, uint32(t.paramCount))
			for i := 0; i < t.paramCount; i++ {
				sec.WriteByte(0x7f) // i32
			}
			writeULEB(sec, uint32(t.returnCount))
			for i := 0; i < t.returnCount; i++ {
				sec.WriteByte(0x7f) // i32
			}
		}
	})

	writeSection(&module, 2, func(sec *bytes.Buffer) {
		writeULEB(sec, 2)

		writeName(sec, "wasi_snapshot_preview1")
		writeName(sec, "fd_write")
		sec.WriteByte(0x00) // import kind: func
		writeULEB(sec, fdWriteType)

		writeName(sec, "wasi_snapshot_preview1")
		writeName(sec, "proc_exit")
		sec.WriteByte(0x00) // import kind: func
		writeULEB(sec, procExitType)
	})

	writeSection(&module, 3, func(sec *bytes.Buffer) {
		writeULEB(sec, uint32(len(funcTypeIdx)))
		for _, idx := range funcTypeIdx {
			writeULEB(sec, idx)
		}
	})

	writeSection(&module, 5, func(sec *bytes.Buffer) {
		writeULEB(sec, 1)   // one memory
		sec.WriteByte(0x00) // limits: min only
		writeULEB(sec, memoryMinPages)
	})

	writeSection(&module, 6, func(sec *bytes.Buffer) {
		writeULEB(sec, uint32(1+obj.GlobalSlots)) // heap plus lowered global slots
		sec.WriteByte(0x7f)
		sec.WriteByte(0x01) // mutable
		writeI32Const(sec, int32(heapBase))
		sec.WriteByte(0x0b) // end init expr
		for i := 0; i < obj.GlobalSlots; i++ {
			sec.WriteByte(0x7f)
			sec.WriteByte(0x01) // mutable
			init := int32(0)
			if i < len(obj.GlobalInits) {
				init = obj.GlobalInits[i]
			}
			writeI32Const(sec, init)
			sec.WriteByte(0x0b) // end init expr
		}
	})

	writeSection(&module, 7, func(sec *bytes.Buffer) {
		writeULEB(sec, 2)

		writeName(sec, "memory")
		sec.WriteByte(0x02) // export kind: memory
		writeULEB(sec, 0)

		writeName(sec, "_start")
		sec.WriteByte(0x00) // export kind: func
		writeULEB(sec, startFuncIdx)
	})

	writeSection(&module, 10, func(sec *bytes.Buffer) {
		writeULEB(sec, uint32(len(codeBodies)))
		for _, body := range codeBodies {
			writeULEB(sec, uint32(len(body)))
			sec.Write(body)
		}
	})

	if len(data.bytes) > 0 {
		writeSection(&module, 11, func(sec *bytes.Buffer) {
			writeULEB(sec, 1)   // one segment
			sec.WriteByte(0x00) // active segment for memidx 0
			sec.WriteByte(0x41) // i32.const
			writeULEB(sec, dataBase)
			sec.WriteByte(0x0b) // end expr
			writeULEB(sec, uint32(len(data.bytes)))
			sec.Write(data.bytes)
		})
	}

	return module.Bytes(), nil
}

type wasmFuncType struct {
	paramCount  int
	returnCount int
}

type dataBuilder struct {
	bytes []byte
	seen  map[string]uint32
}

func newDataBuilder() *dataBuilder {
	return &dataBuilder{seen: make(map[string]uint32)}
}

func (d *dataBuilder) addString(raw []byte) uint32 {
	key := string(raw)
	if off, ok := d.seen[key]; ok {
		return off
	}
	off := uint32(len(d.bytes))
	d.bytes = append(d.bytes, raw...)
	d.seen[key] = off
	return off
}

func (d *dataBuilder) maxUsed() uint32 {
	return dataBase + uint32(len(d.bytes))
}

func alignedWASMHeapBase(maxUsed uint32) (uint32, error) {
	const mask = wasmHeapAlign - 1
	if maxUsed > ^uint32(0)-mask {
		return 0, fmt.Errorf("wasm backend: static data exceeds addressable heap layout")
	}
	return (maxUsed + mask) &^ mask, nil
}

func wasmMemoryMinPagesForBytes(used uint32) uint32 {
	pages := (uint64(used) + uint64(wasmPageSize) - 1) / uint64(wasmPageSize)
	if pages == 0 {
		return 1
	}
	return uint32(pages)
}

func compileFunction(fn Function, data *dataBuilder, funcIndexByName map[string]uint32, fdWriteImport int, heapGlobalIndex uint32) ([]byte, error) {
	if fn.LocalSlots < fn.ParamSlots {
		return nil, fmt.Errorf("wasm backend: function '%s' has invalid local slots", fn.Name)
	}

	baseLocals := fn.LocalSlots - fn.ParamSlots
	tempPtr := fn.LocalSlots
	tempLen := fn.LocalSlots + 1
	tempIdx := fn.LocalSlots + 2
	tempVal := fn.LocalSlots + 3
	tempByteLen := fn.LocalSlots + 4
	pcLocal := fn.LocalSlots + 5
	extraLocals := 6
	localDeclCount := baseLocals + extraLocals

	var body bytes.Buffer
	if localDeclCount > 0 {
		writeULEB(&body, 1) // one local group
		writeULEB(&body, uint32(localDeclCount))
		body.WriteByte(0x7f) // i32
	} else {
		writeULEB(&body, 0)
	}

	hasControlFlow := false
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel || instr.Kind == ir.IRJmp || instr.Kind == ir.IRJmpIfZero {
			hasControlFlow = true
			break
		}
	}

	if hasControlFlow {
		return compileFunctionWithControlFlow(fn, data, funcIndexByName, fdWriteImport, heapGlobalIndex, tempPtr, tempLen, tempIdx, tempVal, tempByteLen, pcLocal, &body)
	}

	stackDepth := 0
	terminated := false
	pop := func(n int, opname string) error {
		if stackDepth < n {
			return fmt.Errorf("wasm backend: stack underflow in '%s' (%s)", fn.Name, opname)
		}
		stackDepth -= n
		return nil
	}
	push := func(n int) { stackDepth += n }

	for _, instr := range fn.Instrs {
		switch instr.Kind {
		case ir.IRStrLit:
			dataOff := data.addString(instr.Str)
			writeI32Const(&body, int32(dataBase+dataOff))
			writeI32Const(&body, int32(len(instr.Str)))
			push(2)
		case ir.IRConstI32:
			writeI32Const(&body, instr.Imm)
			push(1)
		case ir.IRLoadLocal:
			body.WriteByte(0x20) // local.get
			writeULEB(&body, uint32(instr.Local))
			push(1)
		case ir.IRStoreLocal:
			if err := pop(1, "store_local"); err != nil {
				return nil, err
			}
			body.WriteByte(0x21) // local.set
			writeULEB(&body, uint32(instr.Local))
		case ir.IRLoadGlobal:
			globalIndex, err := wasmDataGlobalIndex(fn.Name, heapGlobalIndex, instr.Local)
			if err != nil {
				return nil, err
			}
			body.WriteByte(0x23) // global.get
			writeULEB(&body, globalIndex)
			push(1)
		case ir.IRStoreGlobal:
			if err := pop(1, "store_global"); err != nil {
				return nil, err
			}
			globalIndex, err := wasmDataGlobalIndex(fn.Name, heapGlobalIndex, instr.Local)
			if err != nil {
				return nil, err
			}
			body.WriteByte(0x24) // global.set
			writeULEB(&body, globalIndex)
		case ir.IRAddI32:
			if err := pop(2, "add_i32"); err != nil {
				return nil, err
			}
			body.WriteByte(0x6a) // i32.add
			push(1)
		case ir.IRSubI32:
			if err := pop(2, "sub_i32"); err != nil {
				return nil, err
			}
			body.WriteByte(0x6b) // i32.sub
			push(1)
		case ir.IRNegI32:
			if err := pop(1, "neg_i32"); err != nil {
				return nil, err
			}
			writeI32Const(&body, -1)
			body.WriteByte(0x6c) // i32.mul
			push(1)
		case ir.IRMulI32:
			if err := pop(2, "mul_i32"); err != nil {
				return nil, err
			}
			body.WriteByte(0x6c) // i32.mul
			push(1)
		case ir.IRDivI32:
			if err := pop(2, "div_i32"); err != nil {
				return nil, err
			}
			body.WriteByte(0x6d) // i32.div_s
			push(1)
		case ir.IRModI32:
			if err := pop(2, "mod_i32"); err != nil {
				return nil, err
			}
			body.WriteByte(0x6f) // i32.rem_s
			push(1)
		case ir.IRCmpEqI32:
			if err := pop(2, "cmp_eq_i32"); err != nil {
				return nil, err
			}
			body.WriteByte(0x46) // i32.eq
			push(1)
		case ir.IRCmpLtI32:
			if err := pop(2, "cmp_lt_i32"); err != nil {
				return nil, err
			}
			body.WriteByte(0x48) // i32.lt_s
			push(1)
		case ir.IRCmpGtI32:
			if err := pop(2, "cmp_gt_i32"); err != nil {
				return nil, err
			}
			body.WriteByte(0x4a) // i32.gt_s
			push(1)
		case ir.IRCmpGeI32:
			if err := pop(2, "cmp_ge_i32"); err != nil {
				return nil, err
			}
			body.WriteByte(0x4e) // i32.ge_s
			push(1)
		case ir.IRCmpLeI32:
			if err := pop(2, "cmp_le_i32"); err != nil {
				return nil, err
			}
			body.WriteByte(0x4c) // i32.le_s
			push(1)
		case ir.IRCmpNeI32:
			if err := pop(2, "cmp_ne_i32"); err != nil {
				return nil, err
			}
			body.WriteByte(0x47) // i32.ne
			push(1)
		case ir.IRCall:
			if err := pop(instr.ArgSlots, "call"); err != nil {
				return nil, err
			}
			target, ok := funcIndexByName[instr.Name]
			if !ok {
				return nil, fmt.Errorf("wasm backend: function '%s' calls unsupported symbol '%s'", fn.Name, instr.Name)
			}
			body.WriteByte(0x10) // call
			writeULEB(&body, target)
			push(instr.RetSlots)
		case ir.IRSymAddr:
			writeI32Const(&body, int32(wasmSymbolToken(instr.Name)))
			push(1)
		case ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32:
			if err := pop(1, "make_slice"); err != nil {
				return nil, err
			}
			body.WriteByte(0x21) // local.set tempLen
			writeULEB(&body, uint32(tempLen))
			body.WriteByte(0x20) // local.get tempLen
			writeULEB(&body, uint32(tempLen))
			switch instr.Kind {
			case ir.IRMakeSliceU16:
				writeI32Const(&body, 1)
				body.WriteByte(0x74) // i32.shl
			case ir.IRMakeSliceI32:
				writeI32Const(&body, 2)
				body.WriteByte(0x74) // i32.shl
			}
			body.WriteByte(0x21) // local.set tempByteLen
			writeULEB(&body, uint32(tempByteLen))
			body.WriteByte(0x23) // global.get heap
			writeULEB(&body, heapGlobalIndex)
			body.WriteByte(0x21) // local.set tempPtr
			writeULEB(&body, uint32(tempPtr))
			body.WriteByte(0x23) // global.get heap
			writeULEB(&body, heapGlobalIndex)
			body.WriteByte(0x20) // local.get tempByteLen
			writeULEB(&body, uint32(tempByteLen))
			body.WriteByte(0x6a) // i32.add
			body.WriteByte(0x24) // global.set heap
			writeULEB(&body, heapGlobalIndex)
			body.WriteByte(0x20) // local.get tempPtr
			writeULEB(&body, uint32(tempPtr))
			body.WriteByte(0x20) // local.get tempLen
			writeULEB(&body, uint32(tempLen))
			push(2)
		case ir.IRIslandNew:
			if err := pop(1, "island_new"); err != nil {
				return nil, err
			}
			body.WriteByte(0x21) // local.set tempByteLen
			writeULEB(&body, uint32(tempByteLen))
			body.WriteByte(0x23) // global.get heap
			writeULEB(&body, heapGlobalIndex)
			body.WriteByte(0x21) // local.set tempPtr
			writeULEB(&body, uint32(tempPtr))
			body.WriteByte(0x23) // global.get heap
			writeULEB(&body, heapGlobalIndex)
			body.WriteByte(0x20) // local.get tempByteLen
			writeULEB(&body, uint32(tempByteLen))
			body.WriteByte(0x6a) // i32.add
			body.WriteByte(0x24) // global.set heap
			writeULEB(&body, heapGlobalIndex)
			body.WriteByte(0x20) // local.get tempPtr
			writeULEB(&body, uint32(tempPtr))
			push(1)
		case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
			if err := pop(2, "island_make_slice"); err != nil {
				return nil, err
			}
			body.WriteByte(0x21) // local.set tempLen
			writeULEB(&body, uint32(tempLen))
			body.WriteByte(0x21) // local.set tempPtr (discard island handle)
			writeULEB(&body, uint32(tempPtr))
			body.WriteByte(0x20) // local.get tempLen
			writeULEB(&body, uint32(tempLen))
			switch instr.Kind {
			case ir.IRIslandMakeSliceU16:
				writeI32Const(&body, 1)
				body.WriteByte(0x74) // i32.shl
			case ir.IRIslandMakeSliceI32:
				writeI32Const(&body, 2)
				body.WriteByte(0x74) // i32.shl
			}
			body.WriteByte(0x21) // local.set tempByteLen
			writeULEB(&body, uint32(tempByteLen))
			body.WriteByte(0x23) // global.get heap
			writeULEB(&body, heapGlobalIndex)
			body.WriteByte(0x21) // local.set tempPtr
			writeULEB(&body, uint32(tempPtr))
			body.WriteByte(0x23) // global.get heap
			writeULEB(&body, heapGlobalIndex)
			body.WriteByte(0x20) // local.get tempByteLen
			writeULEB(&body, uint32(tempByteLen))
			body.WriteByte(0x6a) // i32.add
			body.WriteByte(0x24) // global.set heap
			writeULEB(&body, heapGlobalIndex)
			body.WriteByte(0x20) // local.get tempPtr
			writeULEB(&body, uint32(tempPtr))
			body.WriteByte(0x20) // local.get tempLen
			writeULEB(&body, uint32(tempLen))
			push(2)
		case ir.IRIslandFree:
			if err := pop(1, "island_free"); err != nil {
				return nil, err
			}
		case ir.IRIndexLoadI32, ir.IRIndexLoadU8, ir.IRIndexLoadU16:
			if err := pop(3, "index_load"); err != nil {
				return nil, err
			}
			body.WriteByte(0x21) // local.set tempIdx
			writeULEB(&body, uint32(tempIdx))
			body.WriteByte(0x21) // local.set tempLen
			writeULEB(&body, uint32(tempLen))
			body.WriteByte(0x21) // local.set tempPtr
			writeULEB(&body, uint32(tempPtr))
			body.WriteByte(0x20) // local.get tempIdx
			writeULEB(&body, uint32(tempIdx))
			body.WriteByte(0x20) // local.get tempLen
			writeULEB(&body, uint32(tempLen))
			body.WriteByte(0x4f) // i32.ge_u
			body.WriteByte(0x04) // if
			body.WriteByte(0x40) // blocktype empty
			body.WriteByte(0x00) // unreachable
			body.WriteByte(0x0b) // end
			body.WriteByte(0x20) // local.get tempPtr
			writeULEB(&body, uint32(tempPtr))
			body.WriteByte(0x20) // local.get tempIdx
			writeULEB(&body, uint32(tempIdx))
			switch instr.Kind {
			case ir.IRIndexLoadI32:
				writeI32Const(&body, 2)
				body.WriteByte(0x74) // i32.shl
			case ir.IRIndexLoadU16:
				writeI32Const(&body, 1)
				body.WriteByte(0x74) // i32.shl
			}
			body.WriteByte(0x6a) // i32.add
			switch instr.Kind {
			case ir.IRIndexLoadI32:
				body.WriteByte(0x28) // i32.load
				writeULEB(&body, 2)
				writeULEB(&body, 0)
			case ir.IRIndexLoadU16:
				body.WriteByte(0x2f) // i32.load16_u
				writeULEB(&body, 1)
				writeULEB(&body, 0)
			default:
				body.WriteByte(0x2d) // i32.load8_u
				writeULEB(&body, 0)
				writeULEB(&body, 0)
			}
			push(1)
		case ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
			if err := pop(4, "index_store"); err != nil {
				return nil, err
			}
			body.WriteByte(0x21) // local.set tempVal
			writeULEB(&body, uint32(tempVal))
			body.WriteByte(0x21) // local.set tempIdx
			writeULEB(&body, uint32(tempIdx))
			body.WriteByte(0x21) // local.set tempLen
			writeULEB(&body, uint32(tempLen))
			body.WriteByte(0x21) // local.set tempPtr
			writeULEB(&body, uint32(tempPtr))
			body.WriteByte(0x20) // local.get tempIdx
			writeULEB(&body, uint32(tempIdx))
			body.WriteByte(0x20) // local.get tempLen
			writeULEB(&body, uint32(tempLen))
			body.WriteByte(0x4f) // i32.ge_u
			body.WriteByte(0x04) // if
			body.WriteByte(0x40) // blocktype empty
			body.WriteByte(0x00) // unreachable
			body.WriteByte(0x0b) // end
			body.WriteByte(0x20) // local.get tempPtr
			writeULEB(&body, uint32(tempPtr))
			body.WriteByte(0x20) // local.get tempIdx
			writeULEB(&body, uint32(tempIdx))
			switch instr.Kind {
			case ir.IRIndexStoreI32:
				writeI32Const(&body, 2)
				body.WriteByte(0x74) // i32.shl
			case ir.IRIndexStoreU16:
				writeI32Const(&body, 1)
				body.WriteByte(0x74) // i32.shl
			}
			body.WriteByte(0x6a) // i32.add
			body.WriteByte(0x20) // local.get tempVal
			writeULEB(&body, uint32(tempVal))
			switch instr.Kind {
			case ir.IRIndexStoreI32:
				body.WriteByte(0x36) // i32.store
				writeULEB(&body, 2)
				writeULEB(&body, 0)
			case ir.IRIndexStoreU16:
				body.WriteByte(0x3b) // i32.store16
				writeULEB(&body, 1)
				writeULEB(&body, 0)
			default:
				body.WriteByte(0x3a) // i32.store8
				writeULEB(&body, 0)
				writeULEB(&body, 0)
			}
		case ir.IRLabel, ir.IRJmp, ir.IRJmpIfZero:
			return nil, fmt.Errorf("wasm backend: control-flow IR in linear mode for function '%s'", fn.Name)
		case ir.IRWrite:
			if err := pop(2, "write"); err != nil {
				return nil, err
			}
			body.WriteByte(0x21) // local.set tempLen
			writeULEB(&body, uint32(tempLen))
			body.WriteByte(0x21) // local.set tempPtr
			writeULEB(&body, uint32(tempPtr))

			writeI32Const(&body, int32(iovecAddr))
			body.WriteByte(0x20) // local.get tempPtr
			writeULEB(&body, uint32(tempPtr))
			body.WriteByte(0x36) // i32.store
			writeULEB(&body, 2)  // align=4-byte
			writeULEB(&body, 0)  // offset

			writeI32Const(&body, int32(iovecAddr+4))
			body.WriteByte(0x20) // local.get tempLen
			writeULEB(&body, uint32(tempLen))
			body.WriteByte(0x36) // i32.store
			writeULEB(&body, 2)
			writeULEB(&body, 0)

			writeI32Const(&body, 1) // stdout fd
			writeI32Const(&body, int32(iovecAddr))
			writeI32Const(&body, 1)
			writeI32Const(&body, int32(nwrittenPtr))
			body.WriteByte(0x10) // call fd_write
			writeULEB(&body, uint32(fdWriteImport))
			body.WriteByte(0x1a) // drop errno
		case ir.IRReturn:
			if err := pop(fn.ReturnSlots, "return"); err != nil {
				return nil, err
			}
			body.WriteByte(0x0f) // return
			stackDepth = 0
			terminated = true
		default:
			return nil, wasmUnsupportedInstrError(fn.Name, instr.Kind)
		}
	}

	if !terminated {
		if fn.ReturnSlots == 0 {
			body.WriteByte(0x0f) // return
		} else if stackDepth == fn.ReturnSlots {
			body.WriteByte(0x0f) // return with stack value(s)
		} else {
			return nil, fmt.Errorf("wasm backend: function '%s' ended with stack depth %d (want %d)", fn.Name, stackDepth, fn.ReturnSlots)
		}
	}
	body.WriteByte(0x0b) // end

	return body.Bytes(), nil
}

func compileFunctionWithControlFlow(fn Function, data *dataBuilder, funcIndexByName map[string]uint32, fdWriteImport int, heapGlobalIndex uint32, tempPtr int, tempLen int, tempIdx int, tempVal int, tempByteLen int, pcLocal int, body *bytes.Buffer) ([]byte, error) {
	labels, heights, err := verifyControlFlowStackModel(fn)
	if err != nil {
		return nil, err
	}
	starts := map[int]struct{}{0: {}}
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel {
			starts[i] = struct{}{}
		}
		if (instr.Kind == ir.IRJmp || instr.Kind == ir.IRJmpIfZero || instr.Kind == ir.IRReturn) && i+1 < len(fn.Instrs) {
			starts[i+1] = struct{}{}
		}
	}
	var blockStarts []int
	for idx := range starts {
		blockStarts = append(blockStarts, idx)
	}
	sort.Ints(blockStarts)
	type blockRange struct {
		start int
		end   int
	}
	blocks := make([]blockRange, 0, len(blockStarts))
	blockByStart := make(map[int]int, len(blockStarts))
	for i, start := range blockStarts {
		end := len(fn.Instrs)
		if i+1 < len(blockStarts) {
			end = blockStarts[i+1]
		}
		blockByStart[start] = len(blocks)
		blocks = append(blocks, blockRange{start: start, end: end})
	}
	labelToBlock := make(map[int]int, len(labels))
	for label, idx := range labels {
		bid, ok := blockByStart[idx]
		if !ok {
			return nil, fmt.Errorf("wasm backend: internal control-flow block mapping failure in '%s'", fn.Name)
		}
		labelToBlock[label] = bid
		if heights[idx] != 0 {
			return nil, fmt.Errorf("wasm backend: unsupported non-zero stack at label %d in function '%s'", label, fn.Name)
		}
	}

	writeI32Const(body, 0)
	body.WriteByte(0x21) // local.set pc
	writeULEB(body, uint32(pcLocal))
	body.WriteByte(0x02) // block
	body.WriteByte(0x40)
	body.WriteByte(0x03) // loop
	body.WriteByte(0x40)

	for bi, block := range blocks {
		body.WriteByte(0x20) // local.get pc
		writeULEB(body, uint32(pcLocal))
		writeI32Const(body, int32(bi))
		body.WriteByte(0x46) // i32.eq
		body.WriteByte(0x04) // if
		body.WriteByte(0x40)

		stackDepth := 0
		terminated := false
		for i := block.start; i < block.end; i++ {
			instr := fn.Instrs[i]
			switch instr.Kind {
			case ir.IRLabel:
				continue
			case ir.IRJmp:
				if stackDepth != 0 {
					return nil, fmt.Errorf("wasm backend: unsupported non-zero stack at jump in function '%s'", fn.Name)
				}
				target, ok := labelToBlock[instr.Label]
				if !ok {
					return nil, fmt.Errorf("wasm backend: unknown jump label %d in function '%s'", instr.Label, fn.Name)
				}
				writeI32Const(body, int32(target))
				body.WriteByte(0x21) // local.set pc
				writeULEB(body, uint32(pcLocal))
				body.WriteByte(0x0c) // br loop
				writeULEB(body, 1)
				terminated = true
			case ir.IRJmpIfZero:
				if stackDepth < 1 {
					return nil, fmt.Errorf("wasm backend: stack underflow in '%s' (jmp_if_zero)", fn.Name)
				}
				stackDepth--
				if stackDepth != 0 {
					return nil, fmt.Errorf("wasm backend: unsupported non-zero stack after conditional jump in function '%s'", fn.Name)
				}
				target, ok := labelToBlock[instr.Label]
				if !ok {
					return nil, fmt.Errorf("wasm backend: unknown jump label %d in function '%s'", instr.Label, fn.Name)
				}
				body.WriteByte(0x45) // i32.eqz
				body.WriteByte(0x04) // if
				body.WriteByte(0x40)
				writeI32Const(body, int32(target))
				body.WriteByte(0x21) // local.set pc
				writeULEB(body, uint32(pcLocal))
				body.WriteByte(0x0c) // br loop (inside nested if)
				writeULEB(body, 2)
				body.WriteByte(0x0b) // end nested if
				nextBlock := bi + 1
				if nextBlock >= len(blocks) {
					return nil, fmt.Errorf("wasm backend: conditional branch falls off end in function '%s'", fn.Name)
				}
				writeI32Const(body, int32(nextBlock))
				body.WriteByte(0x21) // local.set pc
				writeULEB(body, uint32(pcLocal))
				body.WriteByte(0x0c) // br loop
				writeULEB(body, 1)
				terminated = true
			case ir.IRReturn:
				if stackDepth != fn.ReturnSlots {
					return nil, fmt.Errorf("wasm backend: return stack mismatch in '%s': got %d want %d", fn.Name, stackDepth, fn.ReturnSlots)
				}
				body.WriteByte(0x0f) // return
				stackDepth = 0
				terminated = true
			default:
				nextDepth, err := emitWASINonControlInstr(body, fn, instr, data, funcIndexByName, fdWriteImport, heapGlobalIndex, tempPtr, tempLen, tempIdx, tempVal, tempByteLen, stackDepth)
				if err != nil {
					return nil, err
				}
				stackDepth = nextDepth
			}
			if terminated {
				break
			}
		}
		if !terminated {
			if stackDepth != 0 {
				return nil, fmt.Errorf("wasm backend: unsupported non-zero stack at block fallthrough in function '%s'", fn.Name)
			}
			if bi+1 >= len(blocks) {
				if fn.ReturnSlots == 0 {
					body.WriteByte(0x0f) // return
				} else {
					return nil, fmt.Errorf("wasm backend: function '%s' falls off end in control-flow mode", fn.Name)
				}
			} else {
				writeI32Const(body, int32(bi+1))
				body.WriteByte(0x21) // local.set pc
				writeULEB(body, uint32(pcLocal))
				body.WriteByte(0x0c) // br loop
				writeULEB(body, 1)
			}
		}
		body.WriteByte(0x0b) // end block-if
	}
	body.WriteByte(0x00) // unreachable (invalid pc)
	body.WriteByte(0x0b) // end loop
	body.WriteByte(0x0b) // end block
	body.WriteByte(0x00) // unreachable function fallthrough for multi-result funcs
	body.WriteByte(0x0b) // end func
	return body.Bytes(), nil
}

func emitWASINonControlInstr(body *bytes.Buffer, fn Function, instr ir.IRInstr, data *dataBuilder, funcIndexByName map[string]uint32, fdWriteImport int, heapGlobalIndex uint32, tempPtr int, tempLen int, tempIdx int, tempVal int, tempByteLen int, stackDepth int) (int, error) {
	pop := func(n int, opname string) error {
		if stackDepth < n {
			return fmt.Errorf("wasm backend: stack underflow in '%s' (%s)", fn.Name, opname)
		}
		stackDepth -= n
		return nil
	}
	push := func(n int) { stackDepth += n }

	switch instr.Kind {
	case ir.IRStrLit:
		dataOff := data.addString(instr.Str)
		writeI32Const(body, int32(dataBase+dataOff))
		writeI32Const(body, int32(len(instr.Str)))
		push(2)
	case ir.IRConstI32:
		writeI32Const(body, instr.Imm)
		push(1)
	case ir.IRLoadLocal:
		body.WriteByte(0x20)
		writeULEB(body, uint32(instr.Local))
		push(1)
	case ir.IRStoreLocal:
		if err := pop(1, "store_local"); err != nil {
			return 0, err
		}
		body.WriteByte(0x21)
		writeULEB(body, uint32(instr.Local))
	case ir.IRLoadGlobal:
		globalIndex, err := wasmDataGlobalIndex(fn.Name, heapGlobalIndex, instr.Local)
		if err != nil {
			return 0, err
		}
		body.WriteByte(0x23)
		writeULEB(body, globalIndex)
		push(1)
	case ir.IRStoreGlobal:
		if err := pop(1, "store_global"); err != nil {
			return 0, err
		}
		globalIndex, err := wasmDataGlobalIndex(fn.Name, heapGlobalIndex, instr.Local)
		if err != nil {
			return 0, err
		}
		body.WriteByte(0x24)
		writeULEB(body, globalIndex)
	case ir.IRAddI32:
		if err := pop(2, "add_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x6a)
		push(1)
	case ir.IRSubI32:
		if err := pop(2, "sub_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x6b)
		push(1)
	case ir.IRNegI32:
		if err := pop(1, "neg_i32"); err != nil {
			return 0, err
		}
		writeI32Const(body, -1)
		body.WriteByte(0x6c)
		push(1)
	case ir.IRMulI32:
		if err := pop(2, "mul_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x6c)
		push(1)
	case ir.IRDivI32:
		if err := pop(2, "div_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x6d)
		push(1)
	case ir.IRModI32:
		if err := pop(2, "mod_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x6f)
		push(1)
	case ir.IRCmpEqI32:
		if err := pop(2, "cmp_eq_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x46)
		push(1)
	case ir.IRCmpLtI32:
		if err := pop(2, "cmp_lt_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x48)
		push(1)
	case ir.IRCmpGtI32:
		if err := pop(2, "cmp_gt_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x4a)
		push(1)
	case ir.IRCmpGeI32:
		if err := pop(2, "cmp_ge_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x4e)
		push(1)
	case ir.IRCmpLeI32:
		if err := pop(2, "cmp_le_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x4c)
		push(1)
	case ir.IRCmpNeI32:
		if err := pop(2, "cmp_ne_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x47)
		push(1)
	case ir.IRCall:
		if err := pop(instr.ArgSlots, "call"); err != nil {
			return 0, err
		}
		target, ok := funcIndexByName[instr.Name]
		if !ok {
			return 0, fmt.Errorf("wasm backend: function '%s' calls unsupported symbol '%s'", fn.Name, instr.Name)
		}
		body.WriteByte(0x10)
		writeULEB(body, target)
		push(instr.RetSlots)
	case ir.IRSymAddr:
		writeI32Const(body, int32(wasmSymbolToken(instr.Name)))
		push(1)
	case ir.IRWrite:
		if err := pop(2, "write"); err != nil {
			return 0, err
		}
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempLen))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempPtr))
		writeI32Const(body, int32(iovecAddr))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x36)
		writeULEB(body, 2)
		writeULEB(body, 0)
		writeI32Const(body, int32(iovecAddr+4))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempLen))
		body.WriteByte(0x36)
		writeULEB(body, 2)
		writeULEB(body, 0)
		writeI32Const(body, 1)
		writeI32Const(body, int32(iovecAddr))
		writeI32Const(body, 1)
		writeI32Const(body, int32(nwrittenPtr))
		body.WriteByte(0x10)
		writeULEB(body, uint32(fdWriteImport))
		body.WriteByte(0x1a) // drop errno
	case ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32:
		if err := pop(1, "make_slice"); err != nil {
			return 0, err
		}
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempLen))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempLen))
		switch instr.Kind {
		case ir.IRMakeSliceU16:
			writeI32Const(body, 1)
			body.WriteByte(0x74)
		case ir.IRMakeSliceI32:
			writeI32Const(body, 2)
			body.WriteByte(0x74)
		}
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempByteLen))
		body.WriteByte(0x23)
		writeULEB(body, heapGlobalIndex)
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x23)
		writeULEB(body, heapGlobalIndex)
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempByteLen))
		body.WriteByte(0x6a)
		body.WriteByte(0x24)
		writeULEB(body, heapGlobalIndex)
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempLen))
		push(2)
	case ir.IRIslandNew:
		if err := pop(1, "island_new"); err != nil {
			return 0, err
		}
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempByteLen))
		body.WriteByte(0x23)
		writeULEB(body, heapGlobalIndex)
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x23)
		writeULEB(body, heapGlobalIndex)
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempByteLen))
		body.WriteByte(0x6a)
		body.WriteByte(0x24)
		writeULEB(body, heapGlobalIndex)
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempPtr))
		push(1)
	case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
		if err := pop(2, "island_make_slice"); err != nil {
			return 0, err
		}
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempLen))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempLen))
		switch instr.Kind {
		case ir.IRIslandMakeSliceU16:
			writeI32Const(body, 1)
			body.WriteByte(0x74)
		case ir.IRIslandMakeSliceI32:
			writeI32Const(body, 2)
			body.WriteByte(0x74)
		}
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempByteLen))
		body.WriteByte(0x23)
		writeULEB(body, heapGlobalIndex)
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x23)
		writeULEB(body, heapGlobalIndex)
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempByteLen))
		body.WriteByte(0x6a)
		body.WriteByte(0x24)
		writeULEB(body, heapGlobalIndex)
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempLen))
		push(2)
	case ir.IRIslandFree:
		if err := pop(1, "island_free"); err != nil {
			return 0, err
		}
	case ir.IRIndexLoadI32, ir.IRIndexLoadU8, ir.IRIndexLoadU16:
		if err := pop(3, "index_load"); err != nil {
			return 0, err
		}
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempIdx))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempLen))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempIdx))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempLen))
		body.WriteByte(0x4f)
		body.WriteByte(0x04)
		body.WriteByte(0x40)
		body.WriteByte(0x00)
		body.WriteByte(0x0b)
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempIdx))
		switch instr.Kind {
		case ir.IRIndexLoadI32:
			writeI32Const(body, 2)
			body.WriteByte(0x74)
		case ir.IRIndexLoadU16:
			writeI32Const(body, 1)
			body.WriteByte(0x74)
		}
		body.WriteByte(0x6a)
		switch instr.Kind {
		case ir.IRIndexLoadI32:
			body.WriteByte(0x28)
			writeULEB(body, 2)
			writeULEB(body, 0)
		case ir.IRIndexLoadU16:
			body.WriteByte(0x2f)
			writeULEB(body, 1)
			writeULEB(body, 0)
		default:
			body.WriteByte(0x2d)
			writeULEB(body, 0)
			writeULEB(body, 0)
		}
		push(1)
	case ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
		if err := pop(4, "index_store"); err != nil {
			return 0, err
		}
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempVal))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempIdx))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempLen))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempIdx))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempLen))
		body.WriteByte(0x4f)
		body.WriteByte(0x04)
		body.WriteByte(0x40)
		body.WriteByte(0x00)
		body.WriteByte(0x0b)
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempIdx))
		switch instr.Kind {
		case ir.IRIndexStoreI32:
			writeI32Const(body, 2)
			body.WriteByte(0x74)
		case ir.IRIndexStoreU16:
			writeI32Const(body, 1)
			body.WriteByte(0x74)
		}
		body.WriteByte(0x6a)
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempVal))
		switch instr.Kind {
		case ir.IRIndexStoreI32:
			body.WriteByte(0x36)
			writeULEB(body, 2)
			writeULEB(body, 0)
		case ir.IRIndexStoreU16:
			body.WriteByte(0x3b)
			writeULEB(body, 1)
			writeULEB(body, 0)
		default:
			body.WriteByte(0x3a)
			writeULEB(body, 0)
			writeULEB(body, 0)
		}
	default:
		return 0, wasmUnsupportedInstrError(fn.Name, instr.Kind)
	}
	return stackDepth, nil
}

func verifyControlFlowStackModel(fn Function) (map[int]int, []int, error) {
	labels := make(map[int]int)
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel {
			labels[instr.Label] = i
		}
	}
	heights := make([]int, len(fn.Instrs))
	seen := make([]bool, len(fn.Instrs))
	type state struct {
		idx    int
		height int
	}
	work := []state{{idx: 0, height: 0}}
	for len(work) > 0 {
		cur := work[len(work)-1]
		work = work[:len(work)-1]
		if cur.idx < 0 || cur.idx >= len(fn.Instrs) {
			continue
		}
		if fn.Instrs[cur.idx].Kind == ir.IRLabel && cur.height != 0 {
			return nil, nil, fmt.Errorf("wasm backend: unsupported non-zero stack at label %d in function '%s'", fn.Instrs[cur.idx].Label, fn.Name)
		}
		if seen[cur.idx] {
			if heights[cur.idx] != cur.height {
				return nil, nil, fmt.Errorf("wasm backend: inconsistent stack height at instr %d in '%s'", cur.idx, fn.Name)
			}
			continue
		}
		seen[cur.idx] = true
		heights[cur.idx] = cur.height
		pop, push, ok := wasmStackEffect(fn.Instrs[cur.idx])
		if !ok {
			return nil, nil, wasmUnsupportedInstrError(fn.Name, fn.Instrs[cur.idx].Kind)
		}
		if cur.height < pop {
			return nil, nil, fmt.Errorf("wasm backend: stack underflow at instr %d in '%s'", cur.idx, fn.Name)
		}
		nextHeight := cur.height - pop + push
		switch fn.Instrs[cur.idx].Kind {
		case ir.IRReturn:
			if cur.height != fn.ReturnSlots {
				return nil, nil, fmt.Errorf("wasm backend: return stack mismatch at instr %d in '%s'", cur.idx, fn.Name)
			}
		case ir.IRJmp:
			work = append(work, state{idx: labels[fn.Instrs[cur.idx].Label], height: nextHeight})
		case ir.IRJmpIfZero:
			work = append(work, state{idx: labels[fn.Instrs[cur.idx].Label], height: nextHeight})
			work = append(work, state{idx: cur.idx + 1, height: nextHeight})
		default:
			work = append(work, state{idx: cur.idx + 1, height: nextHeight})
		}
	}
	return labels, heights, nil
}

func wasmStackEffect(instr ir.IRInstr) (int, int, bool) {
	switch instr.Kind {
	case ir.IRWrite:
		return 2, 0, true
	case ir.IRStrLit:
		return 0, 2, true
	case ir.IRConstI32, ir.IRLoadLocal, ir.IRLoadGlobal, ir.IRSymAddr:
		return 0, 1, true
	case ir.IRStoreLocal, ir.IRStoreGlobal:
		return 1, 0, true
	case ir.IRAddI32, ir.IRSubI32, ir.IRCmpEqI32, ir.IRCmpLtI32,
		ir.IRMulI32, ir.IRDivI32, ir.IRModI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return 2, 1, true
	case ir.IRNegI32:
		return 1, 1, true
	case ir.IRCall:
		return instr.ArgSlots, instr.RetSlots, true
	case ir.IRLabel, ir.IRJmp:
		return 0, 0, true
	case ir.IRJmpIfZero:
		return 1, 0, true
	case ir.IRReturn:
		return 0, 0, true
	case ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32:
		return 1, 2, true
	case ir.IRIslandNew:
		return 1, 1, true
	case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
		return 2, 2, true
	case ir.IRIslandFree:
		return 1, 0, true
	case ir.IRIndexLoadI32, ir.IRIndexLoadU8, ir.IRIndexLoadU16:
		return 3, 1, true
	case ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
		return 4, 0, true
	case ir.IRMemReadI32Offset, ir.IRMemReadU8Offset, ir.IRMemReadPtrOffset:
		return 3, 1, true
	case ir.IRMemWriteI32Offset, ir.IRMemWriteU8Offset, ir.IRMemWritePtrOffset:
		return 4, 1, true
	default:
		return 0, 0, false
	}
}

func wasmUnsupportedInstrError(fnName string, kind ir.IRInstrKind) error {
	return fmt.Errorf("wasm backend: unsupported IR instruction %d in function '%s'", kind, fnName)
}

func wasmDataGlobalIndex(fnName string, heapGlobalIndex uint32, slot int) (uint32, error) {
	if slot < 0 {
		return 0, wasmNegativeGlobalSlotError(fnName, slot)
	}
	return heapGlobalIndex + 1 + uint32(slot), nil
}

func wasmGlobalInitializers(globalSlots int, dataPrefix [][]byte) ([]int32, error) {
	inits := make([]int32, globalSlots)
	for i := 0; i < globalSlots && i < len(dataPrefix); i++ {
		if len(dataPrefix[i]) == 0 {
			continue
		}
		if len(dataPrefix[i]) < 4 {
			return nil, fmt.Errorf("wasm backend: global data slot %d is %d bytes, want at least 4", i, len(dataPrefix[i]))
		}
		inits[i] = int32(binary.LittleEndian.Uint32(dataPrefix[i][:4]))
	}
	return inits, nil
}

func wasmNegativeGlobalSlotError(fnName string, slot int) error {
	return fmt.Errorf("wasm backend: negative global slot %d in function '%s'", slot, fnName)
}

func validateWasmObjectFunctions(funcs []Function) error {
	functionNames := make(map[string]struct{}, len(funcs))
	for _, fn := range funcs {
		if err := validateWasmFunctionMetadata(functionNames, fn.Name, fn.ParamSlots, fn.LocalSlots, fn.ReturnSlots); err != nil {
			return err
		}
	}
	return nil
}

func validateWasmFunctionMetadata(seen map[string]struct{}, name string, paramSlots int, localSlots int, returnSlots int) error {
	if name == "" {
		return fmt.Errorf("wasm backend: function name is empty")
	}
	if _, ok := seen[name]; ok {
		return fmt.Errorf("wasm backend: duplicate function '%s'", name)
	}
	seen[name] = struct{}{}
	if paramSlots < 0 || localSlots < 0 || returnSlots < 0 || localSlots < paramSlots {
		return fmt.Errorf("wasm backend: function '%s' has invalid slots params=%d locals=%d returns=%d", name, paramSlots, localSlots, returnSlots)
	}
	return nil
}

func validateWasmObjectGlobalSlots(obj *Object) error {
	if obj.GlobalSlots < 0 {
		return fmt.Errorf("wasm backend: invalid global slot count %d", obj.GlobalSlots)
	}
	for _, fn := range obj.Functions {
		for _, instr := range fn.Instrs {
			if instr.Kind != ir.IRLoadGlobal && instr.Kind != ir.IRStoreGlobal {
				continue
			}
			if instr.Local < 0 {
				return wasmNegativeGlobalSlotError(fn.Name, instr.Local)
			}
			if instr.Local >= obj.GlobalSlots {
				return fmt.Errorf("wasm backend: global slot %d in function '%s' exceeds object global slot count %d", instr.Local, fn.Name, obj.GlobalSlots)
			}
		}
	}
	return nil
}

func validateWasmObjectLabels(funcs []Function) error {
	for _, fn := range funcs {
		if err := validateWasmLabelMetadata(fn.Name, fn.Instrs); err != nil {
			return err
		}
	}
	return nil
}

func validateWasmLabelMetadata(fnName string, instrs []ir.IRInstr) error {
	labels := make(map[int]struct{})
	for _, instr := range instrs {
		if instr.Kind != ir.IRLabel {
			continue
		}
		if instr.Label < 0 {
			return fmt.Errorf("wasm backend: function '%s' negative label %d", fnName, instr.Label)
		}
		if _, exists := labels[instr.Label]; exists {
			return fmt.Errorf("wasm backend: function '%s' duplicate label %d", fnName, instr.Label)
		}
		labels[instr.Label] = struct{}{}
	}
	for _, instr := range instrs {
		if instr.Kind != ir.IRJmp && instr.Kind != ir.IRJmpIfZero {
			continue
		}
		if instr.Label < 0 {
			return fmt.Errorf("wasm backend: function '%s' negative label %d", fnName, instr.Label)
		}
		if _, ok := labels[instr.Label]; !ok {
			return fmt.Errorf("wasm backend: function '%s' unknown label %d", fnName, instr.Label)
		}
	}
	return nil
}

func validateWasmObjectLocalSlots(funcs []Function) error {
	for _, fn := range funcs {
		for _, instr := range fn.Instrs {
			if instr.Kind == ir.IRLoadLocal || instr.Kind == ir.IRStoreLocal {
				if err := validateWasmLocalSlot(fn.Name, fn.LocalSlots, instr.Local); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateWasmLocalSlot(fnName string, localSlots int, slot int) error {
	if slot < 0 || slot >= localSlots {
		return fmt.Errorf("wasm backend: function '%s' local slot %d out of bounds (locals=%d)", fnName, slot, localSlots)
	}
	return nil
}

func validateWasmObjectCalls(funcs []Function) error {
	functionSigs := make(map[string]wasmFunctionSignature, len(funcs))
	for _, fn := range funcs {
		functionSigs[fn.Name] = wasmFunctionSignature{ParamSlots: fn.ParamSlots, ReturnSlots: fn.ReturnSlots}
	}
	for _, fn := range funcs {
		for _, instr := range fn.Instrs {
			if instr.Kind == ir.IRCall {
				if err := validateWasmCallMetadata(fn.Name, instr, functionSigs); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateWasmCallMetadata(fnName string, instr ir.IRInstr, functionSigs map[string]wasmFunctionSignature) error {
	if instr.Name == "" {
		return fmt.Errorf("wasm backend: function '%s' call is missing target name", fnName)
	}
	if instr.ArgSlots < 0 || instr.RetSlots < 0 {
		return fmt.Errorf("wasm backend: function '%s' call %q has negative ABI slots args=%d rets=%d", fnName, instr.Name, instr.ArgSlots, instr.RetSlots)
	}
	sig, ok := functionSigs[instr.Name]
	if !ok {
		return fmt.Errorf("wasm backend: function '%s' calls unsupported symbol '%s'", fnName, instr.Name)
	}
	if instr.ArgSlots != sig.ParamSlots || instr.RetSlots != sig.ReturnSlots {
		return fmt.Errorf("wasm backend: function '%s' call %q ABI mismatch args=%d rets=%d want args=%d rets=%d", fnName, instr.Name, instr.ArgSlots, instr.RetSlots, sig.ParamSlots, sig.ReturnSlots)
	}
	return nil
}

func validateWasmObjectSymbolTokens(funcs []Function) error {
	symbolTokens := make(map[uint32]string)
	for _, fn := range funcs {
		for _, instr := range fn.Instrs {
			if instr.Kind == ir.IRSymAddr {
				if err := validateWasmSymbolToken(symbolTokens, instr.Name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateWasmSymbolToken(seen map[uint32]string, name string) error {
	if name == "" {
		return fmt.Errorf("wasm backend: symbol address is missing name")
	}
	token := wasmSymbolToken(name)
	if previous, ok := seen[token]; ok {
		if previous != name {
			return fmt.Errorf("wasm backend: symbol address token collision: %q and %q both map to 0x%08x", previous, name, token)
		}
		return nil
	}
	seen[token] = name
	return nil
}

var wasmSymbolTokenHash = func(name string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return h.Sum32()
}

func wasmSymbolToken(name string) uint32 {
	return wasmSymbolTokenHash(name)
}

func compileStartFunction(mainFuncIdx uint32, procExitImport int) []byte {
	var body bytes.Buffer
	writeULEB(&body, 1)
	writeULEB(&body, 1)
	body.WriteByte(0x7f) // local i32 rc

	body.WriteByte(0x10) // call main
	writeULEB(&body, mainFuncIdx)
	body.WriteByte(0x21) // local.set 0
	writeULEB(&body, 0)
	body.WriteByte(0x20) // local.get 0
	writeULEB(&body, 0)
	body.WriteByte(0x10) // call proc_exit
	writeULEB(&body, uint32(procExitImport))
	body.WriteByte(0x00) // unreachable
	body.WriteByte(0x0b) // end
	return body.Bytes()
}

func writeSection(dst *bytes.Buffer, id byte, fn func(*bytes.Buffer)) {
	var sec bytes.Buffer
	fn(&sec)
	dst.WriteByte(id)
	writeULEB(dst, uint32(sec.Len()))
	dst.Write(sec.Bytes())
}

func writeName(dst *bytes.Buffer, s string) {
	writeULEB(dst, uint32(len(s)))
	dst.WriteString(s)
}

func writeI32Const(dst *bytes.Buffer, v int32) {
	dst.WriteByte(0x41)
	writeSLEB32(dst, v)
}

func writeULEB(dst *bytes.Buffer, v uint32) {
	var tmp [binary.MaxVarintLen32]byte
	n := binary.PutUvarint(tmp[:], uint64(v))
	dst.Write(tmp[:n])
}

func writeSLEB32(dst *bytes.Buffer, v int32) {
	x := int64(v)
	for {
		b := byte(x & 0x7f)
		x >>= 7
		signSet := (b & 0x40) != 0
		done := (x == 0 && !signSet) || (x == -1 && signSet)
		if !done {
			b |= 0x80
		}
		dst.WriteByte(b)
		if done {
			return
		}
	}
}
