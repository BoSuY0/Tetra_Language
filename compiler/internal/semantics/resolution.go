package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

const importSymbolPrefix = "\x00symbol:"

func collectImportAliases(file *frontend.FileAST) (map[string]string, error) {
	aliases := make(map[string]string)
	topLevel := topLevelDeclarationNames(file)
	for _, imp := range file.Imports {
		if imp.Path == "" {
			return nil, fmt.Errorf("%s: import path required", frontend.FormatPos(imp.At))
		}
		if len(imp.Items) > 0 {
			for _, item := range imp.Items {
				if item == "" {
					return nil, fmt.Errorf("%s: empty selective import", frontend.FormatPos(imp.At))
				}
				if _, exists := aliases[item]; exists {
					return nil, fmt.Errorf("%s: duplicate import alias '%s'", frontend.FormatPos(imp.At), item)
				}
				if _, exists := topLevel[item]; exists {
					return nil, fmt.Errorf("%s: import alias '%s' conflicts with declaration '%s'", frontend.FormatPos(imp.At), item, item)
				}
				aliases[item] = importSymbolPrefix + imp.Path + "." + item
			}
			continue
		}
		if imp.Alias == "" {
			return nil, fmt.Errorf("%s: import alias required", frontend.FormatPos(imp.At))
		}
		if _, exists := aliases[imp.Alias]; exists {
			return nil, fmt.Errorf("%s: duplicate import alias '%s'", frontend.FormatPos(imp.At), imp.Alias)
		}
		if _, exists := topLevel[imp.Alias]; exists {
			return nil, fmt.Errorf("%s: import alias '%s' conflicts with declaration '%s'", frontend.FormatPos(imp.At), imp.Alias, imp.Alias)
		}
		aliases[imp.Alias] = imp.Path
	}
	return aliases, nil
}

func importSymbolTarget(target string) (string, bool) {
	if !strings.HasPrefix(target, importSymbolPrefix) {
		return "", false
	}
	return strings.TrimPrefix(target, importSymbolPrefix), true
}

func topLevelDeclarationNames(file *frontend.FileAST) map[string]struct{} {
	names := map[string]struct{}{}
	for _, fn := range file.Funcs {
		names[fn.Name] = struct{}{}
	}
	for _, glob := range file.Globals {
		names[glob.Name] = struct{}{}
	}
	for _, st := range file.Structs {
		names[st.Name] = struct{}{}
	}
	for _, en := range file.Enums {
		names[en.Name] = struct{}{}
	}
	for _, state := range file.States {
		names[state.Name] = struct{}{}
	}
	for _, view := range file.Views {
		names[view.Name] = struct{}{}
	}
	for _, actor := range file.Actors {
		names[actor.Name] = struct{}{}
	}
	for _, proto := range file.Protocols {
		names[proto.Name] = struct{}{}
	}
	for _, capsule := range file.Capsules {
		if capsule == nil {
			continue
		}
		names[capsule.Name] = struct{}{}
	}
	return names
}

func qualifyName(module, name string) string {
	if module == "" {
		return name
	}
	return module + "." + name
}

func resolveTypeName(ref *frontend.TypeRef, module string, imports map[string]string) (string, error) {
	if ref == nil {
		return "", fmt.Errorf("missing type")
	}
	switch ref.Kind {
	case frontend.TypeRefSlice:
		if ref.Elem == nil {
			return "", fmt.Errorf("%s: missing slice element type", frontend.FormatPos(ref.At))
		}
		elem, err := resolveTypeName(ref.Elem, module, imports)
		if err != nil {
			return "", err
		}
		return "[]" + elem, nil
	case frontend.TypeRefArray:
		if ref.Elem == nil {
			return "", fmt.Errorf("%s: missing array element type", frontend.FormatPos(ref.At))
		}
		if ref.Len <= 0 {
			return "", fmt.Errorf("%s: array size must be positive constant", frontend.FormatPos(ref.At))
		}
		elem, err := resolveTypeName(ref.Elem, module, imports)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("[%d]%s", ref.Len, elem), nil
	case frontend.TypeRefOptional:
		if ref.Elem == nil {
			return "", fmt.Errorf("%s: missing optional payload type", frontend.FormatPos(ref.At))
		}
		elem, err := resolveTypeName(ref.Elem, module, imports)
		if err != nil {
			return "", err
		}
		return optionalTypeName(elem), nil
	case frontend.TypeRefNamed:
		if ref.Name == "" {
			return "", fmt.Errorf("%s: missing type name", frontend.FormatPos(ref.At))
		}
		if canonical, ok := canonicalBuiltinType(ref.Name); ok {
			return canonical, nil
		}
		parts := strings.Split(ref.Name, ".")
		if len(parts) == 1 {
			if target, ok := imports[ref.Name]; ok {
				if symbol, isSymbol := importSymbolTarget(target); isSymbol {
					return symbol, nil
				}
			}
			return qualifyName(module, ref.Name), nil
		}
		if target, ok := imports[parts[0]]; ok {
			if _, isSymbol := importSymbolTarget(target); isSymbol {
				return "", fmt.Errorf("%s: selective import '%s' cannot be used as a namespace", frontend.FormatPos(ref.At), parts[0])
			}
			if len(parts) != 2 {
				return "", fmt.Errorf("%s: expected '%s.<type>'", frontend.FormatPos(ref.At), parts[0])
			}
			return target + "." + parts[1], nil
		}
		return ref.Name, nil
	case frontend.TypeRefFunction:
		for i := range ref.Params {
			paramName, err := resolveTypeName(&ref.Params[i], module, imports)
			if err != nil {
				return "", err
			}
			ref.Params[i].Name = paramName
		}
		if ref.Return == nil {
			return "", fmt.Errorf("%s: missing function return type", frontend.FormatPos(ref.At))
		}
		retName, err := resolveTypeName(ref.Return, module, imports)
		if err != nil {
			return "", err
		}
		ref.Return.Name = retName
		if ref.Throws != nil {
			throwsName, err := resolveTypeName(ref.Throws, module, imports)
			if err != nil {
				return "", err
			}
			ref.Throws.Name = throwsName
		}
		if _, err := normalizeEffects(ref.Uses, ref.At); err != nil {
			return "", err
		}
		return "fnptr", nil
	default:
		return "", fmt.Errorf("%s: unsupported type reference kind %d", frontend.FormatPos(ref.At), ref.Kind)
	}
}

func canonicalBuiltinType(name string) (string, bool) {
	switch name {
	case "i32", "Int":
		return "i32", true
	case "i64", "Int64":
		return "i64", true
	case "u8", "UInt8", "Byte":
		return "u8", true
	case "u16", "UInt16":
		return "u16", true
	case "str", "String":
		return "str", true
	case "bool", "Bool":
		return "bool", true
	case "ptr", "island", "cap.io", "cap.mem", "actor", "consent.token", "secret.i32":
		return name, true
	case "ConsentToken":
		return "consent.token", true
	case "SecretInt":
		return "secret.i32", true
	default:
		return "", false
	}
}

func resolveEnumCaseExpr(expr frontend.Expr, locals map[string]LocalInfo, globals map[string]GlobalInfo, types map[string]*TypeInfo, module string, imports map[string]string) (string, EnumCaseInfo, bool, error) {
	field, ok := expr.(*frontend.FieldAccessExpr)
	if !ok {
		return "", EnumCaseInfo{}, false, nil
	}
	baseName, fields, pos, ok := splitFieldPath(field.Base)
	if !ok {
		return "", EnumCaseInfo{}, false, nil
	}
	if _, exists := locals[baseName]; exists {
		return "", EnumCaseInfo{}, false, nil
	}
	if _, exists := globals[baseName]; exists {
		return "", EnumCaseInfo{}, false, nil
	}
	parts := append([]string{baseName}, fields...)
	ref := frontend.TypeRef{At: pos, Kind: frontend.TypeRefNamed, Name: strings.Join(parts, ".")}
	typeName, err := resolveTypeName(&ref, module, imports)
	if err != nil {
		return "", EnumCaseInfo{}, false, err
	}
	info, ok := types[typeName]
	if !ok || info.Kind != TypeEnum {
		if altName, altInfo, found := findUniqueEnumByShortName(ref.Name, types); found {
			typeName = altName
			info = altInfo
		} else {
			return "", EnumCaseInfo{}, false, nil
		}
	}
	caseInfo, ok := info.CaseMap[field.Field]
	if !ok {
		return "", EnumCaseInfo{}, true, fmt.Errorf("%s: unknown enum case '%s' for '%s'", frontend.FormatPos(field.At), field.Field, displayTypeName(typeName, module))
	}
	if len(caseInfo.PayloadTypes) > 0 {
		return "", EnumCaseInfo{}, true, fmt.Errorf("%s: enum case '%s.%s' requires payload arguments", frontend.FormatPos(field.At), displayTypeName(typeName, module), field.Field)
	}
	if len(caseInfo.PayloadTypes) == 0 && field.Field == "" {
		return "", EnumCaseInfo{}, true, fmt.Errorf("%s: malformed enum case reference", frontend.FormatPos(field.At))
	}
	field.EnumType = typeName
	field.EnumOrdinal = caseInfo.Ordinal
	return typeName, caseInfo, true, nil
}

func resolveEnumCasePattern(pattern *frontend.EnumCasePatternExpr, types map[string]*TypeInfo, module string, imports map[string]string) (string, EnumCaseInfo, bool, error) {
	ref := frontend.TypeRef{At: pattern.At, Kind: frontend.TypeRefNamed, Name: pattern.TypeName}
	typeName, err := resolveTypeName(&ref, module, imports)
	if err != nil {
		return "", EnumCaseInfo{}, false, err
	}
	info, ok := types[typeName]
	if !ok || info.Kind != TypeEnum {
		if altName, altInfo, found := findUniqueEnumByShortName(pattern.TypeName, types); found {
			typeName = altName
			info = altInfo
		} else {
			return "", EnumCaseInfo{}, false, nil
		}
	}
	caseInfo, ok := info.CaseMap[pattern.CaseName]
	if !ok {
		return "", EnumCaseInfo{}, true, fmt.Errorf("%s: unknown enum case '%s' for '%s'", frontend.FormatPos(pattern.At), pattern.CaseName, displayTypeName(typeName, module))
	}
	pattern.EnumType = typeName
	pattern.EnumOrdinal = caseInfo.Ordinal
	pattern.PayloadSlots = append(pattern.PayloadSlots[:0], caseInfo.PayloadSlots...)
	return typeName, caseInfo, true, nil
}

func resolveEnumCaseConstructorCall(e *frontend.CallExpr, types map[string]*TypeInfo, module string, imports map[string]string) (string, EnumCaseInfo, bool, error) {
	parts := strings.Split(e.Name, ".")
	if len(parts) < 2 {
		return "", EnumCaseInfo{}, false, nil
	}
	caseName := parts[len(parts)-1]
	typeRef := frontend.TypeRef{At: e.At, Kind: frontend.TypeRefNamed, Name: strings.Join(parts[:len(parts)-1], ".")}
	typeName, err := resolveTypeName(&typeRef, module, imports)
	if err != nil {
		return "", EnumCaseInfo{}, false, err
	}
	info, ok := types[typeName]
	if !ok || info.Kind != TypeEnum {
		shortName := strings.Join(parts[:len(parts)-1], ".")
		if altName, altInfo, found := findUniqueEnumByShortName(shortName, types); found {
			typeName = altName
			info = altInfo
		} else {
			return "", EnumCaseInfo{}, false, nil
		}
	}
	caseInfo, ok := info.CaseMap[caseName]
	if !ok {
		return "", EnumCaseInfo{}, true, fmt.Errorf("%s: unknown enum case '%s' for '%s'", frontend.FormatPos(e.At), caseName, displayTypeName(typeName, module))
	}
	return typeName, caseInfo, true, nil
}

func findUniqueEnumByShortName(shortName string, types map[string]*TypeInfo) (string, *TypeInfo, bool) {
	var foundName string
	var foundInfo *TypeInfo
	for name, info := range types {
		if info == nil || info.Kind != TypeEnum {
			continue
		}
		if name != shortName && !strings.HasSuffix(name, "."+shortName) {
			continue
		}
		if foundInfo != nil && foundName != name {
			return "", nil, false
		}
		foundName = name
		foundInfo = info
	}
	return foundName, foundInfo, foundInfo != nil
}

func displayTypeName(name, module string) string {
	prefix := module + "."
	if module != "" && strings.HasPrefix(name, prefix) {
		return strings.TrimPrefix(name, prefix)
	}
	return name
}

func symbolBelongsToModule(name, module string) bool {
	if module == "" {
		return !strings.Contains(name, ".")
	}
	return name == module || strings.HasPrefix(name, module+".")
}

func ensureFuncVisible(name string, sig FuncSig, module string, pos frontend.Position) error {
	if symbolBelongsToModule(name, module) || sig.Public || strings.HasPrefix(name, "core.") {
		return nil
	}
	return fmt.Errorf("%s: private function '%s' is not visible from module '%s'", frontend.FormatPos(pos), name, module)
}

func ensureTypeVisible(name string, info *TypeInfo, module string, pos frontend.Position) error {
	if info == nil || symbolBelongsToModule(name, module) || info.Public {
		return nil
	}
	return fmt.Errorf("%s: private type '%s' is not visible from module '%s'", frontend.FormatPos(pos), name, module)
}

func resolveCallName(name string, module string, imports map[string]string, pos frontend.Position) (string, error) {
	parts := strings.Split(name, ".")
	if len(parts) == 1 {
		if target, ok := imports[name]; ok {
			if symbol, isSymbol := importSymbolTarget(target); isSymbol {
				return symbol, nil
			}
		}
		return qualifyName(module, name), nil
	}
	if target, ok := imports[parts[0]]; ok {
		if _, isSymbol := importSymbolTarget(target); isSymbol {
			return "", fmt.Errorf("%s: selective import '%s' cannot be used as a namespace", frontend.FormatPos(pos), parts[0])
		}
		if len(parts) < 2 {
			return "", fmt.Errorf("%s: expected '%s.<func>'", frontend.FormatPos(pos), parts[0])
		}
		suffix := strings.Join(parts[1:], ".")
		if suffix == "" {
			return "", fmt.Errorf("%s: expected '%s.<func>'", frontend.FormatPos(pos), parts[0])
		}
		return target + "." + suffix, nil
	}
	modPath := strings.Join(parts[:len(parts)-1], ".")
	return modPath + "." + parts[len(parts)-1], nil
}

func resolveKnownCallName(name string, funcs map[string]FuncSig, module string, imports map[string]string, pos frontend.Position) (string, error) {
	if _, ok := funcs[name]; ok {
		return name, nil
	}
	resolved, err := resolveCallName(name, module, imports, pos)
	if err != nil {
		return "", err
	}
	if _, ok := funcs[resolved]; ok {
		return resolved, nil
	}
	if module != "" && strings.Contains(name, ".") {
		moduleLocal := qualifyName(module, name)
		if _, ok := funcs[moduleLocal]; ok {
			return moduleLocal, nil
		}
	}
	return resolved, nil
}

type assignTargetInfo struct {
	Name           string
	Mutable        bool
	Const          bool
	TypeName       string
	Offset         int
	Global         bool
	ActorField     bool
	ActorFieldSlot int
}

func resolveAssignTarget(expr frontend.Expr, locals map[string]LocalInfo, globals map[string]GlobalInfo, types map[string]*TypeInfo) (assignTargetInfo, string, error) {
	if idx, ok := expr.(*frontend.IndexExpr); ok {
		baseName, fields, pos, ok := splitFieldPath(idx.Base)
		if !ok {
			return assignTargetInfo{}, "", fmt.Errorf("%s: invalid assignment target", frontend.FormatPos(pos))
		}
		baseType := ""
		baseOffset := 0
		mutable := false
		constant := false
		global := false
		if baseInfo, ok := locals[baseName]; ok {
			baseType = baseInfo.TypeName
			baseOffset = baseInfo.Base
			mutable = baseInfo.Mutable
			constant = baseInfo.Const
		} else if globalInfo, ok := globals[baseName]; ok {
			baseType = globalInfo.TypeName
			baseOffset = globalInfo.DataIndex
			mutable = globalInfo.Mutable
			constant = globalInfo.Const
			global = true
		} else {
			return assignTargetInfo{}, "", fmt.Errorf("%s: unknown identifier '%s'", frontend.FormatPos(pos), baseName)
		}
		if _, err := ensureTypeInfo(baseType, types); err != nil {
			return assignTargetInfo{}, "", err
		}
		baseType, _, _, err := resolveFieldChain(baseType, baseOffset, fields, types, pos)
		if err != nil {
			return assignTargetInfo{}, "", err
		}
		info, err := ensureTypeInfo(baseType, types)
		if err != nil {
			return assignTargetInfo{}, "", err
		}
		if info.Kind == TypeStr {
			return assignTargetInfo{}, "", fmt.Errorf("%s: cannot assign into str", frontend.FormatPos(pos))
		}
		if info.Kind != TypeSlice && info.Kind != TypeArray {
			return assignTargetInfo{}, "", fmt.Errorf("%s: cannot index '%s'", frontend.FormatPos(pos), baseType)
		}
		return assignTargetInfo{Name: baseName, Mutable: mutable, Const: constant, TypeName: info.ElemType, Global: global}, info.ElemType, nil
	}

	baseName, fields, pos, ok := splitFieldPath(expr)
	if !ok {
		return assignTargetInfo{}, "", fmt.Errorf("%s: invalid assignment target", frontend.FormatPos(pos))
	}
	info, ok := locals[baseName]
	if !ok {
		if globalInfo, ok := globals[baseName]; ok {
			if _, err := ensureTypeInfo(globalInfo.TypeName, types); err != nil {
				return assignTargetInfo{}, "", err
			}
			targetType, _, offset, err := resolveFieldChain(globalInfo.TypeName, globalInfo.DataIndex, fields, types, pos)
			if err != nil {
				return assignTargetInfo{}, "", err
			}
			if err := rejectCollectionInternalAssignment(globalInfo.TypeName, fields, types, pos); err != nil {
				return assignTargetInfo{}, "", err
			}
			return assignTargetInfo{Name: baseName, Mutable: globalInfo.Mutable, Const: globalInfo.Const, TypeName: targetType, Offset: offset, Global: true}, targetType, nil
		}
		return assignTargetInfo{}, "", fmt.Errorf("%s: unknown identifier '%s'", frontend.FormatPos(pos), baseName)
	}
	if _, err := ensureTypeInfo(info.TypeName, types); err != nil {
		return assignTargetInfo{}, "", err
	}
	if info.ActorField {
		if len(fields) > 0 {
			return assignTargetInfo{}, "", fmt.Errorf("%s: '%s' is not a struct", frontend.FormatPos(pos), info.TypeName)
		}
		return assignTargetInfo{
			Name:           baseName,
			Mutable:        info.Mutable,
			Const:          info.Const,
			TypeName:       info.TypeName,
			ActorField:     true,
			ActorFieldSlot: info.ActorFieldSlot,
		}, info.TypeName, nil
	}
	targetType, _, offset, err := resolveFieldChain(info.TypeName, info.Base, fields, types, pos)
	if err != nil {
		return assignTargetInfo{}, "", err
	}
	if err := rejectCollectionInternalAssignment(info.TypeName, fields, types, pos); err != nil {
		return assignTargetInfo{}, "", err
	}
	return assignTargetInfo{Name: baseName, Mutable: info.Mutable, Const: info.Const, TypeName: targetType, Offset: offset}, targetType, nil
}

func rejectCollectionInternalAssignment(typeName string, fields []string, types map[string]*TypeInfo, pos frontend.Position) error {
	current := typeName
	for _, field := range fields {
		info, ok := types[current]
		if !ok {
			return fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), current)
		}
		if info.Kind == TypeArray && (field == "ptr" || field == "len") {
			return fmt.Errorf("%s: cannot assign to fixed-array internals ('ptr'/'len'); assign elements via index instead", frontend.FormatPos(pos))
		}
		if info.Kind == TypeSlice && (field == "ptr" || field == "len") {
			return fmt.Errorf("%s: cannot assign to slice internals ('ptr'/'len'); assign elements via index instead", frontend.FormatPos(pos))
		}
		if info.Kind == TypeStr && (field == "ptr" || field == "len") {
			return fmt.Errorf("%s: cannot assign to string internals ('ptr'/'len')", frontend.FormatPos(pos))
		}
		fieldInfo, ok := info.FieldMap[field]
		if !ok {
			return fmt.Errorf("%s: unknown field '%s'", frontend.FormatPos(pos), field)
		}
		current = fieldInfo.TypeName
	}
	return nil
}

func ResolveFieldAccessType(expr frontend.Expr, locals map[string]LocalInfo, globals map[string]GlobalInfo, types map[string]*TypeInfo) (assignTargetInfo, string, error) {
	baseName, fields, pos, ok := splitFieldPath(expr)
	if !ok {
		return assignTargetInfo{}, "", fmt.Errorf("%s: invalid field access", frontend.FormatPos(pos))
	}
	if info, ok := locals[baseName]; ok {
		if _, err := ensureTypeInfo(info.TypeName, types); err != nil {
			return assignTargetInfo{}, "", err
		}
		targetType, _, offset, err := resolveFieldChain(info.TypeName, info.Base, fields, types, pos)
		if err != nil {
			return assignTargetInfo{}, "", err
		}
		return assignTargetInfo{Name: baseName, Mutable: info.Mutable, Const: info.Const, TypeName: targetType, Offset: offset}, targetType, nil
	}
	if info, ok := globals[baseName]; ok {
		if _, err := ensureTypeInfo(info.TypeName, types); err != nil {
			return assignTargetInfo{}, "", err
		}
		targetType, _, offset, err := resolveFieldChain(info.TypeName, info.DataIndex, fields, types, pos)
		if err != nil {
			return assignTargetInfo{}, "", err
		}
		return assignTargetInfo{Name: baseName, Mutable: info.Mutable, Const: info.Const, TypeName: targetType, Offset: offset, Global: true}, targetType, nil
	}
	return assignTargetInfo{}, "", fmt.Errorf("%s: unknown identifier '%s'", frontend.FormatPos(pos), baseName)
}

func splitFieldPath(expr frontend.Expr) (string, []string, frontend.Position, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name, nil, e.At, true
	case *frontend.FieldAccessExpr:
		baseName, fields, pos, ok := splitFieldPath(e.Base)
		if !ok {
			return "", nil, pos, false
		}
		fields = append(fields, e.Field)
		return baseName, fields, e.At, true
	default:
		return "", nil, expr.Pos(), false
	}
}

func resolveFieldChain(typeName string, baseOffset int, fields []string, types map[string]*TypeInfo, pos frontend.Position) (string, int, int, error) {
	offset := baseOffset
	current := typeName
	for _, field := range fields {
		info, ok := types[current]
		if !ok {
			return "", 0, 0, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), current)
		}
		if info.Kind != TypeStruct && info.Kind != TypeSlice && info.Kind != TypeArray && info.Kind != TypeStr {
			return "", 0, 0, fmt.Errorf("%s: '%s' is not a struct", frontend.FormatPos(pos), current)
		}
		fieldInfo, ok := info.FieldMap[field]
		if !ok {
			return "", 0, 0, fmt.Errorf("%s: unknown field '%s'", frontend.FormatPos(pos), field)
		}
		offset += fieldInfo.Offset
		current = fieldInfo.TypeName
	}
	info, ok := types[current]
	if !ok {
		return "", 0, 0, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), current)
	}
	return current, info.SlotCount, offset, nil
}
