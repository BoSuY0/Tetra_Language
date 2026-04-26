package compiler

import (
	"bytes"
	"sort"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

// FormatSource returns canonical v0.18 Flow-style formatting for supported MVP syntax.
func FormatSource(src []byte, filename string) ([]byte, error) {
	comments, err := collectLineComments(src, filename)
	if err != nil {
		return nil, err
	}
	file, err := frontend.ParseFile(stripStandaloneBlockComments(src), filename)
	if err != nil {
		return nil, err
	}
	var p sourcePrinter
	p.file(file)
	return applyLineComments([]byte(p.b.String()), comments), nil
}

func stripStandaloneBlockComments(src []byte) []byte {
	lines := strings.Split(string(src), "\n")
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")
		commentAt, block := commentStart(line)
		if commentAt >= 0 && block && strings.TrimSpace(line[:commentAt]) == "" {
			out = append(out, "")
			commentLine := line[commentAt:]
			for strings.Index(commentLine, "*/") < 0 && i+1 < len(lines) {
				i++
				commentLine = strings.TrimRight(lines[i], "\r")
				out = append(out, "")
			}
			continue
		}
		out = append(out, line)
	}
	return []byte(strings.Join(out, "\n"))
}

type lineComments struct {
	before   map[int][]string
	trailing []string
}

func collectLineComments(src []byte, filename string) (lineComments, error) {
	out := lineComments{before: make(map[int][]string)}
	var pending []string
	codeLine := 0
	lines := strings.Split(string(src), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")
		commentAt, block := commentStart(line)
		if commentAt >= 0 {
			if strings.TrimSpace(line[:commentAt]) != "" {
				return lineComments{}, &frontend.DiagnosticError{Info: frontend.Diagnostic{
					Code:     "TETRA_FMT001",
					Message:  "inline comments are not supported by tetra fmt v0.18; move the comment to its own line or format manually",
					File:     filename,
					Line:     i + 1,
					Column:   commentAt + 1,
					Severity: "error",
					Hint:     "Move the comment to its own line before running tetra fmt.",
				}}
			}
			if !block {
				pending = append(pending, strings.TrimSpace(line[commentAt:]))
				continue
			}

			commentLine := line[commentAt:]
			for {
				end := strings.Index(commentLine, "*/")
				if end >= 0 {
					pending = append(pending, strings.TrimSpace(commentLine[:end+2]))
					if strings.TrimSpace(commentLine[end+2:]) != "" {
						col := commentAt + end + 3
						return lineComments{}, &frontend.DiagnosticError{Info: frontend.Diagnostic{
							Code:     "TETRA_FMT001",
							Message:  "inline comments are not supported by tetra fmt v0.18; move the comment to its own line or format manually",
							File:     filename,
							Line:     i + 1,
							Column:   col,
							Severity: "error",
							Hint:     "Move the comment to its own line before running tetra fmt.",
						}}
					}
					break
				}

				pending = append(pending, strings.TrimSpace(commentLine))
				i++
				if i >= len(lines) {
					return lineComments{}, &frontend.DiagnosticError{Info: frontend.Diagnostic{
						Code:     "TETRA_FMT002",
						Message:  "unterminated block comment",
						File:     filename,
						Line:     len(lines),
						Column:   1,
						Severity: "error",
					}}
				}
				commentLine = strings.TrimRight(lines[i], "\r")
				commentAt = 0
			}
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		if len(pending) > 0 {
			out.before[codeLine] = append(out.before[codeLine], pending...)
			pending = nil
		}
		codeLine += formattedCodeLineCount(line)
	}
	out.trailing = pending
	return out, nil
}

func formattedCodeLineCount(line string) int {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "func ") && strings.Contains(trimmed, " uses ") && strings.HasSuffix(trimmed, ":") {
		return 2
	}
	return 1
}

func commentStart(line string) (int, bool) {
	inString := false
	escaped := false
	for i := 0; i+1 < len(line); i++ {
		ch := line[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch ch {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}
		if ch == '"' {
			inString = true
			continue
		}
		if ch == '/' && line[i+1] == '/' {
			return i, false
		}
		if ch == '/' && line[i+1] == '*' {
			return i, true
		}
	}
	return -1, false
}

func applyLineComments(formatted []byte, comments lineComments) []byte {
	if len(comments.before) == 0 && len(comments.trailing) == 0 {
		return formatted
	}
	lines := strings.Split(strings.TrimSuffix(string(formatted), "\n"), "\n")
	var b bytes.Buffer
	codeLine := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			if pending := comments.before[codeLine]; len(pending) > 0 {
				indent := leadingWhitespace(line)
				for _, comment := range pending {
					b.WriteString(indent)
					b.WriteString(comment)
					b.WriteByte('\n')
				}
			}
			codeLine++
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	for _, comment := range comments.trailing {
		b.WriteString(comment)
		b.WriteByte('\n')
	}
	return b.Bytes()
}

func leadingWhitespace(line string) string {
	for i, r := range line {
		if r != ' ' && r != '\t' {
			return line[:i]
		}
	}
	return line
}

type sourcePrinter struct {
	b bytes.Buffer
}

func (p *sourcePrinter) file(file *frontend.FileAST) {
	if file.Module != "" {
		p.line(0, "module "+file.Module)
		p.blank()
	}
	for _, imp := range file.Imports {
		p.line(0, "import "+imp.Path+" as "+imp.Alias)
	}
	if len(file.Imports) > 0 {
		p.blank()
	}
	for _, en := range file.Enums {
		p.enumDecl(en)
		p.blank()
	}
	for _, st := range file.Structs {
		p.structDecl(st)
		p.blank()
	}
	for _, proto := range file.Protocols {
		p.protocolDecl(proto)
		p.blank()
	}
	for _, ext := range file.Extensions {
		p.extensionDecl(ext)
		p.blank()
	}
	for _, impl := range file.Impls {
		p.implDecl(impl)
		p.blank()
	}
	for _, g := range file.Globals {
		p.globalDecl(g)
	}
	if len(file.Globals) > 0 {
		p.blank()
	}
	for _, fn := range file.Funcs {
		if fn.ExtensionOf != "" {
			continue
		}
		p.funcDecl(fn)
		p.blank()
	}
	for _, test := range file.Tests {
		p.testDecl(test)
		p.blank()
	}
	out := p.b.String()
	p.b.Reset()
	p.b.WriteString(strings.TrimRight(out, "\n"))
	p.b.WriteByte('\n')
}

func (p *sourcePrinter) protocolDecl(proto *frontend.ProtocolDecl) {
	p.line(0, "protocol "+proto.Name+":")
	for _, req := range proto.Requirements {
		p.line(1, formatFuncSigDecl(req))
	}
}

func (p *sourcePrinter) enumDecl(en *frontend.EnumDecl) {
	p.line(0, "enum "+en.Name+":")
	for _, c := range en.Cases {
		p.line(1, "case "+c.Name)
	}
}

func (p *sourcePrinter) structDecl(st *frontend.StructDecl) {
	p.line(0, "struct "+st.Name+":")
	for _, f := range st.Fields {
		p.line(1, f.Name+": "+formatTypeRef(f.Type))
	}
}

func (p *sourcePrinter) extensionDecl(ext *frontend.ExtensionDecl) {
	p.line(0, "extension "+formatTypeRef(ext.Target)+":")
	for _, fn := range ext.Methods {
		p.funcDeclWithNameAt(fn, strings.TrimPrefix(fn.Name, fn.ExtensionOf+"."), 1)
	}
}

func (p *sourcePrinter) implDecl(impl *frontend.ImplDecl) {
	p.line(0, "impl "+formatTypeRef(impl.Type)+": "+formatTypeRef(impl.Protocol))
}

func (p *sourcePrinter) globalDecl(g *frontend.GlobalDecl) {
	kw := "val"
	if g.Mutable {
		kw = "var"
	} else if g.Const {
		kw = "const"
	}
	line := kw + " " + g.Name
	if g.Type.Name != "" || g.Type.Elem != nil {
		line += ": " + formatTypeRef(g.Type)
	}
	if g.Init != nil {
		line += " = " + formatExpr(g.Init)
	}
	p.line(0, line)
}

func (p *sourcePrinter) funcDecl(fn *frontend.FuncDecl) {
	p.funcDeclWithName(fn, fn.Name)
}

func (p *sourcePrinter) funcDeclWithName(fn *frontend.FuncDecl, name string) {
	p.funcDeclWithNameAt(fn, name, 0)
}

func (p *sourcePrinter) funcDeclWithNameAt(fn *frontend.FuncDecl, name string, indent int) {
	if fn.ExportName != "" {
		p.line(indent, "@export("+strconv.Quote(fn.ExportName)+")")
	}
	var params []string
	for _, param := range fn.Params {
		typ := formatTypeRef(param.Type)
		if param.Ownership != "" {
			typ = param.Ownership + " " + typ
		}
		params = append(params, param.Name+": "+typ)
	}
	headerPrefix := "func "
	if fn.Async {
		headerPrefix = "async func "
	}
	typeParams := ""
	if len(fn.TypeParams) > 0 {
		typeParams = "<" + strings.Join(fn.TypeParams, ", ") + ">"
	}
	header := headerPrefix + name + typeParams + "(" + strings.Join(params, ", ") + ") -> " + formatTypeRef(fn.ReturnType)
	if fn.HasThrows {
		header += " throws " + formatTypeRef(fn.Throws)
	}
	if len(fn.Uses) > 0 {
		p.line(indent, header)
		uses := append([]string(nil), fn.Uses...)
		sort.Strings(uses)
		p.line(indent, "uses "+strings.Join(uses, ", ")+":")
	} else {
		p.line(indent, header+":")
	}
	p.stmts(fn.Body, indent+1)
}

func formatFuncSigDecl(sig frontend.FuncSigDecl) string {
	var params []string
	for _, param := range sig.Params {
		typ := formatTypeRef(param.Type)
		if param.Ownership != "" {
			typ = param.Ownership + " " + typ
		}
		params = append(params, param.Name+": "+typ)
	}
	prefix := "func "
	if sig.Async {
		prefix = "async func "
	}
	out := prefix + sig.Name + "(" + strings.Join(params, ", ") + ") -> " + formatTypeRef(sig.ReturnType)
	if sig.HasThrows {
		out += " throws " + formatTypeRef(sig.Throws)
	}
	return out
}

func (p *sourcePrinter) testDecl(test *frontend.TestDecl) {
	p.line(0, "test "+strconv.Quote(test.Name)+":")
	p.stmts(test.Body, 1)
}

func (p *sourcePrinter) stmts(stmts []frontend.Stmt, indent int) {
	for _, stmt := range stmts {
		p.stmt(stmt, indent)
	}
}

func (p *sourcePrinter) ifStmt(s *frontend.IfStmt, indent int, prefix string) {
	p.line(indent, prefix+" "+formatExpr(s.Cond)+":")
	p.stmts(s.Then, indent+1)
	p.elseStmts(s.Else, indent)
}

func (p *sourcePrinter) ifLetStmt(s *frontend.IfLetStmt, indent int, prefix string) {
	p.line(indent, prefix+" let "+s.Name+" = "+formatExpr(s.Value)+":")
	p.stmts(s.Then, indent+1)
	p.elseStmts(s.Else, indent)
}

func (p *sourcePrinter) elseStmts(stmts []frontend.Stmt, indent int) {
	if len(stmts) == 0 {
		return
	}
	if len(stmts) == 1 {
		switch nested := stmts[0].(type) {
		case *frontend.IfStmt:
			p.ifStmt(nested, indent, "else if")
			return
		case *frontend.IfLetStmt:
			p.ifLetStmt(nested, indent, "else if")
			return
		}
	}
	p.line(indent, "else:")
	p.stmts(stmts, indent+1)
}

func (p *sourcePrinter) stmt(stmt frontend.Stmt, indent int) {
	switch s := stmt.(type) {
	case *frontend.PrintStmt:
		p.line(indent, "print("+formatExpr(s.Value)+")")
	case *frontend.ExpectStmt:
		p.line(indent, "expect "+formatExpr(s.Cond))
	case *frontend.ReturnStmt:
		p.line(indent, "return "+formatExpr(s.Value))
	case *frontend.ThrowStmt:
		p.line(indent, "throw "+formatExpr(s.Value))
	case *frontend.BreakStmt:
		p.line(indent, "break")
	case *frontend.ContinueStmt:
		p.line(indent, "continue")
	case *frontend.LetStmt:
		kw := "let"
		if s.Mutable {
			kw = "var"
		} else if s.Const {
			kw = "const"
		}
		line := kw + " " + s.Name
		if s.Type.Name != "" || s.Type.Elem != nil {
			line += ": " + formatTypeRef(s.Type)
		}
		line += " = " + formatExpr(s.Value)
		p.line(indent, line)
	case *frontend.AssignStmt:
		if s.Op != 0 && s.CompoundValue != nil {
			p.line(indent, formatExpr(s.Target)+" "+compoundAssignmentOp(s.Op)+"= "+formatExpr(s.CompoundValue))
			return
		}
		p.line(indent, formatExpr(s.Target)+" = "+formatExpr(s.Value))
	case *frontend.IfStmt:
		p.ifStmt(s, indent, "if")
	case *frontend.IfLetStmt:
		p.ifLetStmt(s, indent, "if")
	case *frontend.WhileStmt:
		p.line(indent, "while "+formatExpr(s.Cond)+":")
		p.stmts(s.Body, indent+1)
	case *frontend.ForRangeStmt:
		if s.Iterable != nil {
			p.line(indent, "for "+s.Name+" in "+formatExpr(s.Iterable)+":")
		} else {
			p.line(indent, "for "+s.Name+" in "+formatExpr(s.Start)+"..<"+formatExpr(s.End)+":")
		}
		p.stmts(s.Body, indent+1)
	case *frontend.MatchStmt:
		p.line(indent, "match "+formatExpr(s.Value)+":")
		for _, c := range s.Cases {
			if c.Default {
				p.line(indent, "case _:")
			} else {
				p.line(indent, "case "+formatExpr(c.Pattern)+":")
			}
			p.stmts(c.Body, indent+1)
		}
	case *frontend.UnsafeStmt:
		p.line(indent, "unsafe:")
		p.stmts(s.Body, indent+1)
	case *frontend.IslandStmt:
		p.line(indent, "island("+formatExpr(s.Size)+") as "+s.Name+":")
		p.stmts(s.Body, indent+1)
	case *frontend.FreeStmt:
		p.line(indent, "free("+formatExpr(s.Value)+")")
	case *frontend.ExprStmt:
		p.line(indent, formatExpr(s.Expr))
	}
}

func (p *sourcePrinter) line(indent int, s string) {
	p.b.WriteString(strings.Repeat(" ", indent*4))
	p.b.WriteString(s)
	p.b.WriteByte('\n')
}

func (p *sourcePrinter) blank() {
	p.b.WriteByte('\n')
}

func formatTypeRef(ref frontend.TypeRef) string {
	switch ref.Kind {
	case frontend.TypeRefSlice:
		return "[]" + formatTypeRef(*ref.Elem)
	case frontend.TypeRefArray:
		return "[" + strconv.Itoa(ref.Len) + "]" + formatTypeRef(*ref.Elem)
	case frontend.TypeRefOptional:
		return formatTypeRef(*ref.Elem) + "?"
	default:
		return ref.Name
	}
}

func formatExpr(expr frontend.Expr) string {
	return formatExprPrec(expr, 0)
}

func formatExprPrec(expr frontend.Expr, parent int) string {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return strconv.Itoa(int(e.Value))
	case *frontend.BoolLitExpr:
		if e.Value {
			return "true"
		}
		return "false"
	case *frontend.NoneLitExpr:
		return "none"
	case *frontend.SomePatternExpr:
		return "some(" + e.Name + ")"
	case *frontend.StringLitExpr:
		return strconv.Quote(string(e.Value))
	case *frontend.IdentExpr:
		return e.Name
	case *frontend.UnaryExpr:
		if e.Op == frontend.TokenBang {
			return "!" + formatExprPrec(e.X, 7)
		}
		return "-" + formatExprPrec(e.X, 7)
	case *frontend.TryExpr:
		return "try " + formatExprPrec(e.X, 7)
	case *frontend.AwaitExpr:
		return "await " + formatExprPrec(e.X, 7)
	case *frontend.BinaryExpr:
		prec := exprPrecedence(e.Op)
		left := formatExprPrec(e.Left, prec)
		right := formatExprPrec(e.Right, prec+1)
		out := left + " " + tokenOp(e.Op) + " " + right
		if prec < parent {
			return "(" + out + ")"
		}
		return out
	case *frontend.CallExpr:
		args := make([]string, 0, len(e.Args))
		for i, arg := range e.Args {
			if i < len(e.ArgLabels) && e.ArgLabels[i] != "" {
				args = append(args, e.ArgLabels[i]+": "+formatExpr(arg))
				continue
			}
			args = append(args, formatExpr(arg))
		}
		return e.Name + "(" + strings.Join(args, ", ") + ")"
	case *frontend.StructLitExpr:
		fields := make([]string, 0, len(e.Fields))
		for _, f := range e.Fields {
			fields = append(fields, f.Name+": "+formatExpr(f.Value))
		}
		return formatTypeRef(e.Type) + "(" + strings.Join(fields, ", ") + ")"
	case *frontend.FieldAccessExpr:
		return formatExprPrec(e.Base, 8) + "." + e.Field
	case *frontend.IndexExpr:
		return formatExprPrec(e.Base, 8) + "[" + formatExpr(e.Index) + "]"
	default:
		return "<expr>"
	}
}

func exprPrecedence(op frontend.TokenType) int {
	switch op {
	case frontend.TokenPipePipe:
		return 1
	case frontend.TokenAmpAmp:
		return 2
	case frontend.TokenEqEq, frontend.TokenBangEq:
		return 3
	case frontend.TokenLess, frontend.TokenLessEq, frontend.TokenGreater, frontend.TokenGreaterEq:
		return 4
	case frontend.TokenPlus, frontend.TokenMinus:
		return 5
	case frontend.TokenStar, frontend.TokenSlash, frontend.TokenPercent:
		return 6
	default:
		return 0
	}
}

func tokenOp(op frontend.TokenType) string {
	switch op {
	case frontend.TokenPipePipe:
		return "||"
	case frontend.TokenAmpAmp:
		return "&&"
	case frontend.TokenEqEq:
		return "=="
	case frontend.TokenBangEq:
		return "!="
	case frontend.TokenLess:
		return "<"
	case frontend.TokenLessEq:
		return "<="
	case frontend.TokenGreater:
		return ">"
	case frontend.TokenGreaterEq:
		return ">="
	case frontend.TokenPlus:
		return "+"
	case frontend.TokenMinus:
		return "-"
	case frontend.TokenStar:
		return "*"
	case frontend.TokenSlash:
		return "/"
	case frontend.TokenPercent:
		return "%"
	default:
		return "?"
	}
}

func compoundAssignmentOp(op frontend.TokenType) string {
	switch op {
	case frontend.TokenPlus:
		return "+"
	case frontend.TokenMinus:
		return "-"
	case frontend.TokenStar:
		return "*"
	case frontend.TokenSlash:
		return "/"
	case frontend.TokenPercent:
		return "%"
	default:
		return "?"
	}
}
