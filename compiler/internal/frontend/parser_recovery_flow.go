package frontend

import (
	"fmt"
	"strings"
)

func plannedFeatureFromToken(tok token) (string, bool) {
	if tok.typ != TokenIdent {
		return "", false
	}
	switch tok.lit {
	case "class":
		return "class declarations", true
	case "trait":
		return "trait declarations", true
	case "interface":
		return "interface declarations", true
	case "typealias":
		return "type alias declarations", true
	case "macro":
		return "macro declarations", true
	default:
		return "", false
	}
}

func plannedFeatureError(pos Position, feature string) error {
	return diagnosticErrorf(pos, "planned feature '%s' is not implemented in the Tetra v1.0 profile", feature)
}

func recoverTopLevelPlannedFeatures(src []byte, filename string) ([]byte, []Diagnostic) {
	lines := strings.Split(string(src), "\n")
	out := make([]string, len(lines))
	var diagnostics []Diagnostic
	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")
		trimmed := strings.TrimSpace(line)
		out[i] = line
		if trimmed == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) == 0 {
			continue
		}
		featureTok := token{typ: TokenIdent, lit: strings.TrimSuffix(fields[0], ":")}
		feature, ok := plannedFeatureFromToken(featureTok)
		if !ok {
			continue
		}
		msg := fmt.Sprintf("planned feature '%s' is not implemented in the Tetra v1.0 profile", feature)
		diagnostics = append(diagnostics, Diagnostic{
			Code:     DiagnosticCodeParse,
			Message:  msg,
			File:     filename,
			Line:     i + 1,
			Column:   1,
			Severity: "error",
			Hint:     hintForDiagnosticMessage(msg),
		})
		out[i] = ""
		for i+1 < len(lines) {
			next := strings.TrimRight(lines[i+1], "\r")
			nextTrimmed := strings.TrimSpace(next)
			if nextTrimmed == "" {
				i++
				out[i] = ""
				continue
			}
			if strings.HasPrefix(next, " ") || strings.HasPrefix(next, "\t") {
				i++
				out[i] = ""
				continue
			}
			break
		}
	}
	return []byte(strings.Join(out, "\n")), diagnostics
}

func parseFileDiagnosticsWithTopLevelRecovery(src []byte, filename string) (*FileAST, []Diagnostic) {
	lines := strings.Split(string(src), "\n")
	var diagnostics []Diagnostic
	removed := map[int]struct{}{}
	for attempt := 0; attempt <= len(lines); attempt++ {
		file, err := ParseFile([]byte(strings.Join(lines, "\n")), filename)
		if err == nil {
			return file, diagnostics
		}
		diag := adjustDiagnosticForRecoveredSyntheticClose(lines, diagnosticFromError(err))
		diagnostics = append(diagnostics, diag)
		if !isRecoverableTopLevelDiagnostic(diag) {
			return nil, diagnostics
		}
		start, end, ok := topLevelDeclarationSpanForDiagnostic(lines, diagnostics[len(diagnostics)-1])
		if !ok {
			return nil, diagnostics
		}
		if _, exists := removed[start]; exists {
			return nil, diagnostics
		}
		removed[start] = struct{}{}
		blankSourceSpan(lines, start, end)
	}
	return nil, diagnostics
}

func isRecoverableTopLevelDiagnostic(diagnostic Diagnostic) bool {
	return diagnostic.Message != "invalid UTF-8 encoding"
}

func topLevelDeclarationSpanForDiagnostic(lines []string, diagnostic Diagnostic) (int, int, bool) {
	if len(lines) == 0 || diagnostic.Line < 1 {
		return 0, 0, false
	}
	idx := diagnostic.Line - 1
	if idx >= len(lines) {
		idx = len(lines) - 1
	}
	if previousIdx, ok := previousDeclarationLineForSyntheticClose(lines, idx, diagnostic); ok {
		idx = previousIdx
	}
	for ; idx >= 0; idx-- {
		line := strings.TrimRight(lines[idx], "\r")
		if isBlankOrCommentLine(line) {
			continue
		}
		if !isTopLevelSourceLine(line) {
			continue
		}
		if !startsRecoverableTopLevelDeclaration(line) {
			return 0, 0, false
		}
		start := includeLeadingAttributeLines(lines, idx)
		end := topLevelDeclarationEnd(lines, start)
		return start, end, true
	}
	return 0, 0, false
}

func adjustDiagnosticForRecoveredSyntheticClose(lines []string, diagnostic Diagnostic) Diagnostic {
	idx := diagnostic.Line - 1
	if prev, ok := previousDeclarationLineForSyntheticClose(lines, idx, diagnostic); ok {
		diagnostic.Line = prev + 1
		diagnostic.Column = firstSourceColumn(lines[prev])
	}
	return diagnostic
}

func previousDeclarationLineForSyntheticClose(lines []string, idx int, diagnostic Diagnostic) (int, bool) {
	if !strings.Contains(diagnostic.Message, "got }") || idx < 1 || idx >= len(lines) {
		return 0, false
	}
	current := strings.TrimRight(lines[idx], "\r")
	currentIsRecoveryBoundary := isBlankOrCommentLine(current) || (isTopLevelSourceLine(current) && startsRecoverableTopLevelDeclaration(current))
	if !currentIsRecoveryBoundary {
		return 0, false
	}
	for prev := idx - 1; prev >= 0; prev-- {
		line := strings.TrimRight(lines[prev], "\r")
		if isBlankOrCommentLine(line) {
			continue
		}
		if isTopLevelSourceLine(line) {
			return 0, false
		}
		return prev, true
	}
	return 0, false
}

func firstSourceColumn(line string) int {
	for idx, r := range line {
		if r != ' ' && r != '\t' && r != '\r' {
			return idx + 1
		}
	}
	return 1
}

func includeLeadingAttributeLines(lines []string, start int) int {
	for idx := start - 1; idx >= 0; idx-- {
		line := strings.TrimRight(lines[idx], "\r")
		if isBlankOrCommentLine(line) {
			continue
		}
		if !isTopLevelAttributeLine(line) {
			break
		}
		start = idx
	}
	return start
}

func topLevelDeclarationEnd(lines []string, start int) int {
	seenDeclarationHeader := !isTopLevelAttributeLine(lines[start])
	for idx := start + 1; idx < len(lines); idx++ {
		line := strings.TrimRight(lines[idx], "\r")
		if isBlankOrCommentLine(line) {
			continue
		}
		if !isTopLevelSourceLine(line) {
			continue
		}
		if !seenDeclarationHeader {
			if isTopLevelAttributeLine(line) {
				continue
			}
			if startsFunctionLikeTopLevelDeclaration(line) {
				seenDeclarationHeader = true
				continue
			}
		}
		return idx
	}
	return len(lines)
}

func blankSourceSpan(lines []string, start, end int) {
	for idx := start; idx < end && idx < len(lines); idx++ {
		lines[idx] = ""
	}
}

func isBlankOrCommentLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed == "" || strings.HasPrefix(trimmed, "//")
}

func isTopLevelSourceLine(line string) bool {
	return !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t")
}

func isTopLevelAttributeLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "@")
}

func startsRecoverableTopLevelDeclaration(line string) bool {
	trimmed := strings.TrimSpace(strings.TrimSuffix(line, ":"))
	if strings.HasPrefix(trimmed, "pub ") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "pub "))
	}
	if strings.HasPrefix(trimmed, "@") {
		return true
	}
	return startsFunctionLikeTopLevelDeclaration(trimmed) || startsNonFunctionTopLevelDeclaration(trimmed)
}

func startsFunctionLikeTopLevelDeclaration(line string) bool {
	trimmed := strings.TrimSpace(strings.TrimSuffix(line, ":"))
	if strings.HasPrefix(trimmed, "pub ") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "pub "))
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return false
	}
	switch fields[0] {
	case "fn", "fun", "func":
		return true
	case "async":
		return len(fields) > 1 && (fields[1] == "fn" || fields[1] == "fun" || fields[1] == "func")
	default:
		return false
	}
}

func startsNonFunctionTopLevelDeclaration(line string) bool {
	fields := strings.Fields(strings.TrimSpace(strings.TrimSuffix(line, ":")))
	if len(fields) == 0 {
		return false
	}
	switch fields[0] {
	case "module", "import", "capsule", "enum", "struct", "state", "view", "extension", "impl", "protocol", "actor", "var", "val", "const", "closure", "property", "test":
		return true
	default:
		return false
	}
}

func diagnosticFromError(err error) Diagnostic {
	if diag, ok := DiagnosticForError(err); ok {
		return diag
	}
	return Diagnostic{
		Code:     DiagnosticCodeParse,
		Message:  err.Error(),
		Severity: "error",
		Hint:     hintForDiagnosticMessage(err.Error()),
	}
}

func (p *parser) parseIfLetValue() (Expr, error) {
	if p.cur.typ != TokenIdent {
		return p.parseExpr()
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

func (p *parser) parseDeferStmt() (Stmt, error) {
	pos := p.cur.pos
	if err := p.next(); err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &DeferStmt{At: pos, Body: body}, nil
}

func (p *parser) parseForRangeStmt() (Stmt, error) {
	pos := p.cur.pos
	if err := p.next(); err != nil {
		return nil, err
	}
	nameTok, err := p.expect(TokenIdent)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenIn); err != nil {
		return nil, err
	}
	prevSuppress := p.suppressStructLiteral
	p.suppressStructLiteral = true
	start, err := p.parseExpr()
	p.suppressStructLiteral = prevSuppress
	if err != nil {
		return nil, err
	}
	var end Expr
	var iterable Expr
	if p.cur.typ == TokenRangeUntil {
		if err := p.next(); err != nil {
			return nil, err
		}
		end, err = p.parseExpr()
		if err != nil {
			return nil, err
		}
	} else {
		iterable = start
		start = nil
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ForRangeStmt{At: pos, Name: nameTok.lit, Start: start, End: end, Iterable: iterable, Body: body}, nil
}

func (p *parser) parseElseBlock() ([]Stmt, error) {
	if p.cur.typ != TokenElse {
		return nil, nil
	}
	if err := p.next(); err != nil {
		return nil, err
	}
	if p.cur.typ == TokenIf {
		stmt, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		return []Stmt{stmt}, nil
	}
	return p.parseBlock()
}

func (p *parser) parseConditionExpr() (Expr, error) {
	if p.cur.typ == TokenLParen {
		if err := p.next(); err != nil {
			return nil, err
		}
		cond, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
		return cond, nil
	}
	prevSuppress := p.suppressStructLiteral
	p.suppressStructLiteral = true
	cond, err := p.parseExpr()
	p.suppressStructLiteral = prevSuppress
	if err != nil {
		return nil, err
	}
	return cond, nil
}

func (p *parser) parseCallArgs() ([]Expr, []string, error) {
	if p.cur.typ == TokenRParen {
		if _, err := p.expect(TokenRParen); err != nil {
			return nil, nil, err
		}
		return nil, nil, nil
	}
	var args []Expr
	var labels []string
	for {
		label := ""
		if p.cur.typ == TokenIdent && p.peek.typ == TokenColon {
			label = p.cur.lit
			if err := p.next(); err != nil {
				return nil, nil, err
			}
			if _, err := p.expect(TokenColon); err != nil {
				return nil, nil, err
			}
		}
		arg, err := p.parseExpr()
		if err != nil {
			return nil, nil, err
		}
		args = append(args, arg)
		labels = append(labels, label)
		if p.cur.typ != TokenComma {
			break
		}
		if err := p.next(); err != nil {
			return nil, nil, err
		}
	}
	if _, err := p.expect(TokenRParen); err != nil {
		return nil, nil, err
	}
	if allCallLabelsEmpty(labels) {
		return args, nil, nil
	}
	return args, labels, nil
}

func allCallLabelsEmpty(labels []string) bool {
	for _, label := range labels {
		if label != "" {
			return false
		}
	}
	return true
}

func (p *parser) parseMatchStmt() (Stmt, error) {
	pos := p.cur.pos
	if err := p.next(); err != nil {
		return nil, err
	}
	value, err := p.parseMatchValue()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}
	var cases []MatchCase
	for p.cur.typ != TokenRBrace && p.cur.typ != TokenEOF {
		if p.cur.typ == TokenSemicolon {
			if err := p.next(); err != nil {
				return nil, err
			}
			continue
		}
		casePos := p.cur.pos
		if _, err := p.expect(TokenCase); err != nil {
			return nil, err
		}
		isDefault := false
		var pattern Expr
		if p.cur.typ == TokenIdent && p.cur.lit == "_" {
			isDefault = true
			pattern = &IdentExpr{At: p.cur.pos, Name: "_"}
			if err := p.next(); err != nil {
				return nil, err
			}
		} else {
			pattern, err = p.parseMatchPattern()
			if err != nil {
				return nil, err
			}
		}
		var guard Expr
		if p.cur.typ == TokenIf {
			if err := p.next(); err != nil {
				return nil, err
			}
			guard, err = p.parseExpr()
			if err != nil {
				return nil, err
			}
		}
		body, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		cases = append(cases, MatchCase{At: casePos, Pattern: pattern, Guard: guard, Default: isDefault, Body: body})
	}
	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	if len(cases) == 0 {
		return nil, diagnosticErrorf(pos, "match requires at least one case")
	}
	return &MatchStmt{At: pos, Value: value, Cases: cases}, nil
}

func (p *parser) parseMatchExpr() (Expr, error) {
	pos := p.cur.pos
	if err := p.next(); err != nil {
		return nil, err
	}
	value, err := p.parseMatchValue()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}
	var cases []MatchExprCase
	for p.cur.typ != TokenRBrace && p.cur.typ != TokenEOF {
		if p.cur.typ == TokenSemicolon {
			if err := p.next(); err != nil {
				return nil, err
			}
			continue
		}
		casePos := p.cur.pos
		if _, err := p.expect(TokenCase); err != nil {
			return nil, err
		}
		isDefault := false
		var pattern Expr
		if p.cur.typ == TokenIdent && p.cur.lit == "_" {
			isDefault = true
			pattern = &IdentExpr{At: p.cur.pos, Name: "_"}
			if err := p.next(); err != nil {
				return nil, err
			}
		} else {
			pattern, err = p.parseMatchPattern()
			if err != nil {
				return nil, err
			}
		}
		var guard Expr
		if p.cur.typ == TokenIf {
			if err := p.next(); err != nil {
				return nil, err
			}
			guard, err = p.parseExpr()
			if err != nil {
				return nil, err
			}
		}
		value, err := p.parseMatchExprCaseBlock()
		if err != nil {
			return nil, err
		}
		cases = append(cases, MatchExprCase{At: casePos, Pattern: pattern, Guard: guard, Default: isDefault, Value: value})
	}
	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	if len(cases) == 0 {
		return nil, diagnosticErrorf(pos, "match requires at least one case")
	}
	return &MatchExpr{At: pos, Value: value, Cases: cases}, nil
}

func (p *parser) parseMatchExprCaseBlock() (Expr, error) {
	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}
	value, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if err := p.consumeOptionalSemicolon(); err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	return value, nil
}

func (p *parser) parseCatchExpr() (Expr, error) {
	pos := p.cur.pos
	if err := p.next(); err != nil {
		return nil, err
	}
	call, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}
	var cases []CatchExprCase
	for p.cur.typ != TokenRBrace && p.cur.typ != TokenEOF {
		if p.cur.typ == TokenSemicolon {
			if err := p.next(); err != nil {
				return nil, err
			}
			continue
		}
		casePos := p.cur.pos
		if _, err := p.expect(TokenCase); err != nil {
			return nil, err
		}
		isDefault := false
		var pattern Expr
		if p.cur.typ == TokenIdent && p.cur.lit == "_" {
			isDefault = true
			pattern = &IdentExpr{At: p.cur.pos, Name: "_"}
			if err := p.next(); err != nil {
				return nil, err
			}
		} else {
			pattern, err = p.parseMatchPattern()
			if err != nil {
				return nil, err
			}
		}
		var guard Expr
		if p.cur.typ == TokenIf {
			if err := p.next(); err != nil {
				return nil, err
			}
			guard, err = p.parseExpr()
			if err != nil {
				return nil, err
			}
		}
		value, err := p.parseMatchExprCaseBlock()
		if err != nil {
			return nil, err
		}
		cases = append(cases, CatchExprCase{At: casePos, Pattern: pattern, Guard: guard, Default: isDefault, Value: value})
	}
	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	if len(cases) == 0 {
		return nil, diagnosticErrorf(pos, "catch requires at least one case")
	}
	return &CatchExpr{At: pos, Call: call, Cases: cases}, nil
}

func (p *parser) parseMatchValue() (Expr, error) {
	if p.cur.typ == TokenIdent {
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
			if p.cur.typ == TokenIdent && p.peek.typ == TokenColon && !isFunctionLikeCallee(parts) {
				typeRef := TypeRef{At: pos, Kind: TypeRefNamed, Name: name, TypeArgs: typeArgs}
				lit, err := p.parseStructCallLiteral(typeRef)
				if err != nil {
					return nil, err
				}
				return p.parsePostfix(lit)
			}
			args, labels, err := p.parseCallArgs()
			if err != nil {
				return nil, err
			}
			return p.parsePostfix(&CallExpr{At: pos, Name: name, TypeArgs: typeArgs, Args: args, ArgLabels: labels})
		}
		if hasTypeArgs {
			return nil, diagnosticErrorf(pos, "generic type arguments require a call or struct literal")
		}
		return buildFieldAccess(parts, pos), nil
	}
	return p.parseExpr()
}

func (p *parser) parseMatchPattern() (Expr, error) {
	switch p.cur.typ {
	case TokenNumber:
		tok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		return numberExprFromToken(tok)
	case TokenIdent:
		pos := p.cur.pos
		parts, err := p.parsePathParts()
		if err != nil {
			return nil, err
		}
		if p.cur.typ == TokenLParen {
			if len(parts) == 1 && parts[0] == "some" {
				if err := p.next(); err != nil {
					return nil, err
				}
				nameTok, err := p.expectPayloadBindingIdent("some pattern binding")
				if err != nil {
					return nil, err
				}
				if p.cur.typ == TokenComma {
					return nil, diagnosticErrorf(p.cur.pos, "some pattern expects one binding")
				}
				if _, err := p.expect(TokenRParen); err != nil {
					return nil, err
				}
				return &SomePatternExpr{At: pos, Name: nameTok.lit}, nil
			}
			if len(parts) >= 2 {
				bindings, err := p.parseEnumPayloadPatternBindings()
				if err != nil {
					return nil, err
				}
				return &EnumCasePatternExpr{At: pos, TypeName: strings.Join(parts[:len(parts)-1], "."), CaseName: parts[len(parts)-1], Bindings: bindings, HasPayload: true}, nil
			}
			return nil, diagnosticErrorf(pos, "payload match patterns require qualified enum case syntax")
		}
		return buildFieldAccess(parts, pos), nil
	case TokenNone:
		tok := p.cur
		if err := p.next(); err != nil {
			return nil, err
		}
		return &NoneLitExpr{At: tok.pos}, nil
	default:
		return nil, p.unexpected("match pattern")
	}
}

func (p *parser) parseEnumPayloadPatternBindings() ([]string, error) {
	payloadPos := p.cur.pos
	if err := p.next(); err != nil {
		return nil, err
	}
	if p.cur.typ == TokenRParen {
		return nil, diagnosticErrorf(payloadPos, "enum payload pattern requires at least one binding")
	}
	var bindings []string
	seenBindings := map[string]struct{}{}
	for {
		nameTok, err := p.expectPayloadBindingIdent("enum payload pattern binding")
		if err != nil {
			return nil, err
		}
		if _, exists := seenBindings[nameTok.lit]; exists {
			return nil, diagnosticErrorf(nameTok.pos, "duplicate enum payload binding '%s'", nameTok.lit)
		}
		seenBindings[nameTok.lit] = struct{}{}
		bindings = append(bindings, nameTok.lit)
		if p.cur.typ != TokenComma {
			break
		}
		commaPos := p.cur.pos
		if err := p.next(); err != nil {
			return nil, err
		}
		if p.cur.typ == TokenRParen {
			return nil, diagnosticErrorf(commaPos, "enum payload pattern does not allow a trailing comma")
		}
	}
	if _, err := p.expect(TokenRParen); err != nil {
		return nil, err
	}
	return bindings, nil
}

func (p *parser) expectPayloadBindingIdent(context string) (token, error) {
	tok, err := p.expect(TokenIdent)
	if err != nil {
		return token{}, err
	}
	if tok.lit == "_" {
		return token{}, diagnosticErrorf(tok.pos, "%s must be a named identifier", context)
	}
	return tok, nil
}
