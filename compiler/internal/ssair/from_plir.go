package ssair

import (
	"strconv"
	"strings"

	"tetra_language/compiler/internal/plir"
)

func FromPLIR(prog *plir.Program) (*Program, error) {
	if prog == nil {
		return nil, nil
	}
	out := &Program{Funcs: make([]Function, 0, len(prog.Funcs))}
	for _, fn := range prog.Funcs {
		ssaFn, err := FromPLIRFunction(fn)
		if err != nil {
			return nil, err
		}
		out.Funcs = append(out.Funcs, ssaFn)
	}
	if err := VerifyProgram(out); err != nil {
		return nil, err
	}
	return out, nil
}

func FromPLIRFunction(fn plir.Function) (Function, error) {
	out := Function{
		Name:       fn.Name,
		ReturnType: TypeVoid,
		Values:     make([]Value, 0, len(fn.Values)+len(fn.Ops)+1),
	}
	known := map[ValueID]Type{}
	addValue := func(id ValueID, typ Type, origin string) {
		if id == "" {
			return
		}
		if _, ok := known[id]; ok {
			return
		}
		known[id] = typ
		out.Values = append(out.Values, Value{ID: id, Type: typ, Origin: origin})
	}
	for _, value := range fn.Values {
		addValue(ValueID(value.ID), typeFromPLIR(value.Type), "plir_value")
	}
	effect := ValueID("effect0")
	addValue(effect, TypeEffect, "entry_effect")
	nextEffect := 1
	entry := Block{ID: "entry", Entry: true}
	for _, op := range fn.Ops {
		for _, input := range op.Inputs {
			addValue(ValueID(input), TypeI32, "plir_input")
		}
		switch op.Kind {
		case plir.OpReturn:
			if len(op.Inputs) > 0 {
				ret := ValueID(op.Inputs[0])
				out.ReturnType = known[ret]
				entry.Term = Terminator{Kind: TermReturn, Value: ret}
			} else {
				out.ReturnType = TypeVoid
				entry.Term = Terminator{Kind: TermReturn}
			}
		case plir.OpCall, plir.OpActorSend:
			result := firstOutput(op)
			if result != "" {
				addValue(result, outputType(op, known), "plir_call_result")
			}
			effectOut := ValueID("effect" + strconv.Itoa(nextEffect))
			nextEffect++
			addValue(effectOut, TypeEffect, "call_effect")
			entry.Instrs = append(entry.Instrs, Instr{
				ID:        op.ID,
				Kind:      OpCall,
				Result:    result,
				Type:      known[result],
				Args:      valueIDs(op.Inputs),
				Call:      plirCallName(op.Note),
				EffectIn:  effect,
				EffectOut: effectOut,
				Note:      op.Note,
			})
			effect = effectOut
		case plir.OpIndexLoad:
			result := firstOutput(op)
			if result != "" {
				addValue(result, TypeI32, "plir_index_load")
			}
			effectOut := ValueID("effect" + strconv.Itoa(nextEffect))
			nextEffect++
			addValue(effectOut, TypeEffect, "memory_effect")
			entry.Instrs = append(entry.Instrs, Instr{
				ID:        op.ID,
				Kind:      OpIndexLoadI32,
				Result:    result,
				Type:      known[result],
				Args:      valueIDs(op.Inputs),
				EffectIn:  effect,
				EffectOut: effectOut,
				Note:      op.Note,
			})
			effect = effectOut
		default:
			result := firstOutput(op)
			if result != "" {
				addValue(result, outputType(op, known), "plir_output")
			}
			entry.Instrs = append(
				entry.Instrs,
				Instr{
					ID:     op.ID,
					Kind:   OpOpaque,
					Result: result,
					Type:   known[result],
					Args:   valueIDs(op.Inputs),
					Note:   op.Note,
				},
			)
		}
	}
	if entry.Term.Kind == TermInvalid {
		entry.Term = Terminator{Kind: TermReturn}
		out.ReturnType = TypeVoid
	}
	out.Blocks = []Block{entry}
	return out, VerifyFunction(out)
}

func firstOutput(op plir.Operation) ValueID {
	if len(op.Outputs) == 0 {
		return ""
	}
	return ValueID(op.Outputs[0])
}

func outputType(op plir.Operation, known map[ValueID]Type) Type {
	out := firstOutput(op)
	if out == "" {
		return TypeVoid
	}
	if typ, ok := known[out]; ok {
		return typ
	}
	return TypeI32
}

func valueIDs(values []string) []ValueID {
	out := make([]ValueID, 0, len(values))
	for _, value := range values {
		out = append(out, ValueID(value))
	}
	return out
}

func typeFromPLIR(typ string) Type {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "", "void":
		return TypeVoid
	case "int", "i32":
		return TypeI32
	case "bool":
		return TypeBool
	case "ptr", "rawptr", "rawptr<u8>":
		return TypePtr
	case "string":
		return TypeString
	default:
		if strings.HasPrefix(strings.ToLower(typ), "[]") {
			return TypePtr
		}
		return TypeI32
	}
}

func plirCallName(note string) string {
	trimmed := strings.TrimSpace(note)
	if trimmed == "" {
		return "plir.call"
	}
	return strings.Fields(trimmed)[0]
}
