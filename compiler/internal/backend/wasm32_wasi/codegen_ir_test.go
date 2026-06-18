package wasm32_wasi

import (
	"bytes"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestLinkObjectSupportsIslandInstructionsInBuildOnlyMode(t *testing.T) {
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
				{Kind: ir.IRIslandReset},
				{Kind: ir.IRStoreLocal, Local: 0},
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

func TestLinkObjectWASIOutputIsDeterministic(t *testing.T) {
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
		t.Fatalf("WASI module output is not deterministic")
	}
}

func TestLinkObjectWASIHeapBaseIsAlignedAfterStaticData(t *testing.T) {
	obj, err := CodegenObject([]ir.IRFunc{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRStrLit, Str: []byte("x")},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRStoreLocal, Local: 0},
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

	const wantStaticEnd = dataBase + 1
	if got := wasmDataSegmentEnd(t, mod); got != wantStaticEnd {
		t.Fatalf("static data end = 0x%x, want 0x%x", got, wantStaticEnd)
	}
	heapBase := wasmI32GlobalInits(t, mod)[0]
	if heapBase != 0x1010 {
		t.Fatalf("heap base = 0x%x, want first 16-byte aligned offset 0x1010", heapBase)
	}
	if heapBase%16 != 0 {
		t.Fatalf("heap base = 0x%x, want 16-byte alignment", heapBase)
	}
	if memoryBytes := wasmMemoryMinPages(t, mod) * wasmPageSize; uint32(heapBase) > memoryBytes {
		t.Fatalf("heap base 0x%x exceeds initial memory size 0x%x", heapBase, memoryBytes)
	}
}

func TestLinkObjectRejectsMissingEntryFunction(t *testing.T) {
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
	if got := err.Error(); !bytes.Contains(
		[]byte(got),
		[]byte("entry function 'missing' not found"),
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkObjectRejectsMultiSlotEntryFunction(t *testing.T) {
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
	if got := err.Error(); !bytes.Contains(
		[]byte(got),
		[]byte("entry function 'main' must return exactly 1 slot"),
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkObjectSupportsControlFlowAndI32ArrayIR(t *testing.T) {
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

func TestLinkObjectWASIRejectsControlFlowNonZeroStack(t *testing.T) {
	cases := []struct {
		name  string
		instr []ir.IRInstr
		want  string
	}{
		{
			name: "label entry",
			instr: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRLabel, Label: 10},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
			want: "unsupported non-zero stack at label 10",
		},
		{
			name: "block fallthrough",
			instr: []ir.IRInstr{
				{Kind: ir.IRLabel, Label: 10},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			want: "unsupported non-zero stack at block fallthrough",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obj, err := CodegenObject([]ir.IRFunc{
				{
					Name:        "main",
					ParamSlots:  0,
					LocalSlots:  0,
					ReturnSlots: 1,
					Instrs:      tc.instr,
				},
			}, "main")
			if err != nil {
				t.Fatalf("CodegenObject: %v", err)
			}
			_, err = LinkObject(obj)
			if err == nil {
				t.Fatalf("expected non-zero stack verifier error")
			}
			if got := err.Error(); !bytes.Contains([]byte(got), []byte(tc.want)) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestLinkObjectSupportsU8U16ArrayIR(t *testing.T) {
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

func TestLinkObjectMakeSliceLengthContractGuards(t *testing.T) {
	obj, err := CodegenObject([]ir.IRFunc{{
		Name:        "main",
		ParamSlots:  0,
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 536870912},
			{Kind: ir.IRMakeSliceI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}}, "main")
	if err != nil {
		t.Fatalf("CodegenObject: %v", err)
	}
	mod, err := LinkObject(obj)
	if err != nil {
		t.Fatalf("LinkObject: %v", err)
	}
	for _, want := range [][]byte{
		{0x48, 0x04, 0x40, 0x00, 0x0b}, // i32.lt_s; if; unreachable; end
		{0x45, 0x04, 0x40},             // i32.eqz; if zero empty-slice path
		{0x4a, 0x04, 0x40, 0x00, 0x0b}, // i32.gt_s; if; unreachable; end
	} {
		if !bytes.Contains(mod, want) {
			t.Fatalf("wasm make_slice length contract missing % x in module:\n% x", want, mod)
		}
	}
}

func TestLinkObjectRawSliceFromPartsBuildsScopedView(t *testing.T) {
	obj, err := CodegenObject([]ir.IRFunc{{
		Name:        "main",
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRRawSliceFromParts},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}}, "main")
	if err != nil {
		t.Fatalf("CodegenObject: %v", err)
	}
	mod, err := LinkObject(obj)
	if err != nil {
		t.Fatalf("LinkObject: %v", err)
	}
	viewProjection := []byte{0x21, 0x06, 0x21, 0x03, 0x21, 0x02, 0x20, 0x02, 0x20, 0x03}
	if !bytes.Contains(mod, viewProjection) {
		t.Fatalf(
			"WASI raw_slice_from_parts missing scoped view projection % x in module:\n% x",
			viewProjection,
			mod,
		)
	}
}

func TestLinkObjectIslandMakeSliceLengthContractGuards(t *testing.T) {
	obj, err := CodegenObject([]ir.IRFunc{{
		Name:        "main",
		ParamSlots:  0,
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 4096},
			{Kind: ir.IRConstI32, Imm: 536870912},
			{Kind: ir.IRIslandMakeSliceI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}}, "main")
	if err != nil {
		t.Fatalf("CodegenObject: %v", err)
	}
	mod, err := LinkObject(obj)
	if err != nil {
		t.Fatalf("LinkObject: %v", err)
	}
	for _, want := range [][]byte{
		{0x48, 0x04, 0x40, 0x00, 0x0b},
		{0x45, 0x04, 0x40},
		{0x4a, 0x04, 0x40, 0x00, 0x0b},
	} {
		if !bytes.Contains(mod, want) {
			t.Fatalf(
				"wasm island_make_slice length contract missing % x in module:\n% x",
				want,
				mod,
			)
		}
	}
}
