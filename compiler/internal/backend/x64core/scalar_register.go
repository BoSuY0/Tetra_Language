package x64core

import (
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
)

func emitScalarRegisterFunction(e *x64.Emitter, fn ir.IRFunc, abi x64abi.ABI, callPatches *[]x64obj.CallPatch, opt x64.CodegenOptions, flush runtimeHeapTelemetryFlushFunc) (bool, error) {
	if opt.DisableMachinePaths {
		return false, nil
	}
	if opt.EffectiveRegisterWidthBits() != 64 {
		return false, nil
	}
	hasCall := irFuncHasCall(fn)
	var callKind scalarCallABIKind
	var callInfo machine.CallABIInfo
	if hasCall {
		var ok bool
		callKind, callInfo, ok = scalarCallABIFromBackendABI(abi)
		if !ok {
			return false, nil
		}
		if _, ok, err := machine.ScalarIntFunctionFromStackIRWithCallABI(fn, callInfo); err != nil || !ok {
			return ok, err
		}
	} else {
		if _, ok, err := machine.ScalarIntFunctionFromStackIR(fn); err != nil || !ok {
			return ok, err
		}
	}
	maxStack, err := scalarRegisterMaxStack(fn)
	if err != nil {
		return true, err
	}
	frameSlots := fn.LocalSlots + maxStack
	e.PushRbp()
	e.MovRbpRsp()
	localSize := x64.AlignStackSize(frameSlots * 8)
	if localSize > 0 {
		e.SubRspImm32(int32(localSize))
	}
	abi.SpillParams(e, fn)
	for i := fn.ParamSlots; i < fn.LocalSlots; i++ {
		e.MovMem64RbpDispImm(scalarRegisterSlotOffset(i), 0)
	}

	depth := 0
	scratchOffset := func(stackIndex int) int32 {
		return scalarRegisterSlotOffset(fn.LocalSlots + stackIndex)
	}
	pushEAX := func() {
		e.MovMem64RbpDispRax(scratchOffset(depth))
		depth++
	}
	popToEAX := func() error {
		if depth <= 0 {
			return fmt.Errorf("x64 scalar register backend: %s stack underflow", fn.Name)
		}
		depth--
		e.MovRaxFromRbpDisp(scratchOffset(depth))
		return nil
	}
	for _, instr := range fn.Instrs {
		switch instr.Kind {
		case ir.IRConstI32:
			e.MovEaxImm32(uint32(instr.Imm))
			pushEAX()
		case ir.IRLoadLocal:
			if instr.Local < 0 || instr.Local >= fn.LocalSlots {
				return true, fmt.Errorf("x64 scalar register backend: %s local %d out of bounds", fn.Name, instr.Local)
			}
			e.MovRaxFromRbpDisp(scalarRegisterSlotOffset(instr.Local))
			pushEAX()
		case ir.IRStoreLocal:
			if instr.Local < 0 || instr.Local >= fn.LocalSlots {
				return true, fmt.Errorf("x64 scalar register backend: %s local %d out of bounds", fn.Name, instr.Local)
			}
			if err := popToEAX(); err != nil {
				return true, err
			}
			e.MovMem64RbpDispRax(scalarRegisterSlotOffset(instr.Local))
		case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32, ir.IRDivI32, ir.IRModI32, ir.IRCmpEqI32, ir.IRCmpLtI32,
			ir.IRCmpGtI32, ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
			if depth < 2 {
				return true, fmt.Errorf("x64 scalar register backend: %s binary stack underflow", fn.Name)
			}
			right := scratchOffset(depth - 1)
			left := scratchOffset(depth - 2)
			e.MovEaxFromRbpDisp(right)
			e.MovEcxEax()
			e.MovEaxFromRbpDisp(left)
			switch instr.Kind {
			case ir.IRAddI32:
				e.AddEaxEcx()
			case ir.IRSubI32:
				e.SubEaxEcx()
			case ir.IRMulI32:
				e.ImulEaxEcx()
			case ir.IRDivI32:
				e.Cdq()
				e.IdivEcx()
			case ir.IRModI32:
				e.Cdq()
				e.IdivEcx()
				e.MovMem64RbpDispRdx(scratchOffset(depth - 2))
				depth--
				continue
			case ir.IRCmpEqI32:
				e.CmpEaxEcx()
				e.SeteAl()
				e.MovzxEaxAl()
			case ir.IRCmpLtI32:
				e.CmpEaxEcx()
				e.SetlAl()
				e.MovzxEaxAl()
			case ir.IRCmpGtI32:
				e.CmpEaxEcx()
				e.SetgAl()
				e.MovzxEaxAl()
			case ir.IRCmpGeI32:
				e.CmpEaxEcx()
				e.SetgeAl()
				e.MovzxEaxAl()
			case ir.IRCmpLeI32:
				e.CmpEaxEcx()
				e.SetleAl()
				e.MovzxEaxAl()
			case ir.IRCmpNeI32:
				e.CmpEaxEcx()
				e.SetneAl()
				e.MovzxEaxAl()
			}
			depth--
			e.MovMem64RbpDispRax(scratchOffset(depth - 1))
		case ir.IRNegI32:
			if err := popToEAX(); err != nil {
				return true, err
			}
			e.NegEax()
			pushEAX()
		case ir.IRCall:
			if err := emitScalarRegisterCall(e, callKind, instr, &depth, scratchOffset, callPatches); err != nil {
				return true, err
			}
		case ir.IRReturn:
			if err := popToEAX(); err != nil {
				return true, err
			}
			if depth != 0 {
				return true, fmt.Errorf("x64 scalar register backend: %s return leaves %d extra values", fn.Name, depth)
			}
			if err := flush.emit(); err != nil {
				return true, err
			}
			e.Leave()
			e.Ret()
		default:
			return false, nil
		}
	}
	return true, nil
}

func scalarRegisterSlotOffset(slot int) int32 {
	return -int32((slot + 1) * 8)
}

func scalarRegisterMaxStack(fn ir.IRFunc) (int, error) {
	depth := 0
	maxDepth := 0
	push := func(n int) {
		depth += n
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	pop := func(n int, kind ir.IRInstrKind) error {
		if depth < n {
			return fmt.Errorf("x64 scalar register backend: %s stack underflow at ir.%d", fn.Name, kind)
		}
		depth -= n
		return nil
	}
	for _, instr := range fn.Instrs {
		switch instr.Kind {
		case ir.IRConstI32, ir.IRLoadLocal:
			push(1)
		case ir.IRStoreLocal:
			if err := pop(1, instr.Kind); err != nil {
				return 0, err
			}
		case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32, ir.IRDivI32, ir.IRModI32, ir.IRCmpEqI32, ir.IRCmpLtI32,
			ir.IRCmpGtI32, ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
			if err := pop(2, instr.Kind); err != nil {
				return 0, err
			}
			push(1)
		case ir.IRNegI32:
			if err := pop(1, instr.Kind); err != nil {
				return 0, err
			}
			push(1)
		case ir.IRCall:
			if instr.Name == "" || instr.ArgSlots < 0 || instr.RetSlots < 0 || instr.RetSlots > 1 {
				return 0, fmt.Errorf("x64 scalar register backend: unsupported call ABI at ir.%d", instr.Kind)
			}
			if err := pop(instr.ArgSlots, instr.Kind); err != nil {
				return 0, err
			}
			push(instr.RetSlots)
		case ir.IRReturn:
			if err := pop(1, instr.Kind); err != nil {
				return 0, err
			}
		default:
			return 0, fmt.Errorf("x64 scalar register backend: unsupported ir.%d", instr.Kind)
		}
	}
	return maxDepth, nil
}
