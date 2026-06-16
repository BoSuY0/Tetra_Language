package frontend

import "strings"

func (p *parser) parseExpr() (Expr, error) {
	return p.parseLogicalOr()
}

func (p *parser) parseLogicalOr() (Expr, error) {
	left, err := p.parseLogicalAnd()
	if err != nil {
		return nil, err
	}
	for p.cur.typ == TokenPipePipe {
		opTok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		right, err := p.parseLogicalAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{At: opTok.pos, Op: opTok.typ, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseLogicalAnd() (Expr, error) {
	left, err := p.parseEquality()
	if err != nil {
		return nil, err
	}
	for p.cur.typ == TokenAmpAmp {
		opTok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		right, err := p.parseEquality()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{At: opTok.pos, Op: opTok.typ, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseEquality() (Expr, error) {
	left, err := p.parseRelational()
	if err != nil {
		return nil, err
	}
	if p.cur.typ == TokenEqEq || p.cur.typ == TokenBangEq {
		opTok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		right, err := p.parseRelational()
		if err != nil {
			return nil, err
		}
		if p.cur.typ == TokenEqEq || p.cur.typ == TokenBangEq {
			return nil, diagnosticErrorf(p.cur.pos, "chaining equality operators is not supported")
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
	if p.cur.typ == TokenLess || p.cur.typ == TokenGreater || p.cur.typ == TokenGreaterEq || p.cur.typ == TokenLessEq {
		opTok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		if p.cur.typ == TokenLess || p.cur.typ == TokenGreater || p.cur.typ == TokenGreaterEq || p.cur.typ == TokenLessEq {
			return nil, diagnosticErrorf(p.cur.pos, "chaining relational operators is not supported")
		}
		left = &BinaryExpr{At: opTok.pos, Op: opTok.typ, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseAdditive() (Expr, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}
	for p.cur.typ == TokenPlus || p.cur.typ == TokenMinus {
		opTok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		right, err := p.parseMultiplicative()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{At: opTok.pos, Op: opTok.typ, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseMultiplicative() (Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.cur.typ == TokenStar || p.cur.typ == TokenSlash || p.cur.typ == TokenPercent {
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
	if p.cur.typ == TokenTry {
		opTok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		x, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &TryExpr{At: opTok.pos, X: x}, nil
	}
	if p.cur.typ == TokenAwait {
		opTok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		x, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &AwaitExpr{At: opTok.pos, X: x}, nil
	}
	if p.cur.typ == TokenMinus && p.peek.typ == TokenNumber && p.peek.num == maxI32NumberLiteral+1 {
		opTok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		if err := p.next(); err != nil {
			return nil, err
		}
		return p.parsePostfix(&NumberExpr{At: opTok.pos, Value: minI32NumberLiteral})
	}
	if p.cur.typ == TokenMinus || p.cur.typ == TokenBang {
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
		num, err := numberExprFromToken(tok)
		if err != nil {
			return nil, err
		}
		return p.parsePostfix(num)
	case TokenTrue:
		tok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		return p.parsePostfix(&BoolLitExpr{At: tok.pos, Value: true})
	case TokenFalse:
		tok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		return p.parsePostfix(&BoolLitExpr{At: tok.pos, Value: false})
	case TokenNone:
		tok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		return p.parsePostfix(&NoneLitExpr{At: tok.pos})
	case TokenString:
		tok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		return p.parsePostfix(&StringLitExpr{At: tok.pos, Value: tok.str})
	case TokenIdent:
		if p.cur.lit == "closure" && p.peek.typ == TokenLParen {
			return nil, diagnosticErrorf(p.cur.pos, "closure literal expressions use 'fn(...) -> Type'; named callables use top-level 'closure Name(...)' declarations")
		}
		pos := p.cur.pos
		parts, err := p.parsePathParts()
		if err != nil {
			return nil, err
		}
		name := strings.Join(parts, ".")
		typeArgs, hasTypeArgs, err := p.tryParseCallTypeArgs()
		if err != nil {
			return nil, err
		}
		if p.cur.typ == TokenLParen {
			if err := p.next(); err != nil {
				return nil, err
			}
			if p.cur.typ == TokenIdent && p.peek.typ == TokenColon {
				if !isFunctionLikeCallee(parts) {
					typeRef := TypeRef{At: pos, Kind: TypeRefNamed, Name: name, TypeArgs: typeArgs}
					lit, err := p.parseStructCallLiteral(typeRef)
					if err != nil {
						return nil, err
					}
					return p.parsePostfix(lit)
				}
			}
			args, labels, err := p.parseCallArgs()
			if err != nil {
				return nil, err
			}
			return p.parsePostfix(&CallExpr{At: pos, Name: name, TypeArgs: typeArgs, Args: args, ArgLabels: labels})
		}
		if !hasTypeArgs {
			typeArgs, hasTypeArgs, err = p.tryParseStructLiteralTypeArgs()
			if err != nil {
				return nil, err
			}
		}
		if p.cur.typ == TokenLBrace && !p.suppressStructLiteral {
			typeRef := TypeRef{At: pos, Kind: TypeRefNamed, Name: name, TypeArgs: typeArgs}
			lit, err := p.parseStructLiteral(typeRef)
			if err != nil {
				return nil, err
			}
			return p.parsePostfix(lit)
		}
		if hasTypeArgs {
			return nil, diagnosticErrorf(pos, "generic type arguments require a call or struct literal")
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
	case TokenFn, TokenFun:
		return p.parseClosureExpr()
	case TokenMatch:
		return p.parseMatchExpr()
	case TokenCatch:
		return p.parseCatchExpr()
	default:
		return nil, p.unexpected("expression")
	}
}

func (p *parser) parseClosureExpr() (Expr, error) {
	pos := p.cur.pos
	if err := p.next(); err != nil {
		return nil, err
	}
	if p.cur.typ == TokenIdent {
		return nil, diagnosticErrorf(p.cur.pos, "closure literals cannot have names; use top-level closure declarations for named callables")
	}
	typeParams, typeParamBounds, err := p.parseTypeParamsWithBounds(true)
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
	retType, returnOwnership, err := p.parseReturnTypeRef()
	if err != nil {
		return nil, err
	}
	var throws TypeRef
	hasThrows := false
	if p.cur.typ == TokenThrows {
		hasThrows = true
		if err := p.next(); err != nil {
			return nil, err
		}
		parsed, err := p.parseTypeRef()
		if err != nil {
			return nil, err
		}
		throws = parsed
	}
	uses, clauses, err := p.parseFunctionModifiers()
	if err != nil {
		return nil, err
	}
	var body []Stmt
	if p.cur.typ == TokenAssign {
		returnPos := p.cur.pos
		if err := p.next(); err != nil {
			return nil, err
		}
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		body = []Stmt{&ReturnStmt{At: returnPos, Value: expr}}
	} else {
		body, err = p.parseBlock()
		if err != nil {
			return nil, err
		}
	}
	p.closureSeq++
	name := "__closure_" + itoa(pos.Line) + "_" + itoa(pos.Col) + "_" + itoa(p.closureSeq)
	decl := &FuncDecl{
		Pos:             pos,
		Name:            name,
		Synthetic:       true,
		Closure:         true,
		TypeParams:      typeParams,
		TypeParamBounds: typeParamBounds,
		ReturnType:      retType,
		ReturnOwnership: returnOwnership,
		Throws:          throws,
		HasThrows:       hasThrows,
		Params:          params,
		Uses:            uses,
		SemanticClauses: clauses,
		Body:            body,
	}
	p.closureDecls = append(p.closureDecls, decl)
	return &ClosureExpr{At: pos, Name: name, Decl: decl}, nil
}

func (p *parser) parsePostfix(base Expr) (Expr, error) {
	for {
		if p.cur.typ == TokenLBracket {
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
			continue
		}
		if p.cur.typ == TokenDot {
			at := p.cur.pos
			if err := p.next(); err != nil {
				return nil, err
			}
			fieldTok, err := p.expectPathPart()
			if err != nil {
				return nil, err
			}
			typeArgs, hasTypeArgs, err := p.tryParseCallTypeArgs()
			if err != nil {
				return nil, err
			}
			if p.cur.typ == TokenLParen {
				if err := p.next(); err != nil {
					return nil, err
				}
				args, labels, err := p.parseCallArgs()
				if err != nil {
					return nil, err
				}
				args = append([]Expr{base}, args...)
				if labels != nil {
					labels = append([]string{""}, labels...)
				}
				base = &CallExpr{At: at, Name: "__method." + fieldTok.lit, TypeArgs: typeArgs, Args: args, ArgLabels: labels}
				continue
			}
			if hasTypeArgs {
				return nil, diagnosticErrorf(at, "generic type arguments require a call or struct literal")
			}
			base = &FieldAccessExpr{At: at, Base: base, Field: fieldTok.lit}
			continue
		}
		return base, nil
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

func cloneCompoundTarget(expr Expr) Expr {
	switch e := expr.(type) {
	case *IdentExpr:
		return &IdentExpr{At: e.At, Name: e.Name}
	case *FieldAccessExpr:
		return &FieldAccessExpr{At: e.At, Base: cloneCompoundTarget(e.Base), Field: e.Field, EnumType: e.EnumType, EnumOrdinal: e.EnumOrdinal}
	case *IndexExpr:
		return &IndexExpr{At: e.At, Base: cloneCompoundTarget(e.Base), Index: cloneCompoundTarget(e.Index)}
	case *NumberExpr:
		return &NumberExpr{At: e.At, Value: e.Value}
	case *BoolLitExpr:
		return &BoolLitExpr{At: e.At, Value: e.Value}
	case *NoneLitExpr:
		return &NoneLitExpr{At: e.At}
	case *StringLitExpr:
		return &StringLitExpr{At: e.At, Value: append([]byte(nil), e.Value...)}
	case *UnaryExpr:
		return &UnaryExpr{At: e.At, Op: e.Op, X: cloneCompoundTarget(e.X)}
	case *BinaryExpr:
		return &BinaryExpr{At: e.At, Op: e.Op, Left: cloneCompoundTarget(e.Left), Right: cloneCompoundTarget(e.Right)}
	case *CallExpr:
		args := make([]Expr, 0, len(e.Args))
		for _, arg := range e.Args {
			args = append(args, cloneCompoundTarget(arg))
		}
		labels := append([]string(nil), e.ArgLabels...)
		typeArgs := append([]TypeRef(nil), e.TypeArgs...)
		return &CallExpr{At: e.At, Name: e.Name, TypeArgs: typeArgs, Args: args, ArgLabels: labels, ResolvedType: e.ResolvedType}
	default:
		return expr
	}
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
	return diagnosticErrorf(p.cur.pos, "expected %s, got %s", expect, TokenName(p.cur.typ))
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
	case TokenConst:
		return "const"
	case TokenModule:
		return "module"
	case TokenImport:
		return "import"
	case TokenPub:
		return "pub"
	case TokenAs:
		return "as"
	case TokenUses:
		return "uses"
	case TokenIf:
		return "if"
	case TokenElse:
		return "else"
	case TokenWhile:
		return "while"
	case TokenFor:
		return "for"
	case TokenIn:
		return "in"
	case TokenEnum:
		return "enum"
	case TokenCase:
		return "case"
	case TokenMatch:
		return "match"
	case TokenTrue:
		return "true"
	case TokenFalse:
		return "false"
	case TokenNone:
		return "none"
	case TokenThrows:
		return "throws"
	case TokenTry:
		return "try"
	case TokenThrow:
		return "throw"
	case TokenCatch:
		return "catch"
	case TokenAsync:
		return "async"
	case TokenAwait:
		return "await"
	case TokenBreak:
		return "break"
	case TokenContinue:
		return "continue"
	case TokenReturn:
		return "return"
	case TokenPrint:
		return "print"
	case TokenUnsafe:
		return "unsafe"
	case TokenTest:
		return "test"
	case TokenExpect:
		return "expect"
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
	case TokenStar:
		return "*"
	case TokenSlash:
		return "/"
	case TokenPercent:
		return "%"
	case TokenGreater:
		return ">"
	case TokenGreaterEq:
		return ">="
	case TokenLessEq:
		return "<="
	case TokenBangEq:
		return "!="
	case TokenAmpAmp:
		return "&&"
	case TokenPipePipe:
		return "||"
	case TokenBang:
		return "!"
	case TokenComma:
		return ","
	case TokenDot:
		return "."
	case TokenRangeUntil:
		return "..<"
	case TokenQuestion:
		return "?"
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
