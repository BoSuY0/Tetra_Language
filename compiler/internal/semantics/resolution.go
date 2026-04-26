package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

func collectImportAliases(file *frontend.FileAST) (map[string]string, error) {
	aliases := make(map[string]string)
	for _, imp := range file.Imports {
		if imp.Alias == "" {
			return nil, fmt.Errorf("%s: import alias required", frontend.FormatPos(imp.At))
		}
		if _, exists := aliases[imp.Alias]; exists {
			return nil, fmt.Errorf("%s: duplicate import alias '%s'", frontend.FormatPos(imp.At), imp.Alias)
		}
		aliases[imp.Alias] = imp.Path
	}
	return aliases, nil
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
		if ref.Len < 0 {
			return "", fmt.Errorf("%s: invalid array length", frontend.FormatPos(ref.At))
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
			return qualifyName(module, ref.Name), nil
		}
		if target, ok := imports[parts[0]]; ok {
			if len(parts) != 2 {
				return "", fmt.Errorf("%s: expected '%s.<type>'", frontend.FormatPos(ref.At), parts[0])
			}
			return target + "." + parts[1], nil
		}
		return ref.Name, nil
	default:
		return "", fmt.Errorf("%s: unsupported type", frontend.FormatPos(ref.At))
	}
}

func canonicalBuiltinType(name string) (string, bool) {
	switch name {
	case "i32", "Int":
		return "i32", true
	case "u8", "UInt8", "Byte":
		return "u8", true
	case "str", "String":
		return "str", true
	case "bool", "Bool":
		return "bool", true
	case "ptr", "island", "cap.io", "cap.mem", "actor":
		return name, true
	default:
		return "", false
	}
}

func resolveEnumCaseExpr(expr frontend.Expr, locals map[string]LocalInfo, globals map[string]GlobalInfo, types map[string]*TypeInfo, module string, imports map[string]string) (string, EnumCaseInfo, bool, error) {
	field, ok := expr.(*frontend.FieldAccessExpr)
	if !ok {
		return "", EnumCaseInfo{}, false, nil
	}
	base, ok := field.Base.(*frontend.IdentExpr)
	if !ok {
		return "", EnumCaseInfo{}, false, nil
	}
	if _, exists := locals[base.Name]; exists {
		return "", EnumCaseInfo{}, false, nil
	}
	if _, exists := globals[base.Name]; exists {
		return "", EnumCaseInfo{}, false, nil
	}
	ref := frontend.TypeRef{At: base.At, Kind: frontend.TypeRefNamed, Name: base.Name}
	typeName, err := resolveTypeName(&ref, module, imports)
	if err != nil {
		return "", EnumCaseInfo{}, false, err
	}
	info, ok := types[typeName]
	if !ok || info.Kind != TypeEnum {
		return "", EnumCaseInfo{}, false, nil
	}
	caseInfo, ok := info.CaseMap[field.Field]
	if !ok {
		return "", EnumCaseInfo{}, true, fmt.Errorf("%s: unknown enum case '%s' for '%s'", frontend.FormatPos(field.At), field.Field, displayTypeName(typeName, module))
	}
	field.EnumType = typeName
	field.EnumOrdinal = caseInfo.Ordinal
	return typeName, caseInfo, true, nil
}

func displayTypeName(name, module string) string {
	prefix := module + "."
	if module != "" && strings.HasPrefix(name, prefix) {
		return strings.TrimPrefix(name, prefix)
	}
	return name
}

func resolveCallName(name string, module string, imports map[string]string, pos frontend.Position) (string, error) {
	parts := strings.Split(name, ".")
	if len(parts) == 1 {
		return qualifyName(module, name), nil
	}
	if target, ok := imports[parts[0]]; ok {
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

type assignTargetInfo struct {
	Name     string
	Mutable  bool
	Const    bool
	TypeName string
	Offset   int
}

func resolveAssignTarget(expr frontend.Expr, locals map[string]LocalInfo, types map[string]*TypeInfo) (assignTargetInfo, string, error) {
	if idx, ok := expr.(*frontend.IndexExpr); ok {
		baseName, fields, pos, ok := splitFieldPath(idx.Base)
		if !ok {
			return assignTargetInfo{}, "", fmt.Errorf("%s: invalid assignment target", frontend.FormatPos(pos))
		}
		baseInfo, ok := locals[baseName]
		if !ok {
			return assignTargetInfo{}, "", fmt.Errorf("%s: unknown identifier '%s'", frontend.FormatPos(pos), baseName)
		}
		if _, err := ensureTypeInfo(baseInfo.TypeName, types); err != nil {
			return assignTargetInfo{}, "", err
		}
		baseType, _, _, err := resolveFieldChain(baseInfo.TypeName, baseInfo.Base, fields, types, pos)
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
		if info.Kind != TypeSlice {
			return assignTargetInfo{}, "", fmt.Errorf("%s: cannot index '%s'", frontend.FormatPos(pos), baseType)
		}
		return assignTargetInfo{Name: baseName, Mutable: baseInfo.Mutable, Const: baseInfo.Const, TypeName: info.ElemType}, info.ElemType, nil
	}

	baseName, fields, pos, ok := splitFieldPath(expr)
	if !ok {
		return assignTargetInfo{}, "", fmt.Errorf("%s: invalid assignment target", frontend.FormatPos(pos))
	}
	info, ok := locals[baseName]
	if !ok {
		return assignTargetInfo{}, "", fmt.Errorf("%s: unknown identifier '%s'", frontend.FormatPos(pos), baseName)
	}
	if _, err := ensureTypeInfo(info.TypeName, types); err != nil {
		return assignTargetInfo{}, "", err
	}
	targetType, _, offset, err := resolveFieldChain(info.TypeName, info.Base, fields, types, pos)
	if err != nil {
		return assignTargetInfo{}, "", err
	}
	return assignTargetInfo{Name: baseName, Mutable: info.Mutable, Const: info.Const, TypeName: targetType, Offset: offset}, targetType, nil
}

func ResolveFieldAccessType(expr frontend.Expr, locals map[string]LocalInfo, types map[string]*TypeInfo) (assignTargetInfo, string, error) {
	baseName, fields, pos, ok := splitFieldPath(expr)
	if !ok {
		return assignTargetInfo{}, "", fmt.Errorf("%s: invalid field access", frontend.FormatPos(pos))
	}
	info, ok := locals[baseName]
	if !ok {
		return assignTargetInfo{}, "", fmt.Errorf("%s: unknown identifier '%s'", frontend.FormatPos(pos), baseName)
	}
	if _, err := ensureTypeInfo(info.TypeName, types); err != nil {
		return assignTargetInfo{}, "", err
	}
	targetType, _, offset, err := resolveFieldChain(info.TypeName, info.Base, fields, types, pos)
	if err != nil {
		return assignTargetInfo{}, "", err
	}
	return assignTargetInfo{Name: baseName, Mutable: info.Mutable, Const: info.Const, TypeName: targetType, Offset: offset}, targetType, nil
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
		if info.Kind != TypeStruct && info.Kind != TypeSlice && info.Kind != TypeStr {
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
