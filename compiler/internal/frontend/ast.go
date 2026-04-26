package frontend

type Position struct {
	File string
	Line int
	Col  int
}

type FileAST struct {
	Path       string
	Src        []byte
	Module     string
	Imports    []ImportDecl
	Enums      []*EnumDecl
	Structs    []*StructDecl
	Protocols  []*ProtocolDecl
	Extensions []*ExtensionDecl
	Impls      []*ImplDecl
	Globals    []*GlobalDecl
	Funcs      []*FuncDecl
	Tests      []*TestDecl
}

type ImportDecl struct {
	At    Position
	Path  string
	Alias string
}

type Program struct {
	Enums      []*EnumDecl
	Structs    []*StructDecl
	Protocols  []*ProtocolDecl
	Extensions []*ExtensionDecl
	Impls      []*ImplDecl
	Funcs      []*FuncDecl
	Tests      []*TestDecl
}

type GlobalDecl struct {
	At      Position
	Name    string
	Type    TypeRef
	Mutable bool
	Const   bool
	Init    Expr
}

type FuncDecl struct {
	Pos         Position
	Name        string
	ExportName  string
	Async       bool
	ExtensionOf string
	TypeParams  []string
	ReturnType  TypeRef
	Throws      TypeRef
	HasThrows   bool
	Params      []ParamDecl
	Uses        []string
	Body        []Stmt
}

type TestDecl struct {
	At   Position
	Name string
	Body []Stmt
}

type Stmt interface {
	stmtNode()
	Pos() Position
}

type Expr interface {
	exprNode()
	Pos() Position
}

type ParamDecl struct {
	At        Position
	Name      string
	Type      TypeRef
	Ownership string
}

type TypeRefKind int

const (
	TypeRefNamed TypeRefKind = iota
	TypeRefSlice
	TypeRefArray
	TypeRefOptional
)

type TypeRef struct {
	At   Position
	Kind TypeRefKind
	Name string
	Elem *TypeRef
	Len  int
}

type StructDecl struct {
	At     Position
	Name   string
	Fields []FieldDecl
}

type FieldDecl struct {
	At   Position
	Name string
	Type TypeRef
}

type EnumDecl struct {
	At    Position
	Name  string
	Cases []EnumCaseDecl
}

type EnumCaseDecl struct {
	At   Position
	Name string
}

type ExtensionDecl struct {
	At      Position
	Target  TypeRef
	Methods []*FuncDecl
}

type ProtocolDecl struct {
	At           Position
	Name         string
	Requirements []FuncSigDecl
}

type ImplDecl struct {
	At       Position
	Type     TypeRef
	Protocol TypeRef
}

type FuncSigDecl struct {
	At         Position
	Name       string
	Async      bool
	ReturnType TypeRef
	Throws     TypeRef
	HasThrows  bool
	Params     []ParamDecl
}

type PrintStmt struct {
	At    Position
	Value Expr
}

func (s *PrintStmt) stmtNode() {}
func (s *PrintStmt) Pos() Position {
	return s.At
}

type ReturnStmt struct {
	At    Position
	Value Expr
}

func (s *ReturnStmt) stmtNode() {}
func (s *ReturnStmt) Pos() Position {
	return s.At
}

type ThrowStmt struct {
	At    Position
	Value Expr
}

func (s *ThrowStmt) stmtNode() {}
func (s *ThrowStmt) Pos() Position {
	return s.At
}

type LetStmt struct {
	At      Position
	Name    string
	Type    TypeRef
	Mutable bool
	Const   bool
	Value   Expr
}

func (s *LetStmt) stmtNode() {}
func (s *LetStmt) Pos() Position {
	return s.At
}

type AssignStmt struct {
	At            Position
	Target        Expr
	Value         Expr
	Op            TokenType
	CompoundValue Expr
}

func (s *AssignStmt) stmtNode() {}
func (s *AssignStmt) Pos() Position {
	return s.At
}

type IfStmt struct {
	At   Position
	Cond Expr
	Then []Stmt
	Else []Stmt
}

func (s *IfStmt) stmtNode() {}
func (s *IfStmt) Pos() Position {
	return s.At
}

type IfLetStmt struct {
	At         Position
	Name       string
	Value      Expr
	ValueLocal string
	Then       []Stmt
	Else       []Stmt
}

func (s *IfLetStmt) stmtNode() {}
func (s *IfLetStmt) Pos() Position {
	return s.At
}

type WhileStmt struct {
	At   Position
	Cond Expr
	Body []Stmt
}

func (s *WhileStmt) stmtNode() {}
func (s *WhileStmt) Pos() Position {
	return s.At
}

type BreakStmt struct {
	At Position
}

func (s *BreakStmt) stmtNode() {}
func (s *BreakStmt) Pos() Position {
	return s.At
}

type ContinueStmt struct {
	At Position
}

func (s *ContinueStmt) stmtNode() {}
func (s *ContinueStmt) Pos() Position {
	return s.At
}

type ForRangeStmt struct {
	At            Position
	Name          string
	Start         Expr
	End           Expr
	Iterable      Expr
	IterableLocal string
	IndexLocal    string
	EndLocal      string
	Body          []Stmt
}

func (s *ForRangeStmt) stmtNode() {}
func (s *ForRangeStmt) Pos() Position {
	return s.At
}

type MatchStmt struct {
	At             Position
	Value          Expr
	ScrutineeLocal string
	Cases          []MatchCase
}

func (s *MatchStmt) stmtNode() {}
func (s *MatchStmt) Pos() Position {
	return s.At
}

type MatchCase struct {
	At      Position
	Pattern Expr
	Default bool
	Body    []Stmt
}

type FreeStmt struct {
	At       Position
	Value    Expr
	Implicit bool
}

func (s *FreeStmt) stmtNode() {}
func (s *FreeStmt) Pos() Position {
	return s.At
}

type UnsafeStmt struct {
	At   Position
	Body []Stmt
}

func (s *UnsafeStmt) stmtNode() {}
func (s *UnsafeStmt) Pos() Position {
	return s.At
}

type IslandStmt struct {
	At   Position
	Size Expr
	Name string
	Body []Stmt
}

func (s *IslandStmt) stmtNode() {}
func (s *IslandStmt) Pos() Position {
	return s.At
}

type ExprStmt struct {
	At   Position
	Expr Expr
}

type ExpectStmt struct {
	At   Position
	Cond Expr
}

func (s *ExpectStmt) stmtNode() {}
func (s *ExpectStmt) Pos() Position {
	return s.At
}

func (s *ExprStmt) stmtNode() {}
func (s *ExprStmt) Pos() Position {
	return s.At
}

type NumberExpr struct {
	At    Position
	Value int32
}

func (e *NumberExpr) exprNode() {}
func (e *NumberExpr) Pos() Position {
	return e.At
}

type BoolLitExpr struct {
	At    Position
	Value bool
}

func (e *BoolLitExpr) exprNode() {}
func (e *BoolLitExpr) Pos() Position {
	return e.At
}

type NoneLitExpr struct {
	At Position
}

func (e *NoneLitExpr) exprNode() {}
func (e *NoneLitExpr) Pos() Position {
	return e.At
}

type SomePatternExpr struct {
	At   Position
	Name string
}

func (e *SomePatternExpr) exprNode() {}
func (e *SomePatternExpr) Pos() Position {
	return e.At
}

type IdentExpr struct {
	At   Position
	Name string
}

func (e *IdentExpr) exprNode() {}
func (e *IdentExpr) Pos() Position {
	return e.At
}

type BinaryExpr struct {
	At    Position
	Op    TokenType
	Left  Expr
	Right Expr
}

func (e *BinaryExpr) exprNode() {}
func (e *BinaryExpr) Pos() Position {
	return e.At
}

type UnaryExpr struct {
	At Position
	Op TokenType
	X  Expr
}

func (e *UnaryExpr) exprNode() {}
func (e *UnaryExpr) Pos() Position {
	return e.At
}

type TryExpr struct {
	At Position
	X  Expr
}

func (e *TryExpr) exprNode() {}
func (e *TryExpr) Pos() Position {
	return e.At
}

type AwaitExpr struct {
	At Position
	X  Expr
}

func (e *AwaitExpr) exprNode() {}
func (e *AwaitExpr) Pos() Position {
	return e.At
}

type CallExpr struct {
	At        Position
	Name      string
	Args      []Expr
	ArgLabels []string
}

func (e *CallExpr) exprNode() {}
func (e *CallExpr) Pos() Position {
	return e.At
}

type StructLitExpr struct {
	At     Position
	Type   TypeRef
	Fields []StructFieldInit
}

func (e *StructLitExpr) exprNode() {}
func (e *StructLitExpr) Pos() Position {
	return e.At
}

type StructFieldInit struct {
	At    Position
	Name  string
	Value Expr
}

type StringLitExpr struct {
	At    Position
	Value []byte
}

func (e *StringLitExpr) exprNode() {}
func (e *StringLitExpr) Pos() Position {
	return e.At
}

type FieldAccessExpr struct {
	At          Position
	Base        Expr
	Field       string
	EnumType    string
	EnumOrdinal int32
}

func (e *FieldAccessExpr) exprNode() {}
func (e *FieldAccessExpr) Pos() Position {
	return e.At
}

type IndexExpr struct {
	At    Position
	Base  Expr
	Index Expr
}

func (e *IndexExpr) exprNode() {}
func (e *IndexExpr) Pos() Position {
	return e.At
}

func FormatPos(pos Position) string {
	if pos.File != "" {
		return pos.File + ":" + itoa(pos.Line) + ":" + itoa(pos.Col)
	}
	return "line " + itoa(pos.Line) + ":" + itoa(pos.Col)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := false
	if v < 0 {
		neg = true
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
