package x64core

import (
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
)

type scalarCallABIKind int

const (
	scalarCallABISysV scalarCallABIKind = iota + 1
	scalarCallABIWin64
)

func scalarCallABIFromBackendABI(abi x64abi.ABI) (scalarCallABIKind, machine.CallABIInfo, bool) {
	switch abi.(type) {
	case *x64abi.SysVUnix:
		return scalarCallABISysV, machine.SysVCallABIInfo(), true
	case *x64abi.Win64:
		return scalarCallABIWin64, machine.Win64CallABIInfo(), true
	default:
		return 0, machine.CallABIInfo{}, false
	}
}

func irFuncHasCall(fn ir.IRFunc) bool {
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall {
			return true
		}
	}
	return false
}

func emitScalarRegisterCall(
	e *x64.Emitter,
	kind scalarCallABIKind,
	instr ir.IRInstr,
	depth *int,
	scratchOffset func(int) int32,
	callPatches *[]x64obj.CallPatch,
) error {
	if e == nil || depth == nil || scratchOffset == nil || callPatches == nil {
		return fmt.Errorf("x64 scalar register backend: missing call emission state")
	}
	if instr.Name == "" {
		return fmt.Errorf("x64 scalar register backend: call is missing target name")
	}
	if instr.ArgSlots < 0 || instr.RetSlots < 0 {
		return fmt.Errorf("x64 scalar register backend: call %q has negative ABI slots args=%d rets=%d", instr.Name, instr.ArgSlots, instr.RetSlots)
	}
	if instr.RetSlots > 1 {
		return fmt.Errorf("x64 scalar register backend: call %q has unsupported register return slots %d", instr.Name, instr.RetSlots)
	}
	maxArgs := scalarRegisterCallMaxArgs(kind)
	if instr.ArgSlots > maxArgs {
		return fmt.Errorf("x64 scalar register backend: call %q has unsupported register arg slots %d (max=%d)", instr.Name, instr.ArgSlots, maxArgs)
	}
	if *depth < instr.ArgSlots {
		return fmt.Errorf("x64 scalar register backend: stack underflow in call to %q", instr.Name)
	}

	argBase := *depth - instr.ArgSlots
	for i := 0; i < instr.ArgSlots; i++ {
		e.MovRaxFromRbpDisp(scratchOffset(argBase + i))
		emitMoveRaxToScalarCallArg(e, kind, i)
	}
	*depth -= instr.ArgSlots
	emitScalarCallFramePrologue(e, kind)
	at := e.CallRel32()
	*callPatches = append(*callPatches, x64obj.CallPatch{At: at, Name: instr.Name})
	emitScalarCallFrameEpilogue(e, kind)
	if instr.RetSlots == 1 {
		e.MovMem64RbpDispRax(scratchOffset(*depth))
		*depth++
	}
	return nil
}

func scalarRegisterCallMaxArgs(kind scalarCallABIKind) int {
	switch kind {
	case scalarCallABIWin64:
		return 4
	default:
		return 6
	}
}

func emitMoveRaxToScalarCallArg(e *x64.Emitter, kind scalarCallABIKind, arg int) {
	if kind == scalarCallABIWin64 {
		switch arg {
		case 0:
			e.MovRcxRax()
		case 1:
			e.MovRdxRax()
		case 2:
			e.MovR8Rax()
		case 3:
			e.MovR9Rax()
		}
		return
	}
	switch arg {
	case 0:
		e.MovRdiRax()
	case 1:
		e.MovRsiRax()
	case 2:
		e.MovRdxRax()
	case 3:
		e.MovRcxRax()
	case 4:
		e.MovR8Rax()
	case 5:
		e.MovR9Rax()
	}
}

func emitScalarCallFramePrologue(e *x64.Emitter, kind scalarCallABIKind) {
	if kind == scalarCallABIWin64 {
		e.SubRspImm32(32)
	}
}

func emitScalarCallFrameEpilogue(e *x64.Emitter, kind scalarCallABIKind) {
	if kind == scalarCallABIWin64 {
		e.AddRspImm32(32)
	}
}

func emitScalarLoopCall(
	e *x64.Emitter,
	kind scalarCallABIKind,
	name string,
	callPatches *[]x64obj.CallPatch,
) error {
	if e == nil || callPatches == nil {
		return fmt.Errorf("x64 scalar call-loop backend: missing call emission state")
	}
	if name == "" {
		return fmt.Errorf("x64 scalar call-loop backend: call is missing target name")
	}
	emitScalarCallFramePrologue(e, kind)
	at := e.CallRel32()
	*callPatches = append(*callPatches, x64obj.CallPatch{At: at, Name: name})
	emitScalarCallFrameEpilogue(e, kind)
	return nil
}
