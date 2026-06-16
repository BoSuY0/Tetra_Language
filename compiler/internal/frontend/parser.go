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

// ParseFileDiagnostics performs limited top-level recovery for independent Flow
// declarations, then parses the remaining source.
func ParseFileDiagnostics(src []byte, filename string) (*FileAST, []Diagnostic) {
	recovered, diagnostics := recoverTopLevelPlannedFeatures(src, filename)
	file, recoveredDiagnostics := parseFileDiagnosticsWithTopLevelRecovery(recovered, filename)
	diagnostics = append(diagnostics, recoveredDiagnostics...)
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

const (
	maxI32NumberLiteral = int64(1<<31 - 1)
	minI32NumberLiteral = int32(-1 << 31)
)

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
			case "repr":
				if seenFunc || seenView || seenActor || seenGlobal {
					return nil, diagnosticErrorf(p.cur.pos, "struct must appear before globals/functions")
				}
				seenStruct = true
				st, err := p.parseReprStructDecl(public)
				if err != nil {
					return nil, err
				}
				file.Structs = append(file.Structs, st)
				continue
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
	retType, returnOwnership, err := p.parseReturnTypeRef()
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
	return FuncSigDecl{At: nameTok.pos, Name: nameTok.lit, TypeParams: typeParams, Async: async, ReturnType: retType, ReturnOwnership: returnOwnership, Throws: throws, HasThrows: hasThrows, Params: params, Uses: uses}, nil
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
	seen := map[string]struct{}{}
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
		key := strings.Join(parts, ".")
		if _, ok := seen[key]; ok {
			return nil, diagnosticErrorf(entryPos, "duplicate capsule metadata key '%s'", key)
		}
		seen[key] = struct{}{}
		if err := p.consumeOptionalSemicolon(); err != nil {
			return nil, err
		}
		entries = append(entries, CapsuleEntryDecl{
			At:    entryPos,
			Key:   key,
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
		ReturnOwnership: returnOwnership,
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
	}, nil
}
