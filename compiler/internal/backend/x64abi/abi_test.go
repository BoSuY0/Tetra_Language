package x64abi

import (
	"bytes"
	"fmt"
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
		for retSlots := 0; retSlots <= 2; retSlots++ {
			t.Run(tc.name+"/"+argCountName(retSlots), func(t *testing.T) {
				e := &x64.Emitter{}
				stackDepth := 0
				var callPatches []x64obj.CallPatch
				err := tc.abi.EmitCall(e, ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "callee",
					ArgSlots: 0,
					RetSlots: retSlots,
				}, &stackDepth, &callPatches)
				if err != nil {
					t.Fatalf("EmitCall: %v", err)
				}
				if len(callPatches) != 1 || callPatches[0].Name != "callee" {
					t.Fatalf("call patches = %#v", callPatches)
				}
				if stackDepth != retSlots {
					t.Fatalf("stack depth = %d, want %d", stackDepth, retSlots)
				}

				wantSuffix := &x64.Emitter{}
				if retSlots >= 1 {
					wantSuffix.PushRax()
				}
				if retSlots >= 2 {
					wantSuffix.PushRdx()
				}
				if !bytes.HasSuffix(e.Buf, wantSuffix.Buf) {
					t.Fatalf("return-slot push suffix mismatch\n got=% x\nwant suffix=% x", e.Buf, wantSuffix.Buf)
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

			err = tc.abi.EmitCall(e, ir.IRInstr{Kind: ir.IRCall, Name: "underflow", ArgSlots: 1}, &stackDepth, &callPatches)
			if err == nil {
				t.Fatalf("expected stack underflow error")
			}
		})
	}
}

func argCountName(n int) string {
	return fmt.Sprintf("args_%02d", n)
}
