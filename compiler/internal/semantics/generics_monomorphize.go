package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

func monomorphizeStmts(
	stmts []frontend.Stmt,
	env map[string]string,
	functionLocals map[string]frontend.TypeRef,
	returnType frontend.TypeRef,
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
) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			if returnType.Kind == frontend.TypeRefFunction {
				if closure, ok := s.Value.(*frontend.ClosureExpr); ok && closure != nil && closure.Decl != nil && len(closure.Decl.TypeParams) > 0 {
					outerLocals := monomorphizeEnvLocals(env)
					if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
						return unsupportedGenericClosureCaptureError(pos, name)
					}
					replacement, specialized, err := monomorphizeGenericClosureLiteralValue(closure, returnType, "function return", generics, created, createdByFile, work, fileImports, module, imports, structCtx)
					if err != nil {
						return err
					}
					if specialized {
						s.Value = replacement
					}
				}
				replacement, specialized, err := monomorphizeGenericFunctionValueExpr(s.Value, returnType, "function return", generics, created, createdByFile, work, fileImports, module, imports, structCtx)
				if err != nil {
					return err
				}
				if specialized {
					s.Value = replacement
				}
			}
			if _, err := monomorphizeExpr(s.Value, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.ThrowStmt:
			if _, err := monomorphizeExpr(s.Value, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.DeferStmt:
			if err := monomorphizeStmts(s.Body, cloneStringMap(env), cloneFunctionTypeMap(functionLocals), returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.BreakStmt, *frontend.ContinueStmt:
		case *frontend.PrintStmt:
			if _, err := monomorphizeExpr(s.Value, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.ExpectStmt:
			if _, err := monomorphizeExpr(s.Cond, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.FreeStmt:
			if _, err := monomorphizeExpr(s.Value, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.LetStmt:
			if err := monomorphizeFunctionTypedBinding(s, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			valType, err := monomorphizeExpr(s.Value, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
			if err != nil {
				return err
			}
			if s.Type.Name != "" || s.Type.Elem != nil {
				resolved, err := resolveTypeName(&s.Type, module, imports)
				if err != nil {
					return err
				}
				env[s.Name] = resolved
			} else if s.Type.Kind == frontend.TypeRefFunction {
				functionLocals[s.Name] = s.Type
				env[s.Name] = genericTypeName(s.Type)
			} else {
				env[s.Name] = valType
			}
			closureBindingKey := genericClosureBindingKey(s.Name)
			delete(env, closureBindingKey)
			if !s.Mutable {
				if closure, ok := s.Value.(*frontend.ClosureExpr); ok && closure != nil && closure.Decl != nil && len(closure.Decl.TypeParams) > 0 {
					outerLocals := monomorphizeEnvLocals(env)
					if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
						return unsupportedGenericClosureCaptureError(pos, name)
					}
					env[closureBindingKey] = qualifyName(module, closure.Name)
				}
			}
		case *frontend.AssignStmt:
			if _, err := monomorphizeExpr(s.Target, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				if declared, exists := functionLocals[id.Name]; exists && declared.Kind == frontend.TypeRefFunction {
					if closure, ok := s.Value.(*frontend.ClosureExpr); ok && closure != nil && closure.Decl != nil && len(closure.Decl.TypeParams) > 0 {
						outerLocals := monomorphizeEnvLocals(env)
						if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
							return unsupportedGenericClosureCaptureError(pos, name)
						}
						replacement, specialized, err := monomorphizeGenericClosureLiteralValue(closure, declared, fmt.Sprintf("function-typed assignment to '%s'", id.Name), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
						if err != nil {
							return err
						}
						if specialized {
							s.Value = replacement
						}
					}
					replacement, specialized, err := monomorphizeGenericFunctionValueExpr(s.Value, declared, fmt.Sprintf("function-typed assignment to '%s'", id.Name), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
					if err != nil {
						return err
					}
					if specialized {
						s.Value = replacement
					}
				}
			} else if declared, targetName, ok := functionTypeForFieldAssignmentTarget(s.Target, env, structDecls); ok && declared.Kind == frontend.TypeRefFunction {
				if closure, ok := s.Value.(*frontend.ClosureExpr); ok && closure != nil && closure.Decl != nil && len(closure.Decl.TypeParams) > 0 {
					outerLocals := monomorphizeEnvLocals(env)
					if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
						return unsupportedGenericClosureCaptureError(pos, name)
					}
					replacement, specialized, err := monomorphizeGenericClosureLiteralValue(closure, declared, fmt.Sprintf("function-typed assignment to '%s'", targetName), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
					if err != nil {
						return err
					}
					if specialized {
						s.Value = replacement
					}
				}
				replacement, specialized, err := monomorphizeGenericFunctionValueExpr(s.Value, declared, fmt.Sprintf("function-typed assignment to '%s'", targetName), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
				if err != nil {
					return err
				}
				if specialized {
					s.Value = replacement
				}
			}
			if _, err := monomorphizeExpr(s.Value, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.IfStmt:
			if _, err := monomorphizeExpr(s.Cond, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			if err := monomorphizeStmts(s.Then, cloneStringMap(env), cloneFunctionTypeMap(functionLocals), returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			if err := monomorphizeStmts(s.Else, cloneStringMap(env), cloneFunctionTypeMap(functionLocals), returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.IfLetStmt:
			valueType, err := monomorphizeExpr(s.Value, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
			if err != nil {
				return err
			}
			thenEnv := cloneStringMap(env)
			if elem, ok := optionalElemName(valueType); ok {
				thenEnv[s.Name] = elem
			} else {
				thenEnv[s.Name] = "i32"
			}
			if err := monomorphizeStmts(s.Then, thenEnv, cloneFunctionTypeMap(functionLocals), returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			if err := monomorphizeStmts(s.Else, cloneStringMap(env), cloneFunctionTypeMap(functionLocals), returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.WhileStmt:
			if _, err := monomorphizeExpr(s.Cond, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			if err := monomorphizeStmts(s.Body, cloneStringMap(env), cloneFunctionTypeMap(functionLocals), returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.ForRangeStmt:
			bodyEnv := cloneStringMap(env)
			if s.Iterable != nil {
				iterType, err := monomorphizeExpr(s.Iterable, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
				if err != nil {
					return err
				}
				if elem, ok := monomorphizeIterableElemType(iterType); ok {
					bodyEnv[s.Name] = elem
				} else {
					bodyEnv[s.Name] = "i32"
				}
			} else {
				if _, err := monomorphizeExpr(s.Start, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
					return err
				}
				if _, err := monomorphizeExpr(s.End, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
					return err
				}
				bodyEnv[s.Name] = "i32"
			}
			if err := monomorphizeStmts(s.Body, bodyEnv, cloneFunctionTypeMap(functionLocals), returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.MatchStmt:
			scrutType, err := monomorphizeExpr(s.Value, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
			if err != nil {
				return err
			}
			someElemType, hasSomeElem := optionalElemName(scrutType)
			for _, c := range s.Cases {
				caseEnv := cloneStringMap(env)
				if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok && hasSomeElem {
					caseEnv[some.Name] = someElemType
				}
				if !c.Default {
					if _, err := monomorphizeExpr(c.Pattern, caseEnv, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
						return err
					}
				}
				if c.Guard != nil {
					if _, err := monomorphizeExpr(c.Guard, caseEnv, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
						return err
					}
				}
				if err := monomorphizeStmts(c.Body, caseEnv, cloneFunctionTypeMap(functionLocals), returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
					return err
				}
			}
		case *frontend.UnsafeStmt:
			if err := monomorphizeStmts(s.Body, env, functionLocals, returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.IslandStmt:
			if _, err := monomorphizeExpr(s.Size, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			bodyEnv := cloneStringMap(env)
			bodyEnv[s.Name] = "island"
			if err := monomorphizeStmts(s.Body, bodyEnv, cloneFunctionTypeMap(functionLocals), returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.ExprStmt:
			if _, err := monomorphizeExpr(s.Expr, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		}
	}
	return nil
}

func monomorphizeFunctionTypedBinding(
	stmt *frontend.LetStmt,
	env map[string]string,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) error {
	if stmt.Type.Kind != frontend.TypeRefFunction {
		return nil
	}
	if closure, ok := stmt.Value.(*frontend.ClosureExpr); ok && closure != nil && closure.Decl != nil && len(closure.Decl.TypeParams) > 0 {
		outerLocals := monomorphizeEnvLocals(env)
		if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
			return unsupportedGenericClosureCaptureError(pos, name)
		}
		replacement, specialized, err := monomorphizeGenericClosureLiteralValue(closure, stmt.Type, fmt.Sprintf("function-typed local '%s'", stmt.Name), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
		if err != nil || !specialized {
			return err
		}
		stmt.Value = replacement
		return nil
	}
	replacement, specialized, err := monomorphizeGenericFunctionValueExpr(stmt.Value, stmt.Type, fmt.Sprintf("function-typed local '%s'", stmt.Name), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
	if err != nil || !specialized {
		return err
	}
	stmt.Value = replacement
	return nil
}

func monomorphizeGenericClosureLiteralValue(
	closure *frontend.ClosureExpr,
	declared frontend.TypeRef,
	context string,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) (frontend.Expr, bool, error) {
	if declared.Kind != frontend.TypeRefFunction || closure == nil || closure.Decl == nil || len(closure.Decl.TypeParams) == 0 {
		return closure, false, nil
	}
	fullOriginal := qualifyName(module, closure.Name)
	generic, ok := generics[fullOriginal]
	if !ok {
		return closure, false, nil
	}
	declaredParams, declaredReturn, _, err := functionTypeRefSignatureAndEffects(declared, module, imports)
	if err != nil {
		return nil, false, err
	}
	if len(declaredParams) != len(generic.decl.Params) {
		return nil, false, fmt.Errorf("%s: %s generic closure literal parameter count mismatch: expected %d, got %d", frontend.FormatPos(closure.At), context, len(declaredParams), len(generic.decl.Params))
	}
	subst := map[string]string{}
	for i, param := range generic.decl.Params {
		if err := bindGenericType(param.Type, declaredParams[i], generic.decl.TypeParams, subst); err != nil {
			return nil, false, fmt.Errorf("%s: %v", frontend.FormatPos(closure.At), err)
		}
	}
	if err := bindGenericType(generic.decl.ReturnType, declaredReturn, generic.decl.TypeParams, subst); err != nil {
		return nil, false, fmt.Errorf("%s: %v", frontend.FormatPos(closure.At), err)
	}
	for _, tp := range generic.decl.TypeParams {
		if subst[tp] == "" {
			return nil, false, fmt.Errorf("%s: cannot infer generic argument '%s' for %s", frontend.FormatPos(closure.At), tp, context)
		}
	}
	if err := checkGenericProtocolBounds(closure.At, closure.Name, generic, subst, module); err != nil {
		return nil, false, err
	}
	name := mangleGenericName(generic.decl.Name, generic.decl.TypeParams, subst)
	fullName := genericInstanceFullName(generic, name)
	clone, exists := created[fullName]
	if !exists {
		clone = cloneGenericFunc(generic.decl, name, subst)
		cloneImports := fileImports[generic.file]
		if structCtx != nil {
			if err := rewriteGenericFuncStructRefs(structCtx, clone, generic.module, cloneImports); err != nil {
				return nil, false, err
			}
		}
		created[fullName] = clone
		if _, ok := createdByFile[generic.file]; !ok {
			createdByFile[generic.file] = map[string]*frontend.FuncDecl{}
		}
		createdByFile[generic.file][name] = clone
		*work = append(*work, genericWorkItem{fn: clone, module: generic.module, imports: fileImports[generic.file]})
	}
	return &frontend.ClosureExpr{At: closure.At, Name: name, Decl: clone}, true, nil
}

func monomorphizeGenericFunctionValueExpr(
	expr frontend.Expr,
	declared frontend.TypeRef,
	context string,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) (frontend.Expr, bool, error) {
	switch init := expr.(type) {
	case *frontend.IdentExpr:
		fullName, specialized, err := monomorphizeGenericFunctionValue(init, declared, context, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
		if err != nil || !specialized {
			return expr, specialized, err
		}
		init.Name = fullName
		return init, true, nil
	case *frontend.FieldAccessExpr:
		name := callbackArgumentName(init)
		if name == "" {
			return expr, false, nil
		}
		asIdent := &frontend.IdentExpr{At: init.At, Name: name}
		fullName, specialized, err := monomorphizeGenericFunctionValue(asIdent, declared, context, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
		if err != nil || !specialized {
			return expr, specialized, err
		}
		return &frontend.IdentExpr{At: init.At, Name: fullName}, true, nil
	default:
		return expr, false, nil
	}
}

func monomorphizeGenericFunctionValue(
	init *frontend.IdentExpr,
	declared frontend.TypeRef,
	context string,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) (string, bool, error) {
	if declared.Kind != frontend.TypeRefFunction {
		return "", false, nil
	}
	resolved, err := resolveCallName(init.Name, module, imports, init.At)
	if err != nil {
		return "", false, nil
	}
	generic, ok := generics[resolved]
	if !ok {
		return "", false, nil
	}
	declaredParams, declaredReturn, _, err := functionTypeRefSignatureAndEffects(declared, module, imports)
	if err != nil {
		return "", false, err
	}
	if len(declaredParams) != len(generic.decl.Params) {
		return "", false, fmt.Errorf("%s: %s generic function symbol '%s' parameter count mismatch: expected %d, got %d", frontend.FormatPos(init.At), context, init.Name, len(declaredParams), len(generic.decl.Params))
	}
	subst := map[string]string{}
	for i, param := range generic.decl.Params {
		if err := bindGenericType(param.Type, declaredParams[i], generic.decl.TypeParams, subst); err != nil {
			return "", false, fmt.Errorf("%s: %v", frontend.FormatPos(init.At), err)
		}
	}
	if err := bindGenericType(generic.decl.ReturnType, declaredReturn, generic.decl.TypeParams, subst); err != nil {
		return "", false, fmt.Errorf("%s: %v", frontend.FormatPos(init.At), err)
	}
	for _, tp := range generic.decl.TypeParams {
		if subst[tp] == "" {
			return "", false, fmt.Errorf("%s: cannot infer generic argument '%s' for %s", frontend.FormatPos(init.At), tp, context)
		}
	}
	if err := checkGenericProtocolBounds(init.At, init.Name, generic, subst, module); err != nil {
		return "", false, err
	}
	name := mangleGenericName(generic.decl.Name, generic.decl.TypeParams, subst)
	fullName := genericInstanceFullName(generic, name)
	if _, exists := created[fullName]; !exists {
		clone := cloneGenericFunc(generic.decl, name, subst)
		cloneImports := fileImports[generic.file]
		if structCtx != nil {
			if err := rewriteGenericFuncStructRefs(structCtx, clone, generic.module, cloneImports); err != nil {
				return "", false, err
			}
		}
		created[fullName] = clone
		if _, ok := createdByFile[generic.file]; !ok {
			createdByFile[generic.file] = map[string]*frontend.FuncDecl{}
		}
		createdByFile[generic.file][name] = clone
		*work = append(*work, genericWorkItem{fn: clone, module: generic.module, imports: fileImports[generic.file]})
	}
	return fullName, true, nil
}

func monomorphizeFunctionTypedCallArgs(
	call *frontend.CallExpr,
	callee *frontend.FuncDecl,
	env map[string]string,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) error {
	limit := len(call.Args)
	if len(callee.Params) < limit {
		limit = len(callee.Params)
	}
	for i := 0; i < limit; i++ {
		if callee.Params[i].Type.Kind != frontend.TypeRefFunction {
			continue
		}
		if closure, ok := call.Args[i].(*frontend.ClosureExpr); ok && closure != nil && closure.Decl != nil && len(closure.Decl.TypeParams) > 0 {
			outerLocals := monomorphizeEnvLocals(env)
			if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
				return unsupportedGenericClosureCallbackCaptureError(pos, name)
			}
			replacement, specialized, err := monomorphizeGenericClosureLiteralValue(closure, callee.Params[i].Type, fmt.Sprintf("callback argument for '%s'", call.Name), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
			if err != nil {
				return err
			}
			if specialized {
				call.Args[i] = replacement
				continue
			}
		}
		replacement, specialized, err := monomorphizeGenericFunctionValueExpr(call.Args[i], callee.Params[i].Type, fmt.Sprintf("callback argument for '%s'", call.Name), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
		if err != nil {
			return err
		}
		if specialized {
			call.Args[i] = replacement
		}
	}
	return nil
}

func resolveMonomorphizeCallName(
	name string,
	module string,
	imports map[string]string,
	funcDecls map[string]*frontend.FuncDecl,
	generics map[string]genericDef,
	pos frontend.Position,
) (string, error) {
	if _, ok := funcDecls[name]; ok {
		return name, nil
	}
	if _, ok := generics[name]; ok {
		return name, nil
	}
	resolved, err := resolveCallName(name, module, imports, pos)
	if err != nil {
		return "", err
	}
	if _, ok := funcDecls[resolved]; ok {
		return resolved, nil
	}
	if _, ok := generics[resolved]; ok {
		return resolved, nil
	}
	if module != "" && strings.Contains(name, ".") {
		moduleLocal := qualifyName(module, name)
		if _, ok := funcDecls[moduleLocal]; ok {
			return moduleLocal, nil
		}
		if _, ok := generics[moduleLocal]; ok {
			return moduleLocal, nil
		}
	}
	return resolved, nil
}

func monomorphizeConcreteReturnType(callee *frontend.FuncDecl, resolvedName string, module string, imports map[string]string) (string, error) {
	if callee == nil {
		return "", nil
	}
	calleeModule := module
	if callee.Name != "" {
		suffix := "." + callee.Name
		if resolvedName == callee.Name {
			calleeModule = ""
		} else if strings.HasSuffix(resolvedName, suffix) {
			calleeModule = strings.TrimSuffix(resolvedName, suffix)
		}
	}
	ret := substituteTypeRef(callee.ReturnType, nil)
	resolved, err := resolveTypeName(&ret, calleeModule, imports)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func monomorphizeFunctionTypedEnumPayloadArgs(
	call *frontend.CallExpr,
	enumDecls map[string]*frontend.EnumDecl,
	env map[string]string,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) error {
	parts := strings.Split(call.Name, ".")
	if len(parts) < 2 {
		return nil
	}
	caseName := parts[len(parts)-1]
	typeRef := frontend.TypeRef{At: call.At, Kind: frontend.TypeRefNamed, Name: strings.Join(parts[:len(parts)-1], ".")}
	typeName, err := resolveTypeName(&typeRef, module, imports)
	if err != nil {
		return nil
	}
	enumDecl, ok := enumDecls[typeName]
	if !ok {
		return nil
	}
	var enumCase *frontend.EnumCaseDecl
	for i := range enumDecl.Cases {
		if enumDecl.Cases[i].Name == caseName {
			enumCase = &enumDecl.Cases[i]
			break
		}
	}
	if enumCase == nil {
		return nil
	}
	limit := len(call.Args)
	if len(enumCase.Payload) < limit {
		limit = len(enumCase.Payload)
	}
	for i := 0; i < limit; i++ {
		declared := enumCase.Payload[i]
		if declared.Kind != frontend.TypeRefFunction {
			continue
		}
		if closure, ok := call.Args[i].(*frontend.ClosureExpr); ok && closure != nil && closure.Decl != nil && len(closure.Decl.TypeParams) > 0 {
			outerLocals := monomorphizeEnvLocals(env)
			if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
				return unsupportedGenericClosureCaptureError(pos, name)
			}
			replacement, specialized, err := monomorphizeGenericClosureLiteralValue(closure, declared, fmt.Sprintf("enum payload '%s.%s[%d]'", typeRef.Name, caseName, i+1), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
			if err != nil {
				return err
			}
			if specialized {
				call.Args[i] = replacement
				continue
			}
		}
		replacement, specialized, err := monomorphizeGenericFunctionValueExpr(call.Args[i], declared, fmt.Sprintf("enum payload '%s.%s[%d]'", typeRef.Name, caseName, i+1), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
		if err != nil {
			return err
		}
		if specialized {
			call.Args[i] = replacement
		}
	}
	return nil
}

func monomorphizeEnumConstructorTypeName(call *frontend.CallExpr, enumDecls map[string]*frontend.EnumDecl, module string, imports map[string]string) (string, bool) {
	parts := strings.Split(call.Name, ".")
	if len(parts) < 2 {
		return "", false
	}
	caseName := parts[len(parts)-1]
	typeRef := frontend.TypeRef{At: call.At, Kind: frontend.TypeRefNamed, Name: strings.Join(parts[:len(parts)-1], ".")}
	typeName, err := resolveTypeName(&typeRef, module, imports)
	if err != nil {
		return "", false
	}
	enumDecl, ok := enumDecls[typeName]
	if !ok {
		return "", false
	}
	for i := range enumDecl.Cases {
		if enumDecl.Cases[i].Name == caseName {
			return typeName, true
		}
	}
	return "", false
}

func functionTypeForFieldAssignmentTarget(expr frontend.Expr, env map[string]string, structDecls map[string]*frontend.StructDecl) (frontend.TypeRef, string, bool) {
	return functionTypeForFieldAssignmentTargetFrom(expr, "", env, structDecls)
}

func functionTypeRefFromFuncDecl(fn *frontend.FuncDecl) frontend.TypeRef {
	params := make([]frontend.TypeRef, 0, len(fn.Params))
	paramOwnership := make([]string, 0, len(fn.Params))
	for _, param := range fn.Params {
		params = append(params, param.Type)
		paramOwnership = append(paramOwnership, param.Ownership)
	}
	ret := fn.ReturnType
	out := frontend.TypeRef{
		At:             fn.Pos,
		Kind:           frontend.TypeRefFunction,
		Params:         params,
		ParamOwnership: paramOwnership,
		Return:         &ret,
		Uses:           append([]string(nil), fn.Uses...),
	}
	if fn.HasThrows {
		throws := fn.Throws
		out.Throws = &throws
	}
	return out
}

func functionTypeForFieldAssignmentTargetFrom(expr frontend.Expr, path string, env map[string]string, structDecls map[string]*frontend.StructDecl) (frontend.TypeRef, string, bool) {
	switch target := expr.(type) {
	case *frontend.IdentExpr:
		typeName := env[target.Name]
		if typeName == "" {
			return frontend.TypeRef{}, "", false
		}
		if path == "" {
			path = target.Name
		}
		return frontend.TypeRef{Kind: frontend.TypeRefNamed, Name: typeName}, path, true
	case *frontend.FieldAccessExpr:
		baseType, basePath, ok := functionTypeForFieldAssignmentTargetFrom(target.Base, path, env, structDecls)
		if !ok || baseType.Name == "" {
			return frontend.TypeRef{}, "", false
		}
		decl, ok := structDecls[baseType.Name]
		if !ok {
			return frontend.TypeRef{}, "", false
		}
		for _, field := range decl.Fields {
			if field.Name != target.Field {
				continue
			}
			if basePath == "" {
				basePath = target.Field
			} else {
				basePath += "." + target.Field
			}
			return qualifyFieldAssignmentPathType(field.Type, baseType.Name), basePath, true
		}
	}
	return frontend.TypeRef{}, "", false
}

func qualifyFieldAssignmentPathType(ref frontend.TypeRef, ownerType string) frontend.TypeRef {
	if ref.Kind != frontend.TypeRefNamed || ref.Name == "" {
		return ref
	}
	if _, ok := canonicalBuiltinType(ref.Name); ok {
		return ref
	}
	if strings.Contains(ref.Name, ".") {
		return ref
	}
	if idx := strings.LastIndex(ownerType, "."); idx >= 0 {
		ref.Name = qualifyName(ownerType[:idx], ref.Name)
	}
	return ref
}
