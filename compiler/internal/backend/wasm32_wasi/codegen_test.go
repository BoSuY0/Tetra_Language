package wasm32_wasi

import (
	"bytes"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestLinkObjectWritesWASMHeaderAndStartExport(t *testing.T) {
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
	if !bytes.Contains(mod, []byte("_start")) {
		t.Fatalf("missing _start export")
	}
}

func TestLinkObjectWASIImportExportShape(t *testing.T) {
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
	if got := imports["wasi_snapshot_preview1"]; !stringSetHas(got, "fd_write") || !stringSetHas(got, "proc_exit") {
		t.Fatalf("WASI imports = %#v", imports)
	}
	if _, ok := imports["tetra_web_v0.4.0"]; ok {
		t.Fatalf("WASI module imported web host namespace: %#v", imports)
	}

	exports := wasmExports(t, mod)
	if exports["memory"] != 0x02 {
		t.Fatalf("memory export kind = 0x%x, want memory", exports["memory"])
	}
	if exports["_start"] != 0x00 {
		t.Fatalf("_start export kind = 0x%x, want func", exports["_start"])
	}
	if _, ok := exports["tetra_main"]; ok {
		t.Fatalf("WASI module exported tetra_main: %#v", exports)
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

func TestCodegenObjectRejectsInvalidFunctionMetadata(t *testing.T) {
	cases := []struct {
		name  string
		funcs []ir.IRFunc
		want  string
	}{
		{
			name: "empty name",
			funcs: []ir.IRFunc{
				wasmIRFuncForMetadataTest("", 0, 0, 1),
			},
			want: "function name is empty",
		},
		{
			name: "duplicate name",
			funcs: []ir.IRFunc{
				wasmIRFuncForMetadataTest("main", 0, 0, 1),
				wasmIRFuncForMetadataTest("main", 0, 0, 1),
			},
			want: "duplicate function 'main'",
		},
		{
			name: "negative params",
			funcs: []ir.IRFunc{
				wasmIRFuncForMetadataTest("main", -1, 0, 1),
			},
			want: "function 'main' has invalid slots",
		},
		{
			name: "negative locals",
			funcs: []ir.IRFunc{
				wasmIRFuncForMetadataTest("main", 0, -1, 1),
			},
			want: "function 'main' has invalid slots",
		},
		{
			name: "negative returns",
			funcs: []ir.IRFunc{
				wasmIRFuncForMetadataTest("main", 0, 0, -1),
			},
			want: "function 'main' has invalid slots",
		},
		{
			name: "params exceed locals",
			funcs: []ir.IRFunc{
				wasmIRFuncForMetadataTest("main", 2, 1, 1),
			},
			want: "function 'main' has invalid slots",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CodegenObject(tc.funcs, "main")
			if err == nil {
				t.Fatalf("expected function metadata diagnostic")
			}
			if got := err.Error(); !bytes.Contains([]byte(got), []byte(tc.want)) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestLinkObjectRejectsInvalidFunctionMetadata(t *testing.T) {
	cases := []struct {
		name  string
		funcs []Function
		want  string
	}{
		{
			name: "empty name",
			funcs: []Function{
				wasmObjectFunctionForMetadataTest("", 0, 0, 1),
			},
			want: "function name is empty",
		},
		{
			name: "duplicate name",
			funcs: []Function{
				wasmObjectFunctionForMetadataTest("main", 0, 0, 1),
				wasmObjectFunctionForMetadataTest("main", 0, 0, 1),
			},
			want: "duplicate function 'main'",
		},
		{
			name: "negative params",
			funcs: []Function{
				wasmObjectFunctionForMetadataTest("main", -1, 0, 1),
			},
			want: "function 'main' has invalid slots",
		},
		{
			name: "negative locals",
			funcs: []Function{
				wasmObjectFunctionForMetadataTest("main", 0, -1, 1),
			},
			want: "function 'main' has invalid slots",
		},
		{
			name: "negative returns",
			funcs: []Function{
				wasmObjectFunctionForMetadataTest("main", 0, 0, -1),
			},
			want: "function 'main' has invalid slots",
		},
		{
			name: "params exceed locals",
			funcs: []Function{
				wasmObjectFunctionForMetadataTest("main", 2, 1, 1),
			},
			want: "function 'main' has invalid slots",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LinkObject(&Object{MainName: "main", Functions: tc.funcs})
			if err == nil {
				t.Fatalf("expected function metadata diagnostic")
			}
			if got := err.Error(); !bytes.Contains([]byte(got), []byte(tc.want)) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestCodegenObjectRejectsInvalidCallMetadata(t *testing.T) {
	cases := []struct {
		name string
		call ir.IRInstr
		want string
	}{
		{
			name: "missing target",
			call: ir.IRInstr{Kind: ir.IRCall, ArgSlots: 0, RetSlots: 0},
			want: "call is missing target name",
		},
		{
			name: "negative args",
			call: ir.IRInstr{Kind: ir.IRCall, Name: "helper", ArgSlots: -1, RetSlots: 1},
			want: `call "helper" has negative ABI slots`,
		},
		{
			name: "negative returns",
			call: ir.IRInstr{Kind: ir.IRCall, Name: "helper", ArgSlots: 0, RetSlots: -1},
			want: `call "helper" has negative ABI slots`,
		},
		{
			name: "unknown target before stack simulation",
			call: ir.IRInstr{Kind: ir.IRCall, Name: "missing", ArgSlots: 1, RetSlots: 0},
			want: `calls unsupported symbol 'missing'`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CodegenObject(wasmIRFuncsWithCallForMetadataTest(tc.call), "main")
			if err == nil {
				t.Fatalf("expected call metadata diagnostic")
			}
			if got := err.Error(); !bytes.Contains([]byte(got), []byte(tc.want)) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestLinkObjectRejectsInvalidCallMetadata(t *testing.T) {
	cases := []struct {
		name string
		call ir.IRInstr
		want string
	}{
		{
			name: "missing target",
			call: ir.IRInstr{Kind: ir.IRCall, ArgSlots: 0, RetSlots: 0},
			want: "call is missing target name",
		},
		{
			name: "negative args",
			call: ir.IRInstr{Kind: ir.IRCall, Name: "helper", ArgSlots: -1, RetSlots: 1},
			want: `call "helper" has negative ABI slots`,
		},
		{
			name: "negative returns",
			call: ir.IRInstr{Kind: ir.IRCall, Name: "helper", ArgSlots: 0, RetSlots: -1},
			want: `call "helper" has negative ABI slots`,
		},
		{
			name: "unknown target before stack simulation",
			call: ir.IRInstr{Kind: ir.IRCall, Name: "missing", ArgSlots: 1, RetSlots: 0},
			want: `calls unsupported symbol 'missing'`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LinkObject(&Object{
				MainName:  "main",
				Functions: wasmObjectFunctionsWithCallForMetadataTest(tc.call),
			})
			if err == nil {
				t.Fatalf("expected call metadata diagnostic")
			}
			if got := err.Error(); !bytes.Contains([]byte(got), []byte(tc.want)) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestCodegenObjectRejectsLocalSlotOutOfBounds(t *testing.T) {
	cases := []struct {
		name  string
		instr ir.IRInstr
		want  string
	}{
		{
			name:  "load negative",
			instr: ir.IRInstr{Kind: ir.IRLoadLocal, Local: -1},
			want:  "local slot -1 out of bounds (locals=1)",
		},
		{
			name:  "load one past",
			instr: ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
			want:  "local slot 1 out of bounds (locals=1)",
		},
		{
			name:  "store negative",
			instr: ir.IRInstr{Kind: ir.IRStoreLocal, Local: -1},
			want:  "local slot -1 out of bounds (locals=1)",
		},
		{
			name:  "store one past",
			instr: ir.IRInstr{Kind: ir.IRStoreLocal, Local: 1},
			want:  "local slot 1 out of bounds (locals=1)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CodegenObject(wasmIRFuncsWithLocalForMetadataTest(tc.instr), "main")
			if err == nil {
				t.Fatalf("expected local slot diagnostic")
			}
			if got := err.Error(); !bytes.Contains([]byte(got), []byte(tc.want)) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestCodegenObjectRejectsCallSignatureMismatch(t *testing.T) {
	_, err := CodegenObject(wasmIRFuncsWithCallSignatureMismatchTest(), "main")
	if err == nil {
		t.Fatalf("expected call signature diagnostic")
	}
	if got := err.Error(); !bytes.Contains([]byte(got), []byte(`call "helper" ABI mismatch`)) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkObjectRejectsCallSignatureMismatch(t *testing.T) {
	_, err := LinkObject(&Object{
		MainName:  "main",
		Functions: wasmObjectFunctionsWithCallSignatureMismatchTest(),
	})
	if err == nil {
		t.Fatalf("expected call signature diagnostic")
	}
	if got := err.Error(); !bytes.Contains([]byte(got), []byte(`call "helper" ABI mismatch`)) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkObjectRejectsLocalSlotOutOfBounds(t *testing.T) {
	cases := []struct {
		name  string
		instr ir.IRInstr
		want  string
	}{
		{
			name:  "load negative",
			instr: ir.IRInstr{Kind: ir.IRLoadLocal, Local: -1},
			want:  "local slot -1 out of bounds (locals=1)",
		},
		{
			name:  "load one past",
			instr: ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
			want:  "local slot 1 out of bounds (locals=1)",
		},
		{
			name:  "store negative",
			instr: ir.IRInstr{Kind: ir.IRStoreLocal, Local: -1},
			want:  "local slot -1 out of bounds (locals=1)",
		},
		{
			name:  "store one past",
			instr: ir.IRInstr{Kind: ir.IRStoreLocal, Local: 1},
			want:  "local slot 1 out of bounds (locals=1)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LinkObject(&Object{
				MainName:  "main",
				Functions: wasmObjectFunctionsWithLocalForMetadataTest(tc.instr),
			})
			if err == nil {
				t.Fatalf("expected local slot diagnostic")
			}
			if got := err.Error(); !bytes.Contains([]byte(got), []byte(tc.want)) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestCodegenObjectRejectsInvalidLabelMetadata(t *testing.T) {
	cases := []struct {
		name   string
		instrs []ir.IRInstr
		want   string
	}{
		{
			name: "unknown jump label",
			instrs: []ir.IRInstr{
				{Kind: ir.IRJmp, Label: 99},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
			want: "unknown label 99",
		},
		{
			name: "unknown conditional jump label",
			instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRJmpIfZero, Label: 99},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
			want: "unknown label 99",
		},
		{
			name: "duplicate label",
			instrs: []ir.IRInstr{
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
			want: "duplicate label 1",
		},
		{
			name: "negative label",
			instrs: []ir.IRInstr{
				{Kind: ir.IRJmp, Label: -1},
				{Kind: ir.IRLabel, Label: -1},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
			want: "negative label -1",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CodegenObject(wasmIRFuncsWithLabelMetadataTest(tc.instrs), "main")
			if err == nil {
				t.Fatalf("expected label metadata diagnostic")
			}
			if got := err.Error(); !bytes.Contains([]byte(got), []byte(tc.want)) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestLinkObjectRejectsInvalidLabelMetadata(t *testing.T) {
	cases := []struct {
		name   string
		instrs []ir.IRInstr
		want   string
	}{
		{
			name: "unknown jump label",
			instrs: []ir.IRInstr{
				{Kind: ir.IRJmp, Label: 99},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
			want: "unknown label 99",
		},
		{
			name: "unknown conditional jump label",
			instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRJmpIfZero, Label: 99},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
			want: "unknown label 99",
		},
		{
			name: "duplicate label",
			instrs: []ir.IRInstr{
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
			want: "duplicate label 1",
		},
		{
			name: "negative label",
			instrs: []ir.IRInstr{
				{Kind: ir.IRJmp, Label: -1},
				{Kind: ir.IRLabel, Label: -1},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
			want: "negative label -1",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LinkObject(&Object{
				MainName:  "main",
				Functions: wasmObjectFunctionsWithLabelMetadataTest(tc.instrs),
			})
			if err == nil {
				t.Fatalf("expected label metadata diagnostic")
			}
			if got := err.Error(); !bytes.Contains([]byte(got), []byte(tc.want)) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestLinkObjectSupportsNegI32(t *testing.T) {
	obj, err := CodegenObject([]ir.IRFunc{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRNegI32},
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
	if !bytes.Contains(mod, []byte{0x41, 0x7f, 0x6c}) {
		t.Fatalf("missing i32.const -1 + i32.mul negation sequence")
	}
}

func TestCodegenObjectRejectsNegativeGlobalSlots(t *testing.T) {
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

func TestLinkObjectRejectsGlobalSlotBeyondObjectCount(t *testing.T) {
	_, err := LinkObject(&Object{
		MainName:    "main",
		GlobalSlots: 1,
		Functions: []Function{
			{
				Name:        "main",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRLoadGlobal, Local: 1},
					{Kind: ir.IRReturn},
				},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected global slot count diagnostic")
	}
	if got := err.Error(); !bytes.Contains([]byte(got), []byte("global slot 1 in function 'main' exceeds object global slot count 1")) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkObjectEmitsDeclaredGlobalSlotAccess(t *testing.T) {
	mod, err := LinkObject(&Object{
		MainName:    "main",
		GlobalSlots: 2,
		GlobalInits: []int32{7, 9},
		Functions: []Function{
			{
				Name:        "main",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 42},
					{Kind: ir.IRStoreGlobal, Local: 1},
					{Kind: ir.IRLoadGlobal, Local: 1},
					{Kind: ir.IRReturn},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("LinkObject: %v", err)
	}
	inits := wasmI32GlobalInits(t, mod)
	if len(inits) != 3 {
		t.Fatalf("global init count = %d, want heap plus 2 lowered globals", len(inits))
	}
	if inits[1] != 7 || inits[2] != 9 {
		t.Fatalf("lowered global inits = %v, want slots 7 and 9", inits[1:])
	}
	if !bytes.Contains(mod, []byte{0x41, 0x2a, 0x24, 0x02, 0x23, 0x02, 0x0f}) {
		t.Fatalf("missing global.set/global.get sequence for lowered slot 1")
	}
}

func TestLinkObjectAllowsRepeatedSymAddrSymbol(t *testing.T) {
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

func TestCodegenObjectAllowsRepeatedSymAddrSymbolAcrossFunctions(t *testing.T) {
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

func TestCodegenObjectRejectsMissingSymAddrName(t *testing.T) {
	_, err := CodegenObject([]ir.IRFunc{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRSymAddr},
				{Kind: ir.IRReturn},
			},
		},
	}, "main")
	if err == nil {
		t.Fatalf("expected missing symbol address name diagnostic")
	}
	if got := err.Error(); !bytes.Contains([]byte(got), []byte("symbol address is missing name")) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkObjectRejectsMissingSymAddrName(t *testing.T) {
	_, err := LinkObject(&Object{
		MainName: "main",
		Functions: []Function{
			{
				Name:        "main",
				ParamSlots:  0,
				LocalSlots:  0,
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRSymAddr},
					{Kind: ir.IRReturn},
				},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected missing symbol address name diagnostic")
	}
	if got := err.Error(); !bytes.Contains([]byte(got), []byte("symbol address is missing name")) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCodegenObjectRejectsSymAddrTokenCollision(t *testing.T) {
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

func TestLinkObjectRejectsSymAddrTokenCollision(t *testing.T) {
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

func wasmIRFuncForMetadataTest(name string, params int, locals int, returns int) ir.IRFunc {
	return ir.IRFunc{
		Name:        name,
		ParamSlots:  params,
		LocalSlots:  locals,
		ReturnSlots: returns,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}

func wasmObjectFunctionForMetadataTest(name string, params int, locals int, returns int) Function {
	return Function{
		Name:        name,
		ParamSlots:  params,
		LocalSlots:  locals,
		ReturnSlots: returns,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}

func wasmIRFuncsWithCallForMetadataTest(call ir.IRInstr) []ir.IRFunc {
	return []ir.IRFunc{
		{
			Name:        "helper",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				call,
				{Kind: ir.IRReturn},
			},
		},
	}
}

func wasmObjectFunctionsWithCallForMetadataTest(call ir.IRInstr) []Function {
	return []Function{
		{
			Name:        "helper",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				call,
				{Kind: ir.IRReturn},
			},
		},
	}
}

func wasmIRFuncsWithCallSignatureMismatchTest() []ir.IRFunc {
	return []ir.IRFunc{
		{
			Name:        "helper",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRCall, Name: "helper", ArgSlots: 0, RetSlots: 1},
				{Kind: ir.IRReturn},
			},
		},
	}
}

func wasmObjectFunctionsWithCallSignatureMismatchTest() []Function {
	return []Function{
		{
			Name:        "helper",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRCall, Name: "helper", ArgSlots: 0, RetSlots: 1},
				{Kind: ir.IRReturn},
			},
		},
	}
}

func wasmIRFuncsWithLocalForMetadataTest(instr ir.IRInstr) []ir.IRFunc {
	return []ir.IRFunc{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs:      wasmInstrsWithLocalForMetadataTest(instr),
		},
	}
}

func wasmObjectFunctionsWithLocalForMetadataTest(instr ir.IRInstr) []Function {
	return []Function{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs:      wasmInstrsWithLocalForMetadataTest(instr),
		},
	}
}

func wasmInstrsWithLocalForMetadataTest(instr ir.IRInstr) []ir.IRInstr {
	if instr.Kind == ir.IRStoreLocal {
		return []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 42},
			instr,
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		}
	}
	return []ir.IRInstr{
		instr,
		{Kind: ir.IRReturn},
	}
}

func wasmIRFuncsWithLabelMetadataTest(instrs []ir.IRInstr) []ir.IRFunc {
	return []ir.IRFunc{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs:      instrs,
		},
	}
}

func wasmObjectFunctionsWithLabelMetadataTest(instrs []ir.IRInstr) []Function {
	return []Function{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs:      instrs,
		},
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

func wasmMemoryMinPages(t *testing.T, mod []byte) uint32 {
	t.Helper()
	payload := wasmSection(t, mod, 5)
	if payload == nil {
		t.Fatalf("missing memory section")
	}
	pos := 0
	count := int(readULEBForTest(t, payload, &pos))
	if count != 1 {
		t.Fatalf("memory count = %d, want 1", count)
	}
	if pos >= len(payload) {
		t.Fatalf("truncated memory limits")
	}
	flags := payload[pos]
	pos++
	if flags != 0x00 {
		t.Fatalf("memory limits flags = 0x%x, want min-only", flags)
	}
	return readULEBForTest(t, payload, &pos)
}

func wasmI32GlobalInits(t *testing.T, mod []byte) []int32 {
	t.Helper()
	payload := wasmSection(t, mod, 6)
	if payload == nil {
		t.Fatalf("missing global section")
	}
	pos := 0
	count := int(readULEBForTest(t, payload, &pos))
	out := make([]int32, 0, count)
	for i := 0; i < count; i++ {
		if pos+3 > len(payload) {
			t.Fatalf("truncated global %d", i)
		}
		valueType := payload[pos]
		pos++
		mutable := payload[pos]
		pos++
		opcode := payload[pos]
		pos++
		if valueType != 0x7f || mutable != 0x01 || opcode != 0x41 {
			t.Fatalf("global %d header = type 0x%x mutable 0x%x opcode 0x%x, want mutable i32.const", i, valueType, mutable, opcode)
		}
		out = append(out, readSLEB32ForTest(t, payload, &pos))
		if pos >= len(payload) || payload[pos] != 0x0b {
			t.Fatalf("global %d missing init expr end", i)
		}
		pos++
	}
	return out
}

func wasmDataSegmentEnd(t *testing.T, mod []byte) uint32 {
	t.Helper()
	payload := wasmSection(t, mod, 11)
	if payload == nil {
		t.Fatalf("missing data section")
	}
	pos := 0
	count := int(readULEBForTest(t, payload, &pos))
	if count != 1 {
		t.Fatalf("data segment count = %d, want 1", count)
	}
	mode := readULEBForTest(t, payload, &pos)
	if mode != 0 {
		t.Fatalf("data segment mode = %d, want active memidx 0", mode)
	}
	if pos >= len(payload) || payload[pos] != 0x41 {
		t.Fatalf("data segment missing i32.const offset")
	}
	pos++
	offset := uint32(readSLEB32ForTest(t, payload, &pos))
	if pos >= len(payload) || payload[pos] != 0x0b {
		t.Fatalf("data segment missing offset expr end")
	}
	pos++
	size := readULEBForTest(t, payload, &pos)
	return offset + size
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

func readSLEB32ForTest(t *testing.T, b []byte, pos *int) int32 {
	t.Helper()
	var result int32
	var shift uint
	var ch byte
	for {
		if *pos >= len(b) {
			t.Fatalf("truncated sleb")
		}
		ch = b[*pos]
		*pos++
		result |= int32(ch&0x7f) << shift
		shift += 7
		if ch&0x80 == 0 {
			break
		}
		if shift >= 35 {
			t.Fatalf("sleb too large")
		}
	}
	if shift < 32 && ch&0x40 != 0 {
		result |= ^0 << shift
	}
	return result
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
