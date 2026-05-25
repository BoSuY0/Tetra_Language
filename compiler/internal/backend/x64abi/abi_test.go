package x64abi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/ir"
)

func TestSysVSpillParamsZeroThroughTenArgs(t *testing.T) {
	cases := []struct {
		name string
		abi  *SysVUnix
	}{
		{name: "linux", abi: LinuxSysV()},
		{name: "linux-x32", abi: LinuxX32SysV()},
		{name: "macos", abi: MacSysV()},
	}

	for _, tc := range cases {
		for params := 0; params <= 10; params++ {
			t.Run(tc.name+"/"+argCountName(params), func(t *testing.T) {
				got := &x64.Emitter{}
				tc.abi.SpillParams(got, ir.IRFunc{ParamSlots: params})

				want := &x64.Emitter{}
				for i := 0; i < params; i++ {
					off := -int32((i + 1) * 8)
					switch i {
					case 0:
						want.MovMem64RbpDispRdi(off)
					case 1:
						want.MovMem64RbpDispRsi(off)
					case 2:
						want.MovMem64RbpDispRdx(off)
					case 3:
						want.MovMem64RbpDispRcx(off)
					case 4:
						want.MovMem64RbpDispR8(off)
					case 5:
						want.MovMem64RbpDispR9(off)
					default:
						stackOff := int32(16 + 8*(i-6))
						want.MovRaxFromRbpDisp(stackOff)
						want.MovMem64RbpDispRax(off)
					}
				}

				if !bytes.Equal(got.Buf, want.Buf) {
					t.Fatalf("spill bytes mismatch\n got=% x\nwant=% x", got.Buf, want.Buf)
				}
			})
		}
	}
}

func TestLinuxX32SysVUsesX32SyscallNumbers(t *testing.T) {
	const x32Bit = uint32(0x40000000)

	e := &x64.Emitter{}
	stackDepth := 2
	if err := LinuxX32SysV().EmitWriteStdout(e, &stackDepth, nil); err != nil {
		t.Fatalf("EmitWriteStdout: %v", err)
	}
	if !containsMovEaxImm32(e.Buf, x32Bit+1) {
		t.Fatalf("x32 write syscall number missing from bytes: % x", e.Buf)
	}
	if containsMovEaxImm32(e.Buf, 1) {
		t.Fatalf("x32 write emitted plain x64 syscall number: % x", e.Buf)
	}

	e = &x64.Emitter{}
	if err := LinuxX32SysV().EmitExit(e, 0, 0, nil); err != nil {
		t.Fatalf("EmitExit: %v", err)
	}
	if !containsMovEaxImm32(e.Buf, x32Bit+60) {
		t.Fatalf("x32 exit syscall number missing from bytes: % x", e.Buf)
	}
	if containsMovEaxImm32(e.Buf, 60) {
		t.Fatalf("x32 exit emitted plain x64 syscall number: % x", e.Buf)
	}
}

func containsMovEaxImm32(buf []byte, imm uint32) bool {
	for i := 0; i+5 <= len(buf); i++ {
		if buf[i] == 0xB8 && binary.LittleEndian.Uint32(buf[i+1:i+5]) == imm {
			return true
		}
	}
	return false
}

func TestWin64SpillParamsZeroThroughTenArgs(t *testing.T) {
	abi := NewWin64()

	for params := 0; params <= 10; params++ {
		t.Run(argCountName(params), func(t *testing.T) {
			got := &x64.Emitter{}
			abi.SpillParams(got, ir.IRFunc{ParamSlots: params})

			want := &x64.Emitter{}
			for i := 0; i < params; i++ {
				off := -int32((i + 1) * 8)
				switch i {
				case 0:
					want.MovMem64RbpDispRcx(off)
				case 1:
					want.MovMem64RbpDispRdx(off)
				case 2:
					want.MovMem64RbpDispR8(off)
				case 3:
					want.MovMem64RbpDispR9(off)
				default:
					stackOff := int32(48 + 8*(i-4))
					want.MovRaxFromRbpDisp(stackOff)
					want.MovMem64RbpDispRax(off)
				}
			}

			if !bytes.Equal(got.Buf, want.Buf) {
				t.Fatalf("spill bytes mismatch\n got=% x\nwant=% x", got.Buf, want.Buf)
			}
		})
	}
}

func TestEmitCallReturnSlotLayout(t *testing.T) {
	cases := []struct {
		name string
		abi  ABI
	}{
		{name: "sysv", abi: LinuxSysV()},
		{name: "win64", abi: NewWin64()},
	}

	for _, tc := range cases {
		for _, ret := range []struct {
			slots int
			regs  []string
		}{
			{slots: 0},
			{slots: 1, regs: []string{"rax"}},
			{slots: 2, regs: []string{"rax", "rdx"}},
			{slots: 3, regs: []string{"rax", "rdx", "r8"}},
			{slots: 4, regs: []string{"rax", "rdx", "r8", "r9"}},
			{slots: 5, regs: []string{"rax", "rdx", "r8", "r9", "r10"}},
			{slots: 6, regs: []string{"rax", "rdx", "r8", "r9", "r10", "r11"}},
			{slots: 7, regs: []string{"rax", "rdx", "r8", "r9", "r10", "r11", "r12"}},
			{slots: 8, regs: []string{"rax", "rdx", "r8", "r9", "r10", "r11", "r12", "r13"}},
			{slots: 9, regs: []string{"rax", "rdx", "r8", "r9", "r10", "r11", "r12", "r13", "r14"}},
			{slots: 10, regs: []string{"rax", "rdx", "r8", "r9", "r10", "r11", "r12", "r13", "r14", "r15"}},
		} {
			t.Run(tc.name+"/"+returnSlotName(ret.slots), func(t *testing.T) {
				e := &x64.Emitter{}
				stackDepth := 0
				var callPatches []x64obj.CallPatch
				err := tc.abi.EmitCall(e, ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "callee",
					ArgSlots: 0,
					RetSlots: ret.slots,
				}, &stackDepth, &callPatches)
				if err != nil {
					t.Fatalf("EmitCall: %v", err)
				}
				if len(callPatches) != 1 || callPatches[0].Name != "callee" {
					t.Fatalf("call patches = %#v", callPatches)
				}
				if stackDepth != ret.slots {
					t.Fatalf("stack depth = %d, want %d", stackDepth, ret.slots)
				}

				wantSuffix := &x64.Emitter{}
				emitReturnSlotPushes(wantSuffix, ret.regs)
				if !bytes.HasSuffix(e.Buf, wantSuffix.Buf) {
					t.Fatalf("return-slot push suffix mismatch for registers %v\n got=% x\nwant suffix=% x", ret.regs, e.Buf, wantSuffix.Buf)
				}
			})
		}
	}
}

func TestEmitCallRejectsInvalidABIInputs(t *testing.T) {
	cases := []struct {
		name string
		abi  ABI
	}{
		{name: "sysv", abi: LinuxSysV()},
		{name: "win64", abi: NewWin64()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := &x64.Emitter{}
			var callPatches []x64obj.CallPatch
			stackDepth := 0
			err := tc.abi.EmitCall(e, ir.IRInstr{Kind: ir.IRCall, Name: "bad", ArgSlots: -1}, &stackDepth, &callPatches)
			if err == nil {
				t.Fatalf("expected invalid argument count error")
			}

			err = tc.abi.EmitCall(e, ir.IRInstr{Kind: ir.IRCall, ArgSlots: 0}, &stackDepth, &callPatches)
			if err == nil || !strings.Contains(err.Error(), "call is missing target name") {
				t.Fatalf("expected missing target error, got %v", err)
			}

			err = tc.abi.EmitCall(e, ir.IRInstr{Kind: ir.IRCall, Name: "bad_return", RetSlots: -1}, &stackDepth, &callPatches)
			if err == nil || !strings.Contains(err.Error(), `call "bad_return" has negative ABI slots`) {
				t.Fatalf("expected negative return slots error, got %v", err)
			}

			err = tc.abi.EmitCall(e, ir.IRInstr{Kind: ir.IRCall, Name: "too_many_returns", RetSlots: 11}, &stackDepth, &callPatches)
			if err == nil || !strings.Contains(err.Error(), `call "too_many_returns" has unsupported return slots`) {
				t.Fatalf("expected unsupported return slots error, got %v", err)
			}

			err = tc.abi.EmitCall(e, ir.IRInstr{Kind: ir.IRCall, Name: "underflow", ArgSlots: 1}, &stackDepth, &callPatches)
			if err == nil {
				t.Fatalf("expected stack underflow error")
			}
		})
	}
}

func TestEmitIslandNewDebugInitializesDebugHeader(t *testing.T) {
	cases := []struct {
		name            string
		abi             ABI
		wantAllocImport string
	}{
		{name: "sysv", abi: LinuxSysV()},
		{name: "win64", abi: NewWin64(), wantAllocImport: winImportVirtualAlloc},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := &x64.Emitter{}
			stackDepth := 1
			var importPatches []x64obj.ImportPatch
			if err := tc.abi.EmitIslandNew(e, &stackDepth, x64.CodegenOptions{IslandsDebug: true}, &importPatches); err != nil {
				t.Fatalf("EmitIslandNew: %v", err)
			}
			if stackDepth != 1 {
				t.Fatalf("stack depth = %d, want 1", stackDepth)
			}

			header := &x64.Emitter{}
			header.MovMem32RaxPtrImm32(0, x64.IslandsDebugPageSize)
			if !bytes.Contains(e.Buf, header.Buf) {
				t.Fatalf("debug island header size not emitted\n got=% x\nwant contains=% x", e.Buf, header.Buf)
			}
			freedMarkerClear := &x64.Emitter{}
			freedMarkerClear.MovMem32RaxPtrImm32(12, 0)
			if !bytes.Contains(e.Buf, freedMarkerClear.Buf) {
				t.Fatalf("debug island freed marker clear not emitted\n got=% x\nwant contains=% x", e.Buf, freedMarkerClear.Buf)
			}
			if tc.wantAllocImport != "" && !hasImportPatch(importPatches, tc.wantAllocImport) {
				t.Fatalf("import patches = %#v, want %s", importPatches, tc.wantAllocImport)
			}
		})
	}
}

func TestEmitIslandFreeDebugEmitsDoubleFreeGuard(t *testing.T) {
	cases := []struct {
		name              string
		abi               ABI
		wantExitCodeBytes func() []byte
		wantProtectImport string
	}{
		{
			name: "sysv",
			abi:  LinuxSysV(),
			wantExitCodeBytes: func() []byte {
				want := &x64.Emitter{}
				want.MovEdiImm32(2)
				return want.Buf
			},
		},
		{
			name: "win64",
			abi:  NewWin64(),
			wantExitCodeBytes: func() []byte {
				want := &x64.Emitter{}
				want.MovEcxImm32(2)
				return want.Buf
			},
			wantProtectImport: winImportVirtualProtect,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := &x64.Emitter{}
			stackDepth := 1
			var importPatches []x64obj.ImportPatch
			if err := tc.abi.EmitIslandFree(e, &stackDepth, x64.CodegenOptions{IslandsDebug: true}, &importPatches); err != nil {
				t.Fatalf("EmitIslandFree: %v", err)
			}
			if stackDepth != 0 {
				t.Fatalf("stack depth = %d, want 0", stackDepth)
			}

			freedCheck := &x64.Emitter{}
			freedCheck.MovEaxFromRdiDisp(12)
			freedCheck.TestEaxEax()
			if !bytes.Contains(e.Buf, freedCheck.Buf) {
				t.Fatalf("debug island freed check not emitted\n got=% x\nwant contains=% x", e.Buf, freedCheck.Buf)
			}
			if !bytes.Contains(e.Buf, tc.wantExitCodeBytes()) {
				t.Fatalf("debug island exit code 2 not emitted\n got=% x", e.Buf)
			}
			freedMarkerSet := &x64.Emitter{}
			freedMarkerSet.MovRaxRdi()
			freedMarkerSet.MovMem32RaxPtrImm32(12, 1)
			if !bytes.Contains(e.Buf, freedMarkerSet.Buf) {
				t.Fatalf("debug island freed marker set not emitted\n got=% x\nwant contains=% x", e.Buf, freedMarkerSet.Buf)
			}
			protectLen := &x64.Emitter{}
			protectLen.SubEaxImm32(x64.IslandsDebugPageSize)
			if !bytes.Contains(e.Buf, protectLen.Buf) {
				t.Fatalf("debug island protected length adjustment not emitted\n got=% x\nwant contains=% x", e.Buf, protectLen.Buf)
			}
			if tc.wantProtectImport != "" && !hasImportPatch(importPatches, tc.wantProtectImport) {
				t.Fatalf("import patches = %#v, want %s", importPatches, tc.wantProtectImport)
			}
		})
	}
}

func TestSysVAllocBytesEmitsDeterministicMmapFailureGuard(t *testing.T) {
	e := &x64.Emitter{}
	stackDepth := 1
	if err := LinuxSysV().EmitAllocBytes(e, &stackDepth, x64.CodegenOptions{}, nil); err != nil {
		t.Fatalf("EmitAllocBytes: %v", err)
	}
	if stackDepth != 1 {
		t.Fatalf("stack depth = %d, want 1", stackDepth)
	}

	cmpMmapErrorRange := []byte{0x48, 0x3D, 0x01, 0xF0, 0xFF, 0xFF}
	if !bytes.Contains(e.Buf, cmpMmapErrorRange) {
		t.Fatalf("alloc_bytes missing mmap error-range compare\n got=% x\nwant contains=% x", e.Buf, cmpMmapErrorRange)
	}
	jaeRel32 := []byte{0x0F, 0x83}
	if !bytes.Contains(e.Buf, jaeRel32) {
		t.Fatalf("alloc_bytes missing mmap failure branch\n got=% x", e.Buf)
	}
	exit2 := &x64.Emitter{}
	exit2.MovEdiImm32(2)
	if !bytes.Contains(e.Buf, exit2.Buf) {
		t.Fatalf("alloc_bytes missing deterministic failure exit code 2\n got=% x", e.Buf)
	}
}

func hasImportPatch(patches []x64obj.ImportPatch, name string) bool {
	for _, patch := range patches {
		if patch.Name == name {
			return true
		}
	}
	return false
}

func TestABIEdgeCallStackArgCases(t *testing.T) {
	cases := []struct {
		name         string
		abi          ABI
		argSlots     int
		containsCode []byte
	}{
		{
			name:         "sysv_arg7_spills_to_stack_area",
			abi:          LinuxSysV(),
			argSlots:     7,
			containsCode: []byte{0x48, 0x89, 0x84, 0x24}, // mov [rsp+disp], rax
		},
		{
			name:         "win64_arg5_uses_shadow_space_frame",
			abi:          NewWin64(),
			argSlots:     5,
			containsCode: []byte{0x48, 0x81, 0xEC, 0x28, 0x00, 0x00, 0x00}, // sub rsp, 40
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := &x64.Emitter{}
			stackDepth := tc.argSlots
			var callPatches []x64obj.CallPatch
			err := tc.abi.EmitCall(e, ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "callee",
				ArgSlots: tc.argSlots,
				RetSlots: 2,
			}, &stackDepth, &callPatches)
			if err != nil {
				t.Fatalf("EmitCall: %v", err)
			}
			if len(callPatches) != 1 || callPatches[0].Name != "callee" {
				t.Fatalf("call patches = %#v", callPatches)
			}
			if stackDepth != 2 {
				t.Fatalf("stack depth = %d, want 2", stackDepth)
			}
			if !bytes.Contains(e.Buf, tc.containsCode) {
				t.Fatalf("expected call sequence to contain % x\nbuf=% x", tc.containsCode, e.Buf)
			}
		})
	}
}

func TestABIDiagnosticMethodsRejectMissingPointers(t *testing.T) {
	cases := []struct {
		name string
		abi  ABI
	}{
		{name: "sysv", abi: LinuxSysV()},
		{name: "win64", abi: NewWin64()},
	}

	for _, tc := range cases {
		t.Run(tc.name+"/call_missing_pointers", func(t *testing.T) {
			e := &x64.Emitter{}
			err := tc.abi.EmitCall(e, ir.IRInstr{Kind: ir.IRCall, Name: "callee"}, nil, nil)
			if err == nil || !strings.Contains(err.Error(), "internal error") {
				t.Fatalf("expected internal error, got %v", err)
			}
		})

		t.Run(tc.name+"/write_missing_stack", func(t *testing.T) {
			e := &x64.Emitter{}
			err := tc.abi.EmitWriteStdout(e, nil, nil)
			if err == nil || !strings.Contains(err.Error(), "internal error") {
				t.Fatalf("expected internal error, got %v", err)
			}
		})

		t.Run(tc.name+"/alloc_missing_stack", func(t *testing.T) {
			e := &x64.Emitter{}
			err := tc.abi.EmitAllocBytes(e, nil, x64.CodegenOptions{}, nil)
			if err == nil || !strings.Contains(err.Error(), "internal error") {
				t.Fatalf("expected internal error, got %v", err)
			}
		})

		t.Run(tc.name+"/make_slice_missing_stack", func(t *testing.T) {
			e := &x64.Emitter{}
			err := tc.abi.EmitMakeSlice(e, ir.IRMakeSliceU8, nil, x64.CodegenOptions{}, nil)
			if err == nil || !strings.Contains(err.Error(), "internal error") {
				t.Fatalf("expected internal error, got %v", err)
			}
		})
	}

	t.Run("win64_exit_missing_imports", func(t *testing.T) {
		e := &x64.Emitter{}
		err := NewWin64().EmitExit(e, 0, 0, nil)
		if err == nil || !strings.Contains(err.Error(), "internal error") {
			t.Fatalf("expected internal error, got %v", err)
		}
	})
}

func TestEmitMakeSliceZeroLengthBypassesAllocator(t *testing.T) {
	cases := []struct {
		name          string
		abi           ABI
		importPatches *[]x64obj.ImportPatch
	}{
		{name: "linux", abi: LinuxSysV()},
		{name: "linux-x32", abi: LinuxX32SysV()},
		{name: "macos", abi: MacSysV()},
		{name: "win64", abi: NewWin64(), importPatches: &[]x64obj.ImportPatch{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := &x64.Emitter{}
			stackDepth := 1
			e.PushRax()
			if err := tc.abi.EmitMakeSlice(e, ir.IRMakeSliceI32, &stackDepth, x64.CodegenOptions{}, tc.importPatches); err != nil {
				t.Fatalf("EmitMakeSlice: %v", err)
			}
			if stackDepth != 2 {
				t.Fatalf("stackDepth = %d, want 2", stackDepth)
			}
			if !bytes.Contains(e.Buf, []byte{0x48, 0x85, 0xC0, 0x0F, 0x84}) {
				t.Fatalf("make_slice missing zero-length test/jz bypass:\n% x", e.Buf)
			}
			if !bytes.Contains(e.Buf, []byte{0x50, 0x50}) {
				t.Fatalf("make_slice empty branch does not push ptr/len zeros:\n% x", e.Buf)
			}
		})
	}
}

func argCountName(n int) string {
	return fmt.Sprintf("args_%02d", n)
}

func returnSlotName(n int) string {
	return fmt.Sprintf("return_slots_%02d", n)
}

func emitReturnSlotPushes(e *x64.Emitter, regs []string) {
	for _, reg := range regs {
		switch reg {
		case "rax":
			e.PushRax()
		case "rdx":
			e.PushRdx()
		case "r8":
			e.PushR8()
		case "r9":
			e.PushR9()
		case "r10":
			e.PushR10()
		case "r11":
			e.PushR11()
		case "r12":
			e.PushR12()
		case "r13":
			e.PushR13()
		case "r14":
			e.PushR14()
		case "r15":
			e.PushR15()
		default:
			panic(fmt.Sprintf("unknown return register %q", reg))
		}
	}
}
