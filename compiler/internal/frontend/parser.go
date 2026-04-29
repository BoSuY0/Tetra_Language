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
	if file.Module != "" || len(file.Imports) > 0 || len(file.Globals) > 0 || len(file.Capsules) > 0 {
		return nil, fmt.Errorf("module/import/global/capsule declarations require ParseFile")
	}
	return &Program{
		Capsules:   file.Capsules,
		Enums:      file.Enums,
		Structs:    file.Structs,
		States:     file.States,
		Views:      file.Views,
		Actors:     file.Actors,
		Protocols:  file.Protocols,
		Extensions: file.Extensions,
		Impls:      file.Impls,
		Funcs:      file.Funcs,
		Tests:      file.Tests,
	}, nil
}

func ParseFile(src []byte, filename string) (*FileAST, error) {
	parseSrc, err := canonicalizeFlowSyntax(src, filename)
	if err != nil {
		return nil, err
	}
	return parsePreparedSource(src, parseSrc, filename)
}

// ParseFileDiagnostics performs limited top-level recovery for independent
// release-boundary declarations, then parses the remaining source.
func ParseFileDiagnostics(src []byte, filename string) (*FileAST, []Diagnostic) {
	recovered, diagnostics := recoverTopLevelPlannedFeatures(src, filename)
	if len(diagnostics) == 0 {
		file, err := ParseFile(src, filename)
		if err != nil {
			return nil, []Diagnostic{diagnosticFromError(err)}
		}
		return file, nil
	}
	file, err := ParseFile(recovered, filename)
	if err != nil {
		diagnostics = append(diagnostics, diagnosticFromError(err))
		return nil, diagnostics
	}
	return file, diagnostics
}

func ParseFileWithMigrationNormalization(src []byte, filename string) (*FileAST, error) {
	parseSrc, err := normalizeFlowSyntax(src, filename)
	if err != nil {
		return nil, err
	}
	return parsePreparedSource(src, parseSrc, filename)
}

func parsePreparedSource(raw []byte, parseSrc []byte, filename string) (*FileAST, error) {
	p, err := newParser(parseSrc, filename)
	if err != nil {
		return nil, err
	}
	p.flowBridged = string(raw) != string(parseSrc)
	file, err := p.parseFile()
	if err != nil {
		return nil, err
	}
	file.Path = filename
	file.Src = append([]byte(nil), raw...)
	return file, nil
}

type parser struct {
	l                     *lexer
	cur                   token
	peek                  token
	suppressStructLiteral bool
	flowBridged           bool
	closureSeq            int
	closureDecls          []*FuncDecl
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
	seenEnum := false
	seenState := false
	seenView := false
	seenActor := false
	seenGlobal := false
	seenCapsule := false
	for p.cur.typ != TokenEOF {
		public := false
		if p.cur.typ == TokenPub {
			public = true
			if err := p.next(); err != nil {
				return nil, err
			}
		}
		switch p.cur.typ {
		case TokenSemicolon:
			if public {
				return nil, diagnosticErrorf(p.cur.pos, "pub must be followed by a declaration or import")
			}
			if err := p.next(); err != nil {
				return nil, err
			}
			continue
		case TokenModule:
			if public {
				return nil, diagnosticErrorf(p.cur.pos, "pub cannot apply to module declarations")
			}
			if seenFunc || seenStruct || seenEnum || seenState || seenView || seenActor || seenGlobal || seenCapsule {
				return nil, diagnosticErrorf(p.cur.pos, "module must appear before declarations")
			}
			if file.Module != "" {
				return nil, diagnosticErrorf(p.cur.pos, "duplicate module declaration")
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
			if seenFunc || seenStruct || seenEnum || seenState || seenView || seenActor || seenGlobal || seenCapsule {
				return nil, diagnosticErrorf(p.cur.pos, "import must appear before declarations")
			}
			if err := p.next(); err != nil {
				return nil, err
			}
			path, items, err := p.parseImportPath()
			if err != nil {
				return nil, err
			}
			alias := ""
			if len(items) > 0 {
				if p.cur.typ == TokenAs {
					return nil, diagnosticErrorf(p.cur.pos, "selective imports cannot use module alias")
				}
			} else if p.cur.typ == TokenAs {
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
			file.Imports = append(file.Imports, ImportDecl{At: pos, Path: path, Alias: alias, Items: items, Public: public})
			if err := p.consumeOptionalSemicolon(); err != nil {
				return nil, err
			}
			continue
		case TokenStruct:
			if seenFunc || seenView || seenActor || seenGlobal {
				return nil, diagnosticErrorf(p.cur.pos, "struct must appear before globals/functions")
			}
			seenStruct = true
			st, err := p.parseStructDecl(public)
			if err != nil {
				return nil, err
			}
			file.Structs = append(file.Structs, st)
			continue
		case TokenEnum:
			if seenFunc || seenView || seenActor || seenGlobal {
				return nil, diagnosticErrorf(p.cur.pos, "enum must appear before globals/functions")
			}
			seenEnum = true
			en, err := p.parseEnumDecl(public)
			if err != nil {
				return nil, err
			}
			file.Enums = append(file.Enums, en)
			continue
		case TokenIdent:
			switch p.cur.lit {
			case "state":
				if seenFunc || seenView || seenActor || seenGlobal {
					return nil, diagnosticErrorf(p.cur.pos, "state must appear before views/globals/functions")
				}
				seenState = true
				st, err := p.parseStateDecl(public)
				if err != nil {
					return nil, err
				}
				file.States = append(file.States, st)
				continue
			case "view":
				if seenFunc || seenActor || seenGlobal {
					return nil, diagnosticErrorf(p.cur.pos, "view must appear before globals/functions")
				}
				seenView = true
				view, err := p.parseViewDecl(public)
				if err != nil {
					return nil, err
				}
				file.Views = append(file.Views, view)
				continue
			case "impl":
				if public {
					return nil, diagnosticErrorf(p.cur.pos, "pub cannot apply to impl declarations")
				}
				if seenFunc || seenView || seenActor || seenGlobal {
					return nil, diagnosticErrorf(p.cur.pos, "impl must appear before globals/functions")
				}
				impl, err := p.parseImplDecl()
				if err != nil {
					return nil, err
				}
				file.Impls = append(file.Impls, impl)
				continue
			case "protocol":
				if seenFunc || seenView || seenActor || seenGlobal {
					return nil, diagnosticErrorf(p.cur.pos, "protocol must appear before globals/functions")
				}
				proto, err := p.parseProtocolDecl(public)
				if err != nil {
					return nil, err
				}
				file.Protocols = append(file.Protocols, proto)
				continue
			case "extension":
				if seenFunc || seenView || seenActor || seenGlobal {
					return nil, diagnosticErrorf(p.cur.pos, "extension must appear before globals/functions")
				}
				ext, err := p.parseExtensionDecl(public)
				if err != nil {
					return nil, err
				}
				file.Extensions = append(file.Extensions, ext)
				file.Funcs = append(file.Funcs, ext.Methods...)
				continue
			case "actor":
				if seenFunc || seenView || seenGlobal {
					return nil, diagnosticErrorf(p.cur.pos, "actor must appear before globals/functions")
				}
				seenActor = true
				actor, err := p.parseActorDecl(public)
				if err != nil {
					return nil, err
				}
				file.Actors = append(file.Actors, actor)
				file.Funcs = append(file.Funcs, actor.Methods...)
				continue
			case "closure":
				seenFunc = true
				fn, err := p.parseClosureDecl(public)
				if err != nil {
					return nil, err
				}
				file.Funcs = append(file.Funcs, fn)
				continue
			case "property":
				if seenFunc || seenActor {
					return nil, diagnosticErrorf(p.cur.pos, "global must appear before functions")
				}
				seenGlobal = true
				glob, err := p.parsePropertyDecl(public)
				if err != nil {
					return nil, err
				}
				file.Globals = append(file.Globals, glob)
				continue
			case "capsule":
				if seenFunc || seenGlobal {
					return nil, diagnosticErrorf(p.cur.pos, "capsule must appear before globals/functions")
				}
				seenCapsule = true
				capsule, err := p.parseCapsuleDecl(public)
				if err != nil {
					return nil, err
				}
				file.Capsules = append(file.Capsules, capsule)
				continue
			default:
				if feature, ok := plannedFeatureFromToken(p.cur); ok {
					return nil, plannedFeatureError(p.cur.pos, feature)
				}
				return nil, p.unexpected("module/import/capsule/enum/struct/state/view/extension/var/val/const/fn/closure")
			}
		case TokenVar, TokenVal, TokenConst:
			if seenFunc || seenActor {
				return nil, diagnosticErrorf(p.cur.pos, "global must appear before functions")
			}
			seenGlobal = true
			glob, err := p.parseGlobalDecl(public)
			if err != nil {
				return nil, err
			}
			file.Globals = append(file.Globals, glob)
			continue
		case TokenAt, TokenFn, TokenFun, TokenAsync:
			seenFunc = true
			fn, err := p.parseFuncDecl(public)
			if err != nil {
				return nil, err
			}
			file.Funcs = append(file.Funcs, fn)
		case TokenTest:
			if public {
				return nil, diagnosticErrorf(p.cur.pos, "pub cannot apply to tests")
			}
			seenFunc = true
			test, err := p.parseTestDecl()
			if err != nil {
				return nil, err
			}
			file.Tests = append(file.Tests, test)
		default:
			if feature, ok := plannedFeatureFromToken(p.cur); ok {
				return nil, plannedFeatureError(p.cur.pos, feature)
			}
			return nil, p.unexpected("module/import/capsule/enum/struct/state/view/extension/var/val/fn/closure")
		}
	}
	file.Funcs = append(file.Funcs, p.closureDecls...)
	return file, nil
}

func (p *parser) parseImplDecl() (*ImplDecl, error) {
	pos := p.cur.pos
	if p.cur.typ != TokenIdent || p.cur.lit != "impl" {
		return nil, p.unexpected("impl")
	}
	if err := p.next(); err != nil {
		return nil, err
	}
	typePos := p.cur.pos
	typeParts, err := p.parsePathParts()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenColon); err != nil {
		return nil, err
	}
	protoPos := p.cur.pos
	protoParts, err := p.parsePathParts()
	if err != nil {
		return nil, err
	}
	if err := p.consumeOptionalSemicolon(); err != nil {
		return nil, err
	}
	return &ImplDecl{
		At:       pos,
		Type:     TypeRef{At: typePos, Kind: TypeRefNamed, Name: strings.Join(typeParts, ".")},
		Protocol: TypeRef{At: protoPos, Kind: TypeRefNamed, Name: strings.Join(protoParts, ".")},
	}, nil
}

func (p *parser) parseProtocolDecl(public bool) (*ProtocolDecl, error) {
	pos := p.cur.pos
	if p.cur.typ != TokenIdent || p.cur.lit != "protocol" {
		return nil, p.unexpected("protocol")
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
	var requirements []FuncSigDecl
	seen := map[string]struct{}{}
	for p.cur.typ != TokenRBrace && p.cur.typ != TokenEOF {
		if p.cur.typ == TokenSemicolon {
			if err := p.next(); err != nil {
				return nil, err
			}
			continue
		}
		req, err := p.parseFuncSignatureDecl()
		if err != nil {
			return nil, err
		}
		if _, exists := seen[req.Name]; exists {
			return nil, diagnosticErrorf(req.At, "duplicate protocol requirement '%s'", req.Name)
		}
		seen[req.Name] = struct{}{}
		requirements = append(requirements, req)
	}
	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	if len(requirements) == 0 {
		return nil, diagnosticErrorf(pos, "protocol requires at least one requirement")
	}
	return &ProtocolDecl{At: pos, Name: nameTok.lit, Public: public, Requirements: requirements}, nil
}

func (p *parser) parseFuncSignatureDecl() (FuncSigDecl, error) {
	async := false
	if p.cur.typ == TokenAsync {
		async = true
		if err := p.next(); err != nil {
			return FuncSigDecl{}, err
		}
	}
	if p.cur.typ != TokenFn && p.cur.typ != TokenFun {
		return FuncSigDecl{}, p.unexpected("fn/fun")
	}
	if err := p.next(); err != nil {
		return FuncSigDecl{}, err
	}
	nameTok, err := p.expect(TokenIdent)
	if err != nil {
		return FuncSigDecl{}, err
	}
	typeParams, err := p.parseTypeParams()
	if err != nil {
		return FuncSigDecl{}, err
	}
	if _, err := p.expect(TokenLParen); err != nil {
		return FuncSigDecl{}, err
	}
	params, err := p.parseParams()
	if err != nil {
		return FuncSigDecl{}, err
	}
	if _, err := p.expect(TokenRParen); err != nil {
		return FuncSigDecl{}, err
	}
	switch p.cur.typ {
	case TokenArrow, TokenColon:
		if err := p.next(); err != nil {
			return FuncSigDecl{}, err
		}
	default:
		return FuncSigDecl{}, p.unexpected("-> or :")
	}
	retType, err := p.parseTypeRef()
	if err != nil {
		return FuncSigDecl{}, err
	}
	var throws TypeRef
	hasThrows := false
	if p.cur.typ == TokenThrows {
		hasThrows = true
		if err := p.next(); err != nil {
			return FuncSigDecl{}, err
		}
		parsed, err := p.parseTypeRef()
		if err != nil {
			return FuncSigDecl{}, err
		}
		throws = parsed
	}
	uses, clauses, err := p.parseFunctionModifiers()
	if err != nil {
		return FuncSigDecl{}, err
	}
	if len(clauses) > 0 {
		return FuncSigDecl{}, diagnosticErrorf(clauses[0].At, "semantic clauses are not allowed in protocol requirements")
	}
	if err := p.consumeOptionalSemicolon(); err != nil {
		return FuncSigDecl{}, err
	}
	return FuncSigDecl{At: nameTok.pos, Name: nameTok.lit, TypeParams: typeParams, Async: async, ReturnType: retType, Throws: throws, HasThrows: hasThrows, Params: params, Uses: uses}, nil
}

func (p *parser) parseExtensionDecl(public bool) (*ExtensionDecl, error) {
	pos := p.cur.pos
	if p.cur.typ != TokenIdent || p.cur.lit != "extension" {
		return nil, p.unexpected("extension")
	}
	if err := p.next(); err != nil {
		return nil, err
	}
	target := TypeRef{At: p.cur.pos, Kind: TypeRefNamed}
	parts, err := p.parsePathParts()
	if err != nil {
		return nil, err
	}
	target.Name = strings.Join(parts, ".")
	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}
	var methods []*FuncDecl
	for p.cur.typ != TokenRBrace && p.cur.typ != TokenEOF {
		if p.cur.typ == TokenSemicolon {
			if err := p.next(); err != nil {
				return nil, err
			}
			continue
		}
		fn, err := p.parseFuncDecl(public)
		if err != nil {
			return nil, err
		}
		fn.ExtensionOf = target.Name
		fn.Name = target.Name + "." + fn.Name
		methods = append(methods, fn)
	}
	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	if len(methods) == 0 {
		return nil, diagnosticErrorf(pos, "extension requires at least one method")
	}
	return &ExtensionDecl{At: pos, Target: target, Public: public, Methods: methods}, nil
}

func (p *parser) parseActorDecl(public bool) (*ActorDecl, error) {
	pos := p.cur.pos
	if p.cur.typ != TokenIdent || p.cur.lit != "actor" {
		return nil, p.unexpected("actor")
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
	var fields []StateFieldDecl
	seenFields := map[string]struct{}{}
	var methods []*FuncDecl
	seenMethods := map[string]struct{}{}
	for p.cur.typ != TokenRBrace && p.cur.typ != TokenEOF {
		if p.cur.typ == TokenSemicolon {
			if err := p.next(); err != nil {
				return nil, err
			}
			continue
		}
		switch p.cur.typ {
		case TokenVar, TokenVal, TokenConst:
			field, err := p.parseActorStateField()
			if err != nil {
				return nil, err
			}
			if _, exists := seenFields[field.Name]; exists {
				return nil, diagnosticErrorf(field.At, "duplicate actor state field '%s'", field.Name)
			}
			seenFields[field.Name] = struct{}{}
			fields = append(fields, field)
			continue
		case TokenAt, TokenFn, TokenFun, TokenAsync:
		default:
			return nil, p.unsupportedActorMemberDiagnostic()
		}
		fn, err := p.parseFuncDecl(public)
		if err != nil {
			return nil, err
		}
		for _, param := range fn.Params {
			if param.Name == "self" {
				return nil, diagnosticErrorf(param.At, "actor methods do not support explicit self parameters yet; use core.self() inside the method body")
			}
		}
		if _, exists := seenMethods[fn.Name]; exists {
			return nil, diagnosticErrorf(fn.Pos, "duplicate actor method '%s'", fn.Name)
		}
		seenMethods[fn.Name] = struct{}{}
		fn.Name = nameTok.lit + "." + fn.Name
		methods = append(methods, fn)
	}
	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	if len(methods) == 0 {
		return nil, diagnosticErrorf(pos, "actor requires at least one method")
	}
	return &ActorDecl{
		At:      pos,
		Name:    nameTok.lit,
		Public:  public,
		Fields:  fields,
		Methods: methods,
	}, nil
}

func (p *parser) parseActorStateField() (StateFieldDecl, error) {
	declTok := p.cur
	fieldPos := declTok.pos
	if err := p.next(); err != nil {
		return StateFieldDecl{}, err
	}
	fieldNameTok, err := p.expect(TokenIdent)
	if err != nil {
		return StateFieldDecl{}, err
	}
	if _, err := p.expect(TokenColon); err != nil {
		return StateFieldDecl{}, err
	}
	typeRef, err := p.parseTypeRef()
	if err != nil {
		return StateFieldDecl{}, err
	}
	var init Expr
	if p.cur.typ == TokenAssign {
		if err := p.next(); err != nil {
			return StateFieldDecl{}, err
		}
		initExpr, err := p.parseExpr()
		if err != nil {
			return StateFieldDecl{}, err
		}
		init = initExpr
	}
	if err := p.consumeOptionalSemicolon(); err != nil {
		return StateFieldDecl{}, err
	}
	return StateFieldDecl{
		At:      fieldPos,
		Name:    fieldNameTok.lit,
		Type:    typeRef,
		Mutable: declTok.typ == TokenVar,
		Const:   declTok.typ == TokenConst,
		Init:    init,
	}, nil
}

func (p *parser) unsupportedActorMemberDiagnostic() error {
	if p.cur.typ == TokenIdent {
		switch p.cur.lit {
		case "state":
			return diagnosticErrorf(p.cur.pos, "actor declarations do not support nested state blocks yet; define state outside the actor")
		case "self":
			return diagnosticErrorf(p.cur.pos, "actor declarations do not support self members yet; use core.self() inside a func method")
		default:
			return diagnosticErrorf(p.cur.pos, "actor state fields must use 'val' or 'const'")
		}
	}
	return diagnosticErrorf(p.cur.pos, "actor declarations currently support state fields and func methods only")
}

func (p *parser) parseTestDecl() (*TestDecl, error) {
	pos := p.cur.pos
	if err := p.next(); err != nil {
		return nil, err
	}
	nameTok, err := p.expect(TokenString)
	if err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &TestDecl{At: pos, Name: string(nameTok.str), Body: body}, nil
}

func (p *parser) parseGlobalDecl(public bool) (*GlobalDecl, error) {
	if p.cur.typ != TokenVar && p.cur.typ != TokenVal && p.cur.typ != TokenConst {
		return nil, p.unexpected("var/val/const")
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
		return nil, diagnosticErrorf(p.cur.pos, "global var requires an explicit type annotation")
	}

	var init Expr
	switch declTok.typ {
	case TokenVar:
		if p.cur.typ == TokenAssign {
			if err := p.next(); err != nil {
				return nil, err
			}
			expr, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			init = expr
		}
	case TokenVal, TokenConst:
		if _, err := p.expect(TokenAssign); err != nil {
			return nil, err
		}
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		init = expr
	default:
		return nil, p.unexpected("var/val/const")
	}
	if err := p.consumeOptionalSemicolon(); err != nil {
		return nil, err
	}
	mutable := declTok.typ == TokenVar
	return &GlobalDecl{At: pos, Name: nameTok.lit, Type: typeRef, Mutable: mutable, Const: declTok.typ == TokenConst, Public: public, Init: init}, nil
}

func (p *parser) parsePropertyDecl(public bool) (*GlobalDecl, error) {
	if p.cur.typ != TokenIdent || p.cur.lit != "property" {
		return nil, p.unexpected("property")
	}
	pos := p.cur.pos
	if err := p.next(); err != nil {
		return nil, err
	}
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
	var init Expr
	if p.cur.typ == TokenAssign {
		if err := p.next(); err != nil {
			return nil, err
		}
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		init = expr
	}
	if err := p.consumeOptionalSemicolon(); err != nil {
		return nil, err
	}
	return &GlobalDecl{
		At:     pos,
		Name:   nameTok.lit,
		Type:   typeRef,
		Public: public,
		Init:   init,
	}, nil
}

func (p *parser) parseCapsuleDecl(public bool) (*CapsuleDecl, error) {
	if p.cur.typ != TokenIdent || p.cur.lit != "capsule" {
		return nil, p.unexpected("capsule")
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
	var entries []CapsuleEntryDecl
	for p.cur.typ != TokenRBrace && p.cur.typ != TokenEOF {
		if p.cur.typ == TokenSemicolon {
			if err := p.next(); err != nil {
				return nil, err
			}
			continue
		}
		entryPos := p.cur.pos
		parts, err := p.parsePathParts()
		if err != nil {
			return nil, err
		}
		if p.cur.typ == TokenColon {
			if err := p.next(); err != nil {
				return nil, err
			}
		}
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.consumeOptionalSemicolon(); err != nil {
			return nil, err
		}
		entries = append(entries, CapsuleEntryDecl{
			At:    entryPos,
			Key:   strings.Join(parts, "."),
			Value: value,
		})
	}
	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, diagnosticErrorf(pos, "capsule requires at least one metadata entry")
	}
	return &CapsuleDecl{
		At:      pos,
		Name:    nameTok.lit,
		Public:  public,
		Entries: entries,
	}, nil
}

func (p *parser) parseStateDecl(public bool) (*StateDecl, error) {
	pos := p.cur.pos
	if p.cur.typ != TokenIdent || p.cur.lit != "state" {
		return nil, p.unexpected("state")
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
	var fields []StateFieldDecl
	seen := map[string]struct{}{}
	for p.cur.typ != TokenRBrace && p.cur.typ != TokenEOF {
		if p.cur.typ == TokenSemicolon {
			if err := p.next(); err != nil {
				return nil, err
			}
			continue
		}
		if p.cur.typ != TokenVar && p.cur.typ != TokenVal && p.cur.typ != TokenConst {
			return nil, p.unexpected("var/val/const")
		}
		declTok := p.cur
		fieldPos := p.cur.pos
		if err := p.next(); err != nil {
			return nil, err
		}
		fieldNameTok, err := p.expect(TokenIdent)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[fieldNameTok.lit]; exists {
			return nil, diagnosticErrorf(fieldNameTok.pos, "duplicate state field '%s'", fieldNameTok.lit)
		}
		seen[fieldNameTok.lit] = struct{}{}
		if _, err := p.expect(TokenColon); err != nil {
			return nil, err
		}
		typeRef, err := p.parseTypeRef()
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
		fields = append(fields, StateFieldDecl{
			At:      fieldPos,
			Name:    fieldNameTok.lit,
			Type:    typeRef,
			Mutable: declTok.typ == TokenVar,
			Const:   declTok.typ == TokenConst,
			Init:    value,
		})
	}
	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		return nil, diagnosticErrorf(pos, "state requires at least one field")
	}
	return &StateDecl{At: pos, Name: nameTok.lit, Public: public, Fields: fields}, nil
}

func (p *parser) parseViewDecl(public bool) (*ViewDecl, error) {
	pos := p.cur.pos
	if p.cur.typ != TokenIdent || p.cur.lit != "view" {
		return nil, p.unexpected("view")
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
	stateTok, err := p.expect(TokenIdent)
	if err != nil {
		return nil, err
	}
	if stateTok.lit != "state" {
		return nil, diagnosticErrorf(stateTok.pos, "view parameter must be named 'state'")
	}
	if _, err := p.expect(TokenColon); err != nil {
		return nil, err
	}
	stateRef, err := p.parseTypeRef()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenRParen); err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}

	view := &ViewDecl{At: pos, Name: nameTok.lit, Public: public, StateName: stateRef}
	for p.cur.typ != TokenRBrace && p.cur.typ != TokenEOF {
		if p.cur.typ == TokenSemicolon {
			if err := p.next(); err != nil {
				return nil, err
			}
			continue
		}
		if p.cur.typ != TokenIdent {
			return nil, p.unexpected("bind/event/command/style/accessibility")
		}
		switch p.cur.lit {
		case "bind":
			entryPos := p.cur.pos
			if err := p.next(); err != nil {
				return nil, err
			}
			nameTok, err := p.expect(TokenIdent)
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
			view.Bindings = append(view.Bindings, ViewBindingDecl{At: entryPos, Name: nameTok.lit, Type: typ, Value: value})
		case "event":
			entryPos := p.cur.pos
			if err := p.next(); err != nil {
				return nil, err
			}
			eventTok, err := p.expect(TokenIdent)
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(TokenArrow); err != nil {
				return nil, err
			}
			cmdTok, err := p.expect(TokenIdent)
			if err != nil {
				return nil, err
			}
			if err := p.consumeOptionalSemicolon(); err != nil {
				return nil, err
			}
			view.Events = append(view.Events, ViewEventDecl{At: entryPos, Name: eventTok.lit, Command: cmdTok.lit})
		case "command":
			entryPos := p.cur.pos
			if err := p.next(); err != nil {
				return nil, err
			}
			cmdTok, err := p.expect(TokenIdent)
			if err != nil {
				return nil, err
			}
			body, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			view.Commands = append(view.Commands, ViewCommandDecl{At: entryPos, Name: cmdTok.lit, Body: body})
		case "style":
			entryPos := p.cur.pos
			if err := p.next(); err != nil {
				return nil, err
			}
			nameTok, err := p.expect(TokenIdent)
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
			view.Styles = append(view.Styles, ViewStyleDecl{At: entryPos, Name: nameTok.lit, Type: typ, Value: value})
		case "accessibility", "a11y":
			entryPos := p.cur.pos
			if err := p.next(); err != nil {
				return nil, err
			}
			nameTok, err := p.expect(TokenIdent)
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
			view.Accessibility = append(view.Accessibility, ViewAccessibilityDecl{At: entryPos, Name: nameTok.lit, Type: typ, Value: value})
		default:
			return nil, p.unexpected("bind/event/command/style/accessibility")
		}
	}
	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	if len(view.Commands) == 0 {
		return nil, diagnosticErrorf(pos, "view requires at least one command")
	}
	return view, nil
}

func (p *parser) parseFuncDecl(public bool) (*FuncDecl, error) {
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
			return nil, diagnosticErrorf(attrTok.pos, "unknown attribute '@%s'", nameTok.lit)
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
			return nil, diagnosticErrorf(attrTok.pos, "duplicate @export attribute")
		}
		exportName = string(valTok.str)
		if exportName == "" {
			return nil, diagnosticErrorf(valTok.pos, "@export name must not be empty")
		}
		if err := p.consumeOptionalSemicolon(); err != nil {
			return nil, err
		}
	}

	async := false
	if p.cur.typ == TokenAsync {
		async = true
		if err := p.next(); err != nil {
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

	retType, err := p.parseTypeRef()
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
		if err := p.consumeOptionalSemicolon(); err != nil {
			return nil, err
		}
		body = []Stmt{&ReturnStmt{At: returnPos, Value: expr}}
	} else {
		body, err = p.parseBlock()
		if err != nil {
			return nil, err
		}
	}

	fn := &FuncDecl{
		Pos:             nameTok.pos,
		Name:            nameTok.lit,
		ExportName:      exportName,
		Public:          public,
		Async:           async,
		TypeParams:      typeParams,
		TypeParamBounds: typeParamBounds,
		ReturnType:      retType,
		Throws:          throws,
		HasThrows:       hasThrows,
		Params:          params,
		Uses:            uses,
		SemanticClauses: clauses,
		Body:            body,
	}
	return fn, nil
}

func (p *parser) parseClosureDecl(public bool) (*FuncDecl, error) {
	if p.cur.typ != TokenIdent || p.cur.lit != "closure" {
		return nil, p.unexpected("closure")
	}
	if err := p.next(); err != nil {
		return nil, err
	}

	nameTok, err := p.expect(TokenIdent)
	if err != nil {
		return nil, err
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
	retType, err := p.parseTypeRef()
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
		if err := p.consumeOptionalSemicolon(); err != nil {
			return nil, err
		}
		body = []Stmt{&ReturnStmt{At: returnPos, Value: expr}}
	} else {
		body, err = p.parseBlock()
		if err != nil {
			return nil, err
		}
	}
	return &FuncDecl{
		Pos:             nameTok.pos,
		Name:            nameTok.lit,
		Public:          public,
		TypeParams:      typeParams,
		TypeParamBounds: typeParamBounds,
		ReturnType:      retType,
		Throws:          throws,
		HasThrows:       hasThrows,
		Params:          params,
		Uses:            uses,
		SemanticClauses: clauses,
		Body:            body,
	}, nil
}

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
			ownership = p.cur.lit
			if err := p.next(); err != nil {
				return nil, err
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

func (p *parser) parseStructDecl(public bool) (*StructDecl, error) {
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
	return &StructDecl{At: nameTok.pos, Name: nameTok.lit, TypeParams: typeParams, Public: public, Fields: fields}, nil
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
	for p.cur.typ != TokenRParen {
		param, err := p.parseTypeRef()
		if err != nil {
			return TypeRef{}, err
		}
		params = append(params, param)
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
	ret, err := p.parseTypeRef()
	if err != nil {
		return TypeRef{}, err
	}
	uses, clauses, err := p.parseFunctionModifiers()
	if err != nil {
		return TypeRef{}, err
	}
	if len(clauses) > 0 {
		return TypeRef{}, diagnosticErrorf(clauses[0].At, "semantic clauses are not allowed in function types")
	}
	return TypeRef{At: at, Kind: TypeRefFunction, Params: params, Return: &ret, Uses: uses}, nil
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

func plannedFeatureFromToken(tok token) (string, bool) {
	if tok.typ != TokenIdent {
		return "", false
	}
	return "", false
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
		return &NumberExpr{At: tok.pos, Value: int32(tok.num)}, nil
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
		return p.parsePostfix(&NumberExpr{At: tok.pos, Value: int32(tok.num)})
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
	retType, err := p.parseTypeRef()
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
		TypeParams:      typeParams,
		TypeParamBounds: typeParamBounds,
		ReturnType:      retType,
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
