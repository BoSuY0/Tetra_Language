package x64core

import (
	"bytes"
	"encoding/binary"
	"hash/fnv"
	"strings"
	"testing"
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
)

// ---- emit_atomic_test.go ----

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

	x32 := emitOneFuncWithOptions(
		t,
		x64abi.LinuxSysV(),
		fn,
		x64.CodegenOptions{PointerWidthBits: 32},
	)
	assertContainsBytes(t, "x32 atomic exchange guard width", x32, addEdxImm32Bytes(4))
	assertContainsBytes(t, "x32 atomic exchange 32-bit xchg", x32, []byte{0x44, 0x87, 0x07})
	assertNotContainsBytes(t, "x32 atomic exchange 64-bit xchg", x32, []byte{0x4C, 0x87, 0x07})
	assertContainsBytes(t, "x32 seq_cst fence", x32, []byte{0x0F, 0xAE, 0xF0})

	x64Code := emitOneFuncWithOptions(
		t,
		x64abi.LinuxSysV(),
		fn,
		x64.CodegenOptions{PointerWidthBits: 64},
	)
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
		code := emitOneFuncWithOptions(
			t,
			x64abi.LinuxSysV(),
			fn,
			x64.CodegenOptions{PointerWidthBits: pointerWidth},
		)
		assertNotContainsBytes(
			t,
			"non-seq-cst fence must not emit mfence",
			code,
			[]byte{0x0F, 0xAE, 0xF0},
		)
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
	x32Load := emitOneFuncWithOptions(
		t,
		x64abi.LinuxSysV(),
		loadFn,
		x64.CodegenOptions{PointerWidthBits: 32},
	)
	assertContainsBytes(t, "x32 atomic load guard width", x32Load, addEdxImm32Bytes(4))
	wantLoad32 := &x64.Emitter{}
	wantLoad32.MovEaxFromRaxPtr()
	assertContainsBytes(t, "x32 atomic load 32-bit load", x32Load, wantLoad32.Buf)
	forbidLoad64 := &x64.Emitter{}
	forbidLoad64.MovRaxFromRdiDisp(0)
	assertNotContainsBytes(t, "x32 atomic load 64-bit load", x32Load, forbidLoad64.Buf)

	x64Load := emitOneFuncWithOptions(
		t,
		x64abi.LinuxSysV(),
		loadFn,
		x64.CodegenOptions{PointerWidthBits: 64},
	)
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
	x32Store := emitOneFuncWithOptions(
		t,
		x64abi.LinuxSysV(),
		storeFn,
		x64.CodegenOptions{PointerWidthBits: 32},
	)
	assertContainsBytes(t, "x32 atomic store guard width", x32Store, addEdxImm32Bytes(4))
	assertContainsBytes(
		t,
		"x32 atomic store zero-extends returned pointer",
		x32Store,
		[]byte{0x45, 0x89, 0xC1},
	)
	assertNotContainsBytes(
		t,
		"x32 atomic store 64-bit return copy",
		x32Store,
		[]byte{0x4D, 0x89, 0xC1},
	)
	assertContainsBytes(t, "x32 atomic store 32-bit xchg", x32Store, []byte{0x44, 0x87, 0x07})
	assertNotContainsBytes(t, "x32 atomic store 64-bit xchg", x32Store, []byte{0x4C, 0x87, 0x07})

	x64Store := emitOneFuncWithOptions(
		t,
		x64abi.LinuxSysV(),
		storeFn,
		x64.CodegenOptions{PointerWidthBits: 64},
	)
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

	x32 := emitOneFuncWithOptions(
		t,
		x64abi.LinuxSysV(),
		fn,
		x64.CodegenOptions{PointerWidthBits: 32},
	)
	assertContainsBytes(t, "x32 atomic cas guard width", x32, addEdxImm32Bytes(4))
	assertContainsBytes(
		t,
		"x32 atomic cas zero-extends expected pointer into accumulator",
		x32,
		[]byte{0x44, 0x89, 0xC8},
	)
	assertNotContainsBytes(
		t,
		"x32 atomic cas 64-bit expected pointer copy",
		x32,
		[]byte{0x4C, 0x89, 0xC8},
	)
	assertContainsBytes(
		t,
		"x32 atomic cas 32-bit lock cmpxchg",
		x32,
		[]byte{0xF0, 0x44, 0x0F, 0xB1, 0x07},
	)
	assertNotContainsBytes(
		t,
		"x32 atomic cas 64-bit lock cmpxchg",
		x32,
		[]byte{0xF0, 0x4C, 0x0F, 0xB1, 0x07},
	)

	x64Code := emitOneFuncWithOptions(
		t,
		x64abi.LinuxSysV(),
		fn,
		x64.CodegenOptions{PointerWidthBits: 64},
	)
	assertContainsBytes(t, "x64 atomic cas guard width", x64Code, addEdxImm32Bytes(8))
	assertContainsBytes(
		t,
		"x64 atomic cas 64-bit lock cmpxchg",
		x64Code,
		[]byte{0xF0, 0x4C, 0x0F, 0xB1, 0x07},
	)
	assertNotContainsBytes(
		t,
		"x64 atomic cas 32-bit lock cmpxchg",
		x64Code,
		[]byte{0xF0, 0x44, 0x0F, 0xB1, 0x07},
	)
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

	x32 := emitOneFuncWithOptions(
		t,
		x64abi.LinuxSysV(),
		fn,
		x64.CodegenOptions{PointerWidthBits: 32},
	)
	assertContainsBytes(t, "x32 atomic fetch_add guard width", x32, addEdxImm32Bytes(4))
	assertContainsBytes(
		t,
		"x32 atomic fetch_add 32-bit lock xadd",
		x32,
		[]byte{0xF0, 0x44, 0x0F, 0xC1, 0x07},
	)
	assertNotContainsBytes(
		t,
		"x32 atomic fetch_add 64-bit lock xadd",
		x32,
		[]byte{0xF0, 0x4C, 0x0F, 0xC1, 0x07},
	)

	x64Code := emitOneFuncWithOptions(
		t,
		x64abi.LinuxSysV(),
		fn,
		x64.CodegenOptions{PointerWidthBits: 64},
	)
	assertContainsBytes(t, "x64 atomic fetch_add guard width", x64Code, addEdxImm32Bytes(8))
	assertContainsBytes(
		t,
		"x64 atomic fetch_add 64-bit lock xadd",
		x64Code,
		[]byte{0xF0, 0x4C, 0x0F, 0xC1, 0x07},
	)
	assertNotContainsBytes(
		t,
		"x64 atomic fetch_add 32-bit lock xadd",
		x64Code,
		[]byte{0xF0, 0x44, 0x0F, 0xC1, 0x07},
	)
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

	x32 := emitOneFuncWithOptions(
		t,
		x64abi.LinuxSysV(),
		fn,
		x64.CodegenOptions{PointerWidthBits: 32},
	)
	assertContainsBytes(t, "x32 atomic fetch_sub guard width", x32, addEdxImm32Bytes(4))
	assertContainsBytes(t, "x32 atomic fetch_sub neg r8d", x32, []byte{0x41, 0xF7, 0xD8})
	assertContainsBytes(
		t,
		"x32 atomic fetch_sub 32-bit lock xadd",
		x32,
		[]byte{0xF0, 0x44, 0x0F, 0xC1, 0x07},
	)
	assertNotContainsBytes(t, "x32 atomic fetch_sub 64-bit neg r8", x32, []byte{0x49, 0xF7, 0xD8})
	assertNotContainsBytes(
		t,
		"x32 atomic fetch_sub 64-bit lock xadd",
		x32,
		[]byte{0xF0, 0x4C, 0x0F, 0xC1, 0x07},
	)

	x64Code := emitOneFuncWithOptions(
		t,
		x64abi.LinuxSysV(),
		fn,
		x64.CodegenOptions{PointerWidthBits: 64},
	)
	assertContainsBytes(t, "x64 atomic fetch_sub guard width", x64Code, addEdxImm32Bytes(8))
	assertContainsBytes(t, "x64 atomic fetch_sub neg r8", x64Code, []byte{0x49, 0xF7, 0xD8})
	assertContainsBytes(
		t,
		"x64 atomic fetch_sub 64-bit lock xadd",
		x64Code,
		[]byte{0xF0, 0x4C, 0x0F, 0xC1, 0x07},
	)
	assertNotContainsBytes(
		t,
		"x64 atomic fetch_sub 32-bit neg r8d",
		x64Code,
		[]byte{0x41, 0xF7, 0xD8},
	)
	assertNotContainsBytes(
		t,
		"x64 atomic fetch_sub 32-bit lock xadd",
		x64Code,
		[]byte{0xF0, 0x44, 0x0F, 0xC1, 0x07},
	)
}

func TestAtomicPointerLogicalFetchOpsUseCASLoopWithConfiguredPointerWidth(t *testing.T) {
	cases := []struct {
		name  string
		kind  ir.IRInstrKind
		op32  []byte
		op64  []byte
		label string
	}{
		{
			name:  "and",
			kind:  ir.IRAtomicFetchAndPtr,
			op32:  []byte{0x45, 0x21, 0xC2},
			op64:  []byte{0x4D, 0x21, 0xC2},
			label: "and r10,r8",
		},
		{
			name:  "or",
			kind:  ir.IRAtomicFetchOrPtr,
			op32:  []byte{0x45, 0x09, 0xC2},
			op64:  []byte{0x4D, 0x09, 0xC2},
			label: "or r10,r8",
		},
		{
			name:  "xor",
			kind:  ir.IRAtomicFetchXorPtr,
			op32:  []byte{0x45, 0x31, 0xC2},
			op64:  []byte{0x4D, 0x31, 0xC2},
			label: "xor r10,r8",
		},
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

			x32 := emitOneFuncWithOptions(
				t,
				x64abi.LinuxSysV(),
				fn,
				x64.CodegenOptions{PointerWidthBits: 32},
			)
			assertContainsBytes(
				t,
				"x32 atomic fetch_"+tc.name+" guard width",
				x32,
				addEdxImm32Bytes(4),
			)
			assertContainsBytes(t, "x32 atomic fetch_"+tc.name+" load", x32, []byte{0x8B, 0x07})
			assertContainsBytes(
				t,
				"x32 atomic fetch_"+tc.name+" mov r10d,eax",
				x32,
				[]byte{0x41, 0x89, 0xC2},
			)
			assertContainsBytes(t, "x32 atomic fetch_"+tc.name+" "+tc.label, x32, tc.op32)
			assertContainsBytes(
				t,
				"x32 atomic fetch_"+tc.name+" lock cmpxchg r10d",
				x32,
				[]byte{0xF0, 0x44, 0x0F, 0xB1, 0x17},
			)
			assertContainsBytes(
				t,
				"x32 atomic fetch_"+tc.name+" retry branch",
				x32,
				[]byte{0x0F, 0x85},
			)
			assertNotContainsBytes(
				t,
				"x32 atomic fetch_"+tc.name+" qword cmpxchg",
				x32,
				[]byte{0xF0, 0x4C, 0x0F, 0xB1, 0x17},
			)

			x64Code := emitOneFuncWithOptions(
				t,
				x64abi.LinuxSysV(),
				fn,
				x64.CodegenOptions{PointerWidthBits: 64},
			)
			assertContainsBytes(
				t,
				"x64 atomic fetch_"+tc.name+" guard width",
				x64Code,
				addEdxImm32Bytes(8),
			)
			wantLoad64 := &x64.Emitter{}
			wantLoad64.MovRaxFromRdiDisp(0)
			assertContainsBytes(t, "x64 atomic fetch_"+tc.name+" load", x64Code, wantLoad64.Buf)
			assertContainsBytes(
				t,
				"x64 atomic fetch_"+tc.name+" mov r10,rax",
				x64Code,
				[]byte{0x49, 0x89, 0xC2},
			)
			assertContainsBytes(t, "x64 atomic fetch_"+tc.name+" "+tc.label, x64Code, tc.op64)
			assertContainsBytes(
				t,
				"x64 atomic fetch_"+tc.name+" lock cmpxchg r10",
				x64Code,
				[]byte{0xF0, 0x4C, 0x0F, 0xB1, 0x17},
			)
			assertContainsBytes(
				t,
				"x64 atomic fetch_"+tc.name+" retry branch",
				x64Code,
				[]byte{0x0F, 0x85},
			)
			assertNotContainsBytes(
				t,
				"x64 atomic fetch_"+tc.name+" dword cmpxchg",
				x64Code,
				[]byte{0xF0, 0x44, 0x0F, 0xB1, 0x17},
			)
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
			wantBytes: [][]byte{
				{0x8B, 0x07},
				{0x45, 0x21, 0xC2},
				{0xF0, 0x44, 0x0F, 0xB1, 0x17},
				{0x0F, 0x85},
			},
		},
		{
			name: "fetch_or",
			kind: ir.IRAtomicFetchOrI32,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{
				{0x8B, 0x07},
				{0x45, 0x09, 0xC2},
				{0xF0, 0x44, 0x0F, 0xB1, 0x17},
				{0x0F, 0x85},
			},
		},
		{
			name: "fetch_xor",
			kind: ir.IRAtomicFetchXorI32,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{
				{0x8B, 0x07},
				{0x45, 0x31, 0xC2},
				{0xF0, 0x44, 0x0F, 0xB1, 0x17},
				{0x0F, 0x85},
			},
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
				code := emitOneFuncWithOptions(
					t,
					x64abi.LinuxSysV(),
					fn,
					x64.CodegenOptions{PointerWidthBits: pointerWidth},
				)
				assertContainsBytes(
					t,
					"atomic i32 "+tc.name+" guard width",
					code,
					addEdxImm32Bytes(4),
				)
				for _, want := range tc.wantBytes {
					assertContainsBytes(t, "atomic i32 "+tc.name, code, want)
				}
				assertNotContainsBytes(
					t,
					"atomic i32 "+tc.name+" qword xadd",
					code,
					[]byte{0xF0, 0x4C, 0x0F, 0xC1, 0x07},
				)
				assertNotContainsBytes(
					t,
					"atomic i32 "+tc.name+" qword cmpxchg r8",
					code,
					[]byte{0xF0, 0x4C, 0x0F, 0xB1, 0x07},
				)
				assertNotContainsBytes(
					t,
					"atomic i32 "+tc.name+" qword cmpxchg r10",
					code,
					[]byte{0xF0, 0x4C, 0x0F, 0xB1, 0x17},
				)
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
			wantBytes: [][]byte{
				wantLoad64.Buf,
				{0x4D, 0x21, 0xC2},
				{0xF0, 0x4C, 0x0F, 0xB1, 0x17},
				{0x0F, 0x85},
			},
		},
		{
			name: "fetch_or",
			kind: ir.IRAtomicFetchOrI64,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{
				wantLoad64.Buf,
				{0x4D, 0x09, 0xC2},
				{0xF0, 0x4C, 0x0F, 0xB1, 0x17},
				{0x0F, 0x85},
			},
		},
		{
			name: "fetch_xor",
			kind: ir.IRAtomicFetchXorI64,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{
				wantLoad64.Buf,
				{0x4D, 0x31, 0xC2},
				{0xF0, 0x4C, 0x0F, 0xB1, 0x17},
				{0x0F, 0x85},
			},
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
				code := emitOneFuncWithOptions(
					t,
					x64abi.LinuxSysV(),
					fn,
					x64.CodegenOptions{PointerWidthBits: pointerWidth},
				)
				assertContainsBytes(
					t,
					"atomic i64 "+tc.name+" guard width",
					code,
					addEdxImm32Bytes(8),
				)
				for _, want := range tc.wantBytes {
					assertContainsBytes(t, "atomic i64 "+tc.name, code, want)
				}
				assertNotContainsBytes(
					t,
					"atomic i64 "+tc.name+" dword xadd",
					code,
					[]byte{0xF0, 0x44, 0x0F, 0xC1, 0x07},
				)
				assertNotContainsBytes(
					t,
					"atomic i64 "+tc.name+" dword cmpxchg r8",
					code,
					[]byte{0xF0, 0x44, 0x0F, 0xB1, 0x07},
				)
				assertNotContainsBytes(
					t,
					"atomic i64 "+tc.name+" dword cmpxchg r10",
					code,
					[]byte{0xF0, 0x44, 0x0F, 0xB1, 0x17},
				)
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
			wantBytes: [][]byte{
				{0x45, 0x0F, 0xB6, 0xC0},
				{0x4D, 0x89, 0xC1},
				{0x44, 0x86, 0x07},
				{0x41, 0x51},
			},
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
			wantBytes: [][]byte{
				{0x41, 0xF6, 0xD8},
				{0xF0, 0x44, 0x0F, 0xC0, 0x07},
				{0x45, 0x0F, 0xB6, 0xC0},
			},
		},
		{
			name: "fetch_and",
			kind: ir.IRAtomicFetchAndI8,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{
				{0x0F, 0xB6, 0x07},
				{0x45, 0x21, 0xC2},
				{0xF0, 0x44, 0x0F, 0xB0, 0x17},
				{0x0F, 0x85},
				{0x0F, 0xB6, 0xC0},
			},
		},
		{
			name: "fetch_or",
			kind: ir.IRAtomicFetchOrI8,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{
				{0x0F, 0xB6, 0x07},
				{0x45, 0x09, 0xC2},
				{0xF0, 0x44, 0x0F, 0xB0, 0x17},
				{0x0F, 0x85},
				{0x0F, 0xB6, 0xC0},
			},
		},
		{
			name: "fetch_xor",
			kind: ir.IRAtomicFetchXorI8,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{
				{0x0F, 0xB6, 0x07},
				{0x45, 0x31, 0xC2},
				{0xF0, 0x44, 0x0F, 0xB0, 0x17},
				{0x0F, 0x85},
				{0x0F, 0xB6, 0xC0},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			instrs := append([]ir.IRInstr{}, tc.stack...)
			instrs = append(instrs, ir.IRInstr{Kind: tc.kind}, ir.IRInstr{Kind: ir.IRReturn})
			fn := ir.IRFunc{Name: "__test_atomic_i8_" + tc.name, ReturnSlots: 1, Instrs: instrs}

			for _, pointerWidth := range []int{32, 64} {
				code := emitOneFuncWithOptions(
					t,
					x64abi.LinuxSysV(),
					fn,
					x64.CodegenOptions{PointerWidthBits: pointerWidth},
				)
				assertContainsBytes(
					t,
					"atomic i8 "+tc.name+" guard width",
					code,
					addEdxImm32Bytes(1),
				)
				for _, want := range tc.wantBytes {
					assertContainsBytes(t, "atomic i8 "+tc.name, code, want)
				}
				assertNotContainsBytes(
					t,
					"atomic i8 "+tc.name+" dword xadd",
					code,
					[]byte{0xF0, 0x44, 0x0F, 0xC1, 0x07},
				)
				assertNotContainsBytes(
					t,
					"atomic i8 "+tc.name+" qword xadd",
					code,
					[]byte{0xF0, 0x4C, 0x0F, 0xC1, 0x07},
				)
				assertNotContainsBytes(
					t,
					"atomic i8 "+tc.name+" dword cmpxchg r8",
					code,
					[]byte{0xF0, 0x44, 0x0F, 0xB1, 0x07},
				)
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
			wantBytes: [][]byte{
				{0x45, 0x0F, 0xB7, 0xC0},
				{0x4D, 0x89, 0xC1},
				{0x66, 0x44, 0x87, 0x07},
				{0x41, 0x51},
			},
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
			wantBytes: [][]byte{
				{0x66, 0x41, 0xF7, 0xD8},
				{0xF0, 0x66, 0x44, 0x0F, 0xC1, 0x07},
				{0x45, 0x0F, 0xB7, 0xC0},
			},
		},
		{
			name: "fetch_and",
			kind: ir.IRAtomicFetchAndI16,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{
				{0x0F, 0xB7, 0x07},
				{0x45, 0x21, 0xC2},
				{0xF0, 0x66, 0x44, 0x0F, 0xB1, 0x17},
				{0x0F, 0x85},
				{0x0F, 0xB7, 0xC0},
			},
		},
		{
			name: "fetch_or",
			kind: ir.IRAtomicFetchOrI16,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{
				{0x0F, 0xB7, 0x07},
				{0x45, 0x09, 0xC2},
				{0xF0, 0x66, 0x44, 0x0F, 0xB1, 0x17},
				{0x0F, 0x85},
				{0x0F, 0xB7, 0xC0},
			},
		},
		{
			name: "fetch_xor",
			kind: ir.IRAtomicFetchXorI16,
			stack: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 1},
			},
			wantBytes: [][]byte{
				{0x0F, 0xB7, 0x07},
				{0x45, 0x31, 0xC2},
				{0xF0, 0x66, 0x44, 0x0F, 0xB1, 0x17},
				{0x0F, 0x85},
				{0x0F, 0xB7, 0xC0},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			instrs := append([]ir.IRInstr{}, tc.stack...)
			instrs = append(instrs, ir.IRInstr{Kind: tc.kind}, ir.IRInstr{Kind: ir.IRReturn})
			fn := ir.IRFunc{Name: "__test_atomic_i16_" + tc.name, ReturnSlots: 1, Instrs: instrs}

			for _, pointerWidth := range []int{32, 64} {
				code := emitOneFuncWithOptions(
					t,
					x64abi.LinuxSysV(),
					fn,
					x64.CodegenOptions{PointerWidthBits: pointerWidth},
				)
				assertContainsBytes(
					t,
					"atomic i16 "+tc.name+" guard width",
					code,
					addEdxImm32Bytes(2),
				)
				for _, want := range tc.wantBytes {
					assertContainsBytes(t, "atomic i16 "+tc.name, code, want)
				}
				assertNotContainsBytes(
					t,
					"atomic i16 "+tc.name+" dword xadd",
					code,
					[]byte{0xF0, 0x44, 0x0F, 0xC1, 0x07},
				)
				assertNotContainsBytes(
					t,
					"atomic i16 "+tc.name+" qword xadd",
					code,
					[]byte{0xF0, 0x4C, 0x0F, 0xC1, 0x07},
				)
				assertNotContainsBytes(
					t,
					"atomic i16 "+tc.name+" dword cmpxchg r8",
					code,
					[]byte{0xF0, 0x44, 0x0F, 0xB1, 0x07},
				)
			}
		})
	}
}

// ---- emit_test.go ----

type unsupportedCtxSwitchABI struct {
	*x64abi.SysVUnix
}

type emitArtifacts struct {
	code         []byte
	dataBlobs    [][]byte
	leaPatches   []x64obj.LeaPatch
	callPatches  []x64obj.CallPatch
	importPaches []x64obj.ImportPatch
}

func emitOneFunc(t *testing.T, abi x64abi.ABI, fn ir.IRFunc) []byte {
	t.Helper()
	return emitOneFuncWithOptions(t, abi, fn, x64.CodegenOptions{})
}

func emitOneFuncWithOptions(
	t *testing.T,
	abi x64abi.ABI,
	fn ir.IRFunc,
	opt x64.CodegenOptions,
) []byte {
	t.Helper()

	emitFn := NewEmitFunc(abi)
	e := &x64.Emitter{}
	var dataBlobs [][]byte
	var leaPatches []x64obj.LeaPatch
	var callPatches []x64obj.CallPatch
	var importPatches []x64obj.ImportPatch
	if err := emitFn(e, fn, &dataBlobs, &leaPatches, &callPatches, &importPatches, opt); err != nil {
		t.Fatalf("emit: %v", err)
	}
	return e.Buf
}

func emitWithArtifacts(t *testing.T, abi x64abi.ABI, fn ir.IRFunc) emitArtifacts {
	t.Helper()
	return emitWithArtifactsWithOptions(t, abi, fn, x64.CodegenOptions{})
}

func emitWithArtifactsWithOptions(
	t *testing.T,
	abi x64abi.ABI,
	fn ir.IRFunc,
	opt x64.CodegenOptions,
) emitArtifacts {
	t.Helper()

	emitFn := NewEmitFunc(abi)
	e := &x64.Emitter{}
	var dataBlobs [][]byte
	var leaPatches []x64obj.LeaPatch
	var callPatches []x64obj.CallPatch
	var importPatches []x64obj.ImportPatch
	if err := emitFn(e, fn, &dataBlobs, &leaPatches, &callPatches, &importPatches, opt); err != nil {
		t.Fatalf("emit: %v", err)
	}
	return emitArtifacts{
		code:         e.Buf,
		dataBlobs:    dataBlobs,
		leaPatches:   leaPatches,
		callPatches:  callPatches,
		importPaches: importPatches,
	}
}

func TestRuntimeHeapTelemetryDoesNotReferenceActorSnapshotWithoutActorDomainOption(t *testing.T) {
	artifacts := emitRuntimeHeapTelemetryMain(t, false)
	for _, patch := range artifacts.callPatches {
		if patch.Name == runtimeHeapTelemetryActorSnapshotSymbol {
			t.Fatalf(
				"non-actor heap telemetry emitted actor snapshot call patch: %#v",
				artifacts.callPatches,
			)
		}
	}
}

func TestRuntimeHeapTelemetryReferencesActorSnapshotOnlyWithActorDomainOption(t *testing.T) {
	artifacts := emitRuntimeHeapTelemetryMain(t, true)
	found := false
	for _, patch := range artifacts.callPatches {
		if patch.Name == runtimeHeapTelemetryActorSnapshotSymbol {
			found = true
		}
	}
	if !found {
		t.Fatalf(
			"actor heap telemetry did not emit %s call patch: %#v",
			runtimeHeapTelemetryActorSnapshotSymbol,
			artifacts.callPatches,
		)
	}
}

func TestRuntimeHeapTelemetryActorJSONIncludesMailboxBudgetBackpressureFields(t *testing.T) {
	raw, _, template := runtimeHeapTelemetryActorJSON("app", true)
	text := string(raw)
	for _, field := range []string{
		`"mailbox_current_bytes"`,
		`"mailbox_peak_bytes"`,
		`"byte_budget"`,
		`"over_budget_count"`,
		`"backpressure_events"`,
	} {
		if !strings.Contains(text, field) {
			t.Fatalf("actor heap telemetry template missing %s in:\n%s", field, text)
		}
	}
	numbers := template.numbers[0]
	for label, off := range map[string]int32{
		"mailbox_current_bytes": numbers.mailboxCurrent,
		"mailbox_peak_bytes":    numbers.mailboxPeak,
		"byte_budget":           numbers.byteBudget,
		"over_budget_count":     numbers.overBudgetCount,
		"backpressure_events":   numbers.backpressureEvents,
	} {
		if off <= 0 {
			t.Fatalf("actor heap telemetry field %s offset = %d, want populated offset", label, off)
		}
	}
}

func emitRuntimeHeapTelemetryMain(t *testing.T, actorDomains bool) emitArtifacts {
	t.Helper()
	return emitWithArtifactsWithOptions(t, x64abi.LinuxSysV(), ir.IRFunc{
		Name:        "main",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}, x64.CodegenOptions{
		EmitRuntimeHeapTelemetry:         true,
		RuntimeHeapTelemetryActorDomains: actorDomains,
		RuntimeHeapTelemetryDir:          "telemetry",
		RuntimeHeapTelemetryProgram:      "app",
		RuntimeHeapTelemetryMain:         "main",
	})
}

func TestIRSymAddrUsesFunctionAddressPatchKind(t *testing.T) {
	artifacts := emitWithArtifacts(t, x64abi.LinuxSysV(), ir.IRFunc{
		Name:        "main",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRSymAddr, Name: "callback_target"},
			{Kind: ir.IRReturn},
		},
	})
	if len(artifacts.callPatches) != 1 {
		t.Fatalf(
			"callPatches len = %d, want 1: %#v",
			len(artifacts.callPatches),
			artifacts.callPatches,
		)
	}
	patch := artifacts.callPatches[0]
	if patch.Name != "callback_target" {
		t.Fatalf("patch name = %q, want callback_target", patch.Name)
	}
	if patch.Kind != x64obj.PatchFuncAddrRel32 {
		t.Fatalf("patch kind = %v, want PatchFuncAddrRel32", patch.Kind)
	}
	if !bytes.Contains(artifacts.code, []byte{0x48, 0x8D, 0x05}) {
		t.Fatalf("IRSymAddr did not emit lea rax, [rip+disp32]: % x", artifacts.code)
	}
}

func TestCtxSwitchUnsupportedABIDiagnostic(t *testing.T) {
	emitFn := NewEmitFunc(&unsupportedCtxSwitchABI{SysVUnix: x64abi.LinuxSysV()})
	e := &x64.Emitter{}
	var dataBlobs [][]byte
	var leaPatches []x64obj.LeaPatch
	var callPatches []x64obj.CallPatch
	var importPatches []x64obj.ImportPatch
	err := emitFn(e, ir.IRFunc{
		Name:        "__test_ctx_switch_unknown",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRCtxSwitch},
			{Kind: ir.IRReturn},
		},
	}, &dataBlobs, &leaPatches, &callPatches, &importPatches, x64.CodegenOptions{})
	if err == nil {
		t.Fatalf("expected unsupported ABI error")
	}
	if !strings.Contains(err.Error(), "ctx_switch: unsupported ABI") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIRRawSliceFromPartsEmitsLengthContractGuards(t *testing.T) {
	code := emitOneFunc(t, x64abi.LinuxSysV(), ir.IRFunc{
		Name:        "main",
		ReturnSlots: 2,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRCapMem},
			{Kind: ir.IRRawSliceFromParts, Imm: 2},
			{Kind: ir.IRReturn},
		},
	})
	if !bytes.Contains(code, []byte{0x85, 0xC9, 0x0F, 0x8C}) {
		t.Fatalf("raw_slice_from_parts did not emit signed i32 negative-length guard: % x", code)
	}
	wantCmp := []byte{0x48, 0x81, 0xF9, 0xFF, 0xFF, 0xFF, 0x1F}
	if !bytes.Contains(code, wantCmp) {
		t.Fatalf(
			"raw_slice_i32_from_parts did not emit i32 byte-overflow guard % x in: % x",
			wantCmp,
			code,
		)
	}
	wantExit2 := []byte{0xBF, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x3C, 0x00, 0x00, 0x00, 0x0F, 0x05}
	if !bytes.Contains(code, wantExit2) {
		t.Fatalf("raw_slice_from_parts did not emit deterministic trap exit 2: % x", code)
	}
}

func findCtxSwitchInternalTarget(t *testing.T, buf []byte) (callOp int, target int) {
	t.Helper()

	for i := 0; i+5 <= len(buf); i++ {
		if buf[i] != 0xE8 {
			continue
		}
		disp := int32(binary.LittleEndian.Uint32(buf[i+1 : i+5]))
		target := i + 5 + int(disp)
		if target < 0 || target >= len(buf) {
			continue
		}
		// Both SysV and Win64 save RBX first.
		if buf[target] == 0x53 {
			return i, target
		}
	}
	t.Fatalf("ctx_switch internal call target not found")
	return 0, 0
}

func expectedCtxSwitchSysV() []byte {
	e := &x64.Emitter{}
	e.PushRbx()
	e.PushRbp()
	e.PushR12()
	e.PushR13()
	e.PushR14()
	e.PushR15()
	e.MovMem64RdiDispRsp(0)
	e.MovRdiRsi()
	e.MovRspFromRdiDisp(0)
	e.PopR15()
	e.PopR14()
	e.PopR13()
	e.PopR12()
	e.PopRbp()
	e.PopRbx()
	e.Ret()
	return e.Buf
}

func expectedCtxSwitchWin64() []byte {
	e := &x64.Emitter{}
	e.PushRbx()
	e.PushRbp()
	e.PushRdi()
	e.PushRsi()
	e.PushR12()
	e.PushR13()
	e.PushR14()
	e.PushR15()
	e.MovRdiRcx()
	e.MovMem64RdiDispRsp(0)
	e.MovRdiRdx()
	e.MovRspFromRdiDisp(0)
	e.PopR15()
	e.PopR14()
	e.PopR13()
	e.PopR12()
	e.PopRsi()
	e.PopRdi()
	e.PopRbp()
	e.PopRbx()
	e.Ret()
	return e.Buf
}

func TestCtxSwitchEmissionSysV(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "__test_ctx_switch_sysv",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRCtxSwitch},
			{Kind: ir.IRReturn},
		},
	}

	buf := emitOneFunc(t, x64abi.LinuxSysV(), fn)
	_, target := findCtxSwitchInternalTarget(t, buf)

	want := expectedCtxSwitchSysV()
	if target+len(want) > len(buf) {
		t.Fatalf(
			"ctx_switch target slice out of bounds: target=%d want=%d len=%d",
			target,
			len(want),
			len(buf),
		)
	}
	got := buf[target : target+len(want)]
	if !bytes.Equal(got, want) {
		t.Fatalf("ctx_switch SysV internal stub mismatch\n got=% x\nwant=% x", got, want)
	}

	shadow := &x64.Emitter{}
	shadow.SubRspImm32(32)
	if bytes.Contains(buf, shadow.Buf) {
		t.Fatalf("unexpected Win64 shadow-space adjustment in SysV ctx_switch")
	}
}

func TestCtxSwitchEmissionWin64(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "__test_ctx_switch_win64",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRCtxSwitch},
			{Kind: ir.IRReturn},
		},
	}

	buf := emitOneFunc(t, x64abi.NewWin64(), fn)
	callOp, target := findCtxSwitchInternalTarget(t, buf)

	want := expectedCtxSwitchWin64()
	if target+len(want) > len(buf) {
		t.Fatalf(
			"ctx_switch target slice out of bounds: target=%d want=%d len=%d",
			target,
			len(want),
			len(buf),
		)
	}
	got := buf[target : target+len(want)]
	if !bytes.Equal(got, want) {
		t.Fatalf("ctx_switch Win64 internal stub mismatch\n got=% x\nwant=% x", got, want)
	}

	sub := &x64.Emitter{}
	sub.SubRspImm32(32)
	add := &x64.Emitter{}
	add.AddRspImm32(32)

	if callOp < len(sub.Buf) {
		t.Fatalf(
			"call opcode too early to contain prologue shadow-space adjustment: callOp=%d",
			callOp,
		)
	}
	if !bytes.Equal(buf[callOp-len(sub.Buf):callOp], sub.Buf) {
		t.Fatalf("missing Win64 shadow-space prologue before ctx_switch call")
	}
	callEnd := callOp + 5
	if callEnd+len(add.Buf) > len(buf) {
		t.Fatalf("call end slice out of bounds")
	}
	if !bytes.Equal(buf[callEnd:callEnd+len(add.Buf)], add.Buf) {
		t.Fatalf("missing Win64 shadow-space epilogue after ctx_switch call")
	}
}

func TestObjectEmitSharedLiteralAddsDataRelocArtifacts(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "__test_strlit",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRStrLit, Str: []byte("shared-data")},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
	art := emitWithArtifacts(t, x64abi.LinuxSysV(), fn)
	if len(art.dataBlobs) != 1 {
		t.Fatalf("data blob count = %d, want 1", len(art.dataBlobs))
	}
	if string(art.dataBlobs[0]) != "shared-data" {
		t.Fatalf("unexpected data blob: %q", string(art.dataBlobs[0]))
	}
	if len(art.leaPatches) != 1 {
		t.Fatalf("lea patch count = %d, want 1", len(art.leaPatches))
	}
	if art.leaPatches[0].DataIndex != 0 {
		t.Fatalf("lea patch data index = %d, want 0", art.leaPatches[0].DataIndex)
	}
	if art.leaPatches[0].At < 0 || art.leaPatches[0].At+4 > len(art.code) {
		t.Fatalf("lea patch offset out of range: at=%d len=%d", art.leaPatches[0].At, len(art.code))
	}
}

func TestScalarRegisterCallEmissionUsesTargetABIFrames(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 41},
			{Kind: ir.IRCall, Name: "inc", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRReturn},
		},
	}

	sysv := emitWithArtifacts(t, x64abi.LinuxSysV(), fn)
	if len(sysv.callPatches) != 1 || sysv.callPatches[0].Name != "inc" {
		t.Fatalf("SysV call patches = %#v, want inc", sysv.callPatches)
	}
	for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
		if bytes.Contains(sysv.code, forbidden) {
			t.Fatalf(
				"SysV register call emitted stack-machine push/pop byte % x: % x",
				forbidden,
				sysv.code,
			)
		}
	}
	shadow := &x64.Emitter{}
	shadow.SubRspImm32(32)
	if bytes.Contains(sysv.code, shadow.Buf) {
		t.Fatalf("SysV register call emitted Win64 shadow space: % x", sysv.code)
	}

	win := emitWithArtifacts(t, x64abi.NewWin64(), fn)
	if len(win.callPatches) != 1 || win.callPatches[0].Name != "inc" {
		t.Fatalf("Win64 call patches = %#v, want inc", win.callPatches)
	}
	for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
		if bytes.Contains(win.code, forbidden) {
			t.Fatalf(
				"Win64 register call emitted stack-machine push/pop byte % x: % x",
				forbidden,
				win.code,
			)
		}
	}
	addShadow := &x64.Emitter{}
	addShadow.AddRspImm32(32)
	callAt := bytes.IndexByte(win.code, 0xE8)
	if callAt < len(shadow.Buf) {
		t.Fatalf("Win64 call opcode too early for shadow-space prologue: % x", win.code)
	}
	if !bytes.Equal(win.code[callAt-len(shadow.Buf):callAt], shadow.Buf) {
		t.Fatalf("Win64 register call missing shadow-space prologue: % x", win.code)
	}
	callEnd := callAt + 5
	if callEnd+len(addShadow.Buf) > len(win.code) ||
		!bytes.Equal(win.code[callEnd:callEnd+len(addShadow.Buf)], addShadow.Buf) {
		t.Fatalf("Win64 register call missing shadow-space epilogue: % x", win.code)
	}
}

func TestABIDiagnosticEmitSharedRejectsMissingInputs(t *testing.T) {
	emitFn := NewEmitFunc(nil)
	err := emitFn(nil, ir.IRFunc{Name: "__test"}, nil, nil, nil, nil, x64.CodegenOptions{})
	if err == nil || !strings.Contains(err.Error(), "missing ABI") {
		t.Fatalf("unexpected missing ABI error: %v", err)
	}

	emitFn = NewEmitFunc(x64abi.LinuxSysV())
	var dataBlobs [][]byte
	var leaPatches []x64obj.LeaPatch
	var callPatches []x64obj.CallPatch
	err = emitFn(
		nil,
		ir.IRFunc{Name: "__test"},
		&dataBlobs,
		&leaPatches,
		&callPatches,
		nil,
		x64.CodegenOptions{},
	)
	if err == nil || !strings.Contains(err.Error(), "missing emitter") {
		t.Fatalf("unexpected missing emitter error: %v", err)
	}
}

func TestABIDiagnosticEmitSharedRejectsInvalidFrameSlots(t *testing.T) {
	cases := []struct {
		name string
		abi  x64abi.ABI
	}{
		{name: "sysv", abi: x64abi.LinuxSysV()},
		{name: "win64", abi: x64abi.NewWin64()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			emitFn := NewEmitFunc(tc.abi)
			e := &x64.Emitter{}
			var dataBlobs [][]byte
			var leaPatches []x64obj.LeaPatch
			var callPatches []x64obj.CallPatch
			var importPatches []x64obj.ImportPatch
			err := emitFn(e, ir.IRFunc{
				Name:        "__test_invalid_frame_slots",
				ParamSlots:  2,
				LocalSlots:  1,
				ReturnSlots: 0,
			}, &dataBlobs, &leaPatches, &callPatches, &importPatches, x64.CodegenOptions{})
			if err == nil {
				t.Fatalf("expected invalid frame slot diagnostic")
			}
			if !strings.Contains(
				err.Error(),
				"function '__test_invalid_frame_slots' has invalid slots",
			) {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(e.Buf) != 0 {
				t.Fatalf("emitted %d bytes before rejecting invalid slots", len(e.Buf))
			}
		})
	}
}

func TestABIDiagnosticEmitSharedRejectsLocalSlotOutOfBounds(t *testing.T) {
	cases := []struct {
		name  string
		instr []ir.IRInstr
		want  string
	}{
		{
			name: "load_negative",
			instr: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: -1},
			},
			want: "local slot -1 out of bounds",
		},
		{
			name: "load_past_end",
			instr: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 1},
			},
			want: "local slot 1 out of bounds",
		},
		{
			name: "store_negative",
			instr: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRStoreLocal, Local: -1},
			},
			want: "local slot -1 out of bounds",
		},
		{
			name: "store_past_end",
			instr: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRStoreLocal, Local: 1},
			},
			want: "local slot 1 out of bounds",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			emitFn := NewEmitFunc(x64abi.LinuxSysV())
			e := &x64.Emitter{}
			var dataBlobs [][]byte
			var leaPatches []x64obj.LeaPatch
			var callPatches []x64obj.CallPatch
			var importPatches []x64obj.ImportPatch
			err := emitFn(e, ir.IRFunc{
				Name:        "__test_bad_local",
				ParamSlots:  0,
				LocalSlots:  1,
				ReturnSlots: 0,
				Instrs:      tc.instr,
			}, &dataBlobs, &leaPatches, &callPatches, &importPatches, x64.CodegenOptions{})
			if err == nil {
				t.Fatalf("expected local slot diagnostic")
			}
			for _, want := range []string{tc.want, "function '__test_bad_local'", "locals=1"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error = %v, want substring %q", err, want)
				}
			}
		})
	}
}

func TestABIDiagnosticEmitSharedRejectsNegativeGlobalSlots(t *testing.T) {
	cases := []struct {
		name  string
		instr []ir.IRInstr
	}{
		{
			name: "load_global",
			instr: []ir.IRInstr{
				{Kind: ir.IRLoadGlobal, Local: -1},
			},
		},
		{
			name: "store_global",
			instr: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRStoreGlobal, Local: -1},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			emitFn := NewEmitFunc(x64abi.LinuxSysV())
			e := &x64.Emitter{}
			var dataBlobs [][]byte
			var leaPatches []x64obj.LeaPatch
			var callPatches []x64obj.CallPatch
			var importPatches []x64obj.ImportPatch
			err := emitFn(e, ir.IRFunc{
				Name:        "__test_bad_global",
				ParamSlots:  0,
				LocalSlots:  0,
				ReturnSlots: 0,
				Instrs:      tc.instr,
			}, &dataBlobs, &leaPatches, &callPatches, &importPatches, x64.CodegenOptions{})
			if err == nil {
				t.Fatalf("expected global slot diagnostic")
			}
			if !strings.Contains(
				err.Error(),
				"global slot -1 out of bounds in function '__test_bad_global'",
			) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestABIDiagnosticEmitSharedRejectsDuplicateLabels(t *testing.T) {
	emitFn := NewEmitFunc(x64abi.LinuxSysV())
	e := &x64.Emitter{}
	var dataBlobs [][]byte
	var leaPatches []x64obj.LeaPatch
	var callPatches []x64obj.CallPatch
	var importPatches []x64obj.ImportPatch
	err := emitFn(e, ir.IRFunc{
		Name:        "__test_duplicate_label",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 0,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLabel, Label: 7},
			{Kind: ir.IRLabel, Label: 7},
		},
	}, &dataBlobs, &leaPatches, &callPatches, &importPatches, x64.CodegenOptions{})
	if err == nil {
		t.Fatalf("expected duplicate label diagnostic")
	}
	if !strings.Contains(err.Error(), "duplicate label 7 in function '__test_duplicate_label'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestABIDiagnosticEmitSharedRejectsNegativeBranchLabels(t *testing.T) {
	emitFn := NewEmitFunc(x64abi.LinuxSysV())
	e := &x64.Emitter{}
	var dataBlobs [][]byte
	var leaPatches []x64obj.LeaPatch
	var callPatches []x64obj.CallPatch
	var importPatches []x64obj.ImportPatch
	err := emitFn(e, ir.IRFunc{
		Name:        "__test_negative_label",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRJmp, Label: -1},
			{Kind: ir.IRLabel, Label: -1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}, &dataBlobs, &leaPatches, &callPatches, &importPatches, x64.CodegenOptions{})
	if err == nil {
		t.Fatalf("expected negative label diagnostic")
	}
	if !strings.Contains(err.Error(), "negative label -1 in function '__test_negative_label'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestABIDiagnosticEmitSharedRejectsMissingSymAddrName(t *testing.T) {
	emitFn := NewEmitFunc(x64abi.LinuxSysV())
	e := &x64.Emitter{}
	var dataBlobs [][]byte
	var leaPatches []x64obj.LeaPatch
	var callPatches []x64obj.CallPatch
	var importPatches []x64obj.ImportPatch
	err := emitFn(e, ir.IRFunc{
		Name:        "__test_missing_symbol_name",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRSymAddr},
			{Kind: ir.IRReturn},
		},
	}, &dataBlobs, &leaPatches, &callPatches, &importPatches, x64.CodegenOptions{})
	if err == nil {
		t.Fatalf("expected missing symbol address name diagnostic")
	}
	if !strings.Contains(err.Error(), "symbol address is missing name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestABIDiagnosticEmitSharedRejectsUnsupportedReturnSlots(t *testing.T) {
	emitFn := NewEmitFunc(x64abi.LinuxSysV())
	e := &x64.Emitter{}
	var dataBlobs [][]byte
	var leaPatches []x64obj.LeaPatch
	var callPatches []x64obj.CallPatch
	var importPatches []x64obj.ImportPatch
	err := emitFn(e, ir.IRFunc{
		Name:        "__test_bad_return_slots",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 11,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRConstI32, Imm: 5},
			{Kind: ir.IRConstI32, Imm: 6},
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRConstI32, Imm: 8},
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRConstI32, Imm: 10},
			{Kind: ir.IRConstI32, Imm: 11},
			{Kind: ir.IRReturn},
		},
	}, &dataBlobs, &leaPatches, &callPatches, &importPatches, x64.CodegenOptions{})
	if err == nil || !strings.Contains(err.Error(), "unsupported return slots") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestABIBuildOnlyEmitSharedAcrossABIs(t *testing.T) {
	cases := []struct {
		name string
		abi  x64abi.ABI
	}{
		{name: "sysv", abi: x64abi.LinuxSysV()},
		{name: "win64", abi: x64abi.NewWin64()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn := ir.IRFunc{
				Name:        "__test_build_only_" + tc.name,
				ParamSlots:  0,
				LocalSlots:  0,
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRReturn},
				},
			}
			buf := emitOneFunc(t, tc.abi, fn)
			if len(buf) == 0 {
				t.Fatalf("empty emission")
			}
		})
	}
}

func TestPointerMemoryOpsHonorConfiguredPointerWidth(t *testing.T) {
	readFn := ir.IRFunc{
		Name:        "__test_ptr_read_x32",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRMemReadPtr},
			{Kind: ir.IRReturn},
		},
	}
	read := emitOneFuncWithOptions(
		t,
		x64abi.LinuxSysV(),
		readFn,
		x64.CodegenOptions{PointerWidthBits: 32},
	)
	assertContainsBytes(t, "x32 ptr read guard width", read, addEdxImm32Bytes(4))
	wantLoad32 := &x64.Emitter{}
	wantLoad32.MovEaxFromRaxPtr()
	assertContainsBytes(t, "x32 ptr read 32-bit load", read, wantLoad32.Buf)
	forbidLoad64 := &x64.Emitter{}
	forbidLoad64.MovRaxFromRdiDisp(0)
	assertNotContainsBytes(t, "x32 ptr read 64-bit load", read, forbidLoad64.Buf)

	writeFn := ir.IRFunc{
		Name:        "__test_ptr_write_x32",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRMemWritePtr},
			{Kind: ir.IRReturn},
		},
	}
	write := emitOneFuncWithOptions(
		t,
		x64abi.LinuxSysV(),
		writeFn,
		x64.CodegenOptions{PointerWidthBits: 32},
	)
	assertContainsBytes(t, "x32 ptr write guard width", write, addEdxImm32Bytes(4))
	assertContainsBytes(
		t,
		"x32 ptr write zero-extends returned pointer",
		write,
		[]byte{0x45, 0x89, 0xC0},
	)
	wantStore32 := &x64.Emitter{}
	wantStore32.MovMem32RdiDispR8d(0)
	assertContainsBytes(t, "x32 ptr write 32-bit store", write, wantStore32.Buf)
	forbidStore64 := &x64.Emitter{}
	forbidStore64.MovMem64RdiDispR8(0)
	assertNotContainsBytes(t, "x32 ptr write 64-bit store", write, forbidStore64.Buf)

	archWriteFn := ir.IRFunc{
		Name:        "__test_arch_ptr_write_x32",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRMemWriteArchPtr},
			{Kind: ir.IRReturn},
		},
	}
	archWrite := emitOneFuncWithOptions(
		t,
		x64abi.LinuxSysV(),
		archWriteFn,
		x64.CodegenOptions{PointerWidthBits: 32, RegisterWidthBits: 64},
	)
	assertContainsBytes(t, "x32 arch ptr write guard width", archWrite, addEdxImm32Bytes(8))
	assertContainsBytes(t, "x32 arch ptr write 64-bit store", archWrite, forbidStore64.Buf)
	assertNotContainsBytes(t, "x32 arch ptr write 32-bit store", archWrite, wantStore32.Buf)

	offsetReadFn := ir.IRFunc{
		Name:        "__test_ptr_offset_read_x32",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRMemReadPtrOffset},
			{Kind: ir.IRReturn},
		},
	}
	offsetRead := emitOneFuncWithOptions(
		t,
		x64abi.LinuxSysV(),
		offsetReadFn,
		x64.CodegenOptions{PointerWidthBits: 32},
	)
	assertContainsBytes(t, "x32 ptr offset read guard width", offsetRead, addEdxImm32Bytes(4))
	assertContainsBytes(t, "x32 ptr offset read 32-bit load", offsetRead, wantLoad32.Buf)
	assertNotContainsBytes(t, "x32 ptr offset read 64-bit load", offsetRead, forbidLoad64.Buf)

	offsetWriteFn := ir.IRFunc{
		Name:        "__test_ptr_offset_write_x32",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRMemWritePtrOffset},
			{Kind: ir.IRReturn},
		},
	}
	offsetWrite := emitOneFuncWithOptions(
		t,
		x64abi.LinuxSysV(),
		offsetWriteFn,
		x64.CodegenOptions{PointerWidthBits: 32},
	)
	assertContainsBytes(t, "x32 ptr offset write guard width", offsetWrite, addEdxImm32Bytes(4))
	assertContainsBytes(
		t,
		"x32 ptr offset write zero-extends returned pointer",
		offsetWrite,
		[]byte{0x45, 0x89, 0xC0},
	)
	assertContainsBytes(t, "x32 ptr offset write 32-bit store", offsetWrite, wantStore32.Buf)
	assertNotContainsBytes(t, "x32 ptr offset write 64-bit store", offsetWrite, forbidStore64.Buf)

	archOffsetWriteFn := ir.IRFunc{
		Name:        "__test_arch_ptr_offset_write_x32",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRMemWriteArchPtrOffset},
			{Kind: ir.IRReturn},
		},
	}
	archOffsetWrite := emitOneFuncWithOptions(
		t,
		x64abi.LinuxSysV(),
		archOffsetWriteFn,
		x64.CodegenOptions{PointerWidthBits: 32, RegisterWidthBits: 64},
	)
	assertContainsBytes(
		t,
		"x32 arch ptr offset write guard width",
		archOffsetWrite,
		addEdxImm32Bytes(8),
	)
	assertContainsBytes(
		t,
		"x32 arch ptr offset write 64-bit store",
		archOffsetWrite,
		forbidStore64.Buf,
	)
	assertNotContainsBytes(
		t,
		"x32 arch ptr offset write 32-bit store",
		archOffsetWrite,
		wantStore32.Buf,
	)
}

func TestPointerMemoryOpsDefaultToX64PointerWidth(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "__test_ptr_read_x64",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRMemReadPtr},
			{Kind: ir.IRReturn},
		},
	}
	read := emitOneFunc(t, x64abi.LinuxSysV(), fn)
	assertContainsBytes(t, "x64 ptr read guard width", read, addEdxImm32Bytes(8))
	wantLoad64 := &x64.Emitter{}
	wantLoad64.MovRaxFromRdiDisp(0)
	assertContainsBytes(t, "x64 ptr read 64-bit load", read, wantLoad64.Buf)
}

func TestPointerMemoryOpsRejectInvalidConfiguredPointerWidth(t *testing.T) {
	emitFn := NewEmitFunc(x64abi.LinuxSysV())
	e := &x64.Emitter{}
	var dataBlobs [][]byte
	var leaPatches []x64obj.LeaPatch
	var callPatches []x64obj.CallPatch
	var importPatches []x64obj.ImportPatch
	err := emitFn(
		e,
		ir.IRFunc{Name: "__test_bad_pointer_width"},
		&dataBlobs,
		&leaPatches,
		&callPatches,
		&importPatches,
		x64.CodegenOptions{PointerWidthBits: 48},
	)
	if err == nil || !strings.Contains(err.Error(), "unsupported pointer width 48") {
		t.Fatalf("expected unsupported pointer width diagnostic, got %v", err)
	}
}

func TestHashTableLookupRegisterEmitterAcceptsOnlyExactShape(t *testing.T) {
	fn := hashTableLookupStackIRFunc()

	direct := &x64.Emitter{}
	ok, err := emitHashTableLookupRegisterFunction(
		direct,
		fn,
		x64abi.LinuxSysV(),
		x64.CodegenOptions{},
		nil,
	)
	if err != nil {
		t.Fatalf("emitHashTableLookupRegisterFunction exact: %v", err)
	}
	if !ok {
		t.Fatalf(
			"emitHashTableLookupRegisterFunction did not accept exact p25.hash_table.lookup shape",
		)
	}
	routed := emitOneFunc(t, x64abi.LinuxSysV(), fn)
	if !bytes.Equal(routed, direct.Buf) {
		t.Fatalf(
			"NewEmitFunc did not route exact hash lookup through register emitter\nrouted=% x\ndirect=% x",
			routed,
			direct.Buf,
		)
	}

	nearMiss := hashTableLookupStackIRFunc()
	nearMiss.Instrs[18].Kind = ir.IRStoreLocal
	nearMiss.Instrs[18].Local = 6
	miss := &x64.Emitter{}
	ok, err = emitHashTableLookupRegisterFunction(
		miss,
		nearMiss,
		x64abi.LinuxSysV(),
		x64.CodegenOptions{},
		nil,
	)
	if err != nil || ok {
		t.Fatalf(
			"emitHashTableLookupRegisterFunction near-miss ok=%v err=%v, want strict fallback without error",
			ok,
			err,
		)
	}
}

func TestHashTableMainRegisterEmitterEmitsExactNativeSlicePath(t *testing.T) {
	fn := hashTableMainStackIRFunc()
	plan, ok, err := machine.HashTableMainPlanFromStackIR(fn)
	if err != nil {
		t.Fatalf("HashTableMainPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("HashTableMainPlanFromStackIR did not accept exact hash main shape")
	}

	direct := &x64.Emitter{}
	var directCalls []x64obj.CallPatch
	ok, err = emitHashTableMainRegisterFunction(
		direct,
		fn,
		x64abi.LinuxSysV(),
		&directCalls,
		x64.CodegenOptions{},
		nil,
	)
	if err != nil {
		t.Fatalf("emitHashTableMainRegisterFunction exact: %v", err)
	}
	if !ok {
		t.Fatalf("emitHashTableMainRegisterFunction did not accept exact p25.hash_table.main shape")
	}
	assertHasRegisterFrame(t, "hash_table main", direct.Buf, fn)
	length := &x64.Emitter{}
	length.MovEaxImm32(uint32(plan.Length))
	assertContainsBytes(t, "hash_table main length constant", direct.Buf, length.Buf)
	assertCallPatchCounts(t, directCalls, map[string]int{
		"p25.hash_table.lookup": 1,
	})

	routed := emitWithArtifacts(t, x64abi.LinuxSysV(), fn)
	if !bytes.Equal(routed.code, direct.Buf) {
		t.Fatalf(
			"NewEmitFunc did not route exact hash_table main through register emitter\nrouted=% x\ndirect=% x",
			routed.code,
			direct.Buf,
		)
	}
	assertCallPatchCounts(t, routed.callPatches, map[string]int{
		"p25.hash_table.lookup": 1,
	})
}

func TestHashTableMainRegisterEmitterRejectsNearMisses(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   func() ir.IRFunc
		opt  x64.CodegenOptions
	}{
		{
			name: "altered_call_target",
			fn: func() ir.IRFunc {
				fn := hashTableMainStackIRFunc()
				fn.Instrs[61].Name = "p25.hash_table.lookup_probe"
				return fn
			},
		},
		{
			name: "altered_call_arg_slots",
			fn: func() ir.IRFunc {
				fn := hashTableMainStackIRFunc()
				fn.Instrs[61].ArgSlots = 5
				return fn
			},
		},
		{
			name: "altered_call_return_slots",
			fn: func() ir.IRFunc {
				fn := hashTableMainStackIRFunc()
				fn.Instrs[61].RetSlots = 2
				return fn
			},
		},
		{
			name: "disable_machine_paths",
			fn: func() ir.IRFunc {
				return hashTableMainStackIRFunc()
			},
			opt: x64.CodegenOptions{DisableMachinePaths: true},
		},
		{
			name: "register_width_32",
			fn: func() ir.IRFunc {
				return hashTableMainStackIRFunc()
			},
			opt: x64.CodegenOptions{RegisterWidthBits: 32},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			miss := &x64.Emitter{}
			var calls []x64obj.CallPatch
			ok, err := emitHashTableMainRegisterFunction(
				miss,
				tc.fn(),
				x64abi.LinuxSysV(),
				&calls,
				tc.opt,
				nil,
			)
			if err != nil || ok {
				t.Fatalf(
					"emitHashTableMainRegisterFunction near-miss ok=%v err=%v, want strict fallback",
					ok,
					err,
				)
			}
		})
	}
}

func TestPostgreSQLFrameTypeAtRegisterEmitterAcceptsOnlyExactShape(t *testing.T) {
	fn := postgresqlFrameTypeAtStackIRFunc()

	direct := &x64.Emitter{}
	ok, err := emitPostgreSQLFrameTypeAtRegisterFunction(
		direct,
		fn,
		x64abi.LinuxSysV(),
		x64.CodegenOptions{},
		nil,
	)
	if err != nil {
		t.Fatalf("emitPostgreSQLFrameTypeAtRegisterFunction exact: %v", err)
	}
	if !ok {
		t.Fatalf(
			"emitPostgreSQLFrameTypeAtRegisterFunction did not accept exact frame_type_at shape",
		)
	}
	wantLoad := &x64.Emitter{}
	wantLoad.MovzxEaxBytePtrRsiRcx()
	assertContainsBytes(t, "frame_type_at register load", direct.Buf, wantLoad.Buf)
	for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
		assertNotContainsBytes(t, "frame_type_at stack-machine push/pop", direct.Buf, forbidden)
	}

	routed := emitOneFunc(t, x64abi.LinuxSysV(), fn)
	if !bytes.Equal(routed, direct.Buf) {
		t.Fatalf(
			"NewEmitFunc did not route exact frame_type_at through register emitter\nrouted=% x\ndirect=% x",
			routed,
			direct.Buf,
		)
	}

	nearMiss := postgresqlFrameTypeAtStackIRFunc()
	nearMiss.Instrs[3].Kind = ir.IRIndexLoadU8
	nearMiss.Instrs[3].ProofID = ""
	miss := &x64.Emitter{}
	ok, err = emitPostgreSQLFrameTypeAtRegisterFunction(
		miss,
		nearMiss,
		x64abi.LinuxSysV(),
		x64.CodegenOptions{},
		nil,
	)
	if err != nil || ok {
		t.Fatalf(
			("emitPostgreSQLFrameTypeAtRegisterFunction near-miss ok=%v " +
				"err=%v, want strict fallback without error"),
			ok,
			err,
		)
	}
}

func TestPostgreSQLInoutWriterRegisterEmitterAcceptsOnlyExactShape(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   ir.IRFunc
	}{
		{name: "write_i32_be_at", fn: postgresqlInoutWriterI32StackIRFunc()},
		{name: "write_i16_be_at", fn: postgresqlInoutWriterI16StackIRFunc()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			direct := &x64.Emitter{}
			ok, err := emitPostgreSQLInoutWriterRegisterFunction(
				direct,
				tc.fn,
				x64abi.LinuxSysV(),
				x64.CodegenOptions{},
				nil,
			)
			if err != nil {
				t.Fatalf("emitPostgreSQLInoutWriterRegisterFunction exact: %v", err)
			}
			if !ok {
				t.Fatalf(
					"emitPostgreSQLInoutWriterRegisterFunction did not accept exact %s shape",
					tc.name,
				)
			}
			wantStore := &x64.Emitter{}
			wantStore.MovMem8RaxPtrCl()
			assertContainsBytes(t, tc.name+" register u8 store", direct.Buf, wantStore.Buf)
			for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
				assertNotContainsBytes(t, tc.name+" stack-machine push/pop", direct.Buf, forbidden)
			}

			routed := emitOneFunc(t, x64abi.LinuxSysV(), tc.fn)
			if !bytes.Equal(routed, direct.Buf) {
				t.Fatalf(
					"NewEmitFunc did not route exact %s through register emitter\nrouted=% x\ndirect=% x",
					tc.name,
					routed,
					direct.Buf,
				)
			}
		})
	}

	nearMiss := postgresqlInoutWriterI32StackIRFunc()
	nearMiss.ReturnSlots = 1
	miss := &x64.Emitter{}
	ok, err := emitPostgreSQLInoutWriterRegisterFunction(
		miss,
		nearMiss,
		x64abi.LinuxSysV(),
		x64.CodegenOptions{},
		nil,
	)
	if err != nil || ok {
		t.Fatalf(
			("emitPostgreSQLInoutWriterRegisterFunction near-miss ok=%v " +
				"err=%v, want strict fallback without error"),
			ok,
			err,
		)
	}
}

func TestInoutWriterHelperSummaryRegisterEmitterAcceptsExactWriterHelpers(t *testing.T) {
	for _, tc := range []struct {
		name       string
		helperName string
		stores     int
	}{
		{
			name:       "json_write_message_object",
			helperName: "p25.json_parse_stringify.write_message_object",
			stores:     27,
		},
		{
			name:       "http_write_plaintext_response",
			helperName: "p25.http_plaintext_json.write_plaintext_response",
			stores:     24,
		},
		{
			name:       "http_write_json_response",
			helperName: "p25.http_plaintext_json.write_json_response",
			stores:     21,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := inoutWriterHelperSummaryWriterStackIRFunc(tc.helperName, tc.stores)
			direct := &x64.Emitter{}
			ok, err := emitInoutWriterHelperSummaryRegisterFunction(
				direct,
				fn,
				x64abi.LinuxSysV(),
				x64.CodegenOptions{},
				nil,
			)
			if err != nil {
				t.Fatalf("emitInoutWriterHelperSummaryRegisterFunction exact: %v", err)
			}
			if !ok {
				t.Fatalf(
					"emitInoutWriterHelperSummaryRegisterFunction did not accept exact %s shape",
					tc.name,
				)
			}

			firstStore := &x64.Emitter{}
			firstStore.MovMem8RdiDispDl(0)
			assertContainsBytes(t, tc.name+" first u8 store", direct.Buf, firstStore.Buf)
			lastStore := &x64.Emitter{}
			lastStore.MovMem8RdiDispDl(int32(tc.stores - 1))
			assertContainsBytes(t, tc.name+" last u8 store", direct.Buf, lastStore.Buf)

			firstValue := &x64.Emitter{}
			firstValue.MovEdxImm32(65)
			assertContainsBytes(t, tc.name+" first const byte", direct.Buf, firstValue.Buf)
			lastValue := &x64.Emitter{}
			lastValue.MovEdxImm32(uint32(65 + (tc.stores-1)%26))
			assertContainsBytes(t, tc.name+" last const byte", direct.Buf, lastValue.Buf)

			wantReturn := &x64.Emitter{}
			wantReturn.MovEaxImm32(uint32(tc.stores))
			assertContainsBytes(t, tc.name+" scalar return constant", direct.Buf, wantReturn.Buf)

			routed := emitOneFunc(t, x64abi.LinuxSysV(), fn)
			if !bytes.Equal(routed, direct.Buf) {
				t.Fatalf(
					"NewEmitFunc did not route exact %s through helper-summary emitter\nrouted=% x\ndirect=% x",
					tc.name,
					routed,
					direct.Buf,
				)
			}
		})
	}
}

func TestInoutWriterHelperSummaryRegisterEmitterRejectsNearMisses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*ir.IRFunc)
	}{
		{
			name: "wrong_helper_name",
			mutate: func(fn *ir.IRFunc) {
				fn.Name = "p25.json_parse_stringify.write_other"
			},
		},
		{
			name: "wrong_ReturnSlots",
			mutate: func(fn *ir.IRFunc) {
				fn.ReturnSlots = 2
			},
		},
		{
			name: "missing_proof_family",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[4].ProofID = ""
			},
		},
		{
			name: "wrong_proof_family",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[4].ProofID = "proof:helper-offset:start:dst:15:5"
			},
		},
		{
			name: "dynamic_index",
			mutate: func(fn *ir.IRFunc) {
				fn.LocalSlots = 3
				fn.Instrs[2] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 2}
			},
		},
		{
			name: "non_constant_store_value",
			mutate: func(fn *ir.IRFunc) {
				fn.LocalSlots = 3
				fn.Instrs[3] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 2}
			},
		},
		{
			name: "extra_IRCall",
			mutate: func(fn *ir.IRFunc) {
				returnAt := len(fn.Instrs) - 1
				fn.Instrs = append(
					fn.Instrs[:returnAt],
					append(
						[]ir.IRInstr{{Kind: ir.IRCall, Name: "side_effect", ArgSlots: 0, RetSlots: 0}},
						fn.Instrs[returnAt:]...,
					)...,
				)
			},
		},
		{
			name: "extra_label",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs = append(
					fn.Instrs[:5],
					append([]ir.IRInstr{{Kind: ir.IRLabel, Label: 7}}, fn.Instrs[5:]...)...,
				)
			},
		},
		{
			name: "extra_jump",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs = append(
					fn.Instrs[:5],
					append([]ir.IRInstr{{Kind: ir.IRJmp, Label: 7}}, fn.Instrs[5:]...)...,
				)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := inoutWriterHelperSummaryWriterStackIRFunc(
				"p25.json_parse_stringify.write_message_object",
				27,
			)
			tc.mutate(&fn)
			miss := &x64.Emitter{}
			ok, err := emitInoutWriterHelperSummaryRegisterFunction(
				miss,
				fn,
				x64abi.LinuxSysV(),
				x64.CodegenOptions{},
				nil,
			)
			if err != nil || ok {
				t.Fatalf(
					("emitInoutWriterHelperSummaryRegisterFunction near-miss ok=%v " +
						"err=%v, want strict fallback without error"),
					ok,
					err,
				)
			}
		})
	}
}

func TestInoutWriterHelperSummaryRegisterEmitterHonorsMachinePathOptions(t *testing.T) {
	fn := inoutWriterHelperSummaryWriterStackIRFunc(
		"p25.json_parse_stringify.write_message_object",
		27,
	)
	for _, tc := range []struct {
		name string
		opt  x64.CodegenOptions
	}{
		{
			name: "DisableMachinePaths",
			opt:  x64.CodegenOptions{DisableMachinePaths: true},
		},
		{
			name: "register_width_32",
			opt:  x64.CodegenOptions{RegisterWidthBits: 32},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			miss := &x64.Emitter{}
			ok, err := emitInoutWriterHelperSummaryRegisterFunction(
				miss,
				fn,
				x64abi.LinuxSysV(),
				tc.opt,
				nil,
			)
			if err != nil || ok {
				t.Fatalf(
					"emitInoutWriterHelperSummaryRegisterFunction ok=%v err=%v, want strict fallback",
					ok,
					err,
				)
			}
		})
	}
}

func TestInoutWriterHelperSummaryCallerRegisterEmitterAcceptsExactCallers(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   ir.IRFunc
	}{
		{
			name: "json_main",
			fn: inoutWriterHelperSummaryCallerExactJSONStackIRFunc(ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "p25.json_parse_stringify.write_message_object",
				ArgSlots: 2,
				RetSlots: 3,
			}),
		},
		{
			name: "http_main",
			fn: inoutWriterHelperSummaryCallerExactHTTPStackIRFunc(
				ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "p25.http_plaintext_json.write_plaintext_response",
					ArgSlots: 2,
					RetSlots: 3,
				},
				ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "p25.http_plaintext_json.write_json_response",
					ArgSlots: 2,
					RetSlots: 3,
				},
			),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			direct := &x64.Emitter{}
			ok, err := emitInoutWriterHelperSummaryCallerRegisterFunction(
				direct,
				tc.fn,
				x64abi.LinuxSysV(),
				x64.CodegenOptions{},
				nil,
			)
			if err != nil {
				t.Fatalf("emitInoutWriterHelperSummaryCallerRegisterFunction exact: %v", err)
			}
			if !ok {
				t.Fatalf(
					"emitInoutWriterHelperSummaryCallerRegisterFunction did not accept exact %s shape",
					tc.name,
				)
			}

			assertHasRegisterFrame(t, tc.name, direct.Buf, tc.fn)
			assertReturnsZeroInEAX(t, tc.name, direct.Buf)
			assertNotContainsBytes(t, tc.name+" generic call opcode", direct.Buf, []byte{0xE8})

			routed := emitWithArtifacts(t, x64abi.LinuxSysV(), tc.fn)
			if len(routed.callPatches) != 0 {
				t.Fatalf("%s routed through call patches: %#v", tc.name, routed.callPatches)
			}
			if !bytes.Equal(routed.code, direct.Buf) {
				t.Fatalf(
					"NewEmitFunc did not route exact %s through helper-summary "+
						"caller emitter\nrouted=% x\ndirect=% x",
					tc.name,
					routed.code,
					direct.Buf,
				)
			}
		})
	}
}

func TestInoutWriterHelperSummaryCallerRegisterEmitterRejectsNearMisses(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   func() ir.IRFunc
	}{
		{
			name: "wrong_caller_name",
			fn: func() ir.IRFunc {
				fn := inoutWriterHelperSummaryCallerExactJSONStackIRFunc(ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "p25.json_parse_stringify.write_message_object",
					ArgSlots: 2,
					RetSlots: 3,
				})
				fn.Name = "p25.json_parse_stringify.other"
				return fn
			},
		},
		{
			name: "wrong_helper_name",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryCallerExactJSONStackIRFunc(ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "p25.json_parse_stringify.write_other",
					ArgSlots: 2,
					RetSlots: 3,
				})
			},
		},
		{
			name: "missing_helper_call",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryCallerExactJSONStackIRFunc()
			},
		},
		{
			name: "duplicate_helper_call",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryCallerExactJSONStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_message_object",
						ArgSlots: 2,
						RetSlots: 3,
					},
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_message_object",
						ArgSlots: 2,
						RetSlots: 3,
					},
				)
			},
		},
		{
			name: "wrong_helper_arg_slots",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryCallerExactJSONStackIRFunc(ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "p25.json_parse_stringify.write_message_object",
					ArgSlots: 1,
					RetSlots: 3,
				})
			},
		},
		{
			name: "wrong_helper_ret_slots",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryCallerExactJSONStackIRFunc(ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "p25.json_parse_stringify.write_message_object",
					ArgSlots: 2,
					RetSlots: 2,
				})
			},
		},
		{
			name: "mixed_safe_unsafe_multi_slot_calls",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryCallerExactHTTPStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.http_plaintext_json.write_plaintext_response",
						ArgSlots: 2,
						RetSlots: 3,
					},
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.http_plaintext_json.unverified_writer",
						ArgSlots: 2,
						RetSlots: 3,
					},
				)
			},
		},
		{
			name: "generic_aggregate_return_sample",
			fn: func() ir.IRFunc {
				return ir.IRFunc{
					Name:        "slice_header_return",
					ParamSlots:  0,
					LocalSlots:  0,
					ReturnSlots: 3,
					Instrs: []ir.IRInstr{
						{Kind: ir.IRConstI32, Imm: 0},
						{Kind: ir.IRConstI32, Imm: 0},
						{Kind: ir.IRConstI32, Imm: 0},
						{Kind: ir.IRReturn},
					},
				}
			},
		},
		{
			name: "wrong_caller_return_slots",
			fn: func() ir.IRFunc {
				fn := inoutWriterHelperSummaryCallerExactJSONStackIRFunc(ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "p25.json_parse_stringify.write_message_object",
					ArgSlots: 2,
					RetSlots: 3,
				})
				fn.ReturnSlots = 2
				return fn
			},
		},
		{
			name: "missing_scalar_return_constants",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryCallerJSONStackIRFunc(ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "p25.json_parse_stringify.write_message_object",
					ArgSlots: 2,
					RetSlots: 3,
				})
			},
		},
		{
			name: "wrong_success_scalar_return",
			fn: func() ir.IRFunc {
				fn := inoutWriterHelperSummaryCallerExactJSONStackIRFunc(ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "p25.json_parse_stringify.write_message_object",
					ArgSlots: 2,
					RetSlots: 3,
				})
				replaceInoutWriterHelperSummaryCallerConstBeforeReturn(&fn, 0, 2)
				return fn
			},
		},
		{
			name: "wrong_failure_scalar_return",
			fn: func() ir.IRFunc {
				fn := inoutWriterHelperSummaryCallerExactJSONStackIRFunc(ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "p25.json_parse_stringify.write_message_object",
					ArgSlots: 2,
					RetSlots: 3,
				})
				replaceInoutWriterHelperSummaryCallerConstBeforeReturn(&fn, 1, 2)
				return fn
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			miss := &x64.Emitter{}
			ok, err := emitInoutWriterHelperSummaryCallerRegisterFunction(
				miss,
				tc.fn(),
				x64abi.LinuxSysV(),
				x64.CodegenOptions{},
				nil,
			)
			if err != nil || ok {
				t.Fatalf(
					"emitInoutWriterHelperSummaryCallerRegisterFunction "+
						"near-miss ok=%v err=%v, want strict fallback",
					ok,
					err,
				)
			}
		})
	}
}

func TestInoutWriterHelperSummaryCallerRegisterEmitterHonorsMachinePathOptions(t *testing.T) {
	fn := inoutWriterHelperSummaryCallerExactJSONStackIRFunc(ir.IRInstr{
		Kind:     ir.IRCall,
		Name:     "p25.json_parse_stringify.write_message_object",
		ArgSlots: 2,
		RetSlots: 3,
	})
	for _, tc := range []struct {
		name string
		opt  x64.CodegenOptions
	}{
		{
			name: "DisableMachinePaths",
			opt:  x64.CodegenOptions{DisableMachinePaths: true},
		},
		{
			name: "register_width_32",
			opt:  x64.CodegenOptions{RegisterWidthBits: 32},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			miss := &x64.Emitter{}
			ok, err := emitInoutWriterHelperSummaryCallerRegisterFunction(
				miss,
				fn,
				x64abi.LinuxSysV(),
				tc.opt,
				nil,
			)
			if err != nil || ok {
				t.Fatalf(
					"emitInoutWriterHelperSummaryCallerRegisterFunction ok=%v err=%v, want strict fallback",
					ok,
					err,
				)
			}
		})
	}
}

func TestParallelMapReduceMainRegisterEmitterEmitsTaskSpawnJoinPairs(t *testing.T) {
	fn := parallelMapReduceMainStackIRFunc()

	direct := &x64.Emitter{}
	var directCalls []x64obj.CallPatch
	ok, err := emitParallelMapReduceMainRegisterFunction(
		direct,
		fn,
		x64abi.LinuxSysV(),
		&directCalls,
		x64.CodegenOptions{},
		nil,
	)
	if err != nil {
		t.Fatalf("emitParallelMapReduceMainRegisterFunction exact: %v", err)
	}
	if !ok {
		t.Fatalf("emitParallelMapReduceMainRegisterFunction did not accept exact benchmark main")
	}
	assertHasRegisterFrame(t, "parallel_map_reduce main", direct.Buf, fn)
	assertCallPatchCounts(t, directCalls, map[string]int{
		"__tetra_task_spawn_i32": 3,
		"__tetra_task_join_i32":  3,
	})
	statusStore := &x64.Emitter{}
	statusStore.MovMem64RbpDispRdx(scalarRegisterSlotOffset(1))
	assertContainsBytes(t, "parallel_map_reduce status slot store", direct.Buf, statusStore.Buf)

	routed := emitWithArtifacts(t, x64abi.LinuxSysV(), fn)
	if !bytes.Equal(routed.code, direct.Buf) {
		t.Fatalf(
			"NewEmitFunc did not route exact parallel_map_reduce main "+
				"through register emitter\nrouted=% x\ndirect=% x",
			routed.code,
			direct.Buf,
		)
	}
	assertCallPatchCounts(t, routed.callPatches, map[string]int{
		"__tetra_task_spawn_i32": 3,
		"__tetra_task_join_i32":  3,
	})
}

func TestParallelMapReduceMainRegisterEmitterRejectsNearMisses(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   func() ir.IRFunc
		opt  x64.CodegenOptions
	}{
		{
			name: "different_runtime_call_ret_slots_2",
			fn: func() ir.IRFunc {
				fn := parallelMapReduceMainStackIRFunc()
				fn.Instrs[1].Name = "__tetra_task_poll_i32"
				return fn
			},
		},
		{
			name: "spawn_ret_slots_3",
			fn: func() ir.IRFunc {
				fn := parallelMapReduceMainStackIRFunc()
				fn.Instrs[1].RetSlots = 3
				return fn
			},
		},
		{
			name: "missing_right_join",
			fn: func() ir.IRFunc {
				fn := parallelMapReduceMainStackIRFunc()
				fn.Instrs = append(fn.Instrs[:19], fn.Instrs[22:]...)
				return fn
			},
		},
		{
			name: "disable_machine_paths",
			fn: func() ir.IRFunc {
				return parallelMapReduceMainStackIRFunc()
			},
			opt: x64.CodegenOptions{DisableMachinePaths: true},
		},
		{
			name: "register_width_32",
			fn: func() ir.IRFunc {
				return parallelMapReduceMainStackIRFunc()
			},
			opt: x64.CodegenOptions{RegisterWidthBits: 32},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			miss := &x64.Emitter{}
			var calls []x64obj.CallPatch
			ok, err := emitParallelMapReduceMainRegisterFunction(
				miss,
				tc.fn(),
				x64abi.LinuxSysV(),
				&calls,
				tc.opt,
				nil,
			)
			if err != nil || ok {
				t.Fatalf(
					"emitParallelMapReduceMainRegisterFunction near-miss ok=%v err=%v, want strict fallback",
					ok,
					err,
				)
			}
		})
	}
}

func TestActorPingPongRuntimeCallRegisterEmitterEmitsExactScalarShapes(t *testing.T) {
	for _, tc := range []struct {
		name      string
		fn        ir.IRFunc
		wantCalls map[string]int
	}{
		{
			name: "pong",
			fn:   actorPingPongPongStackIRFunc(),
			wantCalls: map[string]int{
				"__tetra_actor_recv":   1,
				"__tetra_actor_sender": 1,
				"__tetra_actor_send":   1,
			},
		},
		{
			name: "main",
			fn:   actorPingPongMainStackIRFunc(),
			wantCalls: map[string]int{
				"__tetra_actor_spawn": 1,
				"__tetra_actor_send":  1,
				"__tetra_actor_recv":  1,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, ok, err := machine.ActorPingPongRuntimeCallPlanFromStackIRWithCallABI(
				tc.fn,
				machine.SysVCallABIInfo(),
			)
			if err != nil {
				t.Fatalf("ActorPingPongRuntimeCallPlanFromStackIRWithCallABI: %v", err)
			}
			if !ok {
				t.Fatalf("ActorPingPongRuntimeCallPlanFromStackIRWithCallABI rejected %s", tc.name)
			}
			direct := &x64.Emitter{}
			var directCalls []x64obj.CallPatch
			ok, err = emitActorPingPongRuntimeCallRegisterFunction(
				direct,
				tc.fn,
				x64abi.LinuxSysV(),
				&directCalls,
				x64.CodegenOptions{},
				nil,
			)
			if err != nil {
				t.Fatalf("emitActorPingPongRuntimeCallRegisterFunction exact: %v", err)
			}
			if !ok {
				t.Fatalf("emitActorPingPongRuntimeCallRegisterFunction rejected %s", tc.name)
			}
			actorFrame := &x64.Emitter{}
			actorFrame.SubRspImm32(int32(x64.AlignStackSize((tc.fn.LocalSlots + 2) * 8)))
			assertContainsBytes(t, "actor_ping_pong "+tc.name+" frame", direct.Buf, actorFrame.Buf)
			assertCallPatchCounts(t, directCalls, tc.wantCalls)
			assertReturnsZeroInEAX(t, "actor_ping_pong "+tc.name, direct.Buf)

			routed := emitWithArtifacts(t, x64abi.LinuxSysV(), tc.fn)
			if !bytes.Equal(routed.code, direct.Buf) {
				t.Fatalf(
					"NewEmitFunc did not route exact actor ping-pong %s through "+
						"register emitter path %s\nrouted=% x\ndirect=% x",
					tc.name,
					plan.Path,
					routed.code,
					direct.Buf,
				)
			}
			assertCallPatchCounts(t, routed.callPatches, tc.wantCalls)
		})
	}
}

func TestActorPingPongRuntimeCallRegisterEmitterRejectsNearMisses(t *testing.T) {
	insert := func(fn ir.IRFunc, at int, instr ir.IRInstr) ir.IRFunc {
		instrs := append([]ir.IRInstr(nil), fn.Instrs[:at]...)
		instrs = append(instrs, instr)
		instrs = append(instrs, fn.Instrs[at:]...)
		fn.Instrs = instrs
		return fn
	}
	for _, tc := range []struct {
		name string
		fn   func() ir.IRFunc
		opt  x64.CodegenOptions
	}{
		{
			name: "pong_extra_runtime_call",
			fn: func() ir.IRFunc {
				return insert(
					actorPingPongPongStackIRFunc(),
					2,
					ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_recv_poll", ArgSlots: 0, RetSlots: 2},
				)
			},
		},
		{
			name: "pong_typed_message_send",
			fn: func() ir.IRFunc {
				fn := actorPingPongPongStackIRFunc()
				fn.Instrs[8] = ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "__tetra_actor_send_msg",
					ArgSlots: 3,
					RetSlots: 1,
				}
				return fn
			},
		},
		{
			name: "main_recv_multi_slot",
			fn: func() ir.IRFunc {
				fn := actorPingPongMainStackIRFunc()
				fn.Instrs[8] = ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "__tetra_actor_recv_msg",
					ArgSlots: 0,
					RetSlots: 2,
				}
				return fn
			},
		},
		{
			name: "main_different_branch_literal",
			fn: func() ir.IRFunc {
				fn := actorPingPongMainStackIRFunc()
				fn.Instrs[10].Imm = 43
				return fn
			},
		},
		{
			name: "disable_machine_paths",
			fn: func() ir.IRFunc {
				return actorPingPongMainStackIRFunc()
			},
			opt: x64.CodegenOptions{DisableMachinePaths: true},
		},
		{
			name: "register_width_32",
			fn: func() ir.IRFunc {
				return actorPingPongMainStackIRFunc()
			},
			opt: x64.CodegenOptions{RegisterWidthBits: 32},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			miss := &x64.Emitter{}
			var calls []x64obj.CallPatch
			ok, err := emitActorPingPongRuntimeCallRegisterFunction(
				miss,
				tc.fn(),
				x64abi.LinuxSysV(),
				&calls,
				tc.opt,
				nil,
			)
			if err != nil || ok {
				t.Fatalf(
					"emitActorPingPongRuntimeCallRegisterFunction near-miss ok=%v err=%v, want strict fallback",
					ok,
					err,
				)
			}
		})
	}
}

func TestSliceSumMainRegisterEmitterEmitsExactBenchmarkLoopNest(t *testing.T) {
	fn := sliceSumMainStackIRFunc()
	plan, ok, err := machine.SliceSumMainPlanFromStackIR(fn)
	if err != nil {
		t.Fatalf("SliceSumMainPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("SliceSumMainPlanFromStackIR did not accept exact benchmark main")
	}
	if plan.Function.Target != "slice-sum-main" {
		t.Fatalf("slice_sum main machine target = %q, want slice-sum-main", plan.Function.Target)
	}

	direct := &x64.Emitter{}
	ok, err = emitSliceSumMainRegisterFunction(
		direct,
		fn,
		x64abi.LinuxSysV(),
		x64.CodegenOptions{},
		nil,
	)
	if err != nil {
		t.Fatalf("emitSliceSumMainRegisterFunction exact: %v", err)
	}
	if !ok {
		t.Fatalf("emitSliceSumMainRegisterFunction did not accept exact benchmark main")
	}
	assertHasRegisterFrame(t, "slice_sum main", direct.Buf, fn)
	modulus := &x64.Emitter{}
	modulus.MovR8dImm32(uint32(plan.FillModulus))
	assertContainsBytes(t, "slice_sum fill modulus", direct.Buf, modulus.Buf)
	repeat := &x64.Emitter{}
	repeat.CmpRcxImm32(plan.RepeatCount)
	assertContainsBytes(t, "slice_sum repeat count", direct.Buf, repeat.Buf)
	assertReturnsZeroInEAX(t, "slice_sum main", direct.Buf)

	routed := emitWithArtifacts(t, x64abi.LinuxSysV(), fn)
	if !bytes.Equal(routed.code, direct.Buf) {
		t.Fatalf(
			"NewEmitFunc did not route exact slice_sum main through "+
				"register emitter\nrouted=% x\ndirect=% x",
			routed.code,
			direct.Buf,
		)
	}
	if len(routed.callPatches) != 0 {
		t.Fatalf("slice_sum main emitted call patches: %#v", routed.callPatches)
	}
}

func TestSliceSumMainRegisterEmitterRejectsNearMisses(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   func() ir.IRFunc
		opt  x64.CodegenOptions
	}{
		{
			name: "altered_modulus",
			fn: func() ir.IRFunc {
				fn := sliceSumMainStackIRFunc()
				fn.Instrs[17].Imm = 96
				return fn
			},
		},
		{
			name: "checked_load_without_proof",
			fn: func() ir.IRFunc {
				fn := sliceSumMainStackIRFunc()
				fn.Instrs[46].Kind = ir.IRIndexLoadI32
				fn.Instrs[46].ProofID = ""
				return fn
			},
		},
		{
			name: "disable_machine_paths",
			fn: func() ir.IRFunc {
				return sliceSumMainStackIRFunc()
			},
			opt: x64.CodegenOptions{DisableMachinePaths: true},
		},
		{
			name: "register_width_32",
			fn: func() ir.IRFunc {
				return sliceSumMainStackIRFunc()
			},
			opt: x64.CodegenOptions{RegisterWidthBits: 32},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			miss := &x64.Emitter{}
			ok, err := emitSliceSumMainRegisterFunction(
				miss,
				tc.fn(),
				x64abi.LinuxSysV(),
				tc.opt,
				nil,
			)
			if err != nil || ok {
				t.Fatalf(
					"emitSliceSumMainRegisterFunction near-miss ok=%v err=%v, want strict fallback",
					ok,
					err,
				)
			}
		})
	}
}

func TestMatrixMultiplyMainRegisterEmitterEmitsExactBenchmarkLoopNest(t *testing.T) {
	fn := matrixMultiplyMainStackIRFunc()
	plan, ok, err := machine.MatrixMultiplyMainPlanFromStackIR(fn)
	if err != nil {
		t.Fatalf("MatrixMultiplyMainPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("MatrixMultiplyMainPlanFromStackIR did not accept exact benchmark main")
	}
	if plan.Function.Target != "matrix-multiply-main" {
		t.Fatalf(
			"matrix_multiply main machine target = %q, want matrix-multiply-main",
			plan.Function.Target,
		)
	}

	direct := &x64.Emitter{}
	ok, err = emitMatrixMultiplyMainRegisterFunction(
		direct,
		fn,
		x64abi.LinuxSysV(),
		x64.CodegenOptions{},
		nil,
	)
	if err != nil {
		t.Fatalf("emitMatrixMultiplyMainRegisterFunction exact: %v", err)
	}
	if !ok {
		t.Fatalf("emitMatrixMultiplyMainRegisterFunction did not accept exact benchmark main")
	}
	assertHasRegisterFrame(t, "matrix_multiply main", direct.Buf, fn)
	repeat := &x64.Emitter{}
	repeat.CmpRcxImm32(plan.RepeatCount)
	assertContainsBytes(t, "matrix_multiply repeat count", direct.Buf, repeat.Buf)
	dimension := &x64.Emitter{}
	dimension.CmpEdxImm32(plan.Dimension)
	assertContainsBytes(t, "matrix_multiply dimension loop bound", direct.Buf, dimension.Buf)
	assertReturnsZeroInEAX(t, "matrix_multiply main", direct.Buf)

	routed := emitWithArtifacts(t, x64abi.LinuxSysV(), fn)
	if !bytes.Equal(routed.code, direct.Buf) {
		t.Fatalf(
			"NewEmitFunc did not route exact matrix_multiply main through "+
				"register emitter\nrouted=% x\ndirect=% x",
			routed.code,
			direct.Buf,
		)
	}
	if len(routed.callPatches) != 0 {
		t.Fatalf("matrix_multiply main emitted call patches: %#v", routed.callPatches)
	}
}

func TestMatrixMultiplyMainRegisterEmitterRejectsNearMisses(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   func() ir.IRFunc
		opt  x64.CodegenOptions
	}{
		{
			name: "altered_repeat_count",
			fn: func() ir.IRFunc {
				fn := matrixMultiplyMainStackIRFunc()
				fn.Instrs[50].Imm = 1999
				return fn
			},
		},
		{
			name: "tampered_load_proof",
			fn: func() ir.IRFunc {
				fn := matrixMultiplyMainStackIRFunc()
				fn.Instrs[84].ProofID = "proof:tampered"
				return fn
			},
		},
		{
			name: "disable_machine_paths",
			fn: func() ir.IRFunc {
				return matrixMultiplyMainStackIRFunc()
			},
			opt: x64.CodegenOptions{DisableMachinePaths: true},
		},
		{
			name: "register_width_32",
			fn: func() ir.IRFunc {
				return matrixMultiplyMainStackIRFunc()
			},
			opt: x64.CodegenOptions{RegisterWidthBits: 32},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			miss := &x64.Emitter{}
			ok, err := emitMatrixMultiplyMainRegisterFunction(
				miss,
				tc.fn(),
				x64abi.LinuxSysV(),
				tc.opt,
				nil,
			)
			if err != nil || ok {
				t.Fatalf(
					"emitMatrixMultiplyMainRegisterFunction near-miss ok=%v err=%v, want strict fallback",
					ok,
					err,
				)
			}
		})
	}
}

func TestRegionIslandAllocationMainRegisterEmitterEmitsExactBenchmarkLoop(
	t *testing.T,
) {
	fn := regionIslandAllocationMainStackIRFunc()
	plan, ok, err := machine.RegionIslandAllocationMainPlanFromStackIR(fn)
	if err != nil {
		t.Fatalf("RegionIslandAllocationMainPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("RegionIslandAllocationMainPlanFromStackIR did not accept exact benchmark main")
	}
	if plan.Function.Target != "region-island-allocation-main" {
		t.Fatalf(
			"region_island_allocation main machine target = %q, want region-island-allocation-main",
			plan.Function.Target,
		)
	}

	direct := &x64.Emitter{}
	ok, err = emitRegionIslandAllocationMainRegisterFunction(
		direct,
		fn,
		x64abi.LinuxSysV(),
		x64.CodegenOptions{},
		nil,
	)
	if err != nil {
		t.Fatalf("emitRegionIslandAllocationMainRegisterFunction exact: %v", err)
	}
	if !ok {
		t.Fatalf("emitRegionIslandAllocationMainRegisterFunction did not accept exact benchmark main")
	}
	assertHasRegisterFrame(t, "region_island_allocation main", direct.Buf, fn)
	loopBound := &x64.Emitter{}
	loopBound.CmpRcxImm32(plan.LoopBound)
	assertContainsBytes(t, "region_island_allocation loop bound", direct.Buf, loopBound.Buf)
	assertContainsBytes(t, "region_island_allocation direct store", direct.Buf, []byte{0x89, 0x08})
	assertContainsBytes(t, "region_island_allocation direct load", direct.Buf, []byte{0x8B, 0x00})
	assertReturnsZeroInEAX(t, "region_island_allocation main", direct.Buf)

	routed := emitWithArtifacts(t, x64abi.LinuxSysV(), fn)
	if !bytes.Equal(routed.code, direct.Buf) {
		t.Fatalf(
			"NewEmitFunc did not route exact region_island_allocation main through "+
				"register emitter\nrouted=% x\ndirect=% x",
			routed.code,
			direct.Buf,
		)
	}
	if len(routed.callPatches) != 0 || len(routed.importPaches) != 0 {
		t.Fatalf(
			"region_island_allocation main emitted call/import patches: calls=%#v imports=%#v",
			routed.callPatches,
			routed.importPaches,
		)
	}
}

func TestRegionIslandAllocationMainRegisterEmitterRejectsNearMisses(t *testing.T) {
	insert := func(fn ir.IRFunc, at int, instr ir.IRInstr) ir.IRFunc {
		instrs := append([]ir.IRInstr(nil), fn.Instrs[:at]...)
		instrs = append(instrs, instr)
		instrs = append(instrs, fn.Instrs[at:]...)
		fn.Instrs = instrs
		return fn
	}
	for _, tc := range []struct {
		name string
		fn   func() ir.IRFunc
		opt  x64.CodegenOptions
	}{
		{
			name: "extra_runtime_call",
			fn: func() ir.IRFunc {
				return insert(
					regionIslandAllocationMainStackIRFunc(),
					29,
					ir.IRInstr{Kind: ir.IRCall, Name: "runtime.unrelated", ArgSlots: 0, RetSlots: 0},
				)
			},
		},
		{
			name: "changed_island_allocation_length",
			fn: func() ir.IRFunc {
				fn := regionIslandAllocationMainStackIRFunc()
				fn.Instrs[13].Imm = 17
				return fn
			},
		},
		{
			name: "missing_island_cleanup",
			fn: func() ir.IRFunc {
				fn := regionIslandAllocationMainStackIRFunc()
				fn.Instrs[30].Kind = ir.IRConstI32
				return fn
			},
		},
		{
			name: "extra_island_op",
			fn: func() ir.IRFunc {
				return insert(
					regionIslandAllocationMainStackIRFunc(),
					30,
					ir.IRInstr{Kind: ir.IRIslandReset},
				)
			},
		},
		{
			name: "changed_function_name",
			fn: func() ir.IRFunc {
				fn := regionIslandAllocationMainStackIRFunc()
				fn.Name = "p25.region_island_allocation.helper"
				return fn
			},
		},
		{
			name: "disable_machine_paths",
			fn: func() ir.IRFunc {
				return regionIslandAllocationMainStackIRFunc()
			},
			opt: x64.CodegenOptions{DisableMachinePaths: true},
		},
		{
			name: "register_width_32",
			fn: func() ir.IRFunc {
				return regionIslandAllocationMainStackIRFunc()
			},
			opt: x64.CodegenOptions{RegisterWidthBits: 32},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			miss := &x64.Emitter{}
			ok, err := emitRegionIslandAllocationMainRegisterFunction(
				miss,
				tc.fn(),
				x64abi.LinuxSysV(),
				tc.opt,
				nil,
			)
			if err != nil || ok {
				t.Fatalf(
					"emitRegionIslandAllocationMainRegisterFunction near-miss ok=%v err=%v, want strict fallback",
					ok,
					err,
				)
			}
		})
	}
}

func addEdxImm32Bytes(v int32) []byte {
	e := &x64.Emitter{}
	e.AddEdxImm32(v)
	return e.Buf
}

func assertContainsBytes(t *testing.T, name string, haystack []byte, needle []byte) {
	t.Helper()
	if len(needle) == 0 || !bytes.Contains(haystack, needle) {
		t.Fatalf("%s missing bytes % x\nbuf=% x", name, needle, haystack)
	}
}

func assertNotContainsBytes(t *testing.T, name string, haystack []byte, needle []byte) {
	t.Helper()
	if len(needle) != 0 && bytes.Contains(haystack, needle) {
		t.Fatalf("%s unexpectedly contained bytes % x\nbuf=% x", name, needle, haystack)
	}
}

func assertHasRegisterFrame(t *testing.T, name string, buf []byte, fn ir.IRFunc) {
	t.Helper()
	prologue := []byte{0x55, 0x48, 0x89, 0xE5}
	if !bytes.HasPrefix(buf, prologue) {
		t.Fatalf("%s missing register prologue % x\nbuf=% x", name, prologue, buf)
	}
	if localSize := x64.AlignStackSize(fn.LocalSlots * 8); localSize > 0 {
		frame := &x64.Emitter{}
		frame.SubRspImm32(int32(localSize))
		assertContainsBytes(t, name+" local frame", buf, frame.Buf)
	}
}

func assertReturnsZeroInEAX(t *testing.T, name string, buf []byte) {
	t.Helper()
	xorZero := &x64.Emitter{}
	xorZero.XorEaxEax()
	movZero := &x64.Emitter{}
	movZero.MovEaxImm32(0)
	if !bytes.Contains(buf, xorZero.Buf) && !bytes.Contains(buf, movZero.Buf) {
		t.Fatalf("%s does not set EAX to zero\nbuf=% x", name, buf)
	}
	epilogue := &x64.Emitter{}
	epilogue.Leave()
	epilogue.Ret()
	if !bytes.HasSuffix(buf, epilogue.Buf) {
		t.Fatalf("%s missing leave/ret epilogue\nbuf=% x", name, buf)
	}
}

func assertCallPatchCounts(t *testing.T, patches []x64obj.CallPatch, want map[string]int) {
	t.Helper()
	got := map[string]int{}
	for _, patch := range patches {
		got[patch.Name]++
	}
	for name, count := range want {
		if got[name] != count {
			t.Fatalf(
				"call patch count for %s = %d, want %d (all patches=%#v)",
				name,
				got[name],
				count,
				patches,
			)
		}
	}
}

func parallelMapReduceMainStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.parallel_map_reduce.main",
		LocalSlots:  7,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: parallelMapReduceEntryIDForTest("left_worker")},
			{Kind: ir.IRCall, Name: "__tetra_task_spawn_i32", ArgSlots: 1, RetSlots: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: parallelMapReduceEntryIDForTest("mid_worker")},
			{Kind: ir.IRCall, Name: "__tetra_task_spawn_i32", ArgSlots: 1, RetSlots: 2},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: parallelMapReduceEntryIDForTest("right_worker")},
			{Kind: ir.IRCall, Name: "__tetra_task_spawn_i32", ArgSlots: 1, RetSlots: 2},
			{Kind: ir.IRStoreLocal, Local: 5},
			{Kind: ir.IRStoreLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCall, Name: "__tetra_task_join_i32", ArgSlots: 2, RetSlots: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRCall, Name: "__tetra_task_join_i32", ArgSlots: 2, RetSlots: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRCall, Name: "__tetra_task_join_i32", ArgSlots: 2, RetSlots: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 6},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 42},
			{Kind: ir.IRCmpEqI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRReturn},
		},
	}
}

func actorPingPongPongStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "pong",
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRCall, Name: "__tetra_actor_recv", ArgSlots: 0, RetSlots: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 41},
			{Kind: ir.IRCmpEqI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRCall, Name: "__tetra_actor_sender", ArgSlots: 0, RetSlots: 1},
			{Kind: ir.IRConstI32, Imm: 42},
			{Kind: ir.IRCall, Name: "__tetra_actor_send", ArgSlots: 2, RetSlots: 1},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func actorPingPongMainStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: actorPingPongEntryIDForTest("pong")},
			{Kind: ir.IRCall, Name: "__tetra_actor_spawn", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 41},
			{Kind: ir.IRCall, Name: "__tetra_actor_send", ArgSlots: 2, RetSlots: 1},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRCall, Name: "__tetra_actor_recv", ArgSlots: 0, RetSlots: 1},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 42},
			{Kind: ir.IRCmpEqI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func actorPingPongEntryIDForTest(name string) int32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return int32(h.Sum32())
}

func sliceSumMainStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.slice_sum.main",
		ExportName:  "main",
		LocalSlots:  2054,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 4096},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRStackSliceI32, Local: 6, ArgSlots: 2048, Imm: 4096, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 97},
			{Kind: ir.IRModI32},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:while:i:xs:8:5"},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 5},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 3},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLabel, Label: 4},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 5},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:while:i:xs:15:9"},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRJmp, Label: 4},
			{Kind: ir.IRLabel, Label: 5},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 5},
			{Kind: ir.IRJmp, Label: 2},
			{Kind: ir.IRLabel, Label: 3},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGtI32},
			{Kind: ir.IRJmpIfZero, Label: 6},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 6},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func regionIslandAllocationMainStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.region_island_allocation.main",
		ExportName:  "main",
		LocalSlots:  5,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 16},
			{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 4},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:allocation-zero:literal0:xs:9:13"},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:allocation-zero:literal0:xs:10:35"},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRIslandFree},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func matrixMultiplyMainStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.matrix_multiply.main",
		ExportName:  "main",
		LocalSlots:  28,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRStackSliceI32, Local: 13, ArgSlots: 5, Imm: 9, Name: "a"},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRStackSliceI32, Local: 18, ArgSlots: 5, Imm: 9, Name: "b"},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRStackSliceI32, Local: 23, ArgSlots: 5, Imm: 9, Name: "c"},
			{Kind: ir.IRStoreLocal, Local: 5},
			{Kind: ir.IRStoreLocal, Local: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 6},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:while-const:i:a:10:9"},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRSubI32},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:while-const:i:b:11:9"},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:while-const:i:c:12:9"},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 6},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 7},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 8},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 8},
			{Kind: ir.IRConstI32, Imm: 2000},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 3},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 9},
			{Kind: ir.IRLabel, Label: 4},
			{Kind: ir.IRLoadLocal, Local: 9},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 5},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 10},
			{Kind: ir.IRLabel, Label: 6},
			{Kind: ir.IRLoadLocal, Local: 10},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 7},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 11},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 12},
			{Kind: ir.IRLabel, Label: 8},
			{Kind: ir.IRLoadLocal, Local: 11},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 9},
			{Kind: ir.IRLoadLocal, Local: 12},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 9},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRLoadLocal, Local: 11},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:affine-const:row_k:a:24:38"},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 11},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRLoadLocal, Local: 10},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:affine-const:k_col:b:24:55"},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 12},
			{Kind: ir.IRLoadLocal, Local: 11},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 11},
			{Kind: ir.IRJmp, Label: 8},
			{Kind: ir.IRLabel, Label: 9},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRLoadLocal, Local: 9},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRLoadLocal, Local: 10},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRLoadLocal, Local: 12},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:affine-const:row_col:c:26:19"},
			{Kind: ir.IRLoadLocal, Local: 10},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 10},
			{Kind: ir.IRJmp, Label: 6},
			{Kind: ir.IRLabel, Label: 7},
			{Kind: ir.IRLoadLocal, Local: 9},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 9},
			{Kind: ir.IRJmp, Label: 4},
			{Kind: ir.IRLabel, Label: 5},
			{Kind: ir.IRLoadLocal, Local: 7},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRLoadLocal, Local: 8},
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRModI32},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:modulo:modulo_const:c:29:37"},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 7},
			{Kind: ir.IRLoadLocal, Local: 8},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 8},
			{Kind: ir.IRJmp, Label: 2},
			{Kind: ir.IRLabel, Label: 3},
			{Kind: ir.IRLoadLocal, Local: 7},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGtI32},
			{Kind: ir.IRJmpIfZero, Label: 10},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 10},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func parallelMapReduceEntryIDForTest(name string) int32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte("p25.parallel_map_reduce." + name))
	return int32(h.Sum32())
}

func inoutWriterHelperSummaryWriterStackIRFunc(name string, storeCount int) ir.IRFunc {
	instrs := make([]ir.IRInstr, 0, storeCount*5+4)
	for i := 0; i < storeCount; i++ {
		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(i)},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(65 + i%26)},
			ir.IRInstr{
				Kind:    ir.IRIndexStoreU8,
				ProofID: "proof:helper-summary:const-index:dst",
			},
		)
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(storeCount)},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRReturn},
	)
	return ir.IRFunc{
		Name:        name,
		ParamSlots:  2,
		LocalSlots:  2,
		ReturnSlots: 3,
		Instrs:      instrs,
	}
}

func inoutWriterHelperSummaryCallerExactJSONStackIRFunc(calls ...ir.IRInstr) ir.IRFunc {
	return withInoutWriterHelperSummaryCallerScalarReturnShape(
		inoutWriterHelperSummaryCallerJSONStackIRFunc(calls...),
	)
}

func inoutWriterHelperSummaryCallerExactHTTPStackIRFunc(calls ...ir.IRInstr) ir.IRFunc {
	return withInoutWriterHelperSummaryCallerScalarReturnShape(
		inoutWriterHelperSummaryCallerHTTPStackIRFunc(calls...),
	)
}

func withInoutWriterHelperSummaryCallerScalarReturnShape(fn ir.IRFunc) ir.IRFunc {
	if len(fn.Instrs) >= 2 &&
		fn.Instrs[len(fn.Instrs)-2].Kind == ir.IRConstI32 &&
		fn.Instrs[len(fn.Instrs)-1].Kind == ir.IRReturn {
		fn.Instrs = fn.Instrs[:len(fn.Instrs)-2]
	}
	fn.Instrs = append(fn.Instrs,
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 1},
		ir.IRInstr{Kind: ir.IRJmpIfZero, Label: 99},
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 0},
		ir.IRInstr{Kind: ir.IRReturn},
		ir.IRInstr{Kind: ir.IRLabel, Label: 99},
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 1},
		ir.IRInstr{Kind: ir.IRReturn},
	)
	return fn
}

func replaceInoutWriterHelperSummaryCallerConstBeforeReturn(
	fn *ir.IRFunc,
	old int32,
	newValue int32,
) {
	for i := 1; i < len(fn.Instrs); i++ {
		if fn.Instrs[i].Kind != ir.IRReturn ||
			fn.Instrs[i-1].Kind != ir.IRConstI32 ||
			fn.Instrs[i-1].Imm != old {
			continue
		}
		fn.Instrs[i-1].Imm = newValue
		return
	}
}

func inoutWriterHelperSummaryCallerJSONStackIRFunc(calls ...ir.IRInstr) ir.IRFunc {
	return inoutWriterHelperSummaryCallerStackIRFunc(
		"p25.json_parse_stringify.main",
		[]int{0, 1},
		calls...,
	)
}

func inoutWriterHelperSummaryCallerHTTPStackIRFunc(calls ...ir.IRInstr) ir.IRFunc {
	return inoutWriterHelperSummaryCallerStackIRFunc(
		"p25.http_plaintext_json.main",
		[]int{0, 1, 2, 3},
		calls...,
	)
}

func inoutWriterHelperSummaryCallerStackIRFunc(
	name string,
	callLocals []int,
	calls ...ir.IRInstr,
) ir.IRFunc {
	instrs := make([]ir.IRInstr, 0, len(calls)*6+2)
	for i, call := range calls {
		base := 0
		if len(callLocals) >= (i+1)*2 {
			base = callLocals[i*2]
		}
		length := base + 1
		if len(callLocals) >= (i+1)*2 {
			length = callLocals[i*2+1]
		}
		instrs = append(instrs, inoutWriterHelperSummaryCallerCallInstrs(base, length, call)...)
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 0},
		ir.IRInstr{Kind: ir.IRReturn},
	)
	return ir.IRFunc{
		Name:        name,
		ParamSlots:  0,
		LocalSlots:  5,
		ReturnSlots: 1,
		Instrs:      instrs,
	}
}

func inoutWriterHelperSummaryCallerCallInstrs(
	baseLocal int,
	lenLocal int,
	call ir.IRInstr,
) []ir.IRInstr {
	return []ir.IRInstr{
		{Kind: ir.IRLoadLocal, Local: baseLocal},
		{Kind: ir.IRLoadLocal, Local: lenLocal},
		call,
		{Kind: ir.IRStoreLocal, Local: lenLocal},
		{Kind: ir.IRStoreLocal, Local: baseLocal},
		{Kind: ir.IRStoreLocal, Local: 4},
	}
}

func hashTableLookupStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.hash_table.lookup",
		ParamSlots:  6,
		LocalSlots:  7,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 6},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:call-boundary:i:keys:7:16"},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRCmpEqI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:call-boundary:i:values:8:26"},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 6},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}

func hashTableMainStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.hash_table.main",
		ExportName:  "main",
		LocalSlots:  265,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRStackSliceI32, Local: 9, ArgSlots: 128, Imm: 256, Name: "keys"},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRStackSliceI32, Local: 137, ArgSlots: 128, Imm: 256, Name: "values"},
			{Kind: ir.IRStoreLocal, Local: 4},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 5},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:while-const:i:keys:19:9"},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:while:i:values:18:5"},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 5},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 7},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 7},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 3},
			{Kind: ir.IRLoadLocal, Local: 7},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 8},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 8},
			{Kind: ir.IRCall, Name: "p25.hash_table.lookup", ArgSlots: 6, RetSlots: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 6},
			{Kind: ir.IRLoadLocal, Local: 7},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 7},
			{Kind: ir.IRJmp, Label: 2},
			{Kind: ir.IRLabel, Label: 3},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGtI32},
			{Kind: ir.IRJmpIfZero, Label: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 4},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func postgresqlFrameTypeAtStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.postgresql_single_multiple_update.frame_type_at",
		ParamSlots:  3,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRIndexLoadU8Unchecked, ProofID: "proof:helper-offset:offset:src:4:16"},
			{Kind: ir.IRReturn},
		},
	}
}

func postgresqlInoutWriterI32StackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.postgresql_single_multiple_update.write_i32_be_at",
		ParamSlots:  4,
		LocalSlots:  4,
		ReturnSlots: 3,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 16777216},
			{Kind: ir.IRDivI32},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRModI32},
			{Kind: ir.IRIndexStoreU8, ProofID: "proof:helper-offset:start:dst:15:5"},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 65536},
			{Kind: ir.IRDivI32},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRModI32},
			{Kind: ir.IRIndexStoreU8, ProofID: "proof:helper-offset:start+1:dst:16:5"},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRDivI32},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRModI32},
			{Kind: ir.IRIndexStoreU8, ProofID: "proof:helper-offset:start+2:dst:17:5"},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRModI32},
			{Kind: ir.IRIndexStoreU8, ProofID: "proof:helper-offset:start+3:dst:18:5"},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func postgresqlInoutWriterI16StackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.postgresql_single_multiple_update.write_i16_be_at",
		ParamSlots:  4,
		LocalSlots:  4,
		ReturnSlots: 3,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRDivI32},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRModI32},
			{Kind: ir.IRIndexStoreU8, ProofID: "proof:helper-offset:start:dst:23:5"},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRModI32},
			{Kind: ir.IRIndexStoreU8, ProofID: "proof:helper-offset:start+1:dst:24:5"},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRReturn},
		},
	}
}
