package semantics

import (
	"fmt"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics/model"
)

type CheckedProgram = model.CheckedProgram
type CheckedFunc = model.CheckedFunc
type ActorStateField = model.ActorStateField
type LocalInfo = model.LocalInfo
type FunctionFieldInfo = model.FunctionFieldInfo
type CallableEscapeKind = model.CallableEscapeKind
type GlobalInfo = model.GlobalInfo
type GlobalArrayBackingInfo = model.GlobalArrayBackingInfo
type FuncSig = model.FuncSig
type ResourceProvenance = model.ResourceProvenance
type ReturnRegionSummary = model.ReturnRegionSummary
type ReturnResourceSummary = model.ReturnResourceSummary
type CheckedStruct = model.CheckedStruct
type CheckedEnum = model.CheckedEnum
type CheckedProtocol = model.CheckedProtocol
type CheckedUIState = model.CheckedUIState
type CheckedUIView = model.CheckedUIView
type TypeKind = model.TypeKind
type FieldInfo = model.FieldInfo
type TypeInfo = model.TypeInfo
type EnumCaseInfo = model.EnumCaseInfo

const (
	FnPtrEnvSlotCount       = model.FnPtrEnvSlotCount
	FnPtrSlotCount          = model.FnPtrSlotCount
	CallableHandleSlotCount = model.CallableHandleSlotCount

	CallableEscapeLocalSnapshot = model.CallableEscapeLocalSnapshot
	CallableEscapeHeap          = model.CallableEscapeHeap
	CallableEscapeGlobal        = model.CallableEscapeGlobal
	CallableEscapeThread        = model.CallableEscapeThread

	TypeI32      = model.TypeI32
	TypeI64      = model.TypeI64
	TypeU8       = model.TypeU8
	TypeBool     = model.TypeBool
	TypePtr      = model.TypePtr
	TypeSlice    = model.TypeSlice
	TypeStr      = model.TypeStr
	TypeStruct   = model.TypeStruct
	TypeArray    = model.TypeArray
	TypeIsland   = model.TypeIsland
	TypeCap      = model.TypeCap
	TypeActor    = model.TypeActor
	TypeEnum     = model.TypeEnum
	TypeOptional = model.TypeOptional

	MaxActorStateSlots = model.MaxActorStateSlots
)

func makeSliceTypeInfo(name, elem string) *TypeInfo {
	fieldMap := map[string]FieldInfo{
		"ptr": {Name: "ptr", TypeName: "ptr", Offset: 0, SlotCount: 1},
		"len": {Name: "len", TypeName: "i32", Offset: 1, SlotCount: 1},
	}
	fields := []FieldInfo{fieldMap["ptr"], fieldMap["len"]}
	return &TypeInfo{
		Name:      name,
		Kind:      TypeSlice,
		Public:    true,
		Fields:    fields,
		FieldMap:  fieldMap,
		SlotCount: 2,
		ElemType:  elem,
	}
}

func makeArrayTypeInfo(name, elem string, n int) *TypeInfo {
	fieldMap := map[string]FieldInfo{
		"ptr": {Name: "ptr", TypeName: "ptr", Offset: 0, SlotCount: 1},
		"len": {Name: "len", TypeName: "i32", Offset: 1, SlotCount: 1},
	}
	fields := []FieldInfo{fieldMap["ptr"], fieldMap["len"]}
	return &TypeInfo{
		Name:      name,
		Kind:      TypeArray,
		Public:    true,
		Fields:    fields,
		FieldMap:  fieldMap,
		SlotCount: 2,
		ElemType:  elem,
		ArrayLen:  n,
	}
}

func makeStrTypeInfo() *TypeInfo {
	info := makeSliceTypeInfo("str", "u8")
	info.Kind = TypeStr
	return info
}

func makeStructTypeInfo(name string, fields []FieldInfo) *TypeInfo {
	fieldMap := make(map[string]FieldInfo, len(fields))
	offset := 0
	structFields := make([]FieldInfo, 0, len(fields))
	for _, field := range fields {
		slotCount := field.SlotCount
		if slotCount <= 0 {
			slotCount = 1
		}
		resolved := FieldInfo{
			Name:           field.Name,
			TypeName:       field.TypeName,
			Offset:         offset,
			SlotCount:      slotCount,
			UserAssignable: true,
		}
		offset += slotCount
		structFields = append(structFields, resolved)
		fieldMap[resolved.Name] = resolved
	}
	return &TypeInfo{
		Name:      name,
		Kind:      TypeStruct,
		Public:    true,
		Repr:      frontend.StructReprDefault,
		Fields:    structFields,
		FieldMap:  fieldMap,
		SlotCount: offset,
	}
}

func baseTypes() map[string]*TypeInfo {
	types := map[string]*TypeInfo{
		"i32":           {Name: "i32", Kind: TypeI32, SlotCount: 1},
		"i64":           {Name: "i64", Kind: TypeI64, SlotCount: 1, Public: true},
		"u8":            {Name: "u8", Kind: TypeU8, SlotCount: 1, Public: true},
		"u16":           {Name: "u16", Kind: TypeU8, SlotCount: 1, Public: true},
		"c_int":         {Name: "c_int", Kind: TypeI32, SlotCount: 1, Public: true},
		"c_uint":        {Name: "c_uint", Kind: TypeI32, SlotCount: 1, Public: true},
		"bool":          {Name: "bool", Kind: TypeBool, SlotCount: 1, Public: true},
		"ptr":           {Name: "ptr", Kind: TypePtr, SlotCount: 1, Public: true},
		"fnptr":         {Name: "fnptr", Kind: TypePtr, SlotCount: FnPtrSlotCount, Public: true},
		"str":           makeStrTypeInfo(),
		"actor":         {Name: "actor", Kind: TypeActor, SlotCount: 1, Public: true},
		"task.error":    {Name: "task.error", Kind: TypeI32, SlotCount: 1, Public: true},
		"task.group":    {Name: "task.group", Kind: TypeI32, SlotCount: 1, Public: true},
		"island":        {Name: "island", Kind: TypeIsland, SlotCount: 1, Public: true},
		"cap.io":        {Name: "cap.io", Kind: TypeCap, SlotCount: 1, Public: true},
		"cap.mem":       {Name: "cap.mem", Kind: TypeCap, SlotCount: 1, Public: true},
		"consent.token": {Name: "consent.token", Kind: TypeCap, SlotCount: 1, Public: true},
		"secret.i32":    {Name: "secret.i32", Kind: TypeStruct, SlotCount: 1, Public: true},
	}
	types["i32"].Public = true
	types["task.i32"] = makeStructTypeInfo("task.i32", []FieldInfo{
		{Name: "value", TypeName: "i32"},
		{Name: "error", TypeName: "task.error"},
	})
	types["task.result_i32"] = makeStructTypeInfo("task.result_i32", []FieldInfo{
		{Name: "value", TypeName: "i32"},
		{Name: "error", TypeName: "task.error"},
	})
	types["actor.msg"] = makeStructTypeInfo("actor.msg", []FieldInfo{
		{Name: "value", TypeName: "i32"},
		{Name: "tag", TypeName: "i32"},
	})
	types["actor.recv_result_i32"] = makeStructTypeInfo("actor.recv_result_i32", []FieldInfo{
		{Name: "value", TypeName: "i32"},
		{Name: "error", TypeName: "task.error"},
	})
	types["actor.recv_msg_result"] = makeStructTypeInfo("actor.recv_msg_result", []FieldInfo{
		{Name: "value", TypeName: "i32"},
		{Name: "tag", TypeName: "i32"},
		{Name: "error", TypeName: "task.error"},
	})
	for _, name := range []string{
		"task.i32",
		"task.result_i32",
		"actor.msg",
		"actor.recv_result_i32",
		"actor.recv_msg_result",
	} {
		types[name].Repr = frontend.StructReprC
	}
	return types
}

func addILP32NativeScalarTypes(types map[string]*TypeInfo) {
	for _, name := range []string{
		"usize",
		"isize",
		"size_t",
		"ssize_t",
		"native_int",
		"native_uint",
		"c_long",
		"c_ulong",
	} {
		types[name] = &TypeInfo{Name: name, Kind: TypeI32, SlotCount: 1, Public: true}
	}
	types["rawptr"] = &TypeInfo{Name: "rawptr", Kind: TypePtr, SlotCount: 1, Public: true}
	types["nullable_ptr"] = &TypeInfo{Name: "nullable_ptr", Kind: TypePtr, SlotCount: 1, Public: true}
	types["ref"] = &TypeInfo{Name: "ref", Kind: TypePtr, SlotCount: 1, Public: true}
}

func IsILP32NativeScalarType(name string) bool {
	switch strings.TrimSpace(name) {
	case "usize", "isize", "size_t", "ssize_t", "native_int", "native_uint", "c_long", "c_ulong":
		return true
	default:
		return false
	}
}

func IsILP32UnsignedNativeScalarType(name string) bool {
	switch strings.TrimSpace(name) {
	case "usize", "size_t", "native_uint", "c_ulong":
		return true
	default:
		return false
	}
}

func TypedTaskHandleTypeName(errorType string, types map[string]*TypeInfo) string {
	if info, ok := types[errorType]; ok && info.SlotCount == 1 {
		return "task.i32"
	}
	return "task.i32.throws." + errorType
}

func IsTypedTaskHandleTypeName(typeName string) bool {
	return strings.HasPrefix(typeName, "task.i32.throws.")
}

func TypedTaskHandleTypesCompatible(expected, actual string) bool {
	if expected == "task.i32" && IsTypedTaskHandleTypeName(actual) {
		return true
	}
	if IsTypedTaskHandleTypeName(expected) && actual == "task.i32" {
		return true
	}
	return false
}

func EnsureTypedTaskHandleType(errorType string, types map[string]*TypeInfo) (string, *TypeInfo, error) {
	errorInfo, ok := types[errorType]
	if !ok {
		return "", nil, fmt.Errorf("unknown type '%s'", errorType)
	}
	if errorInfo.Kind != TypeEnum {
		return "", nil, fmt.Errorf("typed task error argument must be an enum")
	}
	if errorInfo.SlotCount == 1 {
		info, ok := types["task.i32"]
		if !ok {
			return "", nil, fmt.Errorf("unknown type 'task.i32'")
		}
		return "task.i32", info, nil
	}
	handleSlots := errorInfo.SlotCount + 2
	if handleSlots > 8 {
		return "", nil, fmt.Errorf("typed task supports at most 8 slots, got %d for error type '%s'", handleSlots, errorType)
	}
	name := TypedTaskHandleTypeName(errorType, types)
	if info, ok := types[name]; ok {
		return name, info, nil
	}
	info := makeStructTypeInfo(name, []FieldInfo{
		{Name: "value", TypeName: "i32"},
		{Name: "error", TypeName: errorType, SlotCount: errorInfo.SlotCount},
		{Name: "status", TypeName: "task.error"},
	})
	info.Public = true
	types[name] = info
	return name, info, nil
}

func ensureTypeInfo(name string, types map[string]*TypeInfo) (*TypeInfo, error) {
	if info, ok := types[name]; ok {
		return info, nil
	}
	if elem, ok := optionalElemName(name); ok {
		elemInfo, err := ensureTypeInfo(elem, types)
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
		elemInfo, ok := types[elem]
		if !ok || !isSupportedCollectionElemType(elemInfo) {
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
		elemInfo, ok := types[elem]
		if !ok || !isSupportedCollectionElemType(elemInfo) {
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
	return nil, fmt.Errorf("unknown type '%s'", name)
}

func targetLayoutOnlyScalarError(name string) error {
	return fmt.Errorf("target-layout scalar type '%s' is not supported in source-level Tetra yet; it is reserved for compiler target layout/ABI classifiers until native-int/codegen support is implemented", name)
}

func isTargetLayoutOnlyScalar(name string) bool {
	switch strings.TrimSpace(name) {
	case "i8", "i16", "u32", "u64", "uint",
		"usize", "isize", "size_t", "ssize_t", "native_int", "native_uint",
		"c_long", "c_ulong", "f32", "f64", "ref", "nullable_ptr", "rawptr":
		return true
	default:
		return false
	}
}

func typesCompatible(expected, actual string) bool {
	if expected == actual {
		return true
	}
	if TypedTaskHandleTypesCompatible(expected, actual) {
		return true
	}
	if expected == "none" || actual == "none" {
		if _, ok := optionalElemName(expected); ok && actual == "none" {
			return true
		}
		if _, ok := optionalElemName(actual); ok && expected == "none" {
			return true
		}
		return false
	}
	if elem, ok := optionalElemName(expected); ok && typesCompatible(elem, actual) {
		return true
	}
	if isInt32Like(expected) && isInt32Like(actual) {
		return true
	}
	return false
}

func typesCompatibleWithNullPtr(expected, actual string, expr frontend.Expr) bool {
	if !smallIntLiteralFits(expected, actual, expr) {
		return false
	}
	if typesCompatible(expected, actual) {
		return true
	}
	if expected == "ptr" && actual == "fnptr" {
		_, ok := expr.(*frontend.ClosureExpr)
		return ok
	}
	if isNullablePointerScalarType(expected) && actual == "i32" && isNullPtrLiteral(expr) {
		return true
	}
	return false
}

func isNullablePointerScalarType(name string) bool {
	switch strings.TrimSpace(name) {
	case "ptr", "rawptr", "nullable_ptr":
		return true
	default:
		return false
	}
}

func smallIntLiteralFits(expected, actual string, expr frontend.Expr) bool {
	if actual != "i32" {
		return true
	}
	rangeType := expected
	for {
		elem, ok := optionalElemName(rangeType)
		if !ok {
			break
		}
		rangeType = elem
	}
	if rangeType != "u8" && rangeType != "u16" && rangeType != "c_uint" && !IsILP32UnsignedNativeScalarType(rangeType) {
		return true
	}
	v, ok, overflow := evalConstI32(expr)
	if !ok {
		return true
	}
	if overflow {
		return false
	}
	switch rangeType {
	case "u8":
		return v >= 0 && v <= 255
	case "u16":
		return v >= 0 && v <= 65535
	case "c_uint":
		return v >= 0
	case "usize", "size_t", "native_uint", "c_ulong":
		return v >= 0
	default:
		return true
	}
}

func isNullPtrLiteral(expr frontend.Expr) bool {
	n, ok := expr.(*frontend.NumberExpr)
	return ok && n.Value == 0
}

func constI32(expr frontend.Expr) (int32, bool) {
	v, ok, overflow := evalConstI32(expr)
	if !ok || overflow {
		return 0, false
	}
	return int32(v), true
}

const (
	minConstI32 int64 = -1 << 31
	maxConstI32 int64 = 1<<31 - 1
)

func evalConstI32(expr frontend.Expr) (int64, bool, bool) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return int64(e.Value), true, false
	case *frontend.UnaryExpr:
		if e.Op != frontend.TokenMinus {
			return 0, false, false
		}
		v, ok, overflow := evalConstI32(e.X)
		if !ok || overflow {
			return 0, ok, overflow
		}
		return checkedConstI32(-v)
	case *frontend.BinaryExpr:
		left, ok, overflow := evalConstI32(e.Left)
		if !ok || overflow {
			return 0, ok, overflow
		}
		right, ok, overflow := evalConstI32(e.Right)
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

func checkedConstI32(v int64) (int64, bool, bool) {
	if v < minConstI32 || v > maxConstI32 {
		return 0, true, true
	}
	return v, true, false
}

func isInt32Like(name string) bool {
	return name == "i32" || name == "u8" || name == "u16" || name == "c_int" || name == "c_uint" || name == "task.error" || IsILP32NativeScalarType(name)
}

func isConditionType(name string) bool {
	return name == "bool" || isInt32Like(name)
}

func isReservedTypeName(name string) bool {
	switch name {
	case "i32", "i64", "Int64", "u8", "u16", "c_int", "c_uint", "bool", "Bool", "ptr", "fnptr", "rawptr", "nullable_ptr", "ref", "str", "String",
		"usize", "isize", "size_t", "ssize_t", "native_int", "native_uint", "c_long", "c_ulong",
		"actor", "actor.msg", "actor.recv_result_i32", "actor.recv_msg_result",
		"task.error", "task.group", "task.i32", "task.result_i32",
		"island", "cap.io", "cap.mem", "consent.token", "secret.i32":
		return true
	default:
		return false
	}
}

func optionalElemName(name string) (string, bool) {
	if strings.HasSuffix(name, "?") {
		return strings.TrimSuffix(name, "?"), true
	}
	return "", false
}

func optionalTypeName(elem string) string {
	return elem + "?"
}

func isPrintableType(name string, types map[string]*TypeInfo) bool {
	info, err := ensureTypeInfo(name, types)
	if err != nil {
		return false
	}
	if info.Kind == TypeStr {
		return true
	}
	if info.Kind == TypeSlice && info.ElemType == "u8" {
		return true
	}
	return false
}

func sliceElemName(name string) (string, bool) {
	if strings.HasPrefix(name, "[]") {
		return name[2:], true
	}
	return "", false
}

func isArrayTypeName(name string) bool {
	return strings.HasPrefix(name, "[") && strings.Contains(name, "]")
}

func parseArrayTypeName(name string) (int, string, bool) {
	if !strings.HasPrefix(name, "[") {
		return 0, "", false
	}
	end := strings.Index(name, "]")
	if end <= 1 || end+1 > len(name) {
		return 0, "", false
	}
	n, err := strconv.Atoi(name[1:end])
	if err != nil {
		return 0, "", false
	}
	elem := name[end+1:]
	if elem == "" {
		return 0, "", false
	}
	return n, elem, true
}

func isSupportedArrayElemType(name string, types map[string]*TypeInfo) bool {
	info, ok := types[name]
	if !ok {
		return false
	}
	return isSupportedCollectionElemType(info)
}

func isSupportedCollectionElemType(info *TypeInfo) bool {
	if info == nil {
		return false
	}
	switch info.Name {
	case "i32", "u8", "u16", "c_int", "c_uint", "bool",
		"usize", "isize", "size_t", "ssize_t", "native_int", "native_uint", "c_long", "c_ulong":
		return true
	}
	return info.Kind == TypeStruct && info.Name != "secret.i32" && info.SlotCount == 1
}

func isSupportedActorStateScalarType(name string) bool {
	return name == "i32" || name == "bool" || name == "u8" || name == "u16" || name == "c_int" || name == "c_uint" || name == "task.error" || IsILP32NativeScalarType(name)
}

func funcSigActorTaskTransferSafe(sig FuncSig, types map[string]*TypeInfo) bool {
	return funcSigActorTaskTransferUnsafeReason(sig, types) == ""
}

func funcSigActorTaskTransferUnsafeReason(sig FuncSig, types map[string]*TypeInfo) string {
	for i, typeName := range sig.ParamTypes {
		ownership := ""
		if i < len(sig.ParamOwnership) {
			ownership = sig.ParamOwnership[i]
		}
		if ownership == "borrow" || ownership == "inout" {
			return fmt.Sprintf("parameter %d uses %s ownership", i+1, ownership)
		}
		if !typeActorTaskSendable(typeName, types, map[string]bool{}) {
			return fmt.Sprintf("parameter %d type '%s' is not sendable", i+1, typeName)
		}
	}
	if sig.ReturnType == "" {
		return ""
	}
	if !typeActorTaskSendable(sig.ReturnType, types, map[string]bool{}) {
		return fmt.Sprintf("return type '%s' is not sendable", sig.ReturnType)
	}
	return ""
}

func typeActorTaskSendable(typeName string, types map[string]*TypeInfo, seen map[string]bool) bool {
	if _, ok := surfaceActorTaskBoundaryValueType(typeName, types); ok {
		return false
	}
	if seen[typeName] {
		return true
	}
	seen[typeName] = true
	info, ok := types[typeName]
	if !ok {
		return false
	}
	switch info.Kind {
	case TypeI32, TypeI64, TypeU8, TypeBool, TypeActor:
		return true
	case TypeEnum:
		for _, c := range info.EnumCases {
			for _, payload := range c.PayloadTypes {
				if !typeActorTaskSendable(payload, types, seen) {
					return false
				}
			}
		}
		return true
	case TypeOptional:
		return typeActorTaskSendable(info.ElemType, types, seen)
	case TypeStruct:
		for _, field := range info.Fields {
			if !typeActorTaskSendable(field.TypeName, types, seen) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func typeActorTaskSendabilityUnsafeReason(typeName string, types map[string]*TypeInfo, seen map[string]bool) string {
	if surfaceType, ok := surfaceActorTaskBoundaryValueType(typeName, types); ok {
		return fmt.Sprintf("surface value '%s' cannot cross actor/task boundary", surfaceType)
	}
	if seen[typeName] {
		return ""
	}
	seen[typeName] = true
	info, ok := types[typeName]
	if !ok {
		return fmt.Sprintf("unknown type '%s'", typeName)
	}
	switch info.Kind {
	case TypeI32, TypeI64, TypeU8, TypeBool, TypeActor:
		return ""
	case TypeEnum:
		for _, c := range info.EnumCases {
			for _, payload := range c.PayloadTypes {
				if reason := typeActorTaskSendabilityUnsafeReason(payload, types, seen); reason != "" {
					return reason
				}
			}
		}
		return ""
	case TypeOptional:
		return typeActorTaskSendabilityUnsafeReason(info.ElemType, types, seen)
	case TypeStruct:
		for _, field := range info.Fields {
			if reason := typeActorTaskSendabilityUnsafeReason(field.TypeName, types, seen); reason != "" {
				return reason
			}
		}
		return ""
	default:
		return fmt.Sprintf("type '%s' is not sendable", typeName)
	}
}
