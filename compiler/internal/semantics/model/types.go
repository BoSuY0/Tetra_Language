package model

import "tetra_language/compiler/internal/frontend"

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
	Name                  string
	Module                string
	Decl                  *frontend.FuncDecl
	Imports               map[string]string
	Locals                map[string]LocalInfo
	ActorState            map[string]ActorStateField
	LocalSlots            int
	ParamSlots            int
	ReturnType            string
	ReturnOwnership       string
	ThrowsType            string
	Async                 bool
	ReturnSlots           int
	Effects               []string
	TouchesMutableGlobals bool
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
	Base                          int
	SlotCount                     int
	TypeName                      string
	Mutable                       bool
	Const                         bool
	ActorField                    bool
	ActorFieldSlot                int
	ActorFieldInit                int32
	FunctionValue                 string
	FunctionParamName             string
	GenericFunctionValue          bool
	FunctionCaptures              []frontend.ClosureCapture
	FunctionEscapeCaptures        []frontend.ClosureCapture
	FunctionTouchesMutableGlobals bool
	FunctionReturnSnapshotAlias   bool
	FunctionDirectSnapshotAlias   bool
	FunctionEscapeKind            CallableEscapeKind
	FunctionHandleValue           bool
	FunctionEnumPayload           bool
	FunctionTypeValue             bool
	FunctionParamTypes            []string
	FunctionParamOwnership        []string
	FunctionReturnType            string
	FunctionReturnOwnership       string
	FunctionThrowsType            string
	FunctionEffects               []string
	FunctionFields                map[string]FunctionFieldInfo
	EnumPayloadFunctions          map[string]FunctionFieldInfo
	EnumPayloadFields             map[string]FunctionFieldInfo
	SurfaceFramePixelsSource      string
}

type FunctionFieldInfo struct {
	FunctionValue                 string
	FunctionParamName             string
	FunctionCaptures              []frontend.ClosureCapture
	FunctionEscapeCaptures        []frontend.ClosureCapture
	FunctionTouchesMutableGlobals bool
	FunctionReturnSnapshotAlias   bool
	FunctionDirectSnapshotAlias   bool
	FunctionEscapeKind            CallableEscapeKind
	FunctionHandleValue           bool
	FunctionParamTypes            []string
	FunctionParamOwnership        []string
	FunctionReturnType            string
	FunctionReturnOwnership       string
	FunctionThrowsType            string
	FunctionEffects               []string
}

const (
	FnPtrEnvSlotCount       = 8
	FnPtrSlotCount          = 1 + FnPtrEnvSlotCount
	CallableHandleSlotCount = 4
)

type CallableEscapeKind string

const (
	CallableEscapeLocalSnapshot CallableEscapeKind = "local-snapshot"
	CallableEscapeHeap          CallableEscapeKind = "heap"
	CallableEscapeGlobal        CallableEscapeKind = "global"
	CallableEscapeThread        CallableEscapeKind = "thread"
)

type GlobalInfo struct {
	DataIndex               int
	TypeName                string
	Mutable                 bool
	Const                   bool
	Public                  bool
	FunctionValue           string
	FunctionTypeValue       bool
	FunctionParamTypes      []string
	FunctionParamOwnership  []string
	FunctionReturnType      string
	FunctionReturnOwnership string
	FunctionThrowsType      string
	FunctionEffects         []string
	FunctionEscapeKind      CallableEscapeKind
	FunctionHandleValue     bool
	HasStringLiteralInit    bool
	StringLiteralInit       []byte
	ArrayBackings           []GlobalArrayBackingInfo
}

type GlobalArrayBackingInfo struct {
	HeaderOffset int
	ElemType     string
	Len          int
}

type FuncSig struct {
	Generic                             bool
	Public                              bool
	HasNoAlloc                          bool
	HasNoBlock                          bool
	HasRealtime                         bool
	HasBudget                           bool
	Budget                              int32
	ParamNames                          []string
	ParamTypes                          []string
	ParamFunctionTypes                  []bool
	ParamFunctionParams                 [][]string
	ParamFunctionOwnership              [][]string
	ParamFunctionReturns                []string
	ParamFunctionReturnOwnership        []string
	ParamFunctionThrows                 []string
	ParamFunctionEffects                [][]string
	ParamOwnership                      []string
	ParamSlots                          int
	ReturnType                          string
	ReturnOwnership                     string
	ReturnFunctionType                  bool
	ReturnFunctionParams                []string
	ReturnFunctionParamOwnership        []string
	ReturnFunctionReturn                string
	ReturnFunctionReturnOwnership       string
	ReturnFunctionThrows                string
	ReturnFunctionEffects               []string
	ReturnFunctionSymbol                string
	ReturnFunctionParamName             string
	ReturnFunctionCaptures              []frontend.ClosureCapture
	ReturnFunctionTouchesMutableGlobals bool
	ReturnFunctionEscapeKind            CallableEscapeKind
	ReturnFunctionHandleValue           bool
	ReturnFunctionFields                map[string]FunctionFieldInfo
	ReturnEnumPayloadFunctions          map[string]FunctionFieldInfo
	ReturnEnumPayloadFields             map[string]FunctionFieldInfo
	ThrowsType                          string
	Async                               bool
	ReturnSlots                         int
	ReturnRegionParam                   int
	ReturnRegionSummary                 ReturnRegionSummary
	ReturnResourceParam                 int
	ReturnResourcePath                  string
	ReturnResourceSummary               ReturnResourceSummary
	ThrowResourceSummary                ReturnResourceSummary
	Effects                             []string
	TouchesMutableGlobals               bool
}

type ResourceProvenance struct {
	ParamIndex int
	ParamPath  string
}

type ReturnRegionSummary map[string]int

type ReturnResourceSummary map[string][]ResourceProvenance

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
	TypeI64
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

const MaxActorStateSlots = 8

type FieldInfo struct {
	Name                    string
	TypeName                string
	Offset                  int
	SlotCount               int
	UserAssignable          bool
	FunctionTypeValue       bool
	FunctionTypeRef         frontend.TypeRef
	FunctionParamTypes      []string
	FunctionParamOwnership  []string
	FunctionReturnType      string
	FunctionReturnOwnership string
	FunctionThrowsType      string
	FunctionEffects         []string
}

type TypeInfo struct {
	Name              string
	Kind              TypeKind
	Public            bool
	Repr              string
	Fields            []FieldInfo
	FieldMap          map[string]FieldInfo
	SlotCount         int
	ElemType          string
	ArrayLen          int
	EnumCases         []EnumCaseInfo
	CaseMap           map[string]EnumCaseInfo
	RuntimeOwned      bool
	UserConstructible bool
	UserAssignable    bool
	ActorSendable     bool
}

type EnumCaseInfo struct {
	Name                      string
	Ordinal                   int32
	PayloadTypes              []string
	PayloadSlots              []int
	PayloadFunctionTypes      []bool
	PayloadFunctionRefs       []frontend.TypeRef
	PayloadFunctionParams     [][]string
	PayloadFunctionOwns       [][]string
	PayloadFunctionReturns    []string
	PayloadFunctionReturnOwns []string
	PayloadFunctionThrows     []string
	PayloadFunctionEffects    [][]string
	SlotCount                 int
}
