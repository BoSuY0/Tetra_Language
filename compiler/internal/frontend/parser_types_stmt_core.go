package frontend

import "strings"

func (p *parser) parseFunctionModifiers() ([]string, []SemanticClause, error) {
	var uses []string
	var clauses []SemanticClause
	seenUses := false
	seenClauses := map[string]Position{}
	for {
		if p.cur.typ == TokenUses {
			if seenUses {
				return nil, nil, diagnosticErrorf(p.cur.pos, "duplicate uses clause")
			}
			seenUses = true
			if err := p.next(); err != nil {
				return nil, nil, err
			}
			for {
				capability, err := p.parsePath()
				if err != nil {
					return nil, nil, err
				}
				uses = append(uses, capability)
				if p.cur.typ != TokenComma {
					break
				}
				if err := p.next(); err != nil {
					return nil, nil, err
				}
			}
			continue
		}
		clause, ok, err := p.parseSemanticClause()
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			break
		}
		if first, exists := seenClauses[clause.Name]; exists {
			return nil, nil, diagnosticErrorf(clause.At, "duplicate semantic clause '%s' (first at %s)", clause.Name, FormatPos(first))
		}
		seenClauses[clause.Name] = clause.At
		clauses = append(clauses, clause)
	}
	return uses, clauses, nil
}

func (p *parser) parseSemanticClause() (SemanticClause, bool, error) {
	if p.cur.typ != TokenIdent {
		return SemanticClause{}, false, nil
	}
	switch p.cur.lit {
	case "noalloc", "noblock", "realtime", "nothrow", "privacy":
		clause := SemanticClause{At: p.cur.pos, Name: p.cur.lit}
		if err := p.next(); err != nil {
			return SemanticClause{}, false, err
		}
		if p.cur.typ == TokenLParen {
			return SemanticClause{}, false, diagnosticErrorf(clause.At, "semantic clause '%s' does not take arguments", clause.Name)
		}
		return clause, true, nil
	case "budget":
		clause := SemanticClause{At: p.cur.pos, Name: "budget"}
		if err := p.next(); err != nil {
			return SemanticClause{}, false, err
		}
		if _, err := p.expect(TokenLParen); err != nil {
			return SemanticClause{}, false, err
		}
		value, err := p.parseExpr()
		if err != nil {
			return SemanticClause{}, false, err
		}
		if _, err := p.expect(TokenRParen); err != nil {
			return SemanticClause{}, false, err
		}
		clause.Value = value
		return clause, true, nil
	case "consent":
		clause := SemanticClause{At: p.cur.pos, Name: "consent"}
		if err := p.next(); err != nil {
			return SemanticClause{}, false, err
		}
		if _, err := p.expect(TokenLParen); err != nil {
			return SemanticClause{}, false, err
		}
		value, err := p.parseExpr()
		if err != nil {
			return SemanticClause{}, false, err
		}
		if _, err := p.expect(TokenRParen); err != nil {
			return SemanticClause{}, false, err
		}
		clause.Value = value
		return clause, true, nil
	default:
		return SemanticClause{}, false, nil
	}
}

func (p *parser) parseTypeParams() ([]string, error) {
	names, _, err := p.parseTypeParamsWithBounds(false)
	return names, err
}

func (p *parser) parseTypeParamsWithBounds(allowBounds bool) ([]string, []TypeParamBound, error) {
	if p.cur.typ != TokenLess {
		return nil, nil, nil
	}
	if err := p.next(); err != nil {
		return nil, nil, err
	}
	var out []string
	var bounds []TypeParamBound
	seen := map[string]struct{}{}
	for {
		tok, err := p.expect(TokenIdent)
		if err != nil {
			return nil, nil, err
		}
		if _, exists := seen[tok.lit]; exists {
			return nil, nil, diagnosticErrorf(tok.pos, "duplicate type parameter '%s'", tok.lit)
		}
		seen[tok.lit] = struct{}{}
		out = append(out, tok.lit)
		if p.cur.typ == TokenColon {
			if !allowBounds {
				return nil, nil, diagnosticErrorf(p.cur.pos, "generic type parameter bounds are only supported on functions")
			}
			if err := p.next(); err != nil {
				return nil, nil, err
			}
			bound, err := p.parseTypeRef()
			if err != nil {
				return nil, nil, err
			}
			bounds = append(bounds, TypeParamBound{At: tok.pos, Name: tok.lit, Bound: bound})
		}
		if p.cur.typ != TokenComma {
			break
		}
		if err := p.next(); err != nil {
			return nil, nil, err
		}
	}
	if _, err := p.expect(TokenGreater); err != nil {
		return nil, nil, err
	}
	return out, bounds, nil
}

func (p *parser) tryParseCallTypeArgs() ([]TypeRef, bool, error) {
	return p.tryParseTypeArgsBefore(TokenLParen)
}

func (p *parser) tryParseStructLiteralTypeArgs() ([]TypeRef, bool, error) {
	return p.tryParseTypeArgsBefore(TokenLBrace)
}

func (p *parser) tryParseTypeArgsBefore(next TokenType) ([]TypeRef, bool, error) {
	if p.cur.typ != TokenLess {
		return nil, false, nil
	}
	savedLexer := *p.l
	savedCur := p.cur
	savedPeek := p.peek
	restore := func() {
		*p.l = savedLexer
		p.cur = savedCur
		p.peek = savedPeek
	}
	if err := p.next(); err != nil {
		restore()
		return nil, false, err
	}
	var args []TypeRef
	for {
		arg, err := p.parseTypeRef()
		if err != nil {
			restore()
			return nil, false, nil
		}
		args = append(args, arg)
		if p.cur.typ == TokenComma {
			if err := p.next(); err != nil {
				restore()
				return nil, false, err
			}
			continue
		}
		break
	}
	if p.cur.typ != TokenGreater {
		restore()
		return nil, false, nil
	}
	if err := p.next(); err != nil {
		restore()
		return nil, false, err
	}
	if p.cur.typ != next {
		restore()
		return nil, false, nil
	}
	return args, true, nil
}

func (p *parser) parseTypeArgs() ([]TypeRef, error) {
	if _, err := p.expect(TokenLess); err != nil {
		return nil, err
	}
	var args []TypeRef
	for {
		arg, err := p.parseTypeRef()
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
	if _, err := p.expect(TokenGreater); err != nil {
		return nil, err
	}
	return args, nil
}

func (p *parser) parsePath() (string, error) {
	parts, err := p.parsePathParts()
	if err != nil {
		return "", err
	}
	return strings.Join(parts, "."), nil
}

func (p *parser) parseImportPath() (string, []string, error) {
	first, err := p.expectPathPart()
	if err != nil {
		return "", nil, err
	}
	parts := []string{first.lit}
	for p.cur.typ == TokenDot {
		if err := p.next(); err != nil {
			return "", nil, err
		}
		if p.cur.typ == TokenLBrace {
			items, err := p.parseImportItems()
			if err != nil {
				return "", nil, err
			}
			return strings.Join(parts, "."), items, nil
		}
		partTok, err := p.expectPathPart()
		if err != nil {
			return "", nil, err
		}
		parts = append(parts, partTok.lit)
	}
	return strings.Join(parts, "."), nil, nil
}

func (p *parser) parseImportItems() ([]string, error) {
	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}
	var items []string
	seen := map[string]struct{}{}
	for p.cur.typ != TokenRBrace && p.cur.typ != TokenEOF {
		tok, err := p.expect(TokenIdent)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[tok.lit]; exists {
			return nil, diagnosticErrorf(tok.pos, "duplicate selective import '%s'", tok.lit)
		}
		seen[tok.lit] = struct{}{}
		items = append(items, tok.lit)
		if p.cur.typ != TokenComma {
			break
		}
		if err := p.next(); err != nil {
			return nil, err
		}
	}
	if len(items) == 0 {
		return nil, diagnosticErrorf(p.cur.pos, "selective import requires at least one name")
	}
	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	return items, nil
}

func (p *parser) parsePathParts() ([]string, error) {
	first, err := p.expectPathPart()
	if err != nil {
		return nil, err
	}
	parts := []string{first.lit}
	for p.cur.typ == TokenDot {
		if err := p.next(); err != nil {
			return nil, err
		}
		partTok, err := p.expectPathPart()
		if err != nil {
			return nil, err
		}
		parts = append(parts, partTok.lit)
	}
	return parts, nil
}

func (p *parser) expectPathPart() (token, error) {
	switch p.cur.typ {
	case TokenIdent, TokenTest, TokenAsync:
		tok := p.cur
		if err := p.next(); err != nil {
			return token{}, err
		}
		return tok, nil
	default:
		return token{}, p.unexpected("identifier")
	}
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
		ownership := ""
		if p.cur.typ == TokenIdent && isOwnershipMarker(p.cur.lit) {
			ownershipTok := p.cur
			ownership = ownershipTok.lit
			if err := p.next(); err != nil {
				return nil, err
			}
			if p.cur.typ == TokenIdent && isOwnershipMarker(p.cur.lit) {
				return nil, diagnosticErrorf(p.cur.pos, "ownership marker '%s' cannot follow ownership marker '%s'; use exactly one of borrow, inout, or consume before the parameter type", p.cur.lit, ownershipTok.lit)
			}
			if !p.startsTypeRef() {
				return nil, diagnosticErrorf(ownershipTok.pos, "ownership marker '%s' must be followed by a parameter type", ownershipTok.lit)
			}
		}
		typeRef, err := p.parseTypeRef()
		if err != nil {
			return nil, err
		}
		params = append(params, ParamDecl{At: nameTok.pos, Name: nameTok.lit, Type: typeRef, Ownership: ownership})
		if p.cur.typ != TokenComma {
			break
		}
		if err := p.next(); err != nil {
			return nil, err
		}
	}
	return params, nil
}

func isOwnershipMarker(name string) bool {
	return name == "borrow" || name == "inout" || name == "consume"
}

func (p *parser) startsTypeRef() bool {
	return p.cur.typ == TokenIdent || p.cur.typ == TokenFn || p.cur.typ == TokenLBracket
}

func numberExprFromToken(tok token) (*NumberExpr, error) {
	if tok.num > maxI32NumberLiteral {
		return nil, integerLiteralRangeError(tok)
	}
	return &NumberExpr{At: tok.pos, Value: int32(tok.num)}, nil
}

func integerLiteralRangeError(tok token) error {
	return diagnosticErrorf(tok.pos, "integer literal %s exceeds i32 range 0..2147483647", tok.lit)
}

func isFunctionLikeCallee(parts []string) bool {
	if len(parts) == 0 {
		return false
	}
	name := parts[len(parts)-1]
	if name == "" {
		return false
	}
	ch := name[0]
	return ch == '_' || (ch >= 'a' && ch <= 'z')
}

func (p *parser) parseReprStructDecl(public bool) (*StructDecl, error) {
	reprTok := p.cur
	if reprTok.typ != TokenIdent || reprTok.lit != "repr" {
		return nil, p.unexpected("repr")
	}
	if err := p.next(); err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenLParen); err != nil {
		return nil, err
	}
	nameTok, err := p.expect(TokenIdent)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenRParen); err != nil {
		return nil, err
	}
	if nameTok.lit != StructReprC {
		return nil, diagnosticErrorf(nameTok.pos, "unsupported struct representation '%s'", nameTok.lit)
	}
	return p.parseStructDeclWithRepr(public, StructReprC)
}

func (p *parser) parseStructDecl(public bool) (*StructDecl, error) {
	return p.parseStructDeclWithRepr(public, StructReprDefault)
}

func (p *parser) parseStructDeclWithRepr(public bool, repr string) (*StructDecl, error) {
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
	typeParams, err := p.parseTypeParams()
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
	return &StructDecl{At: nameTok.pos, Name: nameTok.lit, TypeParams: typeParams, Repr: repr, Public: public, Fields: fields}, nil
}

func (p *parser) parseEnumDecl(public bool) (*EnumDecl, error) {
	if p.cur.typ != TokenEnum {
		return nil, p.unexpected("enum")
	}
	pos := p.cur.pos
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
	var cases []EnumCaseDecl
	for p.cur.typ != TokenRBrace && p.cur.typ != TokenEOF {
		if p.cur.typ == TokenSemicolon || p.cur.typ == TokenComma {
			if err := p.next(); err != nil {
				return nil, err
			}
			continue
		}
		casePos := p.cur.pos
		if _, err := p.expect(TokenCase); err != nil {
			return nil, err
		}
		nameTok, err := p.expect(TokenIdent)
		if err != nil {
			return nil, err
		}
		payload, hasPayload, err := p.parseEnumCasePayload()
		if err != nil {
			return nil, err
		}
		cases = append(cases, EnumCaseDecl{At: casePos, Name: nameTok.lit, Payload: payload, HasPayload: hasPayload})
		if p.cur.typ == TokenComma || p.cur.typ == TokenSemicolon {
			if err := p.next(); err != nil {
				return nil, err
			}
		}
	}
	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	if len(cases) == 0 {
		return nil, diagnosticErrorf(pos, "enum '%s' must declare at least one case", nameTok.lit)
	}
	return &EnumDecl{At: pos, Name: nameTok.lit, Public: public, Cases: cases}, nil
}

func (p *parser) parseEnumCasePayload() ([]TypeRef, bool, error) {
	if p.cur.typ != TokenLParen {
		return nil, false, nil
	}
	payloadPos := p.cur.pos
	if err := p.next(); err != nil {
		return nil, false, err
	}
	if p.cur.typ == TokenRParen {
		return nil, false, diagnosticErrorf(payloadPos, "enum payload list must contain at least one type")
	}
	var payload []TypeRef
	for {
		typ, err := p.parseTypeRef()
		if err != nil {
			return nil, false, err
		}
		payload = append(payload, typ)
		if p.cur.typ != TokenComma {
			break
		}
		commaPos := p.cur.pos
		if err := p.next(); err != nil {
			return nil, false, err
		}
		if p.cur.typ == TokenRParen {
			return nil, false, diagnosticErrorf(commaPos, "enum payload declaration does not allow a trailing comma")
		}
	}
	if _, err := p.expect(TokenRParen); err != nil {
		return nil, false, err
	}
	return payload, true, nil
}

func (p *parser) parseTypeRef() (TypeRef, error) {
	ref, err := p.parseTypeRefPrimary()
	if err != nil {
		return TypeRef{}, err
	}
	for p.cur.typ == TokenQuestion {
		at := p.cur.pos
		if err := p.next(); err != nil {
			return TypeRef{}, err
		}
		elem := ref
		ref = TypeRef{At: at, Kind: TypeRefOptional, Elem: &elem}
	}
	return ref, nil
}

func (p *parser) parseReturnTypeRef() (TypeRef, string, error) {
	ownership := ""
	if p.cur.typ == TokenIdent && p.cur.lit == "borrow" {
		ownershipTok := p.cur
		ownership = ownershipTok.lit
		if err := p.next(); err != nil {
			return TypeRef{}, "", err
		}
		if p.cur.typ == TokenIdent && isOwnershipMarker(p.cur.lit) {
			return TypeRef{}, "", diagnosticErrorf(p.cur.pos, "ownership marker '%s' cannot follow return ownership marker '%s'; use exactly one return ownership marker before the return type", p.cur.lit, ownershipTok.lit)
		}
		if !p.startsTypeRef() {
			return TypeRef{}, "", diagnosticErrorf(ownershipTok.pos, "expected return type after `borrow`")
		}
	}
	ref, err := p.parseTypeRef()
	if err != nil {
		return TypeRef{}, "", err
	}
	return ref, ownership, nil
}

func (p *parser) parseTypeRefPrimary() (TypeRef, error) {
	if p.cur.typ == TokenFn {
		return p.parseFunctionTypeRef()
	}
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
		if lenTok.num > maxI32NumberLiteral {
			return TypeRef{}, integerLiteralRangeError(lenTok)
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
	typeArgs, err := p.parseOptionalNamedTypeArgs()
	if err != nil {
		return TypeRef{}, err
	}
	return TypeRef{At: first.pos, Kind: TypeRefNamed, Name: strings.Join(parts, "."), TypeArgs: typeArgs}, nil
}

func (p *parser) parseFunctionTypeRef() (TypeRef, error) {
	at := p.cur.pos
	if _, err := p.expect(TokenFn); err != nil {
		return TypeRef{}, err
	}
	if _, err := p.expect(TokenLParen); err != nil {
		return TypeRef{}, err
	}
	params := make([]TypeRef, 0, 2)
	paramOwnership := make([]string, 0, 2)
	for p.cur.typ != TokenRParen {
		ownership := ""
		if p.cur.typ == TokenIdent && isOwnershipMarker(p.cur.lit) {
			ownershipTok := p.cur
			ownership = ownershipTok.lit
			if err := p.next(); err != nil {
				return TypeRef{}, err
			}
			if p.cur.typ == TokenIdent && isOwnershipMarker(p.cur.lit) {
				return TypeRef{}, diagnosticErrorf(p.cur.pos, "ownership marker '%s' cannot follow ownership marker '%s'; use exactly one of borrow, inout, or consume before the function type parameter", p.cur.lit, ownershipTok.lit)
			}
			if !p.startsTypeRef() {
				return TypeRef{}, diagnosticErrorf(ownershipTok.pos, "ownership marker '%s' must be followed by a function type parameter", ownershipTok.lit)
			}
		}
		param, err := p.parseTypeRef()
		if err != nil {
			return TypeRef{}, err
		}
		params = append(params, param)
		paramOwnership = append(paramOwnership, ownership)
		if p.cur.typ != TokenComma {
			break
		}
		commaPos := p.cur.pos
		if err := p.next(); err != nil {
			return TypeRef{}, err
		}
		if p.cur.typ == TokenRParen {
			return TypeRef{}, diagnosticErrorf(commaPos, "function type parameter list does not allow a trailing comma")
		}
	}
	if _, err := p.expect(TokenRParen); err != nil {
		return TypeRef{}, err
	}
	if _, err := p.expect(TokenArrow); err != nil {
		return TypeRef{}, err
	}
	ret, returnOwnership, err := p.parseReturnTypeRef()
	if err != nil {
		return TypeRef{}, err
	}
	var throws *TypeRef
	if p.cur.typ == TokenThrows {
		if err := p.next(); err != nil {
			return TypeRef{}, err
		}
		throwRef, err := p.parseTypeRef()
		if err != nil {
			return TypeRef{}, err
		}
		throws = &throwRef
	}
	uses, clauses, err := p.parseFunctionModifiers()
	if err != nil {
		return TypeRef{}, err
	}
	if len(clauses) > 0 {
		return TypeRef{}, diagnosticErrorf(clauses[0].At, "semantic clauses are not allowed in function types")
	}
	return TypeRef{At: at, Kind: TypeRefFunction, Params: params, ParamOwnership: paramOwnership, Return: &ret, ReturnOwnership: returnOwnership, Throws: throws, Uses: uses}, nil
}

func (p *parser) parseOptionalNamedTypeArgs() ([]TypeRef, error) {
	if p.cur.typ != TokenLess {
		return nil, nil
	}
	return p.parseTypeArgs()
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
	case TokenExpect:
		pos := p.cur.pos
		if err := p.next(); err != nil {
			return nil, err
		}
		cond, err := p.parseExpr()
		if err != nil {
			if p.flowBridged && pos.Col > 1 {
				err = shiftDiagnosticColumn(err, 1-pos.Col)
			}
			return nil, err
		}
		if err := p.consumeOptionalSemicolon(); err != nil {
			return nil, err
		}
		return &ExpectStmt{At: pos, Cond: cond}, nil
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
	case TokenThrow:
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
		return &ThrowStmt{At: pos, Value: expr}, nil
	case TokenBreak:
		pos := p.cur.pos
		if err := p.next(); err != nil {
			return nil, err
		}
		if err := p.consumeOptionalSemicolon(); err != nil {
			return nil, err
		}
		return &BreakStmt{At: pos}, nil
	case TokenContinue:
		pos := p.cur.pos
		if err := p.next(); err != nil {
			return nil, err
		}
		if err := p.consumeOptionalSemicolon(); err != nil {
			return nil, err
		}
		return &ContinueStmt{At: pos}, nil
	case TokenLet, TokenVar, TokenVal, TokenConst:
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
		mutable := declTok.typ == TokenVar
		return &LetStmt{At: pos, Name: nameTok.lit, Type: typeRef, Mutable: mutable, Const: declTok.typ == TokenConst, Value: expr}, nil
	case TokenIf:
		pos := p.cur.pos
		if err := p.next(); err != nil {
			return nil, err
		}
		if p.cur.typ == TokenLet {
			if err := p.next(); err != nil {
				return nil, err
			}
			pattern, err := p.parseMatchPattern()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(TokenAssign); err != nil {
				return nil, err
			}
			value, err := p.parseIfLetValue()
			if err != nil {
				return nil, err
			}
			thenBlock, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			elseBlock, err := p.parseElseBlock()
			if err != nil {
				return nil, err
			}
			name := ""
			if ident, ok := pattern.(*IdentExpr); ok {
				name = ident.Name
				pattern = nil
			}
			return &IfLetStmt{At: pos, Name: name, Pattern: pattern, Value: value, Then: thenBlock, Else: elseBlock}, nil
		}
		cond, err := p.parseConditionExpr()
		if err != nil {
			return nil, err
		}
		thenBlock, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		elseBlock, err := p.parseElseBlock()
		if err != nil {
			return nil, err
		}
		return &IfStmt{At: pos, Cond: cond, Then: thenBlock, Else: elseBlock}, nil
	case TokenWhile:
		pos := p.cur.pos
		if err := p.next(); err != nil {
			return nil, err
		}
		cond, err := p.parseConditionExpr()
		if err != nil {
			return nil, err
		}
		body, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		return &WhileStmt{At: pos, Cond: cond, Body: body}, nil
	case TokenFor:
		return p.parseForRangeStmt()
	case TokenMatch:
		return p.parseMatchStmt()
	case TokenFn, TokenFun:
		pos := p.cur.pos
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.consumeOptionalSemicolon(); err != nil {
			return nil, err
		}
		return &ExprStmt{At: pos, Expr: expr}, nil
	case TokenIdent:
		if p.cur.lit == "defer" {
			return p.parseDeferStmt()
		}
		if feature, ok := plannedFeatureFromToken(p.cur); ok {
			return nil, plannedFeatureError(p.cur.pos, feature)
		}
		if p.cur.lit == "island" && p.peek.typ == TokenLParen {
			return p.parseIslandStmt()
		}
		pos := p.cur.pos
		parts, err := p.parsePathParts()
		if err != nil {
			return nil, err
		}
		typeArgs, _, err := p.tryParseCallTypeArgs()
		if err != nil {
			return nil, err
		}
		if p.cur.typ == TokenLParen {
			name := strings.Join(parts, ".")
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
					if p.cur.typ == TokenAssign {
						return nil, diagnosticErrorf(pos, "cannot assign to struct literal")
					}
					if err := p.consumeOptionalSemicolon(); err != nil {
						return nil, err
					}
					return &ExprStmt{At: pos, Expr: lit}, nil
				}
			}
			args, labels, err := p.parseCallArgs()
			if err != nil {
				return nil, err
			}
			callExpr := &CallExpr{At: pos, Name: name, TypeArgs: typeArgs, Args: args, ArgLabels: labels}
			if p.cur.typ != TokenAssign {
				if err := p.consumeOptionalSemicolon(); err != nil {
					return nil, err
				}
				return &ExprStmt{At: pos, Expr: callExpr}, nil
			}
			return nil, diagnosticErrorf(pos, "cannot assign to function call")
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
		op, opPos, err := p.parseAssignmentOp()
		if err != nil {
			return nil, err
		}
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		compoundValue := Expr(nil)
		if op != 0 {
			compoundValue = value
			value = &BinaryExpr{
				At:    opPos,
				Op:    op,
				Left:  cloneCompoundTarget(expr),
				Right: value,
			}
		}
		if err := p.consumeOptionalSemicolon(); err != nil {
			return nil, err
		}
		return &AssignStmt{At: pos, Target: expr, Value: value, Op: op, CompoundValue: compoundValue}, nil
	default:
		if feature, ok := plannedFeatureFromToken(p.cur); ok {
			return nil, plannedFeatureError(p.cur.pos, feature)
		}
		return nil, p.unexpected("statement")
	}
}

func (p *parser) parseAssignmentOp() (TokenType, Position, error) {
	if p.cur.typ == TokenAssign {
		pos := p.cur.pos
		if err := p.next(); err != nil {
			return 0, Position{}, err
		}
		return 0, pos, nil
	}
	if isCompoundAssignmentOp(p.cur.typ) && p.peek.typ == TokenAssign {
		op := p.cur.typ
		pos := p.cur.pos
		if err := p.next(); err != nil {
			return 0, Position{}, err
		}
		if _, err := p.expect(TokenAssign); err != nil {
			return 0, Position{}, err
		}
		return op, pos, nil
	}
	_, err := p.expect(TokenAssign)
	return 0, Position{}, err
}

func isCompoundAssignmentOp(tt TokenType) bool {
	switch tt {
	case TokenPlus, TokenMinus, TokenStar, TokenSlash, TokenPercent:
		return true
	default:
		return false
	}
}
