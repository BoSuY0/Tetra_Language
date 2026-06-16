package lower

import (
	"sort"
	"strings"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics"
)

func collectFunctionTypedParamTargets(checked *semantics.CheckedProgram, module string) map[string]map[string][]string {
	if checked == nil {
		return nil
	}
	const returnFieldPrefix = "$return."
	const returnFunctionLocal = "$return.fn"
	const returnEnumPayloadLocal = "$return.enum"
	funcsByName := make(map[string]semantics.CheckedFunc, len(checked.Funcs))
	for _, fn := range checked.Funcs {
		funcsByName[fn.Name] = fn
	}
	targetSets := map[string]map[string]map[string]bool{}
	var edges []callableTargetEdge
	var moduleGlobalEdges []moduleGlobalTargetEdge
	var enumPayloadEdges []enumPayloadTargetEdge

	addTarget := func(callee, paramName, targetSymbol string) bool {
		if callee == "" || paramName == "" || targetSymbol == "" {
			return false
		}
		if _, ok := targetSets[callee]; !ok {
			targetSets[callee] = map[string]map[string]bool{}
		}
		if _, ok := targetSets[callee][paramName]; !ok {
			targetSets[callee][paramName] = map[string]bool{}
		}
		if targetSets[callee][paramName][targetSymbol] {
			return false
		}
		targetSets[callee][paramName][targetSymbol] = true
		return true
	}

	addModuleGlobalTarget := func(moduleName, globalName, targetSymbol string) bool {
		changed := false
		for _, fn := range checked.Funcs {
			if fn.Module != moduleName {
				continue
			}
			if addTarget(fn.Name, globalName, targetSymbol) {
				changed = true
			}
		}
		return changed
	}

	enumPayloadSourceName := func(localName, payloadKey string) string {
		return "$enum." + localName + "." + payloadKey
	}

	enumPayloadTargetSets := map[string]map[string]map[string]map[string]bool{}
	addEnumPayloadTarget := func(funcName, localName, payloadKey, targetSymbol string) bool {
		if funcName == "" || localName == "" || payloadKey == "" || targetSymbol == "" {
			return false
		}
		if _, ok := enumPayloadTargetSets[funcName]; !ok {
			enumPayloadTargetSets[funcName] = map[string]map[string]map[string]bool{}
		}
		if _, ok := enumPayloadTargetSets[funcName][localName]; !ok {
			enumPayloadTargetSets[funcName][localName] = map[string]map[string]bool{}
		}
		if _, ok := enumPayloadTargetSets[funcName][localName][payloadKey]; !ok {
			enumPayloadTargetSets[funcName][localName][payloadKey] = map[string]bool{}
		}
		if enumPayloadTargetSets[funcName][localName][payloadKey][targetSymbol] {
			return false
		}
		enumPayloadTargetSets[funcName][localName][payloadKey][targetSymbol] = true
		addTarget(funcName, enumPayloadSourceName(localName, payloadKey), targetSymbol)
		return true
	}

	addEnumPayloadTargetsForLocal := func(caller semantics.CheckedFunc, localName string, payloads map[string]semantics.FunctionFieldInfo) {
		for payloadKey, payload := range payloads {
			if payload.FunctionValue != "" {
				addEnumPayloadTarget(caller.Name, localName, payloadKey, payload.FunctionValue)
			}
		}
	}

	addEnumPayloadFunctionReturnEdgesForLocal := func(caller semantics.CheckedFunc, localName string, value frontend.Expr) {
		call, ok := value.(*frontend.CallExpr)
		if !ok {
			return
		}
		_, caseInfo, ok := enumCaseConstructorInfoForTargets(call, checked.Types)
		if !ok {
			return
		}
		for i, arg := range call.Args {
			if i >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[i] {
				continue
			}
			argCall, ok := arg.(*frontend.CallExpr)
			if !ok {
				continue
			}
			resolved, ok := resolvedCallableFunctionName(argCall.Name, checked.FuncSigs)
			if !ok {
				continue
			}
			if sourceSig, exists := checked.FuncSigs[resolved]; exists && sourceSig.ReturnFunctionType {
				edges = append(edges, callableTargetEdge{
					callee:      caller.Name,
					param:       enumPayloadSourceName(localName, enumPayloadTargetKey(caseInfo.Ordinal, i)),
					sourceFunc:  resolved,
					sourceParam: returnFunctionLocal,
				})
			}
		}
	}
	addEnumPayloadConstructorArgEdgesForLocal := func(caller semantics.CheckedFunc, localName string, value frontend.Expr) {
		call, ok := value.(*frontend.CallExpr)
		if !ok {
			return
		}
		_, caseInfo, ok := enumCaseConstructorInfoForTargets(call, checked.Types)
		if !ok {
			return
		}
		for i, arg := range call.Args {
			if i >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[i] {
				continue
			}
			payloadSource := enumPayloadSourceName(localName, enumPayloadTargetKey(caseInfo.Ordinal, i))
			if target, ok := callableTargetFromAssignedExpr(arg, caller, checked.FuncSigs, checked.GlobalsByModule[caller.Module]); ok {
				addTarget(caller.Name, payloadSource, target)
			}
			if id, ok := arg.(*frontend.IdentExpr); ok {
				if source, exists := caller.Locals[id.Name]; exists && source.FunctionTypeValue {
					edges = append(edges, callableTargetEdge{
						callee:      caller.Name,
						param:       payloadSource,
						sourceFunc:  caller.Name,
						sourceParam: id.Name,
					})
				} else if source, exists := checked.GlobalsByModule[caller.Module][id.Name]; exists && source.FunctionTypeValue {
					edges = append(edges, callableTargetEdge{
						callee:      caller.Name,
						param:       payloadSource,
						sourceFunc:  caller.Name,
						sourceParam: id.Name,
					})
				}
				continue
			}
			if sourceFieldName := functionTypedFieldNameFromExpr(arg); sourceFieldName != "" {
				if _, sourceOK, _ := resolveFunctionFieldName(sourceFieldName, caller.Locals); sourceOK {
					edges = append(edges, callableTargetEdge{
						callee:      caller.Name,
						param:       payloadSource,
						sourceFunc:  caller.Name,
						sourceParam: sourceFieldName,
					})
				}
				continue
			}
			argCall, ok := arg.(*frontend.CallExpr)
			if !ok {
				continue
			}
			resolved, ok := resolvedCallableFunctionName(argCall.Name, checked.FuncSigs)
			if !ok {
				continue
			}
			if sourceSig, exists := checked.FuncSigs[resolved]; exists && sourceSig.ReturnFunctionType {
				edges = append(edges, callableTargetEdge{
					callee:      caller.Name,
					param:       payloadSource,
					sourceFunc:  resolved,
					sourceParam: returnFunctionLocal,
				})
			}
		}
	}

	for _, fn := range checked.Funcs {
		if sig, ok := checked.FuncSigs[fn.Name]; ok {
			if sig.ReturnFunctionType && sig.ReturnFunctionSymbol != "" {
				addTarget(fn.Name, returnFunctionLocal, sig.ReturnFunctionSymbol)
			}
			for fieldName, field := range sig.ReturnFunctionFields {
				if field.FunctionValue != "" {
					addTarget(fn.Name, returnFieldPrefix+fieldName, field.FunctionValue)
				}
			}
			addEnumPayloadTargetsForLocal(fn, returnEnumPayloadLocal, sig.ReturnEnumPayloadFunctions)
			for fieldName, field := range sig.ReturnEnumPayloadFields {
				if field.FunctionValue != "" {
					addTarget(fn.Name, returnFieldPrefix+fieldName, field.FunctionValue)
				}
			}
		}
		for name, local := range fn.Locals {
			if local.FunctionTypeValue && local.FunctionValue != "" {
				addTarget(fn.Name, name, local.FunctionValue)
			}
			for fieldName, field := range local.FunctionFields {
				if field.FunctionValue != "" {
					addTarget(fn.Name, name+"."+fieldName, field.FunctionValue)
				}
			}
			addEnumPayloadTargetsForLocal(fn, name, local.EnumPayloadFunctions)
		}
		for name, global := range checked.GlobalsByModule[fn.Module] {
			if global.FunctionTypeValue && global.FunctionValue != "" {
				addTarget(fn.Name, name, global.FunctionValue)
			}
		}
	}

	var walkExpr func(frontend.Expr, semantics.CheckedFunc)
	var walkStmt func(frontend.Stmt, semantics.CheckedFunc)

	addEdge := func(edge callableTargetEdge) {
		edges = append(edges, edge)
	}
	addEnumPayloadEdge := func(edge enumPayloadTargetEdge) {
		enumPayloadEdges = append(enumPayloadEdges, edge)
	}
	addStructLiteralFieldEdges := func(caller semantics.CheckedFunc, destFunc, destPrefix, structType string, value frontend.Expr, destFields map[string]semantics.FunctionFieldInfo) {
		addStructLiteralFieldEdgesForTargets(caller, destFunc, destPrefix, structType, value, destFields, checked.Types, checked.FuncSigs, checked.GlobalsByModule[caller.Module], addTarget, addEdge)
	}
	addStructLiteralEnumPayloadFieldEdges := func(caller semantics.CheckedFunc, destFunc, destPrefix, structType string, value frontend.Expr, destFields map[string]semantics.FunctionFieldInfo) {
		addStructLiteralEnumPayloadFieldEdgesForTargets(caller, destFunc, destPrefix, structType, value, destFields, checked.Types, checked.FuncSigs, addTarget, addEdge, addEnumPayloadEdge)
	}

	addCallTargets := func(call *frontend.CallExpr, caller semantics.CheckedFunc) {
		if local, ok := caller.Locals[call.Name]; ok && local.FunctionTypeValue && local.FunctionValue != "" {
			addTarget(caller.Name, call.Name, local.FunctionValue)
		}
		resolved := call.Name
		if builtin, ok := semantics.ResolveBuiltinAlias(resolved); ok {
			resolved = builtin
		}
		calleeSig, ok := checked.FuncSigs[resolved]
		if !ok || len(calleeSig.ParamFunctionTypes) == 0 {
			return
		}
		callee, ok := funcsByName[resolved]
		if !ok || len(callee.Decl.Params) == 0 {
			return
		}
		for i, isFuncParam := range calleeSig.ParamFunctionTypes {
			if !isFuncParam || i >= len(call.Args) || i >= len(callee.Decl.Params) {
				continue
			}
			paramName := callee.Decl.Params[i].Name
			if paramName == "" {
				continue
			}
			if closure, ok := call.Args[i].(*frontend.ClosureExpr); ok {
				addTarget(resolved, paramName, callableClosureTargetName(caller, closure, checked.FuncSigs))
				continue
			}
			if fieldName := functionTypedFieldNameFromExpr(call.Args[i]); fieldName != "" {
				if field, ok, _ := resolveFunctionFieldName(fieldName, caller.Locals); ok {
					if field.FunctionValue != "" {
						addTarget(resolved, paramName, field.FunctionValue)
					}
					edges = append(edges, callableTargetEdge{
						callee:      resolved,
						param:       paramName,
						sourceFunc:  caller.Name,
						sourceParam: fieldName,
					})
					continue
				}
				if global, ok := checked.GlobalsByModule[caller.Module][fieldName]; ok && global.FunctionTypeValue {
					if global.FunctionValue != "" {
						addTarget(resolved, paramName, global.FunctionValue)
					}
					edges = append(edges, callableTargetEdge{
						callee:      resolved,
						param:       paramName,
						sourceFunc:  caller.Name,
						sourceParam: fieldName,
					})
					continue
				}
			}
			if argCall, ok := call.Args[i].(*frontend.CallExpr); ok {
				if source, ok := resolvedCallableFunctionName(argCall.Name, checked.FuncSigs); ok {
					if sourceSig, exists := checked.FuncSigs[source]; exists && sourceSig.ReturnFunctionType {
						edges = append(edges, callableTargetEdge{
							callee:      resolved,
							param:       paramName,
							sourceFunc:  source,
							sourceParam: returnFunctionLocal,
						})
						continue
					}
				}
			}
			id, ok := call.Args[i].(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if local, ok := caller.Locals[id.Name]; ok {
				if !local.FunctionTypeValue && local.FunctionValue == "" {
					continue
				}
				if local.FunctionValue != "" {
					addTarget(resolved, paramName, local.FunctionValue)
				}
				edges = append(edges, callableTargetEdge{
					callee:      resolved,
					param:       paramName,
					sourceFunc:  caller.Name,
					sourceParam: id.Name,
				})
				continue
			}
			if global, ok := checked.GlobalsByModule[caller.Module][id.Name]; ok && global.FunctionTypeValue {
				if global.FunctionValue != "" {
					addTarget(resolved, paramName, global.FunctionValue)
				}
				edges = append(edges, callableTargetEdge{
					callee:      resolved,
					param:       paramName,
					sourceFunc:  caller.Name,
					sourceParam: id.Name,
				})
				continue
			}
			if _, ok := checked.FuncSigs[id.Name]; ok {
				addTarget(resolved, paramName, id.Name)
			}
		}
		for i, param := range callee.Decl.Params {
			if i >= len(call.Args) {
				continue
			}
			paramLocal, ok := callee.Locals[param.Name]
			if !ok {
				continue
			}
			if len(paramLocal.FunctionFields) > 0 {
				sourceFields := map[string]semantics.FunctionFieldInfo(nil)
				sourcePrefix := ""
				if id, ok := call.Args[i].(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && len(source.FunctionFields) > 0 {
						sourceFields = source.FunctionFields
						sourcePrefix = id.Name + "."
					}
				}
				addStructLiteralFieldEdges(caller, resolved, param.Name+".", paramLocal.TypeName, call.Args[i], paramLocal.FunctionFields)
				if len(sourceFields) > 0 {
					for fieldName := range paramLocal.FunctionFields {
						destFieldName := param.Name + "." + fieldName
						if source, ok := sourceFields[fieldName]; ok && source.FunctionValue != "" {
							addTarget(resolved, destFieldName, source.FunctionValue)
						}
						edges = append(edges, callableTargetEdge{
							callee:      resolved,
							param:       destFieldName,
							sourceFunc:  caller.Name,
							sourceParam: sourcePrefix + fieldName,
						})
					}
				}
			}
			if len(paramLocal.EnumPayloadFunctions) > 0 {
				if payloads := enumPayloadTargetsFromExpr(call.Args[i], caller, checked.FuncSigs, checked.Types); len(payloads) > 0 {
					addEnumPayloadTargetsForLocal(semantics.CheckedFunc{Name: resolved}, param.Name, payloads)
				}
				if id, ok := call.Args[i].(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && len(source.EnumPayloadFunctions) > 0 {
						for payloadKey := range paramLocal.EnumPayloadFunctions {
							enumPayloadEdges = append(enumPayloadEdges, enumPayloadTargetEdge{
								destFunc:         resolved,
								destLocal:        param.Name,
								destPayloadKey:   payloadKey,
								sourceFunc:       caller.Name,
								sourceLocal:      id.Name,
								sourcePayloadKey: payloadKey,
							})
						}
					}
				} else if argCall, ok := call.Args[i].(*frontend.CallExpr); ok {
					if sourceName, ok := resolvedCallableFunctionName(argCall.Name, checked.FuncSigs); ok {
						if sourceSig, exists := checked.FuncSigs[sourceName]; exists && len(sourceSig.ReturnEnumPayloadFunctions) > 0 {
							for payloadKey := range paramLocal.EnumPayloadFunctions {
								enumPayloadEdges = append(enumPayloadEdges, enumPayloadTargetEdge{
									destFunc:         resolved,
									destLocal:        param.Name,
									destPayloadKey:   payloadKey,
									sourceFunc:       sourceName,
									sourceLocal:      returnEnumPayloadLocal,
									sourcePayloadKey: payloadKey,
								})
							}
						}
					}
				}
			}
		}
	}

	walkExpr = func(expr frontend.Expr, caller semantics.CheckedFunc) {
		switch e := expr.(type) {
		case *frontend.CallExpr:
			addCallTargets(e, caller)
			for _, arg := range e.Args {
				walkExpr(arg, caller)
			}
		case *frontend.StructLitExpr:
			for _, field := range e.Fields {
				walkExpr(field.Value, caller)
			}
		case *frontend.FieldAccessExpr:
			walkExpr(e.Base, caller)
		case *frontend.IndexExpr:
			walkExpr(e.Base, caller)
			walkExpr(e.Index, caller)
		case *frontend.BinaryExpr:
			walkExpr(e.Left, caller)
			walkExpr(e.Right, caller)
		case *frontend.UnaryExpr:
			walkExpr(e.X, caller)
		case *frontend.TryExpr:
			walkExpr(e.X, caller)
		case *frontend.CatchExpr:
			walkExpr(e.Call, caller)
		case *frontend.AwaitExpr:
			walkExpr(e.X, caller)
		}
	}

	walkStmt = func(stmt frontend.Stmt, caller semantics.CheckedFunc) {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			walkExpr(s.Value, caller)
		case *frontend.ExpectStmt:
			walkExpr(s.Cond, caller)
		case *frontend.ReturnStmt:
			if sig, ok := checked.FuncSigs[caller.Name]; ok && sig.ReturnFunctionType {
				if target, ok := callableTargetFromAssignedExpr(s.Value, caller, checked.FuncSigs, checked.GlobalsByModule[caller.Module]); ok {
					addTarget(caller.Name, returnFunctionLocal, target)
				}
				if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && source.FunctionTypeValue {
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       returnFunctionLocal,
							sourceFunc:  caller.Name,
							sourceParam: id.Name,
						})
					} else if source, exists := checked.GlobalsByModule[caller.Module][id.Name]; exists && source.FunctionTypeValue {
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       returnFunctionLocal,
							sourceFunc:  caller.Name,
							sourceParam: id.Name,
						})
					}
				} else if sourceFieldName := functionTypedFieldNameFromExpr(s.Value); sourceFieldName != "" {
					if _, sourceOK, _ := resolveFunctionFieldName(sourceFieldName, caller.Locals); sourceOK {
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       returnFunctionLocal,
							sourceFunc:  caller.Name,
							sourceParam: sourceFieldName,
						})
					}
				} else if call, ok := s.Value.(*frontend.CallExpr); ok {
					if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
						if sourceSig, exists := checked.FuncSigs[resolved]; exists && sourceSig.ReturnFunctionType {
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       returnFunctionLocal,
								sourceFunc:  resolved,
								sourceParam: returnFunctionLocal,
							})
						}
					}
				}
			}
			if sig, ok := checked.FuncSigs[caller.Name]; ok && len(sig.ReturnFunctionFields) > 0 {
				if sourcePrefix := functionTypedFieldNameFromExpr(s.Value); sourcePrefix != "" {
					for fieldName, field := range sig.ReturnFunctionFields {
						returnFieldName := returnFieldPrefix + fieldName
						if field.FunctionValue != "" {
							addTarget(caller.Name, returnFieldName, field.FunctionValue)
						}
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       returnFieldName,
							sourceFunc:  caller.Name,
							sourceParam: sourcePrefix + "." + fieldName,
						})
					}
				} else if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && len(source.FunctionFields) > 0 {
						for fieldName, field := range sig.ReturnFunctionFields {
							returnFieldName := returnFieldPrefix + fieldName
							if field.FunctionValue != "" {
								addTarget(caller.Name, returnFieldName, field.FunctionValue)
							}
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       returnFieldName,
								sourceFunc:  caller.Name,
								sourceParam: id.Name + "." + fieldName,
							})
						}
					}
				} else if _, ok := s.Value.(*frontend.StructLitExpr); ok {
					addStructLiteralFieldEdges(caller, caller.Name, returnFieldPrefix, sig.ReturnType, s.Value, sig.ReturnFunctionFields)
				} else if call, ok := s.Value.(*frontend.CallExpr); ok {
					if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
						if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(sourceSig.ReturnFunctionFields) > 0 {
							for fieldName, field := range sig.ReturnFunctionFields {
								returnFieldName := returnFieldPrefix + fieldName
								if field.FunctionValue != "" {
									addTarget(caller.Name, returnFieldName, field.FunctionValue)
								}
								edges = append(edges, callableTargetEdge{
									callee:      caller.Name,
									param:       returnFieldName,
									sourceFunc:  resolved,
									sourceParam: returnFieldPrefix + fieldName,
								})
							}
						}
					}
				}
			}
			if sig, ok := checked.FuncSigs[caller.Name]; ok && len(sig.ReturnEnumPayloadFunctions) > 0 {
				if payloads := enumPayloadTargetsFromExpr(s.Value, caller, checked.FuncSigs, checked.Types); len(payloads) > 0 {
					addEnumPayloadTargetsForLocal(caller, returnEnumPayloadLocal, payloads)
				}
				addEnumPayloadConstructorArgEdgesForLocal(caller, returnEnumPayloadLocal, s.Value)
				if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && len(source.EnumPayloadFunctions) > 0 {
						for payloadKey := range sig.ReturnEnumPayloadFunctions {
							enumPayloadEdges = append(enumPayloadEdges, enumPayloadTargetEdge{
								destFunc:         caller.Name,
								destLocal:        returnEnumPayloadLocal,
								destPayloadKey:   payloadKey,
								sourceFunc:       caller.Name,
								sourceLocal:      id.Name,
								sourcePayloadKey: payloadKey,
							})
						}
					}
				} else if call, ok := s.Value.(*frontend.CallExpr); ok {
					if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
						if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(sourceSig.ReturnEnumPayloadFunctions) > 0 {
							for payloadKey := range sig.ReturnEnumPayloadFunctions {
								enumPayloadEdges = append(enumPayloadEdges, enumPayloadTargetEdge{
									destFunc:         caller.Name,
									destLocal:        returnEnumPayloadLocal,
									destPayloadKey:   payloadKey,
									sourceFunc:       resolved,
									sourceLocal:      returnEnumPayloadLocal,
									sourcePayloadKey: payloadKey,
								})
							}
						}
					}
				}
			}
			if sig, ok := checked.FuncSigs[caller.Name]; ok && len(sig.ReturnEnumPayloadFields) > 0 {
				if sourcePrefix := functionTypedFieldNameFromExpr(s.Value); sourcePrefix != "" {
					for fieldName, field := range sig.ReturnEnumPayloadFields {
						returnFieldName := returnFieldPrefix + fieldName
						if field.FunctionValue != "" {
							addTarget(caller.Name, returnFieldName, field.FunctionValue)
						}
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       returnFieldName,
							sourceFunc:  caller.Name,
							sourceParam: sourcePrefix + "." + fieldName,
						})
					}
				} else if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && len(source.EnumPayloadFields) > 0 {
						for fieldName, field := range sig.ReturnEnumPayloadFields {
							returnFieldName := returnFieldPrefix + fieldName
							if field.FunctionValue != "" {
								addTarget(caller.Name, returnFieldName, field.FunctionValue)
							}
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       returnFieldName,
								sourceFunc:  caller.Name,
								sourceParam: id.Name + "." + fieldName,
							})
						}
					}
				} else if _, ok := s.Value.(*frontend.StructLitExpr); ok {
					addStructLiteralEnumPayloadFieldEdges(caller, caller.Name, returnFieldPrefix, sig.ReturnType, s.Value, sig.ReturnEnumPayloadFields)
				} else if call, ok := s.Value.(*frontend.CallExpr); ok {
					if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
						if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(sourceSig.ReturnEnumPayloadFields) > 0 {
							for fieldName, field := range sig.ReturnEnumPayloadFields {
								returnFieldName := returnFieldPrefix + fieldName
								if field.FunctionValue != "" {
									addTarget(caller.Name, returnFieldName, field.FunctionValue)
								}
								edges = append(edges, callableTargetEdge{
									callee:      caller.Name,
									param:       returnFieldName,
									sourceFunc:  resolved,
									sourceParam: returnFieldPrefix + fieldName,
								})
							}
						}
					}
				}
			}
			walkExpr(s.Value, caller)
		case *frontend.ThrowStmt:
			walkExpr(s.Value, caller)
		case *frontend.LetStmt:
			if payloads := enumPayloadTargetsFromExpr(s.Value, caller, checked.FuncSigs, checked.Types); len(payloads) > 0 {
				addEnumPayloadTargetsForLocal(caller, s.Name, payloads)
				addEnumPayloadFunctionReturnEdgesForLocal(caller, s.Name, s.Value)
			}
			if dest, ok := caller.Locals[s.Name]; ok && len(dest.EnumPayloadFunctions) > 0 {
				addEnumPayloadConstructorArgEdgesForLocal(caller, s.Name, s.Value)
				if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && len(source.EnumPayloadFunctions) > 0 {
						for payloadKey := range dest.EnumPayloadFunctions {
							enumPayloadEdges = append(enumPayloadEdges, enumPayloadTargetEdge{
								destFunc:         caller.Name,
								destLocal:        s.Name,
								destPayloadKey:   payloadKey,
								sourceFunc:       caller.Name,
								sourceLocal:      id.Name,
								sourcePayloadKey: payloadKey,
							})
						}
					}
				} else if call, ok := s.Value.(*frontend.CallExpr); ok {
					if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
						if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(sourceSig.ReturnEnumPayloadFunctions) > 0 {
							for payloadKey := range dest.EnumPayloadFunctions {
								enumPayloadEdges = append(enumPayloadEdges, enumPayloadTargetEdge{
									destFunc:         caller.Name,
									destLocal:        s.Name,
									destPayloadKey:   payloadKey,
									sourceFunc:       resolved,
									sourceLocal:      returnEnumPayloadLocal,
									sourcePayloadKey: payloadKey,
								})
							}
						}
					}
				}
			}
			if dest, ok := caller.Locals[s.Name]; ok && dest.FunctionTypeValue {
				if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && source.FunctionTypeValue {
						if source.FunctionValue != "" {
							addTarget(caller.Name, s.Name, source.FunctionValue)
						}
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       s.Name,
							sourceFunc:  caller.Name,
							sourceParam: id.Name,
						})
					}
				} else if fieldName := functionTypedFieldNameFromExpr(s.Value); fieldName != "" {
					if field, ok, _ := resolveFunctionFieldName(fieldName, caller.Locals); ok {
						if field.FunctionValue != "" {
							addTarget(caller.Name, s.Name, field.FunctionValue)
						}
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       s.Name,
							sourceFunc:  caller.Name,
							sourceParam: fieldName,
						})
					}
				} else if call, ok := s.Value.(*frontend.CallExpr); ok {
					if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
						if sourceSig, exists := checked.FuncSigs[resolved]; exists && sourceSig.ReturnFunctionType {
							if sourceSig.ReturnFunctionSymbol != "" {
								addTarget(caller.Name, s.Name, sourceSig.ReturnFunctionSymbol)
							}
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       s.Name,
								sourceFunc:  resolved,
								sourceParam: returnFunctionLocal,
							})
						}
					}
				}
			}
			if dest, ok := caller.Locals[s.Name]; ok && len(dest.FunctionFields) > 0 {
				if sourcePrefix := functionTypedFieldNameFromExpr(s.Value); sourcePrefix != "" {
					for fieldName, field := range dest.FunctionFields {
						destFieldName := s.Name + "." + fieldName
						if field.FunctionValue != "" {
							addTarget(caller.Name, destFieldName, field.FunctionValue)
						}
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       destFieldName,
							sourceFunc:  caller.Name,
							sourceParam: sourcePrefix + "." + fieldName,
						})
					}
				} else if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && len(source.FunctionFields) > 0 {
						for fieldName, field := range dest.FunctionFields {
							destFieldName := s.Name + "." + fieldName
							if field.FunctionValue != "" {
								addTarget(caller.Name, destFieldName, field.FunctionValue)
							}
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       destFieldName,
								sourceFunc:  caller.Name,
								sourceParam: id.Name + "." + fieldName,
							})
						}
					}
				} else if call, ok := s.Value.(*frontend.CallExpr); ok {
					if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
						if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(sourceSig.ReturnFunctionFields) > 0 {
							for fieldName, field := range dest.FunctionFields {
								destFieldName := s.Name + "." + fieldName
								if field.FunctionValue != "" {
									addTarget(caller.Name, destFieldName, field.FunctionValue)
								}
								edges = append(edges, callableTargetEdge{
									callee:      caller.Name,
									param:       destFieldName,
									sourceFunc:  resolved,
									sourceParam: returnFieldPrefix + fieldName,
								})
							}
						}
					}
				} else if _, ok := s.Value.(*frontend.StructLitExpr); ok {
					addStructLiteralFieldEdges(caller, caller.Name, s.Name+".", dest.TypeName, s.Value, dest.FunctionFields)
				}
			}
			if dest, ok := caller.Locals[s.Name]; ok && len(dest.EnumPayloadFields) > 0 {
				if sourcePrefix := functionTypedFieldNameFromExpr(s.Value); sourcePrefix != "" {
					for fieldName, field := range dest.EnumPayloadFields {
						destFieldName := s.Name + "." + fieldName
						if field.FunctionValue != "" {
							addTarget(caller.Name, destFieldName, field.FunctionValue)
						}
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       destFieldName,
							sourceFunc:  caller.Name,
							sourceParam: sourcePrefix + "." + fieldName,
						})
					}
				} else if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && len(source.EnumPayloadFields) > 0 {
						for fieldName, field := range dest.EnumPayloadFields {
							destFieldName := s.Name + "." + fieldName
							if field.FunctionValue != "" {
								addTarget(caller.Name, destFieldName, field.FunctionValue)
							}
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       destFieldName,
								sourceFunc:  caller.Name,
								sourceParam: id.Name + "." + fieldName,
							})
						}
					}
				} else if call, ok := s.Value.(*frontend.CallExpr); ok {
					if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
						if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(sourceSig.ReturnEnumPayloadFields) > 0 {
							for fieldName, field := range dest.EnumPayloadFields {
								destFieldName := s.Name + "." + fieldName
								if field.FunctionValue != "" {
									addTarget(caller.Name, destFieldName, field.FunctionValue)
								}
								edges = append(edges, callableTargetEdge{
									callee:      caller.Name,
									param:       destFieldName,
									sourceFunc:  resolved,
									sourceParam: returnFieldPrefix + fieldName,
								})
							}
						}
					}
				} else if _, ok := s.Value.(*frontend.StructLitExpr); ok {
					addStructLiteralEnumPayloadFieldEdges(caller, caller.Name, s.Name+".", dest.TypeName, s.Value, dest.EnumPayloadFields)
				}
			}
			walkExpr(s.Value, caller)
		case *frontend.AssignStmt:
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				if local, exists := caller.Locals[id.Name]; exists && local.FunctionTypeValue {
					if target, ok := callableTargetFromAssignedExpr(s.Value, caller, checked.FuncSigs, checked.GlobalsByModule[caller.Module]); ok {
						addTarget(caller.Name, id.Name, target)
					}
					if valueID, ok := s.Value.(*frontend.IdentExpr); ok {
						if source, sourceExists := checked.GlobalsByModule[caller.Module][valueID.Name]; sourceExists && source.FunctionTypeValue {
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       id.Name,
								sourceFunc:  caller.Name,
								sourceParam: valueID.Name,
							})
						}
					}
					if call, ok := s.Value.(*frontend.CallExpr); ok {
						if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
							if sourceSig, exists := checked.FuncSigs[resolved]; exists && sourceSig.ReturnFunctionType {
								edges = append(edges, callableTargetEdge{
									callee:      caller.Name,
									param:       id.Name,
									sourceFunc:  resolved,
									sourceParam: returnFunctionLocal,
								})
							}
						}
					}
				}
				if global, exists := checked.GlobalsByModule[caller.Module][id.Name]; exists && global.FunctionTypeValue {
					if target, ok := callableTargetFromAssignedExpr(s.Value, caller, checked.FuncSigs, checked.GlobalsByModule[caller.Module]); ok {
						addTarget(caller.Name, id.Name, target)
						if global.Mutable {
							addModuleGlobalTarget(caller.Module, id.Name, target)
						}
					}
					if valueID, ok := s.Value.(*frontend.IdentExpr); ok {
						if source, sourceExists := caller.Locals[valueID.Name]; sourceExists && source.FunctionTypeValue {
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       id.Name,
								sourceFunc:  caller.Name,
								sourceParam: valueID.Name,
							})
							if global.Mutable {
								moduleGlobalEdges = append(moduleGlobalEdges, moduleGlobalTargetEdge{
									module:      caller.Module,
									global:      id.Name,
									sourceFunc:  caller.Name,
									sourceParam: valueID.Name,
								})
							}
						}
					} else if sourceFieldName := functionTypedFieldNameFromExpr(s.Value); sourceFieldName != "" {
						if _, sourceOK, _ := resolveFunctionFieldName(sourceFieldName, caller.Locals); sourceOK {
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       id.Name,
								sourceFunc:  caller.Name,
								sourceParam: sourceFieldName,
							})
							if global.Mutable {
								moduleGlobalEdges = append(moduleGlobalEdges, moduleGlobalTargetEdge{
									module:      caller.Module,
									global:      id.Name,
									sourceFunc:  caller.Name,
									sourceParam: sourceFieldName,
								})
							}
						}
					} else if call, ok := s.Value.(*frontend.CallExpr); ok {
						if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
							if sourceSig, sourceExists := checked.FuncSigs[resolved]; sourceExists && sourceSig.ReturnFunctionType {
								edges = append(edges, callableTargetEdge{
									callee:      caller.Name,
									param:       id.Name,
									sourceFunc:  resolved,
									sourceParam: returnFunctionLocal,
								})
							}
						}
					}
				}
				if local, exists := caller.Locals[id.Name]; exists {
					if info, ok := checked.Types[local.TypeName]; ok && info.Kind == semantics.TypeEnum {
						if payloads := enumPayloadTargetsFromExpr(s.Value, caller, checked.FuncSigs, checked.Types); len(payloads) > 0 {
							addEnumPayloadTargetsForLocal(caller, id.Name, payloads)
							addEnumPayloadFunctionReturnEdgesForLocal(caller, id.Name, s.Value)
						}
						addEnumPayloadConstructorArgEdgesForLocal(caller, id.Name, s.Value)
						if idValue, ok := s.Value.(*frontend.IdentExpr); ok {
							if source, exists := caller.Locals[idValue.Name]; exists && len(source.EnumPayloadFunctions) > 0 {
								for payloadKey := range local.EnumPayloadFunctions {
									enumPayloadEdges = append(enumPayloadEdges, enumPayloadTargetEdge{
										destFunc:         caller.Name,
										destLocal:        id.Name,
										destPayloadKey:   payloadKey,
										sourceFunc:       caller.Name,
										sourceLocal:      idValue.Name,
										sourcePayloadKey: payloadKey,
									})
								}
							}
						} else if call, ok := s.Value.(*frontend.CallExpr); ok {
							if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
								if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(sourceSig.ReturnEnumPayloadFunctions) > 0 {
									for payloadKey := range local.EnumPayloadFunctions {
										enumPayloadEdges = append(enumPayloadEdges, enumPayloadTargetEdge{
											destFunc:         caller.Name,
											destLocal:        id.Name,
											destPayloadKey:   payloadKey,
											sourceFunc:       resolved,
											sourceLocal:      returnEnumPayloadLocal,
											sourcePayloadKey: payloadKey,
										})
									}
								}
							}
						}
					}
					if info, ok := checked.Types[local.TypeName]; ok && info.Kind == semantics.TypeStruct && len(local.FunctionFields) > 0 {
						if sourcePrefix := functionTypedFieldNameFromExpr(s.Value); sourcePrefix != "" {
							for fieldName, field := range local.FunctionFields {
								destFieldName := id.Name + "." + fieldName
								if field.FunctionValue != "" {
									addTarget(caller.Name, destFieldName, field.FunctionValue)
								}
								edges = append(edges, callableTargetEdge{
									callee:      caller.Name,
									param:       destFieldName,
									sourceFunc:  caller.Name,
									sourceParam: sourcePrefix + "." + fieldName,
								})
							}
						} else if valueID, ok := s.Value.(*frontend.IdentExpr); ok {
							if source, exists := caller.Locals[valueID.Name]; exists && len(source.FunctionFields) > 0 {
								for fieldName, field := range local.FunctionFields {
									destFieldName := id.Name + "." + fieldName
									if field.FunctionValue != "" {
										addTarget(caller.Name, destFieldName, field.FunctionValue)
									}
									edges = append(edges, callableTargetEdge{
										callee:      caller.Name,
										param:       destFieldName,
										sourceFunc:  caller.Name,
										sourceParam: valueID.Name + "." + fieldName,
									})
								}
							}
						} else if call, ok := s.Value.(*frontend.CallExpr); ok {
							if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
								if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(sourceSig.ReturnFunctionFields) > 0 {
									for fieldName, field := range local.FunctionFields {
										destFieldName := id.Name + "." + fieldName
										if field.FunctionValue != "" {
											addTarget(caller.Name, destFieldName, field.FunctionValue)
										}
										edges = append(edges, callableTargetEdge{
											callee:      caller.Name,
											param:       destFieldName,
											sourceFunc:  resolved,
											sourceParam: returnFieldPrefix + fieldName,
										})
									}
								}
							}
						} else if _, ok := s.Value.(*frontend.StructLitExpr); ok {
							addStructLiteralFieldEdges(caller, caller.Name, id.Name+".", local.TypeName, s.Value, local.FunctionFields)
						}
					}
					if info, ok := checked.Types[local.TypeName]; ok && info.Kind == semantics.TypeStruct && len(local.EnumPayloadFields) > 0 {
						if sourcePrefix := functionTypedFieldNameFromExpr(s.Value); sourcePrefix != "" {
							for fieldName, field := range local.EnumPayloadFields {
								destFieldName := id.Name + "." + fieldName
								if field.FunctionValue != "" {
									addTarget(caller.Name, destFieldName, field.FunctionValue)
								}
								edges = append(edges, callableTargetEdge{
									callee:      caller.Name,
									param:       destFieldName,
									sourceFunc:  caller.Name,
									sourceParam: sourcePrefix + "." + fieldName,
								})
							}
						} else if valueID, ok := s.Value.(*frontend.IdentExpr); ok {
							if source, exists := caller.Locals[valueID.Name]; exists && len(source.EnumPayloadFields) > 0 {
								for fieldName, field := range local.EnumPayloadFields {
									destFieldName := id.Name + "." + fieldName
									if field.FunctionValue != "" {
										addTarget(caller.Name, destFieldName, field.FunctionValue)
									}
									edges = append(edges, callableTargetEdge{
										callee:      caller.Name,
										param:       destFieldName,
										sourceFunc:  caller.Name,
										sourceParam: valueID.Name + "." + fieldName,
									})
								}
							}
						} else if call, ok := s.Value.(*frontend.CallExpr); ok {
							if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
								if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(sourceSig.ReturnEnumPayloadFields) > 0 {
									for fieldName, field := range local.EnumPayloadFields {
										destFieldName := id.Name + "." + fieldName
										if field.FunctionValue != "" {
											addTarget(caller.Name, destFieldName, field.FunctionValue)
										}
										edges = append(edges, callableTargetEdge{
											callee:      caller.Name,
											param:       destFieldName,
											sourceFunc:  resolved,
											sourceParam: returnFieldPrefix + fieldName,
										})
									}
								}
							}
						} else if _, ok := s.Value.(*frontend.StructLitExpr); ok {
							addStructLiteralEnumPayloadFieldEdges(caller, caller.Name, id.Name+".", local.TypeName, s.Value, local.EnumPayloadFields)
						}
					}
				}
			} else if fieldName := functionTypedFieldNameFromExpr(s.Target); fieldName != "" {
				if _, ok, _ := resolveFunctionFieldName(fieldName, caller.Locals); ok {
					if target, ok := callableTargetFromAssignedExpr(s.Value, caller, checked.FuncSigs, checked.GlobalsByModule[caller.Module]); ok {
						addTarget(caller.Name, fieldName, target)
					}
					if id, ok := s.Value.(*frontend.IdentExpr); ok {
						if source, exists := caller.Locals[id.Name]; exists && source.FunctionTypeValue {
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       fieldName,
								sourceFunc:  caller.Name,
								sourceParam: id.Name,
							})
						} else if source, exists := checked.GlobalsByModule[caller.Module][id.Name]; exists && source.FunctionTypeValue {
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       fieldName,
								sourceFunc:  caller.Name,
								sourceParam: id.Name,
							})
						}
					} else if sourceFieldName := functionTypedFieldNameFromExpr(s.Value); sourceFieldName != "" {
						if _, sourceOK, _ := resolveFunctionFieldName(sourceFieldName, caller.Locals); sourceOK {
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       fieldName,
								sourceFunc:  caller.Name,
								sourceParam: sourceFieldName,
							})
						}
					} else if call, ok := s.Value.(*frontend.CallExpr); ok {
						if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
							if sourceSig, exists := checked.FuncSigs[resolved]; exists && sourceSig.ReturnFunctionType {
								edges = append(edges, callableTargetEdge{
									callee:      caller.Name,
									param:       fieldName,
									sourceFunc:  resolved,
									sourceParam: returnFunctionLocal,
								})
							}
						}
					}
				} else {
					parts := strings.Split(fieldName, ".")
					if len(parts) >= 2 {
						baseName := parts[0]
						fieldPath := parts[1:]
						if local, exists := caller.Locals[baseName]; exists && len(local.FunctionFields) > 0 {
							targetType, _, _, err := resolveFieldChainLower(local.TypeName, local.Base, fieldPath, checked.Types, s.Target.Pos())
							if err == nil {
								if info, ok := checked.Types[targetType]; ok && info.Kind == semantics.TypeStruct {
									fieldPrefix := strings.Join(fieldPath, ".") + "."
									destFields := trimFunctionFields(local.FunctionFields, fieldPrefix)
									if len(destFields) > 0 {
										destPrefix := fieldName + "."
										if sourcePrefix := functionTypedFieldNameFromExpr(s.Value); sourcePrefix != "" {
											for nestedName, field := range destFields {
												destFieldName := destPrefix + nestedName
												if field.FunctionValue != "" {
													addTarget(caller.Name, destFieldName, field.FunctionValue)
												}
												edges = append(edges, callableTargetEdge{
													callee:      caller.Name,
													param:       destFieldName,
													sourceFunc:  caller.Name,
													sourceParam: sourcePrefix + "." + nestedName,
												})
											}
										} else if valueID, ok := s.Value.(*frontend.IdentExpr); ok {
											if source, exists := caller.Locals[valueID.Name]; exists && len(source.FunctionFields) > 0 {
												for nestedName, field := range destFields {
													destFieldName := destPrefix + nestedName
													if field.FunctionValue != "" {
														addTarget(caller.Name, destFieldName, field.FunctionValue)
													}
													edges = append(edges, callableTargetEdge{
														callee:      caller.Name,
														param:       destFieldName,
														sourceFunc:  caller.Name,
														sourceParam: valueID.Name + "." + nestedName,
													})
												}
											}
										} else if call, ok := s.Value.(*frontend.CallExpr); ok {
											if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
												if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(sourceSig.ReturnFunctionFields) > 0 {
													for nestedName, field := range destFields {
														destFieldName := destPrefix + nestedName
														if field.FunctionValue != "" {
															addTarget(caller.Name, destFieldName, field.FunctionValue)
														}
														edges = append(edges, callableTargetEdge{
															callee:      caller.Name,
															param:       destFieldName,
															sourceFunc:  resolved,
															sourceParam: returnFieldPrefix + nestedName,
														})
													}
												}
											}
										} else if _, ok := s.Value.(*frontend.StructLitExpr); ok {
											addStructLiteralFieldEdges(caller, caller.Name, destPrefix, targetType, s.Value, destFields)
										}
									}
								}
							}
						}
					}
				}
			}
			walkExpr(s.Target, caller)
			walkExpr(s.Value, caller)
		case *frontend.ExprStmt:
			walkExpr(s.Expr, caller)
		case *frontend.IfStmt:
			walkExpr(s.Cond, caller)
			for _, inner := range s.Then {
				walkStmt(inner, caller)
			}
			for _, inner := range s.Else {
				walkStmt(inner, caller)
			}
		case *frontend.IfLetStmt:
			walkExpr(s.Value, caller)
			for _, inner := range s.Then {
				walkStmt(inner, caller)
			}
			for _, inner := range s.Else {
				walkStmt(inner, caller)
			}
		case *frontend.WhileStmt:
			walkExpr(s.Cond, caller)
			for _, inner := range s.Body {
				walkStmt(inner, caller)
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				walkExpr(s.Iterable, caller)
			} else {
				walkExpr(s.Start, caller)
				walkExpr(s.End, caller)
			}
			for _, inner := range s.Body {
				walkStmt(inner, caller)
			}
		case *frontend.MatchStmt:
			walkExpr(s.Value, caller)
			if id, ok := s.Value.(*frontend.IdentExpr); ok {
				for _, c := range s.Cases {
					enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr)
					if !ok {
						continue
					}
					caseInfo, ok := enumCasePatternInfoForTargets(enumPat, checked.Types)
					if !ok {
						continue
					}
					for i, binding := range enumPat.Bindings {
						if i >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[i] {
							continue
						}
						payloadKey := enumPayloadTargetKey(caseInfo.Ordinal, i)
						for target := range enumPayloadTargetSets[caller.Name][id.Name][payloadKey] {
							addTarget(caller.Name, binding, target)
						}
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       binding,
							sourceFunc:  caller.Name,
							sourceParam: enumPayloadSourceName(id.Name, payloadKey),
						})
					}
				}
			} else if payloads := enumPayloadTargetsFromExpr(s.Value, caller, checked.FuncSigs, checked.Types); len(payloads) > 0 {
				sourcePrefix := functionTypedFieldNameFromExpr(s.Value)
				for _, c := range s.Cases {
					enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr)
					if !ok {
						continue
					}
					caseInfo, ok := enumCasePatternInfoForTargets(enumPat, checked.Types)
					if !ok {
						continue
					}
					for i, binding := range enumPat.Bindings {
						if i >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[i] {
							continue
						}
						payloadKey := enumPayloadTargetKey(caseInfo.Ordinal, i)
						payload, ok := payloads[payloadKey]
						if !ok {
							continue
						}
						if payload.FunctionValue != "" {
							addTarget(caller.Name, binding, payload.FunctionValue)
						}
						if sourcePrefix != "" {
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       binding,
								sourceFunc:  caller.Name,
								sourceParam: sourcePrefix + "#" + payloadKey,
							})
						}
					}
				}
			}
			for _, c := range s.Cases {
				if !c.Default {
					walkExpr(c.Pattern, caller)
				}
				for _, inner := range c.Body {
					walkStmt(inner, caller)
				}
			}
		case *frontend.DeferStmt:
			for _, inner := range s.Body {
				walkStmt(inner, caller)
			}
		case *frontend.UnsafeStmt:
			for _, inner := range s.Body {
				walkStmt(inner, caller)
			}
		case *frontend.IslandStmt:
			walkExpr(s.Size, caller)
			for _, inner := range s.Body {
				walkStmt(inner, caller)
			}
		case *frontend.FreeStmt:
			walkExpr(s.Value, caller)
		}
	}

	for _, fn := range checked.Funcs {
		if module != "" && fn.Module != module {
			continue
		}
		if fn.Decl == nil {
			continue
		}
		for _, stmt := range fn.Decl.Body {
			walkStmt(stmt, fn)
		}
	}

	for changed := true; changed; {
		changed = false
		for _, edge := range enumPayloadEdges {
			for target := range enumPayloadTargetSets[edge.sourceFunc][edge.sourceLocal][edge.sourcePayloadKey] {
				if addEnumPayloadTarget(edge.destFunc, edge.destLocal, edge.destPayloadKey, target) {
					changed = true
				}
			}
			for target := range targetSets[edge.sourceFunc][enumPayloadSourceName(edge.sourceLocal, edge.sourcePayloadKey)] {
				if addEnumPayloadTarget(edge.destFunc, edge.destLocal, edge.destPayloadKey, target) {
					changed = true
				}
			}
		}
		for _, edge := range edges {
			for target := range targetSets[edge.sourceFunc][edge.sourceParam] {
				if addTarget(edge.callee, edge.param, target) {
					changed = true
				}
			}
		}
		for _, edge := range moduleGlobalEdges {
			for target := range targetSets[edge.sourceFunc][edge.sourceParam] {
				if addModuleGlobalTarget(edge.module, edge.global, target) {
					changed = true
				}
			}
		}
	}

	out := map[string]map[string][]string{}
	for funcName, params := range targetSets {
		out[funcName] = map[string][]string{}
		for paramName, symbols := range params {
			list := make([]string, 0, len(symbols))
			for symbol := range symbols {
				list = append(list, symbol)
			}
			sort.Strings(list)
			out[funcName][paramName] = list
		}
	}
	return out
}
