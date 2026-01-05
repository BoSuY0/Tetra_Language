package deps

import (
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
			collectCalleesFromStmt(stmt, mod, out)
		}
	}
	return deps
}

func collectCalleesFromStmt(stmt frontend.Stmt, mod string, out map[string]struct{}) {
	switch s := stmt.(type) {
	case *frontend.PrintStmt:
		collectCalleesFromExpr(s.Value, mod, out)
	case *frontend.ReturnStmt:
		collectCalleesFromExpr(s.Value, mod, out)
	case *frontend.LetStmt:
		collectCalleesFromExpr(s.Value, mod, out)
	case *frontend.AssignStmt:
		collectCalleesFromExpr(s.Target, mod, out)
		collectCalleesFromExpr(s.Value, mod, out)
	case *frontend.IfStmt:
		collectCalleesFromExpr(s.Cond, mod, out)
		for _, inner := range s.Then {
			collectCalleesFromStmt(inner, mod, out)
		}
		for _, inner := range s.Else {
			collectCalleesFromStmt(inner, mod, out)
		}
	case *frontend.WhileStmt:
		collectCalleesFromExpr(s.Cond, mod, out)
		for _, inner := range s.Body {
			collectCalleesFromStmt(inner, mod, out)
		}
	}
}

func collectCalleesFromExpr(expr frontend.Expr, mod string, out map[string]struct{}) {
	switch e := expr.(type) {
	case *frontend.CallExpr:
		targetModule := cache.ModuleOf(e.Name)
		if targetModule != mod {
			out[e.Name] = struct{}{}
		}
		for _, arg := range e.Args {
			collectCalleesFromExpr(arg, mod, out)
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			collectCalleesFromExpr(field.Value, mod, out)
		}
	case *frontend.FieldAccessExpr:
		collectCalleesFromExpr(e.Base, mod, out)
	case *frontend.IndexExpr:
		collectCalleesFromExpr(e.Base, mod, out)
		collectCalleesFromExpr(e.Index, mod, out)
	case *frontend.BinaryExpr:
		collectCalleesFromExpr(e.Left, mod, out)
		collectCalleesFromExpr(e.Right, mod, out)
	case *frontend.UnaryExpr:
		collectCalleesFromExpr(e.X, mod, out)
	case *frontend.IdentExpr, *frontend.NumberExpr, *frontend.StringLitExpr:
		return
	}
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
	case *frontend.PrintStmt:
		collectTypesFromExpr(s.Value, mod, addType)
	}
}

func collectTypesFromExpr(expr frontend.Expr, mod string, addType func(string, string)) {
	switch e := expr.(type) {
	case *frontend.StructLitExpr:
		addType(mod, e.Type.Name)
		for _, field := range e.Fields {
			collectTypesFromExpr(field.Value, mod, addType)
		}
	case *frontend.FieldAccessExpr:
		collectTypesFromExpr(e.Base, mod, addType)
	case *frontend.IndexExpr:
		collectTypesFromExpr(e.Base, mod, addType)
		collectTypesFromExpr(e.Index, mod, addType)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			collectTypesFromExpr(arg, mod, addType)
		}
	case *frontend.BinaryExpr:
		collectTypesFromExpr(e.Left, mod, addType)
		collectTypesFromExpr(e.Right, mod, addType)
	case *frontend.UnaryExpr:
		collectTypesFromExpr(e.X, mod, addType)
	case *frontend.IdentExpr, *frontend.NumberExpr, *frontend.StringLitExpr:
		return
	}
}
