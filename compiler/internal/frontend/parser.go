package frontend

import (
	"fmt"
	"strings"
)

func Parse(src []byte) (*Program, error) {
	file, err := ParseFile(src, "")
	if err != nil {
		return nil, err
	}
	if file.Module != "" || len(file.Imports) > 0 || len(file.Globals) > 0 {
		return nil, fmt.Errorf("module/import/global declarations require ParseFile")
	}
	return &Program{Structs: file.Structs, Funcs: file.Funcs}, nil
}

func ParseFile(src []byte, filename string) (*FileAST, error) {
	p, err := newParser(src, filename)
	if err != nil {
		return nil, err
	}
	file, err := p.parseFile()
	if err != nil {
		return nil, err
	}
	file.Path = filename
	file.Src = append([]byte(nil), src...)
	return file, nil
}

type parser struct {
	l    *lexer
	cur  token
	peek token
}

func newParser(src []byte, filename string) (*parser, error) {
	p := &parser{l: newLexer(src, filename)}
	tok, err := p.l.nextToken()
	if err != nil {
		return nil, err
	}
	p.cur = tok
	tok, err = p.l.nextToken()
	if err != nil {
		return nil, err
	}
	p.peek = tok
	return p, nil
}

func (p *parser) next() error {
	tok, err := p.l.nextToken()
	if err != nil {
		return err
	}
	p.cur = p.peek
	p.peek = tok
	return nil
}

func (p *parser) parseFile() (*FileAST, error) {
	file := &FileAST{}
	seenFunc := false
	seenStruct := false
	seenGlobal := false
	for p.cur.typ != TokenEOF {
		switch p.cur.typ {
		case TokenSemicolon:
			if err := p.next(); err != nil {
				return nil, err
			}
			continue
		case TokenModule:
			if seenFunc || seenStruct || seenGlobal {
				return nil, fmt.Errorf("%s: module must appear before declarations", FormatPos(p.cur.pos))
			}
			if file.Module != "" {
				return nil, fmt.Errorf("%s: duplicate module declaration", FormatPos(p.cur.pos))
			}
			if err := p.next(); err != nil {
				return nil, err
			}
			path, err := p.parsePath()
			if err != nil {
				return nil, err
			}
			file.Module = path
			if err := p.consumeOptionalSemicolon(); err != nil {
				return nil, err
			}
			continue
		case TokenImport:
			pos := p.cur.pos
			if seenFunc || seenStruct || seenGlobal {
				return nil, fmt.Errorf("%s: import must appear before declarations", FormatPos(p.cur.pos))
			}
			if err := p.next(); err != nil {
				return nil, err
			}
			path, err := p.parsePath()
			if err != nil {
				return nil, err
			}
			alias := ""
			if p.cur.typ == TokenAs {
				if err := p.next(); err != nil {
					return nil, err
				}
				aliasTok, err := p.expect(TokenIdent)
				if err != nil {
					return nil, err
				}
				alias = aliasTok.lit
			} else {
				parts := strings.Split(path, ".")
				alias = parts[len(parts)-1]
			}
			file.Imports = append(file.Imports, ImportDecl{At: pos, Path: path, Alias: alias})
			if err := p.consumeOptionalSemicolon(); err != nil {
				return nil, err
			}
			continue
		case TokenStruct:
			if seenFunc || seenGlobal {
				return nil, fmt.Errorf("%s: struct must appear before globals/functions", FormatPos(p.cur.pos))
			}
			seenStruct = true
			st, err := p.parseStructDecl()
			if err != nil {
				return nil, err
			}
			file.Structs = append(file.Structs, st)
			continue
		case TokenVar, TokenVal:
			if seenFunc {
				return nil, fmt.Errorf("%s: global must appear before functions", FormatPos(p.cur.pos))
			}
			seenGlobal = true
			glob, err := p.parseGlobalDecl()
			if err != nil {
				return nil, err
			}
			file.Globals = append(file.Globals, glob)
			continue
		case TokenAt, TokenFn, TokenFun:
			seenFunc = true
			fn, err := p.parseFuncDecl()
			if err != nil {
				return nil, err
			}
			file.Funcs = append(file.Funcs, fn)
		default:
			return nil, p.unexpected("module/import/struct/var/val/fn")
		}
	}
	return file, nil
}

func (p *parser) parseGlobalDecl() (*GlobalDecl, error) {
	if p.cur.typ != TokenVar && p.cur.typ != TokenVal {
		return nil, p.unexpected("var/val")
	}
	pos := p.cur.pos
	declTok := p.cur
	if err := p.next(); err != nil {
		return nil, err
	}
	nameTok, err := p.expect(TokenIdent)
	if err != nil {
		return nil, err
	}
	typeRef := TypeRef{At: pos, Kind: TypeRefNamed, Name: ""}
	if p.cur.typ == TokenColon {
		if err := p.next(); err != nil {
			return nil, err
		}
		parsed, err := p.parseTypeRef()
		if err != nil {
			return nil, err
		}
		typeRef = parsed
	} else if declTok.typ == TokenVar {
		return nil, fmt.Errorf("%s: global var requires an explicit type annotation", FormatPos(p.cur.pos))
	}

	var init Expr
	switch declTok.typ {
	case TokenVar:
		if p.cur.typ == TokenAssign {
			return nil, fmt.Errorf("%s: global var initializers are not supported yet", FormatPos(p.cur.pos))
		}
	case TokenVal:
		if _, err := p.expect(TokenAssign); err != nil {
			return nil, err
		}
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		init = expr
	default:
		return nil, p.unexpected("var/val")
	}
	if err := p.consumeOptionalSemicolon(); err != nil {
		return nil, err
	}
	mutable := declTok.typ == TokenVar
	return &GlobalDecl{At: pos, Name: nameTok.lit, Type: typeRef, Mutable: mutable, Init: init}, nil
}

func (p *parser) parseFuncDecl() (*FuncDecl, error) {
	exportName := ""
	for p.cur.typ == TokenAt {
		attrTok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		nameTok, err := p.expect(TokenIdent)
		if err != nil {
			return nil, err
		}
		if nameTok.lit != "export" {
			return nil, fmt.Errorf("%s: unknown attribute '@%s'", FormatPos(attrTok.pos), nameTok.lit)
		}
		if _, err := p.expect(TokenLParen); err != nil {
			return nil, err
		}
		valTok, err := p.expect(TokenString)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
		if exportName != "" {
			return nil, fmt.Errorf("%s: duplicate @export attribute", FormatPos(attrTok.pos))
		}
		exportName = string(valTok.str)
		if exportName == "" {
			return nil, fmt.Errorf("%s: @export name must not be empty", FormatPos(valTok.pos))
		}
		if err := p.consumeOptionalSemicolon(); err != nil {
			return nil, err
		}
	}

	if p.cur.typ != TokenFn && p.cur.typ != TokenFun {
		return nil, p.unexpected("fn/fun")
	}
	if err := p.next(); err != nil {
		return nil, err
	}

	nameTok, err := p.expect(TokenIdent)
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(TokenLParen); err != nil {
		return nil, err
	}
	params, err := p.parseParams()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenRParen); err != nil {
		return nil, err
	}
	switch p.cur.typ {
	case TokenArrow, TokenColon:
		if err := p.next(); err != nil {
			return nil, err
		}
	default:
		return nil, p.unexpected("-> or :")
	}

	retType, err := p.parseTypeRef()
	if err != nil {
		return nil, err
	}

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	fn := &FuncDecl{Pos: nameTok.pos, Name: nameTok.lit, ExportName: exportName, ReturnType: retType, Params: params, Body: body}
	return fn, nil
}

func (p *parser) parsePath() (string, error) {
	parts, err := p.parsePathParts()
	if err != nil {
		return "", err
	}
	return strings.Join(parts, "."), nil
}

func (p *parser) parsePathParts() ([]string, error) {
	first, err := p.expect(TokenIdent)
	if err != nil {
		return nil, err
	}
	parts := []string{first.lit}
	for p.cur.typ == TokenDot {
		if err := p.next(); err != nil {
			return nil, err
		}
		partTok, err := p.expect(TokenIdent)
		if err != nil {
			return nil, err
		}
		parts = append(parts, partTok.lit)
	}
	return parts, nil
}

func (p *parser) parseParams() ([]ParamDecl, error) {
	if p.cur.typ == TokenRParen {
		return nil, nil
	}
	var params []ParamDecl
	for {
		nameTok, err := p.expect(TokenIdent)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenColon); err != nil {
			return nil, err
		}
		typeRef, err := p.parseTypeRef()
		if err != nil {
			return nil, err
		}
		params = append(params, ParamDecl{At: nameTok.pos, Name: nameTok.lit, Type: typeRef})
		if p.cur.typ != TokenComma {
			break
		}
		if err := p.next(); err != nil {
			return nil, err
		}
	}
	return params, nil
}

func (p *parser) parseStructDecl() (*StructDecl, error) {
	if p.cur.typ != TokenStruct {
		return nil, p.unexpected("struct")
	}
	if err := p.next(); err != nil {
		return nil, err
	}
	nameTok, err := p.expect(TokenIdent)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}
	var fields []FieldDecl
	for p.cur.typ != TokenRBrace && p.cur.typ != TokenEOF {
		if p.cur.typ == TokenSemicolon || p.cur.typ == TokenComma {
			if err := p.next(); err != nil {
				return nil, err
			}
			continue
		}
		fieldTok, err := p.expect(TokenIdent)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenColon); err != nil {
			return nil, err
		}
		typ, err := p.parseTypeRef()
		if err != nil {
			return nil, err
		}
		fields = append(fields, FieldDecl{At: fieldTok.pos, Name: fieldTok.lit, Type: typ})
		if p.cur.typ == TokenComma {
			if err := p.next(); err != nil {
				return nil, err
			}
			continue
		}
	}
	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	return &StructDecl{At: nameTok.pos, Name: nameTok.lit, Fields: fields}, nil
}

func (p *parser) parseTypeRef() (TypeRef, error) {
	if p.cur.typ == TokenLBracket {
		at := p.cur.pos
		if err := p.next(); err != nil {
			return TypeRef{}, err
		}
		if p.cur.typ == TokenRBracket {
			if err := p.next(); err != nil {
				return TypeRef{}, err
			}
			elem, err := p.parseTypeRef()
			if err != nil {
				return TypeRef{}, err
			}
			return TypeRef{At: at, Kind: TypeRefSlice, Elem: &elem}, nil
		}
		if p.cur.typ != TokenNumber {
			return TypeRef{}, p.unexpected("number or ]")
		}
		lenTok := p.cur
		if err := p.next(); err != nil {
			return TypeRef{}, err
		}
		if _, err := p.expect(TokenRBracket); err != nil {
			return TypeRef{}, err
		}
		elem, err := p.parseTypeRef()
		if err != nil {
			return TypeRef{}, err
		}
		return TypeRef{At: at, Kind: TypeRefArray, Elem: &elem, Len: int(lenTok.num)}, nil
	}

	first, err := p.expect(TokenIdent)
	if err != nil {
		return TypeRef{}, err
	}
	parts := []string{first.lit}
	for p.cur.typ == TokenDot {
		if err := p.next(); err != nil {
			return TypeRef{}, err
		}
		partTok, err := p.expect(TokenIdent)
		if err != nil {
			return TypeRef{}, err
		}
		parts = append(parts, partTok.lit)
	}
	return TypeRef{At: first.pos, Kind: TypeRefNamed, Name: strings.Join(parts, ".")}, nil
}

func (p *parser) parseBlock() ([]Stmt, error) {
	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}
	var stmts []Stmt
	for p.cur.typ != TokenRBrace && p.cur.typ != TokenEOF {
		if p.cur.typ == TokenSemicolon {
			if err := p.next(); err != nil {
				return nil, err
			}
			continue
		}
		stmt, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, stmt)
	}
	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	return stmts, nil
}

func (p *parser) parseStmt() (Stmt, error) {
	switch p.cur.typ {
	case TokenPrint:
		pos := p.cur.pos
		if err := p.next(); err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenLParen); err != nil {
			return nil, err
		}
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
		if err := p.consumeOptionalSemicolon(); err != nil {
			return nil, err
		}
		return &PrintStmt{At: pos, Value: expr}, nil
	case TokenFree:
		pos := p.cur.pos
		if err := p.next(); err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenLParen); err != nil {
			return nil, err
		}
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
		if err := p.consumeOptionalSemicolon(); err != nil {
			return nil, err
		}
		return &FreeStmt{At: pos, Value: expr, Implicit: false}, nil
	case TokenUnsafe:
		return p.parseUnsafeStmt()
	case TokenReturn:
		pos := p.cur.pos
		if err := p.next(); err != nil {
			return nil, err
		}
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.consumeOptionalSemicolon(); err != nil {
			return nil, err
		}
		return &ReturnStmt{At: pos, Value: expr}, nil
	case TokenLet, TokenVar, TokenVal:
		pos := p.cur.pos
		declTok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		nameTok, err := p.expect(TokenIdent)
		if err != nil {
			return nil, err
		}
		typeRef := TypeRef{At: pos, Kind: TypeRefNamed, Name: ""}
		if p.cur.typ == TokenColon {
			if err := p.next(); err != nil {
				return nil, err
			}
			parsed, err := p.parseTypeRef()
			if err != nil {
				return nil, err
			}
			typeRef = parsed
		}
		if _, err := p.expect(TokenAssign); err != nil {
			return nil, err
		}
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.consumeOptionalSemicolon(); err != nil {
			return nil, err
		}
		mutable := declTok.typ != TokenVal
		return &LetStmt{At: pos, Name: nameTok.lit, Type: typeRef, Mutable: mutable, Value: expr}, nil
	case TokenIf:
		pos := p.cur.pos
		if err := p.next(); err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenLParen); err != nil {
			return nil, err
		}
		cond, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
		thenBlock, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		var elseBlock []Stmt
		if p.cur.typ == TokenElse {
			if err := p.next(); err != nil {
				return nil, err
			}
			elseBlock, err = p.parseBlock()
			if err != nil {
				return nil, err
			}
		}
		return &IfStmt{At: pos, Cond: cond, Then: thenBlock, Else: elseBlock}, nil
	case TokenWhile:
		pos := p.cur.pos
		if err := p.next(); err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenLParen); err != nil {
			return nil, err
		}
		cond, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
		body, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		return &WhileStmt{At: pos, Cond: cond, Body: body}, nil
	case TokenIdent:
		if p.cur.lit == "island" && p.peek.typ == TokenLParen {
			return p.parseIslandStmt()
		}
		pos := p.cur.pos
		target, err := p.parseLValue()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenAssign); err != nil {
			return nil, err
		}
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.consumeOptionalSemicolon(); err != nil {
			return nil, err
		}
		return &AssignStmt{At: pos, Target: target, Value: value}, nil
	default:
		return nil, p.unexpected("statement")
	}
}

func (p *parser) parseIslandStmt() (Stmt, error) {
	pos := p.cur.pos
	if err := p.next(); err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenLParen); err != nil {
		return nil, err
	}
	size, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenRParen); err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenAs); err != nil {
		return nil, err
	}
	nameTok, err := p.expect(TokenIdent)
	if err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &IslandStmt{At: pos, Size: size, Name: nameTok.lit, Body: body}, nil
}

func (p *parser) parseUnsafeStmt() (Stmt, error) {
	pos := p.cur.pos
	if err := p.next(); err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &UnsafeStmt{At: pos, Body: body}, nil
}

func (p *parser) parseExpr() (Expr, error) {
	return p.parseEquality()
}

func (p *parser) parseEquality() (Expr, error) {
	left, err := p.parseRelational()
	if err != nil {
		return nil, err
	}
	if p.cur.typ == TokenEqEq {
		opTok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		right, err := p.parseRelational()
		if err != nil {
			return nil, err
		}
		if p.cur.typ == TokenEqEq {
			return nil, fmt.Errorf("%s: chaining '==' is not supported", FormatPos(p.cur.pos))
		}
		left = &BinaryExpr{At: opTok.pos, Op: opTok.typ, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseRelational() (Expr, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}
	if p.cur.typ == TokenLess {
		opTok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		if p.cur.typ == TokenLess {
			return nil, fmt.Errorf("%s: chaining '<' is not supported", FormatPos(p.cur.pos))
		}
		left = &BinaryExpr{At: opTok.pos, Op: opTok.typ, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseAdditive() (Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.cur.typ == TokenPlus || p.cur.typ == TokenMinus {
		opTok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{At: opTok.pos, Op: opTok.typ, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseUnary() (Expr, error) {
	if p.cur.typ == TokenMinus {
		opTok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		x, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{At: opTok.pos, Op: opTok.typ, X: x}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (Expr, error) {
	switch p.cur.typ {
	case TokenNumber:
		tok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		return p.parsePostfix(&NumberExpr{At: tok.pos, Value: int32(tok.num)})
	case TokenString:
		tok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		return p.parsePostfix(&StringLitExpr{At: tok.pos, Value: tok.str})
	case TokenIdent:
		pos := p.cur.pos
		parts, err := p.parsePathParts()
		if err != nil {
			return nil, err
		}
		name := strings.Join(parts, ".")
		if p.cur.typ == TokenLParen {
			if err := p.next(); err != nil {
				return nil, err
			}
			if p.cur.typ == TokenIdent && p.peek.typ == TokenColon {
				typeRef := TypeRef{At: pos, Kind: TypeRefNamed, Name: name}
				lit, err := p.parseStructCallLiteral(typeRef)
				if err != nil {
					return nil, err
				}
				return p.parsePostfix(lit)
			}
			var args []Expr
			if p.cur.typ != TokenRParen {
				for {
					arg, err := p.parseExpr()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
					if p.cur.typ != TokenComma {
						break
					}
					if err := p.next(); err != nil {
						return nil, err
					}
				}
			}
			if _, err := p.expect(TokenRParen); err != nil {
				return nil, err
			}
			return p.parsePostfix(&CallExpr{At: pos, Name: name, Args: args})
		}
		if p.cur.typ == TokenLBrace {
			typeRef := TypeRef{At: pos, Kind: TypeRefNamed, Name: name}
			lit, err := p.parseStructLiteral(typeRef)
			if err != nil {
				return nil, err
			}
			return p.parsePostfix(lit)
		}
		return p.parsePostfix(buildFieldAccess(parts, pos))
	case TokenLParen:
		if err := p.next(); err != nil {
			return nil, err
		}
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
		return p.parsePostfix(expr)
	default:
		return nil, p.unexpected("expression")
	}
}

func (p *parser) parsePostfix(base Expr) (Expr, error) {
	for {
		if p.cur.typ != TokenLBracket {
			return base, nil
		}
		at := p.cur.pos
		if err := p.next(); err != nil {
			return nil, err
		}
		index, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenRBracket); err != nil {
			return nil, err
		}
		base = &IndexExpr{At: at, Base: base, Index: index}
	}
}

func (p *parser) parseStructLiteral(typeRef TypeRef) (*StructLitExpr, error) {
	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}
	var fields []StructFieldInit
	for p.cur.typ != TokenRBrace && p.cur.typ != TokenEOF {
		if p.cur.typ == TokenSemicolon || p.cur.typ == TokenComma {
			if err := p.next(); err != nil {
				return nil, err
			}
			continue
		}
		fieldTok, err := p.expect(TokenIdent)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenColon); err != nil {
			return nil, err
		}
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		fields = append(fields, StructFieldInit{At: fieldTok.pos, Name: fieldTok.lit, Value: value})
		if p.cur.typ == TokenComma {
			if err := p.next(); err != nil {
				return nil, err
			}
		}
	}
	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	return &StructLitExpr{At: typeRef.At, Type: typeRef, Fields: fields}, nil
}

func (p *parser) parseStructCallLiteral(typeRef TypeRef) (*StructLitExpr, error) {
	var fields []StructFieldInit
	for p.cur.typ != TokenRParen && p.cur.typ != TokenEOF {
		if p.cur.typ == TokenSemicolon || p.cur.typ == TokenComma {
			if err := p.next(); err != nil {
				return nil, err
			}
			continue
		}
		fieldTok, err := p.expect(TokenIdent)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenColon); err != nil {
			return nil, err
		}
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		fields = append(fields, StructFieldInit{At: fieldTok.pos, Name: fieldTok.lit, Value: value})
		if p.cur.typ == TokenComma {
			if err := p.next(); err != nil {
				return nil, err
			}
		}
	}
	if _, err := p.expect(TokenRParen); err != nil {
		return nil, err
	}
	return &StructLitExpr{At: typeRef.At, Type: typeRef, Fields: fields}, nil
}

func (p *parser) parseLValue() (Expr, error) {
	if p.cur.typ != TokenIdent {
		return nil, p.unexpected("identifier")
	}
	pos := p.cur.pos
	parts, err := p.parsePathParts()
	if err != nil {
		return nil, err
	}
	expr := buildFieldAccess(parts, pos)
	for p.cur.typ == TokenLBracket {
		if err := p.next(); err != nil {
			return nil, err
		}
		index, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenRBracket); err != nil {
			return nil, err
		}
		expr = &IndexExpr{At: pos, Base: expr, Index: index}
	}
	return expr, nil
}
func buildFieldAccess(parts []string, at Position) Expr {
	if len(parts) == 0 {
		return &IdentExpr{At: at, Name: ""}
	}
	base := Expr(&IdentExpr{At: at, Name: parts[0]})
	for i := 1; i < len(parts); i++ {
		base = &FieldAccessExpr{At: at, Base: base, Field: parts[i]}
	}
	return base
}

func (p *parser) expect(tt TokenType) (token, error) {
	if p.cur.typ != tt {
		return token{}, p.unexpected(TokenName(tt))
	}
	tok := p.cur
	if err := p.next(); err != nil {
		return token{}, err
	}
	return tok, nil
}

func (p *parser) consumeOptionalSemicolon() error {
	if p.cur.typ == TokenSemicolon {
		return p.next()
	}
	return nil
}

func (p *parser) unexpected(expect string) error {
	return fmt.Errorf("%s: expected %s, got %s", FormatPos(p.cur.pos), expect, TokenName(p.cur.typ))
}

func TokenName(tt TokenType) string {
	switch tt {
	case TokenEOF:
		return "EOF"
	case TokenIdent:
		return "identifier"
	case TokenNumber:
		return "number"
	case TokenString:
		return "string"
	case TokenFn:
		return "fn"
	case TokenFun:
		return "fun"
	case TokenLet:
		return "let"
	case TokenVar:
		return "var"
	case TokenVal:
		return "val"
	case TokenModule:
		return "module"
	case TokenImport:
		return "import"
	case TokenAs:
		return "as"
	case TokenIf:
		return "if"
	case TokenElse:
		return "else"
	case TokenWhile:
		return "while"
	case TokenReturn:
		return "return"
	case TokenPrint:
		return "print"
	case TokenUnsafe:
		return "unsafe"
	case TokenStruct:
		return "struct"
	case TokenArrow:
		return "->"
	case TokenColon:
		return ":"
	case TokenAssign:
		return "="
	case TokenEqEq:
		return "=="
	case TokenPlus:
		return "+"
	case TokenMinus:
		return "-"
	case TokenLess:
		return "<"
	case TokenComma:
		return ","
	case TokenDot:
		return "."
	case TokenLBracket:
		return "["
	case TokenRBracket:
		return "]"
	case TokenLParen:
		return "("
	case TokenRParen:
		return ")"
	case TokenLBrace:
		return "{"
	case TokenRBrace:
		return "}"
	case TokenSemicolon:
		return ";"
	default:
		return "?"
	}
}
