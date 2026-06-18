package deps

import (
	"strings"

	"tetra_language/compiler/internal/cache"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics"
)

func CollectExternalCalleesByModule(
	checked *semantics.CheckedProgram,
) map[string]map[string]struct{} {
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
		globals := checked.GlobalsByModule[mod]
		for _, stmt := range fn.Decl.Body {
			collectCalleesFromStmt(stmt, mod, out, checked.Types, fn.Locals, globals, fn.Imports)
		}
	}
	return deps
}

func collectCalleesFromStmt(
	stmt frontend.Stmt,
	mod string,
	out map[string]struct{},
	types map[string]*semantics.TypeInfo,
	locals map[string]semantics.LocalInfo,
	globals map[string]semantics.GlobalInfo,
	imports map[string]string,
) {
	switch s := stmt.(type) {
	case *frontend.PrintStmt:
		collectCalleesFromExpr(s.Value, mod, out, types, locals, globals, imports)
	case *frontend.ReturnStmt:
		collectCalleesFromExpr(s.Value, mod, out, types, locals, globals, imports)
	case *frontend.ThrowStmt:
		collectCalleesFromExpr(s.Value, mod, out, types, locals, globals, imports)
	case *frontend.DeferStmt:
		for _, inner := range s.Body {
			collectCalleesFromStmt(inner, mod, out, types, locals, globals, imports)
		}
	case *frontend.BreakStmt, *frontend.ContinueStmt:
	case *frontend.LetStmt:
		collectCalleesFromExpr(s.Value, mod, out, types, locals, globals, imports)
	case *frontend.AssignStmt:
		collectCalleesFromExpr(s.Target, mod, out, types, locals, globals, imports)
		collectCalleesFromExpr(s.Value, mod, out, types, locals, globals, imports)
	case *frontend.IfStmt:
		collectCalleesFromExpr(s.Cond, mod, out, types, locals, globals, imports)
		for _, inner := range s.Then {
			collectCalleesFromStmt(inner, mod, out, types, locals, globals, imports)
		}
		for _, inner := range s.Else {
			collectCalleesFromStmt(inner, mod, out, types, locals, globals, imports)
		}
	case *frontend.WhileStmt:
		collectCalleesFromExpr(s.Cond, mod, out, types, locals, globals, imports)
		for _, inner := range s.Body {
			collectCalleesFromStmt(inner, mod, out, types, locals, globals, imports)
		}
	case *frontend.ForRangeStmt:
		if s.Iterable != nil {
			collectCalleesFromExpr(s.Iterable, mod, out, types, locals, globals, imports)
		} else {
			collectCalleesFromExpr(s.Start, mod, out, types, locals, globals, imports)
			collectCalleesFromExpr(s.End, mod, out, types, locals, globals, imports)
		}
		for _, inner := range s.Body {
			collectCalleesFromStmt(inner, mod, out, types, locals, globals, imports)
		}
	case *frontend.MatchStmt:
		collectCalleesFromExpr(s.Value, mod, out, types, locals, globals, imports)
		for _, c := range s.Cases {
			if !c.Default {
				collectCalleesFromExpr(c.Pattern, mod, out, types, locals, globals, imports)
			}
			if c.Guard != nil {
				collectCalleesFromExpr(c.Guard, mod, out, types, locals, globals, imports)
			}
			for _, inner := range c.Body {
				collectCalleesFromStmt(inner, mod, out, types, locals, globals, imports)
			}
		}
	case *frontend.ExprStmt:
		collectCalleesFromExpr(s.Expr, mod, out, types, locals, globals, imports)
	}
}

func collectCalleesFromExpr(
	expr frontend.Expr,
	mod string,
	out map[string]struct{},
	types map[string]*semantics.TypeInfo,
	locals map[string]semantics.LocalInfo,
	globals map[string]semantics.GlobalInfo,
	imports map[string]string,
) {
	switch e := expr.(type) {
	case *frontend.MatchExpr:
		collectCalleesFromExpr(e.Value, mod, out, types, locals, globals, imports)
		for _, c := range e.Cases {
			if !c.Default {
				collectCalleesFromExpr(c.Pattern, mod, out, types, locals, globals, imports)
			}
			if c.Guard != nil {
				collectCalleesFromExpr(c.Guard, mod, out, types, locals, globals, imports)
			}
			collectCalleesFromExpr(c.Value, mod, out, types, locals, globals, imports)
		}
	case *frontend.CatchExpr:
		collectCalleesFromExpr(e.Call, mod, out, types, locals, globals, imports)
		for _, c := range e.Cases {
			if !c.Default {
				collectCalleesFromExpr(c.Pattern, mod, out, types, locals, globals, imports)
			}
			if c.Guard != nil {
				collectCalleesFromExpr(c.Guard, mod, out, types, locals, globals, imports)
			}
			collectCalleesFromExpr(c.Value, mod, out, types, locals, globals, imports)
		}
	case *frontend.CallExpr:
		if isEnumCaseConstructorName(e.Name, mod, types, imports) {
			for _, arg := range e.Args {
				collectCalleesFromExpr(arg, mod, out, types, locals, globals, imports)
			}
			return
		}
		if local, ok := locals[e.Name]; ok && local.FunctionTypeValue {
			for _, arg := range e.Args {
				collectCalleesFromExpr(arg, mod, out, types, locals, globals, imports)
			}
			return
		}
		if isFunctionFieldCallName(e.Name, locals) {
			for _, arg := range e.Args {
				collectCalleesFromExpr(arg, mod, out, types, locals, globals, imports)
			}
			return
		}
		if global, ok := globals[e.Name]; ok && global.FunctionTypeValue {
			if global.FunctionValue != "" && cache.ModuleOf(global.FunctionValue) != mod {
				out[global.FunctionValue] = struct{}{}
			}
			for _, arg := range e.Args {
				collectCalleesFromExpr(arg, mod, out, types, locals, globals, imports)
			}
			return
		}
		targetModule := cache.ModuleOf(e.Name)
		if targetModule != mod {
			out[e.Name] = struct{}{}
		}
		for _, arg := range e.Args {
			collectCalleesFromExpr(arg, mod, out, types, locals, globals, imports)
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			collectCalleesFromExpr(field.Value, mod, out, types, locals, globals, imports)
		}
	case *frontend.FieldAccessExpr:
		if isEnumCaseOrImportedTypeFieldAccess(e, types, imports) {
			return
		}
		if target, ok := functionTypedGlobalFieldTargetFromExpr(e, globals); ok {
			if cache.ModuleOf(target) != mod {
				out[target] = struct{}{}
			}
			return
		}
		if target, ok := importedFunctionTargetName(e, imports); ok && cache.ModuleOf(target) != mod {
			out[target] = struct{}{}
		}
		collectCalleesFromExpr(e.Base, mod, out, types, locals, globals, imports)
	case *frontend.IndexExpr:
		collectCalleesFromExpr(e.Base, mod, out, types, locals, globals, imports)
		collectCalleesFromExpr(e.Index, mod, out, types, locals, globals, imports)
	case *frontend.BinaryExpr:
		collectCalleesFromExpr(e.Left, mod, out, types, locals, globals, imports)
		collectCalleesFromExpr(e.Right, mod, out, types, locals, globals, imports)
	case *frontend.UnaryExpr:
		collectCalleesFromExpr(e.X, mod, out, types, locals, globals, imports)
	case *frontend.IdentExpr, *frontend.NumberExpr, *frontend.BoolLitExpr, *frontend.StringLitExpr:
		return
	}
}

func isEnumCaseConstructorName(
	name string,
	mod string,
	types map[string]*semantics.TypeInfo,
	imports map[string]string,
) bool {
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return false
	}
	caseName := parts[len(parts)-1]
	typeParts := parts[:len(parts)-1]
	candidates := []string{strings.Join(typeParts, ".")}
	if resolved, ok := resolveImportedTypePath(typeParts, imports); ok &&
		resolved != candidates[0] {
		candidates = append(candidates, resolved)
	}
	if mod != "" && len(typeParts) == 1 {
		candidates = append(candidates, mod+"."+typeParts[0])
	}
	for _, typeName := range candidates {
		info, ok := types[typeName]
		if !ok || info.Kind != semantics.TypeEnum {
			continue
		}
		if _, ok := info.CaseMap[caseName]; ok {
			return true
		}
	}
	return false
}

func importedFunctionTargetName(
	expr *frontend.FieldAccessExpr,
	imports map[string]string,
) (string, bool) {
	if expr == nil {
		return "", false
	}
	base, ok := expr.Base.(*frontend.IdentExpr)
	if !ok {
		return "", false
	}
	module, ok := imports[base.Name]
	if !ok || module == "" {
		return "", false
	}
	return module + "." + expr.Field, true
}

func isEnumCaseOrImportedTypeFieldAccess(
	expr *frontend.FieldAccessExpr,
	types map[string]*semantics.TypeInfo,
	imports map[string]string,
) bool {
	parts := fieldAccessParts(expr)
	if len(parts) < 2 {
		return false
	}
	for typeLen := len(parts); typeLen >= 1; typeLen-- {
		typeName, ok := resolveImportedTypePath(parts[:typeLen], imports)
		if !ok {
			continue
		}
		info, exists := types[typeName]
		if !exists {
			continue
		}
		if typeLen == len(parts) {
			return true
		}
		if info.Kind != semantics.TypeEnum || typeLen != len(parts)-1 {
			continue
		}
		_, ok = info.CaseMap[parts[len(parts)-1]]
		return ok
	}
	return false
}

func fieldAccessParts(expr frontend.Expr) []string {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return []string{e.Name}
	case *frontend.FieldAccessExpr:
		parts := fieldAccessParts(e.Base)
		if len(parts) == 0 {
			return nil
		}
		return append(parts, e.Field)
	default:
		return nil
	}
}

func resolveImportedTypePath(parts []string, imports map[string]string) (string, bool) {
	if len(parts) == 0 {
		return "", false
	}
	if module, ok := imports[parts[0]]; ok && module != "" {
		module = strings.TrimPrefix(module, "\x00symbol:")
		if len(parts) == 1 {
			return module, true
		}
		return module + "." + strings.Join(parts[1:], "."), true
	}
	return strings.Join(parts, "."), true
}

func CollectExternalTypesByModule(
	checked *semantics.CheckedProgram,
) map[string]map[string]struct{} {
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

func isFunctionFieldCallName(name string, locals map[string]semantics.LocalInfo) bool {
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return false
	}
	local, ok := locals[parts[0]]
	if !ok || len(local.FunctionFields) == 0 {
		return false
	}
	_, ok = local.FunctionFields[strings.Join(parts[1:], ".")]
	return ok
}

func functionTypedGlobalFieldTargetFromExpr(
	expr *frontend.FieldAccessExpr,
	globals map[string]semantics.GlobalInfo,
) (string, bool) {
	if expr == nil {
		return "", false
	}
	name := fieldAccessName(expr)
	if name == "" {
		return "", false
	}
	global, ok := globals[name]
	if !ok || !global.FunctionTypeValue || global.FunctionValue == "" {
		return "", false
	}
	return global.FunctionValue, true
}

func fieldAccessName(expr frontend.Expr) string {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name
	case *frontend.FieldAccessExpr:
		base := fieldAccessName(e.Base)
		if base == "" || e.Field == "" {
			return ""
		}
		return base + "." + e.Field
	default:
		return ""
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
