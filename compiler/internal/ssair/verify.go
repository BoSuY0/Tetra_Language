package ssair

import "fmt"

func VerifyProgram(prog *Program) error {
	if prog == nil {
		return fmt.Errorf("ssa verifier: missing program")
	}
	for _, fn := range prog.Funcs {
		if err := VerifyFunction(fn); err != nil {
			return err
		}
	}
	return nil
}

func VerifyFunction(fn Function) error {
	if fn.Name == "" {
		return fmt.Errorf("ssa verifier: function name is empty")
	}
	if fn.ReturnType == "" {
		return fmt.Errorf("ssa verifier: %s return type is empty", fn.Name)
	}
	values := map[ValueID]Type{}
	for _, value := range fn.Values {
		if value.ID == "" {
			return fmt.Errorf("ssa verifier: %s has value with empty id", fn.Name)
		}
		if value.Type == "" {
			return fmt.Errorf("ssa verifier: %s value %q has empty type", fn.Name, value.ID)
		}
		if _, exists := values[value.ID]; exists {
			return fmt.Errorf("ssa verifier: %s duplicate value %q", fn.Name, value.ID)
		}
		values[value.ID] = value.Type
	}
	if len(fn.Blocks) == 0 {
		return fmt.Errorf("ssa verifier: %s has no blocks", fn.Name)
	}
	blocks := map[string]Block{}
	entryCount := 0
	for _, block := range fn.Blocks {
		if block.ID == "" {
			return fmt.Errorf("ssa verifier: %s has block with empty id", fn.Name)
		}
		if _, exists := blocks[block.ID]; exists {
			return fmt.Errorf("ssa verifier: %s duplicate block %q", fn.Name, block.ID)
		}
		if block.Entry {
			entryCount++
		}
		blocks[block.ID] = block
		for _, param := range block.Params {
			if _, ok := values[param]; !ok {
				return fmt.Errorf(
					"ssa verifier: %s block %s param references unknown value %q",
					fn.Name,
					block.ID,
					param,
				)
			}
		}
	}
	if entryCount != 1 {
		return fmt.Errorf("ssa verifier: %s entry block count = %d, want 1", fn.Name, entryCount)
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if err := verifyInstr(fn.Name, block.ID, instr, values); err != nil {
				return err
			}
		}
		if err := verifyTerminator(
			fn.Name,
			block.ID,
			block.Term,
			fn.ReturnType,
			values,
			blocks,
		); err != nil {
			return err
		}
	}
	return nil
}

func verifyInstr(fnName string, blockID string, instr Instr, values map[ValueID]Type) error {
	if instr.ID == "" {
		return fmt.Errorf(
			"ssa verifier: %s block %s has instruction with empty id",
			fnName,
			blockID,
		)
	}
	if instr.Kind == "" {
		return fmt.Errorf("ssa verifier: %s instruction %s has empty kind", fnName, instr.ID)
	}
	if instr.Result != "" {
		if _, ok := values[instr.Result]; !ok {
			return fmt.Errorf(
				"ssa verifier: %s instruction %s result references unknown value %q",
				fnName,
				instr.ID,
				instr.Result,
			)
		}
	}
	for _, arg := range instr.Args {
		if _, ok := values[arg]; !ok {
			return fmt.Errorf(
				"ssa verifier: %s instruction %s references unknown value %q",
				fnName,
				instr.ID,
				arg,
			)
		}
	}
	switch instr.Kind {
	case OpCall:
		if instr.Call == "" {
			return fmt.Errorf(
				"ssa verifier: %s instruction %s call target is empty",
				fnName,
				instr.ID,
			)
		}
		if instr.EffectIn == "" || instr.EffectOut == "" {
			return fmt.Errorf(
				"ssa verifier: %s instruction %s call effect tokens are required",
				fnName,
				instr.ID,
			)
		}
		if err := verifyEffectValue(fnName, instr.ID, "effect_in", instr.EffectIn, values); err != nil {
			return err
		}
		if err := verifyEffectValue(fnName, instr.ID, "effect_out", instr.EffectOut, values); err != nil {
			return err
		}
	case OpIndexLoadI32:
		if instr.EffectIn != "" {
			if err := verifyEffectValue(fnName, instr.ID, "effect_in", instr.EffectIn, values); err != nil {
				return err
			}
		}
		if instr.EffectOut != "" {
			if err := verifyEffectValue(
				fnName,
				instr.ID,
				"effect_out",
				instr.EffectOut,
				values,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func verifyEffectValue(
	fnName string,
	instrID string,
	label string,
	id ValueID,
	values map[ValueID]Type,
) error {
	typ, ok := values[id]
	if !ok {
		return fmt.Errorf(
			"ssa verifier: %s instruction %s %s references unknown value %q",
			fnName,
			instrID,
			label,
			id,
		)
	}
	if typ != TypeEffect {
		return fmt.Errorf(
			"ssa verifier: %s instruction %s %s value %q has type %s, want effect",
			fnName,
			instrID,
			label,
			id,
			typ,
		)
	}
	return nil
}

func verifyTerminator(
	fnName string,
	blockID string,
	term Terminator,
	returnType Type,
	values map[ValueID]Type,
	blocks map[string]Block,
) error {
	switch term.Kind {
	case TermReturn:
		if returnType == TypeVoid {
			return nil
		}
		typ, ok := values[term.Value]
		if !ok {
			return fmt.Errorf(
				"ssa verifier: %s block %s return references unknown value %q",
				fnName,
				blockID,
				term.Value,
			)
		}
		if typ != returnType {
			return fmt.Errorf(
				"ssa verifier: %s block %s return value %q has type %s, want %s",
				fnName,
				blockID,
				term.Value,
				typ,
				returnType,
			)
		}
	case TermBranch:
		return verifyBranchTarget(fnName, blockID, term.Target, term.Args, values, blocks)
	case TermCondBr:
		typ, ok := values[term.Cond]
		if !ok {
			return fmt.Errorf(
				"ssa verifier: %s block %s cond branch references unknown value %q",
				fnName,
				blockID,
				term.Cond,
			)
		}
		if typ != TypeBool && typ != TypeI32 {
			return fmt.Errorf(
				"ssa verifier: %s block %s cond value %q has type %s, want bool/i32",
				fnName,
				blockID,
				term.Cond,
				typ,
			)
		}
		if err := verifyBranchTarget(
			fnName,
			blockID,
			term.IfTrue,
			term.IfTrueArgs,
			values,
			blocks,
		); err != nil {
			return err
		}
		if err := verifyBranchTarget(
			fnName,
			blockID,
			term.IfFalse,
			term.IfFalseArgs,
			values,
			blocks,
		); err != nil {
			return err
		}
	default:
		return fmt.Errorf(
			"ssa verifier: %s block %s has invalid terminator %q",
			fnName,
			blockID,
			term.Kind,
		)
	}
	return nil
}

func verifyBranchTarget(
	fnName string,
	blockID string,
	target string,
	args []ValueID,
	values map[ValueID]Type,
	blocks map[string]Block,
) error {
	targetBlock, ok := blocks[target]
	if !ok {
		return fmt.Errorf(
			"ssa verifier: %s block %s branches to unknown block %q",
			fnName,
			blockID,
			target,
		)
	}
	if len(args) != len(targetBlock.Params) {
		return fmt.Errorf(
			"ssa verifier: %s block %s branch to %s passes %d args, want %d",
			fnName,
			blockID,
			target,
			len(args),
			len(targetBlock.Params),
		)
	}
	for i, arg := range args {
		argType, ok := values[arg]
		if !ok {
			return fmt.Errorf(
				"ssa verifier: %s block %s branch references unknown value %q",
				fnName,
				blockID,
				arg,
			)
		}
		paramType := values[targetBlock.Params[i]]
		if argType != paramType {
			return fmt.Errorf(
				"ssa verifier: %s block %s branch arg %q has type %s, target param %q has type %s",
				fnName,
				blockID,
				arg,
				argType,
				targetBlock.Params[i],
				paramType,
			)
		}
	}
	return nil
}
