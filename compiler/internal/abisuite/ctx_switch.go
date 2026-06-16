package abisuite

import (
	"bytes"
	"fmt"

	"tetra_language/compiler/internal/backend/linux_x32"
	"tetra_language/compiler/internal/backend/linux_x86"
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/ir"
)

type CtxSwitchObject struct {
	Target string
	Code   []byte
}

type CtxSwitchDeps struct {
	BuildX86Object func(funcs []ir.IRFunc) (CtxSwitchObject, error)
	BuildX32Object func(funcs []ir.IRFunc) (CtxSwitchObject, error)
}

func CheckX86CtxSwitchObjectSmoke(deps CtxSwitchDeps) error {
	obj, err := buildX86CtxSwitchObject(deps, ctxSwitchSmokeIR("__tetra_x86_ctx_switch_smoke"))
	if err != nil {
		return err
	}
	if !bytes.Contains(obj.Code, ctxSwitchI386Stub()) {
		return fmt.Errorf("x86 ctx_switch object missing i386 context stub")
	}
	if !bytes.Contains(obj.Code, ctxSwitchZeroStatusContinuation()) {
		return fmt.Errorf("x86 ctx_switch object missing zero status continuation")
	}
	return nil
}

func CheckX32CtxSwitchObjectSmoke(deps CtxSwitchDeps) error {
	obj, err := buildX32CtxSwitchObject(deps, ctxSwitchSmokeIR("__tetra_x32_ctx_switch_smoke"))
	if err != nil {
		return err
	}
	if obj.Target != "linux-x32" {
		return fmt.Errorf("x32 ctx_switch object target = %q, want linux-x32", obj.Target)
	}
	if !bytes.Contains(obj.Code, ctxSwitchX32SysVStub()) {
		return fmt.Errorf("x32 ctx_switch object missing SysV x86_64 context stub")
	}
	if bytes.Contains(obj.Code, ctxSwitchX32ShadowSpaceAdjustment()) {
		return fmt.Errorf("x32 ctx_switch object unexpectedly emitted Win64 shadow-space adjustment")
	}
	if !bytes.Contains(obj.Code, ctxSwitchZeroStatusContinuation()) {
		return fmt.Errorf("x32 ctx_switch object missing zero status continuation")
	}
	return nil
}

func buildX86CtxSwitchObject(deps CtxSwitchDeps, funcs []ir.IRFunc) (CtxSwitchObject, error) {
	if deps.BuildX86Object != nil {
		return deps.BuildX86Object(funcs)
	}
	obj, err := linux_x86.CodegenObjectLinuxX86(funcs)
	if err != nil {
		return CtxSwitchObject{}, err
	}
	return CtxSwitchObject{Target: obj.Target, Code: obj.Code}, nil
}

func buildX32CtxSwitchObject(deps CtxSwitchDeps, funcs []ir.IRFunc) (CtxSwitchObject, error) {
	if deps.BuildX32Object != nil {
		return deps.BuildX32Object(funcs)
	}
	obj, err := linux_x32.CodegenObjectLinuxX32(funcs)
	if err != nil {
		return CtxSwitchObject{}, err
	}
	return CtxSwitchObject{Target: obj.Target, Code: obj.Code}, nil
}

func ctxSwitchSmokeIR(name string) []ir.IRFunc {
	return []ir.IRFunc{{
		Name:        name,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRCtxSwitch},
			{Kind: ir.IRReturn},
		},
	}}
}

func ctxSwitchI386Stub() []byte {
	return []byte{0x53, 0x55, 0x56, 0x57, 0x89, 0x20, 0x8B, 0x21, 0x5F, 0x5E, 0x5D, 0x5B, 0xC3}
}

func ctxSwitchX32SysVStub() []byte {
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

func ctxSwitchX32ShadowSpaceAdjustment() []byte {
	e := &x64.Emitter{}
	e.SubRspImm32(32)
	return e.Buf
}

func ctxSwitchZeroStatusContinuation() []byte {
	return []byte{0x31, 0xC0, 0x50}
}
