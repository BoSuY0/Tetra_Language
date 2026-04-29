package semantics

import (
	"fmt"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

type CheckedProgram struct {
	Funcs              []CheckedFunc
	Enums              []CheckedEnum
	Structs            []CheckedStruct
	UIStates           []CheckedUIState
	UIViews            []CheckedUIView
	Protocols          []CheckedProtocol
	FuncSigs           map[string]FuncSig
	Types              map[string]*TypeInfo
	GlobalsByModule    map[string]map[string]GlobalInfo
	GlobalDataByModule map[string][][]byte
	MainIndex          int
	MainName           string
}

type CheckedFunc struct {
	Name        string
	Module      string
	Decl        *frontend.FuncDecl
	Locals      map[string]LocalInfo
	ActorState  map[string]ActorStateField
	LocalSlots  int
	ParamSlots  int
	ReturnType  string
	ThrowsType  string
	Async       bool
	ReturnSlots int
}

type ActorStateField struct {
	Name     string
	Slot     int
	TypeName string
	Mutable  bool
	Const    bool
	Init     int32
}

type LocalInfo struct {
	Base                 int
	SlotCount            int
	TypeName             string
	Mutable              bool
	Const                bool
	ActorField           bool
	ActorFieldSlot       int
	ActorFieldInit       int32
	FunctionValue        string
	GenericFunctionValue bool
	FunctionCaptures     []frontend.ClosureCapture
	FunctionTypeValue    bool
	FunctionParamTypes   []string
	FunctionReturnType   string
}

type GlobalInfo struct {
	DataIndex            int
	TypeName             string
	Mutable              bool
	Const                bool
	HasStringLiteralInit bool
	StringLiteralInit    []byte
}

type FuncSig struct {
	Generic               bool
	Public                bool
	HasNoAlloc            bool
	HasNoBlock            bool
	HasRealtime           bool
	ParamNames            []string
	ParamTypes            []string
	ParamFunctionTypes    []bool
	ParamFunctionParams   [][]string
	ParamFunctionReturns  []string
	ParamOwnership        []string
	ParamSlots            int
	ReturnType            string
	ReturnFunctionType    bool
	ReturnFunctionParams  []string
	ReturnFunctionReturn  string
	ReturnFunctionSymbol  string
	ThrowsType            string
	Async                 bool
	ReturnSlots           int
	ReturnRegionParam     int
	ReturnResourceParam   int
	ReturnResourcePath    string
	Effects               []string
	TouchesMutableGlobals bool
}

type CheckedStruct struct {
	Name   string
	Module string
	Decl   *frontend.StructDecl
}

type CheckedEnum struct {
	Name   string
	Module string
	Decl   *frontend.EnumDecl
}

type CheckedProtocol struct {
	Name   string
	Module string
	Decl   *frontend.ProtocolDecl
}

type CheckedUIState struct {
	Name   string
	Module string
	Decl   *frontend.StateDecl
}

type CheckedUIView struct {
	Name   string
	Module string
	Decl   *frontend.ViewDecl
}

type TypeKind int

const (
	TypeI32 TypeKind = iota
	TypeU8
	TypeBool
	TypePtr
	TypeSlice
	TypeStr
	TypeStruct
	TypeArray
	TypeIsland
	TypeCap
	TypeActor
	TypeEnum
	TypeOptional
)

type FieldInfo struct {
	Name      string
	TypeName  string
	Offset    int
	SlotCount int
}

type TypeInfo struct {
	Name      string
	Kind      TypeKind
	Public    bool
	Fields    []FieldInfo
	FieldMap  map[string]FieldInfo
	SlotCount int
	ElemType  string
	ArrayLen  int
	EnumCases []EnumCaseInfo
	CaseMap   map[string]EnumCaseInfo
}

type EnumCaseInfo struct {
	Name         string
	Ordinal      int32
	PayloadTypes []string
	PayloadSlots []int
	SlotCount    int
}

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
			Name:      field.Name,
			TypeName:  field.TypeName,
			Offset:    offset,
			SlotCount: slotCount,
		}
		offset += slotCount
		structFields = append(structFields, resolved)
		fieldMap[resolved.Name] = resolved
	}
	return &TypeInfo{
		Name:      name,
		Kind:      TypeStruct,
		Public:    true,
		Fields:    structFields,
		FieldMap:  fieldMap,
		SlotCount: offset,
	}
}

func baseTypes() map[string]*TypeInfo {
	types := map[string]*TypeInfo{
		"i32":           {Name: "i32", Kind: TypeI32, SlotCount: 1},
		"u8":            {Name: "u8", Kind: TypeU8, SlotCount: 1, Public: true},
		"u16":           {Name: "u16", Kind: TypeU8, SlotCount: 1, Public: true},
		"bool":          {Name: "bool", Kind: TypeBool, SlotCount: 1, Public: true},
		"ptr":           {Name: "ptr", Kind: TypePtr, SlotCount: 1, Public: true},
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
	return types
}

func TypedTaskHandleTypeName(errorType string, types map[string]*TypeInfo) string {
	if info, ok := types[errorType]; ok && info.SlotCount == 1 {
		return "task.i32"
	}
	return "task.i32.throws." + errorType
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
		if elem != "i32" && elem != "u8" && elem != "u16" && elem != "bool" {
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
		if !isSupportedArrayElemType(elem) {
			return nil, fmt.Errorf("array element type '%s' is not supported", elem)
		}
		info := makeArrayTypeInfo(name, elem, n)
		types[name] = info
		return info, nil
	}
	if isArrayTypeName(name) {
		return nil, fmt.Errorf("invalid array type '%s'", name)
	}
	return nil, fmt.Errorf("unknown type '%s'", name)
}

func typesCompatible(expected, actual string) bool {
	if expected == actual {
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
	if typesCompatible(expected, actual) {
		return true
	}
	if expected == "ptr" && actual == "i32" && isNullPtrLiteral(expr) {
		return true
	}
	return false
}

func isNullPtrLiteral(expr frontend.Expr) bool {
	n, ok := expr.(*frontend.NumberExpr)
	return ok && n.Value == 0
}

func constI32(expr frontend.Expr) (int32, bool) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return e.Value, true
	case *frontend.UnaryExpr:
		if e.Op != frontend.TokenMinus {
			return 0, false
		}
		v, ok := e.X.(*frontend.NumberExpr)
		if !ok {
			return 0, false
		}
		return -v.Value, true
	default:
		return 0, false
	}
}

func isInt32Like(name string) bool {
	return name == "i32" || name == "u8" || name == "u16" || name == "task.error"
}

func isConditionType(name string) bool {
	return name == "bool" || isInt32Like(name)
}

func isReservedTypeName(name string) bool {
	switch name {
	case "i32", "u8", "u16", "bool", "Bool", "ptr", "str", "String",
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

func isSupportedArrayElemType(name string) bool {
	return name == "i32" || name == "bool" || name == "u8" || name == "u16"
}

func isSupportedActorStateScalarType(name string) bool {
	return name == "i32" || name == "bool" || name == "u8" || name == "u16" || name == "task.error"
}

func funcSigActorTaskTransferSafe(sig FuncSig, types map[string]*TypeInfo) bool {
	for i, typeName := range sig.ParamTypes {
		ownership := ""
		if i < len(sig.ParamOwnership) {
			ownership = sig.ParamOwnership[i]
		}
		if ownership == "borrow" || ownership == "inout" {
			return false
		}
		if !typeActorTaskSendable(typeName, types, map[string]bool{}) {
			return false
		}
	}
	return typeActorTaskSendable(sig.ReturnType, types, map[string]bool{})
}

func typeActorTaskSendable(typeName string, types map[string]*TypeInfo, seen map[string]bool) bool {
	if seen[typeName] {
		return true
	}
	seen[typeName] = true
	info, ok := types[typeName]
	if !ok {
		return false
	}
	switch info.Kind {
	case TypeI32, TypeU8, TypeBool, TypeActor, TypeEnum:
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
