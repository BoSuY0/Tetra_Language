package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
	semanticsgenerics "tetra_language/compiler/internal/semantics/generics"
)

func monomorphizeExpr(
	expr frontend.Expr,
	env map[string]string,
	funcDecls map[string]*frontend.FuncDecl,
	structDecls map[string]*frontend.StructDecl,
	enumDecls map[string]*frontend.EnumDecl,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) (string, error) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return "i32", nil
	case *frontend.BoolLitExpr:
		return "bool", nil
	case *frontend.StringLitExpr:
		return "str", nil
	case *frontend.NoneLitExpr:
		return "none", nil
	case *frontend.SomePatternExpr:
		return "", nil
	case *frontend.EnumCasePatternExpr:
		return "", nil
	case *frontend.IdentExpr:
		if tname, ok := env[e.Name]; ok {
			return tname, nil
		}
		resolved, err := resolveMonomorphizeCallName(e.Name, module, imports, funcDecls, generics, e.At)
		if err == nil {
			if fn, ok := funcDecls[resolved]; ok && len(fn.TypeParams) == 0 {
				return genericTypeName(functionTypeRefFromFuncDecl(fn)), nil
			}
		}
		return "", nil
	case *frontend.FieldAccessExpr:
		baseType, err := monomorphizeExpr(e.Base, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
		if err != nil {
			return "", err
		}
		if decl, ok := structDecls[baseType]; ok {
			for _, field := range decl.Fields {
				if field.Name == e.Field {
					return genericTypeName(qualifyFieldAssignmentPathType(field.Type, baseType)), nil
				}
			}
		}
		return "", nil
	case *frontend.IndexExpr:
		if _, err := monomorphizeExpr(e.Base, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
			return "", err
		}
		if _, err := monomorphizeExpr(e.Index, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
			return "", err
		}
		return "", nil
	case *frontend.UnaryExpr:
		_, err := monomorphizeExpr(e.X, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
		return "i32", err
	case *frontend.BinaryExpr:
		if _, err := monomorphizeExpr(e.Left, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
			return "", err
		}
		if _, err := monomorphizeExpr(e.Right, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
			return "", err
		}
		switch e.Op {
		case frontend.TokenEqEq, frontend.TokenBangEq, frontend.TokenLess, frontend.TokenLessEq, frontend.TokenGreater, frontend.TokenGreaterEq,
			frontend.TokenAmpAmp, frontend.TokenPipePipe:
			return "bool", nil
		default:
			return "i32", nil
		}
	case *frontend.TryExpr:
		return monomorphizeExpr(e.X, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
	case *frontend.CatchExpr:
		resultType, err := monomorphizeExpr(e.Call, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
		if err != nil {
			return "", err
		}
		for _, c := range e.Cases {
			caseEnv := cloneStringMap(env)
			if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
				caseEnv[some.Name] = "i32"
			}
			if !c.Default {
				if _, err := monomorphizeExpr(c.Pattern, caseEnv, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
					return "", err
				}
			}
			if c.Guard != nil {
				if _, err := monomorphizeExpr(c.Guard, caseEnv, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
					return "", err
				}
			}
			if _, err := monomorphizeExpr(c.Value, caseEnv, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return "", err
			}
		}
		return resultType, nil
	case *frontend.AwaitExpr:
		return monomorphizeExpr(e.X, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
	case *frontend.StructLitExpr:
		resolved, err := resolveTypeName(&e.Type, module, imports)
		if err != nil {
			return "", err
		}
		if decl, ok := structDecls[resolved]; ok {
			fields := make(map[string]frontend.TypeRef, len(decl.Fields))
			for _, field := range decl.Fields {
				fields[field.Name] = field.Type
			}
			for i := range e.Fields {
				field := &e.Fields[i]
				declared := fields[field.Name]
				if declared.Kind != frontend.TypeRefFunction {
					continue
				}
				if closure, ok := field.Value.(*frontend.ClosureExpr); ok && closure != nil && closure.Decl != nil && len(closure.Decl.TypeParams) > 0 {
					outerLocals := monomorphizeEnvLocals(env)
					if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
						return "", unsupportedGenericClosureCaptureError(pos, name)
					}
					replacement, specialized, err := monomorphizeGenericClosureLiteralValue(closure, declared, fmt.Sprintf("struct field '%s.%s'", e.Type.Name, field.Name), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
					if err != nil {
						return "", err
					}
					if specialized {
						field.Value = replacement
						continue
					}
				}
				replacement, specialized, err := monomorphizeGenericFunctionValueExpr(field.Value, declared, fmt.Sprintf("struct field '%s.%s'", e.Type.Name, field.Name), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
				if err != nil {
					return "", err
				}
				if specialized {
					field.Value = replacement
				}
			}
		}
		for _, field := range e.Fields {
			if _, err := monomorphizeExpr(field.Value, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return "", err
			}
		}
		e.Type.Name = resolved
		return resolved, nil
	case *frontend.CallExpr:
		if err := monomorphizeFunctionTypedEnumPayloadArgs(e, enumDecls, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
			return "", err
		}
		argTypes := make([]string, 0, len(e.Args))
		for _, arg := range e.Args {
			tname, err := monomorphizeExpr(arg, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
			if err != nil {
				return "", err
			}
			argTypes = append(argTypes, tname)
		}
		if enumTypeName, ok := monomorphizeEnumConstructorTypeName(e, enumDecls, module, imports); ok {
			return enumTypeName, nil
		}

		resolved := e.Name
		if localGenericClosure, ok := env[genericClosureBindingKey(e.Name)]; ok {
			if _, isGeneric := generics[localGenericClosure]; isGeneric {
				resolved = localGenericClosure
			}
		}
		if resolved == e.Name {
			if builtin, ok := ResolveBuiltinAlias(e.Name); ok {
				resolved = builtin
			} else {
				var err error
				resolved, err = resolveMonomorphizeCallName(e.Name, module, imports, funcDecls, generics, e.At)
				if err != nil {
					return "", err
				}
			}
		}
		generic, ok := generics[resolved]
		if !ok {
			if callee, exists := funcDecls[resolved]; exists {
				if err := monomorphizeFunctionTypedCallArgs(e, callee, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
					return "", err
				}
				return monomorphizeConcreteReturnType(callee, resolved, module, imports)
			}
			return "", nil
		}
		if len(e.Args) != len(generic.decl.Params) {
			return "", fmt.Errorf("%s: wrong argument count for generic function '%s'", frontend.FormatPos(e.At), e.Name)
		}
		subst := map[string]string{}
		for i, param := range generic.decl.Params {
			if argTypes[i] == "" {
				return "", fmt.Errorf("%s: cannot infer generic argument for '%s' arg %d", frontend.FormatPos(e.Args[i].Pos()), e.Name, i+1)
			}
			if err := bindGenericType(param.Type, argTypes[i], generic.decl.TypeParams, subst); err != nil {
				return "", fmt.Errorf("%s: %v", frontend.FormatPos(e.Args[i].Pos()), err)
			}
		}
		for _, tp := range generic.decl.TypeParams {
			if subst[tp] == "" {
				return "", fmt.Errorf("%s: cannot infer generic argument '%s' for '%s'", frontend.FormatPos(e.At), tp, e.Name)
			}
		}
		if err := checkGenericProtocolBounds(e.At, e.Name, generic, subst, module); err != nil {
			return "", err
		}
		name := mangleGenericName(generic.decl.Name, generic.decl.TypeParams, subst)
		fullName := genericInstanceFullName(generic, name)
		var returnType frontend.TypeRef
		if existing, exists := created[fullName]; exists {
			returnType = existing.ReturnType
		} else {
			clone := cloneGenericFunc(generic.decl, name, subst)
			cloneImports := fileImports[generic.file]
			if structCtx != nil {
				if err := rewriteGenericFuncStructRefs(structCtx, clone, generic.module, cloneImports); err != nil {
					return "", err
				}
			}
			returnType = clone.ReturnType
			created[fullName] = clone
			if _, ok := createdByFile[generic.file]; !ok {
				createdByFile[generic.file] = map[string]*frontend.FuncDecl{}
			}
			createdByFile[generic.file][name] = clone
			*work = append(*work, genericWorkItem{fn: clone, module: generic.module, imports: fileImports[generic.file]})
		}
		e.Name = fullName
		return genericTypeName(returnType), nil
	case *frontend.ClosureExpr:
		return "ptr", nil
	default:
		return "", nil
	}
}

func checkGenericProtocolBounds(callPos frontend.Position, callName string, generic genericDef, subst map[string]string, callerModule string) error {
	for _, bound := range generic.decl.TypeParamBounds {
		actual := subst[bound.Name]
		if actual == "" {
			continue
		}
		boundRef := bound.Bound
		protoName, err := resolveTypeName(&boundRef, generic.module, generic.imports)
		if err != nil {
			return err
		}
		protoInfo, ok := generic.protocols[protoName]
		if !ok {
			if _, isType := generic.knownTypes[protoName]; isType {
				return fmt.Errorf("%s: generic bound '%s' for '%s' must name a protocol, got non-protocol type '%s'",
					frontend.FormatPos(bound.Bound.At),
					displayTypeName(protoName, generic.module),
					bound.Name,
					displayTypeName(protoName, generic.module),
				)
			}
			return fmt.Errorf("%s: unknown protocol bound '%s' for generic parameter '%s'",
				frontend.FormatPos(bound.Bound.At),
				displayTypeName(protoName, generic.module),
				bound.Name,
			)
		}
		if !symbolBelongsToModule(protoName, callerModule) && !protoInfo.public {
			return fmt.Errorf("%s: private protocol '%s' is not visible from module '%s'",
				frontend.FormatPos(callPos),
				protoName,
				callerModule,
			)
		}
		if _, ok := generic.conformances[protocolConformanceKey{typeName: actual, protocol: protoName}]; ok {
			continue
		}
		return fmt.Errorf("%s: generic argument '%s' does not satisfy bound '%s' for '%s' in call to '%s' (missing impl %s: %s)",
			frontend.FormatPos(callPos),
			displayTypeName(actual, generic.module),
			displayTypeName(protoName, generic.module),
			bound.Name,
			callName,
			displayTypeName(actual, generic.module),
			displayTypeName(protoName, generic.module),
		)
	}
	return nil
}

func rewriteGenericFuncStructRefs(ctx *genericStructContext, fn *frontend.FuncDecl, module string, imports map[string]string) error {
	if err := ctx.rewriteTypeRef(&fn.ReturnType, module, imports); err != nil {
		return err
	}
	if fn.HasThrows {
		if err := ctx.rewriteTypeRef(&fn.Throws, module, imports); err != nil {
			return err
		}
	}
	for i := range fn.Params {
		if err := ctx.rewriteTypeRef(&fn.Params[i].Type, module, imports); err != nil {
			return err
		}
	}
	return ctx.rewriteStmts(fn.Body, module, imports)
}

func monomorphizeIterableElemType(typeName string) (string, bool) {
	if strings.HasPrefix(typeName, "[]") {
		return strings.TrimPrefix(typeName, "[]"), true
	}
	if _, elem, ok := parseArrayTypeName(typeName); ok {
		return elem, true
	}
	if typeName == "str" {
		return "u8", true
	}
	return "", false
}

func bindGenericType(param frontend.TypeRef, actual string, typeParams []string, subst map[string]string) error {
	if param.Kind == frontend.TypeRefNamed && contains(typeParams, param.Name) {
		if existing := subst[param.Name]; existing != "" && existing != actual {
			return fmt.Errorf("conflicting generic argument for '%s': %s vs %s", param.Name, existing, actual)
		}
		subst[param.Name] = actual
		return nil
	}
	switch param.Kind {
	case frontend.TypeRefNamed:
		if len(param.TypeArgs) == 0 {
			return nil
		}
		if matched, err := bindGenericNamedTypeArgs(param, actual, typeParams, subst); err != nil || matched {
			return err
		}
	case frontend.TypeRefFunction:
		actualParams, actualReturn, ok := parseGenericFunctionTypeName(actual)
		if !ok {
			return nil
		}
		if len(param.Params) != len(actualParams) {
			return nil
		}
		for i := range param.Params {
			if err := bindGenericType(param.Params[i], actualParams[i], typeParams, subst); err != nil {
				return err
			}
		}
		if param.Return != nil {
			return bindGenericType(*param.Return, actualReturn, typeParams, subst)
		}
	case frontend.TypeRefOptional:
		if param.Elem == nil {
			return nil
		}
		elemActual := actual
		if elem, ok := optionalElemName(actual); ok {
			elemActual = elem
		}
		return bindGenericType(*param.Elem, elemActual, typeParams, subst)
	case frontend.TypeRefSlice:
		if param.Elem == nil {
			return nil
		}
		if elemActual, ok := sliceElemName(actual); ok {
			return bindGenericType(*param.Elem, elemActual, typeParams, subst)
		}
	case frontend.TypeRefArray:
		if param.Elem == nil {
			return nil
		}
		if elemActual, ok := arrayElemName(actual, param.Len); ok {
			return bindGenericType(*param.Elem, elemActual, typeParams, subst)
		}
	}
	return nil
}

func bindGenericNamedTypeArgs(param frontend.TypeRef, actual string, typeParams []string, subst map[string]string) (bool, error) {
	paramBase := lastNameSegment(param.Name)
	actualBase := lastNameSegment(actual)
	prefix := paramBase + "__"
	if paramBase == "" || !strings.HasPrefix(actualBase, prefix) {
		return false, nil
	}
	rest := strings.TrimPrefix(actualBase, prefix)
	for i, arg := range param.TypeArgs {
		argName := genericNamedTypeArgSegmentName(arg)
		if argName == "" {
			return false, nil
		}
		marker := argName + "_"
		if !strings.HasPrefix(rest, marker) {
			return false, nil
		}
		rest = strings.TrimPrefix(rest, marker)
		var encoded string
		if i+1 < len(param.TypeArgs) {
			nextArgName := genericNamedTypeArgSegmentName(param.TypeArgs[i+1])
			if nextArgName == "" {
				return false, nil
			}
			nextMarker := "__" + nextArgName + "_"
			idx := strings.Index(rest, nextMarker)
			if idx < 0 {
				return false, nil
			}
			encoded = rest[:idx]
			rest = rest[idx+len("__"):]
		} else {
			encoded = rest
			rest = ""
		}
		concrete, err := unsanitizeGenericType(encoded)
		if err != nil {
			return true, err
		}
		if canonical, ok := canonicalBuiltinType(concrete); ok {
			concrete = canonical
		}
		if err := bindGenericType(arg, concrete, typeParams, subst); err != nil {
			return true, err
		}
	}
	if rest != "" {
		return false, nil
	}
	return true, nil
}

func genericNamedTypeArgSegmentName(ref frontend.TypeRef) string {
	if ref.Kind == frontend.TypeRefNamed && ref.Name != "" {
		return ref.Name
	}
	return genericTypeName(ref)
}

func arrayElemName(name string, wantLen int) (string, bool) {
	gotLen, elem, ok := parseArrayTypeName(name)
	if !ok || gotLen != wantLen {
		return "", false
	}
	return elem, true
}

func cloneGenericFunc(fn *frontend.FuncDecl, name string, subst map[string]string) *frontend.FuncDecl {
	out := *fn
	out.Name = name
	out.ExportName = ""
	out.TypeParams = nil
	out.TypeParamBounds = nil
	out.Params = cloneParams(fn.Params, subst)
	out.ReturnType = substituteTypeRef(fn.ReturnType, subst)
	out.Throws = substituteTypeRef(fn.Throws, subst)
	out.Uses = append([]string(nil), fn.Uses...)
	out.SemanticClauses = append([]frontend.SemanticClause(nil), fn.SemanticClauses...)
	out.Body = cloneStmts(fn.Body, subst)
	return &out
}

func cloneParams(params []frontend.ParamDecl, subst map[string]string) []frontend.ParamDecl {
	out := make([]frontend.ParamDecl, len(params))
	for i, p := range params {
		out[i] = p
		out[i].Type = substituteTypeRef(p.Type, subst)
	}
	return out
}

func cloneStmts(stmts []frontend.Stmt, subst map[string]string) []frontend.Stmt {
	out := make([]frontend.Stmt, 0, len(stmts))
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			out = append(out, &frontend.ReturnStmt{At: s.At, Value: cloneExpr(s.Value, subst)})
		case *frontend.ThrowStmt:
			out = append(out, &frontend.ThrowStmt{At: s.At, Value: cloneExpr(s.Value, subst)})
		case *frontend.DeferStmt:
			out = append(out, &frontend.DeferStmt{At: s.At, Body: cloneStmts(s.Body, subst)})
		case *frontend.BreakStmt:
			out = append(out, &frontend.BreakStmt{At: s.At})
		case *frontend.ContinueStmt:
			out = append(out, &frontend.ContinueStmt{At: s.At})
		case *frontend.PrintStmt:
			out = append(out, &frontend.PrintStmt{At: s.At, Value: cloneExpr(s.Value, subst)})
		case *frontend.ExpectStmt:
			out = append(out, &frontend.ExpectStmt{At: s.At, Cond: cloneExpr(s.Cond, subst)})
		case *frontend.FreeStmt:
			out = append(out, &frontend.FreeStmt{At: s.At, Value: cloneExpr(s.Value, subst), Implicit: s.Implicit})
		case *frontend.LetStmt:
			out = append(out, &frontend.LetStmt{At: s.At, Name: s.Name, Type: substituteTypeRef(s.Type, subst), Mutable: s.Mutable, Const: s.Const, Value: cloneExpr(s.Value, subst)})
		case *frontend.AssignStmt:
			var compoundValue frontend.Expr
			if s.CompoundValue != nil {
				compoundValue = cloneExpr(s.CompoundValue, subst)
			}
			out = append(out, &frontend.AssignStmt{At: s.At, Target: cloneExpr(s.Target, subst), Value: cloneExpr(s.Value, subst), Op: s.Op, CompoundValue: compoundValue})
		case *frontend.IfStmt:
			out = append(out, &frontend.IfStmt{At: s.At, Cond: cloneExpr(s.Cond, subst), Then: cloneStmts(s.Then, subst), Else: cloneStmts(s.Else, subst)})
		case *frontend.IfLetStmt:
			out = append(out, &frontend.IfLetStmt{At: s.At, Name: s.Name, Pattern: cloneExpr(s.Pattern, subst), Value: cloneExpr(s.Value, subst), ValueLocal: s.ValueLocal, Then: cloneStmts(s.Then, subst), Else: cloneStmts(s.Else, subst)})
		case *frontend.WhileStmt:
			out = append(out, &frontend.WhileStmt{At: s.At, Cond: cloneExpr(s.Cond, subst), Body: cloneStmts(s.Body, subst)})
		case *frontend.ForRangeStmt:
			var start, end, iterable frontend.Expr
			if s.Start != nil {
				start = cloneExpr(s.Start, subst)
			}
			if s.End != nil {
				end = cloneExpr(s.End, subst)
			}
			if s.Iterable != nil {
				iterable = cloneExpr(s.Iterable, subst)
			}
			out = append(out, &frontend.ForRangeStmt{At: s.At, Name: s.Name, Start: start, End: end, Iterable: iterable, IterableLocal: s.IterableLocal, IndexLocal: s.IndexLocal, EndLocal: s.EndLocal, Body: cloneStmts(s.Body, subst)})
		case *frontend.MatchStmt:
			cases := make([]frontend.MatchCase, 0, len(s.Cases))
			for _, c := range s.Cases {
				cases = append(cases, frontend.MatchCase{At: c.At, Pattern: cloneExpr(c.Pattern, subst), Guard: cloneExpr(c.Guard, subst), Default: c.Default, Body: cloneStmts(c.Body, subst)})
			}
			out = append(out, &frontend.MatchStmt{At: s.At, Value: cloneExpr(s.Value, subst), ScrutineeLocal: s.ScrutineeLocal, Cases: cases})
		case *frontend.UnsafeStmt:
			out = append(out, &frontend.UnsafeStmt{At: s.At, Body: cloneStmts(s.Body, subst)})
		case *frontend.IslandStmt:
			out = append(out, &frontend.IslandStmt{At: s.At, Size: cloneExpr(s.Size, subst), Name: s.Name, Body: cloneStmts(s.Body, subst)})
		case *frontend.ExprStmt:
			out = append(out, &frontend.ExprStmt{At: s.At, Expr: cloneExpr(s.Expr, subst)})
		default:
			out = append(out, stmt)
		}
	}
	return out
}

func cloneExpr(expr frontend.Expr, subst map[string]string) frontend.Expr {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return &frontend.NumberExpr{At: e.At, Value: e.Value}
	case *frontend.BoolLitExpr:
		return &frontend.BoolLitExpr{At: e.At, Value: e.Value}
	case *frontend.NoneLitExpr:
		return &frontend.NoneLitExpr{At: e.At}
	case *frontend.SomePatternExpr:
		return &frontend.SomePatternExpr{At: e.At, Name: e.Name}
	case *frontend.EnumCasePatternExpr:
		return &frontend.EnumCasePatternExpr{
			At:           e.At,
			TypeName:     e.TypeName,
			CaseName:     e.CaseName,
			Bindings:     append([]string(nil), e.Bindings...),
			HasPayload:   e.HasPayload,
			EnumType:     e.EnumType,
			EnumOrdinal:  e.EnumOrdinal,
			PayloadSlots: append([]int(nil), e.PayloadSlots...),
		}
	case *frontend.StringLitExpr:
		return &frontend.StringLitExpr{At: e.At, Value: append([]byte(nil), e.Value...)}
	case *frontend.IdentExpr:
		return &frontend.IdentExpr{At: e.At, Name: e.Name}
	case *frontend.FieldAccessExpr:
		return &frontend.FieldAccessExpr{At: e.At, Base: cloneExpr(e.Base, subst), Field: e.Field, EnumType: e.EnumType, EnumOrdinal: e.EnumOrdinal}
	case *frontend.IndexExpr:
		return &frontend.IndexExpr{At: e.At, Base: cloneExpr(e.Base, subst), Index: cloneExpr(e.Index, subst)}
	case *frontend.UnaryExpr:
		return &frontend.UnaryExpr{At: e.At, Op: e.Op, X: cloneExpr(e.X, subst)}
	case *frontend.BinaryExpr:
		return &frontend.BinaryExpr{At: e.At, Op: e.Op, Left: cloneExpr(e.Left, subst), Right: cloneExpr(e.Right, subst)}
	case *frontend.TryExpr:
		return &frontend.TryExpr{At: e.At, X: cloneExpr(e.X, subst)}
	case *frontend.CatchExpr:
		cases := make([]frontend.CatchExprCase, 0, len(e.Cases))
		for _, c := range e.Cases {
			cases = append(cases, frontend.CatchExprCase{At: c.At, Pattern: cloneExpr(c.Pattern, subst), Guard: cloneExpr(c.Guard, subst), Default: c.Default, Value: cloneExpr(c.Value, subst)})
		}
		return &frontend.CatchExpr{At: e.At, Call: cloneExpr(e.Call, subst), ErrorLocal: e.ErrorLocal, ResultLocal: e.ResultLocal, ErrorType: e.ErrorType, ResultType: e.ResultType, Cases: cases}
	case *frontend.AwaitExpr:
		return &frontend.AwaitExpr{At: e.At, X: cloneExpr(e.X, subst)}
	case *frontend.CallExpr:
		args := make([]frontend.Expr, 0, len(e.Args))
		for _, arg := range e.Args {
			args = append(args, cloneExpr(arg, subst))
		}
		labels := append([]string(nil), e.ArgLabels...)
		typeArgs := make([]frontend.TypeRef, len(e.TypeArgs))
		for i := range e.TypeArgs {
			typeArgs[i] = substituteTypeRef(e.TypeArgs[i], subst)
		}
		return &frontend.CallExpr{At: e.At, Name: e.Name, TypeArgs: typeArgs, Args: args, ArgLabels: labels, ResolvedType: e.ResolvedType}
	case *frontend.StructLitExpr:
		fields := make([]frontend.StructFieldInit, 0, len(e.Fields))
		for _, field := range e.Fields {
			fields = append(fields, frontend.StructFieldInit{At: field.At, Name: field.Name, Value: cloneExpr(field.Value, subst)})
		}
		return &frontend.StructLitExpr{At: e.At, Type: substituteTypeRef(e.Type, subst), Fields: fields}
	case *frontend.ClosureExpr:
		return &frontend.ClosureExpr{At: e.At, Name: e.Name, Decl: e.Decl, Captures: cloneClosureCaptures(e.Captures, subst)}
	default:
		return expr
	}
}

func cloneClosureCaptures(captures []frontend.ClosureCapture, subst map[string]string) []frontend.ClosureCapture {
	if len(captures) == 0 {
		return nil
	}
	out := make([]frontend.ClosureCapture, len(captures))
	for i, capture := range captures {
		out[i] = frontend.ClosureCapture{
			At:      capture.At,
			Name:    capture.Name,
			Type:    substituteTypeRef(capture.Type, subst),
			Mutable: capture.Mutable,
		}
	}
	return out
}

func substituteTypeRef(ref frontend.TypeRef, subst map[string]string) frontend.TypeRef {
	out := ref
	if len(ref.TypeArgs) > 0 {
		out.TypeArgs = make([]frontend.TypeRef, len(ref.TypeArgs))
		for i := range ref.TypeArgs {
			out.TypeArgs[i] = substituteTypeRef(ref.TypeArgs[i], subst)
		}
	}
	if len(ref.Params) > 0 {
		out.Params = make([]frontend.TypeRef, len(ref.Params))
		for i := range ref.Params {
			out.Params[i] = substituteTypeRef(ref.Params[i], subst)
		}
	}
	if ref.Kind == frontend.TypeRefNamed {
		if concrete := subst[ref.Name]; concrete != "" {
			return typeRefFromGenericTypeName(ref.At, concrete)
		}
		return out
	}
	if ref.Return != nil {
		ret := substituteTypeRef(*ref.Return, subst)
		out.Return = &ret
	}
	if ref.Elem != nil {
		elem := substituteTypeRef(*ref.Elem, subst)
		out.Elem = &elem
	}
	return out
}

func typeRefFromGenericTypeName(at frontend.Position, name string) frontend.TypeRef {
	if params, ret, ok := parseGenericFunctionTypeName(name); ok {
		paramRefs := make([]frontend.TypeRef, 0, len(params))
		for _, param := range params {
			paramRefs = append(paramRefs, typeRefFromGenericTypeName(at, param))
		}
		retRef := typeRefFromGenericTypeName(at, ret)
		return frontend.TypeRef{At: at, Kind: frontend.TypeRefFunction, Params: paramRefs, Return: &retRef}
	}
	if elem, ok := optionalElemName(name); ok {
		elemRef := typeRefFromGenericTypeName(at, elem)
		return frontend.TypeRef{At: at, Kind: frontend.TypeRefOptional, Elem: &elemRef}
	}
	if elem, ok := sliceElemName(name); ok {
		elemRef := typeRefFromGenericTypeName(at, elem)
		return frontend.TypeRef{At: at, Kind: frontend.TypeRefSlice, Elem: &elemRef}
	}
	if n, elem, ok := parseArrayTypeName(name); ok {
		elemRef := typeRefFromGenericTypeName(at, elem)
		return frontend.TypeRef{At: at, Kind: frontend.TypeRefArray, Elem: &elemRef, Len: n}
	}
	return frontend.TypeRef{At: at, Kind: frontend.TypeRefNamed, Name: name}
}

func parseGenericFunctionTypeName(name string) ([]string, string, bool) {
	name = strings.TrimSpace(name)
	if !strings.HasPrefix(name, "fn(") {
		return nil, "", false
	}
	depth := 1
	closeIdx := -1
	for i := len("fn("); i < len(name); i++ {
		switch name[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				closeIdx = i
				i = len(name)
			}
		}
	}
	if closeIdx < 0 {
		return nil, "", false
	}
	rest := strings.TrimSpace(name[closeIdx+1:])
	if !strings.HasPrefix(rest, "->") {
		return nil, "", false
	}
	paramsText := name[len("fn("):closeIdx]
	params := []string{}
	if strings.TrimSpace(paramsText) != "" {
		params = splitGenericTypeList(paramsText)
	}
	ret := strings.TrimSpace(rest[len("->"):])
	if idx := strings.Index(ret, " uses "); idx >= 0 {
		ret = strings.TrimSpace(ret[:idx])
	}
	if idx := strings.Index(ret, " throws "); idx >= 0 {
		ret = strings.TrimSpace(ret[:idx])
	}
	if ret == "" {
		return nil, "", false
	}
	return params, ret, true
}

func splitGenericTypeList(text string) []string {
	var out []string
	start := 0
	parenDepth := 0
	angleDepth := 0
	bracketDepth := 0
	for i, r := range text {
		switch r {
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '<':
			angleDepth++
		case '>':
			if angleDepth > 0 {
				angleDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case ',':
			if parenDepth == 0 && angleDepth == 0 && bracketDepth == 0 {
				out = append(out, strings.TrimSpace(text[start:i]))
				start = i + len(string(r))
			}
		}
	}
	out = append(out, strings.TrimSpace(text[start:]))
	return out
}

func substituteGenericTypeName(ref frontend.TypeRef, subst map[string]string) string {
	return genericTypeName(substituteTypeRef(ref, subst))
}

func mangleGenericName(base string, order []string, subst map[string]string) string {
	return semanticsgenerics.MangleName(base, order, subst)
}

func sanitizeGenericType(tname string) string {
	return semanticsgenerics.SanitizeType(tname)
}

func unsanitizeGenericType(tname string) (string, error) {
	return semanticsgenerics.UnsanitizeType(tname)
}

func genericTypeName(ref frontend.TypeRef) string {
	return semanticsgenerics.TypeName(ref, canonicalBuiltinType)
}

const genericClosureBindingPrefix = semanticsgenerics.ClosureBindingPrefix

func genericClosureBindingKey(name string) string {
	return semanticsgenerics.ClosureBindingKey(name)
}

func monomorphizeEnvLocals(env map[string]string) map[string]LocalInfo {
	locals := make(map[string]LocalInfo, len(env))
	for name, typeName := range env {
		if strings.HasPrefix(name, genericClosureBindingPrefix) || typeName == "" {
			continue
		}
		locals[name] = LocalInfo{TypeName: typeName}
	}
	return locals
}

func cloneStringMap(src map[string]string) map[string]string {
	return semanticsgenerics.CloneStringMap(src)
}

func cloneFunctionTypeMap(src map[string]frontend.TypeRef) map[string]frontend.TypeRef {
	return semanticsgenerics.CloneFunctionTypeMap(src)
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
