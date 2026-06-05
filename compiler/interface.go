package compiler

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/t4iface"
)

func GenerateInterfaceFile(inputPath string) ([]byte, error) {
	raw, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, err
	}
	return GenerateInterfaceFromSource(raw, inputPath)
}

func GenerateInterfaceFromSource(src []byte, filename string) ([]byte, error) {
	file, err := frontend.ParseFile(src, filename)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	if file.Module != "" {
		fmt.Fprintf(&b, "module %s\n\n", file.Module)
	}
	explicitPublic := interfaceUsesExplicitPublic(file)
	imports := interfaceImportsForPublicSurface(file, explicitPublic)
	for _, imp := range imports {
		writeInterfaceImport(&b, imp, explicitPublic)
	}
	if len(imports) > 0 {
		b.WriteByte('\n')
	}
	for _, en := range file.Enums {
		if interfaceDeclPublic(file, en.Public) {
			writeInterfaceEnum(&b, en, explicitPublic)
		}
	}
	for _, st := range file.Structs {
		if interfaceDeclPublic(file, st.Public) {
			writeInterfaceStruct(&b, st.Name, st.TypeParams, st.Fields, explicitPublic)
		}
	}
	for _, st := range file.States {
		if !interfaceDeclPublic(file, st.Public) {
			continue
		}
		var fields []frontend.FieldDecl
		for _, field := range st.Fields {
			fields = append(fields, frontend.FieldDecl{At: field.At, Name: field.Name, Type: field.Type})
		}
		writeInterfaceStruct(&b, st.Name, nil, fields, explicitPublic)
	}
	for _, proto := range file.Protocols {
		if !interfaceDeclPublic(file, proto.Public) {
			continue
		}
		if explicitPublic {
			b.WriteString("pub ")
		}
		fmt.Fprintf(&b, "protocol %s:\n", proto.Name)
		for _, req := range proto.Requirements {
			fmt.Fprintf(&b, "    %s\n", formatLSPFuncSigDecl(req))
		}
		b.WriteByte('\n')
	}
	for _, ext := range file.Extensions {
		if !interfaceDeclPublic(file, ext.Public) {
			continue
		}
		writeInterfaceExtension(&b, ext, explicitPublic)
	}
	for _, impl := range file.Impls {
		if !interfaceImplPublic(file, impl, explicitPublic) {
			continue
		}
		writeInterfaceImpl(&b, impl)
	}
	for _, fn := range file.Funcs {
		if fn.Synthetic || fn.ExtensionOf != "" || !interfaceDeclPublic(file, fn.Public) {
			continue
		}
		if explicitPublic {
			b.WriteString("pub ")
		}
		fmt.Fprintf(&b, "%s:\n", formatLSPFuncDetail(fn))
		fmt.Fprintf(&b, "%s\n\n", interfaceFunctionBody(fn))
	}
	writeInterfaceHashOnlySurface(&b, file, explicitPublic)
	return t4iface.WithHashHeader(b.Bytes()), nil
}

func InterfaceFingerprintFromSource(src []byte, filename string) (string, error) {
	raw, err := GenerateInterfaceFromSource(src, filename)
	if err != nil {
		return "", err
	}
	hash, _, ok, err := t4iface.SplitHashHeader(raw)
	if err != nil {
		return "", err
	}
	if !ok {
		return t4iface.FingerprintBody(raw), nil
	}
	return hash, nil
}

func InterfaceFingerprintFromT4I(raw []byte) (string, error) {
	return t4iface.ValidateHash(raw)
}

func ValidateInterfaceAgainstSource(src []byte, iface []byte, filename string) error {
	expected, err := InterfaceFingerprintFromSource(src, filename)
	if err != nil {
		return err
	}
	actual, err := InterfaceFingerprintFromT4I(iface)
	if err != nil {
		return err
	}
	if expected != actual {
		return fmt.Errorf("%s: public API mismatch: source %s, interface %s", filename, expected, actual)
	}
	return nil
}

func writeInterfaceImport(b *bytes.Buffer, imp frontend.ImportDecl, explicitPublic bool) {
	if explicitPublic && imp.Public {
		b.WriteString("pub ")
	}
	if len(imp.Items) > 0 {
		fmt.Fprintf(b, "import %s.{%s}\n", imp.Path, strings.Join(imp.Items, ", "))
		return
	}
	if imp.Alias != "" {
		fmt.Fprintf(b, "import %s as %s\n", imp.Path, imp.Alias)
	} else {
		fmt.Fprintf(b, "import %s\n", imp.Path)
	}
}

func interfaceUsesExplicitPublic(file *frontend.FileAST) bool {
	if file == nil {
		return false
	}
	for _, imp := range file.Imports {
		if imp.Public {
			return true
		}
	}
	for _, en := range file.Enums {
		if en.Public {
			return true
		}
	}
	for _, st := range file.Structs {
		if st.Public {
			return true
		}
	}
	for _, st := range file.States {
		if st.Public {
			return true
		}
	}
	for _, view := range file.Views {
		if view.Public {
			return true
		}
	}
	for _, proto := range file.Protocols {
		if proto.Public {
			return true
		}
	}
	for _, ext := range file.Extensions {
		if ext.Public {
			return true
		}
	}
	for _, glob := range file.Globals {
		if glob.Public {
			return true
		}
	}
	for _, fn := range file.Funcs {
		if fn.Public {
			return true
		}
	}
	return false
}

func interfaceDeclPublic(file *frontend.FileAST, public bool) bool {
	if !interfaceUsesExplicitPublic(file) {
		return true
	}
	return public
}

func interfaceImportsForPublicSurface(file *frontend.FileAST, explicitPublic bool) []frontend.ImportDecl {
	if file == nil {
		return nil
	}
	if !explicitPublic {
		return append([]frontend.ImportDecl(nil), file.Imports...)
	}
	refs := interfacePublicTypeRefs(file, explicitPublic)
	out := make([]frontend.ImportDecl, 0, len(file.Imports))
	for _, imp := range file.Imports {
		if imp.Public || interfaceImportUsedByRefs(imp, refs) {
			out = append(out, imp)
		}
	}
	return out
}

func interfacePublicTypeRefs(file *frontend.FileAST, explicitPublic bool) map[string]struct{} {
	refs := map[string]struct{}{}
	add := func(ref frontend.TypeRef) {
		addInterfaceTypeRef(refs, ref)
	}
	for _, en := range file.Enums {
		if !interfaceDeclPublic(file, en.Public) {
			continue
		}
		for _, item := range en.Cases {
			for _, payload := range item.Payload {
				add(payload)
			}
		}
	}
	for _, st := range file.Structs {
		if !interfaceDeclPublic(file, st.Public) {
			continue
		}
		for _, field := range st.Fields {
			add(field.Type)
		}
	}
	for _, st := range file.States {
		if !interfaceDeclPublic(file, st.Public) {
			continue
		}
		for _, field := range st.Fields {
			add(field.Type)
		}
	}
	for _, view := range file.Views {
		if !interfaceDeclPublic(file, view.Public) {
			continue
		}
		add(view.StateName)
		for _, binding := range view.Bindings {
			add(binding.Type)
		}
		for _, style := range view.Styles {
			add(style.Type)
		}
		for _, item := range view.Accessibility {
			add(item.Type)
		}
	}
	for _, proto := range file.Protocols {
		if !interfaceDeclPublic(file, proto.Public) {
			continue
		}
		for _, req := range proto.Requirements {
			addInterfaceFuncSigTypeRefs(refs, req)
		}
	}
	for _, ext := range file.Extensions {
		if !interfaceDeclPublic(file, ext.Public) {
			continue
		}
		add(ext.Target)
		for _, method := range ext.Methods {
			addInterfaceFuncTypeRefs(refs, method)
		}
	}
	for _, impl := range file.Impls {
		if !interfaceImplPublic(file, impl, explicitPublic) {
			continue
		}
		add(impl.Type)
		add(impl.Protocol)
	}
	for _, glob := range file.Globals {
		if !interfaceDeclPublic(file, glob.Public) {
			continue
		}
		add(glob.Type)
	}
	for _, fn := range file.Funcs {
		if fn.Synthetic || fn.ExtensionOf != "" || !interfaceDeclPublic(file, fn.Public) {
			continue
		}
		addInterfaceFuncTypeRefs(refs, fn)
	}
	return refs
}

func addInterfaceFuncTypeRefs(refs map[string]struct{}, fn *frontend.FuncDecl) {
	if fn == nil {
		return
	}
	for _, bound := range fn.TypeParamBounds {
		addInterfaceTypeRef(refs, bound.Bound)
	}
	for _, param := range fn.Params {
		addInterfaceTypeRef(refs, param.Type)
	}
	addInterfaceTypeRef(refs, fn.ReturnType)
	if fn.HasThrows {
		addInterfaceTypeRef(refs, fn.Throws)
	}
}

func addInterfaceFuncSigTypeRefs(refs map[string]struct{}, sig frontend.FuncSigDecl) {
	for _, param := range sig.Params {
		addInterfaceTypeRef(refs, param.Type)
	}
	addInterfaceTypeRef(refs, sig.ReturnType)
	if sig.HasThrows {
		addInterfaceTypeRef(refs, sig.Throws)
	}
}

func addInterfaceTypeRef(refs map[string]struct{}, ref frontend.TypeRef) {
	for _, arg := range ref.TypeArgs {
		addInterfaceTypeRef(refs, arg)
	}
	switch ref.Kind {
	case frontend.TypeRefSlice, frontend.TypeRefArray, frontend.TypeRefOptional:
		if ref.Elem != nil {
			addInterfaceTypeRef(refs, *ref.Elem)
		}
	case frontend.TypeRefFunction:
		for _, param := range ref.Params {
			addInterfaceTypeRef(refs, param)
		}
		if ref.Return != nil {
			addInterfaceTypeRef(refs, *ref.Return)
		}
		if ref.Throws != nil {
			addInterfaceTypeRef(refs, *ref.Throws)
		}
	default:
		if ref.Name != "" {
			refs[ref.Name] = struct{}{}
		}
	}
}

func interfaceImportUsedByRefs(imp frontend.ImportDecl, refs map[string]struct{}) bool {
	if len(refs) == 0 {
		return false
	}
	alias := imp.Alias
	if alias == "" {
		alias = lastPathSegment(imp.Path)
	}
	for name := range refs {
		if name == alias || strings.HasPrefix(name, alias+".") || strings.HasPrefix(name, imp.Path+".") {
			return true
		}
		for _, item := range imp.Items {
			if name == item || lastPathSegment(name) == item {
				return true
			}
		}
	}
	return false
}

func lastPathSegment(path string) string {
	if idx := strings.LastIndex(path, "."); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

func writeInterfaceHashOnlySurface(b *bytes.Buffer, file *frontend.FileAST, explicitPublic bool) {
	wroteHeader := false
	writeHeader := func() {
		if wroteHeader {
			return
		}
		b.WriteString("// hash-only public surface:\n")
		wroteHeader = true
	}
	for _, glob := range file.Globals {
		if !interfaceDeclPublic(file, glob.Public) {
			continue
		}
		writeHeader()
		fmt.Fprintf(b, "// global %s\n", formatLSPGlobalDetail(glob))
	}
	for _, view := range file.Views {
		if !interfaceDeclPublic(file, view.Public) {
			continue
		}
		writeHeader()
		fmt.Fprintf(b, "// view %s(%s)\n", view.Name, formatLSPTypeRef(view.StateName))
		for _, binding := range view.Bindings {
			fmt.Fprintf(b, "// view %s binding %s: %s\n", view.Name, binding.Name, formatLSPTypeRef(binding.Type))
		}
		for _, event := range view.Events {
			fmt.Fprintf(b, "// view %s event %s -> %s\n", view.Name, event.Name, event.Command)
		}
		for _, command := range view.Commands {
			fmt.Fprintf(b, "// view %s command %s\n", view.Name, command.Name)
		}
		for _, style := range view.Styles {
			fmt.Fprintf(b, "// view %s style %s: %s\n", view.Name, style.Name, formatLSPTypeRef(style.Type))
		}
		for _, item := range view.Accessibility {
			fmt.Fprintf(b, "// view %s accessibility %s: %s\n", view.Name, item.Name, formatLSPTypeRef(item.Type))
		}
	}
	for _, fn := range file.Funcs {
		if fn.Synthetic || fn.ExtensionOf != "" || !interfaceDeclPublic(file, fn.Public) {
			continue
		}
		source, ok := interfaceBorrowedReturnExpr(fn)
		if !ok {
			continue
		}
		writeHeader()
		fmt.Fprintf(b, "// func %s lifetime return=borrow source=%s provenance=param lifetime=call\n", formatLSPFuncDetail(fn), source)
	}
	if wroteHeader {
		b.WriteByte('\n')
	}
}

func InterfaceOutputPath(inputPath string) string {
	ext := filepath.Ext(inputPath)
	if ext == "" {
		return inputPath + T4InterfaceExtension
	}
	return strings.TrimSuffix(inputPath, ext) + T4InterfaceExtension
}

func writeInterfaceEnum(b *bytes.Buffer, en *frontend.EnumDecl, explicitPublic bool) {
	if explicitPublic {
		b.WriteString("pub ")
	}
	fmt.Fprintf(b, "enum %s:\n", en.Name)
	for _, item := range en.Cases {
		if len(item.Payload) == 0 {
			fmt.Fprintf(b, "    case %s\n", item.Name)
			continue
		}
		payloads := make([]string, 0, len(item.Payload))
		for _, payload := range item.Payload {
			payloads = append(payloads, formatLSPTypeRef(payload))
		}
		fmt.Fprintf(b, "    case %s(%s)\n", item.Name, strings.Join(payloads, ", "))
	}
	b.WriteByte('\n')
}

func writeInterfaceStruct(b *bytes.Buffer, name string, typeParams []string, fields []frontend.FieldDecl, explicitPublic bool) {
	if explicitPublic {
		b.WriteString("pub ")
	}
	fmt.Fprintf(b, "struct %s%s:\n", name, formatLSPTypeParams(typeParams, nil))
	if len(fields) == 0 {
		b.WriteString("    _empty: Int\n\n")
		return
	}
	for _, field := range fields {
		fmt.Fprintf(b, "    %s: %s\n", field.Name, formatLSPTypeRef(field.Type))
	}
	b.WriteByte('\n')
}

func writeInterfaceExtension(b *bytes.Buffer, ext *frontend.ExtensionDecl, explicitPublic bool) {
	if explicitPublic {
		b.WriteString("pub ")
	}
	target := formatLSPTypeRef(ext.Target)
	fmt.Fprintf(b, "extension %s:\n", target)
	for _, method := range ext.Methods {
		if method == nil || method.Synthetic {
			continue
		}
		fmt.Fprintf(b, "    %s:\n", formatInterfaceExtensionMethodDetail(method, target))
		body := strings.TrimSuffix(interfaceFunctionBody(method), "\n")
		for _, line := range strings.Split(body, "\n") {
			fmt.Fprintf(b, "    %s\n", line)
		}
	}
	b.WriteByte('\n')
}

func formatInterfaceExtensionMethodDetail(fn *frontend.FuncDecl, target string) string {
	if fn == nil {
		return ""
	}
	copyFn := *fn
	prefix := target + "."
	if strings.HasPrefix(copyFn.Name, prefix) {
		copyFn.Name = strings.TrimPrefix(copyFn.Name, prefix)
	}
	return formatLSPFuncDetail(&copyFn)
}

func writeInterfaceImpl(b *bytes.Buffer, impl *frontend.ImplDecl) {
	fmt.Fprintf(b, "impl %s\n\n", formatLSPImplName(impl))
}

func interfaceImplPublic(file *frontend.FileAST, impl *frontend.ImplDecl, explicitPublic bool) bool {
	if impl == nil {
		return false
	}
	if !explicitPublic {
		return true
	}
	return interfaceTypeRefPublic(file, impl.Type) && interfaceTypeRefPublic(file, impl.Protocol)
}

func interfaceTypeRefPublic(file *frontend.FileAST, ref frontend.TypeRef) bool {
	if file == nil || ref.Name == "" {
		return true
	}
	name := ref.Name
	if file.Module != "" {
		name = strings.TrimPrefix(name, file.Module+".")
	}
	for _, en := range file.Enums {
		if en.Name == name {
			return interfaceDeclPublic(file, en.Public)
		}
	}
	for _, st := range file.Structs {
		if st.Name == name {
			return interfaceDeclPublic(file, st.Public)
		}
	}
	for _, st := range file.States {
		if st.Name == name {
			return interfaceDeclPublic(file, st.Public)
		}
	}
	for _, proto := range file.Protocols {
		if proto.Name == name {
			return interfaceDeclPublic(file, proto.Public)
		}
	}
	return true
}

func interfaceReturnLiteral(ref frontend.TypeRef) string {
	if ref.Kind == frontend.TypeRefOptional {
		return "none"
	}
	name := canonicalLSPTypeName(formatLSPTypeRef(ref))
	switch name {
	case "bool":
		return "false"
	case "str":
		return "\"\""
	default:
		return "0"
	}
}

func interfaceReturnExpr(fn *frontend.FuncDecl) string {
	if expr, ok := interfaceTryReturnExpr(fn); ok {
		return expr
	}
	if expr, ok := interfaceOptionalParamReturnExpr(fn); ok {
		return expr
	}
	if expr, ok := interfaceOptionalAggregateReturnExpr(fn); ok {
		return expr
	}
	if fn.ReturnType.Kind == frontend.TypeRefFunction {
		if expr, ok := interfaceFunctionReturnExpr(fn); ok {
			return expr
		}
		return interfaceFunctionClosureLiteral(fn.ReturnType, "        ")
	}
	if expr, ok := interfaceBorrowedReturnExpr(fn); ok {
		return expr
	}
	if expr, ok := interfaceAggregateReturnExpr(fn); ok {
		return expr
	}
	if expr, ok := interfaceSameTypedParameterReturnExpr(fn); ok {
		return expr
	}
	return interfaceReturnLiteral(fn.ReturnType)
}

func interfaceTryReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	if fn == nil || !fn.HasThrows {
		return "", false
	}
	paramNames := map[string]bool{}
	for _, param := range fn.Params {
		paramNames[param.Name] = true
	}
	aliases := map[string]string{}
	for _, stmt := range fn.Body {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames); ok {
				aliases[s.Name] = value
			}
			continue
		case *frontend.AssignStmt:
			target, ok := s.Target.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if _, ok := aliases[target.Name]; !ok {
				continue
			}
			value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames)
			if !ok {
				continue
			}
			aliases[target.Name] = value
			continue
		}
		ret, ok := stmt.(*frontend.ReturnStmt)
		if !ok {
			continue
		}
		if _, ok := ret.Value.(*frontend.TryExpr); !ok {
			continue
		}
		formatted, ok := interfaceAggregateStubExprWithAliases(ret.Value, aliases)
		if !ok {
			formatted, ok = interfaceContractExpr(ret.Value)
		}
		if !ok || (!interfaceExprRefsAnyParam(ret.Value, paramNames) && !interfaceExprRefsAnyAlias(ret.Value, aliases)) {
			return "", false
		}
		return formatted, true
	}
	return "", false
}

func interfaceOptionalParamReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	if fn == nil || fn.ReturnType.Kind != frontend.TypeRefOptional || fn.ReturnType.Elem == nil {
		return "", false
	}
	elemType := formatLSPTypeRef(*fn.ReturnType.Elem)
	paramNames := map[string]bool{}
	paramHasElemType := map[string]bool{}
	for _, param := range fn.Params {
		paramNames[param.Name] = true
		if formatLSPTypeRef(param.Type) == elemType {
			paramHasElemType[param.Name] = true
		}
	}
	optionalLocals := map[string]string{}
	for _, stmt := range fn.Body {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if formatLSPTypeRef(s.Type) == formatLSPTypeRef(fn.ReturnType) {
				optionalLocals[s.Name] = ""
				if value, ok := s.Value.(*frontend.IdentExpr); ok && paramHasElemType[value.Name] {
					optionalLocals[s.Name] = value.Name
				} else if value, ok := interfaceParamPathExpr(s.Value, optionalLocals, paramNames); ok {
					optionalLocals[s.Name] = value
				}
			}
		case *frontend.AssignStmt:
			target, ok := s.Target.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if _, ok := optionalLocals[target.Name]; !ok {
				continue
			}
			value, ok := interfaceParamPathExpr(s.Value, optionalLocals, paramNames)
			if !ok {
				continue
			}
			optionalLocals[target.Name] = value
		case *frontend.ReturnStmt:
			id, ok := s.Value.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if paramName := optionalLocals[id.Name]; paramName != "" {
				return paramName, true
			}
		case *frontend.IfLetStmt:
			value, ok := interfaceParamPathExpr(s.Value, optionalLocals, paramNames)
			if !ok || s.Name == "" {
				continue
			}
			branchAliases := interfaceAliasMapCopy(optionalLocals)
			branchAliases[s.Name] = value
			if expr, ok := interfaceOptionalReturnFromStmts(s.Then, branchAliases); ok {
				return expr, true
			}
		case *frontend.MatchStmt:
			value, ok := interfaceParamPathExpr(s.Value, optionalLocals, paramNames)
			if !ok {
				continue
			}
			for _, c := range s.Cases {
				name, ok := interfaceOptionalSomePatternName(c.Pattern)
				if !ok {
					continue
				}
				branchAliases := interfaceAliasMapCopy(optionalLocals)
				branchAliases[name] = value
				if expr, ok := interfaceOptionalReturnFromStmts(c.Body, branchAliases); ok {
					return expr, true
				}
			}
		}
	}
	return "", false
}

func interfaceOptionalAggregateReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	if fn == nil || fn.ReturnType.Kind != frontend.TypeRefOptional {
		return "", false
	}
	paramNames := map[string]bool{}
	for _, param := range fn.Params {
		paramNames[param.Name] = true
	}
	aliases := map[string]string{}
	optionalAggregates := map[string]string{}
	return interfaceOptionalAggregateReturnFromStmts(fn.Body, aliases, optionalAggregates, paramNames, formatLSPTypeRef(fn.ReturnType))
}

func interfaceOptionalAggregateReturnFromStmts(stmts []frontend.Stmt, aliases, optionalAggregates map[string]string, params map[string]bool, returnType string) (string, bool) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if formatLSPTypeRef(s.Type) == returnType {
				optionalAggregates[s.Name] = ""
				if value, ok := interfaceOptionalAggregateExpr(s.Value, aliases, params); ok {
					optionalAggregates[s.Name] = value
				}
				continue
			}
			if value, ok := interfaceParamPathExpr(s.Value, aliases, params); ok {
				aliases[s.Name] = value
			}
		case *frontend.AssignStmt:
			target, ok := s.Target.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if _, ok := optionalAggregates[target.Name]; ok {
				value, ok := interfaceOptionalAggregateExpr(s.Value, aliases, params)
				if ok {
					optionalAggregates[target.Name] = value
				}
				continue
			}
			if _, ok := aliases[target.Name]; !ok {
				continue
			}
			if value, ok := interfaceParamPathExpr(s.Value, aliases, params); ok {
				aliases[target.Name] = value
			}
		case *frontend.ReturnStmt:
			id, ok := s.Value.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if value := optionalAggregates[id.Name]; value != "" {
				return value, true
			}
		case *frontend.IfStmt:
			thenAliases := interfaceAliasMapCopy(aliases)
			thenAggregates := interfaceAliasMapCopy(optionalAggregates)
			thenExpr, thenReturned := interfaceOptionalAggregateReturnFromStmts(s.Then, thenAliases, thenAggregates, params, returnType)

			elseAliases := interfaceAliasMapCopy(aliases)
			elseAggregates := interfaceAliasMapCopy(optionalAggregates)
			elseExpr, elseReturned := interfaceOptionalAggregateReturnFromStmts(s.Else, elseAliases, elseAggregates, params, returnType)

			if thenReturned && elseReturned && thenExpr == elseExpr {
				return thenExpr, true
			}
			if !thenReturned && !elseReturned {
				interfaceMergeEqualAliasState(aliases, thenAliases, elseAliases)
				interfaceMergeEqualOptionalAggregateState(optionalAggregates, thenAggregates, elseAggregates)
			}
		case *frontend.IfLetStmt:
			thenAliases := interfaceAliasMapCopy(aliases)
			if value, ok := interfaceParamPathExpr(s.Value, aliases, params); ok && s.Name != "" {
				thenAliases[s.Name] = value
			}
			thenAggregates := interfaceAliasMapCopy(optionalAggregates)
			thenExpr, thenReturned := interfaceOptionalAggregateReturnFromStmts(s.Then, thenAliases, thenAggregates, params, returnType)

			elseAliases := interfaceAliasMapCopy(aliases)
			elseAggregates := interfaceAliasMapCopy(optionalAggregates)
			elseExpr, elseReturned := interfaceOptionalAggregateReturnFromStmts(s.Else, elseAliases, elseAggregates, params, returnType)

			if thenReturned && elseReturned && thenExpr == elseExpr {
				return thenExpr, true
			}
			if !thenReturned && !elseReturned {
				interfaceMergeEqualAliasState(aliases, thenAliases, elseAliases)
				interfaceMergeEqualOptionalAggregateState(optionalAggregates, thenAggregates, elseAggregates)
			}
		case *frontend.MatchStmt:
			if expr, ok := interfaceOptionalAggregateMatchReturnExpr(s, aliases, optionalAggregates, params, returnType); ok {
				return expr, true
			}
		}
	}
	return "", false
}

func interfaceBorrowedReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	if fn == nil || fn.ReturnOwnership != "borrow" {
		return "", false
	}
	params := map[string]bool{}
	paramOrder := []string{}
	returnType := formatLSPTypeRef(fn.ReturnType)
	for _, param := range fn.Params {
		if param.Ownership == "borrow" {
			params[param.Name] = true
			if formatLSPTypeRef(param.Type) == returnType {
				paramOrder = append(paramOrder, param.Name)
			}
		}
	}
	if len(params) == 0 {
		return "", false
	}
	for _, stmt := range fn.Body {
		ret, ok := stmt.(*frontend.ReturnStmt)
		if !ok {
			continue
		}
		if source, ok := interfaceBorrowedSourceParamExpr(ret.Value, params); ok {
			return source, true
		}
	}
	if len(paramOrder) == 1 {
		return paramOrder[0], true
	}
	return "", false
}

func interfaceBorrowedSourceParamExpr(expr frontend.Expr, params map[string]bool) (string, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		if params[e.Name] {
			return e.Name, true
		}
	case *frontend.FieldAccessExpr:
		return interfaceBorrowedSourceParamExpr(e.Base, params)
	case *frontend.CallExpr:
		method, ok := interfaceBorrowedViewMethod(e.Name)
		if ok && len(e.Args) > 0 {
			switch method {
			case "borrow", "window", "prefix", "suffix":
				return interfaceBorrowedSourceParamExpr(e.Args[0], params)
			}
		}
		return interfaceBorrowedSourceParamMethodCall(e.Name, params)
	}
	return "", false
}

func interfaceBorrowedSourceParamMethodCall(name string, params map[string]bool) (string, bool) {
	idx := strings.LastIndex(name, ".")
	if idx < 0 {
		return "", false
	}
	receiver := name[:idx]
	method := name[idx+1:]
	switch method {
	case "borrow", "window", "prefix", "suffix":
	default:
		return "", false
	}
	root := receiver
	if dot := strings.Index(root, "."); dot >= 0 {
		root = root[:dot]
	}
	if params[root] {
		return root, true
	}
	return "", false
}

func interfaceBorrowedViewMethod(name string) (string, bool) {
	if strings.HasPrefix(name, "__method.") {
		method := strings.TrimPrefix(name, "__method.")
		switch method {
		case "borrow", "window", "prefix", "suffix":
			return method, true
		}
	}
	if name == "core.string_borrow" {
		return "borrow", true
	}
	if strings.HasPrefix(name, "core.slice_borrow_") {
		return "borrow", true
	}
	if name == "core.string_window" || name == "core.string_prefix" || name == "core.string_suffix" {
		return strings.TrimPrefix(name, "core.string_"), true
	}
	if strings.HasPrefix(name, "core.slice_window_") {
		return "window", true
	}
	if strings.HasPrefix(name, "core.slice_prefix_") {
		return "prefix", true
	}
	if strings.HasPrefix(name, "core.slice_suffix_") {
		return "suffix", true
	}
	return "", false
}

func interfaceOptionalAggregateMatchReturnExpr(match *frontend.MatchStmt, aliases, optionalAggregates map[string]string, params map[string]bool, returnType string) (string, bool) {
	if match == nil || len(match.Cases) == 0 {
		return "", false
	}
	var commonExpr string
	allReturned := true
	caseAliases := make([]map[string]string, 0, len(match.Cases))
	caseAggregates := make([]map[string]string, 0, len(match.Cases))
	for _, c := range match.Cases {
		if c.Guard != nil {
			return "", false
		}
		branchAliases := interfaceAliasMapCopy(aliases)
		branchAggregates := interfaceAliasMapCopy(optionalAggregates)
		expr, returned := interfaceOptionalAggregateReturnFromStmts(c.Body, branchAliases, branchAggregates, params, returnType)
		if returned {
			if commonExpr == "" {
				commonExpr = expr
			} else if commonExpr != expr {
				return "", false
			}
		} else {
			allReturned = false
		}
		caseAliases = append(caseAliases, branchAliases)
		caseAggregates = append(caseAggregates, branchAggregates)
	}
	if allReturned && commonExpr != "" {
		return commonExpr, true
	}
	if !allReturned && commonExpr == "" {
		interfaceMergeEqualAliasStateAcross(aliases, caseAliases)
		interfaceMergeEqualOptionalAggregateStateAcross(optionalAggregates, caseAggregates)
	}
	return "", false
}

func interfaceMergeEqualAliasState(dst, left, right map[string]string) {
	for key := range dst {
		if left[key] == right[key] {
			dst[key] = left[key]
			continue
		}
		delete(dst, key)
	}
}

func interfaceMergeEqualOptionalAggregateState(dst, left, right map[string]string) {
	for key := range dst {
		value, ok := interfaceMergeOptionalAggregateValue(left[key], right[key])
		if ok {
			dst[key] = value
			continue
		}
		dst[key] = ""
	}
}

func interfaceMergeOptionalAggregateValue(values ...string) (string, bool) {
	merged := ""
	for _, value := range values {
		if value == "" {
			continue
		}
		if merged == "" {
			merged = value
			continue
		}
		if merged != value {
			return "", false
		}
	}
	return merged, true
}

func interfaceMergeEqualAliasStateAcross(dst map[string]string, states []map[string]string) {
	for key := range dst {
		value, ok := interfaceCommonStateValue(key, states)
		if ok {
			dst[key] = value
			continue
		}
		delete(dst, key)
	}
}

func interfaceMergeEqualOptionalAggregateStateAcross(dst map[string]string, states []map[string]string) {
	for key := range dst {
		values := make([]string, 0, len(states))
		for _, state := range states {
			values = append(values, state[key])
		}
		value, ok := interfaceMergeOptionalAggregateValue(values...)
		if ok {
			dst[key] = value
			continue
		}
		dst[key] = ""
	}
}

func interfaceCommonStateValue(key string, states []map[string]string) (string, bool) {
	if len(states) == 0 {
		return "", false
	}
	value, ok := states[0][key]
	if !ok {
		return "", false
	}
	for _, state := range states[1:] {
		if state[key] != value {
			return "", false
		}
	}
	return value, true
}

func interfaceOptionalAggregateExpr(expr frontend.Expr, aliases map[string]string, params map[string]bool) (string, bool) {
	if !interfaceDirectAggregateExpr(expr) {
		return "", false
	}
	formatted, ok := interfaceAggregateStubExprWithAliases(expr, aliases)
	if !ok {
		return "", false
	}
	if !interfaceExprRefsAnyParam(expr, params) && !interfaceExprRefsAnyAlias(expr, aliases) {
		return "", false
	}
	return formatted, true
}

func interfaceOptionalReturnFromStmts(stmts []frontend.Stmt, aliases map[string]string) (string, bool) {
	for _, stmt := range stmts {
		ret, ok := stmt.(*frontend.ReturnStmt)
		if !ok {
			continue
		}
		id, ok := ret.Value.(*frontend.IdentExpr)
		if !ok {
			continue
		}
		if paramName := aliases[id.Name]; paramName != "" {
			return paramName, true
		}
	}
	return "", false
}

func interfaceFunctionBody(fn *frontend.FuncDecl) string {
	if body, ok := interfaceFunctionMatchReturnBody(fn); ok {
		return body
	}
	if body, ok := interfaceReturnedClosureCaptureBody(fn); ok {
		return body
	}
	if expr, ok := interfaceThrowExpr(fn); ok {
		return "    throw " + expr
	}
	if body, ok := interfaceBorrowedReturnBody(fn); ok {
		return body
	}
	return "    return " + interfaceReturnExpr(fn)
}

func interfaceBorrowedReturnBody(fn *frontend.FuncDecl) (string, bool) {
	source, ok := interfaceBorrowedReturnExpr(fn)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("    // tetra-interface-lifetime: return=borrow source=%s provenance=param lifetime=call\n    return %s", source, source), true
}

type interfaceCaptureStub struct {
	Name    string
	Type    frontend.TypeRef
	Mutable bool
}

func interfaceThrowExpr(fn *frontend.FuncDecl) (string, bool) {
	if fn == nil || !fn.HasThrows {
		return "", false
	}
	paramNames := map[string]bool{}
	for _, param := range fn.Params {
		paramNames[param.Name] = true
	}
	aliases := map[string]string{}
	for _, stmt := range fn.Body {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames); ok {
				aliases[s.Name] = value
			}
			continue
		case *frontend.AssignStmt:
			target, ok := s.Target.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if _, ok := aliases[target.Name]; !ok {
				continue
			}
			value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames)
			if !ok {
				continue
			}
			aliases[target.Name] = value
			continue
		}
		throwStmt, ok := stmt.(*frontend.ThrowStmt)
		if !ok {
			continue
		}
		formatted, ok := interfaceAggregateStubExprWithAliases(throwStmt.Value, aliases)
		if !ok {
			formatted, ok = interfaceContractExpr(throwStmt.Value)
		}
		if !ok || (!interfaceExprRefsAnyParam(throwStmt.Value, paramNames) && !interfaceExprRefsAnyAlias(throwStmt.Value, aliases)) {
			return "", false
		}
		return formatted, true
	}
	return "", false
}

func interfaceContractExpr(expr frontend.Expr) (string, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name, true
	case *frontend.NumberExpr:
		return fmt.Sprintf("%d", e.Value), true
	case *frontend.BoolLitExpr:
		if e.Value {
			return "true", true
		}
		return "false", true
	case *frontend.NoneLitExpr:
		return "none", true
	case *frontend.StringLitExpr:
		return fmt.Sprintf("%q", string(e.Value)), true
	case *frontend.TryExpr:
		inner, ok := interfaceContractExpr(e.X)
		if !ok {
			return "", false
		}
		return "try " + inner, true
	case *frontend.FieldAccessExpr:
		base, ok := interfaceContractExpr(e.Base)
		if !ok {
			return "", false
		}
		return base + "." + e.Field, true
	case *frontend.CallExpr:
		args := make([]string, 0, len(e.Args))
		for i, arg := range e.Args {
			formatted, ok := interfaceContractExpr(arg)
			if !ok {
				return "", false
			}
			if i < len(e.ArgLabels) && e.ArgLabels[i] != "" {
				formatted = e.ArgLabels[i] + ": " + formatted
			}
			args = append(args, formatted)
		}
		return e.Name + "(" + strings.Join(args, ", ") + ")", true
	case *frontend.StructLitExpr:
		fields := make([]string, 0, len(e.Fields))
		for _, field := range e.Fields {
			formatted, ok := interfaceContractExpr(field.Value)
			if !ok {
				return "", false
			}
			fields = append(fields, field.Name+": "+formatted)
		}
		return formatLSPTypeRef(e.Type) + "(" + strings.Join(fields, ", ") + ")", true
	default:
		return "", false
	}
}

func interfaceExprRefsAnyParam(expr frontend.Expr, params map[string]bool) bool {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return params[e.Name]
	case *frontend.FieldAccessExpr:
		return interfaceExprRefsAnyParam(e.Base, params)
	case *frontend.TryExpr:
		return interfaceExprRefsAnyParam(e.X, params)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			if interfaceExprRefsAnyParam(arg, params) {
				return true
			}
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if interfaceExprRefsAnyParam(field.Value, params) {
				return true
			}
		}
	}
	return false
}

func interfaceExprRefsAnyAlias(expr frontend.Expr, aliases map[string]string) bool {
	if len(aliases) == 0 {
		return false
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return aliases[e.Name] != ""
	case *frontend.FieldAccessExpr:
		return interfaceExprRefsAnyAlias(e.Base, aliases)
	case *frontend.TryExpr:
		return interfaceExprRefsAnyAlias(e.X, aliases)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			if interfaceExprRefsAnyAlias(arg, aliases) {
				return true
			}
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if interfaceExprRefsAnyAlias(field.Value, aliases) {
				return true
			}
		}
	}
	return false
}

func interfaceReturnedClosureCaptureBody(fn *frontend.FuncDecl) (string, bool) {
	if fn.ReturnType.Kind != frontend.TypeRefFunction {
		return "", false
	}
	outerLocals := map[string]interfaceCaptureStub{}
	outerOrder := []string{}
	for _, stmt := range fn.Body {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if _, exists := outerLocals[s.Name]; !exists {
				outerOrder = append(outerOrder, s.Name)
			}
			outerLocals[s.Name] = interfaceCaptureStub{Name: s.Name, Type: s.Type, Mutable: s.Mutable}
		case *frontend.ReturnStmt:
			closure, ok := s.Value.(*frontend.ClosureExpr)
			if !ok || closure.Decl == nil {
				return "", false
			}
			used := map[string]bool{}
			interfaceCollectStmtIdents(closure.Decl.Body, used)
			for _, param := range closure.Decl.Params {
				delete(used, param.Name)
			}
			for local := range interfaceLocalNames(closure.Decl.Body) {
				delete(used, local)
			}
			captures := make([]interfaceCaptureStub, 0, len(outerOrder))
			for _, name := range outerOrder {
				if used[name] {
					captures = append(captures, outerLocals[name])
				}
			}
			if len(captures) == 0 {
				return "", false
			}
			var b strings.Builder
			for _, capture := range captures {
				decl := "let"
				if capture.Mutable {
					decl = "var"
				}
				fmt.Fprintf(&b, "    %s %s: %s = %s\n", decl, capture.Name, formatLSPTypeRef(capture.Type), interfaceReturnLiteral(capture.Type))
			}
			fmt.Fprintf(&b, "    return %s", interfaceCapturedClosureLiteral(closure, captures, "        "))
			return b.String(), true
		}
	}
	return "", false
}

func interfaceCapturedClosureLiteral(closure *frontend.ClosureExpr, captures []interfaceCaptureStub, bodyIndent string) string {
	params := make([]string, 0, len(closure.Decl.Params))
	for _, param := range closure.Decl.Params {
		formatted := formatLSPTypeRef(param.Type)
		if param.Ownership != "" {
			formatted = param.Ownership + " " + formatted
		}
		params = append(params, fmt.Sprintf("%s: %s", param.Name, formatted))
	}
	ret := formatLSPTypeRef(closure.Decl.ReturnType)
	out := "fn(" + strings.Join(params, ", ") + ") -> " + ret
	if closure.Decl.HasThrows {
		out += " throws " + formatLSPTypeRef(closure.Decl.Throws)
	}
	if len(closure.Decl.Uses) > 0 {
		uses := append([]string(nil), closure.Decl.Uses...)
		sort.Strings(uses)
		out += " uses " + strings.Join(uses, ", ")
	}
	var b strings.Builder
	b.WriteString(out)
	b.WriteString(":\n")
	for i, capture := range captures {
		fmt.Fprintf(&b, "%slet __capture_keep%d: %s = %s\n", bodyIndent, i, formatLSPTypeRef(capture.Type), capture.Name)
	}
	fmt.Fprintf(&b, "%sreturn %s", bodyIndent, interfaceReturnLiteral(closure.Decl.ReturnType))
	return b.String()
}

func interfaceLocalNames(stmts []frontend.Stmt) map[string]bool {
	names := map[string]bool{}
	for _, stmt := range stmts {
		if let, ok := stmt.(*frontend.LetStmt); ok {
			names[let.Name] = true
		}
	}
	return names
}

func interfaceCollectStmtIdents(stmts []frontend.Stmt, used map[string]bool) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			interfaceCollectExprIdents(s.Value, used)
		case *frontend.LetStmt:
			interfaceCollectExprIdents(s.Value, used)
		case *frontend.ExprStmt:
			interfaceCollectExprIdents(s.Expr, used)
		case *frontend.IfStmt:
			interfaceCollectExprIdents(s.Cond, used)
			interfaceCollectStmtIdents(s.Then, used)
			interfaceCollectStmtIdents(s.Else, used)
		case *frontend.MatchStmt:
			interfaceCollectExprIdents(s.Value, used)
			for _, c := range s.Cases {
				interfaceCollectExprIdents(c.Guard, used)
				interfaceCollectStmtIdents(c.Body, used)
			}
		}
	}
}

func interfaceCollectExprIdents(expr frontend.Expr, used map[string]bool) {
	switch e := expr.(type) {
	case nil:
		return
	case *frontend.IdentExpr:
		used[e.Name] = true
	case *frontend.BinaryExpr:
		interfaceCollectExprIdents(e.Left, used)
		interfaceCollectExprIdents(e.Right, used)
	case *frontend.UnaryExpr:
		interfaceCollectExprIdents(e.X, used)
	case *frontend.TryExpr:
		interfaceCollectExprIdents(e.X, used)
	case *frontend.AwaitExpr:
		interfaceCollectExprIdents(e.X, used)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			interfaceCollectExprIdents(arg, used)
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			interfaceCollectExprIdents(field.Value, used)
		}
	case *frontend.FieldAccessExpr:
		interfaceCollectExprIdents(e.Base, used)
	case *frontend.IndexExpr:
		interfaceCollectExprIdents(e.Base, used)
		interfaceCollectExprIdents(e.Index, used)
	case *frontend.MatchExpr:
		interfaceCollectExprIdents(e.Value, used)
		for _, c := range e.Cases {
			interfaceCollectExprIdents(c.Guard, used)
			interfaceCollectExprIdents(c.Value, used)
		}
	case *frontend.CatchExpr:
		interfaceCollectExprIdents(e.Call, used)
		for _, c := range e.Cases {
			interfaceCollectExprIdents(c.Guard, used)
			interfaceCollectExprIdents(c.Value, used)
		}
	}
}

func interfaceAggregateReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	paramNames := map[string]bool{}
	paramTypes := map[string]string{}
	for _, param := range fn.Params {
		formatted := formatLSPTypeRef(param.Type)
		paramNames[param.Name] = true
		paramTypes[param.Name] = formatted
	}
	aliases := map[string]string{}
	for _, stmt := range fn.Body {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if s.Type.Kind == frontend.TypeRefOptional {
				aliases[s.Name] = ""
				if value, ok := s.Value.(*frontend.IdentExpr); ok && paramNames[value.Name] {
					aliases[s.Name] = value.Name
				} else if value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames); ok {
					aliases[s.Name] = value
				}
			} else if value, ok := s.Value.(*frontend.IdentExpr); ok && paramTypes[value.Name] == formatLSPTypeRef(s.Type) {
				aliases[s.Name] = value.Name
			} else if value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames); ok {
				aliases[s.Name] = value
			}
			continue
		case *frontend.AssignStmt:
			target, ok := s.Target.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if _, ok := aliases[target.Name]; !ok {
				continue
			}
			value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames)
			if !ok {
				continue
			}
			aliases[target.Name] = value
			continue
		case *frontend.IfLetStmt:
			value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames)
			if !ok || s.Name == "" {
				continue
			}
			branchAliases := interfaceAliasMapCopy(aliases)
			branchAliases[s.Name] = value
			if expr, ok := interfaceAggregateReturnFromBranches(s.Then, branchAliases, s.Else, aliases, paramNames); ok {
				return expr, true
			}
			continue
		case *frontend.MatchStmt:
			value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames)
			if ok {
				for _, c := range s.Cases {
					name, ok := interfaceOptionalSomePatternName(c.Pattern)
					if !ok {
						continue
					}
					branchAliases := interfaceAliasMapCopy(aliases)
					branchAliases[name] = value
					if expr, ok := interfaceAggregateReturnFromStmts(c.Body, branchAliases, paramNames); ok {
						return expr, true
					}
				}
			}
		}
		if expr, ok := interfaceAggregateReturnFromStmts([]frontend.Stmt{stmt}, aliases, paramNames); ok {
			return expr, true
		}
	}
	return "", false
}

func interfaceAggregateReturnFromStmts(stmts []frontend.Stmt, aliases map[string]string, params map[string]bool) (string, bool) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			if !interfaceDirectAggregateExpr(s.Value) {
				continue
			}
			if !interfaceExprContainsClosure(s.Value) && !interfaceExprRefsAnyParam(s.Value, params) && !interfaceExprRefsAnyAlias(s.Value, aliases) {
				continue
			}
			expr, ok := interfaceAggregateStubExprWithAliases(s.Value, aliases)
			if ok {
				return expr, true
			}
		case *frontend.IfStmt:
			if expr, ok := interfaceAggregateReturnFromBranches(s.Then, aliases, s.Else, aliases, params); ok {
				return expr, true
			}
		case *frontend.MatchStmt:
			if expr, ok := interfaceAggregateMatchReturnExpr(s, aliases, params); ok {
				return expr, true
			}
		}
	}
	return "", false
}

func interfaceAggregateReturnFromBranches(thenStmts []frontend.Stmt, thenAliases map[string]string, elseStmts []frontend.Stmt, elseAliases map[string]string, params map[string]bool) (string, bool) {
	thenExpr, thenOK := interfaceAggregateReturnFromStmts(thenStmts, thenAliases, params)
	elseExpr, elseOK := interfaceAggregateReturnFromStmts(elseStmts, elseAliases, params)
	if thenOK && elseOK {
		if thenExpr == elseExpr {
			return thenExpr, true
		}
		return "", false
	}
	if thenOK {
		return thenExpr, true
	}
	if elseOK {
		return elseExpr, true
	}
	return "", false
}

func interfaceAggregateMatchReturnExpr(match *frontend.MatchStmt, aliases map[string]string, params map[string]bool) (string, bool) {
	if match == nil || len(match.Cases) == 0 {
		return "", false
	}
	var commonExpr string
	for _, c := range match.Cases {
		if c.Guard != nil {
			return "", false
		}
		expr, ok := interfaceAggregateReturnFromStmts(c.Body, aliases, params)
		if !ok {
			continue
		}
		if commonExpr == "" {
			commonExpr = expr
			continue
		}
		if commonExpr != expr {
			return "", false
		}
	}
	return commonExpr, commonExpr != ""
}

func interfaceOptionalSomePatternName(expr frontend.Expr) (string, bool) {
	if some, ok := expr.(*frontend.SomePatternExpr); ok {
		return some.Name, some.Name != ""
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call.Name != "some" || len(call.Args) != 1 {
		return "", false
	}
	id, ok := call.Args[0].(*frontend.IdentExpr)
	if !ok || id.Name == "" {
		return "", false
	}
	return id.Name, true
}

func interfaceAliasMapCopy(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func interfaceParamPathExpr(expr frontend.Expr, aliases map[string]string, params map[string]bool) (string, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		if alias := aliases[e.Name]; alias != "" {
			return alias, true
		}
		if params[e.Name] {
			return e.Name, true
		}
	case *frontend.FieldAccessExpr:
		base, ok := interfaceParamPathExpr(e.Base, aliases, params)
		if !ok {
			return "", false
		}
		return base + "." + e.Field, true
	}
	return "", false
}

func interfaceDirectAggregateExpr(expr frontend.Expr) bool {
	switch expr.(type) {
	case *frontend.CallExpr, *frontend.StructLitExpr:
		return true
	default:
		return false
	}
}

func interfaceExprContainsClosure(expr frontend.Expr) bool {
	switch e := expr.(type) {
	case *frontend.ClosureExpr:
		return true
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			if interfaceExprContainsClosure(arg) {
				return true
			}
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if interfaceExprContainsClosure(field.Value) {
				return true
			}
		}
	}
	return false
}

func interfaceAggregateStubExpr(expr frontend.Expr) (string, bool) {
	return interfaceAggregateStubExprWithAliases(expr, nil)
}

func interfaceAggregateStubExprWithAliases(expr frontend.Expr, aliases map[string]string) (string, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		if aliases != nil {
			if alias := aliases[e.Name]; alias != "" {
				return alias, true
			}
		}
		return e.Name, true
	case *frontend.TryExpr:
		inner, ok := interfaceAggregateStubExprWithAliases(e.X, aliases)
		if !ok {
			return "", false
		}
		return "try " + inner, true
	case *frontend.CallExpr:
		args := make([]string, 0, len(e.Args))
		for i, arg := range e.Args {
			formatted, ok := interfaceAggregateStubExprWithAliases(arg, aliases)
			if !ok {
				return "", false
			}
			if i < len(e.ArgLabels) && e.ArgLabels[i] != "" {
				formatted = e.ArgLabels[i] + ": " + formatted
			}
			args = append(args, formatted)
		}
		return e.Name + "(" + strings.Join(args, ", ") + ")", true
	case *frontend.StructLitExpr:
		fields := make([]string, 0, len(e.Fields))
		for _, field := range e.Fields {
			formatted, ok := interfaceAggregateStubExprWithAliases(field.Value, aliases)
			if !ok {
				return "", false
			}
			fields = append(fields, field.Name+": "+formatted)
		}
		return formatLSPTypeRef(e.Type) + "(" + strings.Join(fields, ", ") + ")", true
	case *frontend.FieldAccessExpr:
		base, ok := interfaceAggregateStubExprWithAliases(e.Base, aliases)
		if !ok {
			return "", false
		}
		return base + "." + e.Field, true
	case *frontend.ClosureExpr:
		ref, ok := interfaceClosureTypeRef(e)
		if !ok {
			return "", false
		}
		return interfaceInlineFunctionClosureLiteral(ref), true
	case *frontend.NumberExpr:
		return fmt.Sprintf("%d", e.Value), true
	case *frontend.BoolLitExpr:
		if e.Value {
			return "true", true
		}
		return "false", true
	case *frontend.NoneLitExpr:
		return "none", true
	default:
		return "", false
	}
}

func interfaceClosureTypeRef(closure *frontend.ClosureExpr) (frontend.TypeRef, bool) {
	if closure == nil || closure.Decl == nil {
		return frontend.TypeRef{}, false
	}
	params := make([]frontend.TypeRef, 0, len(closure.Decl.Params))
	ownership := make([]string, 0, len(closure.Decl.Params))
	for _, param := range closure.Decl.Params {
		params = append(params, param.Type)
		ownership = append(ownership, param.Ownership)
	}
	ret := closure.Decl.ReturnType
	ref := frontend.TypeRef{
		Kind:           frontend.TypeRefFunction,
		Params:         params,
		ParamOwnership: ownership,
		Return:         &ret,
		Uses:           append([]string(nil), closure.Decl.Uses...),
	}
	if closure.Decl.HasThrows {
		throws := closure.Decl.Throws
		ref.Throws = &throws
	}
	return ref, true
}

func interfaceSameTypedParameterReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	returnSig := formatLSPTypeRef(fn.ReturnType)
	sameTypedParams := map[string]bool{}
	for _, param := range fn.Params {
		if formatLSPTypeRef(param.Type) == returnSig {
			sameTypedParams[param.Name] = true
		}
	}
	aliases := map[string]string{}
	for _, stmt := range fn.Body {
		if let, ok := stmt.(*frontend.LetStmt); ok {
			if formatLSPTypeRef(let.Type) != returnSig {
				continue
			}
			if id, ok := let.Value.(*frontend.IdentExpr); ok && sameTypedParams[id.Name] {
				aliases[let.Name] = id.Name
			}
			continue
		}
		ret, ok := stmt.(*frontend.ReturnStmt)
		if !ok {
			continue
		}
		id, ok := ret.Value.(*frontend.IdentExpr)
		if ok && sameTypedParams[id.Name] {
			return id.Name, true
		}
		if ok {
			if param := aliases[id.Name]; param != "" {
				return param, true
			}
		}
	}
	return "", false
}

func interfaceFunctionReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	returnSig := formatLSPTypeRef(fn.ReturnType)
	functionParams := map[string]bool{}
	valueParams := map[string]bool{}
	for _, param := range fn.Params {
		valueParams[param.Name] = true
		if param.Type.Kind == frontend.TypeRefFunction && formatLSPTypeRef(param.Type) == returnSig {
			functionParams[param.Name] = true
		}
	}
	aliases := map[string]string{}
	for _, stmt := range fn.Body {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if formatLSPTypeRef(s.Type) != returnSig {
				continue
			}
			if path, ok := interfaceFunctionReturnParamPath(s.Value, aliases, functionParams, valueParams); ok {
				aliases[s.Name] = path
			}
			continue
		case *frontend.AssignStmt:
			target, ok := s.Target.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if _, ok := aliases[target.Name]; !ok {
				continue
			}
			if path, ok := interfaceFunctionReturnParamPath(s.Value, aliases, functionParams, valueParams); ok {
				aliases[target.Name] = path
			}
			continue
		case *frontend.ReturnStmt:
			if path, ok := interfaceFunctionReturnParamPath(s.Value, aliases, functionParams, valueParams); ok {
				return path, true
			}
		}
	}
	return "", false
}

func interfaceFunctionMatchReturnBody(fn *frontend.FuncDecl) (string, bool) {
	if fn == nil || fn.ReturnType.Kind != frontend.TypeRefFunction {
		return "", false
	}
	paramTypes := map[string]string{}
	for _, param := range fn.Params {
		paramTypes[param.Name] = formatLSPTypeRef(param.Type)
	}
	for _, stmt := range fn.Body {
		match, ok := stmt.(*frontend.MatchStmt)
		if !ok || match.Value == nil {
			continue
		}
		valueName := interfaceCallbackArgumentName(match.Value)
		if valueName == "" {
			continue
		}
		valueType := paramTypes[valueName]
		if valueType == "" {
			continue
		}
		var b strings.Builder
		fmt.Fprintf(&b, "    match %s:\n", valueName)
		preservedPayload := false
		for _, c := range match.Cases {
			if c.Guard != nil {
				return "", false
			}
			binding, hasBinding := interfacePatternBindingName(c.Pattern)
			pattern := "_"
			if !c.Default {
				pattern = interfaceFunctionMatchPattern(c.Pattern, valueType)
			}
			fmt.Fprintf(&b, "    case %s:\n", pattern)
			ret, ok := singleReturnExpr(c.Body)
			if !ok {
				return "", false
			}
			if id, ok := ret.(*frontend.IdentExpr); ok && hasBinding && id.Name == binding {
				fmt.Fprintf(&b, "        return %s\n", id.Name)
				preservedPayload = true
				continue
			}
			if expr, ok := interfaceContractExpr(ret); ok && expr != "" {
				fmt.Fprintf(&b, "        return %s\n", expr)
				continue
			}
			fmt.Fprintf(&b, "        return %s\n", interfaceFunctionClosureLiteral(fn.ReturnType, "            "))
		}
		if preservedPayload {
			return strings.TrimRight(b.String(), "\n"), true
		}
	}
	return "", false
}

func interfacePatternBindingName(expr frontend.Expr) (string, bool) {
	switch e := expr.(type) {
	case *frontend.SomePatternExpr:
		return e.Name, e.Name != ""
	case *frontend.EnumCasePatternExpr:
		if len(e.Bindings) == 0 || e.Bindings[0] == "" {
			return "", false
		}
		return e.Bindings[0], true
	case *frontend.CallExpr:
		if len(e.Args) == 0 {
			return "", false
		}
		id, ok := e.Args[0].(*frontend.IdentExpr)
		if !ok || id.Name == "" {
			return "", false
		}
		return id.Name, true
	default:
		return "", false
	}
}

func interfaceFunctionMatchPattern(expr frontend.Expr, enumType string) string {
	switch e := expr.(type) {
	case *frontend.SomePatternExpr:
		if enumType != "" && !strings.HasSuffix(enumType, "?") {
			return enumType + ".some(" + e.Name + ")"
		}
	case *frontend.CallExpr:
		if enumType != "" && len(e.Args) > 0 {
			names := make([]string, 0, len(e.Args))
			for _, arg := range e.Args {
				id, ok := arg.(*frontend.IdentExpr)
				if !ok {
					return interfaceFormatExpr(expr)
				}
				names = append(names, id.Name)
			}
			return enumType + "." + interfaceShortName(e.Name) + "(" + strings.Join(names, ", ") + ")"
		}
	case *frontend.IdentExpr:
		if enumType != "" && !strings.HasSuffix(enumType, "?") {
			return enumType + "." + e.Name
		}
	case *frontend.EnumCasePatternExpr:
		if e.TypeName == "" && enumType != "" {
			if e.HasPayload {
				return enumType + "." + e.CaseName + "(" + strings.Join(e.Bindings, ", ") + ")"
			}
			return enumType + "." + e.CaseName
		}
	}
	return interfaceFormatExpr(expr)
}

func interfaceFormatExpr(expr frontend.Expr) string {
	var p sourcePrinter
	return p.formatExpr(expr)
}

func interfaceShortName(name string) string {
	if idx := strings.LastIndex(name, "."); idx >= 0 && idx+1 < len(name) {
		return name[idx+1:]
	}
	return name
}

func interfaceFunctionReturnParamPath(expr frontend.Expr, aliases map[string]string, functionParams, valueParams map[string]bool) (string, bool) {
	if id, ok := expr.(*frontend.IdentExpr); ok {
		if functionParams[id.Name] {
			return id.Name, true
		}
		if alias := aliases[id.Name]; alias != "" {
			return alias, true
		}
	}
	name := interfaceCallbackArgumentName(expr)
	if name == "" {
		return "", false
	}
	for paramName := range valueParams {
		if name == paramName || strings.HasPrefix(name, paramName+".") {
			return name, true
		}
	}
	return "", false
}

func interfaceCallbackArgumentName(expr frontend.Expr) string {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name
	case *frontend.FieldAccessExpr:
		base := interfaceCallbackArgumentName(e.Base)
		if base == "" {
			return ""
		}
		return base + "." + e.Field
	default:
		return ""
	}
}

func interfaceFunctionClosureLiteral(ref frontend.TypeRef, bodyIndent string) string {
	params := make([]string, 0, len(ref.Params))
	for i, param := range ref.Params {
		formatted := formatLSPTypeRef(param)
		if i < len(ref.ParamOwnership) && ref.ParamOwnership[i] != "" {
			formatted = ref.ParamOwnership[i] + " " + formatted
		}
		params = append(params, fmt.Sprintf("p%d: %s", i, formatted))
	}
	ret := "?"
	body := "0"
	if ref.Return != nil {
		ret = formatLSPTypeRef(*ref.Return)
		if ref.Return.Kind == frontend.TypeRefFunction {
			body = interfaceFunctionClosureLiteral(*ref.Return, bodyIndent+"    ")
		} else {
			body = interfaceReturnLiteral(*ref.Return)
		}
	}
	out := "fn(" + strings.Join(params, ", ") + ") -> " + ret
	if ref.Throws != nil {
		out += " throws " + formatLSPTypeRef(*ref.Throws)
	}
	if len(ref.Uses) > 0 {
		uses := append([]string(nil), ref.Uses...)
		sort.Strings(uses)
		out += " uses " + strings.Join(uses, ", ")
	}
	return out + ":\n" + bodyIndent + "return " + body
}

func interfaceInlineFunctionClosureLiteral(ref frontend.TypeRef) string {
	params := make([]string, 0, len(ref.Params))
	for i, param := range ref.Params {
		formatted := formatLSPTypeRef(param)
		if i < len(ref.ParamOwnership) && ref.ParamOwnership[i] != "" {
			formatted = ref.ParamOwnership[i] + " " + formatted
		}
		params = append(params, fmt.Sprintf("p%d: %s", i, formatted))
	}
	ret := "?"
	body := "0"
	if ref.Return != nil {
		ret = formatLSPTypeRef(*ref.Return)
		if ref.Return.Kind == frontend.TypeRefFunction {
			body = interfaceInlineFunctionClosureLiteral(*ref.Return)
		} else {
			body = interfaceReturnLiteral(*ref.Return)
		}
	}
	out := "fn(" + strings.Join(params, ", ") + ") -> " + ret
	if ref.Throws != nil {
		out += " throws " + formatLSPTypeRef(*ref.Throws)
	}
	if len(ref.Uses) > 0 {
		uses := append([]string(nil), ref.Uses...)
		sort.Strings(uses)
		out += " uses " + strings.Join(uses, ", ")
	}
	return out + " = " + body
}
