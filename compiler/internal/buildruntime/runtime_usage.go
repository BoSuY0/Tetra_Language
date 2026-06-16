package buildruntime

import (
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/runtimeabi"
	"tetra_language/compiler/internal/semantics"
)

func CollectDistributedActorRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
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
			case "core.actor_node_connect", "core.spawn_remote", "core.actor_node_status":
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

func CollectTimeRuntimeUsage(checked *semantics.CheckedProgram) bool {
	used, _ := CollectTimeRuntimeUsagePosition(checked)
	return used
}

func CollectTimeRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
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
			case "core.time_now_ms", "core.sleep_ms", "core.sleep_until", "core.deadline_ms", "core.timer_ready":
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

func CollectFilesystemRuntimeUsage(checked *semantics.CheckedProgram) bool {
	used, _ := CollectFilesystemRuntimeUsagePosition(checked)
	return used
}

func CollectFilesystemRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	if checked == nil {
		return false, frontend.Position{}
	}
	var used bool
	var pos frontend.Position
	var walkExpr func(frontend.Expr)
	var walkStmt func(frontend.Stmt)

	walkExpr = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.CallExpr:
			name := e.Name
			if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
				name = builtin
			}
			if name == "core.fs_exists" {
				used = true
				if pos.Line == 0 && pos.Col == 0 {
					pos = e.At
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
	return used, pos
}

func CollectNetRuntimeUsage(checked *semantics.CheckedProgram) bool {
	return CollectNetRuntimeUsageProfile(checked).Used
}

func CollectNetRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	usage := CollectNetRuntimeUsageProfile(checked)
	return usage.Used, usage.FirstPos
}

type NetRuntimeUsageProfile struct {
	Used            bool
	FirstPos        frontend.Position
	SymbolPositions map[string]frontend.Position
}

func (u NetRuntimeUsageProfile) RequiredSymbols() []string {
	if !u.Used {
		return nil
	}
	out := make([]string, 0, len(u.SymbolPositions))
	for _, symbol := range runtimeabi.RequiredNetSymbols() {
		if _, ok := u.SymbolPositions[symbol]; ok {
			out = append(out, symbol)
		}
	}
	return out
}

func CollectNetRuntimeUsageProfile(checked *semantics.CheckedProgram) NetRuntimeUsageProfile {
	if checked == nil {
		return NetRuntimeUsageProfile{}
	}
	usage := NetRuntimeUsageProfile{SymbolPositions: map[string]frontend.Position{}}
	var walkExpr func(frontend.Expr)
	var walkStmt func(frontend.Stmt)

	walkExpr = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.CallExpr:
			name := e.Name
			if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
				name = builtin
			}
			if symbol, ok := NetRuntimeSymbolForBuiltin(name); ok {
				usage.Used = true
				if usage.FirstPos.Line == 0 && usage.FirstPos.Col == 0 {
					usage.FirstPos = e.At
				}
				if _, exists := usage.SymbolPositions[symbol]; !exists {
					usage.SymbolPositions[symbol] = e.At
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
	return usage
}

func NetRuntimeSymbolForBuiltin(name string) (string, bool) {
	switch name {
	case "core.net_socket_tcp4":
		return "__tetra_net_socket_tcp4", true
	case "core.net_bind_tcp4_loopback":
		return "__tetra_net_bind_tcp4_loopback", true
	case "core.net_connect_tcp4_loopback":
		return "__tetra_net_connect_tcp4_loopback", true
	case "core.net_listen":
		return "__tetra_net_listen", true
	case "core.net_accept4":
		return "__tetra_net_accept4", true
	case "core.net_read":
		return "__tetra_net_read", true
	case "core.net_recv":
		return "__tetra_net_recv", true
	case "core.net_write":
		return "__tetra_net_write", true
	case "core.net_send":
		return "__tetra_net_send", true
	case "core.net_epoll_create":
		return "__tetra_net_epoll_create", true
	case "core.net_epoll_ctl_add_read":
		return "__tetra_net_epoll_ctl_add_read", true
	case "core.net_epoll_ctl_add_read_write":
		return "__tetra_net_epoll_ctl_add_read_write", true
	case "core.net_epoll_ctl_mod_read":
		return "__tetra_net_epoll_ctl_mod_read", true
	case "core.net_epoll_ctl_mod_read_write":
		return "__tetra_net_epoll_ctl_mod_read_write", true
	case "core.net_epoll_ctl_delete":
		return "__tetra_net_epoll_ctl_delete", true
	case "core.net_epoll_wait_one":
		return "__tetra_net_epoll_wait_one", true
	case "core.net_epoll_wait_one_into":
		return "__tetra_net_epoll_wait_one_into", true
	case "core.net_set_nonblocking":
		return "__tetra_net_set_nonblocking", true
	case "core.net_set_reuseport":
		return "__tetra_net_set_reuseport", true
	case "core.net_set_tcp_nodelay":
		return "__tetra_net_set_tcp_nodelay", true
	case "core.net_close":
		return "__tetra_net_close", true
	default:
		return "", false
	}
}

func CollectSurfaceRuntimeUsage(checked *semantics.CheckedProgram) bool {
	used, _ := CollectSurfaceRuntimeUsagePosition(checked)
	return used
}

func CollectSurfaceRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	if checked == nil {
		return false, frontend.Position{}
	}
	var used bool
	var pos frontend.Position
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
			case "core.surface_open", "core.surface_close", "core.surface_poll_event_kind", "core.surface_poll_event_x",
				"core.surface_poll_event_y", "core.surface_poll_event_button", "core.surface_poll_event_into", "core.surface_poll_event_text_len", "core.surface_poll_event_text_into",
				"core.surface_clipboard_write_text", "core.surface_clipboard_read_text_into", "core.surface_poll_composition_into", "core.surface_begin_frame",
				"core.surface_present_rgba", "core.surface_now_ms", "core.surface_request_redraw":
				used = true
				if pos.Line == 0 && pos.Col == 0 {
					pos = e.At
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
	return used, pos
}
