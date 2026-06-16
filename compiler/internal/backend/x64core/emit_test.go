package x64core

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/ir"
)

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

func emitOneFuncWithOptions(t *testing.T, abi x64abi.ABI, fn ir.IRFunc, opt x64.CodegenOptions) []byte {
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

	emitFn := NewEmitFunc(abi)
	e := &x64.Emitter{}
	var dataBlobs [][]byte
	var leaPatches []x64obj.LeaPatch
	var callPatches []x64obj.CallPatch
	var importPatches []x64obj.ImportPatch
	if err := emitFn(e, fn, &dataBlobs, &leaPatches, &callPatches, &importPatches, x64.CodegenOptions{}); err != nil {
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
		t.Fatalf("callPatches len = %d, want 1: %#v", len(artifacts.callPatches), artifacts.callPatches)
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
		t.Fatalf("raw_slice_i32_from_parts did not emit i32 byte-overflow guard % x in: % x", wantCmp, code)
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
		t.Fatalf("ctx_switch target slice out of bounds: target=%d want=%d len=%d", target, len(want), len(buf))
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
		t.Fatalf("ctx_switch target slice out of bounds: target=%d want=%d len=%d", target, len(want), len(buf))
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
		t.Fatalf("call opcode too early to contain prologue shadow-space adjustment: callOp=%d", callOp)
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
			t.Fatalf("SysV register call emitted stack-machine push/pop byte % x: % x", forbidden, sysv.code)
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
			t.Fatalf("Win64 register call emitted stack-machine push/pop byte % x: % x", forbidden, win.code)
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
	if callEnd+len(addShadow.Buf) > len(win.code) || !bytes.Equal(win.code[callEnd:callEnd+len(addShadow.Buf)], addShadow.Buf) {
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
	err = emitFn(nil, ir.IRFunc{Name: "__test"}, &dataBlobs, &leaPatches, &callPatches, nil, x64.CodegenOptions{})
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
			if !strings.Contains(err.Error(), "function '__test_invalid_frame_slots' has invalid slots") {
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
			if !strings.Contains(err.Error(), "global slot -1 out of bounds in function '__test_bad_global'") {
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
	read := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), readFn, x64.CodegenOptions{PointerWidthBits: 32})
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
	write := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), writeFn, x64.CodegenOptions{PointerWidthBits: 32})
	assertContainsBytes(t, "x32 ptr write guard width", write, addEdxImm32Bytes(4))
	assertContainsBytes(t, "x32 ptr write zero-extends returned pointer", write, []byte{0x45, 0x89, 0xC0})
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
	archWrite := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), archWriteFn, x64.CodegenOptions{PointerWidthBits: 32, RegisterWidthBits: 64})
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
	offsetRead := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), offsetReadFn, x64.CodegenOptions{PointerWidthBits: 32})
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
	offsetWrite := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), offsetWriteFn, x64.CodegenOptions{PointerWidthBits: 32})
	assertContainsBytes(t, "x32 ptr offset write guard width", offsetWrite, addEdxImm32Bytes(4))
	assertContainsBytes(t, "x32 ptr offset write zero-extends returned pointer", offsetWrite, []byte{0x45, 0x89, 0xC0})
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
	archOffsetWrite := emitOneFuncWithOptions(t, x64abi.LinuxSysV(), archOffsetWriteFn, x64.CodegenOptions{PointerWidthBits: 32, RegisterWidthBits: 64})
	assertContainsBytes(t, "x32 arch ptr offset write guard width", archOffsetWrite, addEdxImm32Bytes(8))
	assertContainsBytes(t, "x32 arch ptr offset write 64-bit store", archOffsetWrite, forbidStore64.Buf)
	assertNotContainsBytes(t, "x32 arch ptr offset write 32-bit store", archOffsetWrite, wantStore32.Buf)
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
	err := emitFn(e, ir.IRFunc{Name: "__test_bad_pointer_width"}, &dataBlobs, &leaPatches, &callPatches, &importPatches, x64.CodegenOptions{PointerWidthBits: 48})
	if err == nil || !strings.Contains(err.Error(), "unsupported pointer width 48") {
		t.Fatalf("expected unsupported pointer width diagnostic, got %v", err)
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
