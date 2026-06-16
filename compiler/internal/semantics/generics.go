package semantics

import (
	"fmt"
	"sort"
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
