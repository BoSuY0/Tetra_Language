package semantics

import (
	"fmt"
	"sort"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
)

func monomorphizeGenerics(world *module.World) error {
	for _, file := range world.Files {
		generics := map[string]*frontend.FuncDecl{}
		for _, fn := range file.Funcs {
			if len(fn.TypeParams) > 0 {
				generics[fn.Name] = fn
			}
		}
		if len(generics) == 0 {
			continue
		}
		created := map[string]*frontend.FuncDecl{}
		for _, fn := range file.Funcs {
			if len(fn.TypeParams) > 0 {
				continue
			}
			env := map[string]string{}
			for _, param := range fn.Params {
				env[param.Name] = genericTypeName(param.Type)
			}
			if err := monomorphizeStmts(fn.Body, env, generics, created, file); err != nil {
				return err
			}
		}
		if len(created) == 0 {
			continue
		}
		names := make([]string, 0, len(created))
		for name := range created {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			file.Funcs = append(file.Funcs, created[name])
		}
	}
	return nil
}

func monomorphizeStmts(stmts []frontend.Stmt, env map[string]string, generics map[string]*frontend.FuncDecl, created map[string]*frontend.FuncDecl, file *frontend.FileAST) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			if _, err := monomorphizeExpr(s.Value, env, generics, created, file); err != nil {
				return err
			}
		case *frontend.ThrowStmt:
			if _, err := monomorphizeExpr(s.Value, env, generics, created, file); err != nil {
				return err
			}
		case *frontend.BreakStmt, *frontend.ContinueStmt:
		case *frontend.PrintStmt:
			if _, err := monomorphizeExpr(s.Value, env, generics, created, file); err != nil {
				return err
			}
		case *frontend.ExpectStmt:
			if _, err := monomorphizeExpr(s.Cond, env, generics, created, file); err != nil {
				return err
			}
		case *frontend.FreeStmt:
			if _, err := monomorphizeExpr(s.Value, env, generics, created, file); err != nil {
				return err
			}
		case *frontend.LetStmt:
			valType, err := monomorphizeExpr(s.Value, env, generics, created, file)
			if err != nil {
				return err
			}
			if s.Type.Name != "" || s.Type.Elem != nil {
				env[s.Name] = genericTypeName(s.Type)
			} else {
				env[s.Name] = valType
			}
		case *frontend.AssignStmt:
			if _, err := monomorphizeExpr(s.Target, env, generics, created, file); err != nil {
				return err
			}
			if _, err := monomorphizeExpr(s.Value, env, generics, created, file); err != nil {
				return err
			}
		case *frontend.IfStmt:
			if _, err := monomorphizeExpr(s.Cond, env, generics, created, file); err != nil {
				return err
			}
			if err := monomorphizeStmts(s.Then, cloneStringMap(env), generics, created, file); err != nil {
				return err
			}
			if err := monomorphizeStmts(s.Else, cloneStringMap(env), generics, created, file); err != nil {
				return err
			}
		case *frontend.IfLetStmt:
			if _, err := monomorphizeExpr(s.Value, env, generics, created, file); err != nil {
				return err
			}
			thenEnv := cloneStringMap(env)
			thenEnv[s.Name] = "i32"
			if err := monomorphizeStmts(s.Then, thenEnv, generics, created, file); err != nil {
				return err
			}
			if err := monomorphizeStmts(s.Else, cloneStringMap(env), generics, created, file); err != nil {
				return err
			}
		case *frontend.WhileStmt:
			if _, err := monomorphizeExpr(s.Cond, env, generics, created, file); err != nil {
				return err
			}
			if err := monomorphizeStmts(s.Body, cloneStringMap(env), generics, created, file); err != nil {
				return err
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				if _, err := monomorphizeExpr(s.Iterable, env, generics, created, file); err != nil {
					return err
				}
			} else {
				if _, err := monomorphizeExpr(s.Start, env, generics, created, file); err != nil {
					return err
				}
				if _, err := monomorphizeExpr(s.End, env, generics, created, file); err != nil {
					return err
				}
			}
			bodyEnv := cloneStringMap(env)
			bodyEnv[s.Name] = "i32"
			if err := monomorphizeStmts(s.Body, bodyEnv, generics, created, file); err != nil {
				return err
			}
		case *frontend.MatchStmt:
			if _, err := monomorphizeExpr(s.Value, env, generics, created, file); err != nil {
				return err
			}
			for _, c := range s.Cases {
				if !c.Default {
					if _, err := monomorphizeExpr(c.Pattern, env, generics, created, file); err != nil {
						return err
					}
				}
				if err := monomorphizeStmts(c.Body, cloneStringMap(env), generics, created, file); err != nil {
					return err
				}
			}
		case *frontend.UnsafeStmt:
			if err := monomorphizeStmts(s.Body, env, generics, created, file); err != nil {
				return err
			}
		case *frontend.IslandStmt:
			if _, err := monomorphizeExpr(s.Size, env, generics, created, file); err != nil {
				return err
			}
			bodyEnv := cloneStringMap(env)
			bodyEnv[s.Name] = "island"
			if err := monomorphizeStmts(s.Body, bodyEnv, generics, created, file); err != nil {
				return err
			}
		case *frontend.ExprStmt:
			if _, err := monomorphizeExpr(s.Expr, env, generics, created, file); err != nil {
				return err
			}
		}
	}
	return nil
}

func monomorphizeExpr(expr frontend.Expr, env map[string]string, generics map[string]*frontend.FuncDecl, created map[string]*frontend.FuncDecl, file *frontend.FileAST) (string, error) {
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
	case *frontend.IdentExpr:
		return env[e.Name], nil
	case *frontend.FieldAccessExpr:
		if _, err := monomorphizeExpr(e.Base, env, generics, created, file); err != nil {
			return "", err
		}
		return "", nil
	case *frontend.IndexExpr:
		if _, err := monomorphizeExpr(e.Base, env, generics, created, file); err != nil {
			return "", err
		}
		if _, err := monomorphizeExpr(e.Index, env, generics, created, file); err != nil {
			return "", err
		}
		return "", nil
	case *frontend.UnaryExpr:
		_, err := monomorphizeExpr(e.X, env, generics, created, file)
		return "i32", err
	case *frontend.BinaryExpr:
		if _, err := monomorphizeExpr(e.Left, env, generics, created, file); err != nil {
			return "", err
		}
		if _, err := monomorphizeExpr(e.Right, env, generics, created, file); err != nil {
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
		return monomorphizeExpr(e.X, env, generics, created, file)
	case *frontend.AwaitExpr:
		return monomorphizeExpr(e.X, env, generics, created, file)
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if _, err := monomorphizeExpr(field.Value, env, generics, created, file); err != nil {
				return "", err
			}
		}
		return genericTypeName(e.Type), nil
	case *frontend.CallExpr:
		argTypes := make([]string, 0, len(e.Args))
		for _, arg := range e.Args {
			tname, err := monomorphizeExpr(arg, env, generics, created, file)
			if err != nil {
				return "", err
			}
			argTypes = append(argTypes, tname)
		}
		generic, ok := generics[e.Name]
		if !ok {
			return "", nil
		}
		if len(e.Args) != len(generic.Params) {
			return "", fmt.Errorf("%s: wrong argument count for generic function '%s'", frontend.FormatPos(e.At), e.Name)
		}
		subst := map[string]string{}
		for i, param := range generic.Params {
			if argTypes[i] == "" {
				return "", fmt.Errorf("%s: cannot infer generic argument for '%s' arg %d", frontend.FormatPos(e.Args[i].Pos()), e.Name, i+1)
			}
			if err := bindGenericType(param.Type, argTypes[i], generic.TypeParams, subst); err != nil {
				return "", fmt.Errorf("%s: %v", frontend.FormatPos(e.Args[i].Pos()), err)
			}
		}
		for _, tp := range generic.TypeParams {
			if subst[tp] == "" {
				return "", fmt.Errorf("%s: cannot infer generic argument '%s' for '%s'", frontend.FormatPos(e.At), tp, e.Name)
			}
		}
		name := mangleGenericName(generic.Name, generic.TypeParams, subst)
		if _, exists := created[name]; !exists {
			created[name] = cloneGenericFunc(generic, name, subst)
		}
		e.Name = name
		return substituteGenericTypeName(generic.ReturnType, subst), nil
	default:
		return "", nil
	}
}

func bindGenericType(param frontend.TypeRef, actual string, typeParams []string, subst map[string]string) error {
	if param.Kind == frontend.TypeRefNamed && contains(typeParams, param.Name) {
		if existing := subst[param.Name]; existing != "" && existing != actual {
			return fmt.Errorf("conflicting generic argument for '%s': %s vs %s", param.Name, existing, actual)
		}
		subst[param.Name] = actual
		return nil
	}
	return nil
}

func cloneGenericFunc(fn *frontend.FuncDecl, name string, subst map[string]string) *frontend.FuncDecl {
	out := *fn
	out.Name = name
	out.ExportName = ""
	out.TypeParams = nil
	out.Params = cloneParams(fn.Params, subst)
	out.ReturnType = substituteTypeRef(fn.ReturnType, subst)
	out.Throws = substituteTypeRef(fn.Throws, subst)
	out.Uses = append([]string(nil), fn.Uses...)
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
			out = append(out, &frontend.ReturnStmt{At: s.At, Value: cloneExpr(s.Value)})
		case *frontend.ThrowStmt:
			out = append(out, &frontend.ThrowStmt{At: s.At, Value: cloneExpr(s.Value)})
		case *frontend.BreakStmt:
			out = append(out, &frontend.BreakStmt{At: s.At})
		case *frontend.ContinueStmt:
			out = append(out, &frontend.ContinueStmt{At: s.At})
		case *frontend.PrintStmt:
			out = append(out, &frontend.PrintStmt{At: s.At, Value: cloneExpr(s.Value)})
		case *frontend.ExpectStmt:
			out = append(out, &frontend.ExpectStmt{At: s.At, Cond: cloneExpr(s.Cond)})
		case *frontend.FreeStmt:
			out = append(out, &frontend.FreeStmt{At: s.At, Value: cloneExpr(s.Value), Implicit: s.Implicit})
		case *frontend.LetStmt:
			out = append(out, &frontend.LetStmt{At: s.At, Name: s.Name, Type: substituteTypeRef(s.Type, subst), Mutable: s.Mutable, Const: s.Const, Value: cloneExpr(s.Value)})
		case *frontend.AssignStmt:
			var compoundValue frontend.Expr
			if s.CompoundValue != nil {
				compoundValue = cloneExpr(s.CompoundValue)
			}
			out = append(out, &frontend.AssignStmt{At: s.At, Target: cloneExpr(s.Target), Value: cloneExpr(s.Value), Op: s.Op, CompoundValue: compoundValue})
		case *frontend.IfStmt:
			out = append(out, &frontend.IfStmt{At: s.At, Cond: cloneExpr(s.Cond), Then: cloneStmts(s.Then, subst), Else: cloneStmts(s.Else, subst)})
		case *frontend.IfLetStmt:
			out = append(out, &frontend.IfLetStmt{At: s.At, Name: s.Name, Value: cloneExpr(s.Value), ValueLocal: s.ValueLocal, Then: cloneStmts(s.Then, subst), Else: cloneStmts(s.Else, subst)})
		case *frontend.WhileStmt:
			out = append(out, &frontend.WhileStmt{At: s.At, Cond: cloneExpr(s.Cond), Body: cloneStmts(s.Body, subst)})
		case *frontend.ForRangeStmt:
			var start, end, iterable frontend.Expr
			if s.Start != nil {
				start = cloneExpr(s.Start)
			}
			if s.End != nil {
				end = cloneExpr(s.End)
			}
			if s.Iterable != nil {
				iterable = cloneExpr(s.Iterable)
			}
			out = append(out, &frontend.ForRangeStmt{At: s.At, Name: s.Name, Start: start, End: end, Iterable: iterable, IterableLocal: s.IterableLocal, IndexLocal: s.IndexLocal, EndLocal: s.EndLocal, Body: cloneStmts(s.Body, subst)})
		case *frontend.MatchStmt:
			cases := make([]frontend.MatchCase, 0, len(s.Cases))
			for _, c := range s.Cases {
				cases = append(cases, frontend.MatchCase{At: c.At, Pattern: cloneExpr(c.Pattern), Default: c.Default, Body: cloneStmts(c.Body, subst)})
			}
			out = append(out, &frontend.MatchStmt{At: s.At, Value: cloneExpr(s.Value), ScrutineeLocal: s.ScrutineeLocal, Cases: cases})
		case *frontend.UnsafeStmt:
			out = append(out, &frontend.UnsafeStmt{At: s.At, Body: cloneStmts(s.Body, subst)})
		case *frontend.IslandStmt:
			out = append(out, &frontend.IslandStmt{At: s.At, Size: cloneExpr(s.Size), Name: s.Name, Body: cloneStmts(s.Body, subst)})
		case *frontend.ExprStmt:
			out = append(out, &frontend.ExprStmt{At: s.At, Expr: cloneExpr(s.Expr)})
		default:
			out = append(out, stmt)
		}
	}
	return out
}

func cloneExpr(expr frontend.Expr) frontend.Expr {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return &frontend.NumberExpr{At: e.At, Value: e.Value}
	case *frontend.BoolLitExpr:
		return &frontend.BoolLitExpr{At: e.At, Value: e.Value}
	case *frontend.NoneLitExpr:
		return &frontend.NoneLitExpr{At: e.At}
	case *frontend.SomePatternExpr:
		return &frontend.SomePatternExpr{At: e.At, Name: e.Name}
	case *frontend.StringLitExpr:
		return &frontend.StringLitExpr{At: e.At, Value: append([]byte(nil), e.Value...)}
	case *frontend.IdentExpr:
		return &frontend.IdentExpr{At: e.At, Name: e.Name}
	case *frontend.FieldAccessExpr:
		return &frontend.FieldAccessExpr{At: e.At, Base: cloneExpr(e.Base), Field: e.Field, EnumType: e.EnumType, EnumOrdinal: e.EnumOrdinal}
	case *frontend.IndexExpr:
		return &frontend.IndexExpr{At: e.At, Base: cloneExpr(e.Base), Index: cloneExpr(e.Index)}
	case *frontend.UnaryExpr:
		return &frontend.UnaryExpr{At: e.At, Op: e.Op, X: cloneExpr(e.X)}
	case *frontend.BinaryExpr:
		return &frontend.BinaryExpr{At: e.At, Op: e.Op, Left: cloneExpr(e.Left), Right: cloneExpr(e.Right)}
	case *frontend.TryExpr:
		return &frontend.TryExpr{At: e.At, X: cloneExpr(e.X)}
	case *frontend.AwaitExpr:
		return &frontend.AwaitExpr{At: e.At, X: cloneExpr(e.X)}
	case *frontend.CallExpr:
		args := make([]frontend.Expr, 0, len(e.Args))
		for _, arg := range e.Args {
			args = append(args, cloneExpr(arg))
		}
		return &frontend.CallExpr{At: e.At, Name: e.Name, Args: args}
	case *frontend.StructLitExpr:
		fields := make([]frontend.StructFieldInit, 0, len(e.Fields))
		for _, field := range e.Fields {
			fields = append(fields, frontend.StructFieldInit{At: field.At, Name: field.Name, Value: cloneExpr(field.Value)})
		}
		return &frontend.StructLitExpr{At: e.At, Type: e.Type, Fields: fields}
	default:
		return expr
	}
}

func substituteTypeRef(ref frontend.TypeRef, subst map[string]string) frontend.TypeRef {
	out := ref
	if ref.Kind == frontend.TypeRefNamed {
		if concrete := subst[ref.Name]; concrete != "" {
			out.Name = concrete
		}
		return out
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
	return base + "__" + strings.Join(parts, "_")
}

func sanitizeGenericType(tname string) string {
	replacer := strings.NewReplacer(".", "_", "[", "slice_", "]", "", "?", "_opt", " ", "_")
	return replacer.Replace(tname)
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
	default:
		if canonical, ok := canonicalBuiltinType(ref.Name); ok {
			return canonical
		}
		return ref.Name
	}
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
