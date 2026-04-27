package lower

import (
	"fmt"

	"tetra_language/compiler/internal/ir"
)

// IR verifier invariants:
//   - every branch target names a label in the same function;
//   - all control-flow paths entering an instruction agree on stack height;
//   - each instruction has enough input stack slots and leaves a non-negative stack;
//   - local loads/stores reference slots inside IRFunc.LocalSlots;
//   - returns see exactly IRFunc.ReturnSlots values on the stack;
//   - calls declare non-negative argument and return slot counts.
func VerifyProgram(prog *ir.IRProgram) error {
	if prog == nil {
		return fmt.Errorf("ir verifier: missing program")
	}
	for _, fn := range prog.Funcs {
		if err := VerifyFunc(fn); err != nil {
			return err
		}
	}
	return nil
}

func VerifyFunc(fn ir.IRFunc) error {
	labels := make(map[int]int)
	for i, instr := range fn.Instrs {
		if instr.Kind != ir.IRLabel {
			continue
		}
		if _, exists := labels[instr.Label]; exists {
			return verifyError(fn, i, "duplicate label %d", instr.Label)
		}
		labels[instr.Label] = i
	}
	for i, instr := range fn.Instrs {
		switch instr.Kind {
		case ir.IRJmp, ir.IRJmpIfZero:
			if _, ok := labels[instr.Label]; !ok {
				return verifyError(fn, i, "unknown label %d", instr.Label)
			}
		case ir.IRLoadLocal, ir.IRStoreLocal:
			if instr.Local < 0 || instr.Local >= fn.LocalSlots {
				return verifyError(fn, i, "local slot %d out of bounds (locals=%d)", instr.Local, fn.LocalSlots)
			}
		case ir.IRCall:
			if instr.ArgSlots < 0 || instr.RetSlots < 0 {
				return verifyError(fn, i, "call %q has negative ABI slots args=%d rets=%d", instr.Name, instr.ArgSlots, instr.RetSlots)
			}
		}
	}

	if len(fn.Instrs) == 0 {
		return nil
	}

	heights := make([]int, len(fn.Instrs))
	seen := make([]bool, len(fn.Instrs))
	work := []stackState{{idx: 0, height: 0}}
	for len(work) > 0 {
		cur := work[len(work)-1]
		work = work[:len(work)-1]
		if cur.idx < 0 || cur.idx >= len(fn.Instrs) {
			if cur.height != 0 {
				return fmt.Errorf("ir verifier: %s falls off end with stack height %d", fn.Name, cur.height)
			}
			continue
		}
		if seen[cur.idx] {
			if heights[cur.idx] != cur.height {
				return verifyError(fn, cur.idx, "inconsistent stack height: got %d, previously %d", cur.height, heights[cur.idx])
			}
			continue
		}
		seen[cur.idx] = true
		heights[cur.idx] = cur.height

		instr := fn.Instrs[cur.idx]
		pop, push, known := stackEffect(instr)
		if !known {
			return verifyError(fn, cur.idx, "unknown instruction kind %d", instr.Kind)
		}
		if cur.height < pop {
			return verifyError(fn, cur.idx, "stack underflow: need %d slots, have %d", pop, cur.height)
		}
		nextHeight := cur.height - pop + push
		if nextHeight < 0 {
			return verifyError(fn, cur.idx, "negative stack height %d", nextHeight)
		}

		switch instr.Kind {
		case ir.IRReturn:
			if cur.height != fn.ReturnSlots {
				return verifyError(fn, cur.idx, "return expects %d stack slots, have %d", fn.ReturnSlots, cur.height)
			}
		case ir.IRJmp:
			work = append(work, stackState{idx: labels[instr.Label], height: nextHeight})
		case ir.IRJmpIfZero:
			work = append(work, stackState{idx: labels[instr.Label], height: nextHeight})
			work = append(work, stackState{idx: cur.idx + 1, height: nextHeight})
		default:
			work = append(work, stackState{idx: cur.idx + 1, height: nextHeight})
		}
	}

	return nil
}

type stackState struct {
	idx    int
	height int
}

func verifyError(fn ir.IRFunc, idx int, format string, args ...interface{}) error {
	return fmt.Errorf("ir verifier: %s instr %d: %s", fn.Name, idx, fmt.Sprintf(format, args...))
}

func stackEffect(instr ir.IRInstr) (pop int, push int, known bool) {
	switch instr.Kind {
	case ir.IRWrite:
		return 2, 0, true
	case ir.IRStrLit:
		return 0, 2, true
	case ir.IRConstI32, ir.IRLoadLocal, ir.IRLoadGlobal:
		return 0, 1, true
	case ir.IRStoreLocal, ir.IRStoreGlobal:
		return 1, 0, true
	case ir.IRAddI32, ir.IRSubI32, ir.IRCmpEqI32, ir.IRCmpLtI32,
		ir.IRMulI32, ir.IRDivI32, ir.IRModI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return 2, 1, true
	case ir.IRNegI32:
		return 1, 1, true
	case ir.IRCall:
		return instr.ArgSlots, instr.RetSlots, true
	case ir.IRLabel, ir.IRJmp:
		return 0, 0, true
	case ir.IRJmpIfZero:
		return 1, 0, true
	case ir.IRReturn:
		return 0, 0, true
	case ir.IRAllocBytes, ir.IRIslandNew:
		return 1, 1, true
	case ir.IRMakeSliceU8, ir.IRMakeSliceI32:
		return 1, 2, true
	case ir.IRIndexLoadI32, ir.IRIndexLoadU8:
		return 3, 1, true
	case ir.IRIndexStoreI32, ir.IRIndexStoreU8:
		return 4, 0, true
	case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceI32:
		return 2, 2, true
	case ir.IRIslandFree:
		return 1, 0, true
	case ir.IRCapIO, ir.IRCapMem, ir.IRSymAddr:
		return 0, 1, true
	case ir.IRMemReadI32, ir.IRMemReadU8, ir.IRMemReadPtr, ir.IRMmioReadI32:
		return 2, 1, true
	case ir.IRMemWriteI32, ir.IRMemWriteU8, ir.IRMemWritePtr, ir.IRPtrAdd,
		ir.IRMmioWriteI32, ir.IRCtxSwitch:
		return 3, 1, true
	default:
		return 0, 0, false
	}
}
