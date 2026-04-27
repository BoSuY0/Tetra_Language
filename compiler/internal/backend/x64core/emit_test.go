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

func emitOneFunc(t *testing.T, abi x64abi.ABI, fn ir.IRFunc) []byte {
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
	return e.Buf
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
