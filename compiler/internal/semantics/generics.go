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

func monomorphizeGenerics(world *module.World) error {
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
				fullName := qualifyName(file.Module, fn.Name)
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
	for name, def := range generics {
		def.conformances = conformances
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
		if err := monomorphizeStmts(item.fn.Body, env, generics, created, createdByFile, &work, fileImports, item.module, item.imports, structCtx); err != nil {
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
		for _, name := range names {
			file.Funcs = append(file.Funcs, perFile[name])
		}
	}
	if structCtx != nil {
		structCtx.finalize(world)
	}
	return nil
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
			argName, err := resolveTypeName(&ref.TypeArgs[i], module, imports)
			if err != nil {
				return err
			}
			subst[tp] = argName
		}
		name, err := ctx.instantiate(generic, subst)
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

func (ctx *genericStructContext) instantiate(generic genericStructDef, subst map[string]string) (string, error) {
	name := mangleGenericName(generic.decl.Name, generic.decl.TypeParams, subst)
	fullName := qualifyName(generic.module, name)
	if _, exists := ctx.created[fullName]; exists {
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
		if err := ctx.rewriteTypeRef(&clone.Fields[i].Type, generic.module, imports); err != nil {
			return "", err
		}
	}
	return name, nil
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
			if _, err := monomorphizeExpr(s.Value, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.ThrowStmt:
			if _, err := monomorphizeExpr(s.Value, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.DeferStmt:
			if err := monomorphizeStmts(s.Body, cloneStringMap(env), generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.BreakStmt, *frontend.ContinueStmt:
		case *frontend.PrintStmt:
			if _, err := monomorphizeExpr(s.Value, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.ExpectStmt:
			if _, err := monomorphizeExpr(s.Cond, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.FreeStmt:
			if _, err := monomorphizeExpr(s.Value, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.LetStmt:
			valType, err := monomorphizeExpr(s.Value, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
			if err != nil {
				return err
			}
			if s.Type.Name != "" || s.Type.Elem != nil {
				resolved, err := resolveTypeName(&s.Type, module, imports)
				if err != nil {
					return err
				}
				env[s.Name] = resolved
			} else {
				env[s.Name] = valType
			}
			closureBindingKey := genericClosureBindingKey(s.Name)
			delete(env, closureBindingKey)
			if !s.Mutable {
				if closure, ok := s.Value.(*frontend.ClosureExpr); ok && closure != nil && closure.Decl != nil && len(closure.Decl.TypeParams) > 0 {
					outerLocals := monomorphizeEnvLocals(env)
					if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
						return fmt.Errorf("%s: generic closure literals do not support captures in this MVP (captured '%s')", frontend.FormatPos(pos), name)
					}
					env[closureBindingKey] = qualifyName(module, closure.Name)
				}
			}
		case *frontend.AssignStmt:
			if _, err := monomorphizeExpr(s.Target, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			if _, err := monomorphizeExpr(s.Value, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.IfStmt:
			if _, err := monomorphizeExpr(s.Cond, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			if err := monomorphizeStmts(s.Then, cloneStringMap(env), generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			if err := monomorphizeStmts(s.Else, cloneStringMap(env), generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.IfLetStmt:
			valueType, err := monomorphizeExpr(s.Value, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
			if err != nil {
				return err
			}
			thenEnv := cloneStringMap(env)
			if elem, ok := optionalElemName(valueType); ok {
				thenEnv[s.Name] = elem
			} else {
				thenEnv[s.Name] = "i32"
			}
			if err := monomorphizeStmts(s.Then, thenEnv, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			if err := monomorphizeStmts(s.Else, cloneStringMap(env), generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.WhileStmt:
			if _, err := monomorphizeExpr(s.Cond, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			if err := monomorphizeStmts(s.Body, cloneStringMap(env), generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.ForRangeStmt:
			bodyEnv := cloneStringMap(env)
			if s.Iterable != nil {
				iterType, err := monomorphizeExpr(s.Iterable, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
				if err != nil {
					return err
				}
				if elem, ok := monomorphizeIterableElemType(iterType); ok {
					bodyEnv[s.Name] = elem
				} else {
					bodyEnv[s.Name] = "i32"
				}
			} else {
				if _, err := monomorphizeExpr(s.Start, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
					return err
				}
				if _, err := monomorphizeExpr(s.End, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
					return err
				}
				bodyEnv[s.Name] = "i32"
			}
			if err := monomorphizeStmts(s.Body, bodyEnv, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.MatchStmt:
			scrutType, err := monomorphizeExpr(s.Value, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
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
					if _, err := monomorphizeExpr(c.Pattern, caseEnv, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
						return err
					}
				}
				if c.Guard != nil {
					if _, err := monomorphizeExpr(c.Guard, caseEnv, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
						return err
					}
				}
				if err := monomorphizeStmts(c.Body, caseEnv, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
					return err
				}
			}
		case *frontend.UnsafeStmt:
			if err := monomorphizeStmts(s.Body, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.IslandStmt:
			if _, err := monomorphizeExpr(s.Size, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
			bodyEnv := cloneStringMap(env)
			bodyEnv[s.Name] = "island"
			if err := monomorphizeStmts(s.Body, bodyEnv, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		case *frontend.ExprStmt:
			if _, err := monomorphizeExpr(s.Expr, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return err
			}
		}
	}
	return nil
}

func monomorphizeExpr(
	expr frontend.Expr,
	env map[string]string,
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
		return "", nil
	case *frontend.FieldAccessExpr:
		if _, err := monomorphizeExpr(e.Base, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
			return "", err
		}
		return "", nil
	case *frontend.IndexExpr:
		if _, err := monomorphizeExpr(e.Base, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
			return "", err
		}
		if _, err := monomorphizeExpr(e.Index, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
			return "", err
		}
		return "", nil
	case *frontend.UnaryExpr:
		_, err := monomorphizeExpr(e.X, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
		return "i32", err
	case *frontend.BinaryExpr:
		if _, err := monomorphizeExpr(e.Left, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
			return "", err
		}
		if _, err := monomorphizeExpr(e.Right, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
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
		return monomorphizeExpr(e.X, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
	case *frontend.CatchExpr:
		resultType, err := monomorphizeExpr(e.Call, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
		if err != nil {
			return "", err
		}
		for _, c := range e.Cases {
			caseEnv := cloneStringMap(env)
			if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
				caseEnv[some.Name] = "i32"
			}
			if !c.Default {
				if _, err := monomorphizeExpr(c.Pattern, caseEnv, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
					return "", err
				}
			}
			if c.Guard != nil {
				if _, err := monomorphizeExpr(c.Guard, caseEnv, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
					return "", err
				}
			}
			if _, err := monomorphizeExpr(c.Value, caseEnv, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return "", err
			}
		}
		return resultType, nil
	case *frontend.AwaitExpr:
		return monomorphizeExpr(e.X, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if _, err := monomorphizeExpr(field.Value, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx); err != nil {
				return "", err
			}
		}
		resolved, err := resolveTypeName(&e.Type, module, imports)
		if err != nil {
			return "", err
		}
		e.Type.Name = resolved
		return resolved, nil
	case *frontend.CallExpr:
		argTypes := make([]string, 0, len(e.Args))
		for _, arg := range e.Args {
			tname, err := monomorphizeExpr(arg, env, generics, created, createdByFile, work, fileImports, module, imports, structCtx)
			if err != nil {
				return "", err
			}
			argTypes = append(argTypes, tname)
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
				resolved, err = resolveCallName(e.Name, module, imports, e.At)
				if err != nil {
					return "", err
				}
			}
		}
		generic, ok := generics[resolved]
		if !ok {
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
		if err := checkGenericProtocolBounds(e.At, e.Name, generic, subst); err != nil {
			return "", err
		}
		name := mangleGenericName(generic.decl.Name, generic.decl.TypeParams, subst)
		fullName := qualifyName(generic.module, name)
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

func checkGenericProtocolBounds(callPos frontend.Position, callName string, generic genericDef, subst map[string]string) error {
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
			At:   capture.At,
			Name: capture.Name,
			Type: substituteTypeRef(capture.Type, subst),
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
			out.Name = concrete
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
		for _, param := range ref.Params {
			params = append(params, genericTypeName(param))
		}
		ret := "?"
		if ref.Return != nil {
			ret = genericTypeName(*ref.Return)
		}
		out := "fn(" + strings.Join(params, ",") + ")->" + ret
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

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
