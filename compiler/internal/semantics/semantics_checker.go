package semantics

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/islandkernel"
	"tetra_language/compiler/internal/module"
	semanticsdeclarations "tetra_language/compiler/internal/semantics/declarations"
	semanticsflow "tetra_language/compiler/internal/semantics/flow"
	semanticsfunctiontypes "tetra_language/compiler/internal/semantics/functiontypes"
	semanticspolicy "tetra_language/compiler/internal/semantics/policy"
	semanticsresources "tetra_language/compiler/internal/semantics/resources"
	semanticsstatements "tetra_language/compiler/internal/semantics/statements"
	semanticsworld "tetra_language/compiler/internal/semantics/world"
)

// ---- checker.go ----

func Check(prog *frontend.Program) (*CheckedProgram, error) {
	if prog == nil {
		return nil, fmt.Errorf("no program provided")
	}
	file := &frontend.FileAST{
		Module:     "",
		Capsules:   prog.Capsules,
		Enums:      prog.Enums,
		Structs:    prog.Structs,
		States:     prog.States,
		Views:      prog.Views,
		Actors:     prog.Actors,
		Protocols:  prog.Protocols,
		Extensions: prog.Extensions,
		Impls:      prog.Impls,
		Funcs:      prog.Funcs,
		Tests:      prog.Tests,
	}
	world := &module.World{
		EntryModule: "",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"": file},
	}
	return CheckWorldOpt(world, CheckOptions{RequireMain: true})
}

type CheckOptions struct {
	RequireMain              bool
	EnableILP32NativeScalars bool
}

func CheckWorld(world *module.World) (*CheckedProgram, error) {
	return CheckWorldOpt(world, CheckOptions{RequireMain: true})
}

func CheckWorldOpt(world *module.World, opt CheckOptions) (*CheckedProgram, error) {
	if world == nil || len(world.Files) == 0 {
		return nil, fmt.Errorf("no functions found")
	}
	if err := monomorphizeGenerics(world); err != nil {
		return nil, err
	}
	clearMatchPayloadExhaustivenessMarkers(world)
	if err := normalizeExtensionMethodNames(world); err != nil {
		return nil, err
	}

	types := baseTypes()
	if opt.EnableILP32NativeScalars {
		addILP32NativeScalarTypes(types)
	}

	type structContext struct {
		module  string
		imports map[string]string
		public  bool
		decl    *frontend.StructDecl
	}
	type enumContext struct {
		module  string
		imports map[string]string
		public  bool
		decl    *frontend.EnumDecl
	}
	type protocolContext struct {
		module  string
		imports map[string]string
		public  bool
		decl    *frontend.ProtocolDecl
	}
	type actorContext struct {
		module  string
		imports map[string]string
		decl    *frontend.ActorDecl
	}

	structs := make(map[string]structContext)
	enums := make(map[string]enumContext)
	protocols := make(map[string]protocolContext)
	actors := make([]actorContext, 0)
	actorStateFieldsByMethod := make(map[string]map[string]ActorStateField)
	checked := CheckedProgram{
		MainIndex:          -1,
		Types:              types,
		FuncSigs:           make(map[string]FuncSig),
		GlobalsByModule:    make(map[string]map[string]GlobalInfo),
		GlobalDataByModule: make(map[string][][]byte),
	}
	capsulePermissionsByModule := collectCapsulePermissionsByModule(world)
	exportedSymbols := make(map[string]string)

	seenImpls := map[string]frontend.Position{}
	for _, file := range world.Files {
		module := file.Module
		imports, err := collectImportAliases(file)
		if err != nil {
			return nil, err
		}
		if err := validateCapsuleDecls(file); err != nil {
			return nil, err
		}
		for _, en := range file.Enums {
			fullName := qualifyName(module, en.Name)
			if isReservedTypeName(en.Name) {
				return nil, fmt.Errorf(
					"%s: reserved type name '%s'",
					frontend.FormatPos(en.At),
					en.Name,
				)
			}
			if _, exists := enums[fullName]; exists {
				return nil, fmt.Errorf("duplicate enum '%s'", fullName)
			}
			if _, exists := structs[fullName]; exists {
				return nil, fmt.Errorf("duplicate type '%s'", fullName)
			}
			enums[fullName] = enumContext{
				module:  module,
				imports: imports,
				public:  declarationIsPublic(file, en.Public),
				decl:    en,
			}
			checked.Enums = append(
				checked.Enums,
				CheckedEnum{Name: fullName, Module: module, Decl: en},
			)
		}
		for _, st := range file.Structs {
			fullName := qualifyName(module, st.Name)
			if isReservedTypeName(st.Name) {
				return nil, fmt.Errorf(
					"%s: reserved type name '%s'",
					frontend.FormatPos(st.At),
					st.Name,
				)
			}
			if _, exists := structs[fullName]; exists {
				return nil, fmt.Errorf("duplicate struct '%s'", fullName)
			}
			if _, exists := enums[fullName]; exists {
				return nil, fmt.Errorf("duplicate type '%s'", fullName)
			}
			structs[fullName] = structContext{
				module:  module,
				imports: imports,
				public:  declarationIsPublic(file, st.Public),
				decl:    st,
			}
			checked.Structs = append(
				checked.Structs,
				CheckedStruct{Name: fullName, Module: module, Decl: st},
			)
		}
		for _, st := range file.States {
			fullName := qualifyName(module, st.Name)
			if isReservedTypeName(st.Name) {
				return nil, fmt.Errorf(
					"%s: reserved type name '%s'",
					frontend.FormatPos(st.At),
					st.Name,
				)
			}
			if _, exists := structs[fullName]; exists {
				return nil, fmt.Errorf("duplicate state '%s'", fullName)
			}
			if _, exists := enums[fullName]; exists {
				return nil, fmt.Errorf("duplicate type '%s'", fullName)
			}
			synth := stateAsStructDecl(st)
			synth.Public = declarationIsPublic(file, st.Public)
			structs[fullName] = structContext{
				module:  module,
				imports: imports,
				public:  synth.Public,
				decl:    synth,
			}
			checked.UIStates = append(
				checked.UIStates,
				CheckedUIState{Name: fullName, Module: module, Decl: st},
			)
		}
		for _, proto := range file.Protocols {
			fullName := qualifyName(module, proto.Name)
			if isReservedTypeName(proto.Name) {
				return nil, fmt.Errorf(
					"%s: reserved type name '%s'",
					frontend.FormatPos(proto.At),
					proto.Name,
				)
			}
			if _, exists := protocols[fullName]; exists {
				return nil, fmt.Errorf("duplicate protocol '%s'", fullName)
			}
			protocols[fullName] = protocolContext{
				module:  module,
				imports: imports,
				public:  declarationIsPublic(file, proto.Public),
				decl:    proto,
			}
			checked.Protocols = append(
				checked.Protocols,
				CheckedProtocol{Name: fullName, Module: module, Decl: proto},
			)
		}
		for _, actor := range file.Actors {
			if actor == nil {
				continue
			}
			actors = append(actors, actorContext{module: module, imports: imports, decl: actor})
			fields := make(map[string]ActorStateField, len(actor.Fields))
			for _, method := range actor.Methods {
				methodName := checkedFuncFullName(module, method)
				actorStateFieldsByMethod[methodName] = fields
			}
		}
	}

	for name, ctx := range enums {
		caseMap := make(map[string]EnumCaseInfo, len(ctx.decl.Cases))
		cases := make([]EnumCaseInfo, 0, len(ctx.decl.Cases))
		for i, c := range ctx.decl.Cases {
			if _, exists := caseMap[c.Name]; exists {
				return nil, fmt.Errorf(
					"%s: duplicate enum case '%s'",
					frontend.FormatPos(c.At),
					c.Name,
				)
			}
			info := EnumCaseInfo{Name: c.Name, Ordinal: int32(i), SlotCount: 0}
			caseMap[c.Name] = info
			cases = append(cases, info)
		}
		types[name] = &TypeInfo{
			Name:      name,
			Kind:      TypeEnum,
			Public:    ctx.public,
			SlotCount: 1,
			EnumCases: cases,
			CaseMap:   caseMap,
		}
	}

	state := make(map[string]int)
	var buildType func(name string) (*TypeInfo, error)
	buildType = func(name string) (*TypeInfo, error) {
		if info, ok := types[name]; ok {
			return info, nil
		}
		if elem, ok := optionalElemName(name); ok {
			elemInfo, err := buildType(elem)
			if err != nil {
				return nil, err
			}
			info := &TypeInfo{
				Name:      name,
				Kind:      TypeOptional,
				Public:    true,
				SlotCount: elemInfo.SlotCount + 1,
				ElemType:  elem,
			}
			types[name] = info
			return info, nil
		}
		if elem, ok := sliceElemName(name); ok {
			if elem == "" {
				return nil, fmt.Errorf("invalid slice type '%s'", name)
			}
			if isArrayTypeName(elem) {
				return nil, fmt.Errorf("array element types are not supported yet")
			}
			elemInfo, err := buildType(elem)
			if err != nil {
				return nil, err
			}
			if !isSupportedCollectionElemType(elemInfo) {
				return nil, fmt.Errorf("slice element type '%s' is not supported", elem)
			}
			info := makeSliceTypeInfo(name, elem)
			types[name] = info
			return info, nil
		}
		if n, elem, ok := parseArrayTypeName(name); ok {
			if n <= 0 {
				return nil, fmt.Errorf("array size must be positive constant")
			}
			elemInfo, err := buildType(elem)
			if err != nil {
				return nil, err
			}
			if !isSupportedCollectionElemType(elemInfo) {
				return nil, fmt.Errorf("array element type '%s' is not supported", elem)
			}
			info := makeArrayTypeInfo(name, elem, n)
			types[name] = info
			return info, nil
		}
		if isArrayTypeName(name) {
			return nil, fmt.Errorf("invalid array type '%s'", name)
		}
		if isTargetLayoutOnlyScalar(name) {
			return nil, targetLayoutOnlyScalarError(name)
		}
		ctx, ok := structs[name]
		if !ok {
			return nil, fmt.Errorf("unknown type '%s'", name)
		}
		switch state[name] {
		case 1:
			return nil, fmt.Errorf(
				"%s: recursive struct '%s'",
				frontend.FormatPos(ctx.decl.At),
				name,
			)
		case 2:
			if info, ok := types[name]; ok {
				return info, nil
			}
		}
		state[name] = 1

		fieldMap := make(map[string]FieldInfo)
		var fields []FieldInfo
		slotCount := 0
		for i := range ctx.decl.Fields {
			field := &ctx.decl.Fields[i]
			if _, exists := fieldMap[field.Name]; exists {
				return nil, fmt.Errorf(
					"%s: duplicate field '%s'",
					frontend.FormatPos(field.At),
					field.Name,
				)
			}
			resolved, err := resolveTypeName(&field.Type, ctx.module, ctx.imports)
			if err != nil {
				return nil, err
			}
			field.Type.Name = resolved
			fieldType, err := buildType(resolved)
			if err != nil {
				return nil, err
			}
			if err := ensureTypeVisible(resolved, fieldType, ctx.module, field.At); err != nil {
				return nil, err
			}
			if surfaceType, ok := surfaceEphemeralValueType(resolved, types); ok &&
				!surfaceAggregateFieldStorageAllowed(name, field.Name, resolved) {
				return nil, lifetimeDiagnosticf(
					field.At,
					("surface value '%s' cannot be stored in struct field '%s'; " +
						"keep Surface Frame/Event/DrawContext values local to the " +
						"active Surface turn"),
					surfaceType,
					field.Name,
				)
			}
			functionParamTypes := []string(nil)
			functionParamOwnership := []string(nil)
			functionReturnType := ""
			functionReturnOwnership := ""
			functionThrowsType := ""
			functionEffects := []string(nil)
			if field.Type.Kind == frontend.TypeRefFunction {
				functionParamTypes, functionReturnType, functionEffects, err = functionTypeRefSignatureAndEffects(
					field.Type,
					ctx.module,
					ctx.imports,
				)
				if err != nil {
					return nil, err
				}
				functionParamOwnership = functionTypeRefParamOwnership(field.Type)
				functionReturnOwnership = functionTypeRefReturnOwnership(field.Type)
				functionThrowsType, err = functionTypeRefThrowsType(
					field.Type,
					ctx.module,
					ctx.imports,
				)
				if err != nil {
					return nil, err
				}
			}
			info := FieldInfo{
				Name:                    field.Name,
				TypeName:                resolved,
				Offset:                  slotCount,
				SlotCount:               fieldType.SlotCount,
				UserAssignable:          true,
				FunctionTypeValue:       field.Type.Kind == frontend.TypeRefFunction,
				FunctionTypeRef:         field.Type,
				FunctionParamTypes:      functionParamTypes,
				FunctionParamOwnership:  functionParamOwnership,
				FunctionReturnType:      functionReturnType,
				FunctionReturnOwnership: functionReturnOwnership,
				FunctionThrowsType:      functionThrowsType,
				FunctionEffects:         functionEffects,
			}
			fieldMap[field.Name] = info
			fields = append(fields, info)
			slotCount += fieldType.SlotCount
		}

		info := &TypeInfo{
			Name:      name,
			Kind:      TypeStruct,
			Public:    ctx.public,
			Repr:      structReprOrDefault(ctx.decl.Repr),
			Fields:    fields,
			FieldMap:  fieldMap,
			SlotCount: slotCount,
		}
		types[name] = info
		state[name] = 2
		return info, nil
	}

	for name := range structs {
		if _, err := buildType(name); err != nil {
			return nil, err
		}
	}
	for _, ctx := range actors {
		actorName := qualifyName(ctx.module, ctx.decl.Name)
		slot := 0
		for i := range ctx.decl.Fields {
			field := &ctx.decl.Fields[i]
			resolved, err := resolveTypeName(&field.Type, ctx.module, ctx.imports)
			if err != nil {
				return nil, err
			}
			field.Type.Name = resolved
			fieldType, err := buildType(resolved)
			if err != nil {
				return nil, fmt.Errorf(
					"%s: actor '%s' state field '%s': %v",
					frontend.FormatPos(field.At),
					displayTypeName(actorName, ctx.module),
					field.Name,
					err,
				)
			}
			if err := ensureTypeVisible(resolved, fieldType, ctx.module, field.At); err != nil {
				return nil, err
			}
			if !isSupportedActorStateScalarType(resolved) {
				return nil, fmt.Errorf(
					("%s: actor state field '%s' type '%s' is not supported; " +
						"supported actor state field types are Int, Bool, UInt8, " +
						"UInt16, and task.error"),
					frontend.FormatPos(field.At),
					field.Name,
					resolved,
				)
			}
			fieldSlots := fieldType.SlotCount
			if fieldSlots <= 0 {
				fieldSlots = 1
			}
			if slot+fieldSlots > MaxActorStateSlots {
				return nil, fmt.Errorf(
					"%s: actor '%s' state supports at most %d slots, got %d",
					frontend.FormatPos(field.At),
					displayTypeName(actorName, ctx.module),
					MaxActorStateSlots,
					slot+fieldSlots,
				)
			}
			if field.Init == nil {
				return nil, fmt.Errorf(
					"%s: actor state field '%s' requires a compile-time constant initializer",
					frontend.FormatPos(field.At),
					field.Name,
				)
			}
			initType, initValue, ok := evaluateActorStateInitializer(field.Init)
			if !ok {
				return nil, fmt.Errorf(
					"%s: actor state field '%s' initializer must be a compile-time constant Int/Bool expression",
					frontend.FormatPos(field.At),
					field.Name,
				)
			}
			if !typesCompatible(resolved, initType) {
				return nil, fmt.Errorf(
					"%s: actor state field '%s' type mismatch: expected '%s', got '%s'",
					frontend.FormatPos(field.At),
					field.Name,
					resolved,
					initType,
				)
			}
			stateField := ActorStateField{
				Name:     field.Name,
				Slot:     slot,
				TypeName: resolved,
				Mutable:  field.Mutable,
				Const:    field.Const,
				Init:     initValue,
			}
			for _, method := range ctx.decl.Methods {
				methodName := checkedFuncFullName(ctx.module, method)
				if fields, ok := actorStateFieldsByMethod[methodName]; ok {
					fields[field.Name] = stateField
				}
			}
			slot += fieldSlots
		}
	}

	enumPayloadEdges := map[string]map[string]frontend.Position{}
	enumModules := map[string]string{}
	for name, ctx := range enums {
		info := types[name]
		if info == nil || info.Kind != TypeEnum {
			return nil, fmt.Errorf("internal error: enum '%s' has no type info", name)
		}
		enumModules[name] = ctx.module
		if _, ok := enumPayloadEdges[name]; !ok {
			enumPayloadEdges[name] = map[string]frontend.Position{}
		}
		maxPayloadSlots := 0
		for i := range ctx.decl.Cases {
			declCase := &ctx.decl.Cases[i]
			caseInfo := info.EnumCases[i]
			caseInfo.PayloadTypes = caseInfo.PayloadTypes[:0]
			caseInfo.PayloadSlots = caseInfo.PayloadSlots[:0]
			caseInfo.PayloadFunctionTypes = caseInfo.PayloadFunctionTypes[:0]
			caseInfo.PayloadFunctionRefs = caseInfo.PayloadFunctionRefs[:0]
			caseInfo.PayloadFunctionParams = caseInfo.PayloadFunctionParams[:0]
			caseInfo.PayloadFunctionOwns = caseInfo.PayloadFunctionOwns[:0]
			caseInfo.PayloadFunctionReturns = caseInfo.PayloadFunctionReturns[:0]
			caseInfo.PayloadFunctionReturnOwns = caseInfo.PayloadFunctionReturnOwns[:0]
			caseInfo.PayloadFunctionThrows = caseInfo.PayloadFunctionThrows[:0]
			caseInfo.PayloadFunctionEffects = caseInfo.PayloadFunctionEffects[:0]
			totalPayloadSlots := 0
			for j := range declCase.Payload {
				payload := &declCase.Payload[j]
				functionTypeValue := payload.Kind == frontend.TypeRefFunction
				functionParamTypes := []string(nil)
				functionParamOwnership := []string(nil)
				functionReturnType := ""
				functionReturnOwnership := ""
				functionThrowsType := ""
				functionEffects := []string(nil)
				if functionTypeValue {
					var err error
					functionParamTypes, functionReturnType, functionEffects, err = functionTypeRefSignatureAndEffects(
						*payload,
						ctx.module,
						ctx.imports,
					)
					if err != nil {
						return nil, err
					}
					functionParamOwnership = functionTypeRefParamOwnership(*payload)
					functionReturnOwnership = functionTypeRefReturnOwnership(*payload)
					functionThrowsType, err = functionTypeRefThrowsType(
						*payload,
						ctx.module,
						ctx.imports,
					)
					if err != nil {
						return nil, err
					}
				}
				resolved, err := resolveTypeName(payload, ctx.module, ctx.imports)
				if err != nil {
					return nil, err
				}
				if resolved == name {
					return nil, fmt.Errorf(
						"%s: recursive enum payload '%s'",
						frontend.FormatPos(payload.At),
						displayTypeName(name, ctx.module),
					)
				}
				payload.Name = resolved
				payloadInfo, err := buildType(resolved)
				if err != nil {
					return nil, fmt.Errorf(
						"%s: enum '%s' case '%s': %v",
						frontend.FormatPos(payload.At),
						displayTypeName(name, ctx.module),
						declCase.Name,
						err,
					)
				}
				if err := ensureTypeVisible(resolved, payloadInfo, ctx.module, payload.At); err != nil {
					return nil, err
				}
				if surfaceType, ok := surfaceEphemeralValueType(resolved, types); ok {
					return nil, lifetimeDiagnosticf(
						payload.At,
						("surface value '%s' cannot be stored in enum payload '%s'; " +
							"keep Surface Frame/Event/DrawContext values local to the " +
							"active Surface turn"),
						surfaceType,
						declCase.Name,
					)
				}
				if payloadInfo.Kind == TypeEnum {
					enumPayloadEdges[name][resolved] = payload.At
				}
				caseInfo.PayloadTypes = append(caseInfo.PayloadTypes, resolved)
				caseInfo.PayloadSlots = append(caseInfo.PayloadSlots, payloadInfo.SlotCount)
				caseInfo.PayloadFunctionTypes = append(
					caseInfo.PayloadFunctionTypes,
					functionTypeValue,
				)
				caseInfo.PayloadFunctionRefs = append(caseInfo.PayloadFunctionRefs, *payload)
				caseInfo.PayloadFunctionParams = append(
					caseInfo.PayloadFunctionParams,
					functionParamTypes,
				)
				caseInfo.PayloadFunctionOwns = append(
					caseInfo.PayloadFunctionOwns,
					functionParamOwnership,
				)
				caseInfo.PayloadFunctionReturns = append(
					caseInfo.PayloadFunctionReturns,
					functionReturnType,
				)
				caseInfo.PayloadFunctionReturnOwns = append(
					caseInfo.PayloadFunctionReturnOwns,
					functionReturnOwnership,
				)
				caseInfo.PayloadFunctionThrows = append(
					caseInfo.PayloadFunctionThrows,
					functionThrowsType,
				)
				caseInfo.PayloadFunctionEffects = append(
					caseInfo.PayloadFunctionEffects,
					functionEffects,
				)
				totalPayloadSlots += payloadInfo.SlotCount
			}
			caseInfo.SlotCount = totalPayloadSlots
			info.EnumCases[i] = caseInfo
			info.CaseMap[caseInfo.Name] = caseInfo
			if totalPayloadSlots > maxPayloadSlots {
				maxPayloadSlots = totalPayloadSlots
			}
		}
		info.SlotCount = 1 + maxPayloadSlots
	}
	if err := validateEnumPayloadCycles(enumPayloadEdges, enumModules); err != nil {
		return nil, err
	}
	if err := refreshCompositeSlotLayouts(types); err != nil {
		return nil, err
	}

	for name, ctx := range protocols {
		seenReqs := map[string]frontend.Position{}
		for i := range ctx.decl.Requirements {
			req := &ctx.decl.Requirements[i]
			if first, exists := seenReqs[req.Name]; exists {
				return nil, fmt.Errorf(
					"%s: duplicate protocol requirement '%s' (first at %s)",
					frontend.FormatPos(req.At),
					req.Name,
					frontend.FormatPos(first),
				)
			}
			seenReqs[req.Name] = req.At
			reqEffects, err := normalizeEffects(req.Uses, req.At)
			if err != nil {
				return nil, fmt.Errorf(
					"%s: protocol '%s' requirement '%s': %v",
					frontend.FormatPos(req.At),
					name,
					req.Name,
					err,
				)
			}
			req.Uses = reqEffects
			reqTypeParams := make(map[string]struct{}, len(req.TypeParams))
			for _, tp := range req.TypeParams {
				reqTypeParams[tp] = struct{}{}
			}
			retName, retIsGeneric, err := resolveProtocolRequirementTypeRef(
				&req.ReturnType,
				ctx.module,
				ctx.imports,
				reqTypeParams,
			)
			if err != nil {
				return nil, fmt.Errorf(
					"%s: protocol '%s' requirement '%s': %v",
					frontend.FormatPos(req.ReturnType.At),
					name,
					req.Name,
					err,
				)
			}
			req.ReturnType.Name = retName
			if !retIsGeneric {
				retInfo, err := buildType(retName)
				if err != nil {
					return nil, fmt.Errorf(
						"%s: protocol '%s' requirement '%s': %v",
						frontend.FormatPos(req.At),
						name,
						req.Name,
						err,
					)
				}
				if err := ensureTypeVisible(retName, retInfo, ctx.module, req.At); err != nil {
					return nil, err
				}
			}
			if req.HasThrows {
				throwName, throwIsGeneric, err := resolveProtocolRequirementTypeRef(
					&req.Throws,
					ctx.module,
					ctx.imports,
					reqTypeParams,
				)
				if err != nil {
					return nil, fmt.Errorf(
						"%s: protocol '%s' requirement '%s': %v",
						frontend.FormatPos(req.Throws.At),
						name,
						req.Name,
						err,
					)
				}
				req.Throws.Name = throwName
				if !throwIsGeneric {
					throwInfo, err := buildType(throwName)
					if err != nil {
						return nil, fmt.Errorf(
							"%s: protocol '%s' requirement '%s': %v",
							frontend.FormatPos(req.At),
							name,
							req.Name,
							err,
						)
					}
					if err := ensureTypeVisible(throwName, throwInfo, ctx.module, req.Throws.At); err != nil {
						return nil, err
					}
				}
			}
			for j := range req.Params {
				param := &req.Params[j]
				resolved, paramIsGeneric, err := resolveProtocolRequirementTypeRef(
					&param.Type,
					ctx.module,
					ctx.imports,
					reqTypeParams,
				)
				if err != nil {
					return nil, fmt.Errorf(
						"%s: protocol '%s' requirement '%s': %v",
						frontend.FormatPos(param.At),
						name,
						req.Name,
						err,
					)
				}
				param.Type.Name = resolved
				if !paramIsGeneric {
					paramInfo, err := buildType(resolved)
					if err != nil {
						return nil, fmt.Errorf(
							"%s: protocol '%s' requirement '%s': %v",
							frontend.FormatPos(param.At),
							name,
							req.Name,
							err,
						)
					}
					if err := ensureTypeVisible(resolved, paramInfo, ctx.module, param.At); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	builtinSigs, err := builtinFuncSigs(types)
	if err != nil {
		return nil, err
	}

	if err := collectWorldGlobals(world, &checked, types); err != nil {
		return nil, err
	}

	if err := addImportedFunctionTypedGlobalAliases(world, checked.GlobalsByModule); err != nil {
		return nil, err
	}

	for _, file := range world.Files {
		module := file.Module
		imports, err := collectImportAliases(file)
		if err != nil {
			return nil, err
		}
		for _, fn := range file.Funcs {
			fullName := checkedFuncFullName(module, fn)
			if fn.ExportName != "" {
				if fn.ExportName == "core" || strings.HasPrefix(fn.ExportName, "core.") {
					return nil, fmt.Errorf(
						"%s: @export name must not use the 'core.' namespace",
						frontend.FormatPos(fn.Pos),
					)
				}
				if strings.HasPrefix(fn.ExportName, "__tetra_") &&
					!strings.HasPrefix(module, "__") {
					return nil, fmt.Errorf(
						"%s: @export name '%s' is reserved for internal runtime modules",
						frontend.FormatPos(fn.Pos),
						fn.ExportName,
					)
				}
				if other, exists := exportedSymbols[fn.ExportName]; exists {
					return nil, fmt.Errorf(
						"%s: duplicate @export name '%s' (already used by '%s')",
						frontend.FormatPos(fn.Pos),
						fn.ExportName,
						other,
					)
				}
				exportedSymbols[fn.ExportName] = fullName
			}
			if _, exists := builtinSigs[fullName]; exists {
				return nil, fmt.Errorf(
					"%s: cannot redefine builtin '%s'",
					frontend.FormatPos(fn.Pos),
					fullName,
				)
			}
			if _, exists := checked.FuncSigs[fullName]; exists {
				return nil, fmt.Errorf("duplicate function '%s'", fullName)
			}
			if err := validateSemanticClauses(fn); err != nil {
				return nil, err
			}
			if err := validateFunctionParamNames(fn); err != nil {
				return nil, err
			}
			if len(fn.TypeParams) > 0 {
				if err := validateGenericFuncDecl(
					fn,
					module,
					imports,
					collectGenericProtocolInfos(world),
					types,
				); err != nil {
					return nil, err
				}
				if fn.ExportName != "" {
					return nil, fmt.Errorf(
						"%s: generic function '%s' cannot be exported; export a concrete monomorphic wrapper",
						frontend.FormatPos(fn.Pos),
						fn.Name,
					)
				}
				effects, err := normalizeEffects(fn.Uses, fn.Pos)
				if err != nil {
					return nil, err
				}
				genericParamTypes := make(map[string]string, len(fn.Params))
				for _, param := range fn.Params {
					genericParamTypes[param.Name] = genericTypeName(param.Type)
				}
				returnType := genericTypeName(fn.ReturnType)
				throwsType := ""
				if fn.HasThrows {
					throwsType = genericTypeName(fn.Throws)
				}
				if err := validateExportedOpaqueABISignature(
					module,
					fn,
					genericParamTypes,
					returnType,
					types,
				); err != nil {
					return nil, err
				}
				if err := validateFunctionPolicyClauses(
					fn,
					effects,
					genericParamTypes,
					returnType,
					throwsType,
					types,
				); err != nil {
					return nil, err
				}
				if err := validateExportedConsentTokenABISignature(
					module,
					fn,
					genericParamTypes,
					returnType,
					types,
				); err != nil {
					return nil, err
				}
				policy, err := parseFunctionClausePolicy(fn)
				if err != nil {
					return nil, err
				}
				sig, err := buildGenericFuncSig(fullName, funcSigSpec{
					Generic:                      true,
					Public:                       declarationIsPublic(file, fn.Public),
					HasNoAlloc:                   policy.hasNoAlloc,
					HasNoBlock:                   policy.hasNoBlock,
					HasRealtime:                  policy.hasRealtime,
					HasBudget:                    policy.hasBudget,
					Budget:                       policy.budget,
					ParamNames:                   genericParamNames(fn.Params),
					ParamTypes:                   genericParamTypeNames(fn.Params),
					ParamFunctionTypes:           genericParamFunctionKinds(fn.Params),
					ParamFunctionParams:          genericParamFunctionParamTypes(fn.Params),
					ParamFunctionOwnership:       genericParamFunctionOwnership(fn.Params),
					ParamFunctionReturns:         genericParamFunctionReturnTypes(fn.Params),
					ParamFunctionReturnOwnership: genericParamFunctionReturnOwnership(fn.Params),
					ParamFunctionThrows:          genericParamFunctionThrowsTypes(fn.Params),
					ParamFunctionEffects:         genericParamFunctionEffectTypes(fn.Params),
					ParamOwnership:               genericParamOwnership(fn.Params),
					ParamSlots:                   0,
					ReturnType:                   fn.ReturnType.Name,
					ReturnOwnership:              fn.ReturnOwnership,
					ThrowsType:                   fn.Throws.Name,
					Async:                        fn.Async,
					ReturnSlots:                  0,
					ReturnRegionParam:            regionNone,
					ReturnResourceParam:          regionNone,
					ReturnResourcePath:           "",
					Effects:                      effects,
				}, types)
				if err != nil {
					return nil, err
				}
				checked.FuncSigs[fullName] = sig
				continue
			}
			retName, err := resolveTypeName(&fn.ReturnType, module, imports)
			if err != nil {
				return nil, err
			}
			returnFunctionType := fn.ReturnType.Kind == frontend.TypeRefFunction
			returnFunctionParams := []string(nil)
			returnFunctionParamOwnership := []string(nil)
			returnFunctionReturn := ""
			returnFunctionReturnOwnership := ""
			returnFunctionThrows := ""
			returnFunctionEffects := []string(nil)
			if returnFunctionType {
				returnFunctionParams, returnFunctionReturn, returnFunctionEffects, err = functionTypeRefSignatureAndEffects(
					fn.ReturnType,
					module,
					imports,
				)
				if err != nil {
					return nil, err
				}
				returnFunctionParamOwnership = functionTypeRefParamOwnership(fn.ReturnType)
				returnFunctionReturnOwnership = functionTypeRefReturnOwnership(fn.ReturnType)
				returnFunctionThrows, err = functionTypeRefThrowsType(
					fn.ReturnType,
					module,
					imports,
				)
				if err != nil {
					return nil, err
				}
			}
			fn.ReturnType.Name = retName
			retInfo, err := buildType(retName)
			if err != nil {
				return nil, err
			}
			if err := ensureTypeVisible(retName, retInfo, module, fn.ReturnType.At); err != nil {
				return nil, err
			}
			throwsType := ""
			returnSlots := retInfo.SlotCount
			if fn.HasThrows {
				resolvedThrows, err := resolveTypeName(&fn.Throws, module, imports)
				if err != nil {
					return nil, err
				}
				fn.Throws.Name = resolvedThrows
				throwInfo, err := buildType(resolvedThrows)
				if err != nil {
					return nil, err
				}
				if err := ensureTypeVisible(resolvedThrows, throwInfo, module, fn.Throws.At); err != nil {
					return nil, err
				}
				throwsType = resolvedThrows
				returnSlots = throwingReturnSlots(retInfo.SlotCount, throwInfo.SlotCount)
			}
			effects, err := normalizeEffects(fn.Uses, fn.Pos)
			if err != nil {
				return nil, err
			}
			paramTypes := make([]string, 0, len(fn.Params))
			paramNames := make([]string, 0, len(fn.Params))
			paramOwnership := make([]string, 0, len(fn.Params))
			paramSlots := 0
			for i := range fn.Params {
				param := &fn.Params[i]
				resolved, err := resolveTypeName(&param.Type, module, imports)
				if err != nil {
					return nil, err
				}
				param.Type.Name = resolved
				info, err := buildType(resolved)
				if err != nil {
					return nil, err
				}
				if err := ensureTypeVisible(resolved, info, module, param.At); err != nil {
					return nil, err
				}
				paramNames = append(paramNames, param.Name)
				paramTypes = append(paramTypes, resolved)
				paramOwnership = append(paramOwnership, param.Ownership)
				paramSlots += info.SlotCount
			}
			paramTypeByName := make(map[string]string, len(paramNames))
			for i, name := range paramNames {
				paramTypeByName[name] = paramTypes[i]
			}
			if err := validateExportedOpaqueABISignature(
				module,
				fn,
				paramTypeByName,
				retName,
				types,
			); err != nil {
				return nil, err
			}
			if err := validateFunctionPolicyClauses(
				fn,
				effects,
				paramTypeByName,
				retName,
				throwsType,
				types,
			); err != nil {
				return nil, err
			}
			policy, err := parseFunctionClausePolicy(fn)
			if err != nil {
				return nil, err
			}
			sig, err := buildDeclaredFuncSig(fullName, funcSigSpec{
				Public:                        declarationIsPublic(file, fn.Public),
				HasNoAlloc:                    policy.hasNoAlloc,
				HasNoBlock:                    policy.hasNoBlock,
				HasRealtime:                   policy.hasRealtime,
				HasBudget:                     policy.hasBudget,
				Budget:                        policy.budget,
				ParamNames:                    paramNames,
				ParamTypes:                    paramTypes,
				ParamFunctionTypes:            paramFunctionKinds(fn.Params),
				ParamFunctionParams:           paramFunctionParamTypes(fn.Params),
				ParamFunctionOwnership:        paramFunctionOwnership(fn.Params),
				ParamFunctionReturns:          paramFunctionReturnTypes(fn.Params),
				ParamFunctionReturnOwnership:  paramFunctionReturnOwnership(fn.Params),
				ParamFunctionThrows:           paramFunctionThrowsTypes(fn.Params, module, imports),
				ParamFunctionEffects:          paramFunctionEffectTypes(fn.Params),
				ParamOwnership:                paramOwnership,
				ParamSlots:                    paramSlots,
				ReturnType:                    retName,
				ReturnOwnership:               fn.ReturnOwnership,
				ReturnFunctionType:            returnFunctionType,
				ReturnFunctionParams:          returnFunctionParams,
				ReturnFunctionParamOwnership:  returnFunctionParamOwnership,
				ReturnFunctionReturn:          returnFunctionReturn,
				ReturnFunctionReturnOwnership: returnFunctionReturnOwnership,
				ReturnFunctionThrows:          returnFunctionThrows,
				ReturnFunctionEffects:         returnFunctionEffects,
				ThrowsType:                    throwsType,
				Async:                         fn.Async,
				ReturnSlots:                   returnSlots,
				ReturnRegionParam:             regionNone,
				ReturnResourceParam:           initialReturnResourceParam(retName, types),
				ReturnResourcePath:            "",
				Effects:                       effects,
			}, types)
			if err != nil {
				return nil, err
			}
			checked.FuncSigs[fullName] = sig
		}
	}

	if err := addPublicImportFunctionAliases(world, checked.FuncSigs); err != nil {
		return nil, err
	}

	for _, file := range world.Files {
		module := file.Module
		imports, err := collectImportAliases(file)
		if err != nil {
			return nil, err
		}
		for _, impl := range file.Impls {
			typeName, err := resolveTypeName(&impl.Type, module, imports)
			if err != nil {
				return nil, err
			}
			protoName, err := resolveTypeName(&impl.Protocol, module, imports)
			if err != nil {
				return nil, err
			}
			impl.Type.Name = typeName
			impl.Protocol.Name = protoName
			if _, ok := types[typeName]; !ok {
				return nil, fmt.Errorf(
					"%s: impl target type '%s' is not defined",
					frontend.FormatPos(impl.Type.At),
					typeName,
				)
			}
			typeInfo := types[typeName]
			if err := ensureTypeVisible(typeName, typeInfo, module, impl.Type.At); err != nil {
				return nil, err
			}
			implKey := typeName + "->" + protoName
			if first, exists := seenImpls[implKey]; exists {
				return nil, fmt.Errorf(
					"%s: duplicate impl conformance '%s: %s' (first at %s)",
					frontend.FormatPos(impl.At),
					typeName,
					protoName,
					frontend.FormatPos(first),
				)
			}
			seenImpls[implKey] = impl.At
			proto, ok := protocols[protoName]
			if !ok {
				if _, isType := types[protoName]; isType {
					return nil, fmt.Errorf(
						"%s: impl protocol '%s' must name a protocol, got non-protocol type '%s'",
						frontend.FormatPos(impl.Protocol.At),
						displayTypeName(protoName, module),
						displayTypeName(protoName, module),
					)
				}
				return nil, fmt.Errorf(
					"%s: protocol '%s' is not defined",
					frontend.FormatPos(impl.Protocol.At),
					protoName,
				)
			}
			if !symbolBelongsToModule(protoName, module) && !proto.public {
				return nil, fmt.Errorf(
					"%s: private protocol '%s' is not visible from module '%s'",
					frontend.FormatPos(impl.Protocol.At),
					protoName,
					module,
				)
			}
			for _, req := range proto.decl.Requirements {
				methodName := typeName + "." + req.Name
				method, methodModule, methodImports, err := findFuncDecl(world, methodName)
				if err != nil {
					return nil, err
				}
				if method == nil {
					return nil, fmt.Errorf(
						"%s: type '%s' is missing protocol requirement '%s'",
						frontend.FormatPos(impl.At),
						typeName,
						req.Name,
					)
				}
				if err := compareProtocolRequirement(
					typeName,
					protoName,
					req,
					method,
					methodModule,
					methodImports,
				); err != nil {
					return nil, err
				}
			}
		}
	}

	for name, sig := range builtinSigs {
		sig.Public = true
		checked.FuncSigs[name] = sig
	}
	if err := checkUIDecls(world, &checked, types); err != nil {
		return nil, err
	}

	if len(checked.FuncSigs) == 0 {
		return nil, fmt.Errorf("no functions found")
	}

	funcCount := 0
	for _, file := range world.Files {
		funcCount += len(file.Funcs)
	}
	maxIter := funcCount + 1
	analysisFiles := worldFilesImportsFirst(world)
	functionReturnSecretTaint := make(map[string]bool)
	functionParamSecretTaint := make(map[string]map[string]bool)
	for iter := 0; iter < maxIter; iter++ {
		changed := false
		for _, file := range analysisFiles {
			module := file.Module
			interfaceModule := world.InterfaceModules[module]
			imports, err := collectImportAliases(file)
			if err != nil {
				return nil, err
			}
			globals := checked.GlobalsByModule[module]
			for _, fn := range file.Funcs {
				fullName := checkedFuncFullName(module, fn)
				if interfaceModule {
					sig := checked.FuncSigs[fullName]
					updated, err := applyInterfaceFunctionReturnMetadata(
						&sig,
						fn,
						globals,
						checked.FuncSigs,
						types,
						module,
						imports,
					)
					if err != nil {
						return nil, err
					}
					if updated {
						checked.FuncSigs[fullName] = sig
						changed = true
					}
					continue
				}
				if len(fn.TypeParams) > 0 {
					continue
				}
				if len(fn.Body) == 0 {
					return nil, fmt.Errorf("function '%s' must have a body", fullName)
				}

				locals := make(map[string]LocalInfo)
				scopeInfo := newScopeInfo()
				slotIndex := 0
				for _, param := range fn.Params {
					if _, exists := locals[param.Name]; exists {
						return nil, fmt.Errorf(
							"%s: duplicate local '%s'",
							frontend.FormatPos(param.At),
							param.Name,
						)
					}
					paramTypeName, err := resolveTypeName(&param.Type, module, imports)
					if err != nil {
						return nil, err
					}
					param.Type.Name = paramTypeName
					info, err := buildType(paramTypeName)
					if err != nil {
						return nil, err
					}
					functionTypeValue := param.Type.Kind == frontend.TypeRefFunction
					functionParamTypes := []string(nil)
					functionParamOwnership := []string(nil)
					functionReturnType := ""
					functionReturnOwnership := ""
					functionThrowsType := ""
					functionEffects := []string(nil)
					if functionTypeValue {
						functionParamTypes, functionReturnType, functionEffects, err = functionTypeRefSignatureAndEffects(
							param.Type,
							module,
							imports,
						)
						if err != nil {
							return nil, err
						}
						functionParamOwnership = functionTypeRefParamOwnership(param.Type)
						functionReturnOwnership = functionTypeRefReturnOwnership(param.Type)
						functionThrowsType, err = functionTypeRefThrowsType(
							param.Type,
							module,
							imports,
						)
						if err != nil {
							return nil, err
						}
					}
					locals[param.Name] = LocalInfo{
						Base:              slotIndex,
						SlotCount:         info.SlotCount,
						TypeName:          paramTypeName,
						Mutable:           param.Ownership == "inout",
						FunctionTypeValue: functionTypeValue,
						FunctionParamName: functionParamNameForParam(
							param.Name,
							functionTypeValue,
						),
						FunctionParamTypes:      functionParamTypes,
						FunctionParamOwnership:  functionParamOwnership,
						FunctionReturnType:      functionReturnType,
						FunctionReturnOwnership: functionReturnOwnership,
						FunctionThrowsType:      functionThrowsType,
						FunctionEffects:         functionEffects,
						FunctionFields: functionFieldsForStructParameter(
							param.Name,
							paramTypeName,
							types,
						),
						EnumPayloadFunctions: enumPayloadFunctionsForEnumParameter(
							param.Name,
							paramTypeName,
							types,
						),
						EnumPayloadFields: enumPayloadFieldsForStructParameter(
							param.Name,
							paramTypeName,
							types,
						),
					}
					scopeInfo.localScopes[param.Name] = regionNone
					slotIndex += info.SlotCount
				}
				if fields, ok := actorStateFieldsByMethod[fullName]; ok {
					if err := injectActorStateLocals(fields, locals, scopeInfo); err != nil {
						return nil, err
					}
				}
				if err := collectLocals(
					fn.Body,
					locals,
					&slotIndex,
					checked.FuncSigs,
					types,
					module,
					imports,
					scopeInfo,
					globals,
				); err != nil {
					return nil, err
				}
				if !stmtListEndsWithReturnTyped(
					fn.Body,
					locals,
					globals,
					checked.FuncSigs,
					types,
					module,
					imports,
				) {
					if err := reportStmtListMatchPayloadDiagnostics(fn.Body, module); err != nil {
						return nil, err
					}
					return nil, fmt.Errorf("function '%s' must end with return", fullName)
				}
				if err := reportStmtListMatchPayloadDiagnostics(fn.Body, module); err != nil {
					return nil, err
				}
				state := newRegionState(scopeInfo)
				if fields, ok := actorStateFieldsByMethod[fullName]; ok {
					state.actorStateFields = fields
				}
				initParamRegions(fn.Params, state, types)
				sig := checked.FuncSigs[fullName]
				state.throwType = sig.ThrowsType
				state.async = sig.Async
				effects := newEffectContext(
					fullName,
					sig.Effects,
					fn.Uses,
					strings.HasPrefix(module, "__"),
				)
				effects.capsulePerms = capsulePermissionsByModule[module]
				borrowedParams := make(map[string]struct{})
				inoutParams := make(map[string]struct{})
				for _, param := range fn.Params {
					if param.Ownership == "borrow" {
						borrowedParams[param.Name] = struct{}{}
					} else if param.Ownership == "inout" {
						inoutParams[param.Name] = struct{}{}
					}
				}
				policy, err := parseFunctionClausePolicy(fn)
				if err != nil {
					return nil, err
				}
				analysis := newFunctionAnalysisState(
					fn,
					policy,
					fullName,
					functionReturnSecretTaint,
					functionParamSecretTaint,
					types,
				)
				if err := checkStmts(
					fn.Body,
					locals,
					globals,
					checked.FuncSigs,
					types,
					module,
					imports,
					sig.ReturnType,
					borrowedParams,
					inoutParams,
					state,
					effects,
					analysis,
				); err != nil {
					return nil, err
				}
				newReturnParam := regionNone
				newReturnRegionSummary := ReturnRegionSummary(nil)
				if len(state.returnRegionSummary) > 0 {
					newReturnRegionSummary = cloneReturnRegionSummary(state.returnRegionSummary)
					commonParam := regionNone
					for _, paramIndex := range newReturnRegionSummary {
						if commonParam == regionNone {
							commonParam = paramIndex
							continue
						}
						if commonParam != paramIndex {
							commonParam = regionUnknown
							break
						}
					}
					if commonParam >= 0 {
						newReturnParam = commonParam
					}
				} else if state.returnRegionSet && state.returnRegion < regionNone {
					idx, ok := state.paramRegionIndex[state.returnRegion]
					if !ok {
						return nil, fmt.Errorf(
							"%s: return region does not match parameter",
							frontend.FormatPos(fn.Pos),
						)
					}
					newReturnParam = idx
				}
				if sig.ReturnRegionParam != newReturnParam ||
					!returnRegionSummariesEqual(sig.ReturnRegionSummary, newReturnRegionSummary) {
					sig.ReturnRegionParam = newReturnParam
					sig.ReturnRegionSummary = newReturnRegionSummary
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				newReturnResourceParam := regionNone
				newReturnResourcePath := ""
				newReturnResourceSummary := ReturnResourceSummary(nil)
				if typeContainsResourceHandle(sig.ReturnType, types) && state.returnResourceSet {
					newReturnResourceParam = state.returnResourceParam
					newReturnResourcePath = state.returnResourcePath
					newReturnResourceSummary = cloneReturnResourceSummary(
						state.returnResourceSummary,
					)
				} else if typeContainsResourceHandle(sig.ReturnType, types) && state.returnResourceUnknown {
					newReturnResourceParam = regionUnknown
				}
				if sig.ReturnResourceParam != newReturnResourceParam ||
					sig.ReturnResourcePath != newReturnResourcePath ||
					!returnResourceSummariesEqual(
						sig.ReturnResourceSummary,
						newReturnResourceSummary,
					) {
					sig.ReturnResourceParam = newReturnResourceParam
					sig.ReturnResourcePath = newReturnResourcePath
					sig.ReturnResourceSummary = newReturnResourceSummary
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				newThrowResourceSummary := ReturnResourceSummary(nil)
				if typeContainsResourceHandle(sig.ThrowsType, types) &&
					len(state.throwResourceSummary) > 0 {
					newThrowResourceSummary = cloneReturnResourceSummary(state.throwResourceSummary)
				}
				if !returnResourceSummariesEqual(
					sig.ThrowResourceSummary,
					newThrowResourceSummary,
				) {
					sig.ThrowResourceSummary = newThrowResourceSummary
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				if sig.TouchesMutableGlobals != analysis.touchesMutableGlobals {
					sig.TouchesMutableGlobals = analysis.touchesMutableGlobals
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				if sig.ReturnFunctionSymbol != analysis.returnFunctionSymbol {
					sig.ReturnFunctionSymbol = analysis.returnFunctionSymbol
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				if sig.ReturnFunctionParamName != analysis.returnFunctionParamName {
					sig.ReturnFunctionParamName = analysis.returnFunctionParamName
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				if !closureCapturesEqual(
					sig.ReturnFunctionCaptures,
					analysis.returnFunctionCaptures,
				) {
					sig.ReturnFunctionCaptures = append(
						[]frontend.ClosureCapture(nil),
						analysis.returnFunctionCaptures...)
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				if sig.ReturnFunctionEscapeKind != analysis.returnFunctionEscapeKind {
					sig.ReturnFunctionEscapeKind = analysis.returnFunctionEscapeKind
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				if sig.ReturnFunctionHandleValue != analysis.returnFunctionHandleValue {
					sig.ReturnFunctionHandleValue = analysis.returnFunctionHandleValue
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				desiredReturnSlots := sig.ReturnSlots
				if sig.ReturnFunctionHandleValue {
					desiredReturnSlots = CallableHandleSlotCount
				}
				if sig.ReturnSlots != desiredReturnSlots {
					sig.ReturnSlots = desiredReturnSlots
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				if sig.ReturnFunctionTouchesMutableGlobals != analysis.returnFunctionTouchesMutableGlobals {
					sig.ReturnFunctionTouchesMutableGlobals = analysis.returnFunctionTouchesMutableGlobals
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				if !functionFieldMapsEqual(
					sig.ReturnFunctionFields,
					analysis.returnFunctionFields,
				) {
					sig.ReturnFunctionFields = cloneFunctionFieldMap(analysis.returnFunctionFields)
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				if !functionFieldMapsEqual(
					sig.ReturnEnumPayloadFunctions,
					analysis.returnEnumPayloadFunctions,
				) {
					sig.ReturnEnumPayloadFunctions = cloneFunctionFieldMap(
						analysis.returnEnumPayloadFunctions,
					)
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				if !functionFieldMapsEqual(
					sig.ReturnEnumPayloadFields,
					analysis.returnEnumPayloadFields,
				) {
					sig.ReturnEnumPayloadFields = cloneFunctionFieldMap(
						analysis.returnEnumPayloadFields,
					)
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				if functionReturnSecretTaint[fullName] != analysis.returnSecretTaint {
					functionReturnSecretTaint[fullName] = analysis.returnSecretTaint
					changed = true
				}
				if analysis.discoveredParamTaint {
					changed = true
				}
			}
		}
		if !changed {
			break
		}
		if iter == maxIter-1 {
			return nil, fmt.Errorf("region inference did not converge")
		}
	}
	for name, sig := range checked.FuncSigs {
		if typeContainsResourceHandle(sig.ReturnType, types) &&
			sig.ReturnResourceParam == regionUnknown &&
			len(sig.ReturnResourceSummary) == 0 {
			return nil, fmt.Errorf(
				"resource return provenance could not be inferred for function '%s'",
				name,
			)
		}
	}
	for _, file := range analysisFiles {
		module := file.Module
		if world.InterfaceModules[module] {
			continue
		}
		for _, fn := range file.Funcs {
			if len(fn.TypeParams) > 0 {
				continue
			}
			fullName := checkedFuncFullName(module, fn)
			paramTypeByName := make(map[string]string, len(fn.Params))
			for _, param := range fn.Params {
				paramTypeByName[param.Name] = param.Type.Name
			}
			if err := validateExportedConsentTokenABISignature(
				module,
				fn,
				paramTypeByName,
				fn.ReturnType.Name,
				types,
			); err != nil {
				return nil, err
			}
			sig := checked.FuncSigs[fullName]
			if err := validateExportedThrowingABISignature(module, fn, sig.ThrowsType); err != nil {
				return nil, err
			}
		}
	}
	if err := validateBudgetContexts(world, checked.FuncSigs); err != nil {
		return nil, err
	}

	for _, file := range analysisFiles {
		module := file.Module
		interfaceModule := world.InterfaceModules[module]
		imports, err := collectImportAliases(file)
		if err != nil {
			return nil, err
		}
		globals := checked.GlobalsByModule[module]
		for _, fn := range file.Funcs {
			fullName := checkedFuncFullName(module, fn)
			if interfaceModule {
				continue
			}
			if len(fn.TypeParams) > 0 {
				continue
			}
			if fn.Name == "main" {
				if module != world.EntryModule {
					return nil, fmt.Errorf(
						"%s: main must be in entry module",
						frontend.FormatPos(fn.Pos),
					)
				}
				if len(fn.Params) != 0 {
					return nil, fmt.Errorf(
						"%s: main must not have parameters",
						frontend.FormatPos(fn.Pos),
					)
				}
				if checked.FuncSigs[fullName].ReturnType != "i32" {
					return nil, fmt.Errorf("%s: main must return i32", frontend.FormatPos(fn.Pos))
				}
				if checked.FuncSigs[fullName].ThrowsType != "" {
					return nil, fmt.Errorf("%s: main must not throw", frontend.FormatPos(fn.Pos))
				}
				if checked.FuncSigs[fullName].Async {
					return nil, fmt.Errorf("%s: main must not be async", frontend.FormatPos(fn.Pos))
				}
				checked.MainIndex = len(checked.Funcs)
				checked.MainName = fullName
			}
			locals := make(map[string]LocalInfo)
			scopeInfo := newScopeInfo()
			slotIndex := 0
			for _, param := range fn.Params {
				if _, exists := locals[param.Name]; exists {
					return nil, fmt.Errorf(
						"%s: duplicate local '%s'",
						frontend.FormatPos(param.At),
						param.Name,
					)
				}
				paramTypeName, err := resolveTypeName(&param.Type, module, imports)
				if err != nil {
					return nil, err
				}
				param.Type.Name = paramTypeName
				info, err := buildType(paramTypeName)
				if err != nil {
					return nil, err
				}
				functionTypeValue := param.Type.Kind == frontend.TypeRefFunction
				functionParamTypes := []string(nil)
				functionParamOwnership := []string(nil)
				functionReturnType := ""
				functionReturnOwnership := ""
				functionThrowsType := ""
				functionEffects := []string(nil)
				if functionTypeValue {
					functionParamTypes, functionReturnType, functionEffects, err = functionTypeRefSignatureAndEffects(
						param.Type,
						module,
						imports,
					)
					if err != nil {
						return nil, err
					}
					functionParamOwnership = functionTypeRefParamOwnership(param.Type)
					functionReturnOwnership = functionTypeRefReturnOwnership(param.Type)
					functionThrowsType, err = functionTypeRefThrowsType(param.Type, module, imports)
					if err != nil {
						return nil, err
					}
				}
				locals[param.Name] = LocalInfo{
					Base:                    slotIndex,
					SlotCount:               info.SlotCount,
					TypeName:                paramTypeName,
					Mutable:                 false,
					FunctionTypeValue:       functionTypeValue,
					FunctionParamTypes:      functionParamTypes,
					FunctionParamOwnership:  functionParamOwnership,
					FunctionReturnType:      functionReturnType,
					FunctionReturnOwnership: functionReturnOwnership,
					FunctionThrowsType:      functionThrowsType,
					FunctionEffects:         functionEffects,
					FunctionFields: functionFieldsForStructParameter(
						param.Name,
						paramTypeName,
						types,
					),
					EnumPayloadFunctions: enumPayloadFunctionsForEnumParameter(
						param.Name,
						paramTypeName,
						types,
					),
					EnumPayloadFields: enumPayloadFieldsForStructParameter(
						param.Name,
						paramTypeName,
						types,
					),
				}
				scopeInfo.localScopes[param.Name] = regionNone
				slotIndex += info.SlotCount
			}
			if fields, ok := actorStateFieldsByMethod[fullName]; ok {
				if err := injectActorStateLocals(fields, locals, scopeInfo); err != nil {
					return nil, err
				}
			}
			if err := collectLocals(
				fn.Body,
				locals,
				&slotIndex,
				checked.FuncSigs,
				types,
				module,
				imports,
				scopeInfo,
				globals,
			); err != nil {
				return nil, err
			}
			sig := checked.FuncSigs[fullName]
			localSlots := slotIndex
			if sig.ThrowsType != "" {
				throwInfo, err := ensureTypeInfo(sig.ThrowsType, types)
				if err != nil {
					return nil, err
				}
				localSlots += throwingScratchSlots(throwInfo.SlotCount)
			}
			var actorState map[string]ActorStateField
			if fields, ok := actorStateFieldsByMethod[fullName]; ok {
				actorState = make(map[string]ActorStateField, len(fields))
				for name, value := range fields {
					actorState[name] = value
				}
			}
			checked.Funcs = append(checked.Funcs, CheckedFunc{
				Name:                  fullName,
				Module:                module,
				Decl:                  fn,
				Imports:               cloneStringMap(imports),
				Locals:                locals,
				ActorState:            actorState,
				LocalSlots:            localSlots,
				ParamSlots:            sig.ParamSlots,
				ReturnType:            sig.ReturnType,
				ReturnOwnership:       sig.ReturnOwnership,
				ThrowsType:            sig.ThrowsType,
				Async:                 sig.Async,
				ReturnSlots:           sig.ReturnSlots,
				Effects:               append([]string(nil), sig.Effects...),
				TouchesMutableGlobals: sig.TouchesMutableGlobals,
			})
		}
	}

	if checked.MainIndex == -1 {
		if opt.RequireMain {
			return nil, fmt.Errorf("missing main")
		}
	}

	return &checked, nil
}

// ---- checker_abi_secret_protocol.go ----

func validateExportedConsentTokenABISignature(
	_ string,
	fn *frontend.FuncDecl,
	paramTypes map[string]string,
	returnType string,
	types map[string]*TypeInfo,
) error {
	if fn == nil || fn.ExportName == "" {
		return nil
	}
	for _, param := range fn.Params {
		paramType := paramTypes[param.Name]
		if isForgeableConsentTokenType(paramType) {
			return effectDiagnosticf(
				param.At,
				"exported function '%s' cannot expose forgeable consent token '%s' in parameter '%s'",
				fn.Name,
				paramType,
				param.Name,
			)
		}
		if exposure, ok := exportedConsentTokenABIExposureForType(paramType, types); ok {
			return effectDiagnosticf(
				param.At,
				"exported function '%s' cannot expose %s '%s' through parameter '%s' type '%s'",
				fn.Name,
				exposure.Kind,
				exposure.TypeName,
				param.Name,
				paramType,
			)
		}
	}
	if isForgeableConsentTokenType(returnType) {
		return effectDiagnosticf(
			fn.ReturnType.At,
			"exported function '%s' cannot expose forgeable consent token '%s' in return type",
			fn.Name,
			returnType,
		)
	}
	if exposure, ok := exportedConsentTokenABIExposureForType(returnType, types); ok {
		return effectDiagnosticf(
			fn.ReturnType.At,
			"exported function '%s' cannot expose %s '%s' through return type '%s'",
			fn.Name,
			exposure.Kind,
			exposure.TypeName,
			returnType,
		)
	}
	return nil
}

func validateExportedThrowingABISignature(
	_ string,
	fn *frontend.FuncDecl,
	throwsType string,
) error {
	if fn == nil || fn.ExportName == "" || strings.TrimSpace(throwsType) == "" {
		return nil
	}
	return effectDiagnosticf(
		fn.Throws.At,
		("exported function '%s' cannot throw typed error '%s'; " +
			"export a non-throwing wrapper with an explicit result type"),
		fn.Name,
		throwsType,
	)
}

func exportedConsentTokenABIExposureForType(
	typeName string,
	types map[string]*TypeInfo,
) (exportedOpaqueABIExposure, bool) {
	return exportedConsentTokenABIExposureForTypeVisiting(
		strings.TrimSpace(typeName),
		types,
		map[string]bool{},
	)
}

func exportedConsentTokenABIExposureForTypeVisiting(
	typeName string,
	types map[string]*TypeInfo,
	visiting map[string]bool,
) (exportedOpaqueABIExposure, bool) {
	typeName = strings.TrimSpace(typeName)
	if isForgeableConsentTokenType(typeName) {
		return exportedOpaqueABIExposure{Kind: "forgeable consent token", TypeName: typeName}, true
	}
	if elem, ok := optionalElemName(typeName); ok {
		return exportedConsentTokenABIExposureForTypeVisiting(elem, types, visiting)
	}
	if _, elem, ok := parseArrayTypeName(typeName); ok {
		return exportedConsentTokenABIExposureForTypeVisiting(elem, types, visiting)
	}
	info, ok := types[typeName]
	if !ok {
		return exportedOpaqueABIExposure{}, false
	}
	switch info.Kind {
	case TypeStruct:
		if visiting[typeName] {
			return exportedOpaqueABIExposure{}, false
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, field := range info.Fields {
			if exposure, ok := exportedConsentTokenABIExposureForTypeVisiting(
				field.TypeName,
				types,
				visiting,
			); ok {
				return exposure, true
			}
		}
	case TypeEnum:
		if visiting[typeName] {
			return exportedOpaqueABIExposure{}, false
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, enumCase := range info.EnumCases {
			for _, payload := range enumCase.PayloadTypes {
				if exposure, ok := exportedConsentTokenABIExposureForTypeVisiting(
					payload,
					types,
					visiting,
				); ok {
					return exposure, true
				}
			}
		}
	case TypeArray, TypeOptional:
		return exportedConsentTokenABIExposureForTypeVisiting(info.ElemType, types, visiting)
	}
	return exportedOpaqueABIExposure{}, false
}

func isInternalRuntimeABIExport(module string, fn *frontend.FuncDecl) bool {
	if fn == nil || !strings.HasPrefix(fn.ExportName, "__tetra_") {
		return false
	}
	return module == "__rt" || strings.HasPrefix(module, "__rt.")
}

type exportedOpaqueABIExposure struct {
	Kind     string
	TypeName string
}

func exportedOpaqueABIExposureForType(
	typeName string,
	types map[string]*TypeInfo,
	allowRuntimeHandles bool,
) (exportedOpaqueABIExposure, bool) {
	return exportedOpaqueABIExposureForTypeVisiting(
		strings.TrimSpace(typeName),
		types,
		allowRuntimeHandles,
		map[string]bool{},
	)
}

func exportedDefaultStructABIExposureForType(
	typeName string,
	types map[string]*TypeInfo,
) (exportedOpaqueABIExposure, bool) {
	return exportedDefaultStructABIExposureForTypeVisiting(
		strings.TrimSpace(typeName),
		types,
		map[string]bool{},
	)
}

func exportedDefaultStructABIExposureForTypeVisiting(
	typeName string,
	types map[string]*TypeInfo,
	visiting map[string]bool,
) (exportedOpaqueABIExposure, bool) {
	typeName = strings.TrimSpace(typeName)
	if elem, ok := optionalElemName(typeName); ok {
		return exportedDefaultStructABIExposureForTypeVisiting(elem, types, visiting)
	}
	if _, elem, ok := parseArrayTypeName(typeName); ok {
		return exportedDefaultStructABIExposureForTypeVisiting(elem, types, visiting)
	}
	info, ok := types[typeName]
	if !ok {
		return exportedOpaqueABIExposure{}, false
	}
	switch info.Kind {
	case TypeStruct:
		if info.Repr != frontend.StructReprC {
			return exportedOpaqueABIExposure{
				Kind:     "default-layout struct",
				TypeName: info.Name,
			}, true
		}
		if visiting[typeName] {
			return exportedOpaqueABIExposure{}, false
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, field := range info.Fields {
			if exposure, ok := exportedDefaultStructABIExposureForTypeVisiting(
				field.TypeName,
				types,
				visiting,
			); ok {
				return exposure, true
			}
		}
	case TypeArray, TypeOptional:
		return exportedDefaultStructABIExposureForTypeVisiting(info.ElemType, types, visiting)
	}
	return exportedOpaqueABIExposure{}, false
}

func exportedOpaqueABIExposureForTypeVisiting(
	typeName string,
	types map[string]*TypeInfo,
	allowRuntimeHandles bool,
	visiting map[string]bool,
) (exportedOpaqueABIExposure, bool) {
	if isOpaqueCapabilityTokenType(typeName) {
		return exportedOpaqueABIExposure{Kind: "opaque capability token", TypeName: typeName}, true
	}
	if isOpaqueIslandHandleType(typeName) {
		return exportedOpaqueABIExposure{Kind: "opaque island handle", TypeName: typeName}, true
	}
	if isFunctionTypedABIValueType(typeName) {
		return exportedOpaqueABIExposure{Kind: "function-typed value", TypeName: typeName}, true
	}
	if exposure, ok := exportedRawViewABIExposureForType(typeName, types); ok {
		return exposure, true
	}
	if !allowRuntimeHandles {
		if exposure, ok := exportedBoolABIExposureForType(typeName, types); ok {
			return exposure, true
		}
	}
	if !allowRuntimeHandles && isOpaqueRuntimeHandleType(typeName) {
		return exportedOpaqueABIExposure{Kind: "opaque runtime handle", TypeName: typeName}, true
	}
	if elem, ok := optionalElemName(typeName); ok {
		if exposure, ok := exportedOpaqueABIExposureForTypeVisiting(
			elem,
			types,
			allowRuntimeHandles,
			visiting,
		); ok {
			return exposure, true
		}
		return exportedOpaqueABIExposure{
			Kind:     "forgeable optional presence tag",
			TypeName: typeName,
		}, true
	}
	if _, elem, ok := parseArrayTypeName(typeName); ok {
		return exportedOpaqueABIExposureForTypeVisiting(elem, types, allowRuntimeHandles, visiting)
	}
	info, ok := types[typeName]
	if !ok {
		return exportedOpaqueABIExposure{}, false
	}
	switch info.Kind {
	case TypeStruct:
		if visiting[typeName] {
			return exportedOpaqueABIExposure{}, false
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, field := range info.Fields {
			if exposure, ok := exportedOpaqueABIExposureForTypeVisiting(
				field.TypeName,
				types,
				allowRuntimeHandles,
				visiting,
			); ok {
				return exposure, true
			}
		}
	case TypeEnum:
		if visiting[typeName] {
			return exportedOpaqueABIExposure{}, false
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, enumCase := range info.EnumCases {
			for _, payload := range enumCase.PayloadTypes {
				if exposure, ok := exportedOpaqueABIExposureForTypeVisiting(
					payload,
					types,
					allowRuntimeHandles,
					visiting,
				); ok {
					return exposure, true
				}
			}
		}
		return exportedOpaqueABIExposure{
			Kind:     "forgeable enum discriminant",
			TypeName: info.Name,
		}, true
	case TypeArray:
		return exportedOpaqueABIExposureForTypeVisiting(
			info.ElemType,
			types,
			allowRuntimeHandles,
			visiting,
		)
	case TypeOptional:
		if exposure, ok := exportedOpaqueABIExposureForTypeVisiting(
			info.ElemType,
			types,
			allowRuntimeHandles,
			visiting,
		); ok {
			return exposure, true
		}
		return exportedOpaqueABIExposure{
			Kind:     "forgeable optional presence tag",
			TypeName: info.Name,
		}, true
	}
	return exportedOpaqueABIExposure{}, false
}

func isOpaqueIslandHandleType(typeName string) bool {
	return strings.TrimSpace(typeName) == "island"
}

func isForgeableConsentTokenType(typeName string) bool {
	return strings.TrimSpace(typeName) == "consent.token"
}

func isFunctionTypedABIValueType(typeName string) bool {
	return strings.TrimSpace(typeName) == "fnptr"
}

func exportedBoolABIExposureForType(
	typeName string,
	types map[string]*TypeInfo,
) (exportedOpaqueABIExposure, bool) {
	typeName = strings.TrimSpace(typeName)
	info, ok := types[typeName]
	if !ok || info.Kind != TypeBool {
		return exportedOpaqueABIExposure{}, false
	}
	return exportedOpaqueABIExposure{Kind: "unnormalized bool", TypeName: info.Name}, true
}

func exportedRawViewABIExposureForType(
	typeName string,
	types map[string]*TypeInfo,
) (exportedOpaqueABIExposure, bool) {
	typeName = strings.TrimSpace(typeName)
	info, ok := types[typeName]
	if !ok {
		return exportedOpaqueABIExposure{}, false
	}
	switch info.Kind {
	case TypeStr:
		return exportedOpaqueABIExposure{Kind: "raw string view", TypeName: info.Name}, true
	case TypeSlice:
		return exportedOpaqueABIExposure{Kind: "raw slice view", TypeName: info.Name}, true
	case TypeArray:
		return exportedOpaqueABIExposure{Kind: "raw fixed-array view", TypeName: info.Name}, true
	default:
		return exportedOpaqueABIExposure{}, false
	}
}

func isOpaqueCapabilityTokenType(typeName string) bool {
	switch strings.TrimSpace(typeName) {
	case "cap.io", "cap.mem":
		return true
	default:
		return false
	}
}

func isOpaqueRuntimeHandleType(typeName string) bool {
	switch strings.TrimSpace(typeName) {
	case "actor", "task.group", "task.i32":
		return true
	default:
		return false
	}
}

func firstForbiddenEffect(have map[string]struct{}, forbidden []string) string {
	return semanticspolicy.FirstForbiddenEffect(have, forbidden)
}

func typeUsesSecret(typeName string, types map[string]*TypeInfo) bool {
	return typeUsesSecretVisited(strings.TrimSpace(typeName), types, map[string]struct{}{})
}

func functionDeclSignatureUsesSecret(fn *frontend.FuncDecl, types map[string]*TypeInfo) bool {
	if fn == nil {
		return false
	}
	if typeRefUsesSecret(fn.ReturnType, types) {
		return true
	}
	if fn.HasThrows && typeRefUsesSecret(fn.Throws, types) {
		return true
	}
	for _, param := range fn.Params {
		if typeRefUsesSecret(param.Type, types) {
			return true
		}
	}
	return false
}

func typeRefUsesSecret(ref frontend.TypeRef, types map[string]*TypeInfo) bool {
	return typeRefUsesSecretVisited(ref, types, map[string]struct{}{})
}

func typeRefUsesSecretVisited(
	ref frontend.TypeRef,
	types map[string]*TypeInfo,
	visiting map[string]struct{},
) bool {
	switch ref.Kind {
	case frontend.TypeRefFunction:
		for _, param := range ref.Params {
			if typeRefUsesSecretVisited(param, types, visiting) {
				return true
			}
		}
		if ref.Return != nil && typeRefUsesSecretVisited(*ref.Return, types, visiting) {
			return true
		}
		return ref.Throws != nil && typeRefUsesSecretVisited(*ref.Throws, types, visiting)
	case frontend.TypeRefSlice, frontend.TypeRefArray, frontend.TypeRefOptional:
		if ref.Elem != nil {
			return typeRefUsesSecretVisited(*ref.Elem, types, visiting)
		}
	}
	return typeUsesSecretVisited(strings.TrimSpace(ref.Name), types, visiting)
}

func functionSignatureUsesSecretVisited(
	paramTypes []string,
	returnType string,
	throwsType string,
	types map[string]*TypeInfo,
	visiting map[string]struct{},
) bool {
	for _, paramType := range paramTypes {
		if typeUsesSecretVisited(paramType, types, visiting) {
			return true
		}
	}
	return typeUsesSecretVisited(returnType, types, visiting) ||
		typeUsesSecretVisited(throwsType, types, visiting)
}

func functionTypedFieldUsesSecret(
	field FieldInfo,
	types map[string]*TypeInfo,
	visiting map[string]struct{},
) bool {
	return field.FunctionTypeValue &&
		functionSignatureUsesSecretVisited(
			field.FunctionParamTypes,
			field.FunctionReturnType,
			field.FunctionThrowsType,
			types,
			visiting,
		)
}

func enumPayloadFunctionUsesSecret(
	enumCase EnumCaseInfo,
	index int,
	types map[string]*TypeInfo,
	visiting map[string]struct{},
) bool {
	if index < 0 || index >= len(enumCase.PayloadFunctionTypes) ||
		!enumCase.PayloadFunctionTypes[index] {
		return false
	}
	return functionSignatureUsesSecretVisited(
		functionPayloadParamsAt(enumCase, index),
		functionPayloadReturnAt(enumCase, index),
		functionPayloadThrowsAt(enumCase, index),
		types,
		visiting,
	)
}

func functionPayloadParamsAt(enumCase EnumCaseInfo, index int) []string {
	if index >= 0 && index < len(enumCase.PayloadFunctionParams) {
		return enumCase.PayloadFunctionParams[index]
	}
	return nil
}

func functionPayloadReturnAt(enumCase EnumCaseInfo, index int) string {
	if index >= 0 && index < len(enumCase.PayloadFunctionReturns) {
		return enumCase.PayloadFunctionReturns[index]
	}
	return ""
}

func functionPayloadThrowsAt(enumCase EnumCaseInfo, index int) string {
	if index >= 0 && index < len(enumCase.PayloadFunctionThrows) {
		return enumCase.PayloadFunctionThrows[index]
	}
	return ""
}

func exprSecretTainted(
	expr frontend.Expr,
	exprType string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	analysis *functionAnalysisState,
) (bool, error) {
	if expr == nil {
		return false, nil
	}
	if typeUsesSecret(exprType, types) {
		return true, nil
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		if analysis.localSecretTainted(e.Name) {
			return true, nil
		}
		if g, ok := globals[e.Name]; ok {
			return typeUsesSecret(g.TypeName, types), nil
		}
		return false, nil
	case *frontend.ClosureExpr:
		for _, capture := range e.Captures {
			if analysis.localSecretTainted(capture.Name) {
				return true, nil
			}
			if local, ok := locals[capture.Name]; ok && typeUsesSecret(local.TypeName, types) {
				return true, nil
			}
		}
		if len(e.Captures) == 0 && e.Decl != nil {
			for name := range collectClosureCaptures(e.Decl, locals) {
				if analysis.localSecretTainted(name) {
					return true, nil
				}
				if local, ok := locals[name]; ok && typeUsesSecret(local.TypeName, types) {
					return true, nil
				}
			}
		}
		return false, nil
	case *frontend.FieldAccessExpr:
		baseType := ""
		if targetInfo, _, err := ResolveFieldAccessType(e, locals, globals, types); err == nil {
			baseType = targetInfo.TypeName
		}
		return exprSecretTainted(
			e.Base,
			baseType,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			analysis,
		)
	case *frontend.IndexExpr:
		baseTainted, err := exprSecretTainted(
			e.Base,
			"",
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			analysis,
		)
		if err != nil || baseTainted {
			return baseTainted, err
		}
		return exprSecretTainted(e.Index, "", locals, globals, funcs, types, module, imports, analysis)
	case *frontend.CallExpr:
		if e.Name == "core.secret_unseal_i32" || e.Name == "secret_unseal_i32" {
			return true, nil
		}
		if local, ok := locals[e.Name]; ok && local.FunctionTypeValue && analysis.localSecretTainted(
			e.Name,
		) {
			return true, nil
		}
		if enumType, _, ok, err := resolveEnumCaseConstructorCall(
			e,
			types,
			module,
			imports,
		); ok || err != nil {
			if err != nil {
				return false, err
			}
			if typeUsesSecret(enumType, types) {
				return true, nil
			}
			for _, arg := range e.Args {
				tainted, err := exprSecretTainted(
					arg,
					"",
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					analysis,
				)
				if err != nil || tainted {
					return tainted, err
				}
			}
			return false, nil
		}
		if info, ok := types[exprType]; ok && info.Kind == TypeStruct && e.Name == exprType {
			for _, arg := range e.Args {
				tainted, err := exprSecretTainted(
					arg,
					"",
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					analysis,
				)
				if err != nil || tainted {
					return tainted, err
				}
			}
			return false, nil
		}
		resolved, err := resolveCheckedCallName(e.Name, funcs, module, imports, e.At)
		if err != nil {
			for _, arg := range e.Args {
				tainted, taintErr := exprSecretTainted(
					arg,
					"",
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					analysis,
				)
				if taintErr != nil {
					return false, taintErr
				}
				if tainted {
					return false, privacyDiagnosticf(e.At, (("secret-tainted value cannot be passed " +
						"through unknown ") +
						"callback target '%s'"), e.Name)
				}
			}
			return false, nil
		}
		if resolved == "core.secret_unseal_i32" {
			return true, nil
		}
		taintedArgIndexes := make([]int, 0, len(e.Args))
		for idx, arg := range e.Args {
			tainted, err := exprSecretTainted(
				arg,
				"",
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				analysis,
			)
			if err != nil {
				return false, err
			}
			if tainted {
				taintedArgIndexes = append(taintedArgIndexes, idx)
			}
		}
		if len(taintedArgIndexes) > 0 {
			if actorMailboxSendHasSecretPayload(resolved, taintedArgIndexes) {
				return false, privacyDiagnosticf(
					e.At,
					"secret-tainted value cannot be sent through actor mailbox",
				)
			}
			if rawMemoryStoreHasSecretPayload(resolved, taintedArgIndexes) {
				return false, privacyDiagnosticf(
					e.At,
					"secret-tainted value cannot be stored through raw memory",
				)
			}
			if runtimeTimeControlHasSecretPayload(resolved, taintedArgIndexes) {
				return false, privacyDiagnosticf(e.At, "secret-tainted value cannot control runtime time")
			}
			if mmioWriteHasSecretPayload(resolved, taintedArgIndexes) {
				return false, privacyDiagnosticf(e.At, "secret-tainted value cannot be written through MMIO")
			}
			if strings.HasPrefix(resolved, "core.") {
				return true, nil
			}
			sig, ok := funcs[resolved]
			if !ok {
				return false, privacyDiagnosticf(e.At, (("secret-tainted value cannot be passed " +
					"through unknown ") +
					"callback target '%s'"), e.Name)
			}
			if analysis != nil {
				for _, idx := range taintedArgIndexes {
					if idx >= 0 && idx < len(sig.ParamNames) {
						analysis.markFunctionParamSecretTaint(resolved, sig.ParamNames[idx])
					}
				}
			}
			return true, nil
		}
		if analysis != nil && analysis.funcReturnSecretTaint != nil && analysis.funcReturnSecretTaint[resolved] {
			return true, nil
		}
		return false, nil
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			tainted, err := exprSecretTainted(
				field.Value,
				"",
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				analysis,
			)
			if err != nil || tainted {
				return tainted, err
			}
		}
		return false, nil
	case *frontend.UnaryExpr:
		return exprSecretTainted(e.X, "", locals, globals, funcs, types, module, imports, analysis)
	case *frontend.BinaryExpr:
		left, err := exprSecretTainted(
			e.Left,
			"",
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			analysis,
		)
		if err != nil || left {
			return left, err
		}
		return exprSecretTainted(e.Right, "", locals, globals, funcs, types, module, imports, analysis)
	case *frontend.TryExpr:
		return exprSecretTainted(e.X, exprType, locals, globals, funcs, types, module, imports, analysis)
	case *frontend.AwaitExpr:
		return exprSecretTainted(e.X, exprType, locals, globals, funcs, types, module, imports, analysis)
	case *frontend.MatchExpr:
		scrutTainted, err := exprSecretTainted(
			e.Value,
			"",
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			analysis,
		)
		if err != nil || scrutTainted {
			return scrutTainted, err
		}
		for _, c := range e.Cases {
			if c.Guard != nil {
				guardTainted, err := exprSecretTainted(
					c.Guard,
					"",
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					analysis,
				)
				if err != nil || guardTainted {
					return guardTainted, err
				}
			}
			tainted, err := exprSecretTainted(
				c.Value,
				exprType,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				analysis,
			)
			if err != nil || tainted {
				return tainted, err
			}
		}
	case *frontend.CatchExpr:
		callTainted, err := exprSecretTainted(
			e.Call,
			exprType,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			analysis,
		)
		if err != nil || callTainted {
			return callTainted, err
		}
		for _, c := range e.Cases {
			tainted, err := exprSecretTainted(
				c.Value,
				exprType,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				analysis,
			)
			if err != nil || tainted {
				return tainted, err
			}
		}
	}
	return false, nil
}

func actorMailboxSendHasSecretPayload(resolved string, taintedArgIndexes []int) bool {
	for _, idx := range taintedArgIndexes {
		switch resolved {
		case "core.send", "core.send_typed":
			if idx == 1 {
				return true
			}
		case "core.send_msg":
			if idx == 1 || idx == 2 {
				return true
			}
		}
	}
	return false
}

func rawMemoryStoreHasSecretPayload(resolved string, taintedArgIndexes []int) bool {
	switch resolved {
	case "core.store_i32", "core.store_u8", "core.store_ptr":
	default:
		return false
	}
	for _, idx := range taintedArgIndexes {
		if idx == 1 {
			return true
		}
	}
	return false
}

func runtimeTimeControlHasSecretPayload(resolved string, taintedArgIndexes []int) bool {
	switch resolved {
	case "core.sleep_ms", "core.sleep_until":
	default:
		return false
	}
	for _, idx := range taintedArgIndexes {
		if idx == 0 {
			return true
		}
	}
	return false
}

func mmioWriteHasSecretPayload(resolved string, taintedArgIndexes []int) bool {
	if resolved != "core.mmio_write_i32" {
		return false
	}
	for _, idx := range taintedArgIndexes {
		if idx == 1 {
			return true
		}
	}
	return false
}

func bindPatternSecretTaintLocals(
	pattern frontend.Expr,
	fallbackName string,
	tainted bool,
	analysis *functionAnalysisState,
) {
	if !tainted || analysis == nil {
		return
	}
	if fallbackName != "" {
		analysis.setLocalSecretTaint(fallbackName, true)
	}
	switch p := pattern.(type) {
	case *frontend.IdentExpr:
		analysis.setLocalSecretTaint(p.Name, true)
	case *frontend.SomePatternExpr:
		analysis.setLocalSecretTaint(p.Name, true)
	case *frontend.EnumCasePatternExpr:
		for _, binding := range p.Bindings {
			analysis.setLocalSecretTaint(binding, true)
		}
	}
}

func typeUsesSecretVisited(
	typeName string,
	types map[string]*TypeInfo,
	visiting map[string]struct{},
) bool {
	if typeName == "" {
		return false
	}
	if strings.HasPrefix(typeName, "secret.") {
		return true
	}
	if _, seen := visiting[typeName]; seen {
		return false
	}
	visiting[typeName] = struct{}{}
	defer delete(visiting, typeName)

	if info, ok := types[typeName]; ok {
		switch info.Kind {
		case TypeStruct:
			for _, field := range info.Fields {
				if functionTypedFieldUsesSecret(field, types, visiting) ||
					typeUsesSecretVisited(field.TypeName, types, visiting) {
					return true
				}
			}
		case TypeEnum:
			for _, enumCase := range info.EnumCases {
				for index, payloadType := range enumCase.PayloadTypes {
					if enumPayloadFunctionUsesSecret(enumCase, index, types, visiting) ||
						typeUsesSecretVisited(payloadType, types, visiting) {
						return true
					}
				}
			}
		case TypeArray, TypeOptional, TypeSlice:
			return typeUsesSecretVisited(info.ElemType, types, visiting)
		}
	}
	if strings.HasSuffix(typeName, "?") {
		return typeUsesSecretVisited(strings.TrimSuffix(typeName, "?"), types, visiting)
	}
	if strings.HasPrefix(typeName, "[]") {
		return typeUsesSecretVisited(strings.TrimPrefix(typeName, "[]"), types, visiting)
	}
	if _, elem, ok := parseArrayTypeName(typeName); ok {
		return typeUsesSecretVisited(elem, types, visiting)
	}
	return false
}

func findFuncDecl(
	world *module.World,
	name string,
) (*frontend.FuncDecl, string, map[string]string, error) {
	for _, file := range world.Files {
		for _, fn := range file.Funcs {
			if qualifyName(file.Module, fn.Name) == name || fn.Name == name {
				imports, err := collectImportAliases(file)
				if err != nil {
					return nil, "", nil, err
				}
				return fn, file.Module, imports, nil
			}
		}
	}
	return nil, "", nil, nil
}

func compareProtocolRequirement(
	typeName, protoName string,
	req frontend.FuncSigDecl,
	method *frontend.FuncDecl,
	methodModule string,
	methodImports map[string]string,
) error {
	if len(req.TypeParams) != len(method.TypeParams) {
		return fmt.Errorf(
			"%s: method '%s' does not match protocol '%s' requirement '%s': generic parameter count differs",
			frontend.FormatPos(method.Pos),
			method.Name,
			protoName,
			req.Name,
		)
	}
	typeParamMap := make(map[string]string, len(req.TypeParams))
	for i := range req.TypeParams {
		typeParamMap[req.TypeParams[i]] = method.TypeParams[i]
	}
	methodTypeParams := make(map[string]struct{}, len(method.TypeParams))
	for _, name := range method.TypeParams {
		methodTypeParams[name] = struct{}{}
	}
	methodParamTypes := make([]frontend.TypeRef, len(method.Params))
	for i := range method.Params {
		normalized, err := normalizeProtocolComparisonTypeRef(
			method.Params[i].Type,
			methodModule,
			methodImports,
			methodTypeParams,
		)
		if err != nil {
			return fmt.Errorf(
				"%s: method '%s' does not match protocol '%s' requirement '%s': parameter %d type differs: %v",
				frontend.FormatPos(method.Params[i].At),
				method.Name,
				protoName,
				req.Name,
				i+1,
				err,
			)
		}
		methodParamTypes[i] = normalized
	}
	methodReturnType, err := normalizeProtocolComparisonTypeRef(
		method.ReturnType,
		methodModule,
		methodImports,
		methodTypeParams,
	)
	if err != nil {
		return fmt.Errorf(
			"%s: method '%s' does not match protocol '%s' requirement '%s': return type differs: %v",
			frontend.FormatPos(method.ReturnType.At),
			method.Name,
			protoName,
			req.Name,
			err,
		)
	}
	methodThrows := method.Throws
	if method.HasThrows {
		normalized, err := normalizeProtocolComparisonTypeRef(
			method.Throws,
			methodModule,
			methodImports,
			methodTypeParams,
		)
		if err != nil {
			return fmt.Errorf(
				"%s: method '%s' does not match protocol '%s' requirement '%s': throws type differs: %v",
				frontend.FormatPos(method.Throws.At),
				method.Name,
				protoName,
				req.Name,
				err,
			)
		}
		methodThrows = normalized
	}
	if len(req.Params) != len(method.Params) {
		return fmt.Errorf(
			"%s: method '%s' does not match protocol '%s' requirement '%s': parameter count differs",
			frontend.FormatPos(method.Pos),
			method.Name,
			protoName,
			req.Name,
		)
	}
	if len(req.Params) == 0 {
		return fmt.Errorf(
			"%s: method '%s' does not match protocol '%s' requirement '%s': missing self parameter",
			frontend.FormatPos(method.Pos),
			method.Name,
			protoName,
			req.Name,
		)
	}
	if req.Params[0].Name != "self" {
		return fmt.Errorf(
			"%s: protocol '%s' requirement '%s': first parameter must be 'self'",
			frontend.FormatPos(req.Params[0].At),
			protoName,
			req.Name,
		)
	}
	if method.Params[0].Name != "self" {
		return fmt.Errorf(
			"%s: method '%s' does not match protocol '%s' requirement '%s': first parameter must be 'self'",
			frontend.FormatPos(method.Params[0].At),
			method.Name,
			protoName,
			req.Name,
		)
	}
	if genericTypeName(req.Params[0].Type) != typeName {
		return fmt.Errorf(
			"%s: protocol '%s' requirement '%s': self parameter type must be '%s'",
			frontend.FormatPos(req.Params[0].At),
			protoName,
			req.Name,
			typeName,
		)
	}
	if genericTypeName(methodParamTypes[0]) != typeName {
		return fmt.Errorf(
			("%s: method '%s' does not match protocol '%s' requirement " +
				"'%s': self parameter type must be '%s'"),
			frontend.FormatPos(method.Params[0].At),
			method.Name,
			protoName,
			req.Name,
			typeName,
		)
	}
	for i := range req.Params {
		if req.Params[i].Ownership != method.Params[i].Ownership {
			return fmt.Errorf(
				("%s: method '%s' does not match protocol '%s' requirement " +
					"'%s': parameter %d ownership differs: expected '%s', got " +
					"'%s'"),
				frontend.FormatPos(method.Params[i].At),
				method.Name,
				protoName,
				req.Name,
				i+1,
				ownershipDisplay(req.Params[i].Ownership),
				ownershipDisplay(method.Params[i].Ownership),
			)
		}
		if !protocolTypeRefsEquivalent(req.Params[i].Type, methodParamTypes[i], typeParamMap) {
			return fmt.Errorf(
				"%s: method '%s' does not match protocol '%s' requirement '%s': parameter %d type differs",
				frontend.FormatPos(method.Params[i].At),
				method.Name,
				protoName,
				req.Name,
				i+1,
			)
		}
	}
	if !protocolTypeRefsEquivalent(req.ReturnType, methodReturnType, typeParamMap) {
		return fmt.Errorf(
			"%s: method '%s' does not match protocol '%s' requirement '%s': return type differs",
			frontend.FormatPos(method.Pos),
			method.Name,
			protoName,
			req.Name,
		)
	}
	if req.Async != method.Async {
		return fmt.Errorf(
			"%s: method '%s' does not match protocol '%s' requirement '%s': async marker differs",
			frontend.FormatPos(method.Pos),
			method.Name,
			protoName,
			req.Name,
		)
	}
	if req.HasThrows != method.HasThrows ||
		!protocolTypeRefsEquivalent(req.Throws, methodThrows, typeParamMap) {
		return fmt.Errorf(
			"%s: method '%s' does not match protocol '%s' requirement '%s': throws type differs",
			frontend.FormatPos(method.Pos),
			method.Name,
			protoName,
			req.Name,
		)
	}
	reqEffects, err := normalizeEffects(req.Uses, req.At)
	if err != nil {
		return fmt.Errorf(
			"%s: protocol '%s' requirement '%s': %v",
			frontend.FormatPos(req.At),
			protoName,
			req.Name,
			err,
		)
	}
	methodEffects, err := normalizeEffects(method.Uses, method.Pos)
	if err != nil {
		return err
	}
	missing := missingRequiredEffects(reqEffects, methodEffects)
	if len(missing) > 0 {
		return fmt.Errorf(
			("%s: method '%s' for type '%s' does not match protocol '%s' " +
				"requirement '%s': missing required effects %s"),
			frontend.FormatPos(method.Pos),
			method.Name,
			typeName,
			protoName,
			req.Name,
			strings.Join(missing, ", "),
		)
	}
	return nil
}

func normalizeProtocolComparisonTypeRef(
	ref frontend.TypeRef,
	module string,
	imports map[string]string,
	typeParams map[string]struct{},
) (frontend.TypeRef, error) {
	normalized := ref
	name, _, err := resolveProtocolRequirementTypeRef(&normalized, module, imports, typeParams)
	if err != nil {
		return normalized, err
	}
	normalized.Name = name
	return normalized, nil
}

func protocolTypeRefsEquivalent(
	req frontend.TypeRef,
	method frontend.TypeRef,
	typeParamMap map[string]string,
) bool {
	if req.Kind != method.Kind {
		return false
	}
	switch req.Kind {
	case frontend.TypeRefNamed:
		reqName := genericTypeName(req)
		methodName := genericTypeName(method)
		if mapped, ok := typeParamMap[reqName]; ok {
			return mapped == methodName
		}
		if len(req.TypeArgs) != len(method.TypeArgs) {
			return false
		}
		if req.Name != method.Name {
			return false
		}
		for i := range req.TypeArgs {
			if !protocolTypeRefsEquivalent(req.TypeArgs[i], method.TypeArgs[i], typeParamMap) {
				return false
			}
		}
		return true
	case frontend.TypeRefSlice, frontend.TypeRefOptional:
		if req.Elem == nil || method.Elem == nil {
			return req.Elem == nil && method.Elem == nil
		}
		return protocolTypeRefsEquivalent(*req.Elem, *method.Elem, typeParamMap)
	case frontend.TypeRefArray:
		if req.Len != method.Len {
			return false
		}
		if req.Elem == nil || method.Elem == nil {
			return req.Elem == nil && method.Elem == nil
		}
		return protocolTypeRefsEquivalent(*req.Elem, *method.Elem, typeParamMap)
	case frontend.TypeRefFunction:
		if len(req.Params) != len(method.Params) {
			return false
		}
		for i := range req.Params {
			if ownershipAt(req.ParamOwnership, i) != ownershipAt(method.ParamOwnership, i) {
				return false
			}
		}
		for i := range req.Params {
			if !protocolTypeRefsEquivalent(req.Params[i], method.Params[i], typeParamMap) {
				return false
			}
		}
		if req.Return == nil || method.Return == nil {
			return req.Return == nil && method.Return == nil
		}
		if !protocolTypeRefsEquivalent(*req.Return, *method.Return, typeParamMap) {
			return false
		}
		reqEffects, err := normalizeEffects(req.Uses, req.At)
		if err != nil {
			return false
		}
		methodEffects, err := normalizeEffects(method.Uses, method.At)
		if err != nil {
			return false
		}
		return len(missingRequiredEffects(reqEffects, methodEffects)) == 0 &&
			len(missingRequiredEffects(methodEffects, reqEffects)) == 0
	default:
		return genericTypeName(req) == genericTypeName(method)
	}
}

func missingRequiredEffects(required []string, declared []string) []string {
	if len(required) == 0 {
		return nil
	}
	have := make(map[string]struct{}, len(declared))
	for _, effect := range declared {
		have[effect] = struct{}{}
	}
	var missing []string
	for _, effect := range required {
		if _, ok := have[effect]; ok {
			continue
		}
		missing = append(missing, effect)
	}
	return missing
}

func resolveProtocolRequirementTypeRef(
	ref *frontend.TypeRef,
	module string,
	imports map[string]string,
	typeParams map[string]struct{},
) (string, bool, error) {
	if ref == nil {
		return "", false, fmt.Errorf("missing type")
	}
	switch ref.Kind {
	case frontend.TypeRefNamed:
		if ref.Name == "" {
			return "", false, fmt.Errorf("missing type name")
		}
		if _, ok := typeParams[ref.Name]; ok {
			if len(ref.TypeArgs) > 0 {
				return "", false, fmt.Errorf(
					"generic type parameter '%s' cannot have type arguments",
					ref.Name,
				)
			}
			return ref.Name, true, nil
		}
		for i := range ref.TypeArgs {
			argName, _, err := resolveProtocolRequirementTypeRef(
				&ref.TypeArgs[i],
				module,
				imports,
				typeParams,
			)
			if err != nil {
				return "", false, err
			}
			ref.TypeArgs[i].Name = argName
		}
		resolved, err := resolveTypeName(ref, module, imports)
		if err != nil {
			return "", false, err
		}
		ref.Name = resolved
		return resolved, false, nil
	case frontend.TypeRefSlice, frontend.TypeRefArray, frontend.TypeRefOptional:
		if ref.Elem == nil {
			return "", false, fmt.Errorf("missing element type")
		}
		elemName, elemGeneric, err := resolveProtocolRequirementTypeRef(
			ref.Elem,
			module,
			imports,
			typeParams,
		)
		if err != nil {
			return "", false, err
		}
		ref.Elem.Name = elemName
		if elemGeneric {
			return genericTypeName(*ref), true, nil
		}
		resolved, err := resolveTypeName(ref, module, imports)
		if err != nil {
			return "", false, err
		}
		return resolved, false, nil
	case frontend.TypeRefFunction:
		anyGeneric := false
		for i := range ref.Params {
			paramName, paramGeneric, err := resolveProtocolRequirementTypeRef(
				&ref.Params[i],
				module,
				imports,
				typeParams,
			)
			if err != nil {
				return "", false, err
			}
			ref.Params[i].Name = paramName
			anyGeneric = anyGeneric || paramGeneric
		}
		if ref.Return == nil {
			return "", false, fmt.Errorf("missing function return type")
		}
		retName, retGeneric, err := resolveProtocolRequirementTypeRef(
			ref.Return,
			module,
			imports,
			typeParams,
		)
		if err != nil {
			return "", false, err
		}
		ref.Return.Name = retName
		if _, err := normalizeEffects(ref.Uses, ref.At); err != nil {
			return "", false, err
		}
		anyGeneric = anyGeneric || retGeneric
		if anyGeneric {
			return genericTypeName(*ref), true, nil
		}
		return "ptr", false, nil
	default:
		return "", false, fmt.Errorf("unsupported type reference in protocol requirement")
	}
}

func validateGenericTypeRef(ref frontend.TypeRef, params map[string]struct{}) error {
	switch ref.Kind {
	case frontend.TypeRefNamed:
		if ref.Name == "" {
			return fmt.Errorf("missing type name")
		}
		if _, ok := params[ref.Name]; ok {
			return nil
		}
		if _, ok := canonicalBuiltinType(ref.Name); ok {
			return nil
		}
		if strings.Contains(ref.Name, ".") {
			return nil
		}
		return nil
	case frontend.TypeRefSlice, frontend.TypeRefArray, frontend.TypeRefOptional:
		if ref.Elem == nil {
			return fmt.Errorf("%s: missing element type", frontend.FormatPos(ref.At))
		}
		return validateGenericTypeRef(*ref.Elem, params)
	case frontend.TypeRefFunction:
		for _, param := range ref.Params {
			if err := validateGenericTypeRef(param, params); err != nil {
				return err
			}
		}
		if ref.Return == nil {
			return fmt.Errorf("%s: missing function return type", frontend.FormatPos(ref.At))
		}
		if _, err := normalizeEffects(ref.Uses, ref.At); err != nil {
			return err
		}
		return validateGenericTypeRef(*ref.Return, params)
	default:
		return fmt.Errorf(
			"%s: unsupported generic type reference kind %d",
			frontend.FormatPos(ref.At),
			ref.Kind,
		)
	}
}

func genericParamTypeNames(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		out = append(out, formatGenericTypeRef(param.Type))
	}
	return out
}

func genericParamNames(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		out = append(out, param.Name)
	}
	return out
}

func genericParamOwnership(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		out = append(out, param.Ownership)
	}
	return out
}

func genericParamFunctionKinds(params []frontend.ParamDecl) []bool {
	out := make([]bool, 0, len(params))
	for _, param := range params {
		out = append(out, param.Type.Kind == frontend.TypeRefFunction)
	}
	return out
}

func genericParamFunctionParamTypes(params []frontend.ParamDecl) [][]string {
	out := make([][]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, nil)
			continue
		}
		row := make([]string, 0, len(param.Type.Params))
		for _, p := range param.Type.Params {
			row = append(row, formatGenericTypeRef(p))
		}
		out = append(out, row)
	}
	return out
}

func genericParamFunctionOwnership(params []frontend.ParamDecl) [][]string {
	out := make([][]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, nil)
			continue
		}
		out = append(out, functionTypeRefParamOwnership(param.Type))
	}
	return out
}

func genericParamFunctionReturnTypes(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction || param.Type.Return == nil {
			out = append(out, "")
			continue
		}
		out = append(out, formatGenericTypeRef(*param.Type.Return))
	}
	return out
}

func genericParamFunctionThrowsTypes(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction || param.Type.Throws == nil {
			out = append(out, "")
			continue
		}
		out = append(out, formatGenericTypeRef(*param.Type.Throws))
	}
	return out
}

func genericParamFunctionEffectTypes(params []frontend.ParamDecl) [][]string {
	out := make([][]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, nil)
			continue
		}
		effects, err := normalizeEffects(param.Type.Uses, param.Type.At)
		if err != nil {
			out = append(out, nil)
			continue
		}
		out = append(out, effects)
	}
	return out
}

func paramFunctionKinds(params []frontend.ParamDecl) []bool {
	out := make([]bool, 0, len(params))
	for _, param := range params {
		out = append(out, param.Type.Kind == frontend.TypeRefFunction)
	}
	return out
}

func genericParamFunctionReturnOwnership(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, "")
			continue
		}
		out = append(out, functionTypeRefReturnOwnership(param.Type))
	}
	return out
}

func paramFunctionParamTypes(params []frontend.ParamDecl) [][]string {
	out := make([][]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, nil)
			continue
		}
		row := make([]string, 0, len(param.Type.Params))
		for _, p := range param.Type.Params {
			row = append(row, p.Name)
		}
		out = append(out, row)
	}
	return out
}

func paramFunctionOwnership(params []frontend.ParamDecl) [][]string {
	out := make([][]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, nil)
			continue
		}
		out = append(out, functionTypeRefParamOwnership(param.Type))
	}
	return out
}

func paramFunctionReturnTypes(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction || param.Type.Return == nil {
			out = append(out, "")
			continue
		}
		out = append(out, param.Type.Return.Name)
	}
	return out
}

func paramFunctionReturnOwnership(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, "")
			continue
		}
		out = append(out, functionTypeRefReturnOwnership(param.Type))
	}
	return out
}

func paramFunctionThrowsTypes(
	params []frontend.ParamDecl,
	module string,
	imports map[string]string,
) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		throwsType, err := functionTypeRefThrowsType(param.Type, module, imports)
		if err != nil {
			out = append(out, "")
			continue
		}
		out = append(out, throwsType)
	}
	return out
}

func paramFunctionEffectTypes(params []frontend.ParamDecl) [][]string {
	out := make([][]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, nil)
			continue
		}
		effects, err := normalizeEffects(param.Type.Uses, param.Type.At)
		if err != nil {
			out = append(out, nil)
			continue
		}
		out = append(out, effects)
	}
	return out
}

func throwingReturnSlots(successSlots, errorSlots int) int {
	if successSlots == 1 && errorSlots == 1 {
		// Preserve the compact v0.5 layout for existing single-slot typed errors.
		return 2
	}
	return successSlots + errorSlots + 1
}

func throwingScratchSlots(errorSlots int) int {
	if errorSlots <= 0 {
		return 0
	}
	return errorSlots
}

func formatGenericTypeRef(ref frontend.TypeRef) string {
	switch ref.Kind {
	case frontend.TypeRefSlice:
		if ref.Elem == nil {
			return "[]?"
		}
		return "[]" + formatGenericTypeRef(*ref.Elem)
	case frontend.TypeRefArray:
		if ref.Elem == nil {
			return fmt.Sprintf("[%d]?", ref.Len)
		}
		return fmt.Sprintf("[%d]%s", ref.Len, formatGenericTypeRef(*ref.Elem))
	case frontend.TypeRefOptional:
		if ref.Elem == nil {
			return "?"
		}
		return formatGenericTypeRef(*ref.Elem) + "?"
	case frontend.TypeRefFunction:
		parts := make([]string, 0, len(ref.Params))
		for i, param := range ref.Params {
			formatted := formatGenericTypeRef(param)
			if i < len(ref.ParamOwnership) && ref.ParamOwnership[i] != "" {
				formatted = ref.ParamOwnership[i] + " " + formatted
			}
			parts = append(parts, formatted)
		}
		ret := "?"
		if ref.Return != nil {
			ret = formatGenericTypeRef(*ref.Return)
		}
		out := "fn(" + strings.Join(parts, ", ") + ") -> " + ret
		if ref.Throws != nil {
			out += " throws " + formatGenericTypeRef(*ref.Throws)
		}
		if len(ref.Uses) > 0 {
			out += " uses " + strings.Join(ref.Uses, ", ")
		}
		return out
	default:
		if len(ref.TypeArgs) > 0 {
			args := make([]string, 0, len(ref.TypeArgs))
			for _, arg := range ref.TypeArgs {
				args = append(args, formatGenericTypeRef(arg))
			}
			return ref.Name + "<" + strings.Join(args, ", ") + ">"
		}
		return ref.Name
	}
}

// ---- checker_analysis_flow.go ----

type functionAnalysisState struct {
	touchesMutableGlobals               bool
	returnFunctionSymbol                string
	returnFunctionParamName             string
	returnFunctionCaptures              []frontend.ClosureCapture
	returnFunctionTouchesMutableGlobals bool
	returnFunctionEscapeKind            CallableEscapeKind
	returnFunctionHandleValue           bool
	returnFunctionFields                map[string]FunctionFieldInfo
	returnEnumPayloadFunctions          map[string]FunctionFieldInfo
	returnEnumPayloadFields             map[string]FunctionFieldInfo
	borrowedReturnOwner                 string
	secretTaint                         map[string]bool
	surfaceFramePixels                  map[string]string
	currentFuncName                     string
	funcReturnSecretTaint               map[string]bool
	funcParamSecretTaint                map[string]map[string]bool
	discoveredParamTaint                bool
	allowSecretReturn                   bool
	rejectSecretReturn                  bool
	exportedFuncName                    string
	returnSecretTaint                   bool
	secretControlDepth                  int
	surfacePresentedFrames              map[string]frontend.Position
	surfaceFrameOwners                  map[string]string
	surfaceHandleOwners                 map[string]string
}

func newFunctionAnalysisState(
	fn *frontend.FuncDecl,
	policy functionClausePolicy,
	fullName string,
	returnSecretTaint map[string]bool,
	paramSecretTaint map[string]map[string]bool,
	types map[string]*TypeInfo,
) *functionAnalysisState {
	analysis := &functionAnalysisState{
		secretTaint:           make(map[string]bool),
		currentFuncName:       fullName,
		funcReturnSecretTaint: returnSecretTaint,
		funcParamSecretTaint:  paramSecretTaint,
		allowSecretReturn:     policy.hasPrivacy,
		rejectSecretReturn:    fn.ExportName != "",
		exportedFuncName:      fn.Name,
	}
	for _, param := range fn.Params {
		if typeUsesSecret(param.Type.Name, types) {
			analysis.secretTaint[param.Name] = true
		}
	}
	if inbound := paramSecretTaint[fullName]; len(inbound) > 0 {
		for name, tainted := range inbound {
			if tainted {
				analysis.secretTaint[name] = true
			}
		}
	}
	return analysis
}

func recordReturnFunctionCaptures(
	analysis *functionAnalysisState,
	captures []frontend.ClosureCapture,
) {
	if analysis == nil || len(captures) == 0 || len(analysis.returnFunctionCaptures) > 0 {
		return
	}
	analysis.returnFunctionCaptures = append([]frontend.ClosureCapture(nil), captures...)
}

func recordReturnFunctionTargetMutableGlobalUse(analysis *functionAnalysisState, sig FuncSig) {
	if analysis != nil && sig.TouchesMutableGlobals {
		analysis.returnFunctionTouchesMutableGlobals = true
	}
}

func applyInterfaceFunctionReturnMetadata(
	sig *FuncSig,
	fn *frontend.FuncDecl,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (bool, error) {
	if sig == nil || fn == nil {
		return false, nil
	}
	changed := false
	for _, stmt := range fn.Body {
		if throwStmt, ok := stmt.(*frontend.ThrowStmt); ok {
			if typeContainsResourceHandle(sig.ThrowsType, types) {
				state := newRegionState(nil)
				initParamRegions(fn.Params, state, types)
				summary, unknown, err := returnResourceSummaryForExpr(
					throwStmt.Value,
					sig.ThrowsType,
					funcs,
					types,
					module,
					imports,
					state,
				)
				if err != nil {
					return false, err
				}
				if !unknown && !returnResourceSummariesEqual(sig.ThrowResourceSummary, summary) {
					sig.ThrowResourceSummary = cloneReturnResourceSummary(summary)
					changed = true
				}
			}
			continue
		}
		if sig.ReturnFunctionType {
			nestedChanged, err := applyInterfaceFunctionReturnParamMetadataFromNestedStmt(
				sig,
				stmt,
				types,
				module,
				imports,
			)
			if err != nil {
				return false, err
			}
			if nestedChanged {
				changed = true
			}
		}
		ret, ok := stmt.(*frontend.ReturnStmt)
		if !ok {
			continue
		}
		if typeContainsResourceHandle(sig.ThrowsType, types) {
			if tryExpr, ok := ret.Value.(*frontend.TryExpr); ok {
				if call, ok := tryExpr.X.(*frontend.CallExpr); ok {
					resolved := call.Name
					calleeSig, ok := funcs[resolved]
					if !ok {
						var err error
						resolved, err = resolveKnownCallName(
							call.Name,
							funcs,
							module,
							imports,
							call.At,
						)
						if err != nil {
							return false, err
						}
						calleeSig, ok = funcs[resolved]
					}
					if ok && calleeSig.ThrowsType != "" {
						state := newRegionState(nil)
						initParamRegions(fn.Params, state, types)
						if err := recordTryCallThrowResourceSummary(
							call,
							calleeSig,
							funcs,
							types,
							module,
							imports,
							state,
						); err != nil {
							return false, err
						}
						if len(state.throwResourceSummary) > 0 &&
							!returnResourceSummariesEqual(
								sig.ThrowResourceSummary,
								state.throwResourceSummary,
							) {
							sig.ThrowResourceSummary = cloneReturnResourceSummary(
								state.throwResourceSummary,
							)
							changed = true
						}
					}
				}
			}
		}
		if typeContainsResourceHandle(sig.ReturnType, types) {
			state := newRegionState(nil)
			initParamRegions(fn.Params, state, types)
			summary, unknown, err := returnResourceSummaryForExpr(
				ret.Value,
				sig.ReturnType,
				funcs,
				types,
				module,
				imports,
				state,
			)
			if err != nil {
				return false, err
			}
			newReturnResourceParam := regionNone
			newReturnResourcePath := ""
			newReturnResourceSummary := ReturnResourceSummary(nil)
			if unknown {
				newReturnResourceParam = regionUnknown
			} else if len(summary) > 0 {
				newReturnResourceSummary = cloneReturnResourceSummary(summary)
				if provenances := summary[""]; len(provenances) == 1 {
					newReturnResourceParam = provenances[0].ParamIndex
					newReturnResourcePath = provenances[0].ParamPath
				}
			}
			if sig.ReturnResourceParam != newReturnResourceParam ||
				sig.ReturnResourcePath != newReturnResourcePath ||
				!returnResourceSummariesEqual(sig.ReturnResourceSummary, newReturnResourceSummary) {
				sig.ReturnResourceParam = newReturnResourceParam
				sig.ReturnResourcePath = newReturnResourcePath
				sig.ReturnResourceSummary = newReturnResourceSummary
				changed = true
			}
		}
		if typeMayContainRegion(sig.ReturnType, types) &&
			!typeContainsResourceHandle(sig.ReturnType, types) {
			summary, err := returnRegionSummaryForInterfaceExpr(
				ret.Value,
				sig.ReturnType,
				sig.Effects,
				fn,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return false, err
			}
			newReturnRegionParam := regionNone
			if len(summary) > 0 {
				commonParam := regionNone
				for _, paramIndex := range summary {
					if commonParam == regionNone {
						commonParam = paramIndex
						continue
					}
					if commonParam != paramIndex {
						commonParam = regionUnknown
						break
					}
				}
				if commonParam >= 0 {
					newReturnRegionParam = commonParam
				}
			}
			if sig.ReturnRegionParam != newReturnRegionParam ||
				!returnRegionSummariesEqual(sig.ReturnRegionSummary, summary) {
				sig.ReturnRegionParam = newReturnRegionParam
				sig.ReturnRegionSummary = cloneReturnRegionSummary(summary)
				changed = true
			}
		}
		if sig.ReturnFunctionType {
			if closure, ok := ret.Value.(*frontend.ClosureExpr); ok {
				locals, err := interfaceFunctionReturnStubLocals(
					fn.Body,
					ret,
					types,
					module,
					imports,
				)
				if err != nil {
					return false, err
				}
				if err := configureClosureCaptures(
					closure,
					locals,
					funcs,
					types,
					module,
					true,
					"interface function-typed return",
				); err != nil {
					return false, err
				}
				if len(closure.Captures) > 0 {
					target := closureFunctionValueName(closure, funcs, module)
					if sig.ReturnFunctionSymbol != target {
						sig.ReturnFunctionSymbol = target
						changed = true
					}
					if !closureCapturesEqual(sig.ReturnFunctionCaptures, closure.Captures) {
						sig.ReturnFunctionCaptures = append(
							[]frontend.ClosureCapture(nil),
							closure.Captures...)
						changed = true
					}
					captureSlots, err := functionCaptureSlotCount(closure.Captures, types)
					if err != nil {
						return false, err
					}
					escapeKind := CallableEscapeKind("")
					handleValue := false
					if captureSlots > FnPtrEnvSlotCount {
						escapeKind, handleValue, err = classifyCallableEscape(
							callableBoundaryReturn,
							closure.Captures,
							types,
						)
						if err != nil {
							return false, err
						}
					}
					if sig.ReturnFunctionEscapeKind != escapeKind {
						sig.ReturnFunctionEscapeKind = escapeKind
						changed = true
					}
					if sig.ReturnFunctionHandleValue != handleValue {
						sig.ReturnFunctionHandleValue = handleValue
						changed = true
					}
					desiredReturnSlots := sig.ReturnSlots
					if handleValue {
						desiredReturnSlots = CallableHandleSlotCount
					}
					if sig.ReturnSlots != desiredReturnSlots {
						sig.ReturnSlots = desiredReturnSlots
						changed = true
					}
				}
				continue
			}
			if id, ok := ret.Value.(*frontend.IdentExpr); ok {
				for i, name := range sig.ParamNames {
					if name != id.Name || i >= len(sig.ParamFunctionTypes) ||
						!sig.ParamFunctionTypes[i] {
						continue
					}
					if sig.ReturnFunctionParamName != name {
						sig.ReturnFunctionParamName = name
						changed = true
					}
				}
				continue
			}
			fieldPath := callbackArgumentName(ret.Value)
			if fieldPath != "" {
				for _, name := range sig.ParamNames {
					if fieldPath == name || !ownershipPathPrefix(name, fieldPath) {
						continue
					}
					if sig.ReturnFunctionParamName != fieldPath {
						sig.ReturnFunctionParamName = fieldPath
						changed = true
					}
				}
			}
		}
		returnFields, err := functionFieldsFromReturnedStructExpr(
			sig.ReturnType,
			ret.Value,
			nil,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return false, err
		}
		if !functionFieldMapsEqual(sig.ReturnFunctionFields, returnFields) {
			sig.ReturnFunctionFields = cloneFunctionFieldMap(returnFields)
			changed = true
		}
		returnPayloadFields, err := enumPayloadFieldsFromReturnedStructExpr(
			sig.ReturnType,
			ret.Value,
			nil,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return false, err
		}
		if !functionFieldMapsEqual(sig.ReturnEnumPayloadFields, returnPayloadFields) {
			sig.ReturnEnumPayloadFields = cloneFunctionFieldMap(returnPayloadFields)
			changed = true
		}
		returnPayloads, err := enumPayloadFunctionsFromReturnedEnumExpr(
			sig.ReturnType,
			ret.Value,
			nil,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return false, err
		}
		if !functionFieldMapsEqual(sig.ReturnEnumPayloadFunctions, returnPayloads) {
			sig.ReturnEnumPayloadFunctions = cloneFunctionFieldMap(returnPayloads)
			changed = true
		}
	}
	return changed, nil
}

func returnRegionSummaryForInterfaceExpr(
	expr frontend.Expr,
	returnType string,
	effectNames []string,
	fn *frontend.FuncDecl,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (ReturnRegionSummary, error) {
	if expr == nil || fn == nil || !typeMayContainRegion(returnType, types) {
		return nil, nil
	}
	state := newRegionState(nil)
	initParamRegions(fn.Params, state, types)
	locals, err := interfaceParamRegionLocals(fn.Params, types, module, imports)
	if err != nil {
		return nil, err
	}
	effects := newEffectContext(
		module+"."+fn.Name,
		effectNames,
		fn.Uses,
		strings.HasPrefix(module, "__"),
	)
	tname, regionID, err := checkExprWithEffects(
		expr,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
		state,
		effects,
		nil,
	)
	if err != nil {
		return nil, err
	}
	if !typesCompatibleWithNullPtr(returnType, tname, expr) {
		return nil, fmt.Errorf(
			"%s: type mismatch: expected '%s', got '%s'",
			frontend.FormatPos(expr.Pos()),
			returnType,
			tname,
		)
	}
	tree := regionTreeForExpr(returnType, expr, regionID, types, state)
	if len(tree) == 0 {
		return nil, nil
	}
	if err := state.recordReturnRegionSummary(tree, expr.Pos()); err != nil {
		return nil, err
	}
	return cloneReturnRegionSummary(state.returnRegionSummary), nil
}

func interfaceParamRegionLocals(
	params []frontend.ParamDecl,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (map[string]LocalInfo, error) {
	locals := make(map[string]LocalInfo, len(params))
	slotIndex := 0
	for _, param := range params {
		paramTypeName, err := resolveTypeName(&param.Type, module, imports)
		if err != nil {
			return nil, err
		}
		info, err := ensureTypeInfo(paramTypeName, types)
		if err != nil {
			return nil, err
		}
		locals[param.Name] = LocalInfo{
			Base:      slotIndex,
			SlotCount: info.SlotCount,
			TypeName:  paramTypeName,
			Mutable:   param.Ownership == "inout",
		}
		slotIndex += info.SlotCount
	}
	return locals, nil
}

func applyInterfaceFunctionReturnParamMetadataFromNestedStmt(
	sig *FuncSig,
	stmt frontend.Stmt,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (bool, error) {
	match, ok := stmt.(*frontend.MatchStmt)
	if !ok {
		return false, nil
	}
	payloadBindings, err := interfaceMatchFunctionPayloadBindings(
		*sig,
		match,
		types,
		module,
		imports,
	)
	if err != nil {
		return false, err
	}
	changed := false
	for _, c := range match.Cases {
		for _, caseStmt := range c.Body {
			ret, ok := caseStmt.(*frontend.ReturnStmt)
			if !ok {
				continue
			}
			paramRef := interfaceFunctionReturnParamRef(*sig, ret.Value, payloadBindings)
			if paramRef == "" || sig.ReturnFunctionParamName == paramRef {
				continue
			}
			sig.ReturnFunctionParamName = paramRef
			changed = true
		}
	}
	return changed, nil
}

func interfaceFunctionReturnParamRef(
	sig FuncSig,
	expr frontend.Expr,
	payloadBindings map[string]string,
) string {
	if id, ok := expr.(*frontend.IdentExpr); ok {
		if payloadRef := payloadBindings[id.Name]; payloadRef != "" {
			return payloadRef
		}
		for i, name := range sig.ParamNames {
			if name == id.Name && i < len(sig.ParamFunctionTypes) && sig.ParamFunctionTypes[i] {
				return name
			}
		}
	}
	fieldPath := callbackArgumentName(expr)
	if fieldPath == "" {
		return ""
	}
	for _, name := range sig.ParamNames {
		if fieldPath != name && ownershipPathPrefix(name, fieldPath) {
			return fieldPath
		}
	}
	return ""
}

func interfaceMatchFunctionPayloadBindings(
	sig FuncSig,
	match *frontend.MatchStmt,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (map[string]string, error) {
	id, ok := match.Value.(*frontend.IdentExpr)
	if !ok {
		return nil, nil
	}
	paramIndex := -1
	for i, name := range sig.ParamNames {
		if name == id.Name {
			paramIndex = i
			break
		}
	}
	if paramIndex < 0 || paramIndex >= len(sig.ParamTypes) {
		return nil, nil
	}
	scrutType := sig.ParamTypes[paramIndex]
	bindings := map[string]string{}
	for _, c := range match.Cases {
		enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr)
		if !ok {
			continue
		}
		caseType, caseInfo, found, err := resolveEnumCasePattern(enumPat, types, module, imports)
		if err != nil {
			return nil, err
		}
		if !found || caseType != scrutType {
			continue
		}
		for i, binding := range enumPat.Bindings {
			if i >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[i] {
				continue
			}
			bindings[binding] = id.Name + "#" + enumPayloadFunctionKey(caseInfo.Ordinal, i)
		}
	}
	return bindings, nil
}

func interfaceFunctionReturnStubLocals(
	body []frontend.Stmt,
	stop *frontend.ReturnStmt,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (map[string]LocalInfo, error) {
	locals := map[string]LocalInfo{}
	slot := 0
	for _, stmt := range body {
		if stmt == stop {
			break
		}
		let, ok := stmt.(*frontend.LetStmt)
		if !ok {
			continue
		}
		typeName, err := resolveTypeName(&let.Type, module, imports)
		if err != nil {
			return nil, err
		}
		info, ok := types[typeName]
		if !ok {
			return nil, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(let.At), typeName)
		}
		locals[let.Name] = LocalInfo{
			Base:      slot,
			SlotCount: info.SlotCount,
			TypeName:  typeName,
			Mutable:   let.Mutable,
			Const:     let.Const,
		}
		slot += info.SlotCount
	}
	return locals, nil
}

func (analysis *functionAnalysisState) localSecretTainted(name string) bool {
	return analysis != nil && analysis.secretTaint != nil && analysis.secretTaint[name]
}

func (analysis *functionAnalysisState) setLocalSecretTaint(name string, tainted bool) {
	if analysis == nil || name == "" {
		return
	}
	if analysis.secretTaint == nil {
		analysis.secretTaint = make(map[string]bool)
	}
	if tainted {
		analysis.secretTaint[name] = true
		return
	}
	delete(analysis.secretTaint, name)
}

func (analysis *functionAnalysisState) localSurfaceFramePixels(name string) bool {
	_, ok := analysis.localSurfaceFramePixelsSource(name)
	return ok
}

func (analysis *functionAnalysisState) localSurfaceFramePixelsSource(name string) (string, bool) {
	if analysis == nil || analysis.surfaceFramePixels == nil {
		return "", false
	}
	source, ok := analysis.surfaceFramePixels[name]
	return source, ok
}

func (analysis *functionAnalysisState) setLocalSurfaceFramePixelsSource(
	name string,
	frameName string,
) {
	if analysis == nil || name == "" {
		return
	}
	if analysis.surfaceFramePixels == nil {
		analysis.surfaceFramePixels = make(map[string]string)
	}
	if frameName != "" {
		analysis.surfaceFramePixels[name] = frameName
		return
	}
	delete(analysis.surfaceFramePixels, name)
}

func bindLocalSurfaceFramePixelsSource(
	locals map[string]LocalInfo,
	analysis *functionAnalysisState,
	name string,
	frameName string,
) {
	if analysis != nil {
		analysis.setLocalSurfaceFramePixelsSource(name, frameName)
	}
	if info, ok := locals[name]; ok {
		info.SurfaceFramePixelsSource = frameName
		locals[name] = info
	}
}

func (analysis *functionAnalysisState) markSurfaceFramePresented(
	name string,
	pos frontend.Position,
) {
	if analysis == nil || name == "" {
		return
	}
	if analysis.surfacePresentedFrames == nil {
		analysis.surfacePresentedFrames = make(map[string]frontend.Position)
	}
	analysis.surfacePresentedFrames[name] = pos
}

func (analysis *functionAnalysisState) clearSurfaceFramePresented(name string) {
	if analysis == nil || name == "" {
		return
	}
	delete(analysis.surfacePresentedFrames, name)
}

func (analysis *functionAnalysisState) localSurfaceFrameOwner(name string) (string, bool) {
	if analysis == nil || analysis.surfaceFrameOwners == nil {
		return "", false
	}
	owner, ok := analysis.surfaceFrameOwners[name]
	return owner, ok
}

func (analysis *functionAnalysisState) setLocalSurfaceFrameOwner(name string, owner string) {
	if analysis == nil || name == "" {
		return
	}
	if analysis.surfaceFrameOwners == nil {
		analysis.surfaceFrameOwners = make(map[string]string)
	}
	if owner != "" {
		analysis.surfaceFrameOwners[name] = owner
		return
	}
	delete(analysis.surfaceFrameOwners, name)
}

func (analysis *functionAnalysisState) localSurfaceHandleOwner(name string) (string, bool) {
	if analysis == nil || analysis.surfaceHandleOwners == nil {
		return "", false
	}
	owner, ok := analysis.surfaceHandleOwners[name]
	return owner, ok
}

func (analysis *functionAnalysisState) setLocalSurfaceHandleOwner(name string, owner string) {
	if analysis == nil || name == "" {
		return
	}
	if analysis.surfaceHandleOwners == nil {
		analysis.surfaceHandleOwners = make(map[string]string)
	}
	if owner != "" {
		analysis.surfaceHandleOwners[name] = owner
		return
	}
	delete(analysis.surfaceHandleOwners, name)
}

func (analysis *functionAnalysisState) checkSurfaceFramePixelsUsable(
	name string,
	pos frontend.Position,
) error {
	frameName, ok := analysis.localSurfaceFramePixelsSource(name)
	if !ok || frameName == "" || analysis.surfacePresentedFrames == nil {
		return nil
	}
	if _, presented := analysis.surfacePresentedFrames[frameName]; !presented {
		return nil
	}
	return lifetimeDiagnosticf(
		pos,
		("surface frame pixels alias '%s' cannot be used after frame " +
			"'%s' was presented; keep Frame.pixels local to the active " +
			"Surface frame"),
		name,
		frameName,
	)
}

func (analysis *functionAnalysisState) underSecretControl() bool {
	return analysis != nil && analysis.secretControlDepth > 0
}

func (analysis *functionAnalysisState) withSecretControl(tainted bool, fn func() error) error {
	if analysis == nil || !tainted {
		return fn()
	}
	analysis.secretControlDepth++
	defer func() {
		analysis.secretControlDepth--
	}()
	return fn()
}

func (analysis *functionAnalysisState) markFunctionParamSecretTaint(funcName, paramName string) {
	if analysis == nil || funcName == "" || paramName == "" ||
		analysis.funcParamSecretTaint == nil {
		return
	}
	params := analysis.funcParamSecretTaint[funcName]
	if params == nil {
		params = make(map[string]bool)
		analysis.funcParamSecretTaint[funcName] = params
	}
	if !params[paramName] {
		params[paramName] = true
		analysis.discoveredParamTaint = true
	}
}

func (analysis *functionAnalysisState) copySecretTaint() map[string]bool {
	if analysis == nil || len(analysis.secretTaint) == 0 {
		return make(map[string]bool)
	}
	out := make(map[string]bool, len(analysis.secretTaint))
	for name, tainted := range analysis.secretTaint {
		if tainted {
			out[name] = true
		}
	}
	return out
}

func (analysis *functionAnalysisState) restoreSecretTaint(snapshot map[string]bool) {
	if analysis == nil {
		return
	}
	analysis.secretTaint = copySecretTaintMap(snapshot)
}

func copySecretTaintMap(src map[string]bool) map[string]bool {
	if len(src) == 0 {
		return make(map[string]bool)
	}
	dst := make(map[string]bool, len(src))
	for name, tainted := range src {
		if tainted {
			dst[name] = true
		}
	}
	return dst
}

func mergeSecretTaintMaps(a, b map[string]bool) map[string]bool {
	merged := copySecretTaintMap(a)
	for name, tainted := range b {
		if tainted {
			merged[name] = true
		}
	}
	return merged
}

func validateDeferBodyControl(stmts []frontend.Stmt, loopDepth int) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			return fmt.Errorf("%s: return is not allowed in defer", frontend.FormatPos(s.At))
		case *frontend.ThrowStmt:
			return fmt.Errorf("%s: throw is not allowed in defer", frontend.FormatPos(s.At))
		case *frontend.DeferStmt:
			return fmt.Errorf("%s: nested defer is not allowed in defer", frontend.FormatPos(s.At))
		case *frontend.BreakStmt:
			if loopDepth == 0 {
				return fmt.Errorf(("%s: break is not allowed in defer outside a cleanup-local " +
					"loop"), frontend.FormatPos(s.At))
			}
		case *frontend.ContinueStmt:
			if loopDepth == 0 {
				return fmt.Errorf(("%s: continue is not allowed in defer outside a " +
					"cleanup-local loop"), frontend.FormatPos(s.At))
			}
		case *frontend.IfStmt:
			if err := validateDeferBodyControl(s.Then, loopDepth); err != nil {
				return err
			}
			if err := validateDeferBodyControl(s.Else, loopDepth); err != nil {
				return err
			}
		case *frontend.IfLetStmt:
			if err := validateDeferBodyControl(s.Then, loopDepth); err != nil {
				return err
			}
			if err := validateDeferBodyControl(s.Else, loopDepth); err != nil {
				return err
			}
		case *frontend.WhileStmt:
			if err := validateDeferBodyControl(s.Body, loopDepth+1); err != nil {
				return err
			}
		case *frontend.ForRangeStmt:
			if err := validateDeferBodyControl(s.Body, loopDepth+1); err != nil {
				return err
			}
		case *frontend.MatchStmt:
			for _, c := range s.Cases {
				if err := validateDeferBodyControl(c.Body, loopDepth); err != nil {
					return err
				}
			}
		case *frontend.IslandStmt:
			if err := validateDeferBodyControl(s.Body, loopDepth); err != nil {
				return err
			}
		case *frontend.UnsafeStmt:
			if err := validateDeferBodyControl(s.Body, loopDepth); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkDeferBody(
	body []frontend.Stmt,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	returnType string,
	borrowedParams map[string]struct{},
	inoutParams map[string]struct{},
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) error {
	savedRegionVars := copyRegionVars(state.regionVars)
	savedUnknownVars := copyBoolMap(state.unknownVars)
	savedUnknownConflicts := copyRegionConflictMap(state.unknownConflicts)
	savedReachable := state.reachable
	savedConsumedVars := copyConsumedVars(state.consumedVars)
	savedMaybeConsumedVars := copyOwnershipJoinConflicts(state.maybeConsumedVars)
	savedOwnershipAliases := copyStringMap(state.ownershipAliases)
	savedBorrowedPtrAliases := copyStringMap(state.borrowedPtrAliases)
	savedOwnedRegionSliceOwners := copyStringMap(state.ownedRegionSliceOwners)
	savedAwaitInvalidatedBorrow := copyPositionByIntMap(state.awaitInvalidatedBorrow)
	savedConsumedResources := copyConsumedResources(state.consumedResources)
	savedResourceVars := copyResourceVars(state.resourceVars)
	savedUnknownResources := copyUnknownResources(state.unknownResources)
	savedFinalizedResources := copyFinalizedResources(state.finalizedResources)
	savedSecretTaint := analysis.copySecretTaint()
	savedNextResourceID := state.nextResourceID
	savedReturnRegion := state.returnRegion
	savedReturnRegionSet := state.returnRegionSet
	savedReturnRegionSummary := cloneReturnRegionSummary(state.returnRegionSummary)
	savedLoopDepth := state.loopDepth
	savedUnsafeDepth := state.unsafeDepth
	savedAllowThrowDepth := state.allowThrowDepth
	savedAllowThrowCall := state.allowThrowCall
	savedAllowCatchDepth := state.allowCatchDepth
	savedAllowCatchCall := state.allowCatchCall
	savedAllowAwaitDepth := state.allowAwaitDepth
	savedAllowAwaitCall := state.allowAwaitCall
	defer func() {
		state.regionVars = savedRegionVars
		state.unknownVars = savedUnknownVars
		state.unknownConflicts = savedUnknownConflicts
		state.reachable = savedReachable
		state.consumedVars = savedConsumedVars
		state.maybeConsumedVars = savedMaybeConsumedVars
		state.ownershipAliases = savedOwnershipAliases
		state.borrowedPtrAliases = savedBorrowedPtrAliases
		state.ownedRegionSliceOwners = savedOwnedRegionSliceOwners
		state.awaitInvalidatedBorrow = savedAwaitInvalidatedBorrow
		state.consumedResources = savedConsumedResources
		state.resourceVars = savedResourceVars
		state.unknownResources = savedUnknownResources
		state.finalizedResources = savedFinalizedResources
		analysis.restoreSecretTaint(savedSecretTaint)
		state.nextResourceID = savedNextResourceID
		state.returnRegion = savedReturnRegion
		state.returnRegionSet = savedReturnRegionSet
		state.returnRegionSummary = savedReturnRegionSummary
		state.loopDepth = savedLoopDepth
		state.unsafeDepth = savedUnsafeDepth
		state.allowThrowDepth = savedAllowThrowDepth
		state.allowThrowCall = savedAllowThrowCall
		state.allowCatchDepth = savedAllowCatchDepth
		state.allowCatchCall = savedAllowCatchCall
		state.allowAwaitDepth = savedAllowAwaitDepth
		state.allowAwaitCall = savedAllowAwaitCall
	}()
	state.loopDepth = 0
	return checkStmts(
		body,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
		returnType,
		borrowedParams,
		inoutParams,
		state,
		effects,
		analysis,
	)
}

func copyBoolMap(src map[string]bool) map[string]bool {
	if len(src) == 0 {
		return make(map[string]bool)
	}
	dst := make(map[string]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return make(map[string]string)
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func mergeBorrowedPtrAliases(a, b map[string]string) map[string]string {
	if len(a) == 0 && len(b) == 0 {
		return make(map[string]string)
	}
	merged := make(map[string]string)
	for name, owner := range a {
		if owner == "" {
			continue
		}
		merged[name] = owner
	}
	for name, owner := range b {
		if owner == "" {
			continue
		}
		if existing, exists := merged[name]; exists {
			if owner < existing {
				merged[name] = owner
			}
			continue
		}
		merged[name] = owner
	}
	return merged
}

func mergeOwnershipAliases(a, b map[string]string) map[string]string {
	if len(a) == 0 && len(b) == 0 {
		return make(map[string]string)
	}
	merged := make(map[string]string)
	for name, source := range a {
		if source == "" {
			continue
		}
		if rightSource, ok := b[name]; ok && rightSource == source {
			merged[name] = source
		}
	}
	return merged
}

func copyRegionConflictMap(src map[string]regionConflict) map[string]regionConflict {
	if len(src) == 0 {
		return make(map[string]regionConflict)
	}
	dst := make(map[string]regionConflict, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyConsumedVars(src map[string]frontend.Position) map[string]frontend.Position {
	if len(src) == 0 {
		return make(map[string]frontend.Position)
	}
	dst := make(map[string]frontend.Position, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyOwnershipJoinConflicts(
	src map[string]ownershipJoinConflict,
) map[string]ownershipJoinConflict {
	if len(src) == 0 {
		return make(map[string]ownershipJoinConflict)
	}
	dst := make(map[string]ownershipJoinConflict, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyConsumedResources(src map[int]frontend.Position) map[int]frontend.Position {
	if len(src) == 0 {
		return make(map[int]frontend.Position)
	}
	dst := make(map[int]frontend.Position, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyPositionByIntMap(src map[int]frontend.Position) map[int]frontend.Position {
	if len(src) == 0 {
		return make(map[int]frontend.Position)
	}
	dst := make(map[int]frontend.Position, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyResourceVars(src map[string]int) map[string]int {
	if len(src) == 0 {
		return make(map[string]int)
	}
	dst := make(map[string]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyUnknownResources(src map[int]bool) map[int]bool {
	if len(src) == 0 {
		return make(map[int]bool)
	}
	dst := make(map[int]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyFinalizedResources(src map[int]resourceFinalization) map[int]resourceFinalization {
	if len(src) == 0 {
		return make(map[int]resourceFinalization)
	}
	dst := make(map[int]resourceFinalization, len(src))
	for k, v := range src {
		dst[k] = copyResourceFinalization(v)
	}
	return dst
}

func copyResourceFinalization(src resourceFinalization) resourceFinalization {
	dst := src
	if len(src.states) > 0 {
		dst.states = make(map[string]frontend.Position, len(src.states))
		for state, pos := range src.states {
			dst.states[state] = pos
		}
	}
	return dst
}

func mergeConsumedVars(a, b map[string]frontend.Position) map[string]frontend.Position {
	if len(a) == 0 && len(b) == 0 {
		return make(map[string]frontend.Position)
	}
	merged := make(map[string]frontend.Position)
	for name, left := range a {
		if right, ok := b[name]; ok {
			merged[name] = earliestPosition(left, right)
			continue
		}
		merged[name] = left
	}
	for name, right := range b {
		if _, exists := merged[name]; !exists {
			merged[name] = right
		}
	}
	return merged
}

func mergeMaybeConsumedVars(
	a, b flowSnapshot,
	leftLabel, rightLabel string,
) map[string]ownershipJoinConflict {
	// SAFE-003 incremental subset: model local consume states as SSA-like edge
	// joins. A value consumed on only some incoming edges remains unusable, but
	// diagnostics now distinguish maybe-consumed joins from linear local flow.
	merged := make(map[string]ownershipJoinConflict)
	names := make(map[string]struct{})
	for name := range a.consumedVars {
		names[name] = struct{}{}
	}
	for name := range b.consumedVars {
		names[name] = struct{}{}
	}
	for name := range a.maybeConsumedVars {
		names[name] = struct{}{}
	}
	for name := range b.maybeConsumedVars {
		names[name] = struct{}{}
	}
	for name := range names {
		leftConsumed, leftMaybe, leftPos, leftConflict := ownershipSnapshotConsumed(a, name)
		rightConsumed, rightMaybe, rightPos, rightConflict := ownershipSnapshotConsumed(b, name)
		if !leftConsumed && !rightConsumed {
			continue
		}
		if leftConsumed && rightConsumed && !leftMaybe && !rightMaybe {
			continue
		}
		conflict := ownershipJoinConflict{
			leftLabel:     leftLabel,
			leftConsumed:  leftConsumed,
			leftPos:       leftPos,
			rightLabel:    rightLabel,
			rightConsumed: rightConsumed,
			rightPos:      rightPos,
		}
		if leftMaybe {
			conflict.leftConsumed = true
			conflict.leftPos = ownershipJoinConflictPosition(leftConflict)
		}
		if rightMaybe {
			conflict.rightConsumed = true
			conflict.rightPos = ownershipJoinConflictPosition(rightConflict)
		}
		merged[name] = conflict
	}
	return merged
}

func ownershipSnapshotConsumed(
	snap flowSnapshot,
	name string,
) (bool, bool, frontend.Position, ownershipJoinConflict) {
	if conflict, ok := snap.maybeConsumedVars[name]; ok {
		return true, true, ownershipJoinConflictPosition(conflict), conflict
	}
	pos, ok := snap.consumedVars[name]
	return ok, false, pos, ownershipJoinConflict{}
}

func ownershipJoinConflictPosition(conflict ownershipJoinConflict) frontend.Position {
	switch {
	case conflict.leftConsumed && conflict.rightConsumed:
		return earliestPosition(conflict.leftPos, conflict.rightPos)
	case conflict.leftConsumed:
		return conflict.leftPos
	case conflict.rightConsumed:
		return conflict.rightPos
	default:
		return frontend.Position{}
	}
}

func mergeConsumedResources(a, b map[int]frontend.Position) map[int]frontend.Position {
	if len(a) == 0 && len(b) == 0 {
		return make(map[int]frontend.Position)
	}
	merged := make(map[int]frontend.Position)
	for id, left := range a {
		if right, ok := b[id]; ok {
			merged[id] = earliestPosition(left, right)
			continue
		}
		merged[id] = left
	}
	for id, right := range b {
		if _, exists := merged[id]; !exists {
			merged[id] = right
		}
	}
	return merged
}

func mergeFinalizedResources(
	a, b map[int]resourceFinalization,
	leftLabel, rightLabel string,
) map[int]resourceFinalization {
	if len(a) == 0 && len(b) == 0 {
		return make(map[int]resourceFinalization)
	}
	merged := make(map[int]resourceFinalization)
	ids := make(map[int]struct{})
	for id := range a {
		ids[id] = struct{}{}
	}
	for id := range b {
		ids[id] = struct{}{}
	}
	for id := range ids {
		left, leftOK := a[id]
		right, rightOK := b[id]
		if final, ok := mergeResourceFinalizationValues(
			left,
			leftOK,
			right,
			rightOK,
			leftLabel,
			rightLabel,
		); ok {
			merged[id] = final
		}
	}
	return merged
}

func mergeResourceFinalizationValues(
	left resourceFinalization,
	leftOK bool,
	right resourceFinalization,
	rightOK bool,
	leftLabel, rightLabel string,
) (resourceFinalization, bool) {
	if !leftOK && !rightOK {
		return resourceFinalization{}, false
	}
	states := make(map[string]frontend.Position)
	mayBeAvailable := !leftOK || !rightOK
	addFinalizationStates(states, left)
	addFinalizationStates(states, right)
	if leftOK && left.mayBeAvailable {
		mayBeAvailable = true
	}
	if rightOK && right.mayBeAvailable {
		mayBeAvailable = true
	}
	if len(states) == 0 {
		return resourceFinalization{}, false
	}
	if len(states) == 1 && !mayBeAvailable && !left.maybe && !right.maybe {
		for state, pos := range states {
			return resourceFinalization{state: state, pos: pos}, true
		}
	}
	return resourceFinalization{
		state:          firstResourceFinalizationState(states),
		pos:            earliestResourceFinalizationPosition(states),
		maybe:          true,
		mayBeAvailable: mayBeAvailable,
		states:         states,
	}, true
}

func addFinalizationStates(dst map[string]frontend.Position, final resourceFinalization) {
	for state, pos := range resourceFinalizationStatePositions(final) {
		if existing, ok := dst[state]; ok {
			dst[state] = earliestPosition(existing, pos)
			continue
		}
		dst[state] = pos
	}
}

func firstResourceFinalizationState(states map[string]frontend.Position) string {
	first := ""
	for state := range states {
		if first == "" || state < first {
			first = state
		}
	}
	return first
}

func earliestResourceFinalizationPosition(states map[string]frontend.Position) frontend.Position {
	var earliest frontend.Position
	for _, pos := range states {
		earliest = earliestPosition(earliest, pos)
	}
	return earliest
}

func mergeUnknownResources(a, b map[int]bool) map[int]bool {
	if len(a) == 0 && len(b) == 0 {
		return make(map[int]bool)
	}
	merged := make(map[int]bool)
	for id, unknown := range a {
		if unknown {
			merged[id] = true
		}
	}
	for id, unknown := range b {
		if unknown {
			merged[id] = true
		}
	}
	return merged
}

func mergeAwaitInvalidatedBorrowRegions(a, b map[int]frontend.Position) map[int]frontend.Position {
	if len(a) == 0 && len(b) == 0 {
		return make(map[int]frontend.Position)
	}
	merged := copyPositionByIntMap(a)
	for regionID, pos := range b {
		if existing, exists := merged[regionID]; exists {
			merged[regionID] = earliestPosition(existing, pos)
			continue
		}
		merged[regionID] = pos
	}
	return merged
}

func mergeResourceVars(
	state *regionState,
	a, b map[string]int,
	consumed map[int]frontend.Position,
	finalized map[int]resourceFinalization,
	unknown map[int]bool,
	leftLabel, rightLabel string,
) map[string]int {
	if len(a) == 0 && len(b) == 0 {
		return make(map[string]int)
	}
	merged := make(map[string]int)
	for name, left := range a {
		right, ok := b[name]
		if !ok {
			merged[name] = left
			continue
		}
		if left == right {
			merged[name] = left
			continue
		}
		merged[name] = mergeResourceIDs(
			state,
			left,
			right,
			consumed,
			finalized,
			unknown,
			leftLabel,
			rightLabel,
		)
	}
	for name, right := range b {
		if _, exists := merged[name]; !exists {
			merged[name] = right
		}
	}
	return merged
}

func mergeResourceIDs(
	state *regionState,
	left int,
	right int,
	consumed map[int]frontend.Position,
	finalized map[int]resourceFinalization,
	unknown map[int]bool,
	leftLabel, rightLabel string,
) int {
	if state == nil {
		return left
	}
	merged := state.allocateResourceID()
	leftParam, leftParamOK := state.resourceParamIndex[left]
	rightParam, rightParamOK := state.resourceParamIndex[right]
	leftPath := state.resourceParamPath[left]
	rightPath := state.resourceParamPath[right]
	if unknown[left] || unknown[right] {
		unknown[merged] = true
	} else if leftParamOK && rightParamOK && leftParam == rightParam && leftPath == rightPath {
		state.resourceParamIndex[merged] = leftParam
		state.resourceParamPath[merged] = leftPath
	} else {
		unknown[merged] = true
	}
	leftConsumed, leftConsumedOK := consumed[left]
	rightConsumed, rightConsumedOK := consumed[right]
	switch {
	case leftConsumedOK && rightConsumedOK:
		consumed[merged] = earliestPosition(leftConsumed, rightConsumed)
	case leftConsumedOK:
		consumed[merged] = leftConsumed
	case rightConsumedOK:
		consumed[merged] = rightConsumed
	}
	leftFinal, leftFinalOK := finalized[left]
	rightFinal, rightFinalOK := finalized[right]
	if final, ok := mergeResourceFinalizationValues(
		leftFinal,
		leftFinalOK,
		rightFinal,
		rightFinalOK,
		leftLabel,
		rightLabel,
	); ok {
		finalized[merged] = final
	}
	return merged
}

func earliestPosition(a, b frontend.Position) frontend.Position {
	if a.Line == 0 {
		return b
	}
	if b.Line == 0 {
		return a
	}
	if a.Line < b.Line || (a.Line == b.Line && a.Col <= b.Col) {
		return a
	}
	return b
}

func earliestFinalization(a, b resourceFinalization) resourceFinalization {
	if earliestPosition(a.pos, b.pos) == a.pos {
		return a
	}
	return b
}

// ---- checker_declarations.go ----

func validateCapsuleDecls(file *frontend.FileAST) error {
	return semanticsdeclarations.ValidateCapsuleDecls(file)
}

func validateEnumPayloadCycles(
	edges map[string]map[string]frontend.Position,
	enumModules map[string]string,
) error {
	const (
		enumVisitNew = iota
		enumVisitActive
		enumVisitDone
	)
	visitState := map[string]int{}
	var dfs func(string) error
	dfs = func(name string) error {
		visitState[name] = enumVisitActive
		for target, at := range edges[name] {
			if _, ok := enumModules[target]; !ok {
				continue
			}
			switch visitState[target] {
			case enumVisitActive:
				moduleName := enumModules[name]
				if tgt, ok := enumModules[target]; ok {
					moduleName = tgt
				}
				return fmt.Errorf(
					"%s: recursive enum payload '%s'",
					frontend.FormatPos(at),
					displayTypeName(target, moduleName),
				)
			case enumVisitDone:
				continue
			default:
				if err := dfs(target); err != nil {
					return err
				}
			}
		}
		visitState[name] = enumVisitDone
		return nil
	}
	for name := range edges {
		if visitState[name] == enumVisitNew {
			if err := dfs(name); err != nil {
				return err
			}
		}
	}
	return nil
}

func refreshCompositeSlotLayouts(types map[string]*TypeInfo) error {
	if len(types) == 0 {
		return nil
	}
	maxPasses := len(types) + 1
	for pass := 0; pass < maxPasses; pass++ {
		changed := false
		for _, info := range types {
			if info == nil {
				continue
			}
			switch info.Kind {
			case TypeOptional:
				elemInfo, ok := types[info.ElemType]
				if !ok {
					continue
				}
				slotCount := elemInfo.SlotCount + 1
				if info.SlotCount != slotCount {
					info.SlotCount = slotCount
					changed = true
				}
			case TypeStruct:
				if len(info.Fields) == 0 {
					continue
				}
				offset := 0
				fieldMap := make(map[string]FieldInfo, len(info.Fields))
				fields := make([]FieldInfo, len(info.Fields))
				for i, field := range info.Fields {
					slotCount := field.SlotCount
					if fieldInfo, ok := types[field.TypeName]; ok {
						slotCount = fieldInfo.SlotCount
					}
					if slotCount <= 0 {
						slotCount = 1
					}
					field.Offset = offset
					field.SlotCount = slotCount
					fields[i] = field
					fieldMap[field.Name] = field
					offset += slotCount
				}
				if info.SlotCount != offset || !fieldLayoutsEqual(info.Fields, fields) {
					info.SlotCount = offset
					info.Fields = fields
					info.FieldMap = fieldMap
					changed = true
				}
			case TypeEnum:
				maxPayloadSlots := 0
				for i, caseInfo := range info.EnumCases {
					totalPayloadSlots := 0
					for j, payloadType := range caseInfo.PayloadTypes {
						slotCount := 1
						if payloadInfo, ok := types[payloadType]; ok {
							slotCount = payloadInfo.SlotCount
						}
						if slotCount <= 0 {
							slotCount = 1
						}
						if j < len(caseInfo.PayloadSlots) && caseInfo.PayloadSlots[j] != slotCount {
							caseInfo.PayloadSlots[j] = slotCount
							changed = true
						}
						totalPayloadSlots += slotCount
					}
					if caseInfo.SlotCount != totalPayloadSlots {
						caseInfo.SlotCount = totalPayloadSlots
						changed = true
					}
					info.EnumCases[i] = caseInfo
					info.CaseMap[caseInfo.Name] = caseInfo
					if totalPayloadSlots > maxPayloadSlots {
						maxPayloadSlots = totalPayloadSlots
					}
				}
				slotCount := 1 + maxPayloadSlots
				if info.SlotCount != slotCount {
					info.SlotCount = slotCount
					changed = true
				}
			}
		}
		if !changed {
			return nil
		}
	}
	return fmt.Errorf("recursive composite type layout is not supported")
}

func structReprOrDefault(repr string) string {
	if repr == "" {
		return frontend.StructReprDefault
	}
	return repr
}

func fieldLayoutsEqual(a, b []FieldInfo) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func collectCapsulePermissionsByModule(world *module.World) map[string]map[string]struct{} {
	out := map[string]map[string]struct{}{}
	if world == nil {
		return out
	}
	for _, file := range world.Files {
		if file == nil {
			continue
		}
		moduleName := file.Module
		if _, ok := out[moduleName]; !ok {
			out[moduleName] = map[string]struct{}{}
		}
		for _, capsule := range file.Capsules {
			if capsule == nil {
				continue
			}
			for _, entry := range capsule.Entries {
				if granted, ok := capsulePermissionEntry(entry); ok && granted {
					out[moduleName][capsulePermissionFromEntryKey(entry.Key)] = struct{}{}
				}
			}
		}
	}
	return out
}

func capsulePermissionEntry(entry frontend.CapsuleEntryDecl) (bool, bool) {
	switch entry.Key {
	case "permissions.io", "permissions.mem":
		b, ok := entry.Value.(*frontend.BoolLitExpr)
		if !ok {
			return false, true
		}
		return b.Value, true
	default:
		return false, false
	}
}

func capsulePermissionFromEntryKey(key string) string {
	switch key {
	case "permissions.io":
		return "capsule.io"
	case "permissions.mem":
		return "capsule.mem"
	default:
		return ""
	}
}

func isCapsuleMetadataLiteral(expr frontend.Expr) bool {
	return semanticsdeclarations.IsCapsuleMetadataLiteral(expr)
}

func isCapsuleMetadataKey(key string) bool {
	return semanticsdeclarations.IsCapsuleMetadataKey(key)
}

func isCapsuleKeySegment(seg string) bool {
	return semanticsdeclarations.IsCapsuleKeySegment(seg)
}

func stmtListEndsWithReturn(stmts []frontend.Stmt) bool {
	return semanticsstatements.ListEndsWithReturn(stmts)
}

func injectActorStateLocals(
	fields map[string]ActorStateField,
	locals map[string]LocalInfo,
	scopes *scopeInfo,
) error {
	for name, field := range fields {
		if _, exists := locals[name]; exists {
			return fmt.Errorf("duplicate local '%s'", name)
		}
		locals[name] = LocalInfo{
			Base:           -1,
			SlotCount:      1,
			TypeName:       field.TypeName,
			Mutable:        field.Mutable,
			Const:          field.Const,
			ActorField:     true,
			ActorFieldSlot: field.Slot,
			ActorFieldInit: field.Init,
		}
		if scopes != nil && scopes.localScopes != nil {
			scopes.localScopes[name] = regionNone
		}
	}
	return nil
}

func normalizeExtensionMethodNames(world *module.World) error {
	for _, file := range world.Files {
		imports, err := collectImportAliases(file)
		if err != nil {
			return err
		}
		for _, fn := range file.Funcs {
			if fn.ExtensionOf == "" {
				continue
			}
			targetRef := frontend.TypeRef{
				At:   fn.Pos,
				Kind: frontend.TypeRefNamed,
				Name: fn.ExtensionOf,
			}
			resolvedTarget, err := resolveTypeName(&targetRef, file.Module, imports)
			if err != nil {
				return err
			}
			methodName := extensionMethodNamePart(fn.Name)
			if methodName == "" {
				return fmt.Errorf(
					"%s: invalid extension method name '%s'",
					frontend.FormatPos(fn.Pos),
					fn.Name,
				)
			}
			fn.ExtensionOf = resolvedTarget
			fn.Name = resolvedTarget + "." + methodName
		}
	}
	return nil
}

func extensionMethodNamePart(name string) string {
	if idx := strings.LastIndex(name, "."); idx >= 0 && idx+1 < len(name) {
		return name[idx+1:]
	}
	return name
}

func checkedFuncFullName(module string, fn *frontend.FuncDecl) string {
	return semanticsworld.CheckedFuncFullName(module, fn)
}

func validateFunctionParamNames(fn *frontend.FuncDecl) error {
	seen := make(map[string]struct{}, len(fn.Params))
	for _, param := range fn.Params {
		if param.Name == "" {
			return fmt.Errorf("%s: parameter name required", frontend.FormatPos(param.At))
		}
		if _, exists := seen[param.Name]; exists {
			return fmt.Errorf(
				"%s: duplicate parameter '%s'",
				frontend.FormatPos(param.At),
				param.Name,
			)
		}
		seen[param.Name] = struct{}{}
	}
	return nil
}

func addPublicImportFunctionAliases(world *module.World, funcs map[string]FuncSig) error {
	for _, file := range world.Files {
		for _, imp := range file.Imports {
			if !imp.Public || len(imp.Items) == 0 {
				continue
			}
			for _, item := range imp.Items {
				target := qualifyName(imp.Path, item)
				sig, ok := funcs[target]
				if !ok {
					continue
				}
				if err := ensureFuncVisible(target, sig, file.Module, imp.At); err != nil {
					return err
				}
				alias := qualifyName(file.Module, item)
				if _, exists := funcs[alias]; exists && alias != target {
					return fmt.Errorf(
						"%s: re-export '%s' conflicts with function '%s'",
						frontend.FormatPos(imp.At),
						item,
						alias,
					)
				}
				sig.Public = true
				funcs[alias] = sig
			}
		}
	}
	return nil
}

func addImportedFunctionTypedGlobalAliases(
	world *module.World,
	globalsByModule map[string]map[string]GlobalInfo,
) error {
	if world == nil {
		return nil
	}
	for _, file := range world.Files {
		if file == nil {
			continue
		}
		globals := globalsByModule[file.Module]
		if globals == nil {
			globals = make(map[string]GlobalInfo)
			globalsByModule[file.Module] = globals
		}
		imports, err := collectImportAliases(file)
		if err != nil {
			return err
		}
		for alias, target := range imports {
			if symbol, isSymbol := importSymbolTarget(target); isSymbol {
				dot := strings.LastIndex(symbol, ".")
				if dot < 0 || dot == 0 || dot == len(symbol)-1 {
					continue
				}
				moduleName := symbol[:dot]
				globalName := symbol[dot+1:]
				global, ok := globalsByModule[moduleName][globalName]
				if !ok || !global.Public || !global.FunctionTypeValue ||
					global.FunctionValue == "" {
					continue
				}
				if global.Mutable {
					global.FunctionValue = ""
				}
				globals[alias] = global
				globals[symbol] = global
				continue
			}
			importedGlobals := globalsByModule[target]
			for name, global := range importedGlobals {
				if !global.Public || !global.FunctionTypeValue || global.FunctionValue == "" {
					continue
				}
				if global.Mutable {
					global.FunctionValue = ""
				}
				globals[alias+"."+name] = global
				globals[target+"."+name] = global
			}
		}
	}
	return nil
}

func initialReturnResourceParam(returnType string, types map[string]*TypeInfo) int {
	if typeContainsResourceHandle(returnType, types) {
		return regionUnknown
	}
	return regionNone
}

func cloneReturnRegionSummary(in ReturnRegionSummary) ReturnRegionSummary {
	return semanticsflow.CloneReturnRegionSummary(in)
}

func returnRegionSummariesEqual(a, b ReturnRegionSummary) bool {
	return semanticsflow.ReturnRegionSummariesEqual(a, b)
}

func cloneReturnResourceSummary(in ReturnResourceSummary) ReturnResourceSummary {
	return semanticsflow.CloneReturnResourceSummary(in)
}

func returnResourceSummariesEqual(a, b ReturnResourceSummary) bool {
	return semanticsflow.ReturnResourceSummariesEqual(a, b)
}

func stmtListEndsWithReturnTyped(
	stmts []frontend.Stmt,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) bool {
	if len(stmts) == 0 {
		return false
	}
	return stmtEndsWithReturnTyped(
		stmts[len(stmts)-1],
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
	)
}

func stmtEndsWithReturnTyped(
	stmt frontend.Stmt,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) bool {
	switch s := stmt.(type) {
	case *frontend.ReturnStmt, *frontend.ThrowStmt:
		return true
	case *frontend.IfStmt:
		return len(s.Then) > 0 && len(s.Else) > 0 &&
			stmtListEndsWithReturnTyped(s.Then, locals, globals, funcs, types, module, imports) &&
			stmtListEndsWithReturnTyped(s.Else, locals, globals, funcs, types, module, imports)
	case *frontend.IfLetStmt:
		return len(s.Then) > 0 && len(s.Else) > 0 &&
			stmtListEndsWithReturnTyped(s.Then, locals, globals, funcs, types, module, imports) &&
			stmtListEndsWithReturnTyped(s.Else, locals, globals, funcs, types, module, imports)
	case *frontend.MatchStmt:
		for _, c := range s.Cases {
			if !stmtListEndsWithReturnTyped(c.Body, locals, globals, funcs, types, module, imports) {
				return false
			}
		}
		if matchHasDefault(s) || matchHasCompleteOptionalPatterns(s) {
			return true
		}
		return matchHasCompleteEnumPatterns(s, locals, globals, funcs, types, module, imports)
	case *frontend.UnsafeStmt:
		return stmtListEndsWithReturnTyped(s.Body, locals, globals, funcs, types, module, imports)
	default:
		return false
	}
}

func matchHasDefault(s *frontend.MatchStmt) bool {
	for _, c := range s.Cases {
		if c.Default && c.Guard == nil {
			return true
		}
	}
	return false
}

func matchHasCompleteOptionalPatterns(s *frontend.MatchStmt) bool {
	hasNone := false
	hasSome := false
	for _, c := range s.Cases {
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		if _, ok := c.Pattern.(*frontend.NoneLitExpr); ok {
			hasNone = true
		}
		if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
			hasSome = true
		}
	}
	return hasNone && hasSome
}

func matchHasCompleteEnumPatterns(
	s *frontend.MatchStmt,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) bool {
	scrutType, err := inferExprTypeForDecl(s.Value, locals, globals, funcs, types, module, imports)
	if err != nil {
		return false
	}
	info, ok := types[scrutType]
	if !ok || info.Kind != TypeEnum || len(info.EnumCases) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(info.EnumCases))
	for i := range s.Cases {
		c := &s.Cases[i]
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		switch pat := c.Pattern.(type) {
		case *frontend.FieldAccessExpr:
			caseType, caseName, ok := bareEnumPatternTypeAndCase(pat, module, imports)
			if !ok || caseType != scrutType {
				return false
			}
			caseInfo, ok := info.CaseMap[caseName]
			if !ok || len(caseInfo.PayloadTypes) != 0 {
				return false
			}
			pat.EnumType = scrutType
			pat.EnumOrdinal = caseInfo.Ordinal
			seen[caseName] = struct{}{}
		case *frontend.EnumCasePatternExpr:
			caseType, caseInfo, found, err := resolveEnumCasePattern(pat, types, module, imports)
			if err != nil || !found || caseType != scrutType {
				return false
			}
			if err := validateEnumCasePatternPayload(pat, caseType, caseInfo, module); err != nil {
				return false
			}
			seen[pat.CaseName] = struct{}{}
		default:
			return false
		}
	}
	for _, enumCase := range info.EnumCases {
		if _, ok := seen[enumCase.Name]; !ok {
			return false
		}
	}
	return true
}

func matchExprHasCompleteOptionalPatterns(e *frontend.MatchExpr) bool {
	seenNone := false
	seenSome := false
	for _, c := range e.Cases {
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		if _, ok := c.Pattern.(*frontend.NoneLitExpr); ok {
			seenNone = true
		}
		if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
			seenSome = true
		}
	}
	return seenNone && seenSome
}

func matchExprHasCompleteEnumPatterns(
	e *frontend.MatchExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) bool {
	scrutType, err := inferExprTypeForDecl(e.Value, locals, globals, funcs, types, module, imports)
	if err != nil {
		return false
	}
	info, ok := types[scrutType]
	if !ok || info.Kind != TypeEnum || len(info.EnumCases) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(info.EnumCases))
	for i := range e.Cases {
		c := &e.Cases[i]
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		switch pat := c.Pattern.(type) {
		case *frontend.FieldAccessExpr:
			caseType, caseName, ok := bareEnumPatternTypeAndCase(pat, module, imports)
			if !ok || caseType != scrutType {
				return false
			}
			caseInfo, ok := info.CaseMap[caseName]
			if !ok || len(caseInfo.PayloadTypes) != 0 {
				return false
			}
			pat.EnumType = scrutType
			pat.EnumOrdinal = caseInfo.Ordinal
			seen[caseName] = struct{}{}
		case *frontend.EnumCasePatternExpr:
			caseType, caseInfo, found, err := resolveEnumCasePattern(pat, types, module, imports)
			if err != nil || !found || caseType != scrutType {
				return false
			}
			if err := validateEnumCasePatternPayload(pat, caseType, caseInfo, module); err != nil {
				return false
			}
			seen[pat.CaseName] = struct{}{}
		default:
			return false
		}
	}
	for _, enumCase := range info.EnumCases {
		if _, ok := seen[enumCase.Name]; !ok {
			return false
		}
	}
	return true
}

func inferMatchExprType(
	e *frontend.MatchExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (string, error) {
	scrutType, err := inferExprTypeForDecl(e.Value, locals, globals, funcs, types, module, imports)
	if err != nil {
		return "", err
	}
	resultType := ""
	for _, c := range e.Cases {
		armLocals := cloneLocalMap(locals)
		if !c.Default {
			if err := bindMatchPatternLocalsForInference(
				c.Pattern,
				scrutType,
				armLocals,
				types,
				module,
				imports,
			); err != nil {
				return "", err
			}
		}
		armType, err := inferExprTypeForDecl(
			c.Value,
			armLocals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return "", err
		}
		if resultType == "" {
			resultType = armType
			continue
		}
		if !typesCompatibleWithNullPtr(resultType, armType, c.Value) {
			return "", fmt.Errorf(
				"match expression case type mismatch: expected '%s', got '%s'",
				resultType,
				armType,
			)
		}
	}
	if resultType == "" {
		return "", fmt.Errorf("match expression requires at least one case")
	}
	e.ResultType = resultType
	return resultType, nil
}

func inferCatchExprType(
	e *frontend.CatchExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (string, error) {
	call, ok := e.Call.(*frontend.CallExpr)
	if !ok {
		return "", fmt.Errorf("catch expects a throwing function call")
	}
	if builtin, ok := ResolveBuiltinAlias(call.Name); ok &&
		(builtin == "core.task_join_i32_typed" || builtin == "core.task_join_group_i32_typed") {
		if len(call.TypeArgs) != 1 {
			return "", fmt.Errorf("task_join_i32_typed expects one explicit error type argument")
		}
		errorType, err := resolveTypeName(&call.TypeArgs[0], module, imports)
		if err != nil {
			return "", err
		}
		if err := validateTypedTaskErrorType(errorType, types, call.TypeArgs[0].At); err != nil {
			return "", err
		}
		call.TypeArgs[0].Name = errorType
		e.ErrorType = errorType
		e.ResultType = "i32"
		for _, c := range e.Cases {
			armLocals := cloneLocalMap(locals)
			if !c.Default {
				if err := bindMatchPatternLocalsForInference(
					c.Pattern,
					errorType,
					armLocals,
					types,
					module,
					imports,
				); err != nil {
					return "", err
				}
			}
			armType, err := inferExprTypeForDecl(
				c.Value,
				armLocals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return "", err
			}
			if !typesCompatibleWithNullPtr("i32", armType, c.Value) {
				return "", fmt.Errorf(
					"catch expression case type mismatch: expected 'i32', got '%s'",
					armType,
				)
			}
		}
		return "i32", nil
	}
	sig, err := resolveCallSigForInference(call, funcs, module, imports)
	if err != nil {
		return "", err
	}
	if sig.ThrowsType == "" {
		return "", fmt.Errorf("catch expects a throwing function call")
	}
	e.ErrorType = sig.ThrowsType
	e.ResultType = sig.ReturnType
	for _, c := range e.Cases {
		armLocals := cloneLocalMap(locals)
		if !c.Default {
			if err := bindMatchPatternLocalsForInference(
				c.Pattern,
				sig.ThrowsType,
				armLocals,
				types,
				module,
				imports,
			); err != nil {
				return "", err
			}
		}
		armType, err := inferExprTypeForDecl(
			c.Value,
			armLocals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return "", err
		}
		if !typesCompatibleWithNullPtr(sig.ReturnType, armType, c.Value) {
			return "", fmt.Errorf(
				"catch expression case type mismatch: expected '%s', got '%s'",
				sig.ReturnType,
				armType,
			)
		}
	}
	return sig.ReturnType, nil
}

func resolveCallSigForInference(
	call *frontend.CallExpr,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
) (FuncSig, error) {
	var zero FuncSig
	resolved := ""
	if builtin, ok := ResolveBuiltinAlias(call.Name); ok {
		resolved = builtin
	} else if _, ok := funcs[call.Name]; ok {
		resolved = call.Name
	} else {
		name, err := resolveKnownCallName(call.Name, funcs, module, imports, call.At)
		if err != nil {
			return zero, err
		}
		resolved = name
	}
	sig, ok := funcs[resolved]
	if !ok {
		return zero, fmt.Errorf("unknown function '%s'", resolved)
	}
	if sig.Generic {
		return zero, fmt.Errorf(
			"generic function '%s' could not be monomorphized; use inferable value arguments",
			call.Name,
		)
	}
	return sig, nil
}

func catchPatternType(
	pattern frontend.Expr,
	errorType string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) (string, error) {
	info, ok := types[errorType]
	if !ok {
		return "", fmt.Errorf("unknown type '%s'", errorType)
	}
	switch pat := pattern.(type) {
	case *frontend.SomePatternExpr:
		if info.Kind != TypeOptional {
			return "", fmt.Errorf(
				"%s: some pattern requires optional catch value",
				frontend.FormatPos(pat.At),
			)
		}
		return optionalSomePatternType, nil
	case *frontend.EnumCasePatternExpr:
		caseType, caseInfo, found, err := resolveEnumCasePattern(pat, types, module, imports)
		if err != nil {
			return "", err
		}
		if !found {
			return "", fmt.Errorf(
				"%s: unknown enum pattern '%s.%s'",
				frontend.FormatPos(pat.At),
				pat.TypeName,
				pat.CaseName,
			)
		}
		if err := validateEnumCasePatternPayload(pat, caseType, caseInfo, module); err != nil {
			return "", err
		}
		return caseType, nil
	default:
		patType, _, err := checkExprWithEffects(
			pattern,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
		return patType, err
	}
}

func catchExprHasCompleteOptionalPatterns(
	e *frontend.CatchExpr,
	errorType string,
	types map[string]*TypeInfo,
) bool {
	info, ok := types[errorType]
	if !ok || info.Kind != TypeOptional {
		return false
	}
	seenSome := false
	seenNone := false
	for _, c := range e.Cases {
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
			seenSome = true
			continue
		}
		if _, ok := c.Pattern.(*frontend.NoneLitExpr); ok {
			seenNone = true
			continue
		}
		return false
	}
	return seenSome && seenNone
}

func catchExprHasCompleteEnumPatterns(
	e *frontend.CatchExpr,
	errorType string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) bool {
	info, ok := types[errorType]
	if !ok || info.Kind != TypeEnum || len(info.EnumCases) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(info.EnumCases))
	for i := range e.Cases {
		c := &e.Cases[i]
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		switch pat := c.Pattern.(type) {
		case *frontend.FieldAccessExpr:
			caseType, caseName, ok := bareEnumPatternTypeAndCase(pat, module, imports)
			if !ok || caseType != errorType {
				return false
			}
			caseInfo, ok := info.CaseMap[caseName]
			if !ok || len(caseInfo.PayloadTypes) != 0 {
				return false
			}
			pat.EnumType = errorType
			pat.EnumOrdinal = caseInfo.Ordinal
			seen[caseName] = struct{}{}
		case *frontend.EnumCasePatternExpr:
			caseType, caseInfo, found, err := resolveEnumCasePattern(pat, types, module, imports)
			if err != nil || !found || caseType != errorType {
				return false
			}
			if err := validateEnumCasePatternPayload(pat, caseType, caseInfo, module); err != nil {
				return false
			}
			seen[pat.CaseName] = struct{}{}
		default:
			return false
		}
	}
	for _, enumCase := range info.EnumCases {
		if _, ok := seen[enumCase.Name]; !ok {
			return false
		}
	}
	return true
}

func cloneLocalMap(locals map[string]LocalInfo) map[string]LocalInfo {
	out := make(map[string]LocalInfo, len(locals))
	for name, info := range locals {
		out[name] = info
	}
	return out
}

func bindMatchPatternLocalsForInference(
	pattern frontend.Expr,
	scrutType string,
	locals map[string]LocalInfo,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) error {
	info, ok := types[scrutType]
	if !ok {
		return fmt.Errorf("unknown type '%s'", scrutType)
	}
	switch pat := pattern.(type) {
	case *frontend.SomePatternExpr:
		if info.Kind != TypeOptional {
			return fmt.Errorf("%s: some pattern requires optional match value", frontend.FormatPos(pat.At))
		}
		elemInfo, err := ensureTypeInfo(info.ElemType, types)
		if err != nil {
			return err
		}
		locals[pat.Name] = LocalInfo{SlotCount: elemInfo.SlotCount, TypeName: info.ElemType}
	case *frontend.EnumCasePatternExpr:
		caseType, caseInfo, found, err := resolveEnumCasePattern(pat, types, module, imports)
		if err != nil {
			return err
		}
		if !found || caseType != scrutType {
			return fmt.Errorf("%s: enum pattern type mismatch", frontend.FormatPos(pat.At))
		}
		if err := validateEnumCasePatternPayload(pat, caseType, caseInfo, module); err != nil {
			return err
		}
		for i, binding := range pat.Bindings {
			locals[binding] = LocalInfo{
				SlotCount: caseInfo.PayloadSlots[i],
				TypeName:  caseInfo.PayloadTypes[i],
			}
		}
	}
	return nil
}

func validateEnumCasePatternPayload(
	pattern *frontend.EnumCasePatternExpr,
	caseType string,
	caseInfo EnumCaseInfo,
	module string,
) error {
	want := len(caseInfo.PayloadTypes)
	got := len(pattern.Bindings)
	if got > 0 && !pattern.HasPayload {
		pattern.HasPayload = true
	}
	if want == 0 {
		if pattern.HasPayload {
			return fmt.Errorf(
				"%s: enum case '%s.%s' has no payload; use '%s.%s'",
				frontend.FormatPos(pattern.At),
				displayTypeName(caseType, module),
				pattern.CaseName,
				displayTypeName(caseType, module),
				pattern.CaseName,
			)
		}
		if got != 0 {
			return fmt.Errorf(
				"%s: enum case '%s.%s' pattern expects 0 binding(s), got %d",
				frontend.FormatPos(pattern.At),
				displayTypeName(caseType, module),
				pattern.CaseName,
				got,
			)
		}
		return nil
	}
	if !pattern.HasPayload {
		return fmt.Errorf(
			"%s: enum case '%s.%s' carries %d payload value(s); use '%s.%s(%s)'",
			frontend.FormatPos(pattern.At),
			displayTypeName(caseType, module),
			pattern.CaseName,
			want,
			displayTypeName(caseType, module),
			pattern.CaseName,
			placeholderBindingList(want),
		)
	}
	if got != want {
		return fmt.Errorf(
			"%s: enum case '%s.%s' pattern expects %d binding(s), got %d",
			frontend.FormatPos(pattern.At),
			displayTypeName(caseType, module),
			pattern.CaseName,
			want,
			got,
		)
	}
	return nil
}

func placeholderBindingList(n int) string {
	if n <= 0 {
		return ""
	}
	bindings := make([]string, n)
	for i := range bindings {
		bindings[i] = fmt.Sprintf("value%d", i+1)
	}
	return strings.Join(bindings, ", ")
}

func stmtEndsWithReturn(stmt frontend.Stmt) bool {
	return semanticsstatements.EndsWithReturn(stmt)
}

// ---- checker_entry_helpers.go ----

func worldFilesImportsFirst(world *module.World) []*frontend.FileAST {
	if world == nil || len(world.Files) == 0 {
		return nil
	}
	if len(world.ByModule) == 0 {
		return append([]*frontend.FileAST(nil), world.Files...)
	}
	out := make([]*frontend.FileAST, 0, len(world.Files))
	seen := map[string]bool{}
	var visit func(*frontend.FileAST)
	visit = func(file *frontend.FileAST) {
		if file == nil {
			return
		}
		key := file.Module
		if key == "" {
			key = file.Path
		}
		if seen[key] {
			return
		}
		seen[key] = true
		for _, imp := range file.Imports {
			visit(world.ByModule[imp.Path])
		}
		out = append(out, file)
	}
	for _, file := range world.Files {
		visit(file)
	}
	return out
}

func clearMatchPayloadExhaustivenessMarkers(world *module.World) {
	if world == nil {
		return
	}
	for _, file := range world.Files {
		if file == nil {
			continue
		}
		for _, fn := range file.Funcs {
			clearStmtListMatchPayloadMarkers(fn.Body)
		}
		for _, test := range file.Tests {
			clearStmtListMatchPayloadMarkers(test.Body)
		}
	}
}

func clearStmtListMatchPayloadMarkers(stmts []frontend.Stmt) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.IfStmt:
			clearStmtListMatchPayloadMarkers(s.Then)
			clearStmtListMatchPayloadMarkers(s.Else)
		case *frontend.IfLetStmt:
			clearStmtListMatchPayloadMarkers(s.Then)
			clearStmtListMatchPayloadMarkers(s.Else)
		case *frontend.WhileStmt:
			clearStmtListMatchPayloadMarkers(s.Body)
		case *frontend.ForRangeStmt:
			clearStmtListMatchPayloadMarkers(s.Body)
		case *frontend.MatchStmt:
			for i := range s.Cases {
				s.Cases[i].RequiresPayload = false
				s.Cases[i].PayloadArity = 0
				clearStmtListMatchPayloadMarkers(s.Cases[i].Body)
			}
		case *frontend.UnsafeStmt:
			clearStmtListMatchPayloadMarkers(s.Body)
		case *frontend.DeferStmt:
			clearStmtListMatchPayloadMarkers(s.Body)
		}
		clearExprMatchPayloadMarkers(stmtExpr(stmt))
	}
}

func stmtExpr(stmt frontend.Stmt) frontend.Expr {
	switch s := stmt.(type) {
	case *frontend.LetStmt:
		return s.Value
	case *frontend.AssignStmt:
		return s.Value
	case *frontend.ExprStmt:
		return s.Expr
	case *frontend.ReturnStmt:
		return s.Value
	case *frontend.ThrowStmt:
		return s.Value
	case *frontend.PrintStmt:
		return s.Value
	case *frontend.FreeStmt:
		return s.Value
	}
	return nil
}

func reportStmtListMatchPayloadDiagnostics(stmts []frontend.Stmt, module string) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.IfStmt:
			if err := reportStmtListMatchPayloadDiagnostics(s.Then, module); err != nil {
				return err
			}
			if err := reportStmtListMatchPayloadDiagnostics(s.Else, module); err != nil {
				return err
			}
		case *frontend.IfLetStmt:
			if err := reportStmtListMatchPayloadDiagnostics(s.Then, module); err != nil {
				return err
			}
			if err := reportStmtListMatchPayloadDiagnostics(s.Else, module); err != nil {
				return err
			}
		case *frontend.WhileStmt:
			if err := reportStmtListMatchPayloadDiagnostics(s.Body, module); err != nil {
				return err
			}
		case *frontend.ForRangeStmt:
			if err := reportStmtListMatchPayloadDiagnostics(s.Body, module); err != nil {
				return err
			}
		case *frontend.MatchStmt:
			for i := range s.Cases {
				c := &s.Cases[i]
				if c.RequiresPayload {
					return payloadRequiredDiagnostic(c.At, c.Pattern, c.PayloadArity, module)
				}
				if err := reportStmtListMatchPayloadDiagnostics(c.Body, module); err != nil {
					return err
				}
			}
		case *frontend.UnsafeStmt:
			if err := reportStmtListMatchPayloadDiagnostics(s.Body, module); err != nil {
				return err
			}
		case *frontend.DeferStmt:
			if err := reportStmtListMatchPayloadDiagnostics(s.Body, module); err != nil {
				return err
			}
		}
		if err := reportExprMatchPayloadDiagnostics(stmtExpr(stmt), module); err != nil {
			return err
		}
	}
	return nil
}

func reportExprMatchPayloadDiagnostics(expr frontend.Expr, module string) error {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *frontend.MatchExpr:
		for i := range e.Cases {
			c := &e.Cases[i]
			if c.RequiresPayload {
				return payloadRequiredDiagnostic(c.At, c.Pattern, c.PayloadArity, module)
			}
			if err := reportExprMatchPayloadDiagnostics(c.Guard, module); err != nil {
				return err
			}
			if err := reportExprMatchPayloadDiagnostics(c.Value, module); err != nil {
				return err
			}
		}
		return reportExprMatchPayloadDiagnostics(e.Value, module)
	case *frontend.CatchExpr:
		for i := range e.Cases {
			c := &e.Cases[i]
			if c.RequiresPayload {
				return payloadRequiredDiagnostic(c.At, c.Pattern, c.PayloadArity, module)
			}
			if err := reportExprMatchPayloadDiagnostics(c.Guard, module); err != nil {
				return err
			}
			if err := reportExprMatchPayloadDiagnostics(c.Value, module); err != nil {
				return err
			}
		}
		return reportExprMatchPayloadDiagnostics(e.Call, module)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			if err := reportExprMatchPayloadDiagnostics(arg, module); err != nil {
				return err
			}
		}
	case *frontend.UnaryExpr:
		return reportExprMatchPayloadDiagnostics(e.X, module)
	case *frontend.BinaryExpr:
		if err := reportExprMatchPayloadDiagnostics(e.Left, module); err != nil {
			return err
		}
		return reportExprMatchPayloadDiagnostics(e.Right, module)
	case *frontend.FieldAccessExpr:
		return reportExprMatchPayloadDiagnostics(e.Base, module)
	case *frontend.IndexExpr:
		if err := reportExprMatchPayloadDiagnostics(e.Base, module); err != nil {
			return err
		}
		return reportExprMatchPayloadDiagnostics(e.Index, module)
	case *frontend.TryExpr:
		return reportExprMatchPayloadDiagnostics(e.X, module)
	case *frontend.AwaitExpr:
		return reportExprMatchPayloadDiagnostics(e.X, module)
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if err := reportExprMatchPayloadDiagnostics(field.Value, module); err != nil {
				return err
			}
		}
	case *frontend.ClosureExpr:
		if e.Decl != nil {
			return reportStmtListMatchPayloadDiagnostics(e.Decl.Body, module)
		}
	}
	return nil
}

func payloadRequiredDiagnostic(
	pos frontend.Position,
	pattern frontend.Expr,
	arity int,
	module string,
) error {
	if field, ok := pattern.(*frontend.FieldAccessExpr); ok {
		if arity <= 0 {
			arity = 1
		}
		return fmt.Errorf(
			"%s: enum case '%s.%s' carries %d payload value(s); use '%s.%s(%s)'",
			frontend.FormatPos(pos),
			displayTypeName(field.EnumType, module),
			field.Field,
			arity,
			displayTypeName(field.EnumType, module),
			field.Field,
			placeholderBindingList(arity),
		)
	}
	return fmt.Errorf("%s: enum payload pattern requires destructuring", frontend.FormatPos(pos))
}

func clearExprMatchPayloadMarkers(expr frontend.Expr) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *frontend.MatchExpr:
		for i := range e.Cases {
			e.Cases[i].RequiresPayload = false
			e.Cases[i].PayloadArity = 0
			clearExprMatchPayloadMarkers(e.Cases[i].Guard)
			clearExprMatchPayloadMarkers(e.Cases[i].Value)
		}
		clearExprMatchPayloadMarkers(e.Value)
	case *frontend.CatchExpr:
		for i := range e.Cases {
			e.Cases[i].RequiresPayload = false
			e.Cases[i].PayloadArity = 0
			clearExprMatchPayloadMarkers(e.Cases[i].Guard)
			clearExprMatchPayloadMarkers(e.Cases[i].Value)
		}
		clearExprMatchPayloadMarkers(e.Call)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			clearExprMatchPayloadMarkers(arg)
		}
	case *frontend.UnaryExpr:
		clearExprMatchPayloadMarkers(e.X)
	case *frontend.BinaryExpr:
		clearExprMatchPayloadMarkers(e.Left)
		clearExprMatchPayloadMarkers(e.Right)
	case *frontend.FieldAccessExpr:
		clearExprMatchPayloadMarkers(e.Base)
	case *frontend.IndexExpr:
		clearExprMatchPayloadMarkers(e.Base)
		clearExprMatchPayloadMarkers(e.Index)
	case *frontend.TryExpr:
		clearExprMatchPayloadMarkers(e.X)
	case *frontend.AwaitExpr:
		clearExprMatchPayloadMarkers(e.X)
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			clearExprMatchPayloadMarkers(field.Value)
		}
	case *frontend.ClosureExpr:
		if e.Decl != nil {
			clearStmtListMatchPayloadMarkers(e.Decl.Body)
		}
	}
}

func fileUsesExplicitPublic(file *frontend.FileAST) bool {
	if file == nil {
		return false
	}
	for _, imp := range file.Imports {
		if imp.Public {
			return true
		}
	}
	for _, en := range file.Enums {
		if en.Public {
			return true
		}
	}
	for _, st := range file.Structs {
		if st.Public {
			return true
		}
	}
	for _, st := range file.States {
		if st.Public {
			return true
		}
	}
	for _, view := range file.Views {
		if view.Public {
			return true
		}
	}
	for _, proto := range file.Protocols {
		if proto.Public {
			return true
		}
	}
	for _, glob := range file.Globals {
		if glob.Public {
			return true
		}
	}
	for _, fn := range file.Funcs {
		if fn.Public {
			return true
		}
	}
	return false
}

func declarationIsPublic(file *frontend.FileAST, public bool) bool {
	if !fileUsesExplicitPublic(file) {
		return true
	}
	return public
}

type globalConstValue struct {
	TypeName string
	I32      int32
	Bool     bool
}

func inferGlobalConstExprType(
	expr frontend.Expr,
	values map[string]globalConstValue,
) (string, bool) {
	if _, ok := evalGlobalConstBool(expr, values); ok {
		return "bool", true
	}
	if _, ok := evalGlobalConstI32(expr, values); ok {
		return "i32", true
	}
	return "", false
}

func validateGlobalConstExpr(expr frontend.Expr, values map[string]globalConstValue) error {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		if _, ok := values[e.Name]; !ok {
			return fmt.Errorf(
				"%s: unknown constant '%s' in global const expression",
				frontend.FormatPos(e.At),
				e.Name,
			)
		}
	case *frontend.UnaryExpr:
		if err := validateGlobalConstExpr(e.X, values); err != nil {
			return err
		}
		if _, ok, overflow := evalGlobalConstI32Wide(e, values); ok && overflow {
			return fmt.Errorf("%s: overflow in global const expression", frontend.FormatPos(e.At))
		}
	case *frontend.BinaryExpr:
		if err := validateGlobalConstExpr(e.Left, values); err != nil {
			return err
		}
		if err := validateGlobalConstExpr(e.Right, values); err != nil {
			return err
		}
		switch e.Op {
		case frontend.TokenSlash, frontend.TokenPercent:
			right, ok := evalGlobalConstI32(e.Right, values)
			if ok && right == 0 {
				if e.Op == frontend.TokenSlash {
					return fmt.Errorf("%s: division by zero in global const expression", frontend.FormatPos(e.At))
				}
				return fmt.Errorf("%s: modulo by zero in global const expression", frontend.FormatPos(e.At))
			}
		}
		if _, ok, overflow := evalGlobalConstI32Wide(e, values); ok && overflow {
			return fmt.Errorf("%s: overflow in global const expression", frontend.FormatPos(e.At))
		}
	}
	return nil
}

func evalGlobalConstI32(expr frontend.Expr, values map[string]globalConstValue) (int32, bool) {
	v, ok, overflow := evalGlobalConstI32Wide(expr, values)
	if !ok || overflow {
		return 0, false
	}
	return int32(v), true
}

func evalGlobalConstI32Wide(
	expr frontend.Expr,
	values map[string]globalConstValue,
) (int64, bool, bool) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return int64(e.Value), true, false
	case *frontend.IdentExpr:
		v, ok := values[e.Name]
		if !ok || !isGlobalIntLikeType(v.TypeName) {
			return 0, false, false
		}
		return int64(v.I32), true, false
	case *frontend.UnaryExpr:
		if e.Op != frontend.TokenMinus {
			return 0, false, false
		}
		v, ok, overflow := evalGlobalConstI32Wide(e.X, values)
		if !ok || overflow {
			return 0, ok, overflow
		}
		return checkedConstI32(-v)
	case *frontend.BinaryExpr:
		left, ok, overflow := evalGlobalConstI32Wide(e.Left, values)
		if !ok || overflow {
			return 0, ok, overflow
		}
		right, ok, overflow := evalGlobalConstI32Wide(e.Right, values)
		if !ok || overflow {
			return 0, ok, overflow
		}
		switch e.Op {
		case frontend.TokenPlus:
			return checkedConstI32(left + right)
		case frontend.TokenMinus:
			return checkedConstI32(left - right)
		case frontend.TokenStar:
			return checkedConstI32(left * right)
		case frontend.TokenSlash:
			if right == 0 {
				return 0, false, false
			}
			return checkedConstI32(left / right)
		case frontend.TokenPercent:
			if right == 0 {
				return 0, false, false
			}
			return checkedConstI32(left % right)
		default:
			return 0, false, false
		}
	default:
		return 0, false, false
	}
}

func isSupportedGlobalScalarType(name string) bool {
	switch name {
	case "i32", "bool", "ptr", "fnptr", "str", "u8", "u16", "c_int", "c_uint", "task.error":
		return true
	default:
		return isSupportedSliceGlobalType(name) || isSupportedOptionalPtrGlobalType(name) ||
			isSupportedOptionalSliceGlobalType(name)
	}
}

func isSupportedGlobalType(name string, types map[string]*TypeInfo) bool {
	if isSupportedGlobalScalarType(name) {
		return true
	}
	if isSupportedOptionalAggregateGlobalType(name, types) {
		return true
	}
	return isSupportedZeroedAggregateGlobalType(name, types, map[string]bool{})
}

func isSupportedOptionalAggregateGlobalType(name string, types map[string]*TypeInfo) bool {
	elem, ok := optionalElemName(name)
	if !ok {
		return false
	}
	if isSupportedGlobalScalarType(elem) {
		return false
	}
	return isSupportedZeroedAggregateGlobalType(elem, types, map[string]bool{})
}

func isSupportedZeroedAggregateGlobalType(
	name string,
	types map[string]*TypeInfo,
	visiting map[string]bool,
) bool {
	if _, elem, ok := parseArrayTypeName(name); ok {
		return isSupportedGlobalScalarType(elem)
	}
	if visiting[name] {
		return false
	}
	info, ok := types[name]
	if !ok {
		return false
	}
	visiting[name] = true
	defer delete(visiting, name)
	switch info.Kind {
	case TypeArray:
		return isSupportedGlobalScalarType(info.ElemType)
	case TypeStruct:
		for _, field := range info.Fields {
			if field.FunctionTypeValue {
				return false
			}
			if isSupportedGlobalScalarType(field.TypeName) {
				continue
			}
			if !isSupportedZeroedAggregateGlobalType(field.TypeName, types, visiting) {
				return false
			}
		}
		return true
	case TypeEnum:
		for _, enumCase := range info.EnumCases {
			for i, payloadType := range enumCase.PayloadTypes {
				if i < len(enumCase.PayloadFunctionTypes) && enumCase.PayloadFunctionTypes[i] {
					return false
				}
				if isSupportedGlobalScalarType(payloadType) {
					continue
				}
				if !isSupportedZeroedAggregateGlobalType(payloadType, types, visiting) {
					return false
				}
			}
		}
		return true
	default:
		return false
	}
}

func collectGlobalArrayBackings(
	typeName string,
	offset int,
	types map[string]*TypeInfo,
	visiting map[string]bool,
) []GlobalArrayBackingInfo {
	info, ok := types[typeName]
	if !ok || info == nil {
		return nil
	}
	switch info.Kind {
	case TypeArray:
		return []GlobalArrayBackingInfo{{
			HeaderOffset: offset,
			ElemType:     info.ElemType,
			Len:          info.ArrayLen,
		}}
	case TypeStruct:
		if visiting[typeName] {
			return nil
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		var out []GlobalArrayBackingInfo
		for _, field := range info.Fields {
			out = append(
				out,
				collectGlobalArrayBackings(field.TypeName, offset+field.Offset, types, visiting)...)
		}
		return out
	default:
		return nil
	}
}

func isSupportedOptionalSliceGlobalType(name string) bool {
	elem, ok := optionalElemName(name)
	if !ok {
		return false
	}
	sliceElem, ok := sliceElemName(elem)
	if !ok {
		return false
	}
	return isSupportedGlobalSliceElemType(sliceElem)
}

func isSupportedOptionalPtrGlobalType(name string) bool {
	elem, ok := optionalElemName(name)
	return ok && elem == "ptr"
}

func isSupportedSliceGlobalType(name string) bool {
	sliceElem, ok := sliceElemName(name)
	if !ok {
		return false
	}
	return isSupportedGlobalSliceElemType(sliceElem)
}

func isSupportedGlobalSliceElemType(sliceElem string) bool {
	switch sliceElem {
	case "i32", "u8", "u16", "c_int", "c_uint", "bool",
		"usize", "isize", "size_t", "ssize_t", "native_int", "native_uint", "c_long", "c_ulong":
		return true
	default:
		return false
	}
}

func isGlobalIntLikeType(name string) bool {
	switch name {
	case "i32", "u8", "u16", "c_int", "c_uint", "task.error",
		"usize", "isize", "size_t", "ssize_t", "native_int", "native_uint", "c_long", "c_ulong":
		return true
	default:
		return false
	}
}

func validateGlobalIntLikeRange(
	typeName, globalName, kind string,
	pos frontend.Position,
	v int32,
) error {
	switch typeName {
	case "u8":
		if v < 0 || v > 255 {
			return fmt.Errorf(
				"%s: global %s '%s' initializer must be within 0..255 for type u8",
				frontend.FormatPos(pos),
				kind,
				globalName,
			)
		}
	case "u16":
		if v < 0 || v > 65535 {
			return fmt.Errorf(
				"%s: global %s '%s' initializer must be within 0..65535 for type u16",
				frontend.FormatPos(pos),
				kind,
				globalName,
			)
		}
	case "c_uint":
		if v < 0 {
			return fmt.Errorf(
				"%s: global %s '%s' initializer must be non-negative for type c_uint",
				frontend.FormatPos(pos),
				kind,
				globalName,
			)
		}
	case "usize", "size_t", "native_uint", "c_ulong":
		if v < 0 {
			return fmt.Errorf(
				"%s: global %s '%s' initializer must be non-negative for type %s",
				frontend.FormatPos(pos),
				kind,
				globalName,
				typeName,
			)
		}
	}
	return nil
}

func evalGlobalConstBool(expr frontend.Expr, values map[string]globalConstValue) (bool, bool) {
	switch e := expr.(type) {
	case *frontend.BoolLitExpr:
		return e.Value, true
	case *frontend.IdentExpr:
		v, ok := values[e.Name]
		if !ok || v.TypeName != "bool" {
			return false, false
		}
		return v.Bool, true
	case *frontend.UnaryExpr:
		if e.Op != frontend.TokenBang {
			return false, false
		}
		v, ok := evalGlobalConstBool(e.X, values)
		if !ok {
			return false, false
		}
		return !v, true
	case *frontend.BinaryExpr:
		switch e.Op {
		case frontend.TokenAmpAmp:
			left, ok := evalGlobalConstBool(e.Left, values)
			if !ok {
				return false, false
			}
			right, ok := evalGlobalConstBool(e.Right, values)
			if !ok {
				return false, false
			}
			return left && right, true
		case frontend.TokenPipePipe:
			left, ok := evalGlobalConstBool(e.Left, values)
			if !ok {
				return false, false
			}
			right, ok := evalGlobalConstBool(e.Right, values)
			if !ok {
				return false, false
			}
			return left || right, true
		case frontend.TokenEqEq, frontend.TokenBangEq:
			if left, ok := evalGlobalConstI32(e.Left, values); ok {
				right, ok := evalGlobalConstI32(e.Right, values)
				if !ok {
					return false, false
				}
				if e.Op == frontend.TokenEqEq {
					return left == right, true
				}
				return left != right, true
			}
			left, ok := evalGlobalConstBool(e.Left, values)
			if !ok {
				return false, false
			}
			right, ok := evalGlobalConstBool(e.Right, values)
			if !ok {
				return false, false
			}
			if e.Op == frontend.TokenEqEq {
				return left == right, true
			}
			return left != right, true
		case frontend.TokenLess, frontend.TokenLessEq, frontend.TokenGreater, frontend.TokenGreaterEq:
			left, ok := evalGlobalConstI32(e.Left, values)
			if !ok {
				return false, false
			}
			right, ok := evalGlobalConstI32(e.Right, values)
			if !ok {
				return false, false
			}
			switch e.Op {
			case frontend.TokenLess:
				return left < right, true
			case frontend.TokenLessEq:
				return left <= right, true
			case frontend.TokenGreater:
				return left > right, true
			case frontend.TokenGreaterEq:
				return left >= right, true
			}
		}
	}
	return false, false
}

func evaluateActorStateInitializer(expr frontend.Expr) (string, int32, bool) {
	if v, ok := evalGlobalConstI32(expr, nil); ok {
		return "i32", v, true
	}
	if v, ok := evalGlobalConstBool(expr, nil); ok {
		if v {
			return "bool", 1, true
		}
		return "bool", 0, true
	}
	return "", 0, false
}

// ---- checker_function_types_a.go ----

func resolveCheckedCallName(
	name string,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
	pos frontend.Position,
) (string, error) {
	if builtin, ok := ResolveBuiltinAlias(name); ok {
		return builtin, nil
	}
	return resolveKnownCallName(name, funcs, module, imports, pos)
}

func paramDeclOwnership(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		out = append(out, param.Ownership)
	}
	return out
}

func validateFunctionTypeNamedSymbolBinding(
	name string,
	declared frontend.TypeRef,
	init *frontend.IdentExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	allowCapturedAlias bool,
	genericErrorOverride ...func(frontend.Position, string, string) error,
) (string, error) {
	genericError := unsupportedGenericFunctionTypedLocalInitializerError
	if len(genericErrorOverride) > 0 && genericErrorOverride[0] != nil {
		genericError = genericErrorOverride[0]
	}
	if declared.Kind != frontend.TypeRefFunction {
		return "", nil
	}
	if init == nil {
		return "", unsupportedFunctionTypedLocalInitializerSourceError(declared.At, name)
	}
	if localInfo, ok := locals[init.Name]; ok {
		if !localInfo.FunctionTypeValue && localInfo.FunctionValue != "" {
			return validateFunctionTypeClosurePointerBinding(
				name,
				declared,
				init,
				localInfo,
				funcs,
				types,
				module,
				imports,
				genericError,
			)
		}
		if len(localInfo.FunctionCaptures) > 0 {
			if !allowCapturedAlias {
				return "", unsupportedFunctionTypedCaptureAliasError(init.At, name, init.Name)
			}
			captureSlots, err := functionCaptureSlotCount(localInfo.FunctionCaptures, types)
			if err != nil {
				return "", err
			}
			if captureSlots < 1 {
				return "", unsupportedFunctionTypedStorageCaptureError(init.At, name, captureSlots)
			}
			if captureSlots > FnPtrEnvSlotCount && !localInfo.FunctionHandleValue {
				if _, _, err := classifyCallableEscape(
					callableBoundaryLocal,
					localInfo.FunctionCaptures,
					types,
				); err != nil {
					return "", err
				}
			}
			if !localInfo.FunctionTypeValue {
				return "", unsupportedFunctionTypedCaptureAliasError(init.At, name, init.Name)
			}
		}
		if !localInfo.FunctionTypeValue {
			return "", unsupportedFunctionTypedLocalInitializerSourceError(init.At, name)
		}
		if localInfo.FunctionValue == "" {
			validationSig, err := buildInterfaceFuncSig(name, funcSigSpec{
				ParamTypes:          append([]string(nil), localInfo.FunctionParamTypes...),
				ParamOwnership:      append([]string(nil), localInfo.FunctionParamOwnership...),
				ReturnType:          localInfo.FunctionReturnType,
				ReturnOwnership:     localInfo.FunctionReturnOwnership,
				ThrowsType:          localInfo.FunctionThrowsType,
				ReturnRegionParam:   regionNone,
				ReturnResourceParam: regionNone,
				Effects:             append([]string(nil), localInfo.FunctionEffects...),
			}, types)
			if err != nil {
				return "", err
			}
			if err := validateFunctionTypeSymbolSignature(
				name,
				declared,
				validationSig,
				module,
				imports,
				init.At,
			); err != nil {
				return "", err
			}
			return "", nil
		}
		sig, ok := funcs[localInfo.FunctionValue]
		if !ok {
			return "", fmt.Errorf(
				"%s: unknown function symbol '%s'",
				frontend.FormatPos(init.At),
				localInfo.FunctionValue,
			)
		}
		if localInfo.GenericFunctionValue || sig.Generic {
			return "", genericError(init.At, init.Name, name)
		}
		if sig.ThrowsType != "" && localInfo.FunctionReturnType != "" && declared.Throws == nil {
			return "", unsupportedThrowingFunctionTypedLocalInitializerError(
				init.At,
				init.Name,
				name,
			)
		}
		validationSig := sig
		if localInfo.FunctionReturnType != "" {
			explicitSlots, err := functionParamSlotCount(localInfo.FunctionParamTypes, types)
			if err != nil {
				return "", err
			}
			hiddenSlots := sig.ParamSlots - explicitSlots
			if hiddenSlots < 0 ||
				(hiddenSlots > FnPtrEnvSlotCount && !localInfo.FunctionHandleValue) {
				return "", unsupportedFunctionTypedCaptureAliasError(init.At, name, init.Name)
			}
			if hiddenSlots > 0 && !allowCapturedAlias {
				return "", unsupportedFunctionTypedCaptureAliasError(init.At, name, init.Name)
			}
			validationSig.ParamTypes = append([]string(nil), localInfo.FunctionParamTypes...)
			validationSig.ParamOwnership = append(
				[]string(nil),
				localInfo.FunctionParamOwnership...)
			validationSig.ReturnType = localInfo.FunctionReturnType
			validationSig.ReturnOwnership = localInfo.FunctionReturnOwnership
		}
		if err := validateFunctionTypeSymbolSignature(
			name,
			declared,
			validationSig,
			module,
			imports,
			init.At,
		); err != nil {
			return "", err
		}
		if localInfo.Mutable {
			return "", nil
		}
		return localInfo.FunctionValue, nil
	}
	if globalInfo, ok := globals[init.Name]; ok {
		if !globalInfo.FunctionTypeValue || globalInfo.FunctionValue == "" {
			return "", unsupportedFunctionTypedLocalInitializerSourceError(init.At, name)
		}
		sig, ok := funcs[globalInfo.FunctionValue]
		if !ok {
			return "", fmt.Errorf(
				"%s: unknown function symbol '%s'",
				frontend.FormatPos(init.At),
				globalInfo.FunctionValue,
			)
		}
		if sig.Generic {
			return "", genericError(init.At, init.Name, name)
		}
		if sig.ThrowsType != "" && declared.Throws == nil {
			return "", unsupportedThrowingFunctionTypedLocalInitializerError(
				init.At,
				init.Name,
				name,
			)
		}
		if err := validateFunctionTypeSymbolSignature(
			name,
			declared,
			sig,
			module,
			imports,
			init.At,
		); err != nil {
			return "", err
		}
		if globalInfo.Mutable {
			return "", nil
		}
		return globalInfo.FunctionValue, nil
	}
	resolved, err := resolveCheckedCallName(init.Name, funcs, module, imports, init.At)
	if err != nil {
		return "", unsupportedFunctionTypedLocalInitializerSourceError(init.At, name)
	}
	sig, ok := funcs[resolved]
	if !ok {
		return "", unsupportedFunctionTypedLocalInitializerSourceError(init.At, name)
	}
	if err := ensureFuncVisible(resolved, sig, module, init.At); err != nil {
		return "", err
	}
	if sig.Generic {
		return "", genericError(init.At, init.Name, name)
	}
	if sig.ThrowsType != "" && declared.Throws == nil {
		return "", unsupportedThrowingFunctionTypedLocalInitializerError(init.At, init.Name, name)
	}
	if err := validateFunctionTypeSymbolSignature(
		name,
		declared,
		sig,
		module,
		imports,
		init.At,
	); err != nil {
		return "", err
	}
	return resolved, nil
}

func validateFunctionTypeClosurePointerBinding(
	name string,
	declared frontend.TypeRef,
	init *frontend.IdentExpr,
	localInfo LocalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	genericError func(frontend.Position, string, string) error,
) (string, error) {
	if genericError == nil {
		genericError = unsupportedGenericFunctionTypedLocalInitializerError
	}
	sig, ok := funcs[localInfo.FunctionValue]
	if !ok {
		return "", fmt.Errorf(
			"%s: unknown function symbol '%s'",
			frontend.FormatPos(init.At),
			localInfo.FunctionValue,
		)
	}
	if localInfo.GenericFunctionValue || sig.Generic {
		return "", genericError(init.At, init.Name, name)
	}
	if sig.ThrowsType != "" {
		return "", unsupportedThrowingFunctionTypedLocalInitializerError(init.At, init.Name, name)
	}
	visibleSig := sig
	if len(localInfo.FunctionCaptures) > 0 {
		captureSlots, err := functionCaptureSlotCount(localInfo.FunctionCaptures, types)
		if err != nil {
			return "", err
		}
		if captureSlots < 1 {
			return "", unsupportedFunctionTypedStorageCaptureError(init.At, name, captureSlots)
		}
		if captureSlots > FnPtrEnvSlotCount {
			if _, _, err := classifyCallableEscape(
				callableBoundaryLocal,
				localInfo.FunctionCaptures,
				types,
			); err != nil {
				return "", err
			}
		}
		paramTypes, returnType, _, err := functionTypeRefSignatureAndEffects(
			declared,
			module,
			imports,
		)
		if err != nil {
			return "", err
		}
		explicitSlots, err := functionParamSlotCount(paramTypes, types)
		if err != nil {
			return "", err
		}
		if sig.ParamSlots-explicitSlots != captureSlots {
			return "", unsupportedFunctionTypedCaptureAliasError(init.At, name, init.Name)
		}
		visibleSig.ParamTypes = paramTypes
		visibleSig.ParamOwnership = functionTypeRefParamOwnership(declared)
		visibleSig.ParamSlots = explicitSlots
		visibleSig.ReturnType = returnType
	}
	if err := validateFunctionTypeSymbolSignature(
		name,
		declared,
		visibleSig,
		module,
		imports,
		init.At,
	); err != nil {
		return "", err
	}
	return localInfo.FunctionValue, nil
}

func validateFunctionTypeClosurePointerAssignment(
	targetName string,
	targetInfo LocalInfo,
	init *frontend.IdentExpr,
	sourceInfo LocalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	boundary callableEscapeBoundary,
) (string, error) {
	sig, ok := funcs[sourceInfo.FunctionValue]
	if !ok {
		return "", fmt.Errorf(
			"%s: unknown function symbol '%s'",
			frontend.FormatPos(init.At),
			sourceInfo.FunctionValue,
		)
	}
	if sourceInfo.GenericFunctionValue || sig.Generic {
		return "", unsupportedGenericFunctionTypedAssignmentError(init.At, init.Name, targetName)
	}
	if sig.ThrowsType != "" {
		return "", unsupportedThrowingFunctionTypedAssignmentError(init.At, init.Name, targetName)
	}
	visibleSig := sig
	if len(sourceInfo.FunctionCaptures) > 0 {
		captureSlots, err := functionCaptureSlotCount(sourceInfo.FunctionCaptures, types)
		if err != nil {
			return "", err
		}
		if captureSlots < 1 {
			return "", unsupportedFunctionTypedStorageCaptureError(
				init.At,
				targetName,
				captureSlots,
			)
		}
		if captureSlots > FnPtrEnvSlotCount {
			if _, _, err := classifyCallableEscape(
				boundary,
				sourceInfo.FunctionCaptures,
				types,
			); err != nil {
				return "", err
			}
		}
		explicitSlots, err := functionParamSlotCount(targetInfo.FunctionParamTypes, types)
		if err != nil {
			return "", err
		}
		if sig.ParamSlots-explicitSlots != captureSlots {
			return "", unsupportedFunctionTypedCaptureAliasError(init.At, targetName, init.Name)
		}
		visibleSig.ParamTypes = append([]string(nil), targetInfo.FunctionParamTypes...)
		visibleSig.ParamOwnership = append([]string(nil), targetInfo.FunctionParamOwnership...)
		visibleSig.ParamSlots = explicitSlots
		visibleSig.ReturnType = targetInfo.FunctionReturnType
		visibleSig.ReturnOwnership = targetInfo.FunctionReturnOwnership
	}
	if err := validateFunctionInfoAssignable(targetName, targetInfo, visibleSig, init.At); err != nil {
		return "", err
	}
	return sourceInfo.FunctionValue, nil
}

func validateFunctionTypedAssignmentValue(
	targetName string,
	targetInfo LocalInfo,
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	pos frontend.Position,
	allowCapturedLocalStorage bool,
	boundary callableEscapeBoundary,
) error {
	switch v := value.(type) {
	case *frontend.ClosureExpr:
		if err := validateFunctionTypedClosureAssignment(
			targetName,
			targetInfo,
			v,
			locals,
			funcs,
			types,
			module,
			imports,
			pos,
		); err != nil {
			return err
		}
		if allowCapturedLocalStorage && len(v.Captures) > 0 {
			captureSlots, err := functionCaptureSlotCount(v.Captures, types)
			if err != nil {
				return err
			}
			if captureSlots > FnPtrEnvSlotCount {
				if _, _, err := classifyCallableEscape(boundary, v.Captures, types); err != nil {
					return err
				}
				return nil
			}
		}
		if !allowCapturedLocalStorage && len(v.Captures) > 0 {
			return unsupportedGlobalFunctionCaptureStorageError(v.At, targetName)
		}
		return nil
	case *frontend.CallExpr:
		return validateFunctionTypedReturnCallAssignment(
			targetName,
			targetInfo,
			v,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			allowCapturedLocalStorage,
		)
	case *frontend.FieldAccessExpr:
		fieldInfo, ok, err := resolveFunctionFieldArgument(v, locals)
		if err != nil {
			return err
		}
		if ok && !allowCapturedLocalStorage && (len(
			fieldInfo.FunctionCaptures,
		) > 0 || len(
			fieldInfo.FunctionEscapeCaptures,
		) > 0) {
			return unsupportedGlobalFunctionCaptureStorageError(v.At, targetName)
		}
		if ok && !allowCapturedLocalStorage && fieldInfo.FunctionValue == "" && functionFieldInfoHasTargetSet(
			fieldInfo,
		) {
			if fieldInfo.FunctionParamName != "" {
				return unsupportedGlobalFunctionParameterStorageError(
					v.At,
					targetName,
					fieldInfo.FunctionParamName,
				)
			}
			return unsupportedGlobalFunctionCaptureStorageError(v.At, targetName)
		}
		if ok && fieldInfo.FunctionValue == "" && functionFieldInfoHasTargetSet(
			fieldInfo,
		) && allowCapturedLocalStorage {
			fieldSig, err := functionFieldInfoSig(fieldInfo)
			if err != nil {
				return err
			}
			return validateFunctionInfoAssignable(
				targetName,
				targetInfo,
				fieldSig,
				v.At,
			)
		}
		if !ok || fieldInfo.FunctionValue == "" {
			if _, globalSig, globalOK, err := resolveFunctionTypedGlobalFieldAccess(
				v,
				globals,
				funcs,
			); err != nil {
				return err
			} else if globalOK {
				return validateFunctionInfoAssignable(targetName, targetInfo, globalSig, v.At)
			}
			return unsupportedFunctionTypedAssignmentSourceError(v.At, targetName)
		}
		fieldSig, err := functionFieldInfoSig(fieldInfo)
		if err != nil {
			return err
		}
		return validateFunctionInfoAssignable(targetName, targetInfo, fieldSig, v.At)
	}
	id, ok := value.(*frontend.IdentExpr)
	if !ok {
		return unsupportedFunctionTypedAssignmentSourceError(value.Pos(), targetName)
	}
	if sourceInfo, ok := locals[id.Name]; ok {
		if !allowCapturedLocalStorage &&
			(len(sourceInfo.FunctionCaptures) > 0 || len(sourceInfo.FunctionEscapeCaptures) > 0) {
			return unsupportedGlobalFunctionCaptureStorageError(id.At, targetName)
		}
		if !allowCapturedLocalStorage && sourceInfo.FunctionTypeValue &&
			sourceInfo.FunctionValue == "" {
			paramName := sourceInfo.FunctionParamName
			if paramName == "" {
				paramName = id.Name
			}
			return unsupportedGlobalFunctionParameterStorageError(id.At, targetName, paramName)
		}
		if !sourceInfo.FunctionTypeValue && sourceInfo.FunctionValue != "" {
			if _, err := validateFunctionTypeClosurePointerAssignment(
				targetName,
				targetInfo,
				id,
				sourceInfo,
				funcs,
				types,
				boundary,
			); err != nil {
				return err
			}
			return nil
		}
		if sourceInfo.FunctionTypeValue && sourceInfo.FunctionValue == "" &&
			allowCapturedLocalStorage {
			sourceSig, err := buildInterfaceFuncSig(targetName, funcSigSpec{
				ParamTypes:          append([]string(nil), sourceInfo.FunctionParamTypes...),
				ParamOwnership:      append([]string(nil), sourceInfo.FunctionParamOwnership...),
				ReturnType:          sourceInfo.FunctionReturnType,
				ReturnOwnership:     sourceInfo.FunctionReturnOwnership,
				ThrowsType:          sourceInfo.FunctionThrowsType,
				ReturnRegionParam:   regionNone,
				ReturnResourceParam: regionNone,
				Effects:             append([]string(nil), sourceInfo.FunctionEffects...),
			}, types)
			if err != nil {
				return err
			}
			return validateFunctionInfoAssignable(targetName, targetInfo, sourceSig, id.At)
		}
		if !sourceInfo.FunctionTypeValue || sourceInfo.FunctionValue == "" {
			return unsupportedFunctionTypedAssignmentSourceError(id.At, targetName)
		}
		sourceSig, err := buildInterfaceFuncSig(targetName, funcSigSpec{
			ParamTypes:          append([]string(nil), sourceInfo.FunctionParamTypes...),
			ParamOwnership:      append([]string(nil), sourceInfo.FunctionParamOwnership...),
			ReturnType:          sourceInfo.FunctionReturnType,
			ReturnOwnership:     sourceInfo.FunctionReturnOwnership,
			ThrowsType:          sourceInfo.FunctionThrowsType,
			ReturnRegionParam:   regionNone,
			ReturnResourceParam: regionNone,
			Effects:             append([]string(nil), sourceInfo.FunctionEffects...),
		}, types)
		if err != nil {
			return err
		}
		return validateFunctionInfoAssignable(targetName, targetInfo, sourceSig, id.At)
	}
	if globalInfo, ok := globals[id.Name]; ok {
		if !globalInfo.FunctionTypeValue || globalInfo.FunctionValue == "" {
			return unsupportedFunctionTypedAssignmentSourceError(id.At, targetName)
		}
		sig, ok := funcs[globalInfo.FunctionValue]
		if !ok {
			return fmt.Errorf(
				"%s: unknown function symbol '%s'",
				frontend.FormatPos(id.At),
				globalInfo.FunctionValue,
			)
		}
		if sig.Generic {
			return unsupportedGenericFunctionTypedAssignmentError(pos, id.Name, targetName)
		}
		return validateFunctionInfoAssignable(targetName, targetInfo, sig, id.At)
	}
	resolved, err := resolveCheckedCallName(id.Name, funcs, module, imports, id.At)
	if err != nil {
		return unsupportedFunctionTypedAssignmentSourceError(id.At, targetName)
	}
	sig, ok := funcs[resolved]
	if !ok {
		return unsupportedFunctionTypedAssignmentSourceError(id.At, targetName)
	}
	if err := ensureFuncVisible(resolved, sig, module, id.At); err != nil {
		return err
	}
	if sig.Generic {
		return unsupportedGenericFunctionTypedAssignmentError(pos, id.Name, targetName)
	}
	if err := validateFunctionInfoAssignable(targetName, targetInfo, sig, id.At); err != nil {
		return err
	}
	id.Name = resolved
	return nil
}

func allowCapturedGlobalFunctionSnapshot(
	value frontend.Expr,
	locals map[string]LocalInfo,
	types map[string]*TypeInfo,
	state *regionState,
) (bool, error) {
	if closure, ok := value.(*frontend.ClosureExpr); ok {
		if err := rejectMutableGlobalFunctionCaptures(closure.Captures, locals); err != nil {
			return false, err
		}
		if err := rejectBorrowedFunctionCaptures(closure.Captures, state); err != nil {
			return false, err
		}
		if _, _, err := classifyCallableEscape(
			callableBoundaryGlobal,
			closure.Captures,
			types,
		); err != nil {
			return false, err
		}
		return true, nil
	}
	if field, ok := value.(*frontend.FieldAccessExpr); ok {
		fieldInfo, found, err := resolveFunctionFieldArgument(field, locals)
		if err != nil || !found {
			return false, err
		}
		return allowFunctionFieldGlobalSnapshot(fieldInfo, value.Pos(), locals, types, state)
	}
	id, ok := value.(*frontend.IdentExpr)
	if !ok {
		return false, nil
	}
	source, ok := locals[id.Name]
	if !ok {
		return false, nil
	}
	if source.FunctionParamName != "" || source.FunctionValue == "" {
		return false, nil
	}
	if !source.FunctionTypeValue &&
		len(source.FunctionCaptures) > 0 &&
		len(source.FunctionEscapeCaptures) == 0 {
		if err := rejectMutableGlobalFunctionCaptures(source.FunctionCaptures, locals); err != nil {
			return false, err
		}
		if err := rejectBorrowedFunctionCaptures(source.FunctionCaptures, state); err != nil {
			return false, err
		}
		captureSlots, err := functionCaptureSlotCount(source.FunctionCaptures, types)
		if err != nil {
			return false, err
		}
		if captureSlots < 1 {
			return false, nil
		}
		if captureSlots > FnPtrEnvSlotCount {
			if _, _, err := classifyCallableEscape(
				callableBoundaryGlobal,
				source.FunctionCaptures,
				types,
			); err != nil {
				return false, err
			}
		}
		return true, nil
	}
	if source.FunctionTypeValue &&
		source.FunctionDirectSnapshotAlias &&
		len(source.FunctionCaptures) > 0 &&
		len(source.FunctionEscapeCaptures) == 0 {
		if err := rejectMutableGlobalFunctionCaptures(source.FunctionCaptures, locals); err != nil {
			return false, err
		}
		if err := rejectBorrowedFunctionCaptures(source.FunctionCaptures, state); err != nil {
			return false, err
		}
		captureSlots, err := functionCaptureSlotCount(source.FunctionCaptures, types)
		if err != nil {
			return false, err
		}
		if captureSlots < 1 {
			return false, nil
		}
		if captureSlots > FnPtrEnvSlotCount {
			if _, _, err := classifyCallableEscape(
				callableBoundaryGlobal,
				source.FunctionCaptures,
				types,
			); err != nil {
				return false, err
			}
		}
		return true, nil
	}
	if source.FunctionTypeValue &&
		source.FunctionReturnSnapshotAlias &&
		len(source.FunctionCaptures) == 0 &&
		len(source.FunctionEscapeCaptures) > 0 {
		if err := rejectMutableGlobalFunctionCaptures(source.FunctionEscapeCaptures, locals); err != nil {
			return false, err
		}
		if err := rejectBorrowedFunctionCaptures(source.FunctionEscapeCaptures, state); err != nil {
			return false, err
		}
		captureSlots, err := functionCaptureSlotCount(source.FunctionEscapeCaptures, types)
		if err != nil {
			return false, err
		}
		if captureSlots < 1 {
			return false, nil
		}
		if captureSlots > FnPtrEnvSlotCount {
			if _, _, err := classifyCallableEscape(
				callableBoundaryGlobal,
				source.FunctionEscapeCaptures,
				types,
			); err != nil {
				return false, err
			}
		}
		return true, nil
	}
	return false, nil
}

func rejectBorrowedFunctionCaptures(captures []frontend.ClosureCapture, state *regionState) error {
	if state == nil {
		return nil
	}
	for _, capture := range captures {
		for path, regionID := range state.regionVars {
			if path != capture.Name && !ownershipPathPrefix(capture.Name, path) {
				continue
			}
			if _, borrowed := state.borrowedParamOwner(regionID); !borrowed {
				continue
			}
			name := path
			if name == "" {
				name = capture.Name
			}
			return lifetimeDiagnosticf(
				capture.At,
				"borrowed local '%s' cannot escape via function capture",
				name,
			)
		}
	}
	return nil
}

func rejectMutableGlobalFunctionCaptures(
	captures []frontend.ClosureCapture,
	locals map[string]LocalInfo,
) error {
	for _, capture := range captures {
		if capture.Mutable {
			return unsupportedGlobalFunctionMutableCaptureStorageError(capture.At, capture.Name)
		}
		if local, ok := locals[capture.Name]; ok && local.Mutable {
			return unsupportedGlobalFunctionMutableCaptureStorageError(capture.At, capture.Name)
		}
	}
	return nil
}

func allowFunctionFieldGlobalSnapshot(
	info FunctionFieldInfo,
	pos frontend.Position,
	locals map[string]LocalInfo,
	types map[string]*TypeInfo,
	state *regionState,
) (bool, error) {
	if info.FunctionParamName != "" || info.FunctionValue == "" {
		return false, nil
	}
	captures := info.FunctionEscapeCaptures
	if !info.FunctionReturnSnapshotAlias && info.FunctionDirectSnapshotAlias {
		captures = info.FunctionCaptures
	}
	if !info.FunctionReturnSnapshotAlias && !info.FunctionDirectSnapshotAlias {
		return false, nil
	}
	if len(captures) == 0 {
		return false, nil
	}
	if info.FunctionReturnSnapshotAlias && len(info.FunctionCaptures) != 0 {
		return false, nil
	}
	if !info.FunctionReturnSnapshotAlias && len(info.FunctionEscapeCaptures) != 0 {
		return false, nil
	}
	if err := rejectMutableGlobalFunctionCaptures(captures, locals); err != nil {
		return false, err
	}
	if err := rejectBorrowedFunctionCaptures(captures, state); err != nil {
		return false, err
	}
	captureSlots, err := functionCaptureSlotCount(captures, types)
	if err != nil {
		return false, err
	}
	if captureSlots < 1 {
		return false, nil
	}
	if captureSlots > FnPtrEnvSlotCount {
		if _, _, err := classifyCallableEscape(callableBoundaryGlobal, captures, types); err != nil {
			return false, err
		}
	}
	return true, nil
}

func unsupportedGlobalFunctionCaptureStorageError(pos frontend.Position, targetName string) error {
	return lifetimeDiagnosticf(
		pos,
		("captured function value cannot be stored in global " +
			"function-typed value '%s'; global escape requires a direct " +
			"fnptr snapshot with known captures and bounded environment " +
			"slots"),
		targetName,
	)
}

func unsupportedGlobalFunctionMutableCaptureStorageError(
	pos frontend.Position,
	captureName string,
) error {
	return lifetimeDiagnosticf(
		pos,
		("global-escaped function value captures mutable local '%s'; " +
			"mutable by-reference captures require a proven lifetime and " +
			"synchronization model"),
		captureName,
	)
}

func unsupportedGlobalFunctionParameterStorageError(
	pos frontend.Position,
	targetName, paramName string,
) error {
	return lifetimeDiagnosticf(
		pos,
		("function-typed parameter '%s' cannot be stored in global " +
			"function-typed value '%s'; global escape requires a direct " +
			"fnptr snapshot with known captures and bounded environment " +
			"slots"),
		paramName,
		targetName,
	)
}

func functionParamNameForParam(name string, functionTypeValue bool) string {
	if functionTypeValue {
		return name
	}
	return ""
}

func functionAssignmentMetadata(
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
) (string, []frontend.ClosureCapture, []frontend.ClosureCapture, string) {
	switch v := value.(type) {
	case *frontend.ClosureExpr:
		return closureFunctionValueName(
				v,
				funcs,
				module,
			), append(
				[]frontend.ClosureCapture(nil),
				v.Captures...,
			), nil, ""
	case *frontend.CallExpr:
		if callSig, ok := funcs[v.Name]; ok {
			return callSig.ReturnFunctionSymbol, nil, append(
				[]frontend.ClosureCapture(nil),
				callSig.ReturnFunctionCaptures...,
			), callSig.ReturnFunctionParamName
		}
	case *frontend.FieldAccessExpr:
		if fieldInfo, ok, err := resolveFunctionFieldArgument(v, locals); err == nil && ok {
			return fieldInfo.FunctionValue, append(
					[]frontend.ClosureCapture(nil),
					fieldInfo.FunctionCaptures...,
				), append(
					[]frontend.ClosureCapture(nil),
					fieldInfo.FunctionEscapeCaptures...,
				), fieldInfo.FunctionParamName
		}
		if globalInfo, _, ok, err := resolveFunctionTypedGlobalFieldAccess(
			v,
			globals,
			funcs,
		); err == nil && ok {
			return globalInfo.FunctionValue, nil, nil, ""
		}
	case *frontend.IdentExpr:
		if sourceInfo, ok := locals[v.Name]; ok {
			paramName := sourceInfo.FunctionParamName
			if paramName == "" && sourceInfo.FunctionTypeValue && sourceInfo.FunctionValue == "" {
				paramName = v.Name
			}
			return sourceInfo.FunctionValue, append(
					[]frontend.ClosureCapture(nil),
					sourceInfo.FunctionCaptures...,
				), append(
					[]frontend.ClosureCapture(nil),
					sourceInfo.FunctionEscapeCaptures...,
				), paramName
		}
		if globalInfo, ok := globals[v.Name]; ok {
			return globalInfo.FunctionValue, nil, nil, ""
		}
		if resolved, err := resolveCheckedCallName(v.Name, funcs, module, imports, v.At); err == nil {
			return resolved, nil, nil, ""
		}
	}
	return "", nil, nil, ""
}

func functionDirectSnapshotAliasForExpr(value frontend.Expr, locals map[string]LocalInfo) bool {
	switch v := value.(type) {
	case *frontend.ClosureExpr:
		return len(v.Captures) > 0
	case *frontend.IdentExpr:
		source, ok := locals[v.Name]
		return ok && source.FunctionDirectSnapshotAlias
	case *frontend.FieldAccessExpr:
		fieldInfo, ok, err := resolveFunctionFieldArgument(v, locals)
		return err == nil && ok && fieldInfo.FunctionDirectSnapshotAlias
	default:
		return false
	}
}

func functionSymbolTouchesMutableGlobals(name string, funcs map[string]FuncSig) bool {
	if name == "" {
		return false
	}
	if sig, ok := funcs[name]; ok {
		return sig.TouchesMutableGlobals
	}
	return false
}

func functionFieldInfoTouchesMutableGlobals(info FunctionFieldInfo, funcs map[string]FuncSig) bool {
	return info.FunctionTouchesMutableGlobals ||
		functionSymbolTouchesMutableGlobals(info.FunctionValue, funcs)
}

func functionAssignmentValueTouchesMutableGlobals(
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (bool, error) {
	switch v := value.(type) {
	case *frontend.ClosureExpr:
		return functionSymbolTouchesMutableGlobals(closureFunctionValueName(v, funcs, module), funcs), nil
	case *frontend.CallExpr:
		if resolved, err := resolveCheckedCallName(v.Name, funcs, module, imports, v.At); err == nil {
			v.Name = resolved
		}
		callSig, ok := funcs[v.Name]
		if !ok || !callSig.ReturnFunctionType {
			return false, nil
		}
		if callSig.ReturnFunctionTouchesMutableGlobals || functionSymbolTouchesMutableGlobals(
			callSig.ReturnFunctionSymbol,
			funcs,
		) {
			return true, nil
		}
		if callSig.ReturnFunctionParamName == "" {
			return false, nil
		}
		returnInfo, found, err := functionTypedReturnParamRefMetadata(
			callSig,
			callSig.ReturnFunctionParamName,
			v,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil || !found {
			return false, err
		}
		return functionFieldInfoTouchesMutableGlobals(returnInfo, funcs), nil
	case *frontend.FieldAccessExpr:
		if fieldInfo, ok, err := resolveFunctionFieldArgument(v, locals); err != nil {
			return false, err
		} else if ok {
			return functionFieldInfoTouchesMutableGlobals(fieldInfo, funcs), nil
		}
		if globalInfo, _, ok, err := resolveFunctionTypedGlobalFieldAccess(
			v,
			globals,
			funcs,
		); err != nil {
			return false, err
		} else if ok {
			return functionSymbolTouchesMutableGlobals(globalInfo.FunctionValue, funcs), nil
		}
	case *frontend.IdentExpr:
		if sourceInfo, ok := locals[v.Name]; ok {
			return sourceInfo.FunctionTouchesMutableGlobals || functionSymbolTouchesMutableGlobals(
				sourceInfo.FunctionValue,
				funcs,
			), nil
		}
		if globalInfo, ok := globals[v.Name]; ok {
			return functionSymbolTouchesMutableGlobals(globalInfo.FunctionValue, funcs), nil
		}
		if resolved, err := resolveCheckedCallName(v.Name, funcs, module, imports, v.At); err == nil {
			return functionSymbolTouchesMutableGlobals(resolved, funcs), nil
		}
	}
	return false, nil
}

func functionAssignmentMetadataWithReturnParamRefs(
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (string, []frontend.ClosureCapture, []frontend.ClosureCapture, string, error) {
	functionValue, captures, escapeCaptures, functionParamName := functionAssignmentMetadata(
		value,
		locals,
		globals,
		funcs,
		module,
		imports,
	)
	call, ok := value.(*frontend.CallExpr)
	if !ok {
		return functionValue, captures, escapeCaptures, functionParamName, nil
	}
	if resolved, err := resolveCheckedCallName(
		call.Name,
		funcs,
		module,
		imports,
		call.At,
	); err == nil {
		call.Name = resolved
	}
	callSig, ok := funcs[call.Name]
	if !ok || callSig.ReturnFunctionParamName == "" {
		if ok && callSig.ReturnFunctionType && callSig.ReturnFunctionSymbol == "" {
			fallbackValue, fallbackCaptures, fallbackEscapeCaptures, fallbackParamName, err := functionTypedReturnUnknownParamCaptureMetadata(
				callSig,
				call,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return functionValue, captures, escapeCaptures, functionParamName, err
			}
			if len(fallbackCaptures) > 0 || len(fallbackEscapeCaptures) > 0 {
				if fallbackValue != "" {
					functionValue = fallbackValue
				}
				captures = append([]frontend.ClosureCapture(nil), fallbackCaptures...)
				escapeCaptures = append(escapeCaptures, fallbackEscapeCaptures...)
				functionParamName = fallbackParamName
			}
		}
		return functionValue, captures, escapeCaptures, functionParamName, nil
	}
	returnInfo, found, err := functionTypedReturnParamRefMetadata(
		callSig,
		callSig.ReturnFunctionParamName,
		call,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
	)
	if err != nil || !found {
		return functionValue, captures, escapeCaptures, functionParamName, err
	}
	if returnInfo.FunctionValue != "" {
		functionValue = returnInfo.FunctionValue
	}
	functionParamName = returnInfo.FunctionParamName
	captures = append([]frontend.ClosureCapture(nil), returnInfo.FunctionCaptures...)
	escapeCaptures = append(escapeCaptures, returnInfo.FunctionEscapeCaptures...)
	return functionValue, captures, escapeCaptures, functionParamName, nil
}

func functionTypedReturnUnknownParamCaptureMetadata(
	callSig FuncSig,
	call *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (string, []frontend.ClosureCapture, []frontend.ClosureCapture, string, error) {
	functionValue := ""
	var captures []frontend.ClosureCapture
	var escapeCaptures []frontend.ClosureCapture
	functionParamName := ""
	for i, functionParam := range callSig.ParamFunctionTypes {
		if !functionParam || i >= len(call.Args) {
			continue
		}
		argCaptures, argEscapeCaptures, err := functionTypedCallArgumentCaptureMetadata(
			callSig,
			i,
			call.Args[i],
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return "", nil, nil, "", err
		}
		if len(argCaptures) == 0 && len(argEscapeCaptures) == 0 {
			continue
		}
		argValue, _, _, argParamName := functionAssignmentMetadata(
			call.Args[i],
			locals,
			globals,
			funcs,
			module,
			imports,
		)
		if functionValue == "" {
			functionValue = argValue
		}
		if functionParamName == "" {
			functionParamName = argParamName
		}
		captures = append(captures, argCaptures...)
		escapeCaptures = append(escapeCaptures, argEscapeCaptures...)
	}
	return functionValue, captures, escapeCaptures, functionParamName, nil
}

func updateFunctionTypedLocalAssignmentMetadata(
	name string,
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) error {
	localInfo, ok := locals[name]
	if !ok {
		return nil
	}
	functionValue, captures, escapeCaptures, functionParamName, err := functionAssignmentMetadataWithReturnParamRefs(
		value,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
	)
	if err != nil {
		return err
	}
	escapeKind, handleValue, err := functionAssignmentEscapeMetadata(
		value,
		locals,
		funcs,
		types,
		module,
		imports,
		callableBoundaryLocal,
	)
	if err != nil {
		return err
	}
	touchesMutableGlobals, err := functionAssignmentValueTouchesMutableGlobals(
		value,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
	)
	if err != nil {
		return err
	}
	localInfo.FunctionValue = functionValue
	localInfo.FunctionParamName = functionParamName
	localInfo.FunctionCaptures = captures
	localInfo.FunctionEscapeCaptures = escapeCaptures
	localInfo.FunctionTouchesMutableGlobals = touchesMutableGlobals
	localInfo.FunctionReturnSnapshotAlias = isFunctionReturnSnapshotAlias(
		value,
		funcs,
		captures,
		escapeCaptures,
		functionParamName,
	)
	localInfo.FunctionDirectSnapshotAlias = functionDirectSnapshotAliasForExpr(value, locals)
	localInfo.FunctionEscapeKind = escapeKind
	localInfo.FunctionHandleValue = handleValue
	locals[name] = localInfo
	return nil
}

func functionAssignmentEscapeMetadata(
	value frontend.Expr,
	locals map[string]LocalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	boundary callableEscapeBoundary,
) (CallableEscapeKind, bool, error) {
	switch v := value.(type) {
	case *frontend.ClosureExpr:
		if len(v.Captures) == 0 {
			return "", false, nil
		}
		captureSlots, err := functionCaptureSlotCount(v.Captures, types)
		if err != nil {
			return "", false, err
		}
		if captureSlots > FnPtrEnvSlotCount {
			return classifyCallableEscape(boundary, v.Captures, types)
		}
	case *frontend.IdentExpr:
		if source, ok := locals[v.Name]; ok {
			return source.FunctionEscapeKind, source.FunctionHandleValue, nil
		}
	case *frontend.FieldAccessExpr:
		fieldInfo, ok, err := resolveFunctionFieldArgument(v, locals)
		if err != nil {
			return "", false, err
		}
		if ok {
			return fieldInfo.FunctionEscapeKind, fieldInfo.FunctionHandleValue, nil
		}
	case *frontend.CallExpr:
		resolved, err := resolveCheckedCallName(v.Name, funcs, module, imports, v.At)
		if err != nil {
			return "", false, nil
		}
		if callSig, ok := funcs[resolved]; ok && callSig.ReturnFunctionHandleValue {
			return callSig.ReturnFunctionEscapeKind, callSig.ReturnFunctionHandleValue, nil
		}
	}
	return "", false, nil
}

func isFunctionReturnSnapshotAlias(
	value frontend.Expr,
	funcs map[string]FuncSig,
	captures []frontend.ClosureCapture,
	escapeCaptures []frontend.ClosureCapture,
	functionParamName string,
) bool {
	call, ok := value.(*frontend.CallExpr)
	if !ok {
		return false
	}
	callSig, ok := funcs[call.Name]
	return ok &&
		callSig.ReturnFunctionType &&
		callSig.ReturnFunctionParamName == "" &&
		len(captures) == 0 &&
		len(escapeCaptures) > 0 &&
		functionParamName == ""
}

func updateFunctionTypedFieldAssignmentMetadata(
	targetName string,
	targetInfo LocalInfo,
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) error {
	parts := strings.Split(targetName, ".")
	if len(parts) < 2 {
		return nil
	}
	base := parts[0]
	fieldPath := strings.Join(parts[1:], ".")
	localInfo, ok := locals[base]
	if !ok {
		return nil
	}
	if localInfo.FunctionFields == nil {
		localInfo.FunctionFields = map[string]FunctionFieldInfo{}
	}
	functionValue, captures, escapeCaptures, functionParamName, err := functionAssignmentMetadataWithReturnParamRefs(
		value,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
	)
	if err != nil {
		return err
	}
	escapeKind, handleValue, err := functionAssignmentEscapeMetadata(
		value,
		locals,
		funcs,
		types,
		module,
		imports,
		callableBoundaryStructField,
	)
	if err != nil {
		return err
	}
	touchesMutableGlobals, err := functionAssignmentValueTouchesMutableGlobals(
		value,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
	)
	if err != nil {
		return err
	}
	localInfo.FunctionFields[fieldPath] = FunctionFieldInfo{
		FunctionValue:                 functionValue,
		FunctionParamName:             functionParamName,
		FunctionCaptures:              captures,
		FunctionEscapeCaptures:        escapeCaptures,
		FunctionTouchesMutableGlobals: touchesMutableGlobals,
		FunctionReturnSnapshotAlias: isFunctionReturnSnapshotAlias(
			value,
			funcs,
			captures,
			escapeCaptures,
			functionParamName,
		),
		FunctionDirectSnapshotAlias: functionDirectSnapshotAliasForExpr(value, locals),
		FunctionEscapeKind:          escapeKind,
		FunctionHandleValue:         handleValue,
		FunctionParamTypes:          append([]string(nil), targetInfo.FunctionParamTypes...),
		FunctionParamOwnership:      append([]string(nil), targetInfo.FunctionParamOwnership...),
		FunctionReturnType:          targetInfo.FunctionReturnType,
		FunctionReturnOwnership:     targetInfo.FunctionReturnOwnership,
		FunctionThrowsType:          targetInfo.FunctionThrowsType,
		FunctionEffects:             append([]string(nil), targetInfo.FunctionEffects...),
	}
	locals[base] = localInfo
	return nil
}

func updateFunctionTypedStructFieldAssignmentMetadata(
	target frontend.Expr,
	targetType string,
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) error {
	base, fields, _, ok := splitFieldPath(target)
	if !ok || len(fields) == 0 {
		return nil
	}
	localInfo, ok := locals[base]
	if !ok || !localInfo.Mutable {
		return nil
	}
	fieldsForValue, err := functionFieldsFromReturnedStructExpr(
		targetType,
		value,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
	)
	if err != nil {
		return err
	}
	prefix := strings.Join(fields, ".")
	prefixWithDot := prefix + "."
	hadExisting := false
	for fieldName := range localInfo.FunctionFields {
		if fieldName == prefix || strings.HasPrefix(fieldName, prefixWithDot) {
			hadExisting = true
			break
		}
	}
	if len(fieldsForValue) == 0 && !hadExisting {
		return nil
	}
	updated := cloneFunctionFieldMap(localInfo.FunctionFields)
	if updated == nil {
		updated = map[string]FunctionFieldInfo{}
	}
	for fieldName := range updated {
		if fieldName == prefix || strings.HasPrefix(fieldName, prefixWithDot) {
			delete(updated, fieldName)
		}
	}
	for fieldName, fieldInfo := range fieldsForValue {
		updated[prefixWithDot+fieldName] = cloneFunctionFieldInfo(fieldInfo)
	}
	localInfo.FunctionFields = updated
	locals[base] = localInfo
	return nil
}

func updateEnumPayloadStructFieldAssignmentMetadata(
	target frontend.Expr,
	targetType string,
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) error {
	base, fields, _, ok := splitFieldPath(target)
	if !ok || len(fields) == 0 {
		return nil
	}
	localInfo, ok := locals[base]
	if !ok || !localInfo.Mutable {
		return nil
	}
	info, ok := types[targetType]
	if !ok {
		return nil
	}
	prefix := strings.Join(fields, ".")
	updates := map[string]FunctionFieldInfo{}
	switch info.Kind {
	case TypeEnum:
		payloads, err := enumPayloadFunctionsFromConstructor(
			info,
			value,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return err
		}
		if len(payloads) == 0 {
			payloads = enumPayloadFunctionsFromAlias(value, locals)
		}
		if len(payloads) == 0 {
			var err error
			payloads, err = enumPayloadFunctionsFromReturnCall(
				value,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				targetType,
			)
			if err != nil {
				return err
			}
		}
		for payloadKey, payload := range payloads {
			updates[enumPayloadFieldKey(prefix, payloadKey)] = cloneFunctionFieldInfo(payload)
		}
	case TypeStruct:
		fieldsForValue, err := enumPayloadFieldsFromReturnedStructExpr(
			targetType,
			value,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return err
		}
		for fieldName, fieldInfo := range fieldsForValue {
			updates[prefix+"."+fieldName] = cloneFunctionFieldInfo(fieldInfo)
		}
	default:
		return nil
	}
	hadExisting := false
	for fieldName := range localInfo.EnumPayloadFields {
		if enumPayloadFieldMatchesPrefix(fieldName, prefix) {
			hadExisting = true
			break
		}
	}
	if len(updates) == 0 && !hadExisting {
		return nil
	}
	updated := cloneFunctionFieldMap(localInfo.EnumPayloadFields)
	if updated == nil {
		updated = map[string]FunctionFieldInfo{}
	}
	for fieldName := range updated {
		if enumPayloadFieldMatchesPrefix(fieldName, prefix) {
			delete(updated, fieldName)
		}
	}
	for fieldName, fieldInfo := range updates {
		updated[fieldName] = cloneFunctionFieldInfo(fieldInfo)
	}
	localInfo.EnumPayloadFields = updated
	locals[base] = localInfo
	return nil
}

func validateFunctionTypedClosureAssignment(
	targetName string,
	targetInfo LocalInfo,
	closure *frontend.ClosureExpr,
	locals map[string]LocalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	pos frontend.Position,
	contextOverride ...string,
) error {
	context := "function-typed assignment"
	if len(contextOverride) > 0 && contextOverride[0] != "" {
		context = contextOverride[0]
	}
	targetPhrase := functionTypedClosureAssignmentTargetPhrase(context, targetName)
	if closure == nil || closure.Decl == nil {
		return fmt.Errorf(
			"%s: %s must use a closure literal with a body",
			frontend.FormatPos(pos),
			targetPhrase,
		)
	}
	if len(closure.Decl.TypeParams) > 0 {
		return fmt.Errorf(
			"%s: generic closure literals are not supported for %s in this MVP",
			frontend.FormatPos(closure.At),
			targetPhrase,
		)
	}
	explicitParams := explicitClosureParams(closure)
	if len(explicitParams) != len(targetInfo.FunctionParamTypes) {
		return fmt.Errorf(
			"%s: %s parameter count mismatch: expected %d, got %d",
			frontend.FormatPos(closure.At),
			targetPhrase,
			len(targetInfo.FunctionParamTypes),
			len(explicitParams),
		)
	}
	paramTypes := make([]string, 0, len(explicitParams))
	for _, param := range explicitParams {
		typeName, err := resolveTypeName(&param.Type, module, imports)
		if err != nil {
			return err
		}
		paramTypes = append(paramTypes, typeName)
	}
	returnType, err := resolveTypeName(&closure.Decl.ReturnType, module, imports)
	if err != nil {
		return err
	}
	throwsType := ""
	if closure.Decl.HasThrows {
		throwsType, err = resolveTypeName(&closure.Decl.Throws, module, imports)
		if err != nil {
			return err
		}
	}
	closureEffects, err := normalizeEffects(closure.Decl.Uses, closure.Decl.Pos)
	if err != nil {
		return err
	}
	closureSig, err := buildInterfaceFuncSig(targetName, funcSigSpec{
		ParamTypes:          paramTypes,
		ParamOwnership:      paramDeclOwnership(explicitParams),
		ReturnType:          returnType,
		ReturnOwnership:     closure.Decl.ReturnOwnership,
		ThrowsType:          throwsType,
		ReturnRegionParam:   regionNone,
		ReturnResourceParam: regionNone,
		Effects:             closureEffects,
	}, types)
	if err != nil {
		return err
	}
	if err := validateFunctionInfoAssignableWithContext(
		targetName,
		targetInfo,
		closureSig,
		closure.At,
		context,
	); err != nil {
		return err
	}
	return configureClosureCaptures(
		closure,
		locals,
		funcs,
		types,
		module,
		true,
		functionTypedClosureCaptureBoundaryPhrase(context, targetName),
	)
}

func functionTypedClosureAssignmentTargetPhrase(context, targetName string) string {
	if context == "" || context == "function-typed assignment" {
		return fmt.Sprintf("function-typed assignment to '%s'", targetName)
	}
	return fmt.Sprintf("%s '%s'", context, targetName)
}

// ---- checker_function_types_b.go ----

func functionTypedClosureCaptureBoundaryPhrase(context, targetName string) string {
	if context == "callback argument" {
		return fmt.Sprintf("callback argument '%s'", targetName)
	}
	if targetName == "return" {
		return "function-typed return 'closure literal'"
	}
	if context == "" || context == "function-typed assignment" {
		return fmt.Sprintf("function-typed storage '%s'", targetName)
	}
	return fmt.Sprintf("%s '%s'", context, targetName)
}

func validateFunctionTypedReturnCallAssignment(
	targetName string,
	targetInfo LocalInfo,
	call *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	allowCapturedLocalStorage bool,
) error {
	resolvedCall, err := resolveCheckedCallName(call.Name, funcs, module, imports, call.At)
	if err != nil {
		return unsupportedFunctionTypedAssignmentReturnCallSourceError(
			call.At,
			targetName,
			call.Name,
		)
	}
	call.Name = resolvedCall
	callSig, ok := funcs[resolvedCall]
	if !ok || !callSig.ReturnFunctionType {
		return unsupportedFunctionTypedAssignmentReturnCallSourceError(
			call.At,
			targetName,
			call.Name,
		)
	}
	if callSig.ReturnFunctionSymbol != "" {
		targetSig, ok := funcs[callSig.ReturnFunctionSymbol]
		if !ok {
			return fmt.Errorf(
				"%s: unknown returned function symbol '%s'",
				frontend.FormatPos(call.At),
				callSig.ReturnFunctionSymbol,
			)
		}
		if targetSig.Generic {
			return unsupportedGenericFunctionTypedAssignmentError(
				call.At,
				callSig.ReturnFunctionSymbol,
				targetName,
			)
		}
	}
	if !allowCapturedLocalStorage && len(callSig.ReturnFunctionCaptures) > 0 {
		if err := rejectMutableGlobalFunctionCaptures(callSig.ReturnFunctionCaptures, nil); err != nil {
			return err
		}
		captureSlots, err := functionCaptureSlotCount(callSig.ReturnFunctionCaptures, types)
		if err != nil {
			return err
		}
		if captureSlots < 1 || captureSlots > FnPtrEnvSlotCount {
			return unsupportedFunctionTypedStorageCaptureError(call.At, targetName, captureSlots)
		}
	}
	if !allowCapturedLocalStorage && callSig.ReturnFunctionParamName != "" {
		returnInfo, found, err := functionTypedReturnParamRefMetadata(
			callSig,
			callSig.ReturnFunctionParamName,
			call,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil || !found {
			return err
		}
		if returnInfo.FunctionParamName != "" {
			return unsupportedGlobalFunctionParameterStorageError(
				call.At,
				targetName,
				returnInfo.FunctionParamName,
			)
		}
		if len(returnInfo.FunctionCaptures) > 0 || len(returnInfo.FunctionEscapeCaptures) > 0 {
			return unsupportedGlobalFunctionCaptureStorageError(call.At, targetName)
		}
		if returnInfo.FunctionValue == "" &&
			strings.Contains(callSig.ReturnFunctionParamName, "#") {
			return unsupportedGlobalFunctionParameterStorageError(
				call.At,
				targetName,
				callSig.ReturnFunctionParamName,
			)
		}
	}
	if !allowCapturedLocalStorage && callSig.ReturnFunctionSymbol == "" &&
		callSig.ReturnFunctionParamName == "" {
		for i, functionParam := range callSig.ParamFunctionTypes {
			if !functionParam || i >= len(call.Args) {
				continue
			}
			captured, err := functionTypedCallArgumentHasCaptures(
				callSig,
				i,
				call.Args[i],
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return err
			}
			if captured {
				return unsupportedGlobalFunctionCaptureStorageError(call.At, targetName)
			}
		}
	}
	returnedSig, err := buildInterfaceFuncSig(targetName, funcSigSpec{
		ParamTypes:          append([]string(nil), callSig.ReturnFunctionParams...),
		ParamOwnership:      append([]string(nil), callSig.ReturnFunctionParamOwnership...),
		ReturnType:          callSig.ReturnFunctionReturn,
		ReturnOwnership:     callSig.ReturnFunctionReturnOwnership,
		ThrowsType:          callSig.ReturnFunctionThrows,
		ReturnRegionParam:   regionNone,
		ReturnResourceParam: regionNone,
		Effects:             append([]string(nil), callSig.ReturnFunctionEffects...),
	}, types)
	if err != nil {
		return err
	}
	return validateFunctionInfoAssignable(targetName, targetInfo, returnedSig, call.At)
}

func functionTypedCallArgumentHasCaptures(
	callSig FuncSig,
	index int,
	arg frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (bool, error) {
	captures, escapeCaptures, err := functionTypedCallArgumentCaptureMetadata(
		callSig,
		index,
		arg,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
	)
	if err != nil {
		return false, err
	}
	return len(captures) > 0 || len(escapeCaptures) > 0, nil
}

func functionTypedReturnParamRefHasCaptures(
	callSig FuncSig,
	paramRef string,
	call *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (bool, error) {
	captures, escapeCaptures, err := functionTypedReturnParamRefCaptureMetadata(
		callSig,
		paramRef,
		call,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
	)
	if err != nil {
		return false, err
	}
	return len(captures) > 0 || len(escapeCaptures) > 0, nil
}

func functionTypedReturnParamRefCaptureMetadata(
	callSig FuncSig,
	paramRef string,
	call *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) ([]frontend.ClosureCapture, []frontend.ClosureCapture, error) {
	info, ok, err := functionTypedReturnParamRefMetadata(
		callSig,
		paramRef,
		call,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
	)
	if err != nil || !ok {
		return nil, nil, err
	}
	return append(
			[]frontend.ClosureCapture(nil),
			info.FunctionCaptures...), append(
			[]frontend.ClosureCapture(nil),
			info.FunctionEscapeCaptures...), nil
}

func functionTypedReturnParamRefMetadata(
	callSig FuncSig,
	paramRef string,
	call *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (FunctionFieldInfo, bool, error) {
	if paramRef == "" {
		return FunctionFieldInfo{}, false, nil
	}
	for i, name := range callSig.ParamNames {
		if i >= len(call.Args) {
			continue
		}
		if name == paramRef {
			captures, escapeCaptures, err := functionTypedCallArgumentCaptureMetadata(
				callSig,
				i,
				call.Args[i],
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return FunctionFieldInfo{}, false, err
			}
			functionValue, _, _, functionParamName := functionAssignmentMetadata(
				call.Args[i],
				locals,
				globals,
				funcs,
				module,
				imports,
			)
			if functionValue == "" {
				metadataValue, _, _, metadataParamName, err := functionAssignmentMetadataWithReturnParamRefs(
					call.Args[i],
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
				)
				if err != nil {
					return FunctionFieldInfo{}, false, err
				}
				functionValue = metadataValue
				if functionParamName == "" {
					functionParamName = metadataParamName
				}
			}
			touchesMutableGlobals, err := functionAssignmentValueTouchesMutableGlobals(
				call.Args[i],
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return FunctionFieldInfo{}, false, err
			}
			return FunctionFieldInfo{
				FunctionValue:                 functionValue,
				FunctionParamName:             functionParamName,
				FunctionCaptures:              captures,
				FunctionEscapeCaptures:        escapeCaptures,
				FunctionTouchesMutableGlobals: touchesMutableGlobals,
			}, true, nil
		}
		payloadPrefix := name + "#"
		if strings.HasPrefix(paramRef, payloadPrefix) {
			payloadKey := strings.TrimPrefix(paramRef, payloadPrefix)
			payloadInfo, ok, err := functionEnumPayloadInfoFromCallArgument(
				payloadKey,
				callSig,
				i,
				call.Args[i],
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return FunctionFieldInfo{}, false, err
			}
			if !ok {
				return FunctionFieldInfo{}, false, nil
			}
			return payloadInfo, true, nil
		}
		prefix := name + "."
		if !strings.HasPrefix(paramRef, prefix) {
			continue
		}
		fieldPath := strings.TrimPrefix(paramRef, prefix)
		fieldInfo, ok, err := functionFieldInfoFromCallArgument(
			fieldPath,
			callSig,
			i,
			call.Args[i],
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return FunctionFieldInfo{}, false, err
		}
		if !ok {
			return FunctionFieldInfo{}, false, nil
		}
		return fieldInfo, true, nil
	}
	return FunctionFieldInfo{}, false, nil
}

func functionFieldInfoFromCallArgument(
	fieldPath string,
	callSig FuncSig,
	index int,
	arg frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (FunctionFieldInfo, bool, error) {
	fields := functionFieldsFromStructAlias(arg, locals)
	if len(fields) == 0 && index >= 0 && index < len(callSig.ParamTypes) {
		if info, ok := types[callSig.ParamTypes[index]]; ok && info.Kind == TypeStruct {
			var err error
			fields, err = functionFieldsFromStructLiteral(
				"<argument>",
				info,
				arg,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return FunctionFieldInfo{}, false, err
			}
		}
	}
	if len(fields) == 0 && index >= 0 && index < len(callSig.ParamTypes) {
		var err error
		fields, err = functionFieldsFromReturnCall(
			arg,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			callSig.ParamTypes[index],
		)
		if err != nil {
			return FunctionFieldInfo{}, false, err
		}
	}
	if len(fields) == 0 {
		return FunctionFieldInfo{}, false, nil
	}
	fieldInfo, ok := fields[fieldPath]
	if !ok {
		return FunctionFieldInfo{}, false, nil
	}
	return fieldInfo, true, nil
}

func functionEnumPayloadInfoFromCallArgument(
	payloadKey string,
	callSig FuncSig,
	index int,
	arg frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (FunctionFieldInfo, bool, error) {
	payloads := enumPayloadFunctionsFromAlias(arg, locals)
	if len(payloads) == 0 && index >= 0 && index < len(callSig.ParamTypes) {
		info, ok := types[callSig.ParamTypes[index]]
		if ok && info.Kind == TypeEnum {
			var err error
			payloads, err = enumPayloadFunctionsFromConstructor(
				info,
				arg,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return FunctionFieldInfo{}, false, err
			}
		}
	}
	if len(payloads) == 0 && index >= 0 && index < len(callSig.ParamTypes) {
		var err error
		payloads, err = enumPayloadFunctionsFromReturnCall(
			arg,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			callSig.ParamTypes[index],
		)
		if err != nil {
			return FunctionFieldInfo{}, false, err
		}
	}
	if len(payloads) == 0 {
		return FunctionFieldInfo{}, false, nil
	}
	payloadInfo, ok := payloads[payloadKey]
	if !ok {
		return FunctionFieldInfo{}, false, nil
	}
	return payloadInfo, true, nil
}

func functionTypedCallArgumentCaptureMetadata(
	callSig FuncSig,
	index int,
	arg frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) ([]frontend.ClosureCapture, []frontend.ClosureCapture, error) {
	if closure, ok := arg.(*frontend.ClosureExpr); ok {
		paramInfo := functionParamLocalInfo(callSig, index)
		if err := validateFunctionTypedClosureAssignment(
			"closure literal",
			paramInfo,
			closure,
			locals,
			funcs,
			types,
			module,
			imports,
			closure.At,
			"callback argument",
		); err != nil {
			return nil, nil, err
		}
	}
	if call, ok := arg.(*frontend.CallExpr); ok {
		if resolved, err := resolveCheckedCallName(
			call.Name,
			funcs,
			module,
			imports,
			call.At,
		); err == nil {
			call.Name = resolved
		}
	}
	_, captures, escapeCaptures, _, err := functionAssignmentMetadataWithReturnParamRefs(
		arg,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
	)
	if err != nil {
		return nil, nil, err
	}
	return captures, escapeCaptures, nil
}

func validateFunctionInfoAssignable(
	targetName string,
	targetInfo LocalInfo,
	sig FuncSig,
	pos frontend.Position,
) error {
	return validateFunctionInfoAssignableWithContext(
		targetName,
		targetInfo,
		sig,
		pos,
		"function-typed assignment",
	)
}

func validateFunctionInfoAssignableWithContext(
	targetName string,
	targetInfo LocalInfo,
	sig FuncSig,
	pos frontend.Position,
	context string,
) error {
	if len(targetInfo.FunctionParamTypes) != len(sig.ParamTypes) {
		if context == "function-typed assignment" {
			return fmt.Errorf(
				"%s: function-typed assignment to '%s' parameter count mismatch: expected %d, got %d",
				frontend.FormatPos(pos),
				targetName,
				len(targetInfo.FunctionParamTypes),
				len(sig.ParamTypes),
			)
		}
		return fmt.Errorf(
			"%s: %s '%s' parameter count mismatch: expected %d, got %d",
			frontend.FormatPos(pos),
			context,
			targetName,
			len(targetInfo.FunctionParamTypes),
			len(sig.ParamTypes),
		)
	}
	if err := validateFunctionTypeParamOwnership(
		targetInfo.FunctionParamOwnership,
		sig.ParamOwnership,
		len(targetInfo.FunctionParamTypes),
		pos,
		context,
		targetName,
	); err != nil {
		return err
	}
	for i := range targetInfo.FunctionParamTypes {
		if targetInfo.FunctionParamTypes[i] != sig.ParamTypes[i] {
			if context == "function-typed assignment" {
				return fmt.Errorf(
					"%s: function-typed assignment to '%s' parameter %d type mismatch: expected '%s', got '%s'",
					frontend.FormatPos(pos),
					targetName,
					i+1,
					targetInfo.FunctionParamTypes[i],
					sig.ParamTypes[i],
				)
			}
			return fmt.Errorf(
				"%s: %s '%s' parameter %d type mismatch: expected '%s', got '%s'",
				frontend.FormatPos(pos),
				context,
				targetName,
				i+1,
				targetInfo.FunctionParamTypes[i],
				sig.ParamTypes[i],
			)
		}
	}
	if targetInfo.FunctionReturnType != sig.ReturnType {
		if context == "function-typed assignment" {
			return fmt.Errorf(
				"%s: function-typed assignment to '%s' return type mismatch: expected '%s', got '%s'",
				frontend.FormatPos(pos),
				targetName,
				targetInfo.FunctionReturnType,
				sig.ReturnType,
			)
		}
		return fmt.Errorf(
			"%s: %s '%s' return type mismatch: expected '%s', got '%s'",
			frontend.FormatPos(pos),
			context,
			targetName,
			targetInfo.FunctionReturnType,
			sig.ReturnType,
		)
	}
	if targetInfo.FunctionReturnOwnership != sig.ReturnOwnership {
		if context == "function-typed assignment" {
			return fmt.Errorf(
				"%s: function-typed assignment to '%s' return ownership mismatch: expected '%s', got '%s'",
				frontend.FormatPos(pos),
				targetName,
				ownershipDisplay(targetInfo.FunctionReturnOwnership),
				ownershipDisplay(sig.ReturnOwnership),
			)
		}
		return fmt.Errorf(
			"%s: %s '%s' return ownership mismatch: expected '%s', got '%s'",
			frontend.FormatPos(pos),
			context,
			targetName,
			ownershipDisplay(targetInfo.FunctionReturnOwnership),
			ownershipDisplay(sig.ReturnOwnership),
		)
	}
	if targetInfo.FunctionThrowsType != sig.ThrowsType {
		if context == "function-typed assignment" {
			return fmt.Errorf(
				"%s: function-typed assignment to '%s' throws type mismatch: expected '%s', got '%s'",
				frontend.FormatPos(pos),
				targetName,
				targetInfo.FunctionThrowsType,
				sig.ThrowsType,
			)
		}
		return fmt.Errorf(
			"%s: %s '%s' throws type mismatch: expected '%s', got '%s'",
			frontend.FormatPos(pos),
			context,
			targetName,
			targetInfo.FunctionThrowsType,
			sig.ThrowsType,
		)
	}
	return validateFunctionTypeCallableEffects(
		targetInfo.FunctionEffects,
		sig.Effects,
		pos,
		context,
		targetName,
	)
}

func validateFunctionTypeCallableEffects(
	declaredEffects []string,
	targetEffects []string,
	pos frontend.Position,
	context, rawName string,
) error {
	missing := missingRequiredEffects(targetEffects, declaredEffects)
	if len(missing) > 0 {
		return fmt.Errorf(
			"%s: %s '%s' requires effects %s but function type does not declare them",
			frontend.FormatPos(pos),
			context,
			rawName,
			strings.Join(missing, ", "),
		)
	}
	return nil
}

func validateFunctionTypeParamOwnership(
	expected []string,
	actual []string,
	count int,
	pos frontend.Position,
	context, rawName string,
) error {
	for i := 0; i < count; i++ {
		want := ownershipAt(expected, i)
		got := ownershipAt(actual, i)
		if want != got {
			return ownershipDiagnosticf(
				pos,
				"%s '%s' parameter %d ownership mismatch: expected '%s', got '%s'",
				context,
				rawName,
				i+1,
				ownershipDisplay(want),
				ownershipDisplay(got),
			)
		}
	}
	return nil
}

func ownershipAt(ownership []string, index int) string {
	if index < 0 || index >= len(ownership) {
		return ""
	}
	return ownership[index]
}

func ownershipDisplay(ownership string) string {
	if ownership == "" {
		return "owned"
	}
	return ownership
}

func validateFunctionTypeSymbolSignature(
	localName string,
	declared frontend.TypeRef,
	sig FuncSig,
	module string,
	imports map[string]string,
	pos frontend.Position,
) error {
	if len(declared.Params) != len(sig.ParamTypes) {
		return fmt.Errorf(
			"%s: function-typed local '%s' parameter count mismatch: expected %d, got %d",
			frontend.FormatPos(pos),
			localName,
			len(declared.Params),
			len(sig.ParamTypes),
		)
	}
	if err := validateFunctionTypeParamOwnership(
		functionTypeRefParamOwnership(declared),
		sig.ParamOwnership,
		len(declared.Params),
		pos,
		"function-typed local",
		localName,
	); err != nil {
		return err
	}
	for i := range declared.Params {
		want, err := resolveTypeName(&declared.Params[i], module, imports)
		if err != nil {
			return err
		}
		got := sig.ParamTypes[i]
		if want != got {
			return fmt.Errorf(
				"%s: function-typed local '%s' parameter %d type mismatch: expected '%s', got '%s'",
				frontend.FormatPos(pos),
				localName,
				i+1,
				want,
				got,
			)
		}
	}
	if declared.Return == nil {
		return fmt.Errorf("%s: missing function return type", frontend.FormatPos(declared.At))
	}
	wantRet, err := resolveTypeName(declared.Return, module, imports)
	if err != nil {
		return err
	}
	if wantRet != sig.ReturnType {
		return fmt.Errorf(
			"%s: function-typed local '%s' return type mismatch: expected '%s', got '%s'",
			frontend.FormatPos(pos),
			localName,
			wantRet,
			sig.ReturnType,
		)
	}
	if declared.ReturnOwnership != sig.ReturnOwnership {
		return fmt.Errorf(
			"%s: function-typed local '%s' return ownership mismatch: expected '%s', got '%s'",
			frontend.FormatPos(pos),
			localName,
			ownershipDisplay(declared.ReturnOwnership),
			ownershipDisplay(sig.ReturnOwnership),
		)
	}
	wantThrows := ""
	if declared.Throws != nil {
		wantThrows, err = resolveTypeName(declared.Throws, module, imports)
		if err != nil {
			return err
		}
	}
	if wantThrows != sig.ThrowsType {
		return fmt.Errorf(
			"%s: function-typed local '%s' throws type mismatch: expected '%s', got '%s'",
			frontend.FormatPos(pos),
			localName,
			wantThrows,
			sig.ThrowsType,
		)
	}
	declaredEffects, err := functionTypeRefEffects(declared, declared.At)
	if err != nil {
		return err
	}
	if err := validateFunctionTypeCallableEffects(
		declaredEffects,
		sig.Effects,
		pos,
		"function-typed local",
		localName,
	); err != nil {
		return err
	}
	return nil
}

func validateReturnedFunctionSignature(
	callerSig FuncSig,
	returnedSig FuncSig,
	pos frontend.Position,
	rawName string,
) error {
	if !callerSig.ReturnFunctionType {
		return nil
	}
	if len(callerSig.ReturnFunctionParams) != len(returnedSig.ParamTypes) {
		return fmt.Errorf(
			"%s: returned function symbol '%s' has incompatible parameter count: expected %d, got %d",
			frontend.FormatPos(pos),
			rawName,
			len(callerSig.ReturnFunctionParams),
			len(returnedSig.ParamTypes),
		)
	}
	if err := validateFunctionTypeParamOwnership(
		callerSig.ReturnFunctionParamOwnership,
		returnedSig.ParamOwnership,
		len(callerSig.ReturnFunctionParams),
		pos,
		"returned function symbol",
		rawName,
	); err != nil {
		return err
	}
	for i := range callerSig.ReturnFunctionParams {
		if callerSig.ReturnFunctionParams[i] != returnedSig.ParamTypes[i] {
			return fmt.Errorf(
				"%s: returned function symbol '%s' parameter %d type mismatch: expected '%s', got '%s'",
				frontend.FormatPos(pos),
				rawName,
				i+1,
				callerSig.ReturnFunctionParams[i],
				returnedSig.ParamTypes[i],
			)
		}
	}
	if callerSig.ReturnFunctionReturn != returnedSig.ReturnType {
		return fmt.Errorf(
			"%s: returned function symbol '%s' return type mismatch: expected '%s', got '%s'",
			frontend.FormatPos(pos),
			rawName,
			callerSig.ReturnFunctionReturn,
			returnedSig.ReturnType,
		)
	}
	if callerSig.ReturnFunctionReturnOwnership != returnedSig.ReturnOwnership {
		return fmt.Errorf(
			"%s: returned function symbol '%s' return ownership mismatch: expected '%s', got '%s'",
			frontend.FormatPos(pos),
			rawName,
			ownershipDisplay(callerSig.ReturnFunctionReturnOwnership),
			ownershipDisplay(returnedSig.ReturnOwnership),
		)
	}
	if callerSig.ReturnFunctionThrows != returnedSig.ThrowsType {
		return fmt.Errorf(
			"%s: returned function symbol '%s' throws type mismatch: expected '%s', got '%s'",
			frontend.FormatPos(pos),
			rawName,
			callerSig.ReturnFunctionThrows,
			returnedSig.ThrowsType,
		)
	}
	if err := validateFunctionTypeCallableEffects(
		callerSig.ReturnFunctionEffects,
		returnedSig.Effects,
		pos,
		"returned function symbol",
		rawName,
	); err != nil {
		return err
	}
	return nil
}

func functionTypeRefSignature(
	ref frontend.TypeRef,
	module string,
	imports map[string]string,
) ([]string, string, error) {
	if ref.Kind != frontend.TypeRefFunction {
		return nil, "", nil
	}
	paramTypes := make([]string, 0, len(ref.Params))
	for i := range ref.Params {
		tname, err := resolveTypeName(&ref.Params[i], module, imports)
		if err != nil {
			return nil, "", err
		}
		paramTypes = append(paramTypes, tname)
	}
	if ref.Return == nil {
		return nil, "", fmt.Errorf("%s: missing function return type", frontend.FormatPos(ref.At))
	}
	retType, err := resolveTypeName(ref.Return, module, imports)
	if err != nil {
		return nil, "", err
	}
	return paramTypes, retType, nil
}

func functionTypeRefParamOwnership(ref frontend.TypeRef) []string {
	return semanticsfunctiontypes.ParamOwnership(ref)
}

func functionTypeRefReturnOwnership(ref frontend.TypeRef) string {
	return semanticsfunctiontypes.ReturnOwnership(ref)
}

func functionTypeRefEffects(ref frontend.TypeRef, pos frontend.Position) ([]string, error) {
	return semanticsfunctiontypes.Effects(ref, pos, effectDiagnosticf)
}

func functionTypeRefSignatureAndEffects(
	ref frontend.TypeRef,
	module string,
	imports map[string]string,
) ([]string, string, []string, error) {
	params, ret, err := functionTypeRefSignature(ref, module, imports)
	if err != nil {
		return nil, "", nil, err
	}
	effects, err := functionTypeRefEffects(ref, ref.At)
	if err != nil {
		return nil, "", nil, err
	}
	return params, ret, effects, nil
}

func functionTypeRefThrowsType(
	ref frontend.TypeRef,
	module string,
	imports map[string]string,
) (string, error) {
	if ref.Kind != frontend.TypeRefFunction || ref.Throws == nil {
		return "", nil
	}
	return resolveTypeName(ref.Throws, module, imports)
}

func cloneBoolMap(src map[string]bool) map[string]bool {
	if len(src) == 0 {
		return make(map[string]bool)
	}
	dst := make(map[string]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func borrowedPtrOwnerFromExpr(
	expr frontend.Expr,
	state *regionState,
	borrowedParams map[string]struct{},
) (string, bool) {
	id, ok := expr.(*frontend.IdentExpr)
	if ok {
		if _, borrowed := borrowedParams[id.Name]; borrowed {
			return id.Name, true
		}
		if owner, borrowed := state.borrowedPtrAliasOwner(id.Name); borrowed {
			return owner, true
		}
	}
	if path, ok := resourcePathForExpr(expr); ok {
		if owner, borrowed := state.borrowedPtrAliasOwnerInTree(path); borrowed {
			return owner, true
		}
	}
	if field, ok := expr.(*frontend.FieldAccessExpr); ok {
		if owner, borrowed := borrowedPtrOwnerFromExpr(field.Base, state, borrowedParams); borrowed {
			return owner, true
		}
	}
	if index, ok := expr.(*frontend.IndexExpr); ok {
		if owner, borrowed := borrowedPtrOwnerFromExpr(index.Base, state, borrowedParams); borrowed {
			return owner, true
		}
		if source, ok := resourcePathForExpr(index.Base); ok {
			if owner, borrowed := state.borrowedPtrAliasOwnerInTree(source); borrowed {
				return owner, true
			}
		}
	}
	return "", false
}

func bindBorrowedPtrAliasFromExpr(
	name string,
	typeName string,
	expr frontend.Expr,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	borrowedParams map[string]struct{},
) {
	if state == nil || name == "" {
		return
	}
	state.clearBorrowedPtrAliasTree(name)
	if typeName == "ptr" {
		if owner, borrowed := borrowedPtrOwnerFromExpr(expr, state, borrowedParams); borrowed {
			state.bindBorrowedPtrAlias(name, owner)
		}
		return
	}
	if owner, borrowed := borrowedPtrOwnerFromExpr(expr, state, borrowedParams); borrowed {
		if typeMayContainPtr(typeName, types) {
			state.bindBorrowedPtrAlias(name, owner)
		}
	}
	if !typeMayContainPtr(typeName, types) {
		return
	}
	if info, ok := types[typeName]; ok && info.Kind == TypeOptional {
		bindBorrowedPtrAliasFromExpr(
			resourceElementPath(name),
			info.ElemType,
			expr,
			types,
			module,
			imports,
			state,
			borrowedParams,
		)
		return
	}
	if sourcePath, ok := resourcePathForExpr(expr); ok {
		for path, owner := range state.borrowedPtrAliases {
			suffix, ok := resourcePathRelativeTo(path, sourcePath)
			if owner == "" || !ok || suffix == "" {
				continue
			}
			state.bindBorrowedPtrAlias(joinResourcePath(name, suffix), owner)
		}
		return
	}
	info, ok := types[typeName]
	if !ok {
		return
	}
	if info.Kind == TypeEnum {
		call, ok := expr.(*frontend.CallExpr)
		if !ok {
			return
		}
		enumType, caseInfo, found, err := resolveEnumCaseConstructorCall(
			call,
			types,
			module,
			imports,
		)
		if err != nil || !found || enumType != typeName {
			return
		}
		for i, arg := range call.Args {
			if i >= len(caseInfo.PayloadTypes) {
				break
			}
			bindBorrowedPtrAliasFromExpr(
				resourceEnumPayloadPath(name, caseInfo.Ordinal, i),
				caseInfo.PayloadTypes[i],
				arg,
				types,
				module,
				imports,
				state,
				borrowedParams,
			)
		}
		return
	}
	if info.Kind != TypeStruct {
		return
	}
	fieldTypes := make(map[string]string, len(info.Fields))
	for _, field := range info.Fields {
		fieldTypes[field.Name] = field.TypeName
	}
	lit, ok := expr.(*frontend.StructLitExpr)
	if lit != nil {
		for _, field := range lit.Fields {
			fieldType, ok := fieldTypes[field.Name]
			if !ok {
				continue
			}
			bindBorrowedPtrAliasFromExpr(
				joinResourcePath(name, field.Name),
				fieldType,
				field.Value,
				types,
				module,
				imports,
				state,
				borrowedParams,
			)
		}
		return
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || len(call.Args) == 0 || len(call.ArgLabels) != len(call.Args) {
		return
	}
	for i, arg := range call.Args {
		label := call.ArgLabels[i]
		if label == "" {
			return
		}
		fieldType, ok := fieldTypes[label]
		if !ok {
			continue
		}
		bindBorrowedPtrAliasFromExpr(
			joinResourcePath(name, label),
			fieldType,
			arg,
			types,
			module,
			imports,
			state,
			borrowedParams,
		)
	}
}

func compoundIndexTargetHasSideEffects(expr frontend.Expr) bool {
	switch e := expr.(type) {
	case *frontend.IndexExpr:
		return exprMayHaveRuntimeSideEffects(e.Base) || exprMayHaveRuntimeSideEffects(e.Index)
	case *frontend.FieldAccessExpr:
		return compoundIndexTargetHasSideEffects(e.Base)
	default:
		return false
	}
}

func exprMayHaveRuntimeSideEffects(expr frontend.Expr) bool {
	switch e := expr.(type) {
	case nil:
		return false
	case *frontend.CallExpr, *frontend.TryExpr, *frontend.AwaitExpr, *frontend.CatchExpr:
		return true
	case *frontend.FieldAccessExpr:
		return exprMayHaveRuntimeSideEffects(e.Base)
	case *frontend.IndexExpr:
		return exprMayHaveRuntimeSideEffects(e.Base) || exprMayHaveRuntimeSideEffects(e.Index)
	case *frontend.UnaryExpr:
		return exprMayHaveRuntimeSideEffects(e.X)
	case *frontend.BinaryExpr:
		return exprMayHaveRuntimeSideEffects(e.Left) || exprMayHaveRuntimeSideEffects(e.Right)
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if exprMayHaveRuntimeSideEffects(field.Value) {
				return true
			}
		}
		return false
	case *frontend.MatchExpr:
		if exprMayHaveRuntimeSideEffects(e.Value) {
			return true
		}
		for _, c := range e.Cases {
			if exprMayHaveRuntimeSideEffects(
				c.Pattern,
			) || exprMayHaveRuntimeSideEffects(
				c.Guard,
			) || exprMayHaveRuntimeSideEffects(
				c.Value,
			) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func checkBorrowedReturnContract(
	expr frontend.Expr,
	returnType string,
	callerSig FuncSig,
	callerSigOK bool,
	borrowedParams map[string]struct{},
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
	pos frontend.Position,
) error {
	if callerSigOK && callerSig.ReturnOwnership == "borrow" {
		borrowedName, borrowed, err := borrowedOwnerFromExpr(
			expr,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
		if err != nil {
			return err
		}
		if !borrowed {
			return nil
		}
		if borrowedName == "<borrow>" {
			kind, _ := borrowedReturnTypeLabels(returnType, types)
			return lifetimeDiagnosticf(
				pos,
				"borrowed %s return requires caller-visible borrow source",
				kind,
			)
		}
		if _, ok := borrowedParams[borrowedName]; ok {
			if decision := islandkernel.CanBorrow(islandKernelSemanticBorrowRequest(borrowedName)); decision.Decision != islandkernel.Accept {
				return lifetimeDiagnosticf(
					pos,
					"borrowed return owner '%s' rejected by island kernel (%s)",
					borrowedName,
					decision.Reason.Code,
				)
			}
			if err := recordBorrowedReturnOwner(analysis, borrowedName, pos); err != nil {
				return err
			}
			return nil
		}
		kind, _ := borrowedReturnTypeLabels(returnType, types)
		return lifetimeDiagnosticf(
			pos,
			"borrowed %s return derives from local owner '%s'",
			kind,
			borrowedName,
		)
	}
	if err := checkBorrowedAggregateEscape(
		expr,
		returnType,
		"escape through owned return",
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
		state,
		effects,
		analysis,
		pos,
	); err != nil {
		return err
	}
	return checkBorrowedEscape(
		expr,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
		state,
		effects,
		analysis,
		func(borrowedName string) error {
			if decision := islandkernel.CanBorrow(islandKernelSemanticBorrowRequest(borrowedName)); decision.Decision != islandkernel.Accept {
				return lifetimeDiagnosticf(
					pos,
					"borrowed local '%s' cannot escape via return (%s)",
					borrowedName,
					decision.Reason.Code,
				)
			}
			decision := islandkernel.CanReturn(islandkernel.EscapeRequest{
				Ref: islandKernelSemanticBorrowRef(borrowedName),
			})
			if decision.Decision == islandkernel.Accept {
				return nil
			}
			kind, display, directView := borrowedReturnDirectViewLabels(returnType, types)
			if directView {
				return lifetimeDiagnosticf(
					pos,
					"borrowed %s return requires '-> borrow %s' or '.copy()' (%s)",
					kind,
					display,
					decision.Reason.Code,
				)
			}
			return lifetimeDiagnosticf(
				pos,
				"borrowed local '%s' cannot escape via return (%s)",
				borrowedName,
				decision.Reason.Code,
			)
		},
	)
}

func islandKernelSemanticBorrowRequest(owner string) islandkernel.BorrowRequest {
	ref := islandKernelSemanticBorrowRef(owner)
	return islandkernel.BorrowRequest{
		Ref: ref,
		Token: islandkernel.Token{
			IslandID: ref.IslandID,
			Epoch:    ref.Epoch,
			OwnerID:  ref.OwnerID,
		},
	}
}

func islandKernelSemanticBorrowRef(owner string) islandkernel.MemoryRef {
	owner = strings.TrimSpace(owner)
	if owner == "" {
		owner = "<unknown-borrow>"
	}
	return islandkernel.MemoryRef{
		BaseID:      owner,
		IslandID:    "semantic-borrow:" + owner,
		Epoch:       1,
		OwnerID:     owner,
		Provenance:  islandkernel.ProvenanceBorrowedView,
		UnsafeClass: islandkernel.UnsafeSafe,
	}
}

func recordBorrowedReturnOwner(
	analysis *functionAnalysisState,
	owner string,
	pos frontend.Position,
) error {
	if analysis == nil || owner == "" {
		return nil
	}
	if analysis.borrowedReturnOwner == "" {
		analysis.borrowedReturnOwner = owner
		return nil
	}
	if analysis.borrowedReturnOwner != owner {
		return lifetimeDiagnosticf(
			pos,
			("borrowed return has multiple possible owner sources ('%s', " +
				"'%s'); named lifetimes are not supported in v1"),
			analysis.borrowedReturnOwner,
			owner,
		)
	}
	return nil
}

func checkBorrowedAggregateEscape(
	expr frontend.Expr,
	typeName string,
	context string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
	pos frontend.Position,
) error {
	if explicitCopyResultExpr(expr) {
		return nil
	}
	info, ok := types[typeName]
	if !ok {
		return nil
	}
	switch info.Kind {
	case TypeStruct:
		for _, field := range structFieldExprs(expr, info) {
			if err := checkBorrowedAggregateFieldEscape(
				info.Name,
				field.name,
				field.typeName,
				field.value,
				context,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
				pos,
			); err != nil {
				return err
			}
		}
	case TypeEnum:
		if call, ok := expr.(*frontend.CallExpr); ok {
			_, caseInfo, found, err := resolveEnumCaseConstructorCall(call, types, module, imports)
			if err != nil {
				return err
			}
			if !found {
				return nil
			}
			for i, arg := range call.Args {
				if i >= len(caseInfo.PayloadTypes) {
					break
				}
				label := fmt.Sprintf(
					"%s.%s[%d]",
					displayTypeName(typeName, module),
					caseInfo.Name,
					i+1,
				)
				if err := checkBorrowedAggregateFieldEscape(
					displayTypeName(typeName, module),
					label,
					caseInfo.PayloadTypes[i],
					arg,
					context,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
					pos,
				); err != nil {
					return err
				}
			}
		}
	case TypeOptional:
		if call, ok := expr.(*frontend.CallExpr); ok {
			for _, arg := range call.Args {
				if err := checkBorrowedAggregateFieldEscape(
					displayTypeName(typeName, module),
					"$elem",
					info.ElemType,
					arg,
					context,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
					pos,
				); err != nil {
					return err
				}
			}
		} else if borrowedEscapeShouldInspect(info.ElemType, types) {
			if err := checkBorrowedAggregateFieldEscape(
				displayTypeName(typeName, module),
				"$elem",
				info.ElemType,
				expr,
				context,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
				pos,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkBorrowedAggregateFieldEscape(
	aggregateName string,
	fieldName string,
	fieldType string,
	value frontend.Expr,
	context string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
	pos frontend.Position,
) error {
	if !borrowedEscapeShouldInspect(fieldType, types) {
		return nil
	}
	kind, _, directView := borrowedReturnDirectViewLabels(fieldType, types)
	if directView {
		if _, borrowed, err := borrowedOwnerFromExpr(
			value,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		); err != nil {
			return err
		} else if borrowed {
			return lifetimeDiagnosticf(pos, ("aggregate '%s' contains borrowed %s field '%s' that cannot " +
				"%s"), displayTypeName(aggregateName, module), kind, fieldName, context)
		}
	}
	return checkBorrowedAggregateEscape(
		value,
		fieldType,
		context,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
		state,
		effects,
		analysis,
		pos,
	)
}

type structFieldExpr struct {
	name     string
	typeName string
	value    frontend.Expr
}

func structFieldExprs(expr frontend.Expr, info *TypeInfo) []structFieldExpr {
	if info == nil || info.Kind != TypeStruct || expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *frontend.StructLitExpr:
		out := make([]structFieldExpr, 0, len(e.Fields))
		for _, field := range e.Fields {
			fieldInfo, ok := info.FieldMap[field.Name]
			if !ok {
				continue
			}
			out = append(
				out,
				structFieldExpr{name: field.Name, typeName: fieldInfo.TypeName, value: field.Value},
			)
		}
		return out
	case *frontend.CallExpr:
		out := make([]structFieldExpr, 0, len(e.Args))
		if len(e.ArgLabels) == len(e.Args) {
			for i, arg := range e.Args {
				label := e.ArgLabels[i]
				if label == "" {
					continue
				}
				fieldInfo, ok := info.FieldMap[label]
				if !ok {
					continue
				}
				out = append(out, structFieldExpr{name: label, typeName: fieldInfo.TypeName, value: arg})
			}
			return out
		}
		for i, arg := range e.Args {
			if i >= len(info.Fields) {
				break
			}
			field := info.Fields[i]
			out = append(out, structFieldExpr{name: field.Name, typeName: field.TypeName, value: arg})
		}
		return out
	}
	return nil
}

func borrowedReturnTypeLabels(
	typeName string,
	types map[string]*TypeInfo,
) (kind string, display string) {
	kind, display, _ = borrowedReturnDirectViewLabels(typeName, types)
	return kind, display
}

func borrowedReturnDirectViewLabels(
	typeName string,
	types map[string]*TypeInfo,
) (kind string, display string, directView bool) {
	if info, ok := types[typeName]; ok {
		switch info.Kind {
		case TypeStr:
			return "String", "String", true
		case TypeSlice:
			return "slice", typeName, true
		}
	}
	if typeName == "str" || typeName == "String" {
		return "String", "String", true
	}
	return "value", typeName, false
}

// ---- checker_locals.go ----

func matchPatternCompatible(scrutType, patternType string, types map[string]*TypeInfo) bool {
	if scrutType == patternType {
		return true
	}
	if patternType == optionalSomePatternType {
		if scrutInfo, ok := types[scrutType]; ok && scrutInfo.Kind == TypeOptional {
			return true
		}
	}
	if patternType == "none" {
		if scrutInfo, ok := types[scrutType]; ok && scrutInfo.Kind == TypeOptional {
			return true
		}
	}
	if isInt32Like(scrutType) && isInt32Like(patternType) {
		return true
	}
	scrutInfo, scrutOK := types[scrutType]
	patternInfo, patternOK := types[patternType]
	if scrutOK && patternOK && (scrutInfo.Kind == TypeEnum || patternInfo.Kind == TypeEnum) {
		return false
	}
	return false
}

func validateIfLetPattern(
	pattern frontend.Expr,
	valueType string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) error {
	valueInfo, ok := types[valueType]
	if !ok {
		return fmt.Errorf("unknown type '%s'", valueType)
	}
	patType := ""
	if some, ok := pattern.(*frontend.SomePatternExpr); ok {
		if valueInfo.Kind != TypeOptional {
			return fmt.Errorf(
				"%s: some pattern requires optional match value",
				frontend.FormatPos(some.At),
			)
		}
		patType = optionalSomePatternType
	} else if enumPat, ok := pattern.(*frontend.EnumCasePatternExpr); ok {
		caseType, caseInfo, found, err := resolveEnumCasePattern(enumPat, types, module, imports)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf(
				"%s: unknown enum pattern '%s.%s'",
				frontend.FormatPos(enumPat.At),
				enumPat.TypeName,
				enumPat.CaseName,
			)
		}
		if err := validateEnumCasePatternPayload(enumPat, caseType, caseInfo, module); err != nil {
			return err
		}
		patType = caseType
	} else {
		var err error
		patType, _, err = checkExprWithEffects(
			pattern,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
		if err != nil {
			return err
		}
	}
	if valueInfo.Kind == TypeOptional && patType != "none" && patType != optionalSomePatternType {
		return fmt.Errorf(
			"%s: optional if let supports only 'none', 'some(name)', and '_' patterns",
			frontend.FormatPos(pattern.Pos()),
		)
	}
	if !matchPatternCompatible(valueType, patType, types) {
		return fmt.Errorf(
			"%s: if let pattern type mismatch: expected '%s', got '%s'",
			frontend.FormatPos(pattern.Pos()),
			valueType,
			patType,
		)
	}
	return nil
}

const optionalSomePatternType = "__optional_some_pattern"

func matchPatternKey(pattern frontend.Expr, patternType string) string {
	switch p := pattern.(type) {
	case *frontend.NumberExpr:
		return fmt.Sprintf("i32:%d", p.Value)
	case *frontend.NoneLitExpr:
		return "optional:none"
	case *frontend.SomePatternExpr:
		return "optional:some"
	case *frontend.EnumCasePatternExpr:
		if p.EnumType != "" {
			return "enum:" + p.EnumType + "." + p.CaseName
		}
		return "enum:" + p.TypeName + "." + p.CaseName
	case *frontend.FieldAccessExpr:
		if p.EnumType != "" {
			return "enum:" + p.EnumType + "." + p.Field
		}
		return patternType + ":" + p.Field
	default:
		return ""
	}
}

func uniqueHiddenLocal(prefix string, pos frontend.Position, locals map[string]LocalInfo) string {
	base := fmt.Sprintf("%s_%d_%d", prefix, pos.Line, pos.Col)
	name := base
	for i := 1; ; i++ {
		if _, exists := locals[name]; !exists {
			return name
		}
		name = fmt.Sprintf("%s_%d", base, i)
	}
}

func collectScopedLocal(
	name string,
	info LocalInfo,
	pos frontend.Position,
	locals map[string]LocalInfo,
	slotIndex *int,
	scopes *scopeInfo,
) error {
	if existing, exists := locals[name]; exists {
		if scopes == nil {
			return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(pos), name)
		}
		currentScope := scopes.currentScopeID()
		existingScope := scopes.localScopes[name]
		if currentScope == regionNone || existingScope == regionNone ||
			currentScope == existingScope ||
			!localInfosShareScopedStorage(existing, info) {
			return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(pos), name)
		}
		ids := scopes.localScopeSets[name]
		if ids == nil {
			ids = make(map[int]struct{}, 2)
			scopes.localScopeSets[name] = ids
		}
		ids[existingScope] = struct{}{}
		ids[currentScope] = struct{}{}
		scopes.localScopes[name] = currentScope
		return nil
	}
	locals[name] = info
	if scopes != nil {
		scopes.localScopes[name] = scopes.currentScopeID()
	}
	*slotIndex += info.SlotCount
	return nil
}

func localInfosShareScopedStorage(left, right LocalInfo) bool {
	return left.SlotCount == right.SlotCount &&
		left.TypeName == right.TypeName &&
		left.FunctionTypeValue == right.FunctionTypeValue &&
		left.FunctionReturnType == right.FunctionReturnType &&
		left.FunctionReturnOwnership == right.FunctionReturnOwnership &&
		strings.Join(
			left.FunctionParamTypes,
			"\x00",
		) == strings.Join(
			right.FunctionParamTypes,
			"\x00",
		)
}

func collectionElementType(typeName string, types map[string]*TypeInfo) (string, error) {
	info, err := ensureTypeInfo(typeName, types)
	if err != nil {
		return "", err
	}
	switch info.Kind {
	case TypeStr:
		return "u8", nil
	case TypeSlice:
		return info.ElemType, nil
	case TypeArray:
		return info.ElemType, nil
	default:
		return "", fmt.Errorf("for collection requires array, slice, or string, got '%s'", typeName)
	}
}

func collectExprLocals(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	slotIndex *int,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	scopes *scopeInfo,
	globals map[string]GlobalInfo,
) error {
	switch e := expr.(type) {
	case *frontend.MatchExpr:
		if err := collectExprLocals(
			e.Value,
			locals,
			slotIndex,
			funcs,
			types,
			module,
			imports,
			scopes,
			globals,
		); err != nil {
			return err
		}
		scrutType, err := inferExprTypeForDecl(e.Value, locals, globals, funcs, types, module, imports)
		if err != nil {
			return fmt.Errorf("%s: cannot infer match value type: %v", frontend.FormatPos(e.At), err)
		}
		scrutInfo, err := ensureTypeInfo(scrutType, types)
		if err != nil {
			return fmt.Errorf("%s: %v", frontend.FormatPos(e.At), err)
		}
		if scrutInfo.SlotCount != 1 && scrutInfo.Kind != TypeOptional && scrutInfo.Kind != TypeEnum {
			return fmt.Errorf("%s: match value must be single-slot", frontend.FormatPos(e.At))
		}
		resultType, err := inferMatchExprType(e, locals, globals, funcs, types, module, imports)
		if err != nil {
			return fmt.Errorf("%s: cannot infer match expression type: %v", frontend.FormatPos(e.At), err)
		}
		resultInfo, err := ensureTypeInfo(resultType, types)
		if err != nil {
			return fmt.Errorf("%s: %v", frontend.FormatPos(e.At), err)
		}
		e.ScrutineeLocal = uniqueHiddenLocal("__match_expr_value", e.At, locals)
		locals[e.ScrutineeLocal] = LocalInfo{
			Base:      *slotIndex,
			SlotCount: scrutInfo.SlotCount,
			TypeName:  scrutType,
			Mutable:   false,
		}
		if scopes != nil {
			scopes.localScopes[e.ScrutineeLocal] = scopes.currentScopeID()
		}
		*slotIndex += scrutInfo.SlotCount
		e.ResultLocal = uniqueHiddenLocal("__match_expr_result", e.At, locals)
		e.ResultType = resultType
		locals[e.ResultLocal] = LocalInfo{
			Base:      *slotIndex,
			SlotCount: resultInfo.SlotCount,
			TypeName:  resultType,
			Mutable:   false,
		}
		if scopes != nil {
			scopes.localScopes[e.ResultLocal] = scopes.currentScopeID()
		}
		*slotIndex += resultInfo.SlotCount
		caseScopeIDs := make([]int, len(e.Cases))
		for i, c := range e.Cases {
			if scopes != nil {
				caseScopeIDs[i] = scopes.enterScope()
			} else {
				caseScopeIDs[i] = regionNone
			}
			if !c.Default {
				if err := collectPatternLocals(
					c.Pattern,
					scrutType,
					locals,
					slotIndex,
					types,
					module,
					imports,
					scopes,
					globals,
				); err != nil {
					return err
				}
			}
			if c.Guard != nil {
				if err := collectExprLocals(
					c.Guard,
					locals,
					slotIndex,
					funcs,
					types,
					module,
					imports,
					scopes,
					globals,
				); err != nil {
					return err
				}
			}
			if err := collectExprLocals(
				c.Value,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
			}
		}
		if scopes != nil {
			scopes.matchExprScopes[e] = caseScopeIDs
		}
	case *frontend.CatchExpr:
		if err := collectExprLocals(
			e.Call,
			locals,
			slotIndex,
			funcs,
			types,
			module,
			imports,
			scopes,
			globals,
		); err != nil {
			return err
		}
		resultType, err := inferCatchExprType(e, locals, globals, funcs, types, module, imports)
		if err != nil {
			return fmt.Errorf("%s: cannot infer catch expression type: %v", frontend.FormatPos(e.At), err)
		}
		resultInfo, err := ensureTypeInfo(resultType, types)
		if err != nil {
			return fmt.Errorf("%s: %v", frontend.FormatPos(e.At), err)
		}
		errorInfo, err := ensureTypeInfo(e.ErrorType, types)
		if err != nil {
			return fmt.Errorf("%s: %v", frontend.FormatPos(e.At), err)
		}
		e.ErrorLocal = uniqueHiddenLocal("__catch_error", e.At, locals)
		locals[e.ErrorLocal] = LocalInfo{
			Base:      *slotIndex,
			SlotCount: errorInfo.SlotCount,
			TypeName:  e.ErrorType,
			Mutable:   false,
		}
		if scopes != nil {
			scopes.localScopes[e.ErrorLocal] = scopes.currentScopeID()
		}
		*slotIndex += errorInfo.SlotCount
		e.ResultLocal = uniqueHiddenLocal("__catch_result", e.At, locals)
		e.ResultType = resultType
		locals[e.ResultLocal] = LocalInfo{
			Base:      *slotIndex,
			SlotCount: resultInfo.SlotCount,
			TypeName:  resultType,
			Mutable:   false,
		}
		if scopes != nil {
			scopes.localScopes[e.ResultLocal] = scopes.currentScopeID()
		}
		*slotIndex += resultInfo.SlotCount
		caseScopeIDs := make([]int, len(e.Cases))
		for i, c := range e.Cases {
			if scopes != nil {
				caseScopeIDs[i] = scopes.enterScope()
			} else {
				caseScopeIDs[i] = regionNone
			}
			if !c.Default {
				if err := collectPatternLocals(
					c.Pattern,
					e.ErrorType,
					locals,
					slotIndex,
					types,
					module,
					imports,
					scopes,
					globals,
				); err != nil {
					return err
				}
			}
			if c.Guard != nil {
				if err := collectExprLocals(
					c.Guard,
					locals,
					slotIndex,
					funcs,
					types,
					module,
					imports,
					scopes,
					globals,
				); err != nil {
					return err
				}
			}
			if err := collectExprLocals(
				c.Value,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
			}
		}
		if scopes != nil {
			scopes.catchExprScopes[e] = caseScopeIDs
		}
	case *frontend.UnaryExpr:
		return collectExprLocals(e.X, locals, slotIndex, funcs, types, module, imports, scopes, globals)
	case *frontend.BinaryExpr:
		if err := collectExprLocals(
			e.Left,
			locals,
			slotIndex,
			funcs,
			types,
			module,
			imports,
			scopes,
			globals,
		); err != nil {
			return err
		}
		return collectExprLocals(
			e.Right,
			locals,
			slotIndex,
			funcs,
			types,
			module,
			imports,
			scopes,
			globals,
		)
	case *frontend.CallExpr:
		if enumType, caseInfo, ok, err := resolveEnumCaseConstructorCall(
			e,
			types,
			module,
			imports,
		); err != nil {
			return err
		} else if ok {
			for i, arg := range e.Args {
				if i >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[i] {
					continue
				}
				if closure, ok := arg.(*frontend.ClosureExpr); ok {
					label := fmt.Sprintf("%s.%s[%d]", displayTypeName(enumType, module), caseInfo.Name, i+1)
					if err := configureClosureCaptures(
						closure,
						locals,
						funcs,
						types,
						module,
						true,
						functionTypedClosureCaptureBoundaryPhrase("function-typed assignment", label),
					); err != nil {
						return err
					}
				}
			}
		}
		if len(e.ArgLabels) == len(e.Args) {
			allLabeled := len(e.Args) > 0
			byLabel := make(map[string]frontend.Expr, len(e.Args))
			for i, label := range e.ArgLabels {
				if label == "" {
					allLabeled = false
					break
				}
				byLabel[label] = e.Args[i]
			}
			if allLabeled {
				typeRef := frontend.TypeRef{At: e.At, Kind: frontend.TypeRefNamed, Name: e.Name}
				if typeName, err := resolveTypeName(&typeRef, module, imports); err == nil {
					if info, ok := types[typeName]; ok && info.Kind == TypeStruct {
						for _, field := range info.Fields {
							if !field.FunctionTypeValue {
								continue
							}
							arg, ok := byLabel[field.Name]
							if !ok {
								continue
							}
							if closure, ok := arg.(*frontend.ClosureExpr); ok {
								label := fmt.Sprintf("%s.%s", displayTypeName(typeName, module), field.Name)
								if err := configureClosureCaptures(
									closure,
									locals,
									funcs,
									types,
									module,
									true,
									functionTypedClosureCaptureBoundaryPhrase("function-typed assignment", label),
								); err != nil {
									return err
								}
							}
						}
					}
				}
			}
		}
		for _, arg := range e.Args {
			if err := collectExprLocals(
				arg,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
		}
	case *frontend.StructLitExpr:
		typeName, err := resolveTypeName(&e.Type, module, imports)
		if err != nil {
			return err
		}
		if info, ok := types[typeName]; ok && info.Kind == TypeStruct {
			fieldsByName := map[string]FieldInfo{}
			for _, field := range info.Fields {
				fieldsByName[field.Name] = field
			}
			for _, field := range e.Fields {
				fieldInfo, ok := fieldsByName[field.Name]
				if !ok || !fieldInfo.FunctionTypeValue {
					continue
				}
				if closure, ok := field.Value.(*frontend.ClosureExpr); ok {
					label := fmt.Sprintf("%s.%s", displayTypeName(typeName, module), fieldInfo.Name)
					if err := configureClosureCaptures(
						closure,
						locals,
						funcs,
						types,
						module,
						true,
						functionTypedClosureCaptureBoundaryPhrase("function-typed assignment", label),
					); err != nil {
						return err
					}
				}
			}
		}
		for _, field := range e.Fields {
			if err := collectExprLocals(
				field.Value,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
		}
	case *frontend.FieldAccessExpr:
		if e.Base != nil {
			return collectExprLocals(
				e.Base,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			)
		}
	case *frontend.IndexExpr:
		if err := collectExprLocals(
			e.Base,
			locals,
			slotIndex,
			funcs,
			types,
			module,
			imports,
			scopes,
			globals,
		); err != nil {
			return err
		}
		return collectExprLocals(
			e.Index,
			locals,
			slotIndex,
			funcs,
			types,
			module,
			imports,
			scopes,
			globals,
		)
	case *frontend.TryExpr:
		return collectExprLocals(e.X, locals, slotIndex, funcs, types, module, imports, scopes, globals)
	case *frontend.AwaitExpr:
		return collectExprLocals(e.X, locals, slotIndex, funcs, types, module, imports, scopes, globals)
	}
	return nil
}

func collectPatternLocals(
	pattern frontend.Expr,
	scrutType string,
	locals map[string]LocalInfo,
	slotIndex *int,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	scopes *scopeInfo,
	globals map[string]GlobalInfo,
) error {
	info, err := ensureTypeInfo(scrutType, types)
	if err != nil {
		return err
	}
	switch pat := pattern.(type) {
	case *frontend.SomePatternExpr:
		if info.Kind != TypeOptional {
			return fmt.Errorf("%s: some pattern requires optional match value", frontend.FormatPos(pat.At))
		}
		if _, exists := globals[pat.Name]; exists {
			return fmt.Errorf(
				"%s: local '%s' conflicts with global '%s'",
				frontend.FormatPos(pat.At),
				pat.Name,
				pat.Name,
			)
		}
		elemInfo, err := ensureTypeInfo(info.ElemType, types)
		if err != nil {
			return err
		}
		if err := collectScopedLocal(
			pat.Name,
			LocalInfo{
				Base:      *slotIndex,
				SlotCount: elemInfo.SlotCount,
				TypeName:  info.ElemType,
				Mutable:   false,
			},
			pat.At,
			locals,
			slotIndex,
			scopes,
		); err != nil {
			return err
		}
	case *frontend.EnumCasePatternExpr:
		caseType, caseInfo, found, err := resolveEnumCasePattern(pat, types, module, imports)
		if err != nil {
			return err
		}
		if !found || caseType != scrutType {
			return fmt.Errorf("%s: enum pattern type mismatch", frontend.FormatPos(pat.At))
		}
		if err := validateEnumCasePatternPayload(pat, caseType, caseInfo, module); err != nil {
			return err
		}
		for i, binding := range pat.Bindings {
			if _, exists := globals[binding]; exists {
				return fmt.Errorf(
					"%s: local '%s' conflicts with global '%s'",
					frontend.FormatPos(pat.At),
					binding,
					binding,
				)
			}
			localInfo := LocalInfo{
				Base:      *slotIndex,
				SlotCount: caseInfo.PayloadSlots[i],
				TypeName:  caseInfo.PayloadTypes[i],
				Mutable:   false,
			}
			if i < len(caseInfo.PayloadFunctionTypes) && caseInfo.PayloadFunctionTypes[i] {
				localInfo = functionLocalInfoForEnumPayload(caseInfo, i, FunctionFieldInfo{})
				localInfo.Base = *slotIndex
				localInfo.SlotCount = caseInfo.PayloadSlots[i]
				localInfo.TypeName = caseInfo.PayloadTypes[i]
			}
			if err := collectScopedLocal(binding, localInfo, pat.At, locals, slotIndex, scopes); err != nil {
				return err
			}
		}
	}
	return nil
}

func collectLocals(
	stmts []frontend.Stmt,
	locals map[string]LocalInfo,
	slotIndex *int,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	scopes *scopeInfo,
	globals map[string]GlobalInfo,
) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if _, exists := globals[s.Name]; exists {
				return fmt.Errorf(
					"%s: local '%s' conflicts with global '%s'",
					frontend.FormatPos(s.At),
					s.Name,
					s.Name,
				)
			}
			if _, exists := locals[s.Name]; exists {
				return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(s.At), s.Name)
			}
			resolved := ""
			if s.Type.Kind == frontend.TypeRefNamed && s.Type.Name == "" {
				inferred, err := inferExprTypeForDecl(s.Value, locals, globals, funcs, types, module, imports)
				if err != nil {
					return fmt.Errorf("%s: cannot infer type for '%s': %v", frontend.FormatPos(s.At), s.Name, err)
				}
				resolved = inferred
				s.Type = frontend.TypeRef{At: s.At, Kind: frontend.TypeRefNamed, Name: inferred}
			} else {
				var err error
				resolved, err = resolveTypeName(&s.Type, module, imports)
				if err != nil {
					return err
				}
				s.Type.Name = resolved
			}
			info, err := ensureTypeInfo(resolved, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			functionValue := ""
			genericFunctionValue := false
			var functionCaptures []frontend.ClosureCapture
			var functionEscapeCaptures []frontend.ClosureCapture
			functionTouchesMutableGlobals := false
			functionReturnSnapshotAlias := false
			functionDirectSnapshotAlias := false
			functionEscapeKind := CallableEscapeKind("")
			functionHandleValue := false
			functionParamName := ""
			functionTypeValue := s.Type.Kind == frontend.TypeRefFunction
			functionParamTypes := []string(nil)
			functionParamOwnership := []string(nil)
			functionReturnType := ""
			functionReturnOwnership := ""
			functionThrowsType := ""
			functionEffects := []string(nil)
			if functionTypeValue {
				functionParamTypes, functionReturnType, functionEffects, err = functionTypeRefSignatureAndEffects(
					s.Type,
					module,
					imports,
				)
				if err != nil {
					return err
				}
				functionParamOwnership = functionTypeRefParamOwnership(s.Type)
				functionReturnOwnership = functionTypeRefReturnOwnership(s.Type)
				functionThrowsType, err = functionTypeRefThrowsType(s.Type, module, imports)
				if err != nil {
					return err
				}
			}
			functionFields, err := functionFieldsFromStructLiteral(
				s.Name,
				info,
				s.Value,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return err
			}
			if len(functionFields) == 0 {
				functionFields = functionFieldsFromStructAlias(s.Value, locals)
			}
			if len(functionFields) == 0 {
				functionFields, err = functionFieldsFromReturnCall(
					s.Value,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					resolved,
				)
				if err != nil {
					return err
				}
			}
			if len(functionFields) == 0 {
				functionFields = declaredFunctionFieldsForStructType(resolved, types)
			}
			enumPayloadFields, err := enumPayloadFieldsFromStructLiteral(
				info,
				s.Value,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return err
			}
			if len(enumPayloadFields) == 0 {
				enumPayloadFields = enumPayloadFieldsFromStructAlias(s.Value, locals)
			}
			if len(enumPayloadFields) == 0 {
				enumPayloadFields, err = enumPayloadFieldsFromReturnCall(
					s.Value,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					resolved,
				)
				if err != nil {
					return err
				}
			}
			enumPayloadFunctions, err := enumPayloadFunctionsFromConstructor(
				info,
				s.Value,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return err
			}
			if len(enumPayloadFunctions) == 0 {
				enumPayloadFunctions = enumPayloadFunctionsFromAlias(s.Value, locals)
			}
			if len(enumPayloadFunctions) == 0 {
				enumPayloadFunctions, err = enumPayloadFunctionsFromReturnCall(
					s.Value,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					resolved,
				)
				if err != nil {
					return err
				}
			}
			if s.Mutable {
				enumPayloadFunctions = nil
			}
			if closure, ok := s.Value.(*frontend.ClosureExpr); ok {
				if functionTypeValue {
					if err := validateFunctionTypeLiteralBinding(
						s.Name,
						s.Type,
						closure,
						locals,
						module,
						imports,
					); err != nil {
						return err
					}
				}
				captureBoundary := ""
				if functionTypeValue {
					captureBoundary = functionTypedClosureCaptureBoundaryPhrase(
						"function-typed assignment",
						s.Name,
					)
				}
				if err := configureClosureCaptures(
					closure,
					locals,
					funcs,
					types,
					module,
					functionTypeValue,
					captureBoundary,
				); err != nil {
					return err
				}
				if functionTypeValue && len(closure.Captures) > 0 {
					captureSlots, err := functionCaptureSlotCount(closure.Captures, types)
					if err != nil {
						return err
					}
					if captureSlots > FnPtrEnvSlotCount {
						escapeKind, handleValue, err := classifyCallableEscape(
							callableBoundaryLocal,
							closure.Captures,
							types,
						)
						if err != nil {
							return err
						}
						functionEscapeKind = escapeKind
						functionHandleValue = handleValue
					}
				}
				functionValue = qualifyName(module, closure.Name)
				genericFunctionValue = len(closure.Decl.TypeParams) > 0
				functionCaptures = append([]frontend.ClosureCapture(nil), closure.Captures...)
				functionDirectSnapshotAlias = len(closure.Captures) > 0
			} else if functionTypeValue {
				switch init := s.Value.(type) {
				case *frontend.IdentExpr:
					resolved, err := validateFunctionTypeNamedSymbolBinding(
						s.Name,
						s.Type,
						init,
						locals,
						globals,
						funcs,
						types,
						module,
						imports,
						true,
					)
					if err != nil {
						return err
					}
					functionValue = resolved
					if source, ok := locals[init.Name]; ok {
						if source.FunctionParamName != "" {
							functionParamName = source.FunctionParamName
						} else if source.FunctionTypeValue && source.FunctionValue == "" {
							functionParamName = init.Name
						}
						if len(source.FunctionCaptures) > 0 || len(source.FunctionEscapeCaptures) > 0 {
							functionCaptures = append([]frontend.ClosureCapture(nil), source.FunctionCaptures...)
							functionEscapeCaptures = append(
								[]frontend.ClosureCapture(nil),
								source.FunctionEscapeCaptures...,
							)
						}
						functionDirectSnapshotAlias = source.FunctionDirectSnapshotAlias
						functionEscapeKind = source.FunctionEscapeKind
						functionHandleValue = source.FunctionHandleValue
						if len(functionCaptures) > 0 && !functionHandleValue {
							captureSlots, err := functionCaptureSlotCount(functionCaptures, types)
							if err != nil {
								return err
							}
							if captureSlots > FnPtrEnvSlotCount {
								escapeKind, handleValue, err := classifyCallableEscape(
									callableBoundaryLocal,
									functionCaptures,
									types,
								)
								if err != nil {
									return err
								}
								functionEscapeKind = escapeKind
								functionHandleValue = handleValue
							}
						}
					}
				case *frontend.FieldAccessExpr:
					fieldInfo, ok, err := resolveFunctionFieldArgument(init, locals)
					if err != nil {
						return err
					}
					if ok && fieldInfo.FunctionValue == "" && functionFieldInfoHasTargetSet(fieldInfo) {
						targetInfo := LocalInfo{
							FunctionTypeValue:       true,
							FunctionParamTypes:      append([]string(nil), functionParamTypes...),
							FunctionParamOwnership:  append([]string(nil), functionParamOwnership...),
							FunctionReturnType:      functionReturnType,
							FunctionReturnOwnership: functionReturnOwnership,
							FunctionThrowsType:      functionThrowsType,
							FunctionEffects:         append([]string(nil), functionEffects...),
						}
						fieldSig, err := functionFieldInfoSig(fieldInfo)
						if err != nil {
							return err
						}
						if err := validateFunctionInfoAssignable(
							s.Name,
							targetInfo,
							fieldSig,
							init.At,
						); err != nil {
							return err
						}
						functionParamName = fieldInfo.FunctionParamName
						functionCaptures = append([]frontend.ClosureCapture(nil), fieldInfo.FunctionCaptures...)
						functionEscapeCaptures = append(
							[]frontend.ClosureCapture(nil),
							fieldInfo.FunctionEscapeCaptures...,
						)
						functionDirectSnapshotAlias = fieldInfo.FunctionDirectSnapshotAlias
						break
					}
					if !ok || fieldInfo.FunctionValue == "" {
						if globalInfo, globalSig, globalOK, err := resolveFunctionTypedGlobalFieldAccess(
							init,
							globals,
							funcs,
						); err != nil {
							return err
						} else if globalOK {
							if globalSig.Generic {
								return unsupportedGenericFunctionTypedLocalInitializerError(
									init.At,
									callbackArgumentName(init),
									s.Name,
								)
							}
							if err := validateFunctionTypeSymbolSignature(
								s.Name,
								s.Type,
								globalSig,
								module,
								imports,
								init.At,
							); err != nil {
								return err
							}
							functionValue = globalInfo.FunctionValue
							break
						}
						return unsupportedFunctionTypedLocalInitializerSourceError(init.At, s.Name)
					}
					targetSig, ok := funcs[fieldInfo.FunctionValue]
					if !ok {
						return fmt.Errorf(
							"%s: unknown function symbol '%s'",
							frontend.FormatPos(init.At),
							fieldInfo.FunctionValue,
						)
					}
					if targetSig.Generic {
						return unsupportedGenericFunctionTypedLocalInitializerError(
							init.At,
							callbackArgumentName(init),
							s.Name,
						)
					}
					fieldSig, err := functionFieldInfoSig(fieldInfo)
					if err != nil {
						return err
					}
					if err := validateFunctionTypeSymbolSignature(
						s.Name,
						s.Type,
						fieldSig,
						module,
						imports,
						init.At,
					); err != nil {
						return err
					}
					functionValue = fieldInfo.FunctionValue
					functionParamName = fieldInfo.FunctionParamName
					functionCaptures = append([]frontend.ClosureCapture(nil), fieldInfo.FunctionCaptures...)
					functionEscapeCaptures = append(
						[]frontend.ClosureCapture(nil),
						fieldInfo.FunctionEscapeCaptures...,
					)
					functionDirectSnapshotAlias = fieldInfo.FunctionDirectSnapshotAlias
					functionEscapeKind = fieldInfo.FunctionEscapeKind
					functionHandleValue = fieldInfo.FunctionHandleValue
				case *frontend.CallExpr:
					resolved, err := validateFunctionTypeReturnCallBinding(
						s.Name,
						s.Type,
						init,
						funcs,
						module,
						imports,
					)
					if err != nil {
						return err
					}
					functionValue = resolved
					metadataValue, metadataCaptures, metadataEscapeCaptures, metadataParamName, err := functionAssignmentMetadataWithReturnParamRefs(
						init,
						locals,
						globals,
						funcs,
						types,
						module,
						imports,
					)
					if err != nil {
						return err
					}
					if metadataValue != "" {
						functionValue = metadataValue
					}
					functionParamName = metadataParamName
					functionCaptures = append([]frontend.ClosureCapture(nil), metadataCaptures...)
					functionEscapeCaptures = append([]frontend.ClosureCapture(nil), metadataEscapeCaptures...)
					functionReturnSnapshotAlias = isFunctionReturnSnapshotAlias(
						init,
						funcs,
						metadataCaptures,
						metadataEscapeCaptures,
						metadataParamName,
					)
					functionDirectSnapshotAlias = false
					if callSig, ok := funcs[init.Name]; ok && callSig.ReturnFunctionHandleValue {
						functionEscapeKind = callSig.ReturnFunctionEscapeKind
						functionHandleValue = callSig.ReturnFunctionHandleValue
					}
				default:
					return unsupportedFunctionTypedLocalInitializerSourceError(s.At, s.Name)
				}
				functionTouchesMutableGlobals, err = functionAssignmentValueTouchesMutableGlobals(
					s.Value,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
				)
				if err != nil {
					return err
				}
			}
			localSlotCount := info.SlotCount
			if functionTypeValue && functionHandleValue {
				localSlotCount = CallableHandleSlotCount
			}
			surfaceFramePixelsSource := ""
			if source, ok := surfaceFramePixelsSourceExpr(s.Value, locals, globals, types, nil); ok {
				surfaceFramePixelsSource = source
			}
			locals[s.Name] = LocalInfo{
				Base:                          *slotIndex,
				SlotCount:                     localSlotCount,
				TypeName:                      resolved,
				Mutable:                       s.Mutable,
				Const:                         s.Const,
				FunctionValue:                 functionValue,
				FunctionParamName:             functionParamName,
				GenericFunctionValue:          genericFunctionValue,
				FunctionCaptures:              functionCaptures,
				FunctionEscapeCaptures:        functionEscapeCaptures,
				FunctionTouchesMutableGlobals: functionTouchesMutableGlobals,
				FunctionReturnSnapshotAlias:   functionReturnSnapshotAlias,
				FunctionDirectSnapshotAlias:   functionDirectSnapshotAlias,
				FunctionEscapeKind:            functionEscapeKind,
				FunctionHandleValue:           functionHandleValue,
				FunctionTypeValue:             functionTypeValue,
				FunctionParamTypes:            functionParamTypes,
				FunctionParamOwnership:        functionParamOwnership,
				FunctionReturnType:            functionReturnType,
				FunctionReturnOwnership:       functionReturnOwnership,
				FunctionThrowsType:            functionThrowsType,
				FunctionEffects:               functionEffects,
				FunctionFields:                functionFields,
				EnumPayloadFunctions:          enumPayloadFunctions,
				EnumPayloadFields:             enumPayloadFields,
				SurfaceFramePixelsSource:      surfaceFramePixelsSource,
			}
			if scopes != nil {
				scopes.localScopes[s.Name] = scopes.currentScopeID()
			}
			*slotIndex += localSlotCount
			if err := collectExprLocals(
				s.Value,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
		case *frontend.IslandStmt:
			if _, exists := globals[s.Name]; exists {
				return fmt.Errorf(
					"%s: local '%s' conflicts with global '%s'",
					frontend.FormatPos(s.At),
					s.Name,
					s.Name,
				)
			}
			if _, exists := locals[s.Name]; exists {
				return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(s.At), s.Name)
			}
			islandInfo, err := ensureTypeInfo("island", types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			scopeID := regionNone
			if scopes != nil {
				scopeID = scopes.enterScope()
				scopes.localScopes[s.Name] = scopeID
				scopes.islandScopes[s.Name] = scopeID
			}
			locals[s.Name] = LocalInfo{
				Base:      *slotIndex,
				SlotCount: islandInfo.SlotCount,
				TypeName:  "island",
				Mutable:   false,
			}
			*slotIndex += islandInfo.SlotCount
			if err := collectLocals(
				s.Body,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
			}
		case *frontend.IfStmt:
			if err := collectExprLocals(
				s.Cond,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
			thenScopeID := regionNone
			elseScopeID := regionNone
			if scopes != nil {
				thenScopeID = scopes.enterScope()
			}
			if err := collectLocals(
				s.Then,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
			}
			if len(s.Else) > 0 {
				if scopes != nil {
					elseScopeID = scopes.enterScope()
				}
				if err := collectLocals(
					s.Else,
					locals,
					slotIndex,
					funcs,
					types,
					module,
					imports,
					scopes,
					globals,
				); err != nil {
					return err
				}
				if scopes != nil {
					scopes.exitScope()
				}
			}
			if scopes != nil {
				scopes.ifScopes[s] = branchScopeInfo{thenID: thenScopeID, elseID: elseScopeID}
			}
		case *frontend.IfLetStmt:
			if err := collectExprLocals(
				s.Value,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
			valueType, err := inferExprTypeForDecl(s.Value, locals, globals, funcs, types, module, imports)
			if err != nil {
				return fmt.Errorf("%s: cannot infer if-let value type: %v", frontend.FormatPos(s.At), err)
			}
			valueInfo, err := ensureTypeInfo(valueType, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			if s.Pattern == nil && valueInfo.Kind != TypeOptional {
				return fmt.Errorf(
					"%s: if let requires optional value, got '%s'",
					frontend.FormatPos(s.At),
					valueType,
				)
			}
			if s.Pattern != nil && valueInfo.Kind != TypeOptional && valueInfo.Kind != TypeEnum {
				return fmt.Errorf(
					"%s: if let pattern requires optional or enum value, got '%s'",
					frontend.FormatPos(s.At),
					valueType,
				)
			}
			if s.Pattern == nil {
				if _, exists := globals[s.Name]; exists {
					return fmt.Errorf(
						"%s: local '%s' conflicts with global '%s'",
						frontend.FormatPos(s.At),
						s.Name,
						s.Name,
					)
				}
			}
			elemInfo, err := ensureTypeInfo(valueInfo.ElemType, types)
			if s.Pattern == nil && err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			s.ValueLocal = uniqueHiddenLocal("__iflet_value", s.At, locals)
			locals[s.ValueLocal] = LocalInfo{
				Base:      *slotIndex,
				SlotCount: valueInfo.SlotCount,
				TypeName:  valueType,
				Mutable:   false,
			}
			if scopes != nil {
				scopes.localScopes[s.ValueLocal] = scopes.currentScopeID()
			}
			*slotIndex += valueInfo.SlotCount
			thenScopeID := regionNone
			elseScopeID := regionNone
			if scopes != nil {
				thenScopeID = scopes.enterScope()
			}
			if s.Pattern == nil {
				if err := collectScopedLocal(s.Name, LocalInfo{
					Base:      *slotIndex,
					SlotCount: elemInfo.SlotCount,
					TypeName:  valueInfo.ElemType,
					Mutable:   false,
				}, s.At, locals, slotIndex, scopes); err != nil {
					return err
				}
			} else if err := collectPatternLocals(
				s.Pattern,
				valueType,
				locals,
				slotIndex,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
			if err := collectLocals(
				s.Then,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
			}
			if len(s.Else) > 0 {
				if scopes != nil {
					elseScopeID = scopes.enterScope()
				}
				if err := collectLocals(
					s.Else,
					locals,
					slotIndex,
					funcs,
					types,
					module,
					imports,
					scopes,
					globals,
				); err != nil {
					return err
				}
				if scopes != nil {
					scopes.exitScope()
				}
			}
			if scopes != nil {
				scopes.ifLetScopes[s] = branchScopeInfo{thenID: thenScopeID, elseID: elseScopeID}
			}
		case *frontend.WhileStmt:
			if err := collectExprLocals(
				s.Cond,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
			scopeID := regionNone
			if scopes != nil {
				scopeID = scopes.enterScope()
			}
			if err := collectLocals(
				s.Body,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
				scopes.whileScopes[s] = scopeID
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				if err := collectExprLocals(
					s.Iterable,
					locals,
					slotIndex,
					funcs,
					types,
					module,
					imports,
					scopes,
					globals,
				); err != nil {
					return err
				}
			} else {
				if err := collectExprLocals(
					s.Start,
					locals,
					slotIndex,
					funcs,
					types,
					module,
					imports,
					scopes,
					globals,
				); err != nil {
					return err
				}
				if err := collectExprLocals(
					s.End,
					locals,
					slotIndex,
					funcs,
					types,
					module,
					imports,
					scopes,
					globals,
				); err != nil {
					return err
				}
			}
			scopeID := regionNone
			if scopes != nil {
				scopeID = scopes.enterScope()
			}
			if _, exists := globals[s.Name]; exists {
				return fmt.Errorf(
					"%s: local '%s' conflicts with global '%s'",
					frontend.FormatPos(s.At),
					s.Name,
					s.Name,
				)
			}
			if _, exists := locals[s.Name]; exists {
				return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(s.At), s.Name)
			}
			loopType := "i32"
			var iterableInfo *TypeInfo
			if s.Iterable != nil {
				iterType, err := inferExprTypeForDecl(
					s.Iterable,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
				)
				if err != nil {
					return fmt.Errorf("%s: cannot infer for collection type: %v", frontend.FormatPos(s.At), err)
				}
				elemType, err := collectionElementType(iterType, types)
				if err != nil {
					return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
				}
				iterableInfo, err = ensureTypeInfo(iterType, types)
				if err != nil {
					return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
				}
				loopType = elemType
			}
			info, err := ensureTypeInfo(loopType, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			locals[s.Name] = LocalInfo{
				Base:      *slotIndex,
				SlotCount: info.SlotCount,
				TypeName:  loopType,
				Mutable:   false,
			}
			if scopes != nil {
				scopes.localScopes[s.Name] = scopes.currentScopeID()
			}
			*slotIndex += info.SlotCount
			if s.Iterable != nil {
				s.IterableLocal = uniqueHiddenLocal("__for_iter", s.At, locals)
				locals[s.IterableLocal] = LocalInfo{
					Base:      *slotIndex,
					SlotCount: iterableInfo.SlotCount,
					TypeName:  iterableInfo.Name,
					Mutable:   false,
				}
				if scopes != nil {
					scopes.localScopes[s.IterableLocal] = scopes.currentScopeID()
				}
				*slotIndex += iterableInfo.SlotCount
				s.IndexLocal = uniqueHiddenLocal("__for_index", s.At, locals)
				indexInfo := types["i32"]
				locals[s.IndexLocal] = LocalInfo{
					Base:      *slotIndex,
					SlotCount: indexInfo.SlotCount,
					TypeName:  "i32",
					Mutable:   false,
				}
				if scopes != nil {
					scopes.localScopes[s.IndexLocal] = scopes.currentScopeID()
				}
				*slotIndex += indexInfo.SlotCount
			}
			s.EndLocal = uniqueHiddenLocal("__for_end", s.At, locals)
			locals[s.EndLocal] = LocalInfo{
				Base:      *slotIndex,
				SlotCount: 1,
				TypeName:  "i32",
				Mutable:   false,
			}
			if scopes != nil {
				scopes.localScopes[s.EndLocal] = scopes.currentScopeID()
			}
			*slotIndex += info.SlotCount
			if err := collectLocals(
				s.Body,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
				scopes.forScopes[s] = scopeID
			}
		case *frontend.MatchStmt:
			if err := collectExprLocals(
				s.Value,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
			scrutType, err := inferExprTypeForDecl(s.Value, locals, globals, funcs, types, module, imports)
			if err != nil {
				return fmt.Errorf("%s: cannot infer match value type: %v", frontend.FormatPos(s.At), err)
			}
			info, err := ensureTypeInfo(scrutType, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			if info.SlotCount != 1 && info.Kind != TypeOptional && info.Kind != TypeEnum {
				return fmt.Errorf("%s: match value must be single-slot", frontend.FormatPos(s.At))
			}
			s.ScrutineeLocal = uniqueHiddenLocal("__match_value", s.At, locals)
			locals[s.ScrutineeLocal] = LocalInfo{
				Base:      *slotIndex,
				SlotCount: info.SlotCount,
				TypeName:  scrutType,
				Mutable:   false,
			}
			if scopes != nil {
				scopes.localScopes[s.ScrutineeLocal] = scopes.currentScopeID()
			}
			*slotIndex += info.SlotCount
			caseScopeIDs := make([]int, len(s.Cases))
			scrutineeFunctionPayloads, err := enumPayloadFunctionValuesForMatchExpr(
				s.Value,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				scrutType,
			)
			if err != nil {
				return err
			}
			for i, c := range s.Cases {
				if scopes != nil {
					caseScopeIDs[i] = scopes.enterScope()
				} else {
					caseScopeIDs[i] = regionNone
				}
				if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
					if info.Kind != TypeOptional {
						return fmt.Errorf(
							"%s: some pattern requires optional match value",
							frontend.FormatPos(some.At),
						)
					}
					if _, exists := globals[some.Name]; exists {
						return fmt.Errorf(
							"%s: local '%s' conflicts with global '%s'",
							frontend.FormatPos(some.At),
							some.Name,
							some.Name,
						)
					}
					elemInfo, err := ensureTypeInfo(info.ElemType, types)
					if err != nil {
						return fmt.Errorf("%s: %v", frontend.FormatPos(some.At), err)
					}
					if err := collectScopedLocal(some.Name, LocalInfo{
						Base:      *slotIndex,
						SlotCount: elemInfo.SlotCount,
						TypeName:  info.ElemType,
						Mutable:   false,
					}, some.At, locals, slotIndex, scopes); err != nil {
						return err
					}
				}
				if enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr); ok {
					caseType, caseInfo, found, err := resolveEnumCasePattern(enumPat, types, module, imports)
					if err != nil {
						return err
					}
					if !found || caseType != scrutType {
						return fmt.Errorf("%s: enum pattern type mismatch", frontend.FormatPos(enumPat.At))
					}
					if err := validateEnumCasePatternPayload(enumPat, caseType, caseInfo, module); err != nil {
						return err
					}
					for j, binding := range enumPat.Bindings {
						if _, exists := globals[binding]; exists {
							return fmt.Errorf(
								"%s: local '%s' conflicts with global '%s'",
								frontend.FormatPos(enumPat.At),
								binding,
								binding,
							)
						}
						slots := 1
						if j < len(caseInfo.PayloadSlots) {
							slots = caseInfo.PayloadSlots[j]
						}
						localInfo := LocalInfo{
							Base:      *slotIndex,
							SlotCount: slots,
							TypeName:  caseInfo.PayloadTypes[j],
							Mutable:   false,
						}
						if j < len(caseInfo.PayloadFunctionTypes) && caseInfo.PayloadFunctionTypes[j] {
							localInfo = functionLocalInfoForEnumPayload(caseInfo, j, FunctionFieldInfo{})
							localInfo.Base = *slotIndex
							localInfo.SlotCount = slots
							localInfo.TypeName = caseInfo.PayloadTypes[j]
						}
						if err := collectScopedLocal(
							binding,
							localInfo,
							enumPat.At,
							locals,
							slotIndex,
							scopes,
						); err != nil {
							return err
						}
					}
					if err := bindEnumPatternFunctionPayloadLocals(
						enumPat,
						scrutineeFunctionPayloads,
						locals,
						types,
						module,
						imports,
					); err != nil {
						return err
					}
				}
				if c.Guard != nil {
					if err := collectExprLocals(
						c.Guard,
						locals,
						slotIndex,
						funcs,
						types,
						module,
						imports,
						scopes,
						globals,
					); err != nil {
						return err
					}
				}
				if err := collectLocals(
					c.Body,
					locals,
					slotIndex,
					funcs,
					types,
					module,
					imports,
					scopes,
					globals,
				); err != nil {
					return err
				}
				if scopes != nil {
					scopes.exitScope()
				}
			}
			if scopes != nil {
				scopes.matchCaseScopes[s] = caseScopeIDs
			}
		case *frontend.UnsafeStmt:
			scopeID := regionNone
			if scopes != nil {
				scopeID = scopes.enterScope()
			}
			if err := collectLocals(
				s.Body,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
				scopes.unsafeScopes[s] = scopeID
			}
		case *frontend.DeferStmt:
			scopeID := regionNone
			if scopes != nil {
				scopeID = scopes.enterScope()
			}
			if err := collectLocals(
				s.Body,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
				scopes.deferScopes[s] = scopeID
			}
		case *frontend.ExprStmt:
			if err := collectExprLocals(
				s.Expr,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
		case *frontend.ReturnStmt:
			if err := collectExprLocals(
				s.Value,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
		case *frontend.ThrowStmt:
			if err := collectExprLocals(
				s.Value,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
		case *frontend.PrintStmt:
			if err := collectExprLocals(
				s.Value,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
		case *frontend.FreeStmt:
			if err := collectExprLocals(
				s.Value,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
		case *frontend.AssignStmt:
			if err := rejectRepresentationMetadataExprAssignment(
				s.Target,
				locals,
				globals,
				types,
			); err != nil {
				return err
			}
			if err := collectExprLocals(
				s.Target,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
			if err := collectExprLocals(
				s.Value,
				locals,
				slotIndex,
				funcs,
				types,
				module,
				imports,
				scopes,
				globals,
			); err != nil {
				return err
			}
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				if local, exists := locals[id.Name]; exists && local.Mutable {
					if local.FunctionTypeValue {
						if _, ok := s.Value.(*frontend.ClosureExpr); ok {
							_ = updateFunctionTypedLocalAssignmentMetadata(
								id.Name,
								s.Value,
								locals,
								globals,
								funcs,
								types,
								module,
								imports,
							)
							local = locals[id.Name]
						}
					}
					info, infoOK := types[local.TypeName]
					if infoOK && info.Kind == TypeEnum {
						payloads, err := enumPayloadFunctionsFromConstructor(
							info,
							s.Value,
							locals,
							globals,
							funcs,
							types,
							module,
							imports,
						)
						if err != nil {
							return err
						}
						if len(payloads) == 0 {
							payloads = enumPayloadFunctionsFromAlias(s.Value, locals)
						}
						if len(payloads) == 0 {
							payloads, err = enumPayloadFunctionsFromReturnCall(
								s.Value,
								locals,
								globals,
								funcs,
								types,
								module,
								imports,
								local.TypeName,
							)
							if err != nil {
								return err
							}
						}
						local.EnumPayloadFunctions = payloads
						locals[id.Name] = local
					}
					if infoOK && info.Kind == TypeStruct {
						payloadFields, err := enumPayloadFieldsFromReturnedStructExpr(
							local.TypeName,
							s.Value,
							locals,
							globals,
							funcs,
							types,
							module,
							imports,
						)
						if err != nil {
							return err
						}
						if len(payloadFields) > 0 || len(local.EnumPayloadFields) > 0 {
							local.EnumPayloadFields = cloneFunctionFieldMap(payloadFields)
							locals[id.Name] = local
						}
					}
				}
			} else {
				if targetName, fieldInfo, ok, err := functionFieldLocalInfoFromExpr(
					s.Target,
					locals,
					types,
				); err != nil {
					return err
				} else if ok {
					if _, closure := s.Value.(*frontend.ClosureExpr); closure {
						_ = updateFunctionTypedFieldAssignmentMetadata(
							targetName,
							fieldInfo,
							s.Value,
							locals,
							globals,
							funcs,
							types,
							module,
							imports,
						)
					}
				}
				targetType, err := inferExprTypeForDecl(
					s.Target,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
				)
				if err != nil {
					return err
				}
				if info, ok := types[targetType]; ok && (info.Kind == TypeEnum || info.Kind == TypeStruct) {
					if err := updateEnumPayloadStructFieldAssignmentMetadata(
						s.Target,
						targetType,
						s.Value,
						locals,
						globals,
						funcs,
						types,
						module,
						imports,
					); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// ---- checker_policy.go ----

func validateGenericFuncDecl(
	fn *frontend.FuncDecl,
	module string,
	imports map[string]string,
	protocolInfos map[string]genericProtocolInfo,
	types map[string]*TypeInfo,
) error {
	if len(fn.TypeParams) == 0 {
		return nil
	}
	params := map[string]struct{}{}
	for _, name := range fn.TypeParams {
		params[name] = struct{}{}
	}
	boundParams := map[string]struct{}{}
	for _, bound := range fn.TypeParamBounds {
		if _, ok := params[bound.Name]; !ok {
			return fmt.Errorf(
				"%s: generic bound references unknown type parameter '%s'",
				frontend.FormatPos(bound.At),
				bound.Name,
			)
		}
		boundParams[bound.Name] = struct{}{}
		if bound.Bound.Kind != frontend.TypeRefNamed || len(bound.Bound.TypeArgs) > 0 {
			return fmt.Errorf(
				"%s: generic bound for '%s' must name a protocol",
				frontend.FormatPos(bound.Bound.At),
				bound.Name,
			)
		}
		boundRef := bound.Bound
		resolved, err := resolveTypeName(&boundRef, module, imports)
		if err != nil {
			return err
		}
		proto, ok := protocolInfos[resolved]
		if !ok {
			if _, isType := types[resolved]; isType {
				return fmt.Errorf(
					"%s: generic bound '%s' for '%s' must name a protocol, got non-protocol type '%s'",
					frontend.FormatPos(bound.Bound.At),
					displayTypeName(resolved, module),
					bound.Name,
					displayTypeName(resolved, module),
				)
			}
			return fmt.Errorf(
				"%s: unknown protocol bound '%s' for generic parameter '%s'",
				frontend.FormatPos(bound.Bound.At),
				displayTypeName(resolved, module),
				bound.Name,
			)
		}
		if !symbolBelongsToModule(resolved, module) && !proto.public {
			return fmt.Errorf(
				"%s: private protocol '%s' is not visible from module '%s'",
				frontend.FormatPos(bound.Bound.At),
				resolved,
				module,
			)
		}
	}
	if err := validateGenericTypeRef(fn.ReturnType, params); err != nil {
		return fmt.Errorf("%s: %v", frontend.FormatPos(fn.ReturnType.At), err)
	}
	if fn.HasThrows {
		if err := validateGenericTypeRef(fn.Throws, params); err != nil {
			return fmt.Errorf("%s: %v", frontend.FormatPos(fn.Throws.At), err)
		}
	}
	for _, param := range fn.Params {
		if err := validateGenericTypeRef(param.Type, params); err != nil {
			return fmt.Errorf("%s: %v", frontend.FormatPos(param.At), err)
		}
	}
	if err := validateGenericBoundRequirementCalls(fn.Body, boundParams); err != nil {
		return err
	}
	return nil
}

func validateGenericBoundRequirementCalls(
	stmts []frontend.Stmt,
	boundParams map[string]struct{},
) error {
	if len(boundParams) == 0 {
		return nil
	}
	return walkGenericBoundRequirementCallsInStmts(stmts, boundParams)
}

func walkGenericBoundRequirementCallsInStmts(
	stmts []frontend.Stmt,
	boundParams map[string]struct{},
) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Value, boundParams); err != nil {
				return err
			}
		case *frontend.ThrowStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Value, boundParams); err != nil {
				return err
			}
		case *frontend.DeferStmt:
			if err := walkGenericBoundRequirementCallsInStmts(s.Body, boundParams); err != nil {
				return err
			}
		case *frontend.PrintStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Value, boundParams); err != nil {
				return err
			}
		case *frontend.ExpectStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Cond, boundParams); err != nil {
				return err
			}
		case *frontend.FreeStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Value, boundParams); err != nil {
				return err
			}
		case *frontend.LetStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Value, boundParams); err != nil {
				return err
			}
		case *frontend.AssignStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Target, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInExpr(s.Value, boundParams); err != nil {
				return err
			}
		case *frontend.IfStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Cond, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInStmts(s.Then, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInStmts(s.Else, boundParams); err != nil {
				return err
			}
		case *frontend.IfLetStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Pattern, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInExpr(s.Value, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInStmts(s.Then, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInStmts(s.Else, boundParams); err != nil {
				return err
			}
		case *frontend.WhileStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Cond, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInStmts(s.Body, boundParams); err != nil {
				return err
			}
		case *frontend.ForRangeStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Start, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInExpr(s.End, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInExpr(s.Iterable, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInStmts(s.Body, boundParams); err != nil {
				return err
			}
		case *frontend.MatchStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Value, boundParams); err != nil {
				return err
			}
			for i := range s.Cases {
				if err := walkGenericBoundRequirementCallsInExpr(s.Cases[i].Pattern, boundParams); err != nil {
					return err
				}
				if err := walkGenericBoundRequirementCallsInExpr(s.Cases[i].Guard, boundParams); err != nil {
					return err
				}
				if err := walkGenericBoundRequirementCallsInStmts(s.Cases[i].Body, boundParams); err != nil {
					return err
				}
			}
		case *frontend.UnsafeStmt:
			if err := walkGenericBoundRequirementCallsInStmts(s.Body, boundParams); err != nil {
				return err
			}
		case *frontend.IslandStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Size, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInStmts(s.Body, boundParams); err != nil {
				return err
			}
		case *frontend.ExprStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Expr, boundParams); err != nil {
				return err
			}
		}
	}
	return nil
}

func walkGenericBoundRequirementCallsInExpr(
	expr frontend.Expr,
	boundParams map[string]struct{},
) error {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *frontend.CallExpr:
		if parts := strings.Split(e.Name, "."); len(parts) == 2 {
			if _, ok := boundParams[parts[0]]; ok {
				return fmt.Errorf(("%s: calling protocol requirement '%s' through generic bound " +
					"'%s' is not supported in this MVP; specialize the operation " +
					"outside the generic"), frontend.FormatPos(e.At), parts[1], parts[0])
			}
		}
		for _, arg := range e.Args {
			if err := walkGenericBoundRequirementCallsInExpr(arg, boundParams); err != nil {
				return err
			}
		}
	case *frontend.MatchExpr:
		if err := walkGenericBoundRequirementCallsInExpr(e.Value, boundParams); err != nil {
			return err
		}
		for i := range e.Cases {
			if err := walkGenericBoundRequirementCallsInExpr(e.Cases[i].Pattern, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInExpr(e.Cases[i].Guard, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInExpr(e.Cases[i].Value, boundParams); err != nil {
				return err
			}
		}
	case *frontend.CatchExpr:
		if err := walkGenericBoundRequirementCallsInExpr(e.Call, boundParams); err != nil {
			return err
		}
		for i := range e.Cases {
			if err := walkGenericBoundRequirementCallsInExpr(e.Cases[i].Pattern, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInExpr(e.Cases[i].Guard, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInExpr(e.Cases[i].Value, boundParams); err != nil {
				return err
			}
		}
	case *frontend.UnaryExpr:
		return walkGenericBoundRequirementCallsInExpr(e.X, boundParams)
	case *frontend.BinaryExpr:
		if err := walkGenericBoundRequirementCallsInExpr(e.Left, boundParams); err != nil {
			return err
		}
		return walkGenericBoundRequirementCallsInExpr(e.Right, boundParams)
	case *frontend.FieldAccessExpr:
		return walkGenericBoundRequirementCallsInExpr(e.Base, boundParams)
	case *frontend.IndexExpr:
		if err := walkGenericBoundRequirementCallsInExpr(e.Base, boundParams); err != nil {
			return err
		}
		return walkGenericBoundRequirementCallsInExpr(e.Index, boundParams)
	case *frontend.TryExpr:
		return walkGenericBoundRequirementCallsInExpr(e.X, boundParams)
	case *frontend.AwaitExpr:
		return walkGenericBoundRequirementCallsInExpr(e.X, boundParams)
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if err := walkGenericBoundRequirementCallsInExpr(field.Value, boundParams); err != nil {
				return err
			}
		}
	case *frontend.ClosureExpr:
		if e.Decl != nil {
			return walkGenericBoundRequirementCallsInStmts(e.Decl.Body, boundParams)
		}
	}
	return nil
}

func validateSemanticClauses(fn *frontend.FuncDecl) error {
	return semanticspolicy.ValidateSemanticClauses(
		fn,
		constI32,
		privacyDiagnosticf,
		budgetDiagnosticf,
	)
}

func validateBudgetContexts(world *module.World, funcs map[string]FuncSig) error {
	if world == nil {
		return nil
	}
	for _, file := range world.Files {
		if file == nil || world.InterfaceModules[file.Module] {
			continue
		}
		imports, err := collectImportAliases(file)
		if err != nil {
			return err
		}
		for _, fn := range file.Funcs {
			if fn == nil || len(fn.TypeParams) > 0 {
				continue
			}
			callerName := checkedFuncFullName(file.Module, fn)
			callerSig, ok := funcs[callerName]
			if !ok {
				continue
			}
			if err := validateBudgetContextsInStmts(
				fn.Body,
				callerName,
				callerSig,
				funcs,
				file.Module,
				imports,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateBudgetContextsInStmts(
	stmts []frontend.Stmt,
	callerName string,
	callerSig FuncSig,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if err := validateBudgetContextsInExpr(
				s.Value,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
		case *frontend.AssignStmt:
			if err := validateBudgetContextsInExpr(
				s.Target,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
			if err := validateBudgetContextsInExpr(
				s.Value,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
		case *frontend.ExprStmt:
			if err := validateBudgetContextsInExpr(
				s.Expr,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
		case *frontend.ReturnStmt:
			if err := validateBudgetContextsInExpr(
				s.Value,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
		case *frontend.ThrowStmt:
			if err := validateBudgetContextsInExpr(
				s.Value,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
		case *frontend.PrintStmt:
			if err := validateBudgetContextsInExpr(
				s.Value,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
		case *frontend.ExpectStmt:
			if err := validateBudgetContextsInExpr(
				s.Cond,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
		case *frontend.FreeStmt:
			if err := validateBudgetContextsInExpr(
				s.Value,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
		case *frontend.IfStmt:
			if err := validateBudgetContextsInExpr(
				s.Cond,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
			if err := validateBudgetContextsInStmts(
				s.Then,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
			if err := validateBudgetContextsInStmts(
				s.Else,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
		case *frontend.IfLetStmt:
			if err := validateBudgetContextsInExpr(
				s.Value,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
			if err := validateBudgetContextsInStmts(
				s.Then,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
			if err := validateBudgetContextsInStmts(
				s.Else,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
		case *frontend.WhileStmt:
			if err := validateBudgetContextsInExpr(
				s.Cond,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
			if err := validateBudgetContextsInStmts(
				s.Body,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				if err := validateBudgetContextsInExpr(
					s.Iterable,
					callerName,
					callerSig,
					funcs,
					module,
					imports,
				); err != nil {
					return err
				}
			} else {
				if err := validateBudgetContextsInExpr(
					s.Start,
					callerName,
					callerSig,
					funcs,
					module,
					imports,
				); err != nil {
					return err
				}
				if err := validateBudgetContextsInExpr(
					s.End,
					callerName,
					callerSig,
					funcs,
					module,
					imports,
				); err != nil {
					return err
				}
			}
			if err := validateBudgetContextsInStmts(
				s.Body,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
		case *frontend.MatchStmt:
			if err := validateBudgetContextsInExpr(
				s.Value,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
			for _, c := range s.Cases {
				if !c.Default {
					if err := validateBudgetContextsInExpr(
						c.Pattern,
						callerName,
						callerSig,
						funcs,
						module,
						imports,
					); err != nil {
						return err
					}
				}
				if err := validateBudgetContextsInExpr(
					c.Guard,
					callerName,
					callerSig,
					funcs,
					module,
					imports,
				); err != nil {
					return err
				}
				if err := validateBudgetContextsInStmts(
					c.Body,
					callerName,
					callerSig,
					funcs,
					module,
					imports,
				); err != nil {
					return err
				}
			}
		case *frontend.UnsafeStmt:
			if err := validateBudgetContextsInStmts(
				s.Body,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
		case *frontend.DeferStmt:
			if err := validateBudgetContextsInStmts(
				s.Body,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
		case *frontend.IslandStmt:
			if err := validateBudgetContextsInExpr(
				s.Size,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
			if err := validateBudgetContextsInStmts(
				s.Body,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateBudgetContextsInExpr(
	expr frontend.Expr,
	callerName string,
	callerSig FuncSig,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
) error {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *frontend.CallExpr:
		resolved := e.Name
		if builtin, ok := ResolveBuiltinAlias(resolved); ok {
			resolved = builtin
		}
		if err := validateBudgetSpawnContext(
			e,
			resolved,
			callerName,
			callerSig,
			funcs,
			module,
			imports,
		); err != nil {
			return err
		}
		if targetSig, ok := funcs[resolved]; ok {
			if err := validateBudgetContextEdge(
				e.At,
				callerName,
				callerSig,
				"call to '"+resolved+"'",
				targetSig,
			); err != nil {
				return err
			}
		}
		for _, arg := range e.Args {
			if err := validateBudgetContextsInExpr(
				arg,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
		}
	case *frontend.FieldAccessExpr:
		return validateBudgetContextsInExpr(e.Base, callerName, callerSig, funcs, module, imports)
	case *frontend.IndexExpr:
		if err := validateBudgetContextsInExpr(
			e.Base,
			callerName,
			callerSig,
			funcs,
			module,
			imports,
		); err != nil {
			return err
		}
		return validateBudgetContextsInExpr(e.Index, callerName, callerSig, funcs, module, imports)
	case *frontend.BinaryExpr:
		if err := validateBudgetContextsInExpr(
			e.Left,
			callerName,
			callerSig,
			funcs,
			module,
			imports,
		); err != nil {
			return err
		}
		return validateBudgetContextsInExpr(e.Right, callerName, callerSig, funcs, module, imports)
	case *frontend.UnaryExpr:
		return validateBudgetContextsInExpr(e.X, callerName, callerSig, funcs, module, imports)
	case *frontend.TryExpr:
		return validateBudgetContextsInExpr(e.X, callerName, callerSig, funcs, module, imports)
	case *frontend.AwaitExpr:
		return validateBudgetContextsInExpr(e.X, callerName, callerSig, funcs, module, imports)
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if err := validateBudgetContextsInExpr(
				field.Value,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
		}
	case *frontend.MatchExpr:
		if err := validateBudgetContextsInExpr(
			e.Value,
			callerName,
			callerSig,
			funcs,
			module,
			imports,
		); err != nil {
			return err
		}
		for _, c := range e.Cases {
			if err := validateBudgetContextsInExpr(
				c.Pattern,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
			if err := validateBudgetContextsInExpr(
				c.Guard,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
			if err := validateBudgetContextsInExpr(
				c.Value,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
		}
	case *frontend.CatchExpr:
		if err := validateBudgetContextsInExpr(
			e.Call,
			callerName,
			callerSig,
			funcs,
			module,
			imports,
		); err != nil {
			return err
		}
		for _, c := range e.Cases {
			if err := validateBudgetContextsInExpr(
				c.Pattern,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
			if err := validateBudgetContextsInExpr(
				c.Guard,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
			if err := validateBudgetContextsInExpr(
				c.Value,
				callerName,
				callerSig,
				funcs,
				module,
				imports,
			); err != nil {
				return err
			}
		}
	case *frontend.ClosureExpr:
		// Closure declarations are validated as synthetic functions in file.Funcs.
		// Re-validating here under the outer caller budget creates false positives
		// for closures that declare their own budget context.
		return nil
	}
	return nil
}

func validateBudgetSpawnContext(
	call *frontend.CallExpr,
	resolved string,
	callerName string,
	callerSig FuncSig,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
) error {
	workerArg := -1
	contextName := ""
	switch resolved {
	case "core.spawn":
		workerArg = 0
		contextName = "spawn target"
	case "core.spawn_remote":
		workerArg = 1
		contextName = "spawn_remote target"
	case "core.task_spawn_i32", "core.task_spawn_i32_typed":
		workerArg = 0
		contextName = strings.TrimPrefix(resolved, "core.") + " target"
	case "core.task_spawn_group_i32", "core.task_spawn_group_i32_typed":
		workerArg = 1
		contextName = strings.TrimPrefix(resolved, "core.") + " target"
	}
	if workerArg < 0 || workerArg >= len(call.Args) {
		return nil
	}
	lit, ok := call.Args[workerArg].(*frontend.StringLitExpr)
	if !ok || len(lit.Value) == 0 {
		return nil
	}
	target, err := resolveKnownCallName(string(lit.Value), funcs, module, imports, call.At)
	if err != nil {
		return err
	}
	targetSig, ok := funcs[target]
	if !ok {
		return nil
	}
	return validateBudgetContextEdge(
		call.At,
		callerName,
		callerSig,
		contextName+" '"+target+"'",
		targetSig,
	)
}

func validateBudgetContextEdge(
	pos frontend.Position,
	callerName string,
	callerSig FuncSig,
	context string,
	targetSig FuncSig,
) error {
	if !targetSig.HasBudget {
		return nil
	}
	required := targetSig.Budget
	if !callerSig.HasBudget {
		return budgetDiagnosticf(
			pos,
			"budget context for %s requires caller '%s' to declare budget at least %d",
			context,
			callerName,
			required,
		)
	}
	if callerSig.Budget < required {
		return budgetDiagnosticf(
			pos,
			"budget context for %s requires caller budget at least %d, got %d",
			context,
			required,
			callerSig.Budget,
		)
	}
	return nil
}

type functionClausePolicy struct {
	hasNoAlloc   bool
	hasNoBlock   bool
	hasRealtime  bool
	hasBudget    bool
	budget       int32
	hasPrivacy   bool
	consentParam string
}

func parseFunctionClausePolicy(fn *frontend.FuncDecl) (functionClausePolicy, error) {
	policy, err := semanticspolicy.ParseFunctionClausePolicy(fn, constI32, privacyDiagnosticf)
	if err != nil {
		return functionClausePolicy{}, err
	}
	return functionClausePolicy{
		hasNoAlloc:   policy.HasNoAlloc,
		hasNoBlock:   policy.HasNoBlock,
		hasRealtime:  policy.HasRealtime,
		hasBudget:    policy.HasBudget,
		budget:       policy.Budget,
		hasPrivacy:   policy.HasPrivacy,
		consentParam: policy.ConsentParam,
	}, nil
}

func validateFunctionPolicyClauses(
	fn *frontend.FuncDecl,
	effects []string,
	paramTypes map[string]string,
	returnType string,
	throwsType string,
	types map[string]*TypeInfo,
) error {
	policy, err := parseFunctionClausePolicy(fn)
	if err != nil {
		return err
	}
	declaredEffects := effectSet(effects)
	hasEffect := func(name string) bool {
		_, ok := declaredEffects[name]
		return ok
	}

	if hasEffect("budget") && !policy.hasBudget {
		return budgetDiagnosticf(fn.Pos, "uses effect 'budget' requires semantic clause 'budget'")
	}
	if policy.hasBudget && !hasEffect("budget") {
		return budgetDiagnosticf(
			fn.Pos,
			"semantic clause 'budget' requires function '%s' to declare uses effect 'budget'",
			fn.Name,
		)
	}
	if policy.hasNoAlloc && hasEffect("alloc") {
		return effectDiagnosticf(
			fn.Pos,
			"semantic clause 'noalloc' conflicts with declared effect 'alloc'",
		)
	}
	if policy.hasNoBlock {
		if blocked := firstForbiddenEffect(
			declaredEffects,
			[]string{"actors", "control", "io", "link", "mmio", "runtime"},
		); blocked != "" {
			return effectDiagnosticf(
				fn.Pos,
				"semantic clause 'noblock' conflicts with declared effect '%s'",
				blocked,
			)
		}
	}
	if policy.hasRealtime {
		if !policy.hasNoAlloc {
			return effectDiagnosticf(
				fn.Pos,
				"semantic clause 'realtime' requires semantic clause 'noalloc'",
			)
		}
		if !policy.hasNoBlock {
			return effectDiagnosticf(
				fn.Pos,
				"semantic clause 'realtime' requires semantic clause 'noblock'",
			)
		}
		if blocked := firstForbiddenEffect(
			declaredEffects,
			[]string{"actors", "alloc", "control", "io", "link", "mmio", "runtime"},
		); blocked != "" {
			return effectDiagnosticf(
				fn.Pos,
				"semantic clause 'realtime' conflicts with declared effect '%s'",
				blocked,
			)
		}
	}
	if policy.hasPrivacy && !hasEffect("privacy") {
		return privacyDiagnosticf(
			fn.Pos,
			"semantic clause 'privacy' requires function '%s' to declare uses effect 'privacy'",
			fn.Name,
		)
	}
	if hasEffect("privacy") && !policy.hasPrivacy {
		return privacyDiagnosticf(
			fn.Pos,
			"uses effect 'privacy' requires semantic clause 'privacy'",
		)
	}

	signatureHasSecret := typeUsesSecret(returnType, types) || typeUsesSecret(throwsType, types)
	for _, paramType := range paramTypes {
		if typeUsesSecret(paramType, types) {
			signatureHasSecret = true
		}
	}
	if functionDeclSignatureUsesSecret(fn, types) {
		signatureHasSecret = true
	}
	if signatureHasSecret && !policy.hasPrivacy {
		return privacyDiagnosticf(
			fn.Pos,
			"secret types in function signature require semantic clause 'privacy'",
		)
	}
	if signatureHasSecret && policy.consentParam == "" {
		return privacyDiagnosticf(
			fn.Pos,
			"secret types in function signature require semantic clause consent(<token>)",
		)
	}
	if policy.consentParam != "" {
		if !policy.hasPrivacy {
			return privacyDiagnosticf(
				fn.Pos,
				"semantic clause 'consent' requires semantic clause 'privacy'",
			)
		}
		paramType, ok := paramTypes[policy.consentParam]
		if !ok {
			return privacyDiagnosticf(
				fn.Pos,
				"semantic clause 'consent' references unknown parameter '%s'",
				policy.consentParam,
			)
		}
		if paramType != "consent.token" {
			return privacyDiagnosticf(
				fn.Pos,
				"semantic clause 'consent' parameter '%s' must have type consent.token",
				policy.consentParam,
			)
		}
	}
	return nil
}

func validateExportedOpaqueABISignature(
	module string,
	fn *frontend.FuncDecl,
	paramTypes map[string]string,
	returnType string,
	types map[string]*TypeInfo,
) error {
	if fn == nil || fn.ExportName == "" {
		return nil
	}
	allowRuntimeHandles := isInternalRuntimeABIExport(module, fn)
	for _, param := range fn.Params {
		paramType := paramTypes[param.Name]
		if param.Ownership != "" {
			return effectDiagnosticf(
				param.At,
				("exported function '%s' cannot expose ownership marker '%s' " +
					"on parameter '%s'; export a plain FFI-safe wrapper"),
				fn.Name,
				param.Ownership,
				param.Name,
			)
		}
		if isOpaqueCapabilityTokenType(paramType) {
			return effectDiagnosticf(
				param.At,
				"exported function '%s' cannot expose opaque capability token '%s' in parameter '%s'",
				fn.Name,
				paramType,
				param.Name,
			)
		}
		if isOpaqueIslandHandleType(paramType) {
			return effectDiagnosticf(
				param.At,
				"exported function '%s' cannot expose opaque island handle '%s' in parameter '%s'",
				fn.Name,
				paramType,
				param.Name,
			)
		}
		if isFunctionTypedABIValueType(paramType) {
			return effectDiagnosticf(
				param.At,
				"exported function '%s' cannot expose function-typed value '%s' in parameter '%s'",
				fn.Name,
				paramType,
				param.Name,
			)
		}
		if !allowRuntimeHandles {
			if exposure, ok := exportedBoolABIExposureForType(paramType, types); ok {
				return effectDiagnosticf(
					param.At,
					"exported function '%s' cannot expose %s '%s' in parameter '%s'",
					fn.Name,
					exposure.Kind,
					exposure.TypeName,
					param.Name,
				)
			}
		}
		if exposure, ok := exportedRawViewABIExposureForType(paramType, types); ok {
			return effectDiagnosticf(
				param.At,
				"exported function '%s' cannot expose %s '%s' in parameter '%s'",
				fn.Name,
				exposure.Kind,
				exposure.TypeName,
				param.Name,
			)
		}
		if !allowRuntimeHandles && isOpaqueRuntimeHandleType(paramType) {
			return effectDiagnosticf(
				param.At,
				"exported function '%s' cannot expose opaque runtime handle '%s' in parameter '%s'",
				fn.Name,
				paramType,
				param.Name,
			)
		}
		if exposure, ok := exportedOpaqueABIExposureForType(paramType, types, allowRuntimeHandles); ok {
			return effectDiagnosticf(
				param.At,
				"exported function '%s' cannot expose %s '%s' through parameter '%s' type '%s'",
				fn.Name,
				exposure.Kind,
				exposure.TypeName,
				param.Name,
				paramType,
			)
		}
	}
	if isOpaqueCapabilityTokenType(returnType) {
		return effectDiagnosticf(
			fn.ReturnType.At,
			"exported function '%s' cannot expose opaque capability token '%s' in return type",
			fn.Name,
			returnType,
		)
	}
	if isOpaqueIslandHandleType(returnType) {
		return effectDiagnosticf(
			fn.ReturnType.At,
			"exported function '%s' cannot expose opaque island handle '%s' in return type",
			fn.Name,
			returnType,
		)
	}
	if isFunctionTypedABIValueType(returnType) {
		return effectDiagnosticf(
			fn.ReturnType.At,
			"exported function '%s' cannot expose function-typed value '%s' in return type",
			fn.Name,
			returnType,
		)
	}
	if !allowRuntimeHandles {
		if exposure, ok := exportedBoolABIExposureForType(returnType, types); ok {
			return effectDiagnosticf(
				fn.ReturnType.At,
				"exported function '%s' cannot expose %s '%s' in return type",
				fn.Name,
				exposure.Kind,
				exposure.TypeName,
			)
		}
	}
	if exposure, ok := exportedRawViewABIExposureForType(returnType, types); ok {
		return effectDiagnosticf(
			fn.ReturnType.At,
			"exported function '%s' cannot expose %s '%s' in return type",
			fn.Name,
			exposure.Kind,
			exposure.TypeName,
		)
	}
	if !allowRuntimeHandles && isOpaqueRuntimeHandleType(returnType) {
		return effectDiagnosticf(
			fn.ReturnType.At,
			"exported function '%s' cannot expose opaque runtime handle '%s' in return type",
			fn.Name,
			returnType,
		)
	}
	if exposure, ok := exportedOpaqueABIExposureForType(returnType, types, allowRuntimeHandles); ok {
		return effectDiagnosticf(
			fn.ReturnType.At,
			"exported function '%s' cannot expose %s '%s' through return type '%s'",
			fn.Name,
			exposure.Kind,
			exposure.TypeName,
			returnType,
		)
	}
	for _, param := range fn.Params {
		paramType := paramTypes[param.Name]
		if exposure, ok := exportedDefaultStructABIExposureForType(paramType, types); ok {
			return effectDiagnosticf(
				param.At,
				("exported function '%s' parameter '%s' type '%s' requires " +
					"explicit repr(C); default Tetra layout is compiler-owned " +
					"and has no public ABI"),
				fn.Name,
				param.Name,
				exposure.TypeName,
			)
		}
	}
	if exposure, ok := exportedDefaultStructABIExposureForType(returnType, types); ok {
		return effectDiagnosticf(
			fn.ReturnType.At,
			("exported function '%s' return type '%s' requires explicit " +
				"repr(C); default Tetra layout is compiler-owned and has no " +
				"public ABI"),
			fn.Name,
			exposure.TypeName,
		)
	}
	return nil
}

// ---- checker_resource_sources.go ----

func resourceSourceForCallProvenance(
	args []frontend.Expr,
	sig FuncSig,
	provenance ResourceProvenance,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	pos frontend.Position,
) (resourceSourceResult, error) {
	if provenance.ParamIndex < 0 || provenance.ParamIndex >= len(args) ||
		provenance.ParamIndex >= len(sig.ParamTypes) {
		return resourceSourceResult{}, fmt.Errorf(
			"%s: invalid resource signature",
			frontend.FormatPos(pos),
		)
	}
	if provenance.ParamPath == "" {
		return resourceSourceForExpr(args[provenance.ParamIndex], funcs, module, imports, state)
	}
	return resourceSourceForExprLeaf(
		args[provenance.ParamIndex],
		sig.ParamTypes[provenance.ParamIndex],
		provenance.ParamPath,
		funcs,
		types,
		module,
		imports,
		state,
	)
}

func resourceSourceForExpr(
	expr frontend.Expr,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
	state *regionState,
) (resourceSourceResult, error) {
	if expr == nil || state == nil {
		return resourceSourceResult{}, nil
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return resourceSourceForPath(e.Name, state), nil
	case *frontend.FieldAccessExpr:
		path, ok := resourcePathForExpr(e)
		if !ok {
			return resourceSourceResult{}, nil
		}
		return resourceSourceForPath(path, state), nil
	case *frontend.TryExpr:
		return resourceSourceForExpr(e.X, funcs, module, imports, state)
	case *frontend.AwaitExpr:
		return resourceSourceForExpr(e.X, funcs, module, imports, state)
	case *frontend.CallExpr:
		resolved, err := resolveCheckedCallName(e.Name, funcs, module, imports, e.At)
		if err != nil {
			return resourceSourceResult{}, err
		}
		sig, ok := funcs[resolved]
		if !ok || sig.ReturnResourceParam < 0 {
			if ok && sig.ReturnResourceParam == regionUnknown {
				return resourceSourceResult{unknown: true}, nil
			}
			return resourceSourceResult{}, nil
		}
		if sig.ReturnResourceParam >= len(e.Args) {
			return resourceSourceResult{}, fmt.Errorf(
				"%s: invalid resource signature for '%s'",
				frontend.FormatPos(e.At),
				resolved,
			)
		}
		if sig.ReturnResourcePath != "" {
			argPath, ok := resourcePathForExpr(e.Args[sig.ReturnResourceParam])
			if !ok {
				return resourceSourceResult{unknown: true}, nil
			}
			source := resourceSourceForPath(joinResourcePath(argPath, sig.ReturnResourcePath), state)
			if !source.known && !source.unknown {
				return resourceSourceResult{unknown: true}, nil
			}
			return source, nil
		}
		return resourceSourceForExpr(e.Args[sig.ReturnResourceParam], funcs, module, imports, state)
	case *frontend.MatchExpr:
		if e.ResultLocal != "" {
			source := resourceSourceForPath(e.ResultLocal, state)
			if source.known || source.unknown {
				return source, nil
			}
		}
		var merged resourceSourceResult
		set := false
		for _, c := range e.Cases {
			source, err := resourceSourceForExpr(c.Value, funcs, module, imports, state)
			if err != nil {
				return resourceSourceResult{}, err
			}
			if !set {
				merged = source
				set = true
				continue
			}
			merged = mergeResourceSourceResults(merged, source)
		}
		return merged, nil
	case *frontend.CatchExpr:
		merged, err := resourceSourceForExpr(e.Call, funcs, module, imports, state)
		if err != nil {
			return resourceSourceResult{}, err
		}
		set := true
		caseScopes := state.catchExprScopes[e]
		for i, c := range e.Cases {
			caseScopeID := regionNone
			if i < len(caseScopes) {
				caseScopeID = caseScopes[i]
			}
			if caseScopeID == regionNone {
				caseScopeID = patternBindingScopeID(c.Pattern, state)
			}
			var source resourceSourceResult
			err := withActiveScope(state, caseScopeID, func() error {
				var err error
				source, err = resourceSourceForExpr(c.Value, funcs, module, imports, state)
				return err
			})
			if err != nil {
				return resourceSourceResult{}, err
			}
			if !set {
				merged = source
				set = true
				continue
			}
			merged = mergeResourceSourceResults(merged, source)
		}
		return merged, nil
	default:
		return resourceSourceResult{}, nil
	}
}

func mergeResourceSourceResults(a, b resourceSourceResult) resourceSourceResult {
	if a.ambiguous || b.ambiguous {
		return resourceSourceResult{ambiguous: true}
	}
	if a.unknown || b.unknown {
		return resourceSourceResult{unknown: true}
	}
	if !a.known && !b.known {
		return resourceSourceResult{}
	}
	if a.known && b.known && a.name == b.name {
		return a
	}
	return resourceSourceResult{ambiguous: true}
}

// ---- checker_resource_tracking.go ----

type flowSnapshot struct {
	reachable              bool
	consumedVars           map[string]frontend.Position
	maybeConsumedVars      map[string]ownershipJoinConflict
	ownershipAliases       map[string]string
	borrowedPtrAliases     map[string]string
	ownedRegionSliceOwners map[string]string
	awaitInvalidatedBorrow map[int]frontend.Position
	consumedResources      map[int]frontend.Position
	resourceVars           map[string]int
	unknownResources       map[int]bool
	finalizedResources     map[int]resourceFinalization
}

type loopFlowExit struct {
	label string
	vars  map[string]int
	flow  flowSnapshot
	taint map[string]bool
}

type loopFlowFrame struct {
	breaks    []loopFlowExit
	continues []loopFlowExit
}

func snapshotFlow(state *regionState) flowSnapshot {
	return flowSnapshot{
		reachable:              state.reachable,
		consumedVars:           copyConsumedVars(state.consumedVars),
		maybeConsumedVars:      copyOwnershipJoinConflicts(state.maybeConsumedVars),
		ownershipAliases:       copyStringMap(state.ownershipAliases),
		borrowedPtrAliases:     copyStringMap(state.borrowedPtrAliases),
		ownedRegionSliceOwners: copyStringMap(state.ownedRegionSliceOwners),
		awaitInvalidatedBorrow: copyPositionByIntMap(state.awaitInvalidatedBorrow),
		consumedResources:      copyConsumedResources(state.consumedResources),
		resourceVars:           copyResourceVars(state.resourceVars),
		unknownResources:       copyUnknownResources(state.unknownResources),
		finalizedResources:     copyFinalizedResources(state.finalizedResources),
	}
}

func restoreFlow(state *regionState, snap flowSnapshot) {
	state.reachable = snap.reachable
	state.consumedVars = copyConsumedVars(snap.consumedVars)
	state.maybeConsumedVars = copyOwnershipJoinConflicts(snap.maybeConsumedVars)
	state.ownershipAliases = copyStringMap(snap.ownershipAliases)
	state.borrowedPtrAliases = copyStringMap(snap.borrowedPtrAliases)
	state.ownedRegionSliceOwners = copyStringMap(snap.ownedRegionSliceOwners)
	state.awaitInvalidatedBorrow = copyPositionByIntMap(snap.awaitInvalidatedBorrow)
	state.consumedResources = copyConsumedResources(snap.consumedResources)
	state.resourceVars = copyResourceVars(snap.resourceVars)
	state.unknownResources = copyUnknownResources(snap.unknownResources)
	state.finalizedResources = copyFinalizedResources(snap.finalizedResources)
}

func mergeFlow(state *regionState, a, b flowSnapshot) {
	mergeFlowWithLabels(state, a, b, "left", "right")
}

func mergeFlowWithLabels(state *regionState, a, b flowSnapshot, leftLabel, rightLabel string) {
	if !a.reachable && !b.reachable {
		restoreFlow(state, a)
		state.reachable = false
		return
	}
	if !a.reachable {
		restoreFlow(state, b)
		return
	}
	if !b.reachable {
		restoreFlow(state, a)
		return
	}
	consumedResources := mergeConsumedResources(a.consumedResources, b.consumedResources)
	finalizedResources := mergeFinalizedResources(
		a.finalizedResources,
		b.finalizedResources,
		leftLabel,
		rightLabel,
	)
	unknownResources := mergeUnknownResources(a.unknownResources, b.unknownResources)
	state.reachable = true
	state.consumedVars = mergeConsumedVars(a.consumedVars, b.consumedVars)
	state.maybeConsumedVars = mergeMaybeConsumedVars(a, b, leftLabel, rightLabel)
	state.ownershipAliases = mergeOwnershipAliases(a.ownershipAliases, b.ownershipAliases)
	state.borrowedPtrAliases = mergeBorrowedPtrAliases(a.borrowedPtrAliases, b.borrowedPtrAliases)
	state.ownedRegionSliceOwners = mergeOwnershipAliases(
		a.ownedRegionSliceOwners,
		b.ownedRegionSliceOwners,
	)
	state.awaitInvalidatedBorrow = mergeAwaitInvalidatedBorrowRegions(
		a.awaitInvalidatedBorrow,
		b.awaitInvalidatedBorrow,
	)
	state.consumedResources = consumedResources
	state.unknownResources = unknownResources
	state.finalizedResources = finalizedResources
	state.resourceVars = mergeResourceVars(
		state,
		a.resourceVars,
		b.resourceVars,
		consumedResources,
		finalizedResources,
		unknownResources,
		leftLabel,
		rightLabel,
	)
}

func pushLoopFlowFrame(state *regionState) {
	if state == nil {
		return
	}
	state.loopFlowFrames = append(state.loopFlowFrames, loopFlowFrame{})
}

func popLoopFlowFrame(state *regionState) loopFlowFrame {
	if state == nil || len(state.loopFlowFrames) == 0 {
		return loopFlowFrame{}
	}
	frame := state.loopFlowFrames[len(state.loopFlowFrames)-1]
	state.loopFlowFrames = state.loopFlowFrames[:len(state.loopFlowFrames)-1]
	return frame
}

func recordLoopFlowExit(state *regionState, label string, analysis *functionAnalysisState) {
	if state == nil || len(state.loopFlowFrames) == 0 {
		return
	}
	exit := loopFlowExit{
		label: label,
		vars:  copyRegionVars(state.regionVars),
		flow:  snapshotFlow(state),
	}
	if analysis != nil {
		exit.taint = analysis.copySecretTaint()
	}
	frame := &state.loopFlowFrames[len(state.loopFlowFrames)-1]
	if label == "break" {
		frame.breaks = append(frame.breaks, exit)
		return
	}
	frame.continues = append(frame.continues, exit)
}

func mergeLoopFlowExits(state *regionState, analysis *functionAnalysisState, exits []loopFlowExit) {
	if len(exits) == 0 {
		return
	}
	mergedVars := copyRegionVars(exits[0].vars)
	mergedFlow := exits[0].flow
	mergedTaint := cloneBoolMap(exits[0].taint)
	labels := []string{exits[0].label}
	for _, exit := range exits[1:] {
		leftLabel := strings.Join(labels, "/")
		mergeControlFlowWithLabels(
			state,
			analysis,
			mergedVars,
			mergedFlow,
			mergedTaint,
			exit.vars,
			exit.flow,
			exit.taint,
			leftLabel,
			exit.label,
		)
		mergedVars = copyRegionVars(state.regionVars)
		mergedFlow = snapshotFlow(state)
		if analysis != nil {
			mergedTaint = analysis.copySecretTaint()
		}
		labels = append(labels, exit.label)
	}
	state.regionVars = mergedVars
	restoreFlow(state, mergedFlow)
	if analysis != nil {
		analysis.restoreSecretTaint(mergedTaint)
	}
}

func mergeControlFlowWithLabels(
	state *regionState,
	analysis *functionAnalysisState,
	leftVars map[string]int,
	leftFlow flowSnapshot,
	leftTaint map[string]bool,
	rightVars map[string]int,
	rightFlow flowSnapshot,
	rightTaint map[string]bool,
	leftLabel string,
	rightLabel string,
) {
	switch {
	case !leftFlow.reachable && !rightFlow.reachable:
		state.regionVars = copyRegionVars(leftVars)
		restoreFlow(state, leftFlow)
		state.reachable = false
		analysis.restoreSecretTaint(mergeSecretTaintMaps(leftTaint, rightTaint))
	case !leftFlow.reachable:
		state.regionVars = copyRegionVars(rightVars)
		restoreFlow(state, rightFlow)
		analysis.restoreSecretTaint(rightTaint)
	case !rightFlow.reachable:
		state.regionVars = copyRegionVars(leftVars)
		restoreFlow(state, leftFlow)
		analysis.restoreSecretTaint(leftTaint)
	default:
		state.regionVars = mergeRegionVars(leftVars, rightVars)
		mergeFlowWithLabels(state, leftFlow, rightFlow, leftLabel, rightLabel)
		analysis.restoreSecretTaint(mergeSecretTaintMaps(leftTaint, rightTaint))
		recordMergeConflicts(state, leftVars, rightVars, leftLabel, rightLabel)
	}
}

type resourceSourceResult struct {
	name      string
	known     bool
	ambiguous bool
	unknown   bool
}

func bindResourceFromExpr(
	name string,
	typeName string,
	expr frontend.Expr,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if !isResourceHandleType(typeName) {
		state.bindResource(name, "", false)
		return nil
	}
	if typeName == surfaceSurfaceTypeName {
		if owner, ok := surfaceConstructedHandleOwnerPathExpr(expr); ok {
			if _, exists := state.resourceID(owner); exists {
				state.bindResource(name, owner, true)
				return nil
			}
		}
	}
	source, err := resourceSourceForExpr(expr, funcs, module, imports, state)
	if err != nil {
		return err
	}
	if source.ambiguous {
		return ownershipDiagnosticf(expr.Pos(), "resource expression mixes resource provenance")
	}
	if source.unknown {
		state.bindUnknownResource(name)
		return nil
	}
	sourceName := ""
	if source.known {
		sourceName = source.name
		if _, consumed := state.consumedAt(sourceName); consumed {
			state.bindTransferredResource(name, sourceName)
			return nil
		}
	}
	state.bindResource(name, sourceName, true)
	return nil
}

func bindResourceTreeFromExpr(
	name string,
	typeName string,
	expr frontend.Expr,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if state == nil || name == "" {
		return nil
	}
	state.clearResourceTree(name)
	if !typeContainsResourceHandle(typeName, types) {
		state.bindResource(name, "", false)
		return nil
	}
	if isResourceHandleType(typeName) {
		return bindResourceFromExpr(name, typeName, expr, funcs, module, imports, state)
	}
	if info, ok := types[typeName]; ok && info.Kind == TypeOptional {
		if sourcePrefix, ok := resourcePathForExpr(expr); ok &&
			resourceTreeHasPath(sourcePrefix, info.ElemType, types, state) {
			copyResourceTreeFromPath(
				resourceElementPath(name),
				sourcePrefix,
				info.ElemType,
				types,
				state,
			)
			return nil
		}
	}
	if sourcePrefix, ok := resourcePathForExpr(expr); ok {
		copyResourceTreeFromPath(name, sourcePrefix, typeName, types, state)
		return nil
	}
	switch e := expr.(type) {
	case *frontend.TryExpr:
		return bindResourceTreeFromExpr(name, typeName, e.X, funcs, types, module, imports, state)
	case *frontend.AwaitExpr:
		return bindResourceTreeFromExpr(name, typeName, e.X, funcs, types, module, imports, state)
	case *frontend.MatchExpr:
		if e.ResultLocal != "" {
			copyResourceTreeFromPath(name, e.ResultLocal, typeName, types, state)
			return nil
		}
	case *frontend.StructLitExpr:
		info, ok := types[typeName]
		if !ok || info.Kind != TypeStruct {
			markResourceTreeUnknown(name, typeName, types, state)
			return nil
		}
		byName := make(map[string]frontend.Expr, len(e.Fields))
		for _, field := range e.Fields {
			byName[field.Name] = field.Value
		}
		for _, field := range info.Fields {
			value := byName[field.Name]
			if value == nil {
				continue
			}
			if err := bindResourceTreeFromExpr(
				resourceFieldPath(name, field.Name),
				field.TypeName,
				value,
				funcs,
				types,
				module,
				imports,
				state,
			); err != nil {
				return err
			}
		}
		return nil
	case *frontend.CallExpr:
		if info, ok := types[typeName]; ok && info.Kind == TypeStruct && e.Name == typeName {
			for i, field := range info.Fields {
				if i >= len(e.Args) {
					break
				}
				if err := bindResourceTreeFromExpr(
					resourceFieldPath(name, field.Name),
					field.TypeName,
					e.Args[i],
					funcs,
					types,
					module,
					imports,
					state,
				); err != nil {
					return err
				}
			}
			return nil
		}
		resolved, err := resolveCheckedCallName(e.Name, funcs, module, imports, e.At)
		if err != nil {
			return err
		}
		if resolved == "core.recv_typed" || isFreshSystemReceiveResourceReturn(resolved) {
			bindFreshResourceTree(name, typeName, types, state)
			return nil
		}
		enumType, caseInfo, ok, err := resolveEnumCaseConstructorCall(e, types, module, imports)
		if err != nil {
			return err
		}
		if ok && enumType == typeName {
			for i, arg := range e.Args {
				if i >= len(caseInfo.PayloadTypes) {
					break
				}
				if err := bindResourceTreeFromExpr(
					resourceEnumPayloadPath(name, caseInfo.Ordinal, i),
					caseInfo.PayloadTypes[i],
					arg,
					funcs,
					types,
					module,
					imports,
					state,
				); err != nil {
					return err
				}
			}
			return nil
		}
		sig, ok := funcs[resolved]
		if !ok {
			return nil
		}
		if handled, err := bindResourceTreeFromCallSummary(
			name,
			typeName,
			e,
			sig,
			funcs,
			types,
			module,
			imports,
			state,
		); handled || err != nil {
			return err
		}
	}
	markResourceTreeUnknown(name, typeName, types, state)
	return nil
}

func bindOwnedRegionSliceOwnerFromExpr(
	name string,
	typeName string,
	expr frontend.Expr,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if state == nil || name == "" {
		return nil
	}
	info, ok := types[typeName]
	if !ok {
		state.clearOwnedRegionSliceOwnerTree(name)
		return nil
	}
	switch info.Kind {
	case TypeSlice:
		owner := ownedRegionSliceOwnerForExpr(expr, state)
		if owner == "" {
			state.clearOwnedRegionSliceOwnerTree(name)
			return nil
		}
		state.bindOwnedRegionSliceOwner(name, owner)
		return nil
	case TypeStruct:
		state.clearOwnedRegionSliceOwnerTree(name)
		if sourcePrefix, ok := resourcePathForExpr(expr); ok {
			copyOwnedRegionSliceOwnerTreeFromPath(name, sourcePrefix, typeName, types, state)
			return nil
		}
		if lit, ok := expr.(*frontend.StructLitExpr); ok {
			byName := make(map[string]frontend.Expr, len(lit.Fields))
			for _, field := range lit.Fields {
				byName[field.Name] = field.Value
			}
			for _, field := range info.Fields {
				value := byName[field.Name]
				if value == nil {
					continue
				}
				if err := bindOwnedRegionSliceOwnerFromExpr(
					resourceFieldPath(name, field.Name),
					field.TypeName,
					value,
					types,
					module,
					imports,
					state,
				); err != nil {
					return err
				}
			}
			return nil
		}
		if call, ok := expr.(*frontend.CallExpr); ok && call.Name == typeName {
			for i, field := range info.Fields {
				if i >= len(call.Args) {
					break
				}
				if err := bindOwnedRegionSliceOwnerFromExpr(
					resourceFieldPath(name, field.Name),
					field.TypeName,
					call.Args[i],
					types,
					module,
					imports,
					state,
				); err != nil {
					return err
				}
			}
		}
	case TypeEnum:
		state.clearOwnedRegionSliceOwnerTree(name)
		if sourcePrefix, ok := resourcePathForExpr(expr); ok {
			copyOwnedRegionSliceOwnerTreeFromPath(name, sourcePrefix, typeName, types, state)
			return nil
		}
		call, ok := expr.(*frontend.CallExpr)
		if !ok {
			return nil
		}
		enumType, caseInfo, found, err := resolveEnumCaseConstructorCall(
			call,
			types,
			module,
			imports,
		)
		if err != nil {
			return err
		}
		if !found || enumType != typeName {
			return nil
		}
		for i, arg := range call.Args {
			if i >= len(caseInfo.PayloadTypes) {
				break
			}
			if err := bindOwnedRegionSliceOwnerFromExpr(
				resourceEnumPayloadPath(name, caseInfo.Ordinal, i),
				caseInfo.PayloadTypes[i],
				arg,
				types,
				module,
				imports,
				state,
			); err != nil {
				return err
			}
		}
	case TypeOptional:
		state.clearOwnedRegionSliceOwnerTree(name)
		call, ok := expr.(*frontend.CallExpr)
		if !ok || len(call.Args) == 0 {
			return nil
		}
		return bindOwnedRegionSliceOwnerFromExpr(
			resourceElementPath(name),
			info.ElemType,
			call.Args[0],
			types,
			module,
			imports,
			state,
		)
	default:
		state.clearOwnedRegionSliceOwnerTree(name)
	}
	return nil
}

func copyOwnedRegionSliceOwnerTreeFromPath(
	dst string,
	src string,
	typeName string,
	types map[string]*TypeInfo,
	state *regionState,
) {
	if state == nil || dst == "" || src == "" {
		return
	}
	for _, leaf := range regionLeafPaths(typeName, types, "") {
		srcLeaf := joinResourcePath(src, leaf)
		if owner, ok := state.ownedRegionSliceOwner(srcLeaf); ok {
			state.bindOwnedRegionSliceOwner(joinResourcePath(dst, leaf), owner)
		}
	}
}

func ownedRegionSliceOwnerForExpr(expr frontend.Expr, state *regionState) string {
	if state == nil || expr == nil || isExplicitCopyExpr(expr) {
		return ""
	}
	if path, ok := resourcePathForExpr(expr); ok {
		if owner, found := state.ownedRegionSliceOwner(path); found {
			return owner
		}
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || len(call.Args) == 0 {
		return ""
	}
	name := call.Name
	if target, ok := ResolveBuiltinAlias(name); ok {
		name = target
	}
	switch name {
	case "core.island_make_u8",
		"core.island_make_u16",
		"core.island_make_i32",
		"core.island_make_bool":
		if owner, ok := resourcePathForExpr(call.Args[0]); ok {
			return owner
		}
	}
	return ""
}

func bindResourceTreeFromCallSummary(
	name string,
	typeName string,
	call *frontend.CallExpr,
	sig FuncSig,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) (bool, error) {
	if len(sig.ReturnResourceSummary) == 0 {
		return false, nil
	}
	for _, leaf := range resourceLeafPaths(typeName, types, "") {
		dstLeaf := joinResourcePath(name, leaf)
		provenances := sig.ReturnResourceSummary[leaf]
		if len(provenances) == 0 {
			state.bindResource(dstLeaf, "", true)
			continue
		}
		if len(provenances) > 1 {
			state.bindUnknownResource(dstLeaf)
			continue
		}
		source, err := resourceSourceForCallProvenance(
			call.Args,
			sig,
			provenances[0],
			funcs,
			types,
			module,
			imports,
			state,
			call.At,
		)
		if err != nil {
			return true, err
		}
		if source.ambiguous || source.unknown || !source.known {
			state.bindUnknownResource(dstLeaf)
			continue
		}
		if _, consumed := state.consumedAt(source.name); consumed {
			state.bindTransferredResource(dstLeaf, source.name)
			continue
		}
		state.bindResource(dstLeaf, source.name, true)
	}
	return true, nil
}

func isFreshSystemReceiveResourceReturn(resolved string) bool {
	switch resolved {
	case "core.actor_recv_system",
		"core.actor_recv_system_poll",
		"core.actor_recv_system_until",
		"lib.core.actors.recv_system",
		"lib.core.actors.poll_system",
		"lib.core.actors.recv_system_until":
		return true
	default:
		return false
	}
}

func bindResourceTreeFromPathOrUnknown(
	dst string,
	src string,
	typeName string,
	types map[string]*TypeInfo,
	state *regionState,
) {
	if !typeContainsResourceHandle(typeName, types) {
		state.bindResource(dst, "", false)
		return
	}
	copyResourceTreeFromPath(dst, src, typeName, types, state)
}

func bindPatternResourceLocals(
	pattern frontend.Expr,
	fallbackName string,
	scrutineePath string,
	scrutType string,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if state == nil || scrutineePath == "" || !typeContainsResourceHandle(scrutType, types) {
		return nil
	}
	info, ok := types[scrutType]
	if !ok {
		return nil
	}
	if pattern == nil {
		if fallbackName == "" || info.Kind != TypeOptional {
			return nil
		}
		bindResourceTreeFromPathOrUnknown(
			fallbackName,
			resourceElementPath(scrutineePath),
			info.ElemType,
			types,
			state,
		)
		return nil
	}
	switch p := pattern.(type) {
	case *frontend.SomePatternExpr:
		if info.Kind != TypeOptional {
			return nil
		}
		bindResourceTreeFromPathOrUnknown(
			p.Name,
			resourceElementPath(scrutineePath),
			info.ElemType,
			types,
			state,
		)
	case *frontend.EnumCasePatternExpr:
		caseType, caseInfo, found, err := resolveEnumCasePattern(p, types, module, imports)
		if err != nil {
			return err
		}
		if !found || caseType != scrutType {
			return nil
		}
		for i, binding := range p.Bindings {
			if i >= len(caseInfo.PayloadTypes) {
				break
			}
			bindResourceTreeFromPathOrUnknown(
				binding,
				resourceEnumPayloadPath(scrutineePath, caseInfo.Ordinal, i),
				caseInfo.PayloadTypes[i],
				types,
				state,
			)
		}
	}
	return nil
}

func bindPatternOwnershipAliases(
	pattern frontend.Expr,
	fallbackName string,
	scrutineePath string,
	scrutType string,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if state == nil || scrutineePath == "" {
		return nil
	}
	info, ok := types[scrutType]
	if !ok {
		return nil
	}
	if pattern == nil {
		if fallbackName == "" || info.Kind != TypeOptional {
			return nil
		}
		state.bindOwnershipAlias(fallbackName, resourceElementPath(scrutineePath))
		return nil
	}
	switch p := pattern.(type) {
	case *frontend.SomePatternExpr:
		if info.Kind == TypeOptional {
			state.bindOwnershipAlias(p.Name, resourceElementPath(scrutineePath))
		}
	case *frontend.EnumCasePatternExpr:
		caseType, caseInfo, found, err := resolveEnumCasePattern(p, types, module, imports)
		if err != nil {
			return err
		}
		if !found || caseType != scrutType {
			return nil
		}
		for i, binding := range p.Bindings {
			if i >= len(caseInfo.PayloadTypes) {
				break
			}
			state.bindOwnershipAlias(binding, resourceEnumPayloadPath(scrutineePath, caseInfo.Ordinal, i))
		}
	}
	return nil
}

func bindPatternBorrowedPtrAliases(
	pattern frontend.Expr,
	fallbackName string,
	scrutineePath string,
	scrutType string,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if state == nil || scrutineePath == "" {
		return nil
	}
	info, ok := types[scrutType]
	if !ok {
		return nil
	}
	if pattern == nil {
		if fallbackName == "" || info.Kind != TypeOptional {
			return nil
		}
		copyBorrowedPtrAliasesFromPath(
			fallbackName,
			resourceElementPath(scrutineePath),
			info.ElemType,
			types,
			state,
		)
		return nil
	}
	switch p := pattern.(type) {
	case *frontend.SomePatternExpr:
		if info.Kind == TypeOptional {
			copyBorrowedPtrAliasesFromPath(
				p.Name,
				resourceElementPath(scrutineePath),
				info.ElemType,
				types,
				state,
			)
		}
	case *frontend.EnumCasePatternExpr:
		caseType, caseInfo, found, err := resolveEnumCasePattern(p, types, module, imports)
		if err != nil {
			return err
		}
		if !found || caseType != scrutType {
			return nil
		}
		for i, binding := range p.Bindings {
			if i >= len(caseInfo.PayloadTypes) {
				break
			}
			copyBorrowedPtrAliasesFromPath(
				binding,
				resourceEnumPayloadPath(scrutineePath, caseInfo.Ordinal, i),
				caseInfo.PayloadTypes[i],
				types,
				state,
			)
		}
	}
	return nil
}

func copyBorrowedPtrAliasesFromPath(
	dst string,
	src string,
	typeName string,
	types map[string]*TypeInfo,
	state *regionState,
) {
	if state == nil || dst == "" || src == "" || !typeMayContainPtr(typeName, types) {
		return
	}
	state.clearBorrowedPtrAliasTree(dst)
	for _, leaf := range ptrLeafPaths(typeName, types, "") {
		srcLeaf := joinResourcePath(src, leaf)
		if owner, borrowed := state.borrowedPtrAliasOwner(srcLeaf); borrowed {
			state.bindBorrowedPtrAlias(joinResourcePath(dst, leaf), owner)
		}
	}
}

func bindPatternRegionLocals(
	pattern frontend.Expr,
	fallbackName string,
	scrutineePath string,
	scrutType string,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if state == nil || scrutineePath == "" || !typeMayContainRegion(scrutType, types) {
		return nil
	}
	info, ok := types[scrutType]
	if !ok {
		return nil
	}
	if pattern == nil {
		if fallbackName == "" || info.Kind != TypeOptional {
			return nil
		}
		copyRegionTreeFromPath(
			fallbackName,
			resourceElementPath(scrutineePath),
			info.ElemType,
			types,
			state,
		)
		return nil
	}
	switch p := pattern.(type) {
	case *frontend.SomePatternExpr:
		if info.Kind != TypeOptional {
			return nil
		}
		copyRegionTreeFromPath(
			p.Name,
			resourceElementPath(scrutineePath),
			info.ElemType,
			types,
			state,
		)
	case *frontend.EnumCasePatternExpr:
		caseType, caseInfo, found, err := resolveEnumCasePattern(p, types, module, imports)
		if err != nil {
			return err
		}
		if !found || caseType != scrutType {
			return nil
		}
		for i, binding := range p.Bindings {
			if i >= len(caseInfo.PayloadTypes) {
				break
			}
			copyRegionTreeFromPath(
				binding,
				resourceEnumPayloadPath(scrutineePath, caseInfo.Ordinal, i),
				caseInfo.PayloadTypes[i],
				types,
				state,
			)
		}
	}
	return nil
}

func copyResourceTreeFromPath(
	dst string,
	src string,
	typeName string,
	types map[string]*TypeInfo,
	state *regionState,
) {
	for _, leaf := range resourceLeafPaths(typeName, types, "") {
		dstLeaf := joinResourcePath(dst, leaf)
		srcLeaf := joinResourcePath(src, leaf)
		if id, ok := state.resourceID(srcLeaf); ok && !state.resourceUnknown(srcLeaf) {
			if _, consumed := state.consumedAt(srcLeaf); consumed {
				state.bindTransferredResource(dstLeaf, srcLeaf)
				continue
			}
			state.resourceVars[dstLeaf] = id
			continue
		}
		state.bindUnknownResource(dstLeaf)
	}
}

func resourceTreeHasPath(
	prefix string,
	typeName string,
	types map[string]*TypeInfo,
	state *regionState,
) bool {
	if state == nil || prefix == "" {
		return false
	}
	for _, leaf := range resourceLeafPaths(typeName, types, "") {
		if _, ok := state.resourceID(joinResourcePath(prefix, leaf)); ok {
			return true
		}
	}
	return false
}

func markResourceTreeUnknown(
	prefix string,
	typeName string,
	types map[string]*TypeInfo,
	state *regionState,
) {
	for _, leaf := range resourceLeafPaths(typeName, types, "") {
		state.bindUnknownResource(joinResourcePath(prefix, leaf))
	}
}

func bindFreshResourceTree(
	prefix string,
	typeName string,
	types map[string]*TypeInfo,
	state *regionState,
) {
	for _, leaf := range resourceLeafPaths(typeName, types, "") {
		state.bindResource(joinResourcePath(prefix, leaf), "", true)
	}
}

func resourceLeafPaths(typeName string, types map[string]*TypeInfo, prefix string) []string {
	return resourceLeafPathsVisiting(typeName, types, prefix, map[string]bool{})
}

func ptrLeafPaths(typeName string, types map[string]*TypeInfo, prefix string) []string {
	return ptrLeafPathsVisiting(typeName, types, prefix, map[string]bool{})
}

func resourceLeafPathsVisiting(
	typeName string,
	types map[string]*TypeInfo,
	prefix string,
	visiting map[string]bool,
) []string {
	if typeName == surfaceFrameTypeName {
		return nil
	}
	if isResourceHandleType(typeName) {
		return []string{prefix}
	}
	info, ok := types[typeName]
	if !ok {
		return nil
	}
	var out []string
	switch info.Kind {
	case TypeStruct:
		if visiting[typeName] {
			return nil
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, field := range info.Fields {
			out = append(
				out,
				resourceLeafPathsVisiting(
					field.TypeName,
					types,
					resourceFieldPath(prefix, field.Name),
					visiting,
				)...)
		}
	case TypeEnum:
		if visiting[typeName] {
			return nil
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, c := range info.EnumCases {
			for i, payload := range c.PayloadTypes {
				out = append(
					out,
					resourceLeafPathsVisiting(
						payload,
						types,
						resourceEnumPayloadPath(prefix, c.Ordinal, i),
						visiting,
					)...)
			}
		}
	case TypeArray, TypeOptional:
		out = append(
			out,
			resourceLeafPathsVisiting(
				info.ElemType,
				types,
				resourceElementPath(prefix),
				visiting,
			)...)
	}
	return out
}

func ptrLeafPathsVisiting(
	typeName string,
	types map[string]*TypeInfo,
	prefix string,
	visiting map[string]bool,
) []string {
	if typeName == "ptr" {
		return []string{prefix}
	}
	info, ok := types[typeName]
	if !ok {
		return nil
	}
	var out []string
	switch info.Kind {
	case TypeStruct:
		if visiting[typeName] {
			return nil
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, field := range info.Fields {
			out = append(
				out,
				ptrLeafPathsVisiting(
					field.TypeName,
					types,
					resourceFieldPath(prefix, field.Name),
					visiting,
				)...)
		}
	case TypeEnum:
		if visiting[typeName] {
			return nil
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, c := range info.EnumCases {
			for i, payload := range c.PayloadTypes {
				out = append(
					out,
					ptrLeafPathsVisiting(
						payload,
						types,
						resourceEnumPayloadPath(prefix, c.Ordinal, i),
						visiting,
					)...)
			}
		}
	case TypeArray, TypeOptional:
		out = append(
			out,
			ptrLeafPathsVisiting(
				info.ElemType,
				types,
				resourceElementPath(prefix),
				visiting,
			)...)
	}
	return out
}

func resourcePathForExpr(expr frontend.Expr) (string, bool) {
	return semanticsresources.PathForExpr(expr)
}

func resourceFieldPath(prefix string, field string) string {
	return semanticsresources.FieldPath(prefix, field)
}

func resourceEnumPayloadPath(prefix string, ordinal int32, index int) string {
	return semanticsresources.EnumPayloadPath(prefix, ordinal, index)
}

func resourceElementPath(prefix string) string {
	return semanticsresources.Path(prefix).Element().String()
}

func joinResourcePath(prefix string, leaf string) string {
	return semanticsresources.JoinPath(
		semanticsresources.Path(prefix),
		semanticsresources.Path(leaf),
	).String()
}

func resourcePathContains(prefix string, path string) bool {
	prefixPath := semanticsresources.Path(prefix)
	pathPath := semanticsresources.Path(path)
	return prefixPath == pathPath || prefixPath.IsAncestorOf(pathPath)
}

func resourcePathRelativeTo(path string, prefix string) (string, bool) {
	relative, ok := semanticsresources.Path(path).RelativeTo(semanticsresources.Path(prefix))
	return relative.String(), ok
}

func resourceSourceForPath(path string, state *regionState) resourceSourceResult {
	if state == nil || path == "" {
		return resourceSourceResult{}
	}
	if _, ok := state.resourceID(path); !ok {
		return resourceSourceResult{}
	}
	if state.resourceUnknown(path) {
		return resourceSourceResult{unknown: true}
	}
	return resourceSourceResult{name: path, known: true}
}

func returnResourceSummaryForExpr(
	expr frontend.Expr,
	typeName string,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) (ReturnResourceSummary, bool, error) {
	if state == nil || !typeContainsResourceHandle(typeName, types) {
		return nil, false, nil
	}
	summary := ReturnResourceSummary{}
	for _, leaf := range resourceLeafPaths(typeName, types, "") {
		source, err := resourceSourceForExprLeaf(
			expr,
			typeName,
			leaf,
			funcs,
			types,
			module,
			imports,
			state,
		)
		if err != nil {
			return nil, false, err
		}
		if source.ambiguous {
			return nil, false, ownershipDiagnosticf(
				expr.Pos(),
				"resource expression mixes resource provenance",
			)
		}
		if source.unknown {
			return nil, true, nil
		}
		if !source.known {
			continue
		}
		paramIndex, paramPath, ok := state.resourceParamOwner(source.name)
		if !ok {
			continue
		}
		summary[leaf] = appendResourceProvenance(summary[leaf], ResourceProvenance{
			ParamIndex: paramIndex,
			ParamPath:  paramPath,
		})
	}
	if len(summary) == 0 {
		return nil, false, nil
	}
	return summary, false, nil
}

func appendResourceProvenance(
	in []ResourceProvenance,
	provenance ResourceProvenance,
) []ResourceProvenance {
	for _, existing := range in {
		if existing == provenance {
			return in
		}
	}
	return append(in, provenance)
}

func bindCatchErrorResourceSummary(
	errorLocal string,
	call *frontend.CallExpr,
	sig FuncSig,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if state == nil || errorLocal == "" || call == nil || len(sig.ThrowResourceSummary) == 0 {
		return nil
	}
	state.clearResourceTree(errorLocal)
	for leaf, provenances := range sig.ThrowResourceSummary {
		var merged resourceSourceResult
		set := false
		for _, provenance := range provenances {
			source, err := resourceSourceForCallProvenance(
				call.Args,
				sig,
				provenance,
				funcs,
				types,
				module,
				imports,
				state,
				call.At,
			)
			if err != nil {
				return err
			}
			if !set {
				merged = source
				set = true
				continue
			}
			merged = mergeResourceSourceResults(merged, source)
		}
		dst := joinResourcePath(errorLocal, leaf)
		if !set || merged.unknown {
			state.bindUnknownResource(dst)
			continue
		}
		if merged.ambiguous {
			return ownershipDiagnosticf(call.At, "resource expression mixes resource provenance")
		}
		if !merged.known {
			continue
		}
		if _, consumed := state.consumedAt(merged.name); consumed {
			state.bindTransferredResource(dst, merged.name)
			continue
		}
		state.bindResource(dst, merged.name, true)
	}
	return nil
}

func recordTryCallThrowResourceSummary(
	call *frontend.CallExpr,
	sig FuncSig,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if state == nil || call == nil || len(sig.ThrowResourceSummary) == 0 ||
		!typeContainsResourceHandle(sig.ThrowsType, types) {
		return nil
	}
	summary := ReturnResourceSummary{}
	for leaf, provenances := range sig.ThrowResourceSummary {
		for _, provenance := range provenances {
			source, err := resourceSourceForCallProvenance(
				call.Args,
				sig,
				provenance,
				funcs,
				types,
				module,
				imports,
				state,
				call.At,
			)
			if err != nil {
				return err
			}
			if source.ambiguous {
				return ownershipDiagnosticf(
					call.At,
					"resource expression mixes resource provenance",
				)
			}
			if source.unknown || !source.known {
				continue
			}
			paramIndex, paramPath, ok := state.resourceParamOwner(source.name)
			if !ok {
				continue
			}
			summary[leaf] = appendResourceProvenance(summary[leaf], ResourceProvenance{
				ParamIndex: paramIndex,
				ParamPath:  paramPath,
			})
		}
	}
	if len(summary) == 0 {
		return nil
	}
	return state.recordThrowResourceSummary(summary, call.At)
}

func resourceSourceForExprLeaf(
	expr frontend.Expr,
	typeName string,
	leaf string,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) (resourceSourceResult, error) {
	if leaf == "" {
		return resourceSourceForExpr(expr, funcs, module, imports, state)
	}
	if sourcePrefix, ok := resourcePathForExpr(expr); ok {
		source := resourceSourceForPath(joinResourcePath(sourcePrefix, leaf), state)
		if !source.known && !source.unknown {
			if wrapped, handled, err := resourceSourceForOptionalWrappedLeaf(
				expr,
				typeName,
				leaf,
				funcs,
				types,
				module,
				imports,
				state,
			); handled ||
				err != nil {
				return wrapped, err
			}
			return resourceSourceResult{unknown: true}, nil
		}
		return source, nil
	}
	switch e := expr.(type) {
	case *frontend.TryExpr:
		return resourceSourceForExprLeaf(e.X, typeName, leaf, funcs, types, module, imports, state)
	case *frontend.AwaitExpr:
		return resourceSourceForExprLeaf(e.X, typeName, leaf, funcs, types, module, imports, state)
	case *frontend.StructLitExpr:
		return resourceSourceForStructFieldLeaf(
			e.Fields,
			typeName,
			leaf,
			funcs,
			types,
			module,
			imports,
			state,
		)
	case *frontend.CallExpr:
		if info, ok := types[typeName]; ok && info.Kind == TypeStruct && e.Name == typeName {
			fields := make([]frontend.StructFieldInit, 0, len(info.Fields))
			for i, field := range info.Fields {
				if i >= len(e.Args) {
					break
				}
				fields = append(fields, frontend.StructFieldInit{Name: field.Name, Value: e.Args[i]})
			}
			return resourceSourceForStructFieldLeaf(
				fields,
				typeName,
				leaf,
				funcs,
				types,
				module,
				imports,
				state,
			)
		}
		if source, handled, err := resourceSourceForEnumConstructorLeaf(
			e,
			typeName,
			leaf,
			funcs,
			types,
			module,
			imports,
			state,
		); handled || err != nil {
			return source, err
		}
		resolved, err := resolveCheckedCallName(e.Name, funcs, module, imports, e.At)
		if err != nil {
			return resourceSourceResult{}, err
		}
		sig, ok := funcs[resolved]
		if !ok {
			return resourceSourceResult{}, nil
		}
		if sig.ReturnResourceParam == regionUnknown && len(sig.ReturnResourceSummary) == 0 {
			return resourceSourceResult{unknown: true}, nil
		}
		provenances := sig.ReturnResourceSummary[leaf]
		if len(provenances) == 0 {
			return resourceSourceResult{}, nil
		}
		var merged resourceSourceResult
		for i, provenance := range provenances {
			source, err := resourceSourceForCallProvenance(
				e.Args,
				sig,
				provenance,
				funcs,
				types,
				module,
				imports,
				state,
				e.At,
			)
			if err != nil {
				return resourceSourceResult{}, err
			}
			if i == 0 {
				merged = source
				continue
			}
			merged = mergeResourceSourceResults(merged, source)
		}
		return merged, nil
	case *frontend.MatchExpr:
		var merged resourceSourceResult
		set := false
		for _, c := range e.Cases {
			source, err := resourceSourceForExprLeaf(
				c.Value,
				typeName,
				leaf,
				funcs,
				types,
				module,
				imports,
				state,
			)
			if err != nil {
				return resourceSourceResult{}, err
			}
			if !set {
				merged = source
				set = true
				continue
			}
			merged = mergeResourceSourceResults(merged, source)
		}
		return merged, nil
	case *frontend.CatchExpr:
		merged, err := resourceSourceForExprLeaf(
			e.Call,
			typeName,
			leaf,
			funcs,
			types,
			module,
			imports,
			state,
		)
		if err != nil {
			return resourceSourceResult{}, err
		}
		caseScopes := state.catchExprScopes[e]
		for i, c := range e.Cases {
			caseScopeID := regionNone
			if i < len(caseScopes) {
				caseScopeID = caseScopes[i]
			}
			if caseScopeID == regionNone {
				caseScopeID = patternBindingScopeID(c.Pattern, state)
			}
			var source resourceSourceResult
			err := withActiveScope(state, caseScopeID, func() error {
				var err error
				source, err = resourceSourceForExprLeaf(
					c.Value,
					typeName,
					leaf,
					funcs,
					types,
					module,
					imports,
					state,
				)
				return err
			})
			if err != nil {
				return resourceSourceResult{}, err
			}
			merged = mergeResourceSourceResults(merged, source)
		}
		return merged, nil
	default:
		return resourceSourceResult{}, nil
	}
}

func resourceSourceForOptionalWrappedLeaf(
	expr frontend.Expr,
	typeName string,
	leaf string,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) (resourceSourceResult, bool, error) {
	info, ok := types[typeName]
	if !ok || info.Kind != TypeOptional {
		return resourceSourceResult{}, false, nil
	}
	tail, ok := resourceLeafTail(leaf, "$elem")
	if !ok {
		return resourceSourceResult{}, false, nil
	}
	source, err := resourceSourceForExprLeaf(
		expr,
		info.ElemType,
		tail,
		funcs,
		types,
		module,
		imports,
		state,
	)
	if err != nil {
		return resourceSourceResult{}, true, err
	}
	if source.known || source.unknown || source.ambiguous {
		return source, true, nil
	}
	return resourceSourceResult{}, false, nil
}

func resourceSourceForStructFieldLeaf(
	fields []frontend.StructFieldInit,
	typeName string,
	leaf string,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) (resourceSourceResult, error) {
	info, ok := types[typeName]
	if !ok || info.Kind != TypeStruct {
		return resourceSourceResult{}, nil
	}
	byName := make(map[string]frontend.Expr, len(fields))
	for _, field := range fields {
		byName[field.Name] = field.Value
	}
	for _, field := range info.Fields {
		tail, ok := resourceLeafTail(leaf, field.Name)
		if !ok {
			continue
		}
		value := byName[field.Name]
		if value == nil {
			return resourceSourceResult{}, nil
		}
		return resourceSourceForExprLeaf(
			value,
			field.TypeName,
			tail,
			funcs,
			types,
			module,
			imports,
			state,
		)
	}
	return resourceSourceResult{}, nil
}

func resourceSourceForEnumConstructorLeaf(
	call *frontend.CallExpr,
	typeName string,
	leaf string,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) (resourceSourceResult, bool, error) {
	info, ok := types[typeName]
	if !ok || info.Kind != TypeEnum {
		return resourceSourceResult{}, false, nil
	}
	caseType, caseInfo, found, err := resolveEnumCaseConstructorCall(call, types, module, imports)
	if err != nil {
		return resourceSourceResult{}, true, err
	}
	if !found || caseType != typeName {
		return resourceSourceResult{}, false, nil
	}
	for i, payloadType := range caseInfo.PayloadTypes {
		if i >= len(call.Args) {
			break
		}
		tail, ok := resourceLeafTail(leaf, resourceEnumPayloadPath("", caseInfo.Ordinal, i))
		if !ok {
			continue
		}
		source, err := resourceSourceForExprLeaf(
			call.Args[i],
			payloadType,
			tail,
			funcs,
			types,
			module,
			imports,
			state,
		)
		return source, true, err
	}
	return resourceSourceResult{}, true, nil
}

func resourceLeafTail(leaf string, head string) (string, bool) {
	return semanticsresources.LeafTail(leaf, head)
}

// ---- checker_stmt_return.go ----

func checkReturnStmt(
	s *frontend.ReturnStmt,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	returnType string,
	borrowedParams map[string]struct{},
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) error {
	tname := ""
	regionID := regionNone
	handledFunctionReturn := false
	callerSig, callerSigOK := currentCallerSignature(effects, funcs)
	if closure, ok := s.Value.(*frontend.ClosureExpr); ok {
		if !callerSigOK || !callerSig.ReturnFunctionType {
			return unsupportedFunctionValueEscapeError(s.At, callbackArgumentName(s.Value))
		}
		targetInfo := functionReturnLocalInfo(callerSig)
		if err := validateFunctionTypedClosureAssignment(
			"return",
			targetInfo,
			closure,
			locals,
			funcs,
			types,
			module,
			imports,
			closure.At,
		); err != nil {
			return err
		}
		closureSymbol := closure.Name
		if _, ok := funcs[closureSymbol]; !ok && module != "" {
			qualified := qualifyName(module, closure.Name)
			if _, ok := funcs[qualified]; ok {
				closureSymbol = qualified
				closure.Name = qualified
			}
		}
		targetSig, ok := funcs[closureSymbol]
		if !ok {
			return fmt.Errorf(
				"%s: unknown function symbol '%s'",
				frontend.FormatPos(s.At),
				closureSymbol,
			)
		}
		validationSig := targetSig
		if len(closure.Captures) > 0 {
			captureSlots, err := functionCaptureSlotCount(closure.Captures, types)
			if err != nil {
				return err
			}
			if captureSlots < 1 || captureSlots > FnPtrEnvSlotCount {
				if captureSlots < 1 {
					return unsupportedFunctionTypedReturnCaptureError(
						s.At,
						"closure literal",
						captureSlots,
					)
				}
				escapeKind, handleValue, err := classifyCallableEscape(
					callableBoundaryReturn,
					closure.Captures,
					types,
				)
				if err != nil {
					return err
				}
				if analysis != nil {
					analysis.returnFunctionEscapeKind = escapeKind
					analysis.returnFunctionHandleValue = handleValue
				}
			}
			validationSig.ParamTypes = append([]string(nil), callerSig.ReturnFunctionParams...)
			validationSig.ParamOwnership = append(
				[]string(nil),
				callerSig.ReturnFunctionParamOwnership...)
		}
		if err := validateReturnedFunctionSignature(
			callerSig,
			validationSig,
			s.At,
			callbackArgumentName(s.Value),
		); err != nil {
			return err
		}
		if analysis != nil {
			if analysis.returnFunctionSymbol == "" {
				analysis.returnFunctionSymbol = closureSymbol
			}
			recordReturnFunctionTargetMutableGlobalUse(analysis, targetSig)
			recordReturnFunctionCaptures(analysis, closure.Captures)
		}
		tname = returnType
		regionID = regionNone
		handledFunctionReturn = true
	}
	if id, ok := s.Value.(*frontend.IdentExpr); ok {
		if localInfo, exists := locals[id.Name]; exists && localInfo.FunctionTypeValue {
			if !callerSigOK || !callerSig.ReturnFunctionType {
				return unsupportedFunctionValueEscapeError(s.At, id.Name)
			}
			if localInfo.FunctionValue == "" {
				validationSig, err := buildInterfaceFuncSig(id.Name, funcSigSpec{
					ParamTypes:          append([]string(nil), localInfo.FunctionParamTypes...),
					ParamOwnership:      append([]string(nil), localInfo.FunctionParamOwnership...),
					ReturnType:          localInfo.FunctionReturnType,
					ReturnOwnership:     localInfo.FunctionReturnOwnership,
					ThrowsType:          localInfo.FunctionThrowsType,
					ReturnRegionParam:   regionNone,
					ReturnResourceParam: regionNone,
					Effects:             append([]string(nil), localInfo.FunctionEffects...),
				}, types)
				if err != nil {
					return err
				}
				if err := validateReturnedFunctionSignature(
					callerSig,
					validationSig,
					s.At,
					id.Name,
				); err != nil {
					return err
				}
				if analysis != nil {
					analysis.returnFunctionParamName = localInfo.FunctionParamName
					if analysis.returnFunctionParamName == "" {
						analysis.returnFunctionParamName = id.Name
					}
					if localInfo.FunctionTouchesMutableGlobals {
						analysis.returnFunctionTouchesMutableGlobals = true
					}
				}
				tname = localInfo.TypeName
				regionID = regionNone
				handledFunctionReturn = true
			} else {
				if localInfo.GenericFunctionValue {
					return unsupportedGenericFunctionTypedReturnError(s.At, id.Name)
				}
				targetSig, ok := funcs[localInfo.FunctionValue]
				if !ok {
					return fmt.Errorf(
						"%s: unknown function symbol '%s'",
						frontend.FormatPos(s.At),
						localInfo.FunctionValue,
					)
				}
				if targetSig.Generic {
					return unsupportedGenericFunctionTypedReturnError(s.At, id.Name)
				}
				validationSig := targetSig
				if localInfo.FunctionReturnType != "" {
					explicitSlots, err := functionParamSlotCount(localInfo.FunctionParamTypes, types)
					if err != nil {
						return err
					}
					hiddenSlots := targetSig.ParamSlots - explicitSlots
					if hiddenSlots < 0 || (hiddenSlots > FnPtrEnvSlotCount && !localInfo.FunctionHandleValue) {
						return unsupportedFunctionTypedReturnCaptureError(s.At, id.Name, hiddenSlots)
					}
					if hiddenSlots > FnPtrEnvSlotCount && analysis != nil {
						analysis.returnFunctionEscapeKind = localInfo.FunctionEscapeKind
						analysis.returnFunctionHandleValue = localInfo.FunctionHandleValue
					}
					validationSig.ParamTypes = append([]string(nil), localInfo.FunctionParamTypes...)
					validationSig.ParamOwnership = append([]string(nil), localInfo.FunctionParamOwnership...)
					validationSig.ParamSlots = explicitSlots
					validationSig.ReturnType = localInfo.FunctionReturnType
					validationSig.ReturnOwnership = localInfo.FunctionReturnOwnership
					validationSig.ThrowsType = localInfo.FunctionThrowsType
					validationSig.Effects = append([]string(nil), localInfo.FunctionEffects...)
				}
				if err := validateReturnedFunctionSignature(
					callerSig,
					validationSig,
					s.At,
					id.Name,
				); err != nil {
					return err
				}
				if analysis != nil {
					if analysis.returnFunctionSymbol == "" {
						analysis.returnFunctionSymbol = localInfo.FunctionValue
					}
					if localInfo.FunctionTouchesMutableGlobals {
						analysis.returnFunctionTouchesMutableGlobals = true
					}
					recordReturnFunctionTargetMutableGlobalUse(analysis, targetSig)
					recordReturnFunctionCaptures(analysis, localInfo.FunctionCaptures)
					recordReturnFunctionCaptures(analysis, localInfo.FunctionEscapeCaptures)
				}
				tname = localInfo.TypeName
				regionID = regionNone
				handledFunctionReturn = true
			}
		} else if localInfo, exists := locals[id.Name]; exists && localInfo.FunctionValue != "" {
			if !callerSigOK || !callerSig.ReturnFunctionType {
				return unsupportedFunctionValueEscapeError(s.At, id.Name)
			}
			if localInfo.GenericFunctionValue {
				return unsupportedGenericFunctionTypedReturnError(s.At, id.Name)
			}
			targetSig, ok := funcs[localInfo.FunctionValue]
			if !ok {
				return fmt.Errorf(
					"%s: unknown function symbol '%s'",
					frontend.FormatPos(s.At),
					localInfo.FunctionValue,
				)
			}
			if targetSig.Generic {
				return unsupportedGenericFunctionTypedReturnError(s.At, id.Name)
			}
			validationSig := targetSig
			if len(localInfo.FunctionCaptures) > 0 {
				captureSlots, err := functionCaptureSlotCount(localInfo.FunctionCaptures, types)
				if err != nil {
					return err
				}
				if captureSlots < 1 {
					return unsupportedFunctionTypedReturnCaptureError(s.At, id.Name, captureSlots)
				}
				if captureSlots > FnPtrEnvSlotCount {
					escapeKind, handleValue, err := classifyCallableEscape(
						callableBoundaryReturn,
						localInfo.FunctionCaptures,
						types,
					)
					if err != nil {
						return err
					}
					if analysis != nil {
						analysis.returnFunctionEscapeKind = escapeKind
						analysis.returnFunctionHandleValue = handleValue
					}
				}
				validationSig.ParamTypes = append([]string(nil), callerSig.ReturnFunctionParams...)
				validationSig.ParamOwnership = append([]string(nil), callerSig.ReturnFunctionParamOwnership...)
			}
			if err := validateReturnedFunctionSignature(
				callerSig,
				validationSig,
				s.At,
				id.Name,
			); err != nil {
				return err
			}
			if analysis != nil {
				if analysis.returnFunctionSymbol == "" {
					analysis.returnFunctionSymbol = localInfo.FunctionValue
				}
				if localInfo.FunctionTouchesMutableGlobals {
					analysis.returnFunctionTouchesMutableGlobals = true
				}
				recordReturnFunctionTargetMutableGlobalUse(analysis, targetSig)
				recordReturnFunctionCaptures(analysis, localInfo.FunctionCaptures)
				recordReturnFunctionCaptures(analysis, localInfo.FunctionEscapeCaptures)
			}
			tname = returnType
			regionID = regionNone
			handledFunctionReturn = true
		} else if globalInfo, exists := globals[id.Name]; exists && globalInfo.FunctionTypeValue {
			if !callerSigOK || !callerSig.ReturnFunctionType {
				return unsupportedFunctionValueEscapeError(s.At, id.Name)
			}
			markMutableFunctionTypedGlobalSource(s.Value, globals, analysis)
			if globalInfo.FunctionValue == "" {
				if globalInfo.Mutable {
					return unsupportedImportedMutableFunctionTypedGlobalUseError(s.At, id.Name)
				}
				return unsupportedFunctionTypedGlobalTargetError(s.At, id.Name)
			}
			validationSig, err := buildInterfaceFuncSig(id.Name, funcSigSpec{
				ParamTypes:          append([]string(nil), globalInfo.FunctionParamTypes...),
				ParamOwnership:      append([]string(nil), globalInfo.FunctionParamOwnership...),
				ReturnType:          globalInfo.FunctionReturnType,
				ReturnOwnership:     globalInfo.FunctionReturnOwnership,
				ThrowsType:          globalInfo.FunctionThrowsType,
				ReturnRegionParam:   regionNone,
				ReturnResourceParam: regionNone,
				Effects:             append([]string(nil), globalInfo.FunctionEffects...),
			}, types)
			if err != nil {
				return err
			}
			if err := validateReturnedFunctionSignature(
				callerSig,
				validationSig,
				s.At,
				id.Name,
			); err != nil {
				return err
			}
			if analysis != nil && !globalInfo.Mutable && analysis.returnFunctionSymbol == "" {
				analysis.returnFunctionSymbol = globalInfo.FunctionValue
			}
			if analysis != nil && !globalInfo.Mutable {
				if targetSig, ok := funcs[globalInfo.FunctionValue]; ok {
					recordReturnFunctionTargetMutableGlobalUse(analysis, targetSig)
				}
			}
			tname = globalInfo.TypeName
			regionID = regionNone
			handledFunctionReturn = true
		}
	}
	if !handledFunctionReturn {
		if id, ok := s.Value.(*frontend.IdentExpr); ok && callerSigOK &&
			callerSig.ReturnFunctionType {
			if _, exists := locals[id.Name]; exists {
				// Local function-typed values and local closure pointers are handled by
				// the local return path or by the normal expression checker.
			} else if _, exists := globals[id.Name]; exists {
				// Globals are ordinary values, not direct function symbols.
			} else {
				resolved, err := resolveCheckedCallName(id.Name, funcs, module, imports, id.At)
				if err == nil {
					targetSig, ok := funcs[resolved]
					if !ok {
						return fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(s.At), resolved)
					}
					if err := ensureFuncVisible(resolved, targetSig, module, id.At); err != nil {
						return err
					}
					if targetSig.Generic {
						return unsupportedGenericFunctionTypedReturnError(s.At, id.Name)
					}
					if err := validateReturnedFunctionSignature(callerSig, targetSig, s.At, id.Name); err != nil {
						return err
					}
					if analysis != nil {
						if analysis.returnFunctionSymbol == "" {
							analysis.returnFunctionSymbol = resolved
						}
						recordReturnFunctionTargetMutableGlobalUse(analysis, targetSig)
					}
					id.Name = resolved
					tname = returnType
					regionID = regionNone
					handledFunctionReturn = true
				}
			}
		}
	}
	if !handledFunctionReturn {
		if fieldInfo, ok, err := resolveFunctionFieldArgument(s.Value, locals); err != nil {
			return err
		} else if ok {
			if !callerSigOK || !callerSig.ReturnFunctionType {
				return unsupportedFunctionValueEscapeError(s.At, callbackArgumentName(s.Value))
			}
			if fieldInfo.FunctionValue == "" {
				if !functionFieldInfoHasTargetSet(fieldInfo) {
					return fmt.Errorf(("%s: returning function-typed value '%s' requires a " +
						"symbol-backed non-capturing function value in this MVP"), frontend.FormatPos(
						s.At,
					), callbackArgumentName(
						s.Value,
					))
				}
				fieldSig, err := functionFieldInfoSig(fieldInfo)
				if err != nil {
					return err
				}
				if err := validateReturnedFunctionSignature(
					callerSig,
					fieldSig,
					s.At,
					callbackArgumentName(s.Value),
				); err != nil {
					return err
				}
				if analysis != nil {
					analysis.returnFunctionParamName = fieldInfo.FunctionParamName
					if fieldInfo.FunctionTouchesMutableGlobals {
						analysis.returnFunctionTouchesMutableGlobals = true
					}
					recordReturnFunctionCaptures(analysis, fieldInfo.FunctionCaptures)
					recordReturnFunctionCaptures(analysis, fieldInfo.FunctionEscapeCaptures)
				}
				tname = returnType
				regionID = regionNone
				handledFunctionReturn = true
			} else {
				targetSig, ok := funcs[fieldInfo.FunctionValue]
				if !ok {
					return fmt.Errorf(
						"%s: unknown function symbol '%s'",
						frontend.FormatPos(s.At),
						fieldInfo.FunctionValue,
					)
				}
				if targetSig.Generic {
					return unsupportedGenericFunctionTypedReturnError(s.At, callbackArgumentName(s.Value))
				}
				validationSig := targetSig
				if fieldInfo.FunctionReturnType != "" {
					explicitSlots, err := functionParamSlotCount(fieldInfo.FunctionParamTypes, types)
					if err != nil {
						return err
					}
					hiddenSlots := targetSig.ParamSlots - explicitSlots
					if hiddenSlots < 0 || (hiddenSlots > FnPtrEnvSlotCount && !fieldInfo.FunctionHandleValue) {
						return unsupportedFunctionTypedReturnCaptureError(
							s.At,
							callbackArgumentName(s.Value),
							hiddenSlots,
						)
					}
					if hiddenSlots > FnPtrEnvSlotCount && analysis != nil {
						analysis.returnFunctionEscapeKind = fieldInfo.FunctionEscapeKind
						analysis.returnFunctionHandleValue = fieldInfo.FunctionHandleValue
					}
					validationSig.ParamTypes = append([]string(nil), fieldInfo.FunctionParamTypes...)
					validationSig.ParamOwnership = append([]string(nil), fieldInfo.FunctionParamOwnership...)
					validationSig.ParamSlots = explicitSlots
					validationSig.ReturnType = fieldInfo.FunctionReturnType
					validationSig.ReturnOwnership = fieldInfo.FunctionReturnOwnership
					validationSig.Effects = append([]string(nil), fieldInfo.FunctionEffects...)
				}
				if err := validateReturnedFunctionSignature(
					callerSig,
					validationSig,
					s.At,
					callbackArgumentName(s.Value),
				); err != nil {
					return err
				}
				if analysis != nil {
					if analysis.returnFunctionSymbol == "" {
						analysis.returnFunctionSymbol = fieldInfo.FunctionValue
					}
					if fieldInfo.FunctionTouchesMutableGlobals {
						analysis.returnFunctionTouchesMutableGlobals = true
					}
					recordReturnFunctionTargetMutableGlobalUse(analysis, targetSig)
					recordReturnFunctionCaptures(analysis, fieldInfo.FunctionCaptures)
					recordReturnFunctionCaptures(analysis, fieldInfo.FunctionEscapeCaptures)
				}
				tname = returnType
				regionID = regionNone
				handledFunctionReturn = true
			}
		} else if fieldAccess, fieldOK := s.Value.(*frontend.FieldAccessExpr); fieldOK && callerSigOK && callerSig.ReturnFunctionType {
			if globalInfo, globalSig, globalOK, err := resolveFunctionTypedGlobalFieldAccess(
				fieldAccess,
				globals,
				funcs,
			); err != nil {
				return err
			} else if globalOK {
				if err := validateReturnedFunctionSignature(
					callerSig,
					globalSig,
					s.At,
					callbackArgumentName(s.Value),
				); err != nil {
					return err
				}
				if analysis != nil {
					if analysis.returnFunctionSymbol == "" {
						analysis.returnFunctionSymbol = globalInfo.FunctionValue
					}
					recordReturnFunctionTargetMutableGlobalUse(analysis, globalSig)
				}
				tname = returnType
				regionID = regionNone
				handledFunctionReturn = true
			} else if resolved, importedOK := resolveImportedFunctionFieldAccess(
				fieldAccess,
				funcs,
				module,
				imports,
			); importedOK {
				targetSig := funcs[resolved]
				if targetSig.Generic {
					return unsupportedGenericFunctionTypedReturnError(s.At, callbackArgumentName(s.Value))
				}
				if err := validateReturnedFunctionSignature(
					callerSig,
					targetSig,
					s.At,
					callbackArgumentName(s.Value),
				); err != nil {
					return err
				}
				if analysis != nil {
					if analysis.returnFunctionSymbol == "" {
						analysis.returnFunctionSymbol = resolved
					}
					recordReturnFunctionTargetMutableGlobalUse(analysis, targetSig)
				}
				tname = returnType
				regionID = regionNone
				handledFunctionReturn = true
			}
		}
	}
	if !handledFunctionReturn {
		var err error
		tname, regionID, err = checkExprWithEffects(
			s.Value,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
		if err != nil {
			return err
		}
	}
	if callerSigOK && callerSig.ReturnFunctionType && !handledFunctionReturn {
		return unsupportedFunctionTypedReturnSourceError(s.At)
	}
	if err := checkWholeOwnershipValueAvailable(s.Value, types, module, imports, state); err != nil {
		return err
	}
	if surfaceType, ok := surfaceEphemeralValueType(returnType, types); ok &&
		!surfaceEphemeralReturnAllowed(analysis, surfaceType) {
		return lifetimeDiagnosticf(
			s.At,
			("surface value '%s' cannot escape via return; keep Surface " +
				"Frame/Event/DrawContext values local to the active Surface " +
				"turn"),
			surfaceType,
		)
	}
	if surfaceFramePixelsEscapeExpr(s.Value, locals, globals, types, analysis) {
		return lifetimeDiagnosticf(
			s.At,
			("surface frame pixels cannot escape via return; keep " +
				"Frame.pixels local to the active Surface frame"),
		)
	}
	if typeMayContainRegion(tname, types) || typeMayContainPtr(tname, types) {
		if err := checkBorrowedReturnContract(
			s.Value,
			returnType,
			callerSig,
			callerSigOK,
			borrowedParams,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
			s.At,
		); err != nil {
			return err
		}
	}
	if typeMayContainRegion(tname, types) {
		tree := regionTreeForExpr(tname, s.Value, regionID, types, state)
		if err := checkRegionTreeWithinScope(tree, regionNone, s.At, state); err != nil {
			return err
		}
		if !typeContainsResourceHandle(tname, types) {
			if err := state.recordReturnRegionSummary(tree, s.At); err != nil {
				return err
			}
		}
		regionID = commonRegionFromTree(tree)
		if id, ok := s.Value.(*frontend.IdentExpr); ok {
			if _, borrowed := borrowedParams[id.Name]; borrowed {
				return lifetimeDiagnosticf(
					s.At,
					"borrowed local '%s' cannot escape via return",
					id.Name,
				)
			}
		}
	}
	if tname == "ptr" {
		if borrowedName, borrowed := borrowedPtrOwnerFromExpr(s.Value, state, borrowedParams); borrowed {
			return lifetimeDiagnosticf(
				s.At,
				"borrowed local '%s' cannot escape via return",
				borrowedName,
			)
		}
	}
	resourceReturnType := tname
	if typeContainsResourceHandle(returnType, types) {
		resourceReturnType = returnType
	}
	if typeContainsResourceHandle(resourceReturnType, types) {
		summary, unknown, err := returnResourceSummaryForExpr(
			s.Value,
			resourceReturnType,
			funcs,
			types,
			module,
			imports,
			state,
		)
		if err != nil {
			return err
		}
		if unknown {
			state.recordUnknownReturnResource()
		} else if err := state.recordReturnResourceSummary(summary, s.At); err != nil {
			return err
		}
	}
	if len(state.returnRegionSummary) == 0 {
		if err := state.recordReturnRegion(regionID, s.At); err != nil {
			return err
		}
	}
	secretTainted, err := exprSecretTainted(
		s.Value,
		tname,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
		analysis,
	)
	if err != nil {
		return err
	}
	if analysis.underSecretControl() {
		secretTainted = true
	}
	if secretTainted {
		analysis.returnSecretTaint = true
		if analysis.rejectSecretReturn {
			return privacyDiagnosticf(
				s.At,
				"secret-tainted value cannot be returned from @export function '%s'",
				analysis.exportedFuncName,
			)
		}
		if !analysis.allowSecretReturn {
			return privacyDiagnosticf(
				s.At,
				"secret-tainted value requires semantic clause 'privacy' before return",
			)
		}
	}
	if !typesCompatibleWithNullPtr(returnType, tname, s.Value) {
		return fmt.Errorf(
			"%s: return type mismatch: expected '%s', got '%s'",
			frontend.FormatPos(s.At),
			returnType,
			tname,
		)
	}
	if analysis != nil {
		returnFields, err := functionFieldsFromReturnedStructExpr(
			returnType,
			s.Value,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return err
		}
		if len(returnFields) > 0 || len(analysis.returnFunctionFields) > 0 {
			if len(analysis.returnFunctionFields) == 0 {
				analysis.returnFunctionFields = functionFieldReturnSnapshotMap(returnFields)
			} else {
				for fieldName, field := range returnFields {
					mergeFunctionFieldInfoIntoMap(
						analysis.returnFunctionFields,
						fieldName,
						functionFieldInfoAsReturnSnapshot(field),
					)
				}
			}
		}
		returnPayloadFields, err := enumPayloadFieldsFromReturnedStructExpr(
			returnType,
			s.Value,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return err
		}
		if len(returnPayloadFields) > 0 || len(analysis.returnEnumPayloadFields) > 0 {
			if len(analysis.returnEnumPayloadFields) == 0 {
				analysis.returnEnumPayloadFields = functionFieldReturnSnapshotMap(
					returnPayloadFields,
				)
			} else {
				for fieldName, field := range returnPayloadFields {
					mergeFunctionFieldInfoIntoMap(
						analysis.returnEnumPayloadFields,
						fieldName,
						functionFieldInfoAsReturnSnapshot(field),
					)
				}
			}
		}
		returnPayloads, err := enumPayloadFunctionsFromReturnedEnumExpr(
			returnType,
			s.Value,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return err
		}
		if len(returnPayloads) > 0 || len(analysis.returnEnumPayloadFunctions) > 0 {
			if len(analysis.returnEnumPayloadFunctions) == 0 {
				analysis.returnEnumPayloadFunctions = functionFieldReturnSnapshotMap(returnPayloads)
			} else {
				for payloadKey, payload := range returnPayloads {
					mergeFunctionFieldInfoIntoMap(
						analysis.returnEnumPayloadFunctions,
						payloadKey,
						functionFieldInfoAsReturnSnapshot(payload),
					)
				}
			}
		}
	}
	state.reachable = false
	return nil
}

// ---- checker_stmts.go ----

func checkStmts(
	stmts []frontend.Stmt,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	returnType string,
	borrowedParams map[string]struct{},
	inoutParams map[string]struct{},
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) error {
	if state != nil {
		state.pushDeferCaptureFrame()
		defer state.popDeferCaptureFrame()
	}
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			if err := effects.require(s.At, "io"); err != nil {
				return err
			}
			tname, _, err := checkExprWithEffects(
				s.Value,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
			)
			if err != nil {
				return err
			}
			if !isPrintableType(tname, types) {
				return fmt.Errorf("%s: print expects str or []u8", frontend.FormatPos(s.At))
			}
			secretTainted, err := exprSecretTainted(
				s.Value,
				tname,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				analysis,
			)
			if err != nil {
				return err
			}
			if analysis.underSecretControl() {
				secretTainted = true
			}
			if secretTainted {
				return privacyDiagnosticf(s.At, "secret-tainted value cannot be printed")
			}
		case *frontend.BreakStmt:
			if state.loopDepth == 0 {
				return fmt.Errorf("%s: break outside loop", frontend.FormatPos(s.At))
			}
			recordLoopFlowExit(state, "break", analysis)
			if err := state.checkPendingDeferCaptures(s.At); err != nil {
				return err
			}
			state.reachable = false
			return nil
		case *frontend.ContinueStmt:
			if state.loopDepth == 0 {
				return fmt.Errorf("%s: continue outside loop", frontend.FormatPos(s.At))
			}
			recordLoopFlowExit(state, "continue", analysis)
			if err := state.checkPendingDeferCaptures(s.At); err != nil {
				return err
			}
			state.reachable = false
			return nil
		case *frontend.FreeStmt:
			if err := effects.requireAll(s.At, []string{"islands", "mem"}); err != nil {
				return err
			}
			tname, _, err := checkExprWithEffects(
				s.Value,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
			)
			if err != nil {
				return err
			}
			if tname != "island" {
				return fmt.Errorf("%s: free expects island, got '%s'", frontend.FormatPos(s.At), tname)
			}
			if !s.Implicit && !state.inUnsafe() {
				return effectDiagnosticf(s.At, "free is only allowed in unsafe blocks")
			}
			source, err := resourceSourceForExpr(s.Value, funcs, module, imports, state)
			if err != nil {
				return err
			}
			if source.ambiguous {
				return ownershipDiagnosticf(s.Value.Pos(), "resource expression mixes resource provenance")
			}
			if source.unknown {
				name, _ := resourcePathForExpr(s.Value)
				if name == "" {
					name = "<resource>"
				}
				return ownershipDiagnosticf(s.Value.Pos(), (("ambiguous resource provenance for '%s' " +
					"after control-flow ") +
					"merge"), name)
			}
			if source.known {
				state.markResourceFinalized(source.name, "freed", s.Value.Pos())
			}
		case *frontend.ReturnStmt:
			if err := checkReturnStmt(
				s,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				returnType,
				borrowedParams,
				state,
				effects,
				analysis,
			); err != nil {
				return err
			}
		case *frontend.ThrowStmt:
			if state.throwType == "" {
				return fmt.Errorf("%s: throw is only allowed in throwing functions", frontend.FormatPos(s.At))
			}
			tname, _, err := checkExprWithEffects(
				s.Value,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
			)
			if err != nil {
				return err
			}
			if !typesCompatibleWithNullPtr(state.throwType, tname, s.Value) {
				return fmt.Errorf(
					"%s: throw type mismatch: expected '%s', got '%s'",
					frontend.FormatPos(s.At),
					state.throwType,
					tname,
				)
			}
			if surfaceType, ok := surfaceEphemeralValueType(state.throwType, types); ok {
				return lifetimeDiagnosticf(s.At, ("surface value '%s' cannot escape via throw; keep Surface " +
					"Frame/Event/DrawContext values local to the active Surface " +
					"turn"), surfaceType)
			}
			if surfaceType, ok := surfaceEphemeralValueType(tname, types); ok {
				return lifetimeDiagnosticf(s.At, ("surface value '%s' cannot escape via throw; keep Surface " +
					"Frame/Event/DrawContext values local to the active Surface " +
					"turn"), surfaceType)
			}
			if surfaceFramePixelsEscapeExpr(s.Value, locals, globals, types, analysis) {
				return lifetimeDiagnosticf(s.At, ("surface frame pixels cannot escape via throw; keep " +
					"Frame.pixels local to the active Surface frame"))
			}
			if typeMayContainRegion(
				tname,
				types,
			) || typeMayContainPtr(
				tname,
				types,
			) || typeMayContainRegion(
				state.throwType,
				types,
			) || typeMayContainPtr(
				state.throwType,
				types,
			) {
				if err := checkBorrowedEscape(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
					return lifetimeDiagnosticf(s.At, "borrowed local '%s' cannot escape via throw", borrowedName)
				}); err != nil {
					return err
				}
			}
			if typeMayContainRegion(tname, types) || typeMayContainRegion(state.throwType, types) {
				if err := checkRegionTreeWithinScope(
					regionTreeForExpr(tname, s.Value, regionNone, types, state),
					regionNone,
					s.At,
					state,
				); err != nil {
					return err
				}
			}
			if typeContainsResourceHandle(state.throwType, types) {
				summary, unknown, err := returnResourceSummaryForExpr(
					s.Value,
					state.throwType,
					funcs,
					types,
					module,
					imports,
					state,
				)
				if err != nil {
					return err
				}
				if !unknown {
					if err := state.recordThrowResourceSummary(summary, s.At); err != nil {
						return err
					}
				}
			}
			secretTainted, err := exprSecretTainted(
				s.Value,
				tname,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				analysis,
			)
			if err != nil {
				return err
			}
			if analysis.underSecretControl() {
				secretTainted = true
			}
			if secretTainted {
				if analysis.rejectSecretReturn {
					return privacyDiagnosticf(s.At, (("secret-tainted value cannot be thrown from " +
						"@export function ") +
						"'%s'"), analysis.exportedFuncName)
				}
				if !analysis.allowSecretReturn {
					return privacyDiagnosticf(s.At, ("secret-tainted value requires semantic clause 'privacy' " +
						"before throw"))
				}
			}
			state.reachable = false
		case *frontend.DeferStmt:
			if err := validateDeferBodyControl(s.Body, 0); err != nil {
				return err
			}
			scopeID := regionNone
			if state.deferScopes != nil {
				if scoped, ok := state.deferScopes[s]; ok {
					scopeID = scoped
				}
			}
			captures := collectDeferCaptures(s.Body, locals)
			if err := withActiveScope(state, scopeID, func() error {
				return checkDeferBody(
					s.Body,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					returnType,
					borrowedParams,
					inoutParams,
					state,
					effects,
					analysis,
				)
			}); err != nil {
				return err
			}
			state.registerDeferCaptures(captures)
		case *frontend.IslandStmt:
			if err := effects.requireAll(s.At, []string{"alloc", "islands", "mem"}); err != nil {
				return err
			}
			sizeType, _, err := checkExprWithEffects(
				s.Size,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
			)
			if err != nil {
				return err
			}
			if !isInt32Like(sizeType) {
				return fmt.Errorf("%s: island size must be i32/u8", frontend.FormatPos(s.At))
			}
			if err := state.enterIsland(s.Name); err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			if err := checkStmts(
				s.Body,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				returnType,
				borrowedParams,
				inoutParams,
				state,
				effects,
				analysis,
			); err != nil {
				state.exitIsland()
				return err
			}
			state.exitIsland()
		case *frontend.LetStmt:
			state.clearConsumed(s.Name)
			resolved, err := resolveTypeName(&s.Type, module, imports)
			if err != nil {
				return err
			}
			s.Type.Name = resolved
			if _, err := ensureTypeInfo(resolved, types); err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			valType := ""
			valRegion := regionNone
			handledFunctionSymbol := false
			if s.Type.Kind == frontend.TypeRefFunction {
				if _, ok := s.Value.(*frontend.ClosureExpr); ok {
					if info, exists := locals[s.Name]; exists && info.FunctionTypeValue {
						valType = resolved
						valRegion = regionNone
						handledFunctionSymbol = true
					}
				} else if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if info, exists := locals[s.Name]; exists && info.FunctionTypeValue {
						if info.FunctionValue == "" {
							if sourceInfo, sourceExists := locals[id.Name]; !sourceExists || !sourceInfo.FunctionTypeValue {
								return unsupportedFunctionTypedLocalInitializerSourceError(s.At, s.Name)
							}
						}
						valType = resolved
						valRegion = regionNone
						handledFunctionSymbol = true
					}
				} else if _, ok := s.Value.(*frontend.FieldAccessExpr); ok {
					if info, exists := locals[s.Name]; exists && info.FunctionTypeValue && info.FunctionValue != "" {
						valType = resolved
						valRegion = regionNone
						handledFunctionSymbol = true
					}
				}
			}
			if !handledFunctionSymbol {
				var checkErr error
				valType, valRegion, checkErr = checkExprWithEffects(
					s.Value,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
				)
				if checkErr != nil {
					return checkErr
				}
			}
			if !typesCompatibleWithNullPtr(resolved, valType, s.Value) {
				return fmt.Errorf(
					"%s: type mismatch: expected '%s', got '%s'",
					frontend.FormatPos(s.At),
					resolved,
					valType,
				)
			}
			if err := checkWholeOwnershipValueAvailable(s.Value, types, module, imports, state); err != nil {
				return err
			}
			secretTainted, err := exprSecretTainted(
				s.Value,
				valType,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				analysis,
			)
			if err != nil {
				return err
			}
			if typeUsesSecret(resolved, types) {
				secretTainted = true
			}
			if analysis.underSecretControl() {
				secretTainted = true
			}
			analysis.setLocalSecretTaint(s.Name, secretTainted)
			if source, ok := surfaceFramePixelsSourceExpr(s.Value, locals, globals, types, analysis); ok {
				bindLocalSurfaceFramePixelsSource(locals, analysis, s.Name, source)
			} else {
				bindLocalSurfaceFramePixelsSource(locals, analysis, s.Name, "")
			}
			if owner, ok := surfaceHandleOwnerPathExprWithAnalysis(
				s.Value,
				locals,
				globals,
				types,
				analysis,
			); ok {
				analysis.setLocalSurfaceHandleOwner(s.Name, owner)
			} else {
				analysis.setLocalSurfaceHandleOwner(s.Name, "")
			}
			bindSurfaceFrameOwnerForLocal(s.Name, resolved, s.Value, analysis)
			if typeMayContainRegion(resolved, types) {
				scopeID := localScopeID(s.Name, state)
				if err := checkRegionTreeWithinScope(
					regionTreeForExpr(resolved, s.Value, valRegion, types, state),
					scopeID,
					s.At,
					state,
				); err != nil {
					return err
				}
				bindRegionTreeFromExpr(s.Name, resolved, s.Value, valRegion, types, state)
			}
			state.bindRegion(s.Name, valRegion)
			bindBorrowedPtrAliasFromExpr(
				s.Name,
				resolved,
				s.Value,
				types,
				module,
				imports,
				state,
				borrowedParams,
			)
			if err := bindResourceTreeFromExpr(
				s.Name,
				resolved,
				s.Value,
				funcs,
				types,
				module,
				imports,
				state,
			); err != nil {
				return err
			}
			if err := bindOwnedRegionSliceOwnerFromExpr(
				s.Name,
				resolved,
				s.Value,
				types,
				module,
				imports,
				state,
			); err != nil {
				return err
			}
		case *frontend.AssignStmt:
			if s.CompoundValue != nil && compoundIndexTargetHasSideEffects(s.Target) {
				return fmt.Errorf(("%s: compound index assignment target with side effects is " +
					"not supported; use an explicit temporary index"), frontend.FormatPos(s.At))
			}
			if idx, ok := s.Target.(*frontend.IndexExpr); ok {
				if err := rejectRepresentationMetadataExprAssignment(
					idx.Base,
					locals,
					globals,
					types,
				); err != nil {
					return err
				}
				indexType, _, err := checkExprWithEffects(
					idx.Index,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
				)
				if err != nil {
					return err
				}
				if !isInt32Like(indexType) {
					return fmt.Errorf("%s: index must be i32/u8", frontend.FormatPos(idx.At))
				}
				if _, _, err := checkExprWithEffects(
					idx.Base,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
				); err != nil {
					return err
				}
			}
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				if err := state.checkNotConsumed(id.Name, s.At); err != nil {
					return err
				}
				if g, ok := globals[id.Name]; ok {
					if !g.Mutable {
						if g.Const {
							return fmt.Errorf("%s: cannot assign to const '%s'", frontend.FormatPos(s.At), id.Name)
						}
						return fmt.Errorf("%s: cannot assign to val '%s'", frontend.FormatPos(s.At), id.Name)
					}
					if g.FunctionTypeValue {
						if analysis != nil {
							analysis.touchesMutableGlobals = true
						}
						targetInfo := LocalInfo{
							TypeName:                g.TypeName,
							SlotCount:               FnPtrSlotCount,
							FunctionTypeValue:       true,
							FunctionParamTypes:      append([]string(nil), g.FunctionParamTypes...),
							FunctionParamOwnership:  append([]string(nil), g.FunctionParamOwnership...),
							FunctionReturnType:      g.FunctionReturnType,
							FunctionReturnOwnership: g.FunctionReturnOwnership,
							FunctionThrowsType:      g.FunctionThrowsType,
							FunctionEffects:         append([]string(nil), g.FunctionEffects...),
						}
						allowCapturedGlobalSnapshot, err := allowCapturedGlobalFunctionSnapshot(
							s.Value,
							locals,
							types,
							state,
						)
						if err != nil {
							return err
						}
						if err := validateFunctionTypedAssignmentValue(
							id.Name,
							targetInfo,
							s.Value,
							locals,
							globals,
							funcs,
							types,
							module,
							imports,
							s.At,
							allowCapturedGlobalSnapshot,
							callableBoundaryGlobal,
						); err != nil {
							return err
						}
						continue
					}
					valType, _, err := checkExprWithEffects(
						s.Value,
						locals,
						globals,
						funcs,
						types,
						module,
						imports,
						state,
						effects,
						analysis,
					)
					if err != nil {
						return err
					}
					if !typesCompatibleWithNullPtr(g.TypeName, valType, s.Value) {
						return fmt.Errorf(
							"%s: type mismatch: expected '%s', got '%s'",
							frontend.FormatPos(s.At),
							g.TypeName,
							valType,
						)
					}
					if err := checkWholeOwnershipValueAvailable(
						s.Value,
						types,
						module,
						imports,
						state,
					); err != nil {
						return err
					}
					if surfaceType, ok := surfaceEphemeralValueType(g.TypeName, types); ok {
						return lifetimeDiagnosticf(s.At, (("surface value '%s' cannot escape via global " +
							"assignment to ") +
							"'%s'; keep Surface Frame/Event/DrawContext values local to " +
							"the active Surface turn"), surfaceType, id.Name)
					}
					if surfaceType, ok := surfaceEphemeralValueType(valType, types); ok {
						return lifetimeDiagnosticf(s.At, (("surface value '%s' cannot escape via global " +
							"assignment to ") +
							"'%s'; keep Surface Frame/Event/DrawContext values local to " +
							"the active Surface turn"), surfaceType, id.Name)
					}
					if surfaceFramePixelsEscapeExpr(s.Value, locals, globals, types, analysis) {
						return lifetimeDiagnosticf(s.At, (("surface frame pixels cannot escape via global " +
							"assignment to ") +
							"'%s'; keep Frame.pixels local to the active Surface frame"), id.Name)
					}
					if typeMayContainRegion(valType, types) || typeMayContainRegion(g.TypeName, types) ||
						typeMayContainPtr(valType, types) || typeMayContainPtr(g.TypeName, types) {
						if err := checkBorrowedAggregateEscape(
							s.Value,
							g.TypeName,
							"be stored in global",
							locals,
							globals,
							funcs,
							types,
							module,
							imports,
							state,
							effects,
							analysis,
							s.At,
						); err != nil {
							return err
						}
						if err := checkBorrowedEscape(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
							decision := islandkernel.CanStoreGlobal(islandkernel.EscapeRequest{
								Ref: islandKernelSemanticBorrowRef(borrowedName),
							})
							if decision.Decision == islandkernel.Accept {
								return nil
							}
							return lifetimeDiagnosticf(s.At, (("borrowed local '%s' cannot escape via global " +
								"assignment to ") +
								"'%s' (%s)"), borrowedName, id.Name, decision.Reason.Code)
						}); err != nil {
							return err
						}
					}
					if valType == "ptr" {
						if borrowedName, borrowed := borrowedPtrOwnerFromExpr(
							s.Value,
							state,
							borrowedParams,
						); borrowed {
							return lifetimeDiagnosticf(s.At, (("borrowed local '%s' cannot escape via global " +
								"assignment to ") +
								"'%s'"), borrowedName, id.Name)
						}
					}
					secretTainted, err := exprSecretTainted(
						s.Value,
						valType,
						locals,
						globals,
						funcs,
						types,
						module,
						imports,
						analysis,
					)
					if err != nil {
						return err
					}
					if analysis.underSecretControl() {
						secretTainted = true
					}
					if secretTainted {
						return privacyDiagnosticf(
							s.At,
							"secret-tainted value cannot be stored in global '%s'",
							id.Name,
						)
					}
					continue
				}
			}
			targetInfo, targetType, err := resolveAssignTarget(s.Target, locals, globals, types)
			if err != nil {
				return err
			}
			if !targetInfo.Global {
				if err := checkLocalScope(targetInfo.Name, state, s.At); err != nil {
					return err
				}
			}
			if !targetInfo.Mutable {
				if targetInfo.Const {
					return fmt.Errorf("%s: cannot assign to const '%s'", frontend.FormatPos(s.At), targetInfo.Name)
				}
				return fmt.Errorf("%s: cannot assign to val '%s'", frontend.FormatPos(s.At), targetInfo.Name)
			}
			targetOwnershipPath := ""
			if path, ok := canonicalOwnershipAccessPath(s.Target); ok {
				if err := state.checkAssignableOwnershipPath(path, s.At); err != nil {
					return err
				}
				targetOwnershipPath = path
			}
			handledFunctionAssignment := false
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				if localInfo, exists := locals[id.Name]; exists && localInfo.FunctionTypeValue {
					markMutableFunctionTypedGlobalSource(s.Value, globals, analysis)
					if err := validateFunctionTypedAssignmentValue(
						id.Name,
						localInfo,
						s.Value,
						locals,
						globals,
						funcs,
						types,
						module,
						imports,
						s.At,
						true,
						callableBoundaryLocal,
					); err != nil {
						return err
					}
					if err := updateFunctionTypedLocalAssignmentMetadata(
						id.Name,
						s.Value,
						locals,
						globals,
						funcs,
						types,
						module,
						imports,
					); err != nil {
						return err
					}
					handledFunctionAssignment = true
				}
			} else if targetName, fieldInfo, ok, err := functionFieldLocalInfoFromExpr(
				s.Target,
				locals,
				types,
			); err != nil {
				return err
			} else if ok {
				markMutableFunctionTypedGlobalSource(s.Value, globals, analysis)
				if err := validateFunctionTypedAssignmentValue(
					targetName,
					fieldInfo,
					s.Value,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					s.At,
					true,
					callableBoundaryStructField,
				); err != nil {
					return err
				}
				if err := updateFunctionTypedFieldAssignmentMetadata(
					targetName,
					fieldInfo,
					s.Value,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
				); err != nil {
					return err
				}
				handledFunctionAssignment = true
			}
			valType := targetType
			valRegion := regionNone
			if !handledFunctionAssignment {
				var err error
				valType, valRegion, err = checkExprWithEffects(
					s.Value,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
				)
				if err != nil {
					return err
				}
			}
			if !typesCompatibleWithNullPtr(targetType, valType, s.Value) {
				return fmt.Errorf(
					"%s: type mismatch: expected '%s', got '%s'",
					frontend.FormatPos(s.At),
					targetType,
					valType,
				)
			}
			if err := checkWholeOwnershipValueAvailable(s.Value, types, module, imports, state); err != nil {
				return err
			}
			secretTainted := false
			if !handledFunctionAssignment {
				var err error
				secretTainted, err = exprSecretTainted(
					s.Value,
					valType,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					analysis,
				)
				if err != nil {
					return err
				}
			}
			if analysis.underSecretControl() {
				secretTainted = true
			}
			if secretTainted && targetInfo.ActorField {
				return privacyDiagnosticf(s.At, ("secret-tainted value cannot be stored in actor state field " +
					"'%s'"), targetInfo.Name)
			}
			if targetInfo.Global {
				if surfaceType, ok := surfaceEphemeralValueType(targetType, types); ok {
					return lifetimeDiagnosticf(s.At, (("surface value '%s' cannot escape via global " +
						"assignment to ") +
						"'%s'; keep Surface Frame/Event/DrawContext values local to " +
						"the active Surface turn"), surfaceType, targetInfo.Name)
				}
				if surfaceType, ok := surfaceEphemeralValueType(valType, types); ok {
					return lifetimeDiagnosticf(s.At, (("surface value '%s' cannot escape via global " +
						"assignment to ") +
						"'%s'; keep Surface Frame/Event/DrawContext values local to " +
						"the active Surface turn"), surfaceType, targetInfo.Name)
				}
				if surfaceFramePixelsEscapeExpr(s.Value, locals, globals, types, analysis) {
					return lifetimeDiagnosticf(s.At, (("surface frame pixels cannot escape via global " +
						"assignment to ") +
						"'%s'; keep Frame.pixels local to the active Surface frame"), targetInfo.Name)
				}
				if typeMayContainRegion(valType, types) || typeMayContainRegion(targetType, types) ||
					typeMayContainPtr(valType, types) || typeMayContainPtr(targetType, types) {
					if err := checkBorrowedAggregateEscape(
						s.Value,
						targetType,
						"be stored in global",
						locals,
						globals,
						funcs,
						types,
						module,
						imports,
						state,
						effects,
						analysis,
						s.At,
					); err != nil {
						return err
					}
					if err := checkBorrowedEscape(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
						decision := islandkernel.CanStoreGlobal(islandkernel.EscapeRequest{
							Ref: islandKernelSemanticBorrowRef(borrowedName),
						})
						if decision.Decision == islandkernel.Accept {
							return nil
						}
						return lifetimeDiagnosticf(s.At, (("borrowed local '%s' cannot escape via global " +
							"assignment to ") +
							"'%s' (%s)"), borrowedName, targetInfo.Name, decision.Reason.Code)
					}); err != nil {
						return err
					}
				}
				if secretTainted {
					return privacyDiagnosticf(
						s.At,
						"secret-tainted value cannot be stored in global '%s'",
						targetInfo.Name,
					)
				}
				continue
			}
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				analysis.setLocalSecretTaint(id.Name, secretTainted || typeUsesSecret(targetType, types))
				if source, ok := surfaceFramePixelsSourceExpr(s.Value, locals, globals, types, analysis); ok {
					bindLocalSurfaceFramePixelsSource(locals, analysis, id.Name, source)
				} else {
					bindLocalSurfaceFramePixelsSource(locals, analysis, id.Name, "")
				}
				if owner, ok := surfaceHandleOwnerPathExprWithAnalysis(
					s.Value,
					locals,
					globals,
					types,
					analysis,
				); ok {
					analysis.setLocalSurfaceHandleOwner(id.Name, owner)
				} else {
					analysis.setLocalSurfaceHandleOwner(id.Name, "")
				}
				if targetType == surfaceFrameTypeName {
					analysis.clearSurfaceFramePresented(id.Name)
				}
				bindSurfaceFrameOwnerForLocal(id.Name, targetType, s.Value, analysis)
				if localInfo, exists := locals[id.Name]; exists && localInfo.Mutable {
					if fields, err := functionFieldsFromReturnedStructExpr(
						targetType,
						s.Value,
						locals,
						globals,
						funcs,
						types,
						module,
						imports,
					); err != nil {
						return err
					} else if len(fields) > 0 || len(localInfo.FunctionFields) > 0 {
						localInfo.FunctionFields = cloneFunctionFieldMap(fields)
						locals[id.Name] = localInfo
					}
					if payloadFields, err := enumPayloadFieldsFromReturnedStructExpr(
						targetType,
						s.Value,
						locals,
						globals,
						funcs,
						types,
						module,
						imports,
					); err != nil {
						return err
					} else if len(payloadFields) > 0 || len(localInfo.EnumPayloadFields) > 0 {
						localInfo.EnumPayloadFields = cloneFunctionFieldMap(payloadFields)
						locals[id.Name] = localInfo
					}
				}
			} else if info, ok := types[targetType]; ok && info.Kind == TypeStruct {
				if err := updateFunctionTypedStructFieldAssignmentMetadata(
					s.Target,
					targetType,
					s.Value,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
				); err != nil {
					return err
				}
				if err := updateEnumPayloadStructFieldAssignmentMetadata(
					s.Target,
					targetType,
					s.Value,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
				); err != nil {
					return err
				}
			} else if info, ok := types[targetType]; ok && info.Kind == TypeEnum {
				if err := updateEnumPayloadStructFieldAssignmentMetadata(
					s.Target,
					targetType,
					s.Value,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
				); err != nil {
					return err
				}
			} else if secretTainted {
				analysis.setLocalSecretTaint(targetInfo.Name, true)
			}
			if _, outParam := inoutParams[targetInfo.Name]; outParam {
				if surfaceType, ok := surfaceEphemeralValueType(targetType, types); ok {
					return lifetimeDiagnosticf(s.At, ("surface value '%s' cannot escape via inout assignment to " +
						"'%s'; keep Surface Frame/Event/DrawContext values local to " +
						"the active Surface turn"), surfaceType, targetInfo.Name)
				}
				if surfaceType, ok := surfaceEphemeralValueType(valType, types); ok {
					return lifetimeDiagnosticf(s.At, ("surface value '%s' cannot escape via inout assignment to " +
						"'%s'; keep Surface Frame/Event/DrawContext values local to " +
						"the active Surface turn"), surfaceType, targetInfo.Name)
				}
				if surfaceFramePixelsEscapeExpr(s.Value, locals, globals, types, analysis) {
					return lifetimeDiagnosticf(s.At, (("surface frame pixels cannot escape via inout " +
						"assignment to ") +
						"'%s'; keep Frame.pixels local to the active Surface frame"), targetInfo.Name)
				}
				if valRegion < regionNone || typeMayContainPtr(
					targetType,
					types,
				) || typeMayContainPtr(
					valType,
					types,
				) {
					if err := checkBorrowedInoutEscape(
						s.Value,
						targetInfo.Name,
						s.At,
						locals,
						globals,
						funcs,
						types,
						module,
						imports,
						state,
						effects,
						analysis,
					); err != nil {
						return err
					}
				}
			}
			if _, ok := s.Target.(*frontend.IndexExpr); !ok {
				targetResourceName := targetInfo.Name
				if path, ok := resourcePathForExpr(s.Target); ok {
					targetResourceName = path
				}
				if targetType == surfaceFrameTypeName {
					analysis.clearSurfaceFramePresented(targetResourceName)
				}
				bindSurfaceFrameOwnerForLocal(targetResourceName, targetType, s.Value, analysis)
				if typeMayContainRegion(targetType, types) {
					scopeID := localScopeID(targetInfo.Name, state)
					if err := checkRegionTreeWithinScope(
						regionTreeForExpr(targetType, s.Value, valRegion, types, state),
						scopeID,
						s.At,
						state,
					); err != nil {
						return err
					}
					bindRegionTreeFromExpr(targetResourceName, targetType, s.Value, valRegion, types, state)
				}
				state.bindRegion(targetResourceName, valRegion)
				bindBorrowedPtrAliasFromExpr(
					targetResourceName,
					targetType,
					s.Value,
					types,
					module,
					imports,
					state,
					borrowedParams,
				)
				if err := bindResourceTreeFromExpr(
					targetResourceName,
					targetType,
					s.Value,
					funcs,
					types,
					module,
					imports,
					state,
				); err != nil {
					return err
				}
				if err := bindOwnedRegionSliceOwnerFromExpr(
					targetResourceName,
					targetType,
					s.Value,
					types,
					module,
					imports,
					state,
				); err != nil {
					return err
				}
			}
			if targetOwnershipPath != "" {
				state.clearConsumedTree(targetOwnershipPath)
			}
		case *frontend.IfStmt:
			condType, _, err := checkExprWithEffects(
				s.Cond,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
			)
			if err != nil {
				return err
			}
			if !isConditionType(condType) {
				return fmt.Errorf("%s: condition must be bool or i32/u8", frontend.FormatPos(s.At))
			}
			condSecretTainted, err := exprSecretTainted(
				s.Cond,
				condType,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				analysis,
			)
			if err != nil {
				return err
			}
			scopeIDs := branchScopeInfo{thenID: regionNone, elseID: regionNone}
			if scoped, ok := state.ifScopes[s]; ok {
				scopeIDs = scoped
			}
			before := copyRegionVars(state.regionVars)
			beforeFlow := snapshotFlow(state)
			beforeTaint := analysis.copySecretTaint()
			state.regionVars = copyRegionVars(before)
			restoreFlow(state, beforeFlow)
			analysis.restoreSecretTaint(beforeTaint)
			if err := withActiveScope(state, scopeIDs.thenID, func() error {
				return analysis.withSecretControl(condSecretTainted, func() error {
					return checkStmts(
						s.Then,
						locals,
						globals,
						funcs,
						types,
						module,
						imports,
						returnType,
						borrowedParams,
						inoutParams,
						state,
						effects,
						analysis,
					)
				})
			}); err != nil {
				return err
			}
			thenVars := copyRegionVars(state.regionVars)
			thenFlow := snapshotFlow(state)
			thenTaint := analysis.copySecretTaint()
			var elseVars map[string]int
			var elseFlow flowSnapshot
			var elseTaint map[string]bool
			if len(s.Else) > 0 {
				state.regionVars = copyRegionVars(before)
				restoreFlow(state, beforeFlow)
				analysis.restoreSecretTaint(beforeTaint)
				if err := withActiveScope(state, scopeIDs.elseID, func() error {
					return analysis.withSecretControl(condSecretTainted, func() error {
						return checkStmts(
							s.Else,
							locals,
							globals,
							funcs,
							types,
							module,
							imports,
							returnType,
							borrowedParams,
							inoutParams,
							state,
							effects,
							analysis,
						)
					})
				}); err != nil {
					return err
				}
				elseVars = copyRegionVars(state.regionVars)
				elseFlow = snapshotFlow(state)
				elseTaint = analysis.copySecretTaint()
			} else {
				elseVars = before
				elseFlow = beforeFlow
				elseTaint = beforeTaint
			}
			mergeControlFlowWithLabels(
				state,
				analysis,
				thenVars,
				thenFlow,
				thenTaint,
				elseVars,
				elseFlow,
				elseTaint,
				"then",
				"else",
			)
			markUnknownRegions(state)
		case *frontend.IfLetStmt:
			valueType, valueRegion, err := checkExprWithEffects(
				s.Value,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
			)
			if err != nil {
				return err
			}
			valueSecretTainted, err := exprSecretTainted(
				s.Value,
				valueType,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				analysis,
			)
			if err != nil {
				return err
			}
			valueInfo, valueInfoOK := types[valueType]
			if s.Pattern == nil {
				if _, ok := optionalElemName(valueType); !ok {
					return fmt.Errorf(
						"%s: if let requires optional value, got '%s'",
						frontend.FormatPos(s.At),
						valueType,
					)
				}
			} else if !valueInfoOK || (valueInfo.Kind != TypeOptional && valueInfo.Kind != TypeEnum) {
				return fmt.Errorf(
					"%s: if let pattern requires optional or enum value, got '%s'",
					frontend.FormatPos(s.At),
					valueType,
				)
			} else if err := validateIfLetPattern(
				s.Pattern,
				valueType,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
			); err != nil {
				return err
			}
			valueResourcePath := s.ValueLocal
			valueOwnershipPath := valueResourcePath
			if path, ok := resourcePathForExpr(s.Value); ok {
				valueOwnershipPath = path
			}
			if valueResourcePath != "" {
				if err := bindResourceTreeFromExpr(
					valueResourcePath,
					valueType,
					s.Value,
					funcs,
					types,
					module,
					imports,
					state,
				); err != nil {
					return err
				}
				bindRegionTreeFromExpr(valueResourcePath, valueType, s.Value, valueRegion, types, state)
			} else if path, ok := resourcePathForExpr(s.Value); ok {
				valueResourcePath = path
			}
			scopeIDs := branchScopeInfo{thenID: regionNone, elseID: regionNone}
			if scoped, ok := state.ifLetScopes[s]; ok {
				scopeIDs = scoped
			}
			before := copyRegionVars(state.regionVars)
			beforeFlow := snapshotFlow(state)
			beforeTaint := analysis.copySecretTaint()
			analysis.setLocalSecretTaint(s.ValueLocal, valueSecretTainted)
			beforeTaint = mergeSecretTaintMaps(beforeTaint, analysis.copySecretTaint())
			state.regionVars = copyRegionVars(before)
			restoreFlow(state, beforeFlow)
			analysis.restoreSecretTaint(beforeTaint)
			if err := withActiveScope(state, scopeIDs.thenID, func() error {
				if err := bindPatternOwnershipAliases(
					s.Pattern,
					s.Name,
					valueOwnershipPath,
					valueType,
					types,
					module,
					imports,
					state,
				); err != nil {
					return err
				}
				if err := bindPatternBorrowedPtrAliases(
					s.Pattern,
					s.Name,
					valueOwnershipPath,
					valueType,
					types,
					module,
					imports,
					state,
				); err != nil {
					return err
				}
				if err := bindPatternResourceLocals(
					s.Pattern,
					s.Name,
					valueResourcePath,
					valueType,
					types,
					module,
					imports,
					state,
				); err != nil {
					return err
				}
				if err := bindPatternRegionLocals(
					s.Pattern,
					s.Name,
					valueResourcePath,
					valueType,
					types,
					module,
					imports,
					state,
				); err != nil {
					return err
				}
				bindPatternSecretTaintLocals(s.Pattern, s.Name, valueSecretTainted, analysis)
				return analysis.withSecretControl(valueSecretTainted, func() error {
					return checkStmts(
						s.Then,
						locals,
						globals,
						funcs,
						types,
						module,
						imports,
						returnType,
						borrowedParams,
						inoutParams,
						state,
						effects,
						analysis,
					)
				})
			}); err != nil {
				return err
			}
			thenVars := copyRegionVars(state.regionVars)
			thenFlow := snapshotFlow(state)
			thenTaint := analysis.copySecretTaint()
			var elseVars map[string]int
			var elseFlow flowSnapshot
			var elseTaint map[string]bool
			if len(s.Else) > 0 {
				state.regionVars = copyRegionVars(before)
				restoreFlow(state, beforeFlow)
				analysis.restoreSecretTaint(beforeTaint)
				if err := withActiveScope(state, scopeIDs.elseID, func() error {
					return analysis.withSecretControl(valueSecretTainted, func() error {
						return checkStmts(
							s.Else,
							locals,
							globals,
							funcs,
							types,
							module,
							imports,
							returnType,
							borrowedParams,
							inoutParams,
							state,
							effects,
							analysis,
						)
					})
				}); err != nil {
					return err
				}
				elseVars = copyRegionVars(state.regionVars)
				elseFlow = snapshotFlow(state)
				elseTaint = analysis.copySecretTaint()
			} else {
				elseVars = before
				elseFlow = beforeFlow
				elseTaint = beforeTaint
			}
			mergeControlFlowWithLabels(
				state,
				analysis,
				thenVars,
				thenFlow,
				thenTaint,
				elseVars,
				elseFlow,
				elseTaint,
				"then",
				"else",
			)
			markUnknownRegions(state)
		case *frontend.WhileStmt:
			condType, _, err := checkExprWithEffects(
				s.Cond,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
			)
			if err != nil {
				return err
			}
			if !isConditionType(condType) {
				return fmt.Errorf("%s: condition must be bool or i32/u8", frontend.FormatPos(s.At))
			}
			condSecretTainted, err := exprSecretTainted(
				s.Cond,
				condType,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				analysis,
			)
			if err != nil {
				return err
			}
			bodyScopeID := regionNone
			if scoped, ok := state.whileScopes[s]; ok {
				bodyScopeID = scoped
			}
			before := copyRegionVars(state.regionVars)
			beforeFlow := snapshotFlow(state)
			beforeTaint := analysis.copySecretTaint()
			state.regionVars = copyRegionVars(before)
			restoreFlow(state, beforeFlow)
			analysis.restoreSecretTaint(beforeTaint)
			state.loopDepth++
			pushLoopFlowFrame(state)
			if err := withActiveScope(state, bodyScopeID, func() error {
				return analysis.withSecretControl(condSecretTainted, func() error {
					return checkStmts(
						s.Body,
						locals,
						globals,
						funcs,
						types,
						module,
						imports,
						returnType,
						borrowedParams,
						inoutParams,
						state,
						effects,
						analysis,
					)
				})
			}); err != nil {
				popLoopFlowFrame(state)
				state.loopDepth--
				return err
			}
			loopFrame := popLoopFlowFrame(state)
			state.loopDepth--
			bodyVars := copyRegionVars(state.regionVars)
			bodyFlow := snapshotFlow(state)
			bodyTaint := analysis.copySecretTaint()
			exits := []loopFlowExit{{label: "before", vars: before, flow: beforeFlow, taint: beforeTaint}}
			if bodyFlow.reachable {
				exits = append(
					exits,
					loopFlowExit{label: "body", vars: bodyVars, flow: bodyFlow, taint: bodyTaint},
				)
			}
			exits = append(exits, loopFrame.continues...)
			exits = append(exits, loopFrame.breaks...)
			mergeLoopFlowExits(state, analysis, exits)
			markUnknownRegions(state)
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				iterType, _, err := checkExprWithEffects(
					s.Iterable,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
				)
				if err != nil {
					return err
				}
				elemType, err := collectionElementType(iterType, types)
				if err != nil {
					return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
				}
				if loopInfo, ok := locals[s.Name]; ok && loopInfo.TypeName != elemType {
					return fmt.Errorf(("%s: for collection element type mismatch: local '%s' is %s, " +
						"iterable yields %s"), frontend.FormatPos(s.At), s.Name, loopInfo.TypeName, elemType)
				}
			} else {
				startType, _, err := checkExprWithEffects(
					s.Start,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
				)
				if err != nil {
					return err
				}
				endType, _, err := checkExprWithEffects(
					s.End,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
				)
				if err != nil {
					return err
				}
				if !isInt32Like(startType) || !isInt32Like(endType) {
					return fmt.Errorf("%s: for range bounds must be i32/u8", frontend.FormatPos(s.At))
				}
			}
			bodyScopeID := regionNone
			if scoped, ok := state.forScopes[s]; ok {
				bodyScopeID = scoped
			}
			before := copyRegionVars(state.regionVars)
			beforeFlow := snapshotFlow(state)
			beforeTaint := analysis.copySecretTaint()
			state.regionVars = copyRegionVars(before)
			restoreFlow(state, beforeFlow)
			analysis.restoreSecretTaint(beforeTaint)
			state.loopDepth++
			pushLoopFlowFrame(state)
			if err := withActiveScope(state, bodyScopeID, func() error {
				return checkStmts(
					s.Body,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					returnType,
					borrowedParams,
					inoutParams,
					state,
					effects,
					analysis,
				)
			}); err != nil {
				popLoopFlowFrame(state)
				state.loopDepth--
				return err
			}
			loopFrame := popLoopFlowFrame(state)
			state.loopDepth--
			bodyVars := copyRegionVars(state.regionVars)
			bodyFlow := snapshotFlow(state)
			bodyTaint := analysis.copySecretTaint()
			exits := []loopFlowExit{{label: "before", vars: before, flow: beforeFlow, taint: beforeTaint}}
			if bodyFlow.reachable {
				exits = append(
					exits,
					loopFlowExit{label: "body", vars: bodyVars, flow: bodyFlow, taint: bodyTaint},
				)
			}
			exits = append(exits, loopFrame.continues...)
			exits = append(exits, loopFrame.breaks...)
			mergeLoopFlowExits(state, analysis, exits)
			markUnknownRegions(state)
		case *frontend.MatchStmt:
			scrutType, scrutRegion, err := checkExprWithEffects(
				s.Value,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
			)
			if err != nil {
				return err
			}
			scrutSecretTainted, err := exprSecretTainted(
				s.Value,
				scrutType,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				analysis,
			)
			if err != nil {
				return err
			}
			scrutInfo, scrutInfoOK := types[scrutType]
			if !isInt32Like(scrutType) {
				info, ok := types[scrutType]
				if !ok || (info.Kind != TypeEnum && info.Kind != TypeOptional) {
					return fmt.Errorf("%s: match value must be enum or i32/u8", frontend.FormatPos(s.At))
				}
			}
			scrutineeResourcePath := s.ScrutineeLocal
			scrutineeOwnershipPath := scrutineeResourcePath
			if path, ok := resourcePathForExpr(s.Value); ok {
				scrutineeOwnershipPath = path
			}
			if scrutineeResourcePath != "" {
				if err := bindResourceTreeFromExpr(
					scrutineeResourcePath,
					scrutType,
					s.Value,
					funcs,
					types,
					module,
					imports,
					state,
				); err != nil {
					return err
				}
				bindRegionTreeFromExpr(scrutineeResourcePath, scrutType, s.Value, scrutRegion, types, state)
			} else if path, ok := resourcePathForExpr(s.Value); ok {
				scrutineeResourcePath = path
			}
			seenDefault := false
			seenPatterns := map[string]frontend.Position{}
			scrutineeFunctionPayloads, err := enumPayloadFunctionValuesForMatchExpr(
				s.Value,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				scrutType,
			)
			if err != nil {
				return err
			}
			before := copyRegionVars(state.regionVars)
			beforeFlow := snapshotFlow(state)
			beforeTaint := analysis.copySecretTaint()
			analysis.setLocalSecretTaint(s.ScrutineeLocal, scrutSecretTainted)
			beforeTaint = mergeSecretTaintMaps(beforeTaint, analysis.copySecretTaint())
			merged := copyRegionVars(before)
			mergedFlow := beforeFlow
			mergedTaint := beforeTaint
			labels := []string{"fallthrough"}
			caseScopes := state.matchCaseScopes[s]
			for i, c := range s.Cases {
				if seenDefault {
					return fmt.Errorf("%s: match default must be last", frontend.FormatPos(c.At))
				}
				if c.Default {
					seenDefault = true
				} else {
					patType := ""
					if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
						if !scrutInfoOK || scrutInfo.Kind != TypeOptional {
							return fmt.Errorf(
								"%s: some pattern requires optional match value",
								frontend.FormatPos(some.At),
							)
						}
						patType = optionalSomePatternType
					} else if enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr); ok {
						caseType, caseInfo, found, err := resolveEnumCasePattern(enumPat, types, module, imports)
						if err != nil {
							return err
						}
						if !found {
							return fmt.Errorf(
								"%s: unknown enum pattern '%s.%s'",
								frontend.FormatPos(enumPat.At),
								enumPat.TypeName,
								enumPat.CaseName,
							)
						}
						if err := validateEnumCasePatternPayload(enumPat, caseType, caseInfo, module); err != nil {
							return err
						}
						patType = caseType
					} else {
						var err error
						patType, _, err = checkExprWithEffects(
							c.Pattern,
							locals,
							globals,
							funcs,
							types,
							module,
							imports,
							state,
							effects,
							analysis,
						)
						if err != nil {
							return err
						}
					}
					if scrutInfoOK && scrutInfo.Kind == TypeOptional && patType != "none" && patType != optionalSomePatternType {
						return fmt.Errorf(("%s: optional match supports only 'none', 'some(name)', and " +
							"'_' patterns"), frontend.FormatPos(c.At))
					}
					if !matchPatternCompatible(scrutType, patType, types) {
						return fmt.Errorf(
							"%s: match pattern type mismatch: expected '%s', got '%s'",
							frontend.FormatPos(c.At),
							scrutType,
							patType,
						)
					}
					if c.Guard == nil {
						if key := matchPatternKey(c.Pattern, patType); key != "" {
							if first, exists := seenPatterns[key]; exists {
								return fmt.Errorf(
									"%s: duplicate match pattern (first at %s)",
									frontend.FormatPos(c.At),
									frontend.FormatPos(first),
								)
							}
							seenPatterns[key] = c.At
						}
					}
				}
				state.regionVars = copyRegionVars(before)
				restoreFlow(state, beforeFlow)
				analysis.restoreSecretTaint(beforeTaint)
				caseScopeID := regionNone
				if i < len(caseScopes) {
					caseScopeID = caseScopes[i]
				}
				if err := withActiveScope(state, caseScopeID, func() error {
					if err := bindEnumPatternFunctionPayloadLocals(
						c.Pattern,
						scrutineeFunctionPayloads,
						locals,
						types,
						module,
						imports,
					); err != nil {
						return err
					}
					if err := bindPatternOwnershipAliases(
						c.Pattern,
						"",
						scrutineeOwnershipPath,
						scrutType,
						types,
						module,
						imports,
						state,
					); err != nil {
						return err
					}
					if err := bindPatternBorrowedPtrAliases(
						c.Pattern,
						"",
						scrutineeOwnershipPath,
						scrutType,
						types,
						module,
						imports,
						state,
					); err != nil {
						return err
					}
					if err := bindPatternResourceLocals(
						c.Pattern,
						"",
						scrutineeResourcePath,
						scrutType,
						types,
						module,
						imports,
						state,
					); err != nil {
						return err
					}
					if err := bindPatternRegionLocals(
						c.Pattern,
						"",
						scrutineeResourcePath,
						scrutType,
						types,
						module,
						imports,
						state,
					); err != nil {
						return err
					}
					bindPatternSecretTaintLocals(c.Pattern, "", scrutSecretTainted, analysis)
					caseControlSecretTainted := scrutSecretTainted
					if c.Guard != nil {
						guardType, _, err := checkExprWithEffects(
							c.Guard,
							locals,
							globals,
							funcs,
							types,
							module,
							imports,
							state,
							effects,
							analysis,
						)
						if err != nil {
							return err
						}
						if guardType != "bool" {
							return fmt.Errorf("%s: match guard must be Bool", frontend.FormatPos(c.Guard.Pos()))
						}
						guardSecretTainted, err := exprSecretTainted(
							c.Guard,
							guardType,
							locals,
							globals,
							funcs,
							types,
							module,
							imports,
							analysis,
						)
						if err != nil {
							return err
						}
						caseControlSecretTainted = caseControlSecretTainted || guardSecretTainted
					}
					return analysis.withSecretControl(caseControlSecretTainted, func() error {
						return checkStmts(
							c.Body,
							locals,
							globals,
							funcs,
							types,
							module,
							imports,
							returnType,
							borrowedParams,
							inoutParams,
							state,
							effects,
							analysis,
						)
					})
				}); err != nil {
					return err
				}
				caseVars := copyRegionVars(state.regionVars)
				caseFlow := snapshotFlow(state)
				caseTaint := analysis.copySecretTaint()
				mergeControlFlowWithLabels(
					state,
					analysis,
					merged,
					mergedFlow,
					mergedTaint,
					caseVars,
					caseFlow,
					caseTaint,
					strings.Join(labels, "/"),
					fmt.Sprintf("case %d", i+1),
				)
				merged = copyRegionVars(state.regionVars)
				mergedFlow = snapshotFlow(state)
				mergedTaint = analysis.copySecretTaint()
				labels = append(labels, fmt.Sprintf("case %d", i+1))
			}
			if seenDefault {
				state.regionVars = merged
				restoreFlow(state, mergedFlow)
				analysis.restoreSecretTaint(mergedTaint)
			} else {
				mergeControlFlowWithLabels(
					state,
					analysis,
					before,
					beforeFlow,
					beforeTaint,
					merged,
					mergedFlow,
					mergedTaint,
					"before",
					"cases",
				)
			}
			markUnknownRegions(state)
		case *frontend.UnsafeStmt:
			scopeID := regionNone
			if scoped, ok := state.unsafeScopes[s]; ok {
				scopeID = scoped
			}
			state.enterUnsafe()
			if err := withActiveScope(state, scopeID, func() error {
				return checkStmts(
					s.Body,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					returnType,
					borrowedParams,
					inoutParams,
					state,
					effects,
					analysis,
				)
			}); err != nil {
				state.exitUnsafe()
				return err
			}
			state.exitUnsafe()
		case *frontend.ExprStmt:
			tname, _, err := checkExprWithEffects(
				s.Expr,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
			)
			if err != nil {
				return err
			}
			if _, err := exprSecretTainted(
				s.Expr,
				tname,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				analysis,
			); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s: unsupported statement", frontend.FormatPos(s.Pos()))
		}
		if err := state.checkPendingDeferCaptures(stmt.Pos()); err != nil {
			return err
		}
		if !state.reachable {
			return nil
		}
	}
	return nil
}

// ---- checker_world_globals.go ----

func collectWorldGlobals(
	world *module.World,
	checked *CheckedProgram,
	types map[string]*TypeInfo,
) error {
	for _, file := range world.Files {
		module := file.Module
		imports, err := collectImportAliases(file)
		if err != nil {
			return err
		}
		if len(file.Globals) == 0 {
			continue
		}

		fnNames := make(map[string]struct{}, len(file.Funcs))
		for _, fn := range file.Funcs {
			fnNames[fn.Name] = struct{}{}
		}

		globals := make(map[string]GlobalInfo, len(file.Globals))
		constValues := make(map[string]globalConstValue, len(file.Globals))
		var dataBlobs [][]byte
		for _, glob := range file.Globals {
			if glob == nil {
				continue
			}
			if _, exists := globals[glob.Name]; exists {
				return fmt.Errorf(
					"%s: duplicate global '%s'",
					frontend.FormatPos(glob.At),
					glob.Name,
				)
			}
			if _, exists := fnNames[glob.Name]; exists {
				return fmt.Errorf(
					"%s: global '%s' conflicts with function '%s'",
					frontend.FormatPos(glob.At),
					glob.Name,
					glob.Name,
				)
			}

			resolved := ""
			functionTypeValue := glob.Type.Kind == frontend.TypeRefFunction
			functionParamTypes := []string(nil)
			functionParamOwnership := []string(nil)
			functionReturnType := ""
			functionReturnOwnership := ""
			functionThrowsType := ""
			functionEffects := []string(nil)
			if glob.Type.Name != "" || glob.Type.Elem != nil || functionTypeValue {
				var err error
				if functionTypeValue {
					functionParamTypes, functionReturnType, functionEffects, err = functionTypeRefSignatureAndEffects(
						glob.Type,
						module,
						imports,
					)
					if err != nil {
						return err
					}
					functionParamOwnership = functionTypeRefParamOwnership(glob.Type)
					functionReturnOwnership = functionTypeRefReturnOwnership(glob.Type)
					functionThrowsType, err = functionTypeRefThrowsType(glob.Type, module, imports)
					if err != nil {
						return err
					}
				}
				resolved, err = resolveTypeName(&glob.Type, module, imports)
				if err != nil {
					return err
				}
			}
			if resolved == "" {
				if glob.Mutable {
					return fmt.Errorf(
						"%s: global var requires an explicit type annotation",
						frontend.FormatPos(glob.At),
					)
				}
				if glob.Init == nil {
					return fmt.Errorf(
						"%s: global val requires an initializer to infer its type",
						frontend.FormatPos(glob.At),
					)
				}
				if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
					return err
				}
				inferred, ok := inferGlobalConstExprType(glob.Init, constValues)
				if !ok {
					return fmt.Errorf(
						("%s: unsupported global val initializer (type inference " +
							"supports constant numeric/bool expressions)"),
						frontend.FormatPos(glob.At),
					)
				}
				resolved = inferred
			}
			glob.Type.Name = resolved
			if !isSupportedGlobalType(resolved, types) {
				return fmt.Errorf(
					("%s: global '%s' has unsupported type '%s' (allowed: i32, " +
						"bool, ptr, str, u8, u16, task.error)"),
					frontend.FormatPos(glob.At),
					glob.Name,
					resolved,
				)
			}
			typeInfo, err := ensureTypeInfo(resolved, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(glob.At), err)
			}
			if surfaceType, ok := surfaceEphemeralValueType(resolved, types); ok {
				return lifetimeDiagnosticf(
					glob.At,
					("surface value '%s' cannot be stored in global '%s'; keep " +
						"Surface Frame/Event/DrawContext values local to the active " +
						"Surface turn"),
					surfaceType,
					glob.Name,
				)
			}
			functionValue := ""
			if functionTypeValue {
				switch init := glob.Init.(type) {
				case *frontend.IdentExpr:
					if _, ok := fnNames[init.Name]; !ok {
						return unsupportedFunctionTypedGlobalSameModuleInitializerError(init.At, glob.Name)
					}
					functionValue = qualifyName(module, init.Name)
				case *frontend.FieldAccessExpr:
					resolved, targetSig, ok, err := resolveImportedFunctionGlobalInitializer(
						init,
						world,
						module,
						imports,
						types,
					)
					if err != nil {
						return err
					}
					if !ok {
						return unsupportedFunctionTypedGlobalImportedInitializerError(init.At, glob.Name)
					}
					if targetSig.Generic {
						return unsupportedGenericFunctionTypedGlobalInitializerError(
							init.At,
							callbackArgumentName(init),
							glob.Name,
						)
					}
					if err := validateFunctionTypeSymbolSignature(
						glob.Name,
						glob.Type,
						targetSig,
						module,
						imports,
						init.At,
					); err != nil {
						return err
					}
					functionValue = resolved
				case *frontend.ClosureExpr:
					if err := validateFunctionTypeLiteralBinding(
						glob.Name,
						glob.Type,
						init,
						nil,
						module,
						imports,
					); err != nil {
						return err
					}
					functionValue = qualifyName(module, init.Name)
				default:
					return unsupportedFunctionTypedGlobalInitializerSourceError(glob.At, glob.Name)
				}
			}
			stringInit := []byte(nil)
			hasStringInit := false
			if resolved == "str" {
				if glob.Init == nil {
					kind := "val"
					if glob.Mutable {
						kind = "var"
					}
					return fmt.Errorf(
						"%s: global %s '%s' initializer must be a string literal",
						frontend.FormatPos(glob.At),
						kind,
						glob.Name,
					)
				}
				lit, ok := glob.Init.(*frontend.StringLitExpr)
				if !ok {
					kind := "val"
					if glob.Mutable {
						kind = "var"
					}
					return fmt.Errorf(
						"%s: global %s '%s' initializer must be a string literal",
						frontend.FormatPos(glob.Init.Pos()),
						kind,
						glob.Name,
					)
				}
				hasStringInit = true
				stringInit = append([]byte(nil), lit.Value...)
			}

			dataIndex := len(dataBlobs)
			arrayBackings := collectGlobalArrayBackings(resolved, 0, types, map[string]bool{})
			globals[glob.Name] = GlobalInfo{
				DataIndex:               dataIndex,
				TypeName:                resolved,
				Mutable:                 glob.Mutable,
				Const:                   glob.Const,
				Public:                  declarationIsPublic(file, glob.Public),
				FunctionValue:           functionValue,
				FunctionTypeValue:       functionTypeValue,
				FunctionParamTypes:      functionParamTypes,
				FunctionParamOwnership:  functionParamOwnership,
				FunctionReturnType:      functionReturnType,
				FunctionReturnOwnership: functionReturnOwnership,
				FunctionThrowsType:      functionThrowsType,
				FunctionEffects:         functionEffects,
				HasStringLiteralInit:    hasStringInit,
				StringLiteralInit:       stringInit,
				ArrayBackings:           arrayBackings,
			}

			slots := typeInfo.SlotCount
			if slots <= 0 {
				slots = 1
			}
			slotData := make([][]byte, slots)
			for i := 0; i < slots; i++ {
				slotData[i] = make([]byte, 8)
			}
			buf := slotData[0]
			if glob.Mutable {
				if glob.Init != nil {
					switch resolved {
					case "i32", "u8", "u16", "c_int", "c_uint", "task.error":
						if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
							return err
						}
						v, ok := evalGlobalConstI32(glob.Init, constValues)
						if !ok {
							return fmt.Errorf(
								"%s: global var '%s' initializer must be an i32 constant expression",
								frontend.FormatPos(glob.Init.Pos()),
								glob.Name,
							)
						}
						if err := validateGlobalIntLikeRange(
							resolved,
							glob.Name,
							"var",
							glob.Init.Pos(),
							v,
						); err != nil {
							return err
						}
						binary.LittleEndian.PutUint64(buf, uint64(int64(v)))
					case "bool":
						if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
							return err
						}
						v, ok := evalGlobalConstBool(glob.Init, constValues)
						if !ok {
							return fmt.Errorf(
								"%s: global var '%s' initializer must be a bool constant expression",
								frontend.FormatPos(glob.Init.Pos()),
								glob.Name,
							)
						}
						if v {
							binary.LittleEndian.PutUint64(buf, 1)
						} else {
							binary.LittleEndian.PutUint64(buf, 0)
						}
					case "ptr":
						if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
							return err
						}
						v, ok := evalGlobalConstI32(glob.Init, constValues)
						if !ok {
							return fmt.Errorf(
								"%s: global var '%s' initializer for type ptr must be a constant 0",
								frontend.FormatPos(glob.Init.Pos()),
								glob.Name,
							)
						}
						if v != 0 {
							return fmt.Errorf(
								"%s: global var '%s' of type ptr only supports initializer 0",
								frontend.FormatPos(glob.Init.Pos()),
								glob.Name,
							)
						}
						binary.LittleEndian.PutUint64(buf, 0)
					case "str":
						binary.LittleEndian.PutUint64(slotData[0], 0)
						binary.LittleEndian.PutUint64(slotData[1], uint64(len(stringInit)))
					case "fnptr":
						// Function-typed mutable globals are lazily initialized by lowering
						// because function symbol addresses are not static data constants.
					default:
						if isSupportedSliceGlobalType(resolved) {
							return fmt.Errorf(
								"%s: global var '%s' initializer for type %s must be omitted",
								frontend.FormatPos(glob.Init.Pos()),
								glob.Name,
								resolved,
							)
						}
						if isSupportedOptionalPtrGlobalType(resolved) {
							if _, ok := glob.Init.(*frontend.NoneLitExpr); ok {
								break
							}
							return fmt.Errorf(
								"%s: global var '%s' initializer for type %s must be none",
								frontend.FormatPos(glob.Init.Pos()),
								glob.Name,
								resolved,
							)
						}
						if isSupportedOptionalSliceGlobalType(resolved) {
							if glob.Init == nil {
								break
							}
							if _, ok := glob.Init.(*frontend.NoneLitExpr); ok {
								break
							}
							return fmt.Errorf(
								"%s: global var '%s' initializer for type %s must be none",
								frontend.FormatPos(glob.Init.Pos()),
								glob.Name,
								resolved,
							)
						}
						if isSupportedOptionalAggregateGlobalType(resolved, types) {
							if _, ok := glob.Init.(*frontend.NoneLitExpr); ok {
								break
							}
							return fmt.Errorf(
								"%s: global var '%s' initializer for type %s must be none",
								frontend.FormatPos(glob.Init.Pos()),
								glob.Name,
								resolved,
							)
						}
						if isSupportedZeroedAggregateGlobalType(
							resolved,
							types,
							map[string]bool{},
						) {
							return fmt.Errorf(
								"%s: global var '%s' initializer for type %s must be omitted",
								frontend.FormatPos(glob.Init.Pos()),
								glob.Name,
								resolved,
							)
						}
						return fmt.Errorf(
							"%s: unsupported global type '%s'",
							frontend.FormatPos(glob.At),
							resolved,
						)
					}
				}
				dataBlobs = append(dataBlobs, slotData...)
				continue
			}
			if glob.Init == nil {
				switch resolved {
				case "ptr":
					binary.LittleEndian.PutUint64(buf, 0)
				case "i32", "u8", "u16", "c_int", "c_uint", "task.error":
					binary.LittleEndian.PutUint64(buf, 0)
					constValues[glob.Name] = globalConstValue{TypeName: resolved, I32: 0}
				case "bool":
					binary.LittleEndian.PutUint64(buf, 0)
					constValues[glob.Name] = globalConstValue{TypeName: "bool", Bool: false}
				case "str":
					binary.LittleEndian.PutUint64(slotData[0], 0)
					binary.LittleEndian.PutUint64(slotData[1], 0)
				default:
					if isSupportedSliceGlobalType(resolved) {
						dataBlobs = append(dataBlobs, slotData...)
						continue
					}
					if isSupportedOptionalPtrGlobalType(resolved) {
						dataBlobs = append(dataBlobs, slotData...)
						continue
					}
					if isSupportedOptionalSliceGlobalType(resolved) {
						dataBlobs = append(dataBlobs, slotData...)
						continue
					}
					if isSupportedOptionalAggregateGlobalType(resolved, types) {
						dataBlobs = append(dataBlobs, slotData...)
						continue
					}
					if isSupportedZeroedAggregateGlobalType(resolved, types, map[string]bool{}) {
						dataBlobs = append(dataBlobs, slotData...)
						continue
					}
					return fmt.Errorf(
						"%s: unsupported global type '%s'",
						frontend.FormatPos(glob.At),
						resolved,
					)
				}
				dataBlobs = append(dataBlobs, slotData...)
				continue
			}
			switch resolved {
			case "fnptr":
				// Function-typed globals carry symbol metadata in GlobalInfo; data slots
				// stay zeroed because this MVP only permits immutable direct symbols.
			case "ptr":
				if !isNullPtrLiteral(glob.Init) {
					return fmt.Errorf(
						"%s: global val '%s' of type ptr only supports initializer 0",
						frontend.FormatPos(glob.Init.Pos()),
						glob.Name,
					)
				}
				binary.LittleEndian.PutUint64(buf, 0)
			case "i32", "u8", "u16", "c_int", "c_uint", "task.error":
				if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
					return err
				}
				v, ok := evalGlobalConstI32(glob.Init, constValues)
				if !ok {
					return fmt.Errorf(
						"%s: global val '%s' initializer must be an i32 constant expression",
						frontend.FormatPos(glob.Init.Pos()),
						glob.Name,
					)
				}
				if err := validateGlobalIntLikeRange(
					resolved,
					glob.Name,
					"val",
					glob.Init.Pos(),
					v,
				); err != nil {
					return err
				}
				binary.LittleEndian.PutUint64(buf, uint64(int64(v)))
				constValues[glob.Name] = globalConstValue{TypeName: resolved, I32: v}
			case "bool":
				if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
					return err
				}
				v, ok := evalGlobalConstBool(glob.Init, constValues)
				if !ok {
					return fmt.Errorf(
						"%s: global val '%s' initializer must be a bool constant expression",
						frontend.FormatPos(glob.Init.Pos()),
						glob.Name,
					)
				}
				if v {
					binary.LittleEndian.PutUint64(buf, 1)
				} else {
					binary.LittleEndian.PutUint64(buf, 0)
				}
				constValues[glob.Name] = globalConstValue{TypeName: "bool", Bool: v}
			case "str":
				binary.LittleEndian.PutUint64(slotData[0], 0)
				binary.LittleEndian.PutUint64(slotData[1], uint64(len(stringInit)))
			default:
				if isSupportedSliceGlobalType(resolved) {
					return fmt.Errorf(
						"%s: global val '%s' initializer for type %s must be omitted",
						frontend.FormatPos(glob.Init.Pos()),
						glob.Name,
						resolved,
					)
				}
				if isSupportedOptionalPtrGlobalType(resolved) {
					if _, ok := glob.Init.(*frontend.NoneLitExpr); ok {
						dataBlobs = append(dataBlobs, slotData...)
						continue
					}
					return fmt.Errorf(
						"%s: global val '%s' initializer for type %s must be none",
						frontend.FormatPos(glob.Init.Pos()),
						glob.Name,
						resolved,
					)
				}
				if isSupportedOptionalSliceGlobalType(resolved) {
					if _, ok := glob.Init.(*frontend.NoneLitExpr); ok {
						dataBlobs = append(dataBlobs, slotData...)
						continue
					}
					return fmt.Errorf(
						"%s: global val '%s' initializer for type %s must be none",
						frontend.FormatPos(glob.Init.Pos()),
						glob.Name,
						resolved,
					)
				}
				if isSupportedOptionalAggregateGlobalType(resolved, types) {
					if _, ok := glob.Init.(*frontend.NoneLitExpr); ok {
						dataBlobs = append(dataBlobs, slotData...)
						continue
					}
					return fmt.Errorf(
						"%s: global val '%s' initializer for type %s must be none",
						frontend.FormatPos(glob.Init.Pos()),
						glob.Name,
						resolved,
					)
				}
				if isSupportedZeroedAggregateGlobalType(resolved, types, map[string]bool{}) {
					return fmt.Errorf(
						"%s: global val '%s' initializer for type %s must be omitted",
						frontend.FormatPos(glob.Init.Pos()),
						glob.Name,
						resolved,
					)
				}
				return fmt.Errorf(
					"%s: unsupported global type '%s'",
					frontend.FormatPos(glob.At),
					resolved,
				)
			}
			dataBlobs = append(dataBlobs, slotData...)
		}

		checked.GlobalsByModule[module] = globals
		checked.GlobalDataByModule[module] = dataBlobs
	}
	return nil
}
