package x64core

import (
	"testing"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/ir"
)

func TestAtomicPointerExchangeAndFenceHonorConfiguredPointerWidth(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "__test_atomic_exchange_ptr",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRAtomicExchangePtr},
			{Kind: ir.IRAtomicFenceSeqCst},
			{Kind: ir.IRReturn},
		},
	}

	x32 := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), fn, x64.CodegenOptions{PointerWidthBits: 32})
	assertContainsBytes(t, "x32 atomic exchange guard width", x32, addEdxImm32Bytes(4))
	assertContainsBytes(t, "x32 atomic exchange 32-bit xchg", x32, []byte{0x44, 0x87, 0x07})
	assertNotContainsBytes(t, "x32 atomic exchange 64-bit xchg", x32, []byte{0x4C, 0x87, 0x07})
	assertContainsBytes(t, "x32 seq_cst fence", x32, []byte{0x0F, 0xAE, 0xF0})

	x64Code := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), fn, x64.CodegenOptions{PointerWidthBits: 64})
	assertContainsBytes(t, "x64 atomic exchange guard width", x64Code, addEdxImm32Bytes(8))
	assertContainsBytes(t, "x64 atomic exchange 64-bit xchg", x64Code, []byte{0x4C, 0x87, 0x07})
	assertNotContainsBytes(t, "x64 atomic exchange 32-bit xchg", x64Code, []byte{0x44, 0x87, 0x07})
	assertContainsBytes(t, "x64 seq_cst fence", x64Code, []byte{0x0F, 0xAE, 0xF0})
}

func TestAtomicNonSeqCstFencesAreExplicitNoOpsOnX64Family(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "__test_atomic_non_seq_cst_fences",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRAtomicFenceRelaxed},
			{Kind: ir.IRAtomicFenceAcquire},
			{Kind: ir.IRAtomicFenceRelease},
			{Kind: ir.IRAtomicFenceAcqRel},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}

	for _, pointerWidth := range []int{32, 64} {
		code := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), fn, x64.CodegenOptions{PointerWidthBits: pointerWidth})
		assertNotContainsBytes(t, "non-seq-cst fence must not emit mfence", code, []byte{0x0F, 0xAE, 0xF0})
	}
}

func TestAtomicPointerLoadAndStoreHonorConfiguredPointerWidth(t *testing.T) {
	loadFn := ir.IRFunc{
		Name:        "__test_atomic_load_ptr",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRAtomicLoadPtr},
			{Kind: ir.IRReturn},
		},
	}
	x32Load := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), loadFn, x64.CodegenOptions{PointerWidthBits: 32})
	assertContainsBytes(t, "x32 atomic load guard width", x32Load, addEdxImm32Bytes(4))
	wantLoad32 := &x64.Emitter{}
	wantLoad32.MovEaxFromRaxPtr()
	assertContainsBytes(t, "x32 atomic load 32-bit load", x32Load, wantLoad32.Buf)
	forbidLoad64 := &x64.Emitter{}
	forbidLoad64.MovRaxFromRdiDisp(0)
	assertNotContainsBytes(t, "x32 atomic load 64-bit load", x32Load, forbidLoad64.Buf)

	x64Load := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), loadFn, x64.CodegenOptions{PointerWidthBits: 64})
	assertContainsBytes(t, "x64 atomic load guard width", x64Load, addEdxImm32Bytes(8))
	assertContainsBytes(t, "x64 atomic load 64-bit load", x64Load, forbidLoad64.Buf)

	storeFn := ir.IRFunc{
		Name:        "__test_atomic_store_ptr",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAtomicStorePtr},
			{Kind: ir.IRReturn},
		},
	}
	x32Store := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), storeFn, x64.CodegenOptions{PointerWidthBits: 32})
	assertContainsBytes(t, "x32 atomic store guard width", x32Store, addEdxImm32Bytes(4))
	assertContainsBytes(t, "x32 atomic store zero-extends returned pointer", x32Store, []byte{0x45, 0x89, 0xC1})
	assertNotContainsBytes(t, "x32 atomic store 64-bit return copy", x32Store, []byte{0x4D, 0x89, 0xC1})
	assertContainsBytes(t, "x32 atomic store 32-bit xchg", x32Store, []byte{0x44, 0x87, 0x07})
	assertNotContainsBytes(t, "x32 atomic store 64-bit xchg", x32Store, []byte{0x4C, 0x87, 0x07})

	x64Store := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), storeFn, x64.CodegenOptions{PointerWidthBits: 64})
	assertContainsBytes(t, "x64 atomic store guard width", x64Store, addEdxImm32Bytes(8))
	assertContainsBytes(t, "x64 atomic store 64-bit xchg", x64Store, []byte{0x4C, 0x87, 0x07})
	assertNotContainsBytes(t, "x64 atomic store 32-bit xchg", x64Store, []byte{0x44, 0x87, 0x07})
}

func TestAtomicPointerCompareExchangeHonorsConfiguredPointerWidth(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "__test_atomic_compare_exchange_ptr",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAtomicCompareExchangePtr},
			{Kind: ir.IRReturn},
		},
	}

	x32 := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), fn, x64.CodegenOptions{PointerWidthBits: 32})
	assertContainsBytes(t, "x32 atomic cas guard width", x32, addEdxImm32Bytes(4))
	assertContainsBytes(t, "x32 atomic cas zero-extends expected pointer into accumulator", x32, []byte{0x44, 0x89, 0xC8})
	assertNotContainsBytes(t, "x32 atomic cas 64-bit expected pointer copy", x32, []byte{0x4C, 0x89, 0xC8})
	assertContainsBytes(t, "x32 atomic cas 32-bit lock cmpxchg", x32, []byte{0xF0, 0x44, 0x0F, 0xB1, 0x07})
	assertNotContainsBytes(t, "x32 atomic cas 64-bit lock cmpxchg", x32, []byte{0xF0, 0x4C, 0x0F, 0xB1, 0x07})

	x64Code := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), fn, x64.CodegenOptions{PointerWidthBits: 64})
	assertContainsBytes(t, "x64 atomic cas guard width", x64Code, addEdxImm32Bytes(8))
	assertContainsBytes(t, "x64 atomic cas 64-bit lock cmpxchg", x64Code, []byte{0xF0, 0x4C, 0x0F, 0xB1, 0x07})
	assertNotContainsBytes(t, "x64 atomic cas 32-bit lock cmpxchg", x64Code, []byte{0xF0, 0x44, 0x0F, 0xB1, 0x07})
}

func TestAtomicPointerFetchAddHonorsConfiguredPointerWidth(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "__test_atomic_fetch_add_ptr",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 5},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAtomicFetchAddPtr},
			{Kind: ir.IRReturn},
		},
	}

	x32 := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), fn, x64.CodegenOptions{PointerWidthBits: 32})
	assertContainsBytes(t, "x32 atomic fetch_add guard width", x32, addEdxImm32Bytes(4))
	assertContainsBytes(t, "x32 atomic fetch_add 32-bit lock xadd", x32, []byte{0xF0, 0x44, 0x0F, 0xC1, 0x07})
	assertNotContainsBytes(t, "x32 atomic fetch_add 64-bit lock xadd", x32, []byte{0xF0, 0x4C, 0x0F, 0xC1, 0x07})

	x64Code := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), fn, x64.CodegenOptions{PointerWidthBits: 64})
	assertContainsBytes(t, "x64 atomic fetch_add guard width", x64Code, addEdxImm32Bytes(8))
	assertContainsBytes(t, "x64 atomic fetch_add 64-bit lock xadd", x64Code, []byte{0xF0, 0x4C, 0x0F, 0xC1, 0x07})
	assertNotContainsBytes(t, "x64 atomic fetch_add 32-bit lock xadd", x64Code, []byte{0xF0, 0x44, 0x0F, 0xC1, 0x07})
}

func TestAtomicPointerFetchSubHonorsConfiguredPointerWidth(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "__test_atomic_fetch_sub_ptr",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 5},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAtomicFetchSubPtr},
			{Kind: ir.IRReturn},
		},
	}

	x32 := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), fn, x64.CodegenOptions{PointerWidthBits: 32})
	assertContainsBytes(t, "x32 atomic fetch_sub guard width", x32, addEdxImm32Bytes(4))
	assertContainsBytes(t, "x32 atomic fetch_sub neg r8d", x32, []byte{0x41, 0xF7, 0xD8})
	assertContainsBytes(t, "x32 atomic fetch_sub 32-bit lock xadd", x32, []byte{0xF0, 0x44, 0x0F, 0xC1, 0x07})
	assertNotContainsBytes(t, "x32 atomic fetch_sub 64-bit neg r8", x32, []byte{0x49, 0xF7, 0xD8})
	assertNotContainsBytes(t, "x32 atomic fetch_sub 64-bit lock xadd", x32, []byte{0xF0, 0x4C, 0x0F, 0xC1, 0x07})

	x64Code := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), fn, x64.CodegenOptions{PointerWidthBits: 64})
	assertContainsBytes(t, "x64 atomic fetch_sub guard width", x64Code, addEdxImm32Bytes(8))
	assertContainsBytes(t, "x64 atomic fetch_sub neg r8", x64Code, []byte{0x49, 0xF7, 0xD8})
	assertContainsBytes(t, "x64 atomic fetch_sub 64-bit lock xadd", x64Code, []byte{0xF0, 0x4C, 0x0F, 0xC1, 0x07})
	assertNotContainsBytes(t, "x64 atomic fetch_sub 32-bit neg r8d", x64Code, []byte{0x41, 0xF7, 0xD8})
	assertNotContainsBytes(t, "x64 atomic fetch_sub 32-bit lock xadd", x64Code, []byte{0xF0, 0x44, 0x0F, 0xC1, 0x07})
}

func TestAtomicPointerLogicalFetchOpsUseCASLoopWithConfiguredPointerWidth(t *testing.T) {
	cases := []struct {
		name  string
		kind  ir.IRInstrKind
		op32  []byte
		op64  []byte
		label string
	}{
		{name: "and", kind: ir.IRAtomicFetchAndPtr, op32: []byte{0x45, 0x21, 0xC2}, op64: []byte{0x4D, 0x21, 0xC2}, label: "and r10,r8"},
		{name: "or", kind: ir.IRAtomicFetchOrPtr, op32: []byte{0x45, 0x09, 0xC2}, op64: []byte{0x4D, 0x09, 0xC2}, label: "or r10,r8"},
		{name: "xor", kind: ir.IRAtomicFetchXorPtr, op32: []byte{0x45, 0x31, 0xC2}, op64: []byte{0x4D, 0x31, 0xC2}, label: "xor r10,r8"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn := ir.IRFunc{
				Name:        "__test_atomic_fetch_" + tc.name + "_ptr",
				ParamSlots:  0,
				LocalSlots:  0,
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRConstI32, Imm: 5},
					{Kind: ir.IRConstI32, Imm: 1},
					{Kind: tc.kind},
					{Kind: ir.IRReturn},
				},
			}

			x32 := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), fn, x64.CodegenOptions{PointerWidthBits: 32})
			assertContainsBytes(t, "x32 atomic fetch_"+tc.name+" guard width", x32, addEdxImm32Bytes(4))
			assertContainsBytes(t, "x32 atomic fetch_"+tc.name+" load", x32, []byte{0x8B, 0x07})
			assertContainsBytes(t, "x32 atomic fetch_"+tc.name+" mov r10d,eax", x32, []byte{0x41, 0x89, 0xC2})
			assertContainsBytes(t, "x32 atomic fetch_"+tc.name+" "+tc.label, x32, tc.op32)
			assertContainsBytes(t, "x32 atomic fetch_"+tc.name+" lock cmpxchg r10d", x32, []byte{0xF0, 0x44, 0x0F, 0xB1, 0x17})
			assertContainsBytes(t, "x32 atomic fetch_"+tc.name+" retry branch", x32, []byte{0x0F, 0x85})
			assertNotContainsBytes(t, "x32 atomic fetch_"+tc.name+" qword cmpxchg", x32, []byte{0xF0, 0x4C, 0x0F, 0xB1, 0x17})

			x64Code := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), fn, x64.CodegenOptions{PointerWidthBits: 64})
			assertContainsBytes(t, "x64 atomic fetch_"+tc.name+" guard width", x64Code, addEdxImm32Bytes(8))
			wantLoad64 := &x64.Emitter{}
			wantLoad64.MovRaxFromRdiDisp(0)
			assertContainsBytes(t, "x64 atomic fetch_"+tc.name+" load", x64Code, wantLoad64.Buf)
			assertContainsBytes(t, "x64 atomic fetch_"+tc.name+" mov r10,rax", x64Code, []byte{0x49, 0x89, 0xC2})
			assertContainsBytes(t, "x64 atomic fetch_"+tc.name+" "+tc.label, x64Code, tc.op64)
			assertContainsBytes(t, "x64 atomic fetch_"+tc.name+" lock cmpxchg r10", x64Code, []byte{0xF0, 0x4C, 0x0F, 0xB1, 0x17})
			assertContainsBytes(t, "x64 atomic fetch_"+tc.name+" retry branch", x64Code, []byte{0x0F, 0x85})
			assertNotContainsBytes(t, "x64 atomic fetch_"+tc.name+" dword cmpxchg", x64Code, []byte{0xF0, 0x44, 0x0F, 0xB1, 0x17})
		})
	}
}

func TestAtomicI32OpsUseDwordCodegenRegardlessOfPointerWidth(t *testing.T) {
	cases := []struct {
		name      string
		kind      ir.IRInstrKind
		stack     []ir.IRInstr
		wantBytes [][]byte
	}{
		{
			name: "load",
			kind: ir.IRAtomicLoadI32,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x8B, 0x00}},
		},
		{
			name: "store",
			kind: ir.IRAtomicStoreI32,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x44, 0x87, 0x07}},
		},
		{
			name: "exchange",
			kind: ir.IRAtomicExchangeI32,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x44, 0x87, 0x07}},
		},
		{
			name: "compare_exchange",
			kind: ir.IRAtomicCompareExchangeI32,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRConstI32, Imm: 9},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0xF0, 0x44, 0x0F, 0xB1, 0x07}},
		},
		{
			name: "fetch_add",
			kind: ir.IRAtomicFetchAddI32,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0xF0, 0x44, 0x0F, 0xC1, 0x07}},
		},
		{
			name: "fetch_sub",
			kind: ir.IRAtomicFetchSubI32,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x41, 0xF7, 0xD8}, {0xF0, 0x44, 0x0F, 0xC1, 0x07}},
		},
		{
			name: "fetch_and",
			kind: ir.IRAtomicFetchAndI32,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x8B, 0x07}, {0x45, 0x21, 0xC2}, {0xF0, 0x44, 0x0F, 0xB1, 0x17}, {0x0F, 0x85}},
		},
		{
			name: "fetch_or",
			kind: ir.IRAtomicFetchOrI32,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x8B, 0x07}, {0x45, 0x09, 0xC2}, {0xF0, 0x44, 0x0F, 0xB1, 0x17}, {0x0F, 0x85}},
		},
		{
			name: "fetch_xor",
			kind: ir.IRAtomicFetchXorI32,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x8B, 0x07}, {0x45, 0x31, 0xC2}, {0xF0, 0x44, 0x0F, 0xB1, 0x17}, {0x0F, 0x85}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			instrs := append([]ir.IRInstr{}, tc.stack...)
			instrs = append(instrs, ir.IRInstr{Kind: tc.kind}, ir.IRInstr{Kind: ir.IRReturn})
			fn := ir.IRFunc{
				Name:        "__test_atomic_i32_" + tc.name,
				ParamSlots:  0,
				LocalSlots:  0,
				ReturnSlots: 1,
				Instrs:      instrs,
			}

			for _, pointerWidth := range []int{32, 64} {
				code := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), fn, x64.CodegenOptions{PointerWidthBits: pointerWidth})
				assertContainsBytes(t, "atomic i32 "+tc.name+" guard width", code, addEdxImm32Bytes(4))
				for _, want := range tc.wantBytes {
					assertContainsBytes(t, "atomic i32 "+tc.name, code, want)
				}
				assertNotContainsBytes(t, "atomic i32 "+tc.name+" qword xadd", code, []byte{0xF0, 0x4C, 0x0F, 0xC1, 0x07})
				assertNotContainsBytes(t, "atomic i32 "+tc.name+" qword cmpxchg r8", code, []byte{0xF0, 0x4C, 0x0F, 0xB1, 0x07})
				assertNotContainsBytes(t, "atomic i32 "+tc.name+" qword cmpxchg r10", code, []byte{0xF0, 0x4C, 0x0F, 0xB1, 0x17})
			}
		})
	}
}

func TestAtomicI64OpsUseQwordCodegenRegardlessOfPointerWidth(t *testing.T) {
	wantLoad64 := &x64.Emitter{}
	wantLoad64.MovRaxFromRdiDisp(0)

	cases := []struct {
		name      string
		kind      ir.IRInstrKind
		stack     []ir.IRInstr
		wantBytes [][]byte
	}{
		{
			name: "load",
			kind: ir.IRAtomicLoadI64,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{wantLoad64.Buf},
		},
		{
			name: "store",
			kind: ir.IRAtomicStoreI64,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x4C, 0x87, 0x07}},
		},
		{
			name: "exchange",
			kind: ir.IRAtomicExchangeI64,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x4C, 0x87, 0x07}},
		},
		{
			name: "compare_exchange",
			kind: ir.IRAtomicCompareExchangeI64,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRConstI32, Imm: 9},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0xF0, 0x4C, 0x0F, 0xB1, 0x07}},
		},
		{
			name: "fetch_add",
			kind: ir.IRAtomicFetchAddI64,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0xF0, 0x4C, 0x0F, 0xC1, 0x07}},
		},
		{
			name: "fetch_sub",
			kind: ir.IRAtomicFetchSubI64,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x49, 0xF7, 0xD8}, {0xF0, 0x4C, 0x0F, 0xC1, 0x07}},
		},
		{
			name: "fetch_and",
			kind: ir.IRAtomicFetchAndI64,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{wantLoad64.Buf, {0x4D, 0x21, 0xC2}, {0xF0, 0x4C, 0x0F, 0xB1, 0x17}, {0x0F, 0x85}},
		},
		{
			name: "fetch_or",
			kind: ir.IRAtomicFetchOrI64,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{wantLoad64.Buf, {0x4D, 0x09, 0xC2}, {0xF0, 0x4C, 0x0F, 0xB1, 0x17}, {0x0F, 0x85}},
		},
		{
			name: "fetch_xor",
			kind: ir.IRAtomicFetchXorI64,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{wantLoad64.Buf, {0x4D, 0x31, 0xC2}, {0xF0, 0x4C, 0x0F, 0xB1, 0x17}, {0x0F, 0x85}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			instrs := append([]ir.IRInstr{}, tc.stack...)
			instrs = append(instrs, ir.IRInstr{Kind: tc.kind}, ir.IRInstr{Kind: ir.IRReturn})
			fn := ir.IRFunc{
				Name:        "__test_atomic_i64_" + tc.name,
				ParamSlots:  0,
				LocalSlots:  0,
				ReturnSlots: 1,
				Instrs:      instrs,
			}

			for _, pointerWidth := range []int{32, 64} {
				code := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), fn, x64.CodegenOptions{PointerWidthBits: pointerWidth})
				assertContainsBytes(t, "atomic i64 "+tc.name+" guard width", code, addEdxImm32Bytes(8))
				for _, want := range tc.wantBytes {
					assertContainsBytes(t, "atomic i64 "+tc.name, code, want)
				}
				assertNotContainsBytes(t, "atomic i64 "+tc.name+" dword xadd", code, []byte{0xF0, 0x44, 0x0F, 0xC1, 0x07})
				assertNotContainsBytes(t, "atomic i64 "+tc.name+" dword cmpxchg r8", code, []byte{0xF0, 0x44, 0x0F, 0xB1, 0x07})
				assertNotContainsBytes(t, "atomic i64 "+tc.name+" dword cmpxchg r10", code, []byte{0xF0, 0x44, 0x0F, 0xB1, 0x17})
			}
		})
	}
}

func TestAtomicI8OpsUseByteCodegenRegardlessOfPointerWidth(t *testing.T) {
	cases := []struct {
		name      string
		kind      ir.IRInstrKind
		stack     []ir.IRInstr
		wantBytes [][]byte
	}{
		{
			name: "load",
			kind: ir.IRAtomicLoadI8,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x0F, 0xB6, 0x00}},
		},
		{
			name: "store",
			kind: ir.IRAtomicStoreI8,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x45, 0x0F, 0xB6, 0xC0}, {0x4D, 0x89, 0xC1}, {0x44, 0x86, 0x07}, {0x41, 0x51}},
		},
		{
			name: "exchange",
			kind: ir.IRAtomicExchangeI8,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x44, 0x86, 0x07}, {0x45, 0x0F, 0xB6, 0xC0}},
		},
		{
			name: "compare_exchange",
			kind: ir.IRAtomicCompareExchangeI8,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRConstI32, Imm: 9},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0xF0, 0x44, 0x0F, 0xB0, 0x07}, {0x0F, 0xB6, 0xC0}},
		},
		{
			name: "fetch_add",
			kind: ir.IRAtomicFetchAddI8,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0xF0, 0x44, 0x0F, 0xC0, 0x07}, {0x45, 0x0F, 0xB6, 0xC0}},
		},
		{
			name: "fetch_sub",
			kind: ir.IRAtomicFetchSubI8,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x41, 0xF6, 0xD8}, {0xF0, 0x44, 0x0F, 0xC0, 0x07}, {0x45, 0x0F, 0xB6, 0xC0}},
		},
		{
			name: "fetch_and",
			kind: ir.IRAtomicFetchAndI8,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x0F, 0xB6, 0x07}, {0x45, 0x21, 0xC2}, {0xF0, 0x44, 0x0F, 0xB0, 0x17}, {0x0F, 0x85}, {0x0F, 0xB6, 0xC0}},
		},
		{
			name: "fetch_or",
			kind: ir.IRAtomicFetchOrI8,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x0F, 0xB6, 0x07}, {0x45, 0x09, 0xC2}, {0xF0, 0x44, 0x0F, 0xB0, 0x17}, {0x0F, 0x85}, {0x0F, 0xB6, 0xC0}},
		},
		{
			name: "fetch_xor",
			kind: ir.IRAtomicFetchXorI8,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x0F, 0xB6, 0x07}, {0x45, 0x31, 0xC2}, {0xF0, 0x44, 0x0F, 0xB0, 0x17}, {0x0F, 0x85}, {0x0F, 0xB6, 0xC0}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			instrs := append([]ir.IRInstr{}, tc.stack...)
			instrs = append(instrs, ir.IRInstr{Kind: tc.kind}, ir.IRInstr{Kind: ir.IRReturn})
			fn := ir.IRFunc{Name: "__test_atomic_i8_" + tc.name, ReturnSlots: 1, Instrs: instrs}

			for _, pointerWidth := range []int{32, 64} {
				code := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), fn, x64.CodegenOptions{PointerWidthBits: pointerWidth})
				assertContainsBytes(t, "atomic i8 "+tc.name+" guard width", code, addEdxImm32Bytes(1))
				for _, want := range tc.wantBytes {
					assertContainsBytes(t, "atomic i8 "+tc.name, code, want)
				}
				assertNotContainsBytes(t, "atomic i8 "+tc.name+" dword xadd", code, []byte{0xF0, 0x44, 0x0F, 0xC1, 0x07})
				assertNotContainsBytes(t, "atomic i8 "+tc.name+" qword xadd", code, []byte{0xF0, 0x4C, 0x0F, 0xC1, 0x07})
				assertNotContainsBytes(t, "atomic i8 "+tc.name+" dword cmpxchg r8", code, []byte{0xF0, 0x44, 0x0F, 0xB1, 0x07})
			}
		})
	}
}

func TestAtomicI16OpsUseWordCodegenRegardlessOfPointerWidth(t *testing.T) {
	cases := []struct {
		name      string
		kind      ir.IRInstrKind
		stack     []ir.IRInstr
		wantBytes [][]byte
	}{
		{
			name: "load",
			kind: ir.IRAtomicLoadI16,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x0F, 0xB7, 0x00}},
		},
		{
			name: "store",
			kind: ir.IRAtomicStoreI16,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x45, 0x0F, 0xB7, 0xC0}, {0x4D, 0x89, 0xC1}, {0x66, 0x44, 0x87, 0x07}, {0x41, 0x51}},
		},
		{
			name: "exchange",
			kind: ir.IRAtomicExchangeI16,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x66, 0x44, 0x87, 0x07}, {0x45, 0x0F, 0xB7, 0xC0}},
		},
		{
			name: "compare_exchange",
			kind: ir.IRAtomicCompareExchangeI16,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRConstI32, Imm: 9},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0xF0, 0x66, 0x44, 0x0F, 0xB1, 0x07}, {0x0F, 0xB7, 0xC0}},
		},
		{
			name: "fetch_add",
			kind: ir.IRAtomicFetchAddI16,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0xF0, 0x66, 0x44, 0x0F, 0xC1, 0x07}, {0x45, 0x0F, 0xB7, 0xC0}},
		},
		{
			name: "fetch_sub",
			kind: ir.IRAtomicFetchSubI16,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x66, 0x41, 0xF7, 0xD8}, {0xF0, 0x66, 0x44, 0x0F, 0xC1, 0x07}, {0x45, 0x0F, 0xB7, 0xC0}},
		},
		{
			name: "fetch_and",
			kind: ir.IRAtomicFetchAndI16,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x0F, 0xB7, 0x07}, {0x45, 0x21, 0xC2}, {0xF0, 0x66, 0x44, 0x0F, 0xB1, 0x17}, {0x0F, 0x85}, {0x0F, 0xB7, 0xC0}},
		},
		{
			name: "fetch_or",
			kind: ir.IRAtomicFetchOrI16,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x0F, 0xB7, 0x07}, {0x45, 0x09, 0xC2}, {0xF0, 0x66, 0x44, 0x0F, 0xB1, 0x17}, {0x0F, 0x85}, {0x0F, 0xB7, 0xC0}},
		},
		{
			name: "fetch_xor",
			kind: ir.IRAtomicFetchXorI16,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{{0x0F, 0xB7, 0x07}, {0x45, 0x31, 0xC2}, {0xF0, 0x66, 0x44, 0x0F, 0xB1, 0x17}, {0x0F, 0x85}, {0x0F, 0xB7, 0xC0}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			instrs := append([]ir.IRInstr{}, tc.stack...)
			instrs = append(instrs, ir.IRInstr{Kind: tc.kind}, ir.IRInstr{Kind: ir.IRReturn})
			fn := ir.IRFunc{Name: "__test_atomic_i16_" + tc.name, ReturnSlots: 1, Instrs: instrs}

			for _, pointerWidth := range []int{32, 64} {
				code := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), fn, x64.CodegenOptions{PointerWidthBits: pointerWidth})
				assertContainsBytes(t, "atomic i16 "+tc.name+" guard width", code, addEdxImm32Bytes(2))
				for _, want := range tc.wantBytes {
					assertContainsBytes(t, "atomic i16 "+tc.name, code, want)
				}
				assertNotContainsBytes(t, "atomic i16 "+tc.name+" dword xadd", code, []byte{0xF0, 0x44, 0x0F, 0xC1, 0x07})
				assertNotContainsBytes(t, "atomic i16 "+tc.name+" qword xadd", code, []byte{0xF0, 0x4C, 0x0F, 0xC1, 0x07})
				assertNotContainsBytes(t, "atomic i16 "+tc.name+" dword cmpxchg r8", code, []byte{0xF0, 0x44, 0x0F, 0xB1, 0x07})
			}
		})
	}
}
