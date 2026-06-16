package lower

import (
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics"
)

const (
	callableReturnFieldPrefix      = "$return."
	callableReturnFunctionLocal    = "$return.fn"
	callableReturnEnumPayloadLocal = "$return.enum"
)

type callableTargetEdge struct {
	callee      string
	param       string
	sourceFunc  string
	sourceParam string
}

type moduleGlobalTargetEdge struct {
	module      string
	global      string
	sourceFunc  string
	sourceParam string
}

type enumPayloadTargetEdge struct {
	destFunc         string
	destLocal        string
	destPayloadKey   string
	sourceFunc       string
	sourceLocal      string
	sourcePayloadKey string
}

type callableTargetAdder func(callee, paramName, targetSymbol string) bool
type callableTargetEdgeAdder func(callableTargetEdge)
type enumPayloadTargetEdgeAdder func(enumPayloadTargetEdge)

func addStructLiteralFieldEdgesForTargets(
	caller semantics.CheckedFunc,
	destFunc, destPrefix, structType string,
	value frontend.Expr,
	destFields map[string]semantics.FunctionFieldInfo,
	types map[string]*semantics.TypeInfo,
	funcs map[string]semantics.FuncSig,
	globals map[string]semantics.GlobalInfo,
	addTarget callableTargetAdder,
	addEdge callableTargetEdgeAdder,
) {
	lit, ok := value.(*frontend.StructLitExpr)
	if !ok || len(destFields) == 0 {
		return
	}
	typeInfo, ok := types[structType]
	if !ok || typeInfo.Kind != semantics.TypeStruct {
		return
	}
	for _, init := range lit.Fields {
		field, ok := typeInfo.FieldMap[init.Name]
		if !ok {
			continue
		}
		if field.FunctionTypeValue {
			if _, ok := destFields[init.Name]; ok {
				destFieldName := destPrefix + init.Name
				if target, ok := callableTargetFromAssignedExpr(init.Value, caller, funcs, globals); ok {
					addTarget(destFunc, destFieldName, target)
				}
				if sourceFieldName := functionTypedFieldNameFromExpr(init.Value); sourceFieldName != "" {
					addEdge(callableTargetEdge{
						callee:      destFunc,
						param:       destFieldName,
						sourceFunc:  caller.Name,
						sourceParam: sourceFieldName,
					})
				} else if id, ok := init.Value.(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && source.FunctionTypeValue {
						addEdge(callableTargetEdge{
							callee:      destFunc,
							param:       destFieldName,
							sourceFunc:  caller.Name,
							sourceParam: id.Name,
						})
					} else if source, exists := globals[id.Name]; exists && source.FunctionTypeValue {
						addEdge(callableTargetEdge{
							callee:      destFunc,
							param:       destFieldName,
							sourceFunc:  caller.Name,
							sourceParam: id.Name,
						})
					}
				}
				if call, ok := init.Value.(*frontend.CallExpr); ok {
					if resolved, ok := resolvedCallableFunctionName(call.Name, funcs); ok {
						if sourceSig, exists := funcs[resolved]; exists && sourceSig.ReturnFunctionType {
							addEdge(callableTargetEdge{
								callee:      destFunc,
								param:       destFieldName,
								sourceFunc:  resolved,
								sourceParam: callableReturnFunctionLocal,
							})
						}
					}
				}
			}
		}
		nestedPrefix := init.Name + "."
		hasNestedDest := false
		for fieldName := range destFields {
			if strings.HasPrefix(fieldName, nestedPrefix) {
				hasNestedDest = true
				break
			}
		}
		if !hasNestedDest {
			continue
		}
		if call, ok := init.Value.(*frontend.CallExpr); ok {
			if resolved, ok := resolvedCallableFunctionName(call.Name, funcs); ok {
				if sourceSig, exists := funcs[resolved]; exists && len(sourceSig.ReturnFunctionFields) > 0 {
					for fieldName, field := range destFields {
						if !strings.HasPrefix(fieldName, nestedPrefix) {
							continue
						}
						sourceField := strings.TrimPrefix(fieldName, nestedPrefix)
						if _, ok := sourceSig.ReturnFunctionFields[sourceField]; !ok {
							continue
						}
						destFieldName := destPrefix + fieldName
						if field.FunctionValue != "" {
							addTarget(destFunc, destFieldName, field.FunctionValue)
						}
						addEdge(callableTargetEdge{
							callee:      destFunc,
							param:       destFieldName,
							sourceFunc:  resolved,
							sourceParam: callableReturnFieldPrefix + sourceField,
						})
					}
				}
			}
		}
		sourcePrefix := functionTypedFieldNameFromExpr(init.Value)
		if sourcePrefix == "" {
			if id, ok := init.Value.(*frontend.IdentExpr); ok {
				sourcePrefix = id.Name
			}
		}
		if sourcePrefix != "" {
			for fieldName, field := range destFields {
				if !strings.HasPrefix(fieldName, nestedPrefix) {
					continue
				}
				sourceField := strings.TrimPrefix(fieldName, nestedPrefix)
				destFieldName := destPrefix + fieldName
				if field.FunctionValue != "" {
					addTarget(destFunc, destFieldName, field.FunctionValue)
				}
				addEdge(callableTargetEdge{
					callee:      destFunc,
					param:       destFieldName,
					sourceFunc:  caller.Name,
					sourceParam: sourcePrefix + "." + sourceField,
				})
			}
		}
		addStructLiteralFieldEdgesForTargets(caller, destFunc, destPrefix+nestedPrefix, field.TypeName, init.Value, trimFunctionFields(destFields, nestedPrefix), types, funcs, globals, addTarget, addEdge)
	}
}

func addStructLiteralEnumPayloadFieldEdgesForTargets(
	caller semantics.CheckedFunc,
	destFunc, destPrefix, structType string,
	value frontend.Expr,
	destFields map[string]semantics.FunctionFieldInfo,
	types map[string]*semantics.TypeInfo,
	funcs map[string]semantics.FuncSig,
	addTarget callableTargetAdder,
	addEdge callableTargetEdgeAdder,
	addEnumPayloadEdge enumPayloadTargetEdgeAdder,
) {
	lit, ok := value.(*frontend.StructLitExpr)
	if !ok || len(destFields) == 0 {
		return
	}
	typeInfo, ok := types[structType]
	if !ok || typeInfo.Kind != semantics.TypeStruct {
		return
	}
	for _, init := range lit.Fields {
		field, ok := typeInfo.FieldMap[init.Name]
		if !ok {
			continue
		}
		if info, ok := types[field.TypeName]; ok && info.Kind == semantics.TypeEnum {
			fieldPrefix := init.Name + "#"
			if payloads := enumPayloadTargetsFromExpr(init.Value, caller, funcs, types); len(payloads) > 0 {
				for payloadKey, payload := range payloads {
					fieldName := fieldPrefix + payloadKey
					if _, ok := destFields[fieldName]; !ok {
						continue
					}
					if payload.FunctionValue != "" {
						addTarget(destFunc, destPrefix+fieldName, payload.FunctionValue)
					}
				}
			}
			if call, ok := init.Value.(*frontend.CallExpr); ok {
				if resolved, ok := resolvedCallableFunctionName(call.Name, funcs); ok {
					if sourceSig, exists := funcs[resolved]; exists && len(sourceSig.ReturnEnumPayloadFunctions) > 0 {
						for fieldName := range destFields {
							if !strings.HasPrefix(fieldName, fieldPrefix) {
								continue
							}
							payloadKey := strings.TrimPrefix(fieldName, fieldPrefix)
							if _, ok := sourceSig.ReturnEnumPayloadFunctions[payloadKey]; !ok {
								continue
							}
							addEnumPayloadEdge(enumPayloadTargetEdge{
								destFunc:         destFunc,
								destLocal:        strings.TrimSuffix(destPrefix+init.Name, "."),
								destPayloadKey:   payloadKey,
								sourceFunc:       resolved,
								sourceLocal:      callableReturnEnumPayloadLocal,
								sourcePayloadKey: payloadKey,
							})
						}
					}
				}
			}
		}
		nestedPrefix := init.Name + "."
		hasNestedDest := false
		for fieldName := range destFields {
			if strings.HasPrefix(fieldName, nestedPrefix) {
				hasNestedDest = true
				break
			}
		}
		if !hasNestedDest {
			continue
		}
		if call, ok := init.Value.(*frontend.CallExpr); ok {
			if resolved, ok := resolvedCallableFunctionName(call.Name, funcs); ok {
				if sourceSig, exists := funcs[resolved]; exists && len(sourceSig.ReturnEnumPayloadFields) > 0 {
					for fieldName, field := range destFields {
						if !strings.HasPrefix(fieldName, nestedPrefix) {
							continue
						}
						sourceField := strings.TrimPrefix(fieldName, nestedPrefix)
						if _, ok := sourceSig.ReturnEnumPayloadFields[sourceField]; !ok {
							continue
						}
						destFieldName := destPrefix + fieldName
						if field.FunctionValue != "" {
							addTarget(destFunc, destFieldName, field.FunctionValue)
						}
						addEdge(callableTargetEdge{
							callee:      destFunc,
							param:       destFieldName,
							sourceFunc:  resolved,
							sourceParam: callableReturnFieldPrefix + sourceField,
						})
					}
				}
			}
		}
		sourcePrefix := functionTypedFieldNameFromExpr(init.Value)
		if sourcePrefix == "" {
			if id, ok := init.Value.(*frontend.IdentExpr); ok {
				sourcePrefix = id.Name
			}
		}
		if sourcePrefix != "" {
			for fieldName, field := range destFields {
				if !strings.HasPrefix(fieldName, nestedPrefix) {
					continue
				}
				sourceField := strings.TrimPrefix(fieldName, nestedPrefix)
				destFieldName := destPrefix + fieldName
				if field.FunctionValue != "" {
					addTarget(destFunc, destFieldName, field.FunctionValue)
				}
				addEdge(callableTargetEdge{
					callee:      destFunc,
					param:       destFieldName,
					sourceFunc:  caller.Name,
					sourceParam: sourcePrefix + "." + sourceField,
				})
			}
		}
		addStructLiteralEnumPayloadFieldEdgesForTargets(caller, destFunc, destPrefix+nestedPrefix, field.TypeName, init.Value, trimFunctionFields(destFields, nestedPrefix), types, funcs, addTarget, addEdge, addEnumPayloadEdge)
	}
}
