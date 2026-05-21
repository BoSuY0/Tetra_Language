package semantics

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
)

type genericDef struct {
	module       string
	file         *frontend.FileAST
	imports      map[string]string
	decl         *frontend.FuncDecl
	conformances map[protocolConformanceKey]frontend.Position
	protocols    map[string]genericProtocolInfo
	knownTypes   map[string]struct{}
}

type genericStructDef struct {
	module string
	file   *frontend.FileAST
	decl   *frontend.StructDecl
}

type genericWorkItem struct {
	fn      *frontend.FuncDecl
	module  string
	imports map[string]string
}

type protocolConformanceKey struct {
	typeName string
	protocol string
}

type genericProtocolInfo struct {
	module string
	public bool
}

func monomorphizeFuncDeclFullName(module string, fn *frontend.FuncDecl) string {
	if fn != nil && fn.ExtensionOf != "" {
		return fn.Name
	}
	if fn == nil {
		return qualifyName(module, "")
	}
	return qualifyName(module, fn.Name)
}

func genericInstanceFullName(generic genericDef, name string) string {
	if generic.decl != nil && generic.decl.ExtensionOf != "" {
		return name
	}
	return qualifyName(generic.module, name)
}

func monomorphizeGenerics(world *module.World) error {
	if err := normalizeExtensionMethodNames(world); err != nil {
		return err
	}
	fileImports := make(map[*frontend.FileAST]map[string]string, len(world.Files))
	generics := map[string]genericDef{}
	for _, file := range world.Files {
		imports, err := collectImportAliases(file)
		if err != nil {
			return err
		}
		fileImports[file] = imports
		for _, fn := range file.Funcs {
			if len(fn.TypeParams) > 0 {
				fullName := monomorphizeFuncDeclFullName(file.Module, fn)
				generics[fullName] = genericDef{
					module:  file.Module,
					file:    file,
					imports: imports,
					decl:    fn,
				}
			}
		}
	}
	conformances, err := collectProtocolConformances(world, fileImports)
	if err != nil {
		return err
	}
	protocols := collectGenericProtocolInfos(world)
	knownTypes := collectGenericKnownTypes(world)
	for name, def := range generics {
		def.conformances = conformances
		def.protocols = protocols
		def.knownTypes = knownTypes
		generics[name] = def
	}
	structCtx := newGenericStructContext(world, fileImports)
	if structCtx != nil {
		if err := structCtx.rewriteWorld(world); err != nil {
			return err
		}
	}
	if len(generics) == 0 {
		if structCtx != nil {
			structCtx.finalize(world)
		}
		return nil
	}

	funcDecls := map[string]*frontend.FuncDecl{}
	structDecls := map[string]*frontend.StructDecl{}
	enumDecls := map[string]*frontend.EnumDecl{}
	for _, file := range world.Files {
		for _, enum := range file.Enums {
			enumDecls[qualifyName(file.Module, enum.Name)] = enum
		}
		for _, st := range file.Structs {
			if len(st.TypeParams) == 0 {
				structDecls[qualifyName(file.Module, st.Name)] = st
			}
		}
		for _, fn := range file.Funcs {
			funcDecls[monomorphizeFuncDeclFullName(file.Module, fn)] = fn
		}
	}
	if structCtx != nil {
		for name, st := range structCtx.created {
			structDecls[name] = st
		}
	}
	created := map[string]*frontend.FuncDecl{}
	createdByFile := map[*frontend.FileAST]map[string]*frontend.FuncDecl{}
	var work []genericWorkItem
	for _, file := range world.Files {
		imports := fileImports[file]
		for _, fn := range file.Funcs {
			if len(fn.TypeParams) > 0 {
				continue
			}
			work = append(work, genericWorkItem{fn: fn, module: file.Module, imports: imports})
		}
	}
	for _, file := range world.Files {
		imports := fileImports[file]
		for _, glob := range file.Globals {
			if err := monomorphizeFunctionTypedGlobalInitializer(glob, generics, created, createdByFile, &work, fileImports, file.Module, imports, structCtx); err != nil {
				return err
			}
		}
	}

	for i := 0; i < len(work); i++ {
		item := work[i]
		env := map[string]string{}
		for _, param := range item.fn.Params {
			resolved, err := resolveTypeName(&param.Type, item.module, item.imports)
			if err != nil {
				return err
			}
			env[param.Name] = resolved
		}
		if err := monomorphizeStmts(item.fn.Body, env, map[string]frontend.TypeRef{}, item.fn.ReturnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, &work, fileImports, item.module, item.imports, structCtx); err != nil {
			return err
		}
	}

	for _, file := range world.Files {
		perFile := createdByFile[file]
		if len(perFile) == 0 {
			continue
		}
		names := make([]string, 0, len(perFile))
		for name := range perFile {
			names = append(names, name)
		}
		sort.Strings(names)
		generated := make([]*frontend.FuncDecl, 0, len(names))
		for _, name := range names {
			generated = append(generated, perFile[name])
		}
		file.Funcs = append(generated, file.Funcs...)
	}
	if structCtx != nil {
		structCtx.finalize(world)
	}
	return nil
}

func monomorphizeFunctionTypedGlobalInitializer(
	glob *frontend.GlobalDecl,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) error {
	if glob == nil || glob.Type.Kind != frontend.TypeRefFunction || glob.Init == nil {
		return nil
	}
	original := glob.Init
	replacement, specialized, err := monomorphizeGenericFunctionValueExpr(glob.Init, glob.Type, fmt.Sprintf("function-typed global '%s'", glob.Name), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
	if err != nil || !specialized {
		return err
	}
	if id, ok := replacement.(*frontend.IdentExpr); ok {
		switch init := original.(type) {
		case *frontend.FieldAccessExpr:
			glob.Init = &frontend.FieldAccessExpr{At: init.At, Base: init.Base, Field: lastNameSegment(id.Name)}
			return nil
		case *frontend.IdentExpr:
			name := id.Name
			if module != "" {
				name = strings.TrimPrefix(name, module+".")
			}
			glob.Init = &frontend.IdentExpr{At: init.At, Name: name}
			return nil
		}
	}
	glob.Init = replacement
	return nil
}

func lastNameSegment(name string) string {
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

func collectProtocolConformances(world *module.World, fileImports map[*frontend.FileAST]map[string]string) (map[protocolConformanceKey]frontend.Position, error) {
	conformances := map[protocolConformanceKey]frontend.Position{}
	for _, file := range world.Files {
		imports := fileImports[file]
		for _, impl := range file.Impls {
			typeName, err := resolveTypeName(&impl.Type, file.Module, imports)
			if err != nil {
				return nil, err
			}
			protoName, err := resolveTypeName(&impl.Protocol, file.Module, imports)
			if err != nil {
				return nil, err
			}
			conformances[protocolConformanceKey{typeName: typeName, protocol: protoName}] = impl.At
		}
	}
	return conformances, nil
}

func collectGenericProtocolInfos(world *module.World) map[string]genericProtocolInfo {
	protocols := map[string]genericProtocolInfo{}
	for _, file := range world.Files {
		for _, proto := range file.Protocols {
			fullName := qualifyName(file.Module, proto.Name)
			protocols[fullName] = genericProtocolInfo{
				module: file.Module,
				public: declarationIsPublic(file, proto.Public),
			}
		}
	}
	return protocols
}

func collectGenericKnownTypes(world *module.World) map[string]struct{} {
	known := map[string]struct{}{}
	for name := range baseTypes() {
		known[name] = struct{}{}
	}
	for _, file := range world.Files {
		for _, st := range file.Structs {
			known[qualifyName(file.Module, st.Name)] = struct{}{}
		}
		for _, st := range file.States {
			known[qualifyName(file.Module, st.Name)] = struct{}{}
		}
		for _, en := range file.Enums {
			known[qualifyName(file.Module, en.Name)] = struct{}{}
		}
	}
	return known
}

func monomorphizeGenericStructs(world *module.World, fileImports map[*frontend.FileAST]map[string]string) error {
	ctx := newGenericStructContext(world, fileImports)
	if ctx == nil {
		return nil
	}
	if err := ctx.rewriteWorld(world); err != nil {
		return err
	}
	ctx.finalize(world)
	return nil
}

func newGenericStructContext(world *module.World, fileImports map[*frontend.FileAST]map[string]string) *genericStructContext {
	templates := map[string]genericStructDef{}
	for _, file := range world.Files {
		for _, st := range file.Structs {
			if len(st.TypeParams) == 0 {
				continue
			}
			fullName := qualifyName(file.Module, st.Name)
			templates[fullName] = genericStructDef{module: file.Module, file: file, decl: st}
		}
	}
	if len(templates) == 0 {
		return nil
	}

	created := map[string]*frontend.StructDecl{}
	createdByFile := map[*frontend.FileAST]map[string]*frontend.StructDecl{}
	return &genericStructContext{templates: templates, created: created, createdByFile: createdByFile, fileImports: fileImports}
}

func (ctx *genericStructContext) rewriteWorld(world *module.World) error {
	for _, file := range world.Files {
		imports := ctx.fileImports[file]
		for _, st := range file.Structs {
			if len(st.TypeParams) > 0 {
				continue
			}
			for i := range st.Fields {
				if err := ctx.rewriteTypeRef(&st.Fields[i].Type, file.Module, imports); err != nil {
					return err
				}
			}
		}
		for _, enum := range file.Enums {
			for i := range enum.Cases {
				for j := range enum.Cases[i].Payload {
					if err := ctx.rewriteTypeRef(&enum.Cases[i].Payload[j], file.Module, imports); err != nil {
						return err
					}
				}
			}
		}
		for _, glob := range file.Globals {
			if err := ctx.rewriteTypeRef(&glob.Type, file.Module, imports); err != nil {
				return err
			}
			if err := ctx.rewriteExpr(glob.Init, file.Module, imports); err != nil {
				return err
			}
		}
		for _, fn := range file.Funcs {
			if len(fn.TypeParams) > 0 {
				continue
			}
			if err := ctx.rewriteTypeRef(&fn.ReturnType, file.Module, imports); err != nil {
				return err
			}
			if fn.HasThrows {
				if err := ctx.rewriteTypeRef(&fn.Throws, file.Module, imports); err != nil {
					return err
				}
			}
			for i := range fn.Params {
				if err := ctx.rewriteTypeRef(&fn.Params[i].Type, file.Module, imports); err != nil {
					return err
				}
			}
			if err := ctx.rewriteStmts(fn.Body, file.Module, imports); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ctx *genericStructContext) finalize(world *module.World) {
	for _, file := range world.Files {
		kept := file.Structs[:0]
		for _, st := range file.Structs {
			if len(st.TypeParams) == 0 {
				kept = append(kept, st)
			}
		}
		perFile := ctx.createdByFile[file]
		if len(perFile) > 0 {
			names := make([]string, 0, len(perFile))
			for name := range perFile {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				kept = append(kept, perFile[name])
			}
		}
		file.Structs = kept
	}
}

type genericStructContext struct {
	templates     map[string]genericStructDef
	created       map[string]*frontend.StructDecl
	createdByFile map[*frontend.FileAST]map[string]*frontend.StructDecl
	fileImports   map[*frontend.FileAST]map[string]string
}

func (ctx *genericStructContext) rewriteTypeRef(ref *frontend.TypeRef, module string, imports map[string]string) error {
	if ref == nil {
		return nil
	}
	switch ref.Kind {
	case frontend.TypeRefSlice, frontend.TypeRefArray, frontend.TypeRefOptional:
		return ctx.rewriteTypeRef(ref.Elem, module, imports)
	case frontend.TypeRefFunction:
		for i := range ref.Params {
			if err := ctx.rewriteTypeRef(&ref.Params[i], module, imports); err != nil {
				return err
			}
		}
		return ctx.rewriteTypeRef(ref.Return, module, imports)
	case frontend.TypeRefNamed:
		for i := range ref.TypeArgs {
			if err := ctx.rewriteTypeRef(&ref.TypeArgs[i], module, imports); err != nil {
				return err
			}
		}
		base := *ref
		base.TypeArgs = nil
		resolved, err := resolveTypeName(&base, module, imports)
		if err != nil {
			return err
		}
		generic, ok := ctx.templates[resolved]
		if !ok {
			if len(ref.TypeArgs) > 0 {
				return fmt.Errorf("%s: type '%s' does not accept type arguments", frontend.FormatPos(ref.At), ref.Name)
			}
			return nil
		}
		if len(ref.TypeArgs) == 0 {
			return fmt.Errorf("%s: generic struct '%s' requires %d type argument(s)", frontend.FormatPos(ref.At), ref.Name, len(generic.decl.TypeParams))
		}
		if len(ref.TypeArgs) != len(generic.decl.TypeParams) {
			return fmt.Errorf("%s: generic struct '%s' expects %d type argument, got %d", frontend.FormatPos(ref.At), ref.Name, len(generic.decl.TypeParams), len(ref.TypeArgs))
		}
		subst := map[string]string{}
		for i, tp := range generic.decl.TypeParams {
			if containsFunctionTypeRef(ref.TypeArgs[i]) {
				return fmt.Errorf("%s: generic struct '%s' type argument '%s' uses function type; generic struct instantiation cannot carry function-typed values under the supported fnptr ABI", frontend.FormatPos(ref.TypeArgs[i].At), ref.Name, tp)
			}
			argName, err := resolveTypeName(&ref.TypeArgs[i], module, imports)
			if err != nil {
				return err
			}
			subst[tp] = argName
		}
		name, err := ctx.instantiate(generic, subst, ref.At)
		if err != nil {
			return err
		}
		ref.Name = qualifyName(generic.module, name)
		ref.TypeArgs = nil
		return nil
	default:
		return nil
	}
}

func (ctx *genericStructContext) instantiate(generic genericStructDef, subst map[string]string, at frontend.Position) (string, error) {
	name := mangleGenericName(generic.decl.Name, generic.decl.TypeParams, subst)
	fullName := qualifyName(generic.module, name)
	if existing, exists := ctx.created[fullName]; exists {
		if hasGenericStructTemplateRefs(existing) {
			return "", fmt.Errorf("%s: nested generic struct instantiation for '%s' is not supported in this MVP", frontend.FormatPos(at), displayTypeName(fullName, generic.module))
		}
		return name, nil
	}
	clone := *generic.decl
	clone.Name = name
	clone.TypeParams = nil
	clone.Fields = make([]frontend.FieldDecl, len(generic.decl.Fields))
	ctx.created[fullName] = &clone
	if _, ok := ctx.createdByFile[generic.file]; !ok {
		ctx.createdByFile[generic.file] = map[string]*frontend.StructDecl{}
	}
	ctx.createdByFile[generic.file][name] = &clone
	imports := ctx.fileImports[generic.file]
	for i, field := range generic.decl.Fields {
		clone.Fields[i] = field
		clone.Fields[i].Type = substituteTypeRef(field.Type, subst)
		if hasGenericTypeArgs(clone.Fields[i].Type) {
			return "", fmt.Errorf("%s: nested generic struct instantiation for '%s.%s' is not supported in this MVP", frontend.FormatPos(clone.Fields[i].At), displayTypeName(fullName, generic.module), clone.Fields[i].Name)
		}
		if err := ctx.rewriteTypeRef(&clone.Fields[i].Type, generic.module, imports); err != nil {
			return "", err
		}
	}
	if hasGenericStructTemplateRefs(&clone) {
		return "", fmt.Errorf("%s: nested generic struct instantiation for '%s' is not supported in this MVP", frontend.FormatPos(at), displayTypeName(fullName, generic.module))
	}
	return name, nil
}

func hasGenericStructTemplateRefs(st *frontend.StructDecl) bool {
	if st == nil {
		return false
	}
	for i := range st.Fields {
		if hasGenericTypeArgs(st.Fields[i].Type) {
			return true
		}
	}
	return false
}

func hasGenericTypeArgs(ref frontend.TypeRef) bool {
	if len(ref.TypeArgs) > 0 {
		return true
	}
	if ref.Elem != nil && hasGenericTypeArgs(*ref.Elem) {
		return true
	}
	if ref.Return != nil && hasGenericTypeArgs(*ref.Return) {
		return true
	}
	for i := range ref.Params {
		if hasGenericTypeArgs(ref.Params[i]) {
			return true
		}
	}
	return false
}

func hasDirectFunctionTypeRef(ref frontend.TypeRef) bool {
	return ref.Kind == frontend.TypeRefFunction
}

func containsFunctionTypeRef(ref frontend.TypeRef) bool {
	if ref.Kind == frontend.TypeRefFunction {
		return true
	}
	if ref.Elem != nil && containsFunctionTypeRef(*ref.Elem) {
		return true
	}
	if ref.Return != nil && containsFunctionTypeRef(*ref.Return) {
		return true
	}
	for i := range ref.Params {
		if containsFunctionTypeRef(ref.Params[i]) {
			return true
		}
	}
	for i := range ref.TypeArgs {
		if containsFunctionTypeRef(ref.TypeArgs[i]) {
			return true
		}
	}
	return false
}

func (ctx *genericStructContext) rewriteStmts(stmts []frontend.Stmt, module string, imports map[string]string) error {
	for _, stmt := range stmts {
		if err := ctx.rewriteStmt(stmt, module, imports); err != nil {
			return err
		}
	}
	return nil
}

func (ctx *genericStructContext) rewriteStmt(stmt frontend.Stmt, module string, imports map[string]string) error {
	switch s := stmt.(type) {
	case *frontend.ReturnStmt:
		return ctx.rewriteExpr(s.Value, module, imports)
	case *frontend.ThrowStmt:
		return ctx.rewriteExpr(s.Value, module, imports)
	case *frontend.DeferStmt:
		return ctx.rewriteStmts(s.Body, module, imports)
	case *frontend.PrintStmt:
		return ctx.rewriteExpr(s.Value, module, imports)
	case *frontend.ExpectStmt:
		return ctx.rewriteExpr(s.Cond, module, imports)
	case *frontend.FreeStmt:
		return ctx.rewriteExpr(s.Value, module, imports)
	case *frontend.LetStmt:
		if s.Type.Name != "" || s.Type.Elem != nil {
			if err := ctx.rewriteTypeRef(&s.Type, module, imports); err != nil {
				return err
			}
		}
		return ctx.rewriteExpr(s.Value, module, imports)
	case *frontend.AssignStmt:
		if err := ctx.rewriteExpr(s.Target, module, imports); err != nil {
			return err
		}
		if err := ctx.rewriteExpr(s.Value, module, imports); err != nil {
			return err
		}
		return ctx.rewriteExpr(s.CompoundValue, module, imports)
	case *frontend.IfStmt:
		if err := ctx.rewriteExpr(s.Cond, module, imports); err != nil {
			return err
		}
		if err := ctx.rewriteStmts(s.Then, module, imports); err != nil {
			return err
		}
		return ctx.rewriteStmts(s.Else, module, imports)
	case *frontend.IfLetStmt:
		if err := ctx.rewriteExpr(s.Pattern, module, imports); err != nil {
			return err
		}
		if err := ctx.rewriteExpr(s.Value, module, imports); err != nil {
			return err
		}
		if err := ctx.rewriteStmts(s.Then, module, imports); err != nil {
			return err
		}
		return ctx.rewriteStmts(s.Else, module, imports)
	case *frontend.WhileStmt:
		if err := ctx.rewriteExpr(s.Cond, module, imports); err != nil {
			return err
		}
		return ctx.rewriteStmts(s.Body, module, imports)
	case *frontend.ForRangeStmt:
		if err := ctx.rewriteExpr(s.Start, module, imports); err != nil {
			return err
		}
		if err := ctx.rewriteExpr(s.End, module, imports); err != nil {
			return err
		}
		if err := ctx.rewriteExpr(s.Iterable, module, imports); err != nil {
			return err
		}
		return ctx.rewriteStmts(s.Body, module, imports)
	case *frontend.MatchStmt:
		if err := ctx.rewriteExpr(s.Value, module, imports); err != nil {
			return err
		}
		for i := range s.Cases {
			if err := ctx.rewriteExpr(s.Cases[i].Pattern, module, imports); err != nil {
				return err
			}
			if err := ctx.rewriteExpr(s.Cases[i].Guard, module, imports); err != nil {
				return err
			}
			if err := ctx.rewriteStmts(s.Cases[i].Body, module, imports); err != nil {
				return err
			}
		}
		return nil
	case *frontend.UnsafeStmt:
		return ctx.rewriteStmts(s.Body, module, imports)
	case *frontend.IslandStmt:
		if err := ctx.rewriteExpr(s.Size, module, imports); err != nil {
			return err
		}
		return ctx.rewriteStmts(s.Body, module, imports)
	case *frontend.ExprStmt:
		return ctx.rewriteExpr(s.Expr, module, imports)
	default:
		return nil
	}
}

func (ctx *genericStructContext) rewriteExpr(expr frontend.Expr, module string, imports map[string]string) error {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *frontend.BinaryExpr:
		if err := ctx.rewriteExpr(e.Left, module, imports); err != nil {
			return err
		}
		return ctx.rewriteExpr(e.Right, module, imports)
	case *frontend.UnaryExpr:
		return ctx.rewriteExpr(e.X, module, imports)
	case *frontend.TryExpr:
		return ctx.rewriteExpr(e.X, module, imports)
	case *frontend.AwaitExpr:
		return ctx.rewriteExpr(e.X, module, imports)
	case *frontend.CatchExpr:
		if err := ctx.rewriteExpr(e.Call, module, imports); err != nil {
			return err
		}
		for i := range e.Cases {
			if err := ctx.rewriteExpr(e.Cases[i].Pattern, module, imports); err != nil {
				return err
			}
			if err := ctx.rewriteExpr(e.Cases[i].Guard, module, imports); err != nil {
				return err
			}
			if err := ctx.rewriteExpr(e.Cases[i].Value, module, imports); err != nil {
				return err
			}
		}
		return nil
	case *frontend.MatchExpr:
		if err := ctx.rewriteExpr(e.Value, module, imports); err != nil {
			return err
		}
		for i := range e.Cases {
			if err := ctx.rewriteExpr(e.Cases[i].Pattern, module, imports); err != nil {
				return err
			}
			if err := ctx.rewriteExpr(e.Cases[i].Guard, module, imports); err != nil {
				return err
			}
			if err := ctx.rewriteExpr(e.Cases[i].Value, module, imports); err != nil {
				return err
			}
		}
		return nil
	case *frontend.CallExpr:
		for i := range e.TypeArgs {
			if err := ctx.rewriteTypeRef(&e.TypeArgs[i], module, imports); err != nil {
				return err
			}
		}
		for _, arg := range e.Args {
			if err := ctx.rewriteExpr(arg, module, imports); err != nil {
				return err
			}
		}
		return nil
	case *frontend.StructLitExpr:
		if err := ctx.rewriteTypeRef(&e.Type, module, imports); err != nil {
			return err
		}
		for _, field := range e.Fields {
			if err := ctx.rewriteExpr(field.Value, module, imports); err != nil {
				return err
			}
		}
		return nil
	case *frontend.FieldAccessExpr:
		return ctx.rewriteExpr(e.Base, module, imports)
	case *frontend.IndexExpr:
		if err := ctx.rewriteExpr(e.Base, module, imports); err != nil {
			return err
		}
		return ctx.rewriteExpr(e.Index, module, imports)
	default:
		return nil
	}
}

func monomorphizeStmts(
	stmts []frontend.Stmt,
	env map[string]string,
	functionLocals map[string]frontend.TypeRef,
	returnType frontend.TypeRef,
	funcDecls map[string]*frontend.FuncDecl,
	structDecls map[string]*frontend.StructDecl,
	enumDecls map[string]*frontend.EnumDecl,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			if returnType.Kind == frontend.TypeRefFunction {
				if closure, ok := s.Value.(*frontend.ClosureExpr); ok && closure != nil && closure.Decl != nil && len(closure.Decl.TypeParams) > 0 {
					outerLocals := monomorphizeEnvLocals(env)
					if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
						return unsupportedGenericClosureCaptureError(pos, name)
					}
					replacement, specialized, err := monomorphizeGenericClosureLiteralValue(closure, returnType, "function return", generics, created, createdByFile, work, fileImports, module, imports, structCtx)
					if err != nil {
						return err
					}
					if specialized {
						s.Value = replacement
					}
				}
				replacement, specialized, err := monomorphizeGenericFunctionValueExpr(s.Value, returnType, "function return", generics, created, createdByFile, work, fileImports, module, imports, structCtx)
				if err != nil {
					return err
				}
				if specialized {
					s.Value = replacement
				}
			}
			if _, err := monomorphizeExpr(s.Value, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.ThrowStmt:
			if _, err := monomorphizeExpr(s.Value, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.DeferStmt:
			if err := monomorphizeStmts(s.Body, cloneStringMap(env), cloneFunctionTypeMap(functionLocals), returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.BreakStmt, *frontend.ContinueStmt:
		case *frontend.PrintStmt:
			if _, err := monomorphizeExpr(s.Value, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.ExpectStmt:
			if _, err := monomorphizeExpr(s.Cond, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.FreeStmt:
			if _, err := monomorphizeExpr(s.Value, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.LetStmt:
			if err := monomorphizeFunctionTypedBinding(s, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			valType, err := monomorphizeExpr(s.Value, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
			if err != nil {
				return err
			}
			if s.Type.Name != "" || s.Type.Elem != nil {
				resolved, err := resolveTypeName(&s.Type, module, imports)
				if err != nil {
					return err
				}
				env[s.Name] = resolved
			} else if s.Type.Kind == frontend.TypeRefFunction {
				functionLocals[s.Name] = s.Type
				env[s.Name] = genericTypeName(s.Type)
			} else {
				env[s.Name] = valType
			}
			closureBindingKey := genericClosureBindingKey(s.Name)
			delete(env, closureBindingKey)
			if !s.Mutable {
				if closure, ok := s.Value.(*frontend.ClosureExpr); ok && closure != nil && closure.Decl != nil && len(closure.Decl.TypeParams) > 0 {
					outerLocals := monomorphizeEnvLocals(env)
					if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
						return unsupportedGenericClosureCaptureError(pos, name)
					}
					env[closureBindingKey] = qualifyName(module, closure.Name)
				}
			}
		case *frontend.AssignStmt:
			if _, err := monomorphizeExpr(s.Target, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				if declared, exists := functionLocals[id.Name]; exists && declared.Kind == frontend.TypeRefFunction {
					if closure, ok := s.Value.(*frontend.ClosureExpr); ok && closure != nil && closure.Decl != nil && len(closure.Decl.TypeParams) > 0 {
						outerLocals := monomorphizeEnvLocals(env)
						if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
							return unsupportedGenericClosureCaptureError(pos, name)
						}
						replacement, specialized, err := monomorphizeGenericClosureLiteralValue(closure, declared, fmt.Sprintf("function-typed assignment to '%s'", id.Name), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
						if err != nil {
							return err
						}
						if specialized {
							s.Value = replacement
						}
					}
					replacement, specialized, err := monomorphizeGenericFunctionValueExpr(s.Value, declared, fmt.Sprintf("function-typed assignment to '%s'", id.Name), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
					if err != nil {
						return err
					}
					if specialized {
						s.Value = replacement
					}
				}
			} else if declared, targetName, ok := functionTypeForFieldAssignmentTarget(s.Target, env, structDecls); ok && declared.Kind == frontend.TypeRefFunction {
				if closure, ok := s.Value.(*frontend.ClosureExpr); ok && closure != nil && closure.Decl != nil && len(closure.Decl.TypeParams) > 0 {
					outerLocals := monomorphizeEnvLocals(env)
					if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
						return unsupportedGenericClosureCaptureError(pos, name)
					}
					replacement, specialized, err := monomorphizeGenericClosureLiteralValue(closure, declared, fmt.Sprintf("function-typed assignment to '%s'", targetName), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
					if err != nil {
						return err
					}
					if specialized {
						s.Value = replacement
					}
				}
				replacement, specialized, err := monomorphizeGenericFunctionValueExpr(s.Value, declared, fmt.Sprintf("function-typed assignment to '%s'", targetName), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
				if err != nil {
					return err
				}
				if specialized {
					s.Value = replacement
				}
			}
			if _, err := monomorphizeExpr(s.Value, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.IfStmt:
			if _, err := monomorphizeExpr(s.Cond, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			if err := monomorphizeStmts(s.Then, cloneStringMap(env), cloneFunctionTypeMap(functionLocals), returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			if err := monomorphizeStmts(s.Else, cloneStringMap(env), cloneFunctionTypeMap(functionLocals), returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.IfLetStmt:
			valueType, err := monomorphizeExpr(s.Value, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
			if err != nil {
				return err
			}
			thenEnv := cloneStringMap(env)
			if elem, ok := optionalElemName(valueType); ok {
				thenEnv[s.Name] = elem
			} else {
				thenEnv[s.Name] = "i32"
			}
			if err := monomorphizeStmts(s.Then, thenEnv, cloneFunctionTypeMap(functionLocals), returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			if err := monomorphizeStmts(s.Else, cloneStringMap(env), cloneFunctionTypeMap(functionLocals), returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.WhileStmt:
			if _, err := monomorphizeExpr(s.Cond, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			if err := monomorphizeStmts(s.Body, cloneStringMap(env), cloneFunctionTypeMap(functionLocals), returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.ForRangeStmt:
			bodyEnv := cloneStringMap(env)
			if s.Iterable != nil {
				iterType, err := monomorphizeExpr(s.Iterable, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
				if err != nil {
					return err
				}
				if elem, ok := monomorphizeIterableElemType(iterType); ok {
					bodyEnv[s.Name] = elem
				} else {
					bodyEnv[s.Name] = "i32"
				}
			} else {
				if _, err := monomorphizeExpr(s.Start, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
					return err
				}
				if _, err := monomorphizeExpr(s.End, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
					return err
				}
				bodyEnv[s.Name] = "i32"
			}
			if err := monomorphizeStmts(s.Body, bodyEnv, cloneFunctionTypeMap(functionLocals), returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.MatchStmt:
			scrutType, err := monomorphizeExpr(s.Value, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
			if err != nil {
				return err
			}
			someElemType, hasSomeElem := optionalElemName(scrutType)
			for _, c := range s.Cases {
				caseEnv := cloneStringMap(env)
				if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok && hasSomeElem {
					caseEnv[some.Name] = someElemType
				}
				if !c.Default {
					if _, err := monomorphizeExpr(c.Pattern, caseEnv, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
						return err
					}
				}
				if c.Guard != nil {
					if _, err := monomorphizeExpr(c.Guard, caseEnv, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
						return err
					}
				}
				if err := monomorphizeStmts(c.Body, caseEnv, cloneFunctionTypeMap(functionLocals), returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
					return err
				}
			}
		case *frontend.UnsafeStmt:
			if err := monomorphizeStmts(s.Body, env, functionLocals, returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.IslandStmt:
			if _, err := monomorphizeExpr(s.Size, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			bodyEnv := cloneStringMap(env)
			bodyEnv[s.Name] = "island"
			if err := monomorphizeStmts(s.Body, bodyEnv, cloneFunctionTypeMap(functionLocals), returnType, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.ExprStmt:
			if _, err := monomorphizeExpr(s.Expr, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		}
	}
	return nil
}

func monomorphizeFunctionTypedBinding(
	stmt *frontend.LetStmt,
	env map[string]string,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) error {
	if stmt.Type.Kind != frontend.TypeRefFunction {
		return nil
	}
	if closure, ok := stmt.Value.(*frontend.ClosureExpr); ok && closure != nil && closure.Decl != nil && len(closure.Decl.TypeParams) > 0 {
		outerLocals := monomorphizeEnvLocals(env)
		if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
			return unsupportedGenericClosureCaptureError(pos, name)
		}
		replacement, specialized, err := monomorphizeGenericClosureLiteralValue(closure, stmt.Type, fmt.Sprintf("function-typed local '%s'", stmt.Name), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
		if err != nil || !specialized {
			return err
		}
		stmt.Value = replacement
		return nil
	}
	replacement, specialized, err := monomorphizeGenericFunctionValueExpr(stmt.Value, stmt.Type, fmt.Sprintf("function-typed local '%s'", stmt.Name), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
	if err != nil || !specialized {
		return err
	}
	stmt.Value = replacement
	return nil
}

func monomorphizeGenericClosureLiteralValue(
	closure *frontend.ClosureExpr,
	declared frontend.TypeRef,
	context string,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) (frontend.Expr, bool, error) {
	if declared.Kind != frontend.TypeRefFunction || closure == nil || closure.Decl == nil || len(closure.Decl.TypeParams) == 0 {
		return closure, false, nil
	}
	fullOriginal := qualifyName(module, closure.Name)
	generic, ok := generics[fullOriginal]
	if !ok {
		return closure, false, nil
	}
	declaredParams, declaredReturn, _, err := functionTypeRefSignatureAndEffects(declared, module, imports)
	if err != nil {
		return nil, false, err
	}
	if len(declaredParams) != len(generic.decl.Params) {
		return nil, false, fmt.Errorf("%s: %s generic closure literal parameter count mismatch: expected %d, got %d", frontend.FormatPos(closure.At), context, len(declaredParams), len(generic.decl.Params))
	}
	subst := map[string]string{}
	for i, param := range generic.decl.Params {
		if err := bindGenericType(param.Type, declaredParams[i], generic.decl.TypeParams, subst); err != nil {
			return nil, false, fmt.Errorf("%s: %v", frontend.FormatPos(closure.At), err)
		}
	}
	if err := bindGenericType(generic.decl.ReturnType, declaredReturn, generic.decl.TypeParams, subst); err != nil {
		return nil, false, fmt.Errorf("%s: %v", frontend.FormatPos(closure.At), err)
	}
	for _, tp := range generic.decl.TypeParams {
		if subst[tp] == "" {
			return nil, false, fmt.Errorf("%s: cannot infer generic argument '%s' for %s", frontend.FormatPos(closure.At), tp, context)
		}
	}
	if err := checkGenericProtocolBounds(closure.At, closure.Name, generic, subst, module); err != nil {
		return nil, false, err
	}
	name := mangleGenericName(generic.decl.Name, generic.decl.TypeParams, subst)
	fullName := genericInstanceFullName(generic, name)
	clone, exists := created[fullName]
	if !exists {
		clone = cloneGenericFunc(generic.decl, name, subst)
		cloneImports := fileImports[generic.file]
		if structCtx != nil {
			if err := rewriteGenericFuncStructRefs(structCtx, clone, generic.module, cloneImports); err != nil {
				return nil, false, err
			}
		}
		created[fullName] = clone
		if _, ok := createdByFile[generic.file]; !ok {
			createdByFile[generic.file] = map[string]*frontend.FuncDecl{}
		}
		createdByFile[generic.file][name] = clone
		*work = append(*work, genericWorkItem{fn: clone, module: generic.module, imports: fileImports[generic.file]})
	}
	return &frontend.ClosureExpr{At: closure.At, Name: name, Decl: clone}, true, nil
}

func monomorphizeGenericFunctionValueExpr(
	expr frontend.Expr,
	declared frontend.TypeRef,
	context string,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) (frontend.Expr, bool, error) {
	switch init := expr.(type) {
	case *frontend.IdentExpr:
		fullName, specialized, err := monomorphizeGenericFunctionValue(init, declared, context, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
		if err != nil || !specialized {
			return expr, specialized, err
		}
		init.Name = fullName
		return init, true, nil
	case *frontend.FieldAccessExpr:
		name := callbackArgumentName(init)
		if name == "" {
			return expr, false, nil
		}
		asIdent := &frontend.IdentExpr{At: init.At, Name: name}
		fullName, specialized, err := monomorphizeGenericFunctionValue(asIdent, declared, context, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
		if err != nil || !specialized {
			return expr, specialized, err
		}
		return &frontend.IdentExpr{At: init.At, Name: fullName}, true, nil
	default:
		return expr, false, nil
	}
}

func monomorphizeGenericFunctionValue(
	init *frontend.IdentExpr,
	declared frontend.TypeRef,
	context string,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) (string, bool, error) {
	if declared.Kind != frontend.TypeRefFunction {
		return "", false, nil
	}
	resolved, err := resolveCallName(init.Name, module, imports, init.At)
	if err != nil {
		return "", false, nil
	}
	generic, ok := generics[resolved]
	if !ok {
		return "", false, nil
	}
	declaredParams, declaredReturn, _, err := functionTypeRefSignatureAndEffects(declared, module, imports)
	if err != nil {
		return "", false, err
	}
	if len(declaredParams) != len(generic.decl.Params) {
		return "", false, fmt.Errorf("%s: %s generic function symbol '%s' parameter count mismatch: expected %d, got %d", frontend.FormatPos(init.At), context, init.Name, len(declaredParams), len(generic.decl.Params))
	}
	subst := map[string]string{}
	for i, param := range generic.decl.Params {
		if err := bindGenericType(param.Type, declaredParams[i], generic.decl.TypeParams, subst); err != nil {
			return "", false, fmt.Errorf("%s: %v", frontend.FormatPos(init.At), err)
		}
	}
	if err := bindGenericType(generic.decl.ReturnType, declaredReturn, generic.decl.TypeParams, subst); err != nil {
		return "", false, fmt.Errorf("%s: %v", frontend.FormatPos(init.At), err)
	}
	for _, tp := range generic.decl.TypeParams {
		if subst[tp] == "" {
			return "", false, fmt.Errorf("%s: cannot infer generic argument '%s' for %s", frontend.FormatPos(init.At), tp, context)
		}
	}
	if err := checkGenericProtocolBounds(init.At, init.Name, generic, subst, module); err != nil {
		return "", false, err
	}
	name := mangleGenericName(generic.decl.Name, generic.decl.TypeParams, subst)
	fullName := genericInstanceFullName(generic, name)
	if _, exists := created[fullName]; !exists {
		clone := cloneGenericFunc(generic.decl, name, subst)
		cloneImports := fileImports[generic.file]
		if structCtx != nil {
			if err := rewriteGenericFuncStructRefs(structCtx, clone, generic.module, cloneImports); err != nil {
				return "", false, err
			}
		}
		created[fullName] = clone
		if _, ok := createdByFile[generic.file]; !ok {
			createdByFile[generic.file] = map[string]*frontend.FuncDecl{}
		}
		createdByFile[generic.file][name] = clone
		*work = append(*work, genericWorkItem{fn: clone, module: generic.module, imports: fileImports[generic.file]})
	}
	return fullName, true, nil
}

func monomorphizeFunctionTypedCallArgs(
	call *frontend.CallExpr,
	callee *frontend.FuncDecl,
	env map[string]string,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) error {
	limit := len(call.Args)
	if len(callee.Params) < limit {
		limit = len(callee.Params)
	}
	for i := 0; i < limit; i++ {
		if callee.Params[i].Type.Kind != frontend.TypeRefFunction {
			continue
		}
		if closure, ok := call.Args[i].(*frontend.ClosureExpr); ok && closure != nil && closure.Decl != nil && len(closure.Decl.TypeParams) > 0 {
			outerLocals := monomorphizeEnvLocals(env)
			if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
				return unsupportedGenericClosureCallbackCaptureError(pos, name)
			}
			replacement, specialized, err := monomorphizeGenericClosureLiteralValue(closure, callee.Params[i].Type, fmt.Sprintf("callback argument for '%s'", call.Name), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
			if err != nil {
				return err
			}
			if specialized {
				call.Args[i] = replacement
				continue
			}
		}
		replacement, specialized, err := monomorphizeGenericFunctionValueExpr(call.Args[i], callee.Params[i].Type, fmt.Sprintf("callback argument for '%s'", call.Name), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
		if err != nil {
			return err
		}
		if specialized {
			call.Args[i] = replacement
		}
	}
	return nil
}

func resolveMonomorphizeCallName(
	name string,
	module string,
	imports map[string]string,
	funcDecls map[string]*frontend.FuncDecl,
	generics map[string]genericDef,
	pos frontend.Position,
) (string, error) {
	if _, ok := funcDecls[name]; ok {
		return name, nil
	}
	if _, ok := generics[name]; ok {
		return name, nil
	}
	resolved, err := resolveCallName(name, module, imports, pos)
	if err != nil {
		return "", err
	}
	if _, ok := funcDecls[resolved]; ok {
		return resolved, nil
	}
	if _, ok := generics[resolved]; ok {
		return resolved, nil
	}
	if module != "" && strings.Contains(name, ".") {
		moduleLocal := qualifyName(module, name)
		if _, ok := funcDecls[moduleLocal]; ok {
			return moduleLocal, nil
		}
		if _, ok := generics[moduleLocal]; ok {
			return moduleLocal, nil
		}
	}
	return resolved, nil
}

func monomorphizeConcreteReturnType(callee *frontend.FuncDecl, resolvedName string, module string, imports map[string]string) (string, error) {
	if callee == nil {
		return "", nil
	}
	calleeModule := module
	if callee.Name != "" {
		suffix := "." + callee.Name
		if resolvedName == callee.Name {
			calleeModule = ""
		} else if strings.HasSuffix(resolvedName, suffix) {
			calleeModule = strings.TrimSuffix(resolvedName, suffix)
		}
	}
	ret := substituteTypeRef(callee.ReturnType, nil)
	resolved, err := resolveTypeName(&ret, calleeModule, imports)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func monomorphizeFunctionTypedEnumPayloadArgs(
	call *frontend.CallExpr,
	enumDecls map[string]*frontend.EnumDecl,
	env map[string]string,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) error {
	parts := strings.Split(call.Name, ".")
	if len(parts) < 2 {
		return nil
	}
	caseName := parts[len(parts)-1]
	typeRef := frontend.TypeRef{At: call.At, Kind: frontend.TypeRefNamed, Name: strings.Join(parts[:len(parts)-1], ".")}
	typeName, err := resolveTypeName(&typeRef, module, imports)
	if err != nil {
		return nil
	}
	enumDecl, ok := enumDecls[typeName]
	if !ok {
		return nil
	}
	var enumCase *frontend.EnumCaseDecl
	for i := range enumDecl.Cases {
		if enumDecl.Cases[i].Name == caseName {
			enumCase = &enumDecl.Cases[i]
			break
		}
	}
	if enumCase == nil {
		return nil
	}
	limit := len(call.Args)
	if len(enumCase.Payload) < limit {
		limit = len(enumCase.Payload)
	}
	for i := 0; i < limit; i++ {
		declared := enumCase.Payload[i]
		if declared.Kind != frontend.TypeRefFunction {
			continue
		}
		if closure, ok := call.Args[i].(*frontend.ClosureExpr); ok && closure != nil && closure.Decl != nil && len(closure.Decl.TypeParams) > 0 {
			outerLocals := monomorphizeEnvLocals(env)
			if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
				return unsupportedGenericClosureCaptureError(pos, name)
			}
			replacement, specialized, err := monomorphizeGenericClosureLiteralValue(closure, declared, fmt.Sprintf("enum payload '%s.%s[%d]'", typeRef.Name, caseName, i+1), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
			if err != nil {
				return err
			}
			if specialized {
				call.Args[i] = replacement
				continue
			}
		}
		replacement, specialized, err := monomorphizeGenericFunctionValueExpr(call.Args[i], declared, fmt.Sprintf("enum payload '%s.%s[%d]'", typeRef.Name, caseName, i+1), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
		if err != nil {
			return err
		}
		if specialized {
			call.Args[i] = replacement
		}
	}
	return nil
}

func monomorphizeEnumConstructorTypeName(call *frontend.CallExpr, enumDecls map[string]*frontend.EnumDecl, module string, imports map[string]string) (string, bool) {
	parts := strings.Split(call.Name, ".")
	if len(parts) < 2 {
		return "", false
	}
	caseName := parts[len(parts)-1]
	typeRef := frontend.TypeRef{At: call.At, Kind: frontend.TypeRefNamed, Name: strings.Join(parts[:len(parts)-1], ".")}
	typeName, err := resolveTypeName(&typeRef, module, imports)
	if err != nil {
		return "", false
	}
	enumDecl, ok := enumDecls[typeName]
	if !ok {
		return "", false
	}
	for i := range enumDecl.Cases {
		if enumDecl.Cases[i].Name == caseName {
			return typeName, true
		}
	}
	return "", false
}

func functionTypeForFieldAssignmentTarget(expr frontend.Expr, env map[string]string, structDecls map[string]*frontend.StructDecl) (frontend.TypeRef, string, bool) {
	return functionTypeForFieldAssignmentTargetFrom(expr, "", env, structDecls)
}

func functionTypeRefFromFuncDecl(fn *frontend.FuncDecl) frontend.TypeRef {
	params := make([]frontend.TypeRef, 0, len(fn.Params))
	paramOwnership := make([]string, 0, len(fn.Params))
	for _, param := range fn.Params {
		params = append(params, param.Type)
		paramOwnership = append(paramOwnership, param.Ownership)
	}
	ret := fn.ReturnType
	out := frontend.TypeRef{
		At:             fn.Pos,
		Kind:           frontend.TypeRefFunction,
		Params:         params,
		ParamOwnership: paramOwnership,
		Return:         &ret,
		Uses:           append([]string(nil), fn.Uses...),
	}
	if fn.HasThrows {
		throws := fn.Throws
		out.Throws = &throws
	}
	return out
}

func functionTypeForFieldAssignmentTargetFrom(expr frontend.Expr, path string, env map[string]string, structDecls map[string]*frontend.StructDecl) (frontend.TypeRef, string, bool) {
	switch target := expr.(type) {
	case *frontend.IdentExpr:
		typeName := env[target.Name]
		if typeName == "" {
			return frontend.TypeRef{}, "", false
		}
		if path == "" {
			path = target.Name
		}
		return frontend.TypeRef{Kind: frontend.TypeRefNamed, Name: typeName}, path, true
	case *frontend.FieldAccessExpr:
		baseType, basePath, ok := functionTypeForFieldAssignmentTargetFrom(target.Base, path, env, structDecls)
		if !ok || baseType.Name == "" {
			return frontend.TypeRef{}, "", false
		}
		decl, ok := structDecls[baseType.Name]
		if !ok {
			return frontend.TypeRef{}, "", false
		}
		for _, field := range decl.Fields {
			if field.Name != target.Field {
				continue
			}
			if basePath == "" {
				basePath = target.Field
			} else {
				basePath += "." + target.Field
			}
			return qualifyFieldAssignmentPathType(field.Type, baseType.Name), basePath, true
		}
	}
	return frontend.TypeRef{}, "", false
}

func qualifyFieldAssignmentPathType(ref frontend.TypeRef, ownerType string) frontend.TypeRef {
	if ref.Kind != frontend.TypeRefNamed || ref.Name == "" {
		return ref
	}
	if _, ok := canonicalBuiltinType(ref.Name); ok {
		return ref
	}
	if strings.Contains(ref.Name, ".") {
		return ref
	}
	if idx := strings.LastIndex(ownerType, "."); idx >= 0 {
		ref.Name = qualifyName(ownerType[:idx], ref.Name)
	}
	return ref
}

func monomorphizeExpr(
	expr frontend.Expr,
	env map[string]string,
	funcDecls map[string]*frontend.FuncDecl,
	structDecls map[string]*frontend.StructDecl,
	enumDecls map[string]*frontend.EnumDecl,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) (string, error) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return "i32", nil
	case *frontend.BoolLitExpr:
		return "bool", nil
	case *frontend.StringLitExpr:
		return "str", nil
	case *frontend.NoneLitExpr:
		return "none", nil
	case *frontend.SomePatternExpr:
		return "", nil
	case *frontend.EnumCasePatternExpr:
		return "", nil
	case *frontend.IdentExpr:
		if tname, ok := env[e.Name]; ok {
			return tname, nil
		}
		resolved, err := resolveMonomorphizeCallName(e.Name, module, imports, funcDecls, generics, e.At)
		if err == nil {
			if fn, ok := funcDecls[resolved]; ok && len(fn.TypeParams) == 0 {
				return genericTypeName(functionTypeRefFromFuncDecl(fn)), nil
			}
		}
		return "", nil
	case *frontend.FieldAccessExpr:
		baseType, err := monomorphizeExpr(e.Base, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
		if err != nil {
			return "", err
		}
		if decl, ok := structDecls[baseType]; ok {
			for _, field := range decl.Fields {
				if field.Name == e.Field {
					return genericTypeName(qualifyFieldAssignmentPathType(field.Type, baseType)), nil
				}
			}
		}
		return "", nil
	case *frontend.IndexExpr:
		if _, err := monomorphizeExpr(e.Base, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
			return "", err
		}
		if _, err := monomorphizeExpr(e.Index, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
			return "", err
		}
		return "", nil
	case *frontend.UnaryExpr:
		_, err := monomorphizeExpr(e.X, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
		return "i32", err
	case *frontend.BinaryExpr:
		if _, err := monomorphizeExpr(e.Left, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
			return "", err
		}
		if _, err := monomorphizeExpr(e.Right, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
			return "", err
		}
		switch e.Op {
		case frontend.TokenEqEq, frontend.TokenBangEq, frontend.TokenLess, frontend.TokenLessEq, frontend.TokenGreater, frontend.TokenGreaterEq,
			frontend.TokenAmpAmp, frontend.TokenPipePipe:
			return "bool", nil
		default:
			return "i32", nil
		}
	case *frontend.TryExpr:
		return monomorphizeExpr(e.X, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
	case *frontend.CatchExpr:
		resultType, err := monomorphizeExpr(e.Call, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
		if err != nil {
			return "", err
		}
		for _, c := range e.Cases {
			caseEnv := cloneStringMap(env)
			if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
				caseEnv[some.Name] = "i32"
			}
			if !c.Default {
				if _, err := monomorphizeExpr(c.Pattern, caseEnv, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
					return "", err
				}
			}
			if c.Guard != nil {
				if _, err := monomorphizeExpr(c.Guard, caseEnv, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
					return "", err
				}
			}
			if _, err := monomorphizeExpr(c.Value, caseEnv, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return "", err
			}
		}
		return resultType, nil
	case *frontend.AwaitExpr:
		return monomorphizeExpr(e.X, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
	case *frontend.StructLitExpr:
		resolved, err := resolveTypeName(&e.Type, module, imports)
		if err != nil {
			return "", err
		}
		if decl, ok := structDecls[resolved]; ok {
			fields := make(map[string]frontend.TypeRef, len(decl.Fields))
			for _, field := range decl.Fields {
				fields[field.Name] = field.Type
			}
			for i := range e.Fields {
				field := &e.Fields[i]
				declared := fields[field.Name]
				if declared.Kind != frontend.TypeRefFunction {
					continue
				}
				if closure, ok := field.Value.(*frontend.ClosureExpr); ok && closure != nil && closure.Decl != nil && len(closure.Decl.TypeParams) > 0 {
					outerLocals := monomorphizeEnvLocals(env)
					if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
						return "", unsupportedGenericClosureCaptureError(pos, name)
					}
					replacement, specialized, err := monomorphizeGenericClosureLiteralValue(closure, declared, fmt.Sprintf("struct field '%s.%s'", e.Type.Name, field.Name), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
					if err != nil {
						return "", err
					}
					if specialized {
						field.Value = replacement
						continue
					}
				}
				replacement, specialized, err := monomorphizeGenericFunctionValueExpr(field.Value, declared, fmt.Sprintf("struct field '%s.%s'", e.Type.Name, field.Name), generics, created, createdByFile, work, fileImports, module, imports, structCtx)
				if err != nil {
					return "", err
				}
				if specialized {
					field.Value = replacement
				}
			}
		}
		for _, field := range e.Fields {
			if _, err := monomorphizeExpr(field.Value, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return "", err
			}
		}
		e.Type.Name = resolved
		return resolved, nil
	case *frontend.CallExpr:
		if err := monomorphizeFunctionTypedEnumPayloadArgs(e, enumDecls, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
			return "", err
		}
		argTypes := make([]string, 0, len(e.Args))
		for _, arg := range e.Args {
			tname, err := monomorphizeExpr(arg, env, funcDecls, structDecls, enumDecls, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
			if err != nil {
				return "", err
			}
			argTypes = append(argTypes, tname)
		}
		if enumTypeName, ok := monomorphizeEnumConstructorTypeName(e, enumDecls, module, imports); ok {
			return enumTypeName, nil
		}

		resolved := e.Name
		if localGenericClosure, ok := env[genericClosureBindingKey(e.Name)]; ok {
			if _, isGeneric := generics[localGenericClosure]; isGeneric {
				resolved = localGenericClosure
			}
		}
		if resolved == e.Name {
			if builtin, ok := ResolveBuiltinAlias(e.Name); ok {
				resolved = builtin
			} else {
				var err error
				resolved, err = resolveMonomorphizeCallName(e.Name, module, imports, funcDecls, generics, e.At)
				if err != nil {
					return "", err
				}
			}
		}
		generic, ok := generics[resolved]
		if !ok {
			if callee, exists := funcDecls[resolved]; exists {
				if err := monomorphizeFunctionTypedCallArgs(e, callee, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
					return "", err
				}
				return monomorphizeConcreteReturnType(callee, resolved, module, imports)
			}
			return "", nil
		}
		if len(e.Args) != len(generic.decl.Params) {
			return "", fmt.Errorf("%s: wrong argument count for generic function '%s'", frontend.FormatPos(e.At), e.Name)
		}
		subst := map[string]string{}
		for i, param := range generic.decl.Params {
			if argTypes[i] == "" {
				return "", fmt.Errorf("%s: cannot infer generic argument for '%s' arg %d", frontend.FormatPos(e.Args[i].Pos()), e.Name, i+1)
			}
			if err := bindGenericType(param.Type, argTypes[i], generic.decl.TypeParams, subst); err != nil {
				return "", fmt.Errorf("%s: %v", frontend.FormatPos(e.Args[i].Pos()), err)
			}
		}
		for _, tp := range generic.decl.TypeParams {
			if subst[tp] == "" {
				return "", fmt.Errorf("%s: cannot infer generic argument '%s' for '%s'", frontend.FormatPos(e.At), tp, e.Name)
			}
		}
		if err := checkGenericProtocolBounds(e.At, e.Name, generic, subst, module); err != nil {
			return "", err
		}
		name := mangleGenericName(generic.decl.Name, generic.decl.TypeParams, subst)
		fullName := genericInstanceFullName(generic, name)
		var returnType frontend.TypeRef
		if existing, exists := created[fullName]; exists {
			returnType = existing.ReturnType
		} else {
			clone := cloneGenericFunc(generic.decl, name, subst)
			cloneImports := fileImports[generic.file]
			if structCtx != nil {
				if err := rewriteGenericFuncStructRefs(structCtx, clone, generic.module, cloneImports); err != nil {
					return "", err
				}
			}
			returnType = clone.ReturnType
			created[fullName] = clone
			if _, ok := createdByFile[generic.file]; !ok {
				createdByFile[generic.file] = map[string]*frontend.FuncDecl{}
			}
			createdByFile[generic.file][name] = clone
			*work = append(*work, genericWorkItem{fn: clone, module: generic.module, imports: fileImports[generic.file]})
		}
		e.Name = fullName
		return genericTypeName(returnType), nil
	case *frontend.ClosureExpr:
		return "ptr", nil
	default:
		return "", nil
	}
}

func checkGenericProtocolBounds(callPos frontend.Position, callName string, generic genericDef, subst map[string]string, callerModule string) error {
	for _, bound := range generic.decl.TypeParamBounds {
		actual := subst[bound.Name]
		if actual == "" {
			continue
		}
		boundRef := bound.Bound
		protoName, err := resolveTypeName(&boundRef, generic.module, generic.imports)
		if err != nil {
			return err
		}
		protoInfo, ok := generic.protocols[protoName]
		if !ok {
			if _, isType := generic.knownTypes[protoName]; isType {
				return fmt.Errorf("%s: generic bound '%s' for '%s' must name a protocol, got non-protocol type '%s'",
					frontend.FormatPos(bound.Bound.At),
					displayTypeName(protoName, generic.module),
					bound.Name,
					displayTypeName(protoName, generic.module),
				)
			}
			return fmt.Errorf("%s: unknown protocol bound '%s' for generic parameter '%s'",
				frontend.FormatPos(bound.Bound.At),
				displayTypeName(protoName, generic.module),
				bound.Name,
			)
		}
		if !symbolBelongsToModule(protoName, callerModule) && !protoInfo.public {
			return fmt.Errorf("%s: private protocol '%s' is not visible from module '%s'",
				frontend.FormatPos(callPos),
				protoName,
				callerModule,
			)
		}
		if _, ok := generic.conformances[protocolConformanceKey{typeName: actual, protocol: protoName}]; ok {
			continue
		}
		return fmt.Errorf("%s: generic argument '%s' does not satisfy bound '%s' for '%s' in call to '%s' (missing impl %s: %s)",
			frontend.FormatPos(callPos),
			displayTypeName(actual, generic.module),
			displayTypeName(protoName, generic.module),
			bound.Name,
			callName,
			displayTypeName(actual, generic.module),
			displayTypeName(protoName, generic.module),
		)
	}
	return nil
}

func rewriteGenericFuncStructRefs(ctx *genericStructContext, fn *frontend.FuncDecl, module string, imports map[string]string) error {
	if err := ctx.rewriteTypeRef(&fn.ReturnType, module, imports); err != nil {
		return err
	}
	if fn.HasThrows {
		if err := ctx.rewriteTypeRef(&fn.Throws, module, imports); err != nil {
			return err
		}
	}
	for i := range fn.Params {
		if err := ctx.rewriteTypeRef(&fn.Params[i].Type, module, imports); err != nil {
			return err
		}
	}
	return ctx.rewriteStmts(fn.Body, module, imports)
}

func monomorphizeIterableElemType(typeName string) (string, bool) {
	if strings.HasPrefix(typeName, "[]") {
		return strings.TrimPrefix(typeName, "[]"), true
	}
	if _, elem, ok := parseArrayTypeName(typeName); ok {
		return elem, true
	}
	if typeName == "str" {
		return "u8", true
	}
	return "", false
}

func bindGenericType(param frontend.TypeRef, actual string, typeParams []string, subst map[string]string) error {
	if param.Kind == frontend.TypeRefNamed && contains(typeParams, param.Name) {
		if existing := subst[param.Name]; existing != "" && existing != actual {
			return fmt.Errorf("conflicting generic argument for '%s': %s vs %s", param.Name, existing, actual)
		}
		subst[param.Name] = actual
		return nil
	}
	switch param.Kind {
	case frontend.TypeRefFunction:
		actualParams, actualReturn, ok := parseGenericFunctionTypeName(actual)
		if !ok {
			return nil
		}
		if len(param.Params) != len(actualParams) {
			return nil
		}
		for i := range param.Params {
			if err := bindGenericType(param.Params[i], actualParams[i], typeParams, subst); err != nil {
				return err
			}
		}
		if param.Return != nil {
			return bindGenericType(*param.Return, actualReturn, typeParams, subst)
		}
	case frontend.TypeRefOptional:
		if param.Elem == nil {
			return nil
		}
		elemActual := actual
		if elem, ok := optionalElemName(actual); ok {
			elemActual = elem
		}
		return bindGenericType(*param.Elem, elemActual, typeParams, subst)
	case frontend.TypeRefSlice:
		if param.Elem == nil {
			return nil
		}
		if elemActual, ok := sliceElemName(actual); ok {
			return bindGenericType(*param.Elem, elemActual, typeParams, subst)
		}
	case frontend.TypeRefArray:
		if param.Elem == nil {
			return nil
		}
		if elemActual, ok := arrayElemName(actual, param.Len); ok {
			return bindGenericType(*param.Elem, elemActual, typeParams, subst)
		}
	}
	return nil
}

func arrayElemName(name string, wantLen int) (string, bool) {
	gotLen, elem, ok := parseArrayTypeName(name)
	if !ok || gotLen != wantLen {
		return "", false
	}
	return elem, true
}

func cloneGenericFunc(fn *frontend.FuncDecl, name string, subst map[string]string) *frontend.FuncDecl {
	out := *fn
	out.Name = name
	out.ExportName = ""
	out.TypeParams = nil
	out.TypeParamBounds = nil
	out.Params = cloneParams(fn.Params, subst)
	out.ReturnType = substituteTypeRef(fn.ReturnType, subst)
	out.Throws = substituteTypeRef(fn.Throws, subst)
	out.Uses = append([]string(nil), fn.Uses...)
	out.SemanticClauses = append([]frontend.SemanticClause(nil), fn.SemanticClauses...)
	out.Body = cloneStmts(fn.Body, subst)
	return &out
}

func cloneParams(params []frontend.ParamDecl, subst map[string]string) []frontend.ParamDecl {
	out := make([]frontend.ParamDecl, len(params))
	for i, p := range params {
		out[i] = p
		out[i].Type = substituteTypeRef(p.Type, subst)
	}
	return out
}

func cloneStmts(stmts []frontend.Stmt, subst map[string]string) []frontend.Stmt {
	out := make([]frontend.Stmt, 0, len(stmts))
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			out = append(out, &frontend.ReturnStmt{At: s.At, Value: cloneExpr(s.Value, subst)})
		case *frontend.ThrowStmt:
			out = append(out, &frontend.ThrowStmt{At: s.At, Value: cloneExpr(s.Value, subst)})
		case *frontend.DeferStmt:
			out = append(out, &frontend.DeferStmt{At: s.At, Body: cloneStmts(s.Body, subst)})
		case *frontend.BreakStmt:
			out = append(out, &frontend.BreakStmt{At: s.At})
		case *frontend.ContinueStmt:
			out = append(out, &frontend.ContinueStmt{At: s.At})
		case *frontend.PrintStmt:
			out = append(out, &frontend.PrintStmt{At: s.At, Value: cloneExpr(s.Value, subst)})
		case *frontend.ExpectStmt:
			out = append(out, &frontend.ExpectStmt{At: s.At, Cond: cloneExpr(s.Cond, subst)})
		case *frontend.FreeStmt:
			out = append(out, &frontend.FreeStmt{At: s.At, Value: cloneExpr(s.Value, subst), Implicit: s.Implicit})
		case *frontend.LetStmt:
			out = append(out, &frontend.LetStmt{At: s.At, Name: s.Name, Type: substituteTypeRef(s.Type, subst), Mutable: s.Mutable, Const: s.Const, Value: cloneExpr(s.Value, subst)})
		case *frontend.AssignStmt:
			var compoundValue frontend.Expr
			if s.CompoundValue != nil {
				compoundValue = cloneExpr(s.CompoundValue, subst)
			}
			out = append(out, &frontend.AssignStmt{At: s.At, Target: cloneExpr(s.Target, subst), Value: cloneExpr(s.Value, subst), Op: s.Op, CompoundValue: compoundValue})
		case *frontend.IfStmt:
			out = append(out, &frontend.IfStmt{At: s.At, Cond: cloneExpr(s.Cond, subst), Then: cloneStmts(s.Then, subst), Else: cloneStmts(s.Else, subst)})
		case *frontend.IfLetStmt:
			out = append(out, &frontend.IfLetStmt{At: s.At, Name: s.Name, Pattern: cloneExpr(s.Pattern, subst), Value: cloneExpr(s.Value, subst), ValueLocal: s.ValueLocal, Then: cloneStmts(s.Then, subst), Else: cloneStmts(s.Else, subst)})
		case *frontend.WhileStmt:
			out = append(out, &frontend.WhileStmt{At: s.At, Cond: cloneExpr(s.Cond, subst), Body: cloneStmts(s.Body, subst)})
		case *frontend.ForRangeStmt:
			var start, end, iterable frontend.Expr
			if s.Start != nil {
				start = cloneExpr(s.Start, subst)
			}
			if s.End != nil {
				end = cloneExpr(s.End, subst)
			}
			if s.Iterable != nil {
				iterable = cloneExpr(s.Iterable, subst)
			}
			out = append(out, &frontend.ForRangeStmt{At: s.At, Name: s.Name, Start: start, End: end, Iterable: iterable, IterableLocal: s.IterableLocal, IndexLocal: s.IndexLocal, EndLocal: s.EndLocal, Body: cloneStmts(s.Body, subst)})
		case *frontend.MatchStmt:
			cases := make([]frontend.MatchCase, 0, len(s.Cases))
			for _, c := range s.Cases {
				cases = append(cases, frontend.MatchCase{At: c.At, Pattern: cloneExpr(c.Pattern, subst), Guard: cloneExpr(c.Guard, subst), Default: c.Default, Body: cloneStmts(c.Body, subst)})
			}
			out = append(out, &frontend.MatchStmt{At: s.At, Value: cloneExpr(s.Value, subst), ScrutineeLocal: s.ScrutineeLocal, Cases: cases})
		case *frontend.UnsafeStmt:
			out = append(out, &frontend.UnsafeStmt{At: s.At, Body: cloneStmts(s.Body, subst)})
		case *frontend.IslandStmt:
			out = append(out, &frontend.IslandStmt{At: s.At, Size: cloneExpr(s.Size, subst), Name: s.Name, Body: cloneStmts(s.Body, subst)})
		case *frontend.ExprStmt:
			out = append(out, &frontend.ExprStmt{At: s.At, Expr: cloneExpr(s.Expr, subst)})
		default:
			out = append(out, stmt)
		}
	}
	return out
}

func cloneExpr(expr frontend.Expr, subst map[string]string) frontend.Expr {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return &frontend.NumberExpr{At: e.At, Value: e.Value}
	case *frontend.BoolLitExpr:
		return &frontend.BoolLitExpr{At: e.At, Value: e.Value}
	case *frontend.NoneLitExpr:
		return &frontend.NoneLitExpr{At: e.At}
	case *frontend.SomePatternExpr:
		return &frontend.SomePatternExpr{At: e.At, Name: e.Name}
	case *frontend.EnumCasePatternExpr:
		return &frontend.EnumCasePatternExpr{
			At:           e.At,
			TypeName:     e.TypeName,
			CaseName:     e.CaseName,
			Bindings:     append([]string(nil), e.Bindings...),
			HasPayload:   e.HasPayload,
			EnumType:     e.EnumType,
			EnumOrdinal:  e.EnumOrdinal,
			PayloadSlots: append([]int(nil), e.PayloadSlots...),
		}
	case *frontend.StringLitExpr:
		return &frontend.StringLitExpr{At: e.At, Value: append([]byte(nil), e.Value...)}
	case *frontend.IdentExpr:
		return &frontend.IdentExpr{At: e.At, Name: e.Name}
	case *frontend.FieldAccessExpr:
		return &frontend.FieldAccessExpr{At: e.At, Base: cloneExpr(e.Base, subst), Field: e.Field, EnumType: e.EnumType, EnumOrdinal: e.EnumOrdinal}
	case *frontend.IndexExpr:
		return &frontend.IndexExpr{At: e.At, Base: cloneExpr(e.Base, subst), Index: cloneExpr(e.Index, subst)}
	case *frontend.UnaryExpr:
		return &frontend.UnaryExpr{At: e.At, Op: e.Op, X: cloneExpr(e.X, subst)}
	case *frontend.BinaryExpr:
		return &frontend.BinaryExpr{At: e.At, Op: e.Op, Left: cloneExpr(e.Left, subst), Right: cloneExpr(e.Right, subst)}
	case *frontend.TryExpr:
		return &frontend.TryExpr{At: e.At, X: cloneExpr(e.X, subst)}
	case *frontend.CatchExpr:
		cases := make([]frontend.CatchExprCase, 0, len(e.Cases))
		for _, c := range e.Cases {
			cases = append(cases, frontend.CatchExprCase{At: c.At, Pattern: cloneExpr(c.Pattern, subst), Guard: cloneExpr(c.Guard, subst), Default: c.Default, Value: cloneExpr(c.Value, subst)})
		}
		return &frontend.CatchExpr{At: e.At, Call: cloneExpr(e.Call, subst), ErrorLocal: e.ErrorLocal, ResultLocal: e.ResultLocal, ErrorType: e.ErrorType, ResultType: e.ResultType, Cases: cases}
	case *frontend.AwaitExpr:
		return &frontend.AwaitExpr{At: e.At, X: cloneExpr(e.X, subst)}
	case *frontend.CallExpr:
		args := make([]frontend.Expr, 0, len(e.Args))
		for _, arg := range e.Args {
			args = append(args, cloneExpr(arg, subst))
		}
		labels := append([]string(nil), e.ArgLabels...)
		typeArgs := make([]frontend.TypeRef, len(e.TypeArgs))
		for i := range e.TypeArgs {
			typeArgs[i] = substituteTypeRef(e.TypeArgs[i], subst)
		}
		return &frontend.CallExpr{At: e.At, Name: e.Name, TypeArgs: typeArgs, Args: args, ArgLabels: labels, ResolvedType: e.ResolvedType}
	case *frontend.StructLitExpr:
		fields := make([]frontend.StructFieldInit, 0, len(e.Fields))
		for _, field := range e.Fields {
			fields = append(fields, frontend.StructFieldInit{At: field.At, Name: field.Name, Value: cloneExpr(field.Value, subst)})
		}
		return &frontend.StructLitExpr{At: e.At, Type: substituteTypeRef(e.Type, subst), Fields: fields}
	case *frontend.ClosureExpr:
		return &frontend.ClosureExpr{At: e.At, Name: e.Name, Decl: e.Decl, Captures: cloneClosureCaptures(e.Captures, subst)}
	default:
		return expr
	}
}

func cloneClosureCaptures(captures []frontend.ClosureCapture, subst map[string]string) []frontend.ClosureCapture {
	if len(captures) == 0 {
		return nil
	}
	out := make([]frontend.ClosureCapture, len(captures))
	for i, capture := range captures {
		out[i] = frontend.ClosureCapture{
			At:      capture.At,
			Name:    capture.Name,
			Type:    substituteTypeRef(capture.Type, subst),
			Mutable: capture.Mutable,
		}
	}
	return out
}

func substituteTypeRef(ref frontend.TypeRef, subst map[string]string) frontend.TypeRef {
	out := ref
	if len(ref.TypeArgs) > 0 {
		out.TypeArgs = make([]frontend.TypeRef, len(ref.TypeArgs))
		for i := range ref.TypeArgs {
			out.TypeArgs[i] = substituteTypeRef(ref.TypeArgs[i], subst)
		}
	}
	if len(ref.Params) > 0 {
		out.Params = make([]frontend.TypeRef, len(ref.Params))
		for i := range ref.Params {
			out.Params[i] = substituteTypeRef(ref.Params[i], subst)
		}
	}
	if ref.Kind == frontend.TypeRefNamed {
		if concrete := subst[ref.Name]; concrete != "" {
			return typeRefFromGenericTypeName(ref.At, concrete)
		}
		return out
	}
	if ref.Return != nil {
		ret := substituteTypeRef(*ref.Return, subst)
		out.Return = &ret
	}
	if ref.Elem != nil {
		elem := substituteTypeRef(*ref.Elem, subst)
		out.Elem = &elem
	}
	return out
}

func typeRefFromGenericTypeName(at frontend.Position, name string) frontend.TypeRef {
	if params, ret, ok := parseGenericFunctionTypeName(name); ok {
		paramRefs := make([]frontend.TypeRef, 0, len(params))
		for _, param := range params {
			paramRefs = append(paramRefs, typeRefFromGenericTypeName(at, param))
		}
		retRef := typeRefFromGenericTypeName(at, ret)
		return frontend.TypeRef{At: at, Kind: frontend.TypeRefFunction, Params: paramRefs, Return: &retRef}
	}
	if elem, ok := optionalElemName(name); ok {
		elemRef := typeRefFromGenericTypeName(at, elem)
		return frontend.TypeRef{At: at, Kind: frontend.TypeRefOptional, Elem: &elemRef}
	}
	if elem, ok := sliceElemName(name); ok {
		elemRef := typeRefFromGenericTypeName(at, elem)
		return frontend.TypeRef{At: at, Kind: frontend.TypeRefSlice, Elem: &elemRef}
	}
	if n, elem, ok := parseArrayTypeName(name); ok {
		elemRef := typeRefFromGenericTypeName(at, elem)
		return frontend.TypeRef{At: at, Kind: frontend.TypeRefArray, Elem: &elemRef, Len: n}
	}
	return frontend.TypeRef{At: at, Kind: frontend.TypeRefNamed, Name: name}
}

func parseGenericFunctionTypeName(name string) ([]string, string, bool) {
	name = strings.TrimSpace(name)
	if !strings.HasPrefix(name, "fn(") {
		return nil, "", false
	}
	depth := 1
	closeIdx := -1
	for i := len("fn("); i < len(name); i++ {
		switch name[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				closeIdx = i
				i = len(name)
			}
		}
	}
	if closeIdx < 0 {
		return nil, "", false
	}
	rest := strings.TrimSpace(name[closeIdx+1:])
	if !strings.HasPrefix(rest, "->") {
		return nil, "", false
	}
	paramsText := name[len("fn("):closeIdx]
	params := []string{}
	if strings.TrimSpace(paramsText) != "" {
		params = splitGenericTypeList(paramsText)
	}
	ret := strings.TrimSpace(rest[len("->"):])
	if idx := strings.Index(ret, " uses "); idx >= 0 {
		ret = strings.TrimSpace(ret[:idx])
	}
	if idx := strings.Index(ret, " throws "); idx >= 0 {
		ret = strings.TrimSpace(ret[:idx])
	}
	if ret == "" {
		return nil, "", false
	}
	return params, ret, true
}

func splitGenericTypeList(text string) []string {
	var out []string
	start := 0
	parenDepth := 0
	angleDepth := 0
	bracketDepth := 0
	for i, r := range text {
		switch r {
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '<':
			angleDepth++
		case '>':
			if angleDepth > 0 {
				angleDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case ',':
			if parenDepth == 0 && angleDepth == 0 && bracketDepth == 0 {
				out = append(out, strings.TrimSpace(text[start:i]))
				start = i + len(string(r))
			}
		}
	}
	out = append(out, strings.TrimSpace(text[start:]))
	return out
}

func substituteGenericTypeName(ref frontend.TypeRef, subst map[string]string) string {
	return genericTypeName(substituteTypeRef(ref, subst))
}

func mangleGenericName(base string, order []string, subst map[string]string) string {
	var parts []string
	for _, tp := range order {
		parts = append(parts, tp+"_"+sanitizeGenericType(subst[tp]))
	}
	return base + "__" + strings.Join(parts, "__")
}

func sanitizeGenericType(tname string) string {
	if tname == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range tname {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		if r == '_' {
			b.WriteString("__")
			continue
		}
		b.WriteString("_")
		b.WriteString(strconv.FormatInt(int64(r), 16))
		b.WriteString("_")
	}
	return b.String()
}

func genericTypeName(ref frontend.TypeRef) string {
	switch ref.Kind {
	case frontend.TypeRefSlice:
		if ref.Elem == nil {
			return "[]"
		}
		return "[]" + genericTypeName(*ref.Elem)
	case frontend.TypeRefArray:
		if ref.Elem == nil {
			return fmt.Sprintf("[%d]", ref.Len)
		}
		return fmt.Sprintf("[%d]%s", ref.Len, genericTypeName(*ref.Elem))
	case frontend.TypeRefOptional:
		if ref.Elem == nil {
			return "?"
		}
		return genericTypeName(*ref.Elem) + "?"
	case frontend.TypeRefFunction:
		params := make([]string, 0, len(ref.Params))
		for i, param := range ref.Params {
			formatted := genericTypeName(param)
			if i < len(ref.ParamOwnership) && ref.ParamOwnership[i] != "" {
				formatted = ref.ParamOwnership[i] + " " + formatted
			}
			params = append(params, formatted)
		}
		ret := "?"
		if ref.Return != nil {
			ret = genericTypeName(*ref.Return)
		}
		out := "fn(" + strings.Join(params, ",") + ")->" + ret
		if ref.Throws != nil {
			out += " throws " + genericTypeName(*ref.Throws)
		}
		if len(ref.Uses) > 0 {
			out += " uses " + strings.Join(ref.Uses, ",")
		}
		return out
	default:
		if canonical, ok := canonicalBuiltinType(ref.Name); ok {
			return canonical
		}
		if len(ref.TypeArgs) > 0 {
			args := make([]string, 0, len(ref.TypeArgs))
			for _, arg := range ref.TypeArgs {
				args = append(args, genericTypeName(arg))
			}
			return ref.Name + "<" + strings.Join(args, ",") + ">"
		}
		return ref.Name
	}
}

const genericClosureBindingPrefix = "\x00generic-closure:"

func genericClosureBindingKey(name string) string {
	return genericClosureBindingPrefix + name
}

func monomorphizeEnvLocals(env map[string]string) map[string]LocalInfo {
	locals := make(map[string]LocalInfo, len(env))
	for name, typeName := range env {
		if strings.HasPrefix(name, genericClosureBindingPrefix) || typeName == "" {
			continue
		}
		locals[name] = LocalInfo{TypeName: typeName}
	}
	return locals
}

func cloneStringMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneFunctionTypeMap(src map[string]frontend.TypeRef) map[string]frontend.TypeRef {
	dst := make(map[string]frontend.TypeRef, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
