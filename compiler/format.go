package compiler

import (
	"bytes"
	"sort"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

// FormatSource returns canonical Flow formatting for the supported v1 surface.
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
					Code:     DiagnosticCodeFormatter,
					Message:  "inline comments are not supported by tetra fmt for the v1.0 profile; move the comment to its own line or format manually",
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
							Code:     DiagnosticCodeFormatter,
							Message:  "inline comments are not supported by tetra fmt for the v1.0 profile; move the comment to its own line or format manually",
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
						Code:     DiagnosticCodeFormatterCheck,
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
	if isFunctionHeaderLine(trimmed) {
		count := 1
		modifiers := countFunctionModifiers(trimmed)
		if modifiers > 0 {
			count += modifiers
		}
		if strings.Contains(trimmed, " = ") {
			count++
		}
		return count
	}

	if closure, ok := closureHeaderSegment(trimmed); ok {
		count := 1
		modifiers := countFunctionModifiers(closure)
		if modifiers > 0 {
			count += modifiers
		}
		if strings.Contains(closure, " = ") {
			count++
		}
		return count
	}

	return 1
}

func countFunctionModifiers(line string) int {
	count := 1
	if strings.Contains(line, " uses ") {
		count++
	}
	for _, clause := range []string{" noalloc", " noblock", " realtime", " nothrow", " budget("} {
		count += strings.Count(line, clause)
	}
	return count - 1
}

func isFunctionHeaderLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "func ") ||
		strings.HasPrefix(trimmed, "async func ") ||
		strings.HasPrefix(trimmed, "fun ") ||
		strings.HasPrefix(trimmed, "async fun ")
}

func closureHeaderSegment(trimmed string) (string, bool) {
	fnAt := strings.Index(trimmed, "fn(")
	if fnAt < 0 {
		fnAt = strings.Index(trimmed, "fun(")
	}
	if fnAt < 0 {
		return "", false
	}
	assignAt := strings.LastIndex(trimmed[:fnAt], "=")
	if assignAt < 0 {
		return "", false
	}
	return strings.TrimSpace(trimmed[fnAt:]), true
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
	b             bytes.Buffer
	closures      map[string]*frontend.FuncDecl
	emitSynthetic map[string]struct{}
}

type actorMethodGroup struct {
	Name    string
	Fields  []frontend.StateFieldDecl
	Methods []*frontend.FuncDecl
}

func (p *sourcePrinter) file(file *frontend.FileAST) {
	p.closures = make(map[string]*frontend.FuncDecl, len(file.Funcs))
	p.emitSynthetic = make(map[string]struct{})
	for _, fn := range file.Funcs {
		if fn.Synthetic {
			p.closures[fn.Name] = fn
		}
	}

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
	for _, st := range file.States {
		p.stateDecl(st)
		p.blank()
	}
	for _, view := range file.Views {
		p.viewDecl(view)
		p.blank()
	}
	for _, g := range file.Globals {
		p.globalDecl(g)
	}
	if len(file.Globals) > 0 {
		p.blank()
	}
	actorGroups, actorMethodNames := collectActorMethodGroups(file)
	for _, group := range actorGroups {
		p.actorDecl(group)
		p.blank()
	}
	for _, fn := range file.Funcs {
		if fn.ExtensionOf != "" || fn.Synthetic {
			continue
		}
		if _, ok := actorMethodNames[fn.Name]; ok {
			continue
		}
		p.funcDecl(fn)
		p.blank()
	}
	for _, fn := range file.Funcs {
		if !fn.Synthetic {
			continue
		}
		if _, ok := p.emitSynthetic[fn.Name]; !ok {
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

func collectActorMethodGroups(file *frontend.FileAST) ([]actorMethodGroup, map[string]struct{}) {
	methodNames := make(map[string]struct{})
	groupIndex := map[string]int{}
	var groups []actorMethodGroup
	if file != nil {
		for _, actor := range file.Actors {
			if actor == nil {
				continue
			}
			idx := len(groups)
			groupIndex[actor.Name] = idx
			groups = append(groups, actorMethodGroup{
				Name:   actor.Name,
				Fields: append([]frontend.StateFieldDecl(nil), actor.Fields...),
			})
			for _, fn := range actor.Methods {
				if fn == nil {
					continue
				}
				groups[idx].Methods = append(groups[idx].Methods, fn)
				methodNames[fn.Name] = struct{}{}
			}
		}
	}
	for _, fn := range file.Funcs {
		actorName, _, ok := actorMethodName(fn)
		if !ok {
			continue
		}
		if _, exists := methodNames[fn.Name]; exists {
			continue
		}
		idx, exists := groupIndex[actorName]
		if !exists {
			idx = len(groups)
			groupIndex[actorName] = idx
			groups = append(groups, actorMethodGroup{Name: actorName})
		}
		groups[idx].Methods = append(groups[idx].Methods, fn)
		methodNames[fn.Name] = struct{}{}
	}
	return groups, methodNames
}

func actorMethodName(fn *frontend.FuncDecl) (string, string, bool) {
	if fn.ExtensionOf != "" || fn.Synthetic {
		return "", "", false
	}
	parts := strings.Split(fn.Name, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func (p *sourcePrinter) actorDecl(group actorMethodGroup) {
	p.line(0, "actor "+group.Name+":")
	for _, field := range group.Fields {
		kw := "val"
		if field.Const {
			kw = "const"
		}
		line := kw + " " + field.Name + ": " + formatTypeRef(field.Type)
		if field.Init != nil {
			line += " = " + p.formatExpr(field.Init)
		}
		p.line(1, line)
	}
	for _, fn := range group.Methods {
		_, methodName, _ := actorMethodName(fn)
		p.funcDeclWithNameAt(fn, methodName, 1)
	}
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
		line := "case " + c.Name
		if len(c.Payload) > 0 {
			types := make([]string, 0, len(c.Payload))
			for _, typ := range c.Payload {
				types = append(types, formatTypeRef(typ))
			}
			line += "(" + strings.Join(types, ", ") + ")"
		}
		p.line(1, line)
	}
}

func (p *sourcePrinter) structDecl(st *frontend.StructDecl) {
	typeParams := ""
	if len(st.TypeParams) > 0 {
		typeParams = "<" + strings.Join(st.TypeParams, ", ") + ">"
	}
	p.line(0, "struct "+st.Name+typeParams+":")
	for _, f := range st.Fields {
		p.line(1, f.Name+": "+formatTypeRef(f.Type))
	}
}

func (p *sourcePrinter) stateDecl(st *frontend.StateDecl) {
	p.line(0, "state "+st.Name+":")
	for _, field := range st.Fields {
		kw := "val"
		if field.Mutable {
			kw = "var"
		} else if field.Const {
			kw = "const"
		}
		p.line(1, kw+" "+field.Name+": "+formatTypeRef(field.Type)+" = "+p.formatExpr(field.Init))
	}
}

func (p *sourcePrinter) viewDecl(view *frontend.ViewDecl) {
	p.line(0, "view "+view.Name+"(state: "+formatTypeRef(view.StateName)+"):")
	for _, binding := range view.Bindings {
		p.line(1, "bind "+binding.Name+": "+formatTypeRef(binding.Type)+" = "+p.formatExpr(binding.Value))
	}
	for _, event := range view.Events {
		p.line(1, "event "+event.Name+" -> "+event.Command)
	}
	for _, cmd := range view.Commands {
		p.line(1, "command "+cmd.Name+":")
		p.stmts(cmd.Body, 2)
	}
	for _, style := range view.Styles {
		p.line(1, "style "+style.Name+": "+formatTypeRef(style.Type)+" = "+p.formatExpr(style.Value))
	}
	for _, entry := range view.Accessibility {
		p.line(1, "accessibility "+entry.Name+": "+formatTypeRef(entry.Type)+" = "+p.formatExpr(entry.Value))
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
		line += " = " + p.formatExpr(g.Init)
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
	typeParams := ""
	if len(fn.TypeParams) > 0 {
		typeParams = "<" + strings.Join(fn.TypeParams, ", ") + ">"
	}
	header := p.functionHeader("func", fn.Async, name+typeParams, fn.Params, fn.ReturnType, fn.HasThrows, fn.Throws)
	p.emitHeaderWithModifiers(indent, header, fn.Uses, fn.SemanticClauses)
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
	if len(sig.Uses) > 0 {
		uses := append([]string(nil), sig.Uses...)
		sort.Strings(uses)
		out += " uses " + strings.Join(uses, ", ")
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
	p.line(indent, prefix+" "+p.formatExpr(s.Cond)+":")
	p.stmts(s.Then, indent+1)
	p.elseStmts(s.Else, indent)
}

func (p *sourcePrinter) ifLetStmt(s *frontend.IfLetStmt, indent int, prefix string) {
	binding := s.Name
	if s.Pattern != nil {
		binding = p.formatExpr(s.Pattern)
	}
	p.line(indent, prefix+" let "+binding+" = "+p.formatExpr(s.Value)+":")
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

func (p *sourcePrinter) emitMatchExprStatement(indent int, prefix string, expr frontend.Expr) bool {
	match, ok := expr.(*frontend.MatchExpr)
	if !ok {
		return false
	}
	p.line(indent, prefix+"match "+p.formatExpr(match.Value)+":")
	for _, c := range match.Cases {
		guard := ""
		if c.Guard != nil {
			guard = " if " + p.formatExpr(c.Guard)
		}
		if c.Default {
			p.line(indent, "case _"+guard+":")
		} else {
			p.line(indent, "case "+p.formatExpr(c.Pattern)+guard+":")
		}
		p.line(indent+1, p.formatExpr(c.Value))
	}
	return true
}

func (p *sourcePrinter) emitCatchExprStatement(indent int, prefix string, expr frontend.Expr) bool {
	catch, ok := expr.(*frontend.CatchExpr)
	if !ok {
		return false
	}
	p.line(indent, prefix+"catch "+p.formatExpr(catch.Call)+":")
	for _, c := range catch.Cases {
		guard := ""
		if c.Guard != nil {
			guard = " if " + p.formatExpr(c.Guard)
		}
		if c.Default {
			p.line(indent, "case _"+guard+":")
		} else {
			p.line(indent, "case "+p.formatExpr(c.Pattern)+guard+":")
		}
		p.line(indent+1, p.formatExpr(c.Value))
	}
	return true
}

func (p *sourcePrinter) stmt(stmt frontend.Stmt, indent int) {
	switch s := stmt.(type) {
	case *frontend.PrintStmt:
		p.line(indent, "print("+p.formatExpr(s.Value)+")")
	case *frontend.ExpectStmt:
		p.line(indent, "expect "+p.formatExpr(s.Cond))
	case *frontend.ReturnStmt:
		if p.emitClosureValueStatement(indent, "return ", s.Value) {
			return
		}
		if p.emitMatchExprStatement(indent, "return ", s.Value) {
			return
		}
		if p.emitCatchExprStatement(indent, "return ", s.Value) {
			return
		}
		p.line(indent, "return "+p.formatExpr(s.Value))
	case *frontend.ThrowStmt:
		if p.emitClosureValueStatement(indent, "throw ", s.Value) {
			return
		}
		p.line(indent, "throw "+p.formatExpr(s.Value))
	case *frontend.DeferStmt:
		p.line(indent, "defer:")
		p.stmts(s.Body, indent+1)
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
		line += " = "
		if p.emitClosureValueStatement(indent, line, s.Value) {
			return
		}
		if p.emitMatchExprStatement(indent, line, s.Value) {
			return
		}
		if p.emitCatchExprStatement(indent, line, s.Value) {
			return
		}
		p.line(indent, line+p.formatExpr(s.Value))
	case *frontend.AssignStmt:
		if s.Op != 0 && s.CompoundValue != nil {
			p.line(indent, p.formatExpr(s.Target)+" "+compoundAssignmentOp(s.Op)+"= "+p.formatExpr(s.CompoundValue))
			return
		}
		target := p.formatExpr(s.Target) + " = "
		if p.emitClosureValueStatement(indent, target, s.Value) {
			return
		}
		if p.emitMatchExprStatement(indent, target, s.Value) {
			return
		}
		if p.emitCatchExprStatement(indent, target, s.Value) {
			return
		}
		p.line(indent, target+p.formatExpr(s.Value))
	case *frontend.IfStmt:
		p.ifStmt(s, indent, "if")
	case *frontend.IfLetStmt:
		p.ifLetStmt(s, indent, "if")
	case *frontend.WhileStmt:
		p.line(indent, "while "+p.formatExpr(s.Cond)+":")
		p.stmts(s.Body, indent+1)
	case *frontend.ForRangeStmt:
		if s.Iterable != nil {
			p.line(indent, "for "+s.Name+" in "+p.formatExpr(s.Iterable)+":")
		} else {
			p.line(indent, "for "+s.Name+" in "+p.formatExpr(s.Start)+"..<"+p.formatExpr(s.End)+":")
		}
		p.stmts(s.Body, indent+1)
	case *frontend.MatchStmt:
		p.line(indent, "match "+p.formatExpr(s.Value)+":")
		for _, c := range s.Cases {
			guard := ""
			if c.Guard != nil {
				guard = " if " + p.formatExpr(c.Guard)
			}
			if c.Default {
				p.line(indent, "case _"+guard+":")
			} else {
				p.line(indent, "case "+p.formatExpr(c.Pattern)+guard+":")
			}
			p.stmts(c.Body, indent+1)
		}
	case *frontend.UnsafeStmt:
		p.line(indent, "unsafe:")
		p.stmts(s.Body, indent+1)
	case *frontend.IslandStmt:
		p.line(indent, "island("+p.formatExpr(s.Size)+") as "+s.Name+":")
		p.stmts(s.Body, indent+1)
	case *frontend.FreeStmt:
		p.line(indent, "free("+p.formatExpr(s.Value)+")")
	case *frontend.ExprStmt:
		if p.emitClosureValueStatement(indent, "", s.Expr) {
			return
		}
		p.line(indent, p.formatExpr(s.Expr))
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
		if len(ref.TypeArgs) == 0 {
			return ref.Name
		}
		args := make([]string, 0, len(ref.TypeArgs))
		for _, arg := range ref.TypeArgs {
			args = append(args, formatTypeRef(arg))
		}
		return ref.Name + "<" + strings.Join(args, ", ") + ">"
	}
}

func (p *sourcePrinter) functionHeader(keyword string, async bool, name string, params []frontend.ParamDecl, retType frontend.TypeRef, hasThrows bool, throws frontend.TypeRef) string {
	var out []string
	for _, param := range params {
		typ := formatTypeRef(param.Type)
		if param.Ownership != "" {
			typ = param.Ownership + " " + typ
		}
		out = append(out, param.Name+": "+typ)
	}

	head := keyword
	if async {
		head = "async " + head
	}
	if name != "" {
		head += " " + name
	}
	head += "(" + strings.Join(out, ", ") + ") -> " + formatTypeRef(retType)
	if hasThrows {
		head += " throws " + formatTypeRef(throws)
	}
	return head
}

func (p *sourcePrinter) emitHeaderWithModifiers(indent int, header string, uses []string, clauses []frontend.SemanticClause) {
	modifiers := p.functionModifiers(uses, clauses)
	if len(modifiers) == 0 {
		p.line(indent, header+":")
		return
	}

	p.line(indent, header)
	for i, mod := range modifiers {
		if i == len(modifiers)-1 {
			mod += ":"
		}
		p.line(indent, mod)
	}
}

func (p *sourcePrinter) functionModifiers(uses []string, clauses []frontend.SemanticClause) []string {
	out := make([]string, 0, len(clauses)+1)
	if len(uses) > 0 {
		sorted := append([]string(nil), uses...)
		sort.Strings(sorted)
		out = append(out, "uses "+strings.Join(sorted, ", "))
	}
	for _, clause := range clauses {
		out = append(out, p.formatSemanticClause(clause))
	}
	return out
}

func (p *sourcePrinter) formatSemanticClause(clause frontend.SemanticClause) string {
	if clause.Value == nil {
		return clause.Name
	}
	return clause.Name + "(" + p.formatExpr(clause.Value) + ")"
}

func (p *sourcePrinter) closureHeader(fn *frontend.FuncDecl) string {
	return p.functionHeader("fn", false, "", fn.Params, fn.ReturnType, fn.HasThrows, fn.Throws)
}

func (p *sourcePrinter) inlineFunctionModifiers(uses []string, clauses []frontend.SemanticClause) string {
	return strings.Join(p.functionModifiers(uses, clauses), " ")
}

func (p *sourcePrinter) closureExprDecl(expr frontend.Expr) (*frontend.FuncDecl, bool) {
	closureExpr, ok := expr.(*frontend.ClosureExpr)
	if !ok {
		return nil, false
	}
	closure, ok := p.closures[closureExpr.Name]
	if !ok {
		return nil, false
	}
	return closure, true
}

func (p *sourcePrinter) emitClosureValueStatement(indent int, prefix string, expr frontend.Expr) bool {
	closure, ok := p.closureExprDecl(expr)
	if !ok {
		return false
	}
	p.emitHeaderWithModifiers(indent, prefix+p.closureHeader(closure), closure.Uses, closure.SemanticClauses)
	p.stmts(closure.Body, indent+1)
	return true
}

func singleReturnExpr(stmts []frontend.Stmt) (frontend.Expr, bool) {
	if len(stmts) != 1 {
		return nil, false
	}
	ret, ok := stmts[0].(*frontend.ReturnStmt)
	if !ok || ret.Value == nil {
		return nil, false
	}
	return ret.Value, true
}

func (p *sourcePrinter) formatExpr(expr frontend.Expr) string {
	return p.formatExprPrec(expr, 0)
}

func (p *sourcePrinter) formatExprPrec(expr frontend.Expr, parent int) string {
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
	case *frontend.EnumCasePatternExpr:
		if !e.HasPayload {
			return e.TypeName + "." + e.CaseName
		}
		return e.TypeName + "." + e.CaseName + "(" + strings.Join(e.Bindings, ", ") + ")"
	case *frontend.MatchExpr:
		var b strings.Builder
		b.WriteString("match ")
		b.WriteString(p.formatExpr(e.Value))
		b.WriteString(":")
		for _, c := range e.Cases {
			b.WriteByte('\n')
			guard := ""
			if c.Guard != nil {
				guard = " if " + p.formatExpr(c.Guard)
			}
			if c.Default {
				b.WriteString("case _")
				b.WriteString(guard)
				b.WriteString(":")
			} else {
				b.WriteString("case ")
				b.WriteString(p.formatExpr(c.Pattern))
				b.WriteString(guard)
				b.WriteString(":")
			}
			b.WriteByte('\n')
			b.WriteString("    ")
			b.WriteString(p.formatExpr(c.Value))
		}
		return b.String()
	case *frontend.CatchExpr:
		var b strings.Builder
		b.WriteString("catch ")
		b.WriteString(p.formatExpr(e.Call))
		b.WriteString(":")
		for _, c := range e.Cases {
			b.WriteByte('\n')
			guard := ""
			if c.Guard != nil {
				guard = " if " + p.formatExpr(c.Guard)
			}
			if c.Default {
				b.WriteString("case _")
				b.WriteString(guard)
				b.WriteString(":")
			} else {
				b.WriteString("case ")
				b.WriteString(p.formatExpr(c.Pattern))
				b.WriteString(guard)
				b.WriteString(":")
			}
			b.WriteByte('\n')
			b.WriteString("    ")
			b.WriteString(p.formatExpr(c.Value))
		}
		return b.String()
	case *frontend.StringLitExpr:
		return strconv.Quote(string(e.Value))
	case *frontend.IdentExpr:
		return e.Name
	case *frontend.UnaryExpr:
		if e.Op == frontend.TokenBang {
			return "!" + p.formatExprPrec(e.X, 7)
		}
		return "-" + p.formatExprPrec(e.X, 7)
	case *frontend.TryExpr:
		return "try " + p.formatExprPrec(e.X, 7)
	case *frontend.AwaitExpr:
		return "await " + p.formatExprPrec(e.X, 7)
	case *frontend.BinaryExpr:
		prec := exprPrecedence(e.Op)
		left := p.formatExprPrec(e.Left, prec)
		right := p.formatExprPrec(e.Right, prec+1)
		out := left + " " + tokenOp(e.Op) + " " + right
		if prec < parent {
			return "(" + out + ")"
		}
		return out
	case *frontend.CallExpr:
		args := make([]string, 0, len(e.Args))
		for i, arg := range e.Args {
			if i < len(e.ArgLabels) && e.ArgLabels[i] != "" {
				args = append(args, e.ArgLabels[i]+": "+p.formatExpr(arg))
				continue
			}
			args = append(args, p.formatExpr(arg))
		}
		typeArgs := ""
		if len(e.TypeArgs) > 0 {
			parts := make([]string, 0, len(e.TypeArgs))
			for _, arg := range e.TypeArgs {
				parts = append(parts, formatTypeRef(arg))
			}
			typeArgs = "<" + strings.Join(parts, ", ") + ">"
		}
		return e.Name + typeArgs + "(" + strings.Join(args, ", ") + ")"
	case *frontend.StructLitExpr:
		fields := make([]string, 0, len(e.Fields))
		for _, f := range e.Fields {
			fields = append(fields, f.Name+": "+p.formatExpr(f.Value))
		}
		return formatTypeRef(e.Type) + "(" + strings.Join(fields, ", ") + ")"
	case *frontend.ClosureExpr:
		closure, ok := p.closures[e.Name]
		if !ok {
			return "<expr>"
		}
		value, ok := singleReturnExpr(closure.Body)
		if !ok {
			p.emitSynthetic[e.Name] = struct{}{}
			return e.Name
		}
		header := p.closureHeader(closure)
		if mods := p.inlineFunctionModifiers(closure.Uses, closure.SemanticClauses); mods != "" {
			header += " " + mods
		}
		return header + " = " + p.formatExpr(value)
	case *frontend.FieldAccessExpr:
		return p.formatExprPrec(e.Base, 8) + "." + e.Field
	case *frontend.IndexExpr:
		return p.formatExprPrec(e.Base, 8) + "[" + p.formatExpr(e.Index) + "]"
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
