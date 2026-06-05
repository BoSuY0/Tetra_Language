package machine

import (
	"fmt"

	"tetra_language/compiler/internal/ir"
)

func ScalarIntFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	return ScalarIntFunctionFromStackIRWithCallABI(fn, SysVCallABIInfo())
}

func ScalarIntFunctionFromStackIRWithCallABI(fn ir.IRFunc, callABI CallABIInfo) (Function, bool, error) {
	if fn.ReturnSlots != 1 || fn.ParamSlots < 0 || fn.LocalSlots < fn.ParamSlots {
		return Function{}, false, nil
	}
	if err := validateCallABIInfo(callABI); err != nil {
		return Function{}, true, err
	}
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	tempID := 0
	temp := func() VReg {
		reg := VReg(fmt.Sprintf("t%d", tempID))
		tempID++
		return reg
	}
	params := make([]VReg, fn.ParamSlots)
	for i := range params {
		params[i] = local(i)
	}
	stack := []VReg{}
	instrs := []Instr{}
	pop := func(kind ir.IRInstrKind) (VReg, error) {
		if len(stack) == 0 {
			return "", fmt.Errorf("machine scalar lowering: %s stack underflow at %s", fn.Name, scalarIRKindName(kind))
		}
		reg := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		return reg, nil
	}
	push := func(reg VReg) {
		stack = append(stack, reg)
	}
	for _, instr := range fn.Instrs {
		switch instr.Kind {
		case ir.IRConstI32:
			dst := temp()
			instrs = append(instrs, Instr{Op: OpMov, Defs: []VReg{dst}, Imm: int64(instr.Imm)})
			push(dst)
		case ir.IRLoadLocal:
			if instr.Local < 0 || instr.Local >= fn.LocalSlots {
				return Function{}, true, fmt.Errorf("machine scalar lowering: %s local %d out of bounds", fn.Name, instr.Local)
			}
			push(local(instr.Local))
		case ir.IRStoreLocal:
			if instr.Local < 0 || instr.Local >= fn.LocalSlots {
				return Function{}, true, fmt.Errorf("machine scalar lowering: %s local %d out of bounds", fn.Name, instr.Local)
			}
			src, err := pop(instr.Kind)
			if err != nil {
				return Function{}, true, err
			}
			instrs = append(instrs, Instr{Op: OpMov, Defs: []VReg{local(instr.Local)}, Uses: []VReg{src}})
		case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32, ir.IRDivI32, ir.IRModI32, ir.IRCmpEqI32, ir.IRCmpLtI32,
			ir.IRCmpGtI32, ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
			right, err := pop(instr.Kind)
			if err != nil {
				return Function{}, true, err
			}
			left, err := pop(instr.Kind)
			if err != nil {
				return Function{}, true, err
			}
			dst := temp()
			instrs = append(instrs, Instr{Op: scalarMachineOpcode(instr.Kind), Defs: []VReg{dst}, Uses: []VReg{left, right}})
			push(dst)
		case ir.IRNegI32:
			src, err := pop(instr.Kind)
			if err != nil {
				return Function{}, true, err
			}
			dst := temp()
			instrs = append(instrs, Instr{Op: OpSub, Defs: []VReg{dst}, Uses: []VReg{VReg("zero"), src}, Note: "neg"})
			instrs = append([]Instr{{Op: OpMov, Defs: []VReg{VReg("zero")}, Imm: 0}}, instrs...)
			push(dst)
		case ir.IRCall:
			if instr.Name == "" || instr.ArgSlots < 0 || instr.RetSlots < 0 {
				return Function{}, false, nil
			}
			if instr.ArgSlots > callABI.MaxArgSlots || instr.RetSlots > callABI.MaxRetSlots {
				return Function{}, false, nil
			}
			args := make([]VReg, instr.ArgSlots)
			for i := instr.ArgSlots - 1; i >= 0; i-- {
				arg, err := pop(instr.Kind)
				if err != nil {
					return Function{}, true, err
				}
				args[i] = arg
			}
			call := Instr{
				Op:       OpCall,
				Uses:     args,
				Call:     instr.Name,
				ABI:      callABI.Name,
				Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
			}
			if instr.RetSlots == 1 {
				dst := temp()
				call.Defs = []VReg{dst}
				push(dst)
			}
			instrs = append(instrs, call)
		case ir.IRReturn:
			ret, err := pop(instr.Kind)
			if err != nil {
				return Function{}, true, err
			}
			if len(stack) != 0 {
				return Function{}, true, fmt.Errorf("machine scalar lowering: %s return leaves %d extra stack values", fn.Name, len(stack))
			}
			instrs = append(instrs, Instr{Op: OpReturn, Uses: []VReg{ret}})
		default:
			return Function{}, false, nil
		}
	}
	if len(instrs) == 0 || instrs[len(instrs)-1].Op != OpReturn {
		return Function{}, false, nil
	}
	out := Function{
		Name:   fn.Name,
		Target: "scalar-int",
		Params: params,
		Blocks: []Block{{
			Name:   "entry",
			Instrs: instrs,
		}},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, true, err
	}
	return out, true, nil
}

func validateCallABIInfo(info CallABIInfo) error {
	if info.Name == "" {
		return fmt.Errorf("machine scalar lowering: call ABI name is empty")
	}
	if len(info.Clobbers) == 0 {
		return fmt.Errorf("machine scalar lowering: call ABI %q has no caller-saved clobbers", info.Name)
	}
	if info.MaxArgSlots < 0 || info.MaxRetSlots < 0 {
		return fmt.Errorf("machine scalar lowering: call ABI %q has negative slot limits", info.Name)
	}
	return nil
}

func scalarMachineOpcode(kind ir.IRInstrKind) Opcode {
	switch kind {
	case ir.IRAddI32:
		return OpAdd
	case ir.IRSubI32:
		return OpSub
	case ir.IRMulI32:
		return OpMul
	case ir.IRDivI32:
		return OpDiv
	case ir.IRModI32:
		return OpMod
	case ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32, ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return OpCmp
	default:
		return ""
	}
}

func scalarIRKindName(kind ir.IRInstrKind) string {
	return fmt.Sprintf("ir.%d", kind)
}
