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
	returnSlots    int
	nextLabel      int
	cleanupIslands []int
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
	for i := len(l.cleanupIslands) - 1; i >= 0; i-- {
		base := l.cleanupIslands[i]
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRIslandFree, Pos: pos})
	}
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
		slots, err := l.lowerExpr(s.Value)
		if err != nil {
			return err
		}
		if slots != l.returnSlots {
			return fmt.Errorf("%s: return slot mismatch", frontend.FormatPos(s.At))
		}
		l.emitCleanup(s.At)
		l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: s.At})
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
		slots, err := l.lowerExpr(s.Value)
		if err != nil {
			return err
		}
		info, ok := l.locals[s.Name]
		if !ok {
			return fmt.Errorf("unknown local '%s'", s.Name)
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
				slots, err := l.lowerExpr(s.Value)
				if err != nil {
					return err
				}
				if slots != 1 {
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
		slots, err := l.lowerExpr(s.Value)
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
	case *frontend.WhileStmt:
		startLabel := l.newLabel()
		endLabel := l.newLabel()
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: startLabel, Pos: s.At})
		slots, err := l.lowerExpr(s.Cond)
		if err != nil {
			return err
		}
		if slots != 1 {
			return fmt.Errorf("%s: condition must be i32", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: endLabel, Pos: s.At})
		for _, stmt := range s.Body {
			if err := l.lowerStmt(stmt); err != nil {
				return err
			}
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: startLabel, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
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
		default:
			return 0, fmt.Errorf("%s: unsupported unary operator", frontend.FormatPos(e.At))
		}
	case *frontend.BinaryExpr:
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
		case frontend.TokenEqEq:
			l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
		case frontend.TokenLess:
			l.emit(ir.IRInstr{Kind: ir.IRCmpLtI32, Pos: e.At})
		default:
			return 0, fmt.Errorf("%s: unsupported binary operator", frontend.FormatPos(e.At))
		}
		return 1, nil
	default:
		return 0, fmt.Errorf("%s: unsupported expression", frontend.FormatPos(expr.Pos()))
	}
}

type lvalueInfo struct {
	Base      int
	SlotCount int
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
	return lvalueInfo{Base: offset, SlotCount: slotCount}, nil
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
	case *frontend.UnaryExpr, *frontend.BinaryExpr:
		return "i32", nil
	default:
		return "", fmt.Errorf("%s: unsupported expression", frontend.FormatPos(expr.Pos()))
	}
}
