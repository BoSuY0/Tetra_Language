package wasm32_web

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/runtimeabi"
)

const (
	wasmPageSize    = 65536
	wasmHeapAlign   = 16
	dataBase        = uint32(0x1000)
	webImportModule = "tetra_web_v0.4.0"
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

type wasmImportSpec struct {
	Module      string
	Name        string
	ParamSlots  int
	ReturnSlots int
}

func wasmImportsForObject(funcs []Function) []wasmImportSpec {
	imports := []wasmImportSpec{
		{Module: webImportModule, Name: "console_log", ParamSlots: 2, ReturnSlots: 0},
		{Module: webImportModule, Name: "panic", ParamSlots: 3, ReturnSlots: 0},
	}
	usedSurface := make(map[string]struct{})
	for _, fn := range funcs {
		for _, instr := range fn.Instrs {
			if instr.Kind != ir.IRCall {
				continue
			}
			if _, ok := wasmSurfaceImportSignature(instr.Name); ok {
				usedSurface[instr.Name] = struct{}{}
			}
		}
	}
	for _, name := range runtimeabi.RequiredSurfaceSymbols() {
		if _, ok := usedSurface[name]; !ok {
			continue
		}
		sig, _ := wasmSurfaceImportSignature(name)
		imports = append(imports, wasmImportSpec{
			Module:      "tetra_surface_host_v1",
			Name:        name,
			ParamSlots:  sig.ParamSlots,
			ReturnSlots: sig.ReturnSlots,
		})
	}
	return imports
}

func wasmSurfaceImportSignature(name string) (wasmFunctionSignature, bool) {
	if !strings.HasPrefix(name, "__tetra_surface_") {
		return wasmFunctionSignature{}, false
	}
	sig, ok := runtimeabi.SignatureForSymbol(name)
	if !ok {
		return wasmFunctionSignature{}, false
	}
	return wasmFunctionSignature{ParamSlots: sig.ParamSlots, ReturnSlots: sig.ReturnSlots}, true
}

func CodegenObject(funcs []ir.IRFunc, mainName string) (*Object, error) {
	return CodegenObjectWithDataPrefix(funcs, mainName, nil)
}

func CodegenObjectWithDataPrefix(
	funcs []ir.IRFunc,
	mainName string,
	dataPrefix [][]byte,
) (*Object, error) {
	if len(funcs) == 0 {
		return nil, fmt.Errorf("wasm backend: no functions to compile")
	}
	out := make([]Function, 0, len(funcs))
	globalSlots := 0
	symbolTokens := make(map[uint32]string)
	functionNames := make(map[string]struct{}, len(funcs))
	functionSigs := make(map[string]wasmFunctionSignature, len(funcs))
	for _, fn := range funcs {
		if err := validateWasmFunctionMetadata(
			functionNames,
			fn.Name,
			fn.ParamSlots,
			fn.LocalSlots,
			fn.ReturnSlots,
		); err != nil {
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
		functionSigs[fn.Name] = wasmFunctionSignature{
			ParamSlots:  fn.ParamSlots,
			ReturnSlots: fn.ReturnSlots,
		}
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
	return &Object{
		Functions:   out,
		MainName:    mainName,
		GlobalSlots: globalSlots,
		GlobalInits: globalInits,
	}, nil
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

	imports := wasmImportsForObject(obj.Functions)
	importTypeIdx := make([]uint32, len(imports))
	importIndexByName := make(map[string]uint32, len(imports))
	for i, imp := range imports {
		importTypeIdx[i] = typeIndex(imp.ParamSlots, imp.ReturnSlots)
		importIndexByName[imp.Name] = uint32(i)
	}
	consoleLogImport, ok := importIndexByName["console_log"]
	if !ok {
		return nil, fmt.Errorf("wasm backend: missing console_log import")
	}

	funcIndexByName := make(map[string]uint32, len(obj.Functions))
	returnSlotsByName := make(map[string]int, len(obj.Functions))
	funcTypeIdx := make([]uint32, 0, len(obj.Functions))
	for _, imp := range imports {
		funcIndexByName[imp.Name] = importIndexByName[imp.Name]
	}
	for i, fn := range obj.Functions {
		funcIndexByName[fn.Name] = uint32(len(imports) + i)
		returnSlotsByName[fn.Name] = fn.ReturnSlots
		funcTypeIdx = append(funcTypeIdx, typeIndex(fn.ParamSlots, fn.ReturnSlots))
	}
	mainFuncIdx, ok := funcIndexByName[mainName]
	if !ok {
		return nil, fmt.Errorf("wasm backend: entry function '%s' not found", mainName)
	}
	if returnSlotsByName[mainName] != 1 {
		return nil, fmt.Errorf(
			"wasm backend: entry function '%s' must return exactly 1 slot, got %d",
			mainName,
			returnSlotsByName[mainName],
		)
	}

	data := newDataBuilder()
	codeBodies := make([][]byte, 0, len(obj.Functions))
	const heapGlobalIndex = uint32(0)
	for _, fn := range obj.Functions {
		body, err := compileFunction(
			fn,
			data,
			funcIndexByName,
			int(consoleLogImport),
			heapGlobalIndex,
		)
		if err != nil {
			return nil, err
		}
		codeBodies = append(codeBodies, body)
	}

	maxUsed := data.maxUsed()
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
		writeULEB(sec, uint32(len(imports)))
		for i, imp := range imports {
			writeName(sec, imp.Module)
			writeName(sec, imp.Name)
			sec.WriteByte(0x00) // import kind: func
			writeULEB(sec, importTypeIdx[i])
		}
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
		"    throw new Error(\"" + webImportModule + ": missing exported memory\");",
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
		"function createSurfaceHost(instanceRef) {",
		"  const surfaces = new Map();",
		"  let nextHandle = 1;",
		"  let clipboard = new Uint8Array([84, 101, 116]);",
		"  return {",
		"    __tetra_surface_open(titlePtr, titleLen, width, height) {",
		"      const instance = instanceRef.instance;",
		"      const title = instance ? readUTF8(instance, titlePtr | 0, titleLen | 0) : \"\";",
		"      const handle = nextHandle++;",
		"      surfaces.set(handle, { title, width: width | 0, height: height | 0, presented: 0 });",
		"      return handle | 0;",
		"    },",
		"    __tetra_surface_close(handle) {",
		"      surfaces.delete(handle | 0);",
		"      return 0;",
		"    },",
		"    __tetra_surface_poll_event_kind(handle) {",
		"      return surfaces.has(handle | 0) ? 5 : 1;",
		"    },",
		"    __tetra_surface_poll_event_x(handle) {",
		"      return surfaces.has(handle | 0) ? 48 : 0;",
		"    },",
		"    __tetra_surface_poll_event_y(handle) {",
		"      return surfaces.has(handle | 0) ? 96 : 0;",
		"    },",
		"    __tetra_surface_poll_event_button(handle) {",
		"      return surfaces.has(handle | 0) ? 1 : 0;",
		"    },",
		"    __tetra_surface_poll_event_into(handle, eventPtr, eventLen) {",
		"      const instance = instanceRef.instance;",
		"      const surface = surfaces.get(handle | 0);",
		"      if (!surface || !instance || (eventLen | 0) < 9) {",
		"        return 0;",
		"      }",
		"      const view = new DataView(instance.exports.memory.buffer);",
		"      const start = eventPtr >>> 0;",
		"      if (start + 36 > view.byteLength) {",
		"        return 0;",
		"      }",
		"      view.setInt32(start, 5, true);",
		"      view.setInt32(start + 4, 48, true);",
		"      view.setInt32(start + 8, 96, true);",
		"      view.setInt32(start + 12, 1, true);",
		"      view.setInt32(start + 16, 0, true);",
		"      view.setInt32(start + 20, surface.width | 0, true);",
		"      view.setInt32(start + 24, surface.height | 0, true);",
		"      view.setInt32(start + 28, 0, true);",
		"      view.setInt32(start + 32, 0, true);",
		"      return 9;",
		"    },",
		"    __tetra_surface_poll_event_text_len(handle) {",
		"      return surfaces.has(handle | 0) ? 2 : 0;",
		"    },",
		"    __tetra_surface_poll_event_text_into(handle, textPtr, textLen) {",
		"      const instance = instanceRef.instance;",
		"      if (!surfaces.has(handle | 0) || !instance || (textLen | 0) < 2) {",
		"        return 0;",
		"      }",
		"      const view = memoryView(instance);",
		"      const start = textPtr >>> 0;",
		"      if (start + 2 > view.length) {",
		"        return 0;",
		"      }",
		"      view[start] = 79;",
		"      view[start + 1] = 75;",
		"      return 2;",
		"    },",
		"    __tetra_surface_clipboard_write_text(handle, textPtr, textLen) {",
		"      const instance = instanceRef.instance;",
		"      if (!surfaces.has(handle | 0) || !instance || (textLen | 0) < 0) {",
		"        return 0;",
		"      }",
		"      const view = memoryView(instance);",
		"      const start = textPtr >>> 0;",
		"      const len = textLen | 0;",
		"      if (start + len > view.length) {",
		"        return 0;",
		"      }",
		"      clipboard = new Uint8Array(view.subarray(start, start + len));",
		"      return len;",
		"    },",
		"    __tetra_surface_clipboard_read_text_into(handle, textPtr, textLen) {",
		"      const instance = instanceRef.instance;",
		"      if (!surfaces.has(handle | 0) || !instance) {",
		"        return 0;",
		"      }",
		"      const view = memoryView(instance);",
		"      const start = textPtr >>> 0;",
		"      const cap = textLen | 0;",
		"      const copied = Math.min(cap, clipboard.length) | 0;",
		"      if (copied < 0 || start + copied > view.length) {",
		"        return 0;",
		"      }",
		"      view.set(clipboard.subarray(0, copied), start);",
		"      return copied;",
		"    },",
		"    __tetra_surface_poll_composition_into(handle, eventPtr, eventLen) {",
		"      const instance = instanceRef.instance;",
		"      if (!surfaces.has(handle | 0) || !instance || (eventLen | 0) < 4) {",
		"        return 0;",
		"      }",
		"      const view = new DataView(instance.exports.memory.buffer);",
		"      const start = eventPtr >>> 0;",
		"      if (start + 16 > view.byteLength) {",
		"        return 0;",
		"      }",
		"      view.setInt32(start, 1, true);",
		"      view.setInt32(start + 4, 1, true);",
		"      view.setInt32(start + 8, 1, true);",
		"      view.setInt32(start + 12, 1, true);",
		"      return 4;",
		"    },",
		"    __tetra_surface_begin_frame(handle) {",
		"      return surfaces.has(handle | 0) ? 0 : 1;",
		"    },",
		"    __tetra_surface_present_rgba(handle, pixelsPtr, pixelsLen, width, height, stride) {",
		"      const surface = surfaces.get(handle | 0);",
		"      if (!surface) {",
		"        return 1;",
		"      }",
		"      surface.width = width | 0;",
		"      surface.height = height | 0;",
		"      surface.presented = (surface.presented + 1) | 0;",
		("      surface.lastFrame = { pixelsPtr: pixelsPtr | 0, " +
			"pixelsLen: pixelsLen | 0, stride: stride | 0 };"),
		"      return 0;",
		"    },",
		"    __tetra_surface_now_ms() {",
		"      return 0;",
		"    },",
		"    __tetra_surface_request_redraw(handle) {",
		"      return surfaces.has(handle | 0) ? 0 : 1;",
		"    },",
		"  };",
		"}",
		"",
		"function createImports(instanceRef) {",
		"  return {",
		"    \"" + webImportModule + "\": {",
		"      console_log(ptr, len) {",
		"        const instance = instanceRef.instance;",
		"        if (!instance) {",
		"          throw new Error(\"" + webImportModule + ": instance is not ready\");",
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
		"    tetra_surface_host_v1: createSurfaceHost(instanceRef),",
		"  };",
		"}",
		"",
		"export async function instantiateTetra(moduleURL = TETRA_WASM_URL) {",
		"  const response = await fetch(moduleURL);",
		"  if (!response.ok) {",
		"    throw new Error(\"" + webImportModule + ": fetch failed: \" + response.status);",
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
		"    throw new Error(\"" + webImportModule + ": missing tetra_main export\");",
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

func emitHeapBumpAndGrow(
	body *bytes.Buffer,
	heapGlobalIndex uint32,
	tempPtr int,
	tempByteLen int,
	tempVal int,
) {
	body.WriteByte(0x23) // global.get heap
	writeULEB(body, heapGlobalIndex)
	body.WriteByte(0x21) // local.set tempPtr
	writeULEB(body, uint32(tempPtr))

	body.WriteByte(0x23) // global.get heap
	writeULEB(body, heapGlobalIndex)
	body.WriteByte(0x20) // local.get tempByteLen
	writeULEB(body, uint32(tempByteLen))
	body.WriteByte(0x6a) // i32.add
	body.WriteByte(0x21) // local.set tempVal (new heap end)
	writeULEB(body, uint32(tempVal))

	body.WriteByte(0x20) // local.get tempVal
	writeULEB(body, uint32(tempVal))
	body.WriteByte(0x24) // global.set heap
	writeULEB(body, heapGlobalIndex)

	body.WriteByte(0x20) // local.get tempVal
	writeULEB(body, uint32(tempVal))
	body.WriteByte(0x3f) // memory.size
	body.WriteByte(0x00)
	writeI32Const(body, 16)
	body.WriteByte(0x74) // i32.shl: current pages to bytes
	body.WriteByte(0x4b) // i32.gt_u
	body.WriteByte(0x04) // if
	body.WriteByte(0x40)
	body.WriteByte(0x20) // local.get tempVal
	writeULEB(body, uint32(tempVal))
	writeI32Const(body, int32(wasmPageSize-1))
	body.WriteByte(0x6a) // i32.add
	writeI32Const(body, 16)
	body.WriteByte(0x76) // i32.shr_u: required pages
	body.WriteByte(0x3f) // memory.size
	body.WriteByte(0x00)
	body.WriteByte(0x6b) // i32.sub: delta pages
	body.WriteByte(0x40) // memory.grow
	body.WriteByte(0x00)
	body.WriteByte(0x1a) // drop previous page count
	body.WriteByte(0x0b) // end if
}

func compileFunction(
	fn Function,
	data *dataBuilder,
	funcIndexByName map[string]uint32,
	consoleLogImport int,
	heapGlobalIndex uint32,
) ([]byte, error) {
	if fn.LocalSlots < fn.ParamSlots {
		return nil, fmt.Errorf("wasm backend: function '%s' has invalid local slots", fn.Name)
	}

	localDeclCount := fn.LocalSlots - fn.ParamSlots
	tempPtr := fn.LocalSlots
	tempLen := fn.LocalSlots + 1
	tempIdx := fn.LocalSlots + 2
	tempVal := fn.LocalSlots + 3
	tempByteLen := fn.LocalSlots + 4
	pcLocal := fn.LocalSlots + 5
	localDeclCount += 6

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
		return compileFunctionWithControlFlow(
			fn,
			data,
			funcIndexByName,
			consoleLogImport,
			heapGlobalIndex,
			tempPtr,
			tempLen,
			tempIdx,
			tempVal,
			tempByteLen,
			pcLocal,
			&body,
		)
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
				return nil, fmt.Errorf(
					"wasm backend: function '%s' calls unsupported symbol '%s'",
					fn.Name,
					instr.Name,
				)
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
			emitWasmMakeSliceContract(
				&body,
				instr.Kind,
				heapGlobalIndex,
				tempPtr,
				tempLen,
				tempByteLen,
				tempVal,
			)
			body.WriteByte(0x20) // local.get tempPtr
			writeULEB(&body, uint32(tempPtr))
				body.WriteByte(0x20) // local.get tempLen
				writeULEB(&body, uint32(tempLen))
				push(2)
			case ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32:
				if !wasmZeroStackSliceSentinel(instr) {
					return nil, wasmUnsupportedInstrError(fn.Name, instr.Kind)
				}
				if err := pop(1, "zero_stack_slice"); err != nil {
					return nil, err
				}
				emitWasmZeroSliceSentinel(&body)
				push(2)
			case ir.IRRawSliceFromParts:
				if err := pop(3, "raw_slice_from_parts"); err != nil {
					return nil, err
			}
			body.WriteByte(0x21) // local.set tempByteLen, discard cap.mem token
			writeULEB(&body, uint32(tempByteLen))
			body.WriteByte(0x21) // local.set tempLen
			writeULEB(&body, uint32(tempLen))
			body.WriteByte(0x21) // local.set tempPtr
			writeULEB(&body, uint32(tempPtr))
			body.WriteByte(0x20) // local.get tempPtr
			writeULEB(&body, uint32(tempPtr))
			body.WriteByte(0x20) // local.get tempLen
			writeULEB(&body, uint32(tempLen))
			push(2)
		case ir.IRSliceWindow, ir.IRSlicePrefix, ir.IRSliceSuffix:
			popSlots := 3
			if instr.Kind == ir.IRSliceWindow {
				popSlots = 4
			}
			if err := pop(popSlots, "slice_view"); err != nil {
				return nil, err
			}
			emitWasmSliceView(
				&body,
				instr.Kind,
				byte(instr.Imm),
				tempPtr,
				tempLen,
				tempIdx,
				tempVal,
			)
			push(2)
		case ir.IRIslandNew:
			if err := pop(1, "island_new"); err != nil {
				return nil, err
			}
			body.WriteByte(0x21) // local.set tempByteLen
			writeULEB(&body, uint32(tempByteLen))
			emitHeapBumpAndGrow(&body, heapGlobalIndex, tempPtr, tempByteLen, tempVal)
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
			emitWasmMakeSliceContract(
				&body,
				instr.Kind,
				heapGlobalIndex,
				tempPtr,
				tempLen,
				tempByteLen,
				tempVal,
			)
			body.WriteByte(0x20) // local.get tempPtr
			writeULEB(&body, uint32(tempPtr))
			body.WriteByte(0x20) // local.get tempLen
			writeULEB(&body, uint32(tempLen))
			push(2)
		case ir.IRIslandFree:
			if err := pop(1, "island_free"); err != nil {
				return nil, err
			}
		case ir.IRIslandReset:
			if err := pop(1, "island_reset"); err != nil {
				return nil, err
			}
			body.WriteByte(0x21) // local.set tempPtr
			writeULEB(&body, uint32(tempPtr))
			body.WriteByte(0x20) // local.get tempPtr
			writeULEB(&body, uint32(tempPtr))
			push(1)
		case ir.IRIndexLoadI32, ir.IRIndexLoadU8, ir.IRIndexLoadU16,
			ir.IRIndexLoadI32Unchecked, ir.IRIndexLoadU8Unchecked, ir.IRIndexLoadU16Unchecked:
			if err := pop(3, "index_load"); err != nil {
				return nil, err
			}
			body.WriteByte(0x21)
			writeULEB(&body, uint32(tempIdx))
			body.WriteByte(0x21)
			writeULEB(&body, uint32(tempLen))
			body.WriteByte(0x21)
			writeULEB(&body, uint32(tempPtr))
			checked := instr.Kind == ir.IRIndexLoadI32 || instr.Kind == ir.IRIndexLoadU8 ||
				instr.Kind == ir.IRIndexLoadU16
			if checked {
				body.WriteByte(0x20)
				writeULEB(&body, uint32(tempIdx))
				body.WriteByte(0x20)
				writeULEB(&body, uint32(tempLen))
				body.WriteByte(0x4f) // i32.ge_u
				body.WriteByte(0x04)
				body.WriteByte(0x40)
				body.WriteByte(0x00) // unreachable
				body.WriteByte(0x0b)
			}
			body.WriteByte(0x20)
			writeULEB(&body, uint32(tempPtr))
			body.WriteByte(0x20)
			writeULEB(&body, uint32(tempIdx))
			switch instr.Kind {
			case ir.IRIndexLoadI32, ir.IRIndexLoadI32Unchecked:
				writeI32Const(&body, 2)
				body.WriteByte(0x74)
			case ir.IRIndexLoadU16, ir.IRIndexLoadU16Unchecked:
				writeI32Const(&body, 1)
				body.WriteByte(0x74)
			}
			body.WriteByte(0x6a)
			switch instr.Kind {
			case ir.IRIndexLoadI32, ir.IRIndexLoadI32Unchecked:
				body.WriteByte(0x28)
				writeULEB(&body, 2)
				writeULEB(&body, 0)
			case ir.IRIndexLoadU16, ir.IRIndexLoadU16Unchecked:
				body.WriteByte(0x2f)
				writeULEB(&body, 1)
				writeULEB(&body, 0)
			default:
				body.WriteByte(0x2d)
				writeULEB(&body, 0)
				writeULEB(&body, 0)
			}
			push(1)
		case ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
			if err := pop(4, "index_store"); err != nil {
				return nil, err
			}
			body.WriteByte(0x21)
			writeULEB(&body, uint32(tempVal))
			body.WriteByte(0x21)
			writeULEB(&body, uint32(tempIdx))
			body.WriteByte(0x21)
			writeULEB(&body, uint32(tempLen))
			body.WriteByte(0x21)
			writeULEB(&body, uint32(tempPtr))
			body.WriteByte(0x20)
			writeULEB(&body, uint32(tempIdx))
			body.WriteByte(0x20)
			writeULEB(&body, uint32(tempLen))
			body.WriteByte(0x4f)
			body.WriteByte(0x04)
			body.WriteByte(0x40)
			body.WriteByte(0x00)
			body.WriteByte(0x0b)
			body.WriteByte(0x20)
			writeULEB(&body, uint32(tempPtr))
			body.WriteByte(0x20)
			writeULEB(&body, uint32(tempIdx))
			switch instr.Kind {
			case ir.IRIndexStoreI32:
				writeI32Const(&body, 2)
				body.WriteByte(0x74)
			case ir.IRIndexStoreU16:
				writeI32Const(&body, 1)
				body.WriteByte(0x74)
			}
			body.WriteByte(0x6a)
			body.WriteByte(0x20)
			writeULEB(&body, uint32(tempVal))
			switch instr.Kind {
			case ir.IRIndexStoreI32:
				body.WriteByte(0x36)
				writeULEB(&body, 2)
				writeULEB(&body, 0)
			case ir.IRIndexStoreU16:
				body.WriteByte(0x3b)
				writeULEB(&body, 1)
				writeULEB(&body, 0)
			default:
				body.WriteByte(0x3a)
				writeULEB(&body, 0)
				writeULEB(&body, 0)
			}
		case ir.IRLabel, ir.IRJmp, ir.IRJmpIfZero:
			return nil, fmt.Errorf(
				"wasm backend: control-flow IR in linear mode for function '%s'",
				fn.Name,
			)
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
			return nil, wasmUnsupportedInstrError(fn.Name, instr.Kind)
		}
	}

	if !terminated {
		if fn.ReturnSlots == 0 {
			body.WriteByte(0x0f) // return
		} else if stackDepth == fn.ReturnSlots {
			body.WriteByte(0x0f) // return with stack value(s)
		} else {
			return nil, fmt.Errorf(
				"wasm backend: function '%s' ended with stack depth %d (want %d)",
				fn.Name,
				stackDepth,
				fn.ReturnSlots,
			)
		}
	}
	body.WriteByte(0x0b) // end

	return body.Bytes(), nil
}
