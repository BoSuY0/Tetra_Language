package linux_x86

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
)

func TestCodegenObjectLinuxX86PatchesConditionalBranches(t *testing.T) {
	obj, err := CodegenObjectLinuxX86([]ir.IRFunc{{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRConstI32, Imm: 99},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRReturn},
		},
	}})
	if err != nil {
		t.Fatalf("codegen branch object: %v", err)
	}
	jz := []byte{0x0F, 0x84}
	at := bytes.Index(obj.Code, jz)
	if at < 0 {
		t.Fatalf("conditional branch did not emit jz rel32: % x", obj.Code)
	}
	if disp := int32(binary.LittleEndian.Uint32(obj.Code[at+2 : at+6])); disp <= 0 {
		t.Fatalf("conditional branch displacement = %d, want forward jump", disp)
	}
}

func TestCodegenObjectLinuxX86PatchesUnconditionalBranches(t *testing.T) {
	obj, err := CodegenObjectLinuxX86([]ir.IRFunc{{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRConstI32, Imm: 99},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRReturn},
		},
	}})
	if err != nil {
		t.Fatalf("codegen jmp object: %v", err)
	}
	at := bytes.IndexByte(obj.Code, 0xE9)
	if at < 0 {
		t.Fatalf("unconditional branch did not emit jmp rel32: % x", obj.Code)
	}
	if disp := int32(binary.LittleEndian.Uint32(obj.Code[at+1 : at+5])); disp <= 0 {
		t.Fatalf("unconditional branch displacement = %d, want forward jump", disp)
	}
}

func TestCodegenObjectLinuxX86RejectsInvalidBranchLabels(t *testing.T) {
	_, err := CodegenObjectLinuxX86([]ir.IRFunc{{
		Name: "main",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRJmp, Label: -1},
			{Kind: ir.IRReturn},
		},
	}})
	if err == nil || !strings.Contains(err.Error(), "negative label -1") {
		t.Fatalf("negative label error = %v", err)
	}

	_, err = CodegenObjectLinuxX86([]ir.IRFunc{{
		Name: "main",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRJmp, Label: 99},
			{Kind: ir.IRReturn},
		},
	}})
	if err == nil || !strings.Contains(err.Error(), "missing label 99") {
		t.Fatalf("missing label error = %v", err)
	}
}

func TestCodegenObjectLinuxX86EmitsNoArgCallReloc(t *testing.T) {
	obj, err := CodegenObjectLinuxX86([]ir.IRFunc{
		{
			Name:        "callee",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRCall, Name: "callee", ArgSlots: 0, RetSlots: 1},
				{Kind: ir.IRReturn},
			},
		},
	})
	if err != nil {
		t.Fatalf("codegen call object: %v", err)
	}
	if !bytes.Contains(obj.Code, []byte{0xE8, 0, 0, 0, 0}) {
		t.Fatalf("call did not emit rel32 placeholder: % x", obj.Code)
	}
	if len(obj.Relocs) != 1 {
		t.Fatalf("relocs len = %d, want 1: %#v", len(obj.Relocs), obj.Relocs)
	}
	if obj.Relocs[0].Kind != tobj.RelocCallRel32 || obj.Relocs[0].Name != "callee" {
		t.Fatalf("call reloc = %#v, want call reloc to callee", obj.Relocs[0])
	}
}

func TestCodegenObjectLinuxX86EmitsThreeSlotInternalReturn(t *testing.T) {
	obj, err := CodegenObjectLinuxX86([]ir.IRFunc{
		{
			Name:        "callee",
			ReturnSlots: 3,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRConstI32, Imm: 3},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "main",
			ReturnSlots: 3,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRCall, Name: "callee", ArgSlots: 0, RetSlots: 3},
				{Kind: ir.IRReturn},
			},
		},
	})
	if err != nil {
		t.Fatalf("codegen three-slot return object: %v", err)
	}
	for _, want := range [][]byte{
		{0x59, 0x5A, 0x58, 0xC9, 0xC3}, // pop ecx; pop edx; pop eax; leave; ret
		{0x50, 0x52, 0x51},             // push eax; push edx; push ecx
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("three-slot return code missing % x in:\n% x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX86EmitsCallerCleanedStackArguments(t *testing.T) {
	obj, err := CodegenObjectLinuxX86([]ir.IRFunc{
		{
			Name:        "callee",
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 40},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRCall, Name: "callee", ArgSlots: 2, RetSlots: 1},
				{Kind: ir.IRReturn},
			},
		},
	})
	if err != nil {
		t.Fatalf("codegen call with stack args: %v", err)
	}
	if len(obj.Relocs) != 1 || obj.Relocs[0].Kind != tobj.RelocCallRel32 || obj.Relocs[0].Name != "callee" {
		t.Fatalf("call relocs = %#v, want one call reloc to callee", obj.Relocs)
	}
	for _, want := range [][]byte{
		{0x81, 0xEC, 0x08, 0, 0, 0},       // sub esp,8 call argument area
		{0x8B, 0x84, 0x24, 0x0C, 0, 0, 0}, // mov eax,[esp+12] original arg0
		{0x89, 0x84, 0x24, 0, 0, 0, 0},    // mov [esp],eax
		{0x8B, 0x84, 0x24, 0x08, 0, 0, 0}, // mov eax,[esp+8] original arg1
		{0x89, 0x84, 0x24, 0x04, 0, 0, 0}, // mov [esp+4],eax
		{0x81, 0xC4, 0x10, 0, 0, 0},       // add esp,16 caller cleanup
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("x86 call code missing % x in:\n% x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX86EmitsByteWordAtomicLogicalFetchCASLoops(t *testing.T) {
	obj, err := CodegenObjectLinuxX86([]ir.IRFunc{{
		Name:        "atomic_logical",
		LocalSlots:  1,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0x1000},
			{Kind: ir.IRConstI32, Imm: 0x0f},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRAtomicFetchAndI8},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0x1000},
			{Kind: ir.IRConstI32, Imm: 0xf0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRAtomicFetchOrI8},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0x1000},
			{Kind: ir.IRConstI32, Imm: 0xff},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRAtomicFetchXorI8},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0x1000},
			{Kind: ir.IRConstI32, Imm: 0x00ff},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRAtomicFetchAndI16},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0x1000},
			{Kind: ir.IRConstI32, Imm: 0xff00},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRAtomicFetchOrI16},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0x1000},
			{Kind: ir.IRConstI32, Imm: 0xffff},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRAtomicFetchXorI16},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}})
	if err != nil {
		t.Fatalf("codegen byte/word atomic logical object: %v", err)
	}
	for _, want := range [][]byte{
		{0x20, 0xCA, 0xF0, 0x0F, 0xB0, 0x17},             // and dl,cl; lock cmpxchg byte [edi],dl
		{0x08, 0xCA, 0xF0, 0x0F, 0xB0, 0x17},             // or dl,cl; lock cmpxchg byte [edi],dl
		{0x30, 0xCA, 0xF0, 0x0F, 0xB0, 0x17},             // xor dl,cl; lock cmpxchg byte [edi],dl
		{0x66, 0x21, 0xCA, 0x66, 0xF0, 0x0F, 0xB1, 0x17}, // and dx,cx; lock cmpxchg word [edi],dx
		{0x66, 0x09, 0xCA, 0x66, 0xF0, 0x0F, 0xB1, 0x17}, // or dx,cx; lock cmpxchg word [edi],dx
		{0x66, 0x31, 0xCA, 0x66, 0xF0, 0x0F, 0xB1, 0x17}, // xor dx,cx; lock cmpxchg word [edi],dx
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("x86 atomic logical fetch code missing % x in:\n% x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX86NarrowAtomicStoresReturnStoredValue(t *testing.T) {
	obj, err := CodegenObjectLinuxX86([]ir.IRFunc{{
		Name:        "atomic_narrow_store",
		LocalSlots:  1,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0x1000},
			{Kind: ir.IRConstI32, Imm: 0x7f},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRAtomicStoreI8},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0x1000},
			{Kind: ir.IRConstI32, Imm: 0x7fff},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRAtomicStoreI16},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0x1000},
			{Kind: ir.IRConstI32, Imm: 0x33},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRAtomicExchangeI8},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0x1000},
			{Kind: ir.IRConstI32, Imm: 0x3333},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRAtomicExchangeI16},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}})
	if err != nil {
		t.Fatalf("codegen narrow atomic store object: %v", err)
	}
	for _, want := range [][]byte{
		{0x5A, 0x59, 0x5F, 0x89, 0xC8, 0x0F, 0xB6, 0xC0, 0x86, 0x0F, 0x50},       // store u8: return zero-extended stored value
		{0x5A, 0x59, 0x5F, 0x89, 0xC8, 0x0F, 0xB7, 0xC0, 0x66, 0x87, 0x0F, 0x50}, // store u16: return zero-extended stored value
		{0x5A, 0x59, 0x5F, 0x86, 0x0F, 0x0F, 0xB6, 0xC0, 0x50},                   // exchange u8: return old value
		{0x5A, 0x59, 0x5F, 0x66, 0x87, 0x0F, 0x0F, 0xB7, 0xC0, 0x50},             // exchange u16: return old value
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("narrow atomic store/exchange code missing % x in:\n% x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX86EmitsAbsoluteGlobalRelocs(t *testing.T) {
	obj, err := CodegenObjectLinuxX86WithDataPrefix([]ir.IRFunc{{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 42},
			{Kind: ir.IRStoreGlobal, Local: 1},
			{Kind: ir.IRLoadGlobal, Local: 0},
			{Kind: ir.IRReturn},
		},
	}}, [][]byte{
		{7, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
	})
	if err != nil {
		t.Fatalf("codegen global object: %v", err)
	}
	if got, want := len(obj.Data), 16; got != want {
		t.Fatalf("data len = %d, want %d", got, want)
	}
	if got := binary.LittleEndian.Uint32(obj.Data[:4]); got != 7 {
		t.Fatalf("global data[0] = %d, want 7", got)
	}
	if len(obj.Relocs) != 2 {
		t.Fatalf("relocs len = %d, want 2: %#v", len(obj.Relocs), obj.Relocs)
	}
	storeReloc := obj.Relocs[0]
	if storeReloc.Kind != tobj.RelocDataAbs32 || storeReloc.Name != "" || storeReloc.Addend != 8 {
		t.Fatalf("store reloc = %#v, want absolute data reloc to second slot", storeReloc)
	}
	loadReloc := obj.Relocs[1]
	if loadReloc.Kind != tobj.RelocDataAbs32 || loadReloc.Name != "" || loadReloc.Addend != 0 {
		t.Fatalf("load reloc = %#v, want absolute data reloc to first slot", loadReloc)
	}
	storeAt := int(storeReloc.At)
	loadAt := int(loadReloc.At)
	if storeAt <= 0 || storeAt+4 > len(obj.Code) || obj.Code[storeAt-1] != 0xA3 {
		t.Fatalf("store global did not emit mov moffs32,eax before reloc at %d: % x", storeAt, obj.Code)
	}
	if loadAt <= 0 || loadAt+4 > len(obj.Code) || obj.Code[loadAt-1] != 0xA1 {
		t.Fatalf("load global did not emit mov eax,moffs32 before reloc at %d: % x", loadAt, obj.Code)
	}
}

func TestCodegenObjectLinuxX86EmitsAbsoluteFunctionAddressReloc(t *testing.T) {
	obj, err := CodegenObjectLinuxX86([]ir.IRFunc{
		{
			Name:        "callback_target",
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
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRSymAddr, Name: "callback_target"},
				{Kind: ir.IRReturn},
			},
		},
	})
	if err != nil {
		t.Fatalf("codegen function address object: %v", err)
	}
	if len(obj.Relocs) != 1 {
		t.Fatalf("relocs len = %d, want 1: %#v", len(obj.Relocs), obj.Relocs)
	}
	reloc := obj.Relocs[0]
	if reloc.Kind != tobj.RelocFuncAddrAbs32 || reloc.Name != "callback_target" || reloc.Addend != 0 {
		t.Fatalf("function address reloc = %#v, want absolute function address reloc", reloc)
	}
	at := int(reloc.At)
	if at <= 0 || at+4 > len(obj.Code) || obj.Code[at-1] != 0xB8 {
		t.Fatalf("IRSymAddr did not emit mov eax, imm32 before reloc at %d: % x", at, obj.Code)
	}
	if !bytes.Contains(obj.Code[at+4:], []byte{0x50}) {
		t.Fatalf("IRSymAddr did not push materialized function address: % x", obj.Code)
	}
}

func TestCodegenObjectLinuxX86AllocBytesUsesMmap2FailureGuard(t *testing.T) {
	obj, err := CodegenObjectLinuxX86([]ir.IRFunc{{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 16},
			{Kind: ir.IRAllocBytes},
			{Kind: ir.IRReturn},
		},
	}})
	if err != nil {
		t.Fatalf("codegen alloc_bytes object: %v", err)
	}
	for _, want := range [][]byte{
		{0x83, 0xF9, 0x01, 0x0F, 0x8D},                   // cmp ecx,1; jge valid-size
		{0xB8, 0xC0, 0x00, 0x00, 0x00, 0xCD, 0x80},       // mov eax,192; int 0x80 (mmap2)
		{0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83},       // cmp eax,-4095; jae failure
		{0x89, 0x08, 0x83, 0xC0, 0x08, 0x50},             // metadata size, payload pointer
		{0xBB, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x01, 0x00}, // exit(2) diagnostic path
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("alloc_bytes code missing % x in:\n% x", want, obj.Code)
		}
	}
	if bytes.Contains(obj.Code, []byte{0x58, 0xB8, 0, 0, 0, 0, 0x50}) {
		t.Fatalf("alloc_bytes still contains fake pop/mov-zero/push sequence: % x", obj.Code)
	}
}

func TestCodegenObjectLinuxX86EmitsSliceMakeLoadStore(t *testing.T) {
	obj, err := CodegenObjectLinuxX86([]ir.IRFunc{
		{
			Name:        "make_i32",
			ReturnSlots: 2,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 3},
				{Kind: ir.IRMakeSliceI32},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "load_i32",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 4096},
				{Kind: ir.IRConstI32, Imm: 3},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRIndexLoadI32},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "store_i32",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 4096},
				{Kind: ir.IRConstI32, Imm: 3},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRConstI32, Imm: 30},
				{Kind: ir.IRIndexStoreI32},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
		},
	})
	if err != nil {
		t.Fatalf("codegen slice object: %v", err)
	}
	for _, want := range [][]byte{
		{0xC1, 0xE1, 0x02},                         // shl ecx,2 for i32 byte length
		{0xB8, 0xC0, 0x00, 0x00, 0x00, 0xCD, 0x80}, // mmap2 allocation
		{0x39, 0xCA, 0x0F, 0x83},                   // cmp edx,ecx; jae bounds-failure
		{0xC1, 0xE2, 0x02, 0x01, 0xD0, 0x8B, 0x00}, // scale index, add ptr, load i32
		{0xC1, 0xE2, 0x02, 0x01, 0xD0, 0x89, 0x18}, // scale index, add ptr, store i32
		{0xBB, 0x01, 0x00, 0x00, 0x00, 0xB8, 0x01}, // exit(1) bounds path
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("slice code missing % x in:\n% x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX86MakeSliceZeroLengthBypassesMmap(t *testing.T) {
	obj, err := CodegenObjectLinuxX86([]ir.IRFunc{{
		Name:        "make_i32_empty",
		ReturnSlots: 2,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRMakeSliceI32},
			{Kind: ir.IRReturn},
		},
	}})
	if err != nil {
		t.Fatalf("codegen zero-length slice object: %v", err)
	}
	for _, want := range [][]byte{
		{0x83, 0xF9, 0x00, 0x0F, 0x84},             // cmp ecx,0; jz empty-slice
		{0xB8, 0x00, 0x00, 0x00, 0x00, 0x50, 0x51}, // mov eax,0; push ptr; push len
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("zero-length slice code missing % x in:\n% x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX86EmitsRawMemoryOps(t *testing.T) {
	obj, err := CodegenObjectLinuxX86([]ir.IRFunc{{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 16},
			{Kind: ir.IRAllocBytes},
			{Kind: ir.IRConstI32, Imm: 42},
			{Kind: ir.IRCapMem},
			{Kind: ir.IRMemWriteI32},
			{Kind: ir.IRCapMem},
			{Kind: ir.IRMemReadI32},
			{Kind: ir.IRReturn},
		},
	}})
	if err != nil {
		t.Fatalf("codegen raw memory object: %v", err)
	}
	for _, want := range [][]byte{
		{0xBA, 0x00, 0x00, 0x00, 0x00},             // mov edx,0 for base raw access
		{0x83, 0xFA, 0x00, 0x0F, 0x8D},             // cmp edx,0; jge
		{0x81, 0xE7, 0x00, 0xF0, 0xFF, 0xFF},       // and edi,-4096
		{0x8B, 0x0F, 0x83, 0xC7, 0x08},             // load allocation size and advance payload base
		{0x01, 0xC2, 0x83, 0xC2, 0x04, 0x39, 0xCA}, // offset+width <= allocation size
		{0x89, 0x18, 0x53},                         // store i32 and return stored value
		{0x8B, 0x00, 0x50},                         // load i32 and push result
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("raw memory code missing % x in:\n% x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX86EmitsPtrAddAndOffsetRawMemoryOps(t *testing.T) {
	obj, err := CodegenObjectLinuxX86([]ir.IRFunc{{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 4096},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRCapMem},
			{Kind: ir.IRPtrAdd},
			{Kind: ir.IRConstI32, Imm: 4096},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRCapMem},
			{Kind: ir.IRMemWriteU8Offset},
			{Kind: ir.IRConstI32, Imm: 4096},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRCapMem},
			{Kind: ir.IRMemReadU8Offset},
			{Kind: ir.IRReturn},
		},
	}})
	if err != nil {
		t.Fatalf("codegen raw offset memory object: %v", err)
	}
	for _, want := range [][]byte{
		{0x5A, 0x58}, // pop capability, pop offset/base operands around ptr_add
		{0x01, 0xC2, 0x83, 0xC2, 0x01, 0x39, 0xCA}, // offset+width <= allocation size for byte access
		{0x83, 0xEA, 0x01, 0x89, 0xF8, 0x01, 0xD0}, // restore offset, payload base, add offset
		{0x88, 0x18, 0x53},                         // store u8 and return stored value
		{0x0F, 0xB6, 0x00, 0x50},                   // load u8 and zero-extend result
		{0xBB, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x01}, // exit(2) raw bounds path
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("raw offset memory code missing % x in:\n% x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX86EmitsStringLiteralAndWrite(t *testing.T) {
	obj, err := CodegenObjectLinuxX86([]ir.IRFunc{{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRStrLit, Str: []byte("hi\n")},
			{Kind: ir.IRWrite},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}})
	if err != nil {
		t.Fatalf("codegen string/write object: %v", err)
	}
	if string(obj.Data) != "hi\n" {
		t.Fatalf("data = %q, want string literal", string(obj.Data))
	}
	if len(obj.Relocs) != 1 {
		t.Fatalf("relocs len = %d, want 1: %#v", len(obj.Relocs), obj.Relocs)
	}
	reloc := obj.Relocs[0]
	if reloc.Kind != tobj.RelocDataAbs32 || reloc.Name != "" || reloc.Addend != 0 {
		t.Fatalf("string reloc = %#v, want absolute data reloc", reloc)
	}
	at := int(reloc.At)
	if at <= 0 || at+4 > len(obj.Code) || obj.Code[at-1] != 0xB8 {
		t.Fatalf("IRStrLit did not emit mov eax, imm32 before reloc at %d: % x", at, obj.Code)
	}
	for _, want := range [][]byte{
		{0x50, 0xB8, 0x03, 0x00, 0x00, 0x00, 0x50},       // push data pointer; mov len; push len
		{0x5A, 0x59, 0xBB, 0x01, 0x00, 0x00, 0x00},       // pop len/pointer; fd=stdout
		{0xB8, 0x04, 0x00, 0x00, 0x00, 0xCD, 0x80},       // write syscall
		{0xB8, 0x00, 0x00, 0x00, 0x00, 0x50, 0x58, 0xC9}, // return 0
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("string/write code missing % x in:\n% x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX86EmitsIslandAllocationAndFree(t *testing.T) {
	obj, err := CodegenObjectLinuxX86([]ir.IRFunc{
		{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 64},
				{Kind: ir.IRIslandNew},
				{Kind: ir.IRIslandFree},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "make_slice",
			ReturnSlots: 2,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 4096},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRIslandMakeSliceU8},
				{Kind: ir.IRReturn},
			},
		},
	})
	if err != nil {
		t.Fatalf("codegen island object: %v", err)
	}
	for _, want := range [][]byte{
		{0x81, 0xC1, 0x10, 0x00, 0x00, 0x00},                         // add ecx,16 header bytes
		{0xB8, 0xC0, 0x00, 0x00, 0x00, 0xCD, 0x80},                   // mmap2
		{0xC7, 0x00, 0x10, 0x00, 0x00, 0x00},                         // [island+0] next offset
		{0x89, 0x48, 0x04, 0x89, 0x48, 0x08},                         // [island+4]/[island+8] capacity/map length
		{0x8B, 0x10, 0x8B, 0x58, 0x04},                               // load next/capacity
		{0x01, 0xCF, 0x39, 0xDF, 0x0F, 0x87},                         // next+bytes > capacity overflow
		{0x89, 0x38},                                                 // commit new bump offset
		{0x8B, 0x4B, 0x08, 0xB8, 0x5B, 0x00, 0x00, 0x00, 0xCD, 0x80}, // munmap
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("island code missing % x in:\n% x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX86DebugIslandsEmitDoubleFreeGuardAndProtect(t *testing.T) {
	obj, err := CodegenObjectLinuxX86WithOptionsAndDataPrefix([]ir.IRFunc{{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRIslandFree},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}}, nil, x64.CodegenOptions{IslandsDebug: true})
	if err != nil {
		t.Fatalf("codegen debug island object: %v", err)
	}
	for _, want := range [][]byte{
		{0x81, 0xC1, 0x00, 0x10, 0x00, 0x00},                                     // add ecx,4096 debug header bytes
		{0xC7, 0x00, 0x00, 0x10, 0x00, 0x00},                                     // [island+0] next offset = 4096
		{0xC7, 0x40, 0x0C, 0x00, 0x00, 0x00, 0x00},                               // freed marker clear
		{0x8B, 0x43, 0x0C, 0x85, 0xC0, 0x0F, 0x84},                               // load/test freed marker and branch if zero
		{0xBB, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80}, // exit(2)
		{0xC7, 0x43, 0x0C, 0x01, 0x00, 0x00, 0x00},                               // freed marker set
		{0x8B, 0x4B, 0x08, 0x81, 0xE9, 0x00, 0x10, 0x00, 0x00},                   // mprotect length -= 4096
		{0x81, 0xC3, 0x00, 0x10, 0x00, 0x00, 0xBA, 0x00, 0x00, 0x00, 0x00},       // payload base, PROT_NONE
		{0xB8, 0x7D, 0x00, 0x00, 0x00, 0xCD, 0x80},                               // mprotect syscall
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("debug island code missing % x in:\n% x", want, obj.Code)
		}
	}
	if bytes.Contains(obj.Code, []byte{0xB8, 0x5B, 0x00, 0x00, 0x00, 0xCD, 0x80}) {
		t.Fatalf("debug island free emitted munmap instead of guard/protect:\n% x", obj.Code)
	}
}

func TestCodegenObjectLinuxX86EmitsMMIOReadWrite(t *testing.T) {
	obj, err := CodegenObjectLinuxX86([]ir.IRFunc{{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 4096},
			{Kind: ir.IRConstI32, Imm: 123},
			{Kind: ir.IRCapIO},
			{Kind: ir.IRMmioWriteI32},
			{Kind: ir.IRConstI32, Imm: 4096},
			{Kind: ir.IRCapIO},
			{Kind: ir.IRMmioReadI32},
			{Kind: ir.IRReturn},
		},
	}})
	if err != nil {
		t.Fatalf("codegen mmio object: %v", err)
	}
	for _, want := range [][]byte{
		{0x5A, 0x59, 0x58, 0x89, 0x08, 0x51}, // pop cap/value/ptr; store; return value
		{0x5A, 0x58, 0x8B, 0x00, 0x50},       // pop cap/ptr; load; push value
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("mmio code missing % x in:\n% x", want, obj.Code)
		}
	}
}
