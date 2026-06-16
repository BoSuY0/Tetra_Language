package tasks

import (
	"fmt"
	"hash/fnv"
	"sort"

	"tetra_language/compiler/actorwire"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

type Wrapper struct {
	Name              string
	Target            string
	Module            string
	ErrorType         string
	TargetThrowsType  string
	SlotCount         int
	StatusSlot        int
	TargetReturnSlots int
}

type StagedTarget struct {
	SlotCount int
	ErrorType string
}

type UnsupportedFunc func(frontend.Position, string, ...interface{}) error

func WrapperName(target, errorType string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(target))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(errorType))
	return fmt.Sprintf("__tetra_task_typed_%08x", h.Sum32())
}

func ActorMessageTagBase(typeName string) int32 {
	return actorwire.TypedMessageTagBase(typeName)
}

func CollectWrappers(checked *semantics.CheckedProgram, module string) []Wrapper {
	if checked == nil {
		return nil
	}
	targetModules := make(map[string]string, len(checked.Funcs))
	targetReturnSlots := make(map[string]int, len(checked.FuncSigs))
	targetThrowsTypes := make(map[string]string, len(checked.FuncSigs))
	for _, fn := range checked.Funcs {
		targetModules[fn.Name] = fn.Module
	}
	for name, sig := range checked.FuncSigs {
		targetReturnSlots[name] = sig.ReturnSlots
		targetThrowsTypes[name] = sig.ThrowsType
	}
	seen := make(map[string]Wrapper)

	var walkExpr func(frontend.Expr)
	var walkStmt func(frontend.Stmt)
	addCall := func(call *frontend.CallExpr, workerArg int) {
		if len(call.TypeArgs) != 1 || call.TypeArgs[0].Name == "" || len(call.Args) <= workerArg {
			return
		}
		lit, ok := call.Args[workerArg].(*frontend.StringLitExpr)
		if !ok || string(lit.Value) == "" {
			return
		}
		target := string(lit.Value)
		targetModule, targetOK := targetModules[target]
		if !targetOK || (module != "" && targetModule != module) {
			return
		}
		_, handleInfo, err := semantics.EnsureTypedTaskHandleType(call.TypeArgs[0].Name, checked.Types)
		if err != nil {
			return
		}
		name := WrapperName(target, call.TypeArgs[0].Name)
		targetSlots := targetReturnSlots[target]
		if handleInfo.SlotCount > 4 {
			targetSlots = 1
		}
		seen[name] = Wrapper{
			Name:              name,
			Target:            target,
			Module:            targetModule,
			ErrorType:         call.TypeArgs[0].Name,
			TargetThrowsType:  targetThrowsTypes[target],
			SlotCount:         handleInfo.SlotCount,
			StatusSlot:        handleInfo.SlotCount - 1,
			TargetReturnSlots: targetSlots,
		}
	}

	walkExpr = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.CallExpr:
			name := e.Name
			if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
				name = builtin
			}
			switch name {
			case "core.task_spawn_i32_typed":
				addCall(e, 0)
			case "core.task_spawn_group_i32_typed":
				addCall(e, 1)
			}
			for _, arg := range e.Args {
				walkExpr(arg)
			}
		case *frontend.StructLitExpr:
			for _, field := range e.Fields {
				walkExpr(field.Value)
			}
		case *frontend.FieldAccessExpr:
			walkExpr(e.Base)
		case *frontend.IndexExpr:
			walkExpr(e.Base)
			walkExpr(e.Index)
		case *frontend.BinaryExpr:
			walkExpr(e.Left)
			walkExpr(e.Right)
		case *frontend.UnaryExpr:
			walkExpr(e.X)
		case *frontend.TryExpr:
			walkExpr(e.X)
		case *frontend.MatchExpr:
			walkExpr(e.Value)
			for _, c := range e.Cases {
				if c.Pattern != nil {
					walkExpr(c.Pattern)
				}
				if c.Guard != nil {
					walkExpr(c.Guard)
				}
				walkExpr(c.Value)
			}
		case *frontend.CatchExpr:
			walkExpr(e.Call)
			for _, c := range e.Cases {
				if c.Pattern != nil {
					walkExpr(c.Pattern)
				}
				if c.Guard != nil {
					walkExpr(c.Guard)
				}
				walkExpr(c.Value)
			}
		}
	}

	walkStmt = func(stmt frontend.Stmt) {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			walkExpr(s.Value)
		case *frontend.ReturnStmt:
			walkExpr(s.Value)
		case *frontend.ThrowStmt:
			walkExpr(s.Value)
		case *frontend.DeferStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.LetStmt:
			walkExpr(s.Value)
		case *frontend.AssignStmt:
			walkExpr(s.Target)
			walkExpr(s.Value)
		case *frontend.IfStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Then {
				walkStmt(inner)
			}
			for _, inner := range s.Else {
				walkStmt(inner)
			}
		case *frontend.IfLetStmt:
			walkExpr(s.Value)
			if s.Pattern != nil {
				walkExpr(s.Pattern)
			}
			for _, inner := range s.Then {
				walkStmt(inner)
			}
			for _, inner := range s.Else {
				walkStmt(inner)
			}
		case *frontend.WhileStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				walkExpr(s.Iterable)
			} else {
				walkExpr(s.Start)
				walkExpr(s.End)
			}
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.MatchStmt:
			walkExpr(s.Value)
			for _, c := range s.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				for _, inner := range c.Body {
					walkStmt(inner)
				}
			}
		case *frontend.FreeStmt:
			walkExpr(s.Value)
		case *frontend.UnsafeStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.IslandStmt:
			walkExpr(s.Size)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.ExprStmt:
			walkExpr(s.Expr)
		}
	}

	for _, fn := range checked.Funcs {
		if fn.Decl == nil {
			continue
		}
		for _, stmt := range fn.Decl.Body {
			walkStmt(stmt)
		}
	}

	out := make([]Wrapper, 0, len(seen))
	for _, wrapper := range seen {
		out = append(out, wrapper)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func CollectStagedTargets(wrappers []Wrapper) map[string]StagedTarget {
	if len(wrappers) == 0 {
		return nil
	}
	out := map[string]StagedTarget{}
	for _, wrapper := range wrappers {
		if wrapper.SlotCount <= 4 {
			continue
		}
		if wrapper.ErrorType == "" {
			continue
		}
		if wrapper.TargetThrowsType != wrapper.ErrorType {
			continue
		}
		out[wrapper.Target] = StagedTarget{SlotCount: wrapper.SlotCount, ErrorType: wrapper.ErrorType}
	}
	return out
}

func LowerWrapper(wrapper Wrapper, unsupported UnsupportedFunc) (ir.IRFunc, error) {
	if wrapper.SlotCount < 2 || wrapper.SlotCount > 8 {
		return ir.IRFunc{}, unsupportedError(unsupported, "typed task wrapper %s has unsupported slot count %d", wrapper.Name, wrapper.SlotCount)
	}
	discard := wrapper.SlotCount
	var instrs []ir.IRInstr
	if wrapper.SlotCount > 4 {
		if wrapper.TargetReturnSlots != 1 {
			return ir.IRFunc{}, unsupportedError(unsupported, "typed task wrapper %s staged mode requires a 1-slot target return, got %d", wrapper.Name, wrapper.TargetReturnSlots)
		}
		if wrapper.ErrorType != "" && wrapper.TargetThrowsType == wrapper.ErrorType {
			instrs = append(instrs, ir.IRInstr{Kind: ir.IRCall, Name: wrapper.Target, ArgSlots: 0, RetSlots: 1})
			instrs = append(instrs, ir.IRInstr{Kind: ir.IRReturn})
			return ir.IRFunc{
				Name:        wrapper.Name,
				ParamSlots:  0,
				LocalSlots:  0,
				ReturnSlots: 1,
				Instrs:      instrs,
			}, nil
		}
		instrs = append(instrs, ir.IRInstr{Kind: ir.IRCall, Name: wrapper.Target, ArgSlots: 0, RetSlots: 1})
		instrs = append(instrs, ir.IRInstr{Kind: ir.IRStoreLocal, Local: 0})
		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(wrapper.SlotCount)},
			ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_begin", ArgSlots: 1, RetSlots: 1},
			ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: 0},
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
			ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_slot", ArgSlots: 2, RetSlots: 1},
			ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard},
		)
		for slot := 1; slot < wrapper.SlotCount-1; slot++ {
			instrs = append(instrs,
				ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slot)},
				ir.IRInstr{Kind: ir.IRConstI32, Imm: 0},
				ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_slot", ArgSlots: 2, RetSlots: 1},
				ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard},
			)
		}
		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(wrapper.StatusSlot)},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: 0},
			ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_slot", ArgSlots: 2, RetSlots: 1},
			ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: 0},
			ir.IRInstr{Kind: ir.IRReturn},
		)
		return ir.IRFunc{
			Name:        wrapper.Name,
			ParamSlots:  0,
			LocalSlots:  wrapper.SlotCount + 1,
			ReturnSlots: 1,
			Instrs:      instrs,
		}, nil
	}
	instrs = append(instrs, ir.IRInstr{Kind: ir.IRCall, Name: wrapper.Target, ArgSlots: 0, RetSlots: wrapper.SlotCount})
	for slot := wrapper.SlotCount - 1; slot >= 0; slot-- {
		instrs = append(instrs, ir.IRInstr{Kind: ir.IRStoreLocal, Local: slot})
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(wrapper.SlotCount)},
		ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_begin", ArgSlots: 1, RetSlots: 1},
		ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard},
	)
	for slot := 0; slot < wrapper.SlotCount; slot++ {
		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slot)},
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: slot},
			ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_slot", ArgSlots: 2, RetSlots: 1},
			ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard},
		)
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: wrapper.StatusSlot},
		ir.IRInstr{Kind: ir.IRReturn},
	)
	return ir.IRFunc{
		Name:        wrapper.Name,
		ParamSlots:  0,
		LocalSlots:  wrapper.SlotCount + 1,
		ReturnSlots: 1,
		Instrs:      instrs,
	}, nil
}

func unsupportedError(unsupported UnsupportedFunc, format string, args ...interface{}) error {
	if unsupported != nil {
		return unsupported(frontend.Position{}, format, args...)
	}
	return fmt.Errorf(format, args...)
}

func CallExprWithName(call *frontend.CallExpr, name string) *frontend.CallExpr {
	if call == nil || call.Name == name {
		return call
	}
	clone := *call
	clone.Name = name
	return &clone
}

func CallExprWithBuiltinAlias(call *frontend.CallExpr) *frontend.CallExpr {
	if call == nil {
		return nil
	}
	if builtin, ok := semantics.ResolveBuiltinAlias(call.Name); ok {
		return CallExprWithName(call, builtin)
	}
	return call
}

func ThrowingLayout(returnType, throwsType string, types map[string]*semantics.TypeInfo) (int, int, bool, error) {
	if throwsType == "" {
		return 0, 0, false, nil
	}
	retInfo, ok := types[returnType]
	if !ok {
		return 0, 0, false, fmt.Errorf("unknown type '%s'", returnType)
	}
	throwInfo, ok := types[throwsType]
	if !ok {
		return 0, 0, false, fmt.Errorf("unknown type '%s'", throwsType)
	}
	compact := retInfo.SlotCount == 1 && throwInfo.SlotCount == 1
	return retInfo.SlotCount, throwInfo.SlotCount, compact, nil
}

func ThrowingReturnSlotCount(successSlots, errorSlots int) int {
	if successSlots == 1 && errorSlots == 1 {
		return 2
	}
	return successSlots + errorSlots + 1
}

func IsTypedTaskJoinCall(name string) bool {
	return name == "core.task_join_i32_typed" || name == "core.task_join_group_i32_typed"
}

func TypedTaskJoinRuntimeSymbol(slotCount int) string {
	return fmt.Sprintf("__tetra_task_join_typed_%d", slotCount)
}

func IsThrowIntLike(typeName string) bool {
	switch typeName {
	case "i32", "u8", "c_int", "c_uint", "task.error":
		return true
	default:
		return semantics.IsILP32NativeScalarType(typeName)
	}
}
