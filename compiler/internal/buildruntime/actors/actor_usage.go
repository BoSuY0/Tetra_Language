package actors

import (
	"sort"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics"
)

func CollectActorEntries(checked *semantics.CheckedProgram) (bool, []string, int, error) {
	if checked == nil {
		return false, nil, 0, nil
	}
	used := false
	spawnCount := 0
	targets := make(map[string]struct{})

	var walkExpr func(frontend.Expr) error
	var walkStmt func(frontend.Stmt) error

	walkExpr = func(expr frontend.Expr) error {
		switch e := expr.(type) {
		case *frontend.CallExpr:
			name := e.Name
			if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
				name = builtin
			}
			switch name {
			case "core.spawn":
				used = true
				spawnCount++
				if len(e.Args) == 1 {
					if lit, ok := e.Args[0].(*frontend.StringLitExpr); ok {
						name := string(lit.Value)
						if name != "" {
							targets[name] = struct{}{}
						}
					}
				}
			case "core.spawn_remote":
				used = true
				if len(e.Args) == 2 {
					if lit, ok := e.Args[1].(*frontend.StringLitExpr); ok {
						name := string(lit.Value)
						if name != "" {
							targets[name] = struct{}{}
						}
					}
				}
			case "core.actor_node_connect", "core.actor_node_status":
				used = true
			case "core.task_spawn_i32":
				used = true
				spawnCount++
				if len(e.Args) == 1 {
					if lit, ok := e.Args[0].(*frontend.StringLitExpr); ok {
						name := string(lit.Value)
						if name != "" {
							targets[name] = struct{}{}
						}
					}
				}
			case "core.task_spawn_group_i32":
				used = true
				spawnCount++
				if len(e.Args) == 2 {
					if lit, ok := e.Args[1].(*frontend.StringLitExpr); ok {
						name := string(lit.Value)
						if name != "" {
							targets[name] = struct{}{}
						}
					}
				}
			case "core.task_spawn_i32_typed":
				used = true
				spawnCount++
				if len(e.TypeArgs) == 1 && e.TypeArgs[0].Name != "" && len(e.Args) == 1 {
					if lit, ok := e.Args[0].(*frontend.StringLitExpr); ok {
						name := string(lit.Value)
						if name != "" {
							targets[typedTaskRuntimeWrapperName(name, e.TypeArgs[0].Name)] = struct{}{}
						}
					}
				}
			case "core.task_spawn_group_i32_typed":
				used = true
				spawnCount++
				if len(e.TypeArgs) == 1 && e.TypeArgs[0].Name != "" && len(e.Args) == 2 {
					if lit, ok := e.Args[1].(*frontend.StringLitExpr); ok {
						name := string(lit.Value)
						if name != "" {
							targets[typedTaskRuntimeWrapperName(name, e.TypeArgs[0].Name)] = struct{}{}
						}
					}
				}
			case "core.task_group_open", "core.task_group_close", "core.task_group_cancel", "core.task_group_current", "core.task_group_status",
				"core.task_is_canceled", "core.task_checkpoint":
				used = true
			case "core.time_now_ms", "core.sleep_ms", "core.sleep_until", "core.deadline_ms", "core.timer_ready":
				used = true
			case "core.task_join_i32", "core.task_join_result_i32", "core.task_join_until_i32", "core.task_poll_i32", "core.select2_i32":
				used = true
			case "core.task_join_i32_typed", "core.task_join_group_i32_typed":
				used = true
			case "core.send", "core.send_msg", "core.send_typed", "core.recv", "core.recv_msg", "core.recv_poll", "core.recv_until", "core.recv_msg_until", "core.recv_typed", "core.self", "core.sender", "core.yield":
				used = true
			}
			for _, arg := range e.Args {
				if err := walkExpr(arg); err != nil {
					return err
				}
			}
		case *frontend.StructLitExpr:
			for _, field := range e.Fields {
				if err := walkExpr(field.Value); err != nil {
					return err
				}
			}
		case *frontend.FieldAccessExpr:
			return walkExpr(e.Base)
		case *frontend.IndexExpr:
			if err := walkExpr(e.Base); err != nil {
				return err
			}
			return walkExpr(e.Index)
		case *frontend.BinaryExpr:
			if err := walkExpr(e.Left); err != nil {
				return err
			}
			return walkExpr(e.Right)
		case *frontend.UnaryExpr:
			return walkExpr(e.X)
		case *frontend.TryExpr:
			return walkExpr(e.X)
		case *frontend.CatchExpr:
			if err := walkExpr(e.Call); err != nil {
				return err
			}
			for _, c := range e.Cases {
				if !c.Default {
					if err := walkExpr(c.Pattern); err != nil {
						return err
					}
				}
				if err := walkExpr(c.Guard); err != nil {
					return err
				}
				if err := walkExpr(c.Value); err != nil {
					return err
				}
			}
		case *frontend.MatchExpr:
			if err := walkExpr(e.Value); err != nil {
				return err
			}
			for _, c := range e.Cases {
				if !c.Default {
					if err := walkExpr(c.Pattern); err != nil {
						return err
					}
				}
				if err := walkExpr(c.Guard); err != nil {
					return err
				}
				if err := walkExpr(c.Value); err != nil {
					return err
				}
			}
		case *frontend.IdentExpr, *frontend.NumberExpr, *frontend.BoolLitExpr, *frontend.StringLitExpr:
			return nil
		default:
			return nil
		}
		return nil
	}

	walkStmt = func(stmt frontend.Stmt) error {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			return walkExpr(s.Value)
		case *frontend.ReturnStmt:
			return walkExpr(s.Value)
		case *frontend.ThrowStmt:
			return walkExpr(s.Value)
		case *frontend.DeferStmt:
			for _, inner := range s.Body {
				if err := walkStmt(inner); err != nil {
					return err
				}
			}
			return nil
		case *frontend.BreakStmt, *frontend.ContinueStmt:
			return nil
		case *frontend.LetStmt:
			return walkExpr(s.Value)
		case *frontend.AssignStmt:
			if err := walkExpr(s.Target); err != nil {
				return err
			}
			return walkExpr(s.Value)
		case *frontend.IfStmt:
			if err := walkExpr(s.Cond); err != nil {
				return err
			}
			for _, inner := range s.Then {
				if err := walkStmt(inner); err != nil {
					return err
				}
			}
			for _, inner := range s.Else {
				if err := walkStmt(inner); err != nil {
					return err
				}
			}
		case *frontend.IfLetStmt:
			if err := walkExpr(s.Value); err != nil {
				return err
			}
			if s.Pattern != nil {
				if err := walkExpr(s.Pattern); err != nil {
					return err
				}
			}
			for _, inner := range s.Then {
				if err := walkStmt(inner); err != nil {
					return err
				}
			}
			for _, inner := range s.Else {
				if err := walkStmt(inner); err != nil {
					return err
				}
			}
		case *frontend.WhileStmt:
			if err := walkExpr(s.Cond); err != nil {
				return err
			}
			for _, inner := range s.Body {
				if err := walkStmt(inner); err != nil {
					return err
				}
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				if err := walkExpr(s.Iterable); err != nil {
					return err
				}
			} else {
				if err := walkExpr(s.Start); err != nil {
					return err
				}
				if err := walkExpr(s.End); err != nil {
					return err
				}
			}
			for _, inner := range s.Body {
				if err := walkStmt(inner); err != nil {
					return err
				}
			}
		case *frontend.MatchStmt:
			if err := walkExpr(s.Value); err != nil {
				return err
			}
			for _, c := range s.Cases {
				if !c.Default {
					if err := walkExpr(c.Pattern); err != nil {
						return err
					}
				}
				for _, inner := range c.Body {
					if err := walkStmt(inner); err != nil {
						return err
					}
				}
			}
		case *frontend.FreeStmt:
			return walkExpr(s.Value)
		case *frontend.UnsafeStmt:
			for _, inner := range s.Body {
				if err := walkStmt(inner); err != nil {
					return err
				}
			}
		case *frontend.IslandStmt:
			if err := walkExpr(s.Size); err != nil {
				return err
			}
			for _, inner := range s.Body {
				if err := walkStmt(inner); err != nil {
					return err
				}
			}
		default:
			return nil
		}
		return nil
	}

	for _, fn := range checked.Funcs {
		if fn.Decl == nil {
			continue
		}
		for _, stmt := range fn.Decl.Body {
			if err := walkStmt(stmt); err != nil {
				return false, nil, 0, err
			}
		}
	}
	if !used {
		return false, nil, 0, nil
	}

	names := make([]string, 0, len(targets))
	for name := range targets {
		if name == checked.MainName {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	entries := append([]string{checked.MainName}, names...)
	return true, entries, spawnCount, nil
}

func CollectActorStateRuntimeUsage(checked *semantics.CheckedProgram) bool {
	used, _ := CollectActorStateRuntimeUsagePosition(checked)
	return used
}

func CollectActorStateRuntimeUsagePosition(
	checked *semantics.CheckedProgram,
) (bool, frontend.Position) {
	if checked == nil {
		return false, frontend.Position{}
	}
	for _, fn := range checked.Funcs {
		if len(fn.ActorState) > 0 {
			if fn.Decl != nil {
				return true, fn.Decl.Pos
			}
			return true, frontend.Position{}
		}
	}
	return false, frontend.Position{}
}

func CollectActorRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	if checked == nil {
		return false, frontend.Position{}
	}
	var used bool
	var first frontend.Position
	var walkExpr func(frontend.Expr)
	var walkStmt func(frontend.Stmt)

	mark := func(pos frontend.Position) {
		if !used {
			used = true
			first = pos
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
			case "core.spawn",
				"core.send", "core.send_msg", "core.send_typed",
				"core.recv", "core.recv_msg", "core.recv_poll", "core.recv_until", "core.recv_msg_until", "core.recv_typed",
				"core.self", "core.sender", "core.yield":
				mark(e.At)
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
		case *frontend.CatchExpr:
			walkExpr(e.Call)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		case *frontend.MatchExpr:
			walkExpr(e.Value)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
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
	return used, first
}

func CollectTaskRuntimeUsage(checked *semantics.CheckedProgram) bool {
	used, _ := CollectTaskRuntimeUsagePosition(checked)
	return used
}

func CollectTaskRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	if checked == nil {
		return false, frontend.Position{}
	}
	var used bool
	var first frontend.Position
	var walkExpr func(frontend.Expr)
	var walkStmt func(frontend.Stmt)

	mark := func(pos frontend.Position) {
		if !used {
			used = true
			first = pos
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
			case "core.task_spawn_i32", "core.task_spawn_group_i32", "core.task_spawn_i32_typed", "core.task_spawn_group_i32_typed",
				"core.task_join_i32", "core.task_join_result_i32", "core.task_join_until_i32", "core.task_poll_i32", "core.select2_i32",
				"core.task_join_i32_typed", "core.task_join_group_i32_typed",
				"core.task_group_open", "core.task_group_close", "core.task_group_cancel", "core.task_group_current", "core.task_group_status",
				"core.task_is_canceled", "core.task_checkpoint":
				mark(e.At)
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
		case *frontend.CatchExpr:
			walkExpr(e.Call)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		case *frontend.MatchExpr:
			walkExpr(e.Value)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
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
	return used, first
}

func CollectTaskGroupRuntimeUsage(checked *semantics.CheckedProgram) bool {
	if checked == nil {
		return false
	}
	var used bool
	var walkExpr func(frontend.Expr)
	var walkStmt func(frontend.Stmt)

	walkExpr = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.CallExpr:
			name := e.Name
			if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
				name = builtin
			}
			switch name {
			case "core.task_group_open", "core.task_group_close", "core.task_group_cancel", "core.task_group_current", "core.task_group_status",
				"core.task_is_canceled", "core.task_checkpoint",
				"core.task_spawn_group_i32", "core.task_spawn_group_i32_typed":
				used = true
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
		case *frontend.CatchExpr:
			walkExpr(e.Call)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		case *frontend.MatchExpr:
			walkExpr(e.Value)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
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
	return used
}

func CollectTypedTaskRuntimeUsage(checked *semantics.CheckedProgram) (bool, int) {
	if checked == nil {
		return false, 0
	}
	var used bool
	maxSlots := 0
	var walkExpr func(frontend.Expr)
	var walkStmt func(frontend.Stmt)

	walkExpr = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.CallExpr:
			name := e.Name
			if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
				name = builtin
			}
			switch name {
			case "core.task_spawn_i32_typed", "core.task_spawn_group_i32_typed", "core.task_join_i32_typed", "core.task_join_group_i32_typed":
				used = true
				if len(e.TypeArgs) == 1 && e.TypeArgs[0].Name != "" {
					if _, handleInfo, err := semantics.EnsureTypedTaskHandleType(
						e.TypeArgs[0].Name,
						checked.Types,
					); err == nil {
						if handleInfo.SlotCount > maxSlots {
							maxSlots = handleInfo.SlotCount
						}
					}
				}
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
		case *frontend.CatchExpr:
			walkExpr(e.Call)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		case *frontend.MatchExpr:
			walkExpr(e.Value)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
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
	if used && maxSlots < 4 {
		maxSlots = 4
	}
	return used, maxSlots
}
