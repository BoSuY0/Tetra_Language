package semantics

import (
	"encoding/binary"
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
)

func Check(prog *frontend.Program) (*CheckedProgram, error) {
	file := &frontend.FileAST{Module: "", Enums: prog.Enums, Structs: prog.Structs, Protocols: prog.Protocols, Extensions: prog.Extensions, Impls: prog.Impls, Funcs: prog.Funcs, Tests: prog.Tests}
	world := &module.World{
		EntryModule: "",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"": file},
	}
	return CheckWorldOpt(world, CheckOptions{RequireMain: true})
}

type CheckOptions struct {
	RequireMain bool
}

type globalConstValue struct {
	TypeName string
	I32      int32
	Bool     bool
}

func inferGlobalConstExprType(expr frontend.Expr, values map[string]globalConstValue) (string, bool) {
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
			return fmt.Errorf("%s: unknown constant '%s' in global const expression", frontend.FormatPos(e.At), e.Name)
		}
	case *frontend.UnaryExpr:
		return validateGlobalConstExpr(e.X, values)
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
	}
	return nil
}

func evalGlobalConstI32(expr frontend.Expr, values map[string]globalConstValue) (int32, bool) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return e.Value, true
	case *frontend.IdentExpr:
		v, ok := values[e.Name]
		if !ok || v.TypeName != "i32" {
			return 0, false
		}
		return v.I32, true
	case *frontend.UnaryExpr:
		if e.Op != frontend.TokenMinus {
			return 0, false
		}
		v, ok := evalGlobalConstI32(e.X, values)
		if !ok {
			return 0, false
		}
		return -v, true
	case *frontend.BinaryExpr:
		left, ok := evalGlobalConstI32(e.Left, values)
		if !ok {
			return 0, false
		}
		right, ok := evalGlobalConstI32(e.Right, values)
		if !ok {
			return 0, false
		}
		switch e.Op {
		case frontend.TokenPlus:
			return left + right, true
		case frontend.TokenMinus:
			return left - right, true
		case frontend.TokenStar:
			return left * right, true
		case frontend.TokenSlash:
			if right == 0 {
				return 0, false
			}
			return left / right, true
		case frontend.TokenPercent:
			if right == 0 {
				return 0, false
			}
			return left % right, true
		default:
			return 0, false
		}
	default:
		return 0, false
	}
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

	types := baseTypes()

	type structContext struct {
		module  string
		imports map[string]string
		decl    *frontend.StructDecl
	}
	type enumContext struct {
		module  string
		imports map[string]string
		decl    *frontend.EnumDecl
	}
	type protocolContext struct {
		module  string
		imports map[string]string
		decl    *frontend.ProtocolDecl
	}

	structs := make(map[string]structContext)
	enums := make(map[string]enumContext)
	protocols := make(map[string]protocolContext)
	checked := CheckedProgram{
		MainIndex:          -1,
		Types:              types,
		FuncSigs:           make(map[string]FuncSig),
		GlobalsByModule:    make(map[string]map[string]GlobalInfo),
		GlobalDataByModule: make(map[string][][]byte),
	}
	exportedSymbols := make(map[string]string)

	for _, file := range world.Files {
		module := file.Module
		imports, err := collectImportAliases(file)
		if err != nil {
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
			enums[fullName] = enumContext{module: module, imports: imports, decl: en}
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
			structs[fullName] = structContext{module: module, imports: imports, decl: st}
			checked.Structs = append(checked.Structs, CheckedStruct{Name: fullName, Module: module, Decl: st})
		}
		for _, proto := range file.Protocols {
			fullName := qualifyName(module, proto.Name)
			if isReservedTypeName(proto.Name) {
				return nil, fmt.Errorf("%s: reserved type name '%s'", frontend.FormatPos(proto.At), proto.Name)
			}
			if _, exists := protocols[fullName]; exists {
				return nil, fmt.Errorf("duplicate protocol '%s'", fullName)
			}
			protocols[fullName] = protocolContext{module: module, imports: imports, decl: proto}
			checked.Protocols = append(checked.Protocols, CheckedProtocol{Name: fullName, Module: module, Decl: proto})
		}
	}

	for name, ctx := range enums {
		caseMap := make(map[string]EnumCaseInfo, len(ctx.decl.Cases))
		cases := make([]EnumCaseInfo, 0, len(ctx.decl.Cases))
		for i, c := range ctx.decl.Cases {
			if _, exists := caseMap[c.Name]; exists {
				return nil, fmt.Errorf("%s: duplicate enum case '%s'", frontend.FormatPos(c.At), c.Name)
			}
			info := EnumCaseInfo{Name: c.Name, Ordinal: int32(i)}
			caseMap[c.Name] = info
			cases = append(cases, info)
		}
		types[name] = &TypeInfo{
			Name:      name,
			Kind:      TypeEnum,
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
			if elemInfo.SlotCount != 1 {
				return nil, fmt.Errorf("optional payload type '%s' is not supported yet (payload must be single-slot)", elem)
			}
			info := &TypeInfo{
				Name:      name,
				Kind:      TypeOptional,
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
			if elem != "i32" && elem != "u8" {
				return nil, fmt.Errorf("slice element type '%s' is not supported", elem)
			}
			info := makeSliceTypeInfo(name, elem)
			types[name] = info
			return info, nil
		}
		if isArrayTypeName(name) {
			return nil, fmt.Errorf("array types are not supported yet")
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
			info := FieldInfo{
				Name:      field.Name,
				TypeName:  resolved,
				Offset:    slotCount,
				SlotCount: fieldType.SlotCount,
			}
			fieldMap[field.Name] = info
			fields = append(fields, info)
			slotCount += fieldType.SlotCount
		}

		info := &TypeInfo{
			Name:      name,
			Kind:      TypeStruct,
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

	for name, ctx := range protocols {
		seenReqs := map[string]struct{}{}
		for i := range ctx.decl.Requirements {
			req := &ctx.decl.Requirements[i]
			if _, exists := seenReqs[req.Name]; exists {
				return nil, fmt.Errorf("%s: duplicate protocol requirement '%s'", frontend.FormatPos(req.At), req.Name)
			}
			seenReqs[req.Name] = struct{}{}
			retName, err := resolveTypeName(&req.ReturnType, ctx.module, ctx.imports)
			if err != nil {
				return nil, err
			}
			req.ReturnType.Name = retName
			if _, err := buildType(retName); err != nil {
				return nil, fmt.Errorf("%s: protocol '%s' requirement '%s': %v", frontend.FormatPos(req.At), name, req.Name, err)
			}
			if req.HasThrows {
				throwName, err := resolveTypeName(&req.Throws, ctx.module, ctx.imports)
				if err != nil {
					return nil, err
				}
				req.Throws.Name = throwName
				if _, err := buildType(throwName); err != nil {
					return nil, fmt.Errorf("%s: protocol '%s' requirement '%s': %v", frontend.FormatPos(req.At), name, req.Name, err)
				}
			}
			for j := range req.Params {
				param := &req.Params[j]
				resolved, err := resolveTypeName(&param.Type, ctx.module, ctx.imports)
				if err != nil {
					return nil, err
				}
				param.Type.Name = resolved
				if _, err := buildType(resolved); err != nil {
					return nil, fmt.Errorf("%s: protocol '%s' requirement '%s': %v", frontend.FormatPos(param.At), name, req.Name, err)
				}
			}
		}
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
			proto, ok := protocols[protoName]
			if !ok {
				return nil, fmt.Errorf("%s: protocol '%s' is not defined", frontend.FormatPos(impl.Protocol.At), protoName)
			}
			for _, req := range proto.decl.Requirements {
				methodName := typeName + "." + req.Name
				method := findFuncDecl(world, methodName)
				if method == nil {
					return nil, fmt.Errorf("%s: type '%s' is missing protocol requirement '%s'", frontend.FormatPos(impl.At), typeName, req.Name)
				}
				if err := compareProtocolRequirement(typeName, protoName, req, method); err != nil {
					return nil, err
				}
			}
		}
	}

	builtinSigs, err := builtinFuncSigs(types)
	if err != nil {
		return nil, err
	}

	for _, file := range world.Files {
		module := file.Module
		imports, err := collectImportAliases(file)
		if err != nil {
			return nil, err
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
				return nil, fmt.Errorf("%s: duplicate global '%s'", frontend.FormatPos(glob.At), glob.Name)
			}
			if _, exists := fnNames[glob.Name]; exists {
				return nil, fmt.Errorf("%s: global '%s' conflicts with function '%s'", frontend.FormatPos(glob.At), glob.Name, glob.Name)
			}

			resolved := ""
			if glob.Type.Name != "" || glob.Type.Elem != nil {
				var err error
				resolved, err = resolveTypeName(&glob.Type, module, imports)
				if err != nil {
					return nil, err
				}
			}
			if resolved == "" {
				if glob.Mutable {
					return nil, fmt.Errorf("%s: global var requires an explicit type annotation", frontend.FormatPos(glob.At))
				}
				if glob.Init == nil {
					return nil, fmt.Errorf("%s: global val requires an initializer to infer its type", frontend.FormatPos(glob.At))
				}
				if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
					return nil, err
				}
				inferred, ok := inferGlobalConstExprType(glob.Init, constValues)
				if !ok {
					return nil, fmt.Errorf("%s: unsupported global val initializer (type inference supports constant numeric/bool expressions)", frontend.FormatPos(glob.At))
				}
				resolved = inferred
			}
			glob.Type.Name = resolved
			if resolved != "i32" && resolved != "ptr" && resolved != "bool" {
				return nil, fmt.Errorf("%s: global '%s' has unsupported type '%s' (allowed: i32, bool, ptr)", frontend.FormatPos(glob.At), glob.Name, resolved)
			}
			if _, err := ensureTypeInfo(resolved, types); err != nil {
				return nil, fmt.Errorf("%s: %v", frontend.FormatPos(glob.At), err)
			}

			dataIndex := len(dataBlobs)
			globals[glob.Name] = GlobalInfo{DataIndex: dataIndex, TypeName: resolved, Mutable: glob.Mutable, Const: glob.Const}

			buf := make([]byte, 8)
			if glob.Mutable {
				dataBlobs = append(dataBlobs, buf)
				continue
			}
			if glob.Init == nil {
				return nil, fmt.Errorf("%s: global val '%s' requires an initializer", frontend.FormatPos(glob.At), glob.Name)
			}
			switch resolved {
			case "ptr":
				if !isNullPtrLiteral(glob.Init) {
					return nil, fmt.Errorf("%s: global val '%s' of type ptr only supports initializer 0", frontend.FormatPos(glob.Init.Pos()), glob.Name)
				}
				binary.LittleEndian.PutUint64(buf, 0)
			case "i32":
				if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
					return nil, err
				}
				v, ok := evalGlobalConstI32(glob.Init, constValues)
				if !ok {
					return nil, fmt.Errorf("%s: global val '%s' initializer must be an i32 constant expression", frontend.FormatPos(glob.Init.Pos()), glob.Name)
				}
				binary.LittleEndian.PutUint64(buf, uint64(int64(v)))
				constValues[glob.Name] = globalConstValue{TypeName: "i32", I32: v}
			case "bool":
				if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
					return nil, err
				}
				v, ok := evalGlobalConstBool(glob.Init, constValues)
				if !ok {
					return nil, fmt.Errorf("%s: global val '%s' initializer must be a bool constant expression", frontend.FormatPos(glob.Init.Pos()), glob.Name)
				}
				if v {
					binary.LittleEndian.PutUint64(buf, 1)
				} else {
					binary.LittleEndian.PutUint64(buf, 0)
				}
				constValues[glob.Name] = globalConstValue{TypeName: "bool", Bool: v}
			default:
				return nil, fmt.Errorf("%s: unsupported global type '%s'", frontend.FormatPos(glob.At), resolved)
			}
			dataBlobs = append(dataBlobs, buf)
		}

		checked.GlobalsByModule[module] = globals
		checked.GlobalDataByModule[module] = dataBlobs
	}

	for _, file := range world.Files {
		module := file.Module
		imports, err := collectImportAliases(file)
		if err != nil {
			return nil, err
		}
		for _, fn := range file.Funcs {
			fullName := qualifyName(module, fn.Name)
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
			if len(fn.TypeParams) > 0 {
				if err := validateGenericFuncDecl(fn); err != nil {
					return nil, err
				}
				effects, err := normalizeEffects(fn.Uses, fn.Pos)
				if err != nil {
					return nil, err
				}
				checked.FuncSigs[fullName] = FuncSig{
					Generic:           true,
					ParamNames:        genericParamNames(fn.Params),
					ParamTypes:        genericParamTypeNames(fn.Params),
					ParamOwnership:    genericParamOwnership(fn.Params),
					ParamSlots:        0,
					ReturnType:        fn.ReturnType.Name,
					ThrowsType:        fn.Throws.Name,
					Async:             fn.Async,
					ReturnSlots:       0,
					ReturnRegionParam: regionNone,
					Effects:           effects,
				}
				continue
			}
			retName, err := resolveTypeName(&fn.ReturnType, module, imports)
			if err != nil {
				return nil, err
			}
			fn.ReturnType.Name = retName
			retInfo, err := buildType(retName)
			if err != nil {
				return nil, err
			}
			if retInfo.SlotCount > 2 {
				return nil, fmt.Errorf("function '%s' return type too large", fullName)
			}
			throwsType := ""
			returnSlots := retInfo.SlotCount
			if fn.HasThrows {
				if retInfo.SlotCount != 1 {
					return nil, fmt.Errorf("%s: throwing function '%s' must return a single-slot success type in v0.5", frontend.FormatPos(fn.Pos), fullName)
				}
				resolvedThrows, err := resolveTypeName(&fn.Throws, module, imports)
				if err != nil {
					return nil, err
				}
				fn.Throws.Name = resolvedThrows
				throwInfo, err := buildType(resolvedThrows)
				if err != nil {
					return nil, err
				}
				if throwInfo.SlotCount != 1 {
					return nil, fmt.Errorf("%s: throwing function '%s' must use a single-slot error type in v0.5", frontend.FormatPos(fn.Pos), fullName)
				}
				throwsType = resolvedThrows
				returnSlots = 2
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
				paramNames = append(paramNames, param.Name)
				paramTypes = append(paramTypes, resolved)
				paramOwnership = append(paramOwnership, param.Ownership)
				paramSlots += info.SlotCount
			}
			checked.FuncSigs[fullName] = FuncSig{
				ParamNames:        paramNames,
				ParamTypes:        paramTypes,
				ParamOwnership:    paramOwnership,
				ParamSlots:        paramSlots,
				ReturnType:        retName,
				ThrowsType:        throwsType,
				Async:             fn.Async,
				ReturnSlots:       returnSlots,
				ReturnRegionParam: regionNone,
				Effects:           effects,
			}
		}
	}

	for name, sig := range builtinSigs {
		checked.FuncSigs[name] = sig
	}

	if len(checked.FuncSigs) == 0 {
		return nil, fmt.Errorf("no functions found")
	}

	funcCount := 0
	for _, file := range world.Files {
		funcCount += len(file.Funcs)
	}
	maxIter := funcCount + 1
	for iter := 0; iter < maxIter; iter++ {
		changed := false
		for _, file := range world.Files {
			module := file.Module
			imports, err := collectImportAliases(file)
			if err != nil {
				return nil, err
			}
			globals := checked.GlobalsByModule[module]
			for _, fn := range file.Funcs {
				fullName := qualifyName(module, fn.Name)
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
					info, err := buildType(param.Type.Name)
					if err != nil {
						return nil, err
					}
					locals[param.Name] = LocalInfo{
						Base:      slotIndex,
						SlotCount: info.SlotCount,
						TypeName:  param.Type.Name,
						Mutable:   param.Ownership == "inout",
					}
					scopeInfo.localScopes[param.Name] = regionNone
					slotIndex += info.SlotCount
				}
				if err := collectLocals(fn.Body, locals, &slotIndex, checked.FuncSigs, types, module, imports, scopeInfo, globals); err != nil {
					return nil, err
				}
				if !stmtListEndsWithReturnTyped(fn.Body, locals, globals, checked.FuncSigs, types, module, imports) {
					return nil, fmt.Errorf("function '%s' must end with return", fullName)
				}
				state := newRegionState(scopeInfo)
				initParamRegions(fn.Params, state, types)
				sig := checked.FuncSigs[fullName]
				state.throwType = sig.ThrowsType
				state.async = sig.Async
				effects := newEffectContext(fullName, sig.Effects, strings.HasPrefix(module, "__"))
				borrowedParams := make(map[string]struct{})
				for _, param := range fn.Params {
					if param.Ownership == "borrow" {
						borrowedParams[param.Name] = struct{}{}
					}
				}
				if err := checkStmts(fn.Body, locals, globals, checked.FuncSigs, types, module, imports, sig.ReturnType, borrowedParams, state, effects); err != nil {
					return nil, err
				}
				newReturnParam := regionNone
				if state.returnRegionSet && state.returnRegion < regionNone {
					idx, ok := state.paramRegionIndex[state.returnRegion]
					if !ok {
						return nil, fmt.Errorf("%s: return region does not match parameter", frontend.FormatPos(fn.Pos))
					}
					newReturnParam = idx
				}
				if sig.ReturnRegionParam != newReturnParam {
					sig.ReturnRegionParam = newReturnParam
					checked.FuncSigs[fullName] = sig
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

	for _, file := range world.Files {
		module := file.Module
		imports, err := collectImportAliases(file)
		if err != nil {
			return nil, err
		}
		globals := checked.GlobalsByModule[module]
		for _, fn := range file.Funcs {
			fullName := qualifyName(module, fn.Name)
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
				info, err := buildType(param.Type.Name)
				if err != nil {
					return nil, err
				}
				locals[param.Name] = LocalInfo{
					Base:      slotIndex,
					SlotCount: info.SlotCount,
					TypeName:  param.Type.Name,
					Mutable:   false,
				}
				scopeInfo.localScopes[param.Name] = regionNone
				slotIndex += info.SlotCount
			}
			if err := collectLocals(fn.Body, locals, &slotIndex, checked.FuncSigs, types, module, imports, scopeInfo, globals); err != nil {
				return nil, err
			}
			checked.Funcs = append(checked.Funcs, CheckedFunc{
				Name:        fullName,
				Module:      module,
				Decl:        fn,
				Locals:      locals,
				LocalSlots:  slotIndex,
				ParamSlots:  checked.FuncSigs[fullName].ParamSlots,
				ReturnType:  checked.FuncSigs[fullName].ReturnType,
				ThrowsType:  checked.FuncSigs[fullName].ThrowsType,
				Async:       checked.FuncSigs[fullName].Async,
				ReturnSlots: checked.FuncSigs[fullName].ReturnSlots,
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

func stmtListEndsWithReturn(stmts []frontend.Stmt) bool {
	if len(stmts) == 0 {
		return false
	}
	return stmtEndsWithReturn(stmts[len(stmts)-1])
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
	return stmtEndsWithReturnTyped(stmts[len(stmts)-1], locals, globals, funcs, types, module, imports)
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
		if c.Default {
			return true
		}
	}
	return false
}

func matchHasCompleteOptionalPatterns(s *frontend.MatchStmt) bool {
	hasNone := false
	hasSome := false
	for _, c := range s.Cases {
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
	for _, c := range s.Cases {
		if c.Default {
			return true
		}
		patType, err := inferExprTypeForDecl(c.Pattern, locals, globals, funcs, types, module, imports)
		if err != nil || patType != scrutType {
			return false
		}
		field, ok := c.Pattern.(*frontend.FieldAccessExpr)
		if !ok || field.EnumType != scrutType {
			return false
		}
		seen[field.Field] = struct{}{}
	}
	for _, enumCase := range info.EnumCases {
		if _, ok := seen[enumCase.Name]; !ok {
			return false
		}
	}
	return true
}

func stmtEndsWithReturn(stmt frontend.Stmt) bool {
	switch s := stmt.(type) {
	case *frontend.ReturnStmt:
		return true
	case *frontend.ThrowStmt:
		return true
	case *frontend.IfStmt:
		return len(s.Then) > 0 && len(s.Else) > 0 && stmtListEndsWithReturn(s.Then) && stmtListEndsWithReturn(s.Else)
	case *frontend.IfLetStmt:
		return len(s.Then) > 0 && len(s.Else) > 0 && stmtListEndsWithReturn(s.Then) && stmtListEndsWithReturn(s.Else)
	case *frontend.MatchStmt:
		hasDefault := false
		hasNone := false
		hasSome := false
		for _, c := range s.Cases {
			if c.Default {
				hasDefault = true
			} else if _, ok := c.Pattern.(*frontend.NoneLitExpr); ok {
				hasNone = true
			} else if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
				hasSome = true
			}
			if !stmtListEndsWithReturn(c.Body) {
				return false
			}
		}
		return hasDefault || (hasNone && hasSome)
	case *frontend.UnsafeStmt:
		return stmtListEndsWithReturn(s.Body)
	default:
		return false
	}
}

func validateGenericFuncDecl(fn *frontend.FuncDecl) error {
	if len(fn.TypeParams) == 0 {
		return nil
	}
	params := map[string]struct{}{}
	for _, name := range fn.TypeParams {
		params[name] = struct{}{}
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
	return nil
}

func validateSemanticClauses(fn *frontend.FuncDecl) error {
	seen := map[string]frontend.Position{}
	for _, clause := range fn.SemanticClauses {
		if first, exists := seen[clause.Name]; exists {
			return fmt.Errorf("%s: duplicate semantic clause '%s' (first at %s)", frontend.FormatPos(clause.At), clause.Name, frontend.FormatPos(first))
		}
		seen[clause.Name] = clause.At
		switch clause.Name {
		case "noalloc", "noblock", "realtime":
			if clause.Value != nil {
				return fmt.Errorf("%s: semantic clause '%s' does not take arguments", frontend.FormatPos(clause.At), clause.Name)
			}
		case "nothrow":
			if clause.Value != nil {
				return fmt.Errorf("%s: semantic clause 'nothrow' does not take arguments", frontend.FormatPos(clause.At))
			}
			if fn.HasThrows {
				return fmt.Errorf("%s: semantic clause 'nothrow' conflicts with explicit throws type", frontend.FormatPos(clause.At))
			}
		case "budget":
			if clause.Value == nil {
				return fmt.Errorf("%s: semantic clause 'budget' requires an integer argument", frontend.FormatPos(clause.At))
			}
			v, ok := constI32(clause.Value)
			if !ok {
				return fmt.Errorf("%s: semantic clause 'budget' expects an integer constant argument", frontend.FormatPos(clause.Value.Pos()))
			}
			if v < 0 {
				return fmt.Errorf("%s: semantic clause 'budget' requires a non-negative value", frontend.FormatPos(clause.Value.Pos()))
			}
		default:
			return fmt.Errorf("%s: unknown semantic clause '%s'", frontend.FormatPos(clause.At), clause.Name)
		}
	}
	return nil
}

func findFuncDecl(world *module.World, name string) *frontend.FuncDecl {
	for _, file := range world.Files {
		for _, fn := range file.Funcs {
			if qualifyName(file.Module, fn.Name) == name || fn.Name == name {
				return fn
			}
		}
	}
	return nil
}

func compareProtocolRequirement(typeName, protoName string, req frontend.FuncSigDecl, method *frontend.FuncDecl) error {
	if len(req.Params) != len(method.Params) {
		return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': parameter count differs", frontend.FormatPos(method.Pos), method.Name, protoName, req.Name)
	}
	for i := range req.Params {
		if genericTypeName(req.Params[i].Type) != genericTypeName(method.Params[i].Type) {
			return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': parameter %d type differs", frontend.FormatPos(method.Params[i].At), method.Name, protoName, req.Name, i+1)
		}
	}
	if genericTypeName(req.ReturnType) != genericTypeName(method.ReturnType) {
		return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': return type differs", frontend.FormatPos(method.Pos), method.Name, protoName, req.Name)
	}
	if req.Async != method.Async {
		return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': async marker differs", frontend.FormatPos(method.Pos), method.Name, protoName, req.Name)
	}
	if req.HasThrows != method.HasThrows || genericTypeName(req.Throws) != genericTypeName(method.Throws) {
		return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': throws type differs", frontend.FormatPos(method.Pos), method.Name, protoName, req.Name)
	}
	_ = typeName
	return nil
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
			return fmt.Errorf("missing element type")
		}
		return validateGenericTypeRef(*ref.Elem, params)
	default:
		return fmt.Errorf("unsupported type")
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
	default:
		return ref.Name
	}
}

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
	state *regionState,
	effects *effectContext,
) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			if err := effects.require(s.At, "io"); err != nil {
				return err
			}
			tname, _, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects)
			if err != nil {
				return err
			}
			if !isPrintableType(tname, types) {
				return fmt.Errorf("%s: print expects str or []u8", frontend.FormatPos(s.At))
			}
		case *frontend.BreakStmt:
			if state.loopDepth == 0 {
				return fmt.Errorf("%s: break outside loop", frontend.FormatPos(s.At))
			}
		case *frontend.ContinueStmt:
			if state.loopDepth == 0 {
				return fmt.Errorf("%s: continue outside loop", frontend.FormatPos(s.At))
			}
		case *frontend.FreeStmt:
			if err := effects.requireAll(s.At, []string{"islands", "mem"}); err != nil {
				return err
			}
			tname, _, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects)
			if err != nil {
				return err
			}
			if tname != "island" {
				return fmt.Errorf("%s: free expects island, got '%s'", frontend.FormatPos(s.At), tname)
			}
			if !s.Implicit && !state.inUnsafe() {
				return fmt.Errorf("%s: free is only allowed in unsafe blocks", frontend.FormatPos(s.At))
			}
		case *frontend.ReturnStmt:
			tname, regionID, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects)
			if err != nil {
				return err
			}
			if typeMayContainRegion(tname, types) {
				if borrowedName, borrowed := state.borrowedParamOwner(regionID); borrowed {
					return fmt.Errorf("%s: borrowed local '%s' cannot escape via return", frontend.FormatPos(s.At), borrowedName)
				}
				if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if _, borrowed := borrowedParams[id.Name]; borrowed {
						return fmt.Errorf("%s: borrowed local '%s' cannot escape via return", frontend.FormatPos(s.At), id.Name)
					}
				}
			}
			if err := state.recordReturnRegion(regionID, s.At); err != nil {
				return err
			}
			if !typesCompatibleWithNullPtr(returnType, tname, s.Value) {
				return fmt.Errorf("%s: return type mismatch: expected '%s', got '%s'", frontend.FormatPos(s.At), returnType, tname)
			}
		case *frontend.ThrowStmt:
			if state.throwType == "" {
				return fmt.Errorf("%s: throw is only allowed in throwing functions", frontend.FormatPos(s.At))
			}
			tname, _, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects)
			if err != nil {
				return err
			}
			if !typesCompatibleWithNullPtr(state.throwType, tname, s.Value) {
				return fmt.Errorf("%s: throw type mismatch: expected '%s', got '%s'", frontend.FormatPos(s.At), state.throwType, tname)
			}
		case *frontend.IslandStmt:
			if err := effects.requireAll(s.At, []string{"alloc", "islands", "mem"}); err != nil {
				return err
			}
			sizeType, _, err := checkExprWithEffects(s.Size, locals, globals, funcs, types, module, imports, state, effects)
			if err != nil {
				return err
			}
			if !isInt32Like(sizeType) {
				return fmt.Errorf("%s: island size must be i32/u8", frontend.FormatPos(s.At))
			}
			if err := state.enterIsland(s.Name); err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			if err := checkStmts(s.Body, locals, globals, funcs, types, module, imports, returnType, borrowedParams, state, effects); err != nil {
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
			valType, valRegion, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects)
			if err != nil {
				return err
			}
			if !typesCompatibleWithNullPtr(resolved, valType, s.Value) {
				return fmt.Errorf("%s: type mismatch: expected '%s', got '%s'", frontend.FormatPos(s.At), resolved, valType)
			}
			if valRegion >= 0 {
				scopeID := localScopeID(s.Name, state)
				if !state.isScopeWithin(scopeID, valRegion) {
					return fmt.Errorf(
						"%s: slice from scoped island cannot escape to outer scope (value: %s, target: %s)",
						frontend.FormatPos(s.At),
						formatRegionID(state, valRegion),
						formatScopeID(state, scopeID),
					)
				}
				state.regionVars[s.Name] = valRegion
				delete(state.unknownVars, s.Name)
				delete(state.unknownConflicts, s.Name)
			} else if valRegion < regionNone {
				state.regionVars[s.Name] = valRegion
				delete(state.unknownVars, s.Name)
				delete(state.unknownConflicts, s.Name)
			} else {
				delete(state.regionVars, s.Name)
				delete(state.unknownVars, s.Name)
				delete(state.unknownConflicts, s.Name)
			}
		case *frontend.AssignStmt:
			if idx, ok := s.Target.(*frontend.IndexExpr); ok {
				indexType, _, err := checkExprWithEffects(idx.Index, locals, globals, funcs, types, module, imports, state, effects)
				if err != nil {
					return err
				}
				if !isInt32Like(indexType) {
					return fmt.Errorf("%s: index must be i32/u8", frontend.FormatPos(idx.At))
				}
				if _, _, err := checkExprWithEffects(idx.Base, locals, globals, funcs, types, module, imports, state, effects); err != nil {
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
					valType, _, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects)
					if err != nil {
						return err
					}
					if !typesCompatibleWithNullPtr(g.TypeName, valType, s.Value) {
						return fmt.Errorf("%s: type mismatch: expected '%s', got '%s'", frontend.FormatPos(s.At), g.TypeName, valType)
					}
					continue
				}
			}
			targetInfo, targetType, err := resolveAssignTarget(s.Target, locals, types)
			if err != nil {
				return err
			}
			if err := checkLocalScope(targetInfo.Name, state, s.At); err != nil {
				return err
			}
			if !targetInfo.Mutable {
				if targetInfo.Const {
					return fmt.Errorf("%s: cannot assign to const '%s'", frontend.FormatPos(s.At), targetInfo.Name)
				}
				return fmt.Errorf("%s: cannot assign to val '%s'", frontend.FormatPos(s.At), targetInfo.Name)
			}
			valType, valRegion, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects)
			if err != nil {
				return err
			}
			if !typesCompatibleWithNullPtr(targetType, valType, s.Value) {
				return fmt.Errorf("%s: type mismatch: expected '%s', got '%s'", frontend.FormatPos(s.At), targetType, valType)
			}
			if _, ok := s.Target.(*frontend.IndexExpr); !ok {
				if valRegion >= 0 {
					scopeID := localScopeID(targetInfo.Name, state)
					if !state.isScopeWithin(scopeID, valRegion) {
						return fmt.Errorf(
							"%s: slice from scoped island cannot escape to outer scope (value: %s, target: %s)",
							frontend.FormatPos(s.At),
							formatRegionID(state, valRegion),
							formatScopeID(state, scopeID),
						)
					}
					state.regionVars[targetInfo.Name] = valRegion
					delete(state.unknownVars, targetInfo.Name)
					delete(state.unknownConflicts, targetInfo.Name)
				} else if valRegion < regionNone {
					state.regionVars[targetInfo.Name] = valRegion
					delete(state.unknownVars, targetInfo.Name)
					delete(state.unknownConflicts, targetInfo.Name)
				} else {
					delete(state.regionVars, targetInfo.Name)
					delete(state.unknownVars, targetInfo.Name)
					delete(state.unknownConflicts, targetInfo.Name)
				}
			}
		case *frontend.IfStmt:
			condType, _, err := checkExprWithEffects(s.Cond, locals, globals, funcs, types, module, imports, state, effects)
			if err != nil {
				return err
			}
			if !isConditionType(condType) {
				return fmt.Errorf("%s: condition must be bool or i32/u8", frontend.FormatPos(s.At))
			}
			scopeIDs := branchScopeInfo{thenID: regionNone, elseID: regionNone}
			if scoped, ok := state.ifScopes[s]; ok {
				scopeIDs = scoped
			}
			before := copyRegionVars(state.regionVars)
			state.regionVars = copyRegionVars(before)
			if err := withActiveScope(state, scopeIDs.thenID, func() error {
				return checkStmts(s.Then, locals, globals, funcs, types, module, imports, returnType, borrowedParams, state, effects)
			}); err != nil {
				return err
			}
			thenVars := copyRegionVars(state.regionVars)
			var elseVars map[string]int
			if len(s.Else) > 0 {
				state.regionVars = copyRegionVars(before)
				if err := withActiveScope(state, scopeIDs.elseID, func() error {
					return checkStmts(s.Else, locals, globals, funcs, types, module, imports, returnType, borrowedParams, state, effects)
				}); err != nil {
					return err
				}
				elseVars = copyRegionVars(state.regionVars)
			} else {
				elseVars = before
			}
			state.regionVars = mergeRegionVars(thenVars, elseVars)
			recordMergeConflicts(state, thenVars, elseVars, "then", "else")
			markUnknownRegions(state)
		case *frontend.IfLetStmt:
			valueType, _, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects)
			if err != nil {
				return err
			}
			if _, ok := optionalElemName(valueType); !ok {
				return fmt.Errorf("%s: if let requires optional value, got '%s'", frontend.FormatPos(s.At), valueType)
			}
			scopeIDs := branchScopeInfo{thenID: regionNone, elseID: regionNone}
			if scoped, ok := state.ifLetScopes[s]; ok {
				scopeIDs = scoped
			}
			before := copyRegionVars(state.regionVars)
			state.regionVars = copyRegionVars(before)
			if err := withActiveScope(state, scopeIDs.thenID, func() error {
				return checkStmts(s.Then, locals, globals, funcs, types, module, imports, returnType, borrowedParams, state, effects)
			}); err != nil {
				return err
			}
			thenVars := copyRegionVars(state.regionVars)
			var elseVars map[string]int
			if len(s.Else) > 0 {
				state.regionVars = copyRegionVars(before)
				if err := withActiveScope(state, scopeIDs.elseID, func() error {
					return checkStmts(s.Else, locals, globals, funcs, types, module, imports, returnType, borrowedParams, state, effects)
				}); err != nil {
					return err
				}
				elseVars = copyRegionVars(state.regionVars)
			} else {
				elseVars = before
			}
			state.regionVars = mergeRegionVars(thenVars, elseVars)
			recordMergeConflicts(state, thenVars, elseVars, "then", "else")
			markUnknownRegions(state)
		case *frontend.WhileStmt:
			condType, _, err := checkExprWithEffects(s.Cond, locals, globals, funcs, types, module, imports, state, effects)
			if err != nil {
				return err
			}
			if !isConditionType(condType) {
				return fmt.Errorf("%s: condition must be bool or i32/u8", frontend.FormatPos(s.At))
			}
			bodyScopeID := regionNone
			if scoped, ok := state.whileScopes[s]; ok {
				bodyScopeID = scoped
			}
			before := copyRegionVars(state.regionVars)
			state.regionVars = copyRegionVars(before)
			state.loopDepth++
			if err := withActiveScope(state, bodyScopeID, func() error {
				return checkStmts(s.Body, locals, globals, funcs, types, module, imports, returnType, borrowedParams, state, effects)
			}); err != nil {
				state.loopDepth--
				return err
			}
			state.loopDepth--
			bodyVars := copyRegionVars(state.regionVars)
			state.regionVars = mergeRegionVars(before, bodyVars)
			recordMergeConflicts(state, before, bodyVars, "before", "body")
			markUnknownRegions(state)
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				iterType, _, err := checkExprWithEffects(s.Iterable, locals, globals, funcs, types, module, imports, state, effects)
				if err != nil {
					return err
				}
				elemType, err := collectionElementType(iterType, types)
				if err != nil {
					return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
				}
				if loopInfo, ok := locals[s.Name]; ok && loopInfo.TypeName != elemType {
					return fmt.Errorf("%s: for collection element type mismatch: local '%s' is %s, iterable yields %s", frontend.FormatPos(s.At), s.Name, loopInfo.TypeName, elemType)
				}
			} else {
				startType, _, err := checkExprWithEffects(s.Start, locals, globals, funcs, types, module, imports, state, effects)
				if err != nil {
					return err
				}
				endType, _, err := checkExprWithEffects(s.End, locals, globals, funcs, types, module, imports, state, effects)
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
			state.regionVars = copyRegionVars(before)
			state.loopDepth++
			if err := withActiveScope(state, bodyScopeID, func() error {
				return checkStmts(s.Body, locals, globals, funcs, types, module, imports, returnType, borrowedParams, state, effects)
			}); err != nil {
				state.loopDepth--
				return err
			}
			state.loopDepth--
			bodyVars := copyRegionVars(state.regionVars)
			state.regionVars = mergeRegionVars(before, bodyVars)
			recordMergeConflicts(state, before, bodyVars, "before", "body")
			markUnknownRegions(state)
		case *frontend.MatchStmt:
			scrutType, _, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects)
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
			seenDefault := false
			seenPatterns := map[string]frontend.Position{}
			before := copyRegionVars(state.regionVars)
			merged := copyRegionVars(before)
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
							return fmt.Errorf("%s: some pattern requires optional match value", frontend.FormatPos(some.At))
						}
						patType = optionalSomePatternType
					} else {
						var err error
						patType, _, err = checkExprWithEffects(c.Pattern, locals, globals, funcs, types, module, imports, state, effects)
						if err != nil {
							return err
						}
					}
					if scrutInfoOK && scrutInfo.Kind == TypeOptional && patType != "none" && patType != optionalSomePatternType {
						return fmt.Errorf("%s: optional match supports only 'none', 'some(name)', and '_' patterns in v0.7", frontend.FormatPos(c.At))
					}
					if !matchPatternCompatible(scrutType, patType, types) {
						return fmt.Errorf("%s: match pattern type mismatch: expected '%s', got '%s'", frontend.FormatPos(c.At), scrutType, patType)
					}
					if key := matchPatternKey(c.Pattern, patType); key != "" {
						if first, exists := seenPatterns[key]; exists {
							return fmt.Errorf("%s: duplicate match pattern (first at %s)", frontend.FormatPos(c.At), frontend.FormatPos(first))
						}
						seenPatterns[key] = c.At
					}
				}
				state.regionVars = copyRegionVars(before)
				caseScopeID := regionNone
				if i < len(caseScopes) {
					caseScopeID = caseScopes[i]
				}
				if err := withActiveScope(state, caseScopeID, func() error {
					return checkStmts(c.Body, locals, globals, funcs, types, module, imports, returnType, borrowedParams, state, effects)
				}); err != nil {
					return err
				}
				caseVars := copyRegionVars(state.regionVars)
				state.regionVars = mergeRegionVars(merged, caseVars)
				recordMergeConflicts(state, merged, caseVars, strings.Join(labels, "/"), fmt.Sprintf("case %d", i+1))
				merged = copyRegionVars(state.regionVars)
				labels = append(labels, fmt.Sprintf("case %d", i+1))
			}
			if seenDefault {
				state.regionVars = merged
			} else {
				state.regionVars = mergeRegionVars(before, merged)
				recordMergeConflicts(state, before, merged, "before", "cases")
			}
			markUnknownRegions(state)
		case *frontend.UnsafeStmt:
			scopeID := regionNone
			if scoped, ok := state.unsafeScopes[s]; ok {
				scopeID = scoped
			}
			state.enterUnsafe()
			if err := withActiveScope(state, scopeID, func() error {
				return checkStmts(s.Body, locals, globals, funcs, types, module, imports, returnType, borrowedParams, state, effects)
			}); err != nil {
				state.exitUnsafe()
				return err
			}
			state.exitUnsafe()
		case *frontend.ExprStmt:
			_, _, err := checkExprWithEffects(s.Expr, locals, globals, funcs, types, module, imports, state, effects)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s: unsupported statement", frontend.FormatPos(s.Pos()))
		}
	}
	return nil
}

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

const optionalSomePatternType = "__optional_some_pattern"

func matchPatternKey(pattern frontend.Expr, patternType string) string {
	switch p := pattern.(type) {
	case *frontend.NumberExpr:
		return fmt.Sprintf("i32:%d", p.Value)
	case *frontend.NoneLitExpr:
		return "optional:none"
	case *frontend.SomePatternExpr:
		return "optional:some"
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
	default:
		return "", fmt.Errorf("for collection requires slice or string, got '%s'", typeName)
	}
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
				return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(s.At), s.Name, s.Name)
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
			locals[s.Name] = LocalInfo{
				Base:      *slotIndex,
				SlotCount: info.SlotCount,
				TypeName:  resolved,
				Mutable:   s.Mutable,
				Const:     s.Const,
			}
			if scopes != nil {
				scopes.localScopes[s.Name] = scopes.currentScopeID()
			}
			*slotIndex += info.SlotCount
		case *frontend.IslandStmt:
			if _, exists := globals[s.Name]; exists {
				return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(s.At), s.Name, s.Name)
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
			if err := collectLocals(s.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
			}
		case *frontend.IfStmt:
			thenScopeID := regionNone
			elseScopeID := regionNone
			if scopes != nil {
				thenScopeID = scopes.enterScope()
			}
			if err := collectLocals(s.Then, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
			}
			if len(s.Else) > 0 {
				if scopes != nil {
					elseScopeID = scopes.enterScope()
				}
				if err := collectLocals(s.Else, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
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
			if _, exists := globals[s.Name]; exists {
				return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(s.At), s.Name, s.Name)
			}
			if _, exists := locals[s.Name]; exists {
				return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(s.At), s.Name)
			}
			valueType, err := inferExprTypeForDecl(s.Value, locals, globals, funcs, types, module, imports)
			if err != nil {
				return fmt.Errorf("%s: cannot infer if-let value type: %v", frontend.FormatPos(s.At), err)
			}
			valueInfo, err := ensureTypeInfo(valueType, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			if valueInfo.Kind != TypeOptional {
				return fmt.Errorf("%s: if let requires optional value, got '%s'", frontend.FormatPos(s.At), valueType)
			}
			elemInfo, err := ensureTypeInfo(valueInfo.ElemType, types)
			if err != nil {
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
			locals[s.Name] = LocalInfo{
				Base:      *slotIndex,
				SlotCount: elemInfo.SlotCount,
				TypeName:  valueInfo.ElemType,
				Mutable:   false,
			}
			if scopes != nil {
				scopes.localScopes[s.Name] = scopes.currentScopeID()
			}
			*slotIndex += elemInfo.SlotCount
			if err := collectLocals(s.Then, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
			}
			if len(s.Else) > 0 {
				if scopes != nil {
					elseScopeID = scopes.enterScope()
				}
				if err := collectLocals(s.Else, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
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
			scopeID := regionNone
			if scopes != nil {
				scopeID = scopes.enterScope()
			}
			if err := collectLocals(s.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
				scopes.whileScopes[s] = scopeID
			}
		case *frontend.ForRangeStmt:
			scopeID := regionNone
			if scopes != nil {
				scopeID = scopes.enterScope()
			}
			if _, exists := globals[s.Name]; exists {
				return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(s.At), s.Name, s.Name)
			}
			if _, exists := locals[s.Name]; exists {
				return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(s.At), s.Name)
			}
			loopType := "i32"
			var iterableInfo *TypeInfo
			if s.Iterable != nil {
				iterType, err := inferExprTypeForDecl(s.Iterable, locals, globals, funcs, types, module, imports)
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
			if err := collectLocals(s.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
				scopes.forScopes[s] = scopeID
			}
		case *frontend.MatchStmt:
			scrutType, err := inferExprTypeForDecl(s.Value, locals, globals, funcs, types, module, imports)
			if err != nil {
				return fmt.Errorf("%s: cannot infer match value type: %v", frontend.FormatPos(s.At), err)
			}
			info, err := ensureTypeInfo(scrutType, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			if info.SlotCount != 1 && info.Kind != TypeOptional {
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
			for i, c := range s.Cases {
				if scopes != nil {
					caseScopeIDs[i] = scopes.enterScope()
				} else {
					caseScopeIDs[i] = regionNone
				}
				if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
					if info.Kind != TypeOptional {
						return fmt.Errorf("%s: some pattern requires optional match value", frontend.FormatPos(some.At))
					}
					if _, exists := globals[some.Name]; exists {
						return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(some.At), some.Name, some.Name)
					}
					if _, exists := locals[some.Name]; exists {
						return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(some.At), some.Name)
					}
					elemInfo, err := ensureTypeInfo(info.ElemType, types)
					if err != nil {
						return fmt.Errorf("%s: %v", frontend.FormatPos(some.At), err)
					}
					locals[some.Name] = LocalInfo{
						Base:      *slotIndex,
						SlotCount: elemInfo.SlotCount,
						TypeName:  info.ElemType,
						Mutable:   false,
					}
					if scopes != nil {
						scopes.localScopes[some.Name] = scopes.currentScopeID()
					}
					*slotIndex += elemInfo.SlotCount
				}
				if err := collectLocals(c.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
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
			if err := collectLocals(s.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
				scopes.unsafeScopes[s] = scopeID
			}
		case *frontend.ExprStmt:
		}
	}
	return nil
}
