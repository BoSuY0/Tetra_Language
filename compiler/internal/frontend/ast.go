package frontend

type Position struct {
	File string
	Line int
	Col  int
}

type FileAST struct {
	Path    string
	Src     []byte
	Module  string
	Imports []ImportDecl
	Structs []*StructDecl
	Globals []*GlobalDecl
	Funcs   []*FuncDecl
}

type ImportDecl struct {
	At    Position
	Path  string
	Alias string
}

type Program struct {
	Structs []*StructDecl
	Funcs   []*FuncDecl
}

type GlobalDecl struct {
	At      Position
	Name    string
	Type    TypeRef
	Mutable bool
	Init    Expr
}

type FuncDecl struct {
	Pos        Position
	Name       string
	ExportName string
	ReturnType TypeRef
	Params     []ParamDecl
	Body       []Stmt
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
	At   Position
	Name string
	Type TypeRef
}

type TypeRefKind int

const (
	TypeRefNamed TypeRefKind = iota
	TypeRefSlice
	TypeRefArray
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

type LetStmt struct {
	At      Position
	Name    string
	Type    TypeRef
	Mutable bool
	Value   Expr
}

func (s *LetStmt) stmtNode() {}
func (s *LetStmt) Pos() Position {
	return s.At
}

type AssignStmt struct {
	At     Position
	Target Expr
	Value  Expr
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

type WhileStmt struct {
	At   Position
	Cond Expr
	Body []Stmt
}

func (s *WhileStmt) stmtNode() {}
func (s *WhileStmt) Pos() Position {
	return s.At
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

type NumberExpr struct {
	At    Position
	Value int32
}

func (e *NumberExpr) exprNode() {}
func (e *NumberExpr) Pos() Position {
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

type CallExpr struct {
	At   Position
	Name string
	Args []Expr
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
	At    Position
	Base  Expr
	Field string
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
