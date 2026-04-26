package wasm32_web

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"

	"tetra_language/compiler/internal/ir"
)

const (
	wasmPageSize = 65536
	dataBase     = uint32(0x1000)
)

type Function struct {
	Name        string
	ParamSlots  int
	LocalSlots  int
	ReturnSlots int
	Instrs      []ir.IRInstr
}

type Object struct {
	Functions []Function
	MainName  string
}

func CodegenObject(funcs []ir.IRFunc, mainName string) (*Object, error) {
	if len(funcs) == 0 {
		return nil, fmt.Errorf("wasm backend: no functions to compile")
	}
	out := make([]Function, 0, len(funcs))
	for _, fn := range funcs {
		if fn.ReturnSlots > 1 {
			return nil, fmt.Errorf("wasm backend: function '%s' return slots > 1 are not supported", fn.Name)
		}
		if fn.LocalSlots < fn.ParamSlots {
			return nil, fmt.Errorf("wasm backend: function '%s' has invalid slots", fn.Name)
		}
		out = append(out, Function{
			Name:        fn.Name,
			ParamSlots:  fn.ParamSlots,
			LocalSlots:  fn.LocalSlots,
			ReturnSlots: fn.ReturnSlots,
			Instrs:      fn.Instrs,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return &Object{Functions: out, MainName: mainName}, nil
}

func LinkObject(obj *Object) ([]byte, error) {
	if obj == nil {
		return nil, fmt.Errorf("wasm backend: missing object")
	}
	if len(obj.Functions) == 0 {
		return nil, fmt.Errorf("wasm backend: missing functions")
	}
	mainName := obj.MainName
	if mainName == "" {
		mainName = "main"
	}

	const (
		consoleLogImport = iota
		panicImport
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

	consoleLogType := typeIndex(2, 0)
	panicType := typeIndex(3, 0)

	funcIndexByName := make(map[string]uint32, len(obj.Functions))
	funcTypeIdx := make([]uint32, 0, len(obj.Functions))
	for i, fn := range obj.Functions {
		funcIndexByName[fn.Name] = uint32(importCount + i)
		funcTypeIdx = append(funcTypeIdx, typeIndex(fn.ParamSlots, fn.ReturnSlots))
	}
	mainFuncIdx, ok := funcIndexByName[mainName]
	if !ok {
		return nil, fmt.Errorf("wasm backend: entry function '%s' not found", mainName)
	}

	data := newDataBuilder()
	codeBodies := make([][]byte, 0, len(obj.Functions))
	for _, fn := range obj.Functions {
		body, err := compileFunction(fn, data, funcIndexByName, consoleLogImport)
		if err != nil {
			return nil, err
		}
		codeBodies = append(codeBodies, body)
	}

	memoryMinPages := uint32(1)
	maxUsed := data.maxUsed()
	if maxUsed > 0 {
		memoryMinPages = (maxUsed + wasmPageSize - 1) / wasmPageSize
		if memoryMinPages == 0 {
			memoryMinPages = 1
		}
	}

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

		writeName(sec, "tetra_web_v1")
		writeName(sec, "console_log")
		sec.WriteByte(0x00) // import kind: func
		writeULEB(sec, consoleLogType)

		writeName(sec, "tetra_web_v1")
		writeName(sec, "panic")
		sec.WriteByte(0x00) // import kind: func
		writeULEB(sec, panicType)
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

	writeSection(&module, 7, func(sec *bytes.Buffer) {
		writeULEB(sec, 2)

		writeName(sec, "memory")
		sec.WriteByte(0x02) // export kind: memory
		writeULEB(sec, 0)

		writeName(sec, "tetra_main")
		sec.WriteByte(0x00) // export kind: func
		writeULEB(sec, mainFuncIdx)
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

func LoaderModule(wasmFileName string) []byte {
	if wasmFileName == "" {
		wasmFileName = "app.wasm"
	}
	wasmFileName = escapeJSLiteral(wasmFileName)
	src := strings.Join([]string{
		"const TETRA_WASM_URL = new URL(\"" + wasmFileName + "\", import.meta.url);",
		"",
		"function memoryView(instance) {",
		"  const memory = instance.exports.memory;",
		"  if (!(memory instanceof WebAssembly.Memory)) {",
		"    throw new Error(\"tetra_web_v1: missing exported memory\");",
		"  }",
		"  return new Uint8Array(memory.buffer);",
		"}",
		"",
		"function readUTF8(instance, ptr, len) {",
		"  const view = memoryView(instance);",
		"  const start = ptr >>> 0;",
		"  const end = (ptr + len) >>> 0;",
		"  return new TextDecoder().decode(view.subarray(start, end));",
		"}",
		"",
		"function createImports(instanceRef) {",
		"  return {",
		"    tetra_web_v1: {",
		"      console_log(ptr, len) {",
		"        const instance = instanceRef.instance;",
		"        if (!instance) {",
		"          throw new Error(\"tetra_web_v1: instance is not ready\");",
		"        }",
		"        console.log(readUTF8(instance, ptr | 0, len | 0));",
		"      },",
		"      panic(code, ptr, len) {",
		"        const instance = instanceRef.instance;",
		"        let message = \"panic\";",
		"        if (instance) {",
		"          message = readUTF8(instance, ptr | 0, len | 0);",
		"        }",
		"        throw new Error(\"tetra panic(\" + (code | 0) + \"): \" + message);",
		"      },",
		"    },",
		"  };",
		"}",
		"",
		"export async function instantiateTetra(moduleURL = TETRA_WASM_URL) {",
		"  const response = await fetch(moduleURL);",
		"  if (!response.ok) {",
		"    throw new Error(\"tetra_web_v1: fetch failed: \" + response.status);",
		"  }",
		"  const bytes = await response.arrayBuffer();",
		"  const instanceRef = { instance: null };",
		"  const result = await WebAssembly.instantiate(bytes, createImports(instanceRef));",
		"  instanceRef.instance = result.instance;",
		"  return result;",
		"}",
		"",
		"export async function runTetra(moduleURL = TETRA_WASM_URL) {",
		"  const { instance } = await instantiateTetra(moduleURL);",
		"  const tetraMain = instance.exports.tetra_main;",
		"  if (typeof tetraMain !== \"function\") {",
		"    throw new Error(\"tetra_web_v1: missing tetra_main export\");",
		"  }",
		"  return tetraMain() | 0;",
		"}",
	}, "\n")
	return []byte(src + "\n")
}

func escapeJSLiteral(s string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"\n", "\\n",
		"\r", "\\r",
	)
	return replacer.Replace(s)
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

func compileFunction(fn Function, data *dataBuilder, funcIndexByName map[string]uint32, consoleLogImport int) ([]byte, error) {
	if fn.LocalSlots < fn.ParamSlots {
		return nil, fmt.Errorf("wasm backend: function '%s' has invalid local slots", fn.Name)
	}

	localDeclCount := fn.LocalSlots - fn.ParamSlots

	var body bytes.Buffer
	if localDeclCount > 0 {
		writeULEB(&body, 1) // one local group
		writeULEB(&body, uint32(localDeclCount))
		body.WriteByte(0x7f) // i32
	} else {
		writeULEB(&body, 0)
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
		case ir.IRWrite:
			if err := pop(2, "write"); err != nil {
				return nil, err
			}
			body.WriteByte(0x10) // call console_log
			writeULEB(&body, uint32(consoleLogImport))
		case ir.IRReturn:
			if err := pop(fn.ReturnSlots, "return"); err != nil {
				return nil, err
			}
			body.WriteByte(0x0f) // return
			stackDepth = 0
			terminated = true
		default:
			return nil, fmt.Errorf("wasm backend: unsupported IR instruction %d in function '%s'", instr.Kind, fn.Name)
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
