package lower

import (
	"fmt"
	"hash/fnv"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

func Lower(checked *semantics.CheckedProgram) (*ir.IRProgram, error) {
	if checked == nil {
		return nil, fmt.Errorf("missing checked program")
	}
	if len(checked.Funcs) == 0 {
		return nil, fmt.Errorf("expected at least one function")
	}

	prog := ir.IRProgram{MainIndex: checked.MainIndex, MainName: checked.MainName}
	for _, fn := range checked.Funcs {
		irFunc, err := lowerCheckedFunc(fn, checked.Types, checked.FuncSigs, checked.GlobalsByModule[fn.Module])
		if err != nil {
			return nil, err
		}
		prog.Funcs = append(prog.Funcs, irFunc)
	}
	return &prog, nil
}

func LowerModule(checked *semantics.CheckedProgram, module string) ([]ir.IRFunc, error) {
	if checked == nil {
		return nil, fmt.Errorf("missing checked program")
	}
	var out []ir.IRFunc
	for _, fn := range checked.Funcs {
		if fn.Module != module {
			continue
		}
		irFunc, err := lowerCheckedFunc(fn, checked.Types, checked.FuncSigs, checked.GlobalsByModule[fn.Module])
		if err != nil {
			return nil, err
		}
		out = append(out, irFunc)
	}
	return out, nil
}

func LowerModules(checked *semantics.CheckedProgram) (map[string][]ir.IRFunc, error) {
	if checked == nil {
		return nil, fmt.Errorf("missing checked program")
	}
	modules := make(map[string][]ir.IRFunc)
	for _, fn := range checked.Funcs {
		irFunc, err := lowerCheckedFunc(fn, checked.Types, checked.FuncSigs, checked.GlobalsByModule[fn.Module])
		if err != nil {
			return nil, err
		}
		modules[fn.Module] = append(modules[fn.Module], irFunc)
	}
	return modules, nil
}

func lowerCheckedFunc(fn semantics.CheckedFunc, types map[string]*semantics.TypeInfo, funcs map[string]semantics.FuncSig, globals map[string]semantics.GlobalInfo) (ir.IRFunc, error) {
	l := &lowerer{
		locals:      fn.Locals,
		globals:     globals,
		types:       types,
		funcs:       funcs,
		returnType:  fn.ReturnType,
		throwsType:  fn.ThrowsType,
		returnSlots: fn.ReturnSlots,
	}
	for _, stmt := range fn.Decl.Body {
		if err := l.lowerStmt(stmt); err != nil {
			return ir.IRFunc{}, err
		}
	}
	return ir.IRFunc{
		Name:        fn.Name,
		ExportName:  fn.Decl.ExportName,
		ParamSlots:  fn.ParamSlots,
		LocalSlots:  fn.LocalSlots,
		ReturnSlots: fn.ReturnSlots,
		Instrs:      l.instrs,
	}, nil
}

type lowerer struct {
	instrs         []ir.IRInstr
	locals         map[string]semantics.LocalInfo
	globals        map[string]semantics.GlobalInfo
	types          map[string]*semantics.TypeInfo
	funcs          map[string]semantics.FuncSig
	returnType     string
	throwsType     string
	returnSlots    int
	nextLabel      int
	cleanupIslands []int
	loopStack      []loopLabels
}

type loopLabels struct {
	continueLabel int
	breakLabel    int
	cleanupDepth  int
}

func (l *lowerer) newLabel() int {
	id := l.nextLabel
	l.nextLabel++
	return id
}

func (l *lowerer) emit(instr ir.IRInstr) {
	l.instrs = append(l.instrs, instr)
}

func (l *lowerer) emitCleanup(pos frontend.Position) {
	l.emitCleanupSince(0, pos)
}

func (l *lowerer) emitCleanupSince(start int, pos frontend.Position) {
	for i := len(l.cleanupIslands) - 1; i >= 0; i-- {
		if i < start {
			break
		}
		base := l.cleanupIslands[i]
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRIslandFree, Pos: pos})
	}
}

func (l *lowerer) pushLoop(continueLabel, breakLabel int) {
	l.loopStack = append(l.loopStack, loopLabels{
		continueLabel: continueLabel,
		breakLabel:    breakLabel,
		cleanupDepth:  len(l.cleanupIslands),
	})
}

func (l *lowerer) popLoop() {
	l.loopStack = l.loopStack[:len(l.loopStack)-1]
}

func (l *lowerer) currentLoop() (loopLabels, bool) {
	if len(l.loopStack) == 0 {
		return loopLabels{}, false
	}
	return l.loopStack[len(l.loopStack)-1], true
}

func (l *lowerer) lowerStmt(stmt frontend.Stmt) error {
	switch s := stmt.(type) {
	case *frontend.PrintStmt:
		slots, err := l.lowerExpr(s.Value)
		if err != nil {
			return err
		}
		if slots != 2 {
			return fmt.Errorf("%s: print expects str or []u8", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRWrite, Pos: s.At})
	case *frontend.FreeStmt:
		slots, err := l.lowerExpr(s.Value)
		if err != nil {
			return err
		}
		if slots != 1 {
			return fmt.Errorf("%s: free expects island (1 slot)", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRIslandFree, Pos: s.At})
	case *frontend.ReturnStmt:
		slots, err := l.lowerExprAs(s.Value, l.returnType)
		if err != nil {
			return err
		}
		expectedSlots := l.returnSlots
		if l.throwsType != "" {
			expectedSlots = 1
		}
		if slots != expectedSlots {
			return fmt.Errorf("%s: return slot mismatch", frontend.FormatPos(s.At))
		}
		if l.throwsType != "" {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: s.At})
		}
		l.emitCleanup(s.At)
		l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: s.At})
	case *frontend.ThrowStmt:
		if l.throwsType == "" {
			return fmt.Errorf("%s: throw is only allowed in throwing functions", frontend.FormatPos(s.At))
		}
		slots, err := l.lowerExprAs(s.Value, l.throwsType)
		if err != nil {
			return err
		}
		if slots != 1 {
			return fmt.Errorf("%s: throw slot mismatch", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: s.At})
		l.emitCleanup(s.At)
		l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: s.At})
	case *frontend.BreakStmt:
		loop, ok := l.currentLoop()
		if !ok {
			return fmt.Errorf("%s: break outside loop", frontend.FormatPos(s.At))
		}
		l.emitCleanupSince(loop.cleanupDepth, s.At)
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: loop.breakLabel, Pos: s.At})
	case *frontend.ContinueStmt:
		loop, ok := l.currentLoop()
		if !ok {
			return fmt.Errorf("%s: continue outside loop", frontend.FormatPos(s.At))
		}
		l.emitCleanupSince(loop.cleanupDepth, s.At)
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: loop.continueLabel, Pos: s.At})
	case *frontend.IslandStmt:
		slots, err := l.lowerExpr(s.Size)
		if err != nil {
			return err
		}
		if slots != 1 {
			return fmt.Errorf("%s: island size must be i32", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRIslandNew, Pos: s.At})
		info, ok := l.locals[s.Name]
		if !ok {
			return fmt.Errorf("unknown local '%s'", s.Name)
		}
		if info.SlotCount != 1 {
			return fmt.Errorf("%s: island slot mismatch", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: info.Base, Pos: s.At})
		l.cleanupIslands = append(l.cleanupIslands, info.Base)
		for _, stmt := range s.Body {
			if err := l.lowerStmt(stmt); err != nil {
				return err
			}
		}
		l.cleanupIslands = l.cleanupIslands[:len(l.cleanupIslands)-1]
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRIslandFree, Pos: s.At})
	case *frontend.LetStmt:
		info, ok := l.locals[s.Name]
		if !ok {
			return fmt.Errorf("unknown local '%s'", s.Name)
		}
		slots, err := l.lowerExprAs(s.Value, info.TypeName)
		if err != nil {
			return err
		}
		if slots != info.SlotCount {
			return fmt.Errorf("%s: slot mismatch for '%s'", frontend.FormatPos(s.At), s.Name)
		}
		for i := info.SlotCount - 1; i >= 0; i-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: info.Base + i, Pos: s.At})
		}
	case *frontend.AssignStmt:
		if idx, ok := s.Target.(*frontend.IndexExpr); ok {
			elemType, err := l.indexElemType(idx.Base)
			if err != nil {
				return err
			}
			baseSlots, err := l.lowerExpr(idx.Base)
			if err != nil {
				return err
			}
			if baseSlots != 2 {
				return fmt.Errorf("%s: index base slot mismatch", frontend.FormatPos(idx.At))
			}
			idxSlots, err := l.lowerExpr(idx.Index)
			if err != nil {
				return err
			}
			if idxSlots != 1 {
				return fmt.Errorf("%s: index must be i32", frontend.FormatPos(idx.At))
			}
			valSlots, err := l.lowerExpr(s.Value)
			if err != nil {
				return err
			}
			if valSlots != 1 {
				return fmt.Errorf("%s: index assignment expects single-slot value", frontend.FormatPos(s.At))
			}
			switch elemType {
			case "i32":
				l.emit(ir.IRInstr{Kind: ir.IRIndexStoreI32, Pos: s.At})
			case "u8":
				l.emit(ir.IRInstr{Kind: ir.IRIndexStoreU8, Pos: s.At})
			default:
				return fmt.Errorf("%s: unsupported index element type '%s'", frontend.FormatPos(s.At), elemType)
			}
			return nil
		}
		if id, ok := s.Target.(*frontend.IdentExpr); ok {
			if g, ok := l.globals[id.Name]; ok {
				slots, err := l.lowerExprAs(s.Value, g.TypeName)
				if err != nil {
					return err
				}
				if slots != gSlotCount(g.TypeName, l.types) {
					return fmt.Errorf("%s: slot mismatch for assignment", frontend.FormatPos(s.At))
				}
				l.emit(ir.IRInstr{Kind: ir.IRStoreGlobal, Local: g.DataIndex, Pos: s.At})
				return nil
			}
		}
		target, err := l.resolveLValue(s.Target)
		if err != nil {
			return err
		}
		slots, err := l.lowerExprAs(s.Value, target.TypeName)
		if err != nil {
			return err
		}
		if slots != target.SlotCount {
			return fmt.Errorf("%s: slot mismatch for assignment", frontend.FormatPos(s.At))
		}
		for i := target.SlotCount - 1; i >= 0; i-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: target.Base + i, Pos: s.At})
		}
	case *frontend.IfStmt:
		elseLabel := l.newLabel()
		endLabel := -1
		if len(s.Else) > 0 {
			endLabel = l.newLabel()
		}
		slots, err := l.lowerExpr(s.Cond)
		if err != nil {
			return err
		}
		if slots != 1 {
			return fmt.Errorf("%s: condition must be i32", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: elseLabel, Pos: s.At})
		for _, stmt := range s.Then {
			if err := l.lowerStmt(stmt); err != nil {
				return err
			}
		}
		if len(s.Else) > 0 {
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: s.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: elseLabel, Pos: s.At})
		if len(s.Else) > 0 {
			for _, stmt := range s.Else {
				if err := l.lowerStmt(stmt); err != nil {
					return err
				}
			}
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
		}
	case *frontend.IfLetStmt:
		valueInfo, ok := l.locals[s.ValueLocal]
		if !ok {
			return fmt.Errorf("%s: unknown if-let value local", frontend.FormatPos(s.At))
		}
		bindInfo, ok := l.locals[s.Name]
		if !ok {
			return fmt.Errorf("%s: unknown if-let local '%s'", frontend.FormatPos(s.At), s.Name)
		}
		slots, err := l.lowerExpr(s.Value)
		if err != nil {
			return err
		}
		if slots != valueInfo.SlotCount {
			return fmt.Errorf("%s: if-let value slot mismatch", frontend.FormatPos(s.At))
		}
		for i := valueInfo.SlotCount - 1; i >= 0; i-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: valueInfo.Base + i, Pos: s.At})
		}
		elseLabel := l.newLabel()
		endLabel := -1
		if len(s.Else) > 0 {
			endLabel = l.newLabel()
		}
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueInfo.Base + bindInfo.SlotCount, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: elseLabel, Pos: s.At})
		for i := 0; i < bindInfo.SlotCount; i++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueInfo.Base + i, Pos: s.At})
		}
		for i := bindInfo.SlotCount - 1; i >= 0; i-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: bindInfo.Base + i, Pos: s.At})
		}
		for _, stmt := range s.Then {
			if err := l.lowerStmt(stmt); err != nil {
				return err
			}
		}
		if len(s.Else) > 0 {
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: s.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: elseLabel, Pos: s.At})
		if len(s.Else) > 0 {
			for _, stmt := range s.Else {
				if err := l.lowerStmt(stmt); err != nil {
					return err
				}
			}
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
		}
	case *frontend.WhileStmt:
		startLabel := l.newLabel()
		endLabel := l.newLabel()
		l.pushLoop(startLabel, endLabel)
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: startLabel, Pos: s.At})
		slots, err := l.lowerExpr(s.Cond)
		if err != nil {
			l.popLoop()
			return err
		}
		if slots != 1 {
			l.popLoop()
			return fmt.Errorf("%s: condition must be i32", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: endLabel, Pos: s.At})
		for _, stmt := range s.Body {
			if err := l.lowerStmt(stmt); err != nil {
				l.popLoop()
				return err
			}
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: startLabel, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
		l.popLoop()
	case *frontend.ForRangeStmt:
		loopInfo, ok := l.locals[s.Name]
		if !ok {
			return fmt.Errorf("%s: unknown for local '%s'", frontend.FormatPos(s.At), s.Name)
		}
		endInfo, ok := l.locals[s.EndLocal]
		if !ok {
			return fmt.Errorf("%s: unknown for end local", frontend.FormatPos(s.At))
		}
		if s.Iterable != nil {
			iterInfo, ok := l.locals[s.IterableLocal]
			if !ok {
				return fmt.Errorf("%s: unknown for iterable local", frontend.FormatPos(s.At))
			}
			indexInfo, ok := l.locals[s.IndexLocal]
			if !ok {
				return fmt.Errorf("%s: unknown for index local", frontend.FormatPos(s.At))
			}
			iterSlots, err := l.lowerExpr(s.Iterable)
			if err != nil {
				return err
			}
			if iterSlots != iterInfo.SlotCount || iterInfo.SlotCount != 2 {
				return fmt.Errorf("%s: for collection iterable slot mismatch", frontend.FormatPos(s.At))
			}
			for i := iterInfo.SlotCount - 1; i >= 0; i-- {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: iterInfo.Base + i, Pos: s.At})
			}
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: indexInfo.Base, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: iterInfo.Base + 1, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: endInfo.Base, Pos: s.At})
			startLabel := l.newLabel()
			continueLabel := l.newLabel()
			endLabel := l.newLabel()
			l.pushLoop(continueLabel, endLabel)
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: startLabel, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: indexInfo.Base, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: endInfo.Base, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRCmpLtI32, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: endLabel, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: iterInfo.Base, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: iterInfo.Base + 1, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: indexInfo.Base, Pos: s.At})
			switch loopInfo.TypeName {
			case "i32":
				l.emit(ir.IRInstr{Kind: ir.IRIndexLoadI32, Pos: s.At})
			case "u8":
				l.emit(ir.IRInstr{Kind: ir.IRIndexLoadU8, Pos: s.At})
			default:
				return fmt.Errorf("%s: unsupported for collection element type '%s'", frontend.FormatPos(s.At), loopInfo.TypeName)
			}
			if loopInfo.SlotCount != 1 {
				return fmt.Errorf("%s: for collection element slot mismatch", frontend.FormatPos(s.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: loopInfo.Base, Pos: s.At})
			for _, stmt := range s.Body {
				if err := l.lowerStmt(stmt); err != nil {
					l.popLoop()
					return err
				}
			}
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: continueLabel, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: indexInfo.Base, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRAddI32, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: indexInfo.Base, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: startLabel, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
			l.popLoop()
			return nil
		}
		startSlots, err := l.lowerExpr(s.Start)
		if err != nil {
			return err
		}
		if startSlots != 1 || loopInfo.SlotCount != 1 {
			return fmt.Errorf("%s: for range start slot mismatch", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: loopInfo.Base, Pos: s.At})
		endSlots, err := l.lowerExpr(s.End)
		if err != nil {
			return err
		}
		if endSlots != 1 || endInfo.SlotCount != 1 {
			return fmt.Errorf("%s: for range end slot mismatch", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: endInfo.Base, Pos: s.At})
		startLabel := l.newLabel()
		continueLabel := l.newLabel()
		endLabel := l.newLabel()
		l.pushLoop(continueLabel, endLabel)
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: startLabel, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: loopInfo.Base, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: endInfo.Base, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRCmpLtI32, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: endLabel, Pos: s.At})
		for _, stmt := range s.Body {
			if err := l.lowerStmt(stmt); err != nil {
				l.popLoop()
				return err
			}
		}
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: continueLabel, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: loopInfo.Base, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRAddI32, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: loopInfo.Base, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: startLabel, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
		l.popLoop()
	case *frontend.MatchStmt:
		info, ok := l.locals[s.ScrutineeLocal]
		if !ok {
			return fmt.Errorf("%s: unknown match scrutinee local", frontend.FormatPos(s.At))
		}
		valueSlots, err := l.lowerExpr(s.Value)
		if err != nil {
			return err
		}
		if valueSlots != info.SlotCount {
			return fmt.Errorf("%s: match value slot mismatch", frontend.FormatPos(s.At))
		}
		for i := info.SlotCount - 1; i >= 0; i-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: info.Base + i, Pos: s.At})
		}
		endLabel := l.newLabel()
		defaultLabel := -1
		caseLabels := make([]int, len(s.Cases))
		scrutTypeInfo, scrutTypeOK := l.types[info.TypeName]
		for i, c := range s.Cases {
			caseLabels[i] = l.newLabel()
			if c.Default {
				defaultLabel = caseLabels[i]
				continue
			}
			nextLabel := l.newLabel()
			if scrutTypeOK && scrutTypeInfo.Kind == semantics.TypeOptional {
				if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
					l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + info.SlotCount - 1, Pos: c.At})
					l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: c.At})
					l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: caseLabels[i], Pos: c.At})
					l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: c.At})
					continue
				}
				if !isNoneExpr(c.Pattern) {
					return fmt.Errorf("%s: optional match supports only 'none', 'some(name)', and '_' patterns in v0.7", frontend.FormatPos(c.At))
				}
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + info.SlotCount - 1, Pos: c.At})
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: c.At})
			} else {
				if info.SlotCount != 1 {
					return fmt.Errorf("%s: match value slot mismatch", frontend.FormatPos(s.At))
				}
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base, Pos: c.At})
				patSlots, err := l.lowerExpr(c.Pattern)
				if err != nil {
					return err
				}
				if patSlots != 1 {
					return fmt.Errorf("%s: match pattern slot mismatch", frontend.FormatPos(c.At))
				}
			}
			l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: c.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: c.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: caseLabels[i], Pos: c.At})
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: c.At})
		}
		if defaultLabel >= 0 {
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: defaultLabel, Pos: s.At})
		} else {
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: s.At})
		}
		for i, c := range s.Cases {
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: caseLabels[i], Pos: c.At})
			if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
				bindInfo, ok := l.locals[some.Name]
				if !ok {
					return fmt.Errorf("%s: unknown some binding '%s'", frontend.FormatPos(some.At), some.Name)
				}
				if bindInfo.SlotCount != info.SlotCount-1 {
					return fmt.Errorf("%s: optional some binding slot mismatch", frontend.FormatPos(some.At))
				}
				for slot := 0; slot < bindInfo.SlotCount; slot++ {
					l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + slot, Pos: some.At})
				}
				for slot := bindInfo.SlotCount - 1; slot >= 0; slot-- {
					l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: bindInfo.Base + slot, Pos: some.At})
				}
			}
			for _, stmt := range c.Body {
				if err := l.lowerStmt(stmt); err != nil {
					return err
				}
			}
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: c.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
	case *frontend.ExprStmt:
		slots, err := l.lowerExpr(s.Expr)
		if err != nil {
			return err
		}
		for i := 0; i < slots; i++ {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: 0, Pos: s.At})
		}
	case *frontend.UnsafeStmt:
		for _, stmt := range s.Body {
			if err := l.lowerStmt(stmt); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%s: unsupported statement", frontend.FormatPos(s.Pos()))
	}
	return nil
}

func (l *lowerer) lowerExpr(expr frontend.Expr) (int, error) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: e.Value, Pos: e.At})
		return 1, nil
	case *frontend.BoolLitExpr:
		if e.Value {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: e.At})
		} else {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		}
		return 1, nil
	case *frontend.NoneLitExpr:
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		return 2, nil
	case *frontend.StringLitExpr:
		l.emit(ir.IRInstr{Kind: ir.IRStrLit, Str: e.Value, Pos: e.At})
		return 2, nil
	case *frontend.IdentExpr:
		info, ok := l.locals[e.Name]
		if !ok {
			if g, ok := l.globals[e.Name]; ok {
				l.emit(ir.IRInstr{Kind: ir.IRLoadGlobal, Local: g.DataIndex, Pos: e.At})
				return 1, nil
			}
			return 0, fmt.Errorf("%s: unknown local '%s'", frontend.FormatPos(e.At), e.Name)
		}
		for i := 0; i < info.SlotCount; i++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + i, Pos: e.At})
		}
		return info.SlotCount, nil
	case *frontend.FieldAccessExpr:
		if e.EnumType != "" {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: e.EnumOrdinal, Pos: e.At})
			return 1, nil
		}
		target, err := l.resolveLValue(e)
		if err != nil {
			return 0, err
		}
		for i := 0; i < target.SlotCount; i++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: target.Base + i, Pos: e.At})
		}
		return target.SlotCount, nil
	case *frontend.IndexExpr:
		elemType, err := l.indexElemType(e.Base)
		if err != nil {
			return 0, err
		}
		baseSlots, err := l.lowerExpr(e.Base)
		if err != nil {
			return 0, err
		}
		if baseSlots != 2 {
			return 0, fmt.Errorf("%s: index base slot mismatch", frontend.FormatPos(e.At))
		}
		idxSlots, err := l.lowerExpr(e.Index)
		if err != nil {
			return 0, err
		}
		if idxSlots != 1 {
			return 0, fmt.Errorf("%s: index must be i32", frontend.FormatPos(e.At))
		}
		switch elemType {
		case "i32":
			l.emit(ir.IRInstr{Kind: ir.IRIndexLoadI32, Pos: e.At})
			return 1, nil
		case "u8":
			l.emit(ir.IRInstr{Kind: ir.IRIndexLoadU8, Pos: e.At})
			return 1, nil
		default:
			return 0, fmt.Errorf("%s: unsupported index element type '%s'", frontend.FormatPos(e.At), elemType)
		}
	case *frontend.StructLitExpr:
		info, ok := l.types[e.Type.Name]
		if !ok {
			return 0, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(e.At), e.Type.Name)
		}
		fieldMap := make(map[string]frontend.Expr, len(e.Fields))
		for _, field := range e.Fields {
			fieldMap[field.Name] = field.Value
		}
		total := 0
		for _, field := range info.Fields {
			expr, ok := fieldMap[field.Name]
			if !ok {
				return 0, fmt.Errorf("%s: missing field '%s'", frontend.FormatPos(e.At), field.Name)
			}
			slots, err := l.lowerExpr(expr)
			if err != nil {
				return 0, err
			}
			if slots != field.SlotCount {
				return 0, fmt.Errorf("%s: slot mismatch for field '%s'", frontend.FormatPos(e.At), field.Name)
			}
			total += slots
		}
		return total, nil
	case *frontend.TryExpr:
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			return 0, fmt.Errorf("%s: try expects a throwing function call", frontend.FormatPos(e.At))
		}
		slots, err := l.lowerExpr(call)
		if err != nil {
			return 0, err
		}
		if slots != 2 {
			return 0, fmt.Errorf("%s: try expects a two-slot throwing result", frontend.FormatPos(e.At))
		}
		okLabel := l.newLabel()
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: okLabel, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: e.At})
		l.emitCleanup(e.At)
		l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: e.At})
		// The x64 emitter tracks stack depth linearly. This unreachable padding
		// mirrors the success path payload that remains on the real stack at okLabel.
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: okLabel, Pos: e.At})
		return 1, nil
	case *frontend.AwaitExpr:
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			return 0, fmt.Errorf("%s: await expects an async function call", frontend.FormatPos(e.At))
		}
		return l.lowerExpr(call)
	case *frontend.CallExpr:
		if builtin, ok := semantics.ResolveBuiltinAlias(e.Name); ok {
			e.Name = builtin
		}
		switch e.Name {
		case "core.spawn":
			if len(e.Args) != 1 {
				return 0, fmt.Errorf("%s: spawn expects 1 argument", frontend.FormatPos(e.At))
			}
			lit, ok := e.Args[0].(*frontend.StringLitExpr)
			if !ok {
				return 0, fmt.Errorf("%s: spawn expects a string literal", frontend.FormatPos(e.At))
			}
			name := string(lit.Value)
			if name == "" {
				return 0, fmt.Errorf("%s: spawn expects a non-empty name", frontend.FormatPos(e.At))
			}
			h := fnv.New32a()
			_, _ = h.Write([]byte(name))
			id := int32(h.Sum32())
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: id, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_spawn", ArgSlots: 1, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.task_spawn_i32":
			if len(e.Args) != 1 {
				return 0, fmt.Errorf("%s: task_spawn_i32 expects 1 argument", frontend.FormatPos(e.At))
			}
			lit, ok := e.Args[0].(*frontend.StringLitExpr)
			if !ok {
				return 0, fmt.Errorf("%s: task_spawn_i32 expects a string literal", frontend.FormatPos(e.At))
			}
			name := string(lit.Value)
			if name == "" {
				return 0, fmt.Errorf("%s: task_spawn_i32 expects a non-empty name", frontend.FormatPos(e.At))
			}
			sig, ok := l.funcs[name]
			if !ok {
				return 0, fmt.Errorf("%s: unknown task target '%s'", frontend.FormatPos(e.At), name)
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: name, ArgSlots: 0, RetSlots: sig.ReturnSlots, Pos: e.At})
			return sig.ReturnSlots, nil
		case "core.recv":
			if len(e.Args) != 0 {
				return 0, fmt.Errorf("%s: recv expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_recv", ArgSlots: 0, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.self":
			if len(e.Args) != 0 {
				return 0, fmt.Errorf("%s: self expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_self", ArgSlots: 0, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.sender":
			if len(e.Args) != 0 {
				return 0, fmt.Errorf("%s: sender expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_sender", ArgSlots: 0, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.sym_addr":
			if len(e.Args) != 1 {
				return 0, fmt.Errorf("%s: sym_addr expects 1 argument", frontend.FormatPos(e.At))
			}
			lit, ok := e.Args[0].(*frontend.StringLitExpr)
			if !ok {
				return 0, fmt.Errorf("%s: sym_addr expects a string literal", frontend.FormatPos(e.At))
			}
			name := string(lit.Value)
			if name == "" {
				return 0, fmt.Errorf("%s: sym_addr expects a non-empty symbol name", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRSymAddr, Name: name, Pos: e.At})
			return 1, nil
		}
		total := 0
		for _, arg := range e.Args {
			slots, err := l.lowerExpr(arg)
			if err != nil {
				return 0, err
			}
			total += slots
		}
		switch e.Name {
		case "core.cap_io":
			if total != 0 {
				return 0, fmt.Errorf("%s: cap_io expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCapIO, Pos: e.At})
			return 1, nil
		case "core.cap_mem":
			if total != 0 {
				return 0, fmt.Errorf("%s: cap_mem expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCapMem, Pos: e.At})
			return 1, nil
		case "core.alloc_bytes":
			if total != 1 {
				return 0, fmt.Errorf("%s: alloc_bytes expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRAllocBytes, Pos: e.At})
			return 1, nil
		case "core.make_u8":
			if total != 1 {
				return 0, fmt.Errorf("%s: make_u8 expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMakeSliceU8, Pos: e.At})
			return 2, nil
		case "core.make_i32":
			if total != 1 {
				return 0, fmt.Errorf("%s: make_i32 expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMakeSliceI32, Pos: e.At})
			return 2, nil
		case "core.island_new":
			if total != 1 {
				return 0, fmt.Errorf("%s: island_new expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRIslandNew, Pos: e.At})
			return 1, nil
		case "core.island_make_u8":
			if total != 2 {
				return 0, fmt.Errorf("%s: island_make_u8 expects 2 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRIslandMakeSliceU8, Pos: e.At})
			return 2, nil
		case "core.island_make_i32":
			if total != 2 {
				return 0, fmt.Errorf("%s: island_make_i32 expects 2 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRIslandMakeSliceI32, Pos: e.At})
			return 2, nil
		case "core.mmio_read_i32":
			if total != 2 {
				return 0, fmt.Errorf("%s: mmio_read_i32 expects 2 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMmioReadI32, Pos: e.At})
			return 1, nil
		case "core.mmio_write_i32":
			if total != 3 {
				return 0, fmt.Errorf("%s: mmio_write_i32 expects 3 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMmioWriteI32, Pos: e.At})
			return 1, nil
		case "core.load_i32":
			if total != 2 {
				return 0, fmt.Errorf("%s: load_i32 expects 2 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMemReadI32, Pos: e.At})
			return 1, nil
		case "core.store_i32":
			if total != 3 {
				return 0, fmt.Errorf("%s: store_i32 expects 3 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMemWriteI32, Pos: e.At})
			return 1, nil
		case "core.load_u8":
			if total != 2 {
				return 0, fmt.Errorf("%s: load_u8 expects 2 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMemReadU8, Pos: e.At})
			return 1, nil
		case "core.store_u8":
			if total != 3 {
				return 0, fmt.Errorf("%s: store_u8 expects 3 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMemWriteU8, Pos: e.At})
			return 1, nil
		case "core.load_ptr":
			if total != 2 {
				return 0, fmt.Errorf("%s: load_ptr expects 2 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMemReadPtr, Pos: e.At})
			return 1, nil
		case "core.store_ptr":
			if total != 3 {
				return 0, fmt.Errorf("%s: store_ptr expects 3 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMemWritePtr, Pos: e.At})
			return 1, nil
		case "core.ptr_add":
			if total != 3 {
				return 0, fmt.Errorf("%s: ptr_add expects 3 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRPtrAdd, Pos: e.At})
			return 1, nil
		case "core.ctx_switch":
			if total != 3 {
				return 0, fmt.Errorf("%s: ctx_switch expects 3 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCtxSwitch, Pos: e.At})
			return 1, nil
		case "core.task_join_i32":
			if total != 1 {
				return 0, fmt.Errorf("%s: task_join_i32 expects 1 argument", frontend.FormatPos(e.At))
			}
			return 1, nil
		case "core.actor_dispatch":
			if total != 1 {
				return 0, fmt.Errorf("%s: actor_dispatch expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_dispatch", ArgSlots: 1, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.actor_main_entry_id":
			if total != 0 {
				return 0, fmt.Errorf("%s: actor_main_entry_id expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_main_entry_id", ArgSlots: 0, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.send":
			if total != 2 {
				return 0, fmt.Errorf("%s: send expects 2 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_send", ArgSlots: 2, RetSlots: 1, Pos: e.At})
			return 1, nil
		default:
			sig, ok := l.funcs[e.Name]
			if !ok {
				return 0, fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(e.At), e.Name)
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: e.Name, ArgSlots: total, RetSlots: sig.ReturnSlots, Pos: e.At})
			return sig.ReturnSlots, nil
		}
	case *frontend.ClosureExpr:
		l.emit(ir.IRInstr{Kind: ir.IRSymAddr, Name: e.Name, Pos: e.At})
		return 1, nil
	case *frontend.UnaryExpr:
		slots, err := l.lowerExpr(e.X)
		if err != nil {
			return 0, err
		}
		if slots != 1 {
			return 0, fmt.Errorf("%s: unary operand must be i32", frontend.FormatPos(e.At))
		}
		switch e.Op {
		case frontend.TokenMinus:
			l.emit(ir.IRInstr{Kind: ir.IRNegI32, Pos: e.At})
			return 1, nil
		case frontend.TokenBang:
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
			return 1, nil
		default:
			return 0, fmt.Errorf("%s: unsupported unary operator", frontend.FormatPos(e.At))
		}
	case *frontend.BinaryExpr:
		if (e.Op == frontend.TokenEqEq || e.Op == frontend.TokenBangEq) && (isNoneExpr(e.Left) || isNoneExpr(e.Right)) {
			var value frontend.Expr
			if isNoneExpr(e.Left) {
				value = e.Right
			} else {
				value = e.Left
			}
			if err := l.lowerOptionalTag(value); err != nil {
				return 0, err
			}
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
			if e.Op == frontend.TokenEqEq {
				l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
			} else {
				l.emit(ir.IRInstr{Kind: ir.IRCmpNeI32, Pos: e.At})
			}
			return 1, nil
		}
		// Short-circuit &&
		if e.Op == frontend.TokenAmpAmp {
			leftSlots, err := l.lowerExpr(e.Left)
			if err != nil {
				return 0, err
			}
			if leftSlots != 1 {
				return 0, fmt.Errorf("%s: && operand must be i32", frontend.FormatPos(e.At))
			}
			falseLabel := l.newLabel()
			endLabel := l.newLabel()
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: falseLabel, Pos: e.At})
			rightSlots, err := l.lowerExpr(e.Right)
			if err != nil {
				return 0, err
			}
			if rightSlots != 1 {
				return 0, fmt.Errorf("%s: && operand must be i32", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel})
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: falseLabel})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel})
			return 1, nil
		}

		// Short-circuit ||
		if e.Op == frontend.TokenPipePipe {
			leftSlots, err := l.lowerExpr(e.Left)
			if err != nil {
				return 0, err
			}
			if leftSlots != 1 {
				return 0, fmt.Errorf("%s: || operand must be i32", frontend.FormatPos(e.At))
			}
			tryRightLabel := l.newLabel()
			endLabel := l.newLabel()
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: tryRightLabel, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel})
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: tryRightLabel})
			rightSlots, err := l.lowerExpr(e.Right)
			if err != nil {
				return 0, err
			}
			if rightSlots != 1 {
				return 0, fmt.Errorf("%s: || operand must be i32", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel})
			return 1, nil
		}

		leftSlots, err := l.lowerExpr(e.Left)
		if err != nil {
			return 0, err
		}
		rightSlots, err := l.lowerExpr(e.Right)
		if err != nil {
			return 0, err
		}
		if leftSlots != 1 || rightSlots != 1 {
			return 0, fmt.Errorf("%s: binary operands must be i32", frontend.FormatPos(e.At))
		}
		switch e.Op {
		case frontend.TokenPlus:
			l.emit(ir.IRInstr{Kind: ir.IRAddI32, Pos: e.At})
		case frontend.TokenMinus:
			l.emit(ir.IRInstr{Kind: ir.IRSubI32, Pos: e.At})
		case frontend.TokenStar:
			l.emit(ir.IRInstr{Kind: ir.IRMulI32, Pos: e.At})
		case frontend.TokenSlash:
			l.emit(ir.IRInstr{Kind: ir.IRDivI32, Pos: e.At})
		case frontend.TokenPercent:
			l.emit(ir.IRInstr{Kind: ir.IRModI32, Pos: e.At})
		case frontend.TokenEqEq:
			l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
		case frontend.TokenBangEq:
			l.emit(ir.IRInstr{Kind: ir.IRCmpNeI32, Pos: e.At})
		case frontend.TokenLess:
			l.emit(ir.IRInstr{Kind: ir.IRCmpLtI32, Pos: e.At})
		case frontend.TokenLessEq:
			l.emit(ir.IRInstr{Kind: ir.IRCmpLeI32, Pos: e.At})
		case frontend.TokenGreater:
			l.emit(ir.IRInstr{Kind: ir.IRCmpGtI32, Pos: e.At})
		case frontend.TokenGreaterEq:
			l.emit(ir.IRInstr{Kind: ir.IRCmpGeI32, Pos: e.At})
		default:
			return 0, fmt.Errorf("%s: unsupported binary operator", frontend.FormatPos(e.At))
		}
		return 1, nil
	default:
		return 0, fmt.Errorf("%s: unsupported expression", frontend.FormatPos(expr.Pos()))
	}
}

func (l *lowerer) lowerExprAs(expr frontend.Expr, expectedType string) (int, error) {
	expectedInfo, ok := l.types[expectedType]
	if !ok || expectedInfo.Kind != semantics.TypeOptional {
		return l.lowerExpr(expr)
	}
	actualType, err := l.inferExprType(expr)
	if err != nil {
		return 0, err
	}
	if actualType == expectedType || actualType == "none" {
		return l.lowerExpr(expr)
	}
	if actualType != expectedInfo.ElemType {
		return l.lowerExpr(expr)
	}
	slots, err := l.lowerExpr(expr)
	if err != nil {
		return 0, err
	}
	if slots != expectedInfo.SlotCount-1 {
		return 0, fmt.Errorf("%s: optional payload slot mismatch", frontend.FormatPos(expr.Pos()))
	}
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: expr.Pos()})
	return expectedInfo.SlotCount, nil
}

func gSlotCount(typeName string, types map[string]*semantics.TypeInfo) int {
	if info, ok := types[typeName]; ok {
		return info.SlotCount
	}
	return 1
}

type lvalueInfo struct {
	Base      int
	SlotCount int
	TypeName  string
}

func (l *lowerer) resolveLValue(expr frontend.Expr) (lvalueInfo, error) {
	baseName, fields, pos, ok := splitFieldPathLower(expr)
	if !ok {
		return lvalueInfo{}, fmt.Errorf("%s: invalid assignment target", frontend.FormatPos(pos))
	}
	info, ok := l.locals[baseName]
	if !ok {
		return lvalueInfo{}, fmt.Errorf("%s: unknown local '%s'", frontend.FormatPos(pos), baseName)
	}
	targetType, slotCount, offset, err := resolveFieldChainLower(info.TypeName, info.Base, fields, l.types, pos)
	if err != nil {
		return lvalueInfo{}, err
	}
	if _, ok := l.types[targetType]; !ok {
		return lvalueInfo{}, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), targetType)
	}
	return lvalueInfo{Base: offset, SlotCount: slotCount, TypeName: targetType}, nil
}

func splitFieldPathLower(expr frontend.Expr) (string, []string, frontend.Position, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name, nil, e.At, true
	case *frontend.FieldAccessExpr:
		baseName, fields, pos, ok := splitFieldPathLower(e.Base)
		if !ok {
			return "", nil, pos, false
		}
		fields = append(fields, e.Field)
		return baseName, fields, e.At, true
	default:
		return "", nil, expr.Pos(), false
	}
}

func resolveFieldChainLower(typeName string, baseOffset int, fields []string, types map[string]*semantics.TypeInfo, pos frontend.Position) (string, int, int, error) {
	offset := baseOffset
	current := typeName
	for _, field := range fields {
		info, ok := types[current]
		if !ok {
			return "", 0, 0, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), current)
		}
		if info.Kind != semantics.TypeStruct && info.Kind != semantics.TypeSlice && info.Kind != semantics.TypeStr {
			return "", 0, 0, fmt.Errorf("%s: '%s' is not a struct", frontend.FormatPos(pos), current)
		}
		fieldInfo, ok := info.FieldMap[field]
		if !ok {
			return "", 0, 0, fmt.Errorf("%s: unknown field '%s'", frontend.FormatPos(pos), field)
		}
		offset += fieldInfo.Offset
		current = fieldInfo.TypeName
	}
	info, ok := types[current]
	if !ok {
		return "", 0, 0, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), current)
	}
	return current, info.SlotCount, offset, nil
}

func isNoneExpr(expr frontend.Expr) bool {
	_, ok := expr.(*frontend.NoneLitExpr)
	return ok
}

func (l *lowerer) lowerOptionalTag(expr frontend.Expr) error {
	if isNoneExpr(expr) {
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: expr.Pos()})
		return nil
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		info, ok := l.locals[e.Name]
		if !ok {
			return fmt.Errorf("%s: optional comparison to none requires a stored optional value", frontend.FormatPos(e.At))
		}
		typeInfo, ok := l.types[info.TypeName]
		if !ok || typeInfo.Kind != semantics.TypeOptional {
			return fmt.Errorf("%s: optional comparison to none requires optional value", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + typeInfo.SlotCount - 1, Pos: e.At})
		return nil
	case *frontend.FieldAccessExpr:
		target, err := l.resolveLValue(e)
		if err != nil {
			return err
		}
		tname, err := l.inferExprType(e)
		if err != nil {
			return err
		}
		typeInfo, ok := l.types[tname]
		if !ok || typeInfo.Kind != semantics.TypeOptional {
			return fmt.Errorf("%s: optional comparison to none requires optional value", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: target.Base + typeInfo.SlotCount - 1, Pos: e.At})
		return nil
	default:
		return fmt.Errorf("%s: optional comparison to none requires a stored optional value", frontend.FormatPos(expr.Pos()))
	}
}

func (l *lowerer) indexElemType(base frontend.Expr) (string, error) {
	baseType, err := l.inferExprType(base)
	if err != nil {
		return "", err
	}
	info, ok := l.types[baseType]
	if !ok {
		return "", fmt.Errorf("unknown type '%s'", baseType)
	}
	switch info.Kind {
	case semantics.TypeStr:
		return "u8", nil
	case semantics.TypeSlice:
		return info.ElemType, nil
	default:
		return "", fmt.Errorf("%s: cannot index '%s'", frontend.FormatPos(base.Pos()), baseType)
	}
}

func (l *lowerer) inferExprType(expr frontend.Expr) (string, error) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return "i32", nil
	case *frontend.BoolLitExpr:
		return "bool", nil
	case *frontend.NoneLitExpr:
		return "none", nil
	case *frontend.StringLitExpr:
		return "str", nil
	case *frontend.IdentExpr:
		info, ok := l.locals[e.Name]
		if !ok {
			if g, ok := l.globals[e.Name]; ok {
				return g.TypeName, nil
			}
			return "", fmt.Errorf("%s: unknown local '%s'", frontend.FormatPos(e.At), e.Name)
		}
		return info.TypeName, nil
	case *frontend.FieldAccessExpr:
		_, targetType, err := semantics.ResolveFieldAccessType(e, l.locals, l.types)
		if err != nil {
			return "", err
		}
		return targetType, nil
	case *frontend.IndexExpr:
		elem, err := l.indexElemType(e.Base)
		if err != nil {
			return "", err
		}
		return elem, nil
	case *frontend.StructLitExpr:
		return e.Type.Name, nil
	case *frontend.CallExpr:
		if builtin, ok := semantics.ResolveBuiltinAlias(e.Name); ok {
			e.Name = builtin
		}
		sig, ok := l.funcs[e.Name]
		if !ok {
			return "", fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(e.At), e.Name)
		}
		return sig.ReturnType, nil
	case *frontend.ClosureExpr:
		return "ptr", nil
	case *frontend.TryExpr:
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			return "", fmt.Errorf("%s: try expects a throwing function call", frontend.FormatPos(e.At))
		}
		if builtin, ok := semantics.ResolveBuiltinAlias(call.Name); ok {
			call.Name = builtin
		}
		sig, ok := l.funcs[call.Name]
		if !ok {
			return "", fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(call.At), call.Name)
		}
		return sig.ReturnType, nil
	case *frontend.AwaitExpr:
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			return "", fmt.Errorf("%s: await expects an async function call", frontend.FormatPos(e.At))
		}
		if builtin, ok := semantics.ResolveBuiltinAlias(call.Name); ok {
			call.Name = builtin
		}
		sig, ok := l.funcs[call.Name]
		if !ok {
			return "", fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(call.At), call.Name)
		}
		return sig.ReturnType, nil
	case *frontend.UnaryExpr:
		if e.Op == frontend.TokenBang {
			return "bool", nil
		}
		return "i32", nil
	case *frontend.BinaryExpr:
		return "i32", nil
	default:
		return "", fmt.Errorf("%s: unsupported expression", frontend.FormatPos(expr.Pos()))
	}
}
