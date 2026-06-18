package compiler

import (
	"fmt"
	"sort"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

type interfaceCaptureStub struct {
	Name    string
	Type    frontend.TypeRef
	Mutable bool
}

func interfaceThrowExpr(fn *frontend.FuncDecl) (string, bool) {
	if fn == nil || !fn.HasThrows {
		return "", false
	}
	paramNames := map[string]bool{}
	for _, param := range fn.Params {
		paramNames[param.Name] = true
	}
	aliases := map[string]string{}
	for _, stmt := range fn.Body {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames); ok {
				aliases[s.Name] = value
			}
			continue
		case *frontend.AssignStmt:
			target, ok := s.Target.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if _, ok := aliases[target.Name]; !ok {
				continue
			}
			value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames)
			if !ok {
				continue
			}
			aliases[target.Name] = value
			continue
		}
		throwStmt, ok := stmt.(*frontend.ThrowStmt)
		if !ok {
			continue
		}
		formatted, ok := interfaceAggregateStubExprWithAliases(throwStmt.Value, aliases)
		if !ok {
			formatted, ok = interfaceContractExpr(throwStmt.Value)
		}
		if !ok || (!interfaceExprRefsAnyParam(throwStmt.Value, paramNames) && !interfaceExprRefsAnyAlias(throwStmt.Value, aliases)) {
			return "", false
		}
		return formatted, true
	}
	return "", false
}

func interfaceContractExpr(expr frontend.Expr) (string, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name, true
	case *frontend.NumberExpr:
		return fmt.Sprintf("%d", e.Value), true
	case *frontend.BoolLitExpr:
		if e.Value {
			return "true", true
		}
		return "false", true
	case *frontend.NoneLitExpr:
		return "none", true
	case *frontend.StringLitExpr:
		return fmt.Sprintf("%q", string(e.Value)), true
	case *frontend.TryExpr:
		inner, ok := interfaceContractExpr(e.X)
		if !ok {
			return "", false
		}
		return "try " + inner, true
	case *frontend.FieldAccessExpr:
		base, ok := interfaceContractExpr(e.Base)
		if !ok {
			return "", false
		}
		return base + "." + e.Field, true
	case *frontend.CallExpr:
		args := make([]string, 0, len(e.Args))
		for i, arg := range e.Args {
			formatted, ok := interfaceContractExpr(arg)
			if !ok {
				return "", false
			}
			if i < len(e.ArgLabels) && e.ArgLabels[i] != "" {
				formatted = e.ArgLabels[i] + ": " + formatted
			}
			args = append(args, formatted)
		}
		return e.Name + "(" + strings.Join(args, ", ") + ")", true
	case *frontend.StructLitExpr:
		fields := make([]string, 0, len(e.Fields))
		for _, field := range e.Fields {
			formatted, ok := interfaceContractExpr(field.Value)
			if !ok {
				return "", false
			}
			fields = append(fields, field.Name+": "+formatted)
		}
		return formatLSPTypeRef(e.Type) + "(" + strings.Join(fields, ", ") + ")", true
	default:
		return "", false
	}
}

func interfaceExprRefsAnyParam(expr frontend.Expr, params map[string]bool) bool {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return params[e.Name]
	case *frontend.FieldAccessExpr:
		return interfaceExprRefsAnyParam(e.Base, params)
	case *frontend.TryExpr:
		return interfaceExprRefsAnyParam(e.X, params)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			if interfaceExprRefsAnyParam(arg, params) {
				return true
			}
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if interfaceExprRefsAnyParam(field.Value, params) {
				return true
			}
		}
	}
	return false
}

func interfaceExprRefsAnyAlias(expr frontend.Expr, aliases map[string]string) bool {
	if len(aliases) == 0 {
		return false
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return aliases[e.Name] != ""
	case *frontend.FieldAccessExpr:
		return interfaceExprRefsAnyAlias(e.Base, aliases)
	case *frontend.TryExpr:
		return interfaceExprRefsAnyAlias(e.X, aliases)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			if interfaceExprRefsAnyAlias(arg, aliases) {
				return true
			}
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if interfaceExprRefsAnyAlias(field.Value, aliases) {
				return true
			}
		}
	}
	return false
}

func interfaceReturnedClosureCaptureBody(fn *frontend.FuncDecl) (string, bool) {
	if fn.ReturnType.Kind != frontend.TypeRefFunction {
		return "", false
	}
	outerLocals := map[string]interfaceCaptureStub{}
	outerOrder := []string{}
	for _, stmt := range fn.Body {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if _, exists := outerLocals[s.Name]; !exists {
				outerOrder = append(outerOrder, s.Name)
			}
			outerLocals[s.Name] = interfaceCaptureStub{Name: s.Name, Type: s.Type, Mutable: s.Mutable}
		case *frontend.ReturnStmt:
			closure, ok := s.Value.(*frontend.ClosureExpr)
			if !ok || closure.Decl == nil {
				return "", false
			}
			used := map[string]bool{}
			interfaceCollectStmtIdents(closure.Decl.Body, used)
			for _, param := range closure.Decl.Params {
				delete(used, param.Name)
			}
			for local := range interfaceLocalNames(closure.Decl.Body) {
				delete(used, local)
			}
			captures := make([]interfaceCaptureStub, 0, len(outerOrder))
			for _, name := range outerOrder {
				if used[name] {
					captures = append(captures, outerLocals[name])
				}
			}
			if len(captures) == 0 {
				return "", false
			}
			var b strings.Builder
			for _, capture := range captures {
				decl := "let"
				if capture.Mutable {
					decl = "var"
				}
				fmt.Fprintf(&b, "    %s %s: %s = %s\n", decl, capture.Name, formatLSPTypeRef(capture.Type), interfaceReturnLiteral(capture.Type))
			}
			fmt.Fprintf(&b, "    return %s", interfaceCapturedClosureLiteral(closure, captures, "        "))
			return b.String(), true
		}
	}
	return "", false
}

func interfaceCapturedClosureLiteral(closure *frontend.ClosureExpr, captures []interfaceCaptureStub, bodyIndent string) string {
	params := make([]string, 0, len(closure.Decl.Params))
	for _, param := range closure.Decl.Params {
		formatted := formatLSPTypeRef(param.Type)
		if param.Ownership != "" {
			formatted = param.Ownership + " " + formatted
		}
		params = append(params, fmt.Sprintf("%s: %s", param.Name, formatted))
	}
	ret := formatLSPTypeRef(closure.Decl.ReturnType)
	out := "fn(" + strings.Join(params, ", ") + ") -> " + ret
	if closure.Decl.HasThrows {
		out += " throws " + formatLSPTypeRef(closure.Decl.Throws)
	}
	if len(closure.Decl.Uses) > 0 {
		uses := append([]string(nil), closure.Decl.Uses...)
		sort.Strings(uses)
		out += " uses " + strings.Join(uses, ", ")
	}
	var b strings.Builder
	b.WriteString(out)
	b.WriteString(":\n")
	for i, capture := range captures {
		fmt.Fprintf(&b, "%slet __capture_keep%d: %s = %s\n", bodyIndent, i, formatLSPTypeRef(capture.Type), capture.Name)
	}
	fmt.Fprintf(&b, "%sreturn %s", bodyIndent, interfaceReturnLiteral(closure.Decl.ReturnType))
	return b.String()
}

func interfaceLocalNames(stmts []frontend.Stmt) map[string]bool {
	names := map[string]bool{}
	for _, stmt := range stmts {
		if let, ok := stmt.(*frontend.LetStmt); ok {
			names[let.Name] = true
		}
	}
	return names
}

func interfaceCollectStmtIdents(stmts []frontend.Stmt, used map[string]bool) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			interfaceCollectExprIdents(s.Value, used)
		case *frontend.LetStmt:
			interfaceCollectExprIdents(s.Value, used)
		case *frontend.ExprStmt:
			interfaceCollectExprIdents(s.Expr, used)
		case *frontend.IfStmt:
			interfaceCollectExprIdents(s.Cond, used)
			interfaceCollectStmtIdents(s.Then, used)
			interfaceCollectStmtIdents(s.Else, used)
		case *frontend.MatchStmt:
			interfaceCollectExprIdents(s.Value, used)
			for _, c := range s.Cases {
				interfaceCollectExprIdents(c.Guard, used)
				interfaceCollectStmtIdents(c.Body, used)
			}
		}
	}
}

func interfaceCollectExprIdents(expr frontend.Expr, used map[string]bool) {
	switch e := expr.(type) {
	case nil:
		return
	case *frontend.IdentExpr:
		used[e.Name] = true
	case *frontend.BinaryExpr:
		interfaceCollectExprIdents(e.Left, used)
		interfaceCollectExprIdents(e.Right, used)
	case *frontend.UnaryExpr:
		interfaceCollectExprIdents(e.X, used)
	case *frontend.TryExpr:
		interfaceCollectExprIdents(e.X, used)
	case *frontend.AwaitExpr:
		interfaceCollectExprIdents(e.X, used)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			interfaceCollectExprIdents(arg, used)
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			interfaceCollectExprIdents(field.Value, used)
		}
	case *frontend.FieldAccessExpr:
		interfaceCollectExprIdents(e.Base, used)
	case *frontend.IndexExpr:
		interfaceCollectExprIdents(e.Base, used)
		interfaceCollectExprIdents(e.Index, used)
	case *frontend.MatchExpr:
		interfaceCollectExprIdents(e.Value, used)
		for _, c := range e.Cases {
			interfaceCollectExprIdents(c.Guard, used)
			interfaceCollectExprIdents(c.Value, used)
		}
	case *frontend.CatchExpr:
		interfaceCollectExprIdents(e.Call, used)
		for _, c := range e.Cases {
			interfaceCollectExprIdents(c.Guard, used)
			interfaceCollectExprIdents(c.Value, used)
		}
	}
}

func interfaceAggregateReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	paramNames := map[string]bool{}
	paramTypes := map[string]string{}
	for _, param := range fn.Params {
		formatted := formatLSPTypeRef(param.Type)
		paramNames[param.Name] = true
		paramTypes[param.Name] = formatted
	}
	aliases := map[string]string{}
	for _, stmt := range fn.Body {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if s.Type.Kind == frontend.TypeRefOptional {
				aliases[s.Name] = ""
				if value, ok := s.Value.(*frontend.IdentExpr); ok && paramNames[value.Name] {
					aliases[s.Name] = value.Name
				} else if value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames); ok {
					aliases[s.Name] = value
				}
			} else if value, ok := s.Value.(*frontend.IdentExpr); ok && paramTypes[value.Name] == formatLSPTypeRef(s.Type) {
				aliases[s.Name] = value.Name
			} else if value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames); ok {
				aliases[s.Name] = value
			}
			continue
		case *frontend.AssignStmt:
			target, ok := s.Target.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if _, ok := aliases[target.Name]; !ok {
				continue
			}
			value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames)
			if !ok {
				continue
			}
			aliases[target.Name] = value
			continue
		case *frontend.IfLetStmt:
			value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames)
			if !ok || s.Name == "" {
				continue
			}
			branchAliases := interfaceAliasMapCopy(aliases)
			branchAliases[s.Name] = value
			if expr, ok := interfaceAggregateReturnFromBranches(s.Then, branchAliases, s.Else, aliases, paramNames); ok {
				return expr, true
			}
			continue
		case *frontend.MatchStmt:
			value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames)
			if ok {
				for _, c := range s.Cases {
					name, ok := interfaceOptionalSomePatternName(c.Pattern)
					if !ok {
						continue
					}
					branchAliases := interfaceAliasMapCopy(aliases)
					branchAliases[name] = value
					if expr, ok := interfaceAggregateReturnFromStmts(c.Body, branchAliases, paramNames); ok {
						return expr, true
					}
				}
			}
		}
		if expr, ok := interfaceAggregateReturnFromStmts([]frontend.Stmt{stmt}, aliases, paramNames); ok {
			return expr, true
		}
	}
	return "", false
}

func interfaceAggregateReturnFromStmts(stmts []frontend.Stmt, aliases map[string]string, params map[string]bool) (string, bool) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			if !interfaceDirectAggregateExpr(s.Value) {
				continue
			}
			if !interfaceExprContainsClosure(s.Value) && !interfaceExprRefsAnyParam(s.Value, params) && !interfaceExprRefsAnyAlias(s.Value, aliases) {
				continue
			}
			expr, ok := interfaceAggregateStubExprWithAliases(s.Value, aliases)
			if ok {
				return expr, true
			}
		case *frontend.IfStmt:
			if expr, ok := interfaceAggregateReturnFromBranches(s.Then, aliases, s.Else, aliases, params); ok {
				return expr, true
			}
		case *frontend.MatchStmt:
			if expr, ok := interfaceAggregateMatchReturnExpr(s, aliases, params); ok {
				return expr, true
			}
		}
	}
	return "", false
}

func interfaceAggregateReturnFromBranches(thenStmts []frontend.Stmt, thenAliases map[string]string, elseStmts []frontend.Stmt, elseAliases map[string]string, params map[string]bool) (string, bool) {
	thenExpr, thenOK := interfaceAggregateReturnFromStmts(thenStmts, thenAliases, params)
	elseExpr, elseOK := interfaceAggregateReturnFromStmts(elseStmts, elseAliases, params)
	if thenOK && elseOK {
		if thenExpr == elseExpr {
			return thenExpr, true
		}
		return "", false
	}
	if thenOK {
		return thenExpr, true
	}
	if elseOK {
		return elseExpr, true
	}
	return "", false
}

func interfaceAggregateMatchReturnExpr(match *frontend.MatchStmt, aliases map[string]string, params map[string]bool) (string, bool) {
	if match == nil || len(match.Cases) == 0 {
		return "", false
	}
	var commonExpr string
	for _, c := range match.Cases {
		if c.Guard != nil {
			return "", false
		}
		expr, ok := interfaceAggregateReturnFromStmts(c.Body, aliases, params)
		if !ok {
			continue
		}
		if commonExpr == "" {
			commonExpr = expr
			continue
		}
		if commonExpr != expr {
			return "", false
		}
	}
	return commonExpr, commonExpr != ""
}

func interfaceOptionalSomePatternName(expr frontend.Expr) (string, bool) {
	if some, ok := expr.(*frontend.SomePatternExpr); ok {
		return some.Name, some.Name != ""
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call.Name != "some" || len(call.Args) != 1 {
		return "", false
	}
	id, ok := call.Args[0].(*frontend.IdentExpr)
	if !ok || id.Name == "" {
		return "", false
	}
	return id.Name, true
}

func interfaceAliasMapCopy(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func interfaceParamPathExpr(expr frontend.Expr, aliases map[string]string, params map[string]bool) (string, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		if alias := aliases[e.Name]; alias != "" {
			return alias, true
		}
		if params[e.Name] {
			return e.Name, true
		}
	case *frontend.FieldAccessExpr:
		base, ok := interfaceParamPathExpr(e.Base, aliases, params)
		if !ok {
			return "", false
		}
		return base + "." + e.Field, true
	}
	return "", false
}

func interfaceDirectAggregateExpr(expr frontend.Expr) bool {
	switch expr.(type) {
	case *frontend.CallExpr, *frontend.StructLitExpr:
		return true
	default:
		return false
	}
}

func interfaceExprContainsClosure(expr frontend.Expr) bool {
	switch e := expr.(type) {
	case *frontend.ClosureExpr:
		return true
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			if interfaceExprContainsClosure(arg) {
				return true
			}
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if interfaceExprContainsClosure(field.Value) {
				return true
			}
		}
	}
	return false
}

func interfaceAggregateStubExpr(expr frontend.Expr) (string, bool) {
	return interfaceAggregateStubExprWithAliases(expr, nil)
}

func interfaceAggregateStubExprWithAliases(expr frontend.Expr, aliases map[string]string) (string, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		if aliases != nil {
			if alias := aliases[e.Name]; alias != "" {
				return alias, true
			}
		}
		return e.Name, true
	case *frontend.TryExpr:
		inner, ok := interfaceAggregateStubExprWithAliases(e.X, aliases)
		if !ok {
			return "", false
		}
		return "try " + inner, true
	case *frontend.CallExpr:
		args := make([]string, 0, len(e.Args))
		for i, arg := range e.Args {
			formatted, ok := interfaceAggregateStubExprWithAliases(arg, aliases)
			if !ok {
				return "", false
			}
			if i < len(e.ArgLabels) && e.ArgLabels[i] != "" {
				formatted = e.ArgLabels[i] + ": " + formatted
			}
			args = append(args, formatted)
		}
		return e.Name + "(" + strings.Join(args, ", ") + ")", true
	case *frontend.StructLitExpr:
		fields := make([]string, 0, len(e.Fields))
		for _, field := range e.Fields {
			formatted, ok := interfaceAggregateStubExprWithAliases(field.Value, aliases)
			if !ok {
				return "", false
			}
			fields = append(fields, field.Name+": "+formatted)
		}
		return formatLSPTypeRef(e.Type) + "(" + strings.Join(fields, ", ") + ")", true
	case *frontend.FieldAccessExpr:
		base, ok := interfaceAggregateStubExprWithAliases(e.Base, aliases)
		if !ok {
			return "", false
		}
		return base + "." + e.Field, true
	case *frontend.ClosureExpr:
		ref, ok := interfaceClosureTypeRef(e)
		if !ok {
			return "", false
		}
		return interfaceInlineFunctionClosureLiteral(ref), true
	case *frontend.NumberExpr:
		return fmt.Sprintf("%d", e.Value), true
	case *frontend.BoolLitExpr:
		if e.Value {
			return "true", true
		}
		return "false", true
	case *frontend.NoneLitExpr:
		return "none", true
	default:
		return "", false
	}
}

func interfaceClosureTypeRef(closure *frontend.ClosureExpr) (frontend.TypeRef, bool) {
	if closure == nil || closure.Decl == nil {
		return frontend.TypeRef{}, false
	}
	params := make([]frontend.TypeRef, 0, len(closure.Decl.Params))
	ownership := make([]string, 0, len(closure.Decl.Params))
	for _, param := range closure.Decl.Params {
		params = append(params, param.Type)
		ownership = append(ownership, param.Ownership)
	}
	ret := closure.Decl.ReturnType
	ref := frontend.TypeRef{
		Kind:           frontend.TypeRefFunction,
		Params:         params,
		ParamOwnership: ownership,
		Return:         &ret,
		Uses:           append([]string(nil), closure.Decl.Uses...),
	}
	if closure.Decl.HasThrows {
		throws := closure.Decl.Throws
		ref.Throws = &throws
	}
	return ref, true
}

func interfaceSameTypedParameterReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	returnSig := formatLSPTypeRef(fn.ReturnType)
	sameTypedParams := map[string]bool{}
	for _, param := range fn.Params {
		if formatLSPTypeRef(param.Type) == returnSig {
			sameTypedParams[param.Name] = true
		}
	}
	aliases := map[string]string{}
	for _, stmt := range fn.Body {
		if let, ok := stmt.(*frontend.LetStmt); ok {
			if formatLSPTypeRef(let.Type) != returnSig {
				continue
			}
			if id, ok := let.Value.(*frontend.IdentExpr); ok && sameTypedParams[id.Name] {
				aliases[let.Name] = id.Name
			}
			continue
		}
		ret, ok := stmt.(*frontend.ReturnStmt)
		if !ok {
			continue
		}
		id, ok := ret.Value.(*frontend.IdentExpr)
		if ok && sameTypedParams[id.Name] {
			return id.Name, true
		}
		if ok {
			if param := aliases[id.Name]; param != "" {
				return param, true
			}
		}
	}
	return "", false
}

func interfaceFunctionReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	returnSig := formatLSPTypeRef(fn.ReturnType)
	functionParams := map[string]bool{}
	valueParams := map[string]bool{}
	for _, param := range fn.Params {
		valueParams[param.Name] = true
		if param.Type.Kind == frontend.TypeRefFunction && formatLSPTypeRef(param.Type) == returnSig {
			functionParams[param.Name] = true
		}
	}
	aliases := map[string]string{}
	for _, stmt := range fn.Body {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if formatLSPTypeRef(s.Type) != returnSig {
				continue
			}
			if path, ok := interfaceFunctionReturnParamPath(s.Value, aliases, functionParams, valueParams); ok {
				aliases[s.Name] = path
			}
			continue
		case *frontend.AssignStmt:
			target, ok := s.Target.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if _, ok := aliases[target.Name]; !ok {
				continue
			}
			if path, ok := interfaceFunctionReturnParamPath(s.Value, aliases, functionParams, valueParams); ok {
				aliases[target.Name] = path
			}
			continue
		case *frontend.ReturnStmt:
			if path, ok := interfaceFunctionReturnParamPath(s.Value, aliases, functionParams, valueParams); ok {
				return path, true
			}
		}
	}
	return "", false
}

func interfaceFunctionMatchReturnBody(fn *frontend.FuncDecl) (string, bool) {
	if fn == nil || fn.ReturnType.Kind != frontend.TypeRefFunction {
		return "", false
	}
	paramTypes := map[string]string{}
	for _, param := range fn.Params {
		paramTypes[param.Name] = formatLSPTypeRef(param.Type)
	}
	for _, stmt := range fn.Body {
		match, ok := stmt.(*frontend.MatchStmt)
		if !ok || match.Value == nil {
			continue
		}
		valueName := interfaceCallbackArgumentName(match.Value)
		if valueName == "" {
			continue
		}
		valueType := paramTypes[valueName]
		if valueType == "" {
			continue
		}
		var b strings.Builder
		fmt.Fprintf(&b, "    match %s:\n", valueName)
		preservedPayload := false
		for _, c := range match.Cases {
			if c.Guard != nil {
				return "", false
			}
			binding, hasBinding := interfacePatternBindingName(c.Pattern)
			pattern := "_"
			if !c.Default {
				pattern = interfaceFunctionMatchPattern(c.Pattern, valueType)
			}
			fmt.Fprintf(&b, "    case %s:\n", pattern)
			ret, ok := singleReturnExpr(c.Body)
			if !ok {
				return "", false
			}
			if id, ok := ret.(*frontend.IdentExpr); ok && hasBinding && id.Name == binding {
				fmt.Fprintf(&b, "        return %s\n", id.Name)
				preservedPayload = true
				continue
			}
			if expr, ok := interfaceContractExpr(ret); ok && expr != "" {
				fmt.Fprintf(&b, "        return %s\n", expr)
				continue
			}
			fmt.Fprintf(&b, "        return %s\n", interfaceFunctionClosureLiteral(fn.ReturnType, "            "))
		}
		if preservedPayload {
			return strings.TrimRight(b.String(), "\n"), true
		}
	}
	return "", false
}

func interfacePatternBindingName(expr frontend.Expr) (string, bool) {
	switch e := expr.(type) {
	case *frontend.SomePatternExpr:
		return e.Name, e.Name != ""
	case *frontend.EnumCasePatternExpr:
		if len(e.Bindings) == 0 || e.Bindings[0] == "" {
			return "", false
		}
		return e.Bindings[0], true
	case *frontend.CallExpr:
		if len(e.Args) == 0 {
			return "", false
		}
		id, ok := e.Args[0].(*frontend.IdentExpr)
		if !ok || id.Name == "" {
			return "", false
		}
		return id.Name, true
	default:
		return "", false
	}
}

func interfaceFunctionMatchPattern(expr frontend.Expr, enumType string) string {
	switch e := expr.(type) {
	case *frontend.SomePatternExpr:
		if enumType != "" && !strings.HasSuffix(enumType, "?") {
			return enumType + ".some(" + e.Name + ")"
		}
	case *frontend.CallExpr:
		if enumType != "" && len(e.Args) > 0 {
			names := make([]string, 0, len(e.Args))
			for _, arg := range e.Args {
				id, ok := arg.(*frontend.IdentExpr)
				if !ok {
					return interfaceFormatExpr(expr)
				}
				names = append(names, id.Name)
			}
			return enumType + "." + interfaceShortName(e.Name) + "(" + strings.Join(names, ", ") + ")"
		}
	case *frontend.IdentExpr:
		if enumType != "" && !strings.HasSuffix(enumType, "?") {
			return enumType + "." + e.Name
		}
	case *frontend.EnumCasePatternExpr:
		if e.TypeName == "" && enumType != "" {
			if e.HasPayload {
				return enumType + "." + e.CaseName + "(" + strings.Join(e.Bindings, ", ") + ")"
			}
			return enumType + "." + e.CaseName
		}
	}
	return interfaceFormatExpr(expr)
}

func interfaceFormatExpr(expr frontend.Expr) string {
	var p sourcePrinter
	return p.formatExpr(expr)
}

func interfaceShortName(name string) string {
	if idx := strings.LastIndex(name, "."); idx >= 0 && idx+1 < len(name) {
		return name[idx+1:]
	}
	return name
}

func interfaceFunctionReturnParamPath(expr frontend.Expr, aliases map[string]string, functionParams, valueParams map[string]bool) (string, bool) {
	if id, ok := expr.(*frontend.IdentExpr); ok {
		if functionParams[id.Name] {
			return id.Name, true
		}
		if alias := aliases[id.Name]; alias != "" {
			return alias, true
		}
	}
	name := interfaceCallbackArgumentName(expr)
	if name == "" {
		return "", false
	}
	for paramName := range valueParams {
		if name == paramName || strings.HasPrefix(name, paramName+".") {
			return name, true
		}
	}
	return "", false
}

func interfaceCallbackArgumentName(expr frontend.Expr) string {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name
	case *frontend.FieldAccessExpr:
		base := interfaceCallbackArgumentName(e.Base)
		if base == "" {
			return ""
		}
		return base + "." + e.Field
	default:
		return ""
	}
}

func interfaceFunctionClosureLiteral(ref frontend.TypeRef, bodyIndent string) string {
	params := make([]string, 0, len(ref.Params))
	for i, param := range ref.Params {
		formatted := formatLSPTypeRef(param)
		if i < len(ref.ParamOwnership) && ref.ParamOwnership[i] != "" {
			formatted = ref.ParamOwnership[i] + " " + formatted
		}
		params = append(params, fmt.Sprintf("p%d: %s", i, formatted))
	}
	ret := "?"
	body := "0"
	if ref.Return != nil {
		ret = formatLSPTypeRef(*ref.Return)
		if ref.Return.Kind == frontend.TypeRefFunction {
			body = interfaceFunctionClosureLiteral(*ref.Return, bodyIndent+"    ")
		} else {
			body = interfaceReturnLiteral(*ref.Return)
		}
	}
	out := "fn(" + strings.Join(params, ", ") + ") -> " + ret
	if ref.Throws != nil {
		out += " throws " + formatLSPTypeRef(*ref.Throws)
	}
	if len(ref.Uses) > 0 {
		uses := append([]string(nil), ref.Uses...)
		sort.Strings(uses)
		out += " uses " + strings.Join(uses, ", ")
	}
	return out + ":\n" + bodyIndent + "return " + body
}

func interfaceInlineFunctionClosureLiteral(ref frontend.TypeRef) string {
	params := make([]string, 0, len(ref.Params))
	for i, param := range ref.Params {
		formatted := formatLSPTypeRef(param)
		if i < len(ref.ParamOwnership) && ref.ParamOwnership[i] != "" {
			formatted = ref.ParamOwnership[i] + " " + formatted
		}
		params = append(params, fmt.Sprintf("p%d: %s", i, formatted))
	}
	ret := "?"
	body := "0"
	if ref.Return != nil {
		ret = formatLSPTypeRef(*ref.Return)
		if ref.Return.Kind == frontend.TypeRefFunction {
			body = interfaceInlineFunctionClosureLiteral(*ref.Return)
		} else {
			body = interfaceReturnLiteral(*ref.Return)
		}
	}
	out := "fn(" + strings.Join(params, ", ") + ") -> " + ret
	if ref.Throws != nil {
		out += " throws " + formatLSPTypeRef(*ref.Throws)
	}
	if len(ref.Uses) > 0 {
		uses := append([]string(nil), ref.Uses...)
		sort.Strings(uses)
		out += " uses " + strings.Join(uses, ", ")
	}
	return out + " = " + body
}
