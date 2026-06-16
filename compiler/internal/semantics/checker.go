package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
)

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
				return nil, fmt.Errorf("%s: reserved type name '%s'", frontend.FormatPos(en.At), en.Name)
			}
			if _, exists := enums[fullName]; exists {
				return nil, fmt.Errorf("duplicate enum '%s'", fullName)
			}
			if _, exists := structs[fullName]; exists {
				return nil, fmt.Errorf("duplicate type '%s'", fullName)
			}
			enums[fullName] = enumContext{module: module, imports: imports, public: declarationIsPublic(file, en.Public), decl: en}
			checked.Enums = append(checked.Enums, CheckedEnum{Name: fullName, Module: module, Decl: en})
		}
		for _, st := range file.Structs {
			fullName := qualifyName(module, st.Name)
			if isReservedTypeName(st.Name) {
				return nil, fmt.Errorf("%s: reserved type name '%s'", frontend.FormatPos(st.At), st.Name)
			}
			if _, exists := structs[fullName]; exists {
				return nil, fmt.Errorf("duplicate struct '%s'", fullName)
			}
			if _, exists := enums[fullName]; exists {
				return nil, fmt.Errorf("duplicate type '%s'", fullName)
			}
			structs[fullName] = structContext{module: module, imports: imports, public: declarationIsPublic(file, st.Public), decl: st}
			checked.Structs = append(checked.Structs, CheckedStruct{Name: fullName, Module: module, Decl: st})
		}
		for _, st := range file.States {
			fullName := qualifyName(module, st.Name)
			if isReservedTypeName(st.Name) {
				return nil, fmt.Errorf("%s: reserved type name '%s'", frontend.FormatPos(st.At), st.Name)
			}
			if _, exists := structs[fullName]; exists {
				return nil, fmt.Errorf("duplicate state '%s'", fullName)
			}
			if _, exists := enums[fullName]; exists {
				return nil, fmt.Errorf("duplicate type '%s'", fullName)
			}
			synth := stateAsStructDecl(st)
			synth.Public = declarationIsPublic(file, st.Public)
			structs[fullName] = structContext{module: module, imports: imports, public: synth.Public, decl: synth}
			checked.UIStates = append(checked.UIStates, CheckedUIState{Name: fullName, Module: module, Decl: st})
		}
		for _, proto := range file.Protocols {
			fullName := qualifyName(module, proto.Name)
			if isReservedTypeName(proto.Name) {
				return nil, fmt.Errorf("%s: reserved type name '%s'", frontend.FormatPos(proto.At), proto.Name)
			}
			if _, exists := protocols[fullName]; exists {
				return nil, fmt.Errorf("duplicate protocol '%s'", fullName)
			}
			protocols[fullName] = protocolContext{module: module, imports: imports, public: declarationIsPublic(file, proto.Public), decl: proto}
			checked.Protocols = append(checked.Protocols, CheckedProtocol{Name: fullName, Module: module, Decl: proto})
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
				return nil, fmt.Errorf("%s: duplicate enum case '%s'", frontend.FormatPos(c.At), c.Name)
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
			return nil, fmt.Errorf("%s: recursive struct '%s'", frontend.FormatPos(ctx.decl.At), name)
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
				return nil, fmt.Errorf("%s: duplicate field '%s'", frontend.FormatPos(field.At), field.Name)
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
			if surfaceType, ok := surfaceEphemeralValueType(resolved, types); ok && !surfaceAggregateFieldStorageAllowed(name, field.Name, resolved) {
				return nil, lifetimeDiagnosticf(field.At, "surface value '%s' cannot be stored in struct field '%s'; keep Surface Frame/Event/DrawContext values local to the active Surface turn", surfaceType, field.Name)
			}
			functionParamTypes := []string(nil)
			functionParamOwnership := []string(nil)
			functionReturnType := ""
			functionReturnOwnership := ""
			functionThrowsType := ""
			functionEffects := []string(nil)
			if field.Type.Kind == frontend.TypeRefFunction {
				functionParamTypes, functionReturnType, functionEffects, err = functionTypeRefSignatureAndEffects(field.Type, ctx.module, ctx.imports)
				if err != nil {
					return nil, err
				}
				functionParamOwnership = functionTypeRefParamOwnership(field.Type)
				functionReturnOwnership = functionTypeRefReturnOwnership(field.Type)
				functionThrowsType, err = functionTypeRefThrowsType(field.Type, ctx.module, ctx.imports)
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
				return nil, fmt.Errorf("%s: actor '%s' state field '%s': %v", frontend.FormatPos(field.At), displayTypeName(actorName, ctx.module), field.Name, err)
			}
			if err := ensureTypeVisible(resolved, fieldType, ctx.module, field.At); err != nil {
				return nil, err
			}
			if !isSupportedActorStateScalarType(resolved) {
				return nil, fmt.Errorf("%s: actor state field '%s' type '%s' is not supported; supported actor state field types are Int, Bool, UInt8, UInt16, and task.error", frontend.FormatPos(field.At), field.Name, resolved)
			}
			fieldSlots := fieldType.SlotCount
			if fieldSlots <= 0 {
				fieldSlots = 1
			}
			if slot+fieldSlots > MaxActorStateSlots {
				return nil, fmt.Errorf("%s: actor '%s' state supports at most %d slots, got %d", frontend.FormatPos(field.At), displayTypeName(actorName, ctx.module), MaxActorStateSlots, slot+fieldSlots)
			}
			if field.Init == nil {
				return nil, fmt.Errorf("%s: actor state field '%s' requires a compile-time constant initializer", frontend.FormatPos(field.At), field.Name)
			}
			initType, initValue, ok := evaluateActorStateInitializer(field.Init)
			if !ok {
				return nil, fmt.Errorf("%s: actor state field '%s' initializer must be a compile-time constant Int/Bool expression", frontend.FormatPos(field.At), field.Name)
			}
			if !typesCompatible(resolved, initType) {
				return nil, fmt.Errorf("%s: actor state field '%s' type mismatch: expected '%s', got '%s'", frontend.FormatPos(field.At), field.Name, resolved, initType)
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
					functionParamTypes, functionReturnType, functionEffects, err = functionTypeRefSignatureAndEffects(*payload, ctx.module, ctx.imports)
					if err != nil {
						return nil, err
					}
					functionParamOwnership = functionTypeRefParamOwnership(*payload)
					functionReturnOwnership = functionTypeRefReturnOwnership(*payload)
					functionThrowsType, err = functionTypeRefThrowsType(*payload, ctx.module, ctx.imports)
					if err != nil {
						return nil, err
					}
				}
				resolved, err := resolveTypeName(payload, ctx.module, ctx.imports)
				if err != nil {
					return nil, err
				}
				if resolved == name {
					return nil, fmt.Errorf("%s: recursive enum payload '%s'", frontend.FormatPos(payload.At), displayTypeName(name, ctx.module))
				}
				payload.Name = resolved
				payloadInfo, err := buildType(resolved)
				if err != nil {
					return nil, fmt.Errorf("%s: enum '%s' case '%s': %v", frontend.FormatPos(payload.At), displayTypeName(name, ctx.module), declCase.Name, err)
				}
				if err := ensureTypeVisible(resolved, payloadInfo, ctx.module, payload.At); err != nil {
					return nil, err
				}
				if surfaceType, ok := surfaceEphemeralValueType(resolved, types); ok {
					return nil, lifetimeDiagnosticf(payload.At, "surface value '%s' cannot be stored in enum payload '%s'; keep Surface Frame/Event/DrawContext values local to the active Surface turn", surfaceType, declCase.Name)
				}
				if payloadInfo.Kind == TypeEnum {
					enumPayloadEdges[name][resolved] = payload.At
				}
				caseInfo.PayloadTypes = append(caseInfo.PayloadTypes, resolved)
				caseInfo.PayloadSlots = append(caseInfo.PayloadSlots, payloadInfo.SlotCount)
				caseInfo.PayloadFunctionTypes = append(caseInfo.PayloadFunctionTypes, functionTypeValue)
				caseInfo.PayloadFunctionRefs = append(caseInfo.PayloadFunctionRefs, *payload)
				caseInfo.PayloadFunctionParams = append(caseInfo.PayloadFunctionParams, functionParamTypes)
				caseInfo.PayloadFunctionOwns = append(caseInfo.PayloadFunctionOwns, functionParamOwnership)
				caseInfo.PayloadFunctionReturns = append(caseInfo.PayloadFunctionReturns, functionReturnType)
				caseInfo.PayloadFunctionReturnOwns = append(caseInfo.PayloadFunctionReturnOwns, functionReturnOwnership)
				caseInfo.PayloadFunctionThrows = append(caseInfo.PayloadFunctionThrows, functionThrowsType)
				caseInfo.PayloadFunctionEffects = append(caseInfo.PayloadFunctionEffects, functionEffects)
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
				return nil, fmt.Errorf("%s: duplicate protocol requirement '%s' (first at %s)", frontend.FormatPos(req.At), req.Name, frontend.FormatPos(first))
			}
			seenReqs[req.Name] = req.At
			reqEffects, err := normalizeEffects(req.Uses, req.At)
			if err != nil {
				return nil, fmt.Errorf("%s: protocol '%s' requirement '%s': %v", frontend.FormatPos(req.At), name, req.Name, err)
			}
			req.Uses = reqEffects
			reqTypeParams := make(map[string]struct{}, len(req.TypeParams))
			for _, tp := range req.TypeParams {
				reqTypeParams[tp] = struct{}{}
			}
			retName, retIsGeneric, err := resolveProtocolRequirementTypeRef(&req.ReturnType, ctx.module, ctx.imports, reqTypeParams)
			if err != nil {
				return nil, fmt.Errorf("%s: protocol '%s' requirement '%s': %v", frontend.FormatPos(req.ReturnType.At), name, req.Name, err)
			}
			req.ReturnType.Name = retName
			if !retIsGeneric {
				retInfo, err := buildType(retName)
				if err != nil {
					return nil, fmt.Errorf("%s: protocol '%s' requirement '%s': %v", frontend.FormatPos(req.At), name, req.Name, err)
				}
				if err := ensureTypeVisible(retName, retInfo, ctx.module, req.At); err != nil {
					return nil, err
				}
			}
			if req.HasThrows {
				throwName, throwIsGeneric, err := resolveProtocolRequirementTypeRef(&req.Throws, ctx.module, ctx.imports, reqTypeParams)
				if err != nil {
					return nil, fmt.Errorf("%s: protocol '%s' requirement '%s': %v", frontend.FormatPos(req.Throws.At), name, req.Name, err)
				}
				req.Throws.Name = throwName
				if !throwIsGeneric {
					throwInfo, err := buildType(throwName)
					if err != nil {
						return nil, fmt.Errorf("%s: protocol '%s' requirement '%s': %v", frontend.FormatPos(req.At), name, req.Name, err)
					}
					if err := ensureTypeVisible(throwName, throwInfo, ctx.module, req.Throws.At); err != nil {
						return nil, err
					}
				}
			}
			for j := range req.Params {
				param := &req.Params[j]
				resolved, paramIsGeneric, err := resolveProtocolRequirementTypeRef(&param.Type, ctx.module, ctx.imports, reqTypeParams)
				if err != nil {
					return nil, fmt.Errorf("%s: protocol '%s' requirement '%s': %v", frontend.FormatPos(param.At), name, req.Name, err)
				}
				param.Type.Name = resolved
				if !paramIsGeneric {
					paramInfo, err := buildType(resolved)
					if err != nil {
						return nil, fmt.Errorf("%s: protocol '%s' requirement '%s': %v", frontend.FormatPos(param.At), name, req.Name, err)
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
					return nil, fmt.Errorf("%s: @export name must not use the 'core.' namespace", frontend.FormatPos(fn.Pos))
				}
				if strings.HasPrefix(fn.ExportName, "__tetra_") && !strings.HasPrefix(module, "__") {
					return nil, fmt.Errorf("%s: @export name '%s' is reserved for internal runtime modules", frontend.FormatPos(fn.Pos), fn.ExportName)
				}
				if other, exists := exportedSymbols[fn.ExportName]; exists {
					return nil, fmt.Errorf("%s: duplicate @export name '%s' (already used by '%s')", frontend.FormatPos(fn.Pos), fn.ExportName, other)
				}
				exportedSymbols[fn.ExportName] = fullName
			}
			if _, exists := builtinSigs[fullName]; exists {
				return nil, fmt.Errorf("%s: cannot redefine builtin '%s'", frontend.FormatPos(fn.Pos), fullName)
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
				if err := validateGenericFuncDecl(fn, module, imports, collectGenericProtocolInfos(world), types); err != nil {
					return nil, err
				}
				if fn.ExportName != "" {
					return nil, fmt.Errorf("%s: generic function '%s' cannot be exported; export a concrete monomorphic wrapper", frontend.FormatPos(fn.Pos), fn.Name)
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
				if err := validateExportedOpaqueABISignature(module, fn, genericParamTypes, returnType, types); err != nil {
					return nil, err
				}
				if err := validateFunctionPolicyClauses(fn, effects, genericParamTypes, returnType, throwsType, types); err != nil {
					return nil, err
				}
				if err := validateExportedConsentTokenABISignature(module, fn, genericParamTypes, returnType, types); err != nil {
					return nil, err
				}
				policy, err := parseFunctionClausePolicy(fn)
				if err != nil {
					return nil, err
				}
				checked.FuncSigs[fullName] = FuncSig{
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
				}
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
				returnFunctionParams, returnFunctionReturn, returnFunctionEffects, err = functionTypeRefSignatureAndEffects(fn.ReturnType, module, imports)
				if err != nil {
					return nil, err
				}
				returnFunctionParamOwnership = functionTypeRefParamOwnership(fn.ReturnType)
				returnFunctionReturnOwnership = functionTypeRefReturnOwnership(fn.ReturnType)
				returnFunctionThrows, err = functionTypeRefThrowsType(fn.ReturnType, module, imports)
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
			if err := validateExportedOpaqueABISignature(module, fn, paramTypeByName, retName, types); err != nil {
				return nil, err
			}
			if err := validateFunctionPolicyClauses(fn, effects, paramTypeByName, retName, throwsType, types); err != nil {
				return nil, err
			}
			policy, err := parseFunctionClausePolicy(fn)
			if err != nil {
				return nil, err
			}
			checked.FuncSigs[fullName] = FuncSig{
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
			}
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
				return nil, fmt.Errorf("%s: impl target type '%s' is not defined", frontend.FormatPos(impl.Type.At), typeName)
			}
			typeInfo := types[typeName]
			if err := ensureTypeVisible(typeName, typeInfo, module, impl.Type.At); err != nil {
				return nil, err
			}
			implKey := typeName + "->" + protoName
			if first, exists := seenImpls[implKey]; exists {
				return nil, fmt.Errorf("%s: duplicate impl conformance '%s: %s' (first at %s)", frontend.FormatPos(impl.At), typeName, protoName, frontend.FormatPos(first))
			}
			seenImpls[implKey] = impl.At
			proto, ok := protocols[protoName]
			if !ok {
				if _, isType := types[protoName]; isType {
					return nil, fmt.Errorf("%s: impl protocol '%s' must name a protocol, got non-protocol type '%s'", frontend.FormatPos(impl.Protocol.At), displayTypeName(protoName, module), displayTypeName(protoName, module))
				}
				return nil, fmt.Errorf("%s: protocol '%s' is not defined", frontend.FormatPos(impl.Protocol.At), protoName)
			}
			if !symbolBelongsToModule(protoName, module) && !proto.public {
				return nil, fmt.Errorf("%s: private protocol '%s' is not visible from module '%s'", frontend.FormatPos(impl.Protocol.At), protoName, module)
			}
			for _, req := range proto.decl.Requirements {
				methodName := typeName + "." + req.Name
				method, methodModule, methodImports, err := findFuncDecl(world, methodName)
				if err != nil {
					return nil, err
				}
				if method == nil {
					return nil, fmt.Errorf("%s: type '%s' is missing protocol requirement '%s'", frontend.FormatPos(impl.At), typeName, req.Name)
				}
				if err := compareProtocolRequirement(typeName, protoName, req, method, methodModule, methodImports); err != nil {
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
					updated, err := applyInterfaceFunctionReturnMetadata(&sig, fn, globals, checked.FuncSigs, types, module, imports)
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
						return nil, fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(param.At), param.Name)
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
						functionParamTypes, functionReturnType, functionEffects, err = functionTypeRefSignatureAndEffects(param.Type, module, imports)
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
						Mutable:                 param.Ownership == "inout",
						FunctionTypeValue:       functionTypeValue,
						FunctionParamName:       functionParamNameForParam(param.Name, functionTypeValue),
						FunctionParamTypes:      functionParamTypes,
						FunctionParamOwnership:  functionParamOwnership,
						FunctionReturnType:      functionReturnType,
						FunctionReturnOwnership: functionReturnOwnership,
						FunctionThrowsType:      functionThrowsType,
						FunctionEffects:         functionEffects,
						FunctionFields:          functionFieldsForStructParameter(param.Name, paramTypeName, types),
						EnumPayloadFunctions:    enumPayloadFunctionsForEnumParameter(param.Name, paramTypeName, types),
						EnumPayloadFields:       enumPayloadFieldsForStructParameter(param.Name, paramTypeName, types),
					}
					scopeInfo.localScopes[param.Name] = regionNone
					slotIndex += info.SlotCount
				}
				if fields, ok := actorStateFieldsByMethod[fullName]; ok {
					if err := injectActorStateLocals(fields, locals, scopeInfo); err != nil {
						return nil, err
					}
				}
				if err := collectLocals(fn.Body, locals, &slotIndex, checked.FuncSigs, types, module, imports, scopeInfo, globals); err != nil {
					return nil, err
				}
				if !stmtListEndsWithReturnTyped(fn.Body, locals, globals, checked.FuncSigs, types, module, imports) {
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
				effects := newEffectContext(fullName, sig.Effects, fn.Uses, strings.HasPrefix(module, "__"))
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
				analysis := newFunctionAnalysisState(fn, policy, fullName, functionReturnSecretTaint, functionParamSecretTaint, types)
				if err := checkStmts(fn.Body, locals, globals, checked.FuncSigs, types, module, imports, sig.ReturnType, borrowedParams, inoutParams, state, effects, analysis); err != nil {
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
						return nil, fmt.Errorf("%s: return region does not match parameter", frontend.FormatPos(fn.Pos))
					}
					newReturnParam = idx
				}
				if sig.ReturnRegionParam != newReturnParam || !returnRegionSummariesEqual(sig.ReturnRegionSummary, newReturnRegionSummary) {
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
					newReturnResourceSummary = cloneReturnResourceSummary(state.returnResourceSummary)
				} else if typeContainsResourceHandle(sig.ReturnType, types) && state.returnResourceUnknown {
					newReturnResourceParam = regionUnknown
				}
				if sig.ReturnResourceParam != newReturnResourceParam || sig.ReturnResourcePath != newReturnResourcePath || !returnResourceSummariesEqual(sig.ReturnResourceSummary, newReturnResourceSummary) {
					sig.ReturnResourceParam = newReturnResourceParam
					sig.ReturnResourcePath = newReturnResourcePath
					sig.ReturnResourceSummary = newReturnResourceSummary
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				newThrowResourceSummary := ReturnResourceSummary(nil)
				if typeContainsResourceHandle(sig.ThrowsType, types) && len(state.throwResourceSummary) > 0 {
					newThrowResourceSummary = cloneReturnResourceSummary(state.throwResourceSummary)
				}
				if !returnResourceSummariesEqual(sig.ThrowResourceSummary, newThrowResourceSummary) {
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
				if !closureCapturesEqual(sig.ReturnFunctionCaptures, analysis.returnFunctionCaptures) {
					sig.ReturnFunctionCaptures = append([]frontend.ClosureCapture(nil), analysis.returnFunctionCaptures...)
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
				if !functionFieldMapsEqual(sig.ReturnFunctionFields, analysis.returnFunctionFields) {
					sig.ReturnFunctionFields = cloneFunctionFieldMap(analysis.returnFunctionFields)
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				if !functionFieldMapsEqual(sig.ReturnEnumPayloadFunctions, analysis.returnEnumPayloadFunctions) {
					sig.ReturnEnumPayloadFunctions = cloneFunctionFieldMap(analysis.returnEnumPayloadFunctions)
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				if !functionFieldMapsEqual(sig.ReturnEnumPayloadFields, analysis.returnEnumPayloadFields) {
					sig.ReturnEnumPayloadFields = cloneFunctionFieldMap(analysis.returnEnumPayloadFields)
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
		if typeContainsResourceHandle(sig.ReturnType, types) && sig.ReturnResourceParam == regionUnknown && len(sig.ReturnResourceSummary) == 0 {
			return nil, fmt.Errorf("resource return provenance could not be inferred for function '%s'", name)
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
			if err := validateExportedConsentTokenABISignature(module, fn, paramTypeByName, fn.ReturnType.Name, types); err != nil {
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
					return nil, fmt.Errorf("%s: main must be in entry module", frontend.FormatPos(fn.Pos))
				}
				if len(fn.Params) != 0 {
					return nil, fmt.Errorf("%s: main must not have parameters", frontend.FormatPos(fn.Pos))
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
					return nil, fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(param.At), param.Name)
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
					functionParamTypes, functionReturnType, functionEffects, err = functionTypeRefSignatureAndEffects(param.Type, module, imports)
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
					FunctionFields:          functionFieldsForStructParameter(param.Name, paramTypeName, types),
					EnumPayloadFunctions:    enumPayloadFunctionsForEnumParameter(param.Name, paramTypeName, types),
					EnumPayloadFields:       enumPayloadFieldsForStructParameter(param.Name, paramTypeName, types),
				}
				scopeInfo.localScopes[param.Name] = regionNone
				slotIndex += info.SlotCount
			}
			if fields, ok := actorStateFieldsByMethod[fullName]; ok {
				if err := injectActorStateLocals(fields, locals, scopeInfo); err != nil {
					return nil, err
				}
			}
			if err := collectLocals(fn.Body, locals, &slotIndex, checked.FuncSigs, types, module, imports, scopeInfo, globals); err != nil {
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
