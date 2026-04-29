package deps

import (
	"strings"

	"tetra_language/compiler/internal/cache"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics"
)

func CollectExternalCalleesByModule(checked *semantics.CheckedProgram) map[string]map[string]struct{} {
	deps := make(map[string]map[string]struct{})
	if checked == nil {
		return deps
	}
	for _, fn := range checked.Funcs {
		mod := fn.Module
		out := deps[mod]
		if out == nil {
			out = make(map[string]struct{})
			deps[mod] = out
		}
		if fn.Decl == nil {
			continue
		}
		for _, stmt := range fn.Decl.Body {
			collectCalleesFromStmt(stmt, mod, out, checked.Types, fn.Locals)
		}
	}
	return deps
}

func collectCalleesFromStmt(stmt frontend.Stmt, mod string, out map[string]struct{}, types map[string]*semantics.TypeInfo, locals map[string]semantics.LocalInfo) {
	switch s := stmt.(type) {
	case *frontend.PrintStmt:
		collectCalleesFromExpr(s.Value, mod, out, types, locals)
	case *frontend.ReturnStmt:
		collectCalleesFromExpr(s.Value, mod, out, types, locals)
	case *frontend.ThrowStmt:
		collectCalleesFromExpr(s.Value, mod, out, types, locals)
	case *frontend.DeferStmt:
		for _, inner := range s.Body {
			collectCalleesFromStmt(inner, mod, out, types, locals)
		}
	case *frontend.BreakStmt, *frontend.ContinueStmt:
	case *frontend.LetStmt:
		collectCalleesFromExpr(s.Value, mod, out, types, locals)
	case *frontend.AssignStmt:
		collectCalleesFromExpr(s.Target, mod, out, types, locals)
		collectCalleesFromExpr(s.Value, mod, out, types, locals)
	case *frontend.IfStmt:
		collectCalleesFromExpr(s.Cond, mod, out, types, locals)
		for _, inner := range s.Then {
			collectCalleesFromStmt(inner, mod, out, types, locals)
		}
		for _, inner := range s.Else {
			collectCalleesFromStmt(inner, mod, out, types, locals)
		}
	case *frontend.WhileStmt:
		collectCalleesFromExpr(s.Cond, mod, out, types, locals)
		for _, inner := range s.Body {
			collectCalleesFromStmt(inner, mod, out, types, locals)
		}
	case *frontend.ForRangeStmt:
		if s.Iterable != nil {
			collectCalleesFromExpr(s.Iterable, mod, out, types, locals)
		} else {
			collectCalleesFromExpr(s.Start, mod, out, types, locals)
			collectCalleesFromExpr(s.End, mod, out, types, locals)
		}
		for _, inner := range s.Body {
			collectCalleesFromStmt(inner, mod, out, types, locals)
		}
	case *frontend.MatchStmt:
		collectCalleesFromExpr(s.Value, mod, out, types, locals)
		for _, c := range s.Cases {
			if !c.Default {
				collectCalleesFromExpr(c.Pattern, mod, out, types, locals)
			}
			if c.Guard != nil {
				collectCalleesFromExpr(c.Guard, mod, out, types, locals)
			}
			for _, inner := range c.Body {
				collectCalleesFromStmt(inner, mod, out, types, locals)
			}
		}
	case *frontend.ExprStmt:
		collectCalleesFromExpr(s.Expr, mod, out, types, locals)
	}
}

func collectCalleesFromExpr(expr frontend.Expr, mod string, out map[string]struct{}, types map[string]*semantics.TypeInfo, locals map[string]semantics.LocalInfo) {
	switch e := expr.(type) {
	case *frontend.MatchExpr:
		collectCalleesFromExpr(e.Value, mod, out, types, locals)
		for _, c := range e.Cases {
			if !c.Default {
				collectCalleesFromExpr(c.Pattern, mod, out, types, locals)
			}
			if c.Guard != nil {
				collectCalleesFromExpr(c.Guard, mod, out, types, locals)
			}
			collectCalleesFromExpr(c.Value, mod, out, types, locals)
		}
	case *frontend.CatchExpr:
		collectCalleesFromExpr(e.Call, mod, out, types, locals)
		for _, c := range e.Cases {
			if !c.Default {
				collectCalleesFromExpr(c.Pattern, mod, out, types, locals)
			}
			if c.Guard != nil {
				collectCalleesFromExpr(c.Guard, mod, out, types, locals)
			}
			collectCalleesFromExpr(c.Value, mod, out, types, locals)
		}
	case *frontend.CallExpr:
		if isEnumCaseConstructorName(e.Name, types) {
			for _, arg := range e.Args {
				collectCalleesFromExpr(arg, mod, out, types, locals)
			}
			return
		}
		if local, ok := locals[e.Name]; ok && local.FunctionTypeValue {
			for _, arg := range e.Args {
				collectCalleesFromExpr(arg, mod, out, types, locals)
			}
			return
		}
		targetModule := cache.ModuleOf(e.Name)
		if targetModule != mod {
			out[e.Name] = struct{}{}
		}
		for _, arg := range e.Args {
			collectCalleesFromExpr(arg, mod, out, types, locals)
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			collectCalleesFromExpr(field.Value, mod, out, types, locals)
		}
	case *frontend.FieldAccessExpr:
		collectCalleesFromExpr(e.Base, mod, out, types, locals)
	case *frontend.IndexExpr:
		collectCalleesFromExpr(e.Base, mod, out, types, locals)
		collectCalleesFromExpr(e.Index, mod, out, types, locals)
	case *frontend.BinaryExpr:
		collectCalleesFromExpr(e.Left, mod, out, types, locals)
		collectCalleesFromExpr(e.Right, mod, out, types, locals)
	case *frontend.UnaryExpr:
		collectCalleesFromExpr(e.X, mod, out, types, locals)
	case *frontend.IdentExpr, *frontend.NumberExpr, *frontend.BoolLitExpr, *frontend.StringLitExpr:
		return
	}
}

func isEnumCaseConstructorName(name string, types map[string]*semantics.TypeInfo) bool {
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return false
	}
	typeName := strings.Join(parts[:len(parts)-1], ".")
	info, ok := types[typeName]
	if !ok || info.Kind != semantics.TypeEnum {
		return false
	}
	_, ok = info.CaseMap[parts[len(parts)-1]]
	return ok
}

func CollectExternalTypesByModule(checked *semantics.CheckedProgram) map[string]map[string]struct{} {
	deps := make(map[string]map[string]struct{})
	if checked == nil {
		return deps
	}
	addType := func(mod string, name string) {
		if name == "" || name == "i32" || name == "u8" || name == "ptr" || name == "str" {
			return
		}
		if cache.ModuleOf(name) == mod {
			return
		}
		out := deps[mod]
		if out == nil {
			out = make(map[string]struct{})
			deps[mod] = out
		}
		out[name] = struct{}{}
	}

	for _, st := range checked.Structs {
		mod := st.Module
		if st.Decl == nil {
			continue
		}
		for _, field := range st.Decl.Fields {
			addType(mod, field.Type.Name)
		}
	}

	for _, fn := range checked.Funcs {
		mod := fn.Module
		if fn.Decl != nil {
			addType(mod, fn.Decl.ReturnType.Name)
			for _, param := range fn.Decl.Params {
				addType(mod, param.Type.Name)
			}
			for _, stmt := range fn.Decl.Body {
				collectTypesFromStmt(stmt, mod, addType)
			}
		}
	}

	return deps
}

func collectTypesFromStmt(stmt frontend.Stmt, mod string, addType func(string, string)) {
	switch s := stmt.(type) {
	case *frontend.LetStmt:
		addType(mod, s.Type.Name)
		collectTypesFromExpr(s.Value, mod, addType)
	case *frontend.AssignStmt:
		collectTypesFromExpr(s.Target, mod, addType)
		collectTypesFromExpr(s.Value, mod, addType)
	case *frontend.ReturnStmt:
		collectTypesFromExpr(s.Value, mod, addType)
	case *frontend.ThrowStmt:
		collectTypesFromExpr(s.Value, mod, addType)
	case *frontend.DeferStmt:
		for _, inner := range s.Body {
			collectTypesFromStmt(inner, mod, addType)
		}
	case *frontend.BreakStmt, *frontend.ContinueStmt:
	case *frontend.IfStmt:
		collectTypesFromExpr(s.Cond, mod, addType)
		for _, inner := range s.Then {
			collectTypesFromStmt(inner, mod, addType)
		}
		for _, inner := range s.Else {
			collectTypesFromStmt(inner, mod, addType)
		}
	case *frontend.WhileStmt:
		collectTypesFromExpr(s.Cond, mod, addType)
		for _, inner := range s.Body {
			collectTypesFromStmt(inner, mod, addType)
		}
	case *frontend.ForRangeStmt:
		if s.Iterable != nil {
			collectTypesFromExpr(s.Iterable, mod, addType)
		} else {
			collectTypesFromExpr(s.Start, mod, addType)
			collectTypesFromExpr(s.End, mod, addType)
		}
		for _, inner := range s.Body {
			collectTypesFromStmt(inner, mod, addType)
		}
	case *frontend.MatchStmt:
		collectTypesFromExpr(s.Value, mod, addType)
		for _, c := range s.Cases {
			if !c.Default {
				collectTypesFromExpr(c.Pattern, mod, addType)
			}
			if c.Guard != nil {
				collectTypesFromExpr(c.Guard, mod, addType)
			}
			for _, inner := range c.Body {
				collectTypesFromStmt(inner, mod, addType)
			}
		}
	case *frontend.PrintStmt:
		collectTypesFromExpr(s.Value, mod, addType)
	case *frontend.ExprStmt:
		collectTypesFromExpr(s.Expr, mod, addType)
	}
}

func collectTypesFromExpr(expr frontend.Expr, mod string, addType func(string, string)) {
	switch e := expr.(type) {
	case *frontend.MatchExpr:
		collectTypesFromExpr(e.Value, mod, addType)
		for _, c := range e.Cases {
			if !c.Default {
				collectTypesFromExpr(c.Pattern, mod, addType)
			}
			if c.Guard != nil {
				collectTypesFromExpr(c.Guard, mod, addType)
			}
			collectTypesFromExpr(c.Value, mod, addType)
		}
	case *frontend.CatchExpr:
		collectTypesFromExpr(e.Call, mod, addType)
		for _, c := range e.Cases {
			if !c.Default {
				collectTypesFromExpr(c.Pattern, mod, addType)
			}
			if c.Guard != nil {
				collectTypesFromExpr(c.Guard, mod, addType)
			}
			collectTypesFromExpr(c.Value, mod, addType)
		}
	case *frontend.StructLitExpr:
		addType(mod, e.Type.Name)
		for _, field := range e.Fields {
			collectTypesFromExpr(field.Value, mod, addType)
		}
	case *frontend.FieldAccessExpr:
		if e.EnumType != "" {
			addType(mod, e.EnumType)
		}
		collectTypesFromExpr(e.Base, mod, addType)
	case *frontend.EnumCasePatternExpr:
		if e.EnumType != "" {
			addType(mod, e.EnumType)
		}
	case *frontend.IndexExpr:
		collectTypesFromExpr(e.Base, mod, addType)
		collectTypesFromExpr(e.Index, mod, addType)
	case *frontend.CallExpr:
		for _, typeArg := range e.TypeArgs {
			addType(mod, typeArg.Name)
		}
		for _, arg := range e.Args {
			collectTypesFromExpr(arg, mod, addType)
		}
	case *frontend.BinaryExpr:
		collectTypesFromExpr(e.Left, mod, addType)
		collectTypesFromExpr(e.Right, mod, addType)
	case *frontend.UnaryExpr:
		collectTypesFromExpr(e.X, mod, addType)
	case *frontend.IdentExpr, *frontend.NumberExpr, *frontend.BoolLitExpr, *frontend.StringLitExpr:
		return
	}
}
