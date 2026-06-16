package ssair

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/ir"
)

func FromStackIRFunction(fn ir.IRFunc) (Function, bool, error) {
	switch {
	case isSliceSumLoop(fn):
		return sliceSumLoopSSA(fn), true, nil
	case isScalarSumSquaresLoop(fn):
		return scalarSumSquaresLoopSSA(fn), true, nil
	case isScalarProductLoop(fn):
		return scalarProductLoopSSA(fn), true, nil
	case isScalarMaxLoop(fn):
		return scalarMaxLoopSSA(fn), true, nil
	case isScalarAffineLoop(fn):
		return scalarAffineLoopSSA(fn), true, nil
	case isScalarCountdownLoop(fn):
		return scalarCountdownLoopSSA(fn), true, nil
	case isScalarConstBoundTwoArgSuccessCallLoop(fn):
		return scalarConstBoundTwoArgSuccessCallLoopSSA(fn), true, nil
	case isScalarCallLoop(fn):
		return scalarLoopSSA(fn, true), true, nil
	case isScalarLoop(fn):
		return scalarLoopSSA(fn, false), true, nil
	default:
		return scalarLinearSSA(fn)
	}
}

type linearBuilder struct {
	fn          ir.IRFunc
	values      []Value
	instrs      []Instr
	stack       []ValueID
	locals      map[int]ValueID
	nextValue   int
	nextInstr   int
	nextEffect  int
	effectToken ValueID
	term        Terminator
}

func scalarLinearSSA(fn ir.IRFunc) (Function, bool, error) {
	if fn.ReturnSlots != 1 || fn.ParamSlots < 0 || fn.LocalSlots < fn.ParamSlots {
		return Function{}, false, nil
	}
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel || instr.Kind == ir.IRJmp || instr.Kind == ir.IRJmpIfZero {
			return Function{}, false, nil
		}
	}
	b := &linearBuilder{
		fn:          fn,
		locals:      map[int]ValueID{},
		effectToken: "effect0",
		nextEffect:  1,
	}
	for i := 0; i < fn.LocalSlots; i++ {
		id := ValueID(fmt.Sprintf("local%d", i))
		origin := "local"
		if i < fn.ParamSlots {
			origin = "param"
		}
		b.values = append(b.values, Value{ID: id, Type: TypeI32, Origin: origin})
		b.locals[i] = id
	}
	b.values = append(b.values, Value{ID: b.effectToken, Type: TypeEffect, Origin: "entry_effect"})
	for _, instr := range fn.Instrs {
		ok, err := b.lower(instr)
		if err != nil || !ok {
			return Function{}, ok, err
		}
	}
	if len(b.instrs) == 0 {
		return Function{}, false, nil
	}
	out := Function{
		Name:       fn.Name,
		ReturnType: TypeI32,
		Values:     b.values,
		Blocks: []Block{{
			ID:     "entry",
			Entry:  true,
			Instrs: b.instrs,
			Term:   b.term,
		}},
	}
	if b.term.Kind == TermInvalid {
		return Function{}, false, nil
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, true, err
	}
	return out, true, nil
}

func (b *linearBuilder) lower(instr ir.IRInstr) (bool, error) {
	switch instr.Kind {
	case ir.IRConstI32:
		dst := b.newValue(TypeI32, "const")
		b.instrs = append(b.instrs, Instr{ID: b.newInstr(), Kind: OpConstI32, Result: dst, Type: TypeI32, Imm: instr.Imm})
		b.push(dst)
	case ir.IRLoadLocal:
		local, ok := b.locals[instr.Local]
		if !ok {
			return true, fmt.Errorf("ssa stack lowering: %s local %d out of bounds", b.fn.Name, instr.Local)
		}
		b.push(local)
	case ir.IRStoreLocal:
		value, err := b.pop(instr.Kind)
		if err != nil {
			return true, err
		}
		if instr.Local < 0 || instr.Local >= b.fn.LocalSlots {
			return true, fmt.Errorf("ssa stack lowering: %s local %d out of bounds", b.fn.Name, instr.Local)
		}
		b.locals[instr.Local] = value
	case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32, ir.IRDivI32, ir.IRModI32, ir.IRCmpEqI32, ir.IRCmpLtI32,
		ir.IRCmpGtI32, ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		right, err := b.pop(instr.Kind)
		if err != nil {
			return true, err
		}
		left, err := b.pop(instr.Kind)
		if err != nil {
			return true, err
		}
		typ := TypeI32
		if isCompare(instr.Kind) {
			typ = TypeBool
		}
		dst := b.newValue(typ, "instr")
		b.instrs = append(b.instrs, Instr{ID: b.newInstr(), Kind: stackBinaryOp(instr.Kind), Result: dst, Type: typ, Args: []ValueID{left, right}})
		b.push(dst)
	case ir.IRNegI32:
		src, err := b.pop(instr.Kind)
		if err != nil {
			return true, err
		}
		dst := b.newValue(TypeI32, "instr")
		b.instrs = append(b.instrs, Instr{ID: b.newInstr(), Kind: OpNegI32, Result: dst, Type: TypeI32, Args: []ValueID{src}})
		b.push(dst)
	case ir.IRCall:
		if instr.Name == "" || instr.ArgSlots < 0 || instr.RetSlots < 0 || instr.ArgSlots > 6 || instr.RetSlots > 1 {
			return false, nil
		}
		args := make([]ValueID, instr.ArgSlots)
		for i := instr.ArgSlots - 1; i >= 0; i-- {
			arg, err := b.pop(instr.Kind)
			if err != nil {
				return true, err
			}
			args[i] = arg
		}
		effectOut := b.newEffect("call_effect")
		call := Instr{ID: b.newInstr(), Kind: OpCall, Args: args, Call: instr.Name, EffectIn: b.effectToken, EffectOut: effectOut}
		b.effectToken = effectOut
		if instr.RetSlots == 1 {
			dst := b.newValue(TypeI32, "call_result")
			call.Result = dst
			call.Type = TypeI32
			b.push(dst)
		}
		b.instrs = append(b.instrs, call)
	case ir.IRReturn:
		value, err := b.pop(instr.Kind)
		if err != nil {
			return true, err
		}
		if len(b.stack) != 0 {
			return true, fmt.Errorf("ssa stack lowering: %s return leaves %d extra stack values", b.fn.Name, len(b.stack))
		}
		if b.blockTerminated() {
			return false, nil
		}
		b.setReturn(value)
	default:
		return false, nil
	}
	return true, nil
}

func (b *linearBuilder) newValue(typ Type, origin string) ValueID {
	id := ValueID(fmt.Sprintf("v%d", b.nextValue))
	b.nextValue++
	b.values = append(b.values, Value{ID: id, Type: typ, Origin: origin})
	return id
}

func (b *linearBuilder) newEffect(origin string) ValueID {
	id := ValueID(fmt.Sprintf("effect%d", b.nextEffect))
	b.nextEffect++
	b.values = append(b.values, Value{ID: id, Type: TypeEffect, Origin: origin})
	return id
}

func (b *linearBuilder) newInstr() string {
	id := fmt.Sprintf("op%d", b.nextInstr)
	b.nextInstr++
	return id
}

func (b *linearBuilder) push(id ValueID) {
	b.stack = append(b.stack, id)
}

func (b *linearBuilder) pop(kind ir.IRInstrKind) (ValueID, error) {
	if len(b.stack) == 0 {
		return "", fmt.Errorf("ssa stack lowering: %s stack underflow at %d", b.fn.Name, kind)
	}
	id := b.stack[len(b.stack)-1]
	b.stack = b.stack[:len(b.stack)-1]
	return id, nil
}

func (b *linearBuilder) blockTerminated() bool {
	return b.term.Kind != TermInvalid
}

func (b *linearBuilder) setReturn(value ValueID) {
	b.term = Terminator{Kind: TermReturn, Value: value}
}

func scalarLoopSSA(fn ir.IRFunc, withCall bool) Function {
	step := scalarLoopStep(fn, withCall)
	stepValue := Value{ID: "one", Type: TypeI32, Origin: "const"}
	stepInstr := Instr{ID: "const_one", Kind: OpConstI32, Result: "one", Type: TypeI32, Imm: 1}
	stepID := ValueID("one")
	if step != 1 {
		stepValue = Value{ID: "step", Type: TypeI32, Origin: "const"}
		stepInstr = Instr{ID: "const_step", Kind: OpConstI32, Result: "step", Type: TypeI32, Imm: step}
		stepID = "step"
	}
	values := []Value{
		{ID: "local0", Type: TypeI32, Origin: "param"},
		{ID: "zero", Type: TypeI32, Origin: "const"},
		stepValue,
		{ID: "loop.index", Type: TypeI32, Origin: "block_param"},
		{ID: "loop.total", Type: TypeI32, Origin: "block_param"},
		{ID: "body.index", Type: TypeI32, Origin: "block_param"},
		{ID: "body.total", Type: TypeI32, Origin: "block_param"},
		{ID: "exit.total", Type: TypeI32, Origin: "block_param"},
		{ID: "cmp", Type: TypeBool, Origin: "instr"},
		{ID: "next.index", Type: TypeI32, Origin: "instr"},
		{ID: "next.total", Type: TypeI32, Origin: "instr"},
		{ID: "effect0", Type: TypeEffect, Origin: "entry_effect"},
	}
	bodyParams := []ValueID{"body.index", "body.total"}
	loopParams := []ValueID{"loop.index", "loop.total"}
	entryArgs := []ValueID{"zero", "zero"}
	backArgs := []ValueID{"next.index", "next.total"}
	if withCall {
		values = append(values,
			Value{ID: "loop.effect", Type: TypeEffect, Origin: "block_param"},
			Value{ID: "body.effect", Type: TypeEffect, Origin: "block_param"},
			Value{ID: "call.ret", Type: TypeI32, Origin: "call_result"},
			Value{ID: "call.effect", Type: TypeEffect, Origin: "call_effect"},
		)
		loopParams = append(loopParams, "loop.effect")
		bodyParams = append(bodyParams, "body.effect")
		entryArgs = append(entryArgs, "effect0")
		backArgs = append(backArgs, "call.effect")
	}
	bodyInstrs := []Instr{}
	if withCall {
		bodyInstrs = append(bodyInstrs, Instr{ID: "call", Kind: OpCall, Result: "call.ret", Type: TypeI32, Args: []ValueID{"body.index"}, Call: loopCallName(fn), EffectIn: "body.effect", EffectOut: "call.effect"})
		bodyInstrs = append(bodyInstrs, Instr{ID: "add_total", Kind: OpAddI32, Result: "next.total", Type: TypeI32, Args: []ValueID{"body.total", "call.ret"}})
	} else {
		bodyInstrs = append(bodyInstrs, Instr{ID: "add_total", Kind: OpAddI32, Result: "next.total", Type: TypeI32, Args: []ValueID{"body.total", "body.index"}})
	}
	bodyInstrs = append(bodyInstrs, Instr{ID: "inc_index", Kind: OpAddI32, Result: "next.index", Type: TypeI32, Args: []ValueID{"body.index", stepID}})
	return Function{
		Name:       fn.Name,
		ReturnType: TypeI32,
		Values:     values,
		Blocks: []Block{
			{
				ID:     "entry",
				Entry:  true,
				Instrs: []Instr{{ID: "const_zero", Kind: OpConstI32, Result: "zero", Type: TypeI32}, stepInstr},
				Term:   Terminator{Kind: TermBranch, Target: "loop", Args: entryArgs},
			},
			{
				ID:     "loop",
				Params: loopParams,
				Instrs: []Instr{{ID: "cmp", Kind: OpCmpLtI32, Result: "cmp", Type: TypeBool, Args: []ValueID{"loop.index", "local0"}}},
				Term:   Terminator{Kind: TermCondBr, Cond: "cmp", IfTrue: "body", IfTrueArgs: loopParams, IfFalse: "exit", IfFalseArgs: []ValueID{"loop.total"}},
			},
			{
				ID:     "body",
				Params: bodyParams,
				Instrs: bodyInstrs,
				Term:   Terminator{Kind: TermBranch, Target: "loop", Args: backArgs},
			},
			{
				ID:     "exit",
				Params: []ValueID{"exit.total"},
				Term:   Terminator{Kind: TermReturn, Value: "exit.total"},
			},
		},
	}
}

func scalarConstBoundTwoArgSuccessCallLoopSSA(fn ir.IRFunc) Function {
	callName := fn.Instrs[12].Name
	return Function{
		Name:       fn.Name,
		ReturnType: TypeI32,
		Values: []Value{
			{ID: "zero", Type: TypeI32, Origin: "const"},
			{ID: "one", Type: TypeI32, Origin: "const"},
			{ID: "bound", Type: TypeI32, Origin: "const"},
			{ID: "loop.index", Type: TypeI32, Origin: "block_param"},
			{ID: "loop.total", Type: TypeI32, Origin: "block_param"},
			{ID: "loop.effect", Type: TypeEffect, Origin: "block_param"},
			{ID: "body.index", Type: TypeI32, Origin: "block_param"},
			{ID: "body.total", Type: TypeI32, Origin: "block_param"},
			{ID: "body.effect", Type: TypeEffect, Origin: "block_param"},
			{ID: "exit.total", Type: TypeI32, Origin: "block_param"},
			{ID: "cmp.loop", Type: TypeBool, Origin: "instr"},
			{ID: "cmp.success", Type: TypeBool, Origin: "instr"},
			{ID: "call.ret", Type: TypeI32, Origin: "call_result"},
			{ID: "call.effect", Type: TypeEffect, Origin: "call_effect"},
			{ID: "next.index", Type: TypeI32, Origin: "instr"},
			{ID: "next.total", Type: TypeI32, Origin: "instr"},
			{ID: "effect0", Type: TypeEffect, Origin: "entry_effect"},
		},
		Blocks: []Block{
			{
				ID:    "entry",
				Entry: true,
				Instrs: []Instr{
					{ID: "const_zero", Kind: OpConstI32, Result: "zero", Type: TypeI32},
					{ID: "const_one", Kind: OpConstI32, Result: "one", Type: TypeI32, Imm: 1},
					{ID: "const_bound", Kind: OpConstI32, Result: "bound", Type: TypeI32, Imm: fn.Instrs[6].Imm},
				},
				Term: Terminator{Kind: TermBranch, Target: "loop", Args: []ValueID{"zero", "zero", "effect0"}},
			},
			{
				ID:     "loop",
				Params: []ValueID{"loop.index", "loop.total", "loop.effect"},
				Instrs: []Instr{{ID: "cmp_loop", Kind: OpCmpLtI32, Result: "cmp.loop", Type: TypeBool, Args: []ValueID{"loop.index", "bound"}}},
				Term:   Terminator{Kind: TermCondBr, Cond: "cmp.loop", IfTrue: "body", IfTrueArgs: []ValueID{"loop.index", "loop.total", "loop.effect"}, IfFalse: "exit", IfFalseArgs: []ValueID{"loop.total"}},
			},
			{
				ID:     "body",
				Params: []ValueID{"body.index", "body.total", "body.effect"},
				Instrs: []Instr{
					{ID: "call", Kind: OpCall, Result: "call.ret", Type: TypeI32, Args: []ValueID{"body.index", "body.total"}, Call: callName, EffectIn: "body.effect", EffectOut: "call.effect"},
					{ID: "add_total", Kind: OpAddI32, Result: "next.total", Type: TypeI32, Args: []ValueID{"body.total", "call.ret"}},
					{ID: "inc_index", Kind: OpAddI32, Result: "next.index", Type: TypeI32, Args: []ValueID{"body.index", "one"}},
				},
				Term: Terminator{Kind: TermBranch, Target: "loop", Args: []ValueID{"next.index", "next.total", "call.effect"}},
			},
			{
				ID:     "exit",
				Params: []ValueID{"exit.total"},
				Instrs: []Instr{{ID: "cmp_success", Kind: OpCmpGeI32, Result: "cmp.success", Type: TypeBool, Args: []ValueID{"exit.total", "zero"}}},
				Term:   Terminator{Kind: TermCondBr, Cond: "cmp.success", IfTrue: "success", IfFalse: "failure"},
			},
			{
				ID:   "success",
				Term: Terminator{Kind: TermReturn, Value: "zero"},
			},
			{
				ID:   "failure",
				Term: Terminator{Kind: TermReturn, Value: "one"},
			},
		},
	}
}

func scalarSumSquaresLoopSSA(fn ir.IRFunc) Function {
	return Function{
		Name:       fn.Name,
		ReturnType: TypeI32,
		Values: []Value{
			{ID: "local0", Type: TypeI32, Origin: "param"},
			{ID: "zero", Type: TypeI32, Origin: "const"},
			{ID: "one", Type: TypeI32, Origin: "const"},
			{ID: "loop.index", Type: TypeI32, Origin: "block_param"},
			{ID: "loop.total", Type: TypeI32, Origin: "block_param"},
			{ID: "body.index", Type: TypeI32, Origin: "block_param"},
			{ID: "body.total", Type: TypeI32, Origin: "block_param"},
			{ID: "exit.total", Type: TypeI32, Origin: "block_param"},
			{ID: "cmp", Type: TypeBool, Origin: "instr"},
			{ID: "square", Type: TypeI32, Origin: "instr"},
			{ID: "next.index", Type: TypeI32, Origin: "instr"},
			{ID: "next.total", Type: TypeI32, Origin: "instr"},
			{ID: "effect0", Type: TypeEffect, Origin: "entry_effect"},
		},
		Blocks: []Block{
			{
				ID:     "entry",
				Entry:  true,
				Instrs: []Instr{{ID: "const_zero", Kind: OpConstI32, Result: "zero", Type: TypeI32}, {ID: "const_one", Kind: OpConstI32, Result: "one", Type: TypeI32, Imm: 1}},
				Term:   Terminator{Kind: TermBranch, Target: "loop", Args: []ValueID{"zero", "zero"}},
			},
			{
				ID:     "loop",
				Params: []ValueID{"loop.index", "loop.total"},
				Instrs: []Instr{{ID: "cmp", Kind: OpCmpLtI32, Result: "cmp", Type: TypeBool, Args: []ValueID{"loop.index", "local0"}}},
				Term:   Terminator{Kind: TermCondBr, Cond: "cmp", IfTrue: "body", IfTrueArgs: []ValueID{"loop.index", "loop.total"}, IfFalse: "exit", IfFalseArgs: []ValueID{"loop.total"}},
			},
			{
				ID:     "body",
				Params: []ValueID{"body.index", "body.total"},
				Instrs: []Instr{
					{ID: "mul_square", Kind: OpMulI32, Result: "square", Type: TypeI32, Args: []ValueID{"body.index", "body.index"}},
					{ID: "add_total", Kind: OpAddI32, Result: "next.total", Type: TypeI32, Args: []ValueID{"body.total", "square"}},
					{ID: "inc_index", Kind: OpAddI32, Result: "next.index", Type: TypeI32, Args: []ValueID{"body.index", "one"}},
				},
				Term: Terminator{Kind: TermBranch, Target: "loop", Args: []ValueID{"next.index", "next.total"}},
			},
			{
				ID:     "exit",
				Params: []ValueID{"exit.total"},
				Term:   Terminator{Kind: TermReturn, Value: "exit.total"},
			},
		},
	}
}

func scalarProductLoopSSA(fn ir.IRFunc) Function {
	return Function{
		Name:       fn.Name,
		ReturnType: TypeI32,
		Values: []Value{
			{ID: "local0", Type: TypeI32, Origin: "param"},
			{ID: "zero", Type: TypeI32, Origin: "const"},
			{ID: "one", Type: TypeI32, Origin: "const"},
			{ID: "loop.index", Type: TypeI32, Origin: "block_param"},
			{ID: "loop.product", Type: TypeI32, Origin: "block_param"},
			{ID: "body.index", Type: TypeI32, Origin: "block_param"},
			{ID: "body.product", Type: TypeI32, Origin: "block_param"},
			{ID: "exit.product", Type: TypeI32, Origin: "block_param"},
			{ID: "cmp", Type: TypeBool, Origin: "instr"},
			{ID: "factor", Type: TypeI32, Origin: "instr"},
			{ID: "next.index", Type: TypeI32, Origin: "instr"},
			{ID: "next.product", Type: TypeI32, Origin: "instr"},
			{ID: "effect0", Type: TypeEffect, Origin: "entry_effect"},
		},
		Blocks: []Block{
			{
				ID:     "entry",
				Entry:  true,
				Instrs: []Instr{{ID: "const_zero", Kind: OpConstI32, Result: "zero", Type: TypeI32}, {ID: "const_one", Kind: OpConstI32, Result: "one", Type: TypeI32, Imm: 1}},
				Term:   Terminator{Kind: TermBranch, Target: "loop", Args: []ValueID{"zero", "one"}},
			},
			{
				ID:     "loop",
				Params: []ValueID{"loop.index", "loop.product"},
				Instrs: []Instr{{ID: "cmp", Kind: OpCmpLtI32, Result: "cmp", Type: TypeBool, Args: []ValueID{"loop.index", "local0"}}},
				Term:   Terminator{Kind: TermCondBr, Cond: "cmp", IfTrue: "body", IfTrueArgs: []ValueID{"loop.index", "loop.product"}, IfFalse: "exit", IfFalseArgs: []ValueID{"loop.product"}},
			},
			{
				ID:     "body",
				Params: []ValueID{"body.index", "body.product"},
				Instrs: []Instr{
					{ID: "add_factor", Kind: OpAddI32, Result: "factor", Type: TypeI32, Args: []ValueID{"body.index", "one"}},
					{ID: "mul_product", Kind: OpMulI32, Result: "next.product", Type: TypeI32, Args: []ValueID{"body.product", "factor"}},
					{ID: "inc_index", Kind: OpAddI32, Result: "next.index", Type: TypeI32, Args: []ValueID{"body.index", "one"}},
				},
				Term: Terminator{Kind: TermBranch, Target: "loop", Args: []ValueID{"next.index", "next.product"}},
			},
			{
				ID:     "exit",
				Params: []ValueID{"exit.product"},
				Term:   Terminator{Kind: TermReturn, Value: "exit.product"},
			},
		},
	}
}

func scalarMaxLoopSSA(fn ir.IRFunc) Function {
	return Function{
		Name:       fn.Name,
		ReturnType: TypeI32,
		Values: []Value{
			{ID: "local0", Type: TypeI32, Origin: "param"},
			{ID: "zero", Type: TypeI32, Origin: "const"},
			{ID: "one", Type: TypeI32, Origin: "const"},
			{ID: "loop.index", Type: TypeI32, Origin: "block_param"},
			{ID: "loop.max", Type: TypeI32, Origin: "block_param"},
			{ID: "body.index", Type: TypeI32, Origin: "block_param"},
			{ID: "body.max", Type: TypeI32, Origin: "block_param"},
			{ID: "update.index", Type: TypeI32, Origin: "block_param"},
			{ID: "update.max", Type: TypeI32, Origin: "block_param"},
			{ID: "keep.index", Type: TypeI32, Origin: "block_param"},
			{ID: "keep.max", Type: TypeI32, Origin: "block_param"},
			{ID: "exit.max", Type: TypeI32, Origin: "block_param"},
			{ID: "cmp", Type: TypeBool, Origin: "instr"},
			{ID: "max.cmp", Type: TypeBool, Origin: "instr"},
			{ID: "next.index", Type: TypeI32, Origin: "instr"},
			{ID: "effect0", Type: TypeEffect, Origin: "entry_effect"},
		},
		Blocks: []Block{
			{
				ID:     "entry",
				Entry:  true,
				Instrs: []Instr{{ID: "const_zero", Kind: OpConstI32, Result: "zero", Type: TypeI32}, {ID: "const_one", Kind: OpConstI32, Result: "one", Type: TypeI32, Imm: 1}},
				Term:   Terminator{Kind: TermBranch, Target: "loop", Args: []ValueID{"zero", "zero"}},
			},
			{
				ID:     "loop",
				Params: []ValueID{"loop.index", "loop.max"},
				Instrs: []Instr{{ID: "cmp", Kind: OpCmpLtI32, Result: "cmp", Type: TypeBool, Args: []ValueID{"loop.index", "local0"}}},
				Term:   Terminator{Kind: TermCondBr, Cond: "cmp", IfTrue: "body", IfTrueArgs: []ValueID{"loop.index", "loop.max"}, IfFalse: "exit", IfFalseArgs: []ValueID{"loop.max"}},
			},
			{
				ID:     "body",
				Params: []ValueID{"body.index", "body.max"},
				Instrs: []Instr{{ID: "cmp_max", Kind: OpCmpGtI32, Result: "max.cmp", Type: TypeBool, Args: []ValueID{"body.index", "body.max"}}},
				Term:   Terminator{Kind: TermCondBr, Cond: "max.cmp", IfTrue: "update", IfTrueArgs: []ValueID{"body.index", "body.max"}, IfFalse: "keep", IfFalseArgs: []ValueID{"body.index", "body.max"}},
			},
			{
				ID:     "update",
				Params: []ValueID{"update.index", "update.max"},
				Term:   Terminator{Kind: TermBranch, Target: "keep", Args: []ValueID{"update.index", "update.index"}},
			},
			{
				ID:     "keep",
				Params: []ValueID{"keep.index", "keep.max"},
				Instrs: []Instr{{ID: "inc_index", Kind: OpAddI32, Result: "next.index", Type: TypeI32, Args: []ValueID{"keep.index", "one"}}},
				Term:   Terminator{Kind: TermBranch, Target: "loop", Args: []ValueID{"next.index", "keep.max"}},
			},
			{
				ID:     "exit",
				Params: []ValueID{"exit.max"},
				Term:   Terminator{Kind: TermReturn, Value: "exit.max"},
			},
		},
	}
}

func scalarAffineLoopSSA(fn ir.IRFunc) Function {
	scale := fn.Instrs[11].Imm
	bias := fn.Instrs[13].Imm
	return Function{
		Name:       fn.Name,
		ReturnType: TypeI32,
		Values: []Value{
			{ID: "local0", Type: TypeI32, Origin: "param"},
			{ID: "zero", Type: TypeI32, Origin: "const"},
			{ID: "one", Type: TypeI32, Origin: "const"},
			{ID: "scale", Type: TypeI32, Origin: "const"},
			{ID: "bias", Type: TypeI32, Origin: "const"},
			{ID: "loop.index", Type: TypeI32, Origin: "block_param"},
			{ID: "loop.total", Type: TypeI32, Origin: "block_param"},
			{ID: "body.index", Type: TypeI32, Origin: "block_param"},
			{ID: "body.total", Type: TypeI32, Origin: "block_param"},
			{ID: "exit.total", Type: TypeI32, Origin: "block_param"},
			{ID: "cmp", Type: TypeBool, Origin: "instr"},
			{ID: "scaled", Type: TypeI32, Origin: "instr"},
			{ID: "affine", Type: TypeI32, Origin: "instr"},
			{ID: "next.index", Type: TypeI32, Origin: "instr"},
			{ID: "next.total", Type: TypeI32, Origin: "instr"},
			{ID: "effect0", Type: TypeEffect, Origin: "entry_effect"},
		},
		Blocks: []Block{
			{
				ID:    "entry",
				Entry: true,
				Instrs: []Instr{
					{ID: "const_zero", Kind: OpConstI32, Result: "zero", Type: TypeI32},
					{ID: "const_one", Kind: OpConstI32, Result: "one", Type: TypeI32, Imm: 1},
					{ID: "const_scale", Kind: OpConstI32, Result: "scale", Type: TypeI32, Imm: scale},
					{ID: "const_bias", Kind: OpConstI32, Result: "bias", Type: TypeI32, Imm: bias},
				},
				Term: Terminator{Kind: TermBranch, Target: "loop", Args: []ValueID{"zero", "zero"}},
			},
			{
				ID:     "loop",
				Params: []ValueID{"loop.index", "loop.total"},
				Instrs: []Instr{{ID: "cmp", Kind: OpCmpLtI32, Result: "cmp", Type: TypeBool, Args: []ValueID{"loop.index", "local0"}}},
				Term:   Terminator{Kind: TermCondBr, Cond: "cmp", IfTrue: "body", IfTrueArgs: []ValueID{"loop.index", "loop.total"}, IfFalse: "exit", IfFalseArgs: []ValueID{"loop.total"}},
			},
			{
				ID:     "body",
				Params: []ValueID{"body.index", "body.total"},
				Instrs: []Instr{
					{ID: "mul_scaled", Kind: OpMulI32, Result: "scaled", Type: TypeI32, Args: []ValueID{"body.index", "scale"}},
					{ID: "add_affine", Kind: OpAddI32, Result: "affine", Type: TypeI32, Args: []ValueID{"scaled", "bias"}},
					{ID: "add_total", Kind: OpAddI32, Result: "next.total", Type: TypeI32, Args: []ValueID{"body.total", "affine"}},
					{ID: "inc_index", Kind: OpAddI32, Result: "next.index", Type: TypeI32, Args: []ValueID{"body.index", "one"}},
				},
				Term: Terminator{Kind: TermBranch, Target: "loop", Args: []ValueID{"next.index", "next.total"}},
			},
			{
				ID:     "exit",
				Params: []ValueID{"exit.total"},
				Term:   Terminator{Kind: TermReturn, Value: "exit.total"},
			},
		},
	}
}

func scalarCountdownLoopSSA(fn ir.IRFunc) Function {
	return Function{
		Name:       fn.Name,
		ReturnType: TypeI32,
		Values: []Value{
			{ID: "local0", Type: TypeI32, Origin: "param"},
			{ID: "zero", Type: TypeI32, Origin: "const"},
			{ID: "one", Type: TypeI32, Origin: "const"},
			{ID: "loop.countdown", Type: TypeI32, Origin: "block_param"},
			{ID: "loop.total", Type: TypeI32, Origin: "block_param"},
			{ID: "body.countdown", Type: TypeI32, Origin: "block_param"},
			{ID: "body.total", Type: TypeI32, Origin: "block_param"},
			{ID: "exit.total", Type: TypeI32, Origin: "block_param"},
			{ID: "cmp", Type: TypeBool, Origin: "instr"},
			{ID: "next.total", Type: TypeI32, Origin: "instr"},
			{ID: "next.countdown", Type: TypeI32, Origin: "instr"},
			{ID: "effect0", Type: TypeEffect, Origin: "entry_effect"},
		},
		Blocks: []Block{
			{
				ID:     "entry",
				Entry:  true,
				Instrs: []Instr{{ID: "const_zero", Kind: OpConstI32, Result: "zero", Type: TypeI32}, {ID: "const_one", Kind: OpConstI32, Result: "one", Type: TypeI32, Imm: 1}},
				Term:   Terminator{Kind: TermBranch, Target: "loop", Args: []ValueID{"local0", "zero"}},
			},
			{
				ID:     "loop",
				Params: []ValueID{"loop.countdown", "loop.total"},
				Instrs: []Instr{{ID: "cmp", Kind: OpCmpGtI32, Result: "cmp", Type: TypeBool, Args: []ValueID{"loop.countdown", "zero"}}},
				Term:   Terminator{Kind: TermCondBr, Cond: "cmp", IfTrue: "body", IfTrueArgs: []ValueID{"loop.countdown", "loop.total"}, IfFalse: "exit", IfFalseArgs: []ValueID{"loop.total"}},
			},
			{
				ID:     "body",
				Params: []ValueID{"body.countdown", "body.total"},
				Instrs: []Instr{
					{ID: "add_total", Kind: OpAddI32, Result: "next.total", Type: TypeI32, Args: []ValueID{"body.total", "body.countdown"}},
					{ID: "dec_countdown", Kind: OpSubI32, Result: "next.countdown", Type: TypeI32, Args: []ValueID{"body.countdown", "one"}},
				},
				Term: Terminator{Kind: TermBranch, Target: "loop", Args: []ValueID{"next.countdown", "next.total"}},
			},
			{
				ID:     "exit",
				Params: []ValueID{"exit.total"},
				Term:   Terminator{Kind: TermReturn, Value: "exit.total"},
			},
		},
	}
}

func sliceSumLoopSSA(fn ir.IRFunc) Function {
	proofID := fn.Instrs[13].ProofID
	step := fn.Instrs[17].Imm
	stepValue := Value{ID: "one", Type: TypeI32, Origin: "const"}
	stepInstr := Instr{ID: "const_one", Kind: OpConstI32, Result: "one", Type: TypeI32, Imm: 1}
	stepID := ValueID("one")
	if step != 1 {
		stepValue = Value{ID: "step", Type: TypeI32, Origin: "const"}
		stepInstr = Instr{ID: "const_step", Kind: OpConstI32, Result: "step", Type: TypeI32, Imm: step}
		stepID = "step"
	}
	return Function{
		Name:       fn.Name,
		ReturnType: TypeI32,
		Values: []Value{
			{ID: "local0", Type: TypePtr, Origin: "param"},
			{ID: "local1", Type: TypeI32, Origin: "param"},
			{ID: "zero", Type: TypeI32, Origin: "const"},
			stepValue,
			{ID: "effect0", Type: TypeEffect, Origin: "entry_effect"},
			{ID: "loop.index", Type: TypeI32, Origin: "block_param"},
			{ID: "loop.total", Type: TypeI32, Origin: "block_param"},
			{ID: "loop.effect", Type: TypeEffect, Origin: "block_param"},
			{ID: "body.index", Type: TypeI32, Origin: "block_param"},
			{ID: "body.total", Type: TypeI32, Origin: "block_param"},
			{ID: "body.effect", Type: TypeEffect, Origin: "block_param"},
			{ID: "exit.total", Type: TypeI32, Origin: "block_param"},
			{ID: "cmp", Type: TypeBool, Origin: "instr"},
			{ID: "elem", Type: TypeI32, Origin: "instr"},
			{ID: "load.effect", Type: TypeEffect, Origin: "memory_effect"},
			{ID: "next.total", Type: TypeI32, Origin: "instr"},
			{ID: "next.index", Type: TypeI32, Origin: "instr"},
		},
		Blocks: []Block{
			{
				ID:     "entry",
				Entry:  true,
				Instrs: []Instr{{ID: "const_zero", Kind: OpConstI32, Result: "zero", Type: TypeI32}, stepInstr},
				Term:   Terminator{Kind: TermBranch, Target: "loop", Args: []ValueID{"zero", "zero", "effect0"}},
			},
			{
				ID:     "loop",
				Params: []ValueID{"loop.index", "loop.total", "loop.effect"},
				Instrs: []Instr{{ID: "cmp", Kind: OpCmpLtI32, Result: "cmp", Type: TypeBool, Args: []ValueID{"loop.index", "local1"}}},
				Term:   Terminator{Kind: TermCondBr, Cond: "cmp", IfTrue: "body", IfTrueArgs: []ValueID{"loop.index", "loop.total", "loop.effect"}, IfFalse: "exit", IfFalseArgs: []ValueID{"loop.total"}},
			},
			{
				ID:     "body",
				Params: []ValueID{"body.index", "body.total", "body.effect"},
				Instrs: []Instr{
					{ID: "index_load", Kind: OpIndexLoadI32, Result: "elem", Type: TypeI32, Args: []ValueID{"local0", "local1", "body.index"}, EffectIn: "body.effect", EffectOut: "load.effect", ProofID: proofID},
					{ID: "add_total", Kind: OpAddI32, Result: "next.total", Type: TypeI32, Args: []ValueID{"body.total", "elem"}},
					{ID: "inc_index", Kind: OpAddI32, Result: "next.index", Type: TypeI32, Args: []ValueID{"body.index", stepID}},
				},
				Term: Terminator{Kind: TermBranch, Target: "loop", Args: []ValueID{"next.index", "next.total", "load.effect"}},
			},
			{
				ID:     "exit",
				Params: []ValueID{"exit.total"},
				Term:   Terminator{Kind: TermReturn, Value: "exit.total"},
			},
		},
	}
}

func isScalarLoop(fn ir.IRFunc) bool {
	in := fn.Instrs
	return fn.ReturnSlots == 1 && fn.ParamSlots == 1 && fn.LocalSlots >= 3 && len(in) == 21 &&
		isConstStore(in[0], in[1], 0) && isConstStore(in[2], in[3], 0) &&
		in[4].Kind == ir.IRLabel && isLoad(in[5], in[1].Local) && isLoad(in[6], 0) &&
		in[7].Kind == ir.IRCmpLtI32 && in[8].Kind == ir.IRJmpIfZero &&
		isLoad(in[9], in[3].Local) && isLoad(in[10], in[1].Local) && in[11].Kind == ir.IRAddI32 &&
		isStore(in[12], in[3].Local) && isLoad(in[13], in[1].Local) && in[14].Kind == ir.IRConstI32 &&
		validScalarLoopStep(in[14].Imm) && in[15].Kind == ir.IRAddI32 && isStore(in[16], in[1].Local) &&
		in[17].Kind == ir.IRJmp && in[17].Label == in[4].Label && in[18].Kind == ir.IRLabel &&
		in[18].Label == in[8].Label && isLoad(in[19], in[3].Local) && in[20].Kind == ir.IRReturn
}

func scalarLoopStep(fn ir.IRFunc, withCall bool) int32 {
	if withCall {
		return 1
	}
	return fn.Instrs[14].Imm
}

func validScalarLoopStep(step int32) bool {
	return step >= 1 && step <= 127
}

func validScalarAffineConstant(value int32) bool {
	return value >= 1 && value <= 127
}

func isScalarSumSquaresLoop(fn ir.IRFunc) bool {
	in := fn.Instrs
	return fn.ReturnSlots == 1 && fn.ParamSlots == 1 && fn.LocalSlots >= 3 && len(in) == 23 &&
		isConstStore(in[0], in[1], 0) && isConstStore(in[2], in[3], 0) &&
		in[4].Kind == ir.IRLabel && isLoad(in[5], in[1].Local) && isLoad(in[6], 0) &&
		in[7].Kind == ir.IRCmpLtI32 && in[8].Kind == ir.IRJmpIfZero &&
		isLoad(in[9], in[3].Local) && isLoad(in[10], in[1].Local) &&
		isLoad(in[11], in[1].Local) && in[12].Kind == ir.IRMulI32 &&
		in[13].Kind == ir.IRAddI32 && isStore(in[14], in[3].Local) &&
		isLoad(in[15], in[1].Local) && in[16].Kind == ir.IRConstI32 &&
		in[16].Imm == 1 && in[17].Kind == ir.IRAddI32 && isStore(in[18], in[1].Local) &&
		in[19].Kind == ir.IRJmp && in[19].Label == in[4].Label && in[20].Kind == ir.IRLabel &&
		in[20].Label == in[8].Label && isLoad(in[21], in[3].Local) && in[22].Kind == ir.IRReturn
}

func isScalarProductLoop(fn ir.IRFunc) bool {
	in := fn.Instrs
	return fn.ReturnSlots == 1 && fn.ParamSlots == 1 && fn.LocalSlots >= 3 && len(in) == 23 &&
		isConstStore(in[0], in[1], 0) && isConstStore(in[2], in[3], 1) &&
		in[4].Kind == ir.IRLabel && isLoad(in[5], in[1].Local) && isLoad(in[6], 0) &&
		in[7].Kind == ir.IRCmpLtI32 && in[8].Kind == ir.IRJmpIfZero &&
		isLoad(in[9], in[3].Local) && isLoad(in[10], in[1].Local) &&
		in[11].Kind == ir.IRConstI32 && in[11].Imm == 1 && in[12].Kind == ir.IRAddI32 &&
		in[13].Kind == ir.IRMulI32 && isStore(in[14], in[3].Local) &&
		isLoad(in[15], in[1].Local) && in[16].Kind == ir.IRConstI32 &&
		in[16].Imm == 1 && in[17].Kind == ir.IRAddI32 && isStore(in[18], in[1].Local) &&
		in[19].Kind == ir.IRJmp && in[19].Label == in[4].Label && in[20].Kind == ir.IRLabel &&
		in[20].Label == in[8].Label && isLoad(in[21], in[3].Local) && in[22].Kind == ir.IRReturn
}

func isScalarMaxLoop(fn ir.IRFunc) bool {
	in := fn.Instrs
	return fn.ReturnSlots == 1 && fn.ParamSlots == 1 && fn.LocalSlots >= 3 && len(in) == 24 &&
		isConstStore(in[0], in[1], 0) && isConstStore(in[2], in[3], 0) &&
		in[1].Local != in[3].Local && in[1].Local != 0 && in[3].Local != 0 &&
		in[4].Kind == ir.IRLabel && isLoad(in[5], in[3].Local) && isLoad(in[6], 0) &&
		in[7].Kind == ir.IRCmpLtI32 && in[8].Kind == ir.IRJmpIfZero &&
		isLoad(in[9], in[3].Local) && isLoad(in[10], in[1].Local) &&
		in[11].Kind == ir.IRCmpGtI32 && in[12].Kind == ir.IRJmpIfZero &&
		isLoad(in[13], in[3].Local) && isStore(in[14], in[1].Local) &&
		in[15].Kind == ir.IRLabel && in[15].Label == in[12].Label &&
		isLoad(in[16], in[3].Local) && in[17].Kind == ir.IRConstI32 &&
		in[17].Imm == 1 && in[18].Kind == ir.IRAddI32 && isStore(in[19], in[3].Local) &&
		in[20].Kind == ir.IRJmp && in[20].Label == in[4].Label && in[21].Kind == ir.IRLabel &&
		in[21].Label == in[8].Label && isLoad(in[22], in[1].Local) && in[23].Kind == ir.IRReturn
}

func isScalarAffineLoop(fn ir.IRFunc) bool {
	in := fn.Instrs
	return fn.ReturnSlots == 1 && fn.ParamSlots == 1 && fn.LocalSlots >= 3 && len(in) == 25 &&
		isConstStore(in[0], in[1], 0) && isConstStore(in[2], in[3], 0) &&
		in[4].Kind == ir.IRLabel && isLoad(in[5], in[1].Local) && isLoad(in[6], 0) &&
		in[7].Kind == ir.IRCmpLtI32 && in[8].Kind == ir.IRJmpIfZero &&
		isLoad(in[9], in[3].Local) && isLoad(in[10], in[1].Local) &&
		in[11].Kind == ir.IRConstI32 && validScalarAffineConstant(in[11].Imm) &&
		in[12].Kind == ir.IRMulI32 && in[13].Kind == ir.IRConstI32 &&
		validScalarAffineConstant(in[13].Imm) && in[14].Kind == ir.IRAddI32 &&
		in[15].Kind == ir.IRAddI32 && isStore(in[16], in[3].Local) &&
		isLoad(in[17], in[1].Local) && in[18].Kind == ir.IRConstI32 &&
		in[18].Imm == 1 && in[19].Kind == ir.IRAddI32 && isStore(in[20], in[1].Local) &&
		in[21].Kind == ir.IRJmp && in[21].Label == in[4].Label && in[22].Kind == ir.IRLabel &&
		in[22].Label == in[8].Label && isLoad(in[23], in[3].Local) && in[24].Kind == ir.IRReturn
}

func isScalarCountdownLoop(fn ir.IRFunc) bool {
	in := fn.Instrs
	return fn.ReturnSlots == 1 && fn.ParamSlots == 1 && fn.LocalSlots >= 2 && len(in) == 19 &&
		isConstStore(in[0], in[1], 0) && in[1].Local > 0 && in[1].Local < fn.LocalSlots &&
		in[2].Kind == ir.IRLabel && isLoad(in[3], 0) && in[4].Kind == ir.IRConstI32 &&
		in[4].Imm == 0 && in[5].Kind == ir.IRCmpGtI32 && in[6].Kind == ir.IRJmpIfZero &&
		isLoad(in[7], in[1].Local) && isLoad(in[8], 0) && in[9].Kind == ir.IRAddI32 &&
		isStore(in[10], in[1].Local) && isLoad(in[11], 0) && in[12].Kind == ir.IRConstI32 &&
		in[12].Imm == 1 && in[13].Kind == ir.IRSubI32 && isStore(in[14], 0) &&
		in[15].Kind == ir.IRJmp && in[15].Label == in[2].Label && in[16].Kind == ir.IRLabel &&
		in[16].Label == in[6].Label && isLoad(in[17], in[1].Local) && in[18].Kind == ir.IRReturn
}

func isScalarCallLoop(fn ir.IRFunc) bool {
	in := fn.Instrs
	return fn.ReturnSlots == 1 && fn.ParamSlots == 1 && fn.LocalSlots >= 3 && len(in) == 22 &&
		isConstStore(in[0], in[1], 0) && isConstStore(in[2], in[3], 0) &&
		in[4].Kind == ir.IRLabel && isLoad(in[5], in[1].Local) && isLoad(in[6], 0) &&
		in[7].Kind == ir.IRCmpLtI32 && in[8].Kind == ir.IRJmpIfZero &&
		isLoad(in[9], in[3].Local) && isLoad(in[10], in[1].Local) &&
		in[11].Kind == ir.IRCall && in[11].Name != "" && in[11].ArgSlots == 1 && in[11].RetSlots == 1 &&
		in[12].Kind == ir.IRAddI32 && isStore(in[13], in[3].Local) &&
		isLoad(in[14], in[1].Local) && in[15].Kind == ir.IRConstI32 && in[15].Imm == 1 &&
		in[16].Kind == ir.IRAddI32 && isStore(in[17], in[1].Local) &&
		in[18].Kind == ir.IRJmp && in[18].Label == in[4].Label && in[19].Kind == ir.IRLabel &&
		in[19].Label == in[8].Label && isLoad(in[20], in[3].Local) && in[21].Kind == ir.IRReturn
}

func isScalarConstBoundTwoArgSuccessCallLoop(fn ir.IRFunc) bool {
	in := fn.Instrs
	return fn.ReturnSlots == 1 && fn.ParamSlots == 0 && fn.LocalSlots >= 2 && len(in) == 30 &&
		isConstStore(in[0], in[1], 0) && isConstStore(in[2], in[3], 0) &&
		in[1].Local != in[3].Local &&
		in[4].Kind == ir.IRLabel && isLoad(in[5], in[1].Local) &&
		in[6].Kind == ir.IRConstI32 && in[6].Imm > 0 &&
		in[7].Kind == ir.IRCmpLtI32 && in[8].Kind == ir.IRJmpIfZero &&
		isLoad(in[9], in[3].Local) && isLoad(in[10], in[1].Local) &&
		isLoad(in[11], in[3].Local) &&
		in[12].Kind == ir.IRCall && in[12].Name != "" && in[12].ArgSlots == 2 && in[12].RetSlots == 1 &&
		in[13].Kind == ir.IRAddI32 && isStore(in[14], in[3].Local) &&
		isLoad(in[15], in[1].Local) && in[16].Kind == ir.IRConstI32 && in[16].Imm == 1 &&
		in[17].Kind == ir.IRAddI32 && isStore(in[18], in[1].Local) &&
		in[19].Kind == ir.IRJmp && in[19].Label == in[4].Label &&
		in[20].Kind == ir.IRLabel && in[20].Label == in[8].Label &&
		isLoad(in[21], in[3].Local) && in[22].Kind == ir.IRConstI32 && in[22].Imm == 0 &&
		in[23].Kind == ir.IRCmpGeI32 && in[24].Kind == ir.IRJmpIfZero &&
		in[25].Kind == ir.IRConstI32 && in[25].Imm == 0 && in[26].Kind == ir.IRReturn &&
		in[27].Kind == ir.IRLabel && in[27].Label == in[24].Label &&
		in[28].Kind == ir.IRConstI32 && in[28].Imm == 1 && in[29].Kind == ir.IRReturn
}

func isSliceSumLoop(fn ir.IRFunc) bool {
	in := fn.Instrs
	return fn.ReturnSlots == 1 && fn.ParamSlots == 2 && fn.LocalSlots >= 4 && len(in) == 24 &&
		isConstStore(in[0], in[1], 0) && isConstStore(in[2], in[3], 0) &&
		in[4].Kind == ir.IRLabel && isLoad(in[5], in[3].Local) && isLoad(in[6], 1) &&
		in[7].Kind == ir.IRCmpLtI32 && in[8].Kind == ir.IRJmpIfZero &&
		isLoad(in[9], in[1].Local) && isLoad(in[10], 0) && isLoad(in[11], 1) &&
		isLoad(in[12], in[3].Local) && in[13].Kind == ir.IRIndexLoadI32Unchecked &&
		strings.HasPrefix(in[13].ProofID, "proof:while:") && in[14].Kind == ir.IRAddI32 &&
		isStore(in[15], in[1].Local) && isLoad(in[16], in[3].Local) &&
		in[17].Kind == ir.IRConstI32 && validScalarLoopStep(in[17].Imm) && in[18].Kind == ir.IRAddI32 &&
		isStore(in[19], in[3].Local) && in[20].Kind == ir.IRJmp && in[20].Label == in[4].Label &&
		in[21].Kind == ir.IRLabel && in[21].Label == in[8].Label && isLoad(in[22], in[1].Local) &&
		in[23].Kind == ir.IRReturn
}

func loopCallName(fn ir.IRFunc) string {
	if len(fn.Instrs) > 11 && fn.Instrs[11].Kind == ir.IRCall {
		return fn.Instrs[11].Name
	}
	return "callee"
}

func isConstStore(c ir.IRInstr, s ir.IRInstr, imm int32) bool {
	return c.Kind == ir.IRConstI32 && c.Imm == imm && s.Kind == ir.IRStoreLocal
}

func isLoad(instr ir.IRInstr, local int) bool {
	return instr.Kind == ir.IRLoadLocal && instr.Local == local
}

func isStore(instr ir.IRInstr, local int) bool {
	return instr.Kind == ir.IRStoreLocal && instr.Local == local
}

func isCompare(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32, ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return true
	default:
		return false
	}
}

func stackBinaryOp(kind ir.IRInstrKind) OpKind {
	switch kind {
	case ir.IRAddI32:
		return OpAddI32
	case ir.IRSubI32:
		return OpSubI32
	case ir.IRMulI32:
		return OpMulI32
	case ir.IRDivI32:
		return OpDivI32
	case ir.IRModI32:
		return OpModI32
	case ir.IRCmpEqI32:
		return OpCmpEqI32
	case ir.IRCmpLtI32:
		return OpCmpLtI32
	case ir.IRCmpGtI32:
		return OpCmpGtI32
	case ir.IRCmpGeI32:
		return OpCmpGeI32
	case ir.IRCmpLeI32:
		return OpCmpLeI32
	case ir.IRCmpNeI32:
		return OpCmpNeI32
	default:
		return OpOpaque
	}
}
