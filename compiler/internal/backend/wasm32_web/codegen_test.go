package wasm32_web

import (
	"bytes"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestLinkObjectWritesWASMHeaderAndTetraMainExport(t *testing.T) {
	obj, err := CodegenObject([]ir.IRFunc{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
		},
	}, "main")
	if err != nil {
		t.Fatalf("CodegenObject: %v", err)
	}
	mod, err := LinkObject(obj)
	if err != nil {
		t.Fatalf("LinkObject: %v", err)
	}
	if len(mod) < 8 {
		t.Fatalf("module too short: %d", len(mod))
	}
	if !bytes.Equal(mod[:4], []byte{0x00, 0x61, 0x73, 0x6d}) {
		t.Fatalf("missing wasm magic: % x", mod[:4])
	}
	if !bytes.Equal(mod[4:8], []byte{0x01, 0x00, 0x00, 0x00}) {
		t.Fatalf("unexpected version header: % x", mod[4:8])
	}
	if !bytes.Contains(mod, []byte("tetra_main")) {
		t.Fatalf("missing tetra_main export")
	}
}

func TestLinkObjectWebImportExportShape(t *testing.T) {
	obj, err := CodegenObject([]ir.IRFunc{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
		},
	}, "main")
	if err != nil {
		t.Fatalf("CodegenObject: %v", err)
	}
	mod, err := LinkObject(obj)
	if err != nil {
		t.Fatalf("LinkObject: %v", err)
	}

	imports := wasmImports(t, mod)
	if got := imports["tetra_web_v1"]; !stringSetHas(got, "console_log") || !stringSetHas(got, "panic") {
		t.Fatalf("web imports = %#v", imports)
	}
	if _, ok := imports["wasi_snapshot_preview1"]; ok {
		t.Fatalf("web module imported WASI namespace: %#v", imports)
	}

	exports := wasmExports(t, mod)
	if exports["memory"] != 0x02 {
		t.Fatalf("memory export kind = 0x%x, want memory", exports["memory"])
	}
	if exports["tetra_main"] != 0x00 {
		t.Fatalf("tetra_main export kind = 0x%x, want func", exports["tetra_main"])
	}
	if _, ok := exports["_start"]; ok {
		t.Fatalf("web module exported _start: %#v", exports)
	}
}

func TestLinkObjectWebOutputIsDeterministic(t *testing.T) {
	funcs := []ir.IRFunc{
		{
			Name:        "z",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
		},
	}

	obj1, err := CodegenObject(funcs, "main")
	if err != nil {
		t.Fatalf("CodegenObject #1: %v", err)
	}
	obj2, err := CodegenObject(funcs, "main")
	if err != nil {
		t.Fatalf("CodegenObject #2: %v", err)
	}
	mod1, err := LinkObject(obj1)
	if err != nil {
		t.Fatalf("LinkObject #1: %v", err)
	}
	mod2, err := LinkObject(obj2)
	if err != nil {
		t.Fatalf("LinkObject #2: %v", err)
	}
	if !bytes.Equal(mod1, mod2) {
		t.Fatalf("web module output is not deterministic")
	}
}

func TestLinkObjectRejectsUnsupportedInstruction(t *testing.T) {
	obj, err := CodegenObject([]ir.IRFunc{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRAllocBytes},
				{Kind: ir.IRReturn},
			},
		},
	}, "main")
	if err != nil {
		t.Fatalf("CodegenObject: %v", err)
	}
	_, err = LinkObject(obj)
	if err == nil {
		t.Fatalf("expected unsupported instruction error")
	}
	if got := err.Error(); !bytes.Contains([]byte(got), []byte("unsupported IR instruction")) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCodegenObjectWebRejectsNegativeGlobalSlots(t *testing.T) {
	for _, instr := range []ir.IRInstr{
		{Kind: ir.IRLoadGlobal, Local: -1},
		{Kind: ir.IRStoreGlobal, Local: -1},
	} {
		_, err := CodegenObject([]ir.IRFunc{
			{
				Name:        "main",
				ParamSlots:  0,
				LocalSlots:  0,
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					instr,
					{Kind: ir.IRReturn},
				},
			},
		}, "main")
		if err == nil {
			t.Fatalf("expected negative global slot diagnostic for %v", instr.Kind)
		}
		if got := err.Error(); !bytes.Contains([]byte(got), []byte("negative global slot -1 in function 'main'")) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestLinkObjectWebAllowsRepeatedSymAddrSymbol(t *testing.T) {
	obj, err := CodegenObject([]ir.IRFunc{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRSymAddr, Name: "callback"},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRSymAddr, Name: "callback"},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
		},
	}, "main")
	if err != nil {
		t.Fatalf("CodegenObject: %v", err)
	}
	if _, err := LinkObject(obj); err != nil {
		t.Fatalf("LinkObject: %v", err)
	}
}

func TestCodegenObjectWebAllowsRepeatedSymAddrSymbolAcrossFunctions(t *testing.T) {
	_, err := CodegenObject([]ir.IRFunc{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRSymAddr, Name: "callback"},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "helper",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRSymAddr, Name: "callback"},
				{Kind: ir.IRReturn},
			},
		},
	}, "main")
	if err != nil {
		t.Fatalf("CodegenObject: %v", err)
	}
}

func TestCodegenObjectWebRejectsSymAddrTokenCollision(t *testing.T) {
	old := wasmSymbolTokenHash
	wasmSymbolTokenHash = func(string) uint32 { return 0x2a }
	defer func() { wasmSymbolTokenHash = old }()

	_, err := CodegenObject([]ir.IRFunc{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRSymAddr, Name: "alpha"},
				{Kind: ir.IRSymAddr, Name: "beta"},
				{Kind: ir.IRReturn},
			},
		},
	}, "main")
	if err == nil {
		t.Fatalf("expected symbol token collision diagnostic")
	}
	for _, want := range []string{"symbol address token collision", "alpha", "beta", "0x0000002a"} {
		if !bytes.Contains([]byte(err.Error()), []byte(want)) {
			t.Fatalf("error = %v, want substring %q", err, want)
		}
	}
}

func TestLinkObjectWebRejectsSymAddrTokenCollision(t *testing.T) {
	old := wasmSymbolTokenHash
	wasmSymbolTokenHash = func(string) uint32 { return 0x2a }
	defer func() { wasmSymbolTokenHash = old }()

	_, err := LinkObject(&Object{
		MainName: "main",
		Functions: []Function{
			{
				Name:        "main",
				ParamSlots:  0,
				LocalSlots:  0,
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRSymAddr, Name: "alpha"},
					{Kind: ir.IRReturn},
				},
			},
			{
				Name:        "helper",
				ParamSlots:  0,
				LocalSlots:  0,
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRSymAddr, Name: "beta"},
					{Kind: ir.IRReturn},
				},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected symbol token collision diagnostic")
	}
	for _, want := range []string{"symbol address token collision", "alpha", "beta", "0x0000002a"} {
		if !bytes.Contains([]byte(err.Error()), []byte(want)) {
			t.Fatalf("error = %v, want substring %q", err, want)
		}
	}
}

func TestLinkObjectWebSupportsIslandInstructionsInBuildOnlyMode(t *testing.T) {
	obj, err := CodegenObject([]ir.IRFunc{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  5,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 64},
				{Kind: ir.IRIslandNew},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRIslandMakeSliceU8},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRIslandMakeSliceI32},
				{Kind: ir.IRStoreLocal, Local: 4},
				{Kind: ir.IRStoreLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRIslandFree},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
		},
	}, "main")
	if err != nil {
		t.Fatalf("CodegenObject: %v", err)
	}
	if _, err := LinkObject(obj); err != nil {
		t.Fatalf("LinkObject: %v", err)
	}
}

func TestLoaderModuleIncludesRuntimeNamespaceAndEntry(t *testing.T) {
	loader := string(LoaderModule("app.wasm"))
	if !bytes.Contains([]byte(loader), []byte("tetra_web_v1")) {
		t.Fatalf("loader missing runtime namespace:\n%s", loader)
	}
	if !bytes.Contains([]byte(loader), []byte("tetra_main")) {
		t.Fatalf("loader missing tetra_main call:\n%s", loader)
	}
}

func TestLinkObjectWebRejectsMissingEntryFunction(t *testing.T) {
	obj, err := CodegenObject([]ir.IRFunc{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
		},
	}, "missing")
	if err != nil {
		t.Fatalf("CodegenObject: %v", err)
	}
	_, err = LinkObject(obj)
	if err == nil {
		t.Fatalf("expected missing entry error")
	}
	if got := err.Error(); !bytes.Contains([]byte(got), []byte("entry function 'missing' not found")) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkObjectWebRejectsMultiSlotEntryFunction(t *testing.T) {
	obj, err := CodegenObject([]ir.IRFunc{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 2,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRReturn},
			},
		},
	}, "main")
	if err != nil {
		t.Fatalf("CodegenObject: %v", err)
	}
	_, err = LinkObject(obj)
	if err == nil {
		t.Fatalf("expected multi-slot entry error")
	}
	if got := err.Error(); !bytes.Contains([]byte(got), []byte("entry function 'main' must return exactly 1 slot")) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkObjectWebSupportsControlFlowAndI32ArrayIR(t *testing.T) {
	obj, err := CodegenObject([]ir.IRFunc{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  4,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 4},
				{Kind: ir.IRMakeSliceI32},
				{Kind: ir.IRStoreLocal, Local: 1}, // len
				{Kind: ir.IRStoreLocal, Local: 0}, // ptr
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 2}, // i
				{Kind: ir.IRLabel, Label: 10},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRConstI32, Imm: 4},
				{Kind: ir.IRCmpLtI32},
				{Kind: ir.IRJmpIfZero, Label: 20},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRIndexStoreI32},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRIndexLoadI32},
				{Kind: ir.IRStoreLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRJmp, Label: 10},
				{Kind: ir.IRLabel, Label: 20},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
		},
	}, "main")
	if err != nil {
		t.Fatalf("CodegenObject: %v", err)
	}
	if _, err := LinkObject(obj); err != nil {
		t.Fatalf("LinkObject: %v", err)
	}
}

func TestLinkObjectWebSupportsU8U16ArrayIR(t *testing.T) {
	obj, err := CodegenObject([]ir.IRFunc{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  6,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRMakeSliceU8},
				{Kind: ir.IRStoreLocal, Local: 1}, // len8
				{Kind: ir.IRStoreLocal, Local: 0}, // ptr8
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRIndexStoreU8},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRIndexLoadU8},
				{Kind: ir.IRStoreLocal, Local: 2},

				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRMakeSliceU16},
				{Kind: ir.IRStoreLocal, Local: 4}, // len16
				{Kind: ir.IRStoreLocal, Local: 3}, // ptr16
				{Kind: ir.IRLoadLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 4},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRConstI32, Imm: 9},
				{Kind: ir.IRIndexStoreU16},
				{Kind: ir.IRLoadLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 4},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRIndexLoadU16},
				{Kind: ir.IRStoreLocal, Local: 5},

				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
		},
	}, "main")
	if err != nil {
		t.Fatalf("CodegenObject: %v", err)
	}
	if _, err := LinkObject(obj); err != nil {
		t.Fatalf("LinkObject: %v", err)
	}
}

func wasmImports(t *testing.T, mod []byte) map[string][]string {
	t.Helper()
	payload := wasmSection(t, mod, 2)
	if payload == nil {
		t.Fatalf("missing import section")
	}
	pos := 0
	count := int(readULEBForTest(t, payload, &pos))
	out := map[string][]string{}
	for i := 0; i < count; i++ {
		module := readWASMNameForTest(t, payload, &pos)
		name := readWASMNameForTest(t, payload, &pos)
		if pos >= len(payload) {
			t.Fatalf("truncated import kind")
		}
		kind := payload[pos]
		pos++
		if kind != 0x00 {
			t.Fatalf("import %s.%s kind = 0x%x, want func", module, name, kind)
		}
		_ = readULEBForTest(t, payload, &pos)
		out[module] = append(out[module], name)
	}
	return out
}

func wasmExports(t *testing.T, mod []byte) map[string]byte {
	t.Helper()
	payload := wasmSection(t, mod, 7)
	if payload == nil {
		t.Fatalf("missing export section")
	}
	pos := 0
	count := int(readULEBForTest(t, payload, &pos))
	out := map[string]byte{}
	for i := 0; i < count; i++ {
		name := readWASMNameForTest(t, payload, &pos)
		if pos >= len(payload) {
			t.Fatalf("truncated export kind")
		}
		kind := payload[pos]
		pos++
		_ = readULEBForTest(t, payload, &pos)
		out[name] = kind
	}
	return out
}

func wasmSection(t *testing.T, mod []byte, wantID byte) []byte {
	t.Helper()
	if len(mod) < 8 || !bytes.Equal(mod[:4], []byte{0x00, 0x61, 0x73, 0x6d}) {
		t.Fatalf("invalid wasm module")
	}
	pos := 8
	for pos < len(mod) {
		id := mod[pos]
		pos++
		size := int(readULEBForTest(t, mod, &pos))
		if pos+size > len(mod) {
			t.Fatalf("truncated section %d", id)
		}
		payload := mod[pos : pos+size]
		if id == wantID {
			return payload
		}
		pos += size
	}
	return nil
}

func readWASMNameForTest(t *testing.T, b []byte, pos *int) string {
	t.Helper()
	n := int(readULEBForTest(t, b, pos))
	if *pos+n > len(b) {
		t.Fatalf("truncated wasm name")
	}
	name := string(b[*pos : *pos+n])
	*pos += n
	return name
}

func readULEBForTest(t *testing.T, b []byte, pos *int) uint32 {
	t.Helper()
	var out uint32
	var shift uint
	for {
		if *pos >= len(b) {
			t.Fatalf("truncated uleb")
		}
		ch := b[*pos]
		*pos++
		out |= uint32(ch&0x7f) << shift
		if ch&0x80 == 0 {
			return out
		}
		shift += 7
		if shift > 28 {
			t.Fatalf("uleb too large")
		}
	}
}

func stringSetHas(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
