package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

type CheckedProgram struct {
	Funcs              []CheckedFunc
	Enums              []CheckedEnum
	Structs            []CheckedStruct
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
	LocalSlots  int
	ParamSlots  int
	ReturnType  string
	ThrowsType  string
	Async       bool
	ReturnSlots int
}

type LocalInfo struct {
	Base      int
	SlotCount int
	TypeName  string
	Mutable   bool
	Const     bool
}

type GlobalInfo struct {
	DataIndex int
	TypeName  string
	Mutable   bool
	Const     bool
}

type FuncSig struct {
	Generic           bool
	ParamNames        []string
	ParamTypes        []string
	ParamOwnership    []string
	ParamSlots        int
	ReturnType        string
	ThrowsType        string
	Async             bool
	ReturnSlots       int
	ReturnRegionParam int
	Effects           []string
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
	Fields    []FieldInfo
	FieldMap  map[string]FieldInfo
	SlotCount int
	ElemType  string
	ArrayLen  int
	EnumCases []EnumCaseInfo
	CaseMap   map[string]EnumCaseInfo
}

type EnumCaseInfo struct {
	Name    string
	Ordinal int32
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
		Fields:    fields,
		FieldMap:  fieldMap,
		SlotCount: 2,
		ElemType:  elem,
	}
}

func makeStrTypeInfo() *TypeInfo {
	info := makeSliceTypeInfo("str", "u8")
	info.Kind = TypeStr
	return info
}

func baseTypes() map[string]*TypeInfo {
	return map[string]*TypeInfo{
		"i32":     {Name: "i32", Kind: TypeI32, SlotCount: 1},
		"u8":      {Name: "u8", Kind: TypeU8, SlotCount: 1},
		"bool":    {Name: "bool", Kind: TypeBool, SlotCount: 1},
		"ptr":     {Name: "ptr", Kind: TypePtr, SlotCount: 1},
		"str":     makeStrTypeInfo(),
		"actor":   {Name: "actor", Kind: TypeActor, SlotCount: 1},
		"island":  {Name: "island", Kind: TypeIsland, SlotCount: 1},
		"cap.io":  {Name: "cap.io", Kind: TypeCap, SlotCount: 1},
		"cap.mem": {Name: "cap.mem", Kind: TypeCap, SlotCount: 1},
	}
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
	if expected == "u8" && actual == "i32" {
		return true
	}
	if expected == "i32" && actual == "u8" {
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
	return name == "i32" || name == "u8"
}

func isConditionType(name string) bool {
	return name == "bool" || isInt32Like(name)
}

func isReservedTypeName(name string) bool {
	switch name {
	case "i32", "u8", "bool", "Bool", "ptr", "str", "String", "actor", "island", "cap.io", "cap.mem":
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
